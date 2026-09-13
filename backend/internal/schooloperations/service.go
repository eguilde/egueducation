// Package schooloperations owns tenant-scoped operational evidence for a school.
package schooloperations

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"net/http"
	"strings"
	"time"

	"github.com/eguilde/egueducation/internal/audit"
	"github.com/eguilde/egueducation/internal/auth"
	appdb "github.com/eguilde/egueducation/internal/db"
	"github.com/eguilde/egueducation/internal/httpx"
	"github.com/eguilde/egueducation/internal/institution"
	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
)

type Service struct{ pool *appdb.SessionPool }

func NewService(pool *appdb.SessionPool) *Service { return &Service{pool: pool} }

func scope(r *http.Request) (string, string, string, bool) {
	t := strings.TrimSpace(auth.CurrentTenantCodeFromRequest(r))
	i := strings.TrimSpace(auth.CurrentInstitutionIDFromRequest(r))
	a := strings.TrimSpace(auth.CurrentSubjectFromRequest(r))
	return t, i, a, t != "" && i != "" && a != ""
}
func currentPolicy(r *http.Request) string {
	return strings.TrimSpace(institution.CurrentPolicyEvaluationIDFromRequest(r))
}
func datePtr(v *time.Time) *string {
	if v == nil {
		return nil
	}
	x := v.Format(time.DateOnly)
	return &x
}
func (s *Service) ListContracts(w http.ResponseWriter, r *http.Request) {
	t, i, _, ok := scope(r)
	if !ok {
		httpx.JSON(w, 401, map[string]any{"code": "institution_context_required"})
		return
	}
	q := httpx.ParsePageQuery(r.URL.Query(), map[string]struct{}{"contract_number": {}, "supplier_name": {}, "title": {}, "category": {}, "lifecycle_status": {}, "starts_on": {}, "ends_on": {}, "archive_status": {}}, []string{"contract_number", "supplier_name", "title", "category", "lifecycle_status", "starts_on", "ends_on", "total_value", "archive_status"})
	where := " where contract.tenant_code=$1 and contract.institution_id=$2"
	args := []any{t, i}
	for _, f := range []string{"contract_number", "supplier_name", "title", "category", "lifecycle_status", "starts_on", "ends_on", "archive_status"} {
		if v := q.Filters[f]; v != "" {
			if (f == "starts_on" || f == "ends_on") && !validOptionalDate(v) {
				httpx.JSON(w, http.StatusUnprocessableEntity, map[string]any{"code": "invalid_contract_date_filter"})
				return
			}
			args = append(args, v)
			if f == "title" || f == "contract_number" {
				where += " and contract." + f + " ilike '%' || $" + itoa(len(args)) + " || '%'"
			} else if f == "supplier_name" {
				where += " and supplier.display_name ilike '%' || $" + itoa(len(args)) + " || '%'"
			} else if f == "starts_on" || f == "ends_on" {
				where += " and contract." + f + "=$" + itoa(len(args)) + "::date"
			} else {
				where += " and contract." + f + "=$" + itoa(len(args))
			}
		}
	}
	var total int
	from := " from school_contracts contract join app_parties supplier on supplier.tenant_code=contract.tenant_code and supplier.institution_id=contract.institution_id and supplier.id=contract.supplier_party_id"
	if err := s.pool.QueryRow(r.Context(), "select count(*)"+from+where, args...).Scan(&total); err != nil {
		httpx.JSON(w, 500, map[string]any{"code": "school_operations_list_failed"})
		return
	}
	sortColumns := map[string]string{"contract_number": "contract.contract_number", "supplier_name": "supplier.display_name", "title": "contract.title", "category": "contract.category", "lifecycle_status": "contract.lifecycle_status", "starts_on": "contract.starts_on", "ends_on": "contract.ends_on", "total_value": "contract.total_value", "archive_status": "contract.archive_status"}
	sort := sortColumns["contract_number"]
	if q.Sort != "" {
		sort = sortColumns[q.Sort]
	}
	dir := "asc"
	if q.Direction == "desc" {
		dir = "desc"
	}
	args = append(args, q.PageSize, (q.Page-1)*q.PageSize)
	rows, err := s.pool.Query(r.Context(), "select contract.id::text,contract.supplier_party_id::text,supplier.display_name,contract.contract_number,contract.title,contract.category,contract.lifecycle_status,contract.starts_on,contract.ends_on,contract.total_value::float8,contract.currency,contract.expected_version,contract.archive_status"+from+where+" order by "+sort+" "+dir+", contract.id limit $"+itoa(len(args)-1)+" offset $"+itoa(len(args)), args...)
	if err != nil {
		httpx.JSON(w, 500, map[string]any{"code": "school_operations_list_failed"})
		return
	}
	defer rows.Close()
	out := []Contract{}
	for rows.Next() {
		var x Contract
		var a, b *time.Time
		if err = rows.Scan(&x.ID, &x.SupplierPartyID, &x.SupplierName, &x.ContractNumber, &x.Title, &x.Category, &x.LifecycleStatus, &a, &b, &x.TotalValue, &x.Currency, &x.ExpectedVersion, &x.ArchiveStatus); err != nil {
			httpx.JSON(w, 500, map[string]any{"code": "school_operations_list_failed"})
			return
		}
		x.StartsOn = datePtr(a)
		x.EndsOn = datePtr(b)
		out = append(out, x)
	}
	if err = rows.Err(); err != nil {
		httpx.JSON(w, 500, map[string]any{"code": "school_operations_list_failed"})
		return
	}
	httpx.WritePage(w, 200, out, total, q.Page, q.PageSize)
}

func validOptionalDate(value string) bool {
	if strings.TrimSpace(value) == "" {
		return true
	}
	_, err := time.Parse(time.DateOnly, strings.TrimSpace(value))
	return err == nil
}

func validContractDates(startsOn, endsOn string) bool {
	if !validOptionalDate(startsOn) || !validOptionalDate(endsOn) || strings.TrimSpace(startsOn) == "" {
		return false
	}
	if strings.TrimSpace(endsOn) == "" {
		return true
	}
	start, _ := time.Parse(time.DateOnly, strings.TrimSpace(startsOn))
	end, _ := time.Parse(time.DateOnly, strings.TrimSpace(endsOn))
	return !end.Before(start)
}
func itoa(v int) string { return stringInt(v) }
func stringInt(v int) string {
	if v == 0 {
		return "0"
	}
	b := make([]byte, 0, 10)
	for v > 0 {
		b = append([]byte{byte('0' + v%10)}, b...)
		v /= 10
	}
	return string(b)
}
func (s *Service) CreateContract(w http.ResponseWriter, r *http.Request) {
	t, i, a, ok := scope(r)
	if !ok {
		httpx.JSON(w, 401, map[string]any{"code": "institution_context_required"})
		return
	}
	p := currentPolicy(r)
	if p == "" {
		httpx.JSON(w, 409, map[string]any{"code": "policy_evaluation_required"})
		return
	}
	var in CreateContractRequest
	d := json.NewDecoder(http.MaxBytesReader(w, r.Body, 128<<10))
	d.DisallowUnknownFields()
	if d.Decode(&in) != nil {
		httpx.JSON(w, 422, map[string]any{"code": "invalid_school_contract_payload"})
		return
	}
	normalizeCreateContract(&in)
	if in.IdempotencyKey == "" || in.ContractNumber == "" || in.Title == "" || in.TotalValue < 0 || !validContractDates(in.StartsOn, in.EndsOn) || !validCurrency(in.Currency) {
		httpx.JSON(w, 422, map[string]any{"code": "invalid_school_contract_payload"})
		return
	}
	if _, err := uuid.Parse(strings.TrimSpace(in.SupplierPartyID)); err != nil {
		httpx.JSON(w, http.StatusUnprocessableEntity, map[string]any{"code": "invalid_supplier_party_id"})
		return
	}
	fpRaw, _ := json.Marshal(in)
	h := sha256.Sum256(fpRaw)
	fp := hex.EncodeToString(h[:])
	tx, err := s.pool.Begin(r.Context())
	if err != nil {
		httpx.JSON(w, 500, map[string]any{"code": "school_contract_create_failed"})
		return
	}
	defer tx.Rollback(r.Context())
	var existing, existingFingerprint string
	err = tx.QueryRow(r.Context(), "select response_id::text,request_fingerprint from school_operation_idempotency where tenant_code=$1 and institution_id=$2 and operation='contract.create' and idempotency_key=$3 for update", t, i, in.IdempotencyKey).Scan(&existing, &existingFingerprint)
	if err == nil {
		if existingFingerprint != fp {
			httpx.JSON(w, http.StatusConflict, map[string]any{"code": "idempotency_key_payload_conflict"})
			return
		}
		if err := tx.Commit(r.Context()); err != nil {
			httpx.JSON(w, 500, map[string]any{"code": "school_contract_create_failed"})
			return
		}
		httpx.JSON(w, http.StatusCreated, map[string]any{"id": existing, "expected_version": 1, "archive_status": "archive_pending"})
		return
	}
	if !errors.Is(err, pgx.ErrNoRows) {
		httpx.JSON(w, 500, map[string]any{"code": "school_contract_create_failed"})
		return
	}
	// Reserve idempotency before the aggregate. ON CONFLICT avoids a failed
	// transaction during a concurrent replay, then the winner's closed result
	// is returned after its transaction commits.
	var id string
	if err = tx.QueryRow(r.Context(), "select gen_random_uuid()::text").Scan(&id); err != nil {
		httpx.JSON(w, 500, map[string]any{"code": "school_contract_create_failed"})
		return
	}
	err = tx.QueryRow(r.Context(), "insert into school_operation_idempotency(tenant_code,institution_id,operation,idempotency_key,response_id,request_fingerprint,created_by_subject) values($1,$2,'contract.create',$3,$4::uuid,$5,$6) on conflict do nothing returning response_id::text", t, i, in.IdempotencyKey, id, fp, a).Scan(&id)
	if errors.Is(err, pgx.ErrNoRows) {
		if err = tx.QueryRow(r.Context(), "select response_id::text,request_fingerprint from school_operation_idempotency where tenant_code=$1 and institution_id=$2 and operation='contract.create' and idempotency_key=$3 for update", t, i, in.IdempotencyKey).Scan(&existing, &existingFingerprint); err != nil {
			httpx.JSON(w, 500, map[string]any{"code": "school_contract_create_failed"})
			return
		}
		if existingFingerprint != fp {
			httpx.JSON(w, 409, map[string]any{"code": "idempotency_key_payload_conflict"})
			return
		}
		if err = tx.Commit(r.Context()); err != nil {
			httpx.JSON(w, 500, map[string]any{"code": "school_contract_create_failed"})
			return
		}
		httpx.JSON(w, http.StatusCreated, map[string]any{"id": existing, "expected_version": 1, "archive_status": "archive_pending"})
		return
	}
	if err != nil {
		httpx.JSON(w, 500, map[string]any{"code": "school_contract_create_failed"})
		return
	}
	err = tx.QueryRow(r.Context(), `insert into school_contracts(id,tenant_code,institution_id,supplier_party_id,contract_number,title,category,starts_on,ends_on,total_value,currency,policy_evaluation_id,created_by_subject,updated_by_subject) select $1::uuid,$2,$3,supplier.id,$5,$6,$7,$8::date,nullif($9,'')::date,$10,$11,$12::uuid,$13,$13 from app_parties supplier where supplier.tenant_code=$2 and supplier.institution_id=$3 and supplier.id=$4::uuid and supplier.active returning id::text`, id, t, i, in.SupplierPartyID, in.ContractNumber, in.Title, defaultString(in.Category, "general"), in.StartsOn, in.EndsOn, in.TotalValue, defaultString(in.Currency, "RON"), p, a).Scan(&id)
	if err != nil {
		httpx.JSON(w, 422, map[string]any{"code": "school_contract_persist_failed"})
		return
	}
	if _, err = tx.Exec(r.Context(), "insert into school_operations_outbox(tenant_code,institution_id,aggregate_type,aggregate_id,event_type,payload) values($1,$2,'contract',$3::uuid,'contract.archive_pending',jsonb_build_object('contract_id',$3))", t, i, id); err != nil {
		httpx.JSON(w, 500, map[string]any{"code": "school_contract_outbox_failed"})
		return
	}
	if err = audit.Log(r.Context(), tx, audit.Event{ActorSubject: a, Action: "school_operations.contract.created", TargetType: "school_contract", TargetID: id, Summary: "Operational contract draft created"}); err != nil {
		httpx.JSON(w, 500, map[string]any{"code": "school_contract_audit_failed"})
		return
	}
	if err = tx.Commit(r.Context()); err != nil {
		httpx.JSON(w, 500, map[string]any{"code": "school_contract_create_failed"})
		return
	}
	httpx.JSON(w, 201, map[string]any{"id": id, "expected_version": 1, "archive_status": "archive_pending"})
}
func defaultString(v, d string) string {
	if strings.TrimSpace(v) == "" {
		return d
	}
	return strings.TrimSpace(v)
}

func validCurrency(value string) bool {
	value = strings.TrimSpace(value)
	if len(value) != 3 {
		return false
	}
	for _, character := range value {
		if character < 'A' || character > 'Z' {
			return false
		}
	}
	return true
}

func normalizeCreateContract(input *CreateContractRequest) {
	input.SupplierPartyID = strings.TrimSpace(input.SupplierPartyID)
	input.ContractNumber = strings.TrimSpace(input.ContractNumber)
	input.Title = strings.TrimSpace(input.Title)
	input.Category = defaultString(input.Category, "general")
	input.StartsOn = strings.TrimSpace(input.StartsOn)
	input.EndsOn = strings.TrimSpace(input.EndsOn)
	input.Currency = strings.ToUpper(defaultString(input.Currency, "RON"))
	input.IdempotencyKey = strings.TrimSpace(input.IdempotencyKey)
}
func (s *Service) TransitionContract(w http.ResponseWriter, r *http.Request) {
	t, i, a, ok := scope(r)
	if !ok {
		httpx.JSON(w, 401, map[string]any{"code": "institution_context_required"})
		return
	}
	p := currentPolicy(r)
	if p == "" {
		httpx.JSON(w, http.StatusConflict, map[string]any{"code": "policy_evaluation_required"})
		return
	}
	var in TransitionContractRequest
	d := json.NewDecoder(http.MaxBytesReader(w, r.Body, 32<<10))
	d.DisallowUnknownFields()
	if d.Decode(&in) != nil || in.ExpectedVersion < 1 || !validTransition(in.Status) {
		httpx.JSON(w, 422, map[string]any{"code": "invalid_contract_transition"})
		return
	}
	id := chi.URLParam(r, "contractID")
	if _, err := uuid.Parse(strings.TrimSpace(id)); err != nil {
		httpx.JSON(w, http.StatusUnprocessableEntity, map[string]any{"code": "invalid_contract_id"})
		return
	}
	tx, err := s.pool.Begin(r.Context())
	if err != nil {
		httpx.JSON(w, 500, map[string]any{"code": "school_contract_transition_failed"})
		return
	}
	defer tx.Rollback(r.Context())
	tag, err := tx.Exec(r.Context(), `update school_contracts set lifecycle_status=$1,expected_version=expected_version+1,policy_evaluation_id=$2::uuid,updated_by_subject=$3,updated_at=now() where tenant_code=$4 and institution_id=$5 and id=$6::uuid and expected_version=$7 and (($1='verified' and lifecycle_status='draft') or ($1='approved' and lifecycle_status='verified') or ($1='signed' and lifecycle_status='approved') or ($1='active' and lifecycle_status='signed' and starts_on<=current_date and (ends_on is null or ends_on>=current_date)) or ($1 in ('suspended','terminated') and lifecycle_status='active') or ($1='expired' and lifecycle_status in ('active','suspended') and ends_on<current_date) or ($1='archived' and lifecycle_status in ('expired','terminated') and archive_status='archived'))`, in.Status, p, a, t, i, id, in.ExpectedVersion)
	if err != nil {
		httpx.JSON(w, 500, map[string]any{"code": "school_contract_transition_failed"})
		return
	}
	if tag.RowsAffected() == 0 {
		httpx.JSON(w, 409, map[string]any{"code": "contract_transition_or_version_conflict"})
		return
	}
	if _, err = tx.Exec(r.Context(), "insert into school_operations_outbox(tenant_code,institution_id,aggregate_type,aggregate_id,event_type,payload) values($1,$2,'contract',$3::uuid,'contract.transitioned',jsonb_build_object('contract_id',$3,'status',$4,'policy_evaluation_id',$5))", t, i, id, in.Status, p); err != nil {
		httpx.JSON(w, 500, map[string]any{"code": "school_contract_outbox_failed"})
		return
	}
	if err = audit.Log(r.Context(), tx, audit.Event{ActorSubject: a, Action: "school_operations.contract.transitioned", TargetType: "school_contract", TargetID: id, Summary: "Contract lifecycle transitioned to " + in.Status}); err != nil {
		httpx.JSON(w, 500, map[string]any{"code": "school_contract_audit_failed"})
		return
	}
	if err = tx.Commit(r.Context()); err != nil {
		httpx.JSON(w, 500, map[string]any{"code": "school_contract_transition_failed"})
		return
	}
	httpx.JSON(w, 200, map[string]any{"id": id, "lifecycle_status": in.Status, "expected_version": in.ExpectedVersion + 1})
}
func validTransition(s string) bool {
	for _, v := range []string{"verified", "approved", "signed", "active", "suspended", "expired", "terminated", "archived"} {
		if s == v {
			return true
		}
	}
	return false
}
