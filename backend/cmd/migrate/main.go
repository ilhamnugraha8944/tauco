// Command migrate applies versioned PostgreSQL migrations.
package main

import (
	"context"
	"errors"
	"fmt"
	"os"
	"os/signal"
	"strconv"
	"syscall"

	platformdb "github.com/ilhamnugraha8944/tauco/backend/internal/platform/database"
)

func main() {
	os.Exit(run(os.Args[1:], os.LookupEnv))
}

func run(args []string, lookup platformdb.LookupEnv) int {
	if len(args) == 0 {
		printUsage()
		return 2
	}

	cfg, err := platformdb.LoadMigrationConfig(lookup)
	if err != nil {
		_, _ = fmt.Fprintf(os.Stderr, "migration configuration error: %v\n", err)
		return 1
	}

	ctx, stop := signal.NotifyContext(
		context.Background(),
		os.Interrupt,
		syscall.SIGTERM,
	)
	defer stop()

	command := args[0]
	if command == "up" {
		if len(args) != 1 {
			printUsage()
			return 2
		}
		if err := platformdb.BootstrapRoles(ctx, cfg); err != nil {
			_, _ = fmt.Fprintf(os.Stderr, "database bootstrap failed: %v\n", err)
			return 1
		}
	}

	migrator, err := platformdb.NewMigrator(cfg.MigrationURL)
	if err != nil {
		_, _ = fmt.Fprintln(os.Stderr, "migration initialization failed")
		return 1
	}
	defer func() {
		if err := migrator.Close(); err != nil {
			_, _ = fmt.Fprintln(os.Stderr, "migration cleanup failed")
		}
	}()

	switch command {
	case "up":
		err = migrator.Up()
	case "down":
		err = runDown(migrator, args[1:])
	case "steps":
		err = runSteps(migrator, args[1:])
	case "version":
		err = runVersion(migrator, args[1:])
	default:
		printUsage()
		return 2
	}
	if err != nil {
		_, _ = fmt.Fprintf(os.Stderr, "migration command failed: %v\n", err)
		return 1
	}
	return 0
}

type migrationRunner interface {
	DownOne() error
	DownAll() error
	Steps(count int) error
	Version() (version uint, dirty bool, err error)
}

func runDown(migrator migrationRunner, args []string) error {
	switch {
	case len(args) == 0:
		return migrator.DownOne()
	case len(args) == 1 && args[0] == "--all":
		return migrator.DownAll()
	default:
		return errors.New("down accepts only the explicit --all option")
	}
}

func runSteps(migrator migrationRunner, args []string) error {
	if len(args) != 1 {
		return errors.New("steps requires one non-zero integer")
	}
	count, err := strconv.Atoi(args[0])
	if err != nil || count == 0 {
		return errors.New("steps requires one non-zero integer")
	}
	return migrator.Steps(count)
}

func runVersion(migrator migrationRunner, args []string) error {
	if len(args) != 0 {
		return errors.New("version does not accept arguments")
	}
	version, dirty, err := migrator.Version()
	if err != nil {
		return err
	}
	_, err = fmt.Fprintf(os.Stdout, "version=%d dirty=%t\n", version, dirty)
	return err
}

func printUsage() {
	_, _ = fmt.Fprintln(
		os.Stderr,
		"usage: migrate <up|down [--all]|steps <non-zero integer>|version>",
	)
}
