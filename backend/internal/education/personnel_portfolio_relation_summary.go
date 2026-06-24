package education

import (
	"errors"
	"net/http"
	"strings"

	"github.com/go-chi/chi/v5"
	"github.com/jackc/pgx/v5"

	"github.com/eguilde/egueducation/internal/httpx"
)

func (s *Service) PersonnelPortfolioDossierSummary(w http.ResponseWriter, r *http.Request) {
	recordID := strings.TrimSpace(chi.URLParam(r, "recordID"))
	institutionID := s.institutionID(r)

	var summary PersonnelPortfolioDossierSummary
	err := s.pool.QueryRow(r.Context(), `
		select
			id::text,
			full_name,
			role_title,
			school_year,
			has_portfolio
		from education_personnel
		where id = $1::uuid and institution_id = $2
	`, recordID, institutionID).Scan(
		&summary.Personnel.ID,
		&summary.Personnel.FullName,
		&summary.Personnel.RoleTitle,
		&summary.Personnel.SchoolYear,
		&summary.Personnel.HasPortfolio,
	)
	if errors.Is(err, pgx.ErrNoRows) {
		writeEducationNotFound(w, "education_personnel_record_not_found")
		return
	}
	if err != nil {
		httpx.JSON(w, http.StatusInternalServerError, map[string]any{"code": "education_personnel_portfolio_dossier_summary_failed"})
		return
	}

	if err := s.pool.QueryRow(r.Context(), `
		select
			count(*) as total_documents,
			count(*) filter (where file_scope = 'dosar_personal') as personal_file_documents,
			count(*) filter (where file_scope = 'dosar_director') as director_file_documents,
			count(*) filter (where file_scope = 'dosar_director_adjunct') as adjunct_director_file_documents,
			count(*) filter (where sensitive_data = true) as sensitive_documents,
			count(*) filter (where included_in_portfolio = true) as documents_marked_for_portfolio,
			count(*) filter (where document_category = 'evaluare') as evaluation_documents,
			count(*) filter (where document_category in ('studii', 'cariera', 'management')) as administrative_career_documents
		from education_personnel_file_documents
		where personnel_id = $1::uuid and institution_id = $2
	`, recordID, institutionID).Scan(
		&summary.Dossier.TotalDocuments,
		&summary.Dossier.PersonalFileDocuments,
		&summary.Dossier.DirectorFileDocuments,
		&summary.Dossier.AdjunctDirectorFileDocuments,
		&summary.Dossier.SensitiveDocuments,
		&summary.Dossier.DocumentsMarkedForPortfolio,
		&summary.Dossier.EvaluationDocuments,
		&summary.Dossier.AdministrativeCareerDocs,
	); err != nil {
		httpx.JSON(w, http.StatusInternalServerError, map[string]any{"code": "education_personnel_portfolio_dossier_summary_failed"})
		return
	}

	if err := s.pool.QueryRow(r.Context(), `
		with matched_portfolios as (
			select id, last_updated_on, status
			from education_portfolios
			where institution_id = $1
				and school_year = $2
				and lower(trim(owner_name)) = lower(trim($3))
		)
		select
			count(*) as matched_records,
			count(*) filter (where status = 'validated') as validated_records,
			coalesce((
				select count(*)
				from education_portfolio_documents epd
				where epd.institution_id = $1
					and epd.portfolio_id in (select id from matched_portfolios)
			), 0) as total_documents,
			coalesce((
				select count(*)
				from education_portfolio_documents epd
				where epd.institution_id = $1
					and epd.portfolio_id in (select id from matched_portfolios)
					and epd.source_scope = 'portofoliu'
			), 0) as portfolio_scope_documents,
			coalesce((
				select count(*)
				from education_portfolio_documents epd
				where epd.institution_id = $1
					and epd.portfolio_id in (select id from matched_portfolios)
					and epd.source_scope = 'dosar_personal'
			), 0) as personnel_scope_documents,
			coalesce((
				select count(*)
				from education_portfolio_documents epd
				where epd.institution_id = $1
					and epd.portfolio_id in (select id from matched_portfolios)
					and epd.authenticity_status = 'verificat'
			), 0) as verified_documents,
			coalesce(to_char(max(last_updated_on), 'YYYY-MM-DD'), '') as last_updated_on
		from matched_portfolios
	`, institutionID, summary.Personnel.SchoolYear, summary.Personnel.FullName).Scan(
		&summary.Portfolio.MatchedRecords,
		&summary.Portfolio.ValidatedRecords,
		&summary.Portfolio.TotalDocuments,
		&summary.Portfolio.PortfolioScopeDocuments,
		&summary.Portfolio.PersonnelScopeDocuments,
		&summary.Portfolio.VerifiedDocuments,
		&summary.Portfolio.LastUpdatedOn,
	); err != nil {
		httpx.JSON(w, http.StatusInternalServerError, map[string]any{"code": "education_personnel_portfolio_dossier_summary_failed"})
		return
	}

	if err := s.pool.QueryRow(r.Context(), `
		with matched_portfolios as (
			select id
			from education_portfolios
			where institution_id = $1
				and school_year = $2
				and lower(trim(owner_name)) = lower(trim($3))
		)
		select count(*)
		from (
			select distinct lower(trim(epfd.file_reference)) as file_reference
			from education_personnel_file_documents epfd
			join education_portfolio_documents epd
				on lower(trim(epfd.file_reference)) = lower(trim(epd.file_reference))
			where epfd.personnel_id = $4::uuid
				and epfd.institution_id = $1
				and epd.institution_id = $1
				and epd.portfolio_id in (select id from matched_portfolios)
				and trim(coalesce(epfd.file_reference, '')) <> ''
		) mirrored
	`, institutionID, summary.Personnel.SchoolYear, summary.Personnel.FullName, recordID).Scan(&summary.Relation.MirroredFileReferences); err != nil {
		httpx.JSON(w, http.StatusInternalServerError, map[string]any{"code": "education_personnel_portfolio_dossier_summary_failed"})
		return
	}

	summary.Relation.EvaluationResultsEnterPersonnelFile = true
	summary.Relation.AdministrativeDocsEnterPersonnelFile = true
	summary.Relation.InstitutionMayDuplicateOrSeparate = true
	summary.Relation.DuplicationMode = "selectiva"
	summary.Relation.Rules = []string{
		"Documentele administrative rezultate din evaluare se arhiveaza in dosarul personal.",
		"Documentele administrative privind studii, cariera si formare continua raman in dosarul personal.",
		"Portofoliul profesional retine documente de practica, dezvoltare si evidenta profesionala, separat de dosarul personal.",
		"Unitatea poate decide dublarea controlata sau doar referirea documentelor comune intre dosar si portofoliu.",
	}

	blockers := make([]string, 0, 4)
	if summary.Dossier.TotalDocuments == 0 {
		blockers = append(blockers, "dosar_personal_fara_documente")
	}
	if summary.Personnel.HasPortfolio && summary.Portfolio.MatchedRecords == 0 {
		blockers = append(blockers, "portofoliu_lipsa_pentru_cadrul_marcat_cu_portofoliu")
	}
	if summary.Dossier.DocumentsMarkedForPortfolio > 0 && summary.Portfolio.PersonnelScopeDocuments == 0 {
		blockers = append(blockers, "documente_din_dosar_marcate_pentru_portofoliu_dar_neregasite_in_portofoliu")
	}
	if summary.Portfolio.PersonnelScopeDocuments > 0 && summary.Relation.MirroredFileReferences == 0 {
		blockers = append(blockers, "documente_cu_sursa_dosar_personal_fara_referinta_comuna")
	}

	summary.Readiness.Blockers = blockers
	summary.Readiness.ClearDelimitation = len(blockers) == 0

	httpx.JSON(w, http.StatusOK, summary)
}
