//go:build integration

package earchiva

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"

	appdb "github.com/eguilde/egueducation/internal/db"
)

// These tests intentionally require an explicit disposable PostgreSQL endpoint.
// They create and remove their own database and never use application settings.
func TestArchiveMigrationsAndTenantIsolationIntegration(t *testing.T) {
	it := newArchiveIntegrationDatabase(t)
	ctx := context.Background()
	pool := openArchiveIntegrationPool(t, ctx, it.databaseConfig)
	defer pool.Close()

	if err := appdb.Migrate(ctx, pool); err != nil {
		t.Fatalf("migrate clean integration database: %v", err)
	}
	grantArchiveIntegrationAccess(t, ctx, pool)
	assertArchiveMigrationsRecorded(t, ctx, pool)

	service := NewDocumentService(appdb.NewSessionPool(it.readerPool), nil)
	seedArchiveTenantDocument(t, ctx, pool, "inst-001", "Document A", "idempotent-a")
	documentB := seedArchiveTenantDocument(t, ctx, pool, "inst-balotesti", "Document B", "idempotent-b")

	ctxA, releaseA := archiveTenantContext(t, ctx, it.readerPool, "tenant-egueducation", "inst-001")
	defer releaseA()
	if _, err := service.loadDocumentDetail(ctxA, "inst-001", documentB); err != pgx.ErrNoRows {
		t.Fatalf("tenant A must not load tenant B document: got %v", err)
	}
	var visible int
	if err := appdb.NewSessionPool(it.readerPool).QueryRow(ctxA,
		`select count(*) from archive_documents where institution_id = $1`, "inst-balotesti").Scan(&visible); err != nil {
		t.Fatalf("tenant-scoped search query: %v", err)
	}
	if visible != 0 {
		t.Fatalf("tenant A search leaked %d tenant B document(s)", visible)
	}
	if err := appdb.NewSessionPool(it.readerPool).QueryRow(ctxA,
		`select count(*) from archive_document_versions where document_id = $1::uuid and institution_id = $2`, documentB, "inst-001").Scan(&visible); err != nil {
		t.Fatalf("tenant-scoped versions query: %v", err)
	}
	if visible != 0 {
		t.Fatalf("tenant A versions leaked %d tenant B version(s)", visible)
	}

	var documentA string
	if err := pool.QueryRow(ctx, `select id::text from archive_documents where institution_id = 'inst-001' limit 1`).Scan(&documentA); err != nil {
		t.Fatalf("load tenant A document id: %v", err)
	}
	_, err := pool.Exec(ctx, `
		insert into archive_document_versions (
			institution_id, document_id, version_no, mime_type, title, bucket_name, object_key,
			hash_sha256, size_bytes, status, source_bucket, source_object_key, source_sha256
		) values ('inst-001', $1::uuid, 2, 'application/pdf', 'cross tenant', 'archive', 'cross.pdf', 'a', 1, 'active', 'archive', 'cross.pdf', 'a')
	`, documentB)
	assertPostgresConstraint(t, err, "composite archive document/version tenant FK")

	jobID := seedArchiveJob(t, ctx, pool, documentA, "inst-001", "running", time.Now().Add(-2*time.Hour))
	worker := NewIngestionWorkerWithMaxAttempts(appdb.NewSessionPool(it.readerPool), nil, nil, nil, time.Second, 1)
	job, err := worker.claimJob(ctx)
	if err != nil {
		t.Fatalf("claim stale archive job: %v", err)
	}
	if job == nil || job.ID != jobID || job.Attempts != 1 {
		t.Fatalf("stale job was not reclaimed and claimed: %#v", job)
	}
	if err := worker.failJob(ctxA, job, fmt.Errorf("archive OCR produced no searchable text")); err == nil {
		t.Fatal("empty OCR output must fail the claimed job")
	}
	var status string
	if err := appdb.NewSessionPool(it.readerPool).QueryRow(ctxA, `select status from archive_ingestion_jobs where id::text = $1`, jobID).Scan(&status); err != nil {
		t.Fatalf("read failed OCR job: %v", err)
	}
	if status != "failed" {
		t.Fatalf("empty OCR job status = %q, want failed", status)
	}

	_, err = pool.Exec(ctx, `
		insert into archive_documents (institution_id, title, original_file_name, mime_type, source_kind, status, idempotency_key)
		values ('inst-001', 'Repeated', 'repeated.pdf', 'application/pdf', 'import', 'queued', 'idempotent-a')
	`)
	assertPostgresConstraint(t, err, "per-tenant archive idempotency key")
	if _, err := pool.Exec(ctx, `
		insert into archive_documents (institution_id, title, original_file_name, mime_type, source_kind, status, idempotency_key)
		values ('inst-balotesti', 'Same idempotency key other tenant', 'same.pdf', 'application/pdf', 'import', 'queued', 'idempotent-a')
	`); err != nil {
		t.Fatalf("idempotency key must be reusable in another tenant: %v", err)
	}
}

func TestArchiveLegacyMigrationIntegration(t *testing.T) {
	it := newArchiveIntegrationDatabase(t)
	ctx := context.Background()
	pool := openArchiveIntegrationPool(t, ctx, it.databaseConfig)
	defer pool.Close()

	applyArchiveMigrationRange(t, ctx, pool, "0001", "0074")
	legacyDocumentID := uuid.NewString()
	legacyVersionID := uuid.NewString()
	if _, err := pool.Exec(ctx, `
		insert into archive_documents (id, institution_id, title, original_file_name, mime_type, source_kind, status)
		values ($1::uuid, 'inst-001', 'Legacy OCR document', 'legacy.pdf', 'application/pdf', 'legacy_pdf', 'queued')
	`, legacyDocumentID); err != nil {
		t.Fatalf("seed legacy archive document: %v", err)
	}
	if _, err := pool.Exec(ctx, `
		insert into archive_document_versions (
			id, document_id, institution_id, version_no, mime_type, title, bucket_name, object_key, hash_sha256, size_bytes, ocr_text, status
		) values ($1::uuid, $2::uuid, 'inst-001', 1, 'application/pdf', 'Legacy OCR document', 'legacy', 'legacy.pdf', 'legacy-hash', 42, 'legacy searchable text', 'active')
	`, legacyVersionID, legacyDocumentID); err != nil {
		t.Fatalf("seed legacy archive version: %v", err)
	}
	if _, err := pool.Exec(ctx, `
		insert into archive_ingestion_jobs (institution_id, status, source_bucket, source_key, document_id)
		values ('inst-001', 'completed', 'legacy', 'legacy.pdf', $1::uuid),
		       ('inst-001', 'pending', 'legacy', 'legacy.pdf', $1::uuid)
	`, legacyDocumentID); err != nil {
		t.Fatalf("seed legacy ingestion jobs: %v", err)
	}

	applyArchiveMigrationRange(t, ctx, pool, "0075", "0078")
	var sourceBucket, sourceKey, sourceHash, extractedText, textStatus string
	if err := pool.QueryRow(ctx, `
		select source_bucket, source_object_key, source_sha256, extracted_text, text_status
		from archive_document_versions where id::text = $1
	`, legacyVersionID).Scan(&sourceBucket, &sourceKey, &sourceHash, &extractedText, &textStatus); err != nil {
		t.Fatalf("read reconciled legacy version: %v", err)
	}
	if sourceBucket != "legacy" || sourceKey != "legacy.pdf" || sourceHash != "legacy-hash" || extractedText != "legacy searchable text" || textStatus != "processed" {
		t.Fatalf("legacy version reconciliation invalid: bucket=%q key=%q hash=%q text=%q status=%q", sourceBucket, sourceKey, sourceHash, extractedText, textStatus)
	}
	var completed, pendingVersionID string
	if err := pool.QueryRow(ctx, `select status from archive_ingestion_jobs where status = 'succeeded' limit 1`).Scan(&completed); err != nil {
		t.Fatalf("legacy completed job was not converted to succeeded: %v", err)
	}
	if completed != "succeeded" {
		t.Fatalf("legacy completed job status = %q", completed)
	}
	if err := pool.QueryRow(ctx, `select version_id::text from archive_ingestion_jobs where status = 'pending' limit 1`).Scan(&pendingVersionID); err != nil {
		t.Fatalf("legacy pending job must receive immutable version: %v", err)
	}
	if pendingVersionID != legacyVersionID {
		t.Fatalf("legacy pending job version = %q, want %q", pendingVersionID, legacyVersionID)
	}
}

type archiveIntegrationDatabase struct {
	databaseConfig *pgxpool.Config
	readerPool     *pgxpool.Pool
}

func newArchiveIntegrationDatabase(t *testing.T) archiveIntegrationDatabase {
	t.Helper()
	baseURL := strings.TrimSpace(os.Getenv("TEST_DATABASE_URL"))
	if baseURL == "" {
		t.Skip("TEST_DATABASE_URL is not set; PostgreSQL integration tests are intentionally skipped")
	}
	ctx := context.Background()
	baseConfig, err := pgx.ParseConfig(baseURL)
	if err != nil {
		t.Fatalf("parse TEST_DATABASE_URL: %v", err)
	}
	admin, err := pgx.ConnectConfig(ctx, baseConfig)
	if err != nil {
		t.Fatalf("connect TEST_DATABASE_URL: %v", err)
	}
	databaseName := "earchiva_it_" + strings.ReplaceAll(uuid.NewString()[:12], "-", "")
	roleName := "earchiva_it_u_" + strings.ReplaceAll(uuid.NewString()[:12], "-", "")
	rolePassword := uuid.NewString()
	// Do not inherit objects from a locally customized template1. Integration
	// fixtures must always start from the pristine PostgreSQL template0.
	if _, err := admin.Exec(ctx, "create database "+quoteArchiveIdentifier(databaseName)+" template template0"); err != nil {
		admin.Close(ctx)
		t.Fatalf("create disposable integration database: %v", err)
	}
	if _, err := admin.Exec(ctx, "create role "+quoteArchiveIdentifier(roleName)+" login nosuperuser nobypassrls password "+quoteArchiveLiteral(rolePassword)); err != nil {
		_, _ = admin.Exec(ctx, "drop database "+quoteArchiveIdentifier(databaseName))
		admin.Close(ctx)
		t.Fatalf("create restricted integration role: %v", err)
	}
	admin.Close(ctx)

	targetConfig, err := pgxpool.ParseConfig(baseURL)
	if err != nil {
		t.Fatalf("parse target pool config: %v", err)
	}
	targetConfig.ConnConfig.Database = databaseName
	readerConfig := targetConfig.Copy()
	readerConfig.ConnConfig.User = roleName
	readerConfig.ConnConfig.Password = rolePassword
	readerPool, err := pgxpool.NewWithConfig(ctx, readerConfig)
	if err != nil {
		t.Fatalf("open restricted integration pool: %v", err)
	}
	t.Cleanup(func() {
		readerPool.Close()
		cleanup, cleanupErr := pgx.ConnectConfig(context.Background(), baseConfig)
		if cleanupErr != nil {
			t.Errorf("connect to remove integration database: %v", cleanupErr)
			return
		}
		defer cleanup.Close(context.Background())
		if _, err := cleanup.Exec(context.Background(), "drop database if exists "+quoteArchiveIdentifier(databaseName)+" with (force)"); err != nil {
			t.Errorf("drop disposable integration database: %v", err)
		}
		if _, err := cleanup.Exec(context.Background(), "drop role if exists "+quoteArchiveIdentifier(roleName)); err != nil {
			t.Errorf("drop restricted integration role: %v", err)
		}
	})
	return archiveIntegrationDatabase{databaseConfig: targetConfig, readerPool: readerPool}
}

func openArchiveIntegrationPool(t *testing.T, ctx context.Context, databaseConfig *pgxpool.Config) *pgxpool.Pool {
	t.Helper()
	pool, err := pgxpool.NewWithConfig(ctx, databaseConfig.Copy())
	if err != nil {
		t.Fatalf("open integration database: %v", err)
	}
	var currentDatabase string
	if err := pool.QueryRow(ctx, `select current_database()`).Scan(&currentDatabase); err != nil || currentDatabase != databaseConfig.ConnConfig.Database {
		pool.Close()
		t.Fatalf("integration pool database mismatch: got %q want %q (err=%v)", currentDatabase, databaseConfig.ConnConfig.Database, err)
	}
	return pool
}

func applyArchiveMigrationRange(t *testing.T, ctx context.Context, pool *pgxpool.Pool, first, last string) {
	t.Helper()
	files, err := filepath.Glob(filepath.Join("..", "db", "migrations", "*.sql"))
	if err != nil {
		t.Fatalf("list migration files: %v", err)
	}
	sort.Strings(files)
	for _, file := range files {
		name := filepath.Base(file)
		prefix := strings.SplitN(name, "_", 2)[0]
		if prefix < first || prefix > last {
			continue
		}
		content, err := os.ReadFile(file)
		if err != nil {
			t.Fatalf("read migration %s: %v", name, err)
		}
		tx, err := pool.Begin(ctx)
		if err != nil {
			t.Fatalf("begin migration %s: %v", name, err)
		}
		if _, err := tx.Exec(ctx, string(content)); err != nil {
			_ = tx.Rollback(ctx)
			t.Fatalf("apply migration %s: %v", name, err)
		}
		if err := tx.Commit(ctx); err != nil {
			t.Fatalf("commit migration %s: %v", name, err)
		}
	}
	grantArchiveIntegrationAccess(t, ctx, pool)
}

func grantArchiveIntegrationAccess(t *testing.T, ctx context.Context, pool *pgxpool.Pool) {
	t.Helper()
	if _, err := pool.Exec(ctx, "grant usage on schema public to public; grant select, insert, update, delete on all tables in schema public to public; grant usage, select on all sequences in schema public to public"); err != nil {
		t.Fatalf("grant restricted test role archive access: %v", err)
	}
}

func assertArchiveMigrationsRecorded(t *testing.T, ctx context.Context, pool *pgxpool.Pool) {
	t.Helper()
	for _, name := range []string{
		"0068_earchiva_documents.sql", "0069_earchiva_vector_search.sql", "0070_earchiva_schema_contract.sql", "0071_tenant_scoped_identity_grants.sql", "0072_registratura_tenant_parity_foundation.sql", "0073_balotesti_platform_superadmin_login.sql", "0074_tenant_hostname_directory.sql", "0075_earchiva_ingestion_contract.sql", "0076_earchiva_upload_integrity.sql", "0077_earchiva_classification_review.sql", "0078_audit_log_tenant_isolation.sql",
	} {
		var found bool
		if err := pool.QueryRow(ctx, `select exists(select 1 from schema_migrations where version = $1)`, name).Scan(&found); err != nil || !found {
			t.Fatalf("migration %s not recorded (found=%t err=%v)", name, found, err)
		}
	}
}

func seedArchiveTenantDocument(t *testing.T, ctx context.Context, pool *pgxpool.Pool, institutionID, title, idempotencyKey string) string {
	t.Helper()
	documentID := uuid.NewString()
	versionID := uuid.NewString()
	if _, err := pool.Exec(ctx, `
		insert into archive_documents (id, institution_id, title, original_file_name, mime_type, source_kind, status, idempotency_key)
		values ($1::uuid, $2, $3, 'archive.pdf', 'application/pdf', 'import', 'queued', $4)
	`, documentID, institutionID, title, idempotencyKey); err != nil {
		t.Fatalf("seed archive document: %v", err)
	}
	if _, err := pool.Exec(ctx, `
		insert into archive_document_versions (
			id, institution_id, document_id, version_no, mime_type, title, bucket_name, object_key, hash_sha256, size_bytes, status,
			source_bucket, source_object_key, source_sha256
		) values ($1::uuid, $2, $3::uuid, 1, 'application/pdf', $4, 'archive', 'archive.pdf', 'hash', 1, 'active', 'archive', 'archive.pdf', 'hash')
	`, versionID, institutionID, documentID, title); err != nil {
		t.Fatalf("seed archive version: %v", err)
	}
	return documentID
}

func seedArchiveJob(t *testing.T, ctx context.Context, pool *pgxpool.Pool, documentID, institutionID, status string, lockedAt time.Time) string {
	t.Helper()
	var versionID, jobID string
	if err := pool.QueryRow(ctx, `select id::text from archive_document_versions where document_id::text = $1`, documentID).Scan(&versionID); err != nil {
		t.Fatalf("lookup archive version for job: %v", err)
	}
	if err := pool.QueryRow(ctx, `
		insert into archive_ingestion_jobs (institution_id, document_id, version_id, status, source_bucket, source_key, job_type, available_at, locked_at, locked_by)
		values ($1, $2::uuid, $3::uuid, $4, 'archive', 'archive.pdf', 'extract_text', now() - interval '3 hours', $5, 'abandoned-worker')
		returning id::text
	`, institutionID, documentID, versionID, status, lockedAt).Scan(&jobID); err != nil {
		t.Fatalf("seed archive ingestion job: %v", err)
	}
	return jobID
}

func archiveTenantContext(t *testing.T, ctx context.Context, pool *pgxpool.Pool, tenantID, institutionID string) (context.Context, func()) {
	t.Helper()
	bound, release, err := appdb.AcquireRequestConn(ctx, pool, appdb.SessionConfig{TenantID: tenantID, InstitutionID: institutionID, ActorSubject: "archive-integration-test"})
	if err != nil {
		t.Fatalf("bind restricted tenant session: %v", err)
	}
	return bound, release
}

func assertPostgresConstraint(t *testing.T, err error, label string) {
	t.Helper()
	if err == nil {
		t.Fatalf("%s unexpectedly succeeded", label)
	}
	var pgErr *pgconn.PgError
	if !strings.Contains(err.Error(), "violates") || !strings.Contains(err.Error(), "constraint") || !asPostgresError(err, &pgErr) || (pgErr.Code != "23503" && pgErr.Code != "23505") {
		t.Fatalf("%s must fail with an integrity constraint, got %v", label, err)
	}
}

func asPostgresError(err error, target **pgconn.PgError) bool {
	for err != nil {
		if pgErr, ok := err.(*pgconn.PgError); ok {
			*target = pgErr
			return true
		}
		type unwrapper interface{ Unwrap() error }
		wrapped, ok := err.(unwrapper)
		if !ok {
			break
		}
		err = wrapped.Unwrap()
	}
	return false
}

func quoteArchiveIdentifier(value string) string {
	return `"` + strings.ReplaceAll(value, `"`, `""`) + `"`
}
func quoteArchiveLiteral(value string) string {
	return `'` + strings.ReplaceAll(value, `'`, `''`) + `'`
}
