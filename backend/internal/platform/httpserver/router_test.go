package httpserver

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/gin-gonic/gin"
)

func TestRouterLiveness(t *testing.T) {
	gin.SetMode(gin.TestMode)
	router := mustTestRouter(t, RouterOptions{
		RequestIDGenerator: func() string { return "generated-request-id" },
	})

	response := httptest.NewRecorder()
	request := httptest.NewRequest(http.MethodGet, "/health/live", nil)
	router.ServeHTTP(response, request)

	if response.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d", response.Code, http.StatusOK)
	}
	if got := response.Header().Get(RequestIDHeader); got != "generated-request-id" {
		t.Fatalf("%s = %q, want generated request ID", RequestIDHeader, got)
	}
	if got := response.Header().Get("Cache-Control"); got != "no-store" {
		t.Fatalf("Cache-Control = %q, want no-store", got)
	}

	var body livenessResponse
	if err := json.Unmarshal(response.Body.Bytes(), &body); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if body.Status != "ok" {
		t.Fatalf("status body = %q, want ok", body.Status)
	}
}

func TestNewRouterForcesReleaseMode(t *testing.T) {
	gin.SetMode(gin.DebugMode)

	_ = mustTestRouter(t, RouterOptions{})

	if mode := gin.Mode(); mode != gin.ReleaseMode {
		t.Fatalf("Gin mode = %q, want %q", mode, gin.ReleaseMode)
	}
}

func TestNewRouterCanReserveLivenessForGeneratedDelivery(t *testing.T) {
	router := mustTestRouter(t, RouterOptions{
		SkipPlatformLiveness: true,
	})
	router.GET("/health/live", func(c *gin.Context) {
		c.Status(http.StatusNoContent)
	})

	response := httptest.NewRecorder()
	request := httptest.NewRequest(http.MethodGet, "/health/live", nil)
	router.ServeHTTP(response, request)

	if response.Code != http.StatusNoContent {
		t.Fatalf(
			"status = %d, want generated delivery placeholder %d",
			response.Code,
			http.StatusNoContent,
		)
	}
}

func TestRequestIDMiddlewarePropagatesValidUpstreamID(t *testing.T) {
	gin.SetMode(gin.TestMode)
	router := mustTestRouter(t, RouterOptions{
		RequestIDGenerator: func() string { return "generated-request-id" },
	})
	router.GET("/request-id", func(c *gin.Context) {
		fromGin, ginOK := RequestIDFromGinContext(c)
		fromContext, contextOK := RequestIDFromContext(c.Request.Context())
		c.JSON(http.StatusOK, gin.H{
			"fromGin":     fromGin,
			"ginOK":       ginOK,
			"fromContext": fromContext,
			"contextOK":   contextOK,
			"fromHeader":  c.GetHeader(RequestIDHeader),
		})
	})

	response := httptest.NewRecorder()
	request := httptest.NewRequest(http.MethodGet, "/request-id", nil)
	request.Header.Set(RequestIDHeader, "upstream.request-123")
	router.ServeHTTP(response, request)

	if got := response.Header().Get(RequestIDHeader); got != "upstream.request-123" {
		t.Fatalf("%s = %q, want propagated ID", RequestIDHeader, got)
	}

	var body map[string]any
	if err := json.Unmarshal(response.Body.Bytes(), &body); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if got := body["fromGin"]; got != "upstream.request-123" {
		t.Fatalf("Gin request ID = %v, want propagated ID", got)
	}
	if got := body["fromContext"]; got != "upstream.request-123" {
		t.Fatalf("context request ID = %v, want propagated ID", got)
	}
	if got := body["ginOK"]; got != true {
		t.Fatalf("Gin request ID lookup = %v, want true", got)
	}
	if got := body["contextOK"]; got != true {
		t.Fatalf("context request ID lookup = %v, want true", got)
	}
	if got := body["fromHeader"]; got != "upstream.request-123" {
		t.Fatalf("request header ID = %v, want canonical propagated ID", got)
	}
}

func TestRequestIDMiddlewareRejectsUnsafeUpstreamID(t *testing.T) {
	gin.SetMode(gin.TestMode)
	router := mustTestRouter(t, RouterOptions{
		RequestIDGenerator: func() string { return "safe-generated-id" },
	})
	router.GET("/request-id", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{
			"fromHeader": c.GetHeader(RequestIDHeader),
		})
	})

	response := httptest.NewRecorder()
	request := httptest.NewRequest(http.MethodGet, "/request-id", nil)
	request.Header.Set(RequestIDHeader, "unsafe request id")
	router.ServeHTTP(response, request)

	if got := response.Header().Get(RequestIDHeader); got != "safe-generated-id" {
		t.Fatalf("%s = %q, want replacement ID", RequestIDHeader, got)
	}

	var body map[string]any
	if err := json.Unmarshal(response.Body.Bytes(), &body); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if got := body["fromHeader"]; got != "safe-generated-id" {
		t.Fatalf("request header ID = %v, want canonical replacement ID", got)
	}
}

func TestRouterProblemResponses(t *testing.T) {
	gin.SetMode(gin.TestMode)
	router := mustTestRouter(t, RouterOptions{
		RequestIDGenerator: func() string { return "problem-request-id" },
	})

	tests := []struct {
		name       string
		method     string
		target     string
		wantStatus int
		wantCode   string
	}{
		{
			name:       "unknown route",
			method:     http.MethodGet,
			target:     "/does-not-exist",
			wantStatus: http.StatusNotFound,
			wantCode:   "ROUTE_NOT_FOUND",
		},
		{
			name:       "unsupported method",
			method:     http.MethodPost,
			target:     "/health/live",
			wantStatus: http.StatusMethodNotAllowed,
			wantCode:   "METHOD_NOT_ALLOWED",
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			response := httptest.NewRecorder()
			request := httptest.NewRequest(test.method, test.target, nil)
			router.ServeHTTP(response, request)

			if response.Code != test.wantStatus {
				t.Fatalf("status = %d, want %d", response.Code, test.wantStatus)
			}
			if contentType := response.Header().Get("Content-Type"); !strings.HasPrefix(
				contentType,
				ProblemMediaType,
			) {
				t.Fatalf("Content-Type = %q, want %s", contentType, ProblemMediaType)
			}

			var body ProblemResponse
			if err := json.Unmarshal(response.Body.Bytes(), &body); err != nil {
				t.Fatalf("decode response: %v", err)
			}
			if body.Code != test.wantCode {
				t.Fatalf("problem code = %q, want %q", body.Code, test.wantCode)
			}
			if body.Status != test.wantStatus {
				t.Fatalf("problem status = %d, want %d", body.Status, test.wantStatus)
			}
			if body.RequestID != "problem-request-id" {
				t.Fatalf("request ID = %q, want problem-request-id", body.RequestID)
			}
			if body.Instance != test.target {
				t.Fatalf("instance = %q, want %q", body.Instance, test.target)
			}
		})
	}
}

func TestRecoveryMiddlewareReturnsGenericProblemAndReportsPanic(t *testing.T) {
	gin.SetMode(gin.TestMode)

	var (
		reported   PanicReport
		reportedID string
	)
	router := mustTestRouter(t, RouterOptions{
		RequestIDGenerator: func() string { return "panic-request-id" },
		PanicReporter: func(ctx context.Context, report PanicReport) {
			reported = report
			reportedID, _ = RequestIDFromContext(ctx)
		},
	})
	router.GET("/panic", func(*gin.Context) {
		panic("sensitive panic detail")
	})

	response := httptest.NewRecorder()
	request := httptest.NewRequest(http.MethodGet, "/panic", nil)
	router.ServeHTTP(response, request)

	if response.Code != http.StatusInternalServerError {
		t.Fatalf(
			"status = %d, want %d",
			response.Code,
			http.StatusInternalServerError,
		)
	}
	if strings.Contains(response.Body.String(), "sensitive panic detail") {
		t.Fatal("panic detail leaked in response")
	}
	if reportedID != "panic-request-id" {
		t.Fatalf("reported request ID = %q, want panic-request-id", reportedID)
	}
	if reported.Route != "/panic" {
		t.Fatalf("reported route = %q, want /panic", reported.Route)
	}
	if reported.Method != http.MethodGet {
		t.Fatalf("reported method = %q, want GET", reported.Method)
	}
	if reported.Status != http.StatusInternalServerError {
		t.Fatalf(
			"reported status = %d, want %d",
			reported.Status,
			http.StatusInternalServerError,
		)
	}
	if reported.Latency < 0 {
		t.Fatalf("reported latency = %v, want non-negative", reported.Latency)
	}
}

func TestRecoveryMiddlewareSurvivesReporterPanic(t *testing.T) {
	gin.SetMode(gin.TestMode)
	router := mustTestRouter(t, RouterOptions{
		PanicReporter: func(context.Context, PanicReport) {
			panic("reporter failed")
		},
	})
	router.GET("/panic", func(*gin.Context) {
		panic("handler failed")
	})

	response := httptest.NewRecorder()
	request := httptest.NewRequest(http.MethodGet, "/panic", nil)
	router.ServeHTTP(response, request)

	if response.Code != http.StatusInternalServerError {
		t.Fatalf(
			"status = %d, want %d",
			response.Code,
			http.StatusInternalServerError,
		)
	}
}

func TestNewRouterRejectsInvalidTrustedProxy(t *testing.T) {
	gin.SetMode(gin.TestMode)
	_, err := NewRouter(RouterOptions{
		TrustedProxies: []string{"not-a-cidr"},
	})
	if err == nil {
		t.Fatal("NewRouter() error = nil, want invalid proxy error")
	}
}

func mustTestRouter(t *testing.T, options RouterOptions) *gin.Engine {
	t.Helper()

	router, err := NewRouter(options)
	if err != nil {
		t.Fatalf("NewRouter(): %v", err)
	}
	return router
}
