package api

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strconv"
	"strings"

	"github.com/gin-gonic/gin"
	catalogapp "github.com/ilhamnugraha8944/tauco/backend/internal/catalog/application"
	contactapp "github.com/ilhamnugraha8944/tauco/backend/internal/contact/application"
	contentapp "github.com/ilhamnugraha8944/tauco/backend/internal/content/application"
	contentdomain "github.com/ilhamnugraha8944/tauco/backend/internal/content/domain"
	"github.com/ilhamnugraha8944/tauco/backend/internal/contract/requestmeta"
)

const publicReadRetryAfterSeconds int32 = 5

// PublicReadServer implements only the B4 strict operations. The embedded
// interface satisfies later operations without registering their routes.
type PublicReadServer struct {
	StrictServerInterface
	pages    *contentapp.PublishedReader
	products *catalogapp.PublishedReader
	contacts *contactapp.Intake
}

// WithContactIntake enables the B5 contact operation on the public server.
func (server *PublicReadServer) WithContactIntake(intake *contactapp.Intake) error {
	if server == nil || intake == nil {
		return errors.New("public server requires a contact intake")
	}
	server.contacts = intake
	return nil
}

var _ StrictServerInterface = (*PublicReadServer)(nil)

// NewPublicReadServer creates the generated B4 delivery implementation.
func NewPublicReadServer(
	pages *contentapp.PublishedReader,
	products *catalogapp.PublishedReader,
) (*PublicReadServer, error) {
	if pages == nil || products == nil {
		return nil, errors.New("public read server requires content and catalog readers")
	}
	return &PublicReadServer{pages: pages, products: products}, nil
}

// RegisterSafePublicReadHandlers registers only B4 routes. Contact, readiness,
// and metrics remain unavailable until their own gates.
func RegisterSafePublicReadHandlers(
	router gin.IRouter,
	server StrictServerInterface,
	middlewares []StrictMiddlewareFunc,
	baseURL string,
	routeMiddleware ...gin.HandlerFunc,
) {
	handler := NewSafeStrictHandler(server, middlewares)
	options := NewSafeGinServerOptions(baseURL)
	wrapper := ServerInterfaceWrapper{
		Handler:      handler,
		ErrorHandler: options.ErrorHandler,
	}

	router.GET(
		baseURL+"/api/v1/home",
		appendRouteMiddleware(routeMiddleware, rejectUnknownQuery(), wrapper.GetHome)...,
	)
	router.GET(
		baseURL+"/api/v1/about",
		appendRouteMiddleware(routeMiddleware, rejectUnknownQuery(), wrapper.GetAbout)...,
	)
	router.GET(
		baseURL+"/api/v1/tauco-guide",
		appendRouteMiddleware(routeMiddleware, rejectUnknownQuery(), wrapper.GetTaucoGuide)...,
	)
	router.GET(
		baseURL+"/api/v1/products",
		appendRouteMiddleware(routeMiddleware, rejectUnknownQuery("cursor", "limit"), wrapper.ListProducts)...,
	)
	router.GET(
		baseURL+"/api/v1/products/:slug",
		appendRouteMiddleware(routeMiddleware, rejectUnknownQuery(), wrapper.GetProductBySlug)...,
	)
}

func appendRouteMiddleware(prefix []gin.HandlerFunc, handlers ...gin.HandlerFunc) []gin.HandlerFunc {
	combined := make([]gin.HandlerFunc, 0, len(prefix)+len(handlers))
	combined = append(combined, prefix...)
	return append(combined, handlers...)
}

// GetHome returns the current immutable homepage revision.
func (server *PublicReadServer) GetHome(
	ctx context.Context,
	request GetHomeRequestObject,
) (GetHomeResponseObject, error) {
	requestID := publicRequestID(request.Params.XRequestID)
	page, err := server.pages.Get(ctx, contentdomain.PageKeyHome)
	if err != nil {
		return GetHome503ApplicationProblemPlusJSONResponse{
			serviceUnavailableProblem(requestID, "/api/v1/home"),
		}, nil
	}
	data, err := decodeStrictJSON[HomeContent](page.ContentJSON)
	if err != nil {
		return nil, err
	}
	meta, err := NewResponseMeta(requestID)
	if err != nil {
		return nil, err
	}
	etag := checksumETag(string(page.Checksum))
	if matchesIfNoneMatch(request.Params.IfNoneMatch, etag) {
		return GetHome304Response{Headers: notModifiedHeaders(requestID, etag)}, nil
	}
	return GetHome200JSONResponse{
		Body: HomeResponse{Data: data, Meta: meta},
		Headers: GetHome200ResponseHeaders{
			CacheControl: PublicContentCacheControl,
			ETag:         etag,
			XRequestID:   requestID,
		},
	}, nil
}

// GetAbout returns the current immutable About revision.
func (server *PublicReadServer) GetAbout(
	ctx context.Context,
	request GetAboutRequestObject,
) (GetAboutResponseObject, error) {
	requestID := publicRequestID(request.Params.XRequestID)
	page, err := server.pages.Get(ctx, contentdomain.PageKeyAbout)
	if err != nil {
		return GetAbout503ApplicationProblemPlusJSONResponse{
			serviceUnavailableProblem(requestID, "/api/v1/about"),
		}, nil
	}
	data, err := decodeStrictJSON[AboutContent](page.ContentJSON)
	if err != nil {
		return nil, err
	}
	meta, err := NewResponseMeta(requestID)
	if err != nil {
		return nil, err
	}
	etag := checksumETag(string(page.Checksum))
	if matchesIfNoneMatch(request.Params.IfNoneMatch, etag) {
		return GetAbout304Response{Headers: notModifiedHeaders(requestID, etag)}, nil
	}
	return GetAbout200JSONResponse{
		Body: AboutResponse{Data: data, Meta: meta},
		Headers: GetAbout200ResponseHeaders{
			CacheControl: PublicContentCacheControl,
			ETag:         etag,
			XRequestID:   requestID,
		},
	}, nil
}

// GetTaucoGuide returns the current immutable informational guide.
func (server *PublicReadServer) GetTaucoGuide(
	ctx context.Context,
	request GetTaucoGuideRequestObject,
) (GetTaucoGuideResponseObject, error) {
	requestID := publicRequestID(request.Params.XRequestID)
	page, err := server.pages.Get(ctx, contentdomain.PageKeyTaucoGuide)
	if err != nil {
		return GetTaucoGuide503ApplicationProblemPlusJSONResponse{
			serviceUnavailableProblem(requestID, "/api/v1/tauco-guide"),
		}, nil
	}
	data, err := decodeStrictJSON[TaucoGuideContent](page.ContentJSON)
	if err != nil {
		return nil, err
	}
	meta, err := NewResponseMeta(requestID)
	if err != nil {
		return nil, err
	}
	etag := checksumETag(string(page.Checksum))
	if matchesIfNoneMatch(request.Params.IfNoneMatch, etag) {
		return GetTaucoGuide304Response{
			Headers: notModifiedHeaders(requestID, etag),
		}, nil
	}
	return GetTaucoGuide200JSONResponse{
		Body: TaucoGuideResponse{Data: data, Meta: meta},
		Headers: GetTaucoGuide200ResponseHeaders{
			CacheControl: PublicContentCacheControl,
			ETag:         etag,
			XRequestID:   requestID,
		},
	}, nil
}

// ListProducts returns a stable cursor page of published products.
func (server *PublicReadServer) ListProducts(
	ctx context.Context,
	request ListProductsRequestObject,
) (ListProductsResponseObject, error) {
	requestID := publicRequestID(request.Params.XRequestID)
	limit := optionalLimit(request.Params.Limit)
	result, err := server.products.List(ctx, request.Params.Cursor, limit)
	if errors.Is(err, catalogapp.ErrInvalidCursor) {
		return ListProducts400ApplicationProblemPlusJSONResponse{
			ProductListBadRequestApplicationProblemPlusJSONResponse{
				Body: invalidCursorProblem(requestID),
				Headers: ProductListBadRequestResponseHeaders{
					CacheControl: "no-store",
					XRequestID:   requestID,
				},
			},
		}, nil
	}
	if errors.Is(err, catalogapp.ErrInvalidPageLimit) {
		return ListProducts400ApplicationProblemPlusJSONResponse{
			ProductListBadRequestApplicationProblemPlusJSONResponse{
				Body: badRequestProblem(requestID, "/api/v1/products"),
				Headers: ProductListBadRequestResponseHeaders{
					CacheControl: "no-store",
					XRequestID:   requestID,
				},
			},
		}, nil
	}
	if err != nil {
		return ListProducts503ApplicationProblemPlusJSONResponse{
			serviceUnavailableProblem(requestID, "/api/v1/products"),
		}, nil
	}

	shellPage, err := server.pages.Get(ctx, contentdomain.PageKeyProducts)
	if err != nil {
		return ListProducts503ApplicationProblemPlusJSONResponse{
			serviceUnavailableProblem(requestID, "/api/v1/products"),
		}, nil
	}
	shell, err := decodeStrictJSON[productCatalogShell](shellPage.ContentJSON)
	if err != nil {
		return nil, err
	}

	summaries := make([]ProductSummary, 0, len(result.Products))
	for _, product := range result.Products {
		detail, decodeErr := decodeStrictJSON[ProductDetail](product.ContentJSON)
		if decodeErr != nil {
			return nil, decodeErr
		}
		summaries = append(summaries, productSummary(detail))
	}
	data := ProductCatalogContent{
		Metadata:    shell.Metadata,
		Heading:     shell.Heading,
		Description: shell.Description,
		ContactLink: shell.ContactLink,
		Products:    summaries,
	}
	etag := catalogETag(shellPage, result)
	if matchesIfNoneMatch(request.Params.IfNoneMatch, etag) {
		return ListProducts304Response{
			Headers: notModifiedHeaders(requestID, etag),
		}, nil
	}
	response, err := NewListProducts200Response(
		data,
		requestID,
		etag,
		result.NextCursor,
		result.HasMore,
		result.Limit,
	)
	if err != nil {
		return nil, err
	}
	return response, nil
}

// GetProductBySlug returns one current published product or a true 404.
func (server *PublicReadServer) GetProductBySlug(
	ctx context.Context,
	request GetProductBySlugRequestObject,
) (GetProductBySlugResponseObject, error) {
	requestID := publicRequestID(request.Params.XRequestID)
	product, err := server.products.Get(ctx, request.Slug)
	if errors.Is(err, catalogapp.ErrPublishedProductNotFound) {
		return GetProductBySlug404ApplicationProblemPlusJSONResponse{
			ProductNotFoundApplicationProblemPlusJSONResponse{
				Body: productNotFoundProblem(requestID, request.Slug),
				Headers: ProductNotFoundResponseHeaders{
					CacheControl: "no-store",
					XRequestID:   requestID,
				},
			},
		}, nil
	}
	if err != nil {
		return GetProductBySlug503ApplicationProblemPlusJSONResponse{
			serviceUnavailableProblem(
				requestID,
				"/api/v1/products/"+request.Slug,
			),
		}, nil
	}
	data, err := decodeStrictJSON[ProductDetail](product.ContentJSON)
	if err != nil {
		return nil, err
	}
	meta, err := NewResponseMeta(requestID)
	if err != nil {
		return nil, err
	}
	etag := checksumETag(string(product.Checksum))
	if matchesIfNoneMatch(request.Params.IfNoneMatch, etag) {
		return GetProductBySlug304Response{
			Headers: notModifiedHeaders(requestID, etag),
		}, nil
	}
	return GetProductBySlug200JSONResponse{
		Body: ProductDetailResponse{Data: data, Meta: meta},
		Headers: GetProductBySlug200ResponseHeaders{
			CacheControl: PublicContentCacheControl,
			ETag:         etag,
			XRequestID:   requestID,
		},
	}, nil
}

type productCatalogShell struct {
	Metadata    SeoMetadata  `json:"metadata"`
	Heading     string       `json:"heading"`
	Description string       `json:"description"`
	ContactLink InternalLink `json:"contactLink"`
}

func productSummary(detail ProductDetail) ProductSummary {
	return ProductSummary{
		Slug:     detail.Slug,
		Name:     detail.Name,
		Category: detail.Category,
		Summary:  detail.Summary,
		Image:    detail.Image,
		Facts:    append([]ProductFact(nil), detail.Facts...),
	}
}

func decodeStrictJSON[T any](raw json.RawMessage) (T, error) {
	var value T
	decoder := json.NewDecoder(bytes.NewReader(raw))
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(&value); err != nil {
		return value, fmt.Errorf("decode published API content: %w", err)
	}
	var trailing json.RawMessage
	if err := decoder.Decode(&trailing); !errors.Is(err, io.EOF) {
		return value, errors.New("decode published API content: trailing value")
	}
	return value, nil
}

func optionalLimit(limit *Limit) *int {
	if limit == nil {
		return nil
	}
	value := int(*limit)
	return &value
}

func publicRequestID(candidate *XRequestID) string {
	if candidate != nil && requestmeta.Valid(*candidate) {
		return *candidate
	}
	return "request-id-unavailable"
}

func checksumETag(checksum string) string {
	return `"sha256-` + checksum + `"`
}

func catalogETag(
	shell contentdomain.PublishedPage,
	result catalogapp.PublishedProductList,
) string {
	hash := sha256.New()
	_, _ = io.WriteString(hash, string(shell.Checksum))
	_, _ = io.WriteString(hash, "\n"+strconv.Itoa(result.Limit))
	_, _ = io.WriteString(hash, "\n"+strconv.FormatBool(result.HasMore))
	if result.NextCursor != nil {
		_, _ = io.WriteString(hash, "\n"+*result.NextCursor)
	}
	for _, product := range result.Products {
		_, _ = io.WriteString(hash, "\n"+string(product.Checksum))
	}
	return `"sha256-` + hex.EncodeToString(hash.Sum(nil)) + `"`
}

func matchesIfNoneMatch(candidate *IfNoneMatch, etag string) bool {
	if candidate == nil {
		return false
	}
	for _, item := range strings.Split(*candidate, ",") {
		tag := strings.TrimSpace(item)
		if tag == "*" || tag == etag || strings.TrimPrefix(tag, "W/") == etag {
			return true
		}
	}
	return false
}

func notModifiedHeaders(requestID, etag string) NotModifiedResponseHeaders {
	return NotModifiedResponseHeaders{
		CacheControl: PublicContentCacheControl,
		ETag:         etag,
		XRequestID:   requestID,
	}
}

func problem(
	status int32,
	problemType string,
	title string,
	detail string,
	code string,
	requestID string,
	instance string,
) Problem {
	return Problem{
		Status:    status,
		Type:      problemType,
		Title:     title,
		Detail:    detail,
		Code:      code,
		RequestId: requestID,
		Instance:  instance,
	}
}

func badRequestProblem(requestID, instance string) Problem {
	return problem(
		http.StatusBadRequest,
		"urn:tauco-cap-badak:problem:bad-request",
		"Permintaan tidak valid",
		"Format permintaan tidak dapat diproses.",
		"BAD_REQUEST",
		requestID,
		instance,
	)
}

func invalidCursorProblem(requestID string) Problem {
	return problem(
		http.StatusBadRequest,
		"urn:tauco-cap-badak:problem:invalid-cursor",
		"Cursor tidak valid",
		"Mulai kembali dari halaman pertama.",
		"INVALID_CURSOR",
		requestID,
		"/api/v1/products",
	)
}

func productNotFoundProblem(requestID, slug string) Problem {
	return problem(
		http.StatusNotFound,
		"urn:tauco-cap-badak:problem:product-not-found",
		"Produk tidak ditemukan",
		"Produk yang diminta tidak tersedia.",
		"PRODUCT_NOT_FOUND",
		requestID,
		"/api/v1/products/"+slug,
	)
}

func serviceUnavailableProblem(
	requestID,
	instance string,
) ServiceUnavailableApplicationProblemPlusJSONResponse {
	return ServiceUnavailableApplicationProblemPlusJSONResponse{
		Body: problem(
			http.StatusServiceUnavailable,
			"urn:tauco-cap-badak:problem:service-unavailable",
			"Layanan sementara tidak tersedia",
			"Coba kembali beberapa saat lagi.",
			"SERVICE_UNAVAILABLE",
			requestID,
			instance,
		),
		Headers: ServiceUnavailableResponseHeaders{
			CacheControl: "no-store",
			RetryAfter:   publicReadRetryAfterSeconds,
			XRequestID:   requestID,
		},
	}
}

func rejectUnknownQuery(allowed ...string) gin.HandlerFunc {
	allowlist := make(map[string]struct{}, len(allowed))
	for _, key := range allowed {
		allowlist[key] = struct{}{}
	}
	return func(ctx *gin.Context) {
		for key, values := range ctx.Request.URL.Query() {
			_, accepted := allowlist[key]
			if !accepted || len(values) != 1 {
				writeContractProblem(
					ctx,
					http.StatusBadRequest,
					"urn:tauco-cap-badak:problem:bad-request",
					"Permintaan tidak valid",
					"Format permintaan tidak dapat diproses.",
					"BAD_REQUEST",
				)
				ctx.Abort()
				return
			}
		}
		ctx.Next()
	}
}
