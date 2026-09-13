package admission

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"strings"
	"time"

	"github.com/eguilde/egueducation/internal/httpx"
	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
)

func validateCampaign(in *CreateCampaignRequest) bool {
	in.Code, in.Title, in.SchoolYear = strings.TrimSpace(in.Code), strings.TrimSpace(in.Title), strings.TrimSpace(in.SchoolYear)
	in.CapacityUnit, in.Shift = strings.TrimSpace(in.CapacityUnit), strings.TrimSpace(in.Shift)
	if !codePattern.MatchString(in.Code) || in.Title == "" || in.SchoolYear == "" || !validUUID(in.SourceID) || !validUUID(in.OfferingID) || !validUUID(in.LocationID) || !validUUID(in.AuthorizationID) || !validUUID(in.ClassOfferingContextID) || in.CapacityLimit < 1 || in.StudentPlaceLimit < 1 || !validDate(in.OpensOn) || !validDate(in.ClosesOn) {
		return false
	}
	from, _ := time.Parse(time.DateOnly, in.OpensOn)
	to, _ := time.Parse(time.DateOnly, in.ClosesOn)
	if to.Before(from) {
		return false
	}
	if in.DecisionDueOn != nil && (!validDate(*in.DecisionDueOn) || strings.TrimSpace(*in.DecisionDueOn) < in.ClosesOn) {
		return false
	}
	if in.CapacityUnit != "students" && in.CapacityUnit != "study_groups" {
		return false
	}
	if in.Shift != "day" && in.Shift != "afternoon" && in.Shift != "evening" {
		return false
	}
	var basis map[string]any
	if json.Unmarshal(rawObject(in.CapacityBasis), &basis) != nil {
		return false
	}
	seenCode, seenOrdinal := map[string]bool{}, map[int]bool{}
	for i := range in.Criteria {
		c := &in.Criteria[i]
		c.Code = strings.TrimSpace(c.Code)
		c.Title = strings.TrimSpace(c.Title)
		if !codePattern.MatchString(c.Code) || c.Title == "" || c.Ordinal < 1 || seenCode[c.Code] || seenOrdinal[c.Ordinal] {
			return false
		}
		seenCode[c.Code] = true
		seenOrdinal[c.Ordinal] = true
		if c.Kind != "eligibility" && c.Kind != "priority" && c.Kind != "ranking" && c.Kind != "tie_breaker" {
			return false
		}
		var rule map[string]any
		if json.Unmarshal(rawObject(c.RuleSnapshot), &rule) != nil {
			return false
		}
	}
	seenCode, seenOrdinal = map[string]bool{}, map[int]bool{}
	for i := range in.DocumentRequirements {
		d := &in.DocumentRequirements[i]
		d.Code = strings.TrimSpace(d.Code)
		d.Title = strings.TrimSpace(d.Title)
		if !codePattern.MatchString(d.Code) || d.Title == "" || d.Ordinal < 1 || seenCode[d.Code] || seenOrdinal[d.Ordinal] {
			return false
		}
		seenCode[d.Code] = true
		seenOrdinal[d.Ordinal] = true
	}
	return true
}

func (s *Service) CreateCampaign(w http.ResponseWriter, r *http.Request) {
	sc, ok := requestScope(r)
	if !ok {
		writeError(w, errBadScope, "campaign_create_failed")
		return
	}
	key, ok := idempotencyKey(r)
	if !ok {
		httpx.JSON(w, http.StatusBadRequest, map[string]any{"code": "idempotency_key_required"})
		return
	}
	var in CreateCampaignRequest
	if decode(w, r, &in) != nil || !validateCampaign(&in) {
		httpx.JSON(w, http.StatusUnprocessableEntity, map[string]any{"code": "invalid_campaign"})
		return
	}
	tx, err := s.begin(r.Context(), sc)
	if err != nil {
		writeError(w, err, "campaign_create_failed")
		return
	}
	defer tx.Rollback(r.Context())
	if err = requirePermission(r.Context(), tx, sc, permissionManage); err != nil {
		writeError(w, err, "campaign_create_failed")
		return
	}
	id, replay, err := reserve(r.Context(), tx, sc, "admission.campaign.create", key, fingerprint(in), uuid.NewString())
	if err != nil {
		writeError(w, err, "campaign_create_failed")
		return
	}
	if !replay {
		_, err = tx.Exec(r.Context(), `insert into school_admission_campaigns(id,tenant_code,institution_id,source_id,code,title,school_year,offering_id,location_id,authorization_id,class_offering_context_id,capacity_limit,capacity_unit,student_place_limit,capacity_basis,shift,opens_on,closes_on,decision_due_on,status,created_by_subject,updated_by_subject)
			values($1::uuid,$2,$3,$4::uuid,$5,$6,$7,$8::uuid,$9::uuid,$10::uuid,$11::uuid,$12,$13,$14,$15::jsonb,$16,$17::date,$18::date,nullif($19,'')::date,'draft',$20,$20)`, id, sc.tenant, sc.institution, in.SourceID, in.Code, in.Title, in.SchoolYear, in.OfferingID, in.LocationID, in.AuthorizationID, in.ClassOfferingContextID, in.CapacityLimit, in.CapacityUnit, in.StudentPlaceLimit, rawObject(in.CapacityBasis), in.Shift, in.OpensOn, in.ClosesOn, optional(in.DecisionDueOn), sc.actor)
		for _, c := range in.Criteria {
			if err != nil {
				break
			}
			_, err = tx.Exec(r.Context(), `insert into school_admission_criteria(tenant_code,institution_id,campaign_id,code,title,criterion_kind,required,weight,ordinal,rule_snapshot,created_by_subject,updated_by_subject) values($1,$2,$3::uuid,$4,$5,$6,$7,$8,$9,$10::jsonb,$11,$11)`, sc.tenant, sc.institution, id, c.Code, c.Title, c.Kind, c.Required, c.Weight, c.Ordinal, rawObject(c.RuleSnapshot), sc.actor)
		}
		for _, d := range in.DocumentRequirements {
			if err != nil {
				break
			}
			_, err = tx.Exec(r.Context(), `insert into school_admission_document_requirements(tenant_code,institution_id,campaign_id,code,title,required,allowed_mime_types,ordinal,created_by_subject,updated_by_subject) values($1,$2,$3::uuid,$4,$5,$6,$7,$8,$9,$9)`, sc.tenant, sc.institution, id, d.Code, d.Title, d.Required, d.AllowedMIMETypes, d.Ordinal, sc.actor)
		}
		if err == nil {
			err = outbox(r.Context(), tx, sc, "admission_campaign", id, "admission.campaign.created", map[string]any{"campaignId": id, "status": "draft"})
		}
		if err == nil {
			err = auditEvent(r.Context(), tx, sc, "admission.campaign.created", "school_admission_campaign", id)
		}
	}
	if err != nil {
		writeError(w, err, "campaign_create_failed")
		return
	}
	if err = tx.Commit(r.Context()); err != nil {
		writeError(w, err, "campaign_create_failed")
		return
	}
	httpx.JSON(w, http.StatusCreated, CommandResult{ID: id, Status: "draft", ExpectedVersion: 1, Replayed: replay})
}

var campaignTransitions = map[string]map[string]bool{"draft": {"published": true, "cancelled": true}, "published": {"open": true, "cancelled": true}, "open": {"closed": true, "cancelled": true}, "closed": {"archived": true}, "cancelled": {"archived": true}}

func (s *Service) TransitionCampaign(w http.ResponseWriter, r *http.Request) {
	sc, ok := requestScope(r)
	if !ok {
		writeError(w, errBadScope, "campaign_transition_failed")
		return
	}
	id := chi.URLParam(r, "campaignID")
	if !validUUID(id) {
		httpx.JSON(w, 422, map[string]any{"code": "invalid_campaign_id"})
		return
	}
	key, ok := idempotencyKey(r)
	if !ok {
		httpx.JSON(w, 400, map[string]any{"code": "idempotency_key_required"})
		return
	}
	var in TransitionRequest
	if decode(w, r, &in) != nil || in.ExpectedVersion < 1 {
		httpx.JSON(w, 422, map[string]any{"code": "invalid_campaign_transition"})
		return
	}
	in.Status = strings.TrimSpace(in.Status)
	tx, err := s.begin(r.Context(), sc)
	if err != nil {
		writeError(w, err, "campaign_transition_failed")
		return
	}
	defer tx.Rollback(r.Context())
	if err = requirePermission(r.Context(), tx, sc, permissionManage); err != nil {
		writeError(w, err, "campaign_transition_failed")
		return
	}
	_, replay, err := reserve(r.Context(), tx, sc, "admission.campaign.transition", key, fingerprint(struct {
		ID    string
		Input TransitionRequest
	}{id, in}), id)
	if err != nil {
		writeError(w, err, "campaign_transition_failed")
		return
	}
	if !replay {
		var current string
		err = tx.QueryRow(r.Context(), `select status from school_admission_campaigns where tenant_code=$1 and institution_id=$2 and id=$3::uuid and expected_version=$4 for update`, sc.tenant, sc.institution, id, in.ExpectedVersion).Scan(&current)
		if errors.Is(err, pgx.ErrNoRows) {
			err = errVersionConflict
		}
		if err == nil && !campaignTransitions[current][in.Status] {
			err = errInvalidState
		}
		if err == nil && in.Status == "published" {
			var configured bool
			err = tx.QueryRow(r.Context(), `select exists(select 1 from school_admission_criteria where tenant_code=$1 and institution_id=$2 and campaign_id=$3::uuid and required and criterion_kind='eligibility') and exists(select 1 from school_admission_document_requirements where tenant_code=$1 and institution_id=$2 and campaign_id=$3::uuid and required)`, sc.tenant, sc.institution, id).Scan(&configured)
			if err == nil && !configured {
				err = errInvalidInput
			}
		}
		if err == nil && in.Status == "open" {
			var inWindow bool
			err = tx.QueryRow(r.Context(), `select current_date between opens_on and closes_on from school_admission_campaigns where tenant_code=$1 and institution_id=$2 and id=$3::uuid`, sc.tenant, sc.institution, id).Scan(&inWindow)
			if err == nil && !inWindow {
				err = errInvalidState
			}
		}
		if err == nil {
			tag, e := tx.Exec(r.Context(), `update school_admission_campaigns set status=$1,expected_version=expected_version+1,updated_by_subject=$2,updated_at=now() where tenant_code=$3 and institution_id=$4 and id=$5::uuid and expected_version=$6`, in.Status, sc.actor, sc.tenant, sc.institution, id, in.ExpectedVersion)
			err = e
			if err == nil && tag.RowsAffected() != 1 {
				err = errVersionConflict
			}
		}
		if err == nil {
			err = outbox(r.Context(), tx, sc, "admission_campaign", id, "admission.campaign."+in.Status, map[string]any{"campaign_id": id, "status": in.Status})
		}
		if err == nil {
			err = auditEvent(r.Context(), tx, sc, "admission.campaign."+in.Status, "school_admission_campaign", id)
		}
	}
	if err != nil {
		writeError(w, err, "campaign_transition_failed")
		return
	}
	if err = tx.Commit(r.Context()); err != nil {
		writeError(w, err, "campaign_transition_failed")
		return
	}
	httpx.JSON(w, 200, CommandResult{ID: id, Status: in.Status, ExpectedVersion: in.ExpectedVersion + 1, Replayed: replay})
}

func optional(v *string) string {
	if v == nil {
		return ""
	}
	return strings.TrimSpace(*v)
}
func loadCampaignContext(ctx context.Context, tx pgx.Tx, sc scope, id string) (offering, location, authorization, classContext, schoolYear, capacityUnit, shift string, err error) {
	err = tx.QueryRow(ctx, `select offering_id::text,location_id::text,authorization_id::text,class_offering_context_id::text,school_year,capacity_unit,shift from school_admission_campaigns where tenant_code=$1 and institution_id=$2 and id=$3::uuid`, sc.tenant, sc.institution, id).Scan(&offering, &location, &authorization, &classContext, &schoolYear, &capacityUnit, &shift)
	return
}
