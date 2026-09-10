package education

import (
	"net/http"
	"strings"

	authruntime "github.com/eguilde/egueducation/internal/auth"
	"github.com/eguilde/egueducation/internal/httpx"
)

const (
	classesReadPermission         = "education.classes.read"
	classesManagePermission       = "education.classes.manage"
	classesReadAssignedPermission = "education.classes.read_assigned"
)

// requireSchoolClassesAccess keeps handlers compact while making both the
// HTTP layer and RLS use the same manager-vs-assigned-teacher distinction.
func (s *Service) requireSchoolClassesAccess(w http.ResponseWriter, r *http.Request, manage bool) (string, bool, bool) {
	// schoolClassesAccess needs to return semantic errors on the actual writer;
	// the small context bridge avoids duplicating the authorization logic.
	return s.schoolClassesAccessWithWriter(w, r, manage)
}

func (s *Service) schoolClassesAccessWithWriter(w http.ResponseWriter, r *http.Request, manage bool) (assignedUserID string, all bool, ok bool) {
	subject := strings.TrimSpace(authruntime.CurrentSubjectFromRequest(r))
	if subject == "" {
		httpx.JSON(w, http.StatusForbidden, map[string]any{"code": "education_classes_access_denied"})
		return "", false, false
	}
	manager, err := s.authorizeEducationPermissionForSubject(r, subject, EducationDelegationScope{
		PermissionCode: classesManagePermission,
		ResourceType:   "institution",
	})
	if err != nil {
		httpx.JSON(w, http.StatusInternalServerError, map[string]any{"code": "education_classes_permission_check_failed"})
		return "", false, false
	}
	if manager {
		return "", true, true
	}
	if manage {
		httpx.JSON(w, http.StatusForbidden, map[string]any{"code": "education_classes_manage_forbidden"})
		return "", false, false
	}
	reader, err := s.authorizeEducationPermissionForSubject(r, subject, EducationDelegationScope{
		PermissionCode: classesReadPermission,
		ResourceType:   "institution",
	})
	if err != nil {
		httpx.JSON(w, http.StatusInternalServerError, map[string]any{"code": "education_classes_permission_check_failed"})
		return "", false, false
	}
	if reader {
		return "", true, true
	}
	assigned, err := s.authorizeEducationPermissionForSubject(r, subject, EducationDelegationScope{
		PermissionCode: classesReadAssignedPermission,
		ResourceType:   "institution",
	})
	if err != nil {
		httpx.JSON(w, http.StatusInternalServerError, map[string]any{"code": "education_classes_permission_check_failed"})
		return "", false, false
	}
	if !assigned {
		httpx.JSON(w, http.StatusForbidden, map[string]any{"code": "education_classes_read_forbidden"})
		return "", false, false
	}
	actorID, err := s.currentActorUserID(r, subject)
	if err != nil || actorID == "" {
		httpx.JSON(w, http.StatusForbidden, map[string]any{"code": "education_classes_assigned_identity_required"})
		return "", false, false
	}
	return actorID, false, true
}
