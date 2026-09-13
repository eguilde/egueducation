package admission

import (
	"context"
	"fmt"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/eguilde/egueducation/internal/db"
	"github.com/google/uuid"
)

func TestAdmissionPreparationRetentionSnapshotPostgres(t *testing.T) {
	dsn := strings.TrimSpace(os.Getenv("EGUEDUCATION_TEST_DATABASE_URL"))
	if dsn == "" {
		dsn = strings.TrimSpace(os.Getenv("TEST_DATABASE_URL"))
	}
	if dsn == "" {
		t.Skip("real PostgreSQL retention snapshot regression requires TEST_DATABASE_URL")
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
	applicationID, _, policyEvaluationID := seedAdmissionExpiryGraph(t, requestCtx, sessions)
	seedExpiryRetentionAuthority(t, ctx, pool, sessions)
	authority := loadPreparationRetentionAuthority(t, requestCtx, sessions)
	var validPreparationID string

	t.Run("valid snapshot is immutable", func(t *testing.T) {
		id := insertRetentionPreparation(t, requestCtx, sessions, applicationID, policyEvaluationID, authority, "")
		validPreparationID = id
		var policyID, ruleID, sourceID string
		var days int
		if err := sessions.QueryRow(requestCtx, `select retention_policy_id::text,retention_rule_version_id::text,retention_source_id::text,minimum_retention_days from school_admission_legal_preparations where id=$1::uuid`, id).Scan(&policyID, &ruleID, &sourceID, &days); err != nil || policyID != authority.policyID || ruleID != authority.ruleID || sourceID != authority.sourceID || days != authority.days {
			t.Fatalf("stored snapshot err=%v policy=%q rule=%q source=%q days=%d", err, policyID, ruleID, sourceID, days)
		}
		if _, err := sessions.Exec(requestCtx, `update school_admission_legal_preparations set minimum_retention_days=minimum_retention_days+1 where id=$1::uuid`, id); err == nil || (!strings.Contains(err.Error(), "retention snapshot is immutable") && !strings.Contains(err.Error(), "lifecycle records are immutable")) {
			t.Fatalf("snapshot rewrite err=%v", err)
		}
	})

	for _, field := range []string{"retention_policy_id", "retention_rule_version_id", "retention_source_id", "retention_anchor_at", "minimum_retention_days", "required_retention_until"} {
		t.Run("missing "+field, func(t *testing.T) {
			if err := insertRetentionPreparationErr(requestCtx, sessions, applicationID, policyEvaluationID, authority, field); err == nil || !strings.Contains(err.Error(), "retention authority snapshot") {
				t.Fatalf("missing %s err=%v", field, err)
			}
		})
	}
	for _, field := range []string{"retention_rule_version_id", "retention_source_id", "required_retention_until"} {
		t.Run("wrong "+field, func(t *testing.T) {
			if err := insertRetentionPreparationErr(requestCtx, sessions, applicationID, policyEvaluationID, authority, "wrong:"+field); err == nil || !strings.Contains(err.Error(), "retention authority snapshot") {
				t.Fatalf("wrong %s err=%v", field, err)
			}
		})
	}

	t.Run("missing snapshot is not finalizable and cancellation transition remains allowed", func(t *testing.T) {
		if err := validatePreparationRetention(requestCtx, nil, scope{}, preparationRetention{}, time.Now().AddDate(0, 0, 1)); err != errInvalidInput {
			t.Fatalf("legacy snapshot validation err=%v, want fail closed", err)
		}
		if _, err := sessions.Exec(requestCtx, `update school_admission_legal_preparations set status='cancelled',cancellation_reason='legacy_reprepare' where id=$1::uuid`, validPreparationID); err != nil {
			t.Fatalf("cancel preparation: %v", err)
		}
	})
}

type preparationRetentionAuthority struct {
	policyID, ruleID, sourceID string
	days                       int
}

func loadPreparationRetentionAuthority(t *testing.T, ctx context.Context, sessions *db.SessionPool) preparationRetentionAuthority {
	t.Helper()
	var out preparationRetentionAuthority
	if err := sessions.QueryRow(ctx, `select p.id::text,p.rule_version_id::text,r.source_id::text,r.minimum_retention_days from school_admission_dss_retention_policies p join school_admission_retention_rule_versions r on r.id=p.rule_version_id where p.tenant_code='tenant-egueducation' and p.institution_id='inst-001' and p.status='active' and r.status='active'`).Scan(&out.policyID, &out.ruleID, &out.sourceID, &out.days); err != nil {
		t.Fatal(err)
	}
	return out
}

func insertRetentionPreparation(t *testing.T, ctx context.Context, sessions *db.SessionPool, applicationID, policyEvaluationID string, authority preparationRetentionAuthority, bad string) string {
	t.Helper()
	var id string
	if err := insertRetentionPreparationInto(ctx, sessions, applicationID, policyEvaluationID, authority, bad, &id); err != nil {
		t.Fatal(err)
	}
	return id
}

func insertRetentionPreparationErr(ctx context.Context, sessions *db.SessionPool, applicationID, policyEvaluationID string, authority preparationRetentionAuthority, bad string) error {
	var ignored string
	return insertRetentionPreparationInto(ctx, sessions, applicationID, policyEvaluationID, authority, bad, &ignored)
}

func insertRetentionPreparationInto(ctx context.Context, sessions *db.SessionPool, applicationID, policyEvaluationID string, authority preparationRetentionAuthority, bad string, id *string) error {
	expires := time.Now().UTC().Add(5 * time.Minute).Truncate(time.Microsecond)
	required := expires.AddDate(0, 0, authority.days)
	values := map[string]string{"retention_policy_id": "nullif($6,'')::uuid", "retention_rule_version_id": "nullif($7,'')::uuid", "retention_source_id": "nullif($8,'')::uuid", "retention_anchor_at": "nullif($9,'')::timestamptz", "minimum_retention_days": "nullif($10,'')::integer", "required_retention_until": "nullif($11,'')::timestamptz"}
	policyID, ruleID, sourceID := authority.policyID, authority.ruleID, authority.sourceID
	anchor, days, requiredText := expires.Format(time.RFC3339Nano), fmt.Sprint(authority.days), required.Format(time.RFC3339Nano)
	if strings.HasPrefix(bad, "wrong:") {
		switch strings.TrimPrefix(bad, "wrong:") {
		case "retention_rule_version_id":
			ruleID = uuid.NewString()
		case "retention_source_id":
			sourceID = uuid.NewString()
		case "required_retention_until":
			requiredText = expires.Add(-time.Hour).Format(time.RFC3339Nano)
		}
	} else if bad != "" {
		switch bad {
		case "retention_policy_id":
			policyID = ""
		case "retention_rule_version_id":
			ruleID = ""
		case "retention_source_id":
			sourceID = ""
		case "retention_anchor_at":
			anchor = ""
		case "minimum_retention_days":
			days = ""
		case "required_retention_until":
			requiredText = ""
		}
	}
	query := fmt.Sprintf(`insert into school_admission_legal_preparations(id,tenant_code,institution_id,artifact_kind,artifact_id,application_id,policy_evaluation_v2_id,aggregate_expected_version,canonical_payload,canonical_payload_bytes,canonical_payload_sha256,preparation_snapshot,prepared_by_subject,expires_at,retention_policy_id,retention_rule_version_id,retention_source_id,retention_anchor_at,minimum_retention_days,required_retention_until) values($1::uuid,'tenant-egueducation','inst-001','admission_decision',$2::uuid,$3::uuid,$4::uuid,1,'{}','{}',encode(digest('{}','sha256'),'hex'),'{}','expiry-regression-actor',$5,%s,%s,%s,%s,%s,%s) returning id::text`, values["retention_policy_id"], values["retention_rule_version_id"], values["retention_source_id"], values["retention_anchor_at"], values["minimum_retention_days"], values["required_retention_until"])
	args := []any{uuid.NewString(), uuid.NewString(), applicationID, policyEvaluationID, expires, policyID, ruleID, sourceID, anchor, days, requiredText}
	return sessions.QueryRow(ctx, query, args...).Scan(id)
}
