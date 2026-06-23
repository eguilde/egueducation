package db

import (
	"context"
	"fmt"
	"strings"

	"github.com/jackc/pgx/v5/pgxpool"
)

type SchemaScope string

const (
	SchemaScopeGlobal      SchemaScope = "global"
	SchemaScopeInstitution SchemaScope = "institution"
	SchemaScopeTenant      SchemaScope = "tenant"
)

type TableContract struct {
	Name            string
	Scope           SchemaScope
	RequiredColumns []string
	Notes           string
}

func (t TableContract) requiresRLS() bool {
	return t.Scope != SchemaScopeGlobal
}

func (t TableContract) filterColumn() string {
	switch t.Scope {
	case SchemaScopeTenant:
		return "tenant_code"
	case SchemaScopeInstitution:
		return "institution_id"
	default:
		return ""
	}
}

func globalTable(name, notes string) TableContract {
	return TableContract{Name: name, Scope: SchemaScopeGlobal, Notes: notes}
}

func globalTableWithColumns(name, notes string, requiredColumns ...string) TableContract {
	return TableContract{Name: name, Scope: SchemaScopeGlobal, RequiredColumns: requiredColumns, Notes: notes}
}

func institutionTable(name, notes string) TableContract {
	return TableContract{
		Name:            name,
		Scope:           SchemaScopeInstitution,
		RequiredColumns: []string{"institution_id"},
		Notes:           notes,
	}
}

func tenantTable(name, notes string) TableContract {
	return TableContract{
		Name:            name,
		Scope:           SchemaScopeTenant,
		RequiredColumns: []string{"tenant_code"},
		Notes:           notes,
	}
}

func SchemaContract() []TableContract {
	return []TableContract{
		globalTableWithColumns("schema_migrations", "Applied migration ledger.", "version", "applied_at"),
		globalTableWithColumns("app_users", "Identity directory.", "id", "sub", "name", "email", "phone_number", "locale"),
		globalTableWithColumns("app_roles", "Role catalog.", "code", "label"),
		globalTable("app_user_roles", "User-role grants."),
		globalTableWithColumns("app_permissions", "Permission catalog.", "code", "label"),
		globalTable("app_user_permissions", "User-permission grants."),
		globalTableWithColumns("app_modules", "Feature module flags.", "code", "active"),
		globalTable("app_user_modules", "User module grants."),
		globalTableWithColumns("app_session_context", "User session bootstrap context.", "user_id", "institution_id", "institution_name", "auth_methods", "gdpr_capabilities"),
		globalTableWithColumns("app_position_roles", "Position-to-role mapping.", "position_code", "role_code"),
		globalTableWithColumns("app_auth_methods", "Authentication method catalog.", "code", "enabled", "primary_method", "sort_order"),
		globalTableWithColumns("app_nomenclatures", "Shared nomenclatures.", "id", "domain", "code", "label_ro", "label_en", "active", "sort_order"),
		globalTableWithColumns("app_tenants", "Tenant registry keyed by institution.", "code", "subdomain", "institution_id", "display_name", "short_name", "root_org_unit_code", "active"),
		globalTableWithColumns("workflow_definitions", "Global workflow catalog shared across tenants.", "code", "name", "category", "initial_step", "sla_hours", "active"),
		globalTableWithColumns("education_requirement_catalog", "Global education requirement catalog.", "id", "domain", "code", "title_ro", "title_en", "source_ref", "requirement_type", "implementation_status", "priority"),
		globalTableWithColumns("education_portfolio_sections", "Global portfolio section catalog.", "id", "section_code", "component_code", "label_ro", "label_en", "required", "sensitive_data", "sort_order", "active"),
		tenantTable("app_org_units", "Organization units are tenant-scoped."),
		tenantTable("app_memberships", "Memberships are tenant-scoped."),
		institutionTable("registratura_documents", "Incoming/outgoing registry documents."),
		institutionTable("archive_records", "Electronic archive records."),
		institutionTable("workflow_instances", "Runtime workflow instances."),
		institutionTable("education_meetings", "Governance meetings."),
		institutionTable("education_personnel", "Personnel master data."),
		institutionTable("education_portfolios", "Personnel portfolio records."),
		institutionTable("education_mobility_cases", "Mobility cases."),
		institutionTable("education_merit_grants", "Merit grant cases."),
		institutionTable("gdpr_retention_policies", "GDPR retention policies."),
		institutionTable("gdpr_subject_requests", "GDPR subject requests."),
		institutionTable("education_regulations", "Regulation dossiers."),
		institutionTable("education_evaluations", "Evaluation dossiers."),
		institutionTable("education_declarations", "Education declarations."),
		institutionTable("education_decisions", "Education decisions."),
		institutionTable("education_managerial_dossiers", "Managerial dossiers."),
		institutionTable("gdpr_subject_exports", "GDPR export jobs."),
		institutionTable("gdpr_publication_reviews", "Publication review flows."),
		institutionTable("education_meeting_participants", "Meeting participants."),
		institutionTable("education_meeting_documents", "Meeting documents."),
		institutionTable("education_portfolio_documents", "Portfolio documents."),
		institutionTable("education_meeting_votes", "Meeting votes."),
		institutionTable("education_portfolio_checklist", "Portfolio checklist items."),
		institutionTable("education_governance_memberships", "Governance memberships."),
		institutionTable("education_meeting_resolutions", "Meeting resolutions."),
		institutionTable("education_portfolio_transfers", "Portfolio handovers."),
		institutionTable("education_portfolio_reviews", "Portfolio reviews."),
		institutionTable("education_meeting_minutes", "Meeting minutes."),
		institutionTable("education_portfolio_opis", "Portfolio inventory list."),
		institutionTable("education_portfolio_custody", "Portfolio custody history."),
		institutionTable("education_publications", "Publications and notices."),
		institutionTable("education_managerial_documents", "Managerial document flow."),
		institutionTable("education_managerial_workflow_steps", "Managerial workflow steps."),
		institutionTable("education_regulation_versions", "Regulation versioning."),
		institutionTable("education_regulation_workflow_steps", "Regulation workflow steps."),
		institutionTable("education_decision_issuances", "Decision issuance tracking."),
		institutionTable("education_decision_publication_steps", "Decision publication workflow."),
		institutionTable("education_mobility_documents", "Mobility documents."),
		institutionTable("education_mobility_scores", "Mobility scoring rows."),
		institutionTable("education_mobility_appeals", "Mobility appeals."),
		institutionTable("education_merit_documents", "Merit documents."),
		institutionTable("education_merit_scores", "Merit scoring rows."),
		institutionTable("education_merit_appeals", "Merit appeals."),
		institutionTable("education_mobility_final_decisions", "Mobility final decisions."),
		institutionTable("education_mobility_result_issues", "Mobility result issues."),
		institutionTable("education_merit_final_decisions", "Merit final decisions."),
		institutionTable("education_merit_result_issues", "Merit result issues."),
		institutionTable("education_personnel_assignments", "Personnel assignments."),
		institutionTable("education_personnel_disciplinary_cases", "Disciplinary cases."),
		institutionTable("education_personnel_file_documents", "Personnel file documents."),
		institutionTable("education_personnel_access_events", "Personnel file access log."),
		institutionTable("education_evaluation_appeals", "Evaluation appeals."),
		institutionTable("education_evaluation_self_reviews", "Self-review rows."),
		institutionTable("education_evaluation_criteria", "Evaluation criteria rows."),
		institutionTable("education_evaluation_result_issues", "Evaluation result issues."),
		globalTableWithColumns("app_entity_versions", "Cross-module version history captures both tenant and institution context.", "id", "entity_table", "entity_id", "version_no", "change_type", "tenant_code", "institution_id", "snapshot", "changed_by", "changed_at"),
	}
}

func ValidateSchemaContract(ctx context.Context, pool *pgxpool.Pool) error {
	var violations []string

	for _, table := range SchemaContract() {
		exists, err := tableExists(ctx, pool, table.Name)
		if err != nil {
			violations = append(violations, fmt.Sprintf("check table %s: %v", table.Name, err))
			continue
		}
		if !exists {
			violations = append(violations, fmt.Sprintf("schema contract violation: table %s is missing", table.Name))
			continue
		}

		for _, column := range table.RequiredColumns {
			hasColumn, err := columnExists(ctx, pool, table.Name, column)
			if err != nil {
				violations = append(violations, fmt.Sprintf("check column %s.%s: %v", table.Name, column, err))
				continue
			}
			if !hasColumn {
				violations = append(violations, fmt.Sprintf("schema contract violation: table %s is missing required column %s", table.Name, column))
			}
		}

		if table.requiresRLS() {
			if err := validateRowLevelSecurity(ctx, pool, table); err != nil {
				violations = append(violations, err.Error())
			}
		}
	}

	if len(violations) > 0 {
		return fmt.Errorf("schema contract validation failed:\n- %s", strings.Join(violations, "\n- "))
	}

	return nil
}

func tableExists(ctx context.Context, pool *pgxpool.Pool, tableName string) (bool, error) {
	var exists bool
	if err := pool.QueryRow(ctx, `
		select exists (
			select 1
			from information_schema.tables
			where table_schema = 'public'
				and table_name = $1
		)
	`, tableName).Scan(&exists); err != nil {
		return false, err
	}
	return exists, nil
}

func columnExists(ctx context.Context, pool *pgxpool.Pool, tableName, columnName string) (bool, error) {
	var exists bool
	if err := pool.QueryRow(ctx, `
		select exists (
			select 1
			from information_schema.columns
			where table_schema = 'public'
				and table_name = $1
				and column_name = $2
		)
	`, tableName, columnName).Scan(&exists); err != nil {
		return false, err
	}
	return exists, nil
}

func validateRowLevelSecurity(ctx context.Context, pool *pgxpool.Pool, table TableContract) error {
	var relRowSecurity bool
	var relForceRowSecurity bool
	if err := pool.QueryRow(ctx, `
		select c.relrowsecurity, c.relforcerowsecurity
		from pg_class c
		join pg_namespace n on n.oid = c.relnamespace
		where n.nspname = 'public'
			and c.relname = $1
	`, table.Name).Scan(&relRowSecurity, &relForceRowSecurity); err != nil {
		return fmt.Errorf("check row level security for %s: %w", table.Name, err)
	}
	if !relRowSecurity || !relForceRowSecurity {
		return fmt.Errorf("schema contract violation: table %s must have forced row level security enabled", table.Name)
	}

	var policyExists bool
	if err := pool.QueryRow(ctx, `
		select exists (
			select 1
			from pg_policies
			where schemaname = 'public'
				and tablename = $1
				and policyname = 'tenant_isolation'
		)
	`, table.Name).Scan(&policyExists); err != nil {
		return fmt.Errorf("check row level security policy for %s: %w", table.Name, err)
	}
	if !policyExists {
		return fmt.Errorf("schema contract violation: table %s is missing tenant_isolation policy", table.Name)
	}

	if column := table.filterColumn(); column != "" {
		hasColumn, err := columnExists(ctx, pool, table.Name, column)
		if err != nil {
			return fmt.Errorf("check rls discriminator %s.%s: %w", table.Name, column, err)
		}
		if !hasColumn {
			return fmt.Errorf("schema contract violation: table %s is missing rls discriminator column %s", table.Name, column)
		}
	}

	return nil
}
