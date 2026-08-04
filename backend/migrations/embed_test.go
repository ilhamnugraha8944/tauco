package migrations_test

import (
	"io/fs"
	"reflect"
	"sort"
	"strings"
	"testing"

	"github.com/ilhamnugraha8944/tauco/backend/migrations"
)

func TestEmbeddedMigrationSetIsPairedAndContiguous(t *testing.T) {
	t.Parallel()

	entries, err := fs.ReadDir(migrations.FS(), ".")
	if err != nil {
		t.Fatalf("read embedded migrations: %v", err)
	}

	var names []string
	for _, entry := range entries {
		if entry.IsDir() || !strings.HasSuffix(entry.Name(), ".sql") {
			continue
		}
		names = append(names, entry.Name())
		content, readErr := fs.ReadFile(migrations.FS(), entry.Name())
		if readErr != nil {
			t.Fatalf("read %s: %v", entry.Name(), readErr)
		}
		if len(strings.TrimSpace(string(content))) == 0 {
			t.Errorf("%s is empty", entry.Name())
		}
	}
	sort.Strings(names)

	want := []string{
		"000001_database_privileges.down.sql",
		"000001_database_privileges.up.sql",
		"000002_core_schema.down.sql",
		"000002_core_schema.up.sql",
		"000003_integrity_triggers.down.sql",
		"000003_integrity_triggers.up.sql",
		"000004_media_worker_permissions.down.sql",
		"000004_media_worker_permissions.up.sql",
	}
	if !reflect.DeepEqual(names, want) {
		t.Fatalf("embedded migration names = %v, want %v", names, want)
	}
}

func TestMigrationsNeverGenerateApplicationUUIDsOrUseAutoMigrate(t *testing.T) {
	t.Parallel()

	err := fs.WalkDir(migrations.FS(), ".", func(path string, entry fs.DirEntry, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}
		if entry.IsDir() || !strings.HasSuffix(path, ".sql") {
			return nil
		}
		content, err := fs.ReadFile(migrations.FS(), path)
		if err != nil {
			return err
		}
		lower := strings.ToLower(string(content))
		for _, forbidden := range []string{
			"automigrate",
			"gen_random_uuid",
			"uuid_generate",
			"create table public.",
		} {
			if strings.Contains(lower, forbidden) {
				t.Errorf("%s contains forbidden database pattern %q", path, forbidden)
			}
		}
		return nil
	})
	if err != nil {
		t.Fatalf("walk embedded migrations: %v", err)
	}
}
