package earchiva

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"

	"github.com/eguilde/egueducation/internal/audit"
	auth "github.com/eguilde/egueducation/internal/auth"
	appdb "github.com/eguilde/egueducation/internal/db"
	"github.com/eguilde/egueducation/internal/httpx"
)

const (
	archiveRetentionReadPermission    = "earchiva.retention.read"
	archiveRetentionManagePermission  = "earchiva.retention.manage"
	archiveRetentionApprovePermission = "earchiva.retention.approve"
	archiveRetentionSelect            = `select id::text,taxonomy_node_id::text,source_id::text,source_checksum_sha256,anchor_kind,duration_model,minimum_retention_days,effective_from,effective_to,status,expected_version,proposed_by_subject,proposed_at,approved_by_subject,approved_at,retired_by_subject,retired_at,retirement_reason,created_at,updated_at from archive_series_retention_rules`
)

type ArchiveSeriesRetentionRule struct {
	ID                   string  `json:"id"`
	TaxonomyNodeID       string  `json:"taxonomy_node_id"`
	SourceID             string  `json:"source_id"`
	SourceChecksumSHA256 string  `json:"source_checksum_sha256"`
	AnchorKind           string  `json:"anchor_kind"`
	DurationModel        string  `json:"duration_model"`
	MinimumRetentionDays *int    `json:"minimum_retention_days,omitempty"`
	EffectiveFrom        string  `json:"effective_from"`
	EffectiveTo          *string `json:"effective_to,omitempty"`
	Status               string  `json:"status"`
	ExpectedVersion      int     `json:"expected_version"`
	ProposedBySubject    string  `json:"proposed_by_subject"`
	ProposedAt           string  `json:"proposed_at"`
	ApprovedBySubject    string  `json:"approved_by_subject"`
	ApprovedAt           *string `json:"approved_at,omitempty"`
	RetiredBySubject     string  `json:"retired_by_subject"`
	RetiredAt            *string `json:"retired_at,omitempty"`
	RetirementReason     string  `json:"retirement_reason"`
	CreatedAt            string  `json:"created_at"`
	UpdatedAt            string  `json:"updated_at"`
}
type ProposeArchiveSeriesRetentionRuleRequest struct {
	TaxonomyNodeID       string  `json:"taxonomy_node_id"`
	SourceID             string  `json:"source_id"`
	EffectiveFrom        string  `json:"effective_from"`
	EffectiveTo          *string `json:"effective_to,omitempty"`
	AnchorKind           string  `json:"anchor_kind"`
	DurationModel        string  `json:"duration_model"`
	MinimumRetentionDays *int    `json:"minimum_retention_days,omitempty"`
}
type ApproveArchiveSeriesRetentionRuleRequest struct {
	ExpectedVersion int `json:"expected_version"`
}
type RetireArchiveSeriesRetentionRuleRequest struct {
	ExpectedVersion int    `json:"expected_version"`
	Reason          string `json:"reason"`
}
type ArchiveSeriesRetentionService struct{ pool *appdb.SessionPool }

func NewArchiveSeriesRetentionService(pool *appdb.SessionPool) *ArchiveSeriesRetentionService {
	return &ArchiveSeriesRetentionService{pool: pool}
}

func (s *ArchiveSeriesRetentionService) ListArchiveSeriesRetentionRules(w http.ResponseWriter, r *http.Request) {
	tenant, institution, actor, ok := archiveRetentionScope(r)
	if !ok {
		httpx.JSON(w, 403, map[string]any{"code": "missing_institution_context"})
		return
	}
	tx, err := s.pool.Begin(r.Context())
	if err != nil {
		archiveRetentionError(w, err)
		return
	}
	defer tx.Rollback(r.Context()) //nolint:errcheck
	if err = archiveRetentionActorPermission(r.Context(), tx, actor, archiveRetentionReadPermission); err != nil {
		archiveRetentionError(w, err)
		return
	}
	q := httpx.ParsePageQuery(r.URL.Query(), map[string]struct{}{"effective_from": {}, "effective_to": {}, "status": {}, "minimum_retention_days": {}, "created_at": {}, "updated_at": {}}, []string{"taxonomy_node_id", "source_id", "status", "anchor_kind"})
	where, args := archiveRetentionFilters(q.Filters, tenant, institution)
	var total int
	if err = tx.QueryRow(r.Context(), "select count(*) from archive_series_retention_rules where "+where, args...).Scan(&total); err != nil {
		archiveRetentionError(w, err)
		return
	}
	order := "created_at desc, id desc"
	if q.Sort != "" {
		order = q.Sort + " " + q.Direction + ", id " + q.Direction
	}
	args = append(args, q.PageSize, (q.Page-1)*q.PageSize)
	rows, err := tx.Query(r.Context(), archiveRetentionSelect+" where "+where+" order by "+order+fmt.Sprintf(" limit $%d offset $%d", len(args)-1, len(args)), args...)
	if err != nil {
		archiveRetentionError(w, err)
		return
	}
	defer rows.Close()
	items := []ArchiveSeriesRetentionRule{}
	for rows.Next() {
		rule, scanErr := scanArchiveRetentionRule(rows)
		if scanErr != nil {
			archiveRetentionError(w, scanErr)
			return
		}
		items = append(items, rule)
	}
	if err = rows.Err(); err != nil {
		archiveRetentionError(w, err)
		return
	}
	if err = tx.Commit(r.Context()); err != nil {
		archiveRetentionError(w, err)
		return
	}
	httpx.WritePage(w, 200, items, total, q.Page, q.PageSize)
}

func (s *ArchiveSeriesRetentionService) ProposeArchiveSeriesRetentionRule(w http.ResponseWriter, r *http.Request) {
	var in ProposeArchiveSeriesRetentionRuleRequest
	if !decodeArchiveRetentionJSON(w, r, &in) || !validArchiveRetentionProposal(in) {
		httpx.JSON(w, 422, map[string]any{"code": "archive_retention_validation_failed"})
		return
	}
	s.mutate(w, r, "propose", archiveRetentionManagePermission, in, func(ctx context.Context, tx pgx.Tx, tenant, institution, actor string) (string, error) {
		var id string
		err := tx.QueryRow(ctx, `insert into archive_series_retention_rules(tenant_code,institution_id,taxonomy_node_id,source_id,source_checksum_sha256,anchor_kind,duration_model,minimum_retention_days,effective_from,effective_to,proposed_by_subject) select $1,$2,$3::uuid,src.id,src.checksum_sha256,$4,$5,$6,$7::date,$8::date,$9 from school_regulatory_sources src where src.tenant_code=$1 and src.institution_id=$2 and src.id=$10::uuid returning id::text`, tenant, institution, in.TaxonomyNodeID, in.AnchorKind, in.DurationModel, in.MinimumRetentionDays, in.EffectiveFrom, in.EffectiveTo, actor, in.SourceID).Scan(&id)
		return id, err
	})
}

func (s *ArchiveSeriesRetentionService) ApproveArchiveSeriesRetentionRule(w http.ResponseWriter, r *http.Request) {
	var in ApproveArchiveSeriesRetentionRuleRequest
	id := strings.TrimSpace(chi.URLParam(r, "ruleID"))
	if !decodeArchiveRetentionJSON(w, r, &in) || in.ExpectedVersion < 1 || !validArchiveRetentionID(id) {
		httpx.JSON(w, 422, map[string]any{"code": "archive_retention_validation_failed"})
		return
	}
	s.mutate(w, r, "approve", archiveRetentionApprovePermission, struct {
		RuleID  string
		Request ApproveArchiveSeriesRetentionRuleRequest
	}{id, in}, func(ctx context.Context, tx pgx.Tx, tenant, institution, actor string) (string, error) {
		var taxonomy string
		if err := tx.QueryRow(ctx, `select taxonomy_node_id::text from archive_series_retention_rules where tenant_code=$1 and institution_id=$2 and id=$3::uuid for update`, tenant, institution, id).Scan(&taxonomy); err != nil {
			return "", err
		}
		if _, err := tx.Exec(ctx, `select pg_advisory_xact_lock(hashtextextended($1,0))`, tenant+"/"+institution+"/"+taxonomy); err != nil {
			return "", err
		}
		tag, err := tx.Exec(ctx, `update archive_series_retention_rules set status='active' where tenant_code=$1 and institution_id=$2 and id=$3::uuid and status='proposed' and expected_version=$4`, tenant, institution, id, in.ExpectedVersion)
		if err != nil {
			return "", err
		}
		if tag.RowsAffected() != 1 {
			return "", errArchiveRetentionConflict
		}
		return id, nil
	})
}

func (s *ArchiveSeriesRetentionService) RetireArchiveSeriesRetentionRule(w http.ResponseWriter, r *http.Request) {
	var in RetireArchiveSeriesRetentionRuleRequest
	id := strings.TrimSpace(chi.URLParam(r, "ruleID"))
	if !decodeArchiveRetentionJSON(w, r, &in) || in.ExpectedVersion < 1 || strings.TrimSpace(in.Reason) == "" || !validArchiveRetentionID(id) {
		httpx.JSON(w, 422, map[string]any{"code": "archive_retention_validation_failed"})
		return
	}
	s.mutate(w, r, "retire", archiveRetentionManagePermission, struct {
		RuleID  string
		Request RetireArchiveSeriesRetentionRuleRequest
	}{id, in}, func(ctx context.Context, tx pgx.Tx, tenant, institution, actor string) (string, error) {
		tag, err := tx.Exec(ctx, `update archive_series_retention_rules set status='retired',retirement_reason=$1 where tenant_code=$2 and institution_id=$3 and id=$4::uuid and status='active' and expected_version=$5`, strings.TrimSpace(in.Reason), tenant, institution, id, in.ExpectedVersion)
		if err != nil {
			return "", err
		}
		if tag.RowsAffected() != 1 {
			return "", errArchiveRetentionConflict
		}
		return id, nil
	})
}

func (s *ArchiveSeriesRetentionService) mutate(w http.ResponseWriter, r *http.Request, action, permission string, request any, command func(context.Context, pgx.Tx, string, string, string) (string, error)) {
	tenant, institution, actor, ok := archiveRetentionScope(r)
	if !ok {
		httpx.JSON(w, 403, map[string]any{"code": "missing_institution_context"})
		return
	}
	key := strings.TrimSpace(r.Header.Get("Idempotency-Key"))
	if key == "" || len(key) > 200 {
		httpx.JSON(w, 400, map[string]any{"code": "idempotency_key_required"})
		return
	}
	fingerprint, err := archiveRetentionFingerprint(action, request)
	if err != nil {
		httpx.JSON(w, 400, map[string]any{"code": "archive_retention_validation_failed"})
		return
	}
	tx, err := s.pool.Begin(r.Context())
	if err != nil {
		archiveRetentionError(w, err)
		return
	}
	defer tx.Rollback(r.Context()) //nolint:errcheck
	if err = archiveRetentionActorPermission(r.Context(), tx, actor, permission); err != nil {
		archiveRetentionError(w, err)
		return
	}
	if _, err = tx.Exec(r.Context(), `select pg_advisory_xact_lock(hashtextextended($1, 0))`, tenant+"/"+institution+"/"+actor+"/"+action+"/"+key); err != nil {
		archiveRetentionError(w, err)
		return
	}
	var previous, ruleID string
	err = tx.QueryRow(r.Context(), `select request_fingerprint,rule_id::text from archive_series_retention_rule_idempotency where tenant_code=$1 and institution_id=$2 and actor_subject=$3 and action=$4 and idempotency_key=$5 for update`, tenant, institution, actor, action, key).Scan(&previous, &ruleID)
	if err == nil {
		if previous != fingerprint {
			httpx.JSON(w, 409, map[string]any{"code": "idempotency_key_conflict"})
			return
		}
		rule, loadErr := loadArchiveRetentionRule(r.Context(), tx, tenant, institution, ruleID)
		if loadErr != nil {
			archiveRetentionError(w, loadErr)
			return
		}
		if err = tx.Commit(r.Context()); err != nil {
			archiveRetentionError(w, err)
			return
		}
		httpx.JSON(w, 200, rule)
		return
	}
	if !errors.Is(err, pgx.ErrNoRows) {
		archiveRetentionError(w, err)
		return
	}
	ruleID, err = command(r.Context(), tx, tenant, institution, actor)
	if err != nil {
		archiveRetentionError(w, err)
		return
	}
	if _, err = tx.Exec(r.Context(), `insert into archive_series_retention_rule_idempotency(tenant_code,institution_id,actor_subject,action,idempotency_key,request_fingerprint,rule_id) values($1,$2,$3,$4,$5,$6,$7::uuid)`, tenant, institution, actor, action, key, fingerprint, ruleID); err != nil {
		archiveRetentionError(w, err)
		return
	}
	rule, err := loadArchiveRetentionRule(r.Context(), tx, tenant, institution, ruleID)
	if err != nil {
		archiveRetentionError(w, err)
		return
	}
	if err = audit.Log(r.Context(), tx, audit.Event{
		ActorSubject: actor, Action: "earchiva.retention." + action,
		TargetType: "archive_series_retention_rule", TargetID: ruleID,
		Summary: "Archive retention rule " + action + "d.",
	}); err != nil {
		archiveRetentionError(w, err)
		return
	}
	if err = tx.Commit(r.Context()); err != nil {
		archiveRetentionError(w, err)
		return
	}
	status := 200
	if action == "propose" {
		status = 201
	}
	httpx.JSON(w, status, rule)
}

var errArchiveRetentionConflict = errors.New("archive retention conflict")

func archiveRetentionError(w http.ResponseWriter, err error) {
	if errors.Is(err, errAdmissionArtifactForbidden) || errors.Is(err, errAdmissionArtifactScope) {
		httpx.JSON(w, http.StatusForbidden, map[string]any{"code": "archive_retention_forbidden"})
		return
	}
	if errors.Is(err, errArchiveRetentionConflict) {
		httpx.JSON(w, 409, map[string]any{"code": "archive_retention_conflict"})
		return
	}
	var databaseError *pgconn.PgError
	if errors.As(err, &databaseError) && (databaseError.Code == "P0001" || databaseError.Code == "23505") {
		httpx.JSON(w, http.StatusConflict, map[string]any{"code": "archive_retention_conflict"})
		return
	}
	if errors.Is(err, pgx.ErrNoRows) {
		httpx.JSON(w, 404, map[string]any{"code": "archive_retention_rule_not_found"})
		return
	}
	httpx.JSON(w, 500, map[string]any{"code": "archive_retention_failed"})
}
func archiveRetentionScope(r *http.Request) (string, string, string, bool) {
	tenant, institution, actor := strings.TrimSpace(auth.CurrentTenantCodeFromRequest(r)), strings.TrimSpace(auth.CurrentInstitutionIDFromRequest(r)), strings.TrimSpace(auth.CurrentSubjectFromRequest(r))
	return tenant, institution, actor, tenant != "" && institution != "" && actor != ""
}
func archiveRetentionActorPermission(ctx context.Context, tx pgx.Tx, actor, permission string) error {
	return archiveAdmissionActorPermission(ctx, tx, actor, permission)
}
func archiveRetentionFingerprint(action string, request any) (string, error) {
	body, err := json.Marshal(request)
	if err != nil {
		return "", err
	}
	sum := sha256.Sum256(append([]byte(action+"\n"), body...))
	return hex.EncodeToString(sum[:]), nil
}
func validArchiveRetentionProposal(in ProposeArchiveSeriesRetentionRuleRequest) bool {
	if !validArchiveRetentionID(in.TaxonomyNodeID) || !validArchiveRetentionID(in.SourceID) || in.AnchorKind != "intake_received_at" || in.DurationModel != "minimum_days" || in.MinimumRetentionDays == nil || *in.MinimumRetentionDays < 1 || *in.MinimumRetentionDays > 36500 {
		return false
	}
	from, err := time.Parse("2006-01-02", in.EffectiveFrom)
	if err != nil {
		return false
	}
	if in.EffectiveTo == nil {
		return true
	}
	to, err := time.Parse("2006-01-02", *in.EffectiveTo)
	return err == nil && !to.Before(from)
}
func validArchiveRetentionID(value string) bool {
	_, err := uuid.Parse(strings.TrimSpace(value))
	return err == nil
}
func decodeArchiveRetentionJSON(w http.ResponseWriter, r *http.Request, target any) bool {
	r.Body = http.MaxBytesReader(w, r.Body, 1<<20)
	decoder := json.NewDecoder(r.Body)
	decoder.DisallowUnknownFields()
	if decoder.Decode(target) != nil {
		return false
	}
	return decoder.Decode(&struct{}{}) == io.EOF
}
func archiveRetentionFilters(filters map[string]string, tenant, institution string) (string, []any) {
	where := "tenant_code=$1 and institution_id=$2"
	args := []any{tenant, institution}
	for _, key := range []string{"taxonomy_node_id", "source_id", "status", "anchor_kind"} {
		if value := filters[key]; value != "" {
			args = append(args, value)
			where += fmt.Sprintf(" and %s=$%d", key, len(args))
		}
	}
	return where, args
}
func loadArchiveRetentionRule(ctx context.Context, tx pgx.Tx, tenant, institution, id string) (ArchiveSeriesRetentionRule, error) {
	return scanArchiveRetentionRule(tx.QueryRow(ctx, archiveRetentionSelect+" where tenant_code=$1 and institution_id=$2 and id=$3::uuid", tenant, institution, id))
}

type archiveRetentionScanner interface{ Scan(...any) error }

func scanArchiveRetentionRule(row archiveRetentionScanner) (ArchiveSeriesRetentionRule, error) {
	var rule ArchiveSeriesRetentionRule
	var from time.Time
	var to, approvedAt, retiredAt *time.Time
	var proposedAt, createdAt, updatedAt time.Time
	err := row.Scan(&rule.ID, &rule.TaxonomyNodeID, &rule.SourceID, &rule.SourceChecksumSHA256, &rule.AnchorKind, &rule.DurationModel, &rule.MinimumRetentionDays, &from, &to, &rule.Status, &rule.ExpectedVersion, &rule.ProposedBySubject, &proposedAt, &rule.ApprovedBySubject, &approvedAt, &rule.RetiredBySubject, &retiredAt, &rule.RetirementReason, &createdAt, &updatedAt)
	if err != nil {
		return rule, err
	}
	rule.EffectiveFrom = from.Format("2006-01-02")
	rule.EffectiveTo = formatArchiveRetentionDate(to)
	rule.ProposedAt = proposedAt.UTC().Format(time.RFC3339)
	rule.ApprovedAt = formatArchiveRetentionTime(approvedAt)
	rule.RetiredAt = formatArchiveRetentionTime(retiredAt)
	rule.CreatedAt = createdAt.UTC().Format(time.RFC3339)
	rule.UpdatedAt = updatedAt.UTC().Format(time.RFC3339)
	return rule, nil
}
func formatArchiveRetentionDate(value *time.Time) *string {
	if value == nil {
		return nil
	}
	result := value.Format("2006-01-02")
	return &result
}
func formatArchiveRetentionTime(value *time.Time) *string {
	if value == nil {
		return nil
	}
	result := value.UTC().Format(time.RFC3339)
	return &result
}
