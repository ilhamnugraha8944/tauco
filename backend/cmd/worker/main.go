// Command worker processes durable PostgreSQL jobs.
package main

import (
	"context"
	"database/sql"
	"fmt"
	"net"
	"os"
	"os/signal"
	"path/filepath"
	"strconv"
	"syscall"
	"time"

	contactapp "github.com/ilhamnugraha8944/tauco/backend/internal/contact/application"
	contactrepo "github.com/ilhamnugraha8944/tauco/backend/internal/contact/repository"
	contentapp "github.com/ilhamnugraha8944/tauco/backend/internal/content/application"
	jobsapp "github.com/ilhamnugraha8944/tauco/backend/internal/jobs/application"
	jobsdomain "github.com/ilhamnugraha8944/tauco/backend/internal/jobs/domain"
	jobsrepo "github.com/ilhamnugraha8944/tauco/backend/internal/jobs/repository"
	mediaapp "github.com/ilhamnugraha8944/tauco/backend/internal/media/application"
	mediaprocessor "github.com/ilhamnugraha8944/tauco/backend/internal/media/processor"
	mediarepo "github.com/ilhamnugraha8944/tauco/backend/internal/media/repository"
	mediastorage "github.com/ilhamnugraha8944/tauco/backend/internal/media/storage"
	platformcache "github.com/ilhamnugraha8944/tauco/backend/internal/platform/cache"
	"github.com/ilhamnugraha8944/tauco/backend/internal/platform/database"
	mail "github.com/ilhamnugraha8944/tauco/backend/internal/platform/email"
	"golang.org/x/sync/errgroup"
)

func main() { os.Exit(run()) }

func run() int {
	checkOnly := len(os.Args) == 2 && os.Args[1] == "--check"
	if len(os.Args) > 1 && !checkOnly {
		_, _ = fmt.Fprintln(os.Stderr, "usage: worker [--check]")
		return 2
	}
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()
	databaseConfig, err := database.LoadRuntimeConfig(os.LookupEnv)
	if err != nil {
		_, _ = fmt.Fprintln(os.Stderr, "worker database configuration error")
		return 1
	}
	db, err := database.OpenGORM(ctx, databaseConfig)
	if err != nil {
		_, _ = fmt.Fprintln(os.Stderr, "worker database initialization failed")
		return 1
	}
	sqlDB, _ := db.DB()
	defer func() { _ = sqlDB.Close() }()
	cacheStore, err := platformcache.NewRedis(os.Getenv("REDIS_URL"))
	if err != nil {
		_, _ = fmt.Fprintln(os.Stderr, "worker cache configuration error")
		return 1
	}
	defer func() { _ = cacheStore.Close() }()

	contactStore, _ := contactrepo.NewPostgresStore(db)
	port, _ := strconv.Atoi(os.Getenv("SMTP_PORT"))
	sender, err := mail.NewSMTPSender(mail.SMTPConfig{
		Host: os.Getenv("SMTP_HOST"), Port: port,
		From: os.Getenv("SMTP_FROM"), To: os.Getenv("SMTP_TO"),
	})
	if err != nil {
		_, _ = fmt.Fprintln(os.Stderr, "worker email configuration error")
		return 1
	}
	contactHandlers, _ := contactapp.NewJobHandlers(contactStore, sender)
	mediaRepository, _ := mediarepo.NewPostgres(db)
	mediaStore, err := mediastorage.NewLocal(os.Getenv("MEDIA_LOCAL_ROOT"))
	if err != nil {
		_, _ = fmt.Fprintln(os.Stderr, "worker media storage configuration error")
		return 1
	}
	if checkOnly {
		if err := checkReadiness(ctx, sqlDB, cacheStore, os.Getenv("MEDIA_LOCAL_ROOT"), os.Getenv("SMTP_HOST"), port); err != nil {
			_, _ = fmt.Fprintln(os.Stderr, "worker is not ready")
			return 1
		}
		_, _ = fmt.Fprintln(os.Stdout, "worker ready")
		return 0
	}
	mediaHandler, _ := mediaapp.NewVariantHandler(mediaRepository, mediaStore, mediaprocessor.Image{})
	cacheHandler, _ := contentapp.NewCacheInvalidationHandler(cacheStore)
	jobRepository, _ := jobsrepo.NewPostgresRepository(db)
	hostname, _ := os.Hostname()
	config := jobsapp.DefaultConfig(fmt.Sprintf("%s-%d", hostname, os.Getpid()))
	worker, err := jobsapp.NewWorker(jobRepository, map[string]jobsapp.Handler{
		"contact.email_notification": func(ctx context.Context, job jobsdomain.Job) error {
			return contactHandlers.Email(ctx, job.Payload, job.ID)
		},
		"contact.activity_log": func(ctx context.Context, job jobsdomain.Job) error {
			return contactHandlers.Activity(ctx, job.Payload)
		},
		"media.generate_variants": func(ctx context.Context, job jobsdomain.Job) error {
			return mediaHandler.Handle(ctx, job.Payload)
		},
		"content.invalidate_cache": func(ctx context.Context, job jobsdomain.Job) error {
			return cacheHandler.Handle(ctx, job.Payload)
		},
	}, config)
	if err != nil {
		return 1
	}
	group, groupContext := errgroup.WithContext(ctx)
	group.Go(func() error { return worker.Run(groupContext) })
	group.Go(func() error { return runRetention(groupContext, contactStore) })
	if err := group.Wait(); err != nil {
		_, _ = fmt.Fprintf(os.Stderr, "worker stopped unexpectedly: %v\n", err)
		return 1
	}
	return 0
}

type retentionStore interface {
	PurgeExpired(context.Context, time.Time, int) (int64, error)
}

func runRetention(ctx context.Context, store retentionStore) error {
	ticker := time.NewTicker(24 * time.Hour)
	defer ticker.Stop()
	for {
		for {
			deleted, err := store.PurgeExpired(ctx, time.Now().UTC(), 100)
			if err != nil {
				return err
			}
			if deleted < 100 {
				break
			}
		}
		select {
		case <-ctx.Done():
			return nil
		case <-ticker.C:
		}
	}
}

func checkReadiness(ctx context.Context, database *sql.DB, cache interface{ Ping(context.Context) error }, mediaRoot, smtpHost string, smtpPort int) error {
	checkContext, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()
	if err := database.PingContext(checkContext); err != nil {
		return err
	}
	var canPurgeExpiredContacts bool
	if err := database.QueryRowContext(checkContext, `
		SELECT has_function_privilege(
			current_user,
			'tauco_app.tauco_purge_expired_contact_messages(timestamptz,integer)',
			'EXECUTE'
		)
	`).Scan(&canPurgeExpiredContacts); err != nil {
		return fmt.Errorf("check contact retention privilege: %w", err)
	}
	if !canPurgeExpiredContacts {
		return fmt.Errorf("contact retention privilege is unavailable")
	}
	if err := cache.Ping(checkContext); err != nil {
		return err
	}
	root, err := filepath.Abs(mediaRoot)
	if err != nil {
		return err
	}
	probe, err := os.CreateTemp(root, ".readiness-*")
	if err != nil {
		return err
	}
	probePath := probe.Name()
	if err := probe.Close(); err != nil {
		_ = os.Remove(probePath)
		return err
	}
	if err := os.Remove(probePath); err != nil {
		return err
	}
	connection, err := (&net.Dialer{}).DialContext(
		checkContext, "tcp", net.JoinHostPort(smtpHost, strconv.Itoa(smtpPort)),
	)
	if err != nil {
		return err
	}
	return connection.Close()
}
