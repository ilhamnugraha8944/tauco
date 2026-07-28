package application

import (
	"crypto/sha256"
	"encoding/base64"
	"errors"
	"strings"
	"unicode"
	"unicode/utf8"
)

const (
	// DefaultPageLimit is used when a client omits the catalog page limit.
	DefaultPageLimit = 20
	// MaxPageLimit bounds the amount of catalog data returned by one request.
	MaxPageLimit = 50

	maxProductIDBytes = 128
)

var (
	// ErrInvalidCursor is deliberately generic. Cursor adapters must not expose
	// parsing, signature, filter, or secret details to callers.
	ErrInvalidCursor = errors.New("invalid cursor")
	// ErrInvalidPaginationPosition indicates an unusable catalog sort position.
	ErrInvalidPaginationPosition = errors.New("invalid product pagination position")
	// ErrInvalidPageLimit indicates a limit outside the supported catalog range.
	ErrInvalidPageLimit = errors.New("invalid page limit")
	// ErrInvalidProductQueryHash indicates an uninitialized query fingerprint.
	ErrInvalidProductQueryHash = errors.New("invalid product query hash")
)

// ProductPaginationPosition is the immutable keyset position for the catalog's
// deterministic sort_order ASC, id ASC ordering.
//
// Its fields are intentionally private. Values must be created through
// NewProductPaginationPosition so adapters cannot construct invalid cursors by
// mutating exported fields.
type ProductPaginationPosition struct {
	sortOrder int64
	productID string
}

// NewProductPaginationPosition validates and creates a catalog keyset
// position. Product IDs are treated as stable opaque identifiers; their
// concrete UUID representation remains a repository concern.
func NewProductPaginationPosition(
	sortOrder int64,
	productID string,
) (ProductPaginationPosition, error) {
	position := ProductPaginationPosition{
		sortOrder: sortOrder,
		productID: productID,
	}
	if err := position.Validate(); err != nil {
		return ProductPaginationPosition{}, err
	}
	return position, nil
}

// SortOrder returns the first component of the catalog keyset position.
func (position ProductPaginationPosition) SortOrder() int64 {
	return position.sortOrder
}

// ProductID returns the stable tie-breaker component of the catalog keyset
// position.
func (position ProductPaginationPosition) ProductID() string {
	return position.productID
}

// Validate checks invariants even for a zero value created outside the
// constructor.
func (position ProductPaginationPosition) Validate() error {
	if position.sortOrder < 0 || !validProductID(position.productID) {
		return ErrInvalidPaginationPosition
	}
	return nil
}

func validProductID(productID string) bool {
	if productID == "" ||
		len(productID) > maxProductIDBytes ||
		!utf8.ValidString(productID) ||
		strings.TrimSpace(productID) != productID {
		return false
	}

	for _, character := range productID {
		if unicode.IsControl(character) {
			return false
		}
	}
	return true
}

// ProductQueryHash is an immutable SHA-256 fingerprint of the canonical
// catalog filter/query representation. Only the digest, never the raw query,
// is embedded in a cursor.
type ProductQueryHash struct {
	sum         [sha256.Size]byte
	initialized bool
}

// NewProductQueryHash fingerprints the caller's canonical query
// representation. Callers must use the same canonicalization for encode and
// decode.
func NewProductQueryHash(canonicalQuery string) ProductQueryHash {
	return ProductQueryHash{
		sum:         sha256.Sum256([]byte(canonicalQuery)),
		initialized: true,
	}
}

// Encoded returns the digest in unpadded base64url form for canonical cursor
// serialization.
func (queryHash ProductQueryHash) Encoded() string {
	if !queryHash.initialized {
		return ""
	}
	return base64.RawURLEncoding.EncodeToString(queryHash.sum[:])
}

// Validate rejects the zero value so every cursor is bound to a query
// fingerprint created by NewProductQueryHash.
func (queryHash ProductQueryHash) Validate() error {
	if !queryHash.initialized {
		return ErrInvalidProductQueryHash
	}
	return nil
}

// CursorCodec is the application port for opaque product-list cursors.
// Implementations must bind cursors to the supplied query hash.
type CursorCodec interface {
	Encode(
		position ProductPaginationPosition,
		queryHash ProductQueryHash,
	) (string, error)
	Decode(
		encodedCursor string,
		queryHash ProductQueryHash,
	) (ProductPaginationPosition, error)
}

// ResolvePageLimit applies the default only when the request omitted a limit.
// An explicitly supplied zero remains invalid.
func ResolvePageLimit(requested *int) (int, error) {
	if requested == nil {
		return DefaultPageLimit, nil
	}
	if err := ValidatePageLimit(*requested); err != nil {
		return 0, err
	}
	return *requested, nil
}

// ValidatePageLimit checks the inclusive supported catalog page-size range.
func ValidatePageLimit(limit int) error {
	if limit < 1 || limit > MaxPageLimit {
		return ErrInvalidPageLimit
	}
	return nil
}
