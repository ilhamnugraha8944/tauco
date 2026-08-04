package database

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"image"
	"image/color"
	"image/png"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/jackc/pgx/v5"
	"gorm.io/gorm"

	catalogapp "github.com/ilhamnugraha8944/tauco/backend/internal/catalog/application"
	catalogcursor "github.com/ilhamnugraha8944/tauco/backend/internal/catalog/delivery/cursor"
	catalogdomain "github.com/ilhamnugraha8944/tauco/backend/internal/catalog/domain"
	catalogrepo "github.com/ilhamnugraha8944/tauco/backend/internal/catalog/repository"
	contactapp "github.com/ilhamnugraha8944/tauco/backend/internal/contact/application"
	contactdomain "github.com/ilhamnugraha8944/tauco/backend/internal/contact/domain"
	contactrepo "github.com/ilhamnugraha8944/tauco/backend/internal/contact/repository"
	contentapp "github.com/ilhamnugraha8944/tauco/backend/internal/content/application"
	contentdomain "github.com/ilhamnugraha8944/tauco/backend/internal/content/domain"
	"github.com/ilhamnugraha8944/tauco/backend/internal/content/importer"
	contentrepo "github.com/ilhamnugraha8944/tauco/backend/internal/content/repository"
	"github.com/ilhamnugraha8944/tauco/backend/internal/contract/requestmeta"
	"github.com/ilhamnugraha8944/tauco/backend/internal/delivery/api"
	jobsdomain "github.com/ilhamnugraha8944/tauco/backend/internal/jobs/domain"
	jobsrepo "github.com/ilhamnugraha8944/tauco/backend/internal/jobs/repository"
	mediaapp "github.com/ilhamnugraha8944/tauco/backend/internal/media/application"
	mediaprocessor "github.com/ilhamnugraha8944/tauco/backend/internal/media/processor"
	mediarepo "github.com/ilhamnugraha8944/tauco/backend/internal/media/repository"
	mediastorage "github.com/ilhamnugraha8944/tauco/backend/internal/media/storage"
)

const repositoryIntegrationLock int64 = 839_103_221_702

func TestPhase1ASeedAndPublishedRepositories(t *testing.T) {
	baseURL := strings.TrimSpace(os.Getenv("MIGRATION_TEST_DATABASE_URL"))
	if baseURL == "" {
		t.Skip("set MIGRATION_TEST_DATABASE_URL to run PostgreSQL repository integration")
	}

	ctx, cancel := context.WithTimeout(context.Background(), 120*time.Second)
	defer cancel()

	admin, err := pgx.Connect(ctx, baseURL)
	if err != nil {
		t.Fatalf("connect repository integration database: %v", err)
	}
	defer admin.Close(ctx)

	if _, err := admin.Exec(
		ctx,
		"SELECT pg_advisory_lock($1)",
		repositoryIntegrationLock,
	); err != nil {
		t.Fatalf("acquire repository integration lock: %v", err)
	}
	defer func() {
		_, _ = admin.Exec(
			context.Background(),
			"SELECT pg_advisory_unlock($1)",
			repositoryIntegrationLock,
		)
	}()

	suffix := fmt.Sprintf("%d", time.Now().UnixNano())
	databaseName := "tauco_repository_test_" + suffix
	runtimeRoleName := "tauco_repository_runtime_" + suffix
	if len(runtimeRoleName) > 63 {
		runtimeRoleName = runtimeRoleName[:63]
	}
	const runtimePassword = "B3-repository-runtime-test-password"

	databaseIdentifier := pgx.Identifier{databaseName}.Sanitize()
	if _, err := admin.Exec(ctx, "CREATE DATABASE "+databaseIdentifier); err != nil {
		t.Fatalf("create disposable repository database: %v", err)
	}
	defer func() {
		_, _ = admin.Exec(
			context.Background(),
			"DROP DATABASE IF EXISTS "+databaseIdentifier+" WITH (FORCE)",
		)
		_, _ = admin.Exec(
			context.Background(),
			"DROP ROLE IF EXISTS "+pgx.Identifier{runtimeRoleName}.Sanitize(),
		)
	}()

	migrationURL := replaceDatabaseAndUser(
		t,
		baseURL,
		databaseName,
		"",
		"",
	)
	runtimeURL := replaceDatabaseAndUser(
		t,
		baseURL,
		databaseName,
		runtimeRoleName,
		runtimePassword,
	)
	config := MigrationConfig{
		MigrationURL:   migrationURL,
		RuntimeURL:     runtimeURL,
		BootstrapRoles: true,
	}
	if err := BootstrapRoles(ctx, config); err != nil {
		t.Fatalf("BootstrapRoles() error = %v", err)
	}

	migrator, err := NewMigrator(migrationURL)
	if err != nil {
		t.Fatalf("NewMigrator() error = %v", err)
	}
	defer migrator.Close()
	if err := migrator.Up(); err != nil {
		t.Fatalf("migration Up() error = %v", err)
	}

	contentDirectory, err := filepath.Abs(
		filepath.Join("..", "..", "..", "..", "content"),
	)
	if err != nil {
		t.Fatalf("resolve content directory: %v", err)
	}
	plan, err := importer.LoadPhase1ADirectory(contentDirectory)
	if err != nil {
		t.Fatalf("LoadPhase1ADirectory() error = %v", err)
	}

	migrationGORM, err := OpenMigrationGORM(ctx, migrationURL)
	if err != nil {
		t.Fatalf("OpenMigrationGORM() error = %v", err)
	}
	migrationSQL, err := migrationGORM.DB()
	if err != nil {
		t.Fatalf("migration GORM pool: %v", err)
	}
	defer migrationSQL.Close()

	seedRepository, err := contentrepo.NewPostgresRepository(migrationGORM)
	if err != nil {
		t.Fatalf("NewPostgresRepository(migration) error = %v", err)
	}

	// A pre-existing natural key must conflict before any of the four page
	// inserts can commit.
	if err := migrationGORM.Exec(`
INSERT INTO tauco_app.products (
    id, slug, sort_order, created_at, updated_at
) VALUES (
    '019bfc80-0000-7000-8000-000000009801',
    'tauco-cap-badak',
    0,
    '2026-07-27T00:00:00Z',
    '2026-07-27T00:00:00Z'
)`).Error; err != nil {
		t.Fatalf("insert conflicting natural key fixture: %v", err)
	}
	if _, err := contentapp.ApplyPhase1A(
		ctx,
		seedRepository,
		plan,
	); !errors.Is(err, contentapp.ErrSeedConflict) {
		t.Fatalf("conflicting first seed error = %v, want ErrSeedConflict", err)
	}
	assertSeedCounts(t, migrationGORM, 0, 0, 1, 0)
	if err := migrationGORM.Exec(`
DELETE FROM tauco_app.products
WHERE id = '019bfc80-0000-7000-8000-000000009801'`).Error; err != nil {
		t.Fatalf("delete conflicting natural key fixture: %v", err)
	}

	first, err := contentapp.ApplyPhase1A(ctx, seedRepository, plan)
	if err != nil {
		t.Fatalf("first ApplyPhase1A() error = %v", err)
	}
	if first.Inserted != 5 || first.Unchanged != 0 {
		t.Fatalf("first ApplyPhase1A() = %+v, want 5 inserted", first)
	}
	assertSeedCounts(t, migrationGORM, 4, 4, 1, 1)
	assertDeterministicSeedTimestamps(t, migrationGORM, plan)

	replay, err := contentapp.ApplyPhase1A(ctx, seedRepository, plan)
	if err != nil {
		t.Fatalf("replay ApplyPhase1A() error = %v", err)
	}
	if replay.Inserted != 0 || replay.Unchanged != 5 {
		t.Fatalf("replay ApplyPhase1A() = %+v, want 5 unchanged", replay)
	}
	assertSeedCounts(t, migrationGORM, 4, 4, 1, 1)
	assertDeterministicSeedTimestamps(t, migrationGORM, plan)

	changedContent := cloneSeedPlan(plan)
	var productDocument map[string]any
	if err := json.Unmarshal(
		changedContent.Products[0].Revision.ContentJSON,
		&productDocument,
	); err != nil {
		t.Fatalf("decode product content for conflict: %v", err)
	}
	productDocument["summary"] = "Intentional deterministic conflict"
	rawChangedContent, err := json.Marshal(productDocument)
	if err != nil {
		t.Fatalf("encode product content conflict: %v", err)
	}
	canonicalChangedContent, changedChecksum, err :=
		contentdomain.CanonicalJSONChecksum(rawChangedContent)
	if err != nil {
		t.Fatalf("canonicalize product content conflict: %v", err)
	}
	changedContent.Products[0].Revision.ContentJSON = canonicalChangedContent
	changedContent.Products[0].Revision.Checksum = changedChecksum
	if _, err := contentapp.ApplyPhase1A(
		ctx,
		seedRepository,
		changedContent,
	); !errors.Is(err, contentapp.ErrSeedConflict) {
		t.Fatalf("changed-content seed error = %v, want ErrSeedConflict", err)
	}

	changedSortOrder := cloneSeedPlan(plan)
	changedSortOrder.Products[0].SortOrder++
	if _, err := contentapp.ApplyPhase1A(
		ctx,
		seedRepository,
		changedSortOrder,
	); !errors.Is(err, contentapp.ErrSeedConflict) {
		t.Fatalf("changed-sort seed error = %v, want ErrSeedConflict", err)
	}
	assertSeedCounts(t, migrationGORM, 4, 4, 1, 1)

	// UUID uniqueness is global to the deterministic manifest, not merely one
	// PostgreSQL table's primary key.
	if err := migrationGORM.Exec(`
INSERT INTO tauco_app.products (
    id, slug, sort_order, created_at, updated_at
) VALUES (
    ?::uuid,
    'cross-table-seed-collision',
    99,
    '2026-07-27T00:00:00Z',
    '2026-07-27T00:00:00Z'
)`,
		string(plan.Pages[0].Revision.EntityID),
	).Error; err != nil {
		t.Fatalf("insert cross-table UUID collision: %v", err)
	}
	if _, err := contentapp.ApplyPhase1A(
		ctx,
		seedRepository,
		plan,
	); !errors.Is(err, contentapp.ErrSeedConflict) {
		t.Fatalf("cross-table seed error = %v, want ErrSeedConflict", err)
	}
	if err := migrationGORM.Exec(`
DELETE FROM tauco_app.products
WHERE slug = 'cross-table-seed-collision'`).Error; err != nil {
		t.Fatalf("delete cross-table UUID collision: %v", err)
	}
	assertSeedCounts(t, migrationGORM, 4, 4, 1, 1)

	runtimeGORM, err := OpenGORM(ctx, RuntimeConfig{
		URL:               runtimeURL,
		MaxOpenConns:      2,
		MaxIdleConns:      1,
		ConnMaxLifetime:   30 * time.Minute,
		ConnMaxIdleTime:   5 * time.Minute,
		PreferSimpleQuery: true,
	})
	if err != nil {
		t.Fatalf("OpenGORM(runtime) error = %v", err)
	}
	runtimeSQL, err := runtimeGORM.DB()
	if err != nil {
		t.Fatalf("runtime GORM pool: %v", err)
	}
	defer runtimeSQL.Close()

	pageRepository, err := contentrepo.NewPostgresRepository(runtimeGORM)
	if err != nil {
		t.Fatalf("NewPostgresRepository(runtime) error = %v", err)
	}
	assertPublishedPageParity(t, ctx, pageRepository, plan)

	productRepository, err := catalogrepo.NewPostgresRepository(runtimeGORM)
	if err != nil {
		t.Fatalf("NewPostgresRepository(catalog) error = %v", err)
	}
	assertPublishedProductParity(t, ctx, productRepository, plan)
	assertPublicReadHTTP(t, pageRepository, productRepository)
	assertContactTransaction(t, ctx, runtimeGORM, migrationGORM)
	assertDurableJobClaims(t, ctx, runtimeGORM, migrationGORM)
	assertMediaPipeline(t, ctx, runtimeGORM)
	assertCatalogPaginationProbe(
		t,
		ctx,
		migrationGORM,
		productRepository,
		plan,
	)
}

func assertMediaPipeline(t *testing.T, ctx context.Context, runtimeDatabase *gorm.DB) {
	t.Helper()
	store, err := mediastorage.NewLocal(t.TempDir())
	if err != nil {
		t.Fatalf("NewLocal(media) error = %v", err)
	}
	repository, err := mediarepo.NewPostgres(runtimeDatabase)
	if err != nil {
		t.Fatalf("NewPostgres(media) error = %v", err)
	}
	processor := mediaprocessor.Image{}
	ingestor, _ := mediaapp.NewIngestor(repository, store, processor)

	fixture := image.NewNRGBA(image.Rect(0, 0, 700, 400))
	for y := range 400 {
		for x := range 700 {
			fixture.SetNRGBA(x, y, color.NRGBA{R: uint8(x % 255), G: uint8(y % 255), B: 80, A: 255})
		}
	}
	var encoded bytes.Buffer
	if err := png.Encode(&encoded, fixture); err != nil {
		t.Fatalf("encode media fixture: %v", err)
	}
	assetID, replayed, err := ingestor.Ingest(ctx, encoded.Bytes(), "Ilustrasi uji pipeline media", false)
	if err != nil || replayed || assetID == "" {
		t.Fatalf("first media ingest id=%q replayed=%t err=%v", assetID, replayed, err)
	}
	replayedID, replayed, err := ingestor.Ingest(ctx, encoded.Bytes(), "Ilustrasi uji pipeline media", false)
	if err != nil || !replayed || replayedID != assetID {
		t.Fatalf("media replay id=%q replayed=%t err=%v", replayedID, replayed, err)
	}

	handler, _ := mediaapp.NewVariantHandler(repository, store, processor)
	payload, _ := json.Marshal(map[string]string{"mediaAssetId": assetID})
	if err := handler.Handle(ctx, payload); err != nil {
		t.Fatalf("media variant handler: %v", err)
	}
	var status string
	if err := runtimeDatabase.Raw(`SELECT status FROM tauco_app.media_assets WHERE id = ?`, assetID).Scan(&status).Error; err != nil {
		t.Fatalf("read media status: %v", err)
	}
	if status != "ready" {
		t.Fatalf("media status = %q, want ready", status)
	}
	var variantCount, jobCount, activityCount int64
	for query, destination := range map[string]*int64{
		`SELECT count(*) FROM tauco_app.media_variants WHERE media_asset_id = '` + assetID + `'`:                  &variantCount,
		`SELECT count(*) FROM tauco_app.background_jobs WHERE idempotency_key = 'media.variants:` + assetID + `'`: &jobCount,
		`SELECT count(*) FROM tauco_app.activity_logs WHERE id = '` + assetID + `'`:                               &activityCount,
	} {
		if err := runtimeDatabase.Raw(query).Scan(destination).Error; err != nil {
			t.Fatalf("count media pipeline row: %v", err)
		}
	}
	if variantCount != 2 || jobCount != 1 || activityCount != 1 {
		t.Fatalf("media counts variants=%d jobs=%d activity=%d", variantCount, jobCount, activityCount)
	}
	// At-least-once execution must not duplicate variants or activity.
	if err := handler.Handle(ctx, payload); err != nil {
		t.Fatalf("media handler replay: %v", err)
	}
	var replayVariantCount int64
	if err := runtimeDatabase.Raw(`SELECT count(*) FROM tauco_app.media_variants WHERE media_asset_id = ?`, assetID).Scan(&replayVariantCount).Error; err != nil || replayVariantCount != 2 {
		t.Fatalf("media variants after replay = %d, err=%v", replayVariantCount, err)
	}
}

func assertDurableJobClaims(
	t *testing.T,
	ctx context.Context,
	runtimeDatabase *gorm.DB,
	migrationDatabase *gorm.DB,
) {
	t.Helper()
	first, _ := jobsrepo.NewPostgresRepository(runtimeDatabase)
	second, _ := jobsrepo.NewPostgresRepository(runtimeDatabase)
	start := make(chan struct{})
	type claimResult struct {
		owner string
		jobs  []jobsdomain.Job
		err   error
	}
	results := make(chan claimResult, 2)
	for _, worker := range []struct {
		owner string
		repo  *jobsrepo.PostgresRepository
	}{{"worker-a", first}, {"worker-b", second}} {
		go func(owner string, repo *jobsrepo.PostgresRepository) {
			<-start
			jobs, err := repo.Claim(ctx, owner, 2, time.Minute)
			results <- claimResult{owner: owner, jobs: jobs, err: err}
		}(worker.owner, worker.repo)
	}
	close(start)
	claimed := map[string]string{}
	for range 2 {
		result := <-results
		if result.err != nil {
			t.Fatalf("Claim(%s) error = %v", result.owner, result.err)
		}
		for _, job := range result.jobs {
			if prior := claimed[job.ID]; prior != "" {
				t.Fatalf("job %s claimed by %s and %s", job.ID, prior, result.owner)
			}
			claimed[job.ID] = result.owner
		}
	}
	if len(claimed) != 4 {
		t.Fatalf("concurrent claimed jobs = %d, want 4", len(claimed))
	}
	for jobID, owner := range claimed {
		if err := first.Succeed(ctx, jobID, owner); err != nil {
			t.Fatalf("Succeed(%s) error = %v", jobID, err)
		}
	}

	const crashJobID = "019bfc80-0000-7000-8000-000000009701"
	if err := migrationDatabase.Exec(`
		INSERT INTO tauco_app.background_jobs (
			id, kind, payload_json, idempotency_key, max_attempts
		) VALUES (?, 'test.crash', '{}'::jsonb, 'integration-crash-job', 2)
	`, crashJobID).Error; err != nil {
		t.Fatalf("insert crash job: %v", err)
	}
	crashed, err := first.Claim(ctx, "crashed-worker", 1, time.Millisecond)
	if err != nil || len(crashed) != 1 {
		t.Fatalf("crash Claim() = %d/%v", len(crashed), err)
	}
	time.Sleep(5 * time.Millisecond)
	reclaimed, err := second.Claim(ctx, "recovery-worker", 1, time.Minute)
	if err != nil || len(reclaimed) != 1 || reclaimed[0].ID != crashJobID || reclaimed[0].Attempts != 2 {
		t.Fatalf("reclaim = %+v/%v", reclaimed, err)
	}
	if err := second.Fail(ctx, crashJobID, "recovery-worker", time.Now().UTC(), "TEST_FAILURE"); err != nil {
		t.Fatalf("Fail(dead) error = %v", err)
	}
	if err := second.Replay(ctx, crashJobID, "job-replay-integration"); err != nil {
		t.Fatalf("Replay() error = %v", err)
	}
	var status string
	var attempts int
	if err := runtimeDatabase.Raw(`
		SELECT status, attempts FROM tauco_app.background_jobs WHERE id = ?
	`, crashJobID).Row().Scan(&status, &attempts); err != nil {
		t.Fatalf("read replayed job: %v", err)
	}
	if status != "retry" || attempts != 0 {
		t.Fatalf("replayed status/attempts = %s/%d", status, attempts)
	}
}

func assertContactTransaction(
	t *testing.T,
	ctx context.Context,
	runtimeDatabase *gorm.DB,
	migrationDatabase *gorm.DB,
) {
	t.Helper()
	store, err := contactrepo.NewPostgresStore(runtimeDatabase)
	if err != nil {
		t.Fatalf("NewPostgresStore() error = %v", err)
	}
	intake, err := contactapp.NewIntake(store, []byte("01234567890123456789012345678901"))
	if err != nil {
		t.Fatalf("NewIntake() error = %v", err)
	}
	submission := contactapp.Submission{
		Message: contactdomain.Message{
			Name: "Integration", Email: "integration@example.com",
			Subject: contactdomain.SubjectGeneral,
			Body:    "Pesan integration contact yang panjangnya valid.", PrivacyConsent: true,
		},
		IdempotencyKey: "contact-integration-000001",
		RequestID:      "contact-integration-request",
	}
	if result, err := intake.Submit(ctx, submission); err != nil || result.Replayed {
		t.Fatalf("first contact Submit() = %+v/%v", result, err)
	}
	assertTableCount(t, runtimeDatabase, "contact_messages", 1)
	assertTableCount(t, runtimeDatabase, "background_jobs", 2)
	if result, err := intake.Submit(ctx, submission); err != nil || !result.Replayed {
		t.Fatalf("replayed contact Submit() = %+v/%v", result, err)
	}
	assertTableCount(t, runtimeDatabase, "contact_messages", 1)
	assertTableCount(t, runtimeDatabase, "background_jobs", 2)

	submission.Message.Body = "Payload berbeda dengan idempotency key yang sama."
	if _, err := intake.Submit(ctx, submission); !errors.Is(err, contactapp.ErrIdempotencyConflict) {
		t.Fatalf("conflicting contact Submit() error = %v", err)
	}

	maintenanceStore, _ := contactrepo.NewPostgresStore(migrationDatabase)
	maintenanceIntake, _ := contactapp.NewIntake(
		maintenanceStore,
		[]byte("01234567890123456789012345678901"),
	)
	expired := submission
	expired.IdempotencyKey = "contact-integration-expired"
	expired.Message.Body = "Pesan lama yang sudah melewati batas retensi data."
	expired.ConsentAt = time.Now().UTC().AddDate(-1, -1, 0)
	if _, err := maintenanceIntake.Submit(ctx, expired); err != nil {
		t.Fatalf("expired contact Submit() error = %v", err)
	}
	deleted, err := maintenanceStore.PurgeExpired(ctx, time.Now().UTC(), 100)
	if err != nil || deleted != 1 {
		t.Fatalf("PurgeExpired() = %d/%v", deleted, err)
	}
}

func assertTableCount(t *testing.T, database *gorm.DB, table string, want int64) {
	t.Helper()
	var count int64
	if err := database.Raw("SELECT count(*) FROM tauco_app." + table).Scan(&count).Error; err != nil {
		t.Fatalf("count %s: %v", table, err)
	}
	if count != want {
		t.Fatalf("%s count = %d, want %d", table, count, want)
	}
}

func assertPublicReadHTTP(
	t *testing.T,
	pageRepository *contentrepo.PostgresRepository,
	productRepository *catalogrepo.PostgresRepository,
) {
	t.Helper()

	pageReader, err := contentapp.NewPublishedReader(pageRepository)
	if err != nil {
		t.Fatalf("NewPublishedReader(page) error = %v", err)
	}
	codec, err := catalogcursor.NewHMACSHA256(
		[]byte("0123456789abcdef0123456789abcdef"),
	)
	if err != nil {
		t.Fatalf("NewHMACSHA256() error = %v", err)
	}
	productReader, err := catalogapp.NewPublishedReader(
		productRepository,
		codec,
	)
	if err != nil {
		t.Fatalf("NewPublishedReader(product) error = %v", err)
	}
	server, err := api.NewPublicReadServer(pageReader, productReader)
	if err != nil {
		t.Fatalf("NewPublicReadServer() error = %v", err)
	}

	gin.SetMode(gin.TestMode)
	router := gin.New()
	router.Use(func(ctx *gin.Context) {
		ctx.Request.Header.Set(requestmeta.Header, "b4-postgres-test")
		ctx.Next()
	})
	api.RegisterSafePublicReadHandlers(router, server, nil, "")

	for _, path := range []string{
		"/api/v1/home",
		"/api/v1/about",
		"/api/v1/tauco-guide",
		"/api/v1/products",
		"/api/v1/products/tauco-cap-badak",
	} {
		response := httptest.NewRecorder()
		request := httptest.NewRequest(http.MethodGet, path, nil)
		router.ServeHTTP(response, request)
		if response.Code != http.StatusOK {
			t.Fatalf(
				"GET %s status = %d, body = %s",
				path,
				response.Code,
				response.Body,
			)
		}
		if response.Header().Get("ETag") == "" {
			t.Fatalf("GET %s returned an empty ETag", path)
		}
	}

	notFound := httptest.NewRecorder()
	request := httptest.NewRequest(
		http.MethodGet,
		"/api/v1/products/tidak-ada",
		nil,
	)
	router.ServeHTTP(notFound, request)
	if notFound.Code != http.StatusNotFound ||
		!strings.Contains(notFound.Body.String(), "PRODUCT_NOT_FOUND") {
		t.Fatalf(
			"unknown product response = %d/%s",
			notFound.Code,
			notFound.Body,
		)
	}
}

func assertDeterministicSeedTimestamps(
	t *testing.T,
	database *gorm.DB,
	plan contentapp.SeedPlan,
) {
	t.Helper()

	var mismatches int
	if err := database.Raw(`
SELECT
    (SELECT count(*)
     FROM tauco_app.pages
     WHERE created_at <> ? OR updated_at <> ?)
  + (SELECT count(*)
     FROM tauco_app.page_revisions
     WHERE created_at <> ? OR published_at <> ?)
  + (SELECT count(*)
     FROM tauco_app.products
     WHERE created_at <> ? OR updated_at <> ? OR first_published_at <> ?)
  + (SELECT count(*)
     FROM tauco_app.product_revisions
     WHERE created_at <> ? OR published_at <> ?)`,
		plan.Pages[0].Revision.PublishedAt,
		plan.Pages[0].Revision.PublishedAt,
		plan.Pages[0].Revision.PublishedAt,
		plan.Pages[0].Revision.PublishedAt,
		plan.Products[0].Revision.PublishedAt,
		plan.Products[0].Revision.PublishedAt,
		plan.Products[0].Revision.PublishedAt,
		plan.Products[0].Revision.PublishedAt,
		plan.Products[0].Revision.PublishedAt,
	).Scan(&mismatches).Error; err != nil {
		t.Fatalf("verify deterministic timestamps: %v", err)
	}
	if mismatches != 0 {
		t.Fatalf("deterministic timestamp mismatch count = %d", mismatches)
	}
}

func assertPublishedPageParity(
	t *testing.T,
	ctx context.Context,
	repository *contentrepo.PostgresRepository,
	plan contentapp.SeedPlan,
) {
	t.Helper()

	for _, expected := range plan.Pages {
		actual, err := repository.FindPublishedPage(ctx, expected.Key)
		if err != nil {
			t.Fatalf("FindPublishedPage(%q) error = %v", expected.Key, err)
		}
		if actual.PageID != expected.Revision.EntityID ||
			actual.RevisionID != expected.Revision.RevisionID ||
			actual.RevisionNumber != expected.Revision.RevisionNumber ||
			actual.SchemaVersion != expected.Revision.SchemaVersion ||
			actual.Checksum != expected.Revision.Checksum ||
			!actual.PublishedAt.Equal(expected.Revision.PublishedAt) ||
			!bytes.Equal(actual.ContentJSON, expected.Revision.ContentJSON) {
			t.Fatalf(
				"FindPublishedPage(%q) = %+v, does not match seed",
				expected.Key,
				actual,
			)
		}
	}
	if _, err := repository.FindPublishedPage(
		ctx,
		contentdomain.PageKey("unknown"),
	); !errors.Is(err, contentapp.ErrPublishedPageNotFound) {
		t.Fatalf("unknown page error = %v, want ErrPublishedPageNotFound", err)
	}
}

func assertPublishedProductParity(
	t *testing.T,
	ctx context.Context,
	repository *catalogrepo.PostgresRepository,
	plan contentapp.SeedPlan,
) {
	t.Helper()

	expected := plan.Products[0]
	actual, err := repository.FindPublishedProduct(ctx, expected.Slug)
	if err != nil {
		t.Fatalf("FindPublishedProduct() error = %v", err)
	}
	if actual.ProductID != expected.Revision.EntityID ||
		actual.RevisionID != expected.Revision.RevisionID ||
		actual.SortOrder != expected.SortOrder ||
		actual.RevisionNumber != expected.Revision.RevisionNumber ||
		actual.SchemaVersion != expected.Revision.SchemaVersion ||
		actual.Checksum != expected.Revision.Checksum ||
		!actual.PublishedAt.Equal(expected.Revision.PublishedAt) ||
		!bytes.Equal(actual.ContentJSON, expected.Revision.ContentJSON) {
		t.Fatalf("FindPublishedProduct() = %+v, does not match seed", actual)
	}
	if _, err := repository.FindPublishedProduct(
		ctx,
		"unknown-product",
	); !errors.Is(err, catalogapp.ErrPublishedProductNotFound) {
		t.Fatalf("unknown product error = %v, want not found", err)
	}
}

func assertCatalogPaginationProbe(
	t *testing.T,
	ctx context.Context,
	migrationDatabase *gorm.DB,
	repository *catalogrepo.PostgresRepository,
	plan contentapp.SeedPlan,
) {
	t.Helper()

	secondContent, secondChecksum, err := contentdomain.CanonicalJSONChecksum(
		[]byte(`{"name":"Pagination probe"}`),
	)
	if err != nil {
		t.Fatalf("canonicalize pagination fixture: %v", err)
	}
	publishedAt := plan.Products[0].Revision.PublishedAt
	err = migrationDatabase.Transaction(func(transaction *gorm.DB) error {
		if err := transaction.Exec("SET LOCAL ROLE tauco_migrator").Error; err != nil {
			return err
		}
		if err := transaction.Exec(`
INSERT INTO tauco_app.products (
    id, slug, sort_order, published_revision_id, first_published_at,
    created_at, updated_at
) VALUES (
    '019bfc80-0000-7000-8000-000000000102',
    'pagination-probe',
    0,
    '019bfc80-0000-7000-8000-000000000112',
    ?, ?, ?
)`,
			publishedAt,
			publishedAt,
			publishedAt,
		).Error; err != nil {
			return err
		}
		return transaction.Exec(`
INSERT INTO tauco_app.product_revisions (
    id, product_id, revision_number, status, schema_version, content_json,
    content_checksum, created_at, published_at
) VALUES (
    '019bfc80-0000-7000-8000-000000000112',
    '019bfc80-0000-7000-8000-000000000102',
    1, 'published', 1, ?::jsonb, ?, ?, ?
)`,
			string(secondContent),
			string(secondChecksum),
			publishedAt,
			publishedAt,
		).Error
	})
	if err != nil {
		t.Fatalf("insert pagination probe: %v", err)
	}

	firstPage, err := repository.ListPublishedProducts(ctx, nil, 1)
	if err != nil {
		t.Fatalf("ListPublishedProducts(first) error = %v", err)
	}
	if len(firstPage.Products) != 1 ||
		!firstPage.HasMore ||
		firstPage.Products[0].Slug != plan.Products[0].Slug {
		t.Fatalf("first catalog page = %+v", firstPage)
	}

	position := catalogdomain.PaginationPosition{
		SortOrder: firstPage.Products[0].SortOrder,
		ProductID: firstPage.Products[0].ProductID,
	}
	secondPage, err := repository.ListPublishedProducts(ctx, &position, 1)
	if err != nil {
		t.Fatalf("ListPublishedProducts(second) error = %v", err)
	}
	if len(secondPage.Products) != 1 ||
		secondPage.HasMore ||
		secondPage.Products[0].Slug != "pagination-probe" {
		t.Fatalf("second catalog page = %+v", secondPage)
	}
}

func assertSeedCounts(
	t *testing.T,
	database *gorm.DB,
	wantPages,
	wantPageRevisions,
	wantProducts,
	wantProductRevisions int64,
) {
	t.Helper()

	for table, want := range map[string]int64{
		"pages":             wantPages,
		"page_revisions":    wantPageRevisions,
		"products":          wantProducts,
		"product_revisions": wantProductRevisions,
	} {
		var count int64
		result := database.Raw(
			"SELECT count(*) FROM tauco_app." + table,
		).Scan(&count)
		if result.Error != nil {
			t.Fatalf("count %s: %v", table, result.Error)
		}
		if count != want {
			t.Fatalf("%s count = %d, want %d", table, count, want)
		}
	}
}

func cloneSeedPlan(plan contentapp.SeedPlan) contentapp.SeedPlan {
	cloned := plan
	cloned.Pages = append([]contentapp.PageSeed(nil), plan.Pages...)
	cloned.Products = append([]contentapp.ProductSeed(nil), plan.Products...)
	for index := range cloned.Pages {
		cloned.Pages[index].Revision.ContentJSON = append(
			json.RawMessage(nil),
			plan.Pages[index].Revision.ContentJSON...,
		)
	}
	for index := range cloned.Products {
		cloned.Products[index].Revision.ContentJSON = append(
			json.RawMessage(nil),
			plan.Products[index].Revision.ContentJSON...,
		)
	}
	return cloned
}
