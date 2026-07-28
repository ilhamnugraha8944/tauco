package main

import (
	"bytes"
	"context"
	"strings"
	"testing"
)

func TestRunRequiresExplicitContentDirectory(t *testing.T) {
	t.Parallel()

	var stdout, stderr bytes.Buffer
	code := run(
		context.Background(),
		nil,
		func(string) (string, bool) { return "", false },
		&stdout,
		&stderr,
	)
	if code != 2 {
		t.Fatalf("run() code = %d, want 2", code)
	}
	if stdout.Len() != 0 ||
		!strings.Contains(stderr.String(), "usage: seed --content-dir") {
		t.Fatalf(
			"run() stdout/stderr = %q/%q",
			stdout.String(),
			stderr.String(),
		)
	}
}

func TestRunRejectsUnknownArgumentsBeforeReadingConfiguration(t *testing.T) {
	t.Parallel()

	var stdout, stderr bytes.Buffer
	lookupCalls := 0
	code := run(
		context.Background(),
		[]string{"--content-dir", "content", "unexpected"},
		func(string) (string, bool) {
			lookupCalls++
			return "", false
		},
		&stdout,
		&stderr,
	)
	if code != 2 || lookupCalls != 0 {
		t.Fatalf("run() code/lookup calls = %d/%d, want 2/0", code, lookupCalls)
	}
}

func TestRunReportsRedactedConfigurationError(t *testing.T) {
	t.Parallel()

	const secret = "do-not-leak-seed-password"
	var stdout, stderr bytes.Buffer
	code := run(
		context.Background(),
		[]string{"--content-dir", "content"},
		func(key string) (string, bool) {
			if key == "MIGRATION_DATABASE_URL" {
				return "mysql://owner:" + secret + "@localhost/tauco", true
			}
			return "", false
		},
		&stdout,
		&stderr,
	)
	if code != 1 {
		t.Fatalf("run() code = %d, want 1", code)
	}
	if strings.Contains(stderr.String(), secret) {
		t.Fatalf("run() leaked secret in %q", stderr.String())
	}
}
