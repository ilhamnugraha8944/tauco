package application

import (
	"context"
	"errors"
	"regexp"
	"time"

	catalogapp "github.com/ilhamnugraha8944/tauco/backend/internal/catalog/application"
)

var ErrInvalidActivityFilter = errors.New("invalid activity filter")

type Activity struct {
	ID, EventType, EntityType, ActorType string
	EntityID, ActorID, RequestID         *string
	CreatedAt                            time.Time
}

type ActivityFilter struct{ EventType, EntityType *string }

type ActivityPage struct {
	Activities []Activity
	NextCursor *string
	HasMore    bool
	Limit      int
}

type AdminRepository interface {
	ListAdminActivities(context.Context, *catalogapp.ProductPaginationPosition, int, ActivityFilter) ([]Activity, bool, error)
}

type AdminService struct {
	repository AdminRepository
	cursor     catalogapp.CursorCodec
}

func NewAdminService(repository AdminRepository, cursor catalogapp.CursorCodec) (*AdminService, error) {
	if repository == nil || cursor == nil {
		return nil, errors.New("admin activity service requires repository and cursor")
	}
	return &AdminService{repository: repository, cursor: cursor}, nil
}

var activityToken = regexp.MustCompile(`^[a-z][a-z0-9]*([._-][a-z0-9]+)*$`)

func (service *AdminService) List(ctx context.Context, encoded *string, requested *int, filter ActivityFilter) (ActivityPage, error) {
	limit, err := catalogapp.ResolvePageLimit(requested)
	if err != nil {
		return ActivityPage{}, err
	}
	if !validFilter(filter.EventType, 100) || !validFilter(filter.EntityType, 80) {
		return ActivityPage{}, ErrInvalidActivityFilter
	}
	event, entity := "all", "all"
	if filter.EventType != nil {
		event = *filter.EventType
	}
	if filter.EntityType != nil {
		entity = *filter.EntityType
	}
	query := catalogapp.NewProductQueryHash("admin-activity:v1:" + event + ":" + entity)
	var position *catalogapp.ProductPaginationPosition
	if encoded != nil {
		decoded, decodeErr := service.cursor.Decode(*encoded, query)
		if decodeErr != nil {
			return ActivityPage{}, catalogapp.ErrInvalidCursor
		}
		position = &decoded
	}
	activities, more, err := service.repository.ListAdminActivities(ctx, position, limit, filter)
	if err != nil {
		return ActivityPage{}, err
	}
	var next *string
	if more && len(activities) > 0 {
		last := activities[len(activities)-1]
		value, _ := catalogapp.NewProductPaginationPosition(last.CreatedAt.UnixMicro(), last.ID)
		encodedValue, encodeErr := service.cursor.Encode(value, query)
		if encodeErr != nil {
			return ActivityPage{}, encodeErr
		}
		next = &encodedValue
	}
	return ActivityPage{Activities: activities, NextCursor: next, HasMore: more, Limit: limit}, nil
}

func validFilter(value *string, max int) bool {
	return value == nil || (len(*value) >= 3 && len(*value) <= max && activityToken.MatchString(*value))
}
