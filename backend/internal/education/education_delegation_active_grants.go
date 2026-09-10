package education

import (
	"crypto/sha256"
	"encoding/hex"
	"net/http"
	"strings"
	"time"

	authruntime "github.com/eguilde/egueducation/internal/auth"
	"github.com/eguilde/egueducation/internal/httpx"
)

// EducationActiveDelegationGrant is a currently usable, server-revalidated
// delegated authority. It deliberately contains no user identifiers other
// than the active caller's implicit identity, and no delegation management
// permission can be represented here.
type EducationActiveDelegationGrant struct {
	PermissionCode string `json:"permission_code"`
	ResourceType   string `json:"resource_type"`
	ResourceID     string `json:"resource_id"`
}

// EducationActiveDelegationGrantsResponse is intentionally a closed snapshot:
// tenant and institution are request-derived, evaluated_at records the server
// decision time, and revision is stable for an unchanged ordered grant set.
type EducationActiveDelegationGrantsResponse struct {
	TenantCode    string                           `json:"tenant_code"`
	InstitutionID string                           `json:"institution_id"`
	EvaluatedAt   string                           `json:"evaluated_at"`
	Revision      string                           `json:"revision"`
	Grants        []EducationActiveDelegationGrant `json:"grants"`
}

func activeDelegationGrantRevision(grants []EducationActiveDelegationGrant) string {
	hash := sha256.New()
	for _, grant := range grants {
		// A NUL separator makes the serialization unambiguous without exposing
		// any non-canonical database details in the public revision.
		hash.Write([]byte(grant.PermissionCode))
		hash.Write([]byte{0})
		hash.Write([]byte(grant.ResourceType))
		hash.Write([]byte{0})
		hash.Write([]byte(grant.ResourceID))
		hash.Write([]byte{0})
	}
	return hex.EncodeToString(hash.Sum(nil))
}

// ActiveEducationDelegationGrants returns the authenticated actor's accepted,
// currently valid grants. It requires authentication and institution context
// through the enclosing router group, but intentionally does not require a
// direct education.delegations.* permission: a delegate must be able to learn
// which operational permissions are active even after the offer is accepted.
func (s *Service) ActiveEducationDelegationGrants(w http.ResponseWriter, r *http.Request) {
	subject := strings.TrimSpace(authruntime.CurrentSubjectFromRequest(r))
	actorID, err := s.currentActorUserID(r, subject)
	if err != nil {
		httpx.JSON(w, http.StatusInternalServerError, map[string]any{"code": "education_delegation_actor_lookup_failed"})
		return
	}
	if actorID == "" {
		httpx.JSON(w, http.StatusForbidden, map[string]any{"code": "education_delegation_actor_required"})
		return
	}

	scopes, err := s.activeEducationDelegationGrantsForActor(r, actorID)
	if err != nil {
		httpx.JSON(w, http.StatusInternalServerError, map[string]any{"code": "education_active_delegation_grants_failed"})
		return
	}
	grants := make([]EducationActiveDelegationGrant, 0, len(scopes))
	for _, scope := range scopes {
		grants = append(grants, EducationActiveDelegationGrant{
			PermissionCode: scope.PermissionCode,
			ResourceType:   scope.ResourceType,
			ResourceID:     scope.ResourceID,
		})
	}

	// This is identity-specific authorization state, never cacheable by a
	// shared intermediary. Vary makes the auth boundary explicit for clients
	// that retain response metadata.
	w.Header().Set("Cache-Control", "no-store")
	w.Header().Set("Vary", "Authorization")
	httpx.JSON(w, http.StatusOK, EducationActiveDelegationGrantsResponse{
		TenantCode:    strings.TrimSpace(authruntime.CurrentTenantCodeFromRequest(r)),
		InstitutionID: s.institutionID(r),
		EvaluatedAt:   time.Now().UTC().Format(time.RFC3339Nano),
		Revision:      activeDelegationGrantRevision(grants),
		Grants:        grants,
	})
}
