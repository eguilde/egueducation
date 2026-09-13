//go:build integration

package earchiva

import (
	"context"
	"strings"
	"testing"

	appdb "github.com/eguilde/egueducation/internal/db"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
)

func TestPortfolioCustodyIntentSecurityIntegration(t *testing.T) {
	it := newArchiveIntegrationDatabase(t)
	ctx := context.Background()
	admin := openArchiveIntegrationPool(t, ctx, it.databaseConfig)
	defer admin.Close()
	if err := appdb.Migrate(ctx, admin); err != nil {
		t.Fatalf("migrate clean custody-intent database: %v", err)
	}
	grantArchiveIntegrationAccess(t, ctx, admin)

	portfolioA := seedCustodyIntentPortfolio(t, ctx, admin, "inst-001")
	portfolioB := seedCustodyIntentPortfolio(t, ctx, admin, "inst-balotesti")
	sessions := appdb.NewSessionPool(it.readerPool)
	ctxA, releaseA := archiveTenantContext(t, ctx, it.readerPool, "tenant-egueducation", "inst-001")
	defer releaseA()
	ctxB, releaseB := archiveTenantContext(t, ctx, it.readerPool, "tenant-balotesti", "inst-balotesti")
	defer releaseB()

	t.Run("reserved to stored cannot rewrite reservation provenance", func(t *testing.T) {
		intent := seedCustodyUploadIntent(t, ctxA, sessions, "tenant-egueducation", "inst-001", portfolioA)
		_, err := sessions.Exec(ctxA, `
			update portfolio_custody_upload_intents
			set status='stored', object_key='archive/tampered.pdf',
				stored_version_id='storage-version-a', stored_etag='etag-a', stored_size_bytes=3
			where id=$1::uuid
		`, intent.intentID)
		assertCustodyGuardError(t, err, "portfolio custody intent provenance is immutable")

		var status, objectKey string
		if err := sessions.QueryRow(ctxA, `select status,object_key from portfolio_custody_upload_intents where id=$1::uuid`, intent.intentID).Scan(&status, &objectKey); err != nil {
			t.Fatalf("read rejected custody transition: %v", err)
		}
		if status != "reserved" || objectKey != intent.objectKey {
			t.Fatalf("rejected transition mutated intent: status=%q key=%q", status, objectKey)
		}
	})

	t.Run("verified storage identity cannot change on commit", func(t *testing.T) {
		intent := seedCustodyUploadIntent(t, ctxA, sessions, "tenant-egueducation", "inst-001", portfolioA)
		markCustodyIntentStored(t, ctxA, sessions, intent)
		_, err := sessions.Exec(ctxA, `
			update portfolio_custody_upload_intents
			set status='committed', stored_version_id='substituted-version'
			where id=$1::uuid
		`, intent.intentID)
		assertCustodyGuardError(t, err, "verified custody storage identity is immutable")
	})

	t.Run("second identical stored transition observes the winner", func(t *testing.T) {
		intent := seedCustodyUploadIntent(t, ctxA, sessions, "tenant-egueducation", "inst-001", portfolioA)
		markCustodyIntentStored(t, ctxA, sessions, intent)
		tag, err := sessions.Exec(ctxA, `
			update portfolio_custody_upload_intents
			set status='stored',stored_version_id=$1,stored_etag=$2,stored_size_bytes=3
			where id=$3::uuid and status='reserved'
		`, intent.storageVersion, intent.etag, intent.intentID)
		if err != nil || tag.RowsAffected() != 0 {
			t.Fatalf("second stored transition err=%v rows=%d", err, tag.RowsAffected())
		}
		var status, version, etag string
		if err := sessions.QueryRow(ctxA, `select status,stored_version_id,stored_etag from portfolio_custody_upload_intents where id=$1::uuid`, intent.intentID).Scan(&status, &version, &etag); err != nil || status != "stored" || version != intent.storageVersion || etag != intent.etag {
			t.Fatalf("reload stored winner status=%q version=%q etag=%q err=%v", status, version, etag, err)
		}
	})

	t.Run("stored transition rejects null verified size", func(t *testing.T) {
		intent := seedCustodyUploadIntent(t, ctxA, sessions, "tenant-egueducation", "inst-001", portfolioA)
		_, err := sessions.Exec(ctxA, `
			update portfolio_custody_upload_intents
			set status='stored',stored_version_id=$1,stored_etag=$2,stored_size_bytes=null
			where id=$3::uuid
		`, intent.storageVersion, intent.etag, intent.intentID)
		assertCustodyGuardError(t, err, "portfolio custody upload requires verified exact held version")
	})

	t.Run("archive version requires exact matching custody intent", func(t *testing.T) {
		intent := seedCustodyUploadIntent(t, ctxA, sessions, "tenant-egueducation", "inst-001", portfolioA)
		markCustodyIntentStored(t, ctxA, sessions, intent)
		seedCustodyArchiveDocument(t, ctxA, sessions, "inst-001", intent.documentID)

		_, err := sessions.Exec(ctxA, custodyVersionInsertSQL, intent.versionID, "inst-001", intent.documentID,
			intent.bucket, intent.objectKey, strings.Repeat("a", 64), int64(3), "different-storage-version", intent.etag, intent.intentID)
		assertCustodyGuardError(t, err, "matching verified custody intent")

		if _, err := sessions.Exec(ctxA, custodyVersionInsertSQL, intent.versionID, "inst-001", intent.documentID,
			intent.bucket, intent.objectKey, strings.Repeat("a", 64), int64(3), intent.storageVersion, intent.etag, intent.intentID); err != nil {
			t.Fatalf("adopt exact matching custody intent: %v", err)
		}
		_, err = sessions.Exec(ctxA, `update archive_document_versions set portfolio_custody_intent_id=null where id=$1::uuid`, intent.versionID)
		assertCustodyGuardError(t, err, "custody intent")
	})

	t.Run("custody held archive version cannot omit intent", func(t *testing.T) {
		documentID := uuid.NewString()
		versionID := uuid.NewString()
		seedCustodyArchiveDocument(t, ctxA, sessions, "inst-001", documentID)
		_, err := sessions.Exec(ctxA, custodyVersionInsertSQL, versionID, "inst-001", documentID,
			"archive-test", "archive/unlinked.pdf", strings.Repeat("b", 64), int64(3), "unlinked-version", "unlinked-etag", nil)
		assertCustodyGuardError(t, err, "custody intent")
	})

	t.Run("foreign tenant intent is invisible and immutable", func(t *testing.T) {
		foreign := seedCustodyUploadIntent(t, ctxB, sessions, "tenant-balotesti", "inst-balotesti", portfolioB)
		var visible int
		if err := sessions.QueryRow(ctxA, `select count(*) from portfolio_custody_upload_intents where id=$1::uuid`, foreign.intentID).Scan(&visible); err != nil {
			t.Fatalf("query foreign custody intent: %v", err)
		}
		if visible != 0 {
			t.Fatalf("tenant A can see %d tenant B custody intents", visible)
		}
		tag, err := sessions.Exec(ctxA, `update portfolio_custody_upload_intents set status='failed',failure_code='foreign-write' where id=$1::uuid`, foreign.intentID)
		if err != nil {
			t.Fatalf("tenant-isolated update should be an invisible no-op: %v", err)
		}
		if tag.RowsAffected() != 0 {
			t.Fatalf("tenant A changed %d tenant B custody intents", tag.RowsAffected())
		}
		var status string
		if err := sessions.QueryRow(ctxB, `select status from portfolio_custody_upload_intents where id=$1::uuid`, foreign.intentID).Scan(&status); err != nil {
			t.Fatalf("tenant B reads own custody intent: %v", err)
		}
		if status != "reserved" {
			t.Fatalf("foreign update changed tenant B intent to %q", status)
		}
	})
}

type custodyIntentFixture struct {
	intentID            string
	documentID          string
	versionID           string
	bucket              string
	objectKey           string
	storageVersion      string
	etag                string
	expectedFingerprint string
}

func seedCustodyIntentPortfolio(t *testing.T, ctx context.Context, admin *pgxpool.Pool, institutionID string) string {
	t.Helper()
	portfolioID := uuid.NewString()
	ownerUserID := uuid.NewString()
	ownerPersonnelID := uuid.NewString()
	identitySuffix := strings.ReplaceAll(ownerUserID, "-", "")
	if _, err := admin.Exec(ctx, `
		insert into app_users (id,sub,name,email,phone_number,locale,status)
		values ($1::uuid,$2,'Custody Security Teacher',$3,'','ro','active')
	`, ownerUserID, "custody-security-"+identitySuffix, "custody-security-"+identitySuffix+"@example.test"); err != nil {
		t.Fatalf("seed %s custody owner user: %v", institutionID, err)
	}
	tenantID, orgUnit := "tenant-egueducation", "unit-root"
	if institutionID == "inst-balotesti" {
		tenantID, orgUnit = "tenant-balotesti", "unit-balotesti-root"
	}
	if _, err := admin.Exec(ctx, `
		insert into app_memberships (
			user_id,tenant_code,position_code,org_unit_code,organization_name,is_primary,active,start_date
		) values ($1::uuid,$2,'profesor',$3,'Custody Security School',true,true,current_date)
	`, ownerUserID, tenantID, orgUnit); err != nil {
		t.Fatalf("seed %s custody owner membership: %v", institutionID, err)
	}
	if _, err := admin.Exec(ctx, `
		insert into education_personnel (
			id,app_user_id,employee_code,full_name,role_title,employment_type,status,
			evaluation_status,mobility_stage,school_year,institution_id
		) values ($1::uuid,$2::uuid,$3,'Custody Security Teacher','Profesor','titular','active',
			'draft','none','2026-2027',$4)
	`, ownerPersonnelID, ownerUserID, "CUSTODY-SEC-"+identitySuffix, institutionID); err != nil {
		t.Fatalf("seed %s custody owner personnel: %v", institutionID, err)
	}
	if _, err := admin.Exec(ctx, `
		insert into education_portfolios (
			id,portfolio_code,owner_name,owner_role,school_year,status,section_count,
			last_updated_on,transfer_status,institution_id,owner_user_id,owner_personnel_id
		) values ($1::uuid,$2,'Security Teacher','Profesor','2026-2027','draft',0,current_date,'none',$3,$4::uuid,$5::uuid)
	`, portfolioID, "CUSTODY-SEC-"+portfolioID, institutionID, ownerUserID, ownerPersonnelID); err != nil {
		t.Fatalf("seed %s custody portfolio: %v", institutionID, err)
	}
	return portfolioID
}

func seedCustodyUploadIntent(t *testing.T, ctx context.Context, sessions *appdb.SessionPool, tenantID, institutionID, portfolioID string) custodyIntentFixture {
	t.Helper()
	expectedFingerprint := portfolioUploadFingerprint(archiveUploadPayload{
		Title:          "Recovered custody evidence",
		FileName:       "evidence.pdf",
		ChecksumSHA256: strings.Repeat("a", 64),
		FileSize:       3,
		MimeType:       "application/pdf",
	})
	fixture := custodyIntentFixture{
		intentID:            uuid.NewString(),
		documentID:          uuid.NewString(),
		versionID:           uuid.NewString(),
		bucket:              "archive-test",
		objectKey:           "archive/" + institutionID + "/" + uuid.NewString() + "/original.pdf",
		storageVersion:      "storage-" + uuid.NewString(),
		etag:                "etag-" + uuid.NewString(),
		expectedFingerprint: expectedFingerprint,
	}
	if _, err := sessions.Exec(ctx, `
		insert into portfolio_custody_upload_intents (
			id,tenant_code,institution_id,portfolio_id,actor_subject,idempotency_key,
			expected_sha256,expected_size_bytes,expected_request_fingerprint,bucket_name,object_key,reserved_document_id,reserved_version_id
		) values ($1::uuid,$2,$3,$4::uuid,'archive-integration-test',$5,$6,3,$7,$8,$9,$10::uuid,$11::uuid)
	`, fixture.intentID, tenantID, institutionID, portfolioID, "custody-security-"+fixture.intentID,
		strings.Repeat("a", 64), expectedFingerprint, fixture.bucket, fixture.objectKey, fixture.documentID, fixture.versionID); err != nil {
		t.Fatalf("seed %s custody intent: %v", institutionID, err)
	}
	return fixture
}

func markCustodyIntentStored(t *testing.T, ctx context.Context, sessions *appdb.SessionPool, fixture custodyIntentFixture) {
	t.Helper()
	tag, err := sessions.Exec(ctx, `
		update portfolio_custody_upload_intents
		set status='stored',stored_version_id=$1,stored_etag=$2,stored_size_bytes=3
		where id=$3::uuid and status='reserved'
	`, fixture.storageVersion, fixture.etag, fixture.intentID)
	if err != nil {
		t.Fatalf("mark custody intent stored: %v", err)
	}
	if tag.RowsAffected() != 1 {
		t.Fatalf("mark custody intent stored affected %d rows", tag.RowsAffected())
	}
}

func seedCustodyArchiveDocument(t *testing.T, ctx context.Context, sessions *appdb.SessionPool, institutionID, documentID string) {
	t.Helper()
	if _, err := sessions.Exec(ctx, `
		insert into archive_documents (
			id,institution_id,title,original_file_name,mime_type,source_kind,source_system,
			status,original_bucket,original_object_key,artifact_bucket,artifact_object_key,idempotency_key
		) values ($1::uuid,$2,'Custody evidence','evidence.pdf','application/pdf','upload',
			'education_portfolio_own','queued','archive-test','source.pdf','archive-test','artifact.json',$3)
	`, documentID, institutionID, "custody-document-"+documentID); err != nil {
		t.Fatalf("seed custody archive document: %v", err)
	}
}

const custodyVersionInsertSQL = `
	insert into archive_document_versions (
		id,institution_id,document_id,version_no,mime_type,title,bucket_name,object_key,
		hash_sha256,size_bytes,status,source_bucket,source_object_key,source_sha256,source_size_bytes,
		source_object_version_id,source_object_etag,custody_hold_active,portfolio_custody_intent_id
	) values ($1::uuid,$2,$3::uuid,1,'application/pdf','Custody evidence',$4,$5,$6,$7,
		'active',$4,$5,$6,$7,$8,$9,true,$10::uuid)
`

func assertCustodyGuardError(t *testing.T, err error, fragment string) {
	t.Helper()
	if err == nil {
		t.Fatalf("custody guard unexpectedly accepted mutation; want error containing %q", fragment)
	}
	if !strings.Contains(err.Error(), fragment) {
		t.Fatalf("custody guard error = %v, want fragment %q", err, fragment)
	}
}
