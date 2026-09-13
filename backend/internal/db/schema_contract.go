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
	Name             string
	Scope            SchemaScope
	RequiredColumns  []string
	RequiredPolicies []string
	Notes            string
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

func institutionTableWithColumns(name, notes string, requiredColumns ...string) TableContract {
	return TableContract{
		Name:            name,
		Scope:           SchemaScopeInstitution,
		RequiredColumns: append([]string{"institution_id"}, requiredColumns...),
		Notes:           notes,
	}
}

func institutionTableWithColumnsAndPolicies(name, notes string, requiredPolicies []string, requiredColumns ...string) TableContract {
	return TableContract{
		Name:             name,
		Scope:            SchemaScopeInstitution,
		RequiredColumns:  append([]string{"institution_id"}, requiredColumns...),
		RequiredPolicies: append([]string(nil), requiredPolicies...),
		Notes:            notes,
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
		globalTableWithColumns("app_user_identities", "Normalized login identifiers independent from profile fields.", "user_id", "identity_type", "normalized_value", "display_value", "verified_at", "is_primary"),
		globalTableWithColumns("app_platform_roles", "Platform authority catalog; never inferred from tenant roles.", "code", "label"),
		globalTableWithColumns("app_user_platform_roles", "Explicit global platform-role assignments.", "user_id", "role_code"),
		globalTableWithColumns("app_roles", "Role catalog.", "code", "label"),
		globalTableWithColumns("app_user_roles", "Tenant-scoped user-role grants enforced by authorization queries.", "tenant_code"),
		globalTableWithColumns("app_permissions", "Permission catalog.", "code", "label"),
		globalTableWithColumns("app_user_permissions", "Tenant-scoped user-permission grants enforced by authorization queries.", "tenant_code"),
		globalTableWithColumns("app_modules", "Feature module flags.", "code", "active"),
		globalTableWithColumns("app_user_modules", "Tenant-scoped user module grants enforced by authorization queries.", "tenant_code"),
		globalTableWithColumns("app_session_context", "Tenant-keyed user session bootstrap context.", "user_id", "tenant_code", "institution_id", "institution_name", "auth_methods", "gdpr_capabilities"),
		globalTableWithColumns("app_position_roles", "Position-to-role mapping.", "position_code", "role_code"),
		globalTableWithColumns("app_auth_methods", "Authentication method catalog.", "code", "enabled", "primary_method", "sort_order"),
		globalTableWithColumns("oidc_otp_challenges", "OIDC OTP proofs bound to the exact tenant and authentication session.", "user_id", "identity_id", "tenant_code", "authn_session_id", "purpose", "code_hash", "expires_at", "attempts"),
		globalTableWithColumns("app_nomenclatures", "Shared nomenclatures.", "id", "domain", "code", "label_ro", "label_en", "active", "sort_order"),
		globalTableWithColumns("app_tenants", "Tenant registry keyed by institution.", "code", "subdomain", "institution_id", "display_name", "short_name", "root_org_unit_code", "active"),
		globalTableWithColumns("workflow_definitions", "Global workflow catalog shared across tenants.", "code", "name", "category", "initial_step", "sla_hours", "active"),
		globalTableWithColumns("education_requirement_catalog", "Global education requirement catalog.", "id", "domain", "code", "title_ro", "title_en", "source_ref", "requirement_type", "implementation_status", "priority"),
		globalTableWithColumns("education_portfolio_sections", "Global portfolio section catalog.", "id", "section_code", "component_code", "label_ro", "label_en", "required", "sensitive_data", "sort_order", "active", "catalog_version", "source_ref"),
		globalTableWithColumns("education_portfolio_section_catalog_versions", "Versioned legal sources for the global portfolio section catalog.", "code", "source_ref", "status"),
		globalTableWithColumns("education_portfolio_declaration_templates", "Server-issued versioned declaration wording.", "declaration_type", "declaration_version", "declaration_text", "source_ref", "effective_from", "lifecycle_status"),
		institutionTableWithColumns("school_institution_profiles", "Effective-dated public/private/confessional institution classification.", "tenant_code", "version", "status", "school_legal_form", "regulatory_profile", "effective_from", "effective_to", "created_by_subject", "updated_by_subject"),
		institutionTableWithColumns("school_policy_pack_versions", "Versioned institution policy packs with approved sources and checksum.", "tenant_code", "pack_code", "version", "status", "rules", "rules_schema", "checksum_sha256", "created_by_subject", "updated_by_subject"),
		institutionTableWithColumns("school_policy_assignments", "Effective policy assignments for one immutable profile version.", "tenant_code", "policy_pack_version_id", "profile_id", "profile_version", "pack_code", "assignment_kind", "status", "version", "created_by_subject", "created_at", "updated_at"),
		institutionTableWithColumns("school_policy_overrides", "Approved, bounded institution policy overrides.", "tenant_code", "policy_assignment_id", "key", "value", "status", "version", "created_by_subject", "updated_by_subject"),
		institutionTableWithColumns("school_policy_evaluations", "Immutable policy snapshots used by regulated operations.", "tenant_code", "profile_id", "profile_version", "policy_pack_version_ids", "capabilities", "blocked", "checksum_sha256", "evaluated_by_subject", "created_by_subject"),
		institutionTableWithColumns("school_institution_profiles_v2", "Normalized public/private profile versions; unclassified is a controlled migration-only state.", "tenant_code", "version", "status", "legal_form", "effective_from", "profile_series_id", "supersedes_profile_id", "expected_version"),
		institutionTableWithColumns("school_regulatory_sources", "Verified, versioned legal provenance for school policy and authorization decisions.", "tenant_code", "source_kind", "citation", "source_url", "checksum_sha256", "status", "verified_at", "verified_by_subject", "revalidation_owner_subject", "expected_version"),
		institutionTableWithColumns("school_profile_sources", "Typed source bindings for exact normalized institution profiles.", "tenant_code", "profile_id", "source_id", "purpose", "created_by_subject"),
		institutionTableWithColumns("school_institution_party_roles", "Effective-dated founder, funder, budget authority, cult and operator roles.", "tenant_code", "party_id", "role_code", "effective_from", "effective_to", "source_id", "expected_version"),
		institutionTableWithColumns("school_confessional_profiles", "Private-school confessional overlay bound to an exact profile version and cult party.", "tenant_code", "profile_id", "profile_version", "cult_party_id", "cult_code", "protocol_reference", "status", "effective_from", "effective_to", "source_id", "expected_version"),
		institutionTableWithColumns("school_confessional_protocols", "Effective-dated governance, curriculum, personnel and facility protocols for a confessional overlay.", "tenant_code", "confessional_profile_id", "protocol_type", "reference", "status", "effective_from", "effective_to", "source_id"),
		institutionTableWithColumns("school_locations", "Effective-dated physical locations used by authorization decisions.", "tenant_code", "code", "name", "address", "active", "effective_from", "effective_to", "expected_version"),
		institutionTableWithColumns("school_education_offerings", "Effective-dated education level, program, specialization and language offerings.", "tenant_code", "code", "education_level", "specialization_code", "language_code", "title", "active", "effective_from", "effective_to", "expected_version"),
		institutionTableWithColumns("school_offering_authorizations", "Append-oriented authorization/accreditation history per offering and location.", "tenant_code", "offering_id", "location_id", "status", "authority_name", "decision_reference", "capacity", "capacity_unit", "shift", "effective_from", "effective_to", "source_id", "replaces_authorization_id", "expected_version"),
		institutionTableWithColumns("school_funding_instruments", "Effective-dated funding instrument independent from legal form and procurement applicability.", "tenant_code", "code", "public_funding", "school_year", "eligibility_status", "offering_id", "beneficiary_party_id", "tuition_eligible", "source_id", "expected_version"),
		institutionTableWithColumns("school_funding_eligibility_evaluations", "Immutable tri-state funding eligibility snapshot for a beneficiary, offering and date.", "tenant_code", "funding_instrument_id", "evaluated_for_date", "status", "criteria", "evidence", "warnings", "checksum_sha256", "evaluated_by_subject"),
		institutionTableWithColumns("school_procurement_applicability_assessments", "Immutable legal applicability decision separated from funding and legal form.", "tenant_code", "assessment_scope", "funding_instrument_id", "determination", "legal_basis", "rationale", "subject_type", "subject_reference", "criteria", "evidence", "effective_from", "effective_to", "source_id", "decided_by_subject"),
		institutionTableWithColumns("school_education_contracts", "Effective-dated education contracts linking learner, guardian and authorized offering.", "tenant_code", "student_party_id", "guardian_party_id", "offering_id", "contract_number", "status", "starts_on", "ends_on", "tuition_amount", "expected_version", "source_id"),
		institutionTableWithColumns("school_quality_cycles", "CEAC/RAEI and external-quality cycle evidence common to public and private schools.", "tenant_code", "code", "status", "starts_on", "ends_on", "source_id", "expected_version"),
		institutionTableWithColumns("school_network_memberships", "Annual/effective school-network membership and status evidence.", "tenant_code", "network_code", "network_name", "status", "effective_from", "effective_to", "source_id", "expected_version"),
		institutionTableWithColumns("school_profile_cutover_identity", "Immutable v2-to-public-API and optional legacy profile identity bridge.", "id", "tenant_code", "profile_v2_id", "profile_v2_version", "api_profile_id", "api_version", "legacy_profile_id", "mapping_kind"),
		institutionTableWithColumns("school_institution_profile_api_projection", "Compatibility projection derived from a normalized profile version during dual-read cutover.", "id", "tenant_code", "profile_v2_id", "profile_v2_version", "regulatory_profile", "authorization_status", "accreditation_reference", "is_contracting_authority", "has_legal_personality", "tax_identifier", "vat_profile", "treasury_required", "public_funding", "source_reference", "updated_by_subject"),
		institutionTableWithColumns("school_operation_policy_bindings_v2", "Versioned normalized policy-pack bindings for exact v2 profile versions.", "tenant_code", "profile_v2_id", "profile_v2_version", "policy_pack_version_id", "pack_code", "assignment_kind", "status", "expected_version"),
		institutionTableWithColumns("school_operation_policy_overrides_v2", "Versioned approved overrides for normalized policy bindings.", "tenant_code", "binding_id", "key", "value", "status", "expected_version"),
		institutionTableWithColumns("school_operation_policy_input_bindings", "Immutable exact pack and override snapshots used by an operation policy input.", "id", "tenant_code", "input_id", "binding_id", "profile_v2_id", "profile_v2_version", "policy_pack_version_id", "policy_pack_checksum_sha256", "override_snapshot"),
		institutionTableWithColumns("school_operation_policy_inputs", "Immutable v2 operation-policy input with effective decision context.", "tenant_code", "profile_id", "profile_version", "profile_series_id", "effective_on", "decision_kind", "engine_version", "schema_version", "checksum_sha256"),
		institutionTableWithColumns("school_operation_policy_evaluations_v2", "Immutable v2 operation-policy decision, unique for each exact input.", "tenant_code", "input_id", "effective_on", "decision_kind", "engine_version", "schema_version", "allowed", "capabilities", "obligations"),
		institutionTableWithColumnsAndPolicies("school_admission_class_offering_contexts", "Exact effective class, offering, location and authorization admission context.", []string{"school_admission_tenant_isolation"}, "tenant_code", "class_id", "offering_id", "location_id", "authorization_id", "school_year", "shift", "effective_from", "effective_to", "expected_version", "created_by_subject", "updated_by_subject"),
		institutionTableWithColumnsAndPolicies("school_admission_campaigns", "Effective admission campaign with authorization-bounded capacity and an explicit student-place basis.", []string{"school_admission_tenant_isolation"}, "tenant_code", "code", "offering_id", "location_id", "authorization_id", "class_offering_context_id", "capacity_limit", "capacity_unit", "student_place_limit", "capacity_basis", "shift", "opens_on", "closes_on", "status", "expected_version", "created_by_subject", "updated_by_subject"),
		institutionTableWithColumnsAndPolicies("school_admission_criteria", "Versioned campaign eligibility, ranking and tie-breaker criteria.", []string{"school_admission_tenant_isolation"}, "tenant_code", "campaign_id", "code", "criterion_kind", "required", "rule_snapshot", "expected_version", "created_by_subject", "updated_by_subject"),
		institutionTableWithColumnsAndPolicies("school_admission_document_requirements", "Versioned admission evidence requirements for one campaign.", []string{"school_admission_tenant_isolation"}, "tenant_code", "campaign_id", "code", "required", "allowed_mime_types", "expected_version", "created_by_subject", "updated_by_subject"),
		institutionTableWithColumnsAndPolicies("school_admission_applications", "Scoped candidate admission application linked to a party and optional canonical student.", []string{"school_admission_tenant_isolation"}, "tenant_code", "campaign_id", "application_no", "candidate_party_id", "student_id", "status", "consent_snapshot", "expected_version", "created_by_subject", "updated_by_subject"),
		institutionTableWithColumnsAndPolicies("school_admission_capacity_allocations", "One active held or consumed student-place allocation per application.", []string{"school_admission_tenant_isolation"}, "tenant_code", "application_id", "campaign_id", "class_offering_context_id", "authorization_id", "allocated_capacity", "capacity_unit", "shift", "status", "expected_version", "created_by_subject", "updated_by_subject"),
		institutionTableWithColumnsAndPolicies("school_admission_application_representatives", "Scoped guardian, legal-representative and proxy evidence for an application.", []string{"school_admission_tenant_isolation"}, "tenant_code", "application_id", "representative_party_id", "relationship_type", "expected_version", "created_by_subject", "updated_by_subject"),
		institutionTableWithColumnsAndPolicies("school_admission_application_documents", "WORM-snapshotted admission application evidence.", []string{"school_admission_tenant_isolation"}, "tenant_code", "application_id", "document_requirement_id", "status", "archive_document_id", "archive_version_id", "archive_version_no", "archive_source_bucket", "archive_source_object_key", "archive_source_object_version_id", "archive_retention_until", "archive_sha256", "expected_version", "created_by_subject", "updated_by_subject"),
		institutionTableWithColumnsAndPolicies("school_admission_criterion_assessments", "Fail-closed individual assessment of exact campaign criteria.", []string{"school_admission_tenant_isolation"}, "tenant_code", "application_id", "criterion_id", "outcome", "evidence_snapshot", "assessed_by_subject", "assessed_at", "expected_version", "created_by_subject", "updated_by_subject"),
		institutionTableWithColumnsAndPolicies("school_admission_decisions", "Immutable WORM-backed admission decision with exact allowed policy-v2 evaluation.", []string{"school_admission_tenant_isolation"}, "tenant_code", "application_id", "capacity_allocation_id", "enrolment_id", "policy_evaluation_v2_id", "outcome", "decided_by_subject", "decision_snapshot", "archive_document_id", "archive_version_id", "archive_source_object_version_id", "archive_retention_until", "archive_sha256"),
		institutionTableWithColumnsAndPolicies("school_admission_decision_deliveries", "Immutable decision-delivery evidence by channel and provider reference.", []string{"school_admission_tenant_isolation"}, "tenant_code", "decision_id", "channel", "status", "delivery_snapshot", "created_by_subject"),
		institutionTableWithColumnsAndPolicies("school_admission_appeals", "Scoped appeal against the exact admission decision, including late rejection.", []string{"school_admission_tenant_isolation"}, "tenant_code", "application_id", "decision_id", "appeal_no", "status", "expected_version", "created_by_subject", "updated_by_subject"),
		institutionTableWithColumnsAndPolicies("school_admission_appeal_submissions", "Immutable WORM-snapshotted appeal submission evidence.", []string{"school_admission_tenant_isolation"}, "tenant_code", "appeal_id", "submission_no", "archive_document_id", "archive_version_id", "archive_source_object_version_id", "archive_retention_until", "archive_sha256", "created_by_subject"),
		institutionTableWithColumnsAndPolicies("school_admission_appeal_resolutions", "Immutable WORM-backed appeal resolution with an independent policy-v2 evaluation.", []string{"school_admission_tenant_isolation"}, "tenant_code", "appeal_id", "resulting_decision_id", "resulting_outcome", "capacity_allocation_id", "policy_evaluation_v2_id", "outcome", "resolved_by_subject", "archive_document_id", "archive_version_id", "archive_source_object_version_id", "archive_retention_until", "archive_sha256"),
		institutionTableWithColumnsAndPolicies("school_admission_export_manifests", "Immutable WORM-backed admission export manifest.", []string{"school_admission_tenant_isolation"}, "tenant_code", "campaign_id", "application_id", "manifest_version", "manifest_sha256", "manifest", "archive_document_id", "archive_version_id", "archive_source_object_version_id", "archive_retention_until", "archive_sha256", "generated_by_subject"),
		institutionTableWithColumnsAndPolicies("school_admission_idempotency", "Scoped idempotency ledger for admission commands.", []string{"school_admission_tenant_isolation"}, "tenant_code", "actor_subject", "operation_code", "idempotency_key", "request_fingerprint", "response_snapshot"),
		institutionTableWithColumnsAndPolicies("school_admission_outbox", "Durable scoped delivery intents emitted by admission commands.", []string{"school_admission_tenant_isolation"}, "tenant_code", "aggregate_type", "aggregate_id", "event_type", "payload", "occurred_by_subject", "delivery_status", "attempt_count"),
		institutionTableWithColumnsAndPolicies("school_admission_dss_retention_policies", "Effective operational retention policy bound to an approved legal rule version.", []string{"school_admission_dss_tenant_isolation"}, "tenant_code", "institution_id", "status", "minimum_retention_days", "source_id", "rule_version_id", "effective_from", "effective_to", "created_by_subject"),
		institutionTableWithColumnsAndPolicies("school_admission_signed_artifact_bindings", "Immutable admission legal-artifact bridge to exact DSS evidence and WORM provenance.", []string{"school_admission_dss_tenant_isolation"}, "tenant_code", "institution_id", "artifact_kind", "artifact_id", "evidence_id", "validation_id", "policy_evaluation_v2_id", "retention_policy_id", "canonical_legal_payload_sha256", "expected_actor_subject", "archive_document_id", "archive_version_id", "archive_sha256", "archive_size_bytes", "bound_by_subject"),
		institutionTableWithColumnsAndPolicies("school_admission_retention_rule_versions", "Immutable dual-approved legal retention rules with captured regulatory-source checksum.", []string{"school_admission_retention_rule_tenant_isolation"}, "tenant_code", "institution_id", "artifact_kind", "jurisdiction", "minimum_retention_days", "source_id", "source_checksum_sha256", "effective_from", "effective_to", "status", "proposed_by_subject", "approved_by_subject", "approved_at", "expected_version"),
		institutionTableWithColumnsAndPolicies("school_admission_legal_preparations", "Short-lived immutable legal signing preparation with exact canonical payload bytes, aggregate version and capacity hold.", []string{"school_admission_legal_preparation_tenant_isolation"}, "tenant_code", "institution_id", "artifact_kind", "artifact_id", "application_id", "appeal_id", "resulting_decision_id", "capacity_allocation_id", "policy_evaluation_v2_id", "aggregate_expected_version", "canonical_payload", "canonical_payload_bytes", "canonical_payload_sha256", "resulting_decision_payload", "resulting_decision_payload_bytes", "resulting_decision_payload_sha256", "preparation_snapshot", "prepared_by_subject", "prepared_at", "expires_at", "status", "expected_version"),
		institutionTableWithColumnsAndPolicies("school_admission_signer_authorizations", "Immutable dual-director certificate fingerprint authorization for an admission legal signer.", []string{"school_admission_signer_authorization_tenant_isolation"}, "tenant_code", "institution_id", "certificate_sha256", "user_id", "actor_subject", "permission_code", "valid_from", "valid_until", "status", "proposed_by_subject", "approved_by_subject", "approved_at", "revoked_by_subject", "revoked_at", "expected_version"),
		institutionTableWithColumns("school_policy_evaluation_cutover_identity", "Immutable bridge between legacy and v2 policy decisions.", "id", "tenant_code", "legacy_evaluation_id", "v2_evaluation_id", "input_id", "conversion_kind"),
		institutionTableWithColumns("school_policy_cutover_state", "Per-institution explicit legacy, dual, v2 or contracted authorization phase.", "id", "tenant_code", "phase", "expected_version", "shadow_compared_at", "cutover_authorized_by_subject"),
		institutionTableWithColumns("school_regulatory_migration_issues", "Tenant-scoped unresolved migration issues that block unsafe policy cutover.", "tenant_code", "entity_type", "entity_id", "issue_code", "severity", "status", "details", "resolved_by_subject"),
		institutionTableWithColumns("school_operation_idempotency", "Idempotency ledger for operational create commands.", "tenant_code", "operation", "idempotency_key", "response_id", "request_fingerprint", "created_by_subject"),
		institutionTableWithColumns("school_contracts", "Versioned supplier contracts; final legal records are never hard deleted.", "tenant_code", "supplier_party_id", "contract_number", "lifecycle_status", "expected_version", "policy_evaluation_id", "policy_evaluation_v2_id", "archive_status"),
		institutionTableWithColumns("school_contract_versions", "Immutable contract and amendment history.", "tenant_code", "contract_id", "version", "change_kind", "snapshot", "created_by_subject"),
		institutionTableWithColumns("school_contract_obligations", "Contract SLA, guarantee and fulfilment obligations.", "tenant_code", "contract_id", "status", "expected_version", "policy_evaluation_id", "policy_evaluation_v2_id"),
		institutionTableWithColumns("school_utility_points", "Utility consumption points scoped to the institution.", "tenant_code", "utility_type", "expected_version", "policy_evaluation_id", "policy_evaluation_v2_id"),
		institutionTableWithColumns("school_utility_meters", "Utility meter inventory.", "tenant_code", "point_id", "serial_number", "expected_version"),
		institutionTableWithColumns("school_utility_readings", "Immutable dated meter readings.", "tenant_code", "meter_id", "reading_on", "value", "policy_evaluation_id", "policy_evaluation_v2_id"),
		institutionTableWithColumns("school_utility_invoices", "Utility invoice reconciliation evidence.", "tenant_code", "point_id", "invoice_number", "reconciliation_status", "expected_version", "policy_evaluation_id", "policy_evaluation_v2_id"),
		institutionTableWithColumns("school_compliance_obligations", "SSM, PSI and security statutory obligations.", "tenant_code", "domain", "code", "status", "expected_version", "policy_evaluation_id", "policy_evaluation_v2_id"),
		institutionTableWithColumns("school_compliance_inspections", "Compliance inspections linked to obligations.", "tenant_code", "obligation_id", "inspected_on", "outcome", "policy_evaluation_id", "policy_evaluation_v2_id"),
		institutionTableWithColumns("school_compliance_corrective_actions", "Auditable corrective actions from inspections.", "tenant_code", "inspection_id", "status", "expected_version", "policy_evaluation_id", "policy_evaluation_v2_id"),
		institutionTableWithColumns("school_operations_outbox", "Durable archive/workflow delivery intents for operational records.", "tenant_code", "aggregate_type", "aggregate_id", "event_type", "status", "payload"),
		tenantTable("app_org_units", "Organization units are tenant-scoped."),
		tenantTable("app_memberships", "Memberships are tenant-scoped."),
		tenantTable("app_tenant_authorization_versions", "Authoritative tenant authorization version for session and token invalidation."),
		institutionTable("registratura_documents", "Incoming/outgoing registry documents."),
		institutionTable("registre", "Tenant-scoped registries and numbering."),
		institutionTable("registratura_departments", "Registratura department structure."),
		institutionTable("registratura_organizations", "Registratura organizations."),
		institutionTable("registratura_user_departments", "Tenant user department assignments."),
		institutionTable("registratura_user_organizations", "Tenant user organization assignments."),
		institutionTable("registratura_registry_departments", "Registry department access."),
		institutionTable("registratura_organization_departments", "Organization department memberships."),
		institutionTable("registratura_document_departments", "Document department assignments."),
		institutionTable("registratura_document_workflow_events", "Immutable document workflow history."),
		institutionTable("registratura_document_versions", "Immutable document version history."),
		institutionTable("registratura_document_attachments", "Tenant-scoped document attachment metadata."),
		institutionTableWithColumns("registratura_archive_outbox", "Durable tenant-scoped Registratura to eArhiva delivery intents.", "document_id", "event_type", "status", "attempts", "available_at", "locked_at", "locked_by", "delivered_at", "dead_lettered_at", "last_error"),
		institutionTable("app_parties", "Physical persons, legal entities and institutions used by registratura."),
		institutionTable("archive_records", "Electronic archive records."),
		institutionTable("archive_documents", "Independent archive document registry."),
		institutionTableWithColumns("archive_document_versions", "Immutable archive document versions.", "source_bucket", "source_object_key", "source_object_version_id", "source_object_etag", "source_sha256", "source_size_bytes", "retention_until", "legal_hold_active", "artifact_bucket", "artifact_object_key", "page_count", "text_status", "extracted_text", "extracted_metadata", "search_embedding", "created_by", "updated_at"),
		institutionTableWithColumns("archive_document_chunks", "Searchable text chunks for archive documents.", "version_id", "chunk_no", "page_no", "content", "content_tsv"),
		institutionTableWithColumns("archive_document_entities", "Extracted archive document entities.", "document_id", "version_id", "entity_type", "entity_value", "normalized_value", "confidence", "chunk_no", "page_no"),
		institutionTableWithColumns("archive_document_relations", "Archive document relations.", "source_document_id", "relation_type", "relation_value", "confidence", "metadata"),
		institutionTable("archive_taxonomy_nodes", "Archive taxonomy nodes."),
		institutionTableWithColumns("archive_ingestion_jobs", "Archive document ingestion jobs.", "document_id", "version_id", "job_type", "status", "available_at", "attempts", "locked_at", "locked_by", "last_error", "created_by", "updated_at"),
		institutionTableWithColumns("archive_document_classification_reviews", "Human review state for OCR-derived archive classifications.", "document_id", "version_id", "state", "revision", "suggestion", "suggestion_confidence", "suggestion_source", "requires_human_review", "final_classification", "reviewed_by"),
		institutionTableWithColumns("app_audit_log", "Append-only tenant audit trail; unattributed legacy rows remain invisible.", "actor_subject", "action", "target_type", "target_id", "status", "details", "created_at"),
		institutionTableWithColumnsAndPolicies("app_entity_versions", "Append-only tenant- and institution-scoped entity snapshots; unscoped legacy rows remain invisible.", []string{"app_entity_versions_tenant_read", "app_entity_versions_tenant_append"}, "tenant_code", "entity_table", "entity_id", "version_no", "change_type", "snapshot", "changed_by", "changed_at"),
		institutionTable("workflow_instances", "Runtime workflow instances."),
		institutionTableWithColumns("education_meetings", "Governance meetings with immutable actor identities.", "chairperson_user_id", "secretary_user_id"),
		institutionTable("education_personnel", "Personnel master data."),
		institutionTableWithColumnsAndPolicies("education_school_classes", "Tenant- and institution-scoped school classes with lifecycle history.", []string{"education_classes_read", "education_classes_manage"}, "tenant_code", "class_code", "class_name", "school_year", "active"),
		institutionTableWithColumnsAndPolicies("education_students", "Tenant- and institution-scoped student directory.", []string{"education_students_read", "education_students_manage"}, "tenant_code", "student_code", "first_name", "last_name", "party_id", "status"),
		institutionTableWithColumnsAndPolicies("education_student_enrolments", "Temporal class enrolments retained as school evidence.", []string{"education_student_enrolments_read", "education_student_enrolments_manage"}, "tenant_code", "student_id", "class_id", "admission_application_id", "enrolled_from", "status"),
		institutionTableWithColumnsAndPolicies("education_class_homeroom_assignments", "Temporal homeroom assignments bound to canonical personnel identity.", []string{"education_homeroom_assignments_read", "education_homeroom_assignments_manage"}, "tenant_code", "class_id", "personnel_id", "app_user_id", "assigned_from"),
		institutionTableWithColumns("education_portfolios", "Personnel portfolio records with auditable lifecycle safety.", "activity_ceased_on", "retention_period_days", "legal_hold_active", "withdrawn_at", "withdrawal_reason"),
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
		institutionTableWithColumns("education_portfolio_documents", "Portfolio documents retained with append-audit lifecycle provenance and immutable eArhiva evidence.", "status", "withdrawn_at", "withdrawn_by_subject", "withdrawal_reason", "description", "school_year", "subject_discipline", "applicable_class", "competencies", "archive_document_id", "archive_version_id", "archive_version_no", "archive_source_bucket", "archive_source_object_key", "archive_sha256", "last_change_reason"),
		institutionTable("education_meeting_votes", "Meeting votes."),
		institutionTable("education_portfolio_checklist", "Portfolio checklist items."),
		institutionTableWithColumns("education_governance_memberships", "Governance memberships with immutable user identity.", "app_user_id"),
		institutionTable("education_meeting_resolutions", "Meeting resolutions."),
		institutionTableWithColumns("education_portfolio_transfers", "Portfolio handovers retained as lifecycle evidence.", "withdrawn_at", "withdrawal_reason"),
		institutionTableWithColumns("education_portfolio_valorifications", "Portfolio valorification records retained as lifecycle evidence.", "withdrawn_at", "withdrawal_reason"),
		institutionTable("education_committees", "Institution committees and their appointment lifecycle."),
		institutionTable("education_committee_members", "Institution committee membership assignments."),
		institutionTable("education_portfolio_reviews", "Portfolio reviews."),
		institutionTable("education_meeting_minutes", "Meeting minutes."),
		institutionTable("education_portfolio_opis", "Portfolio inventory list."),
		institutionTable("education_portfolio_custody", "Portfolio custody history."),
		institutionTableWithColumnsAndPolicies("education_portfolio_archive_attachment_grants", "Narrow institution-scoped grants for attaching eArhiva evidence to a professional portfolio.", []string{"education_portfolio_archive_attachment_grants_tenant_isolation"}, "archive_document_id", "grantee_user_id", "granted_by_user_id", "created_at"),
		institutionTableWithColumns("education_role_delegations", "Tenant- and institution-scoped director-to-adjunct delegated authority with immutable provenance.", "tenant_code", "delegator_user_id", "delegate_user_id", "permission_code", "resource_type", "resource_id", "status", "valid_from", "valid_until", "offered_by_user_id", "offered_at"),
		institutionTableWithColumns("education_portfolio_procedure_versions", "Tenant- and institution-scoped, versioned portfolio procedures.", "tenant_code", "procedure_code", "version_no", "lifecycle_status"),
		institutionTableWithColumns("education_portfolio_procedure_section_rules", "Tenant- and institution-scoped rules for a versioned portfolio procedure.", "tenant_code", "procedure_id", "section_code"),
		institutionTableWithColumns("education_portfolio_declaration_acknowledgements", "Tenant- and institution-scoped immutable portfolio declaration acknowledgements.", "tenant_code", "portfolio_id", "declaration_type", "declaration_version", "accepted_at"),
		institutionTableWithColumns("education_portfolio_export_manifests", "Tenant- and institution-scoped immutable portfolio evidence export manifests.", "tenant_code", "portfolio_id", "manifest_version", "manifest_sha256", "manifest", "generated_at"),
		institutionTableWithColumns("education_portfolio_valorification_packages", "Tenant- and institution-scoped evidence packages connecting a portfolio to evaluation, mobility, or merit proceedings.", "tenant_code", "portfolio_id", "scope", "status", "created_by_subject", "created_at"),
		institutionTableWithColumns("education_portfolio_valorification_package_documents", "Immutable archive-version snapshots attached to portfolio valorification packages.", "package_id", "archive_document_id", "archive_version_id", "archive_version_no", "archive_source_bucket", "archive_source_object_key", "archive_sha256", "created_by_subject", "created_at"),
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
		institutionTableWithColumns("education_signed_artifact_evidence", "Append-only cryptographic signature evidence for official school artifacts.", "tenant_code", "artifact_type", "artifact_id", "document_sha256", "expected_canonical_legal_payload_sha256", "expected_actor_subject", "signature_format", "signature_level", "submitted_by_subject", "submitted_at"),
		institutionTableWithColumns("education_signed_artifact_validations", "Append-only DSS trust validation history for signed school artifacts.", "tenant_code", "evidence_id", "validation_status", "validated_at", "trusted_list_provider", "validator_provider", "validator_version", "validation_policy", "observed_sha256", "observed_size_bytes", "signed_payload_sha256", "certificate_sha256", "signed_actor_subject", "diagnostic_data", "detailed_report", "simple_report", "etsi_validation_report"),
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

// ValidateRuntimeDatabaseRole prevents a production runtime connection from
// silently bypassing FORCE RLS. Application-level session settings cannot
// constrain a PostgreSQL SUPERUSER or a role carrying BYPASSRLS.
func ValidateRuntimeDatabaseRole(ctx context.Context, pool *pgxpool.Pool) error {
	var role string
	var superuser, bypassRLS bool
	if err := pool.QueryRow(ctx, `
		select current_user, rolsuper, rolbypassrls
		from pg_catalog.pg_roles
		where rolname = current_user
	`).Scan(&role, &superuser, &bypassRLS); err != nil {
		return fmt.Errorf("inspect runtime database role: %w", err)
	}
	return validateRuntimeDatabaseRoleFlags(role, superuser, bypassRLS)
}

func validateRuntimeDatabaseRoleFlags(role string, superuser, bypassRLS bool) error {
	if strings.TrimSpace(role) == "" {
		return fmt.Errorf("runtime database role is empty")
	}
	if superuser || bypassRLS {
		return fmt.Errorf("runtime database role %q must be NOSUPERUSER and NOBYPASSRLS", role)
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

	policyNames := table.RequiredPolicies
	if len(policyNames) == 0 {
		policyNames = []string{"tenant_isolation"}
	}
	for _, policyName := range policyNames {
		var policyExists bool
		if err := pool.QueryRow(ctx, `
			select exists (
				select 1
				from pg_policies
				where schemaname = 'public'
					and tablename = $1
					and policyname = $2
			)
		`, table.Name, policyName).Scan(&policyExists); err != nil {
			return fmt.Errorf("check row level security policy for %s: %w", table.Name, err)
		}
		if !policyExists {
			return fmt.Errorf("schema contract violation: table %s is missing %s policy", table.Name, policyName)
		}
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
