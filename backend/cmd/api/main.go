// Command api starts the Tauco Cap Badak public API process.
package main

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"

	"github.com/ilhamnugraha8944/tauco/backend/internal/auth"
	"github.com/ilhamnugraha8944/tauco/backend/internal/composition"
	"github.com/ilhamnugraha8944/tauco/backend/internal/platform/config"
	"github.com/ilhamnugraha8944/tauco/backend/internal/platform/database"
)

func main() {
	os.Exit(run())
}

func run() int {
	cfg, err := config.Load()
	if err != nil {
		_, _ = fmt.Fprintf(os.Stderr, "api configuration error: %v\n", err)
		return 1
	}

	databaseConfig, err := database.LoadRuntimeConfig(os.LookupEnv)
	if err != nil {
		_, _ = fmt.Fprintf(os.Stderr, "database configuration error: %v\n", err)
		return 1
	}
	var adminAuth *composition.AdminAuthOptions
	if strings.TrimSpace(os.Getenv("ADMIN_DATABASE_URL")) != "" {
		adminDatabase, loadErr := database.LoadAdminRuntimeConfig(os.LookupEnv)
		if loadErr != nil {
			_, _ = fmt.Fprintln(os.Stderr, "admin database configuration error")
			return 1
		}
		authRuntime, loadErr := auth.LoadRuntime(os.LookupEnv)
		if loadErr != nil {
			_, _ = fmt.Fprintln(os.Stderr, "admin auth configuration error")
			return 1
		}
		adminAuth = &composition.AdminAuthOptions{
			Database: adminDatabase, Runtime: authRuntime,
			AllowedOrigins: splitCSV(os.Getenv("ADMIN_ALLOWED_ORIGINS")),
			SecureCookies:  strings.EqualFold(os.Getenv("ADMIN_COOKIE_SECURE"), "true"),
		}
	}

	initializationContext, cancelInitialization := context.WithTimeout(
		context.Background(),
		15*time.Second,
	)
	defer cancelInitialization()
	app, err := composition.NewPublicAPI(
		initializationContext,
		cfg,
		databaseConfig,
		composition.PublicAPISecrets{
			CursorHMAC:    []byte(os.Getenv("CURSOR_HMAC_SECRET")),
			ContactHMAC:   []byte(os.Getenv("CONTACT_HMAC_SECRET")),
			RateHMAC:      []byte(os.Getenv("RATE_LIMIT_HMAC_SECRET")),
			MetricsBearer: []byte(os.Getenv("METRICS_BEARER_TOKEN")),
		},
		composition.PublicAPIInfrastructure{
			RedisURL:          os.Getenv("REDIS_URL"),
			CORSOrigins:       splitCSV(os.Getenv("CORS_ALLOWED_ORIGINS")),
			TrustedProxyCIDRs: splitCSV(os.Getenv("TRUSTED_PROXY_CIDRS")),
			MediaRoot:         os.Getenv("MEDIA_LOCAL_ROOT"),
			AdminAuth:         adminAuth,
		},
	)
	if err != nil {
		_, _ = fmt.Fprintln(os.Stderr, "api initialization failed")
		return 1
	}
	defer func() {
		if err := app.Close(); err != nil {
			_, _ = fmt.Fprintln(os.Stderr, "api logger flush failed")
		}
	}()

	ctx, stop := signal.NotifyContext(
		context.Background(),
		os.Interrupt,
		syscall.SIGTERM,
	)
	defer stop()

	if err := app.Run(ctx); err != nil {
		return 1
	}
	return 0
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
