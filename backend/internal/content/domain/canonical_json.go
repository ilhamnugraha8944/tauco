package domain

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"sort"
	"strings"
	"unicode/utf8"
)

// CanonicalizeJSON implements canonical-json-v1: UTF-8 JSON, lexicographically
// sorted object keys, preserved array order, no insignificant whitespace, no
// HTML escaping, and minimal decimal integers. Content documents do not use
// floating point numbers.
func CanonicalizeJSON(raw []byte) ([]byte, error) {
	if !utf8.Valid(raw) {
		return nil, errors.New("canonical JSON must contain valid UTF-8")
	}
	decoder := json.NewDecoder(bytes.NewReader(raw))
	decoder.UseNumber()
	var document any
	if err := decoder.Decode(&document); err != nil {
		return nil, err
	}
	if err := expectJSONEOF(decoder); err != nil {
		return nil, err
	}

	var output bytes.Buffer
	if err := appendCanonicalJSON(&output, document); err != nil {
		return nil, err
	}
	return output.Bytes(), nil
}

// CanonicalJSONChecksum canonicalizes raw and returns its SHA-256 digest.
func CanonicalJSONChecksum(raw []byte) (
	[]byte,
	SHA256Checksum,
	error,
) {
	canonical, err := CanonicalizeJSON(raw)
	if err != nil {
		return nil, "", err
	}
	digest := sha256.Sum256(canonical)
	return canonical, SHA256Checksum(hex.EncodeToString(digest[:])), nil
}

// ValidateCanonicalJSON binds exact canonical bytes to their declared digest.
func ValidateCanonicalJSON(raw []byte, checksum SHA256Checksum) error {
	if _, err := ParseSHA256Checksum(string(checksum)); err != nil {
		return err
	}
	canonical, actual, err := CanonicalJSONChecksum(raw)
	if err != nil {
		return err
	}
	if !bytes.Equal(raw, canonical) {
		return errors.New("content JSON does not use canonical-json-v1 encoding")
	}
	if actual != checksum {
		return fmt.Errorf(
			"content checksum mismatch: got %s, want %s",
			checksum,
			actual,
		)
	}
	return nil
}

func appendCanonicalJSON(output *bytes.Buffer, value any) error {
	switch typed := value.(type) {
	case nil:
		output.WriteString("null")
	case bool:
		if typed {
			output.WriteString("true")
		} else {
			output.WriteString("false")
		}
	case string:
		encoded, err := marshalJSONString(typed)
		if err != nil {
			return err
		}
		output.Write(encoded)
	case json.Number:
		number := typed.String()
		if !isCanonicalInteger(number) {
			return fmt.Errorf(
				"canonical-json-v1 accepts only minimal decimal integers, got %q",
				number,
			)
		}
		output.WriteString(number)
	case []any:
		output.WriteByte('[')
		for index, child := range typed {
			if index > 0 {
				output.WriteByte(',')
			}
			if err := appendCanonicalJSON(output, child); err != nil {
				return err
			}
		}
		output.WriteByte(']')
	case map[string]any:
		keys := make([]string, 0, len(typed))
		for key := range typed {
			keys = append(keys, key)
		}
		sort.Strings(keys)
		output.WriteByte('{')
		for index, key := range keys {
			if index > 0 {
				output.WriteByte(',')
			}
			encodedKey, err := marshalJSONString(key)
			if err != nil {
				return err
			}
			output.Write(encodedKey)
			output.WriteByte(':')
			if err := appendCanonicalJSON(output, typed[key]); err != nil {
				return err
			}
		}
		output.WriteByte('}')
	default:
		return fmt.Errorf("unsupported canonical JSON value %T", value)
	}
	return nil
}

func marshalJSONString(value string) ([]byte, error) {
	var output bytes.Buffer
	encoder := json.NewEncoder(&output)
	encoder.SetEscapeHTML(false)
	if err := encoder.Encode(value); err != nil {
		return nil, err
	}
	return bytes.TrimSuffix(output.Bytes(), []byte{'\n'}), nil
}

func isCanonicalInteger(value string) bool {
	if value == "0" {
		return true
	}
	if strings.HasPrefix(value, "-") {
		value = strings.TrimPrefix(value, "-")
	}
	if len(value) == 0 || value[0] == '0' {
		return false
	}
	for _, character := range value {
		if character < '0' || character > '9' {
			return false
		}
	}
	return true
}

func expectJSONEOF(decoder *json.Decoder) error {
	var trailing any
	err := decoder.Decode(&trailing)
	if errors.Is(err, io.EOF) {
		return nil
	}
	if err == nil {
		return errors.New("JSON document contains more than one value")
	}
	return err
}
