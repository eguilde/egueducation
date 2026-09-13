// Package regulatorysource owns the evidence-backed lifecycle of tenant-scoped
// regulatory sources. It deliberately does not retrofit evidence onto legacy rows.
package regulatorysource

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/eguilde/egueducation/internal/auth"
	appdb "github.com/eguilde/egueducation/internal/db"
	"github.com/eguilde/egueducation/internal/httpx"
	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
)

type Service struct {
	pool    *appdb.SessionPool
	fetcher evidenceFetcher
}
type evidenceFetcher interface {
	Fetch(context.Context, string) (Evidence, error)
}

func NewService(pool *appdb.SessionPool, fetcher evidenceFetcher) *Service {
	// An optional concrete fetcher becomes a non-nil interface even when its
	// pointer is nil. Preserve the unconfigured service state in that case.
	if configured, ok := fetcher.(*Fetcher); ok && configured == nil {
		fetcher = nil
	}
	return &Service{pool: pool, fetcher: fetcher}
}

func sourceScope(r *http.Request) (string, string, string, bool) {
	t := strings.TrimSpace(auth.CurrentTenantCodeFromRequest(r))
	i := strings.TrimSpace(auth.CurrentInstitutionIDFromRequest(r))
	a := strings.TrimSpace(auth.CurrentSubjectFromRequest(r))
	return t, i, a, t != "" && i != "" && a != ""
}

func (s *Service) RequirePermission(permission string) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			_, _, subject, ok := sourceScope(r)
			if !ok {
				httpx.JSON(w, http.StatusUnauthorized, map[string]any{"code": "institution_context_required"})
				return
			}
			allowed, err := actorHasPermission(r.Context(), s.pool, subject, permission)
			if err != nil {
				httpx.JSON(w, 500, map[string]any{"code": "regulatory_source_authorization_failed"})
				return
			}
			if !allowed {
				httpx.JSON(w, http.StatusForbidden, map[string]any{"code": "regulatory_source_permission_required", "permission": permission})
				return
			}
			next.ServeHTTP(w, r)
		})
	}
}

type permissionQuery interface {
	QueryRow(context.Context, string, ...any) pgx.Row
}

// actorHasPermission deliberately matches the established live database RBAC
// predicate used by archive retention: claims are never trusted after login.
func actorHasPermission(ctx context.Context, q permissionQuery, actor, permission string) (bool, error) {
	var allowed bool
	err := q.QueryRow(ctx, `select exists(
 select 1 from app_users u join app_user_permissions p on p.user_id=u.id and p.tenant_code=public.current_tenant_code() where (u.id::text=$1 or lower(u.sub)=lower($1)) and u.status='active' and p.permission_code=$2 and exists(select 1 from app_memberships m where m.user_id=u.id and m.tenant_code=public.current_tenant_code() and public.education_membership_is_eligible(u.id,public.current_tenant_code(),public.current_institution_id(),m.position_code))
 union all select 1 from app_users u join app_user_roles ur on ur.user_id=u.id and ur.tenant_code=public.current_tenant_code() join app_role_permissions rp on rp.role_code=ur.role_code where (u.id::text=$1 or lower(u.sub)=lower($1)) and u.status='active' and rp.permission_code=$2 and exists(select 1 from app_memberships m where m.user_id=u.id and m.tenant_code=public.current_tenant_code() and public.education_membership_is_eligible(u.id,public.current_tenant_code(),public.current_institution_id(),m.position_code))
 union all select 1 from app_users u join app_memberships m on m.user_id=u.id and m.tenant_code=public.current_tenant_code() join app_position_permissions p on p.position_code=m.position_code where (u.id::text=$1 or lower(u.sub)=lower($1)) and u.status='active' and p.permission_code=$2 and public.education_membership_is_eligible(u.id,public.current_tenant_code(),public.current_institution_id(),m.position_code)
 union all select 1 from app_users u join app_memberships m on m.user_id=u.id and m.tenant_code=public.current_tenant_code() join app_position_roles pr on pr.position_code=m.position_code join app_role_permissions rp on rp.role_code=pr.role_code where (u.id::text=$1 or lower(u.sub)=lower($1)) and u.status='active' and rp.permission_code=$2 and public.education_membership_is_eligible(u.id,public.current_tenant_code(),public.current_institution_id(),m.position_code)
 )`, actor, permission).Scan(&allowed)
	return allowed, err
}

func (s *Service) List(w http.ResponseWriter, r *http.Request) {
	t, i, _, ok := sourceScope(r)
	if !ok {
		httpx.JSON(w, 401, map[string]any{"code": "institution_context_required"})
		return
	}
	q := httpx.ParsePageQuery(r.URL.Query(), map[string]struct{}{"citation": {}, "source_kind": {}, "status": {}, "created_at": {}, "updated_at": {}}, []string{"status", "source_kind", "citation"})
	where := " where source.tenant_code=$1 and source.institution_id=$2"
	args := []any{t, i}
	for _, f := range []string{"status", "source_kind", "citation"} {
		if v := q.Filters[f]; v != "" {
			args = append(args, v)
			if f == "citation" {
				where += " and source.citation ilike '%' || $" + strconv(len(args)) + " || '%'"
			} else {
				where += " and source." + f + "=$" + strconv(len(args))
			}
		}
	}
	var total int
	if err := s.pool.QueryRow(r.Context(), "select count(*) from school_regulatory_sources source"+where, args...).Scan(&total); err != nil {
		httpx.JSON(w, 500, map[string]any{"code": "regulatory_source_list_failed"})
		return
	}
	sort := map[string]string{"citation": "source.citation", "source_kind": "source.source_kind", "status": "source.status", "created_at": "source.created_at"}[q.Sort]
	if sort == "" {
		sort = "source.created_at"
	}
	dir := "asc"
	if q.Direction == "desc" {
		dir = "desc"
	}
	args = append(args, q.PageSize, (q.Page-1)*q.PageSize)
	rows, err := s.pool.Query(r.Context(), sourceSelect+where+" order by "+sort+" "+dir+", source.id limit $"+strconv(len(args)-1)+" offset $"+strconv(len(args)), args...)
	if err != nil {
		httpx.JSON(w, 500, map[string]any{"code": "regulatory_source_list_failed"})
		return
	}
	defer rows.Close()
	out := []RegulatorySource{}
	for rows.Next() {
		x, err := scanSource(rows)
		if err != nil {
			httpx.JSON(w, 500, map[string]any{"code": "regulatory_source_list_failed"})
			return
		}
		out = append(out, x)
	}
	if err = rows.Err(); err != nil {
		httpx.JSON(w, 500, map[string]any{"code": "regulatory_source_list_failed"})
		return
	}
	httpx.WritePage(w, 200, out, total, q.Page, q.PageSize)
}

func (s *Service) Register(w http.ResponseWriter, r *http.Request) {
	t, i, a, ok := sourceScope(r)
	if !ok {
		httpx.JSON(w, 401, map[string]any{"code": "institution_context_required"})
		return
	}
	key := strings.TrimSpace(r.Header.Get("Idempotency-Key"))
	if key == "" || len(key) > 200 {
		httpx.JSON(w, 422, map[string]any{"code": "idempotency_key_required"})
		return
	}
	var in RegisterRegulatorySourceRequest
	if !decode(w, r, &in) || !validRegister(&in) {
		httpx.JSON(w, 422, map[string]any{"code": "invalid_regulatory_source_payload"})
		return
	}
	if err := rejectQueryURL(in.PublisherURL); err != nil {
		httpx.JSON(w, 422, map[string]any{"code": "unsupported_publisher_url_query"})
		return
	}
	requestFingerprint := fingerprint(in)
	tx, err := s.pool.Begin(r.Context())
	if err != nil {
		writeServer(w, "regulatory_source_register_failed")
		return
	}
	defer tx.Rollback(r.Context())
	if allowed, permissionErr := actorHasPermission(r.Context(), tx, a, "school.regulatory_sources.manage"); permissionErr != nil || !allowed {
		httpx.JSON(w, http.StatusForbidden, map[string]any{"code": "regulatory_source_permission_required"})
		return
	}
	if _, err = tx.Exec(r.Context(), "select pg_advisory_xact_lock(hashtextextended($1,0))",
		fingerprint([]string{t, i, a, "register", key})); err != nil {
		writeServer(w, "regulatory_source_command_lock_failed")
		return
	}
	var existingID, existingFP string
	err = tx.QueryRow(r.Context(), "select source_id::text,request_fingerprint from school_regulatory_source_idempotency where tenant_code=$1 and institution_id=$2 and actor_subject=$3 and action='register' and idempotency_key=$4 for update", t, i, a, key).Scan(&existingID, &existingFP)
	if err == nil {
		if existingFP != requestFingerprint {
			httpx.JSON(w, 409, map[string]any{"code": "idempotency_key_payload_conflict"})
			return
		}
		x, err := loadSource(tx, r.Context(), t, i, existingID)
		if err != nil {
			writeServer(w, "regulatory_source_register_failed")
			return
		}
		if err = tx.Commit(r.Context()); err != nil {
			writeServer(w, "regulatory_source_register_failed")
			return
		}
		httpx.JSON(w, 200, x)
		return
	}
	if !errors.Is(err, pgx.ErrNoRows) {
		writeServer(w, "regulatory_source_register_failed")
		return
	}
	var id string
	err = tx.QueryRow(r.Context(), `insert into school_regulatory_sources(tenant_code,institution_id,source_kind,citation,issuer,source_url,effective_from,effective_to,status,created_by_subject,updated_by_subject) values($1,$2,$3,$4,$5,$6,$7::date,$8::date,'draft',$9,$9) returning id::text`, t, i, in.SourceKind, in.Citation, in.Issuer, in.PublisherURL, in.ApplicableFrom, in.ApplicableUntil, a).Scan(&id)
	if err != nil {
		httpx.JSON(w, 422, map[string]any{"code": "regulatory_source_persist_failed"})
		return
	}
	_, err = tx.Exec(r.Context(), "insert into school_regulatory_source_idempotency(tenant_code,institution_id,actor_subject,action,idempotency_key,request_fingerprint,source_id) values($1,$2,$3,'register',$4,$5,$6::uuid)", t, i, a, key, requestFingerprint, id)
	if err != nil {
		writeServer(w, "regulatory_source_register_failed")
		return
	}
	x, err := loadSource(tx, r.Context(), t, i, id)
	if err != nil {
		writeServer(w, "regulatory_source_register_failed")
		return
	}
	if err = sourceAudit(r.Context(), tx, a, id, "registered"); err != nil {
		writeServer(w, "regulatory_source_audit_failed")
		return
	}
	if err = tx.Commit(r.Context()); err != nil {
		writeServer(w, "regulatory_source_register_failed")
		return
	}
	httpx.JSON(w, 201, x)
}

func (s *Service) Activate(w http.ResponseWriter, r *http.Request) {
	t, i, a, ok := sourceScope(r)
	if !ok {
		httpx.JSON(w, 401, map[string]any{"code": "institution_context_required"})
		return
	}
	id, ok := sourceID(w, r)
	if !ok {
		return
	}
	key := strings.TrimSpace(r.Header.Get("Idempotency-Key"))
	if key == "" {
		httpx.JSON(w, 422, map[string]any{"code": "idempotency_key_required"})
		return
	}
	var in ActivateRegulatorySourceRequest
	if !decode(w, r, &in) || in.ExpectedVersion < 1 || strings.TrimSpace(in.Assessment) == "" {
		httpx.JSON(w, 422, map[string]any{"code": "invalid_regulatory_source_activation"})
		return
	}
	if _, err := uuid.Parse(in.EvidenceID); err != nil {
		httpx.JSON(w, 422, map[string]any{"code": "activation_evidence_required"})
		return
	}
	tx, err := s.pool.Begin(r.Context())
	if err != nil {
		writeServer(w, "regulatory_source_activation_failed")
		return
	}
	defer tx.Rollback(r.Context())
	if allowed, permissionErr := actorHasPermission(r.Context(), tx, a, "school.regulatory_sources.approve"); permissionErr != nil || !allowed {
		httpx.JSON(w, http.StatusForbidden, map[string]any{"code": "regulatory_source_permission_required"})
		return
	}
	if _, err = tx.Exec(r.Context(), "select pg_advisory_xact_lock(hashtextextended($1,0))",
		fingerprint([]string{t, i, a, "activate", key})); err != nil {
		writeServer(w, "regulatory_source_command_lock_failed")
		return
	}
	fp := fingerprint(struct {
		SourceID string                          `json:"source_id"`
		Request  ActivateRegulatorySourceRequest `json:"request"`
	}{id, in})
	var replaySourceID, replayFingerprint string
	err = tx.QueryRow(r.Context(), "select source_id::text,request_fingerprint from school_regulatory_source_idempotency where tenant_code=$1 and institution_id=$2 and actor_subject=$3 and action='activate' and idempotency_key=$4 for update", t, i, a, key).Scan(&replaySourceID, &replayFingerprint)
	if err == nil {
		if replaySourceID != id || replayFingerprint != fp {
			httpx.JSON(w, 409, map[string]any{"code": "idempotency_key_payload_conflict"})
			return
		}
		x, e := loadSource(tx, r.Context(), t, i, id)
		if e != nil {
			writeServer(w, "regulatory_source_activation_failed")
			return
		}
		if e = tx.Commit(r.Context()); e != nil {
			writeServer(w, "regulatory_source_activation_failed")
			return
		}
		httpx.JSON(w, 200, x)
		return
	}
	if !errors.Is(err, pgx.ErrNoRows) {
		writeServer(w, "regulatory_source_activation_failed")
		return
	}
	var creator string
	err = tx.QueryRow(r.Context(), "select created_by_subject from school_regulatory_sources where tenant_code=$1 and institution_id=$2 and id=$3::uuid for update", t, i, id).Scan(&creator)
	if errors.Is(err, pgx.ErrNoRows) {
		httpx.JSON(w, 404, map[string]any{"code": "source_not_found"})
		return
	}
	if err != nil {
		writeServer(w, "regulatory_source_activation_failed")
		return
	}
	if creator == a {
		httpx.JSON(w, 409, map[string]any{"code": "activation_creator_conflict"})
		return
	}
	var exists bool
	err = tx.QueryRow(r.Context(), "select exists(select 1 from school_regulatory_source_evidence evidence join school_regulatory_sources source on source.tenant_code=evidence.tenant_code and source.institution_id=evidence.institution_id and source.id=evidence.source_id where evidence.tenant_code=$1 and evidence.institution_id=$2 and evidence.source_id=$3::uuid and evidence.id=$4::uuid and evidence.source_version=source.expected_version and evidence.requested_url=source.source_url and evidence.sha256=source.checksum_sha256)", t, i, id, in.EvidenceID).Scan(&exists)
	if err != nil || !exists {
		httpx.JSON(w, 422, map[string]any{"code": "activation_evidence_required"})
		return
	}
	tag, err := tx.Exec(r.Context(), `update school_regulatory_sources set status='active',expected_version=expected_version+1,updated_by_subject=$1,updated_at=now() where tenant_code=$2 and institution_id=$3 and id=$4::uuid and expected_version=$5 and status='verified'`, a, t, i, id, in.ExpectedVersion)
	if err != nil {
		writeServer(w, "regulatory_source_activation_failed")
		return
	}
	if tag.RowsAffected() == 0 {
		httpx.JSON(w, 409, map[string]any{"code": "source_activation_conflict"})
		return
	}
	_, err = tx.Exec(r.Context(), "insert into school_regulatory_source_activations(tenant_code,institution_id,source_id,evidence_id,assessment,activated_by_subject) values($1,$2,$3::uuid,$4::uuid,$5,$6)", t, i, id, in.EvidenceID, strings.TrimSpace(in.Assessment), a)
	if err != nil {
		writeServer(w, "regulatory_source_activation_failed")
		return
	}
	if _, err = tx.Exec(r.Context(), "insert into school_regulatory_source_idempotency(tenant_code,institution_id,actor_subject,action,idempotency_key,request_fingerprint,source_id,evidence_id) values($1,$2,$3,'activate',$4,$5,$6::uuid,$7::uuid)", t, i, a, key, fp, id, in.EvidenceID); err != nil {
		writeServer(w, "regulatory_source_activation_failed")
		return
	}
	x, err := loadSource(tx, r.Context(), t, i, id)
	if err != nil {
		writeServer(w, "regulatory_source_activation_failed")
		return
	}
	if err = sourceAudit(r.Context(), tx, a, id, "activated"); err != nil {
		writeServer(w, "regulatory_source_audit_failed")
		return
	}
	if err = tx.Commit(r.Context()); err != nil {
		writeServer(w, "regulatory_source_activation_failed")
		return
	}
	httpx.JSON(w, 200, x)
}

func (s *Service) DownloadEvidence(w http.ResponseWriter, r *http.Request) {
	t, i, _, ok := sourceScope(r)
	if !ok {
		httpx.JSON(w, 401, map[string]any{"code": "institution_context_required"})
		return
	}
	id, ok := sourceID(w, r)
	if !ok {
		return
	}
	eid := chi.URLParam(r, "evidenceID")
	if _, err := uuid.Parse(eid); err != nil {
		httpx.JSON(w, 422, map[string]any{"code": "invalid_evidence_id"})
		return
	}
	var content []byte
	var contentType string
	err := s.pool.QueryRow(r.Context(), "select content,content_type from school_regulatory_source_evidence where tenant_code=$1 and institution_id=$2 and source_id=$3::uuid and id=$4::uuid", t, i, id, eid).Scan(&content, &contentType)
	if errors.Is(err, pgx.ErrNoRows) {
		httpx.JSON(w, 404, map[string]any{"code": "evidence_not_found"})
		return
	}
	if err != nil {
		writeServer(w, "regulatory_source_evidence_download_failed")
		return
	}
	// Evidence is an attachment, never inline publisher-controlled content.
	// Retain the original type in storage for audit only; this endpoint has a
	// fixed binary contract.
	w.Header().Set("Content-Type", "application/octet-stream")
	w.Header().Set("Content-Disposition", "attachment; filename=regulatory-source-evidence-"+eid)
	w.Header().Set("X-Content-Type-Options", "nosniff")
	w.Header().Set("Content-Security-Policy", "sandbox")
	w.WriteHeader(200)
	_, _ = w.Write(content)
}

const sourceSelect = `select source.id::text,source.citation,source.source_url,source.issuer,source.source_kind,source.effective_from::text,source.effective_to::text,source.status,source.expected_version,source.created_by_subject,source.created_at,source.updated_at,e.id::text,e.sha256,e.retrieved_at,a.evidence_id::text,a.activated_by_subject,a.activated_at,a.assessment from school_regulatory_sources source left join lateral (select id,sha256,retrieved_at from school_regulatory_source_evidence where tenant_code=source.tenant_code and institution_id=source.institution_id and source_id=source.id order by retrieved_at desc,id desc limit 1) e on true left join lateral (select evidence_id,activated_by_subject,activated_at,assessment from school_regulatory_source_activations where tenant_code=source.tenant_code and institution_id=source.institution_id and source_id=source.id order by activated_at desc,id desc limit 1) a on true`

type rowScanner interface{ Scan(...any) error }

func scanSource(row rowScanner) (RegulatorySource, error) {
	var x RegulatorySource
	var from, until *string
	err := row.Scan(&x.ID, &x.Citation, &x.PublisherURL, &x.Issuer, &x.SourceKind, &from, &until, &x.Status, &x.ExpectedVersion, &x.CreatedBySubject, &x.CreatedAt, &x.UpdatedAt, &x.LatestEvidenceID, &x.LatestEvidenceSHA256, &x.LatestEvidenceRetrievedAt, &x.ActivationEvidenceID, &x.ActivatedBySubject, &x.ActivatedAt, &x.Assessment)
	x.ApplicableFrom = from
	x.ApplicableUntil = until
	return x, err
}

type sourceQuery interface {
	QueryRow(context.Context, string, ...any) pgx.Row
}

func loadSource(q sourceQuery, ctx context.Context, t, i, id string) (RegulatorySource, error) {
	return scanSource(q.QueryRow(ctx, sourceSelect+" where source.tenant_code=$1 and source.institution_id=$2 and source.id=$3::uuid", t, i, id))
}
func decode(w http.ResponseWriter, r *http.Request, out any) bool {
	d := json.NewDecoder(http.MaxBytesReader(w, r.Body, 64<<10))
	d.DisallowUnknownFields()
	return d.Decode(out) == nil && d.Decode(new(any)) == io.EOF
}
func validRegister(in *RegisterRegulatorySourceRequest) bool {
	in.Citation = strings.TrimSpace(in.Citation)
	in.PublisherURL = strings.TrimSpace(in.PublisherURL)
	in.Issuer = strings.TrimSpace(in.Issuer)
	in.SourceKind = strings.TrimSpace(in.SourceKind)
	if in.Citation == "" || in.PublisherURL == "" || in.Issuer == "" {
		return false
	}
	switch in.SourceKind {
	case "law", "government_decision", "ministerial_order", "authorization", "accreditation", "founder_decision", "contract", "other":
	default:
		return false
	}
	u, err := url.Parse(in.PublisherURL)
	if err != nil {
		return false
	}
	invalidURL := u.Scheme != "https" || u.User != nil || u.Fragment != "" || u.Opaque != ""
	invalidPort := u.Port() != "" && u.Port() != "443"
	if invalidURL || invalidPort || !validHostname(strings.ToLower(u.Hostname())) {
		return false
	}
	from, err := time.Parse(time.DateOnly, in.ApplicableFrom)
	if err != nil {
		return false
	}
	if in.ApplicableUntil != nil {
		until, err := time.Parse(time.DateOnly, *in.ApplicableUntil)
		if err != nil || until.Before(from) {
			return false
		}
	}
	return true
}
func rejectQueryURL(raw string) error {
	u, e := url.Parse(raw)
	if e != nil || u.RawQuery != "" || u.ForceQuery {
		return fmt.Errorf("query unsupported")
	}
	return nil
}
func sourceID(w http.ResponseWriter, r *http.Request) (string, bool) {
	id := strings.TrimSpace(chi.URLParam(r, "sourceID"))
	if _, e := uuid.Parse(id); e != nil {
		httpx.JSON(w, 422, map[string]any{"code": "invalid_regulatory_source_id"})
		return "", false
	}
	return id, true
}
func fingerprint(v any) string {
	b, _ := json.Marshal(v)
	h := sha256.Sum256(b)
	return hex.EncodeToString(h[:])
}
func strconv(v int) string { return fmt.Sprintf("%d", v) }
func writeServer(w http.ResponseWriter, code string) {
	httpx.JSON(w, 500, map[string]any{"code": code})
}
