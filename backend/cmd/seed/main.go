// Command seed imports the committed Phase 1A content through the
// deterministic PostgreSQL SeedStore.
package main

import (
	"context"
	"flag"
	"fmt"
	"io"
	"os"
	"os/signal"
	"syscall"

	contentapp "github.com/ilhamnugraha8944/tauco/backend/internal/content/application"
	"github.com/ilhamnugraha8944/tauco/backend/internal/content/importer"
	contentrepo "github.com/ilhamnugraha8944/tauco/backend/internal/content/repository"
	platformdb "github.com/ilhamnugraha8944/tauco/backend/internal/platform/database"
)

func main() {
	ctx, stop := signal.NotifyContext(
		context.Background(),
		os.Interrupt,
		syscall.SIGTERM,
	)
	defer stop()
	os.Exit(run(ctx, os.Args[1:], os.LookupEnv, os.Stdout, os.Stderr))
}

func run(
	ctx context.Context,
	args []string,
	lookup platformdb.LookupEnv,
	stdout io.Writer,
	stderr io.Writer,
) int {
	flags := flag.NewFlagSet("seed", flag.ContinueOnError)
	flags.SetOutput(io.Discard)
	contentDirectory := flags.String(
		"content-dir",
		"",
		"directory containing the committed Phase 1A JSON files",
	)
	if err := flags.Parse(args); err != nil ||
		flags.NArg() != 0 ||
		*contentDirectory == "" {
		_, _ = fmt.Fprintln(
			stderr,
			"usage: seed --content-dir <phase-1a-content-directory>",
		)
		return 2
	}

	config, err := platformdb.LoadMigrationConfig(lookup)
	if err != nil {
		_, _ = fmt.Fprintf(stderr, "seed configuration error: %v\n", err)
		return 1
	}

	// Parse and validate all source content before opening a database
	// connection or starting a transaction.
	plan, err := importer.LoadPhase1ADirectory(*contentDirectory)
	if err != nil {
		_, _ = fmt.Fprintf(stderr, "Phase 1A content validation failed: %v\n", err)
		return 1
	}

	database, err := platformdb.OpenMigrationGORM(ctx, config.MigrationURL)
	if err != nil {
		_, _ = fmt.Fprintln(stderr, "seed database connection failed")
		return 1
	}
	sqlDatabase, err := database.DB()
	if err != nil {
		_, _ = fmt.Fprintln(stderr, "seed database pool initialization failed")
		return 1
	}
	defer func() {
		if err := sqlDatabase.Close(); err != nil {
			_, _ = fmt.Fprintln(stderr, "seed database cleanup failed")
		}
	}()

	repository, err := contentrepo.NewPostgresRepository(database)
	if err != nil {
		_, _ = fmt.Fprintln(stderr, "seed repository initialization failed")
		return 1
	}
	result, err := contentapp.ApplyPhase1A(ctx, repository, plan)
	if err != nil {
		_, _ = fmt.Fprintf(stderr, "Phase 1A seed failed: %v\n", err)
		return 1
	}
	_, _ = fmt.Fprintf(
		stdout,
		"Phase 1A seed complete: inserted=%d unchanged=%d\n",
		result.Inserted,
		result.Unchanged,
	)
	return 0
}
