//go:build integration

package earchiva

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"strings"
	"testing"
	"time"

	"github.com/google/uuid"

	appdb "github.com/eguilde/egueducation/internal/db"
)

func TestRegistraturaArchiveOutboxDeliversTenantScopedArchiveRecordIntegration(t *testing.T) {
	it := newArchiveIntegrationDatabase(t)
	ctx := context.Background()
	pool := openArchiveIntegrationPool(t, ctx, it.databaseConfig)
	defer pool.Close()
	if err := appdb.Migrate(ctx, pool); err != nil {
		t.Fatalf("migrate integration database: %v", err)
	}
	grantArchiveIntegrationAccess(t, ctx, pool)

	tenantCtx, release := archiveTenantContext(t, ctx, it.readerPool, "tenant-egueducation", "inst-001")
	defer release()
	session := appdb.NewSessionPool(it.readerPool)
	documentID := uuid.NewString()
	if _, err := session.Exec(tenantCtx, `
		insert into registratura_documents (id, tenant_code, institution_id, registry_number, subject, document_type, direction, status, correspondent, summary)
		values ($1::uuid, 'tenant-egueducation', 'inst-001', 'REG-OUTBOX-1', 'Document finalizat', 'cerere', 'intrare', 'FINALIZAT', 'Părinte', 'Document test')
	`, documentID); err != nil {
		t.Fatalf("seed finalized registratura document: %v", err)
	}
	if _, err := session.Exec(tenantCtx, `
		insert into registratura_document_attachments (tenant_code, institution_id, document_id, title, file_name, mime_type, storage_key, size_bytes, category, status, uploaded_by, checksum_sha256, scan_status, storage_state)
		values ('tenant-egueducation', 'inst-001', $1::uuid, 'Scanare', 'document.pdf', 'application/pdf', 'registratura/source.pdf', 12, 'scan', 'ready', 'test', repeat('a', 64), 'clean', 'ready')
	`, documentID); err != nil {
		t.Fatalf("seed clean registratura attachment: %v", err)
	}
	var eventID string
	if err := session.QueryRow(tenantCtx, `
		insert into registratura_archive_outbox (tenant_code, institution_id, document_id, event_type, payload)
		values ('tenant-egueducation', 'inst-001', $1::uuid, 'document_finalized', '{}'::jsonb)
		returning id::text
	`, documentID).Scan(&eventID); err != nil {
		t.Fatalf("seed archive outbox event: %v", err)
	}

	storage := newMemoryArchiveStore(map[string][]byte{"registratura/source.pdf": []byte("%PDF-1.7\ntest")})
	worker := &RegistraturaArchiveOutboxWorker{pool: session, storage: storage, maxAttempts: 3, workerID: "integration-worker"}
	event, err := worker.claimEvent(ctx)
	if err != nil || event == nil || event.ID != eventID {
		t.Fatalf("claim outbox event = %#v, %v", event, err)
	}
	if err := worker.processEvent(tenantCtx, event); err != nil {
		t.Fatalf("deliver outbox event: %v", err)
	}

	var status, sourceSystem, reference, jobStatus string
	if err := session.QueryRow(tenantCtx, `select status from registratura_archive_outbox where id::text = $1`, eventID).Scan(&status); err != nil || status != "delivered" {
		t.Fatalf("outbox status = %q, %v; want delivered", status, err)
	}
	if err := session.QueryRow(tenantCtx, `
		select d.source_system, d.external_reference, j.status
		from archive_documents d join archive_ingestion_jobs j on j.document_id = d.id
		where d.institution_id = 'inst-001' and d.idempotency_key = $1
	`, registraturaArchiveIdempotencyKey(documentID)).Scan(&sourceSystem, &reference, &jobStatus); err != nil {
		t.Fatalf("load created archive record/job: %v", err)
	}
	if sourceSystem != "registratura" || reference != documentID || jobStatus != "pending" {
		t.Fatalf("unexpected archive bridge result: source=%q reference=%q job=%q", sourceSystem, reference, jobStatus)
	}
	if len(storage.objects) != 2 { // source plus the deterministic eArhiva original
		t.Fatalf("storage objects = %d, want source and one archive copy", len(storage.objects))
	}

	// Reprocessing after an uncertain delivery acknowledgement must not create a
	// second archive record or another destination object key.
	if _, err := session.Exec(tenantCtx, `update registratura_archive_outbox set status = 'processing' where id::text = $1`, eventID); err != nil {
		t.Fatal(err)
	}
	if err := worker.processEvent(tenantCtx, event); err != nil {
		t.Fatalf("idempotent redelivery: %v", err)
	}
	var archiveCount int
	if err := session.QueryRow(tenantCtx, `select count(*) from archive_documents where institution_id = 'inst-001' and idempotency_key = $1`, registraturaArchiveIdempotencyKey(documentID)).Scan(&archiveCount); err != nil || archiveCount != 1 {
		t.Fatalf("idempotent archive count = %d, %v", archiveCount, err)
	}
}

func TestRegistraturaArchiveOutboxDeadLettersMissingCleanPDFIntegration(t *testing.T) {
	it := newArchiveIntegrationDatabase(t)
	ctx := context.Background()
	pool := openArchiveIntegrationPool(t, ctx, it.databaseConfig)
	defer pool.Close()
	if err := appdb.Migrate(ctx, pool); err != nil {
		t.Fatalf("migrate integration database: %v", err)
	}
	grantArchiveIntegrationAccess(t, ctx, pool)
	tenantCtx, release := archiveTenantContext(t, ctx, it.readerPool, "tenant-egueducation", "inst-001")
	defer release()
	session := appdb.NewSessionPool(it.readerPool)
	documentID := uuid.NewString()
	if _, err := session.Exec(tenantCtx, `
		insert into registratura_documents (id, tenant_code, institution_id, registry_number, subject, document_type, direction, status, correspondent, summary)
		values ($1::uuid, 'tenant-egueducation', 'inst-001', 'REG-OUTBOX-2', 'Fără scanare', 'cerere', 'intrare', 'FINALIZAT', 'Părinte', '')
	`, documentID); err != nil {
		t.Fatal(err)
	}
	var eventID string
	if err := session.QueryRow(tenantCtx, `
		insert into registratura_archive_outbox (tenant_code, institution_id, document_id, event_type, payload)
		values ('tenant-egueducation', 'inst-001', $1::uuid, 'document_finalized', '{}'::jsonb) returning id::text
	`, documentID).Scan(&eventID); err != nil {
		t.Fatal(err)
	}
	worker := &RegistraturaArchiveOutboxWorker{pool: session, storage: newMemoryArchiveStore(nil), maxAttempts: 3, workerID: "integration-worker"}
	event, err := worker.claimEvent(ctx)
	if err != nil || event == nil {
		t.Fatalf("claim event: %#v %v", event, err)
	}
	if err := worker.processEvent(tenantCtx, event); err == nil {
		t.Fatal("missing clean PDF must fail permanently")
	}
	var status string
	var deadLetteredAt *time.Time
	if err := session.QueryRow(tenantCtx, `select status, dead_lettered_at from registratura_archive_outbox where id::text = $1`, eventID).Scan(&status, &deadLetteredAt); err != nil {
		t.Fatal(err)
	}
	if status != "failed" || deadLetteredAt == nil {
		t.Fatalf("missing PDF must dead-letter, got status=%q dead_lettered_at=%v", status, deadLetteredAt)
	}
}

type memoryArchiveStore struct{ objects map[string][]byte }

func newMemoryArchiveStore(objects map[string][]byte) *memoryArchiveStore {
	if objects == nil {
		objects = map[string][]byte{}
	}
	return &memoryArchiveStore{objects: objects}
}
func (s *memoryArchiveStore) Enabled() bool  { return true }
func (s *memoryArchiveStore) Bucket() string { return "test-archive" }
func (s *memoryArchiveStore) OpenObject(_ context.Context, key string) (io.ReadCloser, error) {
	value, ok := s.objects[key]
	if !ok {
		return nil, fmt.Errorf("object %q not found", key)
	}
	return io.NopCloser(bytes.NewReader(value)), nil
}
func (s *memoryArchiveStore) PutObject(_ context.Context, key, _ string, body io.Reader, _ int64) error {
	value, err := io.ReadAll(body)
	if err != nil {
		return err
	}
	s.objects[key] = value
	return nil
}
func (s *memoryArchiveStore) CopyObject(_ context.Context, sourceKey, destinationKey, _ string) error {
	value, ok := s.objects[sourceKey]
	if !ok {
		return fmt.Errorf("object %q not found", sourceKey)
	}
	s.objects[destinationKey] = append([]byte(nil), value...)
	return nil
}
func (s *memoryArchiveStore) OriginalObjectKey(institutionID, documentID, fileName string) string {
	return strings.Join([]string{"archive", institutionID, documentID, "original", fileName}, "/")
}
func (s *memoryArchiveStore) ArtifactObjectKey(institutionID, documentID string, versionNo int) string {
	return fmt.Sprintf("archive/%s/%s/versions/%d/artifact.json", institutionID, documentID, versionNo)
}
