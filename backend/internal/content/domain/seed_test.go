package domain

import "testing"

func TestParseUUIDv7(t *testing.T) {
	t.Parallel()

	valid := "019bfc80-0000-7000-8000-000000000001"
	got, err := ParseUUIDv7(valid)
	if err != nil {
		t.Fatalf("ParseUUIDv7(valid) error = %v", err)
	}
	if got != UUIDv7(valid) {
		t.Fatalf("ParseUUIDv7(valid) = %q, want %q", got, valid)
	}

	for _, value := range []string{
		"",
		"019bfc80-0000-4000-8000-000000000001",
		"019bfc80-0000-7000-7000-000000000001",
		"019BFC80-0000-7000-8000-000000000001",
		"019bfc80-0000-7000-8000-00000000000z",
	} {
		if _, err := ParseUUIDv7(value); err == nil {
			t.Errorf("ParseUUIDv7(%q) unexpectedly succeeded", value)
		}
	}
}

func TestParseSHA256Checksum(t *testing.T) {
	t.Parallel()

	valid := "0123456789abcdef0123456789abcdef0123456789abcdef0123456789abcdef"
	if _, err := ParseSHA256Checksum(valid); err != nil {
		t.Fatalf("ParseSHA256Checksum(valid) error = %v", err)
	}
	for _, value := range []string{
		"",
		"0123456789ABCDEF0123456789ABCDEF0123456789ABCDEF0123456789ABCDEF",
		"z123456789abcdef0123456789abcdef0123456789abcdef0123456789abcdef",
	} {
		if _, err := ParseSHA256Checksum(value); err == nil {
			t.Errorf("ParseSHA256Checksum(%q) unexpectedly succeeded", value)
		}
	}
}

func TestPageKeyAndProductSlugValidation(t *testing.T) {
	t.Parallel()

	for _, key := range []PageKey{
		PageKeyHome,
		PageKeyAbout,
		PageKeyTaucoGuide,
		PageKeyProducts,
	} {
		if !key.Valid() {
			t.Errorf("expected page key %q to be valid", key)
		}
	}
	if PageKey("unknown").Valid() {
		t.Error("unexpected valid unknown page key")
	}

	for _, slug := range []string{"tauco-cap-badak", "produk-2"} {
		if err := ValidateProductSlug(slug); err != nil {
			t.Errorf("ValidateProductSlug(%q) error = %v", slug, err)
		}
	}
	for _, slug := range []string{"", "Tauco", "tauco--badak", "-tauco"} {
		if err := ValidateProductSlug(slug); err == nil {
			t.Errorf("ValidateProductSlug(%q) unexpectedly succeeded", slug)
		}
	}
}
