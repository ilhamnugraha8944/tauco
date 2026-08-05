package application

import (
	"context"
	"errors"
	"time"

	catalogapp "github.com/ilhamnugraha8944/tauco/backend/internal/catalog/application"
)

var (
	ErrAssetNotFound = errors.New("media asset not found")
	ErrInvalidCursor = errors.New("invalid media cursor")
	ErrRetryConflict = errors.New("media asset cannot be retried")
)

const mediaQuery = "admin-media:v1"

type AdminVariant struct {
	Width, Height int
	ObjectKey     string
	Bytes         int64
	SHA256        string
}

type AdminAsset struct {
	ID, Status, MIME, AltText string
	Width, Height             int
	Bytes                     int64
	Decorative                bool
	LastErrorCode             *string
	Variants                  []AdminVariant
	CreatedAt, UpdatedAt      time.Time
}

type AdminPage struct {
	Assets     []AdminAsset
	NextCursor *string
	HasMore    bool
	Limit      int
}

type AdminRepository interface {
	ListAdmin(context.Context, *catalogapp.ProductPaginationPosition, int) ([]AdminAsset, bool, error)
	GetAdmin(context.Context, string) (AdminAsset, error)
	Retry(context.Context, string, string) error
	GetReadyVariant(context.Context, string, *int) (AdminVariant, error)
}

type AdminService struct {
	repository AdminRepository
	cursor     catalogapp.CursorCodec
}

func NewAdminService(repository AdminRepository, cursor catalogapp.CursorCodec) (*AdminService, error) {
	if repository == nil || cursor == nil {
		return nil, errors.New("media admin service requires repository and cursor codec")
	}
	return &AdminService{repository: repository, cursor: cursor}, nil
}

func (service *AdminService) List(ctx context.Context, encoded *string, requested *int) (AdminPage, error) {
	limit, err := catalogapp.ResolvePageLimit(requested)
	if err != nil {
		return AdminPage{}, err
	}
	query := catalogapp.NewProductQueryHash(mediaQuery)
	var position *catalogapp.ProductPaginationPosition
	if encoded != nil {
		decoded, decodeErr := service.cursor.Decode(*encoded, query)
		if decodeErr != nil {
			return AdminPage{}, ErrInvalidCursor
		}
		position = &decoded
	}
	assets, hasMore, err := service.repository.ListAdmin(ctx, position, limit)
	if err != nil {
		return AdminPage{}, err
	}
	var next *string
	if hasMore && len(assets) > 0 {
		last := assets[len(assets)-1]
		position, positionErr := catalogapp.NewProductPaginationPosition(last.CreatedAt.UnixMicro(), last.ID)
		if positionErr != nil {
			return AdminPage{}, positionErr
		}
		value, encodeErr := service.cursor.Encode(position, query)
		if encodeErr != nil {
			return AdminPage{}, encodeErr
		}
		next = &value
	}
	return AdminPage{Assets: assets, NextCursor: next, HasMore: hasMore, Limit: limit}, nil
}

func (service *AdminService) Get(ctx context.Context, id string) (AdminAsset, error) {
	return service.repository.GetAdmin(ctx, id)
}

func (service *AdminService) Retry(ctx context.Context, id, actorID string) (AdminAsset, error) {
	if err := service.repository.Retry(ctx, id, actorID); err != nil {
		return AdminAsset{}, err
	}
	return service.repository.GetAdmin(ctx, id)
}

func (service *AdminService) ReadyVariant(ctx context.Context, id string, width *int) (AdminVariant, error) {
	return service.repository.GetReadyVariant(ctx, id, width)
}
