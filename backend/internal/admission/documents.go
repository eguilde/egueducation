package admission

import (
	"net/http"
	"strings"

	"github.com/eguilde/egueducation/internal/httpx"
	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
)

func finalApplicationDocumentReviewStatus(status string) bool {
	switch status {
	case "accepted", "rejected", "waived":
		return true
	default:
		return false
	}
}

// ReviewApplicationDocument appends archive evidence from the seeded
// requirement row. Archive provenance is immutable in 0148, so it is never
// attached by mutating a prior evidentiary record.
func (s *Service) ReviewApplicationDocument(w http.ResponseWriter, r *http.Request) {
	sc, ok := requestScope(r)
	if !ok {
		writeError(w, errBadScope, "application_document_failed")
		return
	}
	applicationID, sourceID := chi.URLParam(r, "applicationID"), chi.URLParam(r, "documentID")
	if !validUUID(applicationID) || !validUUID(sourceID) {
		httpx.JSON(w, 422, map[string]any{"code": "invalid_application_document_id"})
		return
	}
	key, ok := idempotencyKey(r)
	if !ok {
		httpx.JSON(w, 400, map[string]any{"code": "idempotency_key_required"})
		return
	}
	var in ReviewApplicationDocumentRequest
	if decode(w, r, &in) != nil {
		httpx.JSON(w, 400, map[string]any{"code": "invalid_application_document"})
		return
	}
	in.Status = strings.TrimSpace(in.Status)
	if in.ExpectedVersion < 1 || !finalApplicationDocumentReviewStatus(in.Status) || (in.Status != "waived" && (in.Archive == nil || !validUUID(in.Archive.DocumentID) || !validUUID(in.Archive.VersionID))) {
		httpx.JSON(w, 422, map[string]any{"code": "invalid_application_document"})
		return
	}
	tx, err := s.begin(r.Context(), sc)
	if err != nil {
		writeError(w, err, "application_document_failed")
		return
	}
	defer tx.Rollback(r.Context())
	if err = requirePermission(r.Context(), tx, sc, permissionManage); err != nil {
		writeError(w, err, "application_document_failed")
		return
	}
	if err = requirePermission(r.Context(), tx, sc, "earchiva.read"); err != nil {
		writeError(w, err, "application_document_failed")
		return
	}
	proposedID := sourceID
	if in.Status != "waived" {
		proposedID = uuid.NewString()
	}
	evidenceID, replay, err := reserve(r.Context(), tx, sc, "admission.application.document.review", key, fingerprint(struct {
		ApplicationID, SourceID string
		Input                   ReviewApplicationDocumentRequest
	}{applicationID, sourceID, in}), proposedID)
	if err != nil {
		writeError(w, err, "application_document_failed")
		return
	}
	if !replay {
		var editable bool
		var allowedMIMETypes []string
		err = tx.QueryRow(r.Context(), `select a.status not in ('admitted','rejected','withdrawn','cancelled') and not exists(select 1 from school_admission_decisions decision where decision.tenant_code=a.tenant_code and decision.institution_id=a.institution_id and decision.application_id=a.id),coalesce(requirement.allowed_mime_types,'{}'::text[]) from school_admission_application_documents document join school_admission_applications a on a.tenant_code=document.tenant_code and a.institution_id=document.institution_id and a.id=document.application_id left join school_admission_document_requirements requirement on requirement.tenant_code=document.tenant_code and requirement.institution_id=document.institution_id and requirement.id=document.document_requirement_id where document.tenant_code=$1 and document.institution_id=$2 and document.id=$3::uuid and document.application_id=$4::uuid and document.archive_document_id is null and document.reviewed_at is null for update of document,a`, sc.tenant, sc.institution, sourceID, applicationID).Scan(&editable, &allowedMIMETypes)
		if err == nil && !editable {
			err = errInvalidState
		}
		if in.Status == "waived" {
			if err != nil {
				writeError(w, err, "application_document_failed")
				return
			}
			tag, e := tx.Exec(r.Context(), `update school_admission_application_documents set status='waived',reviewed_at=now(),reviewed_by_subject=$1,review_note=$2,expected_version=expected_version+1,updated_by_subject=$1,updated_at=now() where tenant_code=$3 and institution_id=$4 and id=$5::uuid and application_id=$6::uuid and expected_version=$7 and archive_document_id is null and reviewed_at is null`, sc.actor, strings.TrimSpace(in.ReviewNote), sc.tenant, sc.institution, sourceID, applicationID, in.ExpectedVersion)
			err = e
			if err == nil && tag.RowsAffected() != 1 {
				err = errVersionConflict
			}
		} else {
			var a archiveSnapshot
			a, err = loadArchiveSnapshot(r.Context(), tx, sc.institution, *in.Archive)
			if err == nil && len(allowedMIMETypes) > 0 && !mimeAllowed(allowedMIMETypes, a.mime) {
				err = errInvalidInput
			}
			if err == nil {
				tag, e := tx.Exec(r.Context(), `insert into school_admission_application_documents(id,tenant_code,institution_id,application_id,document_requirement_id,document_kind,status,archive_document_id,archive_version_id,archive_version_no,archive_source_bucket,archive_source_object_key,archive_source_object_version_id,archive_retention_until,archive_sha256,reviewed_at,reviewed_by_subject,review_note,created_by_subject,updated_by_subject)
					select $1::uuid,d.tenant_code,d.institution_id,d.application_id,d.document_requirement_id,d.document_kind,$2,$3::uuid,$4::uuid,$5,$6,$7,$8,$9,$10,now(),$11,$12,$11,$11 from school_admission_application_documents d where d.tenant_code=$13 and d.institution_id=$14 and d.id=$15::uuid and d.application_id=$16::uuid and d.expected_version=$17`, evidenceID, in.Status, in.Archive.DocumentID, in.Archive.VersionID, a.versionNo, a.bucket, a.objectKey, a.objectVersion, a.retention, a.sha, sc.actor, strings.TrimSpace(in.ReviewNote), sc.tenant, sc.institution, sourceID, applicationID, in.ExpectedVersion)
				err = e
				if err == nil && tag.RowsAffected() != 1 {
					err = errVersionConflict
				}
				if err == nil {
					tag, err = tx.Exec(r.Context(), `update school_admission_application_documents set reviewed_at=now(),reviewed_by_subject=$1,review_note=$2,expected_version=expected_version+1,updated_by_subject=$1,updated_at=now() where tenant_code=$3 and institution_id=$4 and id=$5::uuid and application_id=$6::uuid and expected_version=$7 and archive_document_id is null and reviewed_at is null`, sc.actor, strings.TrimSpace(in.ReviewNote), sc.tenant, sc.institution, sourceID, applicationID, in.ExpectedVersion)
					if err == nil && tag.RowsAffected() != 1 {
						err = errVersionConflict
					}
				}
			}
		}
		if err == nil {
			err = outbox(r.Context(), tx, sc, "admission_application", applicationID, "admission.application.document."+in.Status, map[string]any{"application_id": applicationID, "document_id": evidenceID, "source_requirement_record_id": sourceID, "status": in.Status})
		}
		if err == nil {
			err = auditEvent(r.Context(), tx, sc, "admission.application.document."+in.Status, "school_admission_application_document", evidenceID)
		}
	}
	if err != nil {
		writeError(w, err, "application_document_failed")
		return
	}
	if err = tx.Commit(r.Context()); err != nil {
		writeError(w, err, "application_document_failed")
		return
	}
	httpx.JSON(w, 200, CommandResult{ID: evidenceID, Status: in.Status, ExpectedVersion: in.ExpectedVersion + 1, Replayed: replay})
}
