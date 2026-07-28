package api

import (
	"errors"
	"strings"

	"github.com/ilhamnugraha8944/tauco/backend/internal/catalog/application"
	"github.com/ilhamnugraha8944/tauco/backend/internal/contract/requestmeta"
)

const (
	// PublicContentCacheControl is the frozen Phase 1B shared-cache policy.
	PublicContentCacheControl = "public, max-age=0, s-maxage=300, stale-while-revalidate=60"

	maxCursorBytes = 1024
)

// ErrInvalidResponseContract indicates that a generated response would violate
// the public v1 contract.
var ErrInvalidResponseContract = errors.New("invalid API response contract")

// NewResponseMeta creates metadata for a non-list success envelope.
func NewResponseMeta(requestID string) (ResponseMeta, error) {
	if !requestmeta.Valid(requestID) {
		return ResponseMeta{}, ErrInvalidResponseContract
	}
	return ResponseMeta{
		ApiVersion: ResponseMetaApiVersionV1,
		RequestId:  requestID,
	}, nil
}

// NewPageMeta creates generated page metadata while enforcing the invariant
// that nextCursor is non-nil exactly when hasMore is true.
func NewPageMeta(
	nextCursor *string,
	hasMore bool,
	limit int,
) (PageMeta, error) {
	page := PageMeta{
		NextCursor: cloneStringPointer(nextCursor),
		HasMore:    hasMore,
		Limit:      int32(limit),
	}
	if err := ValidatePageMeta(page); err != nil {
		return PageMeta{}, err
	}
	return page, nil
}

// ValidatePageMeta protects paths that receive an already-built generated DTO.
func ValidatePageMeta(page PageMeta) error {
	if page.Limit < 1 || page.Limit > application.MaxPageLimit {
		return ErrInvalidResponseContract
	}
	if page.HasMore {
		if page.NextCursor == nil || !isValidOpaqueCursor(*page.NextCursor) {
			return ErrInvalidResponseContract
		}
		return nil
	}
	if page.NextCursor != nil {
		return ErrInvalidResponseContract
	}
	return nil
}

// NewListResponseMeta creates metadata for a cursor-paginated success envelope.
func NewListResponseMeta(
	requestID string,
	nextCursor *string,
	hasMore bool,
	limit int,
) (ListResponseMeta, error) {
	if !requestmeta.Valid(requestID) {
		return ListResponseMeta{}, ErrInvalidResponseContract
	}
	page, err := NewPageMeta(nextCursor, hasMore, limit)
	if err != nil {
		return ListResponseMeta{}, err
	}
	return ListResponseMeta{
		ApiVersion: ListResponseMetaApiVersionV1,
		Page:       page,
		RequestId:  requestID,
	}, nil
}

// NewListProducts200Response is the required construction path for the strict
// list-products success visitor.
func NewListProducts200Response(
	data ProductCatalogContent,
	requestID string,
	etag string,
	nextCursor *string,
	hasMore bool,
	limit int,
) (ListProducts200JSONResponse, error) {
	if len(data.Products) > application.MaxPageLimit ||
		!isValidStrongETag(etag) {
		return ListProducts200JSONResponse{}, ErrInvalidResponseContract
	}
	meta, err := NewListResponseMeta(
		requestID,
		nextCursor,
		hasMore,
		limit,
	)
	if err != nil {
		return ListProducts200JSONResponse{}, err
	}
	if len(data.Products) > int(meta.Page.Limit) {
		return ListProducts200JSONResponse{}, ErrInvalidResponseContract
	}
	return ListProducts200JSONResponse{
		Body: ProductListResponse{
			Data: data,
			Meta: meta,
		},
		Headers: ListProducts200ResponseHeaders{
			CacheControl: PublicContentCacheControl,
			ETag:         etag,
			XRequestID:   requestID,
		},
	}, nil
}

func cloneStringPointer(value *string) *string {
	if value == nil {
		return nil
	}
	cloned := *value
	return &cloned
}

func isValidOpaqueCursor(cursor string) bool {
	if cursor == "" || len(cursor) > maxCursorBytes {
		return false
	}

	payload, signature, found := strings.Cut(cursor, ".")
	if !found ||
		payload == "" ||
		signature == "" ||
		strings.Contains(signature, ".") {
		return false
	}
	return isBase64URLSegment(payload) && isBase64URLSegment(signature)
}

func isBase64URLSegment(segment string) bool {
	for _, character := range segment {
		switch {
		case character >= 'a' && character <= 'z':
		case character >= 'A' && character <= 'Z':
		case character >= '0' && character <= '9':
		case character == '-', character == '_':
		default:
			return false
		}
	}
	return true
}

func isValidStrongETag(etag string) bool {
	if len(etag) < 3 ||
		strings.HasPrefix(etag, "W/") ||
		etag[0] != '"' ||
		etag[len(etag)-1] != '"' {
		return false
	}
	for _, character := range etag[1 : len(etag)-1] {
		if character == '"' || character < 0x21 || character == 0x7f {
			return false
		}
	}
	return true
}
