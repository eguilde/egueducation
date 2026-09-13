package db

import (
	"os"
	"strings"
	"testing"
)

func TestAdmissionLegalPrepareFinalizeMigrationIsScopedDualAuthorizedAndFailClosed(t *testing.T) {
	raw, err := os.ReadFile("migrations/0152_school_admission_legal_prepare_finalize.sql")
	if err != nil {
		t.Fatal(err)
	}
	text := string(raw)
	for _, want := range []string{
		"school_admission_legal_preparations", "canonical_payload_bytes", "resulting_decision_payload_bytes", "canonical_payload_sha256", "capacity_allocation_id", "aggregate_expected_version", "expires_at", "status in ('prepared','finalized','cancelled','expired')",
		"school_admission_signer_authorizations", "certificate_sha256", "actor_subject", "permission_code", "valid_until", "distinct director approver", "school_admission_actor_has_position('director')",
		"canonical legal preparation payload bytes and hash must match", "resulting decision payload bytes and hash must match", "admission legal preparation lifecycle records are immutable", "admission signer authorization lifecycle records are immutable", "expired signer authorization cannot be approved", "expired preparation requires the server expiry reason",
		"force row level security", "school_admission_legal_preparation_tenant_isolation", "school_admission_signer_authorization_tenant_isolation", "immutable", "school_operations_no_hard_delete",
	} {
		if !strings.Contains(text, want) {
			t.Fatalf("0152 migration lacks %q", want)
		}
	}
}

func TestAdmissionLegalPrepareFinalizeSchemaContractsAreTenantScoped(t *testing.T) {
	contracts := map[string]TableContract{}
	for _, c := range SchemaContract() {
		contracts[c.Name] = c
	}
	for name, policy := range map[string]string{
		"school_admission_legal_preparations":    "school_admission_legal_preparation_tenant_isolation",
		"school_admission_signer_authorizations": "school_admission_signer_authorization_tenant_isolation",
	} {
		c, ok := contracts[name]
		if !ok || c.Scope != SchemaScopeInstitution || len(c.RequiredPolicies) != 1 || c.RequiredPolicies[0] != policy {
			t.Errorf("missing scoped contract for %s: %#v", name, c)
		}
	}
}
