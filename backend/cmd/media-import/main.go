// Command media-import is the internal Phase 1B ingestion boundary. It does
// not expose an HTTP upload route.
package main

import (
	"context"
	"flag"
	"fmt"
	"os"

	mediaapp "github.com/ilhamnugraha8944/tauco/backend/internal/media/application"
	mediaprocessor "github.com/ilhamnugraha8944/tauco/backend/internal/media/processor"
	mediarepo "github.com/ilhamnugraha8944/tauco/backend/internal/media/repository"
	mediastorage "github.com/ilhamnugraha8944/tauco/backend/internal/media/storage"
	"github.com/ilhamnugraha8944/tauco/backend/internal/platform/database"
)

func main() { os.Exit(run()) }

func run() int {
	var path, alt, root string
	var decorative bool
	flag.StringVar(&path, "file", "", "JPEG, PNG, or static WebP source")
	flag.StringVar(&alt, "alt", "", "descriptive alt text")
	flag.BoolVar(&decorative, "decorative", false, "mark media decorative")
	flag.StringVar(&root, "storage-root", os.Getenv("MEDIA_LOCAL_ROOT"), "private local object root")
	flag.Parse()
	if path == "" || root == "" {
		_, _ = fmt.Fprintln(os.Stderr, "--file and --storage-root are required")
		return 2
	}
	info, err := os.Stat(path)
	if err != nil || info.Size() < 1 || info.Size() > mediaapp.MaxUploadBytes {
		_, _ = fmt.Fprintln(os.Stderr, "media file must exist and be at most 10 MiB")
		return 2
	}
	source, err := os.ReadFile(path)
	if err != nil {
		_, _ = fmt.Fprintln(os.Stderr, "cannot read media file")
		return 1
	}
	ctx := context.Background()
	config, err := database.LoadRuntimeConfig(os.LookupEnv)
	if err != nil {
		_, _ = fmt.Fprintln(os.Stderr, "database configuration error")
		return 1
	}
	db, err := database.OpenGORM(ctx, config)
	if err != nil {
		_, _ = fmt.Fprintln(os.Stderr, "database initialization failed")
		return 1
	}
	sqlDB, _ := db.DB()
	defer func() { _ = sqlDB.Close() }()
	repository, _ := mediarepo.NewPostgres(db)
	store, err := mediastorage.NewLocal(root)
	if err != nil {
		_, _ = fmt.Fprintln(os.Stderr, "media storage initialization failed")
		return 1
	}
	ingestor, _ := mediaapp.NewIngestor(repository, store, mediaprocessor.Image{})
	assetID, replayed, err := ingestor.Ingest(ctx, source, alt, decorative)
	if err != nil {
		_, _ = fmt.Fprintln(os.Stderr, "media ingestion failed")
		return 1
	}
	_, _ = fmt.Fprintf(os.Stdout, "mediaAssetId=%s replayed=%t\n", assetID, replayed)
	return 0
}
