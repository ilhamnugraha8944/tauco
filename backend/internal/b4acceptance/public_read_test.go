package b4acceptance_test

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"

	"github.com/gin-gonic/gin"
	catalogapp "github.com/ilhamnugraha8944/tauco/backend/internal/catalog/application"
	catalogcursor "github.com/ilhamnugraha8944/tauco/backend/internal/catalog/delivery/cursor"
	catalogdomain "github.com/ilhamnugraha8944/tauco/backend/internal/catalog/domain"
	contentapp "github.com/ilhamnugraha8944/tauco/backend/internal/content/application"
	contentdomain "github.com/ilhamnugraha8944/tauco/backend/internal/content/domain"
	"github.com/ilhamnugraha8944/tauco/backend/internal/content/importer"
	"github.com/ilhamnugraha8944/tauco/backend/internal/contract/requestmeta"
	"github.com/ilhamnugraha8944/tauco/backend/internal/delivery/api"
)

const publicReadTestSecret = "0123456789abcdef0123456789abcdef"

type pageRepository struct {
	pages map[contentdomain.PageKey]contentdomain.PublishedPage
}

func (repository *pageRepository) FindPublishedPage(
	_ context.Context,
	key contentdomain.PageKey,
) (contentdomain.PublishedPage, error) {
	page, found := repository.pages[key]
	if !found {
		return contentdomain.PublishedPage{}, contentapp.ErrPublishedPageNotFound
	}
	return page, nil
}

type productRepository struct {
	products []catalogdomain.PublishedProduct
}

func (repository *productRepository) FindPublishedProduct(
	_ context.Context,
	slug string,
) (catalogdomain.PublishedProduct, error) {
	for _, product := range repository.products {
		if product.Slug == slug {
			return product, nil
		}
	}
	return catalogdomain.PublishedProduct{},
		catalogapp.ErrPublishedProductNotFound
}

func (repository *productRepository) ListPublishedProducts(
	_ context.Context,
	after *catalogdomain.PaginationPosition,
	limit int,
) (catalogapp.PublishedProductPage, error) {
	start := 0
	if after != nil {
		for index, product := range repository.products {
			if product.SortOrder > after.SortOrder ||
				(product.SortOrder == after.SortOrder &&
					string(product.ProductID) > string(after.ProductID)) {
				start = index
				break
			}
			start = len(repository.products)
		}
	}
	end := min(start+limit, len(repository.products))
	return catalogapp.PublishedProductPage{
		Products: append(
			[]catalogdomain.PublishedProduct(nil),
			repository.products[start:end]...,
		),
		HasMore: end < len(repository.products),
	}, nil
}

func TestPublicReadRoutesMatchPhase1AFixtures(t *testing.T) {
	router := newPublicReadRouter(t)
	tests := []struct {
		path    string
		fixture string
	}{
		{path: "/api/v1/home", fixture: "home.success.json"},
		{path: "/api/v1/about", fixture: "about.success.json"},
		{path: "/api/v1/tauco-guide", fixture: "tauco-guide.success.json"},
		{path: "/api/v1/products", fixture: "products-list.success.json"},
		{
			path:    "/api/v1/products/tauco-cap-badak",
			fixture: "product-detail.success.json",
		},
	}

	for _, test := range tests {
		t.Run(test.path, func(t *testing.T) {
			response := request(router, http.MethodGet, test.path, "")
			if response.Code != http.StatusOK {
				t.Fatalf("status = %d, body = %s", response.Code, response.Body)
			}
			if response.Header().Get("ETag") == "" {
				t.Fatal("ETag is empty")
			}
			if response.Header().Get("Cache-Control") !=
				api.PublicContentCacheControl {
				t.Fatalf(
					"Cache-Control = %q",
					response.Header().Get("Cache-Control"),
				)
			}

			var actual map[string]any
			if err := json.Unmarshal(response.Body.Bytes(), &actual); err != nil {
				t.Fatalf("decode response: %v", err)
			}
			expected := readJSONFixture(t, test.fixture)
			if !reflect.DeepEqual(actual["data"], expected["data"]) {
				t.Fatalf("response data differs from %s", test.fixture)
			}
		})
	}
}

func TestPublicReadConditionalAndNegativeResponses(t *testing.T) {
	router := newPublicReadRouter(t)

	first := request(router, http.MethodGet, "/api/v1/home", "")
	etag := first.Header().Get("ETag")
	notModified := request(router, http.MethodGet, "/api/v1/home", etag)
	if notModified.Code != http.StatusNotModified ||
		notModified.Body.Len() != 0 ||
		notModified.Header().Get("ETag") != etag {
		t.Fatalf(
			"conditional response = %d/%q/%q",
			notModified.Code,
			notModified.Body.String(),
			notModified.Header().Get("ETag"),
		)
	}

	tests := []struct {
		name       string
		method     string
		path       string
		wantStatus int
		wantCode   string
	}{
		{
			name:       "unknown product",
			method:     http.MethodGet,
			path:       "/api/v1/products/tidak-ada",
			wantStatus: http.StatusNotFound,
			wantCode:   "PRODUCT_NOT_FOUND",
		},
		{
			name:       "invalid cursor",
			method:     http.MethodGet,
			path:       "/api/v1/products?cursor=invalid",
			wantStatus: http.StatusBadRequest,
			wantCode:   "INVALID_CURSOR",
		},
		{
			name:       "unknown query",
			method:     http.MethodGet,
			path:       "/api/v1/home?preview=true",
			wantStatus: http.StatusBadRequest,
			wantCode:   "BAD_REQUEST",
		},
		{
			name:       "duplicate query",
			method:     http.MethodGet,
			path:       "/api/v1/products?limit=1&limit=2",
			wantStatus: http.StatusBadRequest,
			wantCode:   "BAD_REQUEST",
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			response := request(router, test.method, test.path, "")
			if response.Code != test.wantStatus {
				t.Fatalf(
					"status = %d, want %d; body=%s",
					response.Code,
					test.wantStatus,
					response.Body,
				)
			}
			var body api.Problem
			if err := json.Unmarshal(response.Body.Bytes(), &body); err != nil {
				t.Fatalf("decode problem: %v", err)
			}
			if body.Code != test.wantCode {
				t.Fatalf("problem code = %q, want %q", body.Code, test.wantCode)
			}
			if !strings.HasPrefix(
				response.Header().Get("Content-Type"),
				"application/problem+json",
			) {
				t.Fatalf(
					"Content-Type = %q",
					response.Header().Get("Content-Type"),
				)
			}
		})
	}

	contact := request(
		router,
		http.MethodPost,
		"/api/v1/contact-messages",
		"",
	)
	if contact.Code != http.StatusNotFound {
		t.Fatalf("unregistered contact status = %d", contact.Code)
	}
}

func newPublicReadRouter(t *testing.T) *gin.Engine {
	t.Helper()

	contentDirectory := filepath.Join("..", "..", "..", "content")
	plan, err := importer.LoadPhase1ADirectory(contentDirectory)
	if err != nil {
		t.Fatalf("LoadPhase1ADirectory() error = %v", err)
	}

	pages := make(map[contentdomain.PageKey]contentdomain.PublishedPage)
	for _, seed := range plan.Pages {
		pages[seed.Key] = contentdomain.PublishedPage{
			PageID:         seed.Revision.EntityID,
			RevisionID:     seed.Revision.RevisionID,
			Key:            seed.Key,
			RevisionNumber: seed.Revision.RevisionNumber,
			SchemaVersion:  seed.Revision.SchemaVersion,
			ContentJSON:    seed.Revision.ContentJSON,
			Checksum:       seed.Revision.Checksum,
			PublishedAt:    seed.Revision.PublishedAt,
		}
	}
	products := make([]catalogdomain.PublishedProduct, 0, len(plan.Products))
	for _, seed := range plan.Products {
		products = append(products, catalogdomain.PublishedProduct{
			ProductID:      seed.Revision.EntityID,
			RevisionID:     seed.Revision.RevisionID,
			Slug:           seed.Slug,
			SortOrder:      seed.SortOrder,
			RevisionNumber: seed.Revision.RevisionNumber,
			SchemaVersion:  seed.Revision.SchemaVersion,
			ContentJSON:    seed.Revision.ContentJSON,
			Checksum:       seed.Revision.Checksum,
			PublishedAt:    seed.Revision.PublishedAt,
		})
	}

	pageReader, err := contentapp.NewPublishedReader(
		&pageRepository{pages: pages},
	)
	if err != nil {
		t.Fatalf("NewPublishedReader(page) error = %v", err)
	}
	codec, err := catalogcursor.NewHMACSHA256([]byte(publicReadTestSecret))
	if err != nil {
		t.Fatalf("NewHMACSHA256() error = %v", err)
	}
	productReader, err := catalogapp.NewPublishedReader(
		&productRepository{products: products},
		codec,
	)
	if err != nil {
		t.Fatalf("NewPublishedReader(product) error = %v", err)
	}
	server, err := api.NewPublicReadServer(pageReader, productReader)
	if err != nil {
		t.Fatalf("NewPublicReadServer() error = %v", err)
	}

	gin.SetMode(gin.TestMode)
	router := gin.New()
	router.Use(func(ctx *gin.Context) {
		requestID := ctx.GetHeader(requestmeta.Header)
		if !requestmeta.Valid(requestID) {
			requestID = "b4-test-request"
			ctx.Request.Header.Set(requestmeta.Header, requestID)
		}
		ctx.Header(requestmeta.Header, requestID)
		ctx.Next()
	})
	api.RegisterSafePublicReadHandlers(router, server, nil, "")
	return router
}

func request(
	router http.Handler,
	method,
	target,
	ifNoneMatch string,
) *httptest.ResponseRecorder {
	response := httptest.NewRecorder()
	request := httptest.NewRequest(method, target, nil)
	request.Header.Set(requestmeta.Header, "b4-test-request")
	if ifNoneMatch != "" {
		request.Header.Set("If-None-Match", ifNoneMatch)
	}
	router.ServeHTTP(response, request)
	return response
}

func readJSONFixture(t *testing.T, filename string) map[string]any {
	t.Helper()
	raw, err := os.ReadFile(
		filepath.Join("..", "..", "..", "contracts", "fixtures", filename),
	)
	if err != nil {
		t.Fatalf("read fixture %s: %v", filename, err)
	}
	var value map[string]any
	if err := json.Unmarshal(raw, &value); err != nil {
		t.Fatalf("decode fixture %s: %v", filename, err)
	}
	return value
}
