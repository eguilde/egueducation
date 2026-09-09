//go:build integration

package education

import (
	"context"
	"strings"
	"testing"
	"time"

	appdb "github.com/eguilde/egueducation/internal/db"
)

// TestPortfolioValorificationPackagesIntegration exercises the DB boundary as
// the application NOBYPASSRLS role: sources are institution-bound, archive
// version metadata is server-derived, and lifecycle evidence cannot be forged
// or removed once a package exists.
func TestPortfolioValorificationPackagesIntegration(t *testing.T) {
	it := newGovernanceIntegrationDatabase(t)
	ctx, cancel := context.WithTimeout(context.Background(), 90*time.Second)
	defer cancel()
	admin := openGovernanceIntegrationPool(t, ctx, it.databaseConfig)
	defer admin.Close()
	if err := appdb.Migrate(ctx, admin); err != nil {
		t.Fatalf("migrate valorification package database: %v", err)
	}
	grantGovernanceIntegrationAccess(t, ctx, admin, it.roleName)
	fixture := seedGovernanceAuthorizationFixture(t, ctx, admin)

	var personnelID, foreignPersonnelID, evaluationID, foreignEvaluationID, wrongYearEvaluationID, mobilityID, meritID, archiveDocumentID, archiveVersionID, purgedArchiveDocumentID, purgedArchiveVersionID string
	if err := admin.QueryRow(ctx, `
		insert into education_personnel (employee_code,full_name,role_title,employment_type,status,evaluation_status,mobility_stage,school_year,institution_id)
		values ('PER-VALPKG-001','Governance Integration Member','Profesor','titular','active','finalized','none','2026-2027',$1)
		returning id::text`, fixture.institutionA).Scan(&personnelID); err != nil {
		t.Fatalf("seed authoritative portfolio personnel: %v", err)
	}
	if _, err := admin.Exec(ctx, `update education_portfolios set owner_personnel_id=$1::uuid where id=$2::uuid`, personnelID, fixture.portfolioID); err != nil {
		t.Fatalf("bind portfolio to authoritative personnel: %v", err)
	}
	if err := admin.QueryRow(ctx, `
		insert into education_evaluations (evaluation_code,employee_code,personnel_id,full_name,role_title,school_year,status,institution_id)
		values ('VALPKG-EVAL-001','PER-VALPKG-001',$2::uuid,'Governance Integration Member','Profesor','2026-2027','draft',$1)
		returning id::text`, fixture.institutionA, personnelID).Scan(&evaluationID); err != nil {
		t.Fatalf("seed authoritative evaluation source: %v", err)
	}
	if err := admin.QueryRow(ctx, `
		insert into education_personnel (employee_code,full_name,role_title,employment_type,status,evaluation_status,mobility_stage,school_year,institution_id)
		values ('PER-VALPKG-OTHER','Other Teacher','Profesor','titular','active','finalized','none','2026-2027',$1)
		returning id::text`, fixture.institutionA).Scan(&foreignPersonnelID); err != nil {
		t.Fatalf("seed foreign source personnel: %v", err)
	}
	if err := admin.QueryRow(ctx, `
		insert into education_evaluations (evaluation_code,employee_code,personnel_id,full_name,role_title,school_year,status,institution_id)
		values ('VALPKG-EVAL-OTHER','PER-VALPKG-OTHER',$2::uuid,'Other Teacher','Profesor','2026-2027','draft',$1)
		returning id::text`, fixture.institutionA, foreignPersonnelID).Scan(&foreignEvaluationID); err != nil {
		t.Fatalf("seed foreign evaluation source: %v", err)
	}
	if err := admin.QueryRow(ctx, `
		insert into education_evaluations (evaluation_code,employee_code,personnel_id,full_name,role_title,school_year,status,institution_id)
		values ('VALPKG-EVAL-YEAR','PER-VALPKG-001',$2::uuid,'Governance Integration Member','Profesor','2025-2026','approved',$1)
		returning id::text`, fixture.institutionA, personnelID).Scan(&wrongYearEvaluationID); err != nil {
		t.Fatalf("seed wrong-year evaluation source: %v", err)
	}
	if err := admin.QueryRow(ctx, `
		insert into education_mobility_cases (case_code,employee_code,personnel_id,full_name,school_year,request_type,stage,status,submitted_on,institution_id)
		values ('VALPKG-MOB-001','PER-VALPKG-001',$2::uuid,'Governance Integration Member','2026-2027','transfer','approved','approved',current_date,$1)
		returning id::text`, fixture.institutionA, personnelID).Scan(&mobilityID); err != nil {
		t.Fatalf("seed canonical mobility source: %v", err)
	}
	if err := admin.QueryRow(ctx, `
		insert into education_merit_grants (grant_code,personnel_id,full_name,role_title,school_year,category,status,decision_date,institution_id)
		values ('VALPKG-MERIT-001',$2::uuid,'Governance Integration Member','Profesor','2026-2027','predare','approved',current_date,$1)
		returning id::text`, fixture.institutionA, personnelID).Scan(&meritID); err != nil {
		t.Fatalf("seed canonical merit source: %v", err)
	}
	if err := admin.QueryRow(ctx, `
		insert into archive_documents (institution_id,title,original_file_name,mime_type,source_kind,status,current_version_no)
		values ($1,'Valorification evidence','valorification.pdf','application/pdf','upload','ready',1)
		returning id::text`, fixture.institutionA).Scan(&archiveDocumentID); err != nil {
		t.Fatalf("seed archive document: %v", err)
	}
	if err := admin.QueryRow(ctx, `
		insert into archive_document_versions (document_id,institution_id,version_no,mime_type,title,bucket_name,object_key,hash_sha256,status,source_bucket,source_object_key,source_sha256)
		values ($1::uuid,$2,1,'application/pdf','Valorification evidence','earhive','evidence.pdf',repeat('a',64),'active','earhive','evidence.pdf',repeat('a',64))
		returning id::text`, archiveDocumentID, fixture.institutionA).Scan(&archiveVersionID); err != nil {
		t.Fatalf("seed immutable archive version: %v", err)
	}
	if err := admin.QueryRow(ctx, `
		insert into archive_documents (institution_id,title,original_file_name,mime_type,source_kind,status,current_version_no)
		values ($1,'Purged evidence','purged.pdf','application/pdf','upload','ready',1)
		returning id::text`, fixture.institutionA).Scan(&purgedArchiveDocumentID); err != nil {
		t.Fatalf("seed purged archive document: %v", err)
	}
	if err := admin.QueryRow(ctx, `
		insert into archive_document_versions (document_id,institution_id,version_no,mime_type,title,bucket_name,object_key,hash_sha256,status,source_bucket,source_object_key,source_sha256)
		values ($1::uuid,$2,1,'application/pdf','Purged evidence','earhive','purged.pdf',repeat('c',64),'purged','earhive','purged.pdf',repeat('c',64))
		returning id::text`, purgedArchiveDocumentID, fixture.institutionA).Scan(&purgedArchiveVersionID); err != nil {
		t.Fatalf("seed purged archive version: %v", err)
	}

	actor := fixture.memberSubject
	sourceCtx, releaseSource := governanceTenantContext(t, ctx, it.readerPool, fixture.tenantA, fixture.institutionA, actor)
	sourceReleased := false
	defer func() {
		if !sourceReleased {
			releaseSource()
		}
	}()
	pool := appdb.NewSessionPool(it.readerPool)
	var packageID string
	if err := pool.QueryRow(sourceCtx, `
		insert into education_portfolio_valorification_packages (tenant_code,institution_id,portfolio_id,scope,source_evaluation_id,created_by_subject)
		values ($1,$2,$3::uuid,'evaluare_profesionala',$4::uuid,'forged') returning id::text`, fixture.tenantA, fixture.institutionA, fixture.portfolioID, evaluationID).Scan(&packageID); err != nil {
		t.Fatalf("create scope-bound package: %v", err)
	}
	if _, err := pool.Exec(sourceCtx, `
		insert into education_portfolio_valorification_packages (tenant_code,institution_id,portfolio_id,scope,source_evaluation_id)
		values ($1,$2,$3::uuid,'evaluare_profesionala',$4::uuid)
	`, fixture.tenantA, fixture.institutionA, fixture.portfolioID, foreignEvaluationID); err == nil || !strings.Contains(err.Error(), "must match portfolio personnel") {
		t.Fatalf("same-tenant foreign personnel source must be rejected; err=%v", err)
	}
	if _, err := pool.Exec(sourceCtx, `
		insert into education_portfolio_valorification_packages (tenant_code,institution_id,portfolio_id,scope,source_evaluation_id)
		values ($1,$2,$3::uuid,'evaluare_profesionala',$4::uuid)
	`, fixture.tenantA, fixture.institutionA, fixture.portfolioID, wrongYearEvaluationID); err == nil || !strings.Contains(err.Error(), "school year") {
		t.Fatalf("same-person wrong-school-year source must be rejected; err=%v", err)
	}
	for _, source := range []struct {
		scope  string
		column string
		id     string
	}{{"mobilitate", "source_mobility_case_id", mobilityID}, {"gradatie_merit", "source_merit_grant_id", meritID}} {
		statement := `insert into education_portfolio_valorification_packages (tenant_code,institution_id,portfolio_id,scope,` + source.column + `) values ($1,$2,$3::uuid,$4,$5::uuid)`
		if _, err := pool.Exec(sourceCtx, statement, fixture.tenantA, fixture.institutionA, fixture.portfolioID, source.scope, source.id); err != nil {
			t.Fatalf("canonical %s source must create a bound package: %v", source.scope, err)
		}
	}
	var createdBy string
	if err := pool.QueryRow(sourceCtx, `select created_by_subject from education_portfolio_valorification_packages where id=$1::uuid`, packageID).Scan(&createdBy); err != nil || createdBy != actor {
		t.Fatalf("created provenance=%q err=%v, want authenticated actor %q", createdBy, err, actor)
	}

	var packageDocumentID string
	if err := pool.QueryRow(sourceCtx, `
		insert into education_portfolio_valorification_package_documents (package_id,institution_id,archive_document_id,archive_version_id,archive_version_no,archive_source_bucket,archive_source_object_key,archive_sha256,created_by_subject)
		values ($1::uuid,$2,$3::uuid,$4::uuid,99,'forged','forged',repeat('b',64),'forged') returning id::text`, packageID, fixture.institutionA, archiveDocumentID, archiveVersionID).Scan(&packageDocumentID); err != nil {
		t.Fatalf("attach archive-version evidence: %v", err)
	}
	var versionNo int
	var sourceBucket, sourceObject, sourceHash, documentActor string
	if err := pool.QueryRow(sourceCtx, `select archive_version_no,archive_source_bucket,archive_source_object_key,archive_sha256,created_by_subject from education_portfolio_valorification_package_documents where id=$1::uuid`, packageDocumentID).Scan(&versionNo, &sourceBucket, &sourceObject, &sourceHash, &documentActor); err != nil {
		t.Fatalf("read server-derived package evidence: %v", err)
	}
	if versionNo != 1 || sourceBucket != "earhive" || sourceObject != "evidence.pdf" || sourceHash != strings.Repeat("a", 64) || documentActor != actor {
		t.Fatalf("archive evidence was not server-derived: version=%d bucket=%q key=%q hash=%q actor=%q", versionNo, sourceBucket, sourceObject, sourceHash, documentActor)
	}
	if _, err := pool.Exec(sourceCtx, `update education_portfolio_valorification_package_documents set archive_sha256=repeat('c',64) where id=$1::uuid`, packageDocumentID); err == nil || !strings.Contains(err.Error(), "immutable") {
		t.Fatalf("package evidence update must fail as immutable; err=%v", err)
	}
	if _, err := pool.Exec(sourceCtx, `
		insert into education_portfolio_valorification_package_documents (package_id,institution_id,archive_document_id,archive_version_id,archive_version_no,archive_source_bucket,archive_source_object_key,archive_sha256)
		values ($1::uuid,$2,$3::uuid,$4::uuid,1,'','',repeat('0',64))
	`, packageID, fixture.institutionA, purgedArchiveDocumentID, purgedArchiveVersionID); err == nil || !strings.Contains(err.Error(), "archive version") {
		t.Fatalf("purged archive version must be rejected as package evidence; err=%v", err)
	}

	if _, err := pool.Exec(sourceCtx, `update education_portfolio_valorification_packages set status='submitted',submitted_by_subject='forged' where id=$1::uuid`, packageID); err != nil {
		t.Fatalf("submit package: %v", err)
	}
	var submittedBy string
	if err := pool.QueryRow(sourceCtx, `select submitted_by_subject from education_portfolio_valorification_packages where id=$1::uuid`, packageID).Scan(&submittedBy); err != nil || submittedBy != actor {
		t.Fatalf("submitted provenance=%q err=%v, want %q", submittedBy, err, actor)
	}
	if _, err := pool.Exec(sourceCtx, `update education_portfolio_valorification_packages set status='validated' where id=$1::uuid`, packageID); err != nil {
		t.Fatalf("validate package: %v", err)
	}
	if _, err := pool.Exec(sourceCtx, `update education_portfolio_valorification_packages set status='completed' where id=$1::uuid`, packageID); err != nil {
		t.Fatalf("complete package: %v", err)
	}
	var versionCount int
	if err := pool.QueryRow(sourceCtx, `select count(*) from app_entity_versions where entity_table='education_portfolio_valorification_packages' and entity_id=$1::uuid`, packageID).Scan(&versionCount); err != nil || versionCount < 3 {
		t.Fatalf("package lifecycle needs immutable entity-version history: versions=%d err=%v", versionCount, err)
	}
	if _, err := pool.Exec(sourceCtx, `delete from education_portfolio_valorification_packages where id=$1::uuid`, packageID); err == nil || !strings.Contains(err.Error(), "cannot be hard-deleted") {
		t.Fatalf("package deletion must be rejected; err=%v", err)
	}

	// The restricted integration pool intentionally has a single connection.
	// Release the source-tenant session before proving cross-tenant isolation so
	// the destination context reuses a connection whose session was cleaned up.
	releaseSource()
	sourceReleased = true
	otherCtx, releaseOther := governanceTenantContext(t, ctx, it.readerPool, fixture.tenantB, fixture.institutionB, "other-tenant-actor")
	defer releaseOther()
	var visible int
	if err := pool.QueryRow(otherCtx, `select count(*) from education_portfolio_valorification_packages where id=$1::uuid`, packageID).Scan(&visible); err != nil || visible != 0 {
		t.Fatalf("other tenant package visibility=%d err=%v, want 0", visible, err)
	}
}
