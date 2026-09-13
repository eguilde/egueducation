package admission

import (
	"context"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/eguilde/egueducation/internal/db"
)

func TestAdmissionPreparationRetentionResponseRoundTripPostgres(t *testing.T) {
	dsn := strings.TrimSpace(os.Getenv("EGUEDUCATION_TEST_DATABASE_URL"))
	if dsn == "" {
		dsn = strings.TrimSpace(os.Getenv("TEST_DATABASE_URL"))
	}
	if dsn == "" {
		t.Skip("real PostgreSQL response regression requires TEST_DATABASE_URL")
	}
	ctx, cancel := context.WithTimeout(context.Background(), 90*time.Second)
	defer cancel()
	pool := newAdmissionExpiryDatabase(t, ctx, dsn)
	if err := db.Migrate(ctx, pool); err != nil {
		t.Fatal(err)
	}
	sessions := db.NewSessionPool(pool)
	requestCtx, release, err := db.AcquireRequestConn(ctx, pool, db.SessionConfig{TenantID: "tenant-egueducation", InstitutionID: "inst-001", ActorSubject: "expiry-regression-actor", IsSuperAdmin: true})
	if err != nil {
		t.Fatal(err)
	}
	defer release()
	applicationID, _, evaluationID := seedAdmissionExpiryGraph(t, requestCtx, sessions)
	seedExpiryRetentionAuthority(t, ctx, pool, sessions)
	authority := loadPreparationRetentionAuthority(t, requestCtx, sessions)
	id := insertRetentionPreparation(t, requestCtx, sessions, applicationID, evaluationID, authority, "")
	var expectedDeadline string
	for _, zone := range []string{"UTC", "Europe/Bucharest", "Pacific/Auckland"} {
		t.Run(zone, func(t *testing.T) {
			tx, err := sessions.Begin(requestCtx)
			if err != nil {
				t.Fatal(err)
			}
			defer tx.Rollback(requestCtx)
			if _, err = tx.Exec(requestCtx, `select set_config('TimeZone',$1,true)`, zone); err != nil {
				t.Fatal(err)
			}
			var out AdmissionLegalPreparation
			sc := scope{tenant: "tenant-egueducation", institution: "inst-001", actor: "expiry-regression-actor"}
			if err = loadPreparation(requestCtx, tx, sc, id, &out); err != nil {
				t.Fatal(err)
			}
			for field, value := range map[string]string{"prepared_at": out.PreparedAt, "expires_at": out.ExpiresAt, "retention_anchor_at": out.RetentionAnchorAt, "required_retention_until": out.RequiredRetentionUntil} {
				if _, err := time.Parse(time.RFC3339Nano, value); err != nil || !strings.HasSuffix(value, "Z") {
					t.Fatalf("%s is not UTC RFC3339: %q (%v)", field, value, err)
				}
			}
			if out.PolicyEvaluationV2ID != evaluationID || out.RetentionPolicyID != authority.policyID || out.RetentionSourceID != authority.sourceID || out.MinimumRetentionDays != authority.days {
				t.Fatalf("lost persisted authority: %+v", out)
			}
			if expectedDeadline == "" {
				expectedDeadline = out.RequiredRetentionUntil
			}
			if out.RequiredRetentionUntil != expectedDeadline || out.RetentionAnchorAt != out.ExpiresAt {
				t.Fatalf("session timezone changed snapshot: %+v", out)
			}
			snapshot := preparationRetentionFromModel(out)
			if err = validatePreparationRetention(requestCtx, tx, sc, snapshot, snapshot.RequiredUntil); err != nil {
				t.Fatalf("persisted response snapshot rejected: %v", err)
			}
			if err = validatePreparationRetention(requestCtx, tx, sc, snapshot, snapshot.RequiredUntil.Add(-time.Microsecond)); err == nil {
				t.Fatal("shortened archive retention accepted")
			}
		})
	}
}
