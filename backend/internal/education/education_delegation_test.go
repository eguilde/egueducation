package education

import (
	"testing"
	"time"
)

func TestEducationDelegationScopeMatches(t *testing.T) {
	permission := "education.portfolios.transfer"
	resourceID := "9d1b623a-f307-4477-a4e6-cd351f30e597"
	for _, tc := range []struct {
		name      string
		granted   EducationDelegationScope
		requested EducationDelegationScope
		want      bool
	}{
		{"institution grant applies inside institution", EducationDelegationScope{PermissionCode: permission, ResourceType: "institution"}, EducationDelegationScope{PermissionCode: permission, ResourceType: "portfolio", ResourceID: resourceID}, true},
		{"resource grant is exact", EducationDelegationScope{PermissionCode: permission, ResourceType: "portfolio", ResourceID: resourceID}, EducationDelegationScope{PermissionCode: permission, ResourceType: "portfolio", ResourceID: resourceID}, true},
		{"resource grant cannot broaden", EducationDelegationScope{PermissionCode: permission, ResourceType: "portfolio", ResourceID: resourceID}, EducationDelegationScope{PermissionCode: permission, ResourceType: "meeting", ResourceID: resourceID}, false},
		{"different permission denied", EducationDelegationScope{PermissionCode: permission, ResourceType: "institution"}, EducationDelegationScope{PermissionCode: "education.portfolios.verify", ResourceType: "institution"}, false},
	} {
		t.Run(tc.name, func(t *testing.T) {
			if got := educationDelegationScopeMatches(tc.granted, tc.requested); got != tc.want {
				t.Fatalf("educationDelegationScopeMatches()=%t, want %t", got, tc.want)
			}
		})
	}
}

func TestDelegationActiveAtHonorsDates(t *testing.T) {
	now := time.Date(2026, 9, 10, 12, 0, 0, 0, time.UTC)
	past := now.AddDate(0, 0, -1)
	future := now.AddDate(0, 0, 1)
	if !delegationActiveAt("accepted", past, nil, now) {
		t.Fatal("accepted unlimited delegation must be active")
	}
	if delegationActiveAt("accepted", past, &past, now) {
		t.Fatal("expired accepted delegation must not authorize")
	}
	if delegationActiveAt("accepted", future, nil, now) {
		t.Fatal("not-yet-valid delegation must not authorize")
	}
	if delegationActiveAt("revoked", past, nil, now) {
		t.Fatal("revoked delegation must not authorize")
	}
}
