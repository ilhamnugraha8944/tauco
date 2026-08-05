package application

import (
	"context"
	"encoding/json"
	"errors"
	"regexp"
	"strings"
	"time"

	contentapp "github.com/ilhamnugraha8944/tauco/backend/internal/content/application"
	contentdomain "github.com/ilhamnugraha8944/tauco/backend/internal/content/domain"
)

var (
	ErrAdminProductNotFound    = errors.New("admin product not found")
	ErrProductRevisionNotFound = errors.New("product revision not found")
	ErrProductPrecondition     = errors.New("product precondition failed")
	ErrProductConflict         = errors.New("product state conflict")
	ErrInvalidProduct          = errors.New("invalid product")
	ErrProductMediaNotReady    = errors.New("product media is not ready")
)

const adminProductsQuery = "admin-products:v1"

type AdminProduct struct {
	ID, Slug                     string
	SKU                          *string
	SortOrder                    int
	PublishedRevisionID          *string
	FirstPublishedAt, ArchivedAt *time.Time
	UpdatedAt                    time.Time
	Revisions                    []contentapp.AdminRevisionSummary
}

func (product AdminProduct) CurrentRevisionID() string {
	if len(product.Revisions) > 0 {
		return product.Revisions[0].ID
	}
	return product.ID
}

type AdminProductPage struct {
	Products   []AdminProduct
	NextCursor *string
	HasMore    bool
	Limit      int
}

type AdminProductRepository interface {
	ListAdminProducts(context.Context, *ProductPaginationPosition, int) ([]AdminProduct, bool, error)
	GetAdminProduct(context.Context, string) (AdminProduct, error)
	CreateAdminProduct(context.Context, string, *string, int, string) (AdminProduct, error)
	UpdateAdminProduct(context.Context, string, string, *string, *string, *int, string) (AdminProduct, error)
	GetAdminProductRevision(context.Context, string, string) (contentapp.AdminRevision, error)
	CreateAdminProductDraft(context.Context, string, string, string, json.RawMessage, string, []contentapp.MediaReference) (contentapp.AdminRevision, error)
	PublishAdminProductRevision(context.Context, string, string, string, string) (contentapp.AdminRevision, error)
	UnpublishAdminProduct(context.Context, string, string, string) error
	ArchiveAdminProduct(context.Context, string, string, string) error
	UnarchiveAdminProduct(context.Context, string, string, string) error
}

type AdminProductService struct {
	repository AdminProductRepository
	cursor     CursorCodec
}

func NewAdminProductService(repository AdminProductRepository, cursor CursorCodec) (*AdminProductService, error) {
	if repository == nil || cursor == nil {
		return nil, errors.New("product admin service requires repository and cursor")
	}
	return &AdminProductService{repository: repository, cursor: cursor}, nil
}

func (service *AdminProductService) List(ctx context.Context, encoded *string, requested *int) (AdminProductPage, error) {
	limit, err := ResolvePageLimit(requested)
	if err != nil {
		return AdminProductPage{}, err
	}
	query := NewProductQueryHash(adminProductsQuery)
	var position *ProductPaginationPosition
	if encoded != nil {
		decoded, decodeErr := service.cursor.Decode(*encoded, query)
		if decodeErr != nil {
			return AdminProductPage{}, ErrInvalidCursor
		}
		position = &decoded
	}
	products, more, err := service.repository.ListAdminProducts(ctx, position, limit)
	if err != nil {
		return AdminProductPage{}, err
	}
	var next *string
	if more && len(products) > 0 {
		last := products[len(products)-1]
		value, positionErr := NewProductPaginationPosition(int64(last.SortOrder), last.ID)
		if positionErr != nil {
			return AdminProductPage{}, positionErr
		}
		encodedValue, encodeErr := service.cursor.Encode(value, query)
		if encodeErr != nil {
			return AdminProductPage{}, encodeErr
		}
		next = &encodedValue
	}
	return AdminProductPage{Products: products, NextCursor: next, HasMore: more, Limit: limit}, nil
}

func (service *AdminProductService) Get(ctx context.Context, id string) (AdminProduct, error) {
	return service.repository.GetAdminProduct(ctx, id)
}

func (service *AdminProductService) Create(ctx context.Context, slug string, sku *string, sortOrder int, actorID string) (AdminProduct, error) {
	if err := validateIdentity(slug, sku, sortOrder); err != nil {
		return AdminProduct{}, err
	}
	return service.repository.CreateAdminProduct(ctx, slug, sku, sortOrder, actorID)
}

func (service *AdminProductService) Update(ctx context.Context, id, ifMatch string, slug, sku *string, sortOrder *int, actorID string) (AdminProduct, error) {
	expected, err := contentapp.RevisionFromETag(ifMatch)
	if err != nil {
		return AdminProduct{}, ErrProductPrecondition
	}
	if slug == nil && sku == nil && sortOrder == nil {
		return AdminProduct{}, ErrInvalidProduct
	}
	if slug != nil && contentdomain.ValidateProductSlug(*slug) != nil {
		return AdminProduct{}, ErrInvalidProduct
	}
	if sku != nil && !validSKU(*sku) {
		return AdminProduct{}, ErrInvalidProduct
	}
	if sortOrder != nil && (*sortOrder < 0 || *sortOrder > 1_000_000) {
		return AdminProduct{}, ErrInvalidProduct
	}
	return service.repository.UpdateAdminProduct(ctx, id, expected, slug, sku, sortOrder, actorID)
}

func (service *AdminProductService) Revision(ctx context.Context, id, revisionID string) (contentapp.AdminRevision, error) {
	return service.repository.GetAdminProductRevision(ctx, id, revisionID)
}

func (service *AdminProductService) SaveDraft(ctx context.Context, id, ifMatch, baseRevisionID, actorID string, raw json.RawMessage) (contentapp.AdminRevision, error) {
	expected, err := contentapp.RevisionFromETag(ifMatch)
	if err != nil || expected != baseRevisionID {
		return contentapp.AdminRevision{}, ErrProductPrecondition
	}
	canonical, checksum, err := contentdomain.CanonicalJSONChecksum(raw)
	if err != nil {
		return contentapp.AdminRevision{}, ErrInvalidProduct
	}
	return service.repository.CreateAdminProductDraft(ctx, id, expected, actorID, canonical, string(checksum), contentapp.ExtractMediaReferences(canonical))
}

func (service *AdminProductService) Publish(ctx context.Context, id, revisionID, ifMatch, actorID string) (contentapp.AdminRevision, error) {
	expected, err := contentapp.RevisionFromETag(ifMatch)
	if err != nil {
		return contentapp.AdminRevision{}, ErrProductPrecondition
	}
	return service.repository.PublishAdminProductRevision(ctx, id, revisionID, expected, actorID)
}

func (service *AdminProductService) Unpublish(ctx context.Context, id, ifMatch, actorID string) error {
	expected, err := contentapp.RevisionFromETag(ifMatch)
	if err != nil {
		return ErrProductPrecondition
	}
	return service.repository.UnpublishAdminProduct(ctx, id, expected, actorID)
}

func (service *AdminProductService) Archive(ctx context.Context, id, ifMatch, actorID string) error {
	expected, err := contentapp.RevisionFromETag(ifMatch)
	if err != nil {
		return ErrProductPrecondition
	}
	return service.repository.ArchiveAdminProduct(ctx, id, expected, actorID)
}

func (service *AdminProductService) Unarchive(ctx context.Context, id, ifMatch, actorID string) error {
	expected, err := contentapp.RevisionFromETag(ifMatch)
	if err != nil {
		return ErrProductPrecondition
	}
	return service.repository.UnarchiveAdminProduct(ctx, id, expected, actorID)
}

func validateIdentity(slug string, sku *string, sortOrder int) error {
	if contentdomain.ValidateProductSlug(slug) != nil || sortOrder < 0 || sortOrder > 1_000_000 {
		return ErrInvalidProduct
	}
	if sku != nil && !validSKU(*sku) {
		return ErrInvalidProduct
	}
	return nil
}

var skuPattern = regexp.MustCompile(`^[A-Z0-9][A-Z0-9._-]{0,79}$`)

func validSKU(value string) bool {
	return value == strings.TrimSpace(value) && skuPattern.MatchString(value)
}
