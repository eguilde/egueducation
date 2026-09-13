package institution

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/eguilde/egueducation/internal/regulatorymodel"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
)

var (
	ErrPolicyV2CutoverRequired = errors.New("institution policy v2 cutover is required")
	ErrPolicyV2ContextInvalid  = errors.New("institution policy v2 context is invalid")
)

// OperationPolicyV2Request contains only operation-specific facts. Tenant and
// institution are deliberately absent: PostgreSQL derives them from the
// host-bound request session. The caller must invoke this function from the
// same transaction that persists the regulated business mutation.
type OperationPolicyV2Request struct {
	OperationCode           string
	EffectiveOn             time.Time
	DecisionKind            string
	OfferingID              string
	LocationID              string
	FundingInstrumentID     string
	ProcurementAssessmentID string
	RevalidatesEvaluationID string
	ActorSubject            string
	Context                 map[string]any
}

type OperationPolicyV2Decision struct {
	InputID      string
	EvaluationID string
	Allowed      bool
	Capabilities []string
	Obligations  []string
	Warnings     []string
}

type policyV2BindingEvidence struct {
	ID               string
	PackID           string
	PackChecksum     string
	AssignmentKind   string
	OverrideSnapshot json.RawMessage
	OverrideChecksum string
}

// EvaluateOperationPolicyV2Tx loads the exact effective regulatory graph,
// evaluates it with the pure fail-closed engine, and persists the input,
// binding provenance and result inside the caller's transaction. No commit is
// performed here, so a later business-write failure rolls back all evidence.
func EvaluateOperationPolicyV2Tx(ctx context.Context, tx pgx.Tx, request OperationPolicyV2Request) (OperationPolicyV2Decision, error) {
	if tx == nil {
		return OperationPolicyV2Decision{}, fmt.Errorf("%w: transaction is required", ErrPolicyV2ContextInvalid)
	}
	if err := validateOperationPolicyV2Request(request); err != nil {
		return OperationPolicyV2Decision{}, err
	}
	return evaluateOperationPolicyV2(ctx, tx, request)
}

func validateOperationPolicyV2Request(request OperationPolicyV2Request) error {
	request.OperationCode = strings.TrimSpace(request.OperationCode)
	request.DecisionKind = strings.TrimSpace(request.DecisionKind)
	request.ActorSubject = strings.TrimSpace(request.ActorSubject)
	if request.OperationCode == "" || request.EffectiveOn.IsZero() || request.ActorSubject == "" {
		return fmt.Errorf("%w: operation_code, effective_on and actor are required", ErrPolicyV2ContextInvalid)
	}
	if request.DecisionKind == "" {
		request.DecisionKind = "operation"
	}
	switch request.DecisionKind {
	case "operation", "publication", "contract", "compliance":
	default:
		return fmt.Errorf("%w: unsupported decision kind", ErrPolicyV2ContextInvalid)
	}
	if (request.OfferingID == "") != (request.LocationID == "") {
		return fmt.Errorf("%w: offering and location must be supplied together", ErrPolicyV2ContextInvalid)
	}
	for name, value := range map[string]string{
		"offering_id": request.OfferingID, "location_id": request.LocationID,
		"funding_instrument_id":     request.FundingInstrumentID,
		"procurement_assessment_id": request.ProcurementAssessmentID,
		"revalidates_evaluation_id": request.RevalidatesEvaluationID,
	} {
		if value != "" {
			if _, err := uuid.Parse(value); err != nil {
				return fmt.Errorf("%w: %s must be a UUID", ErrPolicyV2ContextInvalid, name)
			}
		}
	}
	return nil
}

func evaluateOperationPolicyV2(ctx context.Context, db queryer, request OperationPolicyV2Request) (OperationPolicyV2Decision, error) {
	decisionKind := strings.TrimSpace(request.DecisionKind)
	if decisionKind == "" {
		decisionKind = "operation"
	}
	var tenantCode, institutionID, phase string
	if err := db.QueryRow(ctx, `
		select tenant_code,institution_id,phase
		from school_policy_cutover_state
		where tenant_code=public.current_tenant_code() and institution_id=public.current_institution_id()
		for share
	`).Scan(&tenantCode, &institutionID, &phase); err != nil {
		return OperationPolicyV2Decision{}, fmt.Errorf("load policy v2 cutover state: %w", err)
	}
	if phase != "dual" && phase != "v2" && phase != "contracted" {
		return OperationPolicyV2Decision{}, ErrPolicyV2CutoverRequired
	}

	var profile regulatorymodel.Profile
	var legalForm string
	var profileEffectiveTo *time.Time
	if err := db.QueryRow(ctx, `
		select id::text,profile_series_id::text,version,legal_form,status,effective_from,effective_to
		from school_institution_profiles_v2
		where tenant_code=$1 and institution_id=$2 and status in ('approved','active')
		  and effective_from <= $3 and (effective_to is null or effective_to >= $3)
		order by version desc limit 1
	`, tenantCode, institutionID, request.EffectiveOn.Format(time.DateOnly)).Scan(
		&profile.ID, &profile.ProfileSeriesID, &profile.Version, &legalForm, &profile.Status,
		&profile.EffectiveFrom, &profileEffectiveTo,
	); err != nil {
		return OperationPolicyV2Decision{}, fmt.Errorf("load effective policy v2 profile: %w", err)
	}
	profile.LegalForm = regulatorymodel.LegalForm(legalForm)
	if profileEffectiveTo != nil {
		profile.EffectiveTo = *profileEffectiveTo
	}

	var confessional regulatorymodel.ConfessionalOverlay
	var confessionalEffectiveTo *time.Time
	err := db.QueryRow(ctx, `
		select status,effective_from,effective_to
		from school_confessional_profiles
		where tenant_code=$1 and institution_id=$2 and profile_id=$3::uuid and profile_version=$4
		  and status in ('approved','active') and effective_from <= $5
		  and (effective_to is null or effective_to >= $5)
		order by effective_from desc limit 1
	`, tenantCode, institutionID, profile.ID, profile.Version, request.EffectiveOn.Format(time.DateOnly)).Scan(
		&confessional.Status, &confessional.EffectiveFrom, &confessionalEffectiveTo,
	)
	if err != nil && !errors.Is(err, pgx.ErrNoRows) {
		return OperationPolicyV2Decision{}, fmt.Errorf("load confessional policy overlay: %w", err)
	}
	if err == nil {
		if confessionalEffectiveTo != nil {
			confessional.EffectiveTo = *confessionalEffectiveTo
		}
		profile.Confessional = &confessional
	}

	policyContext := regulatorymodel.OperationPolicyContext{
		OperationCode: request.OperationCode,
		EffectiveOn:   request.EffectiveOn,
		Profile:       profile,
		OfferingID:    request.OfferingID,
		LocationID:    request.LocationID,
	}
	if request.OfferingID != "" {
		rows, queryErr := db.Query(ctx, `
			select offering_id::text,location_id::text,status,effective_from,effective_to
			from school_offering_authorizations
			where tenant_code=$1 and institution_id=$2 and offering_id=$3::uuid and location_id=$4::uuid
			  and effective_from <= $5 and (effective_to is null or effective_to >= $5)
			order by effective_from desc
		`, tenantCode, institutionID, request.OfferingID, request.LocationID, request.EffectiveOn.Format(time.DateOnly))
		if queryErr != nil {
			return OperationPolicyV2Decision{}, fmt.Errorf("load offering authorization evidence: %w", queryErr)
		}
		for rows.Next() {
			var item regulatorymodel.Authorization
			var effectiveTo *time.Time
			if scanErr := rows.Scan(&item.OfferingID, &item.LocationID, &item.Status, &item.EffectiveFrom, &effectiveTo); scanErr != nil {
				rows.Close()
				return OperationPolicyV2Decision{}, fmt.Errorf("scan offering authorization evidence: %w", scanErr)
			}
			if effectiveTo != nil {
				item.EffectiveTo = *effectiveTo
			}
			policyContext.Authorizations = append(policyContext.Authorizations, item)
		}
		if rowsErr := rows.Err(); rowsErr != nil {
			rows.Close()
			return OperationPolicyV2Decision{}, fmt.Errorf("iterate offering authorization evidence: %w", rowsErr)
		}
		rows.Close()
	}

	if request.FundingInstrumentID != "" {
		var item regulatorymodel.FundingInstrument
		var effectiveTo *time.Time
		if err := db.QueryRow(ctx, `
			select id::text,public_funding,eligibility_status,effective_from,effective_to
			from school_funding_instruments
			where tenant_code=$1 and institution_id=$2 and id=$3::uuid
		`, tenantCode, institutionID, request.FundingInstrumentID).Scan(
			&item.ID, &item.Public, &item.EligibilityStatus, &item.EffectiveFrom, &effectiveTo,
		); err != nil {
			return OperationPolicyV2Decision{}, fmt.Errorf("load funding instrument: %w", err)
		}
		if effectiveTo != nil {
			item.EffectiveTo = *effectiveTo
		}
		policyContext.Funding = &item
	}
	if request.ProcurementAssessmentID != "" {
		var item regulatorymodel.ProcurementAssessment
		var effectiveTo *time.Time
		if err := db.QueryRow(ctx, `
			select coalesce(funding_instrument_id::text,''),determination,effective_from,effective_to
			from school_procurement_applicability_assessments
			where tenant_code=$1 and institution_id=$2 and id=$3::uuid
		`, tenantCode, institutionID, request.ProcurementAssessmentID).Scan(
			&item.FundingInstrumentID, &item.Determination, &item.EffectiveFrom, &effectiveTo,
		); err != nil {
			return OperationPolicyV2Decision{}, fmt.Errorf("load procurement assessment: %w", err)
		}
		if effectiveTo != nil {
			item.EffectiveTo = *effectiveTo
		}
		policyContext.Procurement = &item
	}

	bindings, err := loadPolicyV2BindingEvidence(ctx, db, tenantCode, institutionID, profile, request.EffectiveOn)
	if err != nil {
		return OperationPolicyV2Decision{}, err
	}
	kinds := map[string]int{}
	for _, binding := range bindings {
		kinds[binding.AssignmentKind]++
	}
	evaluation := regulatorymodel.Evaluate(policyContext)
	if kinds["common"] != 1 || kinds["legal_form"] != 1 {
		evaluation.Allowed = false
		evaluation.Warnings = append(evaluation.Warnings, "exact common and legal-form policy bindings are required")
	}

	contextSnapshot := struct {
		Facts   regulatorymodel.OperationPolicyContext `json:"facts"`
		Context map[string]any                         `json:"context"`
	}{Facts: policyContext, Context: request.Context}
	contextJSON, contextChecksum, err := canonicalJSONChecksum(contextSnapshot)
	if err != nil {
		return OperationPolicyV2Decision{}, fmt.Errorf("encode policy input: %w", err)
	}
	inputID := uuid.NewString()
	if _, err := db.Exec(ctx, `
		insert into school_operation_policy_inputs(
			id,tenant_code,institution_id,operation_code,profile_id,profile_version,profile_series_id,
			offering_id,location_id,funding_instrument_id,procurement_assessment_id,revalidates_evaluation_id,
			context,checksum_sha256,created_by_subject,effective_on,decision_kind,engine_version,schema_version
		) values($1::uuid,$2,$3,$4,$5::uuid,$6,$7::uuid,nullif($8,'')::uuid,nullif($9,'')::uuid,
			nullif($10,'')::uuid,nullif($11,'')::uuid,nullif($12,'')::uuid,$13::jsonb,$14,$15,$16,$17,'v2',1)
	`, inputID, tenantCode, institutionID, request.OperationCode, profile.ID, profile.Version, profile.ProfileSeriesID,
		request.OfferingID, request.LocationID, request.FundingInstrumentID, request.ProcurementAssessmentID,
		request.RevalidatesEvaluationID, contextJSON, contextChecksum, request.ActorSubject,
		request.EffectiveOn.Format(time.DateOnly), decisionKind); err != nil {
		return OperationPolicyV2Decision{}, fmt.Errorf("persist policy v2 input: %w", err)
	}
	for _, binding := range bindings {
		if _, err := db.Exec(ctx, `
			insert into school_operation_policy_input_bindings(
				tenant_code,institution_id,input_id,binding_id,profile_v2_id,profile_v2_version,
				profile_series_id,policy_pack_version_id,policy_pack_checksum_sha256,
				override_snapshot,override_checksum_sha256,created_by_subject
			) values($1,$2,$3::uuid,$4::uuid,$5::uuid,$6,$7::uuid,$8::uuid,$9,$10::jsonb,$11,$12)
		`, tenantCode, institutionID, inputID, binding.ID, profile.ID, profile.Version,
			profile.ProfileSeriesID, binding.PackID, binding.PackChecksum, binding.OverrideSnapshot,
			binding.OverrideChecksum, request.ActorSubject); err != nil {
			return OperationPolicyV2Decision{}, fmt.Errorf("persist policy v2 binding evidence: %w", err)
		}
	}

	evaluationPayload := struct {
		InputChecksum string   `json:"input_checksum"`
		Allowed       bool     `json:"allowed"`
		Capabilities  []string `json:"capabilities"`
		Obligations   []string `json:"obligations"`
		Warnings      []string `json:"warnings"`
	}{contextChecksum, evaluation.Allowed, evaluation.Capabilities, evaluation.Obligations, evaluation.Warnings}
	_, evaluationChecksum, err := canonicalJSONChecksum(evaluationPayload)
	if err != nil {
		return OperationPolicyV2Decision{}, fmt.Errorf("encode policy v2 evaluation: %w", err)
	}
	capabilitiesJSON, _ := json.Marshal(evaluation.Capabilities)
	obligationsJSON, _ := json.Marshal(evaluation.Obligations)
	warningsJSON, _ := json.Marshal(evaluation.Warnings)
	evaluationID := uuid.NewString()
	if _, err := db.Exec(ctx, `
		insert into school_operation_policy_evaluations_v2(
			id,tenant_code,institution_id,input_id,allowed,capabilities,obligations,warnings,
			checksum_sha256,evaluated_by_subject,effective_on,decision_kind,engine_version,schema_version
		) values($1::uuid,$2,$3,$4::uuid,$5,$6::jsonb,$7::jsonb,$8::jsonb,$9,$10,$11,$12,'v2',1)
	`, evaluationID, tenantCode, institutionID, inputID, evaluation.Allowed, capabilitiesJSON,
		obligationsJSON, warningsJSON, evaluationChecksum, request.ActorSubject,
		request.EffectiveOn.Format(time.DateOnly), decisionKind); err != nil {
		return OperationPolicyV2Decision{}, fmt.Errorf("persist policy v2 evaluation: %w", err)
	}
	return OperationPolicyV2Decision{
		InputID: inputID, EvaluationID: evaluationID, Allowed: evaluation.Allowed,
		Capabilities: evaluation.Capabilities, Obligations: evaluation.Obligations, Warnings: evaluation.Warnings,
	}, nil
}

func loadPolicyV2BindingEvidence(ctx context.Context, db queryer, tenantCode, institutionID string, profile regulatorymodel.Profile, effectiveOn time.Time) ([]policyV2BindingEvidence, error) {
	rows, err := db.Query(ctx, `
		select binding.id::text,binding.policy_pack_version_id::text,pack.checksum_sha256,binding.assignment_kind,
			coalesce(jsonb_agg(jsonb_build_object('key',override.key,'value',override.value)
				order by override.key) filter (where override.id is not null),'[]'::jsonb)
		from school_operation_policy_bindings_v2 binding
		join school_policy_pack_versions pack
		  on pack.tenant_code=binding.tenant_code and pack.institution_id=binding.institution_id
		 and pack.id=binding.policy_pack_version_id and pack.pack_code=binding.pack_code
		 and pack.version=binding.policy_pack_version
		left join school_operation_policy_overrides_v2 override
		  on override.tenant_code=binding.tenant_code and override.institution_id=binding.institution_id
		 and override.binding_id=binding.id and override.status='approved'
		where binding.tenant_code=$1 and binding.institution_id=$2 and binding.profile_v2_id=$3::uuid
		  and binding.profile_v2_version=$4 and binding.profile_series_id=$5::uuid and binding.status='active'
		  and binding.effective_from <= $6 and (binding.effective_to is null or binding.effective_to >= $6)
		  and pack.status='approved' and pack.effective_from <= $6 and (pack.effective_to is null or pack.effective_to >= $6)
		group by binding.id,binding.policy_pack_version_id,pack.checksum_sha256,binding.assignment_kind,binding.pack_code
		order by case binding.assignment_kind when 'common' then 1 when 'legal_form' then 2 when 'funding' then 3 when 'program' then 4 else 5 end,
			binding.pack_code
	`, tenantCode, institutionID, profile.ID, profile.Version, profile.ProfileSeriesID, effectiveOn.Format(time.DateOnly))
	if err != nil {
		return nil, fmt.Errorf("load policy v2 binding evidence: %w", err)
	}
	defer rows.Close()
	bindings := []policyV2BindingEvidence{}
	for rows.Next() {
		var item policyV2BindingEvidence
		if err := rows.Scan(&item.ID, &item.PackID, &item.PackChecksum, &item.AssignmentKind, &item.OverrideSnapshot); err != nil {
			return nil, fmt.Errorf("scan policy v2 binding evidence: %w", err)
		}
		_, item.OverrideChecksum, err = canonicalJSONChecksum(json.RawMessage(item.OverrideSnapshot))
		if err != nil {
			return nil, fmt.Errorf("checksum policy v2 overrides: %w", err)
		}
		bindings = append(bindings, item)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate policy v2 binding evidence: %w", err)
	}
	return bindings, nil
}

func canonicalJSONChecksum(value any) ([]byte, string, error) {
	raw, err := json.Marshal(value)
	if err != nil {
		return nil, "", err
	}
	hash := sha256.Sum256(raw)
	return raw, hex.EncodeToString(hash[:]), nil
}
