package composition

import (
	"context"
	"crypto/sha256"
	"crypto/subtle"
	"database/sql"
	"errors"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	api "github.com/ilhamnugraha8944/tauco/backend/internal/delivery/api"
	"github.com/ilhamnugraha8944/tauco/backend/internal/platform/httpserver"
	"github.com/ilhamnugraha8944/tauco/backend/internal/platform/observability"
)

type readinessCache interface {
	Ping(context.Context) error
}

type operationsDependencies struct {
	Database     *sql.DB
	Cache        readinessCache
	MediaRoot    string
	MetricsToken []byte
	Metrics      *observability.Registry
}

func registerOperations(router gin.IRouter, dependencies operationsDependencies) error {
	if router == nil || dependencies.Database == nil || dependencies.Metrics == nil ||
		len(dependencies.MetricsToken) < 32 {
		return errors.New("invalid operations dependencies")
	}
	router.GET("/health/ready", readinessHandler(dependencies))
	router.GET("/internal/metrics", metricsAuth(dependencies.MetricsToken), metricsHandler(dependencies))
	return nil
}

func readinessHandler(dependencies operationsDependencies) gin.HandlerFunc {
	return func(ctx *gin.Context) {
		checkContext, cancel := context.WithTimeout(ctx.Request.Context(), 2*time.Second)
		defer cancel()
		if err := dependencies.Database.PingContext(checkContext); err != nil {
			httpserver.WriteProblem(ctx, http.StatusServiceUnavailable,
				"urn:tauco-cap-badak:problem:service-unavailable", "Layanan belum siap",
				"Dependency wajib belum tersedia.", "SERVICE_UNAVAILABLE")
			return
		}
		redisState := api.DependencyStateHealthy
		if dependencies.Cache == nil || dependencies.Cache.Ping(checkContext) != nil {
			redisState = api.DependencyStateDegraded
		}
		storageState := api.DependencyStateDegraded
		if info, err := os.Stat(dependencies.MediaRoot); err == nil && info.IsDir() {
			storageState = api.DependencyStateHealthy
		}
		status := api.ReadinessResponseStatusReady
		if redisState == api.DependencyStateDegraded || storageState == api.DependencyStateDegraded {
			status = api.ReadinessResponseStatusDegraded
		}
		ctx.Header("Cache-Control", "no-store")
		ctx.JSON(http.StatusOK, api.ReadinessResponse{
			Status: status,
			Dependencies: api.ReadinessDependencies{
				Postgres: api.ReadinessDependenciesPostgresHealthy,
				Redis:    redisState, Storage: storageState,
			},
		})
	}
}

func metricsAuth(token []byte) gin.HandlerFunc {
	wanted := sha256.Sum256(token)
	return func(ctx *gin.Context) {
		authorization := ctx.GetHeader("Authorization")
		if !strings.HasPrefix(authorization, "Bearer ") {
			ctx.Header("WWW-Authenticate", `Bearer realm="metrics"`)
			httpserver.WriteProblem(ctx, http.StatusUnauthorized,
				"urn:tauco-cap-badak:problem:unauthorized", "Autentikasi diperlukan",
				"Bearer token metrics diperlukan.", "UNAUTHORIZED")
			ctx.Abort()
			return
		}
		candidate := sha256.Sum256([]byte(strings.TrimPrefix(authorization, "Bearer ")))
		if subtle.ConstantTimeCompare(candidate[:], wanted[:]) != 1 {
			httpserver.WriteProblem(ctx, http.StatusForbidden,
				"urn:tauco-cap-badak:problem:forbidden", "Akses ditolak",
				"Token metrics tidak valid.", "FORBIDDEN")
			ctx.Abort()
			return
		}
		ctx.Next()
	}
}

func metricsHandler(dependencies operationsDependencies) gin.HandlerFunc {
	return func(ctx *gin.Context) {
		queryContext, cancel := context.WithTimeout(ctx.Request.Context(), 3*time.Second)
		defer cancel()
		snapshot, err := operationalSnapshot(queryContext, dependencies.Database)
		if err != nil {
			httpserver.WriteProblem(ctx, http.StatusInternalServerError,
				"urn:tauco-cap-badak:problem:internal", "Terjadi kesalahan internal",
				"Metrics tidak dapat dikumpulkan saat ini.", "INTERNAL_SERVER_ERROR")
			return
		}
		ctx.Header("Cache-Control", "no-store")
		ctx.Data(http.StatusOK, "text/plain; version=0.0.4; charset=utf-8", []byte(
			dependencies.Metrics.Render(dependencies.Database.Stats(), snapshot),
		))
	}
}

func operationalSnapshot(ctx context.Context, database *sql.DB) (observability.Snapshot, error) {
	var snapshot observability.Snapshot
	rows, err := database.QueryContext(ctx, `
		SELECT kind, status, count(*)
		FROM tauco_app.background_jobs
		GROUP BY kind, status`)
	if err != nil {
		return snapshot, err
	}
	for rows.Next() {
		var value observability.LabelCount
		if err := rows.Scan(&value.Kind, &value.Status, &value.Count); err != nil {
			return snapshot, err
		}
		snapshot.Jobs = append(snapshot.Jobs, value)
	}
	if err := rows.Err(); err != nil {
		_ = rows.Close()
		return snapshot, err
	}
	if err := rows.Close(); err != nil {
		return snapshot, err
	}
	rows, err = database.QueryContext(ctx, `
		SELECT status, count(*)
		FROM tauco_app.media_assets
		GROUP BY status`)
	if err != nil {
		return snapshot, err
	}
	for rows.Next() {
		var value observability.LabelCount
		if err := rows.Scan(&value.Status, &value.Count); err != nil {
			return snapshot, err
		}
		snapshot.Media = append(snapshot.Media, value)
	}
	if err := rows.Err(); err != nil {
		_ = rows.Close()
		return snapshot, err
	}
	if err := rows.Close(); err != nil {
		return snapshot, err
	}
	err = database.QueryRowContext(ctx, `
		SELECT count(*) FROM tauco_app.contact_messages
		WHERE retention_delete_at <= transaction_timestamp()`).Scan(&snapshot.RetentionDue)
	return snapshot, err
}
