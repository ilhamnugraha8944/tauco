package cursor_test

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/base64"
	"errors"
	"strings"
	"testing"

	"github.com/ilhamnugraha8944/tauco/backend/internal/catalog/application"
	"github.com/ilhamnugraha8944/tauco/backend/internal/catalog/delivery/cursor"
)

const (
	testSecret = "0123456789abcdef0123456789abcdef"
	productID  = "019c0a8d-f070-7a9a-b9f4-d39e50ad7370"
)

func TestHMACSHA256RoundTripAndDeterministicEncoding(t *testing.T) {
	t.Parallel()

	codec := mustCodec(t, []byte(testSecret))
	position := mustPosition(t, 17, productID)
	queryHash := application.NewProductQueryHash(
		"published=true&q=tauco",
	)

	first, err := codec.Encode(position, queryHash)
	if err != nil {
		t.Fatalf("Encode() error = %v", err)
	}
	second, err := codec.Encode(position, queryHash)
	if err != nil {
		t.Fatalf("second Encode() error = %v", err)
	}
	if first != second {
		t.Fatal("Encode() must be deterministic for equal inputs")
	}
	if strings.Contains(first, "=") {
		t.Fatal("cursor must use unpadded base64url")
	}

	decoded, err := codec.Decode(first, queryHash)
	if err != nil {
		t.Fatalf("Decode() error = %v", err)
	}
	if decoded.SortOrder() != position.SortOrder() ||
		decoded.ProductID() != position.ProductID() {
		t.Fatalf("Decode() = %#v, want original position", decoded)
	}
}

func TestHMACSHA256RejectsTamperAndBindingMismatch(t *testing.T) {
	t.Parallel()

	codec := mustCodec(t, []byte(testSecret))
	otherCodec := mustCodec(
		t,
		[]byte("abcdef0123456789abcdef0123456789"),
	)
	position := mustPosition(t, 3, productID)
	queryHash := application.NewProductQueryHash("q=tauco")
	encoded := mustEncode(t, codec, position, queryHash)

	last := encoded[len(encoded)-1]
	replacement := byte('A')
	if last == replacement {
		replacement = 'B'
	}
	tampered := encoded[:len(encoded)-1] + string(replacement)

	tests := []struct {
		name      string
		codec     *cursor.HMACSHA256Codec
		encoded   string
		queryHash application.ProductQueryHash
	}{
		{
			name:      "tampered cursor",
			codec:     codec,
			encoded:   tampered,
			queryHash: queryHash,
		},
		{
			name:      "wrong filter",
			codec:     codec,
			encoded:   encoded,
			queryHash: application.NewProductQueryHash("q=kedelai"),
		},
		{
			name:      "wrong secret",
			codec:     otherCodec,
			encoded:   encoded,
			queryHash: queryHash,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()

			assertInvalidCursor(
				t,
				test.codec,
				test.encoded,
				test.queryHash,
			)
		})
	}
}

func TestHMACSHA256RejectsMalformedCursor(t *testing.T) {
	t.Parallel()

	codec := mustCodec(t, []byte(testSecret))
	queryHash := application.NewProductQueryHash("q=tauco")

	tests := []struct {
		name    string
		encoded string
	}{
		{name: "empty"},
		{name: "no delimiter", encoded: "not-a-cursor"},
		{name: "empty payload", encoded: ".signature"},
		{name: "empty signature", encoded: "payload."},
		{name: "extra delimiter", encoded: "a.b.c"},
		{name: "invalid base64url", encoded: "***.***"},
		{name: "padded base64url", encoded: "e30=.c2ln"},
		{name: "oversize", encoded: strings.Repeat("a", 2048)},
		{
			name: "wrong signature size",
			encoded: signedToken(
				[]byte(`{"version":1}`),
				[]byte(testSecret),
				1,
			),
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()
			assertInvalidCursor(t, codec, test.encoded, queryHash)
		})
	}
}

func TestHMACSHA256StrictCanonicalPayload(t *testing.T) {
	t.Parallel()

	codec := mustCodec(t, []byte(testSecret))
	queryHash := application.NewProductQueryHash("q=tauco")
	hash := queryHash.Encoded()

	tests := []struct {
		name    string
		payload string
	}{
		{
			name: "unknown field",
			payload: `{"version":1,"query_hash":"` + hash +
				`","sort_order":1,"product_id":"` + productID +
				`","extra":true}`,
		},
		{
			name: "trailing JSON",
			payload: `{"version":1,"query_hash":"` + hash +
				`","sort_order":1,"product_id":"` + productID +
				`"}{"other":true}`,
		},
		{
			name: "unsupported version",
			payload: `{"version":2,"query_hash":"` + hash +
				`","sort_order":1,"product_id":"` + productID + `"}`,
		},
		{
			name: "invalid position",
			payload: `{"version":1,"query_hash":"` + hash +
				`","sort_order":-1,"product_id":"` + productID + `"}`,
		},
		{
			name: "missing product ID",
			payload: `{"version":1,"query_hash":"` + hash +
				`","sort_order":1,"product_id":""}`,
		},
		{
			name: "noncanonical whitespace",
			payload: `{"version": 1,"query_hash":"` + hash +
				`","sort_order":1,"product_id":"` + productID + `"}`,
		},
		{
			name: "noncanonical field order",
			payload: `{"query_hash":"` + hash +
				`","version":1,"sort_order":1,"product_id":"` +
				productID + `"}`,
		},
		{
			name: "duplicate field",
			payload: `{"version":1,"version":1,"query_hash":"` + hash +
				`","sort_order":1,"product_id":"` + productID + `"}`,
		},
		{
			name: "wrong query hash encoding",
			payload: `{"version":1,"query_hash":"invalid","sort_order":1,` +
				`"product_id":"` + productID + `"}`,
		},
		{
			name:    "oversize payload",
			payload: strings.Repeat(" ", 600),
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()

			encoded := signedToken(
				[]byte(test.payload),
				[]byte(testSecret),
				sha256.Size,
			)
			assertInvalidCursor(t, codec, encoded, queryHash)
		})
	}
}

func TestHMACSHA256RejectsInvalidPositionOnEncode(t *testing.T) {
	t.Parallel()

	codec := mustCodec(t, []byte(testSecret))
	var invalid application.ProductPaginationPosition

	_, err := codec.Encode(
		invalid,
		application.NewProductQueryHash("q=tauco"),
	)
	if !errors.Is(err, application.ErrInvalidCursor) {
		t.Fatalf(
			"Encode() error = %v, want %v",
			err,
			application.ErrInvalidCursor,
		)
	}
}

func TestHMACSHA256RejectsUninitializedQueryHash(t *testing.T) {
	t.Parallel()

	codec := mustCodec(t, []byte(testSecret))
	position := mustPosition(t, 1, productID)
	var queryHash application.ProductQueryHash

	if _, err := codec.Encode(position, queryHash); !errors.Is(
		err,
		application.ErrInvalidCursor,
	) {
		t.Fatalf("Encode() error = %v, want invalid cursor", err)
	}

	encoded := mustEncode(
		t,
		codec,
		position,
		application.NewProductQueryHash(""),
	)
	assertInvalidCursor(t, codec, encoded, queryHash)
}

func TestHMACSHA256SecretRequirementsAndOwnership(t *testing.T) {
	t.Parallel()

	secret := []byte(testSecret)
	codec := mustCodec(t, secret)
	position := mustPosition(t, 1, productID)
	queryHash := application.NewProductQueryHash("")
	encoded := mustEncode(t, codec, position, queryHash)

	for index := range secret {
		secret[index] = 'x'
	}
	if _, err := codec.Decode(encoded, queryHash); err != nil {
		t.Fatalf("codec must retain an owned secret copy: %v", err)
	}

	if _, err := cursor.NewHMACSHA256(
		[]byte(strings.Repeat("s", cursor.MinSecretBytes-1)),
	); !errors.Is(err, cursor.ErrInvalidSecret) {
		t.Fatalf(
			"NewHMACSHA256() error = %v, want %v",
			err,
			cursor.ErrInvalidSecret,
		)
	}
	if _, err := cursor.NewHMACSHA256(
		[]byte(strings.Repeat("s", cursor.MinSecretBytes)),
	); err != nil {
		t.Fatalf("32-byte secret should be accepted: %v", err)
	}
}

func TestHMACSHA256ErrorsDoNotLeakInputOrSecret(t *testing.T) {
	t.Parallel()

	secret := []byte(testSecret)
	codec := mustCodec(t, secret)
	sensitiveInput := "customer-filter-do-not-log"
	encoded := signedToken(
		[]byte(sensitiveInput),
		[]byte("abcdef0123456789abcdef0123456789"),
		sha256.Size,
	)

	_, err := codec.Decode(
		encoded,
		application.NewProductQueryHash(sensitiveInput),
	)
	if err != application.ErrInvalidCursor {
		t.Fatalf(
			"Decode() error = %v, want exact generic sentinel",
			err,
		)
	}
	for _, sensitive := range []string{
		string(secret),
		sensitiveInput,
		encoded,
	} {
		if strings.Contains(err.Error(), sensitive) {
			t.Fatalf("error leaks sensitive input %q", sensitive)
		}
	}
}

func mustCodec(
	t *testing.T,
	secret []byte,
) *cursor.HMACSHA256Codec {
	t.Helper()

	codec, err := cursor.NewHMACSHA256(secret)
	if err != nil {
		t.Fatalf("NewHMACSHA256() error = %v", err)
	}
	return codec
}

func mustPosition(
	t *testing.T,
	sortOrder int64,
	productID string,
) application.ProductPaginationPosition {
	t.Helper()

	position, err := application.NewProductPaginationPosition(
		sortOrder,
		productID,
	)
	if err != nil {
		t.Fatalf("NewProductPaginationPosition() error = %v", err)
	}
	return position
}

func mustEncode(
	t *testing.T,
	codec *cursor.HMACSHA256Codec,
	position application.ProductPaginationPosition,
	queryHash application.ProductQueryHash,
) string {
	t.Helper()

	encoded, err := codec.Encode(position, queryHash)
	if err != nil {
		t.Fatalf("Encode() error = %v", err)
	}
	return encoded
}

func assertInvalidCursor(
	t *testing.T,
	codec *cursor.HMACSHA256Codec,
	encoded string,
	queryHash application.ProductQueryHash,
) {
	t.Helper()

	_, err := codec.Decode(encoded, queryHash)
	if err != application.ErrInvalidCursor {
		t.Fatalf(
			"Decode() error = %v, want exact generic sentinel",
			err,
		)
	}
}

func signedToken(payload, secret []byte, signatureBytes int) string {
	mac := hmac.New(sha256.New, secret)
	_, _ = mac.Write(payload)
	signature := mac.Sum(nil)
	if signatureBytes < len(signature) {
		signature = signature[:signatureBytes]
	}
	return base64.RawURLEncoding.EncodeToString(payload) + "." +
		base64.RawURLEncoding.EncodeToString(signature)
}
