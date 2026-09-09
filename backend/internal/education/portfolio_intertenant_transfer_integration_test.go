//go:build integration

package education

import (
	"context"
	"strings"
	"testing"
	"time"

	appdb "github.com/eguilde/egueducation/internal/db"
)

// TestIntertenantPortfolioTransferRoutingAndEvidenceContractIntegration proves
// the database boundary with the same NOBYPASSRLS role used by the HTTP
// integration suite.  Handler tests may prove route wiring separately; this
// test deliberately prevents a handler or future job from bypassing tenant
// routing, source/destination responsibilities, or immutable sent evidence.
func TestIntertenantPortfolioTransferRoutingAndEvidenceContractIntegration(t *testing.T) {
	it := newGovernanceIntegrationDatabase(t)
	ctx, cancel := context.WithTimeout(context.Background(), 90*time.Second)
	defer cancel()
	adminPool := openGovernanceIntegrationPool(t, ctx, it.databaseConfig)
	defer adminPool.Close()
	if err := appdb.Migrate(ctx, adminPool); err != nil {
		t.Fatalf("migrate disposable intertenant-transfer database: %v", err)
	}
	grantGovernanceIntegrationAccess(t, ctx, adminPool, it.roleName)
	fixture := seedGovernanceAuthorizationFixture(t, ctx, adminPool)
	if _, err := adminPool.Exec(ctx, `
		insert into app_user_permissions(user_id, permission_code, tenant_code)
		values ($1::uuid, 'education.portfolios.transfer', $2)
		on conflict do nothing
	`, fixture.memberUserID, fixture.tenantA); err != nil {
		t.Fatalf("grant source actor portfolio transfer permission: %v", err)
	}

	sourceCtx, releaseSource := governanceTenantContext(t, ctx, it.readerPool, fixture.tenantA, fixture.institutionA, fixture.memberSubject)
	defer releaseSource()

	var manifestID, transferID string
	if err := appdb.NewSessionPool(it.readerPool).QueryRow(sourceCtx, `
		insert into education_portfolio_export_manifests (
			portfolio_id, institution_id, tenant_code, manifest_version,
			manifest_sha256, manifest, generated_by_subject
		) values ($1::uuid, $2, $3, 'egueducation.portfolio-export-manifest/v1',
			repeat('a', 64), '{"schema":"portfolio-transfer-test"}'::jsonb, $4)
		returning id::text
	`, fixture.portfolioID, fixture.institutionA, fixture.tenantA, fixture.memberSubject).Scan(&manifestID); err != nil {
		t.Fatalf("create immutable source export manifest: %v", err)
	}

	pool := appdb.NewSessionPool(it.readerPool)
	var destinationDirectoryCount int
	if err := pool.QueryRow(sourceCtx, `
		select count(*)
		from public.education_portfolio_transfer_destinations()
		where tenant_code=$1 and institution_id=$2
	`, fixture.tenantB, fixture.institutionB).Scan(&destinationDirectoryCount); err != nil {
		t.Fatalf("load permission-checked transfer destination directory: %v", err)
	}
	if destinationDirectoryCount != 1 {
		t.Fatalf("transfer destination directory contains %d matching destination rows, want 1", destinationDirectoryCount)
	}
	if err := pool.QueryRow(sourceCtx, `
		insert into education_portfolio_transfers (
			portfolio_id, transfer_code, transfer_type, source_institution,
			destination_institution, status, handover_on, institution_id, notes,
			routing_version, source_tenant_code, source_institution_id,
			destination_tenant_code, destination_institution_id
		) values (
			$1::uuid, 'IT-XFER-001', 'mutare', 'Source institution',
			'Destination institution', 'pregatit', current_date, $2, 'integration',
			2, $3, $2, $4, $5
		) returning id::text
	`, fixture.portfolioID, fixture.institutionA, fixture.tenantA, fixture.tenantB, fixture.institutionB).Scan(&transferID); err != nil {
		t.Fatalf("source tenant prepares routed transfer: %v", err)
	}

	// A source actor can seal and send, but must never self-confirm receipt.
	if _, err := pool.Exec(sourceCtx, `
		update education_portfolio_transfers
		set status='trimis', export_manifest_id=$1::uuid, sent_at='2001-01-01', sent_by_subject='forged-sender'
		where id=$2::uuid
	`, manifestID, transferID); err != nil {
		t.Fatalf("source tenant sends manifest-bound transfer: %v", err)
	}
	var sentBy string
	if err := pool.QueryRow(sourceCtx, `
		select sent_by_subject from education_portfolio_transfers where id=$1::uuid
	`, transferID).Scan(&sentBy); err != nil {
		t.Fatalf("load sender provenance: %v", err)
	}
	if sentBy != fixture.memberSubject {
		t.Fatalf("sender provenance=%q, want authenticated actor %q", sentBy, fixture.memberSubject)
	}
	if _, err := pool.Exec(sourceCtx, `
		update education_portfolio_transfers
		set status='receptionat', received_at=now(), received_on=current_date,
			received_by_subject='forged-source-receipt'
		where id=$1::uuid
	`, transferID); err == nil || !strings.Contains(err.Error(), "only the destination institution can receive") {
		t.Fatalf("source must not confirm receipt; err=%v", err)
	}
	releaseSource()

	// The destination sees sent work in its inbox-equivalent RLS projection and
	// is the only normal application tenant allowed to confirm receipt.
	destinationSubject := "integration-destination-receiver"
	destinationCtx, releaseDestination := governanceTenantContext(t, ctx, it.readerPool, fixture.tenantB, fixture.institutionB, destinationSubject)
	defer releaseDestination()
	var inboxCount int
	if err := pool.QueryRow(destinationCtx, `
		select count(*) from education_portfolio_transfers
		where id=$1::uuid and status='trimis'
	`, transferID).Scan(&inboxCount); err != nil {
		t.Fatalf("load destination transfer inbox: %v", err)
	}
	if inboxCount != 1 {
		t.Fatalf("destination inbox sees %d sent transfers, want 1", inboxCount)
	}
	if _, err := pool.Exec(destinationCtx, `
		update education_portfolio_transfers
		set status='receptionat', notes='destination tampering attempt',
			received_at='2001-01-01', received_on='2001-01-01',
			received_by='forged receiver', received_by_subject='forged-receiver'
		where id=$1::uuid
	`, transferID); err == nil || !strings.Contains(err.Error(), "sent portfolio transfer package is immutable") {
		t.Fatalf("destination must not alter the sent package while receiving; err=%v", err)
	}
	if _, err := pool.Exec(destinationCtx, `
		update education_portfolio_transfers
		set status='receptionat', received_at='2001-01-01', received_on='2001-01-01',
			received_by='forged receiver', received_by_subject='forged-receiver'
		where id=$1::uuid
	`, transferID); err != nil {
		t.Fatalf("destination confirms receipt: %v", err)
	}
	var receivedBy string
	if err := pool.QueryRow(destinationCtx, `
		select received_by_subject from education_portfolio_transfers where id=$1::uuid
	`, transferID).Scan(&receivedBy); err != nil {
		t.Fatalf("load receipt provenance: %v", err)
	}
	if receivedBy != destinationSubject {
		t.Fatalf("receipt provenance=%q, want destination actor %q", receivedBy, destinationSubject)
	}
	releaseDestination()
	sourceCtx, releaseClosingSource := governanceTenantContext(t, ctx, it.readerPool, fixture.tenantA, fixture.institutionA, fixture.memberSubject)
	defer releaseClosingSource()
	if _, err := pool.Exec(sourceCtx, `
		update education_portfolio_transfers
		set status='inchis', closed_at='2001-01-01', closed_by_subject='forged-closer'
		where id=$1::uuid
	`, transferID); err != nil {
		t.Fatalf("source closes destination-confirmed transfer: %v", err)
	}
	var closedBy string
	if err := pool.QueryRow(sourceCtx, `
		select closed_by_subject from education_portfolio_transfers where id=$1::uuid
	`, transferID).Scan(&closedBy); err != nil {
		t.Fatalf("load closing provenance: %v", err)
	}
	if closedBy != fixture.memberSubject {
		t.Fatalf("closing provenance=%q, want source actor %q", closedBy, fixture.memberSubject)
	}
	releaseClosingSource()

	// A random request scope has neither source nor destination participant
	// visibility. This proves the inbox cannot leak through an identifier.
	unrelatedCtx, releaseUnrelated := governanceTenantContext(t, ctx, it.readerPool, "tenant-unrelated", "inst-unrelated", "integration-unrelated")
	defer releaseUnrelated()
	var directDirectoryCount int
	if err := pool.QueryRow(unrelatedCtx, `select count(*) from public.education_portfolio_transfer_destination_directory`).Scan(&directDirectoryCount); err != nil {
		t.Fatalf("query direct destination projection under RLS: %v", err)
	}
	if directDirectoryCount != 0 {
		t.Fatalf("actor without transfer permission sees %d direct destination rows, want 0", directDirectoryCount)
	}
	if _, err := pool.Exec(unrelatedCtx, `select * from public.education_portfolio_transfer_destinations()`); err == nil || !strings.Contains(err.Error(), "requires education.portfolios.transfer") {
		t.Fatalf("actor without transfer permission must not enumerate destination tenants; err=%v", err)
	}
	var unrelatedCount int
	if err := pool.QueryRow(unrelatedCtx, `select count(*) from education_portfolio_transfers where id=$1::uuid`, transferID).Scan(&unrelatedCount); err != nil {
		t.Fatalf("query unrelated tenant transfer visibility: %v", err)
	}
	if unrelatedCount != 0 {
		t.Fatalf("unrelated tenant sees %d transfer rows, want 0", unrelatedCount)
	}
	releaseUnrelated()

	// Once sent, route, package and row lifetime are evidentiary. Normal source
	// access still sees the row, so these failures are trigger-enforced rather
	// than an accidental lack of table privilege or RLS visibility.
	sourceCtx, releaseFinalSource := governanceTenantContext(t, ctx, it.readerPool, fixture.tenantA, fixture.institutionA, fixture.memberSubject)
	defer releaseFinalSource()
	if _, err := pool.Exec(sourceCtx, `
		update education_portfolio_transfers set destination_tenant_code='tampered'
		where id=$1::uuid
	`, transferID); err == nil || !strings.Contains(err.Error(), "route is immutable") {
		t.Fatalf("sent transfer route must be immutable; err=%v", err)
	}
	if _, err := pool.Exec(sourceCtx, `
		update education_portfolio_transfers set export_manifest_id=null where id=$1::uuid
	`, transferID); err == nil {
		t.Fatalf("sent transfer package must not be alterable")
	}
	if _, err := pool.Exec(sourceCtx, `delete from education_portfolio_transfers where id=$1::uuid`, transferID); err == nil || !strings.Contains(err.Error(), "cannot be deleted") {
		t.Fatalf("sent transfer must be undeletable; err=%v", err)
	}
}
