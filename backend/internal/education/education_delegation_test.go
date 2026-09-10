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

func TestActiveDelegationGrantRevisionIsStableAndOrderSensitive(t *testing.T) {
	grants := []EducationActiveDelegationGrant{
		{PermissionCode: "education.portfolios.transfer", ResourceType: "institution", ResourceID: ""},
		{PermissionCode: "education.governance.read", ResourceType: "meeting", ResourceID: "11111111-1111-1111-1111-111111111111"},
	}
	first := activeDelegationGrantRevision(grants)
	if len(first) != 64 {
		t.Fatalf("revision must be a SHA-256 hex digest, got %q", first)
	}
	if got := activeDelegationGrantRevision(append([]EducationActiveDelegationGrant(nil), grants...)); got != first {
		t.Fatalf("same ordered grants must produce same revision: got %q want %q", got, first)
	}
	if got := activeDelegationGrantRevision([]EducationActiveDelegationGrant{grants[1], grants[0]}); got == first {
		t.Fatal("ordered grant serialization must not treat reordered grants as the same snapshot")
	}
}

func TestDelegableEducationPermissionExcludesManagement(t *testing.T) {
	if !isDelegableEducationPermission("education.portfolios.transfer") {
		t.Fatal("operational education permission should be delegable")
	}
	if isDelegableEducationPermission("education.delegations.offer") {
		t.Fatal("delegation management permission must never be delegable")
	}
	if isDelegableEducationPermission("registratura.read") {
		t.Fatal("non-Education permission must not be delegable through School grants")
	}
}
