package education

import (
	"encoding/json"
	"errors"
	"log/slog"
	"net/http"
	"strings"

	authruntime "github.com/eguilde/egueducation/internal/auth"
	"github.com/eguilde/egueducation/internal/httpx"
	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
)

const portfolioDeclarationAcknowledgementMethod = "authenticated_web_acknowledgement"

type PortfolioDeclarationTemplate struct {
	DeclarationType    string `json:"declaration_type"`
	DeclarationVersion string `json:"declaration_version"`
	DeclarationText    string `json:"declaration_text"`
	SourceRef          string `json:"source_ref"`
	EffectiveFrom      string `json:"effective_from"`
}

type PortfolioDeclarationAcknowledgement struct {
	ID                  string         `json:"id"`
	PortfolioID         string         `json:"portfolio_id"`
	DeclarationType     string         `json:"declaration_type"`
	DeclarationVersion  string         `json:"declaration_version"`
	DeclarationText     string         `json:"declaration_text"`
	AcceptedAt          string         `json:"accepted_at"`
	AcceptedByUserID    string         `json:"accepted_by_user_id"`
	AttestationMethod   string         `json:"attestation_method"`
	AttestationEvidence map[string]any `json:"attestation_evidence"`
	SignatureEvidence   map[string]any `json:"signature_evidence"`
}

type PortfolioDeclarationEvidenceResponse struct {
	Templates        []PortfolioDeclarationTemplate        `json:"templates"`
	Acknowledgements []PortfolioDeclarationAcknowledgement `json:"acknowledgements"`
}

func writePortfolioDeclarationAcknowledgementFailure(w http.ResponseWriter, stage string, err error) {
	slog.Error("portfolio declaration acknowledgement failed", "stage", stage, "error", err)
	httpx.JSON(w, http.StatusInternalServerError, map[string]any{"code": "portfolio_declaration_acknowledgement_failed"})
}

// PortfolioDeclarationAcknowledgementRequest deliberately excludes the
// declaration wording and version. Both are selected from the server's active
// legal-template store while the acknowledgement is written.
type PortfolioDeclarationAcknowledgementRequest struct {
	Confirmed bool `json:"confirmed"`
}

func scanPortfolioDeclarationTemplate(row pgx.Row, item *PortfolioDeclarationTemplate) error {
	return row.Scan(&item.DeclarationType, &item.DeclarationVersion, &item.DeclarationText, &item.SourceRef, &item.EffectiveFrom)
}

func scanPortfolioDeclarationAcknowledgement(row pgx.Row, item *PortfolioDeclarationAcknowledgement) error {
	return row.Scan(
		&item.ID,
		&item.PortfolioID,
		&item.DeclarationType,
		&item.DeclarationVersion,
		&item.DeclarationText,
		&item.AcceptedAt,
		&item.AcceptedByUserID,
		&item.AttestationMethod,
		&item.AttestationEvidence,
		&item.SignatureEvidence,
	)
}

const portfolioDeclarationTemplateColumns = `
	declaration_type, declaration_version, declaration_text, source_ref,
	to_char(effective_from, 'YYYY-MM-DD')`

const portfolioDeclarationAcknowledgementColumns = `
	id::text, portfolio_id::text, declaration_type, declaration_version,
	declaration_text, to_char(accepted_at, 'YYYY-MM-DD"T"HH24:MI:SS"Z"'),
	accepted_by_user_id::text, attestation_method, attestation_evidence,
	signature_evidence`

func (s *Service) activePortfolioDeclarationTemplates(r *http.Request) ([]PortfolioDeclarationTemplate, error) {
	rows, err := s.pool.Query(r.Context(), `
		select `+portfolioDeclarationTemplateColumns+` from (
			select distinct on (declaration_type) *
			from education_portfolio_declaration_templates
			where lifecycle_status = 'published'
				and effective_from <= current_date
				and (effective_to is null or effective_to >= current_date)
			order by declaration_type, effective_from desc, created_at desc
		) current_template
		order by case declaration_type when 'gdpr_information' then 1 when 'authenticity' then 2 else 99 end
	`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	items := make([]PortfolioDeclarationTemplate, 0, 2)
	for rows.Next() {
		var item PortfolioDeclarationTemplate
		if err := scanPortfolioDeclarationTemplate(rows, &item); err != nil {
			return nil, err
		}
		items = append(items, item)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return items, nil
}

func (s *Service) portfolioDeclarationAcknowledgements(r *http.Request, recordID string) ([]PortfolioDeclarationAcknowledgement, error) {
	rows, err := s.pool.Query(r.Context(), `
		select `+portfolioDeclarationAcknowledgementColumns+`
		from education_portfolio_declaration_acknowledgements
		where portfolio_id = $1::uuid
			and institution_id = $2
			and tenant_code = public.current_tenant_code()
		order by accepted_at asc, declaration_type asc
	`, recordID, s.institutionID(r))
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	items := make([]PortfolioDeclarationAcknowledgement, 0, 2)
	for rows.Next() {
		var item PortfolioDeclarationAcknowledgement
		if err := scanPortfolioDeclarationAcknowledgement(rows, &item); err != nil {
			return nil, err
		}
		items = append(items, item)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return items, nil
}

// PortfolioOwnDeclarationEvidence exposes current, server-issued legal
// wording alongside immutable acknowledgements belonging to the caller's own
// portfolio. It never accepts a client-provided text or version.
func (s *Service) PortfolioOwnDeclarationEvidence(w http.ResponseWriter, r *http.Request) {
	recordID := strings.TrimSpace(chi.URLParam(r, "recordID"))
	if _, err := uuid.Parse(recordID); err != nil {
		httpx.JSON(w, http.StatusBadRequest, map[string]any{"code": "invalid_portfolio_id"})
		return
	}
	if _, allowed, err := s.requireOwnPortfolio(r, recordID, portfolioReadOwnPermission); err != nil || !allowed {
		writePortfolioAccessFailure(w, err)
		return
	}
	templates, err := s.activePortfolioDeclarationTemplates(r)
	if err != nil {
		httpx.JSON(w, http.StatusInternalServerError, map[string]any{"code": "education_portfolio_declaration_templates_failed"})
		return
	}
	acknowledgements, err := s.portfolioDeclarationAcknowledgements(r, recordID)
	if err != nil {
		httpx.JSON(w, http.StatusInternalServerError, map[string]any{"code": "education_portfolio_declaration_evidence_failed"})
		return
	}
	httpx.JSON(w, http.StatusOK, PortfolioDeclarationEvidenceResponse{Templates: templates, Acknowledgements: acknowledgements})
}

// AcknowledgePortfolioOwnDeclaration accepts only an affirmative command. The
// authoritative declaration version/text and the attestation evidence are
// constructed by the server inside one transaction.
func (s *Service) AcknowledgePortfolioOwnDeclaration(w http.ResponseWriter, r *http.Request) {
	recordID := strings.TrimSpace(chi.URLParam(r, "recordID"))
	declarationType := strings.TrimSpace(chi.URLParam(r, "declarationType"))
	if _, err := uuid.Parse(recordID); err != nil {
		httpx.JSON(w, http.StatusBadRequest, map[string]any{"code": "invalid_portfolio_id"})
		return
	}
	if declarationType != "gdpr_information" && declarationType != "authenticity" {
		httpx.JSON(w, http.StatusBadRequest, map[string]any{"code": "invalid_portfolio_declaration_type"})
		return
	}
	var request PortfolioDeclarationAcknowledgementRequest
	if err := json.NewDecoder(r.Body).Decode(&request); err != nil || !request.Confirmed {
		httpx.JSON(w, http.StatusBadRequest, map[string]any{"code": "portfolio_declaration_confirmation_required"})
		return
	}
	actorID, allowed, err := s.requireOwnPortfolio(r, recordID, portfolioManageOwnPermission)
	if err != nil || !allowed {
		writePortfolioAccessFailure(w, err)
		return
	}

	tx, err := s.pool.Begin(r.Context())
	if err != nil {
		writePortfolioDeclarationAcknowledgementFailure(w, "begin", err)
		return
	}
	defer tx.Rollback(r.Context())

	// Re-check and lock the owned portfolio in the same transaction as the
	// evidence insert. This prevents a concurrent ownership/lifecycle change
	// from producing evidence for an inaccessible record.
	var lockedID string
	err = tx.QueryRow(r.Context(), `
		select id::text from education_portfolios
		where id = $1::uuid and institution_id = $2 and owner_user_id = $3::uuid
		for update
	`, recordID, s.institutionID(r), actorID).Scan(&lockedID)
	if errors.Is(err, pgx.ErrNoRows) {
		writePortfolioAccessFailure(w, nil)
		return
	}
	if err != nil {
		writePortfolioDeclarationAcknowledgementFailure(w, "lock_portfolio", err)
		return
	}

	var template PortfolioDeclarationTemplate
	err = scanPortfolioDeclarationTemplate(tx.QueryRow(r.Context(), `
		select `+portfolioDeclarationTemplateColumns+`
		from education_portfolio_declaration_templates
		where declaration_type = $1 and lifecycle_status = 'published'
			and effective_from <= current_date
			and (effective_to is null or effective_to >= current_date)
		order by effective_from desc, created_at desc
		limit 1
		for share
	`, declarationType), &template)
	if errors.Is(err, pgx.ErrNoRows) {
		httpx.JSON(w, http.StatusServiceUnavailable, map[string]any{"code": "portfolio_declaration_template_unavailable"})
		return
	}
	if err != nil {
		writePortfolioDeclarationAcknowledgementFailure(w, "load_template", err)
		return
	}

	var acknowledgement PortfolioDeclarationAcknowledgement
	err = scanPortfolioDeclarationAcknowledgement(tx.QueryRow(r.Context(), `
		insert into education_portfolio_declaration_acknowledgements (
			portfolio_id, institution_id, tenant_code, declaration_type,
			declaration_version, declaration_text, accepted_by_user_id,
			attestation_method, attestation_evidence, signature_evidence
		) values (
			$1::uuid, $2, public.current_tenant_code(), $3, $4, $5, $6::uuid,
			$7, jsonb_build_object(
				'accepted_by_user_id', $6::text,
				'actor_subject', $8,
				'channel', 'authenticated_web'
			), '{}'::jsonb
		)
		on conflict (portfolio_id, declaration_type, declaration_version) do nothing
		returning `+portfolioDeclarationAcknowledgementColumns,
		recordID, s.institutionID(r), declarationType, template.DeclarationVersion,
		template.DeclarationText, actorID, portfolioDeclarationAcknowledgementMethod,
		authruntime.CurrentSubjectFromRequest(r)), &acknowledgement)
	created := err == nil
	if errors.Is(err, pgx.ErrNoRows) {
		err = scanPortfolioDeclarationAcknowledgement(tx.QueryRow(r.Context(), `
			select `+portfolioDeclarationAcknowledgementColumns+`
			from education_portfolio_declaration_acknowledgements
			where portfolio_id = $1::uuid and institution_id = $2
				and tenant_code = public.current_tenant_code()
				and declaration_type = $3 and declaration_version = $4
		`, recordID, s.institutionID(r), declarationType, template.DeclarationVersion), &acknowledgement)
	}
	if err != nil {
		writePortfolioDeclarationAcknowledgementFailure(w, "insert_or_load", err)
		return
	}
	if err := tx.Commit(r.Context()); err != nil {
		writePortfolioDeclarationAcknowledgementFailure(w, "commit", err)
		return
	}
	if created {
		s.logAudit(r, "education.portfolios.declaration.acknowledge", "portfolio_declaration_acknowledgement", acknowledgement.ID, "Portfolio declaration acknowledgement recorded.", map[string]any{"portfolio_id": recordID, "declaration_type": declarationType, "declaration_version": acknowledgement.DeclarationVersion})
	}
	// A retry of the same immutable acknowledgement returns the same evidence,
	// so this command is deliberately idempotent and always has one 200 contract.
	httpx.JSON(w, http.StatusOK, acknowledgement)
}

// PortfolioDeclarationEvidence gives an institution reviewer immutable
// declaration evidence. The route is protected by portfolio read/verify RBAC;
// this query additionally binds evidence to the active institution and RLS
// tenant so a reviewer cannot cross either boundary.
func (s *Service) PortfolioDeclarationEvidence(w http.ResponseWriter, r *http.Request) {
	recordID := strings.TrimSpace(chi.URLParam(r, "recordID"))
	if _, err := uuid.Parse(recordID); err != nil {
		httpx.JSON(w, http.StatusBadRequest, map[string]any{"code": "invalid_portfolio_id"})
		return
	}
	var exists bool
	err := s.pool.QueryRow(r.Context(), `
		select exists(select 1 from education_portfolios where id = $1::uuid and institution_id = $2)
	`, recordID, s.institutionID(r)).Scan(&exists)
	if err != nil {
		httpx.JSON(w, http.StatusInternalServerError, map[string]any{"code": "education_portfolio_declaration_evidence_failed"})
		return
	}
	if !exists {
		writeEducationNotFound(w, "education_portfolio_not_found")
		return
	}
	acknowledgements, err := s.portfolioDeclarationAcknowledgements(r, recordID)
	if err != nil {
		httpx.JSON(w, http.StatusInternalServerError, map[string]any{"code": "education_portfolio_declaration_evidence_failed"})
		return
	}
	httpx.JSON(w, http.StatusOK, PortfolioDeclarationEvidenceResponse{Acknowledgements: acknowledgements})
}
