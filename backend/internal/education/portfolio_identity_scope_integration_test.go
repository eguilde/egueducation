//go:build integration

package education

import (
	"context"
	"strings"
	"testing"
	"time"

	appdb "github.com/eguilde/egueducation/internal/db"
)

// TestPortfolioIdentityAndScopedNaturalCodesIntegration proves the database,
// rather than a UI convention, accepts a school-local code in another tenant
// and rejects a forged portfolio owner/personnel pairing.
func TestPortfolioIdentityAndScopedNaturalCodesIntegration(t *testing.T) {
	it := newGovernanceIntegrationDatabase(t)
	ctx, cancel := context.WithTimeout(context.Background(), 90*time.Second)
	defer cancel()
	admin := openGovernanceIntegrationPool(t, ctx, it.databaseConfig)
	defer admin.Close()
	if err := appdb.Migrate(ctx, admin); err != nil {
		t.Fatalf("migrate identity/scope integration database: %v", err)
	}
	grantGovernanceIntegrationAccess(t, ctx, admin, it.roleName)
	fixture := seedGovernanceAuthorizationFixture(t, ctx, admin)

	// A code is local to an institution. The original fixture personnel and a
	// distinct tenant-B row deliberately share the same registry code.
	if _, err := admin.Exec(ctx, `
		insert into education_personnel (
			employee_code, full_name, role_title, employment_type, status,
			evaluation_status, mobility_stage, school_year, institution_id
		) values ('PER-GOV-OWNER', 'Tenant B teacher', 'Profesor', 'titular', 'active', 'draft', 'none', '2026-2027', $1)
	`, fixture.institutionB); err != nil {
		t.Fatalf("same personnel code must be valid in another institution: %v", err)
	}

	// The database trigger rejects a direct SQL spoof, including callers that
	// bypass HTTP validation. User A cannot claim personnel B's portfolio.
	if _, err := admin.Exec(ctx, `
		insert into education_portfolios (
			portfolio_code, owner_user_id, owner_personnel_id, owner_name, owner_role,
			school_year, status, section_count, last_updated_on, retention_until,
			transfer_status, institution_id
		) values ('PORT-SPOOF-IDENTITY', $1::uuid, $2::uuid, 'Spoofed', 'Profesor',
			'2027-2028', 'draft', 0, current_date, current_date + 365, 'none', $3)
	`, fixture.memberUserID, fixture.foreignPersonnelID, fixture.institutionA); err == nil || !strings.Contains(err.Error(), "canonically linked") {
		t.Fatalf("cross-person portfolio spoof must be rejected by trigger; err=%v", err)
	}

	// A personnel link referenced by a portfolio cannot later be rebound to a
	// different user, preventing a silent ownership takeover.
	if _, err := admin.Exec(ctx, `update education_personnel set app_user_id=$1::uuid where id=$2::uuid`, fixture.foreignMemberUserID, fixture.memberPersonnelID); err == nil || !strings.Contains(err.Error(), "cannot be reassigned") {
		t.Fatalf("referenced personnel identity rebinding must be rejected; err=%v", err)
	}

	service := NewService(appdb.NewSessionPool(it.readerPool))
	ownerCtx, releaseOwner := governanceTenantContext(t, ctx, it.readerPool, fixture.tenantA, fixture.institutionA, fixture.memberSubject)
	defer releaseOwner()
	request := requestWithContext(ownerCtx, fixture.tenantA, fixture.institutionA, fixture.memberSubject)
	personnelID, err := service.resolvePortfolioPersonnelID(request, fixture.memberUserID)
	if err != nil || personnelID != fixture.memberPersonnelID {
		t.Fatalf("self-service must use durable personnel link: id=%q err=%v want=%q", personnelID, err, fixture.memberPersonnelID)
	}
}
