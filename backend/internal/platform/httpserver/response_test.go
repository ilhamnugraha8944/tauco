package httpserver

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/ilhamnugraha8944/tauco/backend/internal/contract/requestmeta"
)

func TestWriteProblemIncludesFieldErrorsAndNoStore(t *testing.T) {
	router := mustTestRouter(t, RouterOptions{
		RequestIDGenerator: func() string { return "problem-request-id" },
	})
	router.POST("/validation", func(c *gin.Context) {
		WriteProblem(
			c,
			http.StatusUnprocessableEntity,
			"urn:tauco-cap-badak:problem:validation",
			"Permintaan tidak valid",
			"Periksa field yang ditandai.",
			"CONTACT_VALIDATION_FAILED",
			ProblemFieldError{
				Field:   "email",
				Code:    "INVALID_EMAIL",
				Message: "Masukkan alamat email yang valid.",
			},
		)
	})

	response := httptest.NewRecorder()
	request := httptest.NewRequest(http.MethodPost, "/validation", nil)
	router.ServeHTTP(response, request)

	if response.Code != http.StatusUnprocessableEntity {
		t.Fatalf(
			"status = %d, want %d",
			response.Code,
			http.StatusUnprocessableEntity,
		)
	}
	if got := response.Header().Get("Cache-Control"); got != "no-store" {
		t.Fatalf("Cache-Control = %q, want no-store", got)
	}
	if got := response.Header().Get("Content-Type"); got != ProblemMediaType {
		t.Fatalf("Content-Type = %q, want %q", got, ProblemMediaType)
	}

	var body ProblemResponse
	if err := json.Unmarshal(response.Body.Bytes(), &body); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if len(body.Errors) != 1 || body.Errors[0].Field != "email" {
		t.Fatalf("errors = %#v, want email field error", body.Errors)
	}
	if body.RequestID != "problem-request-id" {
		t.Fatalf("requestId = %q, want problem-request-id", body.RequestID)
	}
}

func TestProblemResponseUsesBoundedEscapedInstancePath(t *testing.T) {
	router := mustTestRouter(t, RouterOptions{
		RequestIDGenerator: func() string { return "problem-instance-test" },
	})

	for _, target := range []string{
		"/API/V1/Missing",
		"/api/v1/missing/",
		"/API/V1/Missing%20Product/",
	} {
		t.Run(target, func(t *testing.T) {
			request := httptest.NewRequest(http.MethodGet, target, nil)
			response := httptest.NewRecorder()
			router.ServeHTTP(response, request)

			var body ProblemResponse
			if err := json.Unmarshal(response.Body.Bytes(), &body); err != nil {
				t.Fatalf("decode response: %v", err)
			}
			if want := requestmeta.ProblemInstancePath(request); body.Instance != want {
				t.Fatalf("instance = %q, want %q", body.Instance, want)
			}
		})
	}
}
