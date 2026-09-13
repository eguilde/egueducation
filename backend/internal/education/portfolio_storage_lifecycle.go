package education

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"

	"github.com/eguilde/egueducation/internal/audit"
	authruntime "github.com/eguilde/egueducation/internal/auth"
	"github.com/eguilde/egueducation/internal/httpx"
)

const portfolioLifecycleManagePermission = "education.portfolios.school.manage"

// RecordPortfolioActivityCessation persists the human event, its exact
// archive-version work set, and the audit record in one transaction. Storage
// reconciliation is deliberately asynchronous and represented by Operation.
func (s *Service) RecordPortfolioActivityCessation(w http.ResponseWriter, r *http.Request) {
	var req PortfolioCessationRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		httpx.JSON(w, http.StatusBadRequest, map[string]any{"code": "invalid_portfolio_cessation_payload"})
		return
	}
	req.ActivityCeasedOn = strings.TrimSpace(req.ActivityCeasedOn)
	req.Reason = strings.TrimSpace(req.Reason)
	ceasedOn, parseErr := time.Parse("2006-01-02", req.ActivityCeasedOn)
	if parseErr != nil || ceasedOn.After(time.Now().UTC()) || req.Reason == "" {
		httpx.JSON(w, http.StatusBadRequest, map[string]any{"code": "invalid_portfolio_cessation"})
		return
	}

	response, status, err := s.createPortfolioLifecycleOperation(r, strings.TrimSpace(chi.URLParam(r, "recordID")), "cessation_retention", req.ActivityCeasedOn, nil, req.Reason)
	if err != nil {
		writePortfolioLifecycleError(w, status, err)
		return
	}
	httpx.JSON(w, http.StatusAccepted, response)
}

// SetPortfolioLegalHold records explicit human legal-hold intent. The worker
// never changes this portfolio flag; it only reconciles the physical S3 hold
// to the aggregate operational-custody OR legal-hold requirement.
func (s *Service) SetPortfolioLegalHold(w http.ResponseWriter, r *http.Request) {
	var req PortfolioLegalHoldRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		httpx.JSON(w, http.StatusBadRequest, map[string]any{"code": "invalid_portfolio_legal_hold_payload"})
		return
	}
	req.Reason = strings.TrimSpace(req.Reason)
	if req.Reason == "" {
		httpx.JSON(w, http.StatusBadRequest, map[string]any{"code": "portfolio_legal_hold_reason_required"})
		return
	}

	response, status, err := s.createPortfolioLifecycleOperation(r, strings.TrimSpace(chi.URLParam(r, "recordID")), "legal_hold_reconcile", "", &req.Active, req.Reason)
	if err != nil {
		writePortfolioLifecycleError(w, status, err)
		return
	}
	httpx.JSON(w, http.StatusAccepted, response)
}

// PortfolioLifecycleOperationStatus returns status only after applying the
// same institution/owner boundary as the portfolio itself.
func (s *Service) PortfolioLifecycleOperationStatus(w http.ResponseWriter, r *http.Request) {
	recordID := strings.TrimSpace(chi.URLParam(r, "recordID"))
	operationID := strings.TrimSpace(chi.URLParam(r, "operationID"))
	subject := strings.TrimSpace(authruntime.CurrentSubjectFromRequest(r))
	if subject == "" || uuid.Validate(recordID) != nil || uuid.Validate(operationID) != nil {
		writePortfolioAccessFailure(w, nil)
		return
	}

	tx, err := s.pool.Begin(r.Context())
	if err != nil {
		httpx.JSON(w, http.StatusInternalServerError, map[string]any{"code": "portfolio_lifecycle_status_failed"})
		return
	}
	defer tx.Rollback(r.Context()) //nolint:errcheck

	allowed, err := portfolioLifecycleReadAllowed(r.Context(), tx, subject, recordID)
	if err != nil || !allowed {
		writePortfolioAccessFailure(w, err)
		return
	}
	response, err := loadPortfolioLifecycleResponse(r.Context(), tx, recordID, operationID)
	if errors.Is(err, pgx.ErrNoRows) {
		writeEducationNotFound(w, "portfolio_lifecycle_operation_not_found")
		return
	}
	if err != nil {
		httpx.JSON(w, http.StatusInternalServerError, map[string]any{"code": "portfolio_lifecycle_status_failed"})
		return
	}
	if err := tx.Commit(r.Context()); err != nil {
		httpx.JSON(w, http.StatusInternalServerError, map[string]any{"code": "portfolio_lifecycle_status_failed"})
		return
	}
	httpx.JSON(w, http.StatusOK, response)
}

// PortfolioLifecycleOperations lists durable lifecycle commands after the
// portfolio read boundary has been established. Transition counts are
// aggregated in the page query; this endpoint never performs per-item reads.
func (s *Service) PortfolioLifecycleOperations(w http.ResponseWriter, r *http.Request) {
	recordID := strings.TrimSpace(chi.URLParam(r, "recordID"))
	subject := strings.TrimSpace(authruntime.CurrentSubjectFromRequest(r))
	if subject == "" || uuid.Validate(recordID) != nil {
		writePortfolioAccessFailure(w, nil)
		return
	}
	ctx := r.Context()
	tx, err := s.pool.Begin(ctx)
	if err != nil {
		httpx.JSON(w, http.StatusInternalServerError, map[string]any{"code": "portfolio_lifecycle_operations_failed"})
		return
	}
	defer tx.Rollback(ctx) //nolint:errcheck
	allowed, err := portfolioLifecycleReadAllowed(ctx, tx, subject, recordID)
	if err != nil || !allowed {
		writePortfolioAccessFailure(w, err)
		return
	}
	var exists bool
	if err := tx.QueryRow(ctx, `select exists(select 1 from education_portfolios where id=$1::uuid and institution_id=public.current_institution_id())`, recordID).Scan(&exists); err != nil {
		httpx.JSON(w, http.StatusInternalServerError, map[string]any{"code": "portfolio_lifecycle_operations_failed"})
		return
	}
	if !exists {
		writeEducationNotFound(w, "education_portfolio_not_found")
		return
	}

	values := r.URL.Query()
	rawPageValue := values.Get("page")
	rawPage := strings.TrimSpace(rawPageValue)
	if rawPageValue != rawPage {
		httpx.JSON(w, http.StatusBadRequest, map[string]any{"code": "invalid_portfolio_lifecycle_operation_query"})
		return
	}
	if rawPage != "" {
		page, parseErr := strconv.Atoi(rawPage)
		if parseErr != nil || page < 1 || page > 1_000_000 {
			httpx.JSON(w, http.StatusBadRequest, map[string]any{"code": "invalid_portfolio_lifecycle_operation_query"})
			return
		}
	}
	query := httpx.ParsePageQuery(values, map[string]struct{}{"requested_at": {}, "status": {}, "type": {}}, []string{"status", "type", "requested_at"})
	requestedSortValue := values.Get("sort")
	requestedSort := strings.TrimSpace(requestedSortValue)
	requestedDirectionValue := values.Get("direction")
	requestedDirection := strings.TrimSpace(requestedDirectionValue)
	if requestedSortValue != requestedSort || requestedDirectionValue != requestedDirection {
		httpx.JSON(w, http.StatusBadRequest, map[string]any{"code": "invalid_portfolio_lifecycle_operation_query"})
		return
	}
	if requestedSort != "" && requestedSort != "requested_at" && requestedSort != "status" && requestedSort != "type" {
		httpx.JSON(w, http.StatusBadRequest, map[string]any{"code": "invalid_portfolio_lifecycle_operation_query"})
		return
	}
	if requestedDirection != "" && !strings.EqualFold(requestedDirection, "asc") && !strings.EqualFold(requestedDirection, "desc") {
		httpx.JSON(w, http.StatusBadRequest, map[string]any{"code": "invalid_portfolio_lifecycle_operation_query"})
		return
	}
	if query.Sort == "" {
		query.Sort = "requested_at"
	}
	if strings.TrimSpace(r.URL.Query().Get("direction")) == "" {
		query.Direction = "desc"
	}
	if !validPortfolioLifecycleFilters(query.Filters) {
		httpx.JSON(w, http.StatusBadRequest, map[string]any{"code": "invalid_portfolio_lifecycle_operation_filters"})
		return
	}
	// Keep the OFFSET arithmetic bounded and reject pathological scans rather
	// than letting an int overflow become a negative database offset.
	if query.Page < 1 || query.Page > 1_000_000 {
		httpx.JSON(w, http.StatusBadRequest, map[string]any{"code": "invalid_portfolio_lifecycle_operation_query"})
		return
	}
	where := `where o.institution_id=public.current_institution_id() and o.portfolio_id=$1::uuid`
	args := []any{recordID}
	if value := query.Filters["status"]; value != "" {
		args = append(args, value)
		where += fmt.Sprintf(" and o.status=$%d", len(args))
	}
	if value := query.Filters["type"]; value != "" {
		args = append(args, value)
		where += fmt.Sprintf(" and o.operation_type=$%d", len(args))
	}
	if value := query.Filters["requested_at"]; value != "" {
		args = append(args, value)
		where += fmt.Sprintf(" and (o.requested_at at time zone 'UTC')::date=$%d::date", len(args))
	}
	var total int
	if err := tx.QueryRow(ctx, `select count(*) from education_portfolio_lifecycle_operations o `+where, args...).Scan(&total); err != nil {
		httpx.JSON(w, http.StatusInternalServerError, map[string]any{"code": "portfolio_lifecycle_operations_failed"})
		return
	}
	sortColumn := map[string]string{"requested_at": "o.requested_at", "status": "o.status", "type": "o.operation_type"}[query.Sort]
	pageArgs := append(append([]any(nil), args...), query.PageSize, (query.Page-1)*query.PageSize)
	rows, err := tx.Query(ctx, `select o.id::text,o.portfolio_id::text,o.operation_type,o.status,
		to_char(o.requested_at at time zone 'UTC','YYYY-MM-DD"T"HH24:MI:SS.MS"Z"'),
		coalesce(to_char(o.completed_at at time zone 'UTC','YYYY-MM-DD"T"HH24:MI:SS.MS"Z"'),''),
		count(s.id)::int,count(s.id) filter(where s.status='completed')::int,
		count(s.id) filter(where s.status in ('blocked','dead_letter'))::int,o.last_error
		from education_portfolio_lifecycle_operations o
		left join education_portfolio_storage_transitions s on s.operation_id=o.id
		`+where+`
		group by o.id
		order by `+sortColumn+` `+strings.ToUpper(query.Direction)+`,o.id `+strings.ToUpper(query.Direction)+`
		limit $`+fmt.Sprint(len(pageArgs)-1)+` offset $`+fmt.Sprint(len(pageArgs)), pageArgs...)
	if err != nil {
		httpx.JSON(w, http.StatusInternalServerError, map[string]any{"code": "portfolio_lifecycle_operations_failed"})
		return
	}
	defer rows.Close()
	items := make([]PortfolioLifecycleOperation, 0, query.PageSize)
	for rows.Next() {
		item, err := scanPortfolioLifecycleOperation(rows)
		if err != nil {
			httpx.JSON(w, http.StatusInternalServerError, map[string]any{"code": "portfolio_lifecycle_operations_failed"})
			return
		}
		items = append(items, item)
	}
	if err := rows.Err(); err != nil {
		httpx.JSON(w, http.StatusInternalServerError, map[string]any{"code": "portfolio_lifecycle_operations_failed"})
		return
	}
	if err := tx.Commit(ctx); err != nil {
		httpx.JSON(w, http.StatusInternalServerError, map[string]any{"code": "portfolio_lifecycle_operations_failed"})
		return
	}
	httpx.WritePage(w, http.StatusOK, items, total, query.Page, query.PageSize)
}

func validPortfolioLifecycleFilters(filters map[string]string) bool {
	if value := filters["status"]; value != "" {
		if _, ok := map[string]struct{}{"pending": {}, "processing": {}, "completed": {}, "blocked": {}, "dead_letter": {}}[value]; !ok {
			return false
		}
	}
	if value := filters["type"]; value != "" && value != "cessation_retention" && value != "legal_hold_reconcile" {
		return false
	}
	if value := filters["requested_at"]; value != "" {
		if parsed, err := time.Parse("2006-01-02", value); err != nil || parsed.Format("2006-01-02") != value {
			return false
		}
	}
	return true
}

type portfolioLifecycleOperationScanner interface{ Scan(...any) error }

func scanPortfolioLifecycleOperation(row portfolioLifecycleOperationScanner) (PortfolioLifecycleOperation, error) {
	var operation PortfolioLifecycleOperation
	err := row.Scan(&operation.ID, &operation.PortfolioID, &operation.Type, &operation.Status,
		&operation.RequestedAt, &operation.CompletedAt, &operation.TotalVersions,
		&operation.CompletedVersions, &operation.BlockedVersions, &operation.LastError)
	return operation, err
}

// RetryPortfolioLifecycleOperation records an operator-authorized recovery
// attempt without touching completed work or stealing an active worker lease.
func (s *Service) RetryPortfolioLifecycleOperation(w http.ResponseWriter, r *http.Request) {
	recordID := strings.TrimSpace(chi.URLParam(r, "recordID"))
	operationID := strings.TrimSpace(chi.URLParam(r, "operationID"))
	if uuid.Validate(recordID) != nil || uuid.Validate(operationID) != nil {
		writeEducationNotFound(w, "portfolio_lifecycle_operation_not_found")
		return
	}
	var req PortfolioLifecycleRetryRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		httpx.JSON(w, http.StatusBadRequest, map[string]any{"code": "invalid_portfolio_lifecycle_retry_payload"})
		return
	}
	req.Reason = strings.TrimSpace(req.Reason)
	if req.Reason == "" {
		httpx.JSON(w, http.StatusBadRequest, map[string]any{"code": "portfolio_lifecycle_retry_reason_required"})
		return
	}
	ctx := r.Context()
	subject := strings.TrimSpace(authruntime.CurrentSubjectFromRequest(r))
	tx, err := s.pool.Begin(ctx)
	if err != nil {
		httpx.JSON(w, http.StatusInternalServerError, map[string]any{"code": "portfolio_lifecycle_retry_failed"})
		return
	}
	defer tx.Rollback(ctx) //nolint:errcheck
	allowed, err := portfolioLifecycleManageAllowed(ctx, tx, subject)
	if err != nil || !allowed {
		writePortfolioAccessFailure(w, err)
		return
	}
	var status string
	if err := tx.QueryRow(ctx, `select status from education_portfolio_lifecycle_operations
		where id=$1::uuid and portfolio_id=$2::uuid and institution_id=public.current_institution_id() for update`, operationID, recordID).Scan(&status); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			writeEducationNotFound(w, "portfolio_lifecycle_operation_not_found")
		} else {
			httpx.JSON(w, http.StatusInternalServerError, map[string]any{"code": "portfolio_lifecycle_retry_failed"})
		}
		return
	}
	rows, err := tx.Query(ctx, `select id::text,status,attempts,last_error from education_portfolio_storage_transitions
		where operation_id=$1::uuid and institution_id=public.current_institution_id() and status in ('blocked','dead_letter') for update`, operationID)
	if err != nil {
		httpx.JSON(w, http.StatusInternalServerError, map[string]any{"code": "portfolio_lifecycle_retry_failed"})
		return
	}
	type retryTarget struct {
		id, status, lastError string
		attempts              int
	}
	targets := make([]retryTarget, 0)
	for rows.Next() {
		var target retryTarget
		if err := rows.Scan(&target.id, &target.status, &target.attempts, &target.lastError); err != nil {
			rows.Close()
			httpx.JSON(w, http.StatusInternalServerError, map[string]any{"code": "portfolio_lifecycle_retry_failed"})
			return
		}
		targets = append(targets, target)
	}
	rows.Close()
	if err := rows.Err(); err != nil {
		httpx.JSON(w, http.StatusInternalServerError, map[string]any{"code": "portfolio_lifecycle_retry_failed"})
		return
	}
	if len(targets) == 0 {
		httpx.JSON(w, http.StatusConflict, map[string]any{"code": "portfolio_lifecycle_retry_not_available"})
		return
	}
	for _, target := range targets {
		if target.lastError == "portfolio_retention_expired_review_required" {
			httpx.JSON(w, http.StatusConflict, map[string]any{"code": "portfolio_retention_expired_review_required"})
			return
		}
	}
	for _, target := range targets {
		if _, err := tx.Exec(ctx, `insert into education_portfolio_storage_transition_history(
			transition_id,tenant_code,institution_id,operation_id,previous_status,previous_attempts,reason,requested_by_subject)
			values($1::uuid,public.current_tenant_code(),public.current_institution_id(),$2::uuid,$3,$4,$5,$6)`,
			target.id, operationID, target.status, target.attempts, req.Reason, subject); err != nil {
			httpx.JSON(w, http.StatusInternalServerError, map[string]any{"code": "portfolio_lifecycle_retry_failed"})
			return
		}
		if _, err := tx.Exec(ctx, `update education_portfolio_storage_transitions set status='pending',attempts=0,available_at=now(),
			locked_at=null,locked_by='',last_error='',dead_lettered_at=null where id=$1::uuid and status in ('blocked','dead_letter')`, target.id); err != nil {
			httpx.JSON(w, http.StatusInternalServerError, map[string]any{"code": "portfolio_lifecycle_retry_failed"})
			return
		}
	}
	if _, err := tx.Exec(ctx, `update education_portfolio_lifecycle_operations set status='pending',completed_at=null,last_error='' where id=$1::uuid`, operationID); err != nil {
		httpx.JSON(w, http.StatusInternalServerError, map[string]any{"code": "portfolio_lifecycle_retry_failed"})
		return
	}
	if err := audit.Log(ctx, tx, audit.Event{ActorSubject: subject, Action: "education.portfolios.storage_lifecycle_retry", TargetType: "portfolio_lifecycle_operation", TargetID: operationID, Summary: "Portfolio storage lifecycle recovery requested.", Details: map[string]any{"portfolio_id": recordID, "reason": req.Reason, "transition_count": len(targets), "previous_status": status}}); err != nil {
		httpx.JSON(w, http.StatusInternalServerError, map[string]any{"code": "portfolio_lifecycle_retry_failed"})
		return
	}
	if err := tx.Commit(ctx); err != nil {
		httpx.JSON(w, http.StatusInternalServerError, map[string]any{"code": "portfolio_lifecycle_retry_failed"})
		return
	}
	response, err := s.loadPortfolioLifecycleResponse(ctx, recordID, operationID)
	if err != nil {
		httpx.JSON(w, http.StatusInternalServerError, map[string]any{"code": "portfolio_lifecycle_retry_failed"})
		return
	}
	httpx.JSON(w, http.StatusAccepted, response)
}

func (s *Service) createPortfolioLifecycleOperation(r *http.Request, recordID, operationType, cessationDate string, requestedHold *bool, reason string) (PortfolioLifecycleOperationResponse, int, error) {
	if uuid.Validate(recordID) != nil {
		return PortfolioLifecycleOperationResponse{}, http.StatusNotFound, pgx.ErrNoRows
	}
	ctx := r.Context()
	subject := strings.TrimSpace(authruntime.CurrentSubjectFromRequest(r))
	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return PortfolioLifecycleOperationResponse{}, http.StatusInternalServerError, fmt.Errorf("begin portfolio lifecycle command: %w", err)
	}
	defer tx.Rollback(ctx) //nolint:errcheck

	allowed, err := portfolioLifecycleManageAllowed(ctx, tx, subject)
	if err != nil {
		return PortfolioLifecycleOperationResponse{}, http.StatusInternalServerError, err
	}
	if !allowed {
		return PortfolioLifecycleOperationResponse{}, http.StatusForbidden, errPortfolioLifecycleForbidden
	}

	var portfolio PortfolioRecord
	err = scanPortfolioRecord(tx.QueryRow(ctx, `select `+portfolioRecordColumns+` from education_portfolios
		where id=$1::uuid and institution_id=public.current_institution_id() and withdrawn_at is null for update`, recordID), &portfolio)
	if errors.Is(err, pgx.ErrNoRows) {
		return PortfolioLifecycleOperationResponse{}, http.StatusNotFound, err
	}
	if err != nil {
		return PortfolioLifecycleOperationResponse{}, http.StatusInternalServerError, fmt.Errorf("lock portfolio lifecycle record: %w", err)
	}

	if operationType == "cessation_retention" && portfolio.ActivityCeasedOn != "" {
		if portfolio.ActivityCeasedOn != cessationDate {
			return PortfolioLifecycleOperationResponse{}, http.StatusConflict, errPortfolioLifecycleConflict
		}
		var storedReason, operationID string
		err = tx.QueryRow(ctx, `select reason,id::text from education_portfolio_lifecycle_operations
			where institution_id=public.current_institution_id() and portfolio_id=$1::uuid and operation_type='cessation_retention'`, recordID).Scan(&storedReason, &operationID)
		if err != nil || storedReason != reason {
			return PortfolioLifecycleOperationResponse{}, http.StatusConflict, errPortfolioLifecycleConflict
		}
		response, err := loadPortfolioLifecycleResponse(ctx, tx, recordID, operationID)
		if err != nil {
			return PortfolioLifecycleOperationResponse{}, http.StatusInternalServerError, err
		}
		if err := tx.Commit(ctx); err != nil {
			return PortfolioLifecycleOperationResponse{}, http.StatusInternalServerError, err
		}
		return response, http.StatusAccepted, nil
	}

	if operationType == "legal_hold_reconcile" {
		if requestedHold == nil {
			return PortfolioLifecycleOperationResponse{}, http.StatusBadRequest, fmt.Errorf("missing legal hold state")
		}
		var existingID string
		err = tx.QueryRow(ctx, `select id::text from education_portfolio_lifecycle_operations
			where institution_id=public.current_institution_id() and portfolio_id=$1::uuid and operation_type='legal_hold_reconcile'
			  and requested_legal_hold=$2 and reason=$3
			order by requested_at desc limit 1`, recordID, *requestedHold, reason).Scan(&existingID)
		if err == nil && portfolio.LegalHoldActive == *requestedHold {
			response, loadErr := loadPortfolioLifecycleResponse(ctx, tx, recordID, existingID)
			if loadErr != nil {
				return PortfolioLifecycleOperationResponse{}, http.StatusInternalServerError, loadErr
			}
			if err := tx.Commit(ctx); err != nil {
				return PortfolioLifecycleOperationResponse{}, http.StatusInternalServerError, err
			}
			return response, http.StatusAccepted, nil
		}
		if err != nil && !errors.Is(err, pgx.ErrNoRows) {
			return PortfolioLifecycleOperationResponse{}, http.StatusInternalServerError, err
		}
	}

	if operationType == "cessation_retention" {
		err = scanPortfolioRecord(tx.QueryRow(ctx, `update education_portfolios set activity_ceased_on=$1::date,
			activity_cessation_reason=$2,updated_at=now() where id=$3::uuid and institution_id=public.current_institution_id()
			and activity_ceased_on is null and withdrawn_at is null returning `+portfolioRecordColumns, cessationDate, reason, recordID), &portfolio)
	} else {
		err = scanPortfolioRecord(tx.QueryRow(ctx, `update education_portfolios set legal_hold_active=$1,legal_hold_reason=$2,
			legal_hold_set_at=now(),legal_hold_set_by_subject=$3,updated_at=now()
			where id=$4::uuid and institution_id=public.current_institution_id() and withdrawn_at is null returning `+portfolioRecordColumns,
			*requestedHold, reason, subject, recordID), &portfolio)
	}
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return PortfolioLifecycleOperationResponse{}, http.StatusConflict, errPortfolioLifecycleConflict
		}
		return PortfolioLifecycleOperationResponse{}, http.StatusInternalServerError, fmt.Errorf("persist portfolio lifecycle intent: %w", err)
	}

	operationID := uuid.NewString()
	var tenantCode string
	if err := tx.QueryRow(ctx, `select public.current_tenant_code()`).Scan(&tenantCode); err != nil || strings.TrimSpace(tenantCode) == "" {
		return PortfolioLifecycleOperationResponse{}, http.StatusInternalServerError, fmt.Errorf("read lifecycle tenant context: %w", err)
	}
	_, err = tx.Exec(ctx, `insert into education_portfolio_lifecycle_operations(
		id,tenant_code,institution_id,portfolio_id,operation_type,activity_ceased_on,retention_through,requested_legal_hold,reason,requested_by_subject)
		values($1::uuid,$2,public.current_institution_id(),$3::uuid,$4,nullif($5,'')::date,
			case when $4='cessation_retention' then $6::date else null end,$7,$8,$9)`,
		operationID, tenantCode, recordID, operationType, cessationDate, nullIfEmpty(portfolio.RetentionUntil), requestedHold, reason, subject)
	if err != nil {
		return PortfolioLifecycleOperationResponse{}, http.StatusInternalServerError, fmt.Errorf("insert portfolio lifecycle operation: %w", err)
	}

	var invalidSnapshots bool
	if err := tx.QueryRow(ctx, `select exists(
		select 1 from education_portfolio_documents pd
		where pd.institution_id=public.current_institution_id() and pd.portfolio_id=$1::uuid and pd.archive_version_id is not null
		and not exists(select 1 from archive_document_versions v where v.institution_id=pd.institution_id
			and v.id=pd.archive_version_id and v.document_id=pd.archive_document_id and v.version_no=pd.archive_version_no
			and v.source_bucket=pd.archive_source_bucket and v.source_object_key=pd.archive_source_object_key
			and lower(v.source_sha256)=lower(pd.archive_sha256) and btrim(v.source_object_version_id)<>''
			and btrim(v.source_object_etag)<>'' and v.source_size_bytes>0))`, recordID).Scan(&invalidSnapshots); err != nil {
		return PortfolioLifecycleOperationResponse{}, http.StatusInternalServerError, fmt.Errorf("validate lifecycle archive snapshots: %w", err)
	}
	if invalidSnapshots {
		return PortfolioLifecycleOperationResponse{}, http.StatusUnprocessableEntity, errPortfolioLifecycleProvenance
	}

	tag, err := tx.Exec(ctx, `insert into education_portfolio_storage_transitions(
		id,operation_id,tenant_code,institution_id,portfolio_id,archive_document_id,archive_version_id,
		source_bucket,source_object_key,source_object_version_id,source_object_etag,source_sha256,source_size_bytes,required_retention_until)
		select gen_random_uuid(),$1::uuid,$2,pd.institution_id,pd.portfolio_id,pd.archive_document_id,pd.archive_version_id,
		v.source_bucket,v.source_object_key,v.source_object_version_id,v.source_object_etag,lower(v.source_sha256),v.source_size_bytes,
		case when p.retention_until is null then null else (p.retention_until+1)::timestamp at time zone 'UTC' end
		from (select distinct institution_id,portfolio_id,archive_document_id,archive_version_id from education_portfolio_documents
			where institution_id=public.current_institution_id() and portfolio_id=$3::uuid and archive_version_id is not null) pd
		join education_portfolios p on p.institution_id=pd.institution_id and p.id=pd.portfolio_id
		join archive_document_versions v on v.institution_id=pd.institution_id and v.id=pd.archive_version_id and v.document_id=pd.archive_document_id`,
		operationID, tenantCode, recordID)
	if err != nil {
		return PortfolioLifecycleOperationResponse{}, http.StatusInternalServerError, fmt.Errorf("snapshot portfolio storage transitions: %w", err)
	}
	if tag.RowsAffected() == 0 {
		if _, err := tx.Exec(ctx, `update education_portfolio_lifecycle_operations set status='completed',completed_at=now() where id=$1::uuid`, operationID); err != nil {
			return PortfolioLifecycleOperationResponse{}, http.StatusInternalServerError, err
		}
	}

	action := "education.portfolios.activity_ceased"
	summary := "Portfolio cessation recorded; storage retention reconciliation queued."
	if operationType == "legal_hold_reconcile" {
		action = "education.portfolios.legal_hold"
		summary = "Portfolio legal-hold intent recorded; storage reconciliation queued."
	}
	if err := audit.Log(ctx, tx, audit.Event{ActorSubject: subject, Action: action, TargetType: "portfolio_lifecycle_operation", TargetID: operationID, Summary: summary, Details: map[string]any{"portfolio_id": recordID, "operation_type": operationType, "activity_ceased_on": cessationDate, "retention_through": portfolio.RetentionUntil, "requested_legal_hold": requestedHold}}); err != nil {
		return PortfolioLifecycleOperationResponse{}, http.StatusInternalServerError, fmt.Errorf("audit portfolio lifecycle operation: %w", err)
	}
	if err := tx.Commit(ctx); err != nil {
		return PortfolioLifecycleOperationResponse{}, http.StatusInternalServerError, fmt.Errorf("commit portfolio lifecycle operation: %w", err)
	}

	response, err := s.loadPortfolioLifecycleResponse(r.Context(), recordID, operationID)
	if err != nil {
		return PortfolioLifecycleOperationResponse{}, http.StatusInternalServerError, err
	}
	return response, http.StatusAccepted, nil
}

var (
	errPortfolioLifecycleForbidden  = errors.New("portfolio lifecycle access denied")
	errPortfolioLifecycleConflict   = errors.New("portfolio lifecycle command conflicts with persisted intent")
	errPortfolioLifecycleProvenance = errors.New("portfolio lifecycle archive provenance is incomplete")
)

func writePortfolioLifecycleError(w http.ResponseWriter, status int, err error) {
	code := "portfolio_lifecycle_failed"
	switch {
	case errors.Is(err, errPortfolioLifecycleForbidden):
		code = "education_portfolio_access_denied"
	case errors.Is(err, errPortfolioLifecycleConflict):
		code = "portfolio_lifecycle_conflict"
	case errors.Is(err, errPortfolioLifecycleProvenance):
		code = "portfolio_lifecycle_archive_provenance_required"
	case errors.Is(err, pgx.ErrNoRows):
		code = "portfolio_not_found"
	}
	httpx.JSON(w, status, map[string]any{"code": code})
}

func portfolioLifecycleManageAllowed(ctx context.Context, tx pgx.Tx, subject string) (bool, error) {
	if strings.TrimSpace(subject) == "" {
		return false, nil
	}
	allowed, err := educationSubjectHasPermission(ctx, tx, subject, portfolioLifecycleManagePermission)
	if err != nil || !allowed {
		return allowed, err
	}
	var active bool
	err = tx.QueryRow(ctx, `select exists(select 1 from app_users u join app_memberships m on m.user_id=u.id
		where lower(u.sub)=lower($1) and m.tenant_code=public.current_tenant_code() and m.active
		and m.start_date<=current_date and (m.end_date is null or m.end_date>=current_date))`, subject).Scan(&active)
	return active, err
}

func portfolioLifecycleReadAllowed(ctx context.Context, tx pgx.Tx, subject, portfolioID string) (bool, error) {
	for _, permission := range []string{"education.portfolios.school.read", "education.portfolios.read", "education.portfolios.manage"} {
		allowed, err := educationSubjectHasPermission(ctx, tx, subject, permission)
		if err != nil {
			return false, err
		}
		if allowed {
			return true, nil
		}
	}
	allowed, err := educationSubjectHasPermission(ctx, tx, subject, portfolioReadOwnPermission)
	if err != nil || !allowed {
		return allowed, err
	}
	var owns bool
	err = tx.QueryRow(ctx, `select exists(select 1 from education_portfolios p join app_users u on u.id=p.owner_user_id
		join education_personnel person on person.id=p.owner_personnel_id and person.app_user_id=u.id and person.institution_id=p.institution_id
		where p.id=$1::uuid and p.institution_id=public.current_institution_id() and lower(u.sub)=lower($2))`, portfolioID, subject).Scan(&owns)
	return owns, err
}

func (s *Service) loadPortfolioLifecycleResponse(ctx context.Context, portfolioID, operationID string) (PortfolioLifecycleOperationResponse, error) {
	return loadPortfolioLifecycleResponse(ctx, s.pool, portfolioID, operationID)
}

type portfolioLifecycleRow interface {
	QueryRow(context.Context, string, ...any) pgx.Row
	Query(context.Context, string, ...any) (pgx.Rows, error)
}

func loadPortfolioLifecycleResponse(ctx context.Context, db portfolioLifecycleRow, portfolioID, operationID string) (PortfolioLifecycleOperationResponse, error) {
	var response PortfolioLifecycleOperationResponse
	err := db.QueryRow(ctx, `select o.id::text,o.portfolio_id::text,o.operation_type,o.status,
		to_char(o.requested_at at time zone 'UTC','YYYY-MM-DD"T"HH24:MI:SS.MS"Z"'),
		coalesce(to_char(o.completed_at at time zone 'UTC','YYYY-MM-DD"T"HH24:MI:SS.MS"Z"'),''),
		count(s.id)::int,count(s.id) filter(where s.status='completed')::int,
		count(s.id) filter(where s.status in ('blocked','dead_letter'))::int,o.last_error
		from education_portfolio_lifecycle_operations o left join education_portfolio_storage_transitions s on s.operation_id=o.id
		where o.id=$1::uuid and o.portfolio_id=$2::uuid and o.institution_id=public.current_institution_id()
		group by o.id`, operationID, portfolioID).Scan(&response.Operation.ID, &response.Operation.PortfolioID,
		&response.Operation.Type, &response.Operation.Status, &response.Operation.RequestedAt,
		&response.Operation.CompletedAt, &response.Operation.TotalVersions, &response.Operation.CompletedVersions,
		&response.Operation.BlockedVersions, &response.Operation.LastError)
	if err != nil {
		return response, err
	}
	err = scanPortfolioRecord(db.QueryRow(ctx, `select `+portfolioRecordColumns+` from education_portfolios
		where id=$1::uuid and institution_id=public.current_institution_id()`, portfolioID), &response.Portfolio)
	if err != nil {
		return response, err
	}
	rows, err := db.Query(ctx, `select id::text,status,last_error,
		coalesce(to_char(required_retention_until at time zone 'UTC','YYYY-MM-DD"T"HH24:MI:SS.MS"Z"'),''),
		coalesce(to_char(completed_at at time zone 'UTC','YYYY-MM-DD"T"HH24:MI:SS.MS"Z"'),'')
		from education_portfolio_storage_transitions where operation_id=$1::uuid and portfolio_id=$2::uuid
		and institution_id=public.current_institution_id() order by created_at,id`, operationID, portfolioID)
	if err != nil {
		return response, err
	}
	defer rows.Close()
	response.Transitions = make([]PortfolioStorageTransition, 0)
	for rows.Next() {
		var transition PortfolioStorageTransition
		if err = rows.Scan(&transition.ID, &transition.Status, &transition.LastError, &transition.RequiredRetentionUntil, &transition.CompletedAt); err != nil {
			return response, err
		}
		response.Transitions = append(response.Transitions, transition)
	}
	return response, rows.Err()
}

func nullIfEmpty(value string) any {
	if strings.TrimSpace(value) == "" {
		return nil
	}
	return value
}
