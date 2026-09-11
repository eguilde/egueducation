package institution

import "testing"

func TestResolveCapabilitiesIntersectsPolicyModuleAndPermission(t *testing.T) {
	packs := []assignedPolicyPack{{
		AssignmentKind: "legal_form",
		Rules: storedPolicyRuleSet{Capabilities: []storedPolicyCapability{{
			Code: "operations.public_controls", Enabled: true, Reason: "public",
			RequiredDocuments: []string{"referat"}, RequiredApprovals: []string{"ordonator"},
		}}},
	}}

	tests := []struct {
		name        string
		permissions []string
		modules     []string
		enabled     bool
	}{
		{name: "all intersections", permissions: []string{"education.governance.manage"}, modules: []string{"education"}, enabled: true},
		{name: "missing permission", modules: []string{"education"}, enabled: false},
		{name: "missing module", permissions: []string{"education.governance.manage"}, enabled: false},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			result := resolveCapabilities(packs, test.permissions, test.modules)
			if len(result) != 1 || result[0].Enabled != test.enabled {
				t.Fatalf("unexpected capabilities: %#v", result)
			}
			if result[0].RequiredPermission != "education.governance.manage" {
				t.Fatalf("missing explicit permission contract: %#v", result[0])
			}
		})
	}
}

func TestResolveCapabilitiesPublicFundingOverlayCannotReduceObligation(t *testing.T) {
	packs := []assignedPolicyPack{
		{AssignmentKind: "legal_form", Rules: storedPolicyRuleSet{Capabilities: []storedPolicyCapability{{Code: "operations.public_controls", Enabled: false, Reason: "private"}}}},
		{AssignmentKind: "funding", Rules: storedPolicyRuleSet{Capabilities: []storedPolicyCapability{{Code: "operations.public_controls", Enabled: true, Reason: "public funds", RequiredDocuments: []string{"sursa", "referat"}, RequiredApprovals: []string{"cfp"}, WizardSteps: []string{"control"}}}}},
	}
	result := resolveCapabilities(packs, []string{"education.governance.manage"}, []string{"education"})
	if len(result) != 1 || !result[0].Enabled || result[0].Reason != "public funds" {
		t.Fatalf("public funding overlay was not mandatory: %#v", result)
	}
	if len(result[0].RequiredDocuments) != 2 || len(result[0].RequiredApprovals) != 1 || len(result[0].WizardSteps) != 1 {
		t.Fatalf("overlay requirements were not retained: %#v", result[0])
	}
}

func TestResolveCapabilitiesUnknownCodeFailsClosed(t *testing.T) {
	packs := []assignedPolicyPack{{Rules: storedPolicyRuleSet{Capabilities: []storedPolicyCapability{{Code: "unknown.money.transfer", Enabled: true}}}}}
	result := resolveCapabilities(packs, []string{"*"}, []string{"education"})
	if len(result) != 1 || result[0].Enabled {
		t.Fatalf("unknown policy capability must fail closed: %#v", result)
	}
}

func TestResolvePublicationCapabilityForPublicAndPrivateProfiles(t *testing.T) {
	common := assignedPolicyPack{AssignmentKind: "common", Rules: storedPolicyRuleSet{Capabilities: []storedPolicyCapability{{Code: "education.publication.manage", Enabled: true, WizardSteps: []string{"anonimizare"}}}}}
	for _, test := range []struct {
		name     string
		approval string
	}{
		{name: "public", approval: "director"},
		{name: "private", approval: "reprezentant_legal"},
		{name: "confessional", approval: "fondator"},
	} {
		t.Run(test.name, func(t *testing.T) {
			overlay := assignedPolicyPack{AssignmentKind: "legal_form", Rules: storedPolicyRuleSet{Capabilities: []storedPolicyCapability{{Code: "education.publication.manage", Enabled: true, RequiredApprovals: []string{test.approval}}}}}
			result := resolveCapabilities([]assignedPolicyPack{common, overlay}, []string{"education.compliance.manage"}, []string{"education"})
			if len(result) != 1 || !result[0].Enabled || len(result[0].RequiredApprovals) != 1 || result[0].RequiredApprovals[0] != test.approval {
				t.Fatalf("unexpected %s publication policy: %#v", test.name, result)
			}
		})
	}
}

func TestValidateProfileInput(t *testing.T) {
	valid := PutRegulatoryProfileRequest{
		ExpectedVersion: 1, Status: "approved", SchoolLegalForm: "private",
		RegulatoryProfile: "ro.private.preuniversity", AuthorizationStatus: "accredited",
		EffectiveFrom: "2026-09-11", SourceReference: "decision-1",
	}
	if code := validateProfileInput(&valid); code != "" {
		t.Fatalf("valid profile rejected: %s", code)
	}

	invalidForm := valid
	invalidForm.SchoolLegalForm = "public-from-client-tenant"
	if code := validateProfileInput(&invalidForm); code != "invalid_school_legal_form" {
		t.Fatalf("expected legal form validation, got %s", code)
	}

	invalidWindow := valid
	to := "2026-01-01"
	invalidWindow.EffectiveTo = &to
	if code := validateProfileInput(&invalidWindow); code != "invalid_effective_window" {
		t.Fatalf("expected effective window validation, got %s", code)
	}
}
