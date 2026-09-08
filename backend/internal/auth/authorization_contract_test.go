package auth

import "testing"

func TestTokenAuthorizationMatchesSession(t *testing.T) {
	session := SessionContext{
		User:          SessionUser{Roles: []string{"admin", "workflow_admin"}},
		TenantCode:    "tenant-balotesti",
		InstitutionID: "inst-balotesti",
		PlatformRoles: []string{"platform_super_admin"},
		AuthzVersion:  7,
		Permissions:   []string{"admin.read", "registratura.manage"},
	}
	valid := &AccessTokenClaims{
		TenantID:      "tenant-balotesti",
		TenantCode:    "tenant-balotesti",
		InstitutionID: "inst-balotesti",
		JWTID:         "token-id",
		SID:           "session-id",
		ACR:           "urn:eguilde:acr:sms-otp",
		AMR:           []string{"otp"},
		Roles:         []string{"workflow_admin", "admin"},
		PlatformRoles: []string{"platform_super_admin"},
		AuthzVersion:  7,
		Permissions:   []string{"registratura.manage", "admin.read"},
	}
	if !tokenAuthorizationMatchesSession(valid, session, "tenant-balotesti") {
		t.Fatal("equivalent token and database authorization must match regardless of ordering")
	}

	tests := []struct {
		name   string
		mutate func(*AccessTokenClaims)
	}{
		{name: "missing tenant code", mutate: func(value *AccessTokenClaims) { value.TenantCode = "" }},
		{name: "wrong tenant", mutate: func(value *AccessTokenClaims) { value.TenantID = "tenant-other" }},
		{name: "wrong institution", mutate: func(value *AccessTokenClaims) { value.InstitutionID = "inst-other" }},
		{name: "stale version", mutate: func(value *AccessTokenClaims) { value.AuthzVersion = 6 }},
		{name: "missing token id", mutate: func(value *AccessTokenClaims) { value.JWTID = "" }},
		{name: "missing session id", mutate: func(value *AccessTokenClaims) { value.SID = "" }},
		{name: "stale roles", mutate: func(value *AccessTokenClaims) { value.Roles = []string{"admin"} }},
		{name: "stale platform roles", mutate: func(value *AccessTokenClaims) { value.PlatformRoles = nil }},
		{name: "stale permissions", mutate: func(value *AccessTokenClaims) { value.Permissions = []string{"admin.read"} }},
		{name: "duplicate claim", mutate: func(value *AccessTokenClaims) { value.Roles = []string{"admin", "admin"} }},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			copy := *valid
			copy.Roles = append([]string(nil), valid.Roles...)
			copy.PlatformRoles = append([]string(nil), valid.PlatformRoles...)
			copy.Permissions = append([]string(nil), valid.Permissions...)
			test.mutate(&copy)
			if tokenAuthorizationMatchesSession(&copy, session, "tenant-balotesti") {
				t.Fatal("mismatched or stale authorization must fail closed")
			}
		})
	}
}
