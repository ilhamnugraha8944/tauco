package architecture_test

import (
	"go/parser"
	"go/token"
	"io/fs"
	"path/filepath"
	"strconv"
	"strings"
	"testing"
)

var frameworkImports = []string{
	"github.com/gin-gonic/gin",
	"github.com/redis/go-redis",
	"github.com/aws/aws-sdk-go-v2",
	"gorm.io/",
	"go.uber.org/zap",
}

const modulePath = "github.com/ilhamnugraha8944/tauco/backend"

func TestCleanArchitectureDependencyBoundaries(t *testing.T) {
	t.Parallel()

	root := filepath.Clean(filepath.Join("..", ".."))
	err := filepath.WalkDir(root, func(path string, entry fs.DirEntry, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}
		if entry.IsDir() {
			return nil
		}
		if filepath.Ext(path) != ".go" {
			return nil
		}

		relativePath, err := filepath.Rel(root, path)
		if err != nil {
			return err
		}
		normalizedPath := "/" + filepath.ToSlash(relativePath)

		file, err := parser.ParseFile(token.NewFileSet(), path, nil, parser.ImportsOnly)
		if err != nil {
			return err
		}

		for _, imported := range file.Imports {
			importPath, err := strconv.Unquote(imported.Path.Value)
			if err != nil {
				return err
			}

			switch {
			case strings.Contains(normalizedPath, "/domain/"):
				assertImportDoesNotMatch(t, normalizedPath, importPath, frameworkImports)
			case strings.Contains(normalizedPath, "/application/"):
				assertImportDoesNotMatch(t, normalizedPath, importPath, frameworkImports)
			case strings.Contains(normalizedPath, "/delivery/"):
				assertImportDoesNotMatch(t, normalizedPath, importPath, []string{"gorm.io/"})
			}

			if importMatches(importPath, "gorm.io/") &&
				!strings.Contains(normalizedPath, "/repository/") &&
				!strings.Contains(normalizedPath, "/platform/database/") {
				t.Errorf("%s imports %q; GORM is restricted to repository/database infrastructure", normalizedPath, importPath)
			}

			if reason := architectureViolation(normalizedPath, importPath); reason != "" {
				t.Errorf("%s imports %q; %s", normalizedPath, importPath, reason)
			}
		}

		return nil
	})
	if err != nil {
		t.Fatalf("walk backend packages: %v", err)
	}
}

func TestArchitectureViolation(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name       string
		filePath   string
		importPath string
		want       bool
	}{
		{
			name:       "domain may import standard library",
			filePath:   "/internal/catalog/domain/product.go",
			importPath: "time",
		},
		{
			name:       "domain may import another domain package",
			filePath:   "/internal/catalog/domain/product.go",
			importPath: modulePath + "/internal/content/domain",
		},
		{
			name:       "domain cannot import application",
			filePath:   "/internal/catalog/domain/product.go",
			importPath: modulePath + "/internal/catalog/application",
			want:       true,
		},
		{
			name:       "application may import domain",
			filePath:   "/internal/catalog/application/list_products.go",
			importPath: modulePath + "/internal/catalog/domain",
		},
		{
			name:       "application cannot import repository",
			filePath:   "/internal/catalog/application/list_products.go",
			importPath: modulePath + "/internal/catalog/repository",
			want:       true,
		},
		{
			name:       "delivery cannot import platform",
			filePath:   "/internal/catalog/delivery/http/handler.go",
			importPath: modulePath + "/internal/platform/database",
			want:       true,
		},
		{
			name:       "repository may import application port",
			filePath:   "/internal/catalog/repository/postgres.go",
			importPath: modulePath + "/internal/catalog/application",
		},
		{
			name:       "internal package cannot import command",
			filePath:   "/internal/catalog/application/list_products.go",
			importPath: modulePath + "/cmd/api",
			want:       true,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			got := architectureViolation(test.filePath, test.importPath) != ""
			if got != test.want {
				t.Fatalf("architectureViolation() = %t, want %t", got, test.want)
			}
		})
	}
}

func architectureViolation(filePath, importPath string) string {
	if strings.HasPrefix(importPath, modulePath+"/cmd/") {
		return "internal packages must not import command packages"
	}
	if !strings.HasPrefix(importPath, modulePath+"/internal/") {
		return ""
	}

	var forbidden []string
	switch {
	case strings.Contains(filePath, "/domain/"):
		forbidden = []string{
			"application",
			"delivery",
			"repository",
			"platform",
			"composition",
		}
	case strings.Contains(filePath, "/application/"):
		forbidden = []string{
			"delivery",
			"repository",
			"platform",
			"composition",
		}
	case strings.Contains(filePath, "/delivery/"):
		forbidden = []string{
			"repository",
			"platform",
			"composition",
		}
	case strings.Contains(filePath, "/repository/"):
		forbidden = []string{
			"delivery",
			"composition",
		}
	}

	for _, layer := range forbidden {
		if containsPathSegment(importPath, layer) {
			return "dependency points toward an outer Clean Architecture layer"
		}
	}
	return ""
}

func containsPathSegment(path, segment string) bool {
	return strings.Contains(path, "/"+segment+"/") ||
		strings.HasSuffix(path, "/"+segment)
}

func assertImportDoesNotMatch(t *testing.T, filePath, importPath string, forbidden []string) {
	t.Helper()

	for _, prefix := range forbidden {
		if importMatches(importPath, prefix) {
			t.Errorf("%s imports forbidden dependency %q", filePath, importPath)
		}
	}
}

func importMatches(importPath, prefix string) bool {
	return importPath == strings.TrimSuffix(prefix, "/") || strings.HasPrefix(importPath, prefix)
}
