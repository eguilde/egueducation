package regulatorymodel

import (
	"testing"
	"time"
)

var asOf = time.Date(2026, 9, 11, 0, 0, 0, 0, time.UTC)

func activeProfile(form LegalForm) Profile {
	return Profile{ID: "p", ProfileSeriesID: "p", Version: 1, LegalForm: form, Status: "active", EffectiveFrom: asOf.AddDate(0, 0, -1)}
}

func TestEvaluatePublicPrivateAndConfessionalContexts(t *testing.T) {
	for _, tc := range []struct {
		name       string
		profile    Profile
		capability string
	}{
		{"public", activeProfile(LegalFormPublic), "school.public_controls"},
		{"private", activeProfile(LegalFormPrivate), "school.private_controls"},
		{"confessional overlay", Profile{ID: "p", ProfileSeriesID: "p", Version: 1, LegalForm: LegalFormPrivate, Confessional: &ConfessionalOverlay{Status: "active", EffectiveFrom: asOf.AddDate(0, 0, -1)}, Status: "active", EffectiveFrom: asOf.AddDate(0, 0, -1)}, "school.confessional_overlay"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			e := Evaluate(OperationPolicyContext{OperationCode: "school.profile.read", EffectiveOn: asOf, Profile: tc.profile})
			if !e.Allowed || !has(e.Capabilities, tc.capability) {
				t.Fatalf("unexpected evaluation: %#v", e)
			}
		})
	}
}
func TestOfferingLocationNeedsAuthorization(t *testing.T) {
	c := OperationPolicyContext{OperationCode: "school.enrollment.create", EffectiveOn: asOf, Profile: activeProfile(LegalFormPrivate), OfferingID: "o", LocationID: "l"}
	if Evaluate(c).Allowed {
		t.Fatal("unauthorized offering/location admitted")
	}
	c.Authorizations = []Authorization{{OfferingID: "o", LocationID: "l", Status: "accredited", EffectiveFrom: asOf.AddDate(0, 0, -1)}}
	if !Evaluate(c).Allowed {
		t.Fatalf("authorized offering/location blocked: %#v", Evaluate(c))
	}
}

func TestAmbiguousLegacyAuthorizedStatusFailsClosed(t *testing.T) {
	c := OperationPolicyContext{
		OperationCode:  "school.admission.decision.issue",
		EffectiveOn:    asOf,
		Profile:        activeProfile(LegalFormPublic),
		OfferingID:     "o",
		LocationID:     "l",
		Authorizations: []Authorization{{OfferingID: "o", LocationID: "l", Status: "authorized", EffectiveFrom: asOf.AddDate(0, 0, -1)}},
	}
	result := Evaluate(c)
	if result.Allowed {
		t.Fatalf("ambiguous legacy authorized status must fail closed: %#v", result)
	}
}
func TestPublicFundingDoesNotAutoAssumeProcurement(t *testing.T) {
	c := OperationPolicyContext{OperationCode: "school.purchase.create", EffectiveOn: asOf, Profile: activeProfile(LegalFormPrivate), Funding: &FundingInstrument{ID: "fund", Public: true, EligibilityStatus: "eligible", EffectiveFrom: asOf.AddDate(0, 0, -1)}, Procurement: &ProcurementAssessment{FundingInstrumentID: "fund", Determination: "not_applicable", EffectiveFrom: asOf.AddDate(0, 0, -1)}}
	e := Evaluate(c)
	if !e.Allowed || has(e.Capabilities, "school.procurement") || !has(e.Obligations, "funding.public.controls") {
		t.Fatalf("funding incorrectly inferred procurement: %#v", e)
	}
	c.Procurement = &ProcurementAssessment{FundingInstrumentID: "fund", Determination: "applicable", EffectiveFrom: asOf.AddDate(0, 0, -1)}
	e = Evaluate(c)
	if !has(e.Capabilities, "school.procurement") {
		t.Fatalf("express assessment not honored: %#v", e)
	}
}
func TestMismatchedProcurementFundingFailsClosed(t *testing.T) {
	e := Evaluate(OperationPolicyContext{OperationCode: "school.purchase.create", EffectiveOn: asOf, Profile: activeProfile(LegalFormPublic), Funding: &FundingInstrument{ID: "a", EligibilityStatus: "eligible", EffectiveFrom: asOf.AddDate(0, 0, -1)}, Procurement: &ProcurementAssessment{FundingInstrumentID: "b", Determination: "applicable", EffectiveFrom: asOf.AddDate(0, 0, -1)}})
	if e.Allowed {
		t.Fatal("cross-context procurement assessment accepted")
	}
}
func TestMissingAsOfAndRequiredOperationContextFailClosed(t *testing.T) {
	if Evaluate(OperationPolicyContext{OperationCode: "school.profile.read", Profile: activeProfile(LegalFormPublic)}).Allowed {
		t.Fatal("missing effective date was accepted")
	}
	if Evaluate(OperationPolicyContext{OperationCode: "school.enrollment.create", EffectiveOn: asOf, Profile: activeProfile(LegalFormPublic)}).Allowed {
		t.Fatal("enrollment without offering and location was accepted")
	}
	if Evaluate(OperationPolicyContext{OperationCode: "school.purchase.create", EffectiveOn: asOf, Profile: activeProfile(LegalFormPublic)}).Allowed {
		t.Fatal("purchase without procurement assessment was accepted")
	}
}
func TestHistoricalProfileRetainedInContext(t *testing.T) {
	c := OperationPolicyContext{OperationCode: "school.contract.create", EffectiveOn: asOf, Profile: Profile{ID: "profile-v1", ProfileSeriesID: "profile-v1", Version: 1, LegalForm: LegalFormPublic, Status: "active", EffectiveFrom: asOf.AddDate(0, 0, -1)}}
	if e := Evaluate(c); !e.Allowed || c.Profile.Version != 1 || c.Profile.ID != "profile-v1" {
		t.Fatalf("historical profile snapshot altered: %#v %#v", c, e)
	}
}
func TestProfileLineageIsDeterministic(t *testing.T) {
	root := NewProfileRoot("root")
	root.Version = 1
	next, err := NextProfileRevision(root, "revision-2")
	if err != nil || next.ProfileSeriesID != "root" || next.Version != 2 {
		t.Fatalf("invalid lineage: %#v %v", next, err)
	}
}
func has(values []string, target string) bool {
	for _, v := range values {
		if v == target {
			return true
		}
	}
	return false
}
