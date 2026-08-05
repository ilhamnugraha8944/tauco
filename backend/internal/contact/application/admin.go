package application

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	catalogapp "github.com/ilhamnugraha8944/tauco/backend/internal/catalog/application"
)

var (
	ErrAdminMessageNotFound = errors.New("admin contact message not found")
	ErrAdminMessageInvalid  = errors.New("admin contact message request invalid")
	ErrAdminMessageConflict = errors.New("admin contact message precondition failed")
)

type AdminMessage struct {
	ID, Name, Email, Subject, Message, Status string
	Phone                                     *string
	CreatedAt, UpdatedAt                      time.Time
}

func (message AdminMessage) ETag() string {
	return fmt.Sprintf(`"message-%s-%d"`, message.ID, message.UpdatedAt.UnixMicro())
}

type AdminMessagePage struct {
	Messages   []AdminMessage
	NextCursor *string
	HasMore    bool
	Limit      int
}

type AdminMessageRepository interface {
	ListAdminMessages(context.Context, *catalogapp.ProductPaginationPosition, int, *string) ([]AdminMessage, bool, error)
	GetAdminMessage(context.Context, string) (AdminMessage, error)
	UpdateAdminMessageStatus(context.Context, string, time.Time, string, string) (AdminMessage, error)
}

type AdminMessageService struct {
	repository AdminMessageRepository
	cursor     catalogapp.CursorCodec
}

func NewAdminMessageService(repository AdminMessageRepository, cursor catalogapp.CursorCodec) (*AdminMessageService, error) {
	if repository == nil || cursor == nil {
		return nil, errors.New("admin inbox service requires repository and cursor")
	}
	return &AdminMessageService{repository: repository, cursor: cursor}, nil
}

func (service *AdminMessageService) List(ctx context.Context, encoded *string, requested *int, status *string) (AdminMessagePage, error) {
	limit, err := catalogapp.ResolvePageLimit(requested)
	if err != nil {
		return AdminMessagePage{}, err
	}
	if status != nil && !validMessageStatus(*status) {
		return AdminMessagePage{}, ErrAdminMessageInvalid
	}
	filter := "all"
	if status != nil {
		filter = *status
	}
	query := catalogapp.NewProductQueryHash("admin-inbox:v1:" + filter)
	var position *catalogapp.ProductPaginationPosition
	if encoded != nil {
		decoded, decodeErr := service.cursor.Decode(*encoded, query)
		if decodeErr != nil {
			return AdminMessagePage{}, catalogapp.ErrInvalidCursor
		}
		position = &decoded
	}
	messages, more, err := service.repository.ListAdminMessages(ctx, position, limit, status)
	if err != nil {
		return AdminMessagePage{}, err
	}
	var next *string
	if more && len(messages) > 0 {
		last := messages[len(messages)-1]
		value, _ := catalogapp.NewProductPaginationPosition(last.CreatedAt.UnixMicro(), last.ID)
		encodedValue, encodeErr := service.cursor.Encode(value, query)
		if encodeErr != nil {
			return AdminMessagePage{}, encodeErr
		}
		next = &encodedValue
	}
	return AdminMessagePage{Messages: messages, NextCursor: next, HasMore: more, Limit: limit}, nil
}

func (service *AdminMessageService) Get(ctx context.Context, id string) (AdminMessage, error) {
	return service.repository.GetAdminMessage(ctx, id)
}

func (service *AdminMessageService) UpdateStatus(ctx context.Context, id, ifMatch, status, actorID string) (AdminMessage, error) {
	if !validMessageStatus(status) {
		return AdminMessage{}, ErrAdminMessageInvalid
	}
	expected, ok := messageTimeFromETag(ifMatch, id)
	if !ok {
		return AdminMessage{}, ErrAdminMessageConflict
	}
	return service.repository.UpdateAdminMessageStatus(ctx, id, expected, status, actorID)
}

func validMessageStatus(status string) bool {
	return status == "unread" || status == "read" || status == "archived"
}

func messageTimeFromETag(value, id string) (time.Time, bool) {
	prefix := `"message-` + id + `-`
	if !strings.HasPrefix(value, prefix) || !strings.HasSuffix(value, `"`) {
		return time.Time{}, false
	}
	var micros int64
	if _, err := fmt.Sscanf(strings.TrimSuffix(strings.TrimPrefix(value, prefix), `"`), "%d", &micros); err != nil || micros < 1 {
		return time.Time{}, false
	}
	return time.UnixMicro(micros).UTC(), true
}
