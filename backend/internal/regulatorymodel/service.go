// Package regulatorymodel owns the pure Stage 1B regulatory decision contract.
// Persistence and transport remain separate so every caller supplies a host-bound
// tenant/institution context and stores the returned immutable snapshot.
package regulatorymodel

import (
	"fmt"
	"sort"
	"strings"
	"time"
)

type LegalForm string

const (
	LegalFormPublic  LegalForm = "public"
	LegalFormPrivate LegalForm = "private"
)

// ConfessionalOverlay is derived by the persistence adapter from the active
// typed school_confessional_profiles/protocols aggregate. It is never accepted
// from an HTTP command and it is not a third legal form.
type ConfessionalOverlay struct {
	Status                     string
	EffectiveFrom, EffectiveTo time.Time
}

type Profile struct {
	ID                         string
	ProfileSeriesID            string
	Version                    int
	LegalForm                  LegalForm
	Confessional               *ConfessionalOverlay
	Status                     string
	EffectiveFrom, EffectiveTo time.Time
}

// NewProfileRoot and NextProfileRevision define deterministic lineage for the
// persistence adapter: a root owns its series id; every revision inherits it.
func NewProfileRoot(id string) Profile { return Profile{ID: id, ProfileSeriesID: id} }
func NextProfileRevision(current Profile, nextID string) (Profile, error) {
	if strings.TrimSpace(current.ProfileSeriesID) == "" || strings.TrimSpace(nextID) == "" {
		return Profile{}, fmt.Errorf("profile lineage identifiers are required")
	}
	return Profile{ID: nextID, ProfileSeriesID: current.ProfileSeriesID, Version: current.Version + 1, LegalForm: current.LegalForm, Confessional: current.Confessional, Status: "draft"}, nil
}

type Authorization struct {
	OfferingID, LocationID, Status string
	EffectiveFrom, EffectiveTo     time.Time
}
type FundingInstrument struct {
	ID                         string
	Public                     bool
	EligibilityStatus          string
	EffectiveFrom, EffectiveTo time.Time
}
type ProcurementAssessment struct {
	FundingInstrumentID        string
	Determination              string
	EffectiveFrom, EffectiveTo time.Time
}
type OperationPolicyContext struct {
	OperationCode          string
	EffectiveOn            time.Time
	Profile                Profile
	OfferingID, LocationID string
	Authorizations         []Authorization
	Funding                *FundingInstrument
	Procurement            *ProcurementAssessment
}
type Evaluation struct {
	Allowed                             bool
	Capabilities, Obligations, Warnings []string
}

// Evaluate is deliberately conservative. Funding creates public-finance
// obligations but never infers procurement applicability; that decision needs a
// separately effective assessment with its own legal source and audit trail.
func Evaluate(c OperationPolicyContext) Evaluation {
	e := Evaluation{}
	if strings.TrimSpace(c.OperationCode) == "" {
		return denied("operation_code is required")
	}
	at := c.EffectiveOn
	if at.IsZero() {
		return denied("effective_on is required")
	}
	if strings.TrimSpace(c.Profile.ID) == "" || strings.TrimSpace(c.Profile.ProfileSeriesID) == "" || c.Profile.Version < 1 || !effective(c.Profile.EffectiveFrom, c.Profile.EffectiveTo, at) || (c.Profile.Status != "approved" && c.Profile.Status != "active") || (c.Profile.LegalForm != LegalFormPublic && c.Profile.LegalForm != LegalFormPrivate) {
		return denied("effective public/private profile is required")
	}
	e.Capabilities = append(e.Capabilities, "school.core")
	if c.Profile.LegalForm == LegalFormPublic {
		e.Capabilities = append(e.Capabilities, "school.public_controls")
	} else {
		e.Capabilities = append(e.Capabilities, "school.private_controls")
	}
	if c.Profile.Confessional != nil {
		if c.Profile.LegalForm != LegalFormPrivate || (c.Profile.Confessional.Status != "approved" && c.Profile.Confessional.Status != "active") || !effective(c.Profile.Confessional.EffectiveFrom, c.Profile.Confessional.EffectiveTo, at) {
			return denied("effective private confessional overlay is required")
		}
		e.Capabilities = append(e.Capabilities, "school.confessional_overlay")
	}
	requiresOffering := strings.HasPrefix(c.OperationCode, "school.enrollment.") || strings.HasPrefix(c.OperationCode, "school.admission.") || strings.HasPrefix(c.OperationCode, "school.education_contract.")
	if requiresOffering && (strings.TrimSpace(c.OfferingID) == "" || strings.TrimSpace(c.LocationID) == "") {
		return denied("offering and location are required for this operation")
	}
	if c.OfferingID != "" || c.LocationID != "" {
		if !effectiveAuthorization(c.Authorizations, c.OfferingID, c.LocationID, at) {
			return denied("effective offering/location authorization is required")
		}
		e.Capabilities = append(e.Capabilities, "school.offering.authorized")
	}
	if c.Funding != nil {
		if !effective(c.Funding.EffectiveFrom, c.Funding.EffectiveTo, at) || c.Funding.EligibilityStatus != "eligible" {
			return denied("funding instrument is not effectively eligible")
		}
		if c.Funding.Public {
			e.Obligations = append(e.Obligations, "funding.public.controls")
			e.Capabilities = append(e.Capabilities, "school.public_funding")
		}
	}
	requiresProcurementAssessment := strings.HasPrefix(c.OperationCode, "school.purchase.") || strings.HasPrefix(c.OperationCode, "school.procurement.")
	if requiresProcurementAssessment && c.Procurement == nil {
		return denied("procurement applicability assessment is required")
	}
	if c.Procurement != nil {
		if !effective(c.Procurement.EffectiveFrom, c.Procurement.EffectiveTo, at) || c.Procurement.Determination == "" || c.Procurement.Determination == "indeterminate" {
			return denied("procurement applicability assessment is not effective")
		}
		if c.Funding != nil && c.Procurement.FundingInstrumentID != "" && c.Procurement.FundingInstrumentID != c.Funding.ID {
			return denied("procurement assessment is bound to another funding instrument")
		}
		if c.Procurement.Determination == "applicable" {
			e.Obligations = append(e.Obligations, "procurement.assessment.applies")
			e.Capabilities = append(e.Capabilities, "school.procurement")
		}
	}
	e.Allowed = true
	return normalize(e)
}
func effectiveAuthorization(all []Authorization, offering, location string, at time.Time) bool {
	for _, a := range all {
		// Romanian pre-university law defines provisional authorization and
		// accreditation as the two positive legal states for enrolment.  The
		// legacy value "authorized" is ambiguous (it is not a separate legal
		// stage), so it must be classified from its source act before it can
		// authorize a regulated operation.
		if a.OfferingID == offering && a.LocationID == location && effective(a.EffectiveFrom, a.EffectiveTo, at) && (a.Status == "provisional" || a.Status == "accredited") {
			return true
		}
	}
	return false
}
func effective(from, to, at time.Time) bool {
	return !from.IsZero() && !from.After(at) && (to.IsZero() || !to.Before(at))
}
func denied(w string) Evaluation { return Evaluation{Warnings: []string{w}} }
func normalize(e Evaluation) Evaluation {
	e.Capabilities = unique(e.Capabilities)
	e.Obligations = unique(e.Obligations)
	e.Warnings = unique(e.Warnings)
	return e
}
func unique(in []string) []string {
	s := map[string]struct{}{}
	for _, v := range in {
		if v = strings.TrimSpace(v); v != "" {
			s[v] = struct{}{}
		}
	}
	out := make([]string, 0, len(s))
	for v := range s {
		out = append(out, v)
	}
	sort.Strings(out)
	return out
}
func (c OperationPolicyContext) Validate() error {
	if strings.TrimSpace(c.OperationCode) == "" {
		return fmt.Errorf("operation_code is required")
	}
	return nil
}
