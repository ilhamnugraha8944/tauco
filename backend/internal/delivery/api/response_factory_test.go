package api

import (
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestGeneratedResponseMetadataFactories(t *testing.T) {
	t.Parallel()

	meta, err := NewResponseMeta("request-id.123")
	if err != nil {
		t.Fatalf("NewResponseMeta() error = %v", err)
	}
	if meta.RequestId != "request-id.123" ||
		meta.ApiVersion != ResponseMetaApiVersionV1 {
		t.Fatalf("response meta = %#v, want canonical v1 metadata", meta)
	}

	cursor := "payload.signature"
	listMeta, err := NewListResponseMeta(
		"request-id.456",
		&cursor,
		true,
		20,
	)
	if err != nil {
		t.Fatalf("NewListResponseMeta() error = %v", err)
	}
	if listMeta.Page.NextCursor == nil ||
		*listMeta.Page.NextCursor != cursor ||
		!listMeta.Page.HasMore ||
		listMeta.Page.Limit != 20 {
		t.Fatalf("list response meta = %#v, want signed next page", listMeta)
	}

	cursor = "mutated.after.factory"
	if *listMeta.Page.NextCursor != "payload.signature" {
		t.Fatal("response factory must retain an owned cursor copy")
	}
}

func TestGeneratedPageMetaRejectsInvalidContractStates(t *testing.T) {
	t.Parallel()

	validCursor := "payload.signature"
	tests := []struct {
		name       string
		nextCursor *string
		hasMore    bool
		limit      int
	}{
		{name: "has more without cursor", hasMore: true, limit: 20},
		{
			name:       "final page with cursor",
			nextCursor: &validCursor,
			limit:      20,
		},
		{
			name:       "malformed cursor",
			nextCursor: pointerToString("not-signed"),
			hasMore:    true,
			limit:      20,
		},
		{
			name:       "padded cursor",
			nextCursor: pointerToString("cGF5bG9hZA==.c2lnbmF0dXJl"),
			hasMore:    true,
			limit:      20,
		},
		{name: "zero limit", limit: 0},
		{name: "over maximum limit", limit: 51},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()

			_, err := NewPageMeta(
				test.nextCursor,
				test.hasMore,
				test.limit,
			)
			if !errors.Is(err, ErrInvalidResponseContract) {
				t.Fatalf(
					"NewPageMeta() error = %v, want %v",
					err,
					ErrInvalidResponseContract,
				)
			}
		})
	}

	if err := ValidatePageMeta(PageMeta{
		HasMore: true,
		Limit:   20,
	}); !errors.Is(err, ErrInvalidResponseContract) {
		t.Fatalf(
			"ValidatePageMeta() error = %v, want %v",
			err,
			ErrInvalidResponseContract,
		)
	}
}

func TestListProductsFactoryUsesActualGeneratedResponseVisitor(t *testing.T) {
	t.Parallel()

	fixtureRaw := readContractFixture(t, "products-list.success.json")
	var fixture ProductListResponse
	if err := json.Unmarshal(fixtureRaw, &fixture); err != nil {
		t.Fatalf("decode product fixture: %v", err)
	}

	response, err := NewListProducts200Response(
		fixture.Data,
		"strict-response-request-id",
		`"sha256-contract-fixture"`,
		nil,
		false,
		20,
	)
	if err != nil {
		t.Fatalf("NewListProducts200Response() error = %v", err)
	}

	recorder := httptest.NewRecorder()
	if err := response.VisitListProductsResponse(recorder); err != nil {
		t.Fatalf("VisitListProductsResponse() error = %v", err)
	}
	if recorder.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d", recorder.Code, http.StatusOK)
	}
	if got := recorder.Header().Get("X-Request-ID"); got != "strict-response-request-id" {
		t.Fatalf("X-Request-ID = %q, want strict-response-request-id", got)
	}
	if got := recorder.Header().Get("Cache-Control"); got != PublicContentCacheControl {
		t.Fatalf("Cache-Control = %q, want %q", got, PublicContentCacheControl)
	}
	if got := recorder.Header().Get("ETag"); got != `"sha256-contract-fixture"` {
		t.Fatalf("ETag = %q, want strong fixture ETag", got)
	}

	var body ProductListResponse
	if err := json.Unmarshal(recorder.Body.Bytes(), &body); err != nil {
		t.Fatalf("decode generated response visitor body: %v", err)
	}
	if err := ValidatePageMeta(body.Meta.Page); err != nil {
		t.Fatalf("generated response visitor emitted invalid page: %v", err)
	}
	if body.Meta.RequestId != "strict-response-request-id" ||
		body.Meta.ApiVersion != ListResponseMetaApiVersionV1 {
		t.Fatalf("response meta = %#v, want canonical v1 metadata", body.Meta)
	}
}

func TestGeneratedResponseFactoriesRejectUnsafeMetadata(t *testing.T) {
	t.Parallel()

	for _, requestID := range []string{
		"",
		"unsafe request id",
		"visitor@example.com",
		strings.Repeat("a", 129),
	} {
		if _, err := NewResponseMeta(requestID); !errors.Is(
			err,
			ErrInvalidResponseContract,
		) {
			t.Fatalf(
				"NewResponseMeta(%q) error = %v, want %v",
				requestID,
				err,
				ErrInvalidResponseContract,
			)
		}
	}

	fixtureRaw := readContractFixture(t, "products-list.success.json")
	var fixture ProductListResponse
	if err := json.Unmarshal(fixtureRaw, &fixture); err != nil {
		t.Fatalf("decode product fixture: %v", err)
	}
	if _, err := NewListProducts200Response(
		fixture.Data,
		"request-id",
		`W/"weak-etag"`,
		nil,
		false,
		20,
	); !errors.Is(err, ErrInvalidResponseContract) {
		t.Fatalf(
			"weak ETag error = %v, want %v",
			err,
			ErrInvalidResponseContract,
		)
	}

	oversizedData := fixture.Data
	oversizedData.Products = make(
		[]ProductSummary,
		21,
	)
	if _, err := NewListProducts200Response(
		oversizedData,
		"request-id",
		`"strong-etag"`,
		nil,
		false,
		20,
	); !errors.Is(err, ErrInvalidResponseContract) {
		t.Fatalf(
			"oversized data error = %v, want %v",
			err,
			ErrInvalidResponseContract,
		)
	}
}

func pointerToString(value string) *string {
	return &value
}
