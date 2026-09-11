package institution

import (
	"sort"
	"strings"
)

var capabilityAuthorization = map[string]struct {
	Module     string
	Permission string
}{
	"education.core":                     {Module: "education", Permission: "education.read"},
	"education.publication.manage":       {Module: "education", Permission: "education.compliance.manage"},
	"operations.public_controls":         {Module: "education", Permission: "education.governance.manage"},
	"operations.private_controls":        {Module: "education", Permission: "education.governance.manage"},
	"operations.confessional_governance": {Module: "education", Permission: "education.governance.manage"},
}

// resolveCapabilities is deterministic: packs are composed in the caller's
// stable order and duplicate requirements are unioned. An enabling overlay
// (for example public funding over a private profile) cannot be cancelled by
// another pack that merely disables the same capability.
func resolveCapabilities(packs []assignedPolicyPack, permissions []string, activeModules []string) []PolicyCapability {
	permissionSet := stringSet(permissions)
	moduleSet := stringSet(activeModules)
	resolved := map[string]PolicyCapability{}
	policyEnabled := map[string]bool{}

	for _, pack := range packs {
		for _, rule := range pack.Rules.Capabilities {
			code := strings.TrimSpace(rule.Code)
			if code == "" {
				continue
			}
			current := resolved[code]
			current.Code = code
			if rule.Reason != "" && (current.Reason == "" || rule.Enabled) {
				current.Reason = rule.Reason
			}
			current.RequiredDocuments = unionStrings(current.RequiredDocuments, rule.RequiredDocuments)
			current.RequiredApprovals = unionStrings(current.RequiredApprovals, rule.RequiredApprovals)
			current.WizardSteps = unionStrings(current.WizardSteps, rule.WizardSteps)
			policyEnabled[code] = policyEnabled[code] || rule.Enabled
			resolved[code] = current
		}
	}

	items := make([]PolicyCapability, 0, len(resolved))
	for code, item := range resolved {
		requirement, known := capabilityAuthorization[code]
		if !known {
			item.Enabled = false
			item.Reason = "Capabilitatea nu este înregistrată în catalogul serverului"
			items = append(items, item)
			continue
		}
		item.RequiredPermission = requirement.Permission
		item.Enabled = policyEnabled[code]
		if requirement.Module != "" {
			_, item.Enabled = moduleSet[requirement.Module]
			item.Enabled = item.Enabled && policyEnabled[code]
		}
		if requirement.Permission != "" {
			_, permitted := permissionSet[requirement.Permission]
			item.Enabled = item.Enabled && permitted
			if !permitted && policyEnabled[code] {
				item.Reason = "Permisiunea necesară nu este acordată în contextul curent"
			}
		}
		items = append(items, item)
	}
	sort.Slice(items, func(i, j int) bool { return items[i].Code < items[j].Code })
	return items
}

func stringSet(values []string) map[string]struct{} {
	set := make(map[string]struct{}, len(values))
	for _, value := range values {
		if value = strings.TrimSpace(value); value != "" {
			set[value] = struct{}{}
		}
	}
	return set
}

func unionStrings(left, right []string) []string {
	set := stringSet(left)
	for _, value := range right {
		if value = strings.TrimSpace(value); value != "" {
			set[value] = struct{}{}
		}
	}
	values := make([]string, 0, len(set))
	for value := range set {
		values = append(values, value)
	}
	sort.Strings(values)
	return values
}
