package admission

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"regexp"
	"strings"
	"time"

	"github.com/eguilde/egueducation/internal/audit"
	"github.com/eguilde/egueducation/internal/auth"
	"github.com/eguilde/egueducation/internal/db"
	"github.com/eguilde/egueducation/internal/httpx"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
)

const (
	permissionRead             = "education.admissions.read"
	permissionManage           = "education.admissions.manage"
	permissionDecide           = "education.admissions.decide"
	permissionAppeals          = "education.admissions.appeals.manage"
	permissionRetention        = "education.admissions.retention.manage"
	permissionRetentionApprove = "education.admissions.retention.approve"
)

var (
	errBadScope            = errors.New("admission authenticated scope is missing or mismatched")
	errForbidden           = errors.New("admission permission denied")
	errIdempotencyConflict = errors.New("idempotency key was reused with another request")
	errVersionConflict     = errors.New("admission aggregate version conflict")
	errPolicyDenied        = errors.New("admission operation denied by policy")
	errInvalidState        = errors.New("admission state conflict")
	errInvalidInput        = errors.New("admission validation failed")
	codePattern            = regexp.MustCompile(`^[A-Za-z0-9][A-Za-z0-9._/-]{0,63}$`)
)

type Service struct{ pool *db.SessionPool }

func New(pool *db.SessionPool) *Service { return &Service{pool: pool} }

type scope struct{ tenant, institution, actor string }

const permissionSQL = `select exists(
	select 1 from app_users u join app_user_permissions p on p.user_id=u.id and p.tenant_code=public.current_tenant_code() where (u.id::text=$1 or lower(u.sub)=lower($1)) and u.status='active' and p.permission_code=$2 and exists(select 1 from app_memberships m where m.user_id=u.id and m.tenant_code=public.current_tenant_code() and public.education_membership_is_eligible(u.id,public.current_tenant_code(),public.current_institution_id(),m.position_code))
	union all select 1 from app_users u join app_user_roles ur on ur.user_id=u.id and ur.tenant_code=public.current_tenant_code() join app_role_permissions rp on rp.role_code=ur.role_code where (u.id::text=$1 or lower(u.sub)=lower($1)) and u.status='active' and rp.permission_code=$2 and exists(select 1 from app_memberships m where m.user_id=u.id and m.tenant_code=public.current_tenant_code() and public.education_membership_is_eligible(u.id,public.current_tenant_code(),public.current_institution_id(),m.position_code))
	union all select 1 from app_users u join app_memberships m on m.user_id=u.id and m.tenant_code=public.current_tenant_code() join app_position_permissions p on p.position_code=m.position_code where (u.id::text=$1 or lower(u.sub)=lower($1)) and u.status='active' and p.permission_code=$2 and public.education_membership_is_eligible(u.id,public.current_tenant_code(),public.current_institution_id(),m.position_code)
	union all select 1 from app_users u join app_memberships m on m.user_id=u.id and m.tenant_code=public.current_tenant_code() join app_position_roles pr on pr.position_code=m.position_code join app_role_permissions rp on rp.role_code=pr.role_code where (u.id::text=$1 or lower(u.sub)=lower($1)) and u.status='active' and rp.permission_code=$2 and public.education_membership_is_eligible(u.id,public.current_tenant_code(),public.current_institution_id(),m.position_code)
)`

func requestScope(r *http.Request) (scope, bool) {
	s := scope{strings.TrimSpace(auth.CurrentTenantCodeFromRequest(r)), strings.TrimSpace(auth.CurrentInstitutionIDFromRequest(r)), strings.TrimSpace(auth.CurrentSubjectFromRequest(r))}
	return s, s.tenant != "" && s.institution != "" && s.actor != ""
}

// begin binds transaction-local settings only from the authenticated request
// and verifies the connection was already host-bound. It never accepts scope
// from a DTO or URL.
func (s *Service) begin(ctx context.Context, requested scope) (pgx.Tx, error) {
	if s == nil || s.pool == nil || requested.tenant == "" || requested.institution == "" || requested.actor == "" {
		return nil, errBadScope
	}
	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return nil, err
	}
	var tenant, institution, actor string
	err = tx.QueryRow(ctx, `select coalesce(nullif(current_setting('app.tenant_id',true),''),''),coalesce(nullif(current_setting('app.institution_id',true),''),''),coalesce(nullif(current_setting('app.actor_subject',true),''),'')`).Scan(&tenant, &institution, &actor)
	if err != nil || tenant != requested.tenant || institution != requested.institution || actor != requested.actor {
		_ = tx.Rollback(ctx)
		return nil, errBadScope
	}
	for key, value := range map[string]string{"app.tenant_id": tenant, "app.institution_id": institution, "app.actor_subject": actor} {
		if _, err = tx.Exec(ctx, `select set_config($1,$2,true)`, key, value); err != nil {
			_ = tx.Rollback(ctx)
			return nil, err
		}
	}
	return tx, nil
}

func requirePermission(ctx context.Context, tx pgx.Tx, s scope, permission string) error {
	var allowed bool
	err := tx.QueryRow(ctx, permissionSQL, s.actor, permission).Scan(&allowed)
	if err != nil {
		return err
	}
	if !allowed {
		return errForbidden
	}
	return nil
}

func requirePosition(ctx context.Context, tx pgx.Tx, s scope, position string) error {
	var allowed bool
	err := tx.QueryRow(ctx, `select exists(select 1 from app_users u join app_memberships m on m.user_id=u.id and m.tenant_code=public.current_tenant_code() where (u.id::text=$1 or lower(u.sub)=lower($1)) and u.status='active' and m.position_code=$2 and public.education_membership_is_eligible(u.id,public.current_tenant_code(),public.current_institution_id(),m.position_code))`, s.actor, position).Scan(&allowed)
	if err != nil {
		return err
	}
	if !allowed {
		return errForbidden
	}
	return nil
}

func decode(w http.ResponseWriter, r *http.Request, target any) error {
	d := json.NewDecoder(http.MaxBytesReader(w, r.Body, 256<<10))
	d.DisallowUnknownFields()
	if err := d.Decode(target); err != nil {
		return err
	}
	if err := d.Decode(&struct{}{}); err != io.EOF {
		return errors.New("request must contain one JSON value")
	}
	return nil
}

func fingerprint(v any) string {
	raw, _ := json.Marshal(v)
	sum := sha256.Sum256(raw)
	return hex.EncodeToString(sum[:])
}

func idempotencyKey(r *http.Request) (string, bool) {
	key := strings.TrimSpace(r.Header.Get("Idempotency-Key"))
	return key, key != "" && len(key) <= 200
}

func reserve(ctx context.Context, tx pgx.Tx, s scope, operation, key, requestHash, proposedID string) (string, bool, error) {
	response, _ := json.Marshal(map[string]string{"id": proposedID})
	var saved json.RawMessage
	err := tx.QueryRow(ctx, `insert into school_admission_idempotency(tenant_code,institution_id,actor_subject,operation_code,idempotency_key,request_fingerprint,response_status,response_snapshot)
		values($1,$2,$3,$4,$5,$6,200,$7::jsonb) on conflict do nothing returning response_snapshot`, s.tenant, s.institution, s.actor, operation, key, requestHash, response).Scan(&saved)
	if err == nil {
		return proposedID, false, nil
	}
	if !errors.Is(err, pgx.ErrNoRows) {
		return "", false, err
	}
	var storedHash string
	err = tx.QueryRow(ctx, `select request_fingerprint,response_snapshot from school_admission_idempotency where tenant_code=$1 and institution_id=$2 and actor_subject=$3 and operation_code=$4 and idempotency_key=$5 for update`, s.tenant, s.institution, s.actor, operation, key).Scan(&storedHash, &saved)
	if err != nil {
		return "", false, err
	}
	if storedHash != requestHash {
		return "", false, errIdempotencyConflict
	}
	var replay struct {
		ID string `json:"id"`
	}
	if json.Unmarshal(saved, &replay) != nil || replay.ID == "" {
		return "", false, errors.New("invalid idempotency response")
	}
	return replay.ID, true, nil
}

func outbox(ctx context.Context, tx pgx.Tx, s scope, aggregate, id, event string, payload any) error {
	raw, err := json.Marshal(payload)
	if err != nil {
		return err
	}
	_, err = tx.Exec(ctx, `insert into school_admission_outbox(tenant_code,institution_id,aggregate_type,aggregate_id,event_type,payload,occurred_by_subject) values($1,$2,$3,$4::uuid,$5,$6::jsonb,$7)`, s.tenant, s.institution, aggregate, id, event, raw, s.actor)
	return err
}

func auditEvent(ctx context.Context, tx pgx.Tx, s scope, action, target, id string) error {
	return audit.Log(ctx, tx, audit.Event{ActorSubject: s.actor, Action: action, TargetType: target, TargetID: id, Summary: action})
}

type archiveSnapshot struct {
	versionNo                                   int
	bucket, objectKey, objectVersion, sha, mime string
	size                                        int64
	retention                                   time.Time
}

func loadArchiveSnapshot(ctx context.Context, tx pgx.Tx, institution string, ref ArchiveReference) (archiveSnapshot, error) {
	var a archiveSnapshot
	err := tx.QueryRow(ctx, `select v.version_no,v.source_bucket,v.source_object_key,v.source_object_version_id,v.retention_until,lower(v.source_sha256),v.mime_type,v.source_size_bytes
		from archive_documents d join archive_document_versions v on v.institution_id=d.institution_id and v.document_id=d.id
		where d.institution_id=$1 and d.id=$2::uuid and d.status='ready' and v.id=$3::uuid and v.status='active' and v.source_object_version_id<>'' and v.retention_until>now()`, institution, ref.DocumentID, ref.VersionID).Scan(&a.versionNo, &a.bucket, &a.objectKey, &a.objectVersion, &a.retention, &a.sha, &a.mime, &a.size)
	return a, err
}

func writeError(w http.ResponseWriter, err error, fallback string) {
	switch {
	case errors.Is(err, errBadScope):
		httpx.JSON(w, http.StatusUnauthorized, map[string]any{"code": "admission_context_required"})
	case errors.Is(err, errForbidden), errors.Is(err, errPolicyDenied):
		httpx.JSON(w, http.StatusForbidden, map[string]any{"code": "admission_forbidden"})
	case errors.Is(err, errIdempotencyConflict), errors.Is(err, errVersionConflict):
		httpx.JSON(w, http.StatusConflict, map[string]any{"code": err.Error()})
	case errors.Is(err, errInvalidState):
		httpx.JSON(w, http.StatusConflict, map[string]any{"code": "admission_state_conflict"})
	case errors.Is(err, errInvalidInput):
		httpx.JSON(w, http.StatusUnprocessableEntity, map[string]any{"code": "admission_validation_failed"})
	case errors.Is(err, pgx.ErrNoRows):
		httpx.JSON(w, http.StatusNotFound, map[string]any{"code": "admission_not_found"})
	default:
		var pe *pgconn.PgError
		if errors.As(err, &pe) && (pe.Code == "23505" || pe.Code == "23P01") {
			httpx.JSON(w, http.StatusConflict, map[string]any{"code": "admission_conflict"})
			return
		}
		if errors.As(err, &pe) && strings.HasPrefix(pe.Code, "23") {
			httpx.JSON(w, http.StatusUnprocessableEntity, map[string]any{"code": "admission_invariant_failed", "detail": pe.Message})
			return
		}
		httpx.JSON(w, http.StatusInternalServerError, map[string]any{"code": fallback})
	}
}

func validUUID(value string) bool { _, err := uuid.Parse(strings.TrimSpace(value)); return err == nil }
func validDate(value string) bool {
	_, err := time.Parse(time.DateOnly, strings.TrimSpace(value))
	return err == nil
}
func rawObject(value json.RawMessage) json.RawMessage {
	if len(value) == 0 {
		return json.RawMessage(`{}`)
	}
	return value
}

func mimeAllowed(allowed []string, mime string) bool {
	for _, value := range allowed {
		if strings.EqualFold(strings.TrimSpace(value), strings.TrimSpace(mime)) {
			return true
		}
	}
	return false
}
