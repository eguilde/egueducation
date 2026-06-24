package education

import (
	"errors"
	"net/http"
	"strings"

	"github.com/go-chi/chi/v5"
	"github.com/jackc/pgx/v5"

	"github.com/eguilde/egueducation/internal/httpx"
)

type ManagerialPortfolioSummary struct {
	Dossier struct {
		ID                  string `json:"id"`
		DossierCode         string `json:"dossier_code"`
		DossierType         string `json:"dossier_type"`
		Title               string `json:"title"`
		SchoolYear          string `json:"school_year"`
		Status              string `json:"status"`
		OwnerName           string `json:"owner_name"`
		PublicationRequired bool   `json:"publication_required"`
	} `json:"dossier"`
	Portfolio struct {
		MatchedPersonnel         int      `json:"matched_personnel"`
		ManagerialDocuments      int      `json:"managerial_documents"`
		MandatoryDocuments       int      `json:"mandatory_documents"`
		ApprovedDocuments        int      `json:"approved_documents"`
		PublishedDocuments       int      `json:"published_documents"`
		PublicationRequiredDocs  int      `json:"publication_required_documents"`
		MissingMandatoryCategory []string `json:"missing_mandatory_categories"`
	} `json:"portfolio"`
	Workflow struct {
		TotalSteps         int `json:"total_steps"`
		CompletedSteps     int `json:"completed_steps"`
		OpenSteps          int `json:"open_steps"`
		SignatureSteps     int `json:"signature_steps"`
		CompletedSignSteps int `json:"completed_signature_steps"`
	} `json:"workflow"`
	PersonnelFile struct {
		MatchedDocuments   int `json:"matched_documents"`
		ManagementDocs     int `json:"management_documents"`
		SensitiveDocuments int `json:"sensitive_documents"`
		MirroredReferences int `json:"mirrored_references"`
	} `json:"personnel_file"`
	Readiness struct {
		ReadyForReview      bool     `json:"ready_for_review"`
		ReadyForPublication bool     `json:"ready_for_publication"`
		Blockers            []string `json:"blockers"`
	} `json:"readiness"`
}

func (s *Service) ManagerialPortfolioSummary(w http.ResponseWriter, r *http.Request) {
	recordID := strings.TrimSpace(chi.URLParam(r, "recordID"))
	institutionID := s.institutionID(r)

	type dossierRow struct {
		ID                  string
		DossierCode         string
		DossierType         string
		Title               string
		SchoolYear          string
		Status              string
		OwnerName           string
		PublicationRequired bool
	}

	var dossier dossierRow
	err := s.pool.QueryRow(r.Context(), `
		select
			id::text,
			dossier_code,
			dossier_type,
			title,
			school_year,
			status,
			owner_name,
			publication_required
		from education_managerial_dossiers
		where id = $1::uuid and institution_id = $2
	`, recordID, institutionID).Scan(
		&dossier.ID,
		&dossier.DossierCode,
		&dossier.DossierType,
		&dossier.Title,
		&dossier.SchoolYear,
		&dossier.Status,
		&dossier.OwnerName,
		&dossier.PublicationRequired,
	)
	if errors.Is(err, pgx.ErrNoRows) {
		writeEducationNotFound(w, "education_managerial_record_not_found")
		return
	}
	if err != nil {
		httpx.JSON(w, http.StatusInternalServerError, map[string]any{"code": "education_managerial_portfolio_summary_failed"})
		return
	}

	summary := ManagerialPortfolioSummary{}
	summary.Dossier.ID = dossier.ID
	summary.Dossier.DossierCode = dossier.DossierCode
	summary.Dossier.DossierType = dossier.DossierType
	summary.Dossier.Title = dossier.Title
	summary.Dossier.SchoolYear = dossier.SchoolYear
	summary.Dossier.Status = dossier.Status
	summary.Dossier.OwnerName = dossier.OwnerName
	summary.Dossier.PublicationRequired = dossier.PublicationRequired

	err = s.pool.QueryRow(r.Context(), `
		select
			count(*) as managerial_documents,
			count(*) filter (where mandatory) as mandatory_documents,
			count(*) filter (where document_status in ('approved', 'published')) as approved_documents,
			count(*) filter (where document_status = 'published') as published_documents,
			count(*) filter (where publication_required) as publication_required_documents,
			count(*) filter (where document_category = 'evidenta' and mandatory) as evidenta_docs,
			count(*) filter (where document_category = 'planificare' and mandatory) as planificare_docs
		from education_managerial_documents
		where dossier_id = $1::uuid and institution_id = $2
	`, recordID, institutionID).Scan(
		&summary.Portfolio.ManagerialDocuments,
		&summary.Portfolio.MandatoryDocuments,
		&summary.Portfolio.ApprovedDocuments,
		&summary.Portfolio.PublishedDocuments,
		&summary.Portfolio.PublicationRequiredDocs,
		new(int),
		new(int),
	)
	if err != nil {
		httpx.JSON(w, http.StatusInternalServerError, map[string]any{"code": "education_managerial_portfolio_summary_failed"})
		return
	}

	var evidentaDocs int
	var planificareDocs int
	err = s.pool.QueryRow(r.Context(), `
		select
			count(*) filter (where document_category = 'evidenta' and mandatory) as evidenta_docs,
			count(*) filter (where document_category = 'planificare' and mandatory) as planificare_docs
		from education_managerial_documents
		where dossier_id = $1::uuid and institution_id = $2
	`, recordID, institutionID).Scan(&evidentaDocs, &planificareDocs)
	if err != nil {
		httpx.JSON(w, http.StatusInternalServerError, map[string]any{"code": "education_managerial_portfolio_summary_failed"})
		return
	}
	if evidentaDocs == 0 {
		summary.Portfolio.MissingMandatoryCategory = append(summary.Portfolio.MissingMandatoryCategory, "opis_evidenta")
	}
	if planificareDocs == 0 {
		summary.Portfolio.MissingMandatoryCategory = append(summary.Portfolio.MissingMandatoryCategory, "documente_baza_manageriale")
	}

	err = s.pool.QueryRow(r.Context(), `
		select
			count(*) as total_steps,
			count(*) filter (where status = 'completed') as completed_steps,
			count(*) filter (where status in ('pending', 'in_progress', 'returned')) as open_steps,
			count(*) filter (where requires_signature) as signature_steps,
			count(*) filter (where requires_signature and status = 'completed') as completed_signature_steps
		from education_managerial_workflow_steps
		where dossier_id = $1::uuid and institution_id = $2
	`, recordID, institutionID).Scan(
		&summary.Workflow.TotalSteps,
		&summary.Workflow.CompletedSteps,
		&summary.Workflow.OpenSteps,
		&summary.Workflow.SignatureSteps,
		&summary.Workflow.CompletedSignSteps,
	)
	if err != nil {
		httpx.JSON(w, http.StatusInternalServerError, map[string]any{"code": "education_managerial_portfolio_summary_failed"})
		return
	}

	fileScope := ""
	switch dossier.DossierType {
	case "director_portfolio":
		fileScope = "dosar_director"
	case "adjunct_director_portfolio":
		fileScope = "dosar_director_adjunct"
	}

	if fileScope != "" {
		err = s.pool.QueryRow(r.Context(), `
			with matched_personnel as (
				select id
				from education_personnel
				where institution_id = $1
					and full_name = $2
					and (
						($3 = 'dosar_director' and lower(role_title) = 'director')
						or ($3 = 'dosar_director_adjunct' and lower(role_title) = 'director adjunct')
					)
				limit 1
			)
			select
				(select count(*) from matched_personnel) as matched_personnel,
				count(pfd.*) as matched_documents,
				count(*) filter (where pfd.document_category = 'management') as management_documents,
				count(*) filter (where pfd.sensitive_data) as sensitive_documents,
				count(*) filter (
					where exists (
						select 1
						from education_managerial_documents emd
						where emd.dossier_id = $4::uuid
							and emd.institution_id = $1
							and emd.file_reference <> ''
							and emd.file_reference = pfd.file_reference
					)
				) as mirrored_references
			from matched_personnel mp
			left join education_personnel_file_documents pfd on pfd.personnel_id = mp.id and pfd.institution_id = $1 and pfd.file_scope = $3
		`, institutionID, dossier.OwnerName, fileScope, recordID).Scan(
			&summary.Portfolio.MatchedPersonnel,
			&summary.PersonnelFile.MatchedDocuments,
			&summary.PersonnelFile.ManagementDocs,
			&summary.PersonnelFile.SensitiveDocuments,
			&summary.PersonnelFile.MirroredReferences,
		)
		if err != nil {
			httpx.JSON(w, http.StatusInternalServerError, map[string]any{"code": "education_managerial_portfolio_summary_failed"})
			return
		}
	}

	blockers := make([]string, 0, 6)
	if fileScope != "" && summary.Portfolio.MatchedPersonnel == 0 {
		blockers = append(blockers, "personnel_link_missing")
	}
	if summary.Portfolio.ManagerialDocuments == 0 {
		blockers = append(blockers, "managerial_documents_missing")
	}
	if len(summary.Portfolio.MissingMandatoryCategory) > 0 {
		blockers = append(blockers, "mandatory_categories_missing")
	}
	if summary.Portfolio.MandatoryDocuments > summary.Portfolio.ApprovedDocuments {
		blockers = append(blockers, "mandatory_documents_pending")
	}
	if summary.Workflow.TotalSteps == 0 {
		blockers = append(blockers, "workflow_missing")
	}
	if fileScope != "" && summary.PersonnelFile.MatchedDocuments == 0 {
		blockers = append(blockers, "personnel_file_documents_missing")
	}

	summary.Readiness.Blockers = blockers
	summary.Readiness.ReadyForReview = len(blockers) == 0 || (len(blockers) == 1 && blockers[0] == "mandatory_documents_pending")
	summary.Readiness.ReadyForPublication = dossier.PublicationRequired && summary.Portfolio.PublicationRequiredDocs > 0 && summary.Portfolio.PublishedDocuments > 0 && summary.Workflow.OpenSteps == 0

	httpx.JSON(w, http.StatusOK, summary)
}
