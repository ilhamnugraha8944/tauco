package logging

import (
	"regexp"
	"strings"
	"unicode/utf8"
)

const (
	redactedValue = "[REDACTED]"
	maxLogTextLen = 2048
)

var (
	sensitiveJSONPattern = regexp.MustCompile(
		`(?i)("(?:authorization|proxy-authorization|cookie|set-cookie|password|passwd|pwd|token|access_token|refresh_token|id_token|api[-_]?key|secret|client_secret|totp|database_url|dsn)"\s*:\s*)("(?:\\.|[^"])*"|[^,\s}]+)`,
	)
	sensitiveAssignmentPattern = regexp.MustCompile(
		`(?i)\b(authorization|proxy-authorization|cookie|set-cookie|password|passwd|pwd|token|access_token|refresh_token|id_token|api[-_]?key|secret|client_secret|totp|database_url|dsn)\s*([:=])\s*([^\s,;&]+)`,
	)
	authSchemePattern  = regexp.MustCompile(`(?i)\b(bearer|basic)\s+[A-Za-z0-9._~+/=-]+`)
	jwtPattern         = regexp.MustCompile(`\beyJ[A-Za-z0-9_-]+\.[A-Za-z0-9_-]+\.[A-Za-z0-9_-]*\b`)
	uriUserInfoPattern = regexp.MustCompile(
		`(?i)([a-z][a-z0-9+.-]*://)([^@/\s]+)@`,
	)
	emailPattern = regexp.MustCompile(
		`(?i)\b[A-Z0-9._%+\-]+@[A-Z0-9.\-]+\.[A-Z]{2,}\b`,
	)
	ipv4Pattern = regexp.MustCompile(
		`\b(?:25[0-5]|2[0-4]\d|1?\d?\d)(?:\.(?:25[0-5]|2[0-4]\d|1?\d?\d)){3}\b`,
	)
	bracketedIPv6Pattern = regexp.MustCompile(`\[[0-9A-Fa-f:]*:[0-9A-Fa-f:]*\]`)
	ipv6Pattern          = regexp.MustCompile(`\b(?:[0-9A-Fa-f]{1,4}:){2,7}[0-9A-Fa-f]{1,4}\b`)
	phonePattern         = regexp.MustCompile(`(?:\+?\d[\d .()\-]{7,}\d)`)
)

// RedactString removes common secret and direct-PII forms from untrusted error
// text before it is sent to a log sink. It is defense in depth, not a reason to
// log request bodies, headers, configuration structs, or user-provided text.
func RedactString(value string) string {
	if value == "" {
		return ""
	}

	value = authSchemePattern.ReplaceAllString(value, `${1} [REDACTED]`)
	value = sensitiveJSONPattern.ReplaceAllString(value, `${1}"[REDACTED]"`)
	value = sensitiveAssignmentPattern.ReplaceAllString(value, `${1}${2}[REDACTED]`)
	value = jwtPattern.ReplaceAllString(value, redactedValue)
	value = uriUserInfoPattern.ReplaceAllString(value, `${1}[REDACTED]@`)
	value = emailPattern.ReplaceAllString(value, "[REDACTED_EMAIL]")
	value = ipv4Pattern.ReplaceAllString(value, "[REDACTED_IP]")
	value = bracketedIPv6Pattern.ReplaceAllString(value, "[REDACTED_IP]")
	value = ipv6Pattern.ReplaceAllString(value, "[REDACTED_IP]")
	value = phonePattern.ReplaceAllString(value, "[REDACTED_PHONE]")
	value = strings.NewReplacer("\r", `\r`, "\n", `\n`).Replace(value)

	return truncateUTF8(value, maxLogTextLen)
}

func truncateUTF8(value string, limit int) string {
	if len(value) <= limit {
		return value
	}

	value = value[:limit]
	for !utf8.ValidString(value) {
		value = value[:len(value)-1]
	}
	return value + "...[TRUNCATED]"
}

func safeIdentifier(value string, limit int) string {
	value = strings.TrimSpace(value)
	value = strings.Map(func(r rune) rune {
		switch {
		case r >= 'a' && r <= 'z':
			return r
		case r >= 'A' && r <= 'Z':
			return r
		case r >= '0' && r <= '9':
			return r
		case strings.ContainsRune("-_.:/", r):
			return r
		default:
			return -1
		}
	}, value)
	return truncateUTF8(value, limit)
}
