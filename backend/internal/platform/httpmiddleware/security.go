package httpmiddleware

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"math"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/ilhamnugraha8944/tauco/backend/internal/platform/httpserver"
	"github.com/ilhamnugraha8944/tauco/backend/internal/platform/ratelimit"
)

type RatePolicy struct {
	Name   string
	Limit  int
	Window time.Duration
}

func SecurityHeaders() gin.HandlerFunc {
	return func(ctx *gin.Context) {
		ctx.Header("X-Content-Type-Options", "nosniff")
		ctx.Header("X-Frame-Options", "DENY")
		ctx.Header("Referrer-Policy", "no-referrer")
		ctx.Header("Permissions-Policy", "camera=(), microphone=(), geolocation=()")
		ctx.Next()
	}
}

func CORS(origins []string) (gin.HandlerFunc, error) {
	allowlist := make(map[string]struct{}, len(origins))
	for _, origin := range origins {
		origin = strings.TrimSpace(origin)
		parsed, err := url.Parse(origin)
		if err != nil || (parsed.Scheme != "http" && parsed.Scheme != "https") || parsed.Host == "" || parsed.Path != "" || parsed.RawQuery != "" || parsed.Fragment != "" {
			return nil, errors.New("CORS origin must be an exact HTTP(S) origin")
		}
		allowlist[origin] = struct{}{}
	}
	return func(ctx *gin.Context) {
		origin := ctx.GetHeader("Origin")
		if origin == "" {
			ctx.Next()
			return
		}
		if _, allowed := allowlist[origin]; !allowed {
			httpserver.WriteProblem(ctx, http.StatusForbidden,
				"urn:tauco-cap-badak:problem:origin-denied", "Origin tidak diizinkan",
				"Origin permintaan tidak termasuk allowlist.", "ORIGIN_DENIED")
			ctx.Abort()
			return
		}
		ctx.Header("Access-Control-Allow-Origin", origin)
		ctx.Header("Vary", "Origin")
		if ctx.Request.Method == http.MethodOptions {
			method := ctx.GetHeader("Access-Control-Request-Method")
			if method != http.MethodGet && method != http.MethodPost {
				httpserver.WriteProblem(ctx, http.StatusMethodNotAllowed,
					"urn:tauco-cap-badak:problem:method-not-allowed", "Metode tidak didukung",
					"Metode permintaan tidak tersedia.", "METHOD_NOT_ALLOWED")
				ctx.Abort()
				return
			}
			ctx.Header("Access-Control-Allow-Methods", "GET, POST, OPTIONS")
			ctx.Header("Access-Control-Allow-Headers", "Content-Type, Idempotency-Key, X-Request-ID, traceparent")
			ctx.Header("Access-Control-Max-Age", "600")
			ctx.AbortWithStatus(http.StatusNoContent)
			return
		}
		ctx.Next()
	}, nil
}

func RateLimit(limiter *ratelimit.Limiter, secret []byte, policy RatePolicy) (gin.HandlerFunc, error) {
	if limiter == nil || len(secret) < 32 || policy.Name == "" || policy.Limit < 1 || policy.Window <= 0 {
		return nil, errors.New("invalid rate-limit middleware configuration")
	}
	secretCopy := append([]byte(nil), secret...)
	return func(ctx *gin.Context) {
		mac := hmac.New(sha256.New, secretCopy)
		_, _ = mac.Write([]byte(ctx.ClientIP()))
		key := policy.Name + ":" + hex.EncodeToString(mac.Sum(nil))
		allowed, retry := limiter.Allow(ctx.Request.Context(), key, policy.Limit, policy.Window)
		if allowed {
			ctx.Next()
			return
		}
		seconds := max(1, int(math.Ceil(retry.Seconds())))
		ctx.Header("Retry-After", strconv.Itoa(seconds))
		httpserver.WriteProblem(ctx, http.StatusTooManyRequests,
			"urn:tauco-cap-badak:problem:rate-limited", "Terlalu banyak permintaan",
			"Coba kembali setelah waktu tunggu berakhir.", "RATE_LIMITED")
		ctx.Abort()
	}, nil
}

func RejectBody() gin.HandlerFunc {
	return func(ctx *gin.Context) {
		if ctx.Request.ContentLength > 0 || len(ctx.Request.TransferEncoding) > 0 {
			httpserver.WriteProblem(ctx, http.StatusBadRequest,
				"urn:tauco-cap-badak:problem:bad-request", "Permintaan tidak valid",
				"Endpoint ini tidak menerima request body.", "BAD_REQUEST")
			ctx.Abort()
			return
		}
		ctx.Next()
	}
}
