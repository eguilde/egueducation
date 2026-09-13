//go:build integration

package admission

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"
	"time"

	"github.com/eguilde/egueducation/internal/auth"
	"github.com/eguilde/egueducation/internal/config"
	"github.com/eguilde/egueducation/internal/db"
	"github.com/eguilde/egueducation/internal/earchiva"
	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
)

// Authentication and malware scanning are test boundaries here; this exercises
// the actual upload handler, PDF validator, PostgreSQL and immutable MinIO.
// It does not claim OIDC, antivirus or cryptographic signature certification.
func TestAdmissionLegalArtifactUploadMinIOPostgres(t *testing.T) {
	dsn, endpoint := os.Getenv("TEST_DATABASE_URL"), os.Getenv("TEST_WORM_MINIO_ENDPOINT")
	if dsn == "" || endpoint == "" || os.Getenv("ARCHIVE_PDF_VALIDATOR_PATH") == "" {
		t.Skip("requires disposable PostgreSQL, MinIO and ARCHIVE_PDF_VALIDATOR_PATH")
	}
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
	defer cancel()
	pool := newAdmissionExpiryDatabase(t, ctx, dsn)
	if err := db.Migrate(ctx, pool); err != nil {
		t.Fatal(err)
	}
	sessions := db.NewSessionPool(pool)
	requestCtx, release, err := db.AcquireRequestConn(ctx, pool, db.SessionConfig{
		TenantID: "tenant-egueducation", InstitutionID: "inst-001",
		ActorSubject: "expiry-regression-actor", IsSuperAdmin: true,
	})
	if err != nil {
		t.Fatal(err)
	}
	defer release()
	applicationID, _, evaluationID := seedAdmissionExpiryGraph(t, requestCtx, sessions)
	seedExpiryRetentionAuthority(t, ctx, pool, sessions)
	authority := loadPreparationRetentionAuthority(t, requestCtx, sessions)
	preparationID := insertRetentionPreparation(t, requestCtx, sessions, applicationID, evaluationID, authority, "")
	var userID string
	if err := pool.QueryRow(ctx, `insert into app_users(sub,name,email,locale,status)
		values('expiry-regression-actor','Upload test actor','upload@example.test','ro','active') returning id::text`).Scan(&userID); err != nil {
		t.Fatal(err)
	}
	_, err = sessions.Exec(requestCtx, `insert into app_memberships
		(user_id,tenant_code,position_code,org_unit_code,organization_name,is_primary,active,start_date)
		select $1::uuid,'tenant-egueducation','director',m.org_unit_code,m.organization_name,false,true,current_date
		from app_memberships m join app_users u on u.id=m.user_id
		where u.sub='usr-002' and m.tenant_code='tenant-egueducation' and m.position_code='director'
		order by m.is_primary desc limit 1`, userID)
	if err != nil {
		t.Fatal(err)
	}
	_, err = sessions.Exec(requestCtx, `insert into app_user_permissions(user_id,tenant_code,permission_code)
		values($1::uuid,'tenant-egueducation','earchiva.manage'),
		($1::uuid,'tenant-egueducation','education.admissions.decide') on conflict do nothing`, userID)
	if err != nil {
		t.Fatal(err)
	}
	storage, err := earchiva.NewArchiveStorage(ctx, config.Config{
		ArchiveStorageEndpoint: endpoint, ArchiveStorageRegion: "us-east-1",
		ArchiveStorageBucket:       "admission-handler-" + uuid.NewString(),
		ArchiveStorageAccessKey:    os.Getenv("TEST_WORM_MINIO_ACCESS_KEY"),
		ArchiveStorageSecretKey:    os.Getenv("TEST_WORM_MINIO_SECRET_KEY"),
		ArchiveStorageUsePathStyle: true, ArchiveStorageCreateBucket: true,
		ArchiveStorageRequireObjectLock: true,
	})
	if err != nil {
		t.Fatal(err)
	}
	service := earchiva.NewDocumentService(sessions, storage)
	service.SetScanner(legalUploadCleanScanner{})
	router := chi.NewRouter()
	router.Post("/artifacts/{preparationID}/{artifactSlot}", service.UploadAdmissionLegalPreparationArtifact)
	router.Get("/artifacts/{preparationID}/{artifactSlot}", service.GetAdmissionLegalPreparationArtifact)
	router.Get("/documents/{documentID}", service.DownloadOriginal)
	requestCtx = auth.WithSessionContextForIntegration(requestCtx, auth.SessionContext{
		TenantCode: "tenant-egueducation", InstitutionID: "inst-001",
		User: auth.SessionUser{ID: userID, Sub: "expiry-regression-actor"},
	})
	path := "/artifacts/" + preparationID + "/primary"
	content := legalUploadPDF("admission fixture")
	requestKey := "actual-handler-replay"
	send := func(method, target string, pdf []byte) *httptest.ResponseRecorder {
		t.Helper()
		var body bytes.Buffer
		contentType := ""
		if pdf != nil {
			writer := multipart.NewWriter(&body)
			part, err := writer.CreateFormFile("file", "signed.pdf")
			if err != nil {
				t.Fatal(err)
			}
			if _, err := part.Write(pdf); err != nil {
				t.Fatal(err)
			}
			if err := writer.Close(); err != nil {
				t.Fatal(err)
			}
			contentType = writer.FormDataContentType()
		}
		r := httptest.NewRequest(method, target, &body).WithContext(requestCtx)
		r.Header.Set("Content-Type", contentType)
		r.Header.Set("Idempotency-Key", requestKey)
		response := httptest.NewRecorder()
		router.ServeHTTP(response, r)
		return response
	}
	created := send(http.MethodPost, path, content)
	if created.Code != http.StatusCreated {
		t.Fatalf("actual upload: %d %s", created.Code, created.Body.String())
	}
	var artifact earchiva.AdmissionLegalPreparationArtifactResponse
	if err := json.Unmarshal(created.Body.Bytes(), &artifact); err != nil {
		t.Fatal(err)
	}
	if artifact.Document.Status != "queued" || artifact.Version.SourceSizeBytes != int64(len(content)) {
		t.Fatalf("unexpected adopted artifact: %#v", artifact)
	}
	for _, value := range []string{artifact.RetentionUntil, artifact.Document.CreatedAt, artifact.Version.CreatedAt} {
		if _, err := time.Parse(time.RFC3339Nano, value); err != nil {
			t.Fatalf("invalid actual response timestamp %q: %v", value, err)
		}
	}
	replay := send(http.MethodPost, path, content)
	if replay.Code != http.StatusOK {
		t.Fatalf("replay: %d %s", replay.Code, replay.Body.String())
	}
	conflict := send(http.MethodPost, path, legalUploadPDF("different content"))
	if conflict.Code != http.StatusConflict {
		t.Fatalf("changed fingerprint: %d %s", conflict.Code, conflict.Body.String())
	}
	read := send(http.MethodGet, path, nil)
	if read.Code != http.StatusOK {
		t.Fatalf("read bound artifact: %d %s", read.Code, read.Body.String())
	}
	download := send(http.MethodGet, "/documents/"+artifact.Document.ID, nil)
	if download.Code != http.StatusOK || !bytes.Equal(download.Body.Bytes(), content) {
		t.Fatalf("original download: %d %s", download.Code, download.Body.String())
	}
	var intents, versions, jobs int
	err = sessions.QueryRow(requestCtx, `select
		(select count(*) from archive_ingestion_intents where preparation_id=$1::uuid and status='committed'),
		(select count(*) from archive_document_versions where document_id=$2::uuid),
		(select count(*) from archive_ingestion_jobs where document_id=$2::uuid)`,
		preparationID, artifact.Document.ID).Scan(&intents, &versions, &jobs)
	if err != nil || intents != 1 || versions != 1 || jobs != 1 {
		t.Fatalf("duplicate adoption or missing graph: %d/%d/%d %v", intents, versions, jobs, err)
	}

	// Inject a real PostgreSQL adoption failure after S3 accepts the bytes.
	// The durable reservation must survive, and the same HTTP command must
	// recover that version without deleting or creating a replacement object.
	recoveryPreparation := insertRetentionPreparation(
		t, requestCtx, sessions, applicationID, evaluationID, authority, "",
	)
	_, err = pool.Exec(ctx, `create function test_fail_archive_adoption() returns trigger
		language plpgsql as $$ begin raise exception 'injected adoption failure'; end $$;
		create trigger test_fail_archive_adoption before insert on archive_documents
		for each row execute function test_fail_archive_adoption()`)
	if err != nil {
		t.Fatal(err)
	}
	requestKey = "recover-after-adoption-failure"
	recoveryPath := "/artifacts/" + recoveryPreparation + "/primary"
	failed := send(http.MethodPost, recoveryPath, content)
	if failed.Code != http.StatusInternalServerError {
		t.Fatalf("injected adoption failure: %d %s", failed.Code, failed.Body.String())
	}
	var pendingID, pendingKey, pendingSHA, pendingStatus string
	var pendingSize int64
	var pendingRetention time.Time
	err = sessions.QueryRow(requestCtx, `select id::text,object_key,expected_sha256,expected_size_bytes,
		retention_until,status from archive_ingestion_intents where preparation_id=$1::uuid`,
		recoveryPreparation).Scan(&pendingID, &pendingKey, &pendingSHA, &pendingSize, &pendingRetention, &pendingStatus)
	if err != nil || pendingStatus != "reserved" {
		t.Fatalf("lost durable reservation after DB failure: %s %v", pendingStatus, err)
	}
	stored, err := storage.ReconcileImmutableObject(ctx, earchiva.ImmutableArchiveRecovery{
		Key: pendingKey, ExpectedSHA256: pendingSHA, ContentLength: pendingSize,
		RetentionUntil: pendingRetention, LegalHold: true,
		Metadata: map[string]string{
			"intent-id": pendingID, "tenant-code": "tenant-egueducation", "institution-id": "inst-001",
			"preparation-id": recoveryPreparation, "artifact-slot": "primary", "sha256": pendingSHA,
		},
	})
	if err != nil {
		t.Fatalf("accepted WORM object was lost after DB failure: %v", err)
	}
	if _, err = pool.Exec(ctx, `drop trigger test_fail_archive_adoption on archive_documents;
		drop function test_fail_archive_adoption()`); err != nil {
		t.Fatal(err)
	}
	recovered := send(http.MethodPost, recoveryPath, content)
	if recovered.Code != http.StatusOK {
		t.Fatalf("resume after DB failure: %d %s", recovered.Code, recovered.Body.String())
	}
	var adoptedVersion string
	err = sessions.QueryRow(requestCtx, `select stored_version_id from archive_ingestion_intents
		where id=$1::uuid and status='committed'`, pendingID).Scan(&adoptedVersion)
	if err != nil || adoptedVersion != stored.VersionID {
		t.Fatalf("recovery did not adopt original accepted version: %s %v", adoptedVersion, err)
	}
}

type legalUploadCleanScanner struct{}

func (legalUploadCleanScanner) Scan(_ context.Context, reader io.Reader) (bool, error) {
	_, err := io.Copy(io.Discard, reader)
	return err == nil, err
}

func legalUploadPDF(text string) []byte {
	stream := "BT /F1 12 Tf 20 50 Td (" + text + ") Tj ET"
	objects := []string{
		"<< /Type /Catalog /Pages 2 0 R >>",
		"<< /Type /Pages /Kids [3 0 R] /Count 1 >>",
		"<< /Type /Page /Parent 2 0 R /MediaBox [0 0 200 100] /Resources << /Font << /F1 4 0 R >> >> /Contents 5 0 R >>",
		"<< /Type /Font /Subtype /Type1 /BaseFont /Helvetica >>",
		fmt.Sprintf("<< /Length %d >>\nstream\n%s\nendstream", len(stream), stream),
	}
	var output bytes.Buffer
	output.WriteString("%PDF-1.4\n")
	offsets := make([]int, len(objects))
	for index, object := range objects {
		offsets[index] = output.Len()
		fmt.Fprintf(&output, "%d 0 obj\n%s\nendobj\n", index+1, object)
	}
	xref := output.Len()
	fmt.Fprintf(&output, "xref\n0 %d\n0000000000 65535 f \n", len(objects)+1)
	for _, offset := range offsets {
		fmt.Fprintf(&output, "%010d 00000 n \n", offset)
	}
	fmt.Fprintf(&output, "trailer\n<< /Size %d /Root 1 0 R >>\nstartxref\n%d\n%%%%EOF\n", len(objects)+1, xref)
	return output.Bytes()
}
