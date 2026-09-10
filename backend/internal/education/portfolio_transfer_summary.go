package education

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"strings"
	"time"

	authruntime "github.com/eguilde/egueducation/internal/auth"
	"github.com/eguilde/egueducation/internal/httpx"
	"github.com/go-chi/chi/v5"
	"github.com/jackc/pgx/v5"
)

type PortfolioTransferSummary struct {
	Portfolio      PortfolioTransferSummaryPortfolio   `json:"portfolio"`
	Completeness   PortfolioCompletenessSummary        `json:"completeness"`
	Transfer       PortfolioTransferSummaryTransfer    `json:"transfer"`
	Mobility       PortfolioTransferSummaryMobility    `json:"mobility"`
	Valorification PortfolioValorificationSummaryBlock `json:"valorification"`
	Readiness      PortfolioTransferSummaryReadiness   `json:"readiness"`
}

type PortfolioTransferSummaryPortfolio struct {
	ID               string `json:"id"`
	PortfolioCode    string `json:"portfolio_code"`
	OwnerName        string `json:"owner_name"`
	OwnerRole        string `json:"owner_role"`
	OwnerPersonnelID string `json:"owner_personnel_id"`
	SchoolYear       string `json:"school_year"`
	TransferStatus   string `json:"transfer_status"`
}

type PortfolioTransferSummaryTransfer struct {
	TotalEvents      int                                `json:"total_events"`
	PreparedEvents   int                                `json:"prepared_events"`
	SentEvents       int                                `json:"sent_events"`
	ReceivedEvents   int                                `json:"received_events"`
	ClosedEvents     int                                `json:"closed_events"`
	LastTransfer     *PortfolioTransferSummaryLastEvent `json:"last_transfer"`
	CurrentDirection string                             `json:"current_direction"`
}

type PortfolioTransferSummaryLastEvent struct {
	ID                     string `json:"id"`
	TransferCode           string `json:"transfer_code"`
	TransferType           string `json:"transfer_type"`
	SourceInstitution      string `json:"source_institution"`
	DestinationInstitution string `json:"destination_institution"`
	Status                 string `json:"status"`
	HandoverOn             string `json:"handover_on"`
	ReceivedOn             string `json:"received_on"`
	HandoverBy             string `json:"handover_by"`
	ReceivedBy             string `json:"received_by"`
}

type PortfolioTransferSummaryMobility struct {
	MatchedCases        int `json:"matched_cases"`
	ActiveCases         int `json:"active_cases"`
	TransferCases       int `json:"transfer_cases"`
	DetachmentCases     int `json:"detachment_cases"`
	RestrictionCases    int `json:"restriction_cases"`
	CurrentUnitMentions int `json:"current_unit_mentions"`
	DestinationMentions int `json:"destination_mentions"`
}

type PortfolioTransferSummaryReadiness struct {
	ReadyToRequest bool     `json:"ready_to_request"`
	ReadyToSend    bool     `json:"ready_to_send"`
	ReadyToConfirm bool     `json:"ready_to_confirm"`
	ReadyToClose   bool     `json:"ready_to_close"`
	Blockers       []string `json:"blockers"`
}

type PortfolioCompletenessSummary struct {
	TotalDocuments          int      `json:"total_documents"`
	PortfolioDocuments      int      `json:"portfolio_documents"`
	PersonnelDocuments      int      `json:"personnel_documents"`
	SensitiveDocuments      int      `json:"sensitive_documents"`
	TotalChecklistItems     int      `json:"total_checklist_items"`
	MandatoryChecklistItems int      `json:"mandatory_checklist_items"`
	CompletedChecklistItems int      `json:"completed_checklist_items"`
	PartialChecklistItems   int      `json:"partial_checklist_items"`
	MissingChecklistItems   int      `json:"missing_checklist_items"`
	ReviewingChecklistItems int      `json:"reviewing_checklist_items"`
	OpisEntries             int      `json:"opis_entries"`
	CustodyEvents           int      `json:"custody_events"`
	ReviewEvents            int      `json:"review_events"`
	ValorificationEvents    int      `json:"valorification_events"`
	ReadyForReview          bool     `json:"ready_for_review"`
	ReadyForTransfer        bool     `json:"ready_for_transfer"`
	Blockers                []string `json:"blockers"`
}

type PortfolioValorificationSummaryBlock struct {
	TotalEvents       int                                  `json:"total_events"`
	OpenEvents        int                                  `json:"open_events"`
	CompletedEvents   int                                  `json:"completed_events"`
	LegacyEvents      int                                  `json:"legacy_events"`
	LinkedEvaluations int                                  `json:"linked_evaluations"`
	LinkedMobility    int                                  `json:"linked_mobility"`
	LinkedMerit       int                                  `json:"linked_merit"`
	LastEvent         *PortfolioValorificationSummaryEvent `json:"last_event"`
	Scopes            []PortfolioValorificationScopeStat   `json:"scopes"`
}

type PortfolioValorificationSummaryEvent struct {
	ID                 string `json:"id"`
	ValorificationCode string `json:"valorification_code"`
	Scope              string `json:"scope"`
	Status             string `json:"status"`
	RequestedBy        string `json:"requested_by"`
	TargetInstitution  string `json:"target_institution"`
	TargetReference    string `json:"target_reference"`
	StartedOn          string `json:"started_on"`
	CompletedOn        string `json:"completed_on"`
}

type PortfolioValorificationScopeStat struct {
	Scope     string `json:"scope"`
	Total     int    `json:"total"`
	Open      int    `json:"open"`
	Completed int    `json:"completed"`
}

type AdvancePortfolioTransferRequest struct {
	Action string `json:"action"`
}

const portfolioTransferRowColumns = `
	id::text, portfolio_id::text, transfer_code, transfer_type,
	source_institution, destination_institution, status,
	to_char(handover_on, 'YYYY-MM-DD'), coalesce(to_char(received_on, 'YYYY-MM-DD'), ''),
	handover_by, received_by, institution_id, notes,
	routing_version, coalesce(source_tenant_code, ''), coalesce(source_institution_id, ''),
	coalesce(destination_tenant_code, ''), coalesce(destination_institution_id, ''),
	coalesce(export_manifest_id::text, ''),
	coalesce(to_char(sent_at, 'YYYY-MM-DD"T"HH24:MI:SS.US"Z"'), ''), sent_by_subject,
	coalesce(to_char(received_at, 'YYYY-MM-DD"T"HH24:MI:SS.US"Z"'), ''), received_by_subject,
	coalesce(to_char(closed_at, 'YYYY-MM-DD"T"HH24:MI:SS.US"Z"'), ''), closed_by_subject`

type portfolioTransferRowScanner interface {
	Scan(dest ...any) error
}

func scanPortfolioTransfer(row portfolioTransferRowScanner) (PortfolioTransferEvent, error) {
	var item PortfolioTransferEvent
	err := row.Scan(
		&item.ID, &item.PortfolioID, &item.TransferCode, &item.TransferType,
		&item.SourceInstitution, &item.DestinationInstitution, &item.Status,
		&item.HandoverOn, &item.ReceivedOn, &item.HandoverBy, &item.ReceivedBy,
		&item.InstitutionID, &item.Notes, &item.RoutingVersion,
		&item.SourceTenantCode, &item.SourceInstitutionID,
		&item.DestinationTenantCode, &item.DestinationInstitutionID,
		&item.ExportManifestID, &item.SentAt, &item.SentBySubject,
		&item.ReceivedAt, &item.ReceivedBySubject, &item.ClosedAt, &item.ClosedBySubject,
	)
	return item, err
}

func (s *Service) PortfolioTransferSummary(w http.ResponseWriter, r *http.Request) {
	recordID := strings.TrimSpace(chi.URLParam(r, "recordID"))
	institutionID := s.institutionID(r)

	var summary PortfolioTransferSummary
	err := s.pool.QueryRow(r.Context(), `
		select
			id::text,
			portfolio_code,
			owner_name,
			owner_role,
			coalesce(owner_personnel_id::text, ''),
			school_year,
			transfer_status
		from education_portfolios
		where id = $1::uuid and institution_id = $2
	`, recordID, institutionID).Scan(
		&summary.Portfolio.ID,
		&summary.Portfolio.PortfolioCode,
		&summary.Portfolio.OwnerName,
		&summary.Portfolio.OwnerRole,
		&summary.Portfolio.OwnerPersonnelID,
		&summary.Portfolio.SchoolYear,
		&summary.Portfolio.TransferStatus,
	)
	if errors.Is(err, pgx.ErrNoRows) {
		writeEducationNotFound(w, "education_portfolio_not_found")
		return
	}
	if err != nil {
		httpx.JSON(w, http.StatusInternalServerError, map[string]any{"code": "education_portfolio_transfer_summary_failed"})
		return
	}

	if err := s.pool.QueryRow(r.Context(), `
		select
			coalesce((select count(*) from education_portfolio_documents epd where epd.portfolio_id = $1::uuid and epd.institution_id = $2 and epd.status = 'active'), 0) as total_documents,
			coalesce((select count(*) from education_portfolio_documents epd where epd.portfolio_id = $1::uuid and epd.institution_id = $2 and epd.status = 'active' and epd.source_scope = 'portofoliu'), 0) as portfolio_documents,
			coalesce((select count(*) from education_portfolio_documents epd where epd.portfolio_id = $1::uuid and epd.institution_id = $2 and epd.status = 'active' and epd.source_scope = 'dosar_personal'), 0) as personnel_documents,
			coalesce((select count(*) from education_portfolio_documents epd where epd.portfolio_id = $1::uuid and epd.institution_id = $2 and epd.status = 'active' and epd.sensitive_data = true), 0) as sensitive_documents,
			coalesce((select count(*) from education_portfolio_checklist epc where epc.portfolio_id = $1::uuid and epc.institution_id = $2), 0) as total_checklist_items,
			coalesce((select count(*) from education_portfolio_checklist epc where epc.portfolio_id = $1::uuid and epc.institution_id = $2 and epc.mandatory = true), 0) as mandatory_checklist_items,
			coalesce((select count(*) from education_portfolio_checklist epc where epc.portfolio_id = $1::uuid and epc.institution_id = $2 and epc.status = 'complet'), 0) as completed_checklist_items,
			coalesce((select count(*) from education_portfolio_checklist epc where epc.portfolio_id = $1::uuid and epc.institution_id = $2 and epc.status = 'partial'), 0) as partial_checklist_items,
			coalesce((select count(*) from education_portfolio_checklist epc where epc.portfolio_id = $1::uuid and epc.institution_id = $2 and epc.status = 'lipsa'), 0) as missing_checklist_items,
			coalesce((select count(*) from education_portfolio_checklist epc where epc.portfolio_id = $1::uuid and epc.institution_id = $2 and epc.status = 'in_verificare'), 0) as reviewing_checklist_items,
			coalesce((select count(*) from education_portfolio_opis epo where epo.portfolio_id = $1::uuid and epo.institution_id = $2), 0) as opis_entries,
			coalesce((select count(*) from education_portfolio_custody epc where epc.portfolio_id = $1::uuid and epc.institution_id = $2), 0) as custody_events,
			coalesce((select count(*) from education_portfolio_reviews epr where epr.portfolio_id = $1::uuid and epr.institution_id = $2), 0) as review_events,
			coalesce((select count(*) from education_portfolio_valorification_packages package where package.portfolio_id = $1::uuid and package.institution_id = $2 and package.status in ('validated', 'completed')), 0) as valorification_events
	`, recordID, institutionID).Scan(
		&summary.Completeness.TotalDocuments,
		&summary.Completeness.PortfolioDocuments,
		&summary.Completeness.PersonnelDocuments,
		&summary.Completeness.SensitiveDocuments,
		&summary.Completeness.TotalChecklistItems,
		&summary.Completeness.MandatoryChecklistItems,
		&summary.Completeness.CompletedChecklistItems,
		&summary.Completeness.PartialChecklistItems,
		&summary.Completeness.MissingChecklistItems,
		&summary.Completeness.ReviewingChecklistItems,
		&summary.Completeness.OpisEntries,
		&summary.Completeness.CustodyEvents,
		&summary.Completeness.ReviewEvents,
		&summary.Completeness.ValorificationEvents,
	); err != nil {
		httpx.JSON(w, http.StatusInternalServerError, map[string]any{"code": "education_portfolio_transfer_summary_failed"})
		return
	}

	completenessBlockers := make([]string, 0, 4)
	if summary.Completeness.TotalDocuments == 0 {
		completenessBlockers = append(completenessBlockers, "portfolio_documents_missing")
	}
	if summary.Completeness.OpisEntries == 0 {
		completenessBlockers = append(completenessBlockers, "opis_missing")
	}
	if summary.Completeness.MandatoryChecklistItems > summary.Completeness.CompletedChecklistItems {
		completenessBlockers = append(completenessBlockers, "mandatory_checklist_pending")
	}
	if summary.Completeness.ReviewEvents == 0 {
		completenessBlockers = append(completenessBlockers, "no_review_history")
	}
	summary.Completeness.Blockers = completenessBlockers
	summary.Completeness.ReadyForReview = len(completenessBlockers) == 0
	summary.Completeness.ReadyForTransfer = summary.Completeness.ReadyForReview && summary.Completeness.TotalDocuments > 0 && summary.Transfer.TotalEvents > 0

	if err := s.pool.QueryRow(r.Context(), `
		select
			count(*) as total_events,
			count(*) filter (where status = 'pregatit') as prepared_events,
			count(*) filter (where status = 'trimis') as sent_events,
			count(*) filter (where status = 'receptionat') as received_events,
			count(*) filter (where status = 'inchis') as closed_events
		from education_portfolio_transfers
		where portfolio_id = $1::uuid and institution_id = $2 and withdrawn_at is null
	`, recordID, institutionID).Scan(
		&summary.Transfer.TotalEvents,
		&summary.Transfer.PreparedEvents,
		&summary.Transfer.SentEvents,
		&summary.Transfer.ReceivedEvents,
		&summary.Transfer.ClosedEvents,
	); err != nil {
		httpx.JSON(w, http.StatusInternalServerError, map[string]any{"code": "education_portfolio_transfer_summary_failed"})
		return
	}

	var lastTransfer PortfolioTransferSummaryLastEvent
	err = s.pool.QueryRow(r.Context(), `
		select
			id::text,
			transfer_code,
			transfer_type,
			source_institution,
			destination_institution,
			status,
			to_char(handover_on, 'YYYY-MM-DD'),
			coalesce(to_char(received_on, 'YYYY-MM-DD'), ''),
			handover_by,
			received_by
		from education_portfolio_transfers
		where portfolio_id = $1::uuid and institution_id = $2
		order by handover_on desc, created_at desc
		limit 1
	`, recordID, institutionID).Scan(
		&lastTransfer.ID,
		&lastTransfer.TransferCode,
		&lastTransfer.TransferType,
		&lastTransfer.SourceInstitution,
		&lastTransfer.DestinationInstitution,
		&lastTransfer.Status,
		&lastTransfer.HandoverOn,
		&lastTransfer.ReceivedOn,
		&lastTransfer.HandoverBy,
		&lastTransfer.ReceivedBy,
	)
	switch {
	case errors.Is(err, pgx.ErrNoRows):
		summary.Transfer.LastTransfer = nil
	case err != nil:
		httpx.JSON(w, http.StatusInternalServerError, map[string]any{"code": "education_portfolio_transfer_summary_failed"})
		return
	default:
		summary.Transfer.LastTransfer = &lastTransfer
		summary.Transfer.CurrentDirection = fmt.Sprintf("%s -> %s", lastTransfer.SourceInstitution, lastTransfer.DestinationInstitution)
	}

	if err := s.pool.QueryRow(r.Context(), `
		select
			count(*) as matched_cases,
			count(*) filter (where status in ('open', 'pending')) as active_cases,
			count(*) filter (where request_type = 'transfer') as transfer_cases,
			count(*) filter (where request_type = 'detasare') as detachment_cases,
			count(*) filter (where request_type = 'restrangere') as restriction_cases,
			count(*) filter (where trim(coalesce(source_school, '')) <> '') as current_unit_mentions,
			count(*) filter (where trim(coalesce(destination_school, '')) <> '') as destination_mentions
		from education_mobility_cases
		where institution_id = $1
			and school_year = $2
			and personnel_id = nullif($3, '')::uuid
	`, institutionID, summary.Portfolio.SchoolYear, summary.Portfolio.OwnerPersonnelID).Scan(
		&summary.Mobility.MatchedCases,
		&summary.Mobility.ActiveCases,
		&summary.Mobility.TransferCases,
		&summary.Mobility.DetachmentCases,
		&summary.Mobility.RestrictionCases,
		&summary.Mobility.CurrentUnitMentions,
		&summary.Mobility.DestinationMentions,
	); err != nil {
		httpx.JSON(w, http.StatusInternalServerError, map[string]any{"code": "education_portfolio_transfer_summary_failed"})
		return
	}

	if err := s.pool.QueryRow(r.Context(), `
		select
			count(*) filter (where status in ('validated', 'completed')) as total_events,
			count(*) filter (where status = 'validated') as open_events,
			count(*) filter (where status = 'completed') as completed_events,
			(select count(*) from education_portfolio_valorifications legacy where legacy.portfolio_id = $1::uuid and legacy.institution_id = $2) as legacy_events
		from education_portfolio_valorification_packages
		where portfolio_id = $1::uuid and institution_id = $2
	`, recordID, institutionID).Scan(
		&summary.Valorification.TotalEvents,
		&summary.Valorification.OpenEvents,
		&summary.Valorification.CompletedEvents,
		&summary.Valorification.LegacyEvents,
	); err != nil {
		httpx.JSON(w, http.StatusInternalServerError, map[string]any{"code": "education_portfolio_transfer_summary_failed"})
		return
	}

	rows, err := s.pool.Query(r.Context(), `
		select
			scope,
			count(*) filter (where status in ('validated', 'completed')) as total,
			count(*) filter (where status = 'validated') as open,
			count(*) filter (where status = 'completed') as completed
		from education_portfolio_valorification_packages
		where portfolio_id = $1::uuid and institution_id = $2
		group by scope
		order by scope
	`, recordID, institutionID)
	if err != nil {
		httpx.JSON(w, http.StatusInternalServerError, map[string]any{"code": "education_portfolio_transfer_summary_failed"})
		return
	}
	defer rows.Close()
	scopeStats := make([]PortfolioValorificationScopeStat, 0)
	for rows.Next() {
		var stat PortfolioValorificationScopeStat
		if err := rows.Scan(&stat.Scope, &stat.Total, &stat.Open, &stat.Completed); err != nil {
			httpx.JSON(w, http.StatusInternalServerError, map[string]any{"code": "education_portfolio_transfer_summary_failed"})
			return
		}
		scopeStats = append(scopeStats, stat)
	}
	if err := rows.Err(); err != nil {
		httpx.JSON(w, http.StatusInternalServerError, map[string]any{"code": "education_portfolio_transfer_summary_failed"})
		return
	}
	summary.Valorification.Scopes = scopeStats

	var lastValorification PortfolioValorificationSummaryEvent
	err = s.pool.QueryRow(r.Context(), `
		select
			id::text,
			'PKG-' || left(id::text, 8),
			scope,
			status,
			created_by_subject,
			institution_id,
			coalesce(source_evaluation_id::text, source_mobility_case_id::text, source_merit_grant_id::text),
			to_char(created_at, 'YYYY-MM-DD'),
			coalesce(to_char(completed_at, 'YYYY-MM-DD'), '')
		from education_portfolio_valorification_packages
		where portfolio_id = $1::uuid and institution_id = $2 and status in ('validated', 'completed')
		order by coalesce(completed_at, validated_at, created_at) desc, created_at desc
		limit 1
	`, recordID, institutionID).Scan(
		&lastValorification.ID,
		&lastValorification.ValorificationCode,
		&lastValorification.Scope,
		&lastValorification.Status,
		&lastValorification.RequestedBy,
		&lastValorification.TargetInstitution,
		&lastValorification.TargetReference,
		&lastValorification.StartedOn,
		&lastValorification.CompletedOn,
	)
	switch {
	case errors.Is(err, pgx.ErrNoRows):
		summary.Valorification.LastEvent = nil
	case err != nil:
		httpx.JSON(w, http.StatusInternalServerError, map[string]any{"code": "education_portfolio_transfer_summary_failed"})
		return
	default:
		summary.Valorification.LastEvent = &lastValorification
	}

	if err := s.pool.QueryRow(r.Context(), `
		select count(*) from education_evaluations
		where institution_id=$1 and school_year=$2 and personnel_id=nullif($3,'')::uuid
	`, institutionID, summary.Portfolio.SchoolYear, summary.Portfolio.OwnerPersonnelID).Scan(&summary.Valorification.LinkedEvaluations); err != nil {
		httpx.JSON(w, http.StatusInternalServerError, map[string]any{"code": "education_portfolio_transfer_summary_failed"})
		return
	}
	if err := s.pool.QueryRow(r.Context(), `
		select count(*) from education_mobility_cases
		where institution_id=$1 and school_year=$2 and personnel_id=nullif($3,'')::uuid
	`, institutionID, summary.Portfolio.SchoolYear, summary.Portfolio.OwnerPersonnelID).Scan(&summary.Valorification.LinkedMobility); err != nil {
		httpx.JSON(w, http.StatusInternalServerError, map[string]any{"code": "education_portfolio_transfer_summary_failed"})
		return
	}
	if err := s.pool.QueryRow(r.Context(), `
		select count(*) from education_merit_grants
		where institution_id=$1 and school_year=$2 and personnel_id=nullif($3,'')::uuid
	`, institutionID, summary.Portfolio.SchoolYear, summary.Portfolio.OwnerPersonnelID).Scan(&summary.Valorification.LinkedMerit); err != nil {
		httpx.JSON(w, http.StatusInternalServerError, map[string]any{"code": "education_portfolio_transfer_summary_failed"})
		return
	}

	blockers := make([]string, 0, 4)
	summary.Readiness.ReadyToRequest = summary.Transfer.TotalEvents == 0
	summary.Readiness.ReadyToSend = summary.Transfer.LastTransfer != nil && summary.Transfer.LastTransfer.Status == "pregatit"
	summary.Readiness.ReadyToConfirm = summary.Transfer.LastTransfer != nil && summary.Transfer.LastTransfer.Status == "trimis"
	summary.Readiness.ReadyToClose = summary.Transfer.LastTransfer != nil && summary.Transfer.LastTransfer.Status == "receptionat"

	if summary.Mobility.ActiveCases > 0 && summary.Transfer.TotalEvents == 0 {
		blockers = append(blockers, "mobilitate_activa_fara_transfer_portofoliu")
	}
	if summary.Portfolio.TransferStatus != "none" && summary.Transfer.TotalEvents == 0 {
		blockers = append(blockers, "status_portofoliu_fara_eveniment_transfer")
	}
	if summary.Transfer.LastTransfer != nil && strings.EqualFold(strings.TrimSpace(summary.Transfer.LastTransfer.SourceInstitution), strings.TrimSpace(summary.Transfer.LastTransfer.DestinationInstitution)) {
		blockers = append(blockers, "institutie_sursa_si_destinatie_identice")
	}
	if summary.Mobility.DetachmentCases > 0 && summary.Transfer.LastTransfer == nil {
		blockers = append(blockers, "detasare_fara_circuit_digital_portofoliu")
	}
	if summary.Valorification.LinkedEvaluations > 0 && !portfolioValorificationScopePresent(summary.Valorification.Scopes, "evaluare_profesionala") {
		blockers = append(blockers, "evaluare_fara_flux_valorificare_portofoliu")
	}
	if summary.Valorification.LinkedMobility > 0 && !portfolioValorificationScopePresent(summary.Valorification.Scopes, "mobilitate") {
		blockers = append(blockers, "mobilitate_fara_flux_valorificare_portofoliu")
	}
	if summary.Valorification.LinkedMerit > 0 &&
		!portfolioValorificationScopePresent(summary.Valorification.Scopes, "gradatie_merit") &&
		!portfolioValorificationScopePresent(summary.Valorification.Scopes, "distinctie_premiu") {
		blockers = append(blockers, "gradatie_sau_distinctie_fara_flux_valorificare_portofoliu")
	}
	if summary.Transfer.TotalEvents > 0 && summary.Valorification.TotalEvents == 0 {
		blockers = append(blockers, "transfer_fara_fluxuri_de_valorificare_documentate")
	}
	summary.Readiness.Blockers = blockers

	httpx.JSON(w, http.StatusOK, summary)
}

func (s *Service) AdvancePortfolioTransfer(w http.ResponseWriter, r *http.Request) {
	recordID := strings.TrimSpace(chi.URLParam(r, "recordID"))
	itemID := strings.TrimSpace(chi.URLParam(r, "itemID"))
	var req AdvancePortfolioTransferRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		httpx.JSON(w, http.StatusBadRequest, map[string]any{"code": "invalid_portfolio_transfer_advance_payload"})
		return
	}
	req.Action = strings.TrimSpace(req.Action)
	if req.Action == "" {
		httpx.JSON(w, http.StatusBadRequest, map[string]any{"code": "missing_portfolio_transfer_advance_action"})
		return
	}

	institutionID := s.institutionID(r)
	var item PortfolioTransferEvent
	var err error
	switch req.Action {
	case "mark_sent":
		var prepared bool
		if err := s.pool.QueryRow(r.Context(), `select exists(
			select 1 from education_portfolio_transfers
			where id=$1::uuid and portfolio_id=$2::uuid and institution_id=$3
				and status='pregatit' and withdrawn_at is null
		)`, itemID, recordID, institutionID).Scan(&prepared); err != nil {
			httpx.JSON(w, http.StatusInternalServerError, map[string]any{"code": "portfolio_transfer_advance_failed"})
			return
		}
		if !prepared {
			writeEducationNotFound(w, "education_portfolio_transfer_not_found")
			return
		}
		manifest, manifestErr := s.createPortfolioExportManifestEvidence(r, recordID)
		if errors.Is(manifestErr, errPortfolioExportNotReady) {
			httpx.JSON(w, http.StatusUnprocessableEntity, map[string]any{"code": "education_portfolio_export_provenance_incomplete"})
			return
		}
		if manifestErr != nil {
			httpx.JSON(w, http.StatusInternalServerError, map[string]any{"code": "portfolio_transfer_manifest_failed"})
			return
		}
		actorSubject := strings.TrimSpace(authruntime.CurrentSubjectFromRequest(r))
		item, err = scanPortfolioTransfer(s.pool.QueryRow(r.Context(), `
			update education_portfolio_transfers
			set status = 'trimis', export_manifest_id = $4::uuid,
				sent_at = now(), sent_by_subject = $5, updated_at = now()
			where id = $1 and portfolio_id = $2 and institution_id = $3 and status = 'pregatit'
				and (routing_version = 1 or (source_tenant_code=public.current_tenant_code() and source_institution_id=public.current_institution_id()))
			returning `+portfolioTransferRowColumns+`
		`, itemID, recordID, institutionID, manifest.ExportManifestID, actorSubject))
	case "confirm_received":
		// Version 2 transfers must be accepted from the destination tenant's
		// inbox endpoint.  This legacy action remains available only for rows
		// that pre-date tenant-bound routing.
		receivedOn := time.Now().Format("2006-01-02")
		receivedBy, actorErr := s.portfolioOpisCheckedBy(r)
		if actorErr != nil {
			httpx.JSON(w, http.StatusInternalServerError, map[string]any{"code": "portfolio_transfer_advance_failed"})
			return
		}
		item, err = scanPortfolioTransfer(s.pool.QueryRow(r.Context(), `
			update education_portfolio_transfers
			set status = 'receptionat',
				received_on = coalesce(received_on, $4),
				received_by = case when trim(coalesce(received_by, '')) = '' then $5 else received_by end,
				updated_at = now()
			where id = $1 and portfolio_id = $2 and institution_id = $3 and status = 'trimis' and routing_version = 1
			returning `+portfolioTransferRowColumns+`
		`, itemID, recordID, institutionID, receivedOn, receivedBy))
	case "close_transfer":
		actorSubject := strings.TrimSpace(authruntime.CurrentSubjectFromRequest(r))
		item, err = scanPortfolioTransfer(s.pool.QueryRow(r.Context(), `
			update education_portfolio_transfers
			set status = 'inchis', closed_at = case when routing_version=2 then now() else closed_at end,
				closed_by_subject = case when routing_version=2 then $4 else closed_by_subject end,
				updated_at = now()
			where id = $1 and portfolio_id = $2 and institution_id = $3 and status = 'receptionat'
			returning `+portfolioTransferRowColumns+`
		`, itemID, recordID, institutionID, actorSubject))
	default:
		httpx.JSON(w, http.StatusBadRequest, map[string]any{"code": "invalid_portfolio_transfer_advance_action"})
		return
	}
	if errors.Is(err, pgx.ErrNoRows) {
		writeEducationNotFound(w, "education_portfolio_transfer_not_found")
		return
	}
	if err != nil {
		httpx.JSON(w, http.StatusInternalServerError, map[string]any{"code": "portfolio_transfer_advance_failed"})
		return
	}

	if err := s.syncPortfolioTransferStatus(r.Context(), recordID, institutionID); err != nil {
		httpx.JSON(w, http.StatusInternalServerError, map[string]any{"code": "portfolio_transfer_advance_failed"})
		return
	}

	s.logAudit(r, "education.portfolios.transfer.advance", "portfolio_transfer", item.ID, "Portfolio transfer advanced procedurally.", map[string]any{
		"portfolio_id":   item.PortfolioID,
		"transfer_code":  item.TransferCode,
		"result_status":  item.Status,
		"advance_action": req.Action,
	})
	httpx.JSON(w, http.StatusOK, item)
}

// PortfolioTransferInbox exposes only sealed packages addressed to the
// authenticated tenant. Prepared source-side drafts never cross the boundary.
func (s *Service) PortfolioTransferInbox(w http.ResponseWriter, r *http.Request) {
	query := httpx.ParsePageQuery(r.URL.Query(), map[string]struct{}{
		"transfer_code":      {},
		"source_institution": {},
		"status":             {},
	}, []string{"transfer_code", "source_institution", "status"})
	if query.Sort == "" {
		query.Sort = "sent_at"
	}
	clauses := []string{
		"routing_version=2",
		"destination_tenant_code=public.current_tenant_code()",
		"destination_institution_id=public.current_institution_id()",
		"status in ('trimis','receptionat','inchis')",
		"withdrawn_at is null",
	}
	args := make([]any, 0, 5)
	for key, value := range query.Filters {
		value = strings.TrimSpace(value)
		if value == "" {
			continue
		}
		args = append(args, "%"+value+"%")
		placeholder := fmt.Sprintf("$%d", len(args))
		switch key {
		case "transfer_code":
			clauses = append(clauses, "transfer_code ilike "+placeholder)
		case "source_institution":
			clauses = append(clauses, "source_institution ilike "+placeholder)
		case "status":
			clauses = append(clauses, "status ilike "+placeholder)
		}
	}
	whereClause := " where " + strings.Join(clauses, " and ")
	var total int
	if err := s.pool.QueryRow(r.Context(), "select count(*) from education_portfolio_transfers"+whereClause, args...).Scan(&total); err != nil {
		httpx.JSON(w, http.StatusInternalServerError, map[string]any{"code": "portfolio_transfer_inbox_failed"})
		return
	}
	args = append(args, query.PageSize, (query.Page-1)*query.PageSize)
	rows, err := s.pool.Query(r.Context(), fmt.Sprintf(`
		select %s from education_portfolio_transfers
		%s
		order by %s %s, sent_at desc, transfer_code
		limit $%d offset $%d
	`, portfolioTransferRowColumns, whereClause, portfolioTransferInboxSortColumn(query.Sort), strings.ToUpper(query.Direction), len(args)-1, len(args)), args...)
	if err != nil {
		httpx.JSON(w, http.StatusInternalServerError, map[string]any{"code": "portfolio_transfer_inbox_failed"})
		return
	}
	defer rows.Close()
	items := make([]PortfolioTransferEvent, 0, query.PageSize)
	for rows.Next() {
		item, scanErr := scanPortfolioTransfer(rows)
		if scanErr != nil {
			httpx.JSON(w, http.StatusInternalServerError, map[string]any{"code": "portfolio_transfer_inbox_scan_failed"})
			return
		}
		items = append(items, item)
	}
	if err := rows.Err(); err != nil {
		httpx.JSON(w, http.StatusInternalServerError, map[string]any{"code": "portfolio_transfer_inbox_scan_failed"})
		return
	}
	httpx.WritePage(w, http.StatusOK, items, total, query.Page, query.PageSize)
}

func (s *Service) PortfolioTransferDestinations(w http.ResponseWriter, r *http.Request) {
	rows, err := s.pool.Query(r.Context(), `
		select tenant_code, institution_id, display_name, short_name
		from public.education_portfolio_transfer_destinations()
	`)
	if err != nil {
		httpx.JSON(w, http.StatusInternalServerError, map[string]any{"code": "portfolio_transfer_destinations_failed"})
		return
	}
	defer rows.Close()
	items := make([]PortfolioTransferDestination, 0)
	for rows.Next() {
		var item PortfolioTransferDestination
		if err := rows.Scan(&item.TenantCode, &item.InstitutionID, &item.DisplayName, &item.ShortName); err != nil {
			httpx.JSON(w, http.StatusInternalServerError, map[string]any{"code": "portfolio_transfer_destinations_failed"})
			return
		}
		items = append(items, item)
	}
	if err := rows.Err(); err != nil {
		httpx.JSON(w, http.StatusInternalServerError, map[string]any{"code": "portfolio_transfer_destinations_failed"})
		return
	}
	httpx.JSON(w, http.StatusOK, items)
}

func (s *Service) AcceptIntertenantPortfolioTransfer(w http.ResponseWriter, r *http.Request) {
	itemID := strings.TrimSpace(chi.URLParam(r, "itemID"))
	actorSubject := strings.TrimSpace(authruntime.CurrentSubjectFromRequest(r))
	if actorSubject == "" {
		httpx.JSON(w, http.StatusUnauthorized, map[string]any{"code": "portfolio_transfer_actor_required"})
		return
	}
	receivedBy, err := s.portfolioOpisCheckedBy(r)
	if err != nil {
		httpx.JSON(w, http.StatusInternalServerError, map[string]any{"code": "portfolio_transfer_accept_failed"})
		return
	}
	item, err := scanPortfolioTransfer(s.pool.QueryRow(r.Context(), `
		update education_portfolio_transfers
		set status='receptionat', received_on=current_date, received_at=now(),
			received_by=$2, received_by_subject=$3, updated_at=now()
		where id=$1::uuid and routing_version=2 and status='trimis'
			and destination_tenant_code=public.current_tenant_code()
			and destination_institution_id=public.current_institution_id()
			and export_manifest_id is not null and withdrawn_at is null
		returning `+portfolioTransferRowColumns+`
	`, itemID, receivedBy, actorSubject))
	if errors.Is(err, pgx.ErrNoRows) {
		writeEducationNotFound(w, "education_portfolio_transfer_inbox_item_not_found")
		return
	}
	if err != nil {
		httpx.JSON(w, http.StatusInternalServerError, map[string]any{"code": "portfolio_transfer_accept_failed"})
		return
	}
	s.logAudit(r, "education.portfolios.transfer.receive", "portfolio_transfer", item.ID, "Inter-tenant portfolio transfer accepted by destination institution.", map[string]any{
		"source_tenant_code": item.SourceTenantCode, "source_institution_id": item.SourceInstitutionID,
		"destination_tenant_code": item.DestinationTenantCode, "destination_institution_id": item.DestinationInstitutionID,
		"export_manifest_id": item.ExportManifestID, "transfer_code": item.TransferCode,
	})
	httpx.JSON(w, http.StatusOK, item)
}

func portfolioTransferInboxSortColumn(sort string) string {
	switch sort {
	case "transfer_code":
		return "transfer_code"
	case "source_institution":
		return "source_institution"
	case "status":
		return "status"
	default:
		return "sent_at"
	}
}

func (s *Service) syncPortfolioTransferStatus(ctx context.Context, recordID string, institutionID string) error {
	var latestStatus string
	err := s.pool.QueryRow(ctx, `
		select status
		from education_portfolio_transfers
		where portfolio_id = $1::uuid and institution_id = $2
		order by handover_on desc, created_at desc
		limit 1
	`, recordID, institutionID).Scan(&latestStatus)
	switch {
	case errors.Is(err, pgx.ErrNoRows):
		latestStatus = "none"
	case err != nil:
		return fmt.Errorf("load latest portfolio transfer status: %w", err)
	}

	portfolioStatus := latestStatus
	switch latestStatus {
	case "pregatit":
		portfolioStatus = "prepared"
	case "trimis":
		portfolioStatus = "sent"
	case "receptionat", "inchis":
		portfolioStatus = "received"
	case "none":
		portfolioStatus = "none"
	default:
		portfolioStatus = "none"
	}

	if _, err := s.pool.Exec(ctx, `
		update education_portfolios
		set transfer_status = $1, updated_at = now()
		where id = $2::uuid and institution_id = $3
	`, portfolioStatus, recordID, institutionID); err != nil {
		return fmt.Errorf("sync portfolio transfer status on portfolio: %w", err)
	}
	return nil
}

func portfolioValorificationScopePresent(scopes []PortfolioValorificationScopeStat, scope string) bool {
	for _, item := range scopes {
		if item.Scope == scope && item.Total > 0 {
			return true
		}
	}
	return false
}
