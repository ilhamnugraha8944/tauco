package application

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/ilhamnugraha8944/tauco/backend/internal/media/domain"
)

const (
	uploadIntentLifetime = 15 * time.Minute
	presignedPutLifetime = 10 * time.Minute
)

var ErrInvalidUploadIntent = errors.New("invalid media upload intent")

type CreateUploadIntentInput struct {
	MIMEType   string
	Bytes      int64
	SHA256     string
	AltText    string
	Decorative bool
	CreatedBy  string
}

type CreatedUploadIntent struct {
	Intent domain.UploadIntent
	Upload PresignedUpload
}

type UploadService struct {
	repository UploadIntentRepository
	store      QuarantineStore
}

func NewUploadService(repository UploadIntentRepository, store QuarantineStore) (*UploadService, error) {
	if repository == nil || store == nil {
		return nil, errors.New("media upload service requires repository and quarantine storage")
	}
	return &UploadService{repository: repository, store: store}, nil
}

func (service *UploadService) Create(ctx context.Context, input CreateUploadIntentInput) (CreatedUploadIntent, error) {
	draft := domain.UploadIntentDraft{
		ExpectedMIME: input.MIMEType, ExpectedBytes: input.Bytes,
		ExpectedSHA256: input.SHA256, AltText: input.AltText,
		Decorative: input.Decorative, CreatedBy: input.CreatedBy,
		ExpiresAt: time.Now().UTC().Add(uploadIntentLifetime),
	}
	if err := draft.Validate(); err != nil {
		return CreatedUploadIntent{}, fmt.Errorf("%w: %v", ErrInvalidUploadIntent, err)
	}
	intent, err := service.repository.CreateUploadIntent(ctx, draft)
	if err != nil {
		return CreatedUploadIntent{}, fmt.Errorf("create media upload intent: %w", err)
	}
	upload, err := service.store.PresignPut(
		ctx, intent.QuarantineKey, intent.ExpectedMIME, intent.ExpectedSHA256,
		intent.ExpectedBytes, presignedPutLifetime,
	)
	if err != nil {
		return CreatedUploadIntent{}, fmt.Errorf("presign media upload: %w", err)
	}
	return CreatedUploadIntent{Intent: intent, Upload: upload}, nil
}

func (service *UploadService) Get(ctx context.Context, intentID string) (domain.UploadIntent, error) {
	return service.repository.GetUploadIntent(ctx, intentID)
}

func (service *UploadService) Finalize(ctx context.Context, intentID string) (domain.UploadIntent, bool, error) {
	intent, err := service.repository.GetUploadIntent(ctx, intentID)
	if err != nil {
		return domain.UploadIntent{}, false, err
	}
	switch intent.Status {
	case domain.UploadStatusQueued, domain.UploadStatusCompleted:
		return intent, true, nil
	case domain.UploadStatusPending:
		if !intent.ExpiresAt.After(time.Now().UTC()) {
			return domain.UploadIntent{}, false, ErrUploadIntentConflict
		}
	default:
		return domain.UploadIntent{}, false, ErrUploadIntentConflict
	}
	observed, err := service.store.Head(ctx, intent.QuarantineKey)
	if err != nil {
		if errors.Is(err, ErrObjectNotFound) {
			return domain.UploadIntent{}, false, ErrUploadIntentConflict
		}
		return domain.UploadIntent{}, false, fmt.Errorf("inspect quarantined media: %w", err)
	}
	if observed.Bytes != intent.ExpectedBytes || observed.MIMEType != intent.ExpectedMIME || observed.SHA256 != intent.ExpectedSHA256 {
		return domain.UploadIntent{}, false, ErrUploadIntentMetadata
	}
	return service.repository.QueueUploadIntent(
		ctx, intent.ID, observed.MIMEType, observed.Bytes, observed.SHA256,
	)
}
