package database

import (
	"bytes"
	"context"
	"crypto/ed25519"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"image"
	"image/color"
	"image/jpeg"
	"image/png"
	"net/http"
	"net/http/cookiejar"
	"net/http/httptest"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"gorm.io/gorm"

	auditapp "github.com/ilhamnugraha8944/tauco/backend/internal/audit/application"
	auditrepo "github.com/ilhamnugraha8944/tauco/backend/internal/audit/repository"
	authruntime "github.com/ilhamnugraha8944/tauco/backend/internal/auth"
	authapp "github.com/ilhamnugraha8944/tauco/backend/internal/auth/application"
	authdomain "github.com/ilhamnugraha8944/tauco/backend/internal/auth/domain"
	authrepo "github.com/ilhamnugraha8944/tauco/backend/internal/auth/repository"
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
	mediadomain "github.com/ilhamnugraha8944/tauco/backend/internal/media/domain"
	mediaprocessor "github.com/ilhamnugraha8944/tauco/backend/internal/media/processor"
	mediarepo "github.com/ilhamnugraha8944/tauco/backend/internal/media/repository"
	mediastorage "github.com/ilhamnugraha8944/tauco/backend/internal/media/storage"
	"github.com/ilhamnugraha8944/tauco/backend/internal/platform/ratelimit"
)

const repositoryIntegrationLock int64 = 839_103_221_702

func TestD1SupabaseConnectionPolicy(t *testing.T) {
	values := map[string]string{
		"DATABASE_DEPLOYMENT_PROFILE": "supabase",
		"MIGRATION_BOOTSTRAP_ROLES":   "false",
		"MIGRATION_DATABASE_URL":      "postgres://tauco_migration_login.ref:secret@session.example:5432/postgres?sslmode=require",
		"DATABASE_URL":                "postgres://tauco_public_login.ref:secret@pool.example:6543/postgres?sslmode=require",
		"ADMIN_DATABASE_URL":          "postgres://tauco_admin_login.ref:secret@pool.example:6543/postgres?sslmode=require",
	}
	lookup := func(key string) (string, bool) { value, ok := values[key]; return value, ok }
	migration, err := LoadMigrationConfig(lookup)
	if err != nil || migration.Profile != ProfileSupabase || migration.BootstrapRoles {
		t.Fatalf("supabase migration config = %+v, %v", migration, err)
	}
	runtime, err := LoadRuntimeConfig(lookup)
	if err != nil || runtime.MaxOpenConns != 1 || runtime.MaxIdleConns != 0 || !runtime.PreferSimpleQuery {
		t.Fatalf("supabase runtime config = %+v, %v", runtime, err)
	}

	values["MIGRATION_BOOTSTRAP_ROLES"] = "true"
	if _, err := LoadMigrationConfig(lookup); err == nil {
		t.Fatal("supabase profile accepted local role bootstrap")
	}
	values["MIGRATION_BOOTSTRAP_ROLES"] = "false"
	values["DATABASE_MAX_OPEN_CONNS"] = "3"
	if _, err := LoadRuntimeConfig(lookup); err == nil {
		t.Fatal("supabase profile accepted an oversized runtime pool")
	}
	delete(values, "DATABASE_MAX_OPEN_CONNS")
	values["DATABASE_URL"] = "postgres://tauco_public_login.ref:secret@pool.example:5432/postgres?sslmode=require"
	if _, err := LoadRuntimeConfig(lookup); err == nil {
		t.Fatal("supabase runtime accepted the session-pooler port")
	}
	values["DATABASE_URL"] = "postgres://tauco_public_login.ref:secret@pool.example:6543/postgres?sslmode=disable"
	if _, err := LoadRuntimeConfig(lookup); err == nil {
		t.Fatal("supabase runtime accepted disabled TLS")
	}
	values["DATABASE_URL"] = "postgres://tauco_public_login.other:secret@pool.example:6543/postgres?sslmode=require"
	if _, err := LoadMigrationConfig(lookup); err == nil {
		t.Fatal("supabase migration config accepted a runtime URL for another project")
	}
	values["DATABASE_URL"] = "postgres://unexpected.ref:secret@pool.example:6543/postgres?sslmode=require"
	if _, err := LoadMigrationConfig(lookup); err == nil {
		t.Fatal("supabase migration config accepted an unexpected runtime login")
	}
	values["DATABASE_URL"] = "postgres://tauco_public_login.ref:secret@pool.example:6543/postgres?sslmode=require"
	values["ADMIN_DATABASE_URL"] = "postgres://tauco_admin_login.other:secret@pool.example:6543/postgres?sslmode=require"
	if _, err := LoadMigrationConfig(lookup); err == nil {
		t.Fatal("supabase migration config accepted an admin URL for another project")
	}
}

func TestD1ProductionEd25519RuntimeAndJWTPolicy(t *testing.T) {
	_, privateKey, err := ed25519.GenerateKey(rand.Reader)
	if err != nil {
		t.Fatal(err)
	}
	manager, err := authdomain.NewEd25519TokenManager(privateKey, "integration", "admin", "test-key", authapp.AccessTTL)
	if err != nil {
		t.Fatal(err)
	}
	assertJWTRejections(t, manager, privateKey)
	assertProductionAuthRuntime(t, privateKey)
}

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
	adminRoleName := "tauco_repository_admin_" + suffix
	if len(runtimeRoleName) > 63 {
		runtimeRoleName = runtimeRoleName[:63]
	}
	if len(adminRoleName) > 63 {
		adminRoleName = adminRoleName[:63]
	}
	const runtimePassword = "B3-repository-runtime-test-password"
	const adminPassword = "C1-repository-admin-test-password"

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
		_, _ = admin.Exec(
			context.Background(),
			"DROP ROLE IF EXISTS "+pgx.Identifier{adminRoleName}.Sanitize(),
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
	adminURL := replaceDatabaseAndUser(
		t,
		baseURL,
		databaseName,
		adminRoleName,
		adminPassword,
	)
	config := MigrationConfig{
		MigrationURL:   migrationURL,
		RuntimeURL:     runtimeURL,
		AdminURL:       adminURL,
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

	adminGORM, err := OpenAdminGORM(ctx, RuntimeConfig{
		URL: adminURL, MaxOpenConns: 2, MaxIdleConns: 1,
		ConnMaxLifetime: 30 * time.Minute, ConnMaxIdleTime: 5 * time.Minute,
		PreferSimpleQuery: true,
	})
	if err != nil {
		t.Fatalf("OpenAdminGORM() error = %v", err)
	}
	adminSQL, err := adminGORM.DB()
	if err != nil {
		t.Fatalf("admin GORM pool: %v", err)
	}
	defer adminSQL.Close()
	assertAdminAuthLifecycle(t, ctx, adminGORM, migrationGORM)

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
	assertMediaPipeline(t, ctx, runtimeGORM, adminGORM)
	assertAdminContentLifecycle(t, ctx, adminGORM, plan)
	assertAdminProductLifecycle(t, ctx, adminGORM, runtimeGORM, plan)
	assertAdminInboxActivity(t, ctx, adminGORM)
	assertCacheInvalidationPayloads(t, ctx)
	assertCatalogPaginationProbe(
		t,
		ctx,
		migrationGORM,
		productRepository,
		plan,
	)
}

type generationProbe map[string]int

func (probe generationProbe) Bump(_ context.Context, tag string) error { probe[tag]++; return nil }

func assertCacheInvalidationPayloads(t *testing.T, ctx context.Context) {
	t.Helper()
	probe := generationProbe{}
	handler, err := contentapp.NewCacheInvalidationHandler(probe)
	if err != nil {
		t.Fatal(err)
	}
	for _, payload := range []json.RawMessage{
		json.RawMessage(`{"generationTag":"home"}`),
		json.RawMessage(`{"generationTags":["products","product:produk-integration"]}`),
		json.RawMessage(`{"generationTags":["products","product:produk-integration"]}`),
	} {
		if err := handler.Handle(ctx, payload); err != nil {
			t.Fatalf("cache invalidation payload: %v", err)
		}
	}
	if probe["home"] != 1 || probe["products"] != 2 || probe["product:produk-integration"] != 2 {
		t.Fatalf("cache generations=%v", probe)
	}
	if err := handler.Handle(ctx, json.RawMessage(`{"generationTag":"unsafe tag","email":"pii@example.test"}`)); !errors.Is(err, contentapp.ErrInvalidCacheInvalidation) {
		t.Fatalf("unsafe cache payload err=%v", err)
	}
}

func assertAdminProductLifecycle(t *testing.T, ctx context.Context, adminDatabase, runtimeDatabase *gorm.DB, plan contentapp.SeedPlan) {
	t.Helper()
	repository, err := catalogrepo.NewAdminPostgres(adminDatabase)
	if err != nil {
		t.Fatalf("NewAdminPostgres(catalog): %v", err)
	}
	codec, err := catalogcursor.NewHMACSHA256([]byte("integration-product-cursor-secret-32-bytes"))
	if err != nil {
		t.Fatalf("NewHMACSHA256(product admin): %v", err)
	}
	service, err := catalogapp.NewAdminProductService(repository, codec)
	if err != nil {
		t.Fatalf("NewAdminProductService: %v", err)
	}
	var actorID string
	if err := adminDatabase.Raw(`SELECT id::text FROM tauco_app.admin_users ORDER BY created_at LIMIT 1`).Scan(&actorID).Error; err != nil || actorID == "" {
		t.Fatalf("load product actor: %v", err)
	}
	sku := "INTEGRATION-001"
	product, err := service.Create(ctx, "produk-integration", &sku, 90, actorID)
	if err != nil || product.CurrentRevisionID() != product.ID {
		t.Fatalf("create product=%+v err=%v", product, err)
	}
	if len(plan.Products) == 0 {
		t.Fatal("product seed content missing")
	}
	raw := bytes.ReplaceAll(plan.Products[0].Revision.ContentJSON, []byte(`"tauco-cap-badak"`), []byte(`"produk-integration"`))
	raw = bytes.ReplaceAll(raw, []byte(`/produk/tauco-cap-badak`), []byte(`/produk/produk-integration`))
	draft, err := service.SaveDraft(ctx, product.ID, contentapp.RevisionETag(product.ID), product.ID, actorID, raw)
	if err != nil || draft.Status != "draft" {
		t.Fatalf("save product draft=%+v err=%v", draft, err)
	}
	published, err := service.Publish(ctx, product.ID, draft.ID, contentapp.RevisionETag(draft.ID), actorID)
	if err != nil || published.Status != "published" {
		t.Fatalf("publish product=%+v err=%v", published, err)
	}
	changedSlug := "slug-terlarang-setelah-publish"
	if _, err := service.Update(ctx, product.ID, contentapp.RevisionETag(published.ID), &changedSlug, nil, nil, actorID); !errors.Is(err, catalogapp.ErrProductConflict) {
		t.Fatalf("stable product slug err=%v", err)
	}
	publicRepository, _ := catalogrepo.NewPostgresRepository(runtimeDatabase)
	if _, err := publicRepository.FindPublishedProduct(ctx, "produk-integration"); err != nil {
		t.Fatalf("published product unavailable publicly: %v", err)
	}
	if err := service.Unpublish(ctx, product.ID, contentapp.RevisionETag(published.ID), actorID); err != nil {
		t.Fatalf("unpublish product: %v", err)
	}
	if err := service.Archive(ctx, product.ID, contentapp.RevisionETag(published.ID), actorID); err != nil {
		t.Fatalf("archive product: %v", err)
	}
	if _, err := publicRepository.FindPublishedProduct(ctx, "produk-integration"); !errors.Is(err, catalogapp.ErrPublishedProductNotFound) {
		t.Fatalf("archived product public error=%v", err)
	}
	if err := service.Unarchive(ctx, product.ID, contentapp.RevisionETag(published.ID), actorID); err != nil {
		t.Fatalf("unarchive product: %v", err)
	}
	var invalidations, activities int64
	adminDatabase.Raw(`SELECT count(*) FROM tauco_app.background_jobs WHERE kind='content.invalidate_cache' AND idempotency_key LIKE 'product.invalidate:%'`).Scan(&invalidations)
	adminDatabase.Raw(`SELECT count(*) FROM tauco_app.activity_logs WHERE entity_id=?::uuid AND event_type LIKE 'product.%'`, product.ID).Scan(&activities)
	if invalidations != 2 || activities < 6 {
		t.Fatalf("product invalidations=%d activities=%d", invalidations, activities)
	}
}

func assertAdminInboxActivity(t *testing.T, ctx context.Context, adminDatabase *gorm.DB) {
	t.Helper()
	codec, _ := catalogcursor.NewHMACSHA256([]byte("integration-inbox-cursor-secret-32-bytes"))
	inboxRepository, _ := contactrepo.NewPostgresStore(adminDatabase)
	inbox, _ := contactapp.NewAdminMessageService(inboxRepository, codec)
	status := "unread"
	page, err := inbox.List(ctx, nil, intPointer(20), &status)
	if err != nil || len(page.Messages) == 0 {
		t.Fatalf("list admin inbox=%d err=%v", len(page.Messages), err)
	}
	before, err := inbox.Get(ctx, page.Messages[0].ID)
	if err != nil {
		t.Fatalf("get admin inbox: %v", err)
	}
	afterRead, err := inbox.Get(ctx, before.ID)
	if err != nil || !afterRead.UpdatedAt.Equal(before.UpdatedAt) || afterRead.Status != "unread" {
		t.Fatalf("GET mutated inbox before=%+v after=%+v err=%v", before, afterRead, err)
	}
	var actorID string
	adminDatabase.Raw(`SELECT id::text FROM tauco_app.admin_users ORDER BY created_at LIMIT 1`).Scan(&actorID)
	updated, err := inbox.UpdateStatus(ctx, before.ID, before.ETag(), "read", actorID)
	if err != nil || updated.Status != "read" {
		t.Fatalf("update inbox=%+v err=%v", updated, err)
	}
	if _, err := inbox.UpdateStatus(ctx, before.ID, before.ETag(), "archived", actorID); !errors.Is(err, contactapp.ErrAdminMessageConflict) {
		t.Fatalf("stale inbox ETag err=%v", err)
	}
	activityRepository, _ := auditrepo.NewAdminPostgres(adminDatabase)
	activity, _ := auditapp.NewAdminService(activityRepository, codec)
	eventType, entityType := "contact.status_changed", "contact_message"
	activityPage, err := activity.List(ctx, nil, intPointer(20), auditapp.ActivityFilter{EventType: &eventType, EntityType: &entityType})
	if err != nil || len(activityPage.Activities) != 1 || activityPage.Activities[0].EntityID == nil {
		t.Fatalf("activity filter=%+v err=%v", activityPage, err)
	}
	var metadata string
	adminDatabase.Raw(`SELECT metadata_json::text FROM tauco_app.activity_logs WHERE id=?`, activityPage.Activities[0].ID).Scan(&metadata)
	if strings.Contains(metadata, before.Email) || strings.Contains(metadata, before.Name) || !strings.Contains(metadata, "fromStatus") {
		t.Fatalf("activity metadata allowlist violated: %s", metadata)
	}
}

func assertAdminAuthLifecycle(t *testing.T, ctx context.Context, db, migrationDB *gorm.DB) {
	t.Helper()
	store, err := authrepo.NewPostgres(db)
	if err != nil {
		t.Fatalf("NewPostgres(auth) error = %v", err)
	}
	_, privateKey, err := ed25519.GenerateKey(rand.Reader)
	if err != nil {
		t.Fatalf("GenerateKey() error = %v", err)
	}
	tokens, err := authdomain.NewEd25519TokenManager(privateKey, "integration", "admin", "test-key", authapp.AccessTTL)
	if err != nil {
		t.Fatalf("NewEd25519TokenManager() error = %v", err)
	}
	assertJWTRejections(t, tokens, privateKey)
	box, err := authdomain.NewSecretBox([]byte("0123456789abcdef0123456789abcdef"), "test-key")
	if err != nil {
		t.Fatalf("NewSecretBox() error = %v", err)
	}
	service, err := authapp.NewService(store, tokens, box, []byte("integration-recovery-secret-32-bytes-minimum"))
	if err != nil {
		t.Fatalf("NewService() error = %v", err)
	}
	const email, password = "owner@example.test", "correct horse battery staple"
	if err := service.BootstrapAdmin(ctx, email, password); err != nil {
		t.Fatalf("BootstrapAdmin() error = %v", err)
	}
	login, err := service.Login(ctx, email, password, nil, nil, "integration")
	if err != nil || login.Principal.Level != authapp.LevelPassword {
		t.Fatalf("password login = %+v, %v", login.Principal, err)
	}
	setup, err := service.SetupTOTP(ctx, login.Principal)
	if err != nil {
		t.Fatalf("SetupTOTP() error = %v", err)
	}
	code, err := authdomain.CurrentTOTP(setup.Secret, time.Now())
	if err != nil {
		t.Fatalf("CurrentTOTP() error = %v", err)
	}
	enabled, err := service.EnableTOTP(ctx, login.Principal, code)
	if err != nil || len(enabled.Codes) != authapp.RecoveryCount {
		t.Fatalf("EnableTOTP() codes=%d error=%v", len(enabled.Codes), err)
	}
	if _, err := service.Login(ctx, email, password, &code, nil, "integration"); !errors.Is(err, authapp.ErrAuthentication) {
		t.Fatalf("TOTP replay error = %v, want generic authentication error", err)
	}
	expiredCode, _ := authdomain.CurrentTOTP(setup.Secret, time.Now().Add(-2*time.Minute))
	if _, err := service.Login(ctx, email, password, &expiredCode, nil, "integration"); !errors.Is(err, authapp.ErrAuthentication) {
		t.Fatalf("expired TOTP error = %v, want generic authentication error", err)
	}
	recovery := enabled.Codes[0]
	recovered, err := service.Login(ctx, email, password, nil, &recovery, "integration")
	if err != nil || recovered.Principal.Level != authapp.LevelMFA {
		t.Fatalf("recovery login error = %v", err)
	}
	if _, err := service.Login(ctx, email, password, nil, &recovery, "integration"); !errors.Is(err, authapp.ErrAuthentication) {
		t.Fatalf("recovery replay error = %v, want generic authentication error", err)
	}
	rotated, err := service.Refresh(ctx, recovered.Tokens.Refresh, recovered.Tokens.CSRF)
	if err != nil {
		t.Fatalf("Refresh() error = %v", err)
	}
	if _, err := service.ValidateAccess(ctx, rotated.Tokens.Access, true); err != nil {
		t.Fatalf("ValidateAccess() error = %v", err)
	}
	if _, err := service.Refresh(ctx, recovered.Tokens.Refresh, recovered.Tokens.CSRF); !errors.Is(err, authapp.ErrRefreshReused) {
		t.Fatalf("refresh reuse error = %v, want ErrRefreshReused", err)
	}
	if _, err := service.ValidateAccess(ctx, rotated.Tokens.Access, true); !errors.Is(err, authapp.ErrUnauthorized) {
		t.Fatalf("reused session access error = %v, want unauthorized", err)
	}
	assertAdminAuthHTTP(t, ctx, db, migrationDB, service)
}

func assertProductionAuthRuntime(t *testing.T, privateKey ed25519.PrivateKey) {
	t.Helper()
	values := map[string]string{
		"APP_ENV":                        "production",
		"JWT_ED25519_PRIVATE_KEY_BASE64": base64.RawStdEncoding.EncodeToString(privateKey),
		"JWT_PRIVATE_KEY_FILE":           "must-not-be-read.pem",
		"JWT_PUBLIC_KEY_FILE":            "must-not-be-read.pem",
		"JWT_ISSUER":                     "integration",
		"JWT_AUDIENCE":                   "admin",
		"JWT_KEY_ID":                     "test-key",
		"MFA_ENCRYPTION_KEY":             base64.RawStdEncoding.EncodeToString([]byte("0123456789abcdef0123456789abcdef")),
		"MFA_ENCRYPTION_KEY_ID":          "test-key",
		"RECOVERY_CODE_HMAC_SECRET":      "integration-recovery-secret-32-bytes-minimum",
	}
	lookup := func(key string) (string, bool) { value, ok := values[key]; return value, ok }
	runtime, err := authruntime.LoadRuntime(lookup)
	if err != nil {
		t.Fatalf("production Ed25519 runtime error = %v", err)
	}
	userID, _ := uuid.NewV7()
	sessionID, _ := uuid.NewV7()
	raw, _, err := runtime.Tokens.Sign(userID, sessionID, true)
	if err != nil || !strings.HasPrefix(raw, "ey") {
		t.Fatalf("production Ed25519 sign error = %v", err)
	}
	values["JWT_ED25519_PRIVATE_KEY_BASE64"] += "="
	if _, err := authruntime.LoadRuntime(lookup); err == nil {
		t.Fatal("production runtime accepted padded Ed25519 private key")
	}
	delete(values, "JWT_ED25519_PRIVATE_KEY_BASE64")
	if _, err := authruntime.LoadRuntime(lookup); err == nil {
		t.Fatal("production runtime accepted RSA-file-only configuration")
	}
}

func assertJWTRejections(t *testing.T, manager *authdomain.TokenManager, privateKey ed25519.PrivateKey) {
	t.Helper()
	userID, _ := uuid.NewV7()
	sessionID, _ := uuid.NewV7()
	now := time.Now()
	claims := func(issuer string, audience jwt.ClaimStrings, expires time.Time) authdomain.AccessClaims {
		return authdomain.AccessClaims{SessionID: sessionID.String(), MFA: true, RegisteredClaims: jwt.RegisteredClaims{
			Issuer: issuer, Subject: userID.String(), Audience: audience, ID: uuid.NewString(),
			IssuedAt: jwt.NewNumericDate(now), NotBefore: jwt.NewNumericDate(now.Add(-time.Second)), ExpiresAt: jwt.NewNumericDate(expires),
		}}
	}
	sign := func(value authdomain.AccessClaims, typ, kid string, key ed25519.PrivateKey) string {
		token := jwt.NewWithClaims(jwt.SigningMethodEdDSA, value)
		token.Header["typ"] = typ
		token.Header["kid"] = kid
		raw, _ := token.SignedString(key)
		return raw
	}
	_, otherPrivate, _ := ed25519.GenerateKey(rand.Reader)
	cases := []string{
		sign(claims("wrong", jwt.ClaimStrings{"admin"}, now.Add(time.Minute)), "JWT", "test-key", privateKey),
		sign(claims("integration", jwt.ClaimStrings{"wrong"}, now.Add(time.Minute)), "JWT", "test-key", privateKey),
		sign(claims("integration", jwt.ClaimStrings{"admin"}, now.Add(time.Minute)), "wrong", "test-key", privateKey),
		sign(claims("integration", jwt.ClaimStrings{"admin"}, now.Add(time.Minute)), "JWT", "wrong", privateKey),
		sign(claims("integration", jwt.ClaimStrings{"admin"}, now.Add(time.Minute)), "JWT", "test-key", otherPrivate),
		sign(claims("integration", jwt.ClaimStrings{"admin"}, now.Add(-time.Minute)), "JWT", "test-key", privateKey),
	}
	missingIssuedAt := claims("integration", jwt.ClaimStrings{"admin"}, now.Add(time.Minute))
	missingIssuedAt.IssuedAt = nil
	missingNotBefore := claims("integration", jwt.ClaimStrings{"admin"}, now.Add(time.Minute))
	missingNotBefore.NotBefore = nil
	missingID := claims("integration", jwt.ClaimStrings{"admin"}, now.Add(time.Minute))
	missingID.ID = ""
	invalidSubject := claims("integration", jwt.ClaimStrings{"admin"}, now.Add(time.Minute))
	invalidSubject.Subject = "not-a-uuid"
	invalidSession := claims("integration", jwt.ClaimStrings{"admin"}, now.Add(time.Minute))
	invalidSession.SessionID = "not-a-uuid"
	cases = append(cases,
		sign(missingIssuedAt, "JWT", "test-key", privateKey),
		sign(missingNotBefore, "JWT", "test-key", privateKey),
		sign(missingID, "JWT", "test-key", privateKey),
		sign(invalidSubject, "JWT", "test-key", privateKey),
		sign(invalidSession, "JWT", "test-key", privateKey),
	)
	rsaPrivate, _, _ := authdomain.GenerateRSAKeyPair()
	rsaToken := jwt.NewWithClaims(jwt.SigningMethodRS256, claims("integration", jwt.ClaimStrings{"admin"}, now.Add(time.Minute)))
	rsaToken.Header["typ"] = "JWT"
	rsaToken.Header["kid"] = "test-key"
	rawRSA, _ := rsaToken.SignedString(rsaPrivate)
	cases = append(cases, rawRSA)
	hs := jwt.NewWithClaims(jwt.SigningMethodHS256, claims("integration", jwt.ClaimStrings{"admin"}, now.Add(time.Minute)))
	hs.Header["typ"] = "JWT"
	hs.Header["kid"] = "test-key"
	rawHS, _ := hs.SignedString([]byte("not-an-rsa-key"))
	cases = append(cases, rawHS)
	for index, raw := range cases {
		if _, err := manager.Verify(raw); err == nil {
			t.Fatalf("invalid JWT case %d accepted", index)
		}
	}
}

func assertAdminAuthHTTP(t *testing.T, ctx context.Context, db, migrationDB *gorm.DB, service *authapp.Service) {
	t.Helper()
	const origin, email, password = "http://admin.local", "api-owner@example.test", "another correct horse battery staple"
	if err := service.BootstrapAdmin(ctx, email, password); err != nil {
		t.Fatalf("bootstrap HTTP admin: %v", err)
	}
	local, _ := ratelimit.NewLocal(1_000)
	limiter, _ := ratelimit.New(nil, local)
	handler, err := api.NewAdminAuthHandler(service, limiter, api.AdminAuthConfig{
		AllowedOrigins: []string{origin}, RateSecret: []byte("integration-rate-secret-at-least-32-bytes"),
		BFFSecret: []byte("integration-admin-bff-secret-at-least-32-bytes"), SecureCookies: false,
	})
	if err != nil {
		t.Fatalf("NewAdminAuthHandler() error = %v", err)
	}
	router := gin.New()
	handler.Register(router)
	server := httptest.NewServer(router)
	defer server.Close()
	jar, _ := cookiejar.New(nil)
	client := &http.Client{Jar: jar}
	do := func(method, path, body, requestOrigin, csrf, fetchSite string) *http.Response {
		t.Helper()
		request, requestErr := http.NewRequest(method, server.URL+path, bytes.NewBufferString(body))
		if requestErr != nil {
			t.Fatalf("new auth request: %v", requestErr)
		}
		if body != "" {
			request.Header.Set("Content-Type", "application/json")
		}
		if requestOrigin != "" {
			request.Header.Set("Origin", requestOrigin)
		}
		if csrf != "" {
			request.Header.Set("X-CSRF-Token", csrf)
		}
		if fetchSite != "" {
			request.Header.Set("Sec-Fetch-Site", fetchSite)
		}
		request.Header.Set("X-Tauco-Admin-BFF-Secret", "integration-admin-bff-secret-at-least-32-bytes")
		response, requestErr := client.Do(request)
		if requestErr != nil {
			t.Fatalf("auth request: %v", requestErr)
		}
		return response
	}
	loginBody := `{"email":"` + email + `","password":"` + password + `"}`
	directRequest, _ := http.NewRequest(http.MethodPost, server.URL+"/api/v1/admin/auth/login", bytes.NewBufferString(loginBody))
	directRequest.Header.Set("Content-Type", "application/json")
	directRequest.Header.Set("Origin", origin)
	directResponse, directErr := client.Do(directRequest)
	if directErr != nil {
		t.Fatalf("direct admin bypass request error=%v", directErr)
	}
	if directResponse.StatusCode != http.StatusNotFound {
		t.Fatalf("direct admin bypass status=%v", directResponse.StatusCode)
	}
	directResponse.Body.Close()
	response := do(http.MethodPost, "/api/v1/admin/auth/login", loginBody, origin, "", "same-origin")
	if response.StatusCode != http.StatusOK {
		t.Fatalf("login status = %d", response.StatusCode)
	}
	var loginResponse api.AdminAuthResponse
	if err := json.NewDecoder(response.Body).Decode(&loginResponse); err != nil {
		t.Fatalf("decode login: %v", err)
	}
	if len(response.Cookies()) != 3 {
		t.Fatalf("login cookie count = %d, want 3", len(response.Cookies()))
	}
	for _, cookie := range response.Cookies() {
		if cookie.Path != "/" || cookie.Domain != "" || cookie.SameSite != http.SameSiteStrictMode {
			t.Fatalf("unsafe cookie attributes: %+v", cookie)
		}
		if cookie.Name != "tauco_admin_csrf" && !cookie.HttpOnly {
			t.Fatalf("auth cookie is not HttpOnly: %s", cookie.Name)
		}
		if cookie.Name == "tauco_admin_csrf" && cookie.HttpOnly {
			t.Fatal("CSRF cookie must be readable by the BFF/browser client")
		}
	}
	response.Body.Close()
	csrf := authCSRFCookie(t, client, server.URL)
	response = do(http.MethodPost, "/api/v1/admin/auth/totp/setup", "", origin, csrf, "same-origin")
	if response.StatusCode != http.StatusOK {
		t.Fatalf("TOTP setup status = %d", response.StatusCode)
	}
	var setup api.AdminTotpSetupResponse
	if err := json.NewDecoder(response.Body).Decode(&setup); err != nil {
		t.Fatalf("decode TOTP setup: %v", err)
	}
	response.Body.Close()
	code, _ := authdomain.CurrentTOTP(setup.Data.ManualKey, time.Now())
	response = do(http.MethodPost, "/api/v1/admin/auth/totp/enable", `{"totpCode":"`+code+`"}`, origin, csrf, "same-origin")
	if response.StatusCode != http.StatusOK {
		t.Fatalf("TOTP enable status = %d", response.StatusCode)
	}
	response.Body.Close()
	response = do(http.MethodGet, "/api/v1/admin/auth/me", "", "", "", "")
	if response.StatusCode != http.StatusOK {
		t.Fatalf("me status = %d", response.StatusCode)
	}
	response.Body.Close()
	response = do(http.MethodPost, "/api/v1/admin/auth/refresh", "", origin, csrf, "same-origin")
	if response.StatusCode != http.StatusOK {
		t.Fatalf("refresh status = %d", response.StatusCode)
	}
	response.Body.Close()
	csrf = authCSRFCookie(t, client, server.URL)
	nextCode, _ := authdomain.CurrentTOTP(setup.Data.ManualKey, time.Now().Add(30*time.Second))
	response = do(http.MethodPost, "/api/v1/admin/auth/recovery-codes/regenerate", `{"totpCode":"`+nextCode+`"}`, origin, csrf, "same-origin")
	if response.StatusCode != http.StatusOK {
		t.Fatalf("recovery regeneration status = %d", response.StatusCode)
	}
	response.Body.Close()
	if err := migrationDB.Transaction(func(tx *gorm.DB) error {
		if err := tx.Exec("SET LOCAL ROLE tauco_migrator").Error; err != nil {
			return err
		}
		return tx.Exec(`DELETE FROM tauco_app.role_permissions WHERE permission_id = (SELECT id FROM tauco_app.permissions WHERE key = 'account.manage')`).Error
	}); err != nil {
		t.Fatalf("remove permission fixture: %v", err)
	}
	response = do(http.MethodPost, "/api/v1/admin/auth/recovery-codes/regenerate", `{"totpCode":"`+nextCode+`"}`, origin, csrf, "same-origin")
	if response.StatusCode != http.StatusForbidden {
		t.Fatalf("RBAC denial status = %d", response.StatusCode)
	}
	response.Body.Close()
	response = do(http.MethodPost, "/api/v1/admin/auth/logout", "", "http://evil.test", csrf, "cross-site")
	if response.StatusCode != http.StatusForbidden {
		t.Fatalf("cross-site denial status = %d", response.StatusCode)
	}
	response.Body.Close()
	response = do(http.MethodPost, "/api/v1/admin/auth/logout", "", origin, "", "same-origin")
	if response.StatusCode != http.StatusForbidden {
		t.Fatalf("CSRF denial status = %d", response.StatusCode)
	}
	response.Body.Close()
	response = do(http.MethodPost, "/api/v1/admin/auth/logout", "", origin, csrf, "same-origin")
	if response.StatusCode != http.StatusNoContent {
		t.Fatalf("logout status = %d", response.StatusCode)
	}
	response.Body.Close()
	for attempt := 1; attempt <= 6; attempt++ {
		response = do(http.MethodPost, "/api/v1/admin/auth/login", `{"email":"rate@example.test","password":"invalid-password"}`, origin, "", "same-origin")
		if attempt == 6 && response.StatusCode != http.StatusTooManyRequests {
			t.Fatalf("login limiter status = %d", response.StatusCode)
		}
		response.Body.Close()
	}
	var auditCount int64
	if err := db.Raw(`SELECT count(*) FROM tauco_app.activity_logs WHERE event_type LIKE 'auth.%'`).Scan(&auditCount).Error; err != nil || auditCount < 5 {
		t.Fatalf("auth audit count = %d, error=%v", auditCount, err)
	}
}

func authCSRFCookie(t *testing.T, client *http.Client, rawURL string) string {
	t.Helper()
	parsed, err := url.Parse(rawURL)
	if err != nil {
		t.Fatalf("parse server URL: %v", err)
	}
	for _, cookie := range client.Jar.Cookies(parsed) {
		if cookie.Name == "tauco_admin_csrf" {
			return cookie.Value
		}
	}
	t.Fatal("CSRF cookie not found")
	return ""
}

func assertMediaPipeline(t *testing.T, ctx context.Context, runtimeDatabase, adminDatabase *gorm.DB) {
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

	adminRepository, _ := mediarepo.NewPostgres(adminDatabase)
	cursorCodec, _ := catalogcursor.NewHMACSHA256(bytes.Repeat([]byte{0x5c}, 32))
	adminService, _ := mediaapp.NewAdminService(adminRepository, cursorCodec)
	page, err := adminService.List(ctx, nil, intPointer(1))
	if err != nil || len(page.Assets) != 1 || page.Assets[0].ID != assetID || len(page.Assets[0].Variants) != 2 {
		t.Fatalf("admin media list = %+v, err=%v", page, err)
	}
	variant, err := adminService.ReadyVariant(ctx, assetID, nil)
	if err != nil || variant.Width != 640 {
		t.Fatalf("display variant = %+v, err=%v", variant, err)
	}
	if _, err := processor.Normalize([]byte("not-an-image")); err == nil {
		t.Fatal("invalid image upload unexpectedly accepted")
	}
	if _, err := processor.Normalize(make([]byte, mediaapp.MaxUploadBytes+1)); err == nil {
		t.Fatal("oversized image upload unexpectedly accepted")
	}
	assertImageWorkerBudget(t, processor)

	fixture.SetNRGBA(0, 0, color.NRGBA{R: 1, G: 2, B: 3, A: 255})
	encoded.Reset()
	_ = png.Encode(&encoded, fixture)
	failedID, _, err := ingestor.Ingest(ctx, encoded.Bytes(), "Ilustrasi retry media", false)
	if err != nil {
		t.Fatalf("ingest retry media: %v", err)
	}
	if err := repository.MarkFailed(ctx, failedID, "VARIANT_PROCESSING_FAILED"); err != nil {
		t.Fatalf("mark failed media: %v", err)
	}
	var actorID string
	if err := adminDatabase.Raw(`SELECT id::text FROM tauco_app.admin_users ORDER BY created_at LIMIT 1`).Scan(&actorID).Error; err != nil || actorID == "" {
		t.Fatalf("load admin actor: id=%q err=%v", actorID, err)
	}
	retried, err := adminService.Retry(ctx, failedID, actorID)
	if err != nil || retried.Status != "processing" {
		t.Fatalf("retry media = %+v, err=%v", retried, err)
	}
	if _, err := adminService.Retry(ctx, assetID, actorID); !errors.Is(err, mediaapp.ErrRetryConflict) {
		t.Fatalf("retry ready media err=%v, want conflict", err)
	}
	assertUploadIntentLifecycle(t, ctx, repository, adminRepository, adminDatabase, actorID, assetID, failedID, encoded.Bytes())
}

func assertImageWorkerBudget(t *testing.T, processor mediaprocessor.Image) {
	t.Helper()
	const processingLimit = 10 * time.Second

	fixture := image.NewNRGBA(image.Rect(0, 0, 4000, 3125))
	state := uint32(0x9e3779b9)
	for offset := 0; offset < len(fixture.Pix); offset += 4 {
		state ^= state << 13
		state ^= state >> 17
		state ^= state << 5
		fixture.Pix[offset] = byte(state)
		fixture.Pix[offset+1] = byte(state >> 8)
		fixture.Pix[offset+2] = byte(state >> 16)
		fixture.Pix[offset+3] = 255
	}
	var source bytes.Buffer
	if err := jpeg.Encode(&source, fixture, &jpeg.Options{Quality: 75}); err != nil {
		t.Fatalf("encode maximum image fixture: %v", err)
	}
	if source.Len() > mediaapp.MaxUploadBytes {
		t.Fatalf("maximum image fixture is %d bytes, want <= %d", source.Len(), mediaapp.MaxUploadBytes)
	}

	started := time.Now()
	normalized, err := processor.Normalize(source.Bytes())
	normalizeDuration := time.Since(started)
	if err != nil {
		t.Fatalf("normalize maximum image fixture: %v", err)
	}
	if len(normalized.Data) > mediaapp.MaxStoredObjectBytes {
		t.Fatalf("normalized maximum image is %d bytes, want <= %d", len(normalized.Data), mediaapp.MaxStoredObjectBytes)
	}
	if normalizeDuration >= processingLimit {
		t.Fatalf("normalize maximum image took %s, budget %s", normalizeDuration, processingLimit)
	}

	started = time.Now()
	if _, _, err := processor.Variants(normalized.Data); err != nil {
		t.Fatalf("generate maximum image variants: %v", err)
	}
	variantDuration := time.Since(started)
	if variantDuration >= processingLimit {
		t.Fatalf("maximum image variants took %s, budget %s", variantDuration, processingLimit)
	}
	t.Logf("maximum image processing: normalize=%s variants=%s source=%dB normalized=%dB", normalizeDuration, variantDuration, source.Len(), len(normalized.Data))
}

func assertUploadIntentLifecycle(
	t *testing.T,
	ctx context.Context,
	runtimeRepository, adminRepository *mediarepo.Postgres,
	adminDatabase *gorm.DB,
	actorID, assetID, recoveryAssetID string,
	source []byte,
) {
	t.Helper()
	digest := sha256.Sum256(source)
	draft := mediadomain.UploadIntentDraft{
		ExpectedMIME: "image/png", ExpectedBytes: int64(len(source)),
		ExpectedSHA256: fmt.Sprintf("%x", digest), AltText: "Ilustrasi upload intent",
		CreatedBy: actorID, ExpiresAt: time.Now().UTC().Add(10 * time.Minute),
	}
	intent, err := adminRepository.CreateUploadIntent(ctx, draft)
	if err != nil || intent.Status != mediadomain.UploadStatusPending || intent.QuarantineKey != "quarantine/"+intent.ID {
		t.Fatalf("create upload intent = %+v, err=%v", intent, err)
	}
	if got, err := runtimeRepository.GetUploadIntent(ctx, intent.ID); err != nil || got.ID != intent.ID {
		t.Fatalf("runtime get upload intent = %+v, err=%v", got, err)
	}
	if _, _, err := adminRepository.QueueUploadIntent(
		ctx, intent.ID, draft.ExpectedMIME, draft.ExpectedBytes, strings.Repeat("0", 64),
	); !errors.Is(err, mediaapp.ErrUploadIntentMetadata) {
		t.Fatalf("queue mismatched upload intent err=%v", err)
	}
	queued, replayed, err := adminRepository.QueueUploadIntent(
		ctx, intent.ID, draft.ExpectedMIME, draft.ExpectedBytes, draft.ExpectedSHA256,
	)
	if err != nil || replayed || queued.Status != mediadomain.UploadStatusQueued {
		t.Fatalf("queue upload intent = %+v replay=%t err=%v", queued, replayed, err)
	}
	if _, replayed, err := adminRepository.QueueUploadIntent(
		ctx, intent.ID, draft.ExpectedMIME, draft.ExpectedBytes, draft.ExpectedSHA256,
	); err != nil || !replayed {
		t.Fatalf("replay upload intent = %t/%v", replayed, err)
	}
	var ingestJobs int64
	if err := adminDatabase.Raw(`SELECT count(*) FROM tauco_app.background_jobs WHERE idempotency_key = ?`, "media.ingest:"+intent.ID).Scan(&ingestJobs).Error; err != nil || ingestJobs != 1 {
		t.Fatalf("upload ingest jobs = %d, err=%v", ingestJobs, err)
	}
	if err := runtimeRepository.CompleteUploadIntent(ctx, intent.ID, assetID); err != nil {
		t.Fatalf("complete upload intent: %v", err)
	}
	if err := runtimeRepository.CompleteUploadIntent(ctx, intent.ID, assetID); err != nil {
		t.Fatalf("replay complete upload intent: %v", err)
	}
	completed, err := adminRepository.GetUploadIntent(ctx, intent.ID)
	if err != nil || completed.Status != mediadomain.UploadStatusCompleted || completed.MediaAssetID == nil || *completed.MediaAssetID != assetID {
		t.Fatalf("completed upload intent = %+v, err=%v", completed, err)
	}

	failed, err := adminRepository.CreateUploadIntent(ctx, draft)
	if err != nil {
		t.Fatalf("create failed-path upload intent: %v", err)
	}
	if _, _, err := adminRepository.QueueUploadIntent(ctx, failed.ID, draft.ExpectedMIME, draft.ExpectedBytes, draft.ExpectedSHA256); err != nil {
		t.Fatalf("queue failed-path upload intent: %v", err)
	}
	if err := runtimeRepository.FailUploadIntent(ctx, failed.ID, "INGEST_VALIDATION_FAILED"); err != nil {
		t.Fatalf("fail upload intent: %v", err)
	}
	if err := runtimeRepository.FailUploadIntent(ctx, failed.ID, "INGEST_VALIDATION_FAILED"); err != nil {
		t.Fatalf("replay fail upload intent: %v", err)
	}
	if err := runtimeRepository.CompleteUploadIntent(ctx, failed.ID, recoveryAssetID); err != nil {
		t.Fatalf("recover failed upload intent: %v", err)
	}

	expired, err := adminRepository.CreateUploadIntent(ctx, draft)
	if err != nil {
		t.Fatalf("create cleanup upload intent: %v", err)
	}
	if err := adminDatabase.Exec(`
		UPDATE tauco_app.media_upload_intents
		SET created_at = statement_timestamp() - interval '30 minutes',
			expires_at = statement_timestamp() - interval '20 minutes'
		WHERE id = ?`, expired.ID).Error; err != nil {
		t.Fatalf("age cleanup upload intent: %v", err)
	}
	claimed, err := runtimeRepository.ClaimUploadIntentCleanup(ctx, 10)
	if err != nil || len(claimed) != 1 || claimed[0].ID != expired.ID || claimed[0].Status != mediadomain.UploadStatusExpired {
		t.Fatalf("claim upload cleanup = %+v, err=%v", claimed, err)
	}
	if err := runtimeRepository.CompleteUploadIntentCleanup(ctx, expired.ID); err != nil {
		t.Fatalf("complete upload cleanup: %v", err)
	}
	if err := runtimeRepository.CompleteUploadIntentCleanup(ctx, expired.ID); err != nil {
		t.Fatalf("replay upload cleanup: %v", err)
	}
	var retained int64
	if err := adminDatabase.Raw(`SELECT count(*) FROM tauco_app.media_upload_intents`).Scan(&retained).Error; err != nil || retained != 3 {
		t.Fatalf("retained upload intents = %d, err=%v", retained, err)
	}
	assertDirectUploadPipeline(t, ctx, runtimeRepository, adminRepository, adminDatabase, actorID)
}

func assertDirectUploadPipeline(
	t *testing.T,
	ctx context.Context,
	runtimeRepository, adminRepository *mediarepo.Postgres,
	adminDatabase *gorm.DB,
	actorID string,
) {
	t.Helper()
	store := newIntegrationQuarantineStore()
	uploads, _ := mediaapp.NewUploadService(adminRepository, store)
	processor := mediaprocessor.Image{}
	ingestor, _ := mediaapp.NewIngestor(runtimeRepository, store, processor)
	uploadHandler, _ := mediaapp.NewUploadHandler(runtimeRepository, store, ingestor)

	fixture := image.NewNRGBA(image.Rect(0, 0, 720, 480))
	for y := range 480 {
		for x := range 720 {
			fixture.SetNRGBA(x, y, color.NRGBA{R: 40, G: uint8(x % 251), B: uint8(y % 251), A: 255})
		}
	}
	var encoded bytes.Buffer
	if err := png.Encode(&encoded, fixture); err != nil {
		t.Fatalf("encode direct-upload fixture: %v", err)
	}
	source := encoded.Bytes()
	digest := sha256.Sum256(source)
	created, err := uploads.Create(ctx, mediaapp.CreateUploadIntentInput{
		MIMEType: "image/png", Bytes: int64(len(source)), SHA256: fmt.Sprintf("%x", digest),
		AltText: "Ilustrasi direct upload", CreatedBy: actorID,
	})
	if err != nil || created.Intent.Status != mediadomain.UploadStatusPending || created.Upload.URL == "" {
		t.Fatalf("create direct upload = %+v, err=%v", created, err)
	}
	if err := store.PutIfAbsent(ctx, created.Intent.QuarantineKey, "image/png", source); err != nil {
		t.Fatalf("put direct-upload object: %v", err)
	}
	queued, replayed, err := uploads.Finalize(ctx, created.Intent.ID)
	if err != nil || replayed || queued.Status != mediadomain.UploadStatusQueued {
		t.Fatalf("finalize direct upload = %+v replay=%t err=%v", queued, replayed, err)
	}
	payload, _ := json.Marshal(map[string]string{"uploadIntentId": created.Intent.ID})
	if err := uploadHandler.Handle(ctx, payload); err != nil {
		t.Fatalf("handle direct upload: %v", err)
	}
	completed, err := uploads.Get(ctx, created.Intent.ID)
	if err != nil || completed.Status != mediadomain.UploadStatusCompleted || completed.MediaAssetID == nil {
		t.Fatalf("completed direct upload = %+v, err=%v", completed, err)
	}
	if err := uploadHandler.Handle(ctx, payload); err != nil {
		t.Fatalf("replay direct upload handler: %v", err)
	}
	variantHandler, _ := mediaapp.NewVariantHandler(runtimeRepository, store, processor)
	variantPayload, _ := json.Marshal(map[string]string{"mediaAssetId": *completed.MediaAssetID})
	if err := variantHandler.Handle(ctx, variantPayload); err != nil {
		t.Fatalf("direct-upload variant handler: %v", err)
	}
	asset, err := adminRepository.GetAdmin(ctx, *completed.MediaAssetID)
	if err != nil || asset.Status != mediadomain.StatusReady || len(asset.Variants) == 0 {
		t.Fatalf("direct-upload media asset = %+v, err=%v", asset, err)
	}
	if _, err := store.Get(ctx, created.Intent.QuarantineKey); !errors.Is(err, mediaapp.ErrObjectNotFound) {
		t.Fatalf("quarantine object remains after ingest: %v", err)
	}

	invalid := []byte("not-a-valid-image")
	invalidDigest := sha256.Sum256(invalid)
	invalidCreated, err := uploads.Create(ctx, mediaapp.CreateUploadIntentInput{
		MIMEType: "image/png", Bytes: int64(len(invalid)), SHA256: fmt.Sprintf("%x", invalidDigest),
		AltText: "Ilustrasi invalid upload", CreatedBy: actorID,
	})
	if err != nil {
		t.Fatalf("create invalid direct upload: %v", err)
	}
	_ = store.PutIfAbsent(ctx, invalidCreated.Intent.QuarantineKey, "image/png", invalid)
	if _, _, err := uploads.Finalize(ctx, invalidCreated.Intent.ID); err != nil {
		t.Fatalf("finalize invalid direct upload metadata: %v", err)
	}
	invalidPayload, _ := json.Marshal(map[string]string{"uploadIntentId": invalidCreated.Intent.ID})
	if err := uploadHandler.Handle(ctx, invalidPayload); err != nil {
		t.Fatalf("invalid upload should be a permanent handled failure: %v", err)
	}
	failed, _ := uploads.Get(ctx, invalidCreated.Intent.ID)
	if failed.Status != mediadomain.UploadStatusFailed || failed.LastErrorCode == nil {
		t.Fatalf("invalid direct upload status = %+v", failed)
	}

	cleanupCreated, err := uploads.Create(ctx, mediaapp.CreateUploadIntentInput{
		MIMEType: "image/png", Bytes: int64(len(source)), SHA256: fmt.Sprintf("%x", digest),
		AltText: "Ilustrasi cleanup upload", CreatedBy: actorID,
	})
	if err != nil {
		t.Fatalf("create cleanup direct upload: %v", err)
	}
	_ = store.PutIfAbsent(ctx, cleanupCreated.Intent.QuarantineKey, "image/png", source)
	if err := adminDatabase.Exec(`
		UPDATE tauco_app.media_upload_intents
		SET created_at = statement_timestamp() - interval '30 minutes',
			expires_at = statement_timestamp() - interval '20 minutes'
		WHERE id = ?`, cleanupCreated.Intent.ID).Error; err != nil {
		t.Fatalf("age direct-upload cleanup intent: %v", err)
	}
	cleanup, _ := mediaapp.NewUploadCleanup(runtimeRepository, store)
	if count, err := cleanup.RunOnce(ctx, 10); err != nil || count != 1 {
		t.Fatalf("direct-upload cleanup count=%d err=%v", count, err)
	}
	if _, err := store.Get(ctx, cleanupCreated.Intent.QuarantineKey); !errors.Is(err, mediaapp.ErrObjectNotFound) {
		t.Fatalf("cleanup quarantine object remains: %v", err)
	}

	deadCreated, err := uploads.Create(ctx, mediaapp.CreateUploadIntentInput{
		MIMEType: "image/png", Bytes: int64(len(source)), SHA256: fmt.Sprintf("%x", digest),
		AltText: "Ilustrasi dead-job cleanup", CreatedBy: actorID,
	})
	if err != nil {
		t.Fatalf("create dead-job upload: %v", err)
	}
	_ = store.PutIfAbsent(ctx, deadCreated.Intent.QuarantineKey, "image/png", source)
	if _, _, err := uploads.Finalize(ctx, deadCreated.Intent.ID); err != nil {
		t.Fatalf("finalize dead-job upload: %v", err)
	}
	if err := adminDatabase.Exec(`
		UPDATE tauco_app.media_upload_intents
		SET created_at = statement_timestamp() - interval '30 minutes',
			expires_at = statement_timestamp() - interval '20 minutes'
		WHERE id = ?`, deadCreated.Intent.ID).Error; err != nil {
		t.Fatalf("age dead-job upload intent: %v", err)
	}
	if err := adminDatabase.Exec(`
		UPDATE tauco_app.background_jobs
		SET status = 'dead', attempts = max_attempts,
			dead_at = statement_timestamp(), run_at = statement_timestamp()
		WHERE idempotency_key = ?`, "media.ingest:"+deadCreated.Intent.ID).Error; err != nil {
		t.Fatalf("mark upload ingest job dead: %v", err)
	}
	if count, err := cleanup.RunOnce(ctx, 10); err != nil || count != 1 {
		t.Fatalf("dead-job upload cleanup count=%d err=%v", count, err)
	}
	deadIntent, _ := uploads.Get(ctx, deadCreated.Intent.ID)
	if deadIntent.Status != mediadomain.UploadStatusFailed || deadIntent.LastErrorCode == nil || *deadIntent.LastErrorCode != "INGEST_JOB_DEAD" {
		t.Fatalf("dead-job upload intent = %+v", deadIntent)
	}
}

type integrationQuarantineStore struct {
	objects  map[string][]byte
	metadata map[string]mediaapp.ObjectInfo
}

func newIntegrationQuarantineStore() *integrationQuarantineStore {
	return &integrationQuarantineStore{objects: make(map[string][]byte), metadata: make(map[string]mediaapp.ObjectInfo)}
}

func (store *integrationQuarantineStore) PutIfAbsent(_ context.Context, key, mime string, data []byte) error {
	if current, found := store.objects[key]; found {
		if !bytes.Equal(current, data) {
			return errors.New("integration object conflict")
		}
		return nil
	}
	store.objects[key] = append([]byte(nil), data...)
	if _, found := store.metadata[key]; !found {
		digest := sha256.Sum256(data)
		store.metadata[key] = mediaapp.ObjectInfo{Bytes: int64(len(data)), MIMEType: mime, SHA256: fmt.Sprintf("%x", digest)}
	}
	return nil
}

func (store *integrationQuarantineStore) Get(_ context.Context, key string) ([]byte, error) {
	return store.GetBounded(context.Background(), key, mediaapp.MaxStoredObjectBytes)
}

func (store *integrationQuarantineStore) GetBounded(_ context.Context, key string, maximum int64) ([]byte, error) {
	data, found := store.objects[key]
	if !found {
		return nil, mediaapp.ErrObjectNotFound
	}
	if int64(len(data)) > maximum {
		return nil, mediaapp.ErrObjectTooLarge
	}
	return append([]byte(nil), data...), nil
}

func (store *integrationQuarantineStore) PresignPut(
	_ context.Context,
	key, mime, sha256Value string,
	bytesValue int64,
	lifetime time.Duration,
) (mediaapp.PresignedUpload, error) {
	store.metadata[key] = mediaapp.ObjectInfo{Bytes: bytesValue, MIMEType: mime, SHA256: sha256Value}
	return mediaapp.PresignedUpload{
		URL: "https://storage.example/" + key,
		Headers: map[string]string{
			"Content-Type": mime, "X-Amz-Meta-Sha256": sha256Value,
		},
		ExpiresAt: time.Now().UTC().Add(lifetime),
	}, nil
}

func (store *integrationQuarantineStore) Head(_ context.Context, key string) (mediaapp.ObjectInfo, error) {
	data, found := store.objects[key]
	if !found {
		return mediaapp.ObjectInfo{}, mediaapp.ErrObjectNotFound
	}
	metadata := store.metadata[key]
	metadata.Bytes = int64(len(data))
	return metadata, nil
}

func (store *integrationQuarantineStore) Delete(_ context.Context, key string) error {
	delete(store.objects, key)
	return nil
}

func (*integrationQuarantineStore) Health(context.Context) error { return nil }

var _ mediaapp.QuarantineStore = (*integrationQuarantineStore)(nil)

func intPointer(value int) *int { return &value }

func assertAdminContentLifecycle(t *testing.T, ctx context.Context, adminDatabase *gorm.DB, plan contentapp.SeedPlan) {
	t.Helper()
	repository, err := contentrepo.NewAdminPostgres(adminDatabase)
	if err != nil {
		t.Fatalf("NewAdminPostgres(content): %v", err)
	}
	service, _ := contentapp.NewAdminService(repository)
	page, err := service.Get(ctx, "home")
	if err != nil {
		t.Fatalf("get admin home: %v", err)
	}
	var actorID string
	if err := adminDatabase.Raw(`SELECT id::text FROM tauco_app.admin_users ORDER BY created_at LIMIT 1`).Scan(&actorID).Error; err != nil || actorID == "" {
		t.Fatalf("load content actor: %v", err)
	}
	var homeContent json.RawMessage
	for _, item := range plan.Pages {
		if item.Key == contentdomain.PageKeyHome {
			homeContent = item.Revision.ContentJSON
		}
	}
	if len(homeContent) == 0 {
		t.Fatal("home seed content missing")
	}

	type result struct {
		revision contentapp.AdminRevision
		err      error
	}
	start := make(chan struct{})
	results := make(chan result, 2)
	for range 2 {
		go func() {
			<-start
			revision, saveErr := service.SaveDraft(ctx, "home", contentapp.RevisionETag(page.Latest.ID), page.Latest.ID, actorID, homeContent)
			results <- result{revision, saveErr}
		}()
	}
	close(start)
	var draft contentapp.AdminRevision
	preconditions := 0
	for range 2 {
		item := <-results
		if item.err == nil {
			draft = item.revision
		} else if errors.Is(item.err, contentapp.ErrPrecondition) {
			preconditions++
		} else {
			t.Fatalf("concurrent draft error: %v", item.err)
		}
	}
	if draft.ID == "" || preconditions != 1 {
		t.Fatalf("concurrent draft=%+v preconditions=%d", draft, preconditions)
	}
	if _, err := service.SaveDraft(ctx, "tauco-guide", contentapp.RevisionETag(draft.ID), draft.ID, actorID, homeContent); !errors.Is(err, contentapp.ErrAdminPageNotFound) {
		t.Fatalf("tauco guide mutation err=%v", err)
	}

	published, err := service.Publish(ctx, "home", draft.ID, contentapp.RevisionETag(draft.ID), actorID)
	if err != nil || published.Status != "published" || published.ID == draft.ID {
		t.Fatalf("publish home=%+v err=%v", published, err)
	}
	afterPublish, _ := service.Get(ctx, "home")
	if afterPublish.PublishedRevisionID == nil || *afterPublish.PublishedRevisionID != published.ID {
		t.Fatalf("published pointer=%v", afterPublish.PublishedRevisionID)
	}
	if err := service.Unpublish(ctx, "home", contentapp.RevisionETag(published.ID), actorID); err != nil {
		t.Fatalf("unpublish home: %v", err)
	}
	afterUnpublish, _ := service.Get(ctx, "home")
	if afterUnpublish.PublishedRevisionID != nil {
		t.Fatalf("unpublished pointer=%v", afterUnpublish.PublishedRevisionID)
	}

	var jobCount, auditCount int64
	if err := adminDatabase.Raw(`SELECT count(*) FROM tauco_app.background_jobs WHERE kind='content.invalidate_cache'`).Scan(&jobCount).Error; err != nil {
		t.Fatal(err)
	}
	if err := adminDatabase.Raw(`SELECT count(*) FROM tauco_app.activity_logs WHERE event_type IN ('content.draft_saved','content.published','content.unpublished')`).Scan(&auditCount).Error; err != nil {
		t.Fatal(err)
	}
	if jobCount != 2 || auditCount < 3 {
		t.Fatalf("content jobs=%d audit=%d", jobCount, auditCount)
	}

	var processingID string
	if err := adminDatabase.Raw(`SELECT id::text FROM tauco_app.media_assets WHERE status='processing' ORDER BY created_at DESC LIMIT 1`).Scan(&processingID).Error; err != nil || processingID == "" {
		t.Fatalf("load processing media: %v", err)
	}
	blockedRaw := bytes.Replace(homeContent, []byte(`/images/tauco-hero-provisional.webp`), []byte(`/api/v1/media/`+processingID+`/display.webp`), 1)
	blockedDraft, err := service.SaveDraft(ctx, "home", contentapp.RevisionETag(afterUnpublish.Latest.ID), afterUnpublish.Latest.ID, actorID, blockedRaw)
	if err != nil {
		t.Fatalf("save blocked media draft: %v", err)
	}
	if _, err := service.Publish(ctx, "home", blockedDraft.ID, contentapp.RevisionETag(blockedDraft.ID), actorID); !errors.Is(err, contentapp.ErrMediaNotReady) {
		t.Fatalf("publish processing media err=%v", err)
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
