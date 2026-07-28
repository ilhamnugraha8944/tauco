package httpserver

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/ilhamnugraha8944/tauco/backend/internal/contract/requestmeta"
)

const (
	// ProblemMediaType is the RFC 7807-compatible error response media type.
	ProblemMediaType = "application/problem+json"
)

// ProblemFieldError identifies one semantically invalid request field.
type ProblemFieldError struct {
	Field   string `json:"field"`
	Code    string `json:"code"`
	Message string `json:"message"`
}

// ProblemResponse is the stable public error contract.
type ProblemResponse struct {
	Type      string              `json:"type"`
	Title     string              `json:"title"`
	Status    int                 `json:"status"`
	Detail    string              `json:"detail"`
	Instance  string              `json:"instance"`
	Code      string              `json:"code"`
	RequestID string              `json:"requestId"`
	Errors    []ProblemFieldError `json:"errors,omitempty"`
}

// WriteProblem writes an RFC 7807-compatible public error response.
func WriteProblem(
	c *gin.Context,
	status int,
	problemType string,
	title string,
	detail string,
	code string,
	fieldErrors ...ProblemFieldError,
) {
	requestID, _ := RequestIDFromGinContext(c)

	c.Header("Content-Type", ProblemMediaType)
	c.Header("Cache-Control", "no-store")
	c.JSON(status, ProblemResponse{
		Type:      problemType,
		Title:     title,
		Status:    status,
		Detail:    detail,
		Instance:  requestPath(c),
		Code:      code,
		RequestID: requestID,
		Errors:    fieldErrors,
	})
}

func notFoundHandler(c *gin.Context) {
	WriteProblem(
		c,
		http.StatusNotFound,
		"urn:tauco-cap-badak:problem:route-not-found",
		"Route tidak ditemukan",
		"Route yang diminta tidak ditemukan.",
		"ROUTE_NOT_FOUND",
	)
}

func methodNotAllowedHandler(c *gin.Context) {
	WriteProblem(
		c,
		http.StatusMethodNotAllowed,
		"urn:tauco-cap-badak:problem:method-not-allowed",
		"Metode tidak diizinkan",
		"Metode HTTP tidak didukung untuk route ini.",
		"METHOD_NOT_ALLOWED",
	)
}

func requestPath(c *gin.Context) string {
	if c == nil {
		return requestmeta.ProblemInstancePath(nil)
	}

	return requestmeta.ProblemInstancePath(c.Request)
}
