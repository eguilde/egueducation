package institution

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"sort"
	"strings"
	"time"

	"github.com/eguilde/egueducation/internal/audit"
	"github.com/eguilde/egueducation/internal/auth"
	appdb "github.com/eguilde/egueducation/internal/db"
	"github.com/eguilde/egueducation/internal/httpx"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
)

type Service struct {
	pool *appdb.SessionPool
}

type policyEvaluationContextKey struct{}

// CurrentPolicyEvaluationIDFromRequest returns the immutable, server-created
// policy decision attached by RequireCapability. It is never accepted from a
// client payload.
func CurrentPolicyEvaluationIDFromRequest(r *http.Request) string {
	value, _ := r.Context().Value(policyEvaluationContextKey{}).(string)
	return value
}

func NewService(pool *appdb.SessionPool) *Service { return &Service{pool: pool} }

type queryer interface {
	Query(context.Context, string, ...any) (pgx.Rows, error)
	QueryRow(context.Context, string, ...any) pgx.Row
	Exec(context.Context, string, ...any) (pgconn.CommandTag, error)
}

func (s *Service) GetRegulatoryProfile(w http.ResponseWriter, r *http.Request) {
	profile, err := loadCurrentProfile(r.Context(), s.pool)
	if errors.Is(err, pgx.ErrNoRows) {
		httpx.JSON(w, http.StatusNotFound, map[string]any{"code": "regulatory_profile_not_found"})
		return
	}
	if err != nil {
		httpx.JSON(w, http.StatusInternalServerError, map[string]any{"code": "regulatory_profile_read_failed"})
		return
	}
	httpx.JSON(w, http.StatusOK, profile)
}

func (s *Service) PutRegulatoryProfile(w http.ResponseWriter, r *http.Request) {
	var input PutRegulatoryProfileRequest
	decoder := json.NewDecoder(http.MaxBytesReader(w, r.Body, 128<<10))
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(&input); err != nil {
		httpx.JSON(w, http.StatusBadRequest, map[string]any{"code": "invalid_regulatory_profile_payload"})
		return
	}
	if code := validateProfileInput(&input); code != "" {
		httpx.JSON(w, http.StatusUnprocessableEntity, map[string]any{"code": code})
		return
	}

	tenantCode := strings.TrimSpace(auth.CurrentTenantCodeFromRequest(r))
	institutionID := strings.TrimSpace(auth.CurrentInstitutionIDFromRequest(r))
	actor := strings.TrimSpace(auth.CurrentSubjectFromRequest(r))
	if tenantCode == "" || institutionID == "" || actor == "" {
		httpx.JSON(w, http.StatusUnauthorized, map[string]any{"code": "institution_context_required"})
		return
	}

	tx, err := s.pool.Begin(r.Context())
	if err != nil {
		httpx.JSON(w, http.StatusInternalServerError, map[string]any{"code": "regulatory_profile_write_failed"})
		return
	}
	defer func() { _ = tx.Rollback(r.Context()) }()

	var currentID string
	var currentVersion int
	err = tx.QueryRow(r.Context(), `
		select id::text, version
		from school_institution_profiles
		where tenant_code=$1 and institution_id=$2 and status <> 'superseded'
		order by version desc limit 1
		for update
	`, tenantCode, institutionID).Scan(&currentID, &currentVersion)
	if errors.Is(err, pgx.ErrNoRows) {
		httpx.JSON(w, http.StatusNotFound, map[string]any{"code": "regulatory_profile_not_found"})
		return
	}
	if err != nil {
		httpx.JSON(w, http.StatusInternalServerError, map[string]any{"code": "regulatory_profile_write_failed"})
		return
	}
	if currentVersion != input.ExpectedVersion {
		httpx.JSON(w, http.StatusConflict, map[string]any{"code": "regulatory_profile_version_conflict", "current_version": currentVersion})
		return
	}

	var effectiveFrom *time.Time
	if strings.TrimSpace(input.EffectiveFrom) != "" {
		parsed, _ := time.Parse(time.DateOnly, input.EffectiveFrom)
		effectiveFrom = &parsed
	}
	var effectiveTo *time.Time
	if input.EffectiveTo != nil && strings.TrimSpace(*input.EffectiveTo) != "" {
		parsed, _ := time.Parse(time.DateOnly, strings.TrimSpace(*input.EffectiveTo))
		effectiveTo = &parsed
	}
	approved := input.Status == "approved" || input.Status == "active"
	approvedBy := ""
	var approvedAt *time.Time
	if approved {
		approvedBy = actor
		now := time.Now().UTC()
		approvedAt = &now
	}

	input.AuthorizedLevels = normalizedStrings(input.AuthorizedLevels)
	input.ProgramCodes = normalizedStrings(input.ProgramCodes)
	if approved {
		// Close only approved/effective predecessors. Draft creation never changes
		// the live profile, and a future version leaves the predecessor effective
		// until the day before the scheduled transition.
		if _, err = tx.Exec(r.Context(), `
			update school_institution_profiles
			set effective_to=$3::date-1, updated_by_subject=$4, updated_at=now()
			where tenant_code=$1 and institution_id=$2 and status in ('approved','active')
				and effective_from < $3 and (effective_to is null or effective_to >= $3)
		`, tenantCode, institutionID, effectiveFrom, actor); err != nil {
			httpx.JSON(w, http.StatusInternalServerError, map[string]any{"code": "regulatory_profile_write_failed"})
			return
		}
		if _, err = tx.Exec(r.Context(), `
			update school_institution_profiles
			set status='superseded', updated_by_subject=$4, updated_at=now()
			where tenant_code=$1 and institution_id=$2 and status in ('approved','active')
				and effective_from >= $3
		`, tenantCode, institutionID, effectiveFrom, actor); err != nil {
			httpx.JSON(w, http.StatusInternalServerError, map[string]any{"code": "regulatory_profile_write_failed"})
			return
		}
	}
	var profileID string
	err = tx.QueryRow(r.Context(), `
		insert into school_institution_profiles(
			tenant_code,institution_id,version,status,school_legal_form,regulatory_profile,
			authorization_status,accreditation_reference,authorized_levels,has_legal_personality,
			tax_identifier,founder_name,funder_name,budget_authority_name,is_contracting_authority,
			accounting_profile,procurement_profile,payroll_profile,vat_profile,treasury_required,
			public_funding,program_codes,effective_from,effective_to,source_reference,
			approved_by_subject,approved_at,created_by_subject,updated_by_subject
		) values (
			$1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11,$12,$13,$14,$15,$16,$17,$18,$19,$20,
			$21,$22,$23,$24,$25,$26,$27,$28,$28
		) returning id::text
	`, tenantCode, institutionID, currentVersion+1, input.Status, input.SchoolLegalForm,
		input.RegulatoryProfile, input.AuthorizationStatus, input.AccreditationReference,
		input.AuthorizedLevels, input.HasLegalPersonality, input.TaxIdentifier,
		input.FounderName, input.FunderName, input.BudgetAuthorityName,
		input.IsContractingAuthority, input.AccountingProfile, input.ProcurementProfile,
		input.PayrollProfile, input.VATProfile, input.TreasuryRequired, input.PublicFunding,
		input.ProgramCodes, effectiveFrom, effectiveTo, input.SourceReference, approvedBy,
		approvedAt, actor).Scan(&profileID)
	if err != nil {
		if pgErr, ok := err.(*pgconn.PgError); ok && pgErr.Code == "23505" {
			httpx.JSON(w, http.StatusConflict, map[string]any{"code": "regulatory_profile_version_conflict", "current_version": currentVersion})
			return
		}
		httpx.JSON(w, http.StatusUnprocessableEntity, map[string]any{"code": "regulatory_profile_persist_failed"})
		return
	}

	if approved {
		packKinds := map[string]string{"common.ro": "common", "legal-form.ro." + input.SchoolLegalForm: "legal_form"}
		if input.PublicFunding {
			packKinds["funding.public"] = "funding"
		}
		for _, programCode := range input.ProgramCodes {
			packKinds["program."+programCode] = "program"
		}
		packCodes := make([]string, 0, len(packKinds))
		for code := range packKinds {
			packCodes = append(packCodes, code)
		}
		sort.Strings(packCodes)
		for _, code := range packCodes {
			var packID string
			err = tx.QueryRow(r.Context(), `
				select id::text from school_policy_pack_versions
				where tenant_code=$1 and institution_id=$2 and pack_code=$3 and status='approved'
					and effective_from <= $4 and (effective_to is null or effective_to >= $4)
				order by version desc limit 1
			`, tenantCode, institutionID, code, effectiveFrom).Scan(&packID)
			if err != nil {
				httpx.JSON(w, http.StatusUnprocessableEntity, map[string]any{"code": "required_policy_pack_missing", "pack_code": code})
				return
			}
			if _, err = tx.Exec(r.Context(), `
				insert into school_policy_assignments(
					tenant_code,institution_id,policy_pack_version_id,profile_id,profile_version,pack_code,assignment_kind,status,
					effective_from,effective_to,assigned_by_subject,created_by_subject,updated_by_subject
				) values ($1,$2,$3,$4,$5,$6,$7,'active',$8,$9,$10,$10,$10)
			`, tenantCode, institutionID, packID, profileID, currentVersion+1, code, packKinds[code], effectiveFrom, effectiveTo, actor); err != nil {
				httpx.JSON(w, http.StatusUnprocessableEntity, map[string]any{"code": "policy_assignment_persist_failed"})
				return
			}
		}
	}

	if err = audit.Log(r.Context(), tx, audit.Event{
		ActorSubject: actor,
		Action:       "institution.regulatory_profile.versioned",
		TargetType:   "school_institution_profile",
		TargetID:     profileID,
		Summary:      "Institution regulatory profile version created",
		Details: map[string]any{
			"previous_profile_id": currentID,
			"previous_version":    currentVersion,
			"version":             currentVersion + 1,
			"legal_form":          input.SchoolLegalForm,
			"status":              input.Status,
		},
	}); err != nil {
		httpx.JSON(w, http.StatusInternalServerError, map[string]any{"code": "regulatory_profile_audit_failed"})
		return
	}
	if err = tx.Commit(r.Context()); err != nil {
		httpx.JSON(w, http.StatusInternalServerError, map[string]any{"code": "regulatory_profile_write_failed"})
		return
	}

	profile, err := loadCurrentProfile(r.Context(), s.pool)
	if err != nil {
		httpx.JSON(w, http.StatusInternalServerError, map[string]any{"code": "regulatory_profile_read_failed"})
		return
	}
	httpx.JSON(w, http.StatusOK, profile)
}

func (s *Service) GetCapabilities(w http.ResponseWriter, r *http.Request) {
	profile, err := loadEffectiveProfile(r.Context(), s.pool, time.Now().UTC())
	if errors.Is(err, pgx.ErrNoRows) {
		profile, err = loadCurrentProfile(r.Context(), s.pool)
	}
	if err != nil {
		httpx.JSON(w, http.StatusInternalServerError, map[string]any{"code": "institution_capabilities_read_failed"})
		return
	}
	response, err := s.evaluate(r.Context(), s.pool, r, profile, false)
	if err != nil {
		httpx.JSON(w, http.StatusInternalServerError, map[string]any{"code": "institution_capabilities_evaluation_failed"})
		return
	}
	httpx.JSON(w, http.StatusOK, response)
}

func (s *Service) evaluate(ctx context.Context, db queryer, r *http.Request, profile RegulatoryProfile, persist bool) (InstitutionCapabilitiesResponse, error) {
	now := time.Now().UTC()
	warnings := []string{}
	blocked := false
	blockReason := ""
	if profile.Status == "unclassified" || profile.SchoolLegalForm == nil {
		blocked, blockReason = true, "Profilul instituțional nu este clasificat și aprobat"
		warnings = append(warnings, "Citirea rămâne disponibilă; operațiunile reglementate sunt blocate")
	}
	if profile.Status != "approved" && profile.Status != "active" {
		blocked = true
		if blockReason == "" {
			blockReason = "Profilul instituțional nu este aprobat"
		}
	}
	if profile.EffectiveFrom != nil && *profile.EffectiveFrom > now.Format(time.DateOnly) {
		blocked, blockReason = true, "Profilul instituțional nu este încă în vigoare"
	}
	if profile.EffectiveTo != nil && *profile.EffectiveTo < now.Format(time.DateOnly) {
		blocked, blockReason = true, "Profilul instituțional a expirat"
	}

	packs, err := loadAssignedPacks(ctx, db, profile.ID, now)
	if err != nil {
		return InstitutionCapabilitiesResponse{}, err
	}
	if !blocked {
		kinds := map[string]int{}
		for _, pack := range packs {
			kinds[pack.AssignmentKind]++
		}
		if kinds["common"] != 1 || kinds["legal_form"] != 1 || kinds["funding"] > 1 {
			blocked, blockReason = true, "Configurația politicilor este absentă sau ambiguă"
			warnings = append(warnings, "Administratorul trebuie să corecteze asignările policy pack")
		}
	}

	capabilities := []PolicyCapability{}
	if !blocked {
		capabilities = resolveCapabilities(packs, auth.CurrentPermissionsFromRequest(r), auth.CurrentActiveModulesFromRequest(r))
	}
	summaries := make([]PolicyPackSummary, 0, len(packs))
	packIDs := make([]string, 0, len(packs))
	for _, pack := range packs {
		summaries = append(summaries, pack.PolicyPackSummary)
		packIDs = append(packIDs, pack.ID)
	}
	response := InstitutionCapabilitiesResponse{
		TenantCode: auth.CurrentTenantCodeFromRequest(r), InstitutionID: auth.CurrentInstitutionIDFromRequest(r),
		EvaluatedAt: now.Format(time.RFC3339), ProfileID: profile.ID, ProfileVersion: profile.Version,
		ProfileStatus: profile.Status, SchoolLegalForm: profile.SchoolLegalForm, Blocked: blocked,
		BlockReason: blockReason, Warnings: warnings, EffectivePolicies: summaries, Capabilities: capabilities,
	}
	if persist {
		evaluationID, err := persistEvaluation(ctx, db, auth.CurrentSubjectFromRequest(r), profile, packIDs, capabilities, warnings, blocked)
		if err != nil {
			return InstitutionCapabilitiesResponse{}, err
		}
		response.EvaluationID = &evaluationID
	}
	return response, nil
}

// RequireCapability evaluates policy and RBAC from the host-bound request
// context. Missing, expired, ambiguous or unknown policy always fails closed.
func (s *Service) RequireCapability(code string) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			profile, err := loadEffectiveProfile(r.Context(), s.pool, time.Now().UTC())
			if errors.Is(err, pgx.ErrNoRows) {
				httpx.JSON(w, http.StatusConflict, map[string]any{"code": "regulatory_profile_not_effective"})
				return
			}
			if err != nil {
				httpx.JSON(w, http.StatusInternalServerError, map[string]any{"code": "institution_capabilities_evaluation_failed"})
				return
			}
			response, err := s.evaluate(r.Context(), s.pool, r, profile, true)
			if err != nil {
				httpx.JSON(w, http.StatusInternalServerError, map[string]any{"code": "institution_capabilities_evaluation_failed"})
				return
			}
			if response.Blocked {
				httpx.JSON(w, http.StatusConflict, map[string]any{"code": "institution_policy_blocked", "reason": response.BlockReason, "policy_evaluation_id": response.EvaluationID})
				return
			}
			allowed := false
			for _, capability := range response.Capabilities {
				if capability.Code == code && capability.Enabled {
					allowed = true
					break
				}
			}
			if !allowed {
				httpx.JSON(w, http.StatusForbidden, map[string]any{"code": "institution_policy_capability_denied", "capability": code, "policy_evaluation_id": response.EvaluationID})
				return
			}
			if response.EvaluationID == nil {
				httpx.JSON(w, http.StatusInternalServerError, map[string]any{"code": "institution_policy_evaluation_missing"})
				return
			}
			ctx := context.WithValue(r.Context(), policyEvaluationContextKey{}, *response.EvaluationID)
			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}
}

func loadCurrentProfile(ctx context.Context, db queryer) (RegulatoryProfile, error) {
	return loadProfile(ctx, db, `status <> 'superseded'`, nil)
}

func loadEffectiveProfile(ctx context.Context, db queryer, at time.Time) (RegulatoryProfile, error) {
	return loadProfile(ctx, db, `status in ('approved','active') and effective_from <= $1 and (effective_to is null or effective_to >= $1)`, []any{at.Format(time.DateOnly)})
}

func loadProfile(ctx context.Context, db queryer, predicate string, args []any) (RegulatoryProfile, error) {
	var profile RegulatoryProfile
	var effectiveFrom, effectiveTo, approvedAt *time.Time
	var createdAt, updatedAt time.Time
	err := db.QueryRow(ctx, `
		select id::text,tenant_code,institution_id,version,status,school_legal_form,regulatory_profile,
			authorization_status,accreditation_reference,authorized_levels,has_legal_personality,
			tax_identifier,founder_name,funder_name,budget_authority_name,is_contracting_authority,
			accounting_profile,procurement_profile,payroll_profile,vat_profile,treasury_required,
			public_funding,program_codes,effective_from,effective_to,source_reference,
			approved_by_subject,approved_at,created_by_subject,created_at,updated_by_subject,updated_at
		from school_institution_profiles
		where tenant_code=public.current_tenant_code() and institution_id=public.current_institution_id()
			and `+predicate+`
		order by version desc limit 1
	`, args...).Scan(&profile.ID, &profile.TenantCode, &profile.InstitutionID, &profile.Version, &profile.Status,
		&profile.SchoolLegalForm, &profile.RegulatoryProfile, &profile.AuthorizationStatus,
		&profile.AccreditationReference, &profile.AuthorizedLevels, &profile.HasLegalPersonality,
		&profile.TaxIdentifier, &profile.FounderName, &profile.FunderName, &profile.BudgetAuthorityName,
		&profile.IsContractingAuthority, &profile.AccountingProfile, &profile.ProcurementProfile,
		&profile.PayrollProfile, &profile.VATProfile, &profile.TreasuryRequired, &profile.PublicFunding,
		&profile.ProgramCodes, &effectiveFrom, &effectiveTo, &profile.SourceReference,
		&profile.ApprovedBySubject, &approvedAt, &profile.CreatedBySubject, &createdAt,
		&profile.UpdatedBySubject, &updatedAt)
	if err != nil {
		return RegulatoryProfile{}, err
	}
	profile.AuthorizedLevels = nonNil(profile.AuthorizedLevels)
	profile.ProgramCodes = nonNil(profile.ProgramCodes)
	profile.EffectiveFrom = datePointer(effectiveFrom)
	profile.EffectiveTo = datePointer(effectiveTo)
	profile.ApprovedAt = timePointer(approvedAt)
	profile.CreatedAt = createdAt.UTC().Format(time.RFC3339)
	profile.UpdatedAt = updatedAt.UTC().Format(time.RFC3339)
	return profile, nil
}

func loadAssignedPacks(ctx context.Context, db queryer, profileID string, at time.Time) ([]assignedPolicyPack, error) {
	rows, err := db.Query(ctx, `
		select pack.id::text,pack.pack_code,pack.version,pack.regulatory_profile,
			pack.effective_from,pack.effective_to,pack.checksum_sha256,pack.source_references,
			assignment.id::text,assignment.assignment_kind,pack.configurable_keys,pack.rules
		from school_policy_assignments assignment
		join school_policy_pack_versions pack
			on pack.tenant_code=assignment.tenant_code and pack.institution_id=assignment.institution_id
			and pack.id=assignment.policy_pack_version_id
		where assignment.tenant_code=public.current_tenant_code()
			and assignment.institution_id=public.current_institution_id()
			and assignment.profile_id=$1
			and assignment.status='active' and assignment.effective_from <= $2
			and (assignment.effective_to is null or assignment.effective_to >= $2)
			and pack.status='approved' and pack.effective_from <= $2
			and (pack.effective_to is null or pack.effective_to >= $2)
		order by case assignment.assignment_kind when 'common' then 1 when 'legal_form' then 2 when 'funding' then 3 when 'program' then 4 else 5 end,
			pack.pack_code, pack.version
	`, profileID, at.Format(time.DateOnly))
	if err != nil {
		return nil, err
	}
	packs := []assignedPolicyPack{}
	for rows.Next() {
		var pack assignedPolicyPack
		var effectiveFrom time.Time
		var effectiveTo *time.Time
		var rawRules []byte
		if err := rows.Scan(&pack.ID, &pack.Code, &pack.Version, &pack.RegulatoryProfile,
			&effectiveFrom, &effectiveTo, &pack.ChecksumSHA256, &pack.SourceReferences,
			&pack.AssignmentID, &pack.AssignmentKind, &pack.ConfigurableKeys, &rawRules); err != nil {
			return nil, err
		}
		pack.EffectiveFrom = effectiveFrom.Format(time.DateOnly)
		pack.EffectiveTo = datePointer(effectiveTo)
		pack.SourceReferences = nonNil(pack.SourceReferences)
		decoder := json.NewDecoder(strings.NewReader(string(rawRules)))
		decoder.DisallowUnknownFields()
		if err := decoder.Decode(&pack.Rules); err != nil || pack.Rules.Capabilities == nil {
			if err == nil {
				err = errors.New("capabilities array is required")
			}
			return nil, fmt.Errorf("decode policy %s: %w", pack.Code, err)
		}
		packs = append(packs, pack)
	}
	if err := rows.Err(); err != nil {
		rows.Close()
		return nil, err
	}
	rows.Close()
	for index := range packs {
		if err := applyApprovedOverrides(ctx, db, &packs[index]); err != nil {
			return nil, fmt.Errorf("apply policy overrides %s: %w", packs[index].Code, err)
		}
	}
	return packs, nil
}

// applyApprovedOverrides supports only keys explicitly declared by the pack.
// Obligations are additive: an override may improve explanatory text or add
// documents/approvals/steps, but it can never disable a capability or remove a
// baseline requirement.
func applyApprovedOverrides(ctx context.Context, db queryer, pack *assignedPolicyPack) error {
	rows, err := db.Query(ctx, `select key,value from school_policy_overrides where tenant_code=public.current_tenant_code() and institution_id=public.current_institution_id() and policy_assignment_id=$1 and status='approved' order by key`, pack.AssignmentID)
	if err != nil {
		return err
	}
	defer rows.Close()
	allowed := stringSet(pack.ConfigurableKeys)
	for rows.Next() {
		var key string
		var raw []byte
		if err := rows.Scan(&key, &raw); err != nil {
			return err
		}
		if _, ok := allowed[key]; !ok {
			return fmt.Errorf("override key %q is not configurable", key)
		}
		parts := strings.Split(key, ".")
		if len(parts) < 4 || parts[0] != "capabilities" {
			return fmt.Errorf("unsupported override key %q", key)
		}
		field := parts[len(parts)-1]
		code := strings.Join(parts[1:len(parts)-1], ".")
		index := -1
		for i := range pack.Rules.Capabilities {
			if pack.Rules.Capabilities[i].Code == code {
				index = i
				break
			}
		}
		if index < 0 {
			return fmt.Errorf("override capability %q is absent", code)
		}
		rule := &pack.Rules.Capabilities[index]
		switch field {
		case "reason":
			if err := json.Unmarshal(raw, &rule.Reason); err != nil {
				return err
			}
		case "required_documents":
			var values []string
			if err := json.Unmarshal(raw, &values); err != nil {
				return err
			}
			rule.RequiredDocuments = unionStrings(rule.RequiredDocuments, values)
		case "required_approvals":
			var values []string
			if err := json.Unmarshal(raw, &values); err != nil {
				return err
			}
			rule.RequiredApprovals = unionStrings(rule.RequiredApprovals, values)
		case "wizard_steps":
			var values []string
			if err := json.Unmarshal(raw, &values); err != nil {
				return err
			}
			rule.WizardSteps = unionStrings(rule.WizardSteps, values)
		default:
			return fmt.Errorf("override field %q cannot weaken mandatory policy", field)
		}
	}
	return rows.Err()
}

func persistEvaluation(ctx context.Context, db queryer, actor string, profile RegulatoryProfile, packIDs []string, capabilities []PolicyCapability, warnings []string, blocked bool) (string, error) {
	payload := struct {
		ProfileID      string             `json:"profile_id"`
		ProfileVersion int                `json:"profile_version"`
		PackIDs        []string           `json:"pack_ids"`
		Capabilities   []PolicyCapability `json:"capabilities"`
		Warnings       []string           `json:"warnings"`
		Blocked        bool               `json:"blocked"`
		Actor          string             `json:"actor"`
	}{profile.ID, profile.Version, packIDs, capabilities, warnings, blocked, actor}
	raw, err := json.Marshal(payload)
	if err != nil {
		return "", err
	}
	hash := sha256.Sum256(raw)
	checksum := hex.EncodeToString(hash[:])
	capabilityJSON, _ := json.Marshal(capabilities)
	var id string
	err = db.QueryRow(ctx, `
		insert into school_policy_evaluations(
			tenant_code,institution_id,profile_id,profile_version,evaluated_by_subject,
			policy_pack_version_ids,capabilities,warnings,blocked,checksum_sha256,
			created_by_subject,updated_by_subject
		) values (
			public.current_tenant_code(),public.current_institution_id(),$1,$2,$3,$4::uuid[],$5::jsonb,$6,$7,$8,$3,$3
		) on conflict (tenant_code,institution_id,profile_id,checksum_sha256) do nothing
		returning id::text
	`, profile.ID, profile.Version, actor, packIDs, capabilityJSON, warnings, blocked, checksum).Scan(&id)
	if errors.Is(err, pgx.ErrNoRows) {
		err = db.QueryRow(ctx, `select id::text from school_policy_evaluations where tenant_code=public.current_tenant_code() and institution_id=public.current_institution_id() and profile_id=$1 and checksum_sha256=$2`, profile.ID, checksum).Scan(&id)
	}
	return id, err
}

func validateProfileInput(input *PutRegulatoryProfileRequest) string {
	if input.ExpectedVersion <= 0 {
		return "expected_version_required"
	}
	if input.Status != "draft" && input.Status != "approved" && input.Status != "active" {
		return "invalid_regulatory_profile_status"
	}
	if input.SchoolLegalForm != "public" && input.SchoolLegalForm != "private" && input.SchoolLegalForm != "confessional" {
		return "invalid_school_legal_form"
	}
	if strings.TrimSpace(input.RegulatoryProfile) == "" {
		return "regulatory_profile_required"
	}
	validAuthorization := stringSet([]string{"unknown", "provisional", "authorized", "accredited", "suspended", "withdrawn"})
	if _, ok := validAuthorization[input.AuthorizationStatus]; !ok {
		return "invalid_authorization_status"
	}
	if input.Status == "approved" || input.Status == "active" {
		if _, err := time.Parse(time.DateOnly, input.EffectiveFrom); err != nil {
			return "effective_from_required"
		}
		if strings.TrimSpace(input.SourceReference) == "" {
			return "source_reference_required"
		}
	}
	if input.EffectiveTo != nil && strings.TrimSpace(*input.EffectiveTo) != "" {
		to, err := time.Parse(time.DateOnly, strings.TrimSpace(*input.EffectiveTo))
		if err != nil {
			return "invalid_effective_to"
		}
		if input.EffectiveFrom != "" {
			from, err := time.Parse(time.DateOnly, input.EffectiveFrom)
			if err != nil || to.Before(from) {
				return "invalid_effective_window"
			}
		}
	}
	return ""
}

func normalizedStrings(values []string) []string {
	set := stringSet(values)
	result := make([]string, 0, len(set))
	for value := range set {
		result = append(result, value)
	}
	sort.Strings(result)
	return result
}

func nonNil(values []string) []string {
	if values == nil {
		return []string{}
	}
	return values
}
func datePointer(value *time.Time) *string {
	if value == nil {
		return nil
	}
	formatted := value.Format(time.DateOnly)
	return &formatted
}
func timePointer(value *time.Time) *string {
	if value == nil {
		return nil
	}
	formatted := value.UTC().Format(time.RFC3339)
	return &formatted
}
