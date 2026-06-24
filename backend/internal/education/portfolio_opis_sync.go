package education

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"strings"
	"time"

	authruntime "github.com/eguilde/egueducation/internal/auth"
	"github.com/eguilde/egueducation/internal/httpx"
	"github.com/go-chi/chi/v5"
)

var errPortfolioRecordNotFound = errors.New("education portfolio record not found")

func (s *Service) RegeneratePortfolioOpis(w http.ResponseWriter, r *http.Request) {
	recordID := strings.TrimSpace(chi.URLParam(r, "recordID"))
	if recordID == "" {
		httpx.JSON(w, http.StatusBadRequest, map[string]any{"code": "portfolio_id_required"})
		return
	}

	regeneratedEntries, checkedBy, err := s.syncPortfolioOpis(r.Context(), r, recordID, s.institutionID(r))
	if err != nil {
		switch {
		case errors.Is(err, errPortfolioRecordNotFound):
			writeEducationNotFound(w, "education_portfolio_not_found")
		default:
			httpx.JSON(w, http.StatusInternalServerError, map[string]any{"code": "portfolio_opis_regenerate_failed"})
		}
		return
	}

	s.logAudit(r, "education.portfolios.opis.regenerate", "portfolio_record", recordID, "Portfolio opis regenerated from portfolio documents.", map[string]any{
		"portfolio_id":         recordID,
		"regenerated_entries":  regeneratedEntries,
		"checked_by":           checkedBy,
		"synchronization_mode": "documents_to_opis",
	})

	httpx.JSON(w, http.StatusOK, map[string]any{
		"status":              "ok",
		"portfolio_id":        recordID,
		"regenerated_entries": regeneratedEntries,
		"checked_by":          checkedBy,
	})
}

func (s *Service) syncPortfolioOpis(ctx context.Context, r *http.Request, recordID string, institutionID string) (int, string, error) {
	checkedBy, err := s.portfolioOpisCheckedBy(r)
	if err != nil {
		return 0, "", err
	}

	regeneratedEntries, err := s.rebuildPortfolioOpis(ctx, recordID, institutionID, checkedBy)
	if err != nil {
		return 0, checkedBy, err
	}

	return regeneratedEntries, checkedBy, nil
}

func (s *Service) rebuildPortfolioOpis(ctx context.Context, recordID string, institutionID string, checkedBy string) (int, error) {
	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return 0, fmt.Errorf("begin portfolio opis sync: %w", err)
	}
	defer func() {
		_ = tx.Rollback(ctx)
	}()

	var exists bool
	if err := tx.QueryRow(ctx, `
		select exists(
			select 1
			from education_portfolios
			where id = $1::uuid and institution_id = $2
		)
	`, recordID, institutionID).Scan(&exists); err != nil {
		return 0, fmt.Errorf("check portfolio exists for opis sync: %w", err)
	}
	if !exists {
		return 0, errPortfolioRecordNotFound
	}

	if _, err := tx.Exec(ctx, `
		delete from education_portfolio_opis
		where portfolio_id = $1::uuid and institution_id = $2
	`, recordID, institutionID); err != nil {
		return 0, fmt.Errorf("clear portfolio opis before sync: %w", err)
	}

	rows, err := tx.Query(ctx, `
		select
			section_code,
			component_code,
			document_title,
			source_scope,
			coalesce(file_reference, ''),
			to_char(issued_on, 'YYYY-MM-DD'),
			to_char(added_on, 'YYYY-MM-DD')
		from education_portfolio_documents
		where portfolio_id = $1::uuid and institution_id = $2
		order by chronological_index asc, added_on asc, issued_on asc, document_title asc, id asc
	`, recordID, institutionID)
	if err != nil {
		return 0, fmt.Errorf("load portfolio documents for opis sync: %w", err)
	}
	defer rows.Close()

	regeneratedEntries := 0
	for rows.Next() {
		var sectionCode string
		var componentCode string
		var documentTitle string
		var sourceScope string
		var fileReference string
		var issuedOn string
		var addedOn string
		if err := rows.Scan(
			&sectionCode,
			&componentCode,
			&documentTitle,
			&sourceScope,
			&fileReference,
			&issuedOn,
			&addedOn,
		); err != nil {
			return 0, fmt.Errorf("scan portfolio document for opis sync: %w", err)
		}

		regeneratedEntries++
		documentReference := strings.TrimSpace(fileReference)
		if documentReference == "" {
			documentReference = fmt.Sprintf("%s/%s/%s", sectionCode, componentCode, documentTitle)
		}

		checkedOn := strings.TrimSpace(addedOn)
		if checkedOn == "" {
			checkedOn = time.Now().Format("2006-01-02")
		}

		entryTitle := strings.TrimSpace(documentTitle)
		if issuedOn != "" {
			entryTitle = fmt.Sprintf("%s (%s)", entryTitle, issuedOn)
		}

		if _, err := tx.Exec(ctx, `
			insert into education_portfolio_opis (
				portfolio_id,
				section_code,
				component_code,
				entry_title,
				source_scope,
				chronological_index,
				document_reference,
				included_in_transfer,
				checked_on,
				checked_by,
				institution_id,
				notes
			)
			values ($1::uuid, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, '')
		`, recordID, sectionCode, componentCode, entryTitle, sourceScope, regeneratedEntries, documentReference, sourceScope == "portofoliu", checkedOn, checkedBy, institutionID); err != nil {
			return 0, fmt.Errorf("insert portfolio opis entry during sync: %w", err)
		}
	}
	if err := rows.Err(); err != nil {
		return 0, fmt.Errorf("iterate portfolio documents for opis sync: %w", err)
	}

	if err := tx.Commit(ctx); err != nil {
		return 0, fmt.Errorf("commit portfolio opis sync: %w", err)
	}

	return regeneratedEntries, nil
}

func (s *Service) portfolioOpisCheckedBy(r *http.Request) (string, error) {
	subject := strings.TrimSpace(authruntime.CurrentSubjectFromRequest(r))
	if subject == "" {
		return "", nil
	}

	actorName, err := s.currentActorName(r, subject)
	if err != nil {
		return "", fmt.Errorf("resolve actor name for portfolio opis sync: %w", err)
	}
	if actorName != "" {
		return strings.TrimSpace(actorName), nil
	}

	return subject, nil
}
