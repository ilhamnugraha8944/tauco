package httpserver

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"sync/atomic"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/ilhamnugraha8944/tauco/backend/internal/contract/requestmeta"
)

const (
	// RequestIDHeader is the canonical HTTP header used to propagate request IDs.
	RequestIDHeader = requestmeta.Header

	requestIDGinKey = "tauco.request_id"
)

type requestIDContextKey struct{}

// RequestIDGenerator generates a request ID when a valid upstream ID is absent.
type RequestIDGenerator func() string

// PanicReport contains bounded request metadata for a recovered panic. The
// recovered value is deliberately excluded because it may contain secrets or
// direct PII.
type PanicReport struct {
	Route   string
	Method  string
	Status  int
	Latency time.Duration
}

// PanicReporter reports a recovered panic without coupling this package to a
// specific logging implementation. The request ID is available through
// RequestIDFromContext.
type PanicReporter func(context.Context, PanicReport)

var fallbackRequestIDSequence atomic.Uint64

// RequestIDMiddleware propagates a valid upstream request ID or creates a new
// one. The ID is available in the response header, Gin context, and standard
// request context.
func RequestIDMiddleware(generator RequestIDGenerator) gin.HandlerFunc {
	if generator == nil {
		generator = newRequestID
	}

	return func(c *gin.Context) {
		requestID := c.GetHeader(RequestIDHeader)
		if !isValidRequestID(requestID) {
			requestID = generator()
		}
		if !isValidRequestID(requestID) {
			requestID = newRequestID()
		}

		c.Set(requestIDGinKey, requestID)
		c.Header(RequestIDHeader, requestID)
		c.Request.Header.Set(RequestIDHeader, requestID)

		requestContext := context.WithValue(
			c.Request.Context(),
			requestIDContextKey{},
			requestID,
		)
		c.Request = c.Request.WithContext(requestContext)
		c.Next()
	}
}

// RecoveryMiddleware converts panics into a generic problem response. Panic
// details are never returned to the client.
func RecoveryMiddleware(reporter PanicReporter) gin.HandlerFunc {
	return func(c *gin.Context) {
		startedAt := time.Now()
		defer func() {
			recovered := recover()
			if recovered == nil {
				return
			}

			if !c.Writer.Written() {
				WriteProblem(
					c,
					500,
					"urn:tauco-cap-badak:problem:internal",
					"Terjadi kesalahan internal",
					"Permintaan tidak dapat diproses saat ini.",
					"INTERNAL_SERVER_ERROR",
				)
			}

			c.Abort()

			route := c.FullPath()
			if route == "" {
				route = "unmatched"
			}
			safelyReportPanic(
				reporter,
				c.Request.Context(),
				PanicReport{
					Route:   route,
					Method:  c.Request.Method,
					Status:  c.Writer.Status(),
					Latency: time.Since(startedAt),
				},
			)
		}()

		c.Next()
	}
}

// RequestIDFromContext reads the request ID from a standard request context.
func RequestIDFromContext(ctx context.Context) (string, bool) {
	if ctx == nil {
		return "", false
	}

	requestID, ok := ctx.Value(requestIDContextKey{}).(string)
	return requestID, ok && requestID != ""
}

// RequestIDFromGinContext reads the request ID from a Gin context.
func RequestIDFromGinContext(c *gin.Context) (string, bool) {
	if c == nil {
		return "", false
	}

	requestID, ok := c.Get(requestIDGinKey)
	value, validType := requestID.(string)
	return value, ok && validType && value != ""
}

func newRequestID() string {
	var random [16]byte
	if _, err := rand.Read(random[:]); err == nil {
		return hex.EncodeToString(random[:])
	}

	// crypto/rand failure is exceptionally rare. This fallback still avoids
	// returning an empty tracing identifier.
	return fmt.Sprintf(
		"%x-%x",
		time.Now().UTC().UnixNano(),
		fallbackRequestIDSequence.Add(1),
	)
}

func isValidRequestID(requestID string) bool {
	return requestmeta.Valid(requestID)
}

func safelyReportPanic(
	reporter PanicReporter,
	ctx context.Context,
	report PanicReport,
) {
	if reporter == nil {
		return
	}

	defer func() {
		// A failing observability adapter must not prevent the recovery response.
		_ = recover()
	}()
	reporter(ctx, report)
}
