package domain

import (
	"bytes"
	"testing"
)

func TestCanonicalJSONChecksumIsOrderAndWhitespaceIndependent(t *testing.T) {
	t.Parallel()

	left := []byte("{\n  \"z\": [3, 2, 1], \"a\": \"<tauco>\"\n}")
	right := []byte(`{"a":"<tauco>","z":[3,2,1]}`)
	leftCanonical, leftChecksum, err := CanonicalJSONChecksum(left)
	if err != nil {
		t.Fatalf("CanonicalJSONChecksum(left) error = %v", err)
	}
	rightCanonical, rightChecksum, err := CanonicalJSONChecksum(right)
	if err != nil {
		t.Fatalf("CanonicalJSONChecksum(right) error = %v", err)
	}
	if !bytes.Equal(leftCanonical, rightCanonical) {
		t.Fatalf("canonical documents differ:\n%s\n%s", leftCanonical, rightCanonical)
	}
	if leftChecksum != rightChecksum {
		t.Fatalf("checksums differ: %s != %s", leftChecksum, rightChecksum)
	}
	if bytes.Contains(leftCanonical, []byte(`\u003c`)) {
		t.Fatalf("canonical JSON unexpectedly HTML-escaped: %s", leftCanonical)
	}
	if err := ValidateCanonicalJSON(leftCanonical, leftChecksum); err != nil {
		t.Fatalf("ValidateCanonicalJSON(canonical) error = %v", err)
	}
	if err := ValidateCanonicalJSON(left, leftChecksum); err == nil {
		t.Fatal("ValidateCanonicalJSON(noncanonical) unexpectedly succeeded")
	}
}

func TestCanonicalJSONRejectsUnsupportedNumbersAndTrailingValues(t *testing.T) {
	t.Parallel()

	for _, raw := range [][]byte{
		[]byte(`{"value":1.5}`),
		[]byte(`{"value":1e2}`),
		[]byte(`{"value":01}`),
		[]byte(`{} {}`),
		{'"', 0xff, '"'},
	} {
		if _, _, err := CanonicalJSONChecksum(raw); err == nil {
			t.Errorf("CanonicalJSONChecksum(%q) unexpectedly succeeded", raw)
		}
	}
}
