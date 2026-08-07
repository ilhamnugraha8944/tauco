package domain

import (
	"encoding/hex"
	"errors"
	"strings"
	"time"
)

const (
	StatusProcessing = "processing"
	StatusReady      = "ready"
	StatusFailed     = "failed"

	UploadStatusPending   = "pending"
	UploadStatusQueued    = "queued"
	UploadStatusCompleted = "completed"
	UploadStatusFailed    = "failed"
	UploadStatusExpired   = "expired"
)

var VariantWidths = [...]int{320, 640, 1280}

type Asset struct {
	ID             string
	Status         string
	OriginalKey    string
	OriginalMIME   string
	OriginalWidth  int
	OriginalHeight int
	OriginalBytes  int64
	SHA256         string
	AltText        string
	Decorative     bool
}

func (asset Asset) Validate() error {
	if asset.OriginalKey == "" || asset.OriginalMIME == "" ||
		asset.OriginalWidth < 1 || asset.OriginalHeight < 1 ||
		asset.OriginalBytes < 1 || len(asset.SHA256) != 64 {
		return errors.New("media asset metadata is incomplete")
	}
	if asset.Decorative {
		if asset.AltText != "" {
			return errors.New("decorative media must have empty alt text")
		}
		return nil
	}
	if strings.TrimSpace(asset.AltText) != asset.AltText || asset.AltText == "" || len(asset.AltText) > 300 {
		return errors.New("informative media requires canonical alt text")
	}
	return nil
}

type Variant struct {
	Width, Height int
	ObjectKey     string
	MIME          string
	Bytes         int64
	SHA256        string
}

type UploadIntentDraft struct {
	ExpectedMIME   string
	ExpectedBytes  int64
	ExpectedSHA256 string
	AltText        string
	Decorative     bool
	CreatedBy      string
	ExpiresAt      time.Time
}

func (draft UploadIntentDraft) Validate() error {
	switch draft.ExpectedMIME {
	case "image/jpeg", "image/png", "image/webp":
	default:
		return errors.New("unsupported upload MIME type")
	}
	if draft.ExpectedBytes < 1 || draft.ExpectedBytes > 10<<20 {
		return errors.New("upload size must be between 1 byte and 10 MiB")
	}
	decoded, err := hex.DecodeString(draft.ExpectedSHA256)
	if err != nil || len(decoded) != 32 || strings.ToLower(draft.ExpectedSHA256) != draft.ExpectedSHA256 {
		return errors.New("upload SHA-256 must be 64 lowercase hexadecimal characters")
	}
	if draft.CreatedBy == "" || draft.ExpiresAt.IsZero() {
		return errors.New("upload actor and expiry are required")
	}
	if draft.Decorative {
		if draft.AltText != "" {
			return errors.New("decorative media must have empty alt text")
		}
		return nil
	}
	if strings.TrimSpace(draft.AltText) != draft.AltText || draft.AltText == "" || len(draft.AltText) > 300 {
		return errors.New("informative media requires canonical alt text")
	}
	return nil
}

type UploadIntent struct {
	ID, Status, QuarantineKey                  string
	ExpectedMIME, ExpectedSHA256, AltText      string
	ExpectedBytes                              int64
	Decorative                                 bool
	CreatedBy                                  string
	MediaAssetID, LastErrorCode                *string
	ExpiresAt, CreatedAt, UpdatedAt            time.Time
	QueuedAt, CompletedAt, FailedAt, ExpiredAt *time.Time
	CleanupClaimedAt, QuarantineDeletedAt      *time.Time
}
