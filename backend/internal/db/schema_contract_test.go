package db

import "testing"

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
