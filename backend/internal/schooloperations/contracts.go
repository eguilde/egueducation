package schooloperations

import (
	"encoding/json"
	"errors"
	"net/http"
	"strings"
	"time"

	"github.com/eguilde/egueducation/internal/audit"
	"github.com/eguilde/egueducation/internal/httpx"
	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
)

func decodeContractRequest(w http.ResponseWriter, r *http.Request, output any) bool {
	decoder := json.NewDecoder(http.MaxBytesReader(w, r.Body, 128<<10))
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(output); err != nil {
		httpx.JSON(w, http.StatusUnprocessableEntity, map[string]any{"code": "invalid_school_contract_payload"})
		return false
	}
	return true
}

func contractScope(w http.ResponseWriter, r *http.Request, requirePolicy bool) (tenant, institutionID, actor, policyID string, ok bool) {
	tenant, institutionID, actor, ok = scope(r)
	if !ok {
		httpx.JSON(w, http.StatusUnauthorized, map[string]any{"code": "institution_context_required"})
		return
	}
	policyID = currentPolicy(r)
	if requirePolicy && policyID == "" {
		httpx.JSON(w, http.StatusConflict, map[string]any{"code": "policy_evaluation_required"})
		ok = false
	}
	return
}

// Contract returns a host-scoped record and never exposes a foreign tenant's ID.
func (s *Service) Contract(w http.ResponseWriter, r *http.Request) {
	tenant, institutionID, _, _, ok := contractScope(w, r, false)
	if !ok {
		return
	}
	id := strings.TrimSpace(chi.URLParam(r, "contractID"))
	if _, err := uuid.Parse(id); err != nil {
		httpx.JSON(w, http.StatusUnprocessableEntity, map[string]any{"code": "invalid_contract_id"})
		return
	}
	var result Contract
	var startsOn, endsOn *time.Time
	err := s.pool.QueryRow(r.Context(), `
select contract.id::text, contract.supplier_party_id::text, supplier.display_name,
       contract.contract_number, contract.title, contract.category, contract.lifecycle_status, contract.starts_on, contract.ends_on,
       contract.total_value::float8, contract.currency, contract.expected_version, contract.archive_status
from school_contracts contract
join app_parties supplier on supplier.tenant_code=contract.tenant_code and supplier.institution_id=contract.institution_id and supplier.id=contract.supplier_party_id
where contract.tenant_code=$1 and contract.institution_id=$2 and contract.id=$3::uuid`, tenant, institutionID, id).
		Scan(&result.ID, &result.SupplierPartyID, &result.SupplierName, &result.ContractNumber, &result.Title, &result.Category, &result.LifecycleStatus,
			&startsOn, &endsOn, &result.TotalValue, &result.Currency, &result.ExpectedVersion, &result.ArchiveStatus)
	if errors.Is(err, pgx.ErrNoRows) {
		httpx.JSON(w, http.StatusNotFound, map[string]any{"code": "school_contract_not_found"})
		return
	}
	if err != nil {
		httpx.JSON(w, http.StatusInternalServerError, map[string]any{"code": "school_contract_read_failed"})
		return
	}
	result.StartsOn = datePtr(startsOn)
	result.EndsOn = datePtr(endsOn)
	httpx.JSON(w, http.StatusOK, result)
}

func (s *Service) AmendContract(w http.ResponseWriter, r *http.Request) {
	tenant, institutionID, actor, policyID, ok := contractScope(w, r, true)
	if !ok {
		return
	}
	var input AmendContractRequest
	if !decodeContractRequest(w, r, &input) {
		return
	}
	input.Title = strings.TrimSpace(input.Title)
	input.Category = defaultString(input.Category, "general")
	input.StartsOn = strings.TrimSpace(input.StartsOn)
	input.EndsOn = strings.TrimSpace(input.EndsOn)
	input.Currency = strings.ToUpper(defaultString(input.Currency, "RON"))
	if input.ExpectedVersion < 1 || input.Title == "" || input.TotalValue < 0 || !validContractDates(input.StartsOn, input.EndsOn) || !validCurrency(input.Currency) {
		httpx.JSON(w, http.StatusUnprocessableEntity, map[string]any{"code": "invalid_contract_amendment"})
		return
	}
	id := strings.TrimSpace(chi.URLParam(r, "contractID"))
	if _, err := uuid.Parse(id); err != nil {
		httpx.JSON(w, http.StatusUnprocessableEntity, map[string]any{"code": "invalid_contract_id"})
		return
	}
	tx, err := s.pool.Begin(r.Context())
	if err != nil {
		httpx.JSON(w, 500, map[string]any{"code": "school_contract_amend_failed"})
		return
	}
	defer tx.Rollback(r.Context())
	tag, err := tx.Exec(r.Context(), `
update school_contracts set title=$1, category=$2, starts_on=nullif($3,'')::date,
 ends_on=nullif($4,'')::date, total_value=$5, currency=$6, expected_version=expected_version+1,
 policy_evaluation_id=$7::uuid, updated_by_subject=$8, updated_at=now()
where tenant_code=$9 and institution_id=$10 and id=$11::uuid and expected_version=$12
 and lifecycle_status in ('draft','verified','approved')`, input.Title, defaultString(input.Category, "general"),
		input.StartsOn, input.EndsOn, input.TotalValue, defaultString(input.Currency, "RON"), policyID, actor, tenant, institutionID, id, input.ExpectedVersion)
	if err != nil {
		httpx.JSON(w, http.StatusBadRequest, map[string]any{"code": "invalid_contract_amendment"})
		return
	}
	if tag.RowsAffected() != 1 {
		httpx.JSON(w, http.StatusConflict, map[string]any{"code": "contract_version_or_state_conflict"})
		return
	}
	if !s.writeContractEvidence(r, tx, tenant, institutionID, actor, "contract", id, "contract.amended", policyID) {
		httpx.JSON(w, 500, map[string]any{"code": "school_contract_evidence_failed"})
		return
	}
	if err = tx.Commit(r.Context()); err != nil {
		httpx.JSON(w, 500, map[string]any{"code": "school_contract_amend_failed"})
		return
	}
	httpx.JSON(w, http.StatusOK, map[string]any{"id": id, "expected_version": input.ExpectedVersion + 1})
}

func (s *Service) ListContractObligations(w http.ResponseWriter, r *http.Request) {
	tenant, institutionID, _, _, ok := contractScope(w, r, false)
	if !ok {
		return
	}
	contractID := strings.TrimSpace(chi.URLParam(r, "contractID"))
	if _, err := uuid.Parse(contractID); err != nil {
		httpx.JSON(w, http.StatusUnprocessableEntity, map[string]any{"code": "invalid_contract_id"})
		return
	}
	query := httpx.ParsePageQuery(r.URL.Query(), map[string]struct{}{"title": {}, "status": {}}, []string{"title", "status", "due_on"})
	where := " where tenant_code=$1 and institution_id=$2 and contract_id=$3::uuid"
	args := []any{tenant, institutionID, contractID}
	for _, field := range []string{"title", "status"} {
		if value := query.Filters[field]; value != "" {
			args = append(args, value)
			if field == "title" {
				where += " and title ilike '%' || $" + itoa(len(args)) + " || '%'"
			} else {
				where += " and status=$" + itoa(len(args))
			}
		}
	}
	var total int
	if err := s.pool.QueryRow(r.Context(), "select count(*) from school_contract_obligations"+where, args...).Scan(&total); err != nil {
		httpx.JSON(w, http.StatusBadRequest, map[string]any{"code": "invalid_contract_id"})
		return
	}
	sort := "due_on"
	if query.Sort != "" {
		sort = query.Sort
	}
	dir := "asc"
	if query.Direction == "desc" {
		dir = "desc"
	}
	args = append(args, query.PageSize, (query.Page-1)*query.PageSize)
	rows, err := s.pool.Query(r.Context(), "select id::text,contract_id::text,title,due_on,status,sla_hours,guarantee_value::float8,expected_version from school_contract_obligations"+where+" order by "+sort+" "+dir+",id limit $"+itoa(len(args)-1)+" offset $"+itoa(len(args)), args...)
	if err != nil {
		httpx.JSON(w, 500, map[string]any{"code": "school_obligation_list_failed"})
		return
	}
	defer rows.Close()
	items := []ContractObligation{}
	for rows.Next() {
		var item ContractObligation
		var due *time.Time
		if err = rows.Scan(&item.ID, &item.ContractID, &item.Title, &due, &item.Status, &item.SLAHours, &item.GuaranteeValue, &item.ExpectedVersion); err != nil {
			httpx.JSON(w, 500, map[string]any{"code": "school_obligation_list_failed"})
			return
		}
		item.DueOn = datePtr(due)
		items = append(items, item)
	}
	if err = rows.Err(); err != nil {
		httpx.JSON(w, 500, map[string]any{"code": "school_obligation_list_failed"})
		return
	}
	httpx.WritePage(w, 200, items, total, query.Page, query.PageSize)
}

func (s *Service) CreateContractObligation(w http.ResponseWriter, r *http.Request) {
	tenant, institutionID, actor, policyID, ok := contractScope(w, r, true)
	if !ok {
		return
	}
	var input CreateContractObligationRequest
	if !decodeContractRequest(w, r, &input) {
		return
	}
	if strings.TrimSpace(input.Title) == "" || !validOptionalDate(input.DueOn) || (input.GuaranteeValue != nil && *input.GuaranteeValue < 0) {
		httpx.JSON(w, http.StatusUnprocessableEntity, map[string]any{"code": "invalid_contract_obligation"})
		return
	}
	if input.SLAHours != nil && *input.SLAHours < 0 {
		httpx.JSON(w, 422, map[string]any{"code": "invalid_contract_obligation"})
		return
	}
	contractID := strings.TrimSpace(chi.URLParam(r, "contractID"))
	if _, err := uuid.Parse(contractID); err != nil {
		httpx.JSON(w, http.StatusUnprocessableEntity, map[string]any{"code": "invalid_contract_id"})
		return
	}
	tx, err := s.pool.Begin(r.Context())
	if err != nil {
		httpx.JSON(w, 500, map[string]any{"code": "school_obligation_create_failed"})
		return
	}
	defer tx.Rollback(r.Context())
	var id string
	err = tx.QueryRow(r.Context(), `insert into school_contract_obligations(tenant_code,institution_id,contract_id,title,due_on,sla_hours,guarantee_value,policy_evaluation_id,created_by_subject) select $1,$2,$3::uuid,$4,nullif($5,'')::date,$6,$7,$8::uuid,$9 where exists(select 1 from school_contracts where tenant_code=$1 and institution_id=$2 and id=$3::uuid) returning id::text`, tenant, institutionID, contractID, input.Title, input.DueOn, input.SLAHours, input.GuaranteeValue, policyID, actor).Scan(&id)
	if err != nil {
		httpx.JSON(w, 422, map[string]any{"code": "school_obligation_persist_failed"})
		return
	}
	if !s.writeContractEvidence(r, tx, tenant, institutionID, actor, "contract_obligation", id, "contract.obligation_created", policyID) {
		httpx.JSON(w, 500, map[string]any{"code": "school_obligation_evidence_failed"})
		return
	}
	if err = tx.Commit(r.Context()); err != nil {
		httpx.JSON(w, 500, map[string]any{"code": "school_obligation_create_failed"})
		return
	}
	httpx.JSON(w, 201, map[string]any{"id": id, "expected_version": 1})
}

func (s *Service) writeContractEvidence(r *http.Request, tx pgx.Tx, tenant, institutionID, actor, aggregateType, id, event, policyID string) bool {
	if _, err := tx.Exec(r.Context(), `insert into school_operations_outbox(tenant_code,institution_id,aggregate_type,aggregate_id,event_type,payload) values($1,$2,$3,$4::uuid,$5,jsonb_build_object('aggregate_id',$4,'policy_evaluation_id',$6))`, tenant, institutionID, aggregateType, id, event, policyID); err != nil {
		return false
	}
	return audit.Log(r.Context(), tx, audit.Event{ActorSubject: actor, Action: "school_operations." + event, TargetType: aggregateType, TargetID: id, Summary: "School operational evidence recorded"}) == nil
}
