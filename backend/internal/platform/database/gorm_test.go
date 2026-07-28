package database

import (
	"errors"
	"testing"
	"time"
)

func TestLoadRuntimeConfigUsesPoolerSafeDefaults(t *testing.T) {
	t.Parallel()

	config, err := LoadRuntimeConfig(runtimeMapLookup(map[string]string{
		"DATABASE_URL": "postgres://runtime:secret@db.example/tauco?sslmode=require",
	}))
	if err != nil {
		t.Fatalf("LoadRuntimeConfig() error = %v", err)
	}
	if config.MaxOpenConns != 5 ||
		config.MaxIdleConns != 2 ||
		config.ConnMaxLifetime != 30*time.Minute ||
		config.ConnMaxIdleTime != 5*time.Minute ||
		!config.PreferSimpleQuery {
		t.Fatalf("LoadRuntimeConfig() = %+v, want pooler-safe defaults", config)
	}
}

func TestLoadRuntimeConfigRejectsInvalidPoolBoundsWithoutLeakingURL(
	t *testing.T,
) {
	t.Parallel()

	secret := "do-not-leak"
	_, err := LoadRuntimeConfig(runtimeMapLookup(map[string]string{
		"DATABASE_URL":            "postgres://runtime:" + secret + "@db.example/tauco",
		"DATABASE_MAX_OPEN_CONNS": "2",
		"DATABASE_MAX_IDLE_CONNS": "3",
	}))
	if err == nil {
		t.Fatal("LoadRuntimeConfig() unexpectedly succeeded")
	}
	var configError *ConfigError
	if !errors.As(err, &configError) {
		t.Fatalf("LoadRuntimeConfig() error = %T, want ConfigError", err)
	}
	if containsText(err.Error(), secret) {
		t.Fatalf("LoadRuntimeConfig() leaked secret in %q", err)
	}
}

func runtimeMapLookup(values map[string]string) LookupEnv {
	return func(key string) (string, bool) {
		value, exists := values[key]
		return value, exists
	}
}

func containsText(value, fragment string) bool {
	for index := 0; index+len(fragment) <= len(value); index++ {
		if value[index:index+len(fragment)] == fragment {
			return true
		}
	}
	return false
}
