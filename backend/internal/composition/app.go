// Package composition is the application composition root. It is the only
// place that constructs concrete platform adapters for the API process.
package composition

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"net/http"
	"syscall"

	"github.com/gin-gonic/gin"
	catalogapp "github.com/ilhamnugraha8944/tauco/backend/internal/catalog/application"
	"github.com/ilhamnugraha8944/tauco/backend/internal/catalog/delivery/cursor"
	catalogrepo "github.com/ilhamnugraha8944/tauco/backend/internal/catalog/repository"
	contentapp "github.com/ilhamnugraha8944/tauco/backend/internal/content/application"
	contentrepo "github.com/ilhamnugraha8944/tauco/backend/internal/content/repository"
	"github.com/ilhamnugraha8944/tauco/backend/internal/delivery/api"
	"github.com/ilhamnugraha8944/tauco/backend/internal/platform/config"
	"github.com/ilhamnugraha8944/tauco/backend/internal/platform/database"
	"github.com/ilhamnugraha8944/tauco/backend/internal/platform/httpmiddleware"
	"github.com/ilhamnugraha8944/tauco/backend/internal/platform/httpserver"
	"github.com/ilhamnugraha8944/tauco/backend/internal/platform/logging"
)

// App owns the API process dependencies and their lifecycle.
type App struct {
	logger   *logging.Logger
	handler  http.Handler
	server   *httpserver.Server
	database *sql.DB
}

// New validates configuration and wires the B1 platform foundation.
func New(cfg config.Config) (*App, error) {
	return newApp(cfg, nil)
}

// NewPublicAPI wires the B4 public read API to least-privilege PostgreSQL.
func NewPublicAPI(
	ctx context.Context,
	cfg config.Config,
	databaseConfig database.RuntimeConfig,
	cursorSecret []byte,
) (*App, error) {
	gormDatabase, err := database.OpenGORM(ctx, databaseConfig)
	if err != nil {
		return nil, err
	}
	sqlDatabase, err := gormDatabase.DB()
	if err != nil {
		return nil, fmt.Errorf("access PostgreSQL connection pool: %w", err)
	}
	closeDatabase := func() {
		_ = sqlDatabase.Close()
	}

	pageRepository, err := contentrepo.NewPostgresRepository(gormDatabase)
	if err != nil {
		closeDatabase()
		return nil, err
	}
	productRepository, err := catalogrepo.NewPostgresRepository(gormDatabase)
	if err != nil {
		closeDatabase()
		return nil, err
	}
	pageReader, err := contentapp.NewPublishedReader(pageRepository)
	if err != nil {
		closeDatabase()
		return nil, err
	}
	cursorCodec, err := cursor.NewHMACSHA256(cursorSecret)
	if err != nil {
		closeDatabase()
		return nil, err
	}
	productReader, err := catalogapp.NewPublishedReader(
		productRepository,
		cursorCodec,
	)
	if err != nil {
		closeDatabase()
		return nil, err
	}
	publicReadServer, err := api.NewPublicReadServer(pageReader, productReader)
	if err != nil {
		closeDatabase()
		return nil, err
	}

	app, err := newApp(cfg, func(router gin.IRouter) {
		api.RegisterSafePublicReadHandlers(
			router,
			publicReadServer,
			nil,
			"",
		)
	})
	if err != nil {
		closeDatabase()
		return nil, err
	}
	app.database = sqlDatabase
	return app, nil
}

func newApp(
	cfg config.Config,
	registerRoutes func(gin.IRouter),
) (*App, error) {
	if err := cfg.Validate(); err != nil {
		return nil, fmt.Errorf("validate application config: %w", err)
	}

	logger, err := logging.New(cfg.Log, cfg.Environment)
	if err != nil {
		return nil, fmt.Errorf("create application logger: %w", err)
	}

	router, err := httpserver.NewRouter(httpserver.RouterOptions{
		TrustedProxies: []string{},
		PanicReporter: func(ctx context.Context, report httpserver.PanicReport) {
			requestID, _ := httpserver.RequestIDFromContext(ctx)
			logger.Error(
				"http panic recovered",
				logging.RequestID(requestID),
				logging.Route(report.Route),
				logging.Method(report.Method),
				logging.Status(report.Status),
				logging.Latency(report.Latency),
				logging.ErrorCode("HTTP_PANIC_RECOVERED"),
			)
		},
		Middleware: []gin.HandlerFunc{
			httpmiddleware.AccessLog(logger),
		},
	})
	if err != nil {
		_ = syncLogger(logger)
		return nil, fmt.Errorf("create http router: %w", err)
	}
	if registerRoutes != nil {
		registerRoutes(router)
	}

	serverConfig := httpserver.DefaultServerConfig()
	serverConfig.Address = cfg.HTTP.Address()
	serverConfig.ReadHeaderTimeout = cfg.HTTP.ReadHeaderTimeout
	serverConfig.ReadTimeout = cfg.HTTP.ReadTimeout
	serverConfig.WriteTimeout = cfg.HTTP.WriteTimeout
	serverConfig.IdleTimeout = cfg.HTTP.IdleTimeout
	serverConfig.ShutdownTimeout = cfg.HTTP.ShutdownGracePeriod
	serverConfig.ErrorLog = logger.StandardLogger("http-server")

	server, err := httpserver.NewServer(router, serverConfig)
	if err != nil {
		_ = syncLogger(logger)
		return nil, fmt.Errorf("create http server: %w", err)
	}

	return &App{
		logger:  logger,
		handler: router,
		server:  server,
	}, nil
}

// Run serves until the context is canceled or the HTTP server fails.
func (a *App) Run(ctx context.Context) error {
	if a == nil || a.server == nil || a.logger == nil {
		return errors.New("application is not initialized")
	}

	a.logger.Info(
		"service starting",
		logging.Component("http"),
	)
	if err := a.server.Run(ctx); err != nil {
		a.logger.Error(
			"service stopped unexpectedly",
			logging.Component("http"),
			logging.ErrorCode("HTTP_SERVER_FAILED"),
			logging.Cause(err),
		)
		return err
	}
	a.logger.Info(
		"service stopped",
		logging.Component("http"),
	)
	return nil
}

// Handler exposes the composed router for black-box HTTP tests and serverless
// runtime adapters without exposing Gin.
func (a *App) Handler() http.Handler {
	if a == nil {
		return nil
	}
	return a.handler
}

// Close flushes process resources. It is safe to call more than once.
func (a *App) Close() error {
	if a == nil {
		return nil
	}
	var databaseError error
	if a.database != nil {
		databaseError = a.database.Close()
		a.database = nil
	}
	return errors.Join(databaseError, syncLogger(a.logger))
}

func syncLogger(logger *logging.Logger) error {
	if logger == nil {
		return nil
	}
	err := logger.Sync()
	if errors.Is(err, syscall.EINVAL) {
		// Windows and some container stdout implementations do not support
		// fsync. Log entries are already written synchronously by Zap.
		return nil
	}
	return err
}
