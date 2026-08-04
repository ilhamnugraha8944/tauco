// Command ops performs narrowly scoped local operational actions.
package main

import (
	"context"
	"fmt"
	"os"
	"regexp"
	"time"

	"github.com/google/uuid"
	"github.com/ilhamnugraha8944/tauco/backend/internal/contract/requestmeta"
	jobsrepo "github.com/ilhamnugraha8944/tauco/backend/internal/jobs/repository"
	platformcache "github.com/ilhamnugraha8944/tauco/backend/internal/platform/cache"
	"github.com/ilhamnugraha8944/tauco/backend/internal/platform/database"
)

var cacheTag = regexp.MustCompile(`^[a-z0-9][a-z0-9:-]{0,127}$`)

func main() { os.Exit(run()) }

func run() int {
	if len(os.Args) < 3 {
		return usage()
	}
	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()
	switch os.Args[1] {
	case "job-replay":
		if len(os.Args) != 4 || !requestmeta.Valid(os.Args[3]) {
			return usage()
		}
		if _, err := uuid.Parse(os.Args[2]); err != nil {
			return usage()
		}
		config, err := database.LoadRuntimeConfig(os.LookupEnv)
		if err != nil {
			return fail("invalid database configuration")
		}
		db, err := database.OpenGORM(ctx, config)
		if err != nil {
			return fail("database unavailable")
		}
		sqlDB, _ := db.DB()
		defer func() { _ = sqlDB.Close() }()
		repository, _ := jobsrepo.NewPostgresRepository(db)
		if err := repository.Replay(ctx, os.Args[2], os.Args[3]); err != nil {
			return fail("job replay failed")
		}
	case "cache-purge":
		for _, tag := range os.Args[2:] {
			if !cacheTag.MatchString(tag) {
				return usage()
			}
		}
		store, err := platformcache.NewRedis(os.Getenv("REDIS_URL"))
		if err != nil {
			return fail("invalid Redis configuration")
		}
		defer func() { _ = store.Close() }()
		if err := platformcache.Invalidate(ctx, store, os.Args[2:]...); err != nil {
			return fail("cache purge failed")
		}
	default:
		return usage()
	}
	_, _ = fmt.Fprintln(os.Stdout, "operation completed")
	return 0
}

func usage() int {
	_, _ = fmt.Fprintln(os.Stderr, "usage: ops <job-replay JOB_ID REQUEST_ID|cache-purge TAG...>")
	return 2
}

func fail(message string) int {
	_, _ = fmt.Fprintln(os.Stderr, message)
	return 1
}
