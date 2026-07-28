package httpserver

import (
	"fmt"
	"net/http"

	"github.com/gin-gonic/gin"
)

// RouterOptions contains infrastructure-only router dependencies. Business
// routes are registered by delivery packages after this router is constructed.
type RouterOptions struct {
	TrustedProxies     []string
	RequestIDGenerator RequestIDGenerator
	PanicReporter      PanicReporter
	Middleware         []gin.HandlerFunc
	// SkipPlatformLiveness reserves /health/live for generated API delivery.
	// It must be true before B4 registers the OpenAPI server on this router.
	SkipPlatformLiveness bool
}

// NewRouter creates a secure-by-default Gin engine without Gin's logging or
// recovery middleware. Structured logging can be supplied through Middleware.
func NewRouter(options RouterOptions) (*gin.Engine, error) {
	// Gin's debug writer bypasses the structured application logger. Keep the
	// engine in release mode in every environment; LOG_LEVEL controls local
	// verbosity without introducing a second log format.
	gin.SetMode(gin.ReleaseMode)
	router := gin.New()
	router.HandleMethodNotAllowed = true

	// Gin otherwise trusts all proxies by default. A nil or empty allowlist
	// intentionally means no trusted proxy.
	trustedProxies := append([]string(nil), options.TrustedProxies...)
	if err := router.SetTrustedProxies(trustedProxies); err != nil {
		return nil, fmt.Errorf("configure trusted proxies: %w", err)
	}

	router.Use(
		RequestIDMiddleware(options.RequestIDGenerator),
		RecoveryMiddleware(options.PanicReporter),
	)
	router.Use(options.Middleware...)

	if !options.SkipPlatformLiveness {
		router.GET("/health/live", livenessHandler)
	}
	router.NoRoute(notFoundHandler)
	router.NoMethod(methodNotAllowedHandler)

	return router, nil
}

type livenessResponse struct {
	Status string `json:"status"`
}

func livenessHandler(c *gin.Context) {
	c.Header("Cache-Control", "no-store")
	c.JSON(http.StatusOK, livenessResponse{Status: "ok"})
}
