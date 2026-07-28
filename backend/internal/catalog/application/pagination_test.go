package application_test

import (
	"errors"
	"strings"
	"testing"

	"github.com/ilhamnugraha8944/tauco/backend/internal/catalog/application"
)

func TestProductPaginationPosition(t *testing.T) {
	t.Parallel()

	position, err := application.NewProductPaginationPosition(
		12,
		"019c0a8d-f070-7a9a-b9f4-d39e50ad7370",
	)
	if err != nil {
		t.Fatalf("NewProductPaginationPosition() error = %v", err)
	}
	if got := position.SortOrder(); got != 12 {
		t.Fatalf("SortOrder() = %d, want 12", got)
	}
	if got := position.ProductID(); got != "019c0a8d-f070-7a9a-b9f4-d39e50ad7370" {
		t.Fatalf("ProductID() = %q, want stable product ID", got)
	}
	if err := position.Validate(); err != nil {
		t.Fatalf("Validate() error = %v", err)
	}
}

func TestProductPaginationPositionRejectsInvalidValues(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name      string
		sortOrder int64
		productID string
	}{
		{name: "negative sort order", sortOrder: -1, productID: "product-id"},
		{name: "empty product ID", productID: ""},
		{name: "whitespace product ID", productID: " product-id"},
		{name: "control character", productID: "product\nid"},
		{name: "invalid UTF-8", productID: string([]byte{0xff})},
		{name: "oversize product ID", productID: strings.Repeat("a", 129)},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()

			_, err := application.NewProductPaginationPosition(
				test.sortOrder,
				test.productID,
			)
			if !errors.Is(err, application.ErrInvalidPaginationPosition) {
				t.Fatalf(
					"NewProductPaginationPosition() error = %v, want %v",
					err,
					application.ErrInvalidPaginationPosition,
				)
			}
		})
	}

	var zeroValue application.ProductPaginationPosition
	if !errors.Is(
		zeroValue.Validate(),
		application.ErrInvalidPaginationPosition,
	) {
		t.Fatal("zero-value position must be invalid")
	}
}

func TestProductQueryHash(t *testing.T) {
	t.Parallel()

	first := application.NewProductQueryHash("published=true&q=tauco")
	same := application.NewProductQueryHash("published=true&q=tauco")
	different := application.NewProductQueryHash("published=true&q=kedelai")

	if first.Encoded() != same.Encoded() {
		t.Fatal("equal canonical queries must produce equal hashes")
	}
	if first.Encoded() == different.Encoded() {
		t.Fatal("different canonical queries must produce different hashes")
	}
	if len(first.Encoded()) != 43 {
		t.Fatalf("Encoded() length = %d, want 43", len(first.Encoded()))
	}
	if err := first.Validate(); err != nil {
		t.Fatalf("Validate() error = %v", err)
	}

	var zeroValue application.ProductQueryHash
	if !errors.Is(
		zeroValue.Validate(),
		application.ErrInvalidProductQueryHash,
	) {
		t.Fatal("zero-value query hash must be invalid")
	}
	if got := zeroValue.Encoded(); got != "" {
		t.Fatalf("zero-value Encoded() = %q, want empty string", got)
	}
}

func TestResolvePageLimit(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name      string
		requested *int
		want      int
		wantError bool
	}{
		{name: "omitted uses default", want: application.DefaultPageLimit},
		{name: "minimum", requested: intPointer(1), want: 1},
		{name: "default explicit", requested: intPointer(20), want: 20},
		{name: "maximum", requested: intPointer(50), want: 50},
		{name: "explicit zero", requested: intPointer(0), wantError: true},
		{name: "negative", requested: intPointer(-1), wantError: true},
		{name: "over maximum", requested: intPointer(51), wantError: true},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()

			got, err := application.ResolvePageLimit(test.requested)
			if test.wantError {
				if !errors.Is(err, application.ErrInvalidPageLimit) {
					t.Fatalf(
						"ResolvePageLimit() error = %v, want %v",
						err,
						application.ErrInvalidPageLimit,
					)
				}
				return
			}
			if err != nil {
				t.Fatalf("ResolvePageLimit() error = %v", err)
			}
			if got != test.want {
				t.Fatalf("ResolvePageLimit() = %d, want %d", got, test.want)
			}
			if err := application.ValidatePageLimit(got); err != nil {
				t.Fatalf("ValidatePageLimit(%d) error = %v", got, err)
			}
		})
	}
}

func intPointer(value int) *int {
	return &value
}
