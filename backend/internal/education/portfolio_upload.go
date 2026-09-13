package education

import (
	"context"
	"errors"
	"net/http"
	"strings"

	"github.com/eguilde/egueducation/internal/audit"
	authruntime "github.com/eguilde/egueducation/internal/auth"
	"github.com/eguilde/egueducation/internal/earchiva"
	"github.com/eguilde/egueducation/internal/httpx"
	"github.com/go-chi/chi/v5"
	"github.com/jackc/pgx/v5"
)

// PortfolioOwnArchiveUpload derives every archive authority from the own
// portfolio route. The archive service repeats this callback in its final
// transaction after bytes have been staged and stored.
func (s *Service) PortfolioOwnArchiveUpload(archive *earchiva.DocumentService) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		recordID := strings.TrimSpace(chi.URLParam(r, "recordID"))
		if !s.requireOwnPortfolioContent(w, r, portfolioManageOwnPermission, true) {
			return
		}
		actorID, allowed, err := s.requireOwnPortfolio(r, recordID, portfolioManageOwnPermission)
		if err != nil || !allowed {
			writePortfolioAccessFailure(w, err)
			return
		}
		institutionID := s.institutionID(r)
		actor := strings.TrimSpace(authruntime.CurrentSubjectFromRequest(r))
		scope := earchiva.ScopedUploadScope{InstitutionID: institutionID, ActorSubject: actor, OwnerUserID: actorID, PortfolioID: recordID}
		scope.Authorize = func(ctx context.Context, tx pgx.Tx) error {
			permitted, err := educationSubjectHasPermission(ctx, tx, actor, portfolioManageOwnPermission)
			if err != nil {
				return err
			}
			if !permitted {
				return errors.New("portfolio upload permission revoked")
			}
			var owner string
			err = tx.QueryRow(ctx, `select p.owner_user_id::text from education_portfolios p
				join education_personnel person on person.id=p.owner_personnel_id and person.institution_id=p.institution_id and person.app_user_id=p.owner_user_id
				join app_users u on u.id=p.owner_user_id and u.status='active'
				where p.id=$1::uuid and p.institution_id=$2 and p.owner_user_id=$3::uuid
				and p.status in ('draft','returned') and p.withdrawn_at is null and not p.legal_hold_active
				and exists(select 1 from app_memberships m where m.user_id=u.id
				 and m.tenant_code=public.current_tenant_code() and public.education_membership_is_eligible(u.id,public.current_tenant_code(),$2,m.position_code))
				for update of p`, recordID, institutionID, actorID).Scan(&owner)
			if errors.Is(err, pgx.ErrNoRows) {
				return errors.New("portfolio upload authorization changed")
			}
			return err
		}
		scope.PersistAccess = func(ctx context.Context, tx pgx.Tx, documentID string) error {
			if _, err := tx.Exec(ctx, `insert into education_portfolio_archive_attachment_grants(institution_id,archive_document_id,grantee_user_id,granted_by_user_id) values($1,$2::uuid,$3::uuid,$3::uuid) on conflict(institution_id,archive_document_id,grantee_user_id) do nothing`, institutionID, documentID, actorID); err != nil {
				return err
			}
			return audit.Log(ctx, tx, audit.Event{ActorSubject: actor, Action: "education.portfolios.own_archive_uploaded", TargetType: "archive_document", TargetID: documentID, Summary: "Teacher uploaded portfolio archive evidence"})
		}
		if archive == nil {
			httpx.JSON(w, http.StatusServiceUnavailable, map[string]any{"code": "archive_storage_unavailable"})
			return
		}
		archive.UploadPortfolioOwnedDocument(w, r, scope)
	}
}
