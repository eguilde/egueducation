package db

import (
	"slices"
	"testing"
)

func TestSchemaContractHasUniqueTables(t *testing.T) {
	seen := make(map[string]struct{}, len(SchemaContract()))

	for _, table := range SchemaContract() {
		if _, ok := seen[table.Name]; ok {
			t.Fatalf("duplicate table contract entry: %s", table.Name)
		}
		seen[table.Name] = struct{}{}

		switch table.Scope {
		case SchemaScopeTenant, SchemaScopeInstitution:
			if table.filterColumn() == "" {
				t.Fatalf("scoped table %s must have a discriminator column", table.Name)
			}
		}
	}
}

func TestSchemaContractIncludesRecentSchoolSecurityTables(t *testing.T) {
	expected := map[string][]string{
		"education_portfolio_archive_attachment_grants":        {"institution_id", "archive_document_id", "grantee_user_id"},
		"education_role_delegations":                           {"tenant_code", "institution_id", "delegator_user_id", "delegate_user_id", "permission_code", "status", "valid_from"},
		"education_portfolio_valorification_packages":          {"tenant_code", "institution_id", "portfolio_id", "scope", "status"},
		"education_portfolio_valorification_package_documents": {"institution_id", "package_id", "archive_version_id", "archive_sha256"},
	}
	contracts := make(map[string]TableContract, len(SchemaContract()))
	for _, contract := range SchemaContract() {
		contracts[contract.Name] = contract
	}
	for name, requiredColumns := range expected {
		contract, ok := contracts[name]
		if !ok {
			t.Errorf("schema contract omits %s", name)
			continue
		}
		if contract.Scope != SchemaScopeInstitution || !contract.requiresRLS() {
			t.Errorf("%s must require institution RLS, got %#v", name, contract)
		}
		for _, column := range requiredColumns {
			if !slices.Contains(contract.RequiredColumns, column) {
				t.Errorf("schema contract %s omits required column %s", name, column)
			}
		}
	}
}

func TestRuntimeDatabaseRoleMustNotBypassRLS(t *testing.T) {
	for _, test := range []struct {
		name      string
		role      string
		superuser bool
		bypassRLS bool
		wantError bool
	}{
		{name: "application role", role: "egueducation_app"},
		{name: "superuser", role: "postgres", superuser: true, wantError: true},
		{name: "bypass rls", role: "migration_owner", bypassRLS: true, wantError: true},
		{name: "missing role", role: "", wantError: true},
	} {
		t.Run(test.name, func(t *testing.T) {
			err := validateRuntimeDatabaseRoleFlags(test.role, test.superuser, test.bypassRLS)
			if (err != nil) != test.wantError {
				t.Fatalf("validateRuntimeDatabaseRoleFlags() error = %v, wantError %v", err, test.wantError)
			}
		})
	}
}
