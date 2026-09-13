package education

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"github.com/eguilde/egueducation/internal/audit"
	authruntime "github.com/eguilde/egueducation/internal/auth"
	"github.com/eguilde/egueducation/internal/httpx"
	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
)

const portfolioCustodyManagePermission = "education.portfolios.custody.manage"

// PortfolioRetentionDispositions exposes the durable review trail without
// disclosing storage coordinates. It uses the same owner/school read boundary
// as lifecycle operation history and performs all filtering/sorting/paging in
// PostgreSQL.
func (s *Service) PortfolioRetentionDispositions(w http.ResponseWriter, r *http.Request) {
	recordID := strings.TrimSpace(chi.URLParam(r, "recordID"))
	subject := strings.TrimSpace(authruntime.CurrentSubjectFromRequest(r))
	if subject == "" || uuid.Validate(recordID) != nil {
		writeEducationNotFound(w, "portfolio_retention_disposition_not_found")
		return
	}
	ctx := r.Context()
	tx, err := s.pool.Begin(ctx)
	if err != nil {
		httpx.JSON(w, http.StatusInternalServerError, map[string]any{"code": "portfolio_retention_dispositions_failed"})
		return
	}
	defer tx.Rollback(ctx) //nolint:errcheck
	allowed, err := portfolioLifecycleReadAllowed(ctx, tx, subject, recordID)
	if err != nil || !allowed {
		writePortfolioAccessFailure(w, err)
		return
	}
	query := httpx.ParsePageQuery(r.URL.Query(), map[string]struct{}{"requested_at": {}, "status": {}, "decision": {}}, []string{"status", "decision", "requested_at"})
	if query.Sort == "" {
		query.Sort = "requested_at"
	}
	if strings.TrimSpace(r.URL.Query().Get("direction")) == "" {
		query.Direction = "desc"
	}
	if _, ok := map[string]struct{}{"requested_at": {}, "status": {}, "decision": {}}[query.Sort]; !ok || (query.Direction != "asc" && query.Direction != "desc") {
		httpx.JSON(w, http.StatusBadRequest, map[string]any{"code": "invalid_portfolio_retention_disposition_query"})
		return
	}
	if status := query.Filters["status"]; status != "" {
		if _, ok := map[string]struct{}{"submitted": {}, "approved": {}, "rejected": {}, "blocked": {}, "closed": {}}[status]; !ok {
			httpx.JSON(w, http.StatusBadRequest, map[string]any{"code": "invalid_portfolio_retention_disposition_filters"})
			return
		}
	}
	if decision := query.Filters["decision"]; decision != "" && decision != "approved" && decision != "rejected" {
		httpx.JSON(w, http.StatusBadRequest, map[string]any{"code": "invalid_portfolio_retention_disposition_filters"})
		return
	}
	if requestedAt := query.Filters["requested_at"]; requestedAt != "" {
		if parsed, parseErr := time.Parse("2006-01-02", requestedAt); parseErr != nil || parsed.Format("2006-01-02") != requestedAt {
			httpx.JSON(w, http.StatusBadRequest, map[string]any{"code": "invalid_portfolio_retention_disposition_filters"})
			return
		}
	}
	where := `where q.institution_id=public.current_institution_id() and q.portfolio_id=$1::uuid`
	args := []any{recordID}
	if value := query.Filters["status"]; value != "" {
		args = append(args, value)
		where += fmt.Sprintf(" and q.status=$%d", len(args))
	}
	if value := query.Filters["decision"]; value != "" {
		args = append(args, value)
		where += fmt.Sprintf(" and d.decision=$%d", len(args))
	}
	if value := query.Filters["requested_at"]; value != "" {
		args = append(args, value)
		where += fmt.Sprintf(" and (q.requested_at at time zone 'UTC')::date=$%d::date", len(args))
	}
	var total int
	if err = tx.QueryRow(ctx, `select count(*) from portfolio_retention_disposition_requests q left join portfolio_retention_disposition_decisions d on d.request_id=q.id `+where, args...).Scan(&total); err != nil {
		httpx.JSON(w, http.StatusInternalServerError, map[string]any{"code": "portfolio_retention_dispositions_failed"})
		return
	}
	sortColumn := map[string]string{"requested_at": "q.requested_at", "status": "q.status", "decision": "d.decision"}[query.Sort]
	pageArgs := append(append([]any(nil), args...), query.PageSize, (query.Page-1)*query.PageSize)
	rows, err := tx.Query(ctx, `select q.id::text,q.transition_id::text,q.portfolio_id::text,q.status,q.evidence,q.requested_by_subject,
		to_char(q.requested_at at time zone 'UTC','YYYY-MM-DD"T"HH24:MI:SS.MS"Z"'),coalesce(d.decision,''),coalesce(d.reason,''),coalesce(d.decided_by_subject,''),
		coalesce(to_char(d.decided_at at time zone 'UTC','YYYY-MM-DD"T"HH24:MI:SS.MS"Z"'),''),coalesce(receipt.outcome,''),coalesce(o.id::text,''),coalesce(o.status,''),coalesce(o.attempts,0),coalesce(o.last_error_code,'')
		from portfolio_retention_disposition_requests q left join portfolio_retention_disposition_decisions d on d.request_id=q.id
		left join portfolio_retention_disposition_receipts receipt on receipt.request_id=q.id left join portfolio_retention_disposition_operations o on o.request_id=q.id `+where+`
		order by `+sortColumn+` `+strings.ToUpper(query.Direction)+`,q.id `+strings.ToUpper(query.Direction)+` limit $`+fmt.Sprint(len(pageArgs)-1)+` offset $`+fmt.Sprint(len(pageArgs)), pageArgs...)
	if err != nil {
		httpx.JSON(w, http.StatusInternalServerError, map[string]any{"code": "portfolio_retention_dispositions_failed"})
		return
	}
	defer rows.Close()
	items := make([]PortfolioRetentionDisposition, 0, query.PageSize)
	for rows.Next() {
		var item PortfolioRetentionDisposition
		var evidence []byte
		if err = rows.Scan(&item.ID, &item.TransitionID, &item.PortfolioID, &item.Status, &evidence, &item.RequestedBySubject, &item.RequestedAt, &item.Decision, &item.DecisionReason, &item.DecidedBySubject, &item.DecidedAt, &item.Outcome, &item.OperationID, &item.OperationStatus, &item.OperationAttempts, &item.LastErrorCode); err != nil || json.Unmarshal(evidence, &item.Evidence) != nil {
			httpx.JSON(w, http.StatusInternalServerError, map[string]any{"code": "portfolio_retention_dispositions_failed"})
			return
		}
		items = append(items, item)
	}
	if err = rows.Err(); err != nil {
		httpx.JSON(w, http.StatusInternalServerError, map[string]any{"code": "portfolio_retention_dispositions_failed"})
		return
	}
	if err = tx.Commit(ctx); err != nil {
		httpx.JSON(w, http.StatusInternalServerError, map[string]any{"code": "portfolio_retention_dispositions_failed"})
		return
	}
	httpx.WritePage(w, http.StatusOK, items, total, query.Page, query.PageSize)
}

// SubmitPortfolioRetentionDisposition is the owner-only, append-only request
// for an expired retention transition. Exact object/version facts are copied
// from the locked transition rather than accepted from a client.
func (s *Service) SubmitPortfolioRetentionDisposition(w http.ResponseWriter, r *http.Request) {
	recordID := strings.TrimSpace(chi.URLParam(r, "recordID"))
	operationID := strings.TrimSpace(chi.URLParam(r, "operationID"))
	transitionID := strings.TrimSpace(chi.URLParam(r, "transitionID"))
	if uuid.Validate(recordID) != nil || uuid.Validate(operationID) != nil || uuid.Validate(transitionID) != nil {
		writeEducationNotFound(w, "portfolio_lifecycle_operation_not_found")
		return
	}
	_, allowed, err := s.requireOwnPortfolio(r, recordID, portfolioManageOwnPermission)
	if err != nil || !allowed {
		writePortfolioAccessFailure(w, err)
		return
	}
	var req PortfolioRetentionDispositionRequest
	decoder := json.NewDecoder(http.MaxBytesReader(w, r.Body, 16<<10))
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(&req); err != nil || decoder.Decode(&struct{}{}) != io.EOF {
		httpx.JSON(w, http.StatusBadRequest, map[string]any{"code": "portfolio_retention_disposition_evidence_required"})
		return
	}
	req.Evidence.Statement = strings.TrimSpace(req.Evidence.Statement)
	req.Evidence.Reference = strings.TrimSpace(req.Evidence.Reference)
	if req.Evidence.Statement == "" || len(req.Evidence.Statement) > 4000 || len(req.Evidence.Reference) > 500 {
		httpx.JSON(w, http.StatusBadRequest, map[string]any{"code": "portfolio_retention_disposition_evidence_required"})
		return
	}
	evidence, err := json.Marshal(req.Evidence)
	if err != nil {
		httpx.JSON(w, http.StatusBadRequest, map[string]any{"code": "portfolio_retention_disposition_evidence_invalid"})
		return
	}
	ctx := r.Context()
	subject := strings.TrimSpace(authruntime.CurrentSubjectFromRequest(r))
	tx, err := s.pool.Begin(ctx)
	if err != nil {
		httpx.JSON(w, http.StatusInternalServerError, map[string]any{"code": "portfolio_retention_disposition_failed"})
		return
	}
	defer tx.Rollback(ctx) //nolint:errcheck
	var requestID string
	err = tx.QueryRow(ctx, `insert into portfolio_retention_disposition_requests(
		transition_id,tenant_code,institution_id,portfolio_id,archive_document_id,archive_version_id,source_bucket,source_object_key,source_object_version_id,source_object_etag,source_sha256,source_size_bytes,source_mime_type,required_retention_until,evidence,requested_by_subject)
		select t.id,public.current_tenant_code(),t.institution_id,t.portfolio_id,t.archive_document_id,t.archive_version_id,t.source_bucket,t.source_object_key,t.source_object_version_id,t.source_object_etag,t.source_sha256,t.source_size_bytes,v.mime_type,t.required_retention_until,$4::jsonb,$5
		from education_portfolio_storage_transitions t join archive_document_versions v on v.id=t.archive_version_id and v.institution_id=t.institution_id
		where t.id=$3::uuid and t.operation_id=$1::uuid and t.portfolio_id=$2::uuid and t.institution_id=public.current_institution_id()
		and t.status='blocked' and t.last_error='portfolio_retention_expired_review_required' and t.required_retention_until is not null
		returning id::text`, operationID, recordID, transitionID, evidence, subject).Scan(&requestID)
	if errors.Is(err, pgx.ErrNoRows) {
		httpx.JSON(w, http.StatusConflict, map[string]any{"code": "portfolio_retention_disposition_not_available"})
		return
	}
	if err != nil {
		httpx.JSON(w, http.StatusInternalServerError, map[string]any{"code": "portfolio_retention_disposition_failed"})
		return
	}
	if err := audit.Log(ctx, tx, audit.Event{ActorSubject: subject, Action: "education.portfolios.retention_disposition_submitted", TargetType: "portfolio_retention_disposition_request", TargetID: requestID, Summary: "Portfolio retention disposition submitted for independent review.", Details: map[string]any{"portfolio_id": recordID, "operation_id": operationID, "transition_id": transitionID}}); err != nil {
		httpx.JSON(w, http.StatusInternalServerError, map[string]any{"code": "portfolio_retention_disposition_failed"})
		return
	}
	if err := tx.Commit(ctx); err != nil {
		httpx.JSON(w, http.StatusInternalServerError, map[string]any{"code": "portfolio_retention_disposition_failed"})
		return
	}
	httpx.JSON(w, http.StatusAccepted, PortfolioRetentionDispositionCommandResponse{ID: requestID, Status: "submitted"})
}

// DecidePortfolioRetentionDisposition enforces a separate custody and archive
// approver. Approval is refused while a legal hold or another portfolio
// reference still requires custody or a later retention deadline.
func (s *Service) DecidePortfolioRetentionDisposition(w http.ResponseWriter, r *http.Request) {
	recordID := strings.TrimSpace(chi.URLParam(r, "recordID"))
	requestID := strings.TrimSpace(chi.URLParam(r, "requestID"))
	if uuid.Validate(recordID) != nil || uuid.Validate(requestID) != nil {
		writeEducationNotFound(w, "portfolio_retention_disposition_not_found")
		return
	}
	var req PortfolioRetentionDispositionDecisionRequest
	decoder := json.NewDecoder(http.MaxBytesReader(w, r.Body, 8<<10))
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(&req); err != nil || decoder.Decode(&struct{}{}) != io.EOF || strings.TrimSpace(req.Reason) == "" || len(strings.TrimSpace(req.Reason)) > 2000 {
		httpx.JSON(w, http.StatusBadRequest, map[string]any{"code": "portfolio_retention_disposition_decision_invalid"})
		return
	}
	ctx := r.Context()
	subject := strings.TrimSpace(authruntime.CurrentSubjectFromRequest(r))
	tx, err := s.pool.Begin(ctx)
	if err != nil {
		httpx.JSON(w, http.StatusInternalServerError, map[string]any{"code": "portfolio_retention_disposition_decision_failed"})
		return
	}
	defer tx.Rollback(ctx) //nolint:errcheck
	for _, permission := range []string{portfolioCustodyManagePermission, "earchiva.manage"} {
		allowed, permissionErr := educationSubjectHasPermission(ctx, tx, subject, permission)
		if permissionErr != nil || !allowed {
			writePortfolioAccessFailure(w, permissionErr)
			return
		}
	}
	var requester, transitionID string
	var legalHoldOrRequired bool
	err = tx.QueryRow(ctx, `select q.requested_by_subject,q.transition_id::text,
		exists(select 1 from education_portfolio_documents pd join education_portfolios p on p.id=pd.portfolio_id and p.institution_id=pd.institution_id
		 where pd.archive_version_id=q.archive_version_id and pd.institution_id=q.institution_id
		 and (p.legal_hold_active or p.activity_ceased_on is null or (p.retention_until is not null and (p.retention_until+1)::timestamp at time zone 'UTC'>q.required_retention_until)))
		from portfolio_retention_disposition_requests q where q.id=$1::uuid and q.portfolio_id=$2::uuid and q.institution_id=public.current_institution_id() and q.status='submitted' for update`, requestID, recordID).Scan(&requester, &transitionID, &legalHoldOrRequired)
	if errors.Is(err, pgx.ErrNoRows) {
		writeEducationNotFound(w, "portfolio_retention_disposition_not_found")
		return
	}
	if err != nil {
		httpx.JSON(w, http.StatusInternalServerError, map[string]any{"code": "portfolio_retention_disposition_decision_failed"})
		return
	}
	if strings.EqualFold(requester, subject) {
		httpx.JSON(w, http.StatusForbidden, map[string]any{"code": "portfolio_retention_disposition_separation_required"})
		return
	}
	decision := "rejected"
	outcome := "blocked"
	if req.Approve && !legalHoldOrRequired {
		// Approval records only that the review may close while the exact S3
		// version remains held. The adapter independently forbids an expired
		// retention hold-off, so a physical custody release needs a later
		// storage-verification receipt rather than this decision alone.
		decision, outcome = "approved", "retained"
	}
	if _, err := tx.Exec(ctx, `insert into portfolio_retention_disposition_decisions(request_id,decision,reason,decided_by_subject) values($1::uuid,$2,$3,$4)`, requestID, decision, strings.TrimSpace(req.Reason), subject); err != nil {
		httpx.JSON(w, http.StatusInternalServerError, map[string]any{"code": "portfolio_retention_disposition_decision_failed"})
		return
	}
	if decision == "approved" {
		if _, err := tx.Exec(ctx, `update portfolio_retention_disposition_requests set status='approved' where id=$1::uuid and status='submitted'`, requestID); err != nil {
			httpx.JSON(w, http.StatusInternalServerError, map[string]any{"code": "portfolio_retention_disposition_decision_failed"})
			return
		}
		if _, err := tx.Exec(ctx, `insert into portfolio_retention_disposition_operations(request_id,tenant_code,institution_id)
			select id,tenant_code,institution_id from portfolio_retention_disposition_requests where id=$1::uuid`, requestID); err != nil {
			httpx.JSON(w, http.StatusInternalServerError, map[string]any{"code": "portfolio_retention_disposition_decision_failed"})
			return
		}
	} else if _, err := tx.Exec(ctx, `insert into portfolio_retention_disposition_receipts(request_id,transition_id,closed_by_subject,outcome)
		select id,transition_id,$2,$3 from portfolio_retention_disposition_requests where id=$1::uuid`, requestID, subject, outcome); err != nil {
		httpx.JSON(w, http.StatusInternalServerError, map[string]any{"code": "portfolio_retention_disposition_decision_failed"})
		return
	} else if _, err := tx.Exec(ctx, `update portfolio_retention_disposition_requests set status='rejected' where id=$1::uuid and status='submitted'`, requestID); err != nil {
		httpx.JSON(w, http.StatusInternalServerError, map[string]any{"code": "portfolio_retention_disposition_decision_failed"})
		return
	}
	if err := audit.Log(ctx, tx, audit.Event{ActorSubject: subject, Action: "education.portfolios.retention_disposition_decided", TargetType: "portfolio_retention_disposition_request", TargetID: requestID, Summary: "Portfolio retention disposition independently decided.", Details: map[string]any{"portfolio_id": recordID, "transition_id": transitionID, "decision": decision, "blocked_by_legal_hold_or_reference": legalHoldOrRequired}}); err != nil {
		httpx.JSON(w, http.StatusInternalServerError, map[string]any{"code": "portfolio_retention_disposition_decision_failed"})
		return
	}
	if err := tx.Commit(ctx); err != nil {
		httpx.JSON(w, http.StatusInternalServerError, map[string]any{"code": "portfolio_retention_disposition_decision_failed"})
		return
	}
	statusResponse := "rejected"
	if decision == "approved" {
		statusResponse = "approved"
	}
	httpx.JSON(w, http.StatusOK, PortfolioRetentionDispositionCommandResponse{ID: requestID, Status: statusResponse, Decision: decision, Outcome: outcome})
}
