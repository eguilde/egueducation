package admission

import (
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/eguilde/egueducation/internal/httpx"
)

var archivePurposePermission = map[string]string{
	"application_document": permissionManage,
	"decision":             permissionDecide,
	"appeal":               permissionAppeals,
	"appeal_submission":    permissionAppeals,
	"appeal_resolution":    permissionAppeals,
}

func (s *Service) ListRegulatorySources(w http.ResponseWriter, r *http.Request) {
	sc, ok := requestScope(r)
	if !ok {
		writeError(w, errBadScope, "regulatory_source_list_failed")
		return
	}
	tx, err := s.begin(r.Context(), sc)
	if err != nil {
		writeError(w, err, "regulatory_source_list_failed")
		return
	}
	defer tx.Rollback(r.Context())
	if err = requirePermission(r.Context(), tx, sc, permissionRead); err != nil {
		writeError(w, err, "regulatory_source_list_failed")
		return
	}
	q := httpx.ParsePageQuery(r.URL.Query(), map[string]struct{}{"citation": {}, "source_kind": {}, "effective_from": {}}, []string{"citation", "source_kind"})
	sort := map[string]string{"citation": "citation", "source_kind": "source_kind", "effective_from": "effective_from"}[q.Sort]
	if sort == "" {
		sort = "citation"
	}
	where := " where tenant_code=$1 and institution_id=$2 and status='active'"
	args := []any{sc.tenant, sc.institution}
	for _, f := range []string{"citation", "source_kind"} {
		if v := q.Filters[f]; v != "" {
			args = append(args, v)
			if f == "citation" {
				where += fmt.Sprintf(" and citation ilike '%%'||$%d||'%%'", len(args))
			} else {
				where += fmt.Sprintf(" and source_kind=$%d", len(args))
			}
		}
	}
	var total int
	if err = tx.QueryRow(r.Context(), "select count(*) from school_regulatory_sources"+where, args...).Scan(&total); err != nil {
		writeError(w, err, "regulatory_source_list_failed")
		return
	}
	args = append(args, q.PageSize, (q.Page-1)*q.PageSize)
	rows, err := tx.Query(r.Context(), "select id::text,source_kind,citation,article_reference,issuer,effective_from,effective_to from school_regulatory_sources"+where+fmt.Sprintf(" order by %s %s nulls last,id limit $%d offset $%d", sort, q.Direction, len(args)-1, len(args)), args...)
	if err != nil {
		writeError(w, err, "regulatory_source_list_failed")
		return
	}
	defer rows.Close()
	items := []RegulatorySourceOption{}
	for rows.Next() {
		var x RegulatorySourceOption
		var from, to *time.Time
		if err = rows.Scan(&x.ID, &x.SourceKind, &x.Citation, &x.ArticleReference, &x.Issuer, &from, &to); err != nil {
			writeError(w, err, "regulatory_source_list_failed")
			return
		}
		if from != nil {
			v := from.Format(time.DateOnly)
			x.EffectiveFrom = &v
		}
		if to != nil {
			v := to.Format(time.DateOnly)
			x.EffectiveTo = &v
		}
		items = append(items, x)
	}
	if err = rows.Err(); err != nil {
		writeError(w, err, "regulatory_source_list_failed")
		return
	}
	if err = tx.Commit(r.Context()); err != nil {
		writeError(w, err, "regulatory_source_list_failed")
		return
	}
	httpx.WritePage(w, 200, items, total, q.Page, q.PageSize)
}

// Party names are returned only when the actor independently holds the
// Registratura party-read permission, in addition to admission read access.
func (s *Service) ListCandidateParties(w http.ResponseWriter, r *http.Request) {
	sc, ok := requestScope(r)
	if !ok {
		writeError(w, errBadScope, "candidate_party_list_failed")
		return
	}
	tx, err := s.begin(r.Context(), sc)
	if err != nil {
		writeError(w, err, "candidate_party_list_failed")
		return
	}
	defer tx.Rollback(r.Context())
	if err = requirePermission(r.Context(), tx, sc, permissionRead); err == nil {
		err = requirePermission(r.Context(), tx, sc, "registratura.read")
	}
	if err != nil {
		writeError(w, err, "candidate_party_list_failed")
		return
	}
	q := httpx.ParsePageQuery(r.URL.Query(), map[string]struct{}{"display_name": {}, "code": {}}, []string{"display_name", "code"})
	sort := map[string]string{"display_name": "display_name", "code": "code"}[q.Sort]
	if sort == "" {
		sort = "display_name"
	}
	where := " where tenant_code=$1 and institution_id=$2 and active and party_type='physical'"
	args := []any{sc.tenant, sc.institution}
	for _, f := range []string{"display_name", "code"} {
		if v := q.Filters[f]; v != "" {
			args = append(args, v)
			where += fmt.Sprintf(" and %s ilike '%%'||$%d||'%%'", f, len(args))
		}
	}
	var total int
	if err = tx.QueryRow(r.Context(), "select count(*) from app_parties"+where, args...).Scan(&total); err != nil {
		writeError(w, err, "candidate_party_list_failed")
		return
	}
	args = append(args, q.PageSize, (q.Page-1)*q.PageSize)
	rows, err := tx.Query(r.Context(), "select id::text,code,display_name from app_parties"+where+fmt.Sprintf(" order by %s %s,id limit $%d offset $%d", sort, q.Direction, len(args)-1, len(args)), args...)
	if err != nil {
		writeError(w, err, "candidate_party_list_failed")
		return
	}
	defer rows.Close()
	items := []CandidatePartyOption{}
	for rows.Next() {
		var x CandidatePartyOption
		if err = rows.Scan(&x.ID, &x.Code, &x.DisplayName); err != nil {
			writeError(w, err, "candidate_party_list_failed")
			return
		}
		items = append(items, x)
	}
	if err = rows.Err(); err != nil {
		writeError(w, err, "candidate_party_list_failed")
		return
	}
	if err = tx.Commit(r.Context()); err != nil {
		writeError(w, err, "candidate_party_list_failed")
		return
	}
	httpx.WritePage(w, 200, items, total, q.Page, q.PageSize)
}

// Pupil names require the unrestricted institution roster permission; assigned
// teachers cannot use this admission-wide selector.
func (s *Service) ListStudents(w http.ResponseWriter, r *http.Request) {
	sc, ok := requestScope(r)
	if !ok {
		writeError(w, errBadScope, "student_list_failed")
		return
	}
	tx, err := s.begin(r.Context(), sc)
	if err != nil {
		writeError(w, err, "student_list_failed")
		return
	}
	defer tx.Rollback(r.Context())
	if err = requirePermission(r.Context(), tx, sc, permissionRead); err == nil {
		err = requirePermission(r.Context(), tx, sc, "education.classes.read")
	}
	if err != nil {
		writeError(w, err, "student_list_failed")
		return
	}
	q := httpx.ParsePageQuery(r.URL.Query(), map[string]struct{}{"student_code": {}, "last_name": {}, "status": {}}, []string{"student_code", "first_name", "last_name", "status"})
	sort := map[string]string{"student_code": "student_code", "last_name": "last_name", "status": "status"}[q.Sort]
	if sort == "" {
		sort = "last_name"
	}
	where := " where tenant_code=$1 and institution_id=$2"
	args := []any{sc.tenant, sc.institution}
	for _, f := range []string{"student_code", "first_name", "last_name", "status"} {
		if v := q.Filters[f]; v != "" {
			args = append(args, v)
			if f == "status" {
				where += fmt.Sprintf(" and status=$%d", len(args))
			} else {
				where += fmt.Sprintf(" and %s ilike '%%'||$%d||'%%'", f, len(args))
			}
		}
	}
	var total int
	if err = tx.QueryRow(r.Context(), "select count(*) from education_students"+where, args...).Scan(&total); err != nil {
		writeError(w, err, "student_list_failed")
		return
	}
	args = append(args, q.PageSize, (q.Page-1)*q.PageSize)
	rows, err := tx.Query(r.Context(), "select id::text,party_id::text,student_code,first_name,last_name,status from education_students"+where+fmt.Sprintf(" order by %s %s,id limit $%d offset $%d", sort, q.Direction, len(args)-1, len(args)), args...)
	if err != nil {
		writeError(w, err, "student_list_failed")
		return
	}
	defer rows.Close()
	items := []StudentOption{}
	for rows.Next() {
		var x StudentOption
		if err = rows.Scan(&x.ID, &x.PartyID, &x.StudentCode, &x.FirstName, &x.LastName, &x.Status); err != nil {
			writeError(w, err, "student_list_failed")
			return
		}
		items = append(items, x)
	}
	if err = rows.Err(); err != nil {
		writeError(w, err, "student_list_failed")
		return
	}
	if err = tx.Commit(r.Context()); err != nil {
		writeError(w, err, "student_list_failed")
		return
	}
	httpx.WritePage(w, 200, items, total, q.Page, q.PageSize)
}

// Archive choices require independent archive-read authority. The explicit
// tenant join prevents an institution-only legacy archive row being selected
// unless the current canonical tenant owns that institution.
func (s *Service) ListEligibleArchiveVersions(w http.ResponseWriter, r *http.Request) {
	sc, ok := requestScope(r)
	if !ok {
		writeError(w, errBadScope, "archive_version_list_failed")
		return
	}
	purpose := strings.TrimSpace(r.URL.Query().Get("purpose"))
	purposePermission, allowed := archivePurposePermission[purpose]
	if !allowed {
		httpx.JSON(w, 422, map[string]any{"code": "invalid_archive_purpose"})
		return
	}
	tx, err := s.begin(r.Context(), sc)
	if err != nil {
		writeError(w, err, "archive_version_list_failed")
		return
	}
	defer tx.Rollback(r.Context())
	if err = requirePermission(r.Context(), tx, sc, permissionRead); err == nil {
		err = requirePermission(r.Context(), tx, sc, purposePermission)
	}
	if err == nil {
		err = requirePermission(r.Context(), tx, sc, "earchiva.read")
	}
	if err != nil {
		writeError(w, err, "archive_version_list_failed")
		return
	}
	q := httpx.ParsePageQuery(r.URL.Query(), map[string]struct{}{"title": {}, "version_no": {}}, nil)
	sort := map[string]string{"title": "d.title", "version_no": "v.version_no"}[q.Sort]
	if sort == "" {
		sort = "d.title"
	}
	search := strings.TrimSpace(r.URL.Query().Get("q"))
	base := " from app_tenants tenant_scope join archive_documents d on d.institution_id=tenant_scope.institution_id join archive_document_versions v on v.institution_id=d.institution_id and v.document_id=d.id"
	where := " where tenant_scope.code=$1 and tenant_scope.institution_id=$2 and d.status='ready' and v.status='active' and v.source_object_version_id<>'' and v.retention_until is not null and v.source_sha256~'^[0-9a-f]{64}$'"
	args := []any{sc.tenant, sc.institution}
	if search != "" {
		args = append(args, search)
		where += fmt.Sprintf(" and (d.title ilike '%%'||$%d||'%%' or d.original_file_name ilike '%%'||$%d||'%%')", len(args), len(args))
	}
	for _, field := range []string{"title", "version_no"} {
		if value := strings.TrimSpace(q.Filters[field]); value != "" {
			args = append(args, value)
			if field == "title" {
				where += fmt.Sprintf(" and d.title ilike '%%'||$%d||'%%'", len(args))
			} else {
				where += fmt.Sprintf(" and v.version_no::text=$%d", len(args))
			}
		}
	}
	var total int
	if err = tx.QueryRow(r.Context(), "select count(*)"+base+where, args...).Scan(&total); err != nil {
		writeError(w, err, "archive_version_list_failed")
		return
	}
	args = append(args, q.PageSize, (q.Page-1)*q.PageSize)
	rows, err := tx.Query(r.Context(), "select d.id::text,v.id::text,v.version_no,d.title,d.original_file_name,v.mime_type,lower(v.source_sha256),v.retention_until"+base+where+fmt.Sprintf(" order by %s %s,v.id limit $%d offset $%d", sort, q.Direction, len(args)-1, len(args)), args...)
	if err != nil {
		writeError(w, err, "archive_version_list_failed")
		return
	}
	defer rows.Close()
	items := []ArchiveVersionOption{}
	for rows.Next() {
		var x ArchiveVersionOption
		var retention time.Time
		if err = rows.Scan(&x.DocumentID, &x.VersionID, &x.VersionNo, &x.Title, &x.OriginalFileName, &x.MIMEType, &x.SHA256, &retention); err != nil {
			writeError(w, err, "archive_version_list_failed")
			return
		}
		x.RetentionUntil = retention.Format(time.RFC3339Nano)
		items = append(items, x)
	}
	if err = rows.Err(); err != nil {
		writeError(w, err, "archive_version_list_failed")
		return
	}
	if err = tx.Commit(r.Context()); err != nil {
		writeError(w, err, "archive_version_list_failed")
		return
	}
	httpx.WritePage(w, 200, items, total, q.Page, q.PageSize)
}
