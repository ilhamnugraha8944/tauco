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
	mediastorage "github.com/ilhamnugraha8944/tauco/backend/internal/media/storage"
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
	deployment, err := config.LoadDeployment(os.LookupEnv, cfg.Environment)
	if err != nil {
		_, _ = fmt.Fprintf(os.Stderr, "deployment configuration error: %v\n", err)
		return 1
	}
	var adminAuth *composition.AdminAuthOptions
	if deployment.AdminRemoteEnabled {
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
			AllowedOrigins: deployment.AdminOrigins,
			SecureCookies:  deployment.AdminCookieSecure,
			BFFSecret:      deployment.AdminBFFSecret,
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
			RedisURL:           os.Getenv("REDIS_URL"),
			CORSOrigins:        deployment.CORSOrigins,
			TrustedProxyCIDRs:  splitCSV(os.Getenv("TRUSTED_PROXY_CIDRS")),
			MediaRoot:          os.Getenv("MEDIA_LOCAL_ROOT"),
			MediaStorageDriver: deployment.MediaStorageDriver,
			MediaS3: mediastorage.S3Config{
				Endpoint: deployment.MediaS3Endpoint, Region: deployment.MediaS3Region,
				Bucket: deployment.MediaS3Bucket, Prefix: deployment.MediaS3Prefix,
				AccessKeyID: deployment.MediaS3AccessKeyID, SecretAccessKey: deployment.MediaS3SecretKey,
			},
			ContactAPIEnabled: deployment.ContactAPIEnabled,
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
