package earchiva

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/aws/smithy-go"
	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"

	"github.com/eguilde/egueducation/internal/audit"
	authruntime "github.com/eguilde/egueducation/internal/auth"
	"github.com/eguilde/egueducation/internal/httpx"
)

// AdmissionLegalPreparationArtifactResponse is the server-owned resource used
// by the preparation screen to recover a lost response and poll OCR readiness.
// RetentionUntil is derived from the immutable preparation snapshot.
type AdmissionLegalPreparationArtifactResponse struct {
	IntentID       string                 `json:"intent_id"`
	PreparationID  string                 `json:"preparation_id"`
	ArtifactSlot   string                 `json:"artifact_slot"`
	Document       ArchiveDocument        `json:"document"`
	Version        ArchiveDocumentVersion `json:"version"`
	RetentionUntil string                 `json:"retention_until"`
	Replayed       bool                   `json:"replayed"`
}

type admissionWORMIntent struct {
	ID, Tenant, Institution, Actor, Fingerprint, PreparationID, Slot string
	SHA256, DocumentID, VersionID, Bucket, Key                       string
	StoredVersionID                                                  string
	Size                                                             int64
	Retention                                                        time.Time
}

func (s *DocumentService) UploadAdmissionLegalPreparationArtifact(w http.ResponseWriter, r *http.Request) {
	if s.storage == nil || !s.storage.Enabled() {
		httpx.JSON(w, http.StatusServiceUnavailable, map[string]any{"code": "archive_storage_unavailable"})
		return
	}
	preparationID, slot := strings.TrimSpace(chi.URLParam(r, "preparationID")), strings.TrimSpace(chi.URLParam(r, "artifactSlot"))
	if _, err := uuid.Parse(preparationID); err != nil || !validAdmissionArtifactSlot(slot) {
		httpx.JSON(w, http.StatusUnprocessableEntity, map[string]any{"code": "invalid_admission_artifact"})
		return
	}
	key := strings.TrimSpace(r.Header.Get("Idempotency-Key"))
	if key == "" || len(key) > 200 {
		httpx.JSON(w, http.StatusBadRequest, map[string]any{"code": "idempotency_key_required"})
		return
	}
	select {
	case archiveUploadSlots <- struct{}{}:
		defer func() { <-archiveUploadSlots }()
	default:
		httpx.JSON(w, http.StatusTooManyRequests, map[string]any{"code": "archive_upload_busy"})
		return
	}
	r.Body = http.MaxBytesReader(w, r.Body, archiveUploadMaxBytes+archiveUploadOverhead)
	if err := r.ParseMultipartForm(8 << 20); err != nil {
		httpx.JSON(w, http.StatusBadRequest, map[string]any{"code": "invalid_admission_artifact"})
		return
	}
	defer r.MultipartForm.RemoveAll() //nolint:errcheck
	if len(r.MultipartForm.Value) != 0 || len(r.MultipartForm.File) != 1 || len(r.MultipartForm.File["file"]) != 1 {
		httpx.JSON(w, http.StatusBadRequest, map[string]any{"code": "invalid_admission_artifact"})
		return
	}
	file, header, err := r.FormFile("file")
	if err != nil {
		httpx.JSON(w, http.StatusBadRequest, map[string]any{"code": "missing_archive_file"})
		return
	}
	defer file.Close() //nolint:errcheck
	staged, err := stageAndValidateArchivePDF(r.Context(), file, s.scanner)
	if err != nil {
		httpx.JSON(w, http.StatusUnprocessableEntity, map[string]any{"code": "invalid_archive_file", "message": err.Error()})
		return
	}
	defer os.Remove(staged.Path)
	intent, replay, err := s.reserveAdmissionWORMIntent(r.Context(), r, preparationID, slot, key, staged.SHA256, staged.Size)
	if err != nil {
		writeAdmissionArtifactError(w, err)
		return
	}
	if replay {
		response, err := s.loadAdmissionArtifact(r.Context(), intent.Tenant, intent.Institution, intent.Actor, preparationID, slot)
		if err == nil {
			response.Replayed = true
			httpx.JSON(w, http.StatusOK, response)
			return
		}
		if !errors.Is(err, pgx.ErrNoRows) {
			writeAdmissionArtifactError(w, err)
			return
		}
		metadata := admissionIntentMetadata(intent)
		if object, reconcileErr := s.storage.ReconcileImmutableObject(r.Context(), ImmutableArchiveRecovery{Key: intent.Key, VersionID: intent.StoredVersionID, ExpectedSHA256: intent.SHA256, ContentLength: intent.Size, RetentionUntil: intent.Retention, LegalHold: true, Metadata: metadata}); reconcileErr == nil {
			if err = s.adoptAdmissionWORMIntent(r.Context(), r, intent, object, header.Filename, staged.PageCount); err != nil {
				writeAdmissionArtifactError(w, err)
				return
			}
			response, err = s.loadAdmissionArtifact(r.Context(), intent.Tenant, intent.Institution, intent.Actor, preparationID, slot)
			if err != nil {
				writeAdmissionArtifactError(w, err)
				return
			}
			response.Replayed = true
			httpx.JSON(w, http.StatusOK, response)
			return
		} else if !immutableObjectMissing(reconcileErr) {
			httpx.JSON(w, http.StatusBadGateway, map[string]any{"code": "admission_artifact_storage_pending"})
			return
		}
	}
	f, err := os.Open(staged.Path)
	if err != nil {
		httpx.JSON(w, http.StatusInternalServerError, map[string]any{"code": "archive_upload_staging_failed"})
		return
	}
	defer f.Close() //nolint:errcheck
	metadata := admissionIntentMetadata(intent)
	object, putErr := s.storage.PutImmutableObject(r.Context(), ImmutableArchiveWrite{
		Key: intent.Key, ContentType: "application/pdf", Body: f, ContentLength: intent.Size,
		RetentionUntil: intent.Retention, LegalHold: true, Metadata: metadata,
	})
	if putErr != nil {
		// A verification error can still name an accepted WORM version; reconcile
		// its exact bytes before any DB adoption. Transport failures are retained.
		var verification *ImmutableArchiveWriteVerificationError
		if !errors.As(putErr, &verification) {
			httpx.JSON(w, http.StatusBadGateway, map[string]any{"code": "admission_artifact_storage_pending"})
			return
		}
		object = verification.Object
	}
	object, err = s.storage.ReconcileImmutableObject(r.Context(), ImmutableArchiveRecovery{
		Key: intent.Key, VersionID: object.VersionID, ExpectedSHA256: intent.SHA256,
		ContentLength: intent.Size, RetentionUntil: intent.Retention, LegalHold: true, Metadata: metadata,
	})
	if err != nil {
		httpx.JSON(w, http.StatusBadGateway, map[string]any{"code": "admission_artifact_storage_pending"})
		return
	}
	if err = s.adoptAdmissionWORMIntent(r.Context(), r, intent, object, header.Filename, staged.PageCount); err != nil {
		writeAdmissionArtifactError(w, err)
		return
	}
	response, err := s.loadAdmissionArtifact(r.Context(), intent.Tenant, intent.Institution, intent.Actor, preparationID, slot)
	if err != nil {
		writeAdmissionArtifactError(w, err)
		return
	}
	httpx.JSON(w, http.StatusCreated, response)
}

// Only an S3-confirmed absence may lead to a conditional new PUT. A timeout,
// authorization error, or malformed recovery observation stays recoverable and
// must never be converted into an overwrite attempt.
func immutableObjectMissing(err error) bool {
	var apiErr smithy.APIError
	if !errors.As(err, &apiErr) {
		return false
	}
	switch strings.ToLower(apiErr.ErrorCode()) {
	case "nosuchkey", "nosuchversion", "notfound", "404":
		return true
	default:
		return false
	}
}

func (s *DocumentService) GetAdmissionLegalPreparationArtifact(w http.ResponseWriter, r *http.Request) {
	preparationID, slot := strings.TrimSpace(chi.URLParam(r, "preparationID")), strings.TrimSpace(chi.URLParam(r, "artifactSlot"))
	if _, err := uuid.Parse(preparationID); err != nil || !validAdmissionArtifactSlot(slot) {
		httpx.JSON(w, http.StatusUnprocessableEntity, map[string]any{"code": "invalid_admission_artifact"})
		return
	}
	if err := s.authorizeAdmissionArtifactRead(r.Context(), r, preparationID); err != nil {
		writeAdmissionArtifactError(w, err)
		return
	}
	response, err := s.loadAdmissionArtifact(r.Context(), authruntime.CurrentTenantCodeFromRequest(r), authruntime.CurrentInstitutionIDFromRequest(r), authruntime.CurrentSubjectFromRequest(r), preparationID, slot)
	if errors.Is(err, pgx.ErrNoRows) {
		httpx.JSON(w, http.StatusNotFound, map[string]any{"code": "admission_artifact_not_uploaded"})
		return
	}
	writeAdmissionArtifactError(w, err)
	if err == nil {
		httpx.JSON(w, http.StatusOK, response)
	}
}

func (s *DocumentService) authorizeAdmissionArtifactRead(ctx context.Context, r *http.Request, preparationID string) error {
	tenant, institution, actor := strings.TrimSpace(authruntime.CurrentTenantCodeFromRequest(r)), strings.TrimSpace(authruntime.CurrentInstitutionIDFromRequest(r)), strings.TrimSpace(authruntime.CurrentSubjectFromRequest(r))
	if tenant == "" || institution == "" || actor == "" {
		return errAdmissionArtifactScope
	}
	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return err
	}
	defer tx.Rollback(ctx) //nolint:errcheck
	var kind, preparer string
	if err = tx.QueryRow(ctx, `select artifact_kind,prepared_by_subject from school_admission_legal_preparations where tenant_code=$1 and institution_id=$2 and id=$3::uuid`, tenant, institution, preparationID).Scan(&kind, &preparer); err != nil {
		return err
	}
	if preparer != actor {
		return errAdmissionArtifactForbidden
	}
	permission := "education.admissions.decide"
	if kind == "admission_appeal_resolution" {
		permission = "education.admissions.appeals.manage"
	}
	if err = archiveAdmissionActorPermission(ctx, tx, actor, permission); err != nil {
		return err
	}
	return tx.Commit(ctx)
}

func validAdmissionArtifactSlot(slot string) bool {
	return slot == "primary" || slot == "resulting_decision"
}
func validSHA256(value string) bool {
	_, err := hex.DecodeString(value)
	return err == nil && len(value) == 64
}

func admissionArtifactFingerprint(preparationID, slot, sha string, size int64) string {
	sum := sha256.Sum256([]byte(strings.Join([]string{preparationID, slot, strings.ToLower(sha), fmt.Sprintf("%d", size)}, "\n")))
	return hex.EncodeToString(sum[:])
}

func admissionIntentMetadata(i admissionWORMIntent) map[string]string {
	return map[string]string{"intent-id": i.ID, "tenant-code": i.Tenant, "institution-id": i.Institution, "preparation-id": i.PreparationID, "artifact-slot": i.Slot, "sha256": i.SHA256}
}

func (s *DocumentService) reserveAdmissionWORMIntent(ctx context.Context, r *http.Request, preparationID, slot, idempotency, sha string, size int64) (admissionWORMIntent, bool, error) {
	var out admissionWORMIntent
	out.Tenant, out.Institution, out.Actor = strings.TrimSpace(authruntime.CurrentTenantCodeFromRequest(r)), strings.TrimSpace(authruntime.CurrentInstitutionIDFromRequest(r)), strings.TrimSpace(authruntime.CurrentSubjectFromRequest(r))
	if out.Tenant == "" || out.Institution == "" || out.Actor == "" {
		return out, false, errAdmissionArtifactScope
	}
	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return out, false, err
	}
	defer tx.Rollback(ctx) //nolint:errcheck
	var status, preparer, kind string
	err = tx.QueryRow(ctx, `select status,prepared_by_subject,artifact_kind,required_retention_until from school_admission_legal_preparations where tenant_code=$1 and institution_id=$2 and id=$3::uuid for update`, out.Tenant, out.Institution, preparationID).Scan(&status, &preparer, &kind, &out.Retention)
	if err != nil {
		return out, false, err
	}
	if status != "prepared" || preparer != out.Actor || !out.Retention.After(time.Now().UTC()) {
		return out, false, errAdmissionArtifactState
	}
	permission := "education.admissions.decide"
	if kind == "admission_appeal_resolution" {
		permission = "education.admissions.appeals.manage"
	}
	if err = archiveAdmissionActorPermission(ctx, tx, out.Actor, permission); err != nil {
		return out, false, err
	}
	if slot == "resulting_decision" && kind != "admission_appeal_resolution" {
		return out, false, errAdmissionArtifactState
	}
	fingerprint := admissionArtifactFingerprint(preparationID, slot, sha, size)
	var existingFingerprint string
	err = tx.QueryRow(ctx, `select request_fingerprint from archive_ingestion_intents where tenant_code=$1 and institution_id=$2 and actor_subject=$3 and idempotency_key=$4 for update`, out.Tenant, out.Institution, out.Actor, idempotency).Scan(&existingFingerprint)
	if err == nil {
		if existingFingerprint != fingerprint {
			return out, false, errAdmissionArtifactConflict
		}
		if err = tx.QueryRow(ctx, `select id::text,preparation_id::text,artifact_slot,expected_sha256,expected_size_bytes,reserved_document_id::text,reserved_version_id::text,bucket_name,object_key,retention_until,stored_version_id from archive_ingestion_intents where tenant_code=$1 and institution_id=$2 and actor_subject=$3 and idempotency_key=$4`, out.Tenant, out.Institution, out.Actor, idempotency).Scan(&out.ID, &out.PreparationID, &out.Slot, &out.SHA256, &out.Size, &out.DocumentID, &out.VersionID, &out.Bucket, &out.Key, &out.Retention, &out.StoredVersionID); err != nil {
			return out, false, err
		}
		if err = tx.Commit(ctx); err != nil {
			return out, false, err
		}
		return out, true, nil
	}
	if !errors.Is(err, pgx.ErrNoRows) {
		return out, false, err
	}
	out.ID, out.PreparationID, out.Slot, out.SHA256, out.Size = uuid.NewString(), preparationID, slot, sha, size
	out.DocumentID, out.VersionID, out.Bucket = uuid.NewString(), uuid.NewString(), s.storage.Bucket()
	out.Key = fmt.Sprintf("admission/legal/%s/%s/%s/%s.pdf", out.Tenant, out.Institution, preparationID, slot)
	err = tx.QueryRow(ctx, `insert into archive_ingestion_intents(id,tenant_code,institution_id,actor_subject,idempotency_key,request_fingerprint,purpose,preparation_id,artifact_slot,expected_sha256,expected_size_bytes,reserved_document_id,reserved_version_id,bucket_name,object_key,retention_until,legal_hold_active) values($1::uuid,$2,$3,$4,$5,$6,'admission_legal_preparation',$7::uuid,$8,$9,$10,$11::uuid,$12::uuid,$13,$14,$15,true) returning id::text`, out.ID, out.Tenant, out.Institution, out.Actor, idempotency, fingerprint, preparationID, slot, sha, size, out.DocumentID, out.VersionID, out.Bucket, out.Key, out.Retention).Scan(&out.ID)
	if err != nil {
		return out, false, err
	}
	if err = tx.Commit(ctx); err != nil {
		return out, false, err
	}
	return out, false, nil
}

// The route supplies earchiva.manage; this DB check supplies the operation
// specific admission authority after the preparation has been locked.  It is
// intentionally evaluated against the active tenant/session, not a browser
// role claim.
func archiveAdmissionActorPermission(ctx context.Context, tx pgx.Tx, actor, permission string) error {
	var allowed bool
	err := tx.QueryRow(ctx, `select exists(
 select 1 from app_users u join app_user_permissions p on p.user_id=u.id and p.tenant_code=public.current_tenant_code() where (u.id::text=$1 or lower(u.sub)=lower($1)) and u.status='active' and p.permission_code=$2 and exists(select 1 from app_memberships m where m.user_id=u.id and m.tenant_code=public.current_tenant_code() and public.education_membership_is_eligible(u.id,public.current_tenant_code(),public.current_institution_id(),m.position_code))
 union all select 1 from app_users u join app_user_roles ur on ur.user_id=u.id and ur.tenant_code=public.current_tenant_code() join app_role_permissions rp on rp.role_code=ur.role_code where (u.id::text=$1 or lower(u.sub)=lower($1)) and u.status='active' and rp.permission_code=$2 and exists(select 1 from app_memberships m where m.user_id=u.id and m.tenant_code=public.current_tenant_code() and public.education_membership_is_eligible(u.id,public.current_tenant_code(),public.current_institution_id(),m.position_code))
 union all select 1 from app_users u join app_memberships m on m.user_id=u.id and m.tenant_code=public.current_tenant_code() join app_position_permissions p on p.position_code=m.position_code where (u.id::text=$1 or lower(u.sub)=lower($1)) and u.status='active' and p.permission_code=$2 and public.education_membership_is_eligible(u.id,public.current_tenant_code(),public.current_institution_id(),m.position_code)
 union all select 1 from app_users u join app_memberships m on m.user_id=u.id and m.tenant_code=public.current_tenant_code() join app_position_roles pr on pr.position_code=m.position_code join app_role_permissions rp on rp.role_code=pr.role_code where (u.id::text=$1 or lower(u.sub)=lower($1)) and u.status='active' and rp.permission_code=$2 and public.education_membership_is_eligible(u.id,public.current_tenant_code(),public.current_institution_id(),m.position_code)
 )`, actor, permission).Scan(&allowed)
	if err != nil {
		return err
	}
	if !allowed {
		return errAdmissionArtifactForbidden
	}
	return nil
}

func (s *DocumentService) adoptAdmissionWORMIntent(ctx context.Context, r *http.Request, i admissionWORMIntent, object ImmutableArchiveObject, filename string, pages int) error {
	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return err
	}
	defer tx.Rollback(ctx) //nolint:errcheck
	var status, kind, preparer string
	var expiresAt time.Time
	if err = tx.QueryRow(ctx, `select status,artifact_kind,prepared_by_subject,expires_at from school_admission_legal_preparations where tenant_code=$1 and institution_id=$2 and id=$3::uuid for update`, i.Tenant, i.Institution, i.PreparationID).Scan(&status, &kind, &preparer, &expiresAt); err != nil {
		return err
	}
	if status != "prepared" || preparer != i.Actor || !expiresAt.After(time.Now().UTC()) {
		return errAdmissionArtifactState
	}
	if err = admissionPreparationRetentionCurrent(ctx, tx, i.Tenant, i.Institution, i.PreparationID); err != nil {
		return err
	}
	permission := "education.admissions.decide"
	if kind == "admission_appeal_resolution" {
		permission = "education.admissions.appeals.manage"
	}
	if err = archiveAdmissionActorPermission(ctx, tx, i.Actor, permission); err != nil {
		return err
	}
	if err = archiveAdmissionActorPermission(ctx, tx, i.Actor, "earchiva.manage"); err != nil {
		return err
	}
	var intentStatus string
	if err = tx.QueryRow(ctx, `select status from archive_ingestion_intents where id=$1::uuid and tenant_code=$2 and institution_id=$3 for update`, i.ID, i.Tenant, i.Institution).Scan(&intentStatus); err != nil {
		return err
	}
	if intentStatus == "committed" {
		return tx.Commit(ctx)
	}
	if intentStatus != "reserved" && intentStatus != "stored" {
		return errAdmissionArtifactState
	}
	if _, err = tx.Exec(ctx, `update archive_ingestion_intents set status='stored',stored_version_id=$1,stored_etag=$2,stored_size_bytes=$3,stored_retention_until=$4,failure_code='' where id=$5::uuid`, object.VersionID, object.ETag, object.SizeBytes, object.RetentionUntil, i.ID); err != nil {
		return err
	}
	metadata, _ := json.Marshal(map[string]any{"admission_preparation_id": i.PreparationID, "artifact_slot": i.Slot, "source_page_count": pages})
	title := "Admission legal artifact " + i.Slot
	artifactKey := s.storage.ArtifactObjectKey(i.Institution, i.DocumentID, 1)
	if _, err = tx.Exec(ctx, `insert into archive_documents(id,institution_id,title,original_file_name,mime_type,source_kind,source_system,external_reference,status,original_bucket,original_object_key,artifact_bucket,artifact_object_key,metadata,idempotency_key,created_by,current_version_no,received_at) values($1::uuid,$2,$3,$4,'application/pdf','upload','admission',$5,'queued',$6,$7,$6,$8,$9::jsonb,$10,$11,1,now())`, i.DocumentID, i.Institution, title, filename, i.PreparationID, i.Bucket, i.Key, artifactKey, metadata, "admission-worm:"+i.ID, i.Actor); err != nil {
		return err
	}
	if _, err = tx.Exec(ctx, `insert into archive_document_versions(id,institution_id,document_id,version_no,mime_type,title,bucket_name,object_key,hash_sha256,size_bytes,metadata,ocr_text,status,source_bucket,source_object_key,artifact_bucket,artifact_object_key,source_sha256,source_size_bytes,page_count,text_status,extracted_text,extracted_metadata,created_by,source_object_version_id,source_object_etag,retention_until,legal_hold_active,ingestion_intent_id) values($1::uuid,$2,$3::uuid,1,'application/pdf',$4,$5,$6,$7,$8,$9::jsonb,'','active',$5,$6,$5,$10,$7,$8,$11,'pending','',$9::jsonb,$12,$13,$14,$15,true,$16::uuid)`, i.VersionID, i.Institution, i.DocumentID, title, i.Bucket, i.Key, i.SHA256, i.Size, metadata, artifactKey, pages, i.Actor, object.VersionID, object.ETag, object.RetentionUntil, i.ID); err != nil {
		return err
	}
	if _, err = tx.Exec(ctx, `insert into archive_ingestion_jobs(id,institution_id,document_id,version_id,job_type,status,available_at,created_by) values($1::uuid,$2,$3::uuid,$4::uuid,'extract_text','pending',now(),$5)`, uuid.NewString(), i.Institution, i.DocumentID, i.VersionID, i.Actor); err != nil {
		return err
	}
	if err = audit.Log(ctx, tx, audit.Event{ActorSubject: i.Actor, Action: "admission.legal_artifact.upload", TargetType: "archive_document", TargetID: i.DocumentID, Summary: "Preparation-bound WORM legal artifact accepted.", Details: map[string]any{"preparation_id": i.PreparationID, "artifact_slot": i.Slot, "intent_id": i.ID}}); err != nil {
		return err
	}
	if _, err = tx.Exec(ctx, `update archive_ingestion_intents set status='committed',committed_at=now() where id=$1::uuid and status='stored'`, i.ID); err != nil {
		return err
	}
	return tx.Commit(ctx)
}

func admissionPreparationRetentionCurrent(ctx context.Context, tx pgx.Tx, tenant, institution, preparationID string) error {
	var valid bool
	err := tx.QueryRow(ctx, `select exists(select 1 from school_admission_legal_preparations p join school_admission_dss_retention_policies policy on policy.tenant_code=p.tenant_code and policy.institution_id=p.institution_id and policy.id=p.retention_policy_id join school_admission_retention_rule_versions rule on rule.tenant_code=p.tenant_code and rule.institution_id=p.institution_id and rule.id=p.retention_rule_version_id where p.tenant_code=$1 and p.institution_id=$2 and p.id=$3::uuid and p.retention_policy_id is not null and p.required_retention_until is not null and policy.status='active' and policy.rule_version_id=p.retention_rule_version_id and policy.effective_from<=timezone('UTC',now())::date and (policy.effective_to is null or policy.effective_to>=timezone('UTC',now())::date) and policy.effective_from<=(p.expires_at at time zone 'UTC')::date and (policy.effective_to is null or policy.effective_to>=(p.expires_at at time zone 'UTC')::date) and rule.status='active' and rule.source_id=p.retention_source_id and rule.minimum_retention_days=p.minimum_retention_days and rule.effective_from<=timezone('UTC',now())::date and (rule.effective_to is null or rule.effective_to>=timezone('UTC',now())::date) and rule.effective_from<=(p.expires_at at time zone 'UTC')::date and (rule.effective_to is null or rule.effective_to>=(p.expires_at at time zone 'UTC')::date) )`, tenant, institution, preparationID).Scan(&valid)
	if err != nil {
		return err
	}
	if !valid {
		return errAdmissionArtifactState
	}
	return nil
}

func (s *DocumentService) loadAdmissionArtifact(ctx context.Context, tenant, institution, actor, preparationID, slot string) (AdmissionLegalPreparationArtifactResponse, error) {
	var out AdmissionLegalPreparationArtifactResponse
	var retention time.Time
	err := s.pool.QueryRow(ctx, `select i.id::text,i.preparation_id::text,i.artifact_slot,d.id::text,d.institution_id,d.title,d.original_file_name,d.mime_type,d.source_kind,d.source_system,d.external_reference,d.status,d.current_version_no,to_char(d.received_at at time zone 'UTC','YYYY-MM-DD"T"HH24:MI:SS.US"Z"'),to_char(d.created_at at time zone 'UTC','YYYY-MM-DD"T"HH24:MI:SS.US"Z"'),to_char(d.updated_at at time zone 'UTC','YYYY-MM-DD"T"HH24:MI:SS.US"Z"'),v.id::text,v.document_id::text,v.version_no,v.source_sha256,v.source_size_bytes,v.page_count,v.text_status,to_char(v.created_at at time zone 'UTC','YYYY-MM-DD"T"HH24:MI:SS.US"Z"'),to_char(i.retention_until at time zone 'UTC','YYYY-MM-DD"T"HH24:MI:SS.US"Z"') from archive_ingestion_intents i join archive_documents d on d.id=i.reserved_document_id and d.institution_id=i.institution_id join archive_document_versions v on v.id=i.reserved_version_id and v.document_id=d.id and v.institution_id=d.institution_id where i.tenant_code=$1 and i.institution_id=$2 and i.actor_subject=$3 and i.preparation_id=$4::uuid and i.artifact_slot=$5 and i.status='committed'`, tenant, institution, actor, preparationID, slot).Scan(&out.IntentID, &out.PreparationID, &out.ArtifactSlot, &out.Document.ID, &out.Document.InstitutionID, &out.Document.Title, &out.Document.OriginalFileName, &out.Document.MimeType, &out.Document.SourceKind, &out.Document.SourceSystem, &out.Document.ExternalReference, &out.Document.Status, &out.Document.CurrentVersionNo, &out.Document.ReceivedAt, &out.Document.CreatedAt, &out.Document.UpdatedAt, &out.Version.ID, &out.Version.DocumentID, &out.Version.VersionNo, &out.Version.SourceSHA256, &out.Version.SourceSizeBytes, &out.Version.PageCount, &out.Version.TextStatus, &out.Version.CreatedAt, &out.RetentionUntil)
	_ = retention
	return out, err
}

var (
	errAdmissionArtifactScope     = errors.New("admission artifact scope required")
	errAdmissionArtifactState     = errors.New("admission artifact state conflict")
	errAdmissionArtifactConflict  = errors.New("admission artifact idempotency conflict")
	errAdmissionArtifactForbidden = errors.New("admission artifact forbidden")
)

func writeAdmissionArtifactError(w http.ResponseWriter, err error) {
	if err == nil {
		return
	}
	switch {
	case errors.Is(err, errAdmissionArtifactScope):
		httpx.JSON(w, http.StatusUnauthorized, map[string]any{"code": "admission_context_required"})
	case errors.Is(err, errAdmissionArtifactState):
		httpx.JSON(w, http.StatusConflict, map[string]any{"code": "admission_artifact_state_conflict"})
	case errors.Is(err, errAdmissionArtifactConflict):
		httpx.JSON(w, http.StatusConflict, map[string]any{"code": "admission_artifact_idempotency_conflict"})
	case errors.Is(err, errAdmissionArtifactForbidden):
		httpx.JSON(w, http.StatusForbidden, map[string]any{"code": "admission_artifact_forbidden"})
	case errors.Is(err, pgx.ErrNoRows):
		httpx.JSON(w, http.StatusNotFound, map[string]any{"code": "admission_artifact_not_found"})
	default:
		httpx.JSON(w, http.StatusInternalServerError, map[string]any{"code": "admission_artifact_failed"})
	}
}
