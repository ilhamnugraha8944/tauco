// Package migrations exposes the immutable SQL migration set embedded in the
// backend binary. Keeping migrations embedded makes the migration CLI behave
// identically in local development, CI, and a production container.
package migrations

import (
	"embed"
	"io/fs"
)

// files contains every versioned up/down SQL migration.
//
//go:embed *.sql
var files embed.FS

// FS returns a read-only view of the embedded migration directory.
func FS() fs.FS {
	return files
}
