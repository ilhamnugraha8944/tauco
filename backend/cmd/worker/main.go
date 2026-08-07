// Command worker processes durable PostgreSQL jobs.
package main

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"net"
	"os"
	"os/signal"
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
	platformconfig "github.com/ilhamnugraha8944/tauco/backend/internal/platform/config"
	"github.com/ilhamnugraha8944/tauco/backend/internal/platform/database"
	mail "github.com/ilhamnugraha8944/tauco/backend/internal/platform/email"
	"golang.org/x/sync/errgroup"
)

func main() { os.Exit(run()) }

func run() int {
	mode, err := workerMode(os.Args[1:])
	if err != nil {
		_, _ = fmt.Fprintln(os.Stderr, "usage: worker [--check|--once|--cleanup-media]")
		return 2
	}
	appConfig, err := platformconfig.Load()
	if err != nil {
		_, _ = fmt.Fprintln(os.Stderr, "worker application configuration error")
		return 1
	}
	deployment, err := platformconfig.LoadDeployment(os.LookupEnv, appConfig.Environment)
	if err != nil {
		_, _ = fmt.Fprintln(os.Stderr, "worker deployment configuration error")
		return 1
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

	mediaStore, mediaHealth, quarantineStore, err := openMediaStorage(deployment)
	if err != nil {
		_, _ = fmt.Fprintln(os.Stderr, "worker media storage configuration error")
		return 1
	}
	mediaRepository, _ := mediarepo.NewPostgres(db)
	mediaIngestor, _ := mediaapp.NewIngestor(mediaRepository, mediaStore, mediaprocessor.Image{})
	mediaVariantHandler, _ := mediaapp.NewVariantHandler(mediaRepository, mediaStore, mediaprocessor.Image{})

	handlers := map[string]jobsapp.Handler{
		"media.generate_variants": func(ctx context.Context, job jobsdomain.Job) error {
			return mediaVariantHandler.Handle(ctx, job.Payload)
		},
	}
	cacheHandler, _ := contentapp.NewCacheInvalidationHandler(cacheStore)
	handlers["content.invalidate_cache"] = func(ctx context.Context, job jobsdomain.Job) error {
		return cacheHandler.Handle(ctx, job.Payload)
	}
	if quarantineStore != nil {
		uploadHandler, uploadErr := mediaapp.NewUploadHandler(mediaRepository, quarantineStore, mediaIngestor)
		if uploadErr != nil {
			return 1
		}
		handlers["media.ingest_upload"] = func(ctx context.Context, job jobsdomain.Job) error {
			return uploadHandler.Handle(ctx, job.Payload)
		}
	}

	var contactRetention retentionStore
	var smtpHost string
	var smtpPort int
	if deployment.ContactAPIEnabled {
		contactStore, storeErr := contactrepo.NewPostgresStore(db)
		if storeErr != nil {
			return 1
		}
		contactRetention = contactStore
		smtpHost = os.Getenv("SMTP_HOST")
		smtpPort, err = strconv.Atoi(os.Getenv("SMTP_PORT"))
		if err != nil || smtpPort < 1 || smtpPort > 65535 {
			_, _ = fmt.Fprintln(os.Stderr, "worker email configuration error")
			return 1
		}
		sender, senderErr := mail.NewSMTPSender(mail.SMTPConfig{
			Host: smtpHost, Port: smtpPort,
			From: os.Getenv("SMTP_FROM"), To: os.Getenv("SMTP_TO"),
		})
		if senderErr != nil {
			_, _ = fmt.Fprintln(os.Stderr, "worker email configuration error")
			return 1
		}
		contactHandlers, _ := contactapp.NewJobHandlers(contactStore, sender)
		handlers["contact.email_notification"] = func(ctx context.Context, job jobsdomain.Job) error {
			return contactHandlers.Email(ctx, job.Payload, job.ID)
		}
		handlers["contact.activity_log"] = func(ctx context.Context, job jobsdomain.Job) error {
			return contactHandlers.Activity(ctx, job.Payload)
		}
	}

	if mode == "--check" {
		if err := checkReadiness(ctx, sqlDB, cacheStore, mediaHealth, contactRetention != nil, smtpHost, smtpPort); err != nil {
			_, _ = fmt.Fprintf(os.Stderr, "worker is not ready: %v\n", err)
			return 1
		}
		_, _ = fmt.Fprintln(os.Stdout, "worker ready")
		return 0
	}
	if mode == "--cleanup-media" {
		cleanupStore, ok := mediaStore.(mediaapp.ObjectDeleter)
		if !ok {
			_, _ = fmt.Fprintln(os.Stderr, "media cleanup storage is unavailable")
			return 2
		}
		cleanup, _ := mediaapp.NewUploadCleanup(mediaRepository, cleanupStore)
		cleanupContext, cancel := context.WithTimeout(ctx, 25*time.Second)
		defer cancel()
		count, cleanupErr := cleanup.RunOnce(cleanupContext, 25)
		if cleanupErr != nil {
			_, _ = fmt.Fprintln(os.Stderr, "media cleanup failed")
			return 1
		}
		_, _ = fmt.Fprintf(os.Stdout, "media cleanup complete: %d intent(s)\n", count)
		return 0
	}

	jobRepository, _ := jobsrepo.NewPostgresRepository(db)
	hostname, _ := os.Hostname()
	workerConfig := jobsapp.DefaultConfig(fmt.Sprintf("%s-%d", hostname, os.Getpid()))
	if mode == "--once" {
		workerConfig.BatchSize = 1
		workerConfig.Workers = 1
		workerConfig.ChannelCapacity = 1
		workerConfig.Lease = 45 * time.Second
		workerConfig.Heartbeat = 10 * time.Second
	}
	worker, err := jobsapp.NewWorker(jobRepository, handlers, workerConfig)
	if err != nil {
		return 1
	}
	if mode == "--once" {
		oneShotContext, cancel := context.WithTimeout(ctx, 25*time.Second)
		defer cancel()
		if err := worker.RunOnce(oneShotContext); err != nil {
			_, _ = fmt.Fprintln(os.Stderr, "worker one-shot failed")
			return 1
		}
		_, _ = fmt.Fprintln(os.Stdout, "worker one-shot complete")
		return 0
	}

	group, groupContext := errgroup.WithContext(ctx)
	group.Go(func() error { return worker.Run(groupContext) })
	if contactRetention != nil {
		group.Go(func() error { return runRetention(groupContext, contactRetention) })
	}
	if err := group.Wait(); err != nil {
		_, _ = fmt.Fprintf(os.Stderr, "worker stopped unexpectedly: %v\n", err)
		return 1
	}
	return 0
}

func workerMode(arguments []string) (string, error) {
	if len(arguments) == 0 {
		return "", nil
	}
	if len(arguments) == 1 {
		switch arguments[0] {
		case "--check", "--once", "--cleanup-media":
			return arguments[0], nil
		}
	}
	return "", errors.New("invalid worker mode")
}

func openMediaStorage(deployment platformconfig.DeploymentConfig) (mediaapp.ObjectStore, mediaapp.HealthStore, mediaapp.QuarantineStore, error) {
	switch deployment.MediaStorageDriver {
	case "local":
		store, err := mediastorage.NewLocal(os.Getenv("MEDIA_LOCAL_ROOT"))
		return store, store, nil, err
	case "s3":
		store, err := mediastorage.NewS3FromConfig(mediastorage.S3Config{
			Endpoint: deployment.MediaS3Endpoint, Region: deployment.MediaS3Region,
			Bucket: deployment.MediaS3Bucket, Prefix: deployment.MediaS3Prefix,
			AccessKeyID: deployment.MediaS3AccessKeyID, SecretAccessKey: deployment.MediaS3SecretKey,
		})
		return store, store, store, err
	default:
		return nil, nil, nil, errors.New("unsupported media storage driver")
	}
}

type retentionStore interface {
	PurgeExpired(context.Context, time.Time, int) (int64, error)
}

func runRetention(ctx context.Context, store retentionStore) error {
	ticker := time.NewTicker(24 * time.Hour)
	defer ticker.Stop()
	for {
		if _, err := store.PurgeExpired(ctx, time.Now().UTC(), 100); err != nil {
			return err
		}
		select {
		case <-ctx.Done():
			return nil
		case <-ticker.C:
		}
	}
}

func checkReadiness(
	ctx context.Context,
	database *sql.DB,
	cache interface{ Ping(context.Context) error },
	storage mediaapp.HealthStore,
	contactEnabled bool,
	smtpHost string,
	smtpPort int,
) error {
	checkContext, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()
	if err := database.PingContext(checkContext); err != nil {
		return fmt.Errorf("postgres: %w", err)
	}
	if contactEnabled {
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
			return errors.New("contact retention privilege is unavailable")
		}
	}
	if err := cache.Ping(checkContext); err != nil {
		return fmt.Errorf("redis: %w", err)
	}
	if storage == nil {
		return errors.New("media storage: adapter is unavailable")
	}
	if err := storage.Health(checkContext); err != nil {
		return fmt.Errorf("media storage: %w", err)
	}
	if !contactEnabled {
		return nil
	}
	connection, err := (&net.Dialer{}).DialContext(
		checkContext, "tcp", net.JoinHostPort(smtpHost, strconv.Itoa(smtpPort)),
	)
	if err != nil {
		return fmt.Errorf("smtp: %w", err)
	}
	return connection.Close()
}
