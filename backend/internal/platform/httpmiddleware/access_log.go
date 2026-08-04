// Package httpmiddleware contains transport middleware that composes platform
// concerns without coupling the reusable HTTP server to a concrete logger.
package httpmiddleware

import (
	"time"

	"github.com/gin-gonic/gin"
	"github.com/ilhamnugraha8944/tauco/backend/internal/platform/httpserver"
	"github.com/ilhamnugraha8944/tauco/backend/internal/platform/logging"
)

// AccessLog records one bounded, PII-free event for each completed request.
// It logs the registered route template rather than the raw URL or query.
func AccessLog(logger *logging.Logger) gin.HandlerFunc {
	return func(c *gin.Context) {
		startedAt := time.Now()
		c.Next()

		route := c.FullPath()
		if route == "" {
			route = "unmatched"
		}
		requestID, _ := httpserver.RequestIDFromGinContext(c)
		traceID, hasTrace := httpserver.TraceIDFromGinContext(c)

		fields := []logging.Field{
			logging.RequestID(requestID),
			logging.Route(route),
			logging.Method(c.Request.Method),
			logging.Status(c.Writer.Status()),
			logging.Latency(time.Since(startedAt)),
		}
		if hasTrace {
			fields = append(fields, logging.TraceID(traceID))
		}

		switch status := c.Writer.Status(); {
		case status >= 500:
			logger.Error("http request completed", fields...)
		case status >= 400:
			logger.Warn("http request completed", fields...)
		default:
			logger.Info("http request completed", fields...)
		}
	}
}
