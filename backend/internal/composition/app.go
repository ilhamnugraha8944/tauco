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
	"time"

	"github.com/gin-gonic/gin"
	catalogapp "github.com/ilhamnugraha8944/tauco/backend/internal/catalog/application"
	"github.com/ilhamnugraha8944/tauco/backend/internal/catalog/delivery/cursor"
	catalogrepo "github.com/ilhamnugraha8944/tauco/backend/internal/catalog/repository"
	contactapp "github.com/ilhamnugraha8944/tauco/backend/internal/contact/application"
	contactrepo "github.com/ilhamnugraha8944/tauco/backend/internal/contact/repository"
	contentapp "github.com/ilhamnugraha8944/tauco/backend/internal/content/application"
	contentrepo "github.com/ilhamnugraha8944/tauco/backend/internal/content/repository"
	"github.com/ilhamnugraha8944/tauco/backend/internal/delivery/api"
	platformcache "github.com/ilhamnugraha8944/tauco/backend/internal/platform/cache"
	"github.com/ilhamnugraha8944/tauco/backend/internal/platform/config"
	"github.com/ilhamnugraha8944/tauco/backend/internal/platform/database"
	"github.com/ilhamnugraha8944/tauco/backend/internal/platform/httpmiddleware"
	"github.com/ilhamnugraha8944/tauco/backend/internal/platform/httpserver"
	"github.com/ilhamnugraha8944/tauco/backend/internal/platform/logging"
	"github.com/ilhamnugraha8944/tauco/backend/internal/platform/observability"
	"github.com/ilhamnugraha8944/tauco/backend/internal/platform/ratelimit"
)

// App owns the API process dependencies and their lifecycle.
type App struct {
	logger   *logging.Logger
	handler  http.Handler
	server   *httpserver.Server
	database *sql.DB
	cache    *platformcache.Redis
}

// New validates configuration and wires the B1 platform foundation.
func New(cfg config.Config) (*App, error) {
	return newApp(cfg, routerSetup{}, nil)
}

// NewPublicAPI wires the B4 public read API to least-privilege PostgreSQL.
type PublicAPISecrets struct {
	CursorHMAC    []byte
	ContactHMAC   []byte
	RateHMAC      []byte
	MetricsBearer []byte
}

type PublicAPIInfrastructure struct {
	RedisURL          string
	CORSOrigins       []string
	TrustedProxyCIDRs []string
	MediaRoot         string
}

func NewPublicAPI(
	ctx context.Context,
	cfg config.Config,
	databaseConfig database.RuntimeConfig,
	secrets PublicAPISecrets,
	infrastructure PublicAPIInfrastructure,
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

	redisStore, err := platformcache.NewRedis(infrastructure.RedisURL)
	if err != nil {
		closeDatabase()
		return nil, err
	}
	closeAll := func() {
		_ = redisStore.Close()
		closeDatabase()
	}

	metrics := observability.NewRegistry()
	observedCache := platformcache.Observe(redisStore, metrics)
	pagePostgres, err := contentrepo.NewPostgresRepository(gormDatabase)
	if err != nil {
		closeAll()
		return nil, err
	}
	productPostgres, err := catalogrepo.NewPostgresRepository(gormDatabase)
	if err != nil {
		closeAll()
		return nil, err
	}
	pageRepository := contentrepo.NewCached(pagePostgres, observedCache, 5*time.Minute)
	productRepository := catalogrepo.NewCached(productPostgres, observedCache, 5*time.Minute)
	pageReader, err := contentapp.NewPublishedReader(pageRepository)
	if err != nil {
		closeAll()
		return nil, err
	}
	cursorCodec, err := cursor.NewHMACSHA256(secrets.CursorHMAC)
	if err != nil {
		closeAll()
		return nil, err
	}
	productReader, err := catalogapp.NewPublishedReader(
		productRepository,
		cursorCodec,
	)
	if err != nil {
		closeAll()
		return nil, err
	}
	contactStore, err := contactrepo.NewPostgresStore(gormDatabase)
	if err != nil {
		closeAll()
		return nil, err
	}
	contactIntake, err := contactapp.NewIntake(contactStore, secrets.ContactHMAC)
	if err != nil {
		closeAll()
		return nil, err
	}
	publicReadServer, err := api.NewPublicReadServer(pageReader, productReader)
	if err != nil {
		closeAll()
		return nil, err
	}
	if err := publicReadServer.WithContactIntake(contactIntake); err != nil {
		closeAll()
		return nil, err
	}
	localLimiter, _ := ratelimit.NewLocal(10_000)
	limiter, _ := ratelimit.New(redisStore, localLimiter, metrics.ObserveRateLimit)
	publicLimit, err := httpmiddleware.RateLimit(limiter, secrets.RateHMAC, httpmiddleware.RatePolicy{
		Name: "public-read", Limit: 60, Window: time.Minute,
	})
	if err != nil {
		closeAll()
		return nil, err
	}
	contactLimit, err := httpmiddleware.RateLimit(limiter, secrets.RateHMAC, httpmiddleware.RatePolicy{
		Name: "contact", Limit: 5, Window: time.Hour,
	})
	if err != nil {
		closeAll()
		return nil, err
	}
	corsMiddleware, err := httpmiddleware.CORS(infrastructure.CORSOrigins)
	if err != nil {
		closeAll()
		return nil, err
	}

	var registrationError error
	app, err := newApp(cfg, routerSetup{
		trustedProxies: infrastructure.TrustedProxyCIDRs,
		middleware:     []gin.HandlerFunc{metrics.HTTPMiddleware(), httpmiddleware.SecurityHeaders(), corsMiddleware},
	}, func(router gin.IRouter) {
		api.RegisterSafePublicReadHandlers(
			router,
			publicReadServer,
			nil,
			"",
			publicLimit,
			httpmiddleware.RejectBody(),
		)
		api.RegisterSafeContactHandler(router, publicReadServer, nil, "", contactLimit)
		registrationError = registerOperations(router, operationsDependencies{
			Database: sqlDatabase, Cache: redisStore,
			MediaRoot:    infrastructure.MediaRoot,
			MetricsToken: secrets.MetricsBearer, Metrics: metrics,
		})
	})
	if err != nil {
		closeAll()
		return nil, err
	}
	if registrationError != nil {
		_ = app.Close()
		closeAll()
		return nil, registrationError
	}
	app.database = sqlDatabase
	app.cache = redisStore
	return app, nil
}

type routerSetup struct {
	trustedProxies []string
	middleware     []gin.HandlerFunc
}

func newApp(
	cfg config.Config,
	setup routerSetup,
	registerRoutes func(gin.IRouter),
) (*App, error) {
	if err := cfg.Validate(); err != nil {
		return nil, fmt.Errorf("validate application config: %w", err)
	}

	logger, err := logging.New(cfg.Log, cfg.Environment)
	if err != nil {
		return nil, fmt.Errorf("create application logger: %w", err)
	}

	middleware := []gin.HandlerFunc{httpmiddleware.AccessLog(logger)}
	middleware = append(middleware, setup.middleware...)
	router, err := httpserver.NewRouter(httpserver.RouterOptions{
		TrustedProxies: append([]string(nil), setup.trustedProxies...),
		PanicReporter: func(ctx context.Context, report httpserver.PanicReport) {
			requestID, _ := httpserver.RequestIDFromContext(ctx)
			fields := []logging.Field{
				logging.RequestID(requestID),
				logging.Route(report.Route),
				logging.Method(report.Method),
				logging.Status(report.Status),
				logging.Latency(report.Latency),
				logging.ErrorCode("HTTP_PANIC_RECOVERED"),
			}
			if traceID, ok := httpserver.TraceIDFromContext(ctx); ok {
				fields = append(fields, logging.TraceID(traceID))
			}
			logger.Error(
				"http panic recovered",
				fields...,
			)
		},
		Middleware: middleware,
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
	var cacheError error
	if a.cache != nil {
		cacheError = a.cache.Close()
		a.cache = nil
	}
	return errors.Join(databaseError, cacheError, syncLogger(a.logger))
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
