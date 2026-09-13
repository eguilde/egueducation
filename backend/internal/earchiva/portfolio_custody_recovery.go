package earchiva

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"

	"github.com/eguilde/egueducation/internal/audit"
	authruntime "github.com/eguilde/egueducation/internal/auth"
	appdb "github.com/eguilde/egueducation/internal/db"
	"github.com/eguilde/egueducation/internal/httpx"
)

// PortfolioCustodyRecoveryService deliberately exposes no storage location,
// object identity, or raw worker error to the administrative API.
// RecoveryAdmin is kept separate from DocumentService: recovery authorization
// is administrator-only and never borrows the original upload authority.
type RecoveryAdmin struct{ pool *appdb.SessionPool }

type PortfolioCustodyRecoveryOperation struct {
	ID               string  `json:"operation_id"`
	IntentID         string  `json:"intent_id"`
	PortfolioID      string  `json:"portfolio_id"`
	Disposition      string  `json:"disposition"`
	Status           string  `json:"status"`
	Reason           string  `json:"reason"`
	Title            string  `json:"title"`
	OriginalFileName string  `json:"original_file_name"`
	DocumentDate     *string `json:"document_date,omitempty"`
	LastErrorCode    string  `json:"last_error_code,omitempty"`
	CreatedAt        string  `json:"created_at"`
	UpdatedAt        string  `json:"updated_at"`
}

// PortfolioCustodyRecoveryListItem represents either an unrequested stored
// intent (OperationID nil) or one durable recovery operation. It intentionally
// contains no bucket, object key, endpoint, ETag, metadata, or raw error.
type PortfolioCustodyRecoveryListItem struct {
	IntentID         string  `json:"intent_id"`
	OperationID      *string `json:"operation_id,omitempty"`
	PortfolioID      string  `json:"portfolio_id"`
	Status           string  `json:"status"`
	Disposition      *string `json:"disposition,omitempty"`
	Title            *string `json:"title,omitempty"`
	OriginalFileName *string `json:"original_file_name,omitempty"`
	DocumentDate     *string `json:"document_date,omitempty"`
	LastErrorCode    string  `json:"last_error_code,omitempty"`
	CreatedAt        string  `json:"created_at"`
	UpdatedAt        string  `json:"updated_at"`
}
type PortfolioCustodyRecoveryPage struct {
	Items    []PortfolioCustodyRecoveryListItem `json:"items"`
	Page     int                                `json:"page"`
	PageSize int                                `json:"pageSize"`
	Total    int64                              `json:"total"`
}
type ReconcilePortfolioCustodyRequest struct {
	Reason           string  `json:"reason"`
	Disposition      string  `json:"disposition"`
	Title            string  `json:"title"`
	OriginalFileName string  `json:"original_file_name"`
	DocumentDate     *string `json:"document_date,omitempty"`
}

func NewRecoveryAdmin(pool *appdb.SessionPool) *RecoveryAdmin { return &RecoveryAdmin{pool: pool} }

func (s *RecoveryAdmin) List(w http.ResponseWriter, r *http.Request) {
	institution, ok := s.authorized(w, r)
	if !ok {
		return
	}
	query, err := parsePortfolioCustodyRecoveryListQuery(r)
	if err != nil {
		httpx.JSON(w, http.StatusBadRequest, map[string]any{"code": "invalid_portfolio_custody_recovery_query"})
		return
	}
	filter := func(value string) string {
		if value == "" {
			return ""
		}
		return "%" + strings.NewReplacer("\\", "\\\\", "%", "\\%", "_", "\\_").Replace(value) + "%"
	}
	const records = `with records as (
		select o.id::text operation_id,i.id::text intent_id,i.portfolio_id::text portfolio_id,o.status,o.disposition,o.title,o.original_file_name,o.document_date,o.last_error_code,o.created_at,o.updated_at
		from portfolio_custody_recovery_operations o join portfolio_custody_upload_intents i on i.id=o.intent_id
		where o.institution_id=$1
		union all
		select null::text operation_id,i.id::text intent_id,i.portfolio_id::text portfolio_id,'stored'::text status,null::text disposition,null::text title,null::text original_file_name,null::date document_date,''::text last_error_code,i.created_at,i.updated_at
		from portfolio_custody_upload_intents i
		where i.institution_id=$1 and i.status='stored' and i.final_disposition is null
		and not exists (select 1 from portfolio_custody_recovery_operations o where o.intent_id=i.id)
	), filtered as (
		select * from records where ($2='' or status=$2) and ($3='' or disposition=$3)
		and ($4='' or intent_id=$4) and ($5='' or portfolio_id=$5)
		and ($6='' or title ilike $6 escape '\\') and ($7='' or original_file_name ilike $7 escape '\\')
		and ($8::timestamptz is null or created_at >= $8::timestamptz)
		and ($9::timestamptz is null or created_at <= $9::timestamptz)
	)`
	args := []any{institution, query.status, query.disposition, query.intentID, query.portfolioID, filter(query.title), filter(query.originalFileName), query.createdFrom, query.createdTo}
	var total int64
	if err := s.pool.QueryRow(r.Context(), records+` select count(*) from filtered`, args...).Scan(&total); err != nil {
		httpx.JSON(w, 500, map[string]any{"code": "portfolio_custody_recovery_list_failed"})
		return
	}
	rows, err := s.pool.Query(r.Context(), records+fmt.Sprintf(` select operation_id,intent_id,portfolio_id,status,disposition,title,original_file_name,case when document_date is null then null else to_char(document_date,'YYYY-MM-DD') end,last_error_code,to_char(created_at at time zone 'UTC','YYYY-MM-DD"T"HH24:MI:SS"Z"'),to_char(updated_at at time zone 'UTC','YYYY-MM-DD"T"HH24:MI:SS"Z"') from filtered order by %s %s, intent_id asc, coalesce(operation_id,'') asc limit $10 offset $11`, query.sort, query.direction), append(args, query.pageSize, (query.page-1)*query.pageSize)...)
	if err != nil {
		httpx.JSON(w, 500, map[string]any{"code": "portfolio_custody_recovery_list_failed"})
		return
	}
	defer rows.Close()
	items := []PortfolioCustodyRecoveryListItem{}
	for rows.Next() {
		var x PortfolioCustodyRecoveryListItem
		if err := rows.Scan(&x.OperationID, &x.IntentID, &x.PortfolioID, &x.Status, &x.Disposition, &x.Title, &x.OriginalFileName, &x.DocumentDate, &x.LastErrorCode, &x.CreatedAt, &x.UpdatedAt); err != nil {
			httpx.JSON(w, 500, map[string]any{"code": "portfolio_custody_recovery_list_failed"})
			return
		}
		items = append(items, x)
	}
	if err := rows.Err(); err != nil {
		httpx.JSON(w, 500, map[string]any{"code": "portfolio_custody_recovery_list_failed"})
		return
	}
	httpx.JSON(w, http.StatusOK, PortfolioCustodyRecoveryPage{Items: items, Page: query.page, PageSize: query.pageSize, Total: total})
}

type portfolioCustodyRecoveryListQuery struct {
	status, disposition, intentID, portfolioID, title, originalFileName, sort, direction string
	createdFrom, createdTo                                                               *time.Time
	page, pageSize                                                                       int
}

func parsePortfolioCustodyRecoveryListQuery(r *http.Request) (portfolioCustodyRecoveryListQuery, error) {
	values, err := url.ParseQuery(r.URL.RawQuery)
	if err != nil {
		return portfolioCustodyRecoveryListQuery{}, fmt.Errorf("parse query: %w", err)
	}
	allowed := map[string]bool{"page": true, "pageSize": true, "status": true, "disposition": true, "intent_id": true, "portfolio_id": true, "title": true, "original_file_name": true, "created_from": true, "created_to": true, "sort": true, "direction": true}
	for key, values := range values {
		if !allowed[key] || len(values) != 1 {
			return portfolioCustodyRecoveryListQuery{}, fmt.Errorf("invalid query parameter %q", key)
		}
	}
	get := func(key string) string { return strings.TrimSpace(values.Get(key)) }
	parsePage := func(key string, fallback, maximum int) (int, error) {
		raw := get(key)
		if raw == "" {
			return fallback, nil
		}
		value, err := strconv.Atoi(raw)
		if err != nil || value < 1 || value > maximum {
			return 0, fmt.Errorf("invalid %s", key)
		}
		return value, nil
	}
	page, err := parsePage("page", 1, 100000)
	if err != nil {
		return portfolioCustodyRecoveryListQuery{}, err
	}
	pageSize, err := parsePage("pageSize", 25, 100)
	if err != nil {
		return portfolioCustodyRecoveryListQuery{}, err
	}
	q := portfolioCustodyRecoveryListQuery{status: get("status"), disposition: get("disposition"), intentID: get("intent_id"), portfolioID: get("portfolio_id"), title: get("title"), originalFileName: get("original_file_name"), sort: get("sort"), direction: strings.ToLower(get("direction")), page: page, pageSize: pageSize}
	if q.status != "" && !recoveryStatus(q.status) {
		return q, fmt.Errorf("invalid status")
	}
	if q.disposition != "" && !validRecoveryDisposition(q.disposition) {
		return q, fmt.Errorf("invalid disposition")
	}
	for _, id := range []string{q.intentID, q.portfolioID} {
		if id != "" {
			if _, err := uuid.Parse(id); err != nil {
				return q, fmt.Errorf("invalid uuid")
			}
		}
	}
	parseTime := func(key string) (*time.Time, error) {
		raw := get(key)
		if raw == "" {
			return nil, nil
		}
		value, err := time.Parse(time.RFC3339, raw)
		if err != nil {
			return nil, fmt.Errorf("invalid %s", key)
		}
		return &value, nil
	}
	if q.createdFrom, err = parseTime("created_from"); err != nil {
		return q, err
	}
	if q.createdTo, err = parseTime("created_to"); err != nil {
		return q, err
	}
	if q.createdFrom != nil && q.createdTo != nil && q.createdFrom.After(*q.createdTo) {
		return q, fmt.Errorf("invalid date range")
	}
	if q.sort == "" {
		q.sort = "created_at"
	}
	if !map[string]bool{"created_at": true, "intent_id": true, "portfolio_id": true, "status": true, "disposition": true, "title": true, "original_file_name": true}[q.sort] {
		return q, fmt.Errorf("invalid sort")
	}
	if q.direction == "" {
		q.direction = "desc"
	}
	if q.direction != "asc" && q.direction != "desc" {
		return q, fmt.Errorf("invalid direction")
	}
	return q, nil
}

func (s *RecoveryAdmin) Reconcile(w http.ResponseWriter, r *http.Request) {
	institution, ok := s.authorized(w, r)
	if !ok {
		return
	}
	intentID := chi.URLParam(r, "intentID")
	if _, err := uuid.Parse(intentID); err != nil {
		httpx.JSON(w, 404, map[string]any{"code": "portfolio_custody_recovery_not_found"})
		return
	}
	var req ReconcilePortfolioCustodyRequest
	decoder := json.NewDecoder(http.MaxBytesReader(w, r.Body, 32<<10))
	decoder.DisallowUnknownFields()
	if decoder.Decode(&req) != nil || decoder.Decode(&struct{}{}) != io.EOF {
		httpx.JSON(w, 400, map[string]any{"code": "invalid_portfolio_custody_recovery_request"})
		return
	}
	req.Reason = strings.TrimSpace(req.Reason)
	req.Disposition = strings.TrimSpace(req.Disposition)
	req.Title = strings.TrimSpace(req.Title)
	req.OriginalFileName = strings.TrimSpace(req.OriginalFileName)
	if req.Reason == "" || req.Title == "" || req.OriginalFileName == "" || !validRecoveryDisposition(req.Disposition) {
		httpx.JSON(w, 400, map[string]any{"code": "invalid_portfolio_custody_recovery_request"})
		return
	}
	if req.DocumentDate != nil {
		v := strings.TrimSpace(*req.DocumentDate)
		if _, e := time.Parse(time.DateOnly, v); e != nil {
			httpx.JSON(w, 400, map[string]any{"code": "invalid_portfolio_custody_recovery_request"})
			return
		}
		req.DocumentDate = &v
	}
	tx, err := s.pool.Begin(r.Context())
	if err != nil {
		httpx.JSON(w, 500, map[string]any{"code": "portfolio_custody_recovery_failed"})
		return
	}
	defer tx.Rollback(r.Context()) //nolint:errcheck
	var storedFingerprint, checksum, mimeType string
	var sizeBytes int64
	err = tx.QueryRow(r.Context(), `select expected_request_fingerprint,expected_sha256,expected_size_bytes,expected_mime_type from portfolio_custody_upload_intents where id=$1::uuid and institution_id=$2 and status='stored' and final_disposition is null for update`, intentID, institution).Scan(&storedFingerprint, &checksum, &sizeBytes, &mimeType)
	if err == pgx.ErrNoRows {
		httpx.JSON(w, 404, map[string]any{"code": "portfolio_custody_recovery_not_found"})
		return
	}
	if err != nil {
		httpx.JSON(w, 500, map[string]any{"code": "portfolio_custody_recovery_failed"})
		return
	}
	expectedFingerprint := portfolioUploadFingerprint(archiveUploadPayload{Title: req.Title, DocumentDate: req.DocumentDate, FileName: req.OriginalFileName, ChecksumSHA256: checksum, FileSize: sizeBytes, MimeType: mimeType})
	if storedFingerprint != expectedFingerprint {
		httpx.JSON(w, 409, map[string]any{"code": "portfolio_custody_recovery_fingerprint_mismatch"})
		return
	}
	var out PortfolioCustodyRecoveryOperation
	err = tx.QueryRow(r.Context(), `select id::text,intent_id::text,portfolio_id::text,disposition,status,reason,title,original_file_name,case when document_date is null then null else to_char(document_date,'YYYY-MM-DD') end,last_error_code,to_char(created_at at time zone 'UTC','YYYY-MM-DD"T"HH24:MI:SS"Z"'),to_char(updated_at at time zone 'UTC','YYYY-MM-DD"T"HH24:MI:SS"Z"') from portfolio_custody_recovery_operations where intent_id=$1::uuid and status in ('queued','leased') order by created_at desc limit 1`, intentID).Scan(&out.ID, &out.IntentID, &out.PortfolioID, &out.Disposition, &out.Status, &out.Reason, &out.Title, &out.OriginalFileName, &out.DocumentDate, &out.LastErrorCode, &out.CreatedAt, &out.UpdatedAt)
	if err == nil {
		if out.Disposition != req.Disposition || out.Reason != req.Reason || out.Title != req.Title || out.OriginalFileName != req.OriginalFileName || !sameRecoveryDate(out.DocumentDate, req.DocumentDate) {
			httpx.JSON(w, http.StatusConflict, map[string]any{"code": "portfolio_custody_recovery_active_conflict"})
			return
		}
		if err = tx.Commit(r.Context()); err != nil {
			httpx.JSON(w, 500, map[string]any{"code": "portfolio_custody_recovery_failed"})
			return
		}
		httpx.JSON(w, http.StatusAccepted, out)
		return
	}
	if err != pgx.ErrNoRows {
		httpx.JSON(w, 500, map[string]any{"code": "portfolio_custody_recovery_failed"})
		return
	}
	err = tx.QueryRow(r.Context(), `insert into portfolio_custody_recovery_operations(intent_id,tenant_code,institution_id,portfolio_id,requested_by_subject,disposition,reason,title,original_file_name,document_date,expected_fingerprint) select i.id,i.tenant_code,i.institution_id,i.portfolio_id,$2,$3,$4,$5,$6,$7::date,$8 from portfolio_custody_upload_intents i where i.id=$1::uuid and i.institution_id=$9 returning id::text,intent_id::text,portfolio_id::text,disposition,status,reason,title,original_file_name,case when document_date is null then null else to_char(document_date,'YYYY-MM-DD') end,last_error_code,to_char(created_at at time zone 'UTC','YYYY-MM-DD"T"HH24:MI:SS"Z"'),to_char(updated_at at time zone 'UTC','YYYY-MM-DD"T"HH24:MI:SS"Z"')`, intentID, authruntime.CurrentSubjectFromRequest(r), req.Disposition, req.Reason, req.Title, req.OriginalFileName, req.DocumentDate, expectedFingerprint, institution).Scan(&out.ID, &out.IntentID, &out.PortfolioID, &out.Disposition, &out.Status, &out.Reason, &out.Title, &out.OriginalFileName, &out.DocumentDate, &out.LastErrorCode, &out.CreatedAt, &out.UpdatedAt)
	if err != nil {
		httpx.JSON(w, 500, map[string]any{"code": "portfolio_custody_recovery_failed"})
		return
	}
	if err = audit.Log(r.Context(), tx, audit.Event{ActorSubject: authruntime.CurrentSubjectFromRequest(r), Action: "earchiva.portfolio_custody.reconcile_requested", TargetType: "portfolio_custody_recovery_operation", TargetID: out.ID, Summary: "Custody recovery requested.", Details: map[string]any{"intent_id": intentID, "disposition": out.Disposition}}); err != nil {
		httpx.JSON(w, 500, map[string]any{"code": "portfolio_custody_recovery_failed"})
		return
	}
	if err = tx.Commit(r.Context()); err != nil {
		httpx.JSON(w, 500, map[string]any{"code": "portfolio_custody_recovery_failed"})
		return
	}
	httpx.JSON(w, 202, out)
}

func (s *RecoveryAdmin) GetOperation(w http.ResponseWriter, r *http.Request) {
	institution, ok := s.authorized(w, r)
	if !ok {
		return
	}
	id := chi.URLParam(r, "operationID")
	intentID := chi.URLParam(r, "intentID")
	if _, err := uuid.Parse(id); err != nil {
		httpx.JSON(w, http.StatusNotFound, map[string]any{"code": "portfolio_custody_recovery_not_found"})
		return
	}
	if _, err := uuid.Parse(intentID); err != nil {
		httpx.JSON(w, http.StatusNotFound, map[string]any{"code": "portfolio_custody_recovery_not_found"})
		return
	}
	var x PortfolioCustodyRecoveryOperation
	err := s.pool.QueryRow(r.Context(), `select o.id::text,o.intent_id::text,o.portfolio_id::text,o.disposition,o.status,o.reason,o.title,o.original_file_name,case when o.document_date is null then null else to_char(o.document_date,'YYYY-MM-DD') end,o.last_error_code,to_char(o.created_at at time zone 'UTC','YYYY-MM-DD"T"HH24:MI:SS"Z"'),to_char(o.updated_at at time zone 'UTC','YYYY-MM-DD"T"HH24:MI:SS"Z"') from portfolio_custody_recovery_operations o where o.id=$1::uuid and o.intent_id=$2::uuid and o.institution_id=$3`, id, chi.URLParam(r, "intentID"), institution).Scan(&x.ID, &x.IntentID, &x.PortfolioID, &x.Disposition, &x.Status, &x.Reason, &x.Title, &x.OriginalFileName, &x.DocumentDate, &x.LastErrorCode, &x.CreatedAt, &x.UpdatedAt)
	if err == pgx.ErrNoRows {
		httpx.JSON(w, 404, map[string]any{"code": "portfolio_custody_recovery_not_found"})
		return
	}
	if err != nil {
		httpx.JSON(w, 500, map[string]any{"code": "portfolio_custody_recovery_status_failed"})
		return
	}
	httpx.JSON(w, 200, x)
}
func validRecoveryDisposition(v string) bool {
	return v == "teacher_access" || v == "institution_archive_only"
}

func sameRecoveryDate(left, right *string) bool {
	if left == nil || right == nil {
		return left == nil && right == nil
	}
	return *left == *right
}
func recoveryStatus(v string) bool {
	for _, x := range []string{"stored", "queued", "leased", "committed", "blocked", "deadletter"} {
		if v == x {
			return true
		}
	}
	return false
}

// recoveryAdminAuthorized is intentionally a current-database check in
// addition to route middleware. Token claims can be stale after role removal.
func (s *RecoveryAdmin) authorized(w http.ResponseWriter, r *http.Request) (string, bool) {
	institution := strings.TrimSpace(authruntime.CurrentInstitutionIDFromRequest(r))
	subject := strings.TrimSpace(authruntime.CurrentSubjectFromRequest(r))
	if institution == "" || subject == "" {
		httpx.JSON(w, http.StatusForbidden, map[string]any{"code": "portfolio_custody_recovery_forbidden"})
		return "", false
	}
	tx, err := s.pool.Begin(r.Context())
	if err == nil {
		err = archiveAdmissionActorPermission(r.Context(), tx, subject, "earchiva.manage")
	}
	if err == nil {
		err = archiveAdmissionActorPermission(r.Context(), tx, subject, "education.portfolios.custody.manage")
	}
	if tx != nil {
		_ = tx.Rollback(r.Context())
	}
	if err != nil {
		httpx.JSON(w, http.StatusForbidden, map[string]any{"code": "portfolio_custody_recovery_forbidden"})
		return "", false
	}
	return institution, true
}
