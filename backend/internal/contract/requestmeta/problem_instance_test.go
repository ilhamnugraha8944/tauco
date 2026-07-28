package requestmeta_test

import (
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"

	"github.com/ilhamnugraha8944/tauco/backend/internal/contract/requestmeta"
)

func TestProblemInstancePath(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		request *http.Request
		want    string
	}{
		{name: "nil request", request: nil, want: "/"},
		{
			name:    "uppercase trailing path",
			request: httptest.NewRequest(http.MethodGet, "/API/V1/Missing/", nil),
			want:    "/API/V1/Missing/",
		},
		{
			name:    "percent encoded path",
			request: httptest.NewRequest(http.MethodGet, "/API/V1/Missing%20Product/", nil),
			want:    "/API/V1/Missing%20Product/",
		},
		{
			name: "network path fallback",
			request: &http.Request{
				URL: &url.URL{Path: "//external.example/path"},
			},
			want: "/",
		},
		{
			name: "oversized path fallback",
			request: httptest.NewRequest(
				http.MethodGet,
				"/"+strings.Repeat("a", requestmeta.MaxProblemInstanceLength),
				nil,
			),
			want: "/",
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			if got := requestmeta.ProblemInstancePath(test.request); got != test.want {
				t.Fatalf("ProblemInstancePath() = %q, want %q", got, test.want)
			}
		})
	}
}
