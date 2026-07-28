// Command api starts the Tauco Cap Badak public API process.
package main

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"syscall"
	"time"

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

	initializationContext, cancelInitialization := context.WithTimeout(
		context.Background(),
		15*time.Second,
	)
	defer cancelInitialization()
	app, err := composition.NewPublicAPI(
		initializationContext,
		cfg,
		databaseConfig,
		[]byte(os.Getenv("CURSOR_HMAC_SECRET")),
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
