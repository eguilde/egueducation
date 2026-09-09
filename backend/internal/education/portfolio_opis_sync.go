package education

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"strings"

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

	// Keep regeneration as one set-based statement. pgx does not permit Exec on
	// a transaction while a Rows result from the same connection is still open;
	// the former row-by-row implementation therefore failed with "conn busy" as
	// soon as a portfolio contained its first document.
	tag, err := tx.Exec(ctx, `
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
		select
			document.portfolio_id,
			document.section_code,
			document.component_code,
			btrim(document.document_title) || ' (' || to_char(document.issued_on, 'YYYY-MM-DD') || ')',
			document.source_scope,
			row_number() over (
				order by document.chronological_index asc, document.added_on asc,
					document.issued_on asc, document.document_title asc, document.id asc
			)::integer,
			coalesce(
				nullif(btrim(document.file_reference), ''),
				document.section_code || '/' || document.component_code || '/' || document.document_title
			),
			document.source_scope = 'portofoliu',
			document.added_on,
			$3,
			document.institution_id,
			''
		from education_portfolio_documents document
		where document.portfolio_id = $1::uuid and document.institution_id = $2
	`, recordID, institutionID, checkedBy)
	if err != nil {
		return 0, fmt.Errorf("insert portfolio opis entries during sync: %w", err)
	}
	regeneratedEntries := int(tag.RowsAffected())

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
