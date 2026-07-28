package domain

import (
	"encoding/hex"
	"errors"
	"fmt"
	"regexp"
	"strings"
)

const (
	// UUIDv7Length is the canonical textual UUID length, including hyphens.
	UUIDv7Length = 36

	// SHA256ChecksumLength is the lowercase hexadecimal SHA-256 length.
	SHA256ChecksumLength = 64
)

var slugPattern = regexp.MustCompile(`^[a-z0-9]+(?:-[a-z0-9]+)*$`)

// PageKey is a stable natural key for a singleton content document.
type PageKey string

const (
	PageKeyHome       PageKey = "home"
	PageKeyAbout      PageKey = "about"
	PageKeyTaucoGuide PageKey = "tauco-guide"
	PageKeyProducts   PageKey = "products"
)

// Valid reports whether the page key is part of the Phase 1B content model.
func (key PageKey) Valid() bool {
	switch key {
	case PageKeyHome, PageKeyAbout, PageKeyTaucoGuide, PageKeyProducts:
		return true
	default:
		return false
	}
}

// RevisionStatus is the lifecycle state stored with an immutable revision.
type RevisionStatus string

const (
	RevisionStatusDraft     RevisionStatus = "draft"
	RevisionStatusPublished RevisionStatus = "published"
	RevisionStatusArchived  RevisionStatus = "archived"
)

// Valid reports whether status is a recognized revision state.
func (status RevisionStatus) Valid() bool {
	switch status {
	case RevisionStatusDraft, RevisionStatusPublished, RevisionStatusArchived:
		return true
	default:
		return false
	}
}

// UUIDv7 is a canonical lowercase RFC 9562 UUID version 7 string.
type UUIDv7 string

// ParseUUIDv7 validates and returns a canonical UUIDv7 value.
func ParseUUIDv7(value string) (UUIDv7, error) {
	if len(value) != UUIDv7Length {
		return "", fmt.Errorf("UUIDv7 must contain %d characters", UUIDv7Length)
	}
	if value != strings.ToLower(value) {
		return "", errors.New("UUIDv7 must use lowercase hexadecimal")
	}
	if value[8] != '-' || value[13] != '-' || value[18] != '-' || value[23] != '-' {
		return "", errors.New("UUIDv7 has invalid hyphen placement")
	}
	if value[14] != '7' {
		return "", errors.New("UUID must use version 7")
	}
	if !strings.ContainsRune("89ab", rune(value[19])) {
		return "", errors.New("UUIDv7 has an invalid RFC 9562 variant")
	}

	compact := strings.ReplaceAll(value, "-", "")
	if _, err := hex.DecodeString(compact); err != nil {
		return "", fmt.Errorf("UUIDv7 contains non-hexadecimal data: %w", err)
	}
	return UUIDv7(value), nil
}

// SHA256Checksum is a canonical lowercase hexadecimal SHA-256 digest.
type SHA256Checksum string

// ParseSHA256Checksum validates and returns a canonical checksum.
func ParseSHA256Checksum(value string) (SHA256Checksum, error) {
	if len(value) != SHA256ChecksumLength {
		return "", fmt.Errorf(
			"SHA-256 checksum must contain %d hexadecimal characters",
			SHA256ChecksumLength,
		)
	}
	if value != strings.ToLower(value) {
		return "", errors.New("SHA-256 checksum must use lowercase hexadecimal")
	}
	if _, err := hex.DecodeString(value); err != nil {
		return "", fmt.Errorf("SHA-256 checksum is not hexadecimal: %w", err)
	}
	return SHA256Checksum(value), nil
}

// ValidateProductSlug rejects unstable or non-canonical product slugs.
func ValidateProductSlug(slug string) error {
	if len(slug) == 0 || len(slug) > 80 {
		return errors.New("product slug must contain between 1 and 80 characters")
	}
	if !slugPattern.MatchString(slug) {
		return errors.New(
			"product slug must use lowercase letters, numbers, and single hyphens",
		)
	}
	return nil
}
