// Package cursor provides the transport-safe implementation of the catalog
// cursor application port.
package cursor

import (
	"bytes"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"errors"
	"io"
	"strings"

	"github.com/ilhamnugraha8944/tauco/backend/internal/catalog/application"
)

const (
	// MinSecretBytes provides a 256-bit minimum key for HMAC-SHA256.
	MinSecretBytes = 32

	cursorVersion         = 1
	maxEncodedCursorBytes = 1024
	maxPayloadBytes       = 512
)

var ErrInvalidSecret = errors.New(
	"cursor secret must contain at least 32 bytes",
)

type canonicalPayload struct {
	Version   int    `json:"version"`
	QueryHash string `json:"query_hash"`
	SortOrder int64  `json:"sort_order"`
	ProductID string `json:"product_id"`
}

// HMACSHA256Codec encodes canonical JSON payloads and authenticates them using
// HMAC-SHA256. Its secret is copied at construction and is never included in
// returned errors.
type HMACSHA256Codec struct {
	secret []byte
}

var _ application.CursorCodec = (*HMACSHA256Codec)(nil)

// NewHMACSHA256 creates a cursor codec with a copied key. Keys shorter than 32
// bytes are rejected before the codec can be used.
func NewHMACSHA256(secret []byte) (*HMACSHA256Codec, error) {
	if len(secret) < MinSecretBytes {
		return nil, ErrInvalidSecret
	}
	return &HMACSHA256Codec{secret: append([]byte(nil), secret...)}, nil
}

// Encode creates a deterministic base64url payload.signature token.
func (codec *HMACSHA256Codec) Encode(
	position application.ProductPaginationPosition,
	queryHash application.ProductQueryHash,
) (string, error) {
	if codec == nil || len(codec.secret) < MinSecretBytes {
		return "", application.ErrInvalidCursor
	}
	if err := position.Validate(); err != nil {
		return "", application.ErrInvalidCursor
	}
	if err := queryHash.Validate(); err != nil {
		return "", application.ErrInvalidCursor
	}

	payload := canonicalPayload{
		Version:   cursorVersion,
		QueryHash: queryHash.Encoded(),
		SortOrder: position.SortOrder(),
		ProductID: position.ProductID(),
	}
	payloadBytes, err := json.Marshal(payload)
	if err != nil || len(payloadBytes) > maxPayloadBytes {
		return "", application.ErrInvalidCursor
	}

	payloadSegment := base64.RawURLEncoding.EncodeToString(payloadBytes)
	signatureSegment := base64.RawURLEncoding.EncodeToString(
		codec.signature(payloadBytes),
	)
	encodedCursor := payloadSegment + "." + signatureSegment
	if len(encodedCursor) > maxEncodedCursorBytes {
		return "", application.ErrInvalidCursor
	}
	return encodedCursor, nil
}

// Decode authenticates and strictly parses a cursor. Every invalid cursor
// path returns the same sentinel so callers cannot distinguish signature,
// filter, version, size, or serialization failures.
func (codec *HMACSHA256Codec) Decode(
	encodedCursor string,
	queryHash application.ProductQueryHash,
) (application.ProductPaginationPosition, error) {
	invalid := func() (application.ProductPaginationPosition, error) {
		return application.ProductPaginationPosition{},
			application.ErrInvalidCursor
	}

	if codec == nil ||
		len(codec.secret) < MinSecretBytes ||
		encodedCursor == "" ||
		len(encodedCursor) > maxEncodedCursorBytes ||
		queryHash.Validate() != nil {
		return invalid()
	}

	payloadSegment, signatureSegment, found := strings.Cut(
		encodedCursor,
		".",
	)
	if !found ||
		payloadSegment == "" ||
		signatureSegment == "" ||
		strings.Contains(signatureSegment, ".") {
		return invalid()
	}

	payloadBytes, ok := decodeCanonicalBase64URL(payloadSegment)
	if !ok || len(payloadBytes) == 0 || len(payloadBytes) > maxPayloadBytes {
		return invalid()
	}
	signature, ok := decodeCanonicalBase64URL(signatureSegment)
	if !ok || len(signature) != sha256.Size {
		return invalid()
	}

	// hmac.Equal performs the signature comparison in constant time.
	if !hmac.Equal(signature, codec.signature(payloadBytes)) {
		return invalid()
	}

	payload, ok := decodeCanonicalPayload(payloadBytes)
	if !ok || payload.Version != cursorVersion {
		return invalid()
	}

	encodedExpectedHash := queryHash.Encoded()
	if !hmac.Equal(
		[]byte(payload.QueryHash),
		[]byte(encodedExpectedHash),
	) {
		return invalid()
	}

	position, err := application.NewProductPaginationPosition(
		payload.SortOrder,
		payload.ProductID,
	)
	if err != nil {
		return invalid()
	}
	return position, nil
}

func (codec *HMACSHA256Codec) signature(payload []byte) []byte {
	mac := hmac.New(sha256.New, codec.secret)
	_, _ = mac.Write(payload)
	return mac.Sum(nil)
}

func decodeCanonicalBase64URL(encoded string) ([]byte, bool) {
	decoded, err := base64.RawURLEncoding.Strict().DecodeString(encoded)
	if err != nil ||
		base64.RawURLEncoding.EncodeToString(decoded) != encoded {
		return nil, false
	}
	return decoded, true
}

func decodeCanonicalPayload(payloadBytes []byte) (canonicalPayload, bool) {
	var payload canonicalPayload
	decoder := json.NewDecoder(bytes.NewReader(payloadBytes))
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(&payload); err != nil {
		return canonicalPayload{}, false
	}

	var trailing json.RawMessage
	if err := decoder.Decode(&trailing); !errors.Is(err, io.EOF) {
		return canonicalPayload{}, false
	}

	canonical, err := json.Marshal(payload)
	if err != nil || !bytes.Equal(canonical, payloadBytes) {
		return canonicalPayload{}, false
	}
	return payload, true
}
