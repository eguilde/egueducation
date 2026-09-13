//go:build integration

package education

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"
	"time"

	awsconfig "github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/credentials"
	"github.com/aws/aws-sdk-go-v2/service/s3"
	"github.com/eguilde/egueducation/internal/config"
	appdb "github.com/eguilde/egueducation/internal/db"
	"github.com/eguilde/egueducation/internal/earchiva"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

// TestPortfolioOwnArchiveUploadIntegration exercises the teacher-owned HTTP
// boundary against a restricted PostgreSQL session and disposable MinIO. The
// scanner is deliberately fake: this proves command ownership and persistence,
// not an antivirus vendor's verdict.
func TestPortfolioOwnArchiveUploadIntegration(t *testing.T) {
	endpoint := strings.TrimSpace(os.Getenv("TEST_WORM_MINIO_ENDPOINT"))
	if endpoint == "" {
		t.Skip("requires explicit disposable TEST_WORM_MINIO_ENDPOINT")
	}
	accessKey, secretKey := os.Getenv("TEST_WORM_MINIO_ACCESS_KEY"), os.Getenv("TEST_WORM_MINIO_SECRET_KEY")
	if accessKey == "" || secretKey == "" {
		t.Fatal("explicit disposable MinIO credentials required")
	}
	it := newGovernanceIntegrationDatabase(t)
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
	defer cancel()
	admin := openGovernanceIntegrationPool(t, ctx, it.databaseConfig)
	defer admin.Close()
	if err := appdb.Migrate(ctx, admin); err != nil {
		t.Fatalf("migrate disposable portfolio-upload database: %v", err)
	}
	grantGovernanceIntegrationAccess(t, ctx, admin, it.roleName)
	fixture := seedGovernanceAuthorizationFixture(t, ctx, admin)
	grantPortfolioUploadPermissions(t, ctx, admin, fixture)

	storage, err := earchiva.NewArchiveStorage(ctx, config.Config{
		ArchiveStorageEndpoint: endpoint, ArchiveStorageRegion: "us-east-1",
		ArchiveStorageBucket:    "portfolio-upload-it-" + uuid.NewString(),
		ArchiveStorageAccessKey: accessKey, ArchiveStorageSecretKey: secretKey,
		ArchiveStorageUsePathStyle: true, ArchiveStorageCreateBucket: true,
		ArchiveStorageRequireObjectLock: true,
	})
	if err != nil {
		t.Fatalf("create disposable MinIO archive storage: %v", err)
	}

	sessions := appdb.NewSessionPool(it.readerPool)
	educationService := NewService(sessions)
	archive := earchiva.NewDocumentService(sessions, storage)
	archive.SetScanner(portfolioUploadCleanScanner{})
	handler := educationService.PortfolioOwnArchiveUpload(archive)
	pdf := portfolioUploadPDF("owner upload")
	key := "owner-upload-replay"

	ownerCtx, ownerRelease := governanceTenantContext(t, ctx, it.readerPool, fixture.tenantA, fixture.institutionA, fixture.memberSubject)
	created := performPortfolioUpload(t, handler, ownerCtx, fixture, fixture.memberSubject, fixture.portfolioID, key, pdf, nil)
	ownerRelease()
	if created.Code != http.StatusCreated {
		t.Fatalf("teacher own upload: status=%d body=%s", created.Code, created.Body.String())
	}
	var detail earchiva.ArchiveDocumentDetail
	if err := json.Unmarshal(created.Body.Bytes(), &detail); err != nil {
		t.Fatalf("decode created archive document: %v", err)
	}
	if detail.ID == "" || detail.Status != "queued" || detail.SourceKind != "upload" || detail.SourceSystem != "education_portfolio_own" {
		t.Fatalf("unexpected queued portfolio document: %#v", detail)
	}
	assertPortfolioUploadPersistence(t, ctx, admin, fixture, detail.ID, pdf)

	ownerCtx, ownerRelease = governanceTenantContext(t, ctx, it.readerPool, fixture.tenantA, fixture.institutionA, fixture.memberSubject)
	replay := performPortfolioUpload(t, handler, ownerCtx, fixture, fixture.memberSubject, fixture.portfolioID, key, pdf, nil)
	ownerRelease()
	if replay.Code != http.StatusOK {
		t.Fatalf("same upload replay: status=%d body=%s", replay.Code, replay.Body.String())
	}
	var replayDetail earchiva.ArchiveDocumentDetail
	if err := json.Unmarshal(replay.Body.Bytes(), &replayDetail); err != nil || replayDetail.ID != detail.ID {
		t.Fatalf("replay must return original document: id=%q err=%v body=%s", replayDetail.ID, err, replay.Body.String())
	}

	// Fail after the custody-held version is durably marked but before the
	// document transaction commits. The retry must adopt that exact version,
	// rather than issue another S3 PUT for the idempotency key.
	failFinalization := true
	retryScope := earchiva.ScopedUploadScope{
		InstitutionID: fixture.institutionA, ActorSubject: fixture.memberSubject,
		OwnerUserID: fixture.memberUserID, PortfolioID: fixture.portfolioID,
		Authorize: func(context.Context, pgx.Tx) error { return nil },
		PersistAccess: func(context.Context, pgx.Tx, string) error {
			if failFinalization {
				return fmt.Errorf("injected final persistence failure")
			}
			return nil
		},
	}
	retryHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		archive.UploadPortfolioOwnedDocument(w, r, retryScope)
	})
	retryKey := "stored-intent-retry"
	retryPDF := portfolioUploadPDF("stored intent retry")
	retryDigest := sha256.Sum256(retryPDF)
	ownerCtx, ownerRelease = governanceTenantContext(t, ctx, it.readerPool, fixture.tenantA, fixture.institutionA, fixture.memberSubject)
	failedFinalization := performPortfolioUpload(t, retryHandler, ownerCtx, fixture, fixture.memberSubject, fixture.portfolioID, retryKey, retryPDF, nil)
	if failedFinalization.Code != http.StatusInternalServerError {
		ownerRelease()
		t.Fatalf("injected finalization failure: status=%d body=%s", failedFinalization.Code, failedFinalization.Body.String())
	}
	var storedIntentID, storedDocumentID, storedBucket, storedKey, storedVersion, storedStatus string
	if err := sessions.QueryRow(ownerCtx, `select id::text,reserved_document_id::text,bucket_name,object_key,stored_version_id,status from portfolio_custody_upload_intents where institution_id=$1 and portfolio_id=$2::uuid and actor_subject=$3 and expected_sha256=$4`, fixture.institutionA, fixture.portfolioID, fixture.memberSubject, hex.EncodeToString(retryDigest[:])).Scan(&storedIntentID, &storedDocumentID, &storedBucket, &storedKey, &storedVersion, &storedStatus); err != nil {
		ownerRelease()
		t.Fatalf("load stored retry intent after response %s: %v", failedFinalization.Body.String(), err)
	}
	if storedStatus != "stored" {
		ownerRelease()
		t.Fatalf("finalization failure changed custody intent to %q", storedStatus)
	}
	ownerRelease()
	var uncommittedDocuments int
	if err := admin.QueryRow(ctx, `select count(*) from archive_documents where institution_id=$1 and id=$2::uuid`, fixture.institutionA, storedDocumentID).Scan(&uncommittedDocuments); err != nil || uncommittedDocuments != 0 {
		t.Fatalf("failed finalization committed archive document count=%d err=%v", uncommittedDocuments, err)
	}
	var uncommittedAudit int
	if err := admin.QueryRow(ctx, `select count(*) from app_audit_log where target_type='archive_document' and target_id=$1`, storedDocumentID).Scan(&uncommittedAudit); err != nil || uncommittedAudit != 0 {
		t.Fatalf("failed finalization committed audit count=%d err=%v", uncommittedAudit, err)
	}
	failFinalization = false
	ownerCtx, ownerRelease = governanceTenantContext(t, ctx, it.readerPool, fixture.tenantA, fixture.institutionA, fixture.memberSubject)
	storedReplay := performPortfolioUpload(t, retryHandler, ownerCtx, fixture, fixture.memberSubject, fixture.portfolioID, retryKey, retryPDF, nil)
	ownerRelease()
	if storedReplay.Code != http.StatusCreated {
		t.Fatalf("stored intent retry: status=%d body=%s", storedReplay.Code, storedReplay.Body.String())
	}
	var storedDetail earchiva.ArchiveDocumentDetail
	if err := json.Unmarshal(storedReplay.Body.Bytes(), &storedDetail); err != nil {
		t.Fatalf("decode stored retry: %v", err)
	}
	var adoptedVersion string
	if err := admin.QueryRow(ctx, `select source_object_version_id from archive_document_versions where document_id=$1::uuid`, storedDetail.ID).Scan(&adoptedVersion); err != nil || adoptedVersion != storedVersion {
		t.Fatalf("retry must adopt stored exact version got=%q want=%q err=%v", adoptedVersion, storedVersion, err)
	}
	if versions := countPortfolioUploadObjectVersions(t, endpoint, accessKey, secretKey, storedBucket, storedKey); versions != 1 {
		t.Fatalf("stored retry wrote %d object versions for intent %s", versions, storedIntentID)
	}

	ownerCtx, ownerRelease = governanceTenantContext(t, ctx, it.readerPool, fixture.tenantA, fixture.institutionA, fixture.memberSubject)
	changed := performPortfolioUpload(t, handler, ownerCtx, fixture, fixture.memberSubject, fixture.portfolioID, key, portfolioUploadPDF("changed bytes"), nil)
	ownerRelease()
	if changed.Code != http.StatusConflict {
		t.Fatalf("changed bytes with same idempotency key: status=%d body=%s", changed.Code, changed.Body.String())
	}

	// The same idempotency key with changed bytes is a client conflict (409),
	// while unavailable storage is an infrastructure condition (503).
	ownerCtx, ownerRelease = governanceTenantContext(t, ctx, it.readerPool, fixture.tenantA, fixture.institutionA, fixture.memberSubject)
	unavailable := performPortfolioUpload(t, educationService.PortfolioOwnArchiveUpload(earchiva.NewDocumentService(sessions, nil)), ownerCtx, fixture, fixture.memberSubject, fixture.portfolioID, "storage-unavailable", pdf, nil)
	ownerRelease()
	if unavailable.Code != http.StatusServiceUnavailable || !strings.Contains(unavailable.Body.String(), "archive_storage_unavailable") {
		t.Fatalf("unavailable storage must be 503: status=%d body=%s", unavailable.Code, unavailable.Body.String())
	}

	var foreignSubject string
	if err := admin.QueryRow(ctx, `select sub from app_users where id=$1::uuid`, fixture.foreignMemberUserID).Scan(&foreignSubject); err != nil {
		t.Fatalf("load foreign teacher subject: %v", err)
	}
	foreignCtx, foreignRelease := governanceTenantContext(t, ctx, it.readerPool, fixture.tenantA, fixture.institutionA, foreignSubject)
	denied := performPortfolioUpload(t, handler, foreignCtx, fixture, foreignSubject, fixture.portfolioID, "foreign-owner", pdf, nil)
	foreignRelease()
	if denied.Code != http.StatusForbidden {
		t.Fatalf("other teacher upload must be forbidden: status=%d body=%s", denied.Code, denied.Body.String())
	}

	ownerCtx, ownerRelease = governanceTenantContext(t, ctx, it.readerPool, fixture.tenantA, fixture.institutionA, fixture.memberSubject)
	malformed := performPortfolioUpload(t, handler, ownerCtx, fixture, fixture.memberSubject, fixture.portfolioID, "authority-field", pdf, map[string]string{"institution_id": "attacker-controlled"})
	ownerRelease()
	if malformed.Code != http.StatusBadRequest {
		t.Fatalf("browser authority field must be rejected: status=%d body=%s", malformed.Code, malformed.Body.String())
	}

	stateArchive := earchiva.NewDocumentService(sessions, storage)
	stateArchive.SetScanner(portfolioUploadStateChangingScanner{change: func() error {
		_, err := admin.Exec(ctx, `update education_portfolios set status='submitted' where id=$1::uuid`, fixture.foreignPortfolioID)
		return err
	}})
	stateHandler := educationService.PortfolioOwnArchiveUpload(stateArchive)
	foreignCtx, foreignRelease = governanceTenantContext(t, ctx, it.readerPool, fixture.tenantA, fixture.institutionA, foreignSubject)
	stateChanged := performPortfolioUpload(t, stateHandler, foreignCtx, fixture, foreignSubject, fixture.foreignPortfolioID, "state-change", pdf, nil)
	foreignRelease()
	if stateChanged.Code != http.StatusConflict {
		t.Fatalf("state change after staging must fail final callback: status=%d body=%s", stateChanged.Code, stateChanged.Body.String())
	}
	var stateDocuments int
	if err := admin.QueryRow(ctx, `select count(*) from archive_documents where institution_id=$1 and idempotency_key like 'portfolio-upload:%' and metadata->>'portfolio_id'=$2`, fixture.institutionA, fixture.foreignPortfolioID).Scan(&stateDocuments); err != nil {
		t.Fatalf("count state-change documents: %v", err)
	}
	if stateDocuments != 0 {
		t.Fatalf("state-changed upload committed %d archive document(s)", stateDocuments)
	}

	// Even a replay must not return the old document after live membership
	// revocation during scanning; the cached auth context is not authority.
	revokedArchive := earchiva.NewDocumentService(sessions, storage)
	revokedArchive.SetScanner(portfolioUploadStateChangingScanner{change: func() error {
		_, err := admin.Exec(ctx, `update app_memberships set active=false where user_id=$1::uuid and tenant_code=$2`, fixture.memberUserID, fixture.tenantA)
		return err
	}})
	ownerCtx, ownerRelease = governanceTenantContext(t, ctx, it.readerPool, fixture.tenantA, fixture.institutionA, fixture.memberSubject)
	revokedReplay := performPortfolioUpload(t, educationService.PortfolioOwnArchiveUpload(revokedArchive), ownerCtx, fixture, fixture.memberSubject, fixture.portfolioID, key, pdf, nil)
	ownerRelease()
	if revokedReplay.Code != http.StatusConflict {
		t.Fatalf("revoked replay returned status=%d body=%s", revokedReplay.Code, revokedReplay.Body.String())
	}
}

func countPortfolioUploadObjectVersions(t *testing.T, endpoint, accessKey, secretKey, bucket, key string) int {
	t.Helper()
	awsCfg, err := awsconfig.LoadDefaultConfig(context.Background(), awsconfig.WithRegion("us-east-1"), awsconfig.WithCredentialsProvider(credentials.NewStaticCredentialsProvider(accessKey, secretKey, "")))
	if err != nil {
		t.Fatalf("load MinIO config: %v", err)
	}
	client := s3.NewFromConfig(awsCfg, func(options *s3.Options) {
		options.BaseEndpoint = &endpoint
		options.UsePathStyle = true
	})
	output, err := client.ListObjectVersions(context.Background(), &s3.ListObjectVersionsInput{Bucket: &bucket, Prefix: &key})
	if err != nil {
		t.Fatalf("list stored object versions: %v", err)
	}
	count := 0
	for _, version := range output.Versions {
		if version.Key != nil && *version.Key == key {
			count++
		}
	}
	return count
}

func grantPortfolioUploadPermissions(t *testing.T, ctx context.Context, pool *pgxpool.Pool, fixture governanceAuthorizationFixture) {
	t.Helper()
	for _, userID := range []string{fixture.memberUserID, fixture.foreignMemberUserID} {
		if _, err := pool.Exec(ctx, `
			insert into app_user_permissions (user_id, permission_code, tenant_code)
			values ($1::uuid, 'education.portfolios.manage_own', $2)
			on conflict do nothing`, userID, fixture.tenantA); err != nil {
			t.Fatalf("grant teacher own-portfolio upload permission: %v", err)
		}
	}
}

func performPortfolioUpload(t *testing.T, handler http.Handler, requestContext context.Context, fixture governanceAuthorizationFixture, subject, portfolioID, key string, pdf []byte, fields map[string]string) *httptest.ResponseRecorder {
	t.Helper()
	var body bytes.Buffer
	writer := multipart.NewWriter(&body)
	if err := writer.WriteField("title", "Teacher portfolio evidence"); err != nil {
		t.Fatal(err)
	}
	for field, value := range fields {
		if err := writer.WriteField(field, value); err != nil {
			t.Fatal(err)
		}
	}
	part, err := writer.CreateFormFile("file", "portfolio.pdf")
	if err != nil {
		t.Fatal(err)
	}
	if _, err := part.Write(pdf); err != nil {
		t.Fatal(err)
	}
	if err := writer.Close(); err != nil {
		t.Fatal(err)
	}
	request := httptest.NewRequest(http.MethodPost, "http://education.test/education/portfolios/me/"+portfolioID+"/archive-documents", &body).WithContext(requestWithContext(requestContext, fixture.tenantA, fixture.institutionA, subject).Context())
	request.Header.Set("Content-Type", writer.FormDataContentType())
	request.Header.Set("Idempotency-Key", key)
	request = withChiParams(request, map[string]string{"recordID": portfolioID})
	response := httptest.NewRecorder()
	handler.ServeHTTP(response, request)
	return response
}

func assertPortfolioUploadPersistence(t *testing.T, ctx context.Context, pool *pgxpool.Pool, fixture governanceAuthorizationFixture, documentID string, content []byte) {
	t.Helper()
	want := sha256.Sum256(content)
	var status, sourceKind, sourceSystem, sourceBucket, sourceKey, sourceSHA string
	var versionSize int64
	var grants, jobs, ownAudit int
	err := pool.QueryRow(ctx, `
		select d.status, d.source_kind, d.source_system,
			v.source_bucket, v.source_object_key, v.source_sha256, v.source_size_bytes,
			(select count(*) from education_portfolio_archive_attachment_grants g
			 where g.institution_id=d.institution_id and g.archive_document_id=d.id and g.grantee_user_id=$2::uuid),
			(select count(*) from archive_ingestion_jobs j where j.document_id=d.id and j.status='pending'),
			(select count(*) from app_audit_log a where a.target_id=d.id::text and a.action='education.portfolios.own_archive_uploaded')
		from archive_documents d
		join archive_document_versions v on v.document_id=d.id and v.version_no=1
		where d.id=$1::uuid and d.institution_id=$3`, documentID, fixture.memberUserID, fixture.institutionA).Scan(
		&status, &sourceKind, &sourceSystem, &sourceBucket, &sourceKey, &sourceSHA, &versionSize, &grants, &jobs, &ownAudit,
	)
	if err != nil {
		t.Fatalf("load persisted portfolio upload graph: %v", err)
	}
	if status != "queued" || sourceKind != "upload" || sourceSystem != "education_portfolio_own" || sourceBucket == "" || sourceKey == "" || sourceSHA != hex.EncodeToString(want[:]) || versionSize != int64(len(content)) || grants != 1 || jobs != 1 || ownAudit != 1 {
		t.Fatalf("invalid queued archive source/grant/audit graph: status=%q kind=%q system=%q bucket=%q key=%q sha=%q size=%d grants=%d jobs=%d audit=%d", status, sourceKind, sourceSystem, sourceBucket, sourceKey, sourceSHA, versionSize, grants, jobs, ownAudit)
	}
}

type portfolioUploadCleanScanner struct{}

func (portfolioUploadCleanScanner) Scan(_ context.Context, source io.Reader) (bool, error) {
	_, err := io.Copy(io.Discard, source)
	return err == nil, err
}

type portfolioUploadStateChangingScanner struct {
	change func() error
}

func (s portfolioUploadStateChangingScanner) Scan(_ context.Context, source io.Reader) (bool, error) {
	if _, err := io.Copy(io.Discard, source); err != nil {
		return false, err
	}
	if err := s.change(); err != nil {
		return false, err
	}
	return true, nil
}

func portfolioUploadPDF(text string) []byte {
	stream := "BT /F1 12 Tf 20 100 Td (" + text + ") Tj ET\n"
	objects := []string{
		"<</Type/Catalog/Pages 2 0 R>>",
		"<</Type/Pages/Count 1/Kids[3 0 R]>>",
		"<</Type/Page/Parent 2 0 R/MediaBox[0 0 200 200]/Contents 4 0 R/Resources<</Font<</F1 5 0 R>>>>>>",
		fmt.Sprintf("<</Length %d>>stream\n%sendstream", len(stream), stream),
		"<</Type/Font/Subtype/Type1/BaseFont/Helvetica>>",
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
	fmt.Fprintf(&output, "trailer\n<</Size %d/Root 1 0 R>>\nstartxref\n%d\n%%%%EOF\n", len(objects)+1, xref)
	return output.Bytes()
}
