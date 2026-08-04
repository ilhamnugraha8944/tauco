package composition

import (
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/ilhamnugraha8944/tauco/backend/internal/platform/config"
	"github.com/ilhamnugraha8944/tauco/backend/internal/platform/database"
)

func TestPublicAPIWithPostgresAndRedis(t *testing.T) {
	if strings.TrimSpace(os.Getenv("DATABASE_URL")) == "" || strings.TrimSpace(os.Getenv("REDIS_URL")) == "" {
		t.Skip("set DATABASE_URL and REDIS_URL to run composed API integration")
	}
	databaseConfig, err := database.LoadRuntimeConfig(os.LookupEnv)
	if err != nil {
		t.Fatal(err)
	}
	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()
	rateSecret := []byte(fmt.Sprintf("integration-rate-secret-%020d", time.Now().UnixNano()))
	app, err := NewPublicAPI(ctx, testPublicAPIConfig(), databaseConfig, PublicAPISecrets{
		CursorHMAC: []byte("integration-cursor-secret-1234567890"), ContactHMAC: []byte("integration-contact-secret-12345678"), RateHMAC: rateSecret,
		MetricsBearer: []byte("integration-metrics-token-1234567890"),
	}, PublicAPIInfrastructure{
		RedisURL: os.Getenv("REDIS_URL"), CORSOrigins: []string{"http://localhost:3000"}, MediaRoot: t.TempDir(),
	})
	if err != nil {
		t.Fatalf("NewPublicAPI() error = %v", err)
	}
	defer app.Close()

	request := httptest.NewRequest(http.MethodGet, "/api/v1/home", nil)
	request.RemoteAddr = "192.0.2.10:1234"
	request.Header.Set("Origin", "http://localhost:3000")
	response := httptest.NewRecorder()
	app.Handler().ServeHTTP(response, request)
	if response.Code != http.StatusOK || response.Header().Get("Access-Control-Allow-Origin") == "" || response.Header().Get("X-Content-Type-Options") != "nosniff" {
		t.Fatalf("home status=%d headers=%v", response.Code, response.Header())
	}

	request = httptest.NewRequest(http.MethodGet, "/health/ready", nil)
	response = httptest.NewRecorder()
	app.Handler().ServeHTTP(response, request)
	if response.Code != http.StatusOK || !strings.Contains(response.Body.String(), `"postgres":"healthy"`) {
		t.Fatalf("readiness status=%d body=%s", response.Code, response.Body.String())
	}

	request = httptest.NewRequest(http.MethodGet, "/internal/metrics", nil)
	response = httptest.NewRecorder()
	app.Handler().ServeHTTP(response, request)
	if response.Code != http.StatusUnauthorized {
		t.Fatalf("unauthenticated metrics status=%d", response.Code)
	}
	request = httptest.NewRequest(http.MethodGet, "/internal/metrics", nil)
	request.Header.Set("Authorization", "Bearer integration-metrics-token-1234567890")
	request.Header.Set("traceparent", "00-4bf92f3577b34da6a3ce929d0e0e4736-00f067aa0ba902b7-01")
	response = httptest.NewRecorder()
	app.Handler().ServeHTTP(response, request)
	if response.Code != http.StatusOK || !strings.Contains(response.Body.String(), "tauco_db_pool_open_connections") || response.Header().Get("traceparent") == "" {
		t.Fatalf("metrics status=%d headers=%v body=%s", response.Code, response.Header(), response.Body.String())
	}

	request = httptest.NewRequest(http.MethodGet, "/api/v1/home", nil)
	request.RemoteAddr = "192.0.2.10:1234"
	request.Header.Set("Origin", "https://attacker.example")
	response = httptest.NewRecorder()
	app.Handler().ServeHTTP(response, request)
	if response.Code != http.StatusForbidden {
		t.Fatalf("denied origin status=%d", response.Code)
	}

	request = httptest.NewRequest(http.MethodGet, "/api/v1/home", strings.NewReader("body"))
	request.RemoteAddr = "192.0.2.10:1234"
	response = httptest.NewRecorder()
	app.Handler().ServeHTTP(response, request)
	if response.Code != http.StatusBadRequest {
		t.Fatalf("GET body status=%d", response.Code)
	}

	for attempt := 1; attempt <= 6; attempt++ {
		request = httptest.NewRequest(http.MethodPost, "/api/v1/contact-messages", strings.NewReader("invalid"))
		request.RemoteAddr = "192.0.2.10:1234"
		request.Header.Set("Content-Type", "text/plain")
		response = httptest.NewRecorder()
		app.Handler().ServeHTTP(response, request)
		want := http.StatusUnsupportedMediaType
		if attempt == 6 {
			want = http.StatusTooManyRequests
		}
		if response.Code != want {
			t.Fatalf("contact attempt %d status=%d want=%d", attempt, response.Code, want)
		}
	}
}

func testPublicAPIConfig() config.Config {
	return config.Config{
		Environment: config.EnvironmentTest,
		HTTP:        config.HTTPConfig{Host: "127.0.0.1", Port: 8080, ReadHeaderTimeout: time.Second, ReadTimeout: 2 * time.Second, WriteTimeout: 2 * time.Second, IdleTimeout: 5 * time.Second, ShutdownGracePeriod: time.Second},
		Log:         config.LogConfig{Level: "error", Format: "json", Service: "tauco-api"},
	}
}
