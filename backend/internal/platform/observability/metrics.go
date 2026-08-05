// Package observability exposes bounded, PII-free operational metrics without
// requiring a remote telemetry provider.
package observability

import (
	"database/sql"
	"fmt"
	"sort"
	"strconv"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"github.com/gin-gonic/gin"
)

type httpKey struct {
	method string
	route  string
	status int
}

type httpAggregate struct {
	count       uint64
	durationSec float64
}

// LabelCount is one pre-aggregated operational gauge with bounded labels.
type LabelCount struct {
	Kind   string
	Status string
	Count  int64
}

// Snapshot contains state that is inexpensive to aggregate from PostgreSQL at
// scrape time. None of its labels can contain visitor-controlled data.
type Snapshot struct {
	Jobs          []LabelCount
	Media         []LabelCount
	AdminSessions []LabelCount
	Publishing    []LabelCount
	RetentionDue  int64
}

type Registry struct {
	inFlight atomic.Int64
	mu       sync.Mutex
	http     map[httpKey]httpAggregate
	cache    map[string]uint64
	rate     map[string]uint64
}

func NewRegistry() *Registry {
	return &Registry{
		http:  make(map[httpKey]httpAggregate),
		cache: make(map[string]uint64),
		rate:  make(map[string]uint64),
	}
}

func (registry *Registry) HTTPMiddleware() gin.HandlerFunc {
	return func(ctx *gin.Context) {
		started := time.Now()
		registry.inFlight.Add(1)
		defer registry.inFlight.Add(-1)
		ctx.Next()
		route := ctx.FullPath()
		if route == "" {
			route = "unmatched"
		}
		registry.mu.Lock()
		key := httpKey{method: ctx.Request.Method, route: route, status: ctx.Writer.Status()}
		aggregate := registry.http[key]
		aggregate.count++
		aggregate.durationSec += time.Since(started).Seconds()
		registry.http[key] = aggregate
		registry.mu.Unlock()
	}
}

func (registry *Registry) ObserveCache(outcome string) {
	registry.increment(registry.cache, bounded(outcome, "unknown"))
}

func (registry *Registry) ObserveRateLimit(outcome string) {
	registry.increment(registry.rate, bounded(outcome, "unknown"))
}

func (registry *Registry) increment(target map[string]uint64, label string) {
	if registry == nil {
		return
	}
	registry.mu.Lock()
	target[label]++
	registry.mu.Unlock()
}

func (registry *Registry) Render(databaseStats sql.DBStats, snapshot Snapshot) string {
	registry.mu.Lock()
	httpValues := make(map[httpKey]httpAggregate, len(registry.http))
	for key, value := range registry.http {
		httpValues[key] = value
	}
	cacheValues := cloneCounts(registry.cache)
	rateValues := cloneCounts(registry.rate)
	registry.mu.Unlock()

	var output strings.Builder
	output.WriteString("# TYPE tauco_http_requests_total counter\n")
	keys := make([]httpKey, 0, len(httpValues))
	for key := range httpValues {
		keys = append(keys, key)
	}
	sort.Slice(keys, func(i, j int) bool {
		left, right := keys[i], keys[j]
		return left.route+left.method+strconv.Itoa(left.status) < right.route+right.method+strconv.Itoa(right.status)
	})
	for _, key := range keys {
		value := httpValues[key]
		labels := fmt.Sprintf("method=\"%s\",route=\"%s\",status=\"%d\"", key.method, key.route, key.status)
		fmt.Fprintf(&output, "tauco_http_requests_total{%s} %d\n", labels, value.count)
		fmt.Fprintf(&output, "tauco_http_request_duration_seconds_sum{%s} %.9f\n", labels, value.durationSec)
		fmt.Fprintf(&output, "tauco_http_request_duration_seconds_count{%s} %d\n", labels, value.count)
		if key.status >= 400 {
			fmt.Fprintf(&output, "tauco_http_problems_total{%s} %d\n", labels, value.count)
		}
	}
	fmt.Fprintf(&output, "tauco_http_requests_in_flight %d\n", registry.inFlight.Load())
	renderCounts(&output, "tauco_cache_operations_total", "outcome", cacheValues)
	renderCounts(&output, "tauco_rate_limit_total", "outcome", rateValues)
	fmt.Fprintf(&output, "tauco_db_pool_open_connections %d\n", databaseStats.OpenConnections)
	fmt.Fprintf(&output, "tauco_db_pool_in_use_connections %d\n", databaseStats.InUse)
	fmt.Fprintf(&output, "tauco_db_pool_idle_connections %d\n", databaseStats.Idle)
	fmt.Fprintf(&output, "tauco_db_pool_wait_total %d\n", databaseStats.WaitCount)

	sortLabelCounts(snapshot.Jobs)
	for _, value := range snapshot.Jobs {
		fmt.Fprintf(&output, "tauco_background_jobs{kind=\"%s\",status=\"%s\"} %d\n", value.Kind, value.Status, value.Count)
	}
	sortLabelCounts(snapshot.Media)
	for _, value := range snapshot.Media {
		fmt.Fprintf(&output, "tauco_media_assets{status=\"%s\"} %d\n", value.Status, value.Count)
	}
	sortLabelCounts(snapshot.AdminSessions)
	for _, value := range snapshot.AdminSessions {
		fmt.Fprintf(&output, "tauco_admin_sessions{status=\"%s\"} %d\n", value.Status, value.Count)
	}
	sortLabelCounts(snapshot.Publishing)
	for _, value := range snapshot.Publishing {
		fmt.Fprintf(&output, "tauco_publishing_entities{kind=\"%s\",status=\"%s\"} %d\n", value.Kind, value.Status, value.Count)
	}
	fmt.Fprintf(&output, "tauco_contact_retention_due %d\n", snapshot.RetentionDue)
	return output.String()
}

func cloneCounts(source map[string]uint64) map[string]uint64 {
	clone := make(map[string]uint64, len(source))
	for key, value := range source {
		clone[key] = value
	}
	return clone
}

func renderCounts(output *strings.Builder, metric, label string, values map[string]uint64) {
	keys := make([]string, 0, len(values))
	for key := range values {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	for _, key := range keys {
		fmt.Fprintf(output, "%s{%s=\"%s\"} %d\n", metric, label, key, values[key])
	}
}

func sortLabelCounts(values []LabelCount) {
	sort.Slice(values, func(i, j int) bool {
		return values[i].Kind+values[i].Status < values[j].Kind+values[j].Status
	})
}

func bounded(value, fallback string) string {
	if value == "" || len(value) > 32 {
		return fallback
	}
	for _, character := range value {
		if (character < 'a' || character > 'z') && character != '_' && character != '-' {
			return fallback
		}
	}
	return value
}
