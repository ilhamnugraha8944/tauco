package config

import (
	"errors"
	"strings"
	"testing"
	"time"
)

func TestLoadWithLookupUsesSafeLocalDefaults(t *testing.T) {
	t.Parallel()

	cfg, err := LoadWithLookup(mapLookup(nil))
	if err != nil {
		t.Fatalf("LoadWithLookup() error = %v", err)
	}

	if cfg.Environment != EnvironmentLocal {
		t.Fatalf("Environment = %q, want %q", cfg.Environment, EnvironmentLocal)
	}
	if cfg.HTTP.Host != "127.0.0.1" {
		t.Errorf("HTTP.Host = %q, want loopback", cfg.HTTP.Host)
	}
	if cfg.HTTP.Port != 8080 {
		t.Errorf("HTTP.Port = %d, want 8080", cfg.HTTP.Port)
	}
	if cfg.HTTP.Address() != "127.0.0.1:8080" {
		t.Errorf("HTTP.Address() = %q", cfg.HTTP.Address())
	}
	if cfg.HTTP.ShutdownGracePeriod != 10*time.Second {
		t.Errorf("ShutdownGracePeriod = %s, want 10s", cfg.HTTP.ShutdownGracePeriod)
	}
	if cfg.Log.Level != "debug" || cfg.Log.Format != "console" || cfg.Log.Service != "tauco-api" {
		t.Errorf("Log = %#v, want local defaults", cfg.Log)
	}
}

func TestLoadWithLookupAcceptsExplicitProductionConfig(t *testing.T) {
	t.Parallel()

	cfg, err := LoadWithLookup(mapLookup(map[string]string{
		"APP_ENV":                  "production",
		"HTTP_HOST":                "0.0.0.0",
		"PORT":                     "9000",
		"HTTP_READ_HEADER_TIMEOUT": "4s",
		"HTTP_READ_TIMEOUT":        "12s",
		"HTTP_WRITE_TIMEOUT":       "25s",
		"HTTP_IDLE_TIMEOUT":        "75s",
		"SHUTDOWN_GRACE_PERIOD":    "9s",
		"LOG_LEVEL":                "WARN",
		"LOG_FORMAT":               "JSON",
		"SERVICE_NAME":             "tauco-worker",
	}))
	if err != nil {
		t.Fatalf("LoadWithLookup() error = %v", err)
	}

	if cfg.Environment != EnvironmentProduction {
		t.Errorf("Environment = %q", cfg.Environment)
	}
	if cfg.HTTP.Address() != "0.0.0.0:9000" {
		t.Errorf("HTTP.Address() = %q", cfg.HTTP.Address())
	}
	if cfg.Log.Level != "warn" || cfg.Log.Format != "json" {
		t.Errorf("Log = %#v, want normalized production config", cfg.Log)
	}
}

func TestLoadWithLookupRequiresExplicitRemoteBindings(t *testing.T) {
	t.Parallel()

	_, err := LoadWithLookup(mapLookup(map[string]string{
		"APP_ENV": "staging",
	}))

	validation := requireValidationError(t, err)
	for _, field := range []string{"HTTP_HOST", "PORT", "SERVICE_NAME"} {
		if !validation.Has(field) {
			t.Errorf("validation error does not contain %s: %v", field, validation)
		}
	}
}

func TestLoadWithLookupRejectsInvalidValuesWithoutEchoingThem(t *testing.T) {
	t.Parallel()

	const rejected = "do-not-echo-this-secret-like-value"
	_, err := LoadWithLookup(mapLookup(map[string]string{
		"APP_ENV":                  "elsewhere",
		"HTTP_HOST":                "https://example.com/path",
		"PORT":                     rejected,
		"HTTP_READ_HEADER_TIMEOUT": "-1s",
		"HTTP_READ_TIMEOUT":        "1s",
		"HTTP_WRITE_TIMEOUT":       "forever",
		"HTTP_IDLE_TIMEOUT":        "0s",
		"SHUTDOWN_GRACE_PERIOD":    "2m",
		"LOG_LEVEL":                "verbose",
		"LOG_FORMAT":               "text",
		"SERVICE_NAME":             "Tauco API",
	}))

	validation := requireValidationError(t, err)
	for _, field := range []string{
		"APP_ENV",
		"HTTP_HOST",
		"PORT",
		"HTTP_READ_HEADER_TIMEOUT",
		"HTTP_WRITE_TIMEOUT",
		"HTTP_IDLE_TIMEOUT",
		"SHUTDOWN_GRACE_PERIOD",
		"LOG_LEVEL",
		"LOG_FORMAT",
		"SERVICE_NAME",
	} {
		if !validation.Has(field) {
			t.Errorf("validation error does not contain %s: %v", field, validation)
		}
	}
	if strings.Contains(err.Error(), rejected) {
		t.Errorf("error leaked rejected value: %v", err)
	}
}

func TestLoadWithLookupRejectsConflictingPortAliases(t *testing.T) {
	t.Parallel()

	_, err := LoadWithLookup(mapLookup(map[string]string{
		"PORT":      "8080",
		"HTTP_PORT": "9090",
	}))

	validation := requireValidationError(t, err)
	if !validation.Has("PORT") {
		t.Fatalf("validation error does not contain PORT: %v", validation)
	}
}

func TestLoadWithLookupPermitsMatchingPortAliases(t *testing.T) {
	t.Parallel()

	cfg, err := LoadWithLookup(mapLookup(map[string]string{
		"PORT":      "9090",
		"HTTP_PORT": "9090",
	}))
	if err != nil {
		t.Fatalf("LoadWithLookup() error = %v", err)
	}
	if cfg.HTTP.Port != 9090 {
		t.Errorf("HTTP.Port = %d, want 9090", cfg.HTTP.Port)
	}
}

func TestLoadWithLookupRejectsNonJSONAndDebugProductionLogging(t *testing.T) {
	t.Parallel()

	_, err := LoadWithLookup(mapLookup(map[string]string{
		"APP_ENV":      "production",
		"HTTP_HOST":    "::",
		"PORT":         "8080",
		"LOG_LEVEL":    "debug",
		"LOG_FORMAT":   "console",
		"SERVICE_NAME": "tauco-api",
	}))

	validation := requireValidationError(t, err)
	if !validation.Has("LOG_LEVEL") || !validation.Has("LOG_FORMAT") {
		t.Fatalf("expected strict production logging failures, got %v", validation)
	}
}

func TestHTTPConfigAddressSupportsIPv6(t *testing.T) {
	t.Parallel()

	cfg := HTTPConfig{Host: "::", Port: 8080}
	if got := cfg.Address(); got != "[::]:8080" {
		t.Fatalf("Address() = %q, want %q", got, "[::]:8080")
	}
}

func TestConfigValidateRejectsUnsafeProgrammaticConfig(t *testing.T) {
	t.Parallel()

	cfg := Config{
		Environment: EnvironmentProduction,
		HTTP: HTTPConfig{
			Host:                "*",
			Port:                70000,
			ReadHeaderTimeout:   time.Second,
			ReadTimeout:         time.Second,
			WriteTimeout:        time.Second,
			IdleTimeout:         time.Second,
			ShutdownGracePeriod: time.Second,
		},
		Log: LogConfig{
			Level:   "debug",
			Format:  "console",
			Service: "x",
		},
	}

	validation := requireValidationError(t, cfg.Validate())
	for _, field := range []string{"HTTP_HOST", "PORT", "LOG_LEVEL", "LOG_FORMAT", "SERVICE_NAME"} {
		if !validation.Has(field) {
			t.Errorf("validation error does not contain %s: %v", field, validation)
		}
	}
}

func TestLoadWithLookupRejectsNilLookup(t *testing.T) {
	t.Parallel()

	_, err := LoadWithLookup(nil)
	validation := requireValidationError(t, err)
	if !validation.Has("environment") {
		t.Fatalf("validation error = %v", validation)
	}
}

func mapLookup(values map[string]string) LookupEnv {
	return func(key string) (string, bool) {
		value, found := values[key]
		return value, found
	}
}

func requireValidationError(t *testing.T, err error) *ValidationError {
	t.Helper()

	if err == nil {
		t.Fatal("expected validation error, got nil")
	}
	var validation *ValidationError
	if !errors.As(err, &validation) {
		t.Fatalf("error type = %T, want *ValidationError", err)
	}
	return validation
}
