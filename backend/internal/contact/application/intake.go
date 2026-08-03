package application

import (
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"github.com/ilhamnugraha8944/tauco/backend/internal/contact/domain"
)

var (
	ErrInvalidSubmission   = errors.New("invalid contact submission")
	ErrInvalidIdempotency  = errors.New("invalid idempotency key")
	ErrIdempotencyConflict = errors.New("idempotency conflict")
)

const defaultPrivacyNoticeVersion = "2026-07-28"

type Submission struct {
	Message        domain.Message
	IdempotencyKey string
	RequestID      string
	ConsentAt      time.Time
}

type Record struct {
	Message              domain.Message
	IdempotencyKeyHash   string
	RequestPayloadHash   string
	RequestID            string
	PrivacyNoticeVersion string
	ConsentAt            time.Time
	RetentionDeleteAt    time.Time
}

type CreateResult struct {
	Replayed bool
}

type Store interface {
	Create(context.Context, Record) (CreateResult, error)
}

type Intake struct {
	store  Store
	secret []byte
	now    func() time.Time
}

func NewIntake(store Store, secret []byte) (*Intake, error) {
	if store == nil {
		return nil, errors.New("contact intake requires a store")
	}
	if len(secret) < 32 {
		return nil, errors.New("contact idempotency secret must contain at least 32 bytes")
	}
	return &Intake{store: store, secret: append([]byte(nil), secret...), now: time.Now}, nil
}

func (intake *Intake) Submit(ctx context.Context, submission Submission) (CreateResult, error) {
	if intake == nil || intake.store == nil {
		return CreateResult{}, errors.New("contact intake is not initialized")
	}
	if err := submission.Message.Validate(); err != nil {
		return CreateResult{}, ErrInvalidSubmission
	}
	if !validIdempotencyKey(submission.IdempotencyKey) {
		return CreateResult{}, ErrInvalidIdempotency
	}
	consentAt := submission.ConsentAt.UTC()
	if consentAt.IsZero() {
		consentAt = intake.now().UTC()
	}
	payload, err := json.Marshal(submission.Message)
	if err != nil {
		return CreateResult{}, fmt.Errorf("encode contact payload: %w", err)
	}
	record := Record{
		Message:              submission.Message,
		IdempotencyKeyHash:   intake.hmac(submission.IdempotencyKey),
		RequestPayloadHash:   sha256Hex(payload),
		RequestID:            submission.RequestID,
		PrivacyNoticeVersion: defaultPrivacyNoticeVersion,
		ConsentAt:            consentAt,
		RetentionDeleteAt:    domain.RetentionDeleteAt(consentAt),
	}
	result, err := intake.store.Create(ctx, record)
	if err != nil {
		return CreateResult{}, fmt.Errorf("store contact submission: %w", err)
	}
	return result, nil
}

func (intake *Intake) hmac(value string) string {
	digest := hmac.New(sha256.New, intake.secret)
	_, _ = digest.Write([]byte(value))
	return hex.EncodeToString(digest.Sum(nil))
}

func sha256Hex(value []byte) string {
	digest := sha256.Sum256(value)
	return hex.EncodeToString(digest[:])
}

func validIdempotencyKey(value string) bool {
	if len(value) < 16 || len(value) > 128 {
		return false
	}
	for _, character := range []byte(value) {
		if character < 0x21 || character > 0x7e {
			return false
		}
	}
	return true
}
