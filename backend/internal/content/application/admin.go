package application

import (
	"context"
	"encoding/json"
	"errors"
	"regexp"
	"strconv"
	"strings"
	"time"

	contentdomain "github.com/ilhamnugraha8944/tauco/backend/internal/content/domain"
)

var (
	ErrAdminPageNotFound = errors.New("admin page not found")
	ErrRevisionNotFound  = errors.New("page revision not found")
	ErrPrecondition      = errors.New("page revision precondition failed")
	ErrMediaNotReady     = errors.New("referenced media is not ready")
	ErrInvalidPage       = errors.New("invalid editable page")
)

type MediaReference struct {
	AssetID, FieldPath string
	Position           int
}

type AdminRevision struct {
	ID, OwnerID, Status   string
	Number, SchemaVersion int
	Content               json.RawMessage
	CreatedBy             *string
	CreatedAt             time.Time
	PublishedAt           *time.Time
}

type AdminRevisionSummary struct {
	ID, Status  string
	Number      int
	CreatedBy   *string
	CreatedAt   time.Time
	PublishedAt *time.Time
}

type AdminPage struct {
	ID, Key             string
	PublishedRevisionID *string
	UpdatedAt           time.Time
	Latest              AdminRevision
	Revisions           []AdminRevisionSummary
}

type AdminRepository interface {
	GetAdminPage(context.Context, string) (AdminPage, error)
	GetAdminRevision(context.Context, string, string) (AdminRevision, error)
	CreateDraft(context.Context, string, string, string, json.RawMessage, string, []MediaReference) (AdminRevision, error)
	PublishRevision(context.Context, string, string, string, string) (AdminRevision, error)
	Unpublish(context.Context, string, string, string) error
}

type AdminService struct{ repository AdminRepository }

func NewAdminService(repository AdminRepository) (*AdminService, error) {
	if repository == nil {
		return nil, errors.New("content admin service requires repository")
	}
	return &AdminService{repository: repository}, nil
}

func EditableKey(key string) bool { return key == "home" || key == "about" }

func (service *AdminService) Get(ctx context.Context, key string) (AdminPage, error) {
	if !EditableKey(key) {
		return AdminPage{}, ErrAdminPageNotFound
	}
	return service.repository.GetAdminPage(ctx, key)
}

func (service *AdminService) Revision(ctx context.Context, key, revisionID string) (AdminRevision, error) {
	if !EditableKey(key) {
		return AdminRevision{}, ErrAdminPageNotFound
	}
	return service.repository.GetAdminRevision(ctx, key, revisionID)
}

func (service *AdminService) SaveDraft(ctx context.Context, key, ifMatch, baseRevisionID, actorID string, raw json.RawMessage) (AdminRevision, error) {
	if !EditableKey(key) {
		return AdminRevision{}, ErrAdminPageNotFound
	}
	expected, err := revisionFromETag(ifMatch)
	if err != nil || expected != baseRevisionID {
		return AdminRevision{}, ErrPrecondition
	}
	canonical, checksum, err := contentdomain.CanonicalJSONChecksum(raw)
	if err != nil {
		return AdminRevision{}, ErrInvalidPage
	}
	return service.repository.CreateDraft(ctx, key, expected, actorID, canonical, string(checksum), extractMediaReferences(canonical))
}

func (service *AdminService) Publish(ctx context.Context, key, revisionID, ifMatch, actorID string) (AdminRevision, error) {
	if !EditableKey(key) {
		return AdminRevision{}, ErrAdminPageNotFound
	}
	expected, err := revisionFromETag(ifMatch)
	if err != nil {
		return AdminRevision{}, ErrPrecondition
	}
	return service.repository.PublishRevision(ctx, key, revisionID, expected, actorID)
}

func (service *AdminService) Unpublish(ctx context.Context, key, ifMatch, actorID string) error {
	if !EditableKey(key) {
		return ErrAdminPageNotFound
	}
	expected, err := revisionFromETag(ifMatch)
	if err != nil {
		return ErrPrecondition
	}
	return service.repository.Unpublish(ctx, key, expected, actorID)
}

func RevisionETag(revisionID string) string { return `"revision-` + revisionID + `"` }

func revisionFromETag(value string) (string, error) {
	if !strings.HasPrefix(value, `"revision-`) || !strings.HasSuffix(value, `"`) {
		return "", ErrPrecondition
	}
	id := strings.TrimSuffix(strings.TrimPrefix(value, `"revision-`), `"`)
	if !mediaIDPattern.MatchString(id) {
		return "", ErrPrecondition
	}
	return id, nil
}

var mediaIDPattern = regexp.MustCompile(`^[0-9a-f]{8}-[0-9a-f]{4}-7[0-9a-f]{3}-[89ab][0-9a-f]{3}-[0-9a-f]{12}$`)
var mediaSourcePattern = regexp.MustCompile(`^/api/v1/media/([0-9a-f-]{36})/(?:display|variants/[0-9]+)\.webp$`)

func extractMediaReferences(raw json.RawMessage) []MediaReference {
	var value any
	if json.Unmarshal(raw, &value) != nil {
		return nil
	}
	result := []MediaReference{}
	var walk func(any, string)
	walk = func(current any, path string) {
		switch typed := current.(type) {
		case map[string]any:
			for key, child := range typed {
				walk(child, path+"/"+key)
			}
		case []any:
			for index, child := range typed {
				walk(child, path+"/"+strconv.Itoa(index))
			}
		case string:
			if match := mediaSourcePattern.FindStringSubmatch(typed); match != nil && mediaIDPattern.MatchString(match[1]) {
				result = append(result, MediaReference{AssetID: match[1], FieldPath: path, Position: 0})
			}
		}
	}
	walk(value, "")
	return result
}
