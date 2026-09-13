package admission

import (
	"errors"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/eguilde/egueducation/internal/httpx"
	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
)

const (
	permissionSignerManage  = "education.admissions.signer.manage"
	permissionSignerApprove = "education.admissions.signer.approve"
)

func validSignerAuthorization(in ProposeAdmissionSignerAuthorizationRequest) bool {
	if !sha256Text(strings.ToLower(strings.TrimSpace(in.CertificateSHA256))) || !validUUID(in.UserID) || (in.PermissionCode != permissionDecide && in.PermissionCode != permissionAppeals) {
		return false
	}
	until, err := time.Parse(time.RFC3339, strings.TrimSpace(in.ValidUntil))
	return err == nil && until.After(time.Now().UTC()) && until.Before(time.Now().UTC().AddDate(10, 0, 0))
}

func (s *Service) ListAdmissionSignerAuthorizations(w http.ResponseWriter, r *http.Request) {
	sc, ok := requestScope(r)
	if !ok {
		writeError(w, errBadScope, "signer_authorization_list_failed")
		return
	}
	tx, err := s.begin(r.Context(), sc)
	if err != nil {
		writeError(w, err, "signer_authorization_list_failed")
		return
	}
	defer tx.Rollback(r.Context())
	if err = requirePermission(r.Context(), tx, sc, permissionSignerManage); errors.Is(err, errForbidden) {
		err = requirePermission(r.Context(), tx, sc, permissionSignerApprove)
	}
	if err != nil {
		writeError(w, err, "signer_authorization_list_failed")
		return
	}
	q := httpx.ParsePageQuery(r.URL.Query(), map[string]struct{}{"status": {}, "permission_code": {}, "actor_subject": {}}, []string{"status", "permission_code", "actor_subject", "valid_until"})
	where := " where tenant_code=$1 and institution_id=$2"
	args := []any{sc.tenant, sc.institution}
	for _, f := range []string{"status", "permission_code", "actor_subject"} {
		if v := strings.TrimSpace(q.Filters[f]); v != "" {
			args = append(args, v)
			where += fmt.Sprintf(" and %s=$%d", f, len(args))
		}
	}
	var total int
	if err = tx.QueryRow(r.Context(), "select count(*) from school_admission_signer_authorizations"+where, args...).Scan(&total); err != nil {
		writeError(w, err, "signer_authorization_list_failed")
		return
	}
	sort := q.Sort
	if sort == "" {
		sort = "valid_until"
	}
	args = append(args, q.PageSize, (q.Page-1)*q.PageSize)
	rows, err := tx.Query(r.Context(), "select id::text,certificate_sha256,user_id::text,actor_subject,permission_code,valid_until::text,proposed_by_subject,approved_by_subject,status,expected_version from school_admission_signer_authorizations"+where+fmt.Sprintf(" order by %s %s,id limit $%d offset $%d", sort, q.Direction, len(args)-1, len(args)), args...)
	if err != nil {
		writeError(w, err, "signer_authorization_list_failed")
		return
	}
	defer rows.Close()
	items := []AdmissionSignerAuthorization{}
	for rows.Next() {
		var x AdmissionSignerAuthorization
		var version int
		if err = rows.Scan(&x.ID, &x.CertificateSHA256, &x.UserID, &x.ActorSubject, &x.PermissionCode, &x.ValidUntil, &x.ProposedBySubject, &x.ApprovedBySubject, &x.Status, &version); err != nil {
			writeError(w, err, "signer_authorization_list_failed")
			return
		}
		x.ExpectedVersion = version
		items = append(items, x)
	}
	if err = rows.Err(); err != nil {
		writeError(w, err, "signer_authorization_list_failed")
		return
	}
	if err = tx.Commit(r.Context()); err != nil {
		writeError(w, err, "signer_authorization_list_failed")
		return
	}
	httpx.WritePage(w, 200, items, total, q.Page, q.PageSize)
}

func (s *Service) RevokeAdmissionSignerAuthorization(w http.ResponseWriter, r *http.Request) {
	sc, ok := requestScope(r)
	if !ok {
		writeError(w, errBadScope, "signer_authorization_revoke_failed")
		return
	}
	id := chi.URLParam(r, "authorizationID")
	if !validUUID(id) {
		httpx.JSON(w, 422, map[string]any{"code": "invalid_signer_authorization"})
		return
	}
	key, ok := idempotencyKey(r)
	if !ok {
		httpx.JSON(w, 400, map[string]any{"code": "idempotency_key_required"})
		return
	}
	var in RevokeAdmissionSignerAuthorizationRequest
	if decode(w, r, &in) != nil || in.ExpectedVersion < 1 || strings.TrimSpace(in.Reason) == "" || len(strings.TrimSpace(in.Reason)) > 2000 {
		httpx.JSON(w, 422, map[string]any{"code": "invalid_signer_revocation"})
		return
	}
	tx, err := s.begin(r.Context(), sc)
	if err != nil {
		writeError(w, err, "signer_authorization_revoke_failed")
		return
	}
	defer tx.Rollback(r.Context())
	if err = requirePermission(r.Context(), tx, sc, permissionSignerManage); err == nil {
		err = requirePosition(r.Context(), tx, sc, "director")
	}
	if err != nil {
		writeError(w, err, "signer_authorization_revoke_failed")
		return
	}
	_, replay, err := reserve(r.Context(), tx, sc, "admission.signer.revoke", key, fingerprint(struct {
		ID string
		In RevokeAdmissionSignerAuthorizationRequest
	}{id, in}), id)
	if err == nil && !replay {
		tag, e := tx.Exec(r.Context(), `update school_admission_signer_authorizations set status='revoked',revocation_reason=$1 where tenant_code=$2 and institution_id=$3 and id=$4::uuid and status='active' and expected_version=$5`, strings.TrimSpace(in.Reason), sc.tenant, sc.institution, id, in.ExpectedVersion)
		err = e
		if err == nil && tag.RowsAffected() != 1 {
			err = errVersionConflict
		}
	}
	if err == nil {
		err = auditEvent(r.Context(), tx, sc, "admission.signer_authorization.revoked", "school_admission_signer_authorization", id)
	}
	if err != nil {
		writeError(w, err, "signer_authorization_revoke_failed")
		return
	}
	if err = tx.Commit(r.Context()); err != nil {
		writeError(w, err, "signer_authorization_revoke_failed")
		return
	}
	httpx.JSON(w, 201, CommandResult{ID: id, Status: "revoked", Replayed: replay})
}

func (s *Service) ProposeAdmissionSignerAuthorization(w http.ResponseWriter, r *http.Request) {
	sc, ok := requestScope(r)
	if !ok {
		writeError(w, errBadScope, "signer_authorization_failed")
		return
	}
	key, ok := idempotencyKey(r)
	if !ok {
		httpx.JSON(w, 400, map[string]any{"code": "idempotency_key_required"})
		return
	}
	var in ProposeAdmissionSignerAuthorizationRequest
	if decode(w, r, &in) != nil || !validSignerAuthorization(in) {
		httpx.JSON(w, 422, map[string]any{"code": "invalid_signer_authorization"})
		return
	}
	in.CertificateSHA256 = strings.ToLower(strings.TrimSpace(in.CertificateSHA256))
	in.ValidUntil = strings.TrimSpace(in.ValidUntil)
	tx, err := s.begin(r.Context(), sc)
	if err != nil {
		writeError(w, err, "signer_authorization_failed")
		return
	}
	defer tx.Rollback(r.Context())
	if err = requirePermission(r.Context(), tx, sc, permissionSignerManage); err == nil {
		err = requirePosition(r.Context(), tx, sc, "director")
	}
	if err != nil {
		writeError(w, err, "signer_authorization_failed")
		return
	}
	id, replay, err := reserve(r.Context(), tx, sc, "admission.signer.propose", key, fingerprint(in), uuid.NewString())
	if err != nil {
		writeError(w, err, "signer_authorization_failed")
		return
	}
	var out AdmissionSignerAuthorization
	if !replay {
		err = tx.QueryRow(r.Context(), `insert into school_admission_signer_authorizations(id,tenant_code,institution_id,certificate_sha256,user_id,actor_subject,permission_code,valid_until,proposed_by_subject) select $1::uuid,$2,$3,$4,$5::uuid,u.sub,$6,$7::timestamptz,$8 from app_users u where u.id=$5::uuid and u.status='active' and exists(select 1 from app_memberships m where m.user_id=u.id and m.tenant_code=$2 and public.education_membership_is_eligible(u.id,$2,$3,m.position_code)) returning id::text,certificate_sha256,user_id::text,actor_subject,permission_code,valid_until::text,proposed_by_subject,status`, id, sc.tenant, sc.institution, in.CertificateSHA256, in.UserID, in.PermissionCode, in.ValidUntil, sc.actor).Scan(&out.ID, &out.CertificateSHA256, &out.UserID, &out.ActorSubject, &out.PermissionCode, &out.ValidUntil, &out.ProposedBySubject, &out.Status)
	} else {
		err = tx.QueryRow(r.Context(), `select id::text,certificate_sha256,user_id::text,actor_subject,permission_code,valid_until::text,proposed_by_subject,status from school_admission_signer_authorizations where tenant_code=$1 and institution_id=$2 and id=$3::uuid`, sc.tenant, sc.institution, id).Scan(&out.ID, &out.CertificateSHA256, &out.UserID, &out.ActorSubject, &out.PermissionCode, &out.ValidUntil, &out.ProposedBySubject, &out.Status)
		out.Replayed = true
	}
	if err == nil {
		err = auditEvent(r.Context(), tx, sc, "admission.signer_authorization.proposed", "school_admission_signer_authorization", id)
	}
	if err != nil {
		writeError(w, err, "signer_authorization_failed")
		return
	}
	if err = tx.Commit(r.Context()); err != nil {
		writeError(w, err, "signer_authorization_failed")
		return
	}
	httpx.JSON(w, 201, out)
}

func (s *Service) ApproveAdmissionSignerAuthorization(w http.ResponseWriter, r *http.Request) {
	sc, ok := requestScope(r)
	if !ok {
		writeError(w, errBadScope, "signer_authorization_failed")
		return
	}
	key, ok := idempotencyKey(r)
	if !ok {
		httpx.JSON(w, 400, map[string]any{"code": "idempotency_key_required"})
		return
	}
	var in ApproveAdmissionSignerAuthorizationRequest
	if decode(w, r, &in) != nil || !validUUID(in.ProposalID) {
		httpx.JSON(w, 422, map[string]any{"code": "invalid_signer_authorization"})
		return
	}
	tx, err := s.begin(r.Context(), sc)
	if err != nil {
		writeError(w, err, "signer_authorization_failed")
		return
	}
	defer tx.Rollback(r.Context())
	if err = requirePermission(r.Context(), tx, sc, permissionSignerApprove); err == nil {
		err = requirePosition(r.Context(), tx, sc, "director")
	}
	if err != nil {
		writeError(w, err, "signer_authorization_failed")
		return
	}
	id, replay, err := reserve(r.Context(), tx, sc, "admission.signer.approve", key, fingerprint(in), in.ProposalID)
	if err != nil {
		writeError(w, err, "signer_authorization_failed")
		return
	}
	var out AdmissionSignerAuthorization
	if !replay {
		err = tx.QueryRow(r.Context(), `update school_admission_signer_authorizations set status='active' where tenant_code=$1 and institution_id=$2 and id=$3::uuid and status='proposed' returning id::text,certificate_sha256,user_id::text,actor_subject,permission_code,valid_until::text,proposed_by_subject,approved_by_subject,status`, sc.tenant, sc.institution, id).Scan(&out.ID, &out.CertificateSHA256, &out.UserID, &out.ActorSubject, &out.PermissionCode, &out.ValidUntil, &out.ProposedBySubject, &out.ApprovedBySubject, &out.Status)
	} else {
		err = tx.QueryRow(r.Context(), `select id::text,certificate_sha256,user_id::text,actor_subject,permission_code,valid_until::text,proposed_by_subject,approved_by_subject,status from school_admission_signer_authorizations where tenant_code=$1 and institution_id=$2 and id=$3::uuid`, sc.tenant, sc.institution, id).Scan(&out.ID, &out.CertificateSHA256, &out.UserID, &out.ActorSubject, &out.PermissionCode, &out.ValidUntil, &out.ProposedBySubject, &out.ApprovedBySubject, &out.Status)
		out.Replayed = true
	}
	if errors.Is(err, pgx.ErrNoRows) {
		err = errInvalidState
	}
	if err == nil {
		err = auditEvent(r.Context(), tx, sc, "admission.signer_authorization.approved", "school_admission_signer_authorization", id)
	}
	if err != nil {
		writeError(w, err, "signer_authorization_failed")
		return
	}
	if err = tx.Commit(r.Context()); err != nil {
		writeError(w, err, "signer_authorization_failed")
		return
	}
	httpx.JSON(w, 201, out)
}
