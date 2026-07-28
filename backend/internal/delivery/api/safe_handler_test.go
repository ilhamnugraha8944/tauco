package api

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/ilhamnugraha8944/tauco/backend/internal/contract/requestmeta"
)

func TestSafeGeneratedHandlersReturnProblemDetailsWithoutErrorLeak(t *testing.T) {
	gin.SetMode(gin.TestMode)

	tests := []struct {
		name       string
		server     strictServerStub
		target     string
		wantStatus int
	}{
		{
			name:       "transport binder error",
			server:     strictServerStub{},
			target:     "/api/v1/products?limit=not-an-integer",
			wantStatus: http.StatusBadRequest,
		},
		{
			name: "application handler error",
			server: strictServerStub{
				homeError: errors.New("database password=sensitive"),
			},
			target:     "/api/v1/home",
			wantStatus: http.StatusInternalServerError,
		},
		{
			name:       "nil response without error",
			server:     strictServerStub{},
			target:     "/api/v1/home",
			wantStatus: http.StatusInternalServerError,
		},
		{
			name: "response serialization error",
			server: strictServerStub{
				homeResponse: failingHomeResponse{},
			},
			target:     "/api/v1/home",
			wantStatus: http.StatusInternalServerError,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			router := gin.New()
			RegisterSafeHandlers(
				router,
				test.server,
				nil,
				"",
			)

			response := httptest.NewRecorder()
			request := httptest.NewRequest(http.MethodGet, test.target, nil)
			request.Header.Set(requestmeta.Header, "safe-generated-request-id")
			router.ServeHTTP(response, request)

			if response.Code != test.wantStatus {
				t.Fatalf(
					"status = %d, want %d",
					response.Code,
					test.wantStatus,
				)
			}
			if got := response.Header().Get("Content-Type"); got != problemMediaType {
				t.Fatalf(
					"Content-Type = %q, want %q",
					got,
					problemMediaType,
				)
			}
			if got := response.Header().Get(requestmeta.Header); got !=
				"safe-generated-request-id" {
				t.Fatalf(
					"X-Request-ID = %q, want canonical request ID",
					got,
				)
			}
			body := response.Body.String()
			for _, leaked := range []string{
				"not-an-integer",
				"database password",
				errNilStrictHandlerResponse.Error(),
				"sensitive serialization detail",
				`"msg"`,
			} {
				if strings.Contains(body, leaked) {
					t.Fatalf("problem response leaked %q: %s", leaked, body)
				}
			}
			if !strings.Contains(body, `"requestId":"safe-generated-request-id"`) {
				t.Fatalf("problem response is missing request ID: %s", body)
			}
		})
	}
}

type strictServerStub struct {
	StrictServerInterface
	homeError    error
	homeResponse GetHomeResponseObject
}

func (server strictServerStub) GetHome(
	context.Context,
	GetHomeRequestObject,
) (GetHomeResponseObject, error) {
	return server.homeResponse, server.homeError
}

type failingHomeResponse struct{}

func (failingHomeResponse) VisitGetHomeResponse(http.ResponseWriter) error {
	return errors.New("sensitive serialization detail")
}
