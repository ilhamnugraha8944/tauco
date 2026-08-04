package application

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"

	"github.com/ilhamnugraha8944/tauco/backend/internal/media/domain"
)

const MaxUploadBytes = 10 << 20

type ObjectStore interface {
	PutIfAbsent(context.Context, string, string, []byte) error
	Get(context.Context, string) ([]byte, error)
}

type NormalizedImage struct {
	Data          []byte
	MIME          string
	Width, Height int
	SHA256        string
}

type GeneratedVariant struct {
	Data          []byte
	Width, Height int
	SHA256        string
}

type Processor interface {
	Normalize([]byte) (NormalizedImage, error)
	Variants([]byte) ([]GeneratedVariant, []int, error)
}

type Repository interface {
	CreateProcessing(context.Context, domain.Asset) (string, bool, error)
	Load(context.Context, string) (domain.Asset, error)
	MarkReady(context.Context, string, []domain.Variant, []int) error
	MarkFailed(context.Context, string, string) error
}

type Ingestor struct {
	repository Repository
	store      ObjectStore
	processor  Processor
}

func NewIngestor(repository Repository, store ObjectStore, processor Processor) (*Ingestor, error) {
	if repository == nil || store == nil || processor == nil {
		return nil, errors.New("media ingestor requires repository, storage, and processor")
	}
	return &Ingestor{repository: repository, store: store, processor: processor}, nil
}

func (ingestor *Ingestor) Ingest(ctx context.Context, source []byte, alt string, decorative bool) (string, bool, error) {
	if len(source) == 0 || len(source) > MaxUploadBytes {
		return "", false, errors.New("media source must be between 1 byte and 10 MiB")
	}
	normalized, err := ingestor.processor.Normalize(source)
	if err != nil {
		return "", false, err
	}
	key := fmt.Sprintf("media/original/%s.png", normalized.SHA256)
	asset := domain.Asset{
		Status: domain.StatusProcessing, OriginalKey: key,
		OriginalMIME: normalized.MIME, OriginalWidth: normalized.Width,
		OriginalHeight: normalized.Height, OriginalBytes: int64(len(normalized.Data)),
		SHA256: normalized.SHA256, AltText: alt, Decorative: decorative,
	}
	if err := asset.Validate(); err != nil {
		return "", false, err
	}
	if err := ingestor.store.PutIfAbsent(ctx, key, normalized.MIME, normalized.Data); err != nil {
		return "", false, fmt.Errorf("store normalized media: %w", err)
	}
	return ingestor.repository.CreateProcessing(ctx, asset)
}

type VariantHandler struct {
	repository Repository
	store      ObjectStore
	processor  Processor
}

func NewVariantHandler(repository Repository, store ObjectStore, processor Processor) (*VariantHandler, error) {
	if repository == nil || store == nil || processor == nil {
		return nil, errors.New("media variant handler requires repository, storage, and processor")
	}
	return &VariantHandler{repository: repository, store: store, processor: processor}, nil
}

func (handler *VariantHandler) Handle(ctx context.Context, payload json.RawMessage) error {
	var body struct {
		MediaAssetID string `json:"mediaAssetId"`
	}
	if err := json.Unmarshal(payload, &body); err != nil || body.MediaAssetID == "" {
		return errors.New("invalid media job payload")
	}
	asset, err := handler.repository.Load(ctx, body.MediaAssetID)
	if err != nil {
		return err
	}
	source, err := handler.store.Get(ctx, asset.OriginalKey)
	if err != nil {
		_ = handler.repository.MarkFailed(ctx, asset.ID, "ORIGINAL_READ_FAILED")
		return err
	}
	generated, skipped, err := handler.processor.Variants(source)
	if err != nil {
		_ = handler.repository.MarkFailed(ctx, asset.ID, "VARIANT_PROCESSING_FAILED")
		return err
	}
	variants := make([]domain.Variant, 0, len(generated))
	for _, item := range generated {
		key := fmt.Sprintf("media/variant/%s/w%d.webp", asset.SHA256, item.Width)
		if err := handler.store.PutIfAbsent(ctx, key, "image/webp", item.Data); err != nil {
			_ = handler.repository.MarkFailed(ctx, asset.ID, "VARIANT_STORE_FAILED")
			return err
		}
		variants = append(variants, domain.Variant{
			Width: item.Width, Height: item.Height, ObjectKey: key,
			MIME: "image/webp", Bytes: int64(len(item.Data)), SHA256: item.SHA256,
		})
	}
	return handler.repository.MarkReady(ctx, asset.ID, variants, skipped)
}
