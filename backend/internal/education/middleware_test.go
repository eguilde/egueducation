package education

import "testing"

func TestNormalizeEducationIdentityMatchesDiacriticsAndWhitespace(t *testing.T) {
	if got := normalizedEducationIdentity("  Școala   Nr. 1 "); got != "scoala nr. 1" {
		t.Fatalf("normalized identity = %q", got)
	}
}

func TestContainsAnyNormalizedWordUsesNormalizedRoleNames(t *testing.T) {
	if !containsAnyNormalizedWord("Secretar al Consiliului", "secretar") {
		t.Fatal("expected normalized role hint to match")
	}
	if containsAnyNormalizedWord("Membru", "secretar", "presedinte") {
		t.Fatal("unexpected role hint match")
	}
}

func TestGovernanceActorRuleDefaultsFailClosed(t *testing.T) {
	rule := governanceMeetingActorRule{}
	if rule.AllowMeetingChair || rule.AllowMeetingSecretary || rule.RequireVotingRight || len(rule.MembershipRoleHints) != 0 {
		t.Fatal("empty governance actor rule must not grant access")
	}
}

func TestGovernanceActorAllowedUsesImmutableIDsAndFailsClosedForLegacyRows(t *testing.T) {
	meeting := governanceMeetingAccessContext{
		ChairpersonUserID: "chair-id",
		SecretaryUserID:   "secretary-id",
	}
	memberships := []governanceMembershipAccess{
		{AppUserID: "voter-id", RoleName: "Membru CA", VotingRight: true},
		{AppUserID: "non-voter-id", RoleName: "Membru CA", VotingRight: false},
		{AppUserID: "secretar-member-id", RoleName: "Secretar al Consiliului", VotingRight: false},
		{AppUserID: "", RoleName: "Președinte CA", VotingRight: true}, // legacy display-only record
	}

	tests := []struct {
		name        string
		actorUserID string
		rule        governanceMeetingActorRule
		wantAllowed bool
	}{
		{"chair is allowed for close", "chair-id", governanceMeetingActorRule{AllowMeetingChair: true}, true},
		{"chair is denied when rule excludes chair", "chair-id", governanceMeetingActorRule{RequireVotingRight: true}, false},
		{"secretary is allowed for publication", "secretary-id", governanceMeetingActorRule{AllowMeetingSecretary: true}, true},
		{"voting member is allowed", "voter-id", governanceMeetingActorRule{RequireVotingRight: true}, true},
		{"non-voting member is denied", "non-voter-id", governanceMeetingActorRule{RequireVotingRight: true}, false},
		{"role-qualified member is allowed", "secretar-member-id", governanceMeetingActorRule{MembershipRoleHints: []string{"secretar"}}, true},
		{"role mismatch is denied", "voter-id", governanceMeetingActorRule{MembershipRoleHints: []string{"secretar"}}, false},
		{"legacy name-only membership is denied", "legacy-name-only", governanceMeetingActorRule{RequireVotingRight: true}, false},
		{"empty actor is denied", "", governanceMeetingActorRule{AllowMeetingChair: true}, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := governanceActorAllowed(tt.actorUserID, meeting, memberships, tt.rule); got != tt.wantAllowed {
				t.Fatalf("governanceActorAllowed(%q, %#v) = %t, want %t", tt.actorUserID, tt.rule, got, tt.wantAllowed)
			}
		})
	}
}
