package httpmiddleware

import (
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/ilhamnugraha8944/tauco/backend/internal/platform/config"
	"github.com/ilhamnugraha8944/tauco/backend/internal/platform/httpserver"
	"github.com/ilhamnugraha8944/tauco/backend/internal/platform/logging"
)

func TestAccessLogDoesNotExposeRawURL(t *testing.T) {
	gin.SetMode(gin.TestMode)

	reader, writer, err := os.Pipe()
	if err != nil {
		t.Fatalf("create log pipe: %v", err)
	}
	t.Cleanup(func() {
		_ = reader.Close()
		_ = writer.Close()
	})

	originalStdout := os.Stdout
	os.Stdout = writer
	logger, loggerErr := logging.New(config.LogConfig{
		Level:   "info",
		Format:  "json",
		Service: "tauco-api",
	}, config.EnvironmentTest)
	os.Stdout = originalStdout
	if loggerErr != nil {
		t.Fatalf("create logger: %v", loggerErr)
	}

	router, err := httpserver.NewRouter(httpserver.RouterOptions{
		RequestIDGenerator: func() string { return "request-id" },
		Middleware:         []gin.HandlerFunc{AccessLog(logger)},
	})
	if err != nil {
		t.Fatalf("create router: %v", err)
	}
	router.GET("/test/:id", func(c *gin.Context) {
		c.Status(http.StatusNoContent)
	})

	response := httptest.NewRecorder()
	request := httptest.NewRequest(
		http.MethodGet,
		"/test/sensitive-value?email=person@example.com",
		nil,
	)
	router.ServeHTTP(response, request)

	if response.Code != http.StatusNoContent {
		t.Fatalf("status = %d, want %d", response.Code, http.StatusNoContent)
	}

	if err := writer.Close(); err != nil {
		t.Fatalf("close log writer: %v", err)
	}
	output, err := io.ReadAll(reader)
	if err != nil {
		t.Fatalf("read access log: %v", err)
	}
	serialized := string(output)

	for _, expected := range []string{
		`"route":"/test/:id"`,
		`"method":"GET"`,
		`"status":204`,
		`"request_id":"request-id"`,
	} {
		if !strings.Contains(serialized, expected) {
			t.Errorf("access log does not contain %q: %s", expected, serialized)
		}
	}
	for _, forbidden := range []string{"sensitive-value", "person@example.com"} {
		if strings.Contains(serialized, forbidden) {
			t.Errorf("access log contains forbidden value %q: %s", forbidden, serialized)
		}
	}
}
