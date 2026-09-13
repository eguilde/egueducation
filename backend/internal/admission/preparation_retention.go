package admission

import (
	"context"
	"time"

	"github.com/jackc/pgx/v5"
)

type preparationRetention struct {
	PolicyID, RuleID, SourceID string
	Anchor, RequiredUntil      time.Time
	MinimumDays                int
}

func snapshotPreparationRetention(ctx context.Context, tx pgx.Tx, sc scope, expiresAt time.Time) (preparationRetention, error) {
	var out preparationRetention
	out.Anchor = expiresAt.UTC()
	err := tx.QueryRow(ctx, `select p.id::text,p.rule_version_id::text,r.source_id::text,r.minimum_retention_days from school_admission_dss_retention_policies p join school_admission_retention_rule_versions r on r.tenant_code=p.tenant_code and r.institution_id=p.institution_id and r.id=p.rule_version_id and r.status='active' and r.artifact_kind='admission_dss' and r.effective_from<=timezone('UTC',now())::date and (r.effective_to is null or r.effective_to>=timezone('UTC',now())::date) and r.effective_from<=($3 at time zone 'UTC')::date and (r.effective_to is null or r.effective_to>=($3 at time zone 'UTC')::date) where p.tenant_code=$1 and p.institution_id=$2 and p.status='active' and p.effective_from<=timezone('UTC',now())::date and (p.effective_to is null or p.effective_to>=timezone('UTC',now())::date) and p.effective_from<=($3 at time zone 'UTC')::date and (p.effective_to is null or p.effective_to>=($3 at time zone 'UTC')::date) order by p.effective_from desc,p.id desc limit 1`, sc.tenant, sc.institution, out.Anchor).Scan(&out.PolicyID, &out.RuleID, &out.SourceID, &out.MinimumDays)
	if err != nil {
		return preparationRetention{}, err
	}
	out.RequiredUntil = out.Anchor.AddDate(0, 0, out.MinimumDays)
	return out, nil
}

func validatePreparationRetention(ctx context.Context, tx pgx.Tx, sc scope, snapshot preparationRetention, archiveRetention time.Time) error {
	if snapshot.PolicyID == "" || snapshot.RuleID == "" || snapshot.SourceID == "" || snapshot.MinimumDays < 1 || !snapshot.RequiredUntil.Equal(snapshot.Anchor.AddDate(0, 0, snapshot.MinimumDays)) || archiveRetention.Before(snapshot.RequiredUntil) {
		return errInvalidInput
	}
	var currentDays int
	err := tx.QueryRow(ctx, `select r.minimum_retention_days from school_admission_dss_retention_policies p join school_admission_retention_rule_versions r on r.tenant_code=p.tenant_code and r.institution_id=p.institution_id and r.id=p.rule_version_id and r.status='active' and r.artifact_kind='admission_dss' and r.source_id=$5::uuid where p.tenant_code=$1 and p.institution_id=$2 and p.id=$3::uuid and p.rule_version_id=$4::uuid and p.status='active' and p.effective_from<=timezone('UTC',now())::date and (p.effective_to is null or p.effective_to>=timezone('UTC',now())::date) and p.effective_from<=($6 at time zone 'UTC')::date and (p.effective_to is null or p.effective_to>=($6 at time zone 'UTC')::date) and r.effective_from<=timezone('UTC',now())::date and (r.effective_to is null or r.effective_to>=timezone('UTC',now())::date) and r.effective_from<=($6 at time zone 'UTC')::date and (r.effective_to is null or r.effective_to>=($6 at time zone 'UTC')::date)`, sc.tenant, sc.institution, snapshot.PolicyID, snapshot.RuleID, snapshot.SourceID, snapshot.Anchor).Scan(&currentDays)
	if err != nil || currentDays > snapshot.MinimumDays {
		return errInvalidState
	}
	return nil
}

func preparationRetentionFromModel(p AdmissionLegalPreparation) preparationRetention {
	return preparationRetention{PolicyID: p.RetentionPolicyID, RuleID: p.RetentionRuleVersionID, SourceID: p.RetentionSourceID, Anchor: mustParseTimestamp(p.RetentionAnchorAt), MinimumDays: p.MinimumRetentionDays, RequiredUntil: mustParseTimestamp(p.RequiredRetentionUntil)}
}
