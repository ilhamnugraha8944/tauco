package application

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"

	"github.com/ilhamnugraha8944/tauco/backend/internal/media/domain"
)

type UploadHandler struct {
	repository UploadIntentRepository
	store      QuarantineStore
	ingestor   *Ingestor
}

func NewUploadHandler(repository UploadIntentRepository, store QuarantineStore, ingestor *Ingestor) (*UploadHandler, error) {
	if repository == nil || store == nil || ingestor == nil {
		return nil, errors.New("media upload handler requires repository, storage, and ingestor")
	}
	return &UploadHandler{repository: repository, store: store, ingestor: ingestor}, nil
}

func (handler *UploadHandler) Handle(ctx context.Context, payload json.RawMessage) error {
	var body struct {
		UploadIntentID string `json:"uploadIntentId"`
	}
	if err := json.Unmarshal(payload, &body); err != nil || body.UploadIntentID == "" {
		return errors.New("invalid media upload job payload")
	}
	intent, err := handler.repository.GetUploadIntent(ctx, body.UploadIntentID)
	if err != nil {
		return err
	}
	if intent.Status == domain.UploadStatusCompleted || intent.Status == domain.UploadStatusFailed || intent.Status == domain.UploadStatusExpired {
		return nil
	}
	if intent.Status != domain.UploadStatusQueued {
		return ErrUploadIntentConflict
	}

	source, err := handler.store.GetBounded(ctx, intent.QuarantineKey, MaxUploadBytes)
	if err != nil {
		if errors.Is(err, ErrObjectTooLarge) {
			return handler.failPermanently(ctx, intent.ID, "UPLOAD_TOO_LARGE")
		}
		return err
	}
	if int64(len(source)) != intent.ExpectedBytes {
		return handler.failPermanently(ctx, intent.ID, "UPLOAD_SIZE_MISMATCH")
	}
	digest := sha256.Sum256(source)
	if hex.EncodeToString(digest[:]) != intent.ExpectedSHA256 {
		return handler.failPermanently(ctx, intent.ID, "UPLOAD_SHA256_MISMATCH")
	}
	if detected := http.DetectContentType(source); detected != intent.ExpectedMIME {
		return handler.failPermanently(ctx, intent.ID, "UPLOAD_MIME_MISMATCH")
	}

	assetID, _, err := handler.ingestor.Ingest(ctx, source, intent.AltText, intent.Decorative)
	if err != nil {
		if errors.Is(err, ErrInvalidMediaSource) {
			return handler.failPermanently(ctx, intent.ID, "UPLOAD_INVALID_IMAGE")
		}
		return err
	}
	if err := handler.repository.CompleteUploadIntent(ctx, intent.ID, assetID); err != nil {
		return err
	}
	// Deletion is retried by the bounded daily cleanup if the provider is
	// temporarily unavailable. The database remains the source of truth.
	_ = handler.store.Delete(ctx, intent.QuarantineKey)
	return nil
}

func (handler *UploadHandler) failPermanently(ctx context.Context, intentID, code string) error {
	if err := handler.repository.FailUploadIntent(ctx, intentID, code); err != nil {
		return fmt.Errorf("mark media upload failed: %w", err)
	}
	return nil
}

type UploadCleanup struct {
	repository UploadIntentRepository
	store      ObjectDeleter
}

func NewUploadCleanup(repository UploadIntentRepository, store ObjectDeleter) (*UploadCleanup, error) {
	if repository == nil || store == nil {
		return nil, errors.New("media upload cleanup requires repository and storage")
	}
	return &UploadCleanup{repository: repository, store: store}, nil
}

func (cleanup *UploadCleanup) RunOnce(ctx context.Context, limit int) (int, error) {
	intents, err := cleanup.repository.ClaimUploadIntentCleanup(ctx, limit)
	if err != nil {
		return 0, err
	}
	completed := 0
	var cleanupErrors []error
	for _, intent := range intents {
		if err := cleanup.store.Delete(ctx, intent.QuarantineKey); err != nil {
			cleanupErrors = append(cleanupErrors, fmt.Errorf("delete quarantine object %s: %w", intent.ID, err))
			continue
		}
		if err := cleanup.repository.CompleteUploadIntentCleanup(ctx, intent.ID); err != nil {
			cleanupErrors = append(cleanupErrors, fmt.Errorf("complete quarantine cleanup %s: %w", intent.ID, err))
			continue
		}
		completed++
	}
	return completed, errors.Join(cleanupErrors...)
}
