package httpserver

import (
	"context"
	"encoding/hex"
	"strings"

	"github.com/gin-gonic/gin"
)

const TraceparentHeader = "traceparent"

const traceIDGinKey = "tauco.trace_id"

type traceIDContextKey struct{}

// TraceContextMiddleware accepts and propagates a valid W3C traceparent. It
// deliberately does not create spans while no tracing provider is configured.
func TraceContextMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		traceparent := strings.ToLower(strings.TrimSpace(c.GetHeader(TraceparentHeader)))
		traceID, valid := parseTraceparent(traceparent)
		if valid {
			c.Set(traceIDGinKey, traceID)
			c.Header(TraceparentHeader, traceparent)
			c.Request = c.Request.WithContext(context.WithValue(
				c.Request.Context(), traceIDContextKey{}, traceID,
			))
		}
		c.Next()
	}
}

func TraceIDFromContext(ctx context.Context) (string, bool) {
	if ctx == nil {
		return "", false
	}
	traceID, ok := ctx.Value(traceIDContextKey{}).(string)
	return traceID, ok && traceID != ""
}

func TraceIDFromGinContext(c *gin.Context) (string, bool) {
	if c == nil {
		return "", false
	}
	value, ok := c.Get(traceIDGinKey)
	traceID, valid := value.(string)
	return traceID, ok && valid && traceID != ""
}

func parseTraceparent(value string) (string, bool) {
	parts := strings.Split(value, "-")
	if len(parts) != 4 || parts[0] != "00" || len(parts[1]) != 32 ||
		len(parts[2]) != 16 || len(parts[3]) != 2 ||
		allZero(parts[1]) || allZero(parts[2]) {
		return "", false
	}
	for _, part := range parts {
		if _, err := hex.DecodeString(part); err != nil {
			return "", false
		}
	}
	return parts[1], true
}

func allZero(value string) bool {
	return strings.Trim(value, "0") == ""
}
