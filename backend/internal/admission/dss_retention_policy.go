package admission

import (
	"net/http"
	"strings"
	"time"

	"github.com/eguilde/egueducation/internal/httpx"
	"github.com/google/uuid"
)

type ProposeAdmissionRetentionRuleRequest struct {
	ArtifactKind         string  `json:"artifact_kind"`
	MinimumRetentionDays int     `json:"minimum_retention_days"`
	EffectiveFrom        string  `json:"effective_from"`
	EffectiveTo          *string `json:"effective_to"`
	SourceID             string  `json:"source_id"`
}
type ApproveAdmissionRetentionRuleRequest struct {
	RuleVersionID string `json:"rule_version_id"`
}

// ConfigureDSSRetentionPolicyRequest deliberately contains no legal facts.
type ConfigureDSSRetentionPolicyRequest struct {
	RuleVersionID string `json:"rule_version_id"`
	EffectiveFrom string `json:"effective_from"`
}

func validDSSRetentionEffectiveFrom(value string, now time.Time) bool {
	d, err := time.Parse(time.DateOnly, strings.TrimSpace(value))
	if err != nil {
		return false
	}
	n := now.UTC()
	return !d.After(time.Date(n.Year(), n.Month(), n.Day(), 0, 0, 0, 0, time.UTC))
}
func validRetentionRuleProposal(in ProposeAdmissionRetentionRuleRequest) bool {
	if in.ArtifactKind != "admission_dss" || in.MinimumRetentionDays < 1 || in.MinimumRetentionDays > 36500 || !validUUID(in.SourceID) {
		return false
	}
	from, e := time.Parse(time.DateOnly, strings.TrimSpace(in.EffectiveFrom))
	if e != nil {
		return false
	}
	if in.EffectiveTo == nil || strings.TrimSpace(*in.EffectiveTo) == "" {
		return true
	}
	to, e := time.Parse(time.DateOnly, strings.TrimSpace(*in.EffectiveTo))
	return e == nil && !to.Before(from)
}
func nullableDate(v *string) string {
	if v == nil {
		return ""
	}
	return strings.TrimSpace(*v)
}

func (s *Service) ProposeAdmissionRetentionRule(w http.ResponseWriter, r *http.Request) {
	sc, ok := requestScope(r)
	if !ok {
		writeError(w, errBadScope, "admission_retention_rule_failed")
		return
	}
	key, has := idempotencyKey(r)
	if !has {
		httpx.JSON(w, 400, map[string]any{"code": "idempotency_key_required"})
		return
	}
	var in ProposeAdmissionRetentionRuleRequest
	if decode(w, r, &in) != nil || !validRetentionRuleProposal(in) {
		httpx.JSON(w, 422, map[string]any{"code": "invalid_admission_retention_rule"})
		return
	}
	tx, err := s.begin(r.Context(), sc)
	if err != nil {
		writeError(w, err, "admission_retention_rule_failed")
		return
	}
	defer tx.Rollback(r.Context())
	if err = requirePermission(r.Context(), tx, sc, permissionRetention); err != nil {
		writeError(w, err, "admission_retention_rule_failed")
		return
	}
	if err = requirePosition(r.Context(), tx, sc, "director"); err != nil {
		writeError(w, err, "admission_retention_rule_failed")
		return
	}
	id, replay, err := reserve(r.Context(), tx, sc, "admission.retention_rule.propose", key, fingerprint(in), uuid.NewString())
	var out AdmissionRetentionRuleVersion
	if err == nil && !replay {
		err = tx.QueryRow(r.Context(), `insert into school_admission_retention_rule_versions(id,tenant_code,institution_id,artifact_kind,status,minimum_retention_days,effective_from,effective_to,source_id,proposed_by_subject) values($1::uuid,$2,$3,$4,'proposed',$5,$6::date,nullif($7,'')::date,$8::uuid,$9) returning id::text,artifact_kind,status,minimum_retention_days,effective_from::text,coalesce(effective_to::text,''),source_id::text,source_checksum_sha256,proposed_by_subject`, id, sc.tenant, sc.institution, in.ArtifactKind, in.MinimumRetentionDays, in.EffectiveFrom, nullableDate(in.EffectiveTo), in.SourceID, sc.actor).Scan(&out.ID, &out.ArtifactKind, &out.Status, &out.MinimumRetentionDays, &out.EffectiveFrom, &out.EffectiveTo, &out.SourceID, &out.SourceChecksumSHA256, &out.ProposedBySubject)
	}
	if err == nil && !replay {
		err = auditEvent(r.Context(), tx, sc, "admission.retention_rule.proposed", "school_admission_retention_rule_version", id)
	}
	if err != nil {
		writeError(w, err, "admission_retention_rule_failed")
		return
	}
	if err = tx.Commit(r.Context()); err != nil {
		writeError(w, err, "admission_retention_rule_failed")
		return
	}
	if replay {
		out.ID = id
		out.Status = "proposed"
		out.Replayed = true
	}
	httpx.JSON(w, 201, out)
}

func (s *Service) ApproveAdmissionRetentionRule(w http.ResponseWriter, r *http.Request) {
	sc, ok := requestScope(r)
	if !ok {
		writeError(w, errBadScope, "admission_retention_rule_failed")
		return
	}
	key, has := idempotencyKey(r)
	if !has {
		httpx.JSON(w, 400, map[string]any{"code": "idempotency_key_required"})
		return
	}
	var in ApproveAdmissionRetentionRuleRequest
	if decode(w, r, &in) != nil || !validUUID(in.RuleVersionID) {
		httpx.JSON(w, 422, map[string]any{"code": "invalid_admission_retention_rule"})
		return
	}
	tx, err := s.begin(r.Context(), sc)
	if err != nil {
		writeError(w, err, "admission_retention_rule_failed")
		return
	}
	defer tx.Rollback(r.Context())
	if err = requirePermission(r.Context(), tx, sc, permissionRetentionApprove); err != nil {
		writeError(w, err, "admission_retention_rule_failed")
		return
	}
	if err = requirePosition(r.Context(), tx, sc, "director"); err != nil {
		writeError(w, err, "admission_retention_rule_failed")
		return
	}
	id, replay, err := reserve(r.Context(), tx, sc, "admission.retention_rule.approve", key, fingerprint(in), in.RuleVersionID)
	var out AdmissionRetentionRuleVersion
	if err == nil && !replay {
		_, err = tx.Exec(r.Context(), `update school_admission_retention_rule_versions set status='superseded' where tenant_code=$1 and institution_id=$2 and artifact_kind='admission_dss' and status='active'`, sc.tenant, sc.institution)
	}
	if err == nil && !replay {
		err = tx.QueryRow(r.Context(), `update school_admission_retention_rule_versions set status='active' where tenant_code=$1 and institution_id=$2 and id=$3::uuid and status='proposed' returning id::text,artifact_kind,status,minimum_retention_days,effective_from::text,coalesce(effective_to::text,''),source_id::text,source_checksum_sha256,proposed_by_subject,approved_by_subject`, sc.tenant, sc.institution, id).Scan(&out.ID, &out.ArtifactKind, &out.Status, &out.MinimumRetentionDays, &out.EffectiveFrom, &out.EffectiveTo, &out.SourceID, &out.SourceChecksumSHA256, &out.ProposedBySubject, &out.ApprovedBySubject)
	}
	if err == nil && !replay {
		err = auditEvent(r.Context(), tx, sc, "admission.retention_rule.approved", "school_admission_retention_rule_version", id)
	}
	if err != nil {
		writeError(w, err, "admission_retention_rule_failed")
		return
	}
	if err = tx.Commit(r.Context()); err != nil {
		writeError(w, err, "admission_retention_rule_failed")
		return
	}
	if replay {
		out.ID = id
		out.Status = "active"
		out.Replayed = true
	}
	httpx.JSON(w, 200, out)
}

func (s *Service) ConfigureDSSRetentionPolicy(w http.ResponseWriter, r *http.Request) {
	sc, ok := requestScope(r)
	if !ok {
		writeError(w, errBadScope, "dss_retention_policy_failed")
		return
	}
	key, has := idempotencyKey(r)
	if !has {
		httpx.JSON(w, 400, map[string]any{"code": "idempotency_key_required"})
		return
	}
	var in ConfigureDSSRetentionPolicyRequest
	if decode(w, r, &in) != nil || !validUUID(in.RuleVersionID) || !validDSSRetentionEffectiveFrom(in.EffectiveFrom, time.Now()) {
		httpx.JSON(w, 422, map[string]any{"code": "invalid_dss_retention_policy"})
		return
	}
	tx, err := s.begin(r.Context(), sc)
	if err != nil {
		writeError(w, err, "dss_retention_policy_failed")
		return
	}
	defer tx.Rollback(r.Context())
	if err = requirePermission(r.Context(), tx, sc, permissionRetention); err != nil {
		writeError(w, err, "dss_retention_policy_failed")
		return
	}
	if err = requirePosition(r.Context(), tx, sc, "director"); err != nil {
		writeError(w, err, "dss_retention_policy_failed")
		return
	}
	id, replay, err := reserve(r.Context(), tx, sc, "admission.dss_retention_policy.configure", key, fingerprint(in), uuid.NewString())
	var out AdmissionDSSRetentionPolicy
	if err == nil && !replay {
		_, err = tx.Exec(r.Context(), `select pg_advisory_xact_lock(hashtextextended('school_admission_retention_policy:'||$1||':'||$2,0))`, sc.tenant, sc.institution)
	}
	if err == nil && !replay {
		_, err = tx.Exec(r.Context(), `update school_admission_dss_retention_policies set status='superseded',effective_to=least(coalesce(effective_to,$3::date),$3::date) where tenant_code=$1 and institution_id=$2 and status='active'`, sc.tenant, sc.institution, in.EffectiveFrom)
	}
	if err == nil && !replay {
		err = tx.QueryRow(r.Context(), `insert into school_admission_dss_retention_policies(id,tenant_code,institution_id,status,minimum_retention_days,effective_from,source_id,rule_version_id,created_by_subject) values($1::uuid,$2,$3,'active',1,$4::date,(select source_id from school_admission_retention_rule_versions where tenant_code=$2 and institution_id=$3 and id=$5::uuid),$5::uuid,$6) returning id::text,status,rule_version_id::text,minimum_retention_days,effective_from::text,source_id::text`, id, sc.tenant, sc.institution, in.EffectiveFrom, in.RuleVersionID, sc.actor).Scan(&out.ID, &out.Status, &out.RuleVersionID, &out.MinimumRetentionDays, &out.EffectiveFrom, &out.SourceID)
	}
	if err == nil && !replay {
		err = auditEvent(r.Context(), tx, sc, "admission.dss_retention_policy.configured", "school_admission_dss_retention_policy", id)
	}
	if err != nil {
		writeError(w, err, "dss_retention_policy_failed")
		return
	}
	if err = tx.Commit(r.Context()); err != nil {
		writeError(w, err, "dss_retention_policy_failed")
		return
	}
	if replay {
		out.ID = id
		out.Status = "active"
		out.RuleVersionID = in.RuleVersionID
		out.Replayed = true
	}
	httpx.JSON(w, 201, out)
}

func (s *Service) CurrentDSSRetentionPolicy(w http.ResponseWriter, r *http.Request) {
	sc, ok := requestScope(r)
	if !ok {
		writeError(w, errBadScope, "dss_retention_policy_failed")
		return
	}
	tx, err := s.begin(r.Context(), sc)
	if err != nil {
		writeError(w, err, "dss_retention_policy_failed")
		return
	}
	defer tx.Rollback(r.Context())
	if err = requirePermission(r.Context(), tx, sc, permissionRetention); err != nil {
		writeError(w, err, "dss_retention_policy_failed")
		return
	}
	var out AdmissionDSSRetentionPolicy
	err = tx.QueryRow(r.Context(), `select p.id::text,p.status,p.rule_version_id::text,p.minimum_retention_days,p.effective_from::text,p.source_id::text from school_admission_dss_retention_policies p join school_admission_retention_rule_versions rule on rule.tenant_code=p.tenant_code and rule.institution_id=p.institution_id and rule.id=p.rule_version_id and rule.status='active' where p.tenant_code=$1 and p.institution_id=$2 and p.status='active' and p.effective_from<=current_date and (p.effective_to is null or p.effective_to>=current_date) order by p.effective_from desc,p.id desc limit 1`, sc.tenant, sc.institution).Scan(&out.ID, &out.Status, &out.RuleVersionID, &out.MinimumRetentionDays, &out.EffectiveFrom, &out.SourceID)
	if err != nil {
		writeError(w, err, "dss_retention_policy_failed")
		return
	}
	httpx.JSON(w, 200, out)
}
