// Command loadcheck records a repeatable local cold-start and warm-read baseline.
package main

import (
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"sort"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"github.com/ilhamnugraha8944/tauco/backend/internal/composition"
	"github.com/ilhamnugraha8944/tauco/backend/internal/platform/config"
	"github.com/ilhamnugraha8944/tauco/backend/internal/platform/database"
)

const requestCount = 200

func main() { os.Exit(run()) }

func run() int {
	cfg, err := config.Load()
	if err != nil {
		return fail("invalid application configuration")
	}
	cfg.Log.Level = "error"
	databaseConfig, err := database.LoadRuntimeConfig(os.LookupEnv)
	if err != nil {
		return fail("invalid database configuration")
	}
	started := time.Now()
	ctx, cancel := context.WithTimeout(context.Background(), 20*time.Second)
	defer cancel()
	app, err := composition.NewPublicAPI(ctx, cfg, databaseConfig,
		composition.PublicAPISecrets{
			CursorHMAC: []byte(os.Getenv("CURSOR_HMAC_SECRET")), ContactHMAC: []byte(os.Getenv("CONTACT_HMAC_SECRET")),
			RateHMAC: []byte(os.Getenv("RATE_LIMIT_HMAC_SECRET")), MetricsBearer: []byte(os.Getenv("METRICS_BEARER_TOKEN")),
		}, composition.PublicAPIInfrastructure{
			RedisURL: os.Getenv("REDIS_URL"), CORSOrigins: splitCSV(os.Getenv("CORS_ALLOWED_ORIGINS")),
			TrustedProxyCIDRs: splitCSV(os.Getenv("TRUSTED_PROXY_CIDRS")), MediaRoot: os.Getenv("MEDIA_LOCAL_ROOT"),
			MediaStorageDriver: "local", ContactAPIEnabled: true,
		})
	if err != nil {
		return fail("API initialization failed")
	}
	defer func() { _ = app.Close() }()
	cold := time.Since(started)

	for index := range 10 {
		if status := request(app.Handler(), index); status != http.StatusOK {
			return fail("warm-up request failed")
		}
	}
	latencies := make([]time.Duration, requestCount)
	jobs := make(chan int)
	var failures atomic.Int64
	var wait sync.WaitGroup
	for range 16 {
		wait.Add(1)
		go func() {
			defer wait.Done()
			for index := range jobs {
				started := time.Now()
				if status := request(app.Handler(), index+20); status != http.StatusOK {
					failures.Add(1)
				}
				latencies[index] = time.Since(started)
			}
		}()
	}
	for index := range requestCount {
		jobs <- index
	}
	close(jobs)
	wait.Wait()
	sort.Slice(latencies, func(i, j int) bool { return latencies[i] < latencies[j] })
	p95 := latencies[percentileIndex(95, len(latencies))]
	p99 := latencies[percentileIndex(99, len(latencies))]
	errorRate := float64(failures.Load()) / requestCount
	fmt.Printf("cold_start_ms=%d warm_requests=%d p95_ms=%d p99_ms=%d error_rate=%.3f\n",
		cold.Milliseconds(), requestCount, p95.Milliseconds(), p99.Milliseconds(), errorRate)
	if p95 > 100*time.Millisecond || errorRate >= 0.01 {
		return 1
	}
	return 0
}

func request(handler http.Handler, index int) int {
	request := httptest.NewRequest(http.MethodGet, "/api/v1/home", nil)
	request.RemoteAddr = fmt.Sprintf("198.51.100.%d:1234", index%250+1)
	response := httptest.NewRecorder()
	handler.ServeHTTP(response, request)
	return response.Code
}

func percentileIndex(percentile, length int) int {
	index := (percentile*length + 99) / 100
	if index < 1 {
		return 0
	}
	if index > length {
		return length - 1
	}
	return index - 1
}

func splitCSV(value string) []string {
	var result []string
	for _, item := range strings.Split(value, ",") {
		if item = strings.TrimSpace(item); item != "" {
			result = append(result, item)
		}
	}
	return result
}

func fail(message string) int {
	_, _ = fmt.Fprintln(os.Stderr, message)
	return 1
}
