-- Additive bridge from the legacy school policy evidence model to the Stage 1B
-- normalized model.  Runtime remains on the legacy authorization path until a
-- later migration has backfilled, shadow-compared and explicitly cut over each
-- institution.  Nothing in this migration deletes or reinterprets v1 evidence.
select set_config('app.is_super_admin', 'true', true);

-- Legacy institutions created by 0135 were deliberately unclassified.  Make
-- that state representable in v2 only as a controlled migration state; every
-- usable v2 profile still has an exact public/private form and effective date.
alter table school_institution_profiles_v2
	drop constraint if exists school_institution_profiles_v2_status_check;
alter table school_institution_profiles_v2
	drop constraint if exists school_institution_profiles_v2_legal_form_check;
alter table school_institution_profiles_v2
	drop constraint if exists school_profiles_v2_classification;
alter table school_institution_profiles_v2 alter column legal_form drop not null;
alter table school_institution_profiles_v2 alter column effective_from drop not null;
alter table school_institution_profiles_v2
	add constraint school_profiles_v2_status_check
	check (status in ('unclassified', 'draft', 'approved', 'active', 'superseded'));
alter table school_institution_profiles_v2
	add constraint school_profiles_v2_legal_form_check
	check (legal_form is null or legal_form in ('public', 'private'));
alter table school_institution_profiles_v2
	add constraint school_profiles_v2_classification
	check (
		(status = 'unclassified' and legal_form is null and effective_from is null)
		or (status <> 'unclassified' and legal_form in ('public', 'private') and effective_from is not null)
	);

alter table school_policy_overrides
	add constraint school_policy_overrides_scope_id_unique unique (tenant_code, institution_id, id);

-- Stable public API identity avoids exposing the physical v2 row id/version
-- while both policy engines have to coexist.  It is append-only evidence.
create table school_profile_cutover_identity (
	id uuid not null default gen_random_uuid(),
	tenant_code text not null,
	institution_id text not null,
	profile_v2_id uuid not null,
	profile_v2_version integer not null,
	api_profile_id uuid not null,
	api_version integer not null check (api_version > 0),
	legacy_profile_id uuid,
	legacy_profile_version integer,
	mapping_kind text not null check (mapping_kind in ('legacy_import', 'dual_write', 'v2_native')),
	created_by_subject text not null,
	created_at timestamptz not null default now(),
	primary key (tenant_code, institution_id, profile_v2_id, profile_v2_version),
	unique (tenant_code, institution_id, id),
	unique (tenant_code, institution_id, api_profile_id),
	unique (tenant_code, institution_id, api_version),
	unique (tenant_code, institution_id, legacy_profile_id, legacy_profile_version),
	foreign key (tenant_code, institution_id) references app_tenants(code, institution_id) on delete restrict,
	foreign key (tenant_code, institution_id, profile_v2_id, profile_v2_version) references school_institution_profiles_v2(tenant_code, institution_id, id, version) on delete restrict,
	foreign key (tenant_code, institution_id, legacy_profile_id, legacy_profile_version) references school_institution_profiles(tenant_code, institution_id, id, version) on delete restrict,
	check ((legacy_profile_id is null and legacy_profile_version is null) or (legacy_profile_id is not null and legacy_profile_version is not null))
);

-- Compatibility-only projection for public contracts during the bridge.  It
-- is immutable per exact profile version; corrections require a new profile.
create table school_institution_profile_api_projection (
	id uuid not null default gen_random_uuid(),
	tenant_code text not null,
	institution_id text not null,
	profile_v2_id uuid not null,
	profile_v2_version integer not null,
	regulatory_profile text not null default '',
	authorization_status text not null default 'unknown',
	authorized_levels text[] not null default '{}',
	accreditation_reference text not null default '',
	program_codes text[] not null default '{}',
	founder_name text not null default '',
	funder_name text not null default '',
	budget_authority_name text not null default '',
	is_contracting_authority boolean not null default false,
	has_legal_personality boolean not null default true,
	tax_identifier text not null default '',
	accounting_profile text not null default '',
	procurement_profile text not null default '',
	payroll_profile text not null default '',
	vat_profile text not null default 'not_registered',
	treasury_required boolean not null default false,
	public_funding boolean not null default false,
	source_reference text not null default '',
	updated_by_subject text not null,
	updated_at timestamptz not null default now(),
	primary key (tenant_code, institution_id, profile_v2_id, profile_v2_version),
	unique (tenant_code, institution_id, id),
	foreign key (tenant_code, institution_id) references app_tenants(code, institution_id) on delete restrict,
	foreign key (tenant_code, institution_id, profile_v2_id, profile_v2_version) references school_institution_profiles_v2(tenant_code, institution_id, id, version) on delete restrict,
	check (authorization_status in ('unknown', 'provisional', 'authorized', 'accredited', 'suspended', 'withdrawn'))
);

create unique index school_institution_profiles_v2_scope_id_version_series_unique
	on school_institution_profiles_v2(tenant_code, institution_id, id, version, profile_series_id);
create unique index school_policy_pack_versions_scope_id_code_version_unique
	on school_policy_pack_versions(tenant_code, institution_id, id, pack_code, version);

create table school_operation_policy_bindings_v2 (
	id uuid primary key default gen_random_uuid(),
	tenant_code text not null,
	institution_id text not null,
	profile_v2_id uuid not null,
	profile_v2_version integer not null,
	profile_series_id uuid not null,
	policy_pack_version_id uuid not null,
	pack_code text not null,
	policy_pack_version integer not null check (policy_pack_version > 0),
	assignment_kind text not null check (assignment_kind in ('common', 'legal_form', 'funding', 'program', 'institution', 'confessional')),
	status text not null check (status in ('draft', 'active', 'superseded', 'revoked')),
	effective_from date not null,
	effective_to date,
	expected_version integer not null default 1 check (expected_version > 0),
	legacy_assignment_id uuid,
	created_by_subject text not null,
	created_at timestamptz not null default now(),
	updated_by_subject text not null,
	updated_at timestamptz not null default now(),
	unique (tenant_code, institution_id, id),
	unique (tenant_code, institution_id, legacy_assignment_id),
	foreign key (tenant_code, institution_id) references app_tenants(code, institution_id) on delete restrict,
	foreign key (tenant_code, institution_id, profile_v2_id, profile_v2_version, profile_series_id) references school_institution_profiles_v2(tenant_code, institution_id, id, version, profile_series_id) on delete restrict,
	foreign key (tenant_code, institution_id, policy_pack_version_id, pack_code, policy_pack_version) references school_policy_pack_versions(tenant_code, institution_id, id, pack_code, version) on delete restrict,
	foreign key (tenant_code, institution_id, legacy_assignment_id) references school_policy_assignments(tenant_code, institution_id, id) on delete restrict,
	check (effective_to is null or effective_to >= effective_from)
);
create unique index school_operation_policy_bindings_v2_one_active_pack
	on school_operation_policy_bindings_v2(tenant_code, institution_id, profile_v2_id, pack_code)
	where status = 'active';
create index school_operation_policy_bindings_v2_pack_idx
	on school_operation_policy_bindings_v2(tenant_code, institution_id, policy_pack_version_id);
create index school_operation_policy_bindings_v2_profile_idx
	on school_operation_policy_bindings_v2(tenant_code, institution_id, profile_v2_id, profile_v2_version);
create index school_operation_policy_bindings_v2_series_idx
	on school_operation_policy_bindings_v2(tenant_code, institution_id, profile_series_id);
create unique index school_operation_policy_bindings_v2_scope_id_series_pack_unique
	on school_operation_policy_bindings_v2(tenant_code, institution_id, id, profile_series_id, policy_pack_version_id);
create unique index school_operation_policy_bindings_v2_exact_profile_pack_unique
	on school_operation_policy_bindings_v2(tenant_code, institution_id, id, profile_v2_id, profile_v2_version, profile_series_id, policy_pack_version_id);
alter table school_operation_policy_bindings_v2
	add constraint school_operation_policy_bindings_v2_active_effective_excl
	exclude using gist (
		tenant_code with =,
		institution_id with =,
		profile_series_id with =,
		pack_code with =,
		daterange(effective_from, coalesce(effective_to, 'infinity'::date), '[]') with &&
	) where (status = 'active');

create table school_operation_policy_overrides_v2 (
	id uuid primary key default gen_random_uuid(),
	tenant_code text not null,
	institution_id text not null,
	binding_id uuid not null,
	key text not null,
	value jsonb not null,
	justification text not null,
	status text not null check (status in ('draft', 'approved', 'revoked')),
	expected_version integer not null default 1 check (expected_version > 0),
	legacy_override_id uuid,
	approved_by_subject text not null default '',
	approved_at timestamptz,
	created_by_subject text not null,
	created_at timestamptz not null default now(),
	updated_by_subject text not null,
	updated_at timestamptz not null default now(),
	unique (tenant_code, institution_id, id),
	unique (tenant_code, institution_id, binding_id, key),
	unique (tenant_code, institution_id, legacy_override_id),
	foreign key (tenant_code, institution_id) references app_tenants(code, institution_id) on delete restrict,
	foreign key (tenant_code, institution_id, binding_id) references school_operation_policy_bindings_v2(tenant_code, institution_id, id) on delete restrict,
	foreign key (tenant_code, institution_id, legacy_override_id) references school_policy_overrides(tenant_code, institution_id, id) on delete restrict,
	check (jsonb_typeof(value) in ('object', 'array', 'string', 'number', 'boolean', 'null')),
	check (status <> 'approved' or (approved_by_subject <> '' and approved_at is not null))
);

-- Exact rule/override evidence used for each immutable command input.  This is
-- deliberately separate from mutable assignments, so a later change cannot
-- alter the legal context of a past decision.
create unique index school_operation_policy_inputs_scope_id_series_unique
	on school_operation_policy_inputs(tenant_code, institution_id, id, profile_series_id);
create unique index school_operation_policy_inputs_exact_profile_unique
	on school_operation_policy_inputs(tenant_code, institution_id, id, profile_id, profile_version, profile_series_id);
create unique index school_policy_pack_versions_scope_id_checksum_unique
	on school_policy_pack_versions(tenant_code, institution_id, id, checksum_sha256);
create table school_operation_policy_input_bindings (
	id uuid not null default gen_random_uuid(),
	tenant_code text not null,
	institution_id text not null,
	input_id uuid not null,
	binding_id uuid not null,
	profile_v2_id uuid not null,
	profile_v2_version integer not null,
	profile_series_id uuid not null,
	policy_pack_version_id uuid not null,
	policy_pack_checksum_sha256 text not null check (policy_pack_checksum_sha256 ~ '^[a-f0-9]{64}$'),
	override_snapshot jsonb not null default '[]'::jsonb,
	override_checksum_sha256 text not null check (override_checksum_sha256 ~ '^[a-f0-9]{64}$'),
	created_by_subject text not null,
	created_at timestamptz not null default now(),
	primary key (tenant_code, institution_id, input_id, binding_id),
	unique (tenant_code, institution_id, id),
	foreign key (tenant_code, institution_id) references app_tenants(code, institution_id) on delete restrict,
	foreign key (tenant_code, institution_id, input_id, profile_v2_id, profile_v2_version, profile_series_id) references school_operation_policy_inputs(tenant_code, institution_id, id, profile_id, profile_version, profile_series_id) on delete restrict,
	foreign key (tenant_code, institution_id, binding_id, profile_v2_id, profile_v2_version, profile_series_id, policy_pack_version_id) references school_operation_policy_bindings_v2(tenant_code, institution_id, id, profile_v2_id, profile_v2_version, profile_series_id, policy_pack_version_id) on delete restrict,
	foreign key (tenant_code, institution_id, policy_pack_version_id, policy_pack_checksum_sha256) references school_policy_pack_versions(tenant_code, institution_id, id, checksum_sha256) on delete restrict,
	check (jsonb_typeof(override_snapshot) = 'array')
);
create index school_operation_policy_input_bindings_binding_idx
	on school_operation_policy_input_bindings(tenant_code, institution_id, binding_id);
create index school_operation_policy_input_bindings_pack_idx
	on school_operation_policy_input_bindings(tenant_code, institution_id, policy_pack_version_id);

alter table school_operation_policy_inputs add column effective_on date;
alter table school_operation_policy_inputs add column decision_kind text not null default 'operation' check (decision_kind in ('operation', 'publication', 'contract', 'compliance', 'migration', 'legacy_import'));
alter table school_operation_policy_inputs add column engine_version text not null default 'v2-bridge';
alter table school_operation_policy_inputs add column schema_version integer not null default 1 check (schema_version > 0);

alter table school_operation_policy_evaluations_v2 add column effective_on date;
alter table school_operation_policy_evaluations_v2 add column decision_kind text not null default 'operation' check (decision_kind in ('operation', 'publication', 'contract', 'compliance', 'migration', 'legacy_import'));
alter table school_operation_policy_evaluations_v2 add column engine_version text not null default 'v2-bridge';
alter table school_operation_policy_evaluations_v2 add column schema_version integer not null default 1 check (schema_version > 0);
create unique index school_operation_policy_evaluations_v2_scope_id_input_unique
	on school_operation_policy_evaluations_v2(tenant_code, institution_id, id, input_id);

-- A v1 evaluation can have at most one imported/shadow v2 decision.  The map
-- gives old consumers an auditable bridge instead of silently changing FKs.
create table school_policy_evaluation_cutover_identity (
	id uuid not null default gen_random_uuid(),
	tenant_code text not null,
	institution_id text not null,
	legacy_evaluation_id uuid not null,
	v2_evaluation_id uuid not null,
	input_id uuid not null,
	conversion_kind text not null check (conversion_kind in ('historical_import', 'shadow_compare', 'dual_write')),
	created_by_subject text not null,
	created_at timestamptz not null default now(),
	primary key (tenant_code, institution_id, legacy_evaluation_id),
	unique (tenant_code, institution_id, id),
	unique (tenant_code, institution_id, v2_evaluation_id),
	foreign key (tenant_code, institution_id) references app_tenants(code, institution_id) on delete restrict,
	foreign key (tenant_code, institution_id, legacy_evaluation_id) references school_policy_evaluations(tenant_code, institution_id, id) on delete restrict,
	foreign key (tenant_code, institution_id, v2_evaluation_id, input_id) references school_operation_policy_evaluations_v2(tenant_code, institution_id, id, input_id) on delete restrict
);
create index school_policy_evaluation_cutover_identity_input_idx
	on school_policy_evaluation_cutover_identity(tenant_code, institution_id, input_id);

create table school_policy_cutover_state (
	id uuid not null default gen_random_uuid(),
	tenant_code text not null,
	institution_id text not null,
	phase text not null check (phase in ('legacy', 'dual', 'v2', 'contracted')),
	expected_version integer not null default 1 check (expected_version > 0),
	shadow_compared_at timestamptz,
	cutover_authorized_by_subject text not null default '',
	cutover_authorized_at timestamptz,
	updated_by_subject text not null,
	updated_at timestamptz not null default now(),
	primary key (tenant_code, institution_id),
	unique (tenant_code, institution_id, id),
	foreign key (tenant_code, institution_id) references app_tenants(code, institution_id) on delete restrict,
	check (phase not in ('v2', 'contracted') or (shadow_compared_at is not null and cutover_authorized_by_subject <> '' and cutover_authorized_at is not null))
);
insert into school_policy_cutover_state(tenant_code, institution_id, phase, updated_by_subject)
select code, institution_id, 'legacy', 'migration:0145'
from app_tenants
on conflict (tenant_code, institution_id) do nothing;

create table school_regulatory_migration_issues (
	id uuid primary key default gen_random_uuid(),
	tenant_code text not null,
	institution_id text not null,
	entity_type text not null,
	entity_id uuid,
	issue_code text not null,
	severity text not null check (severity in ('info', 'warning', 'blocking')),
	status text not null default 'open' check (status in ('open', 'resolved', 'waived')),
	details jsonb not null default '{}'::jsonb,
	resolved_by_subject text not null default '',
	resolved_at timestamptz,
	created_by_subject text not null,
	created_at timestamptz not null default now(),
	updated_by_subject text not null,
	updated_at timestamptz not null default now(),
	unique (tenant_code, institution_id, id),
	foreign key (tenant_code, institution_id) references app_tenants(code, institution_id) on delete restrict,
	check (jsonb_typeof(details) = 'object'),
	check (status = 'open' or (resolved_by_subject <> '' and resolved_at is not null))
);

-- Consumers retain their v1 FK during the bridge and receive an optional v2
-- provenance FK.  NOT VALID protects new writes without assuming historical
-- data has already been backfilled.
alter table education_publications add column policy_evaluation_v2_id uuid;
alter table education_publications add constraint education_publications_policy_evaluation_v2_fk foreign key (tenant_code, institution_id, policy_evaluation_v2_id) references school_operation_policy_evaluations_v2(tenant_code, institution_id, id) on delete restrict not valid;
alter table education_publications add constraint education_publications_policy_evaluation_bridge_check check (policy_evaluation_id is not null or policy_evaluation_v2_id is not null) not valid;
create index education_publications_policy_evaluation_v2_idx on education_publications(tenant_code, institution_id, policy_evaluation_v2_id);

do $$
declare t text;
begin
	foreach t in array array[
		'school_contracts', 'school_contract_obligations', 'school_utility_points',
		'school_utility_readings', 'school_utility_invoices', 'school_compliance_obligations',
		'school_compliance_inspections', 'school_compliance_corrective_actions'
	]
	loop
		execute format('alter table %I add column policy_evaluation_v2_id uuid', t);
		execute format('alter table %I add constraint %I foreign key (tenant_code, institution_id, policy_evaluation_v2_id) references school_operation_policy_evaluations_v2(tenant_code, institution_id, id) on delete restrict not valid', t, t || '_policy_evaluation_v2_fk');
		execute format('alter table %I add constraint %I check (policy_evaluation_id is not null or policy_evaluation_v2_id is not null) not valid', t, t || '_policy_evaluation_bridge_check');
		execute format('create index %I on %I(tenant_code, institution_id, policy_evaluation_v2_id)', t || '_policy_evaluation_v2_idx', t);
	end loop;
end $$;

-- New bridge tables are fully tenant/institution scoped.  Evidence mappings
-- and input snapshots are immutable; administrative state/issue records are
-- no-hard-delete so their audit trail cannot be erased.
do $$
declare t text;
begin
	foreach t in array array[
		'school_profile_cutover_identity', 'school_institution_profile_api_projection',
		'school_operation_policy_bindings_v2', 'school_operation_policy_overrides_v2',
		'school_operation_policy_input_bindings', 'school_policy_evaluation_cutover_identity',
		'school_policy_cutover_state', 'school_regulatory_migration_issues'
	]
	loop
		execute format('alter table %I enable row level security', t);
		execute format('alter table %I force row level security', t);
		execute format('create policy tenant_isolation on %I using (public.can_bypass_tenant_rls() or (tenant_code = public.current_tenant_code() and institution_id = public.current_institution_id())) with check (public.can_bypass_tenant_rls() or (tenant_code = public.current_tenant_code() and institution_id = public.current_institution_id()))', t);
	end loop;
end $$;

create trigger school_profile_cutover_identity_immutable before update or delete on school_profile_cutover_identity for each row execute function public.school_stage1b_snapshot_immutable();
create trigger school_profile_api_projection_immutable before update or delete on school_institution_profile_api_projection for each row execute function public.school_stage1b_snapshot_immutable();
create trigger school_operation_policy_input_bindings_immutable before update or delete on school_operation_policy_input_bindings for each row execute function public.school_stage1b_snapshot_immutable();
create trigger school_policy_evaluation_cutover_identity_immutable before update or delete on school_policy_evaluation_cutover_identity for each row execute function public.school_stage1b_snapshot_immutable();
create trigger school_profile_api_projection_no_hard_delete before delete on school_institution_profile_api_projection for each row execute function public.school_operations_no_hard_delete();
create trigger school_policy_bindings_v2_no_hard_delete before delete on school_operation_policy_bindings_v2 for each row execute function public.school_operations_no_hard_delete();
create trigger school_policy_overrides_v2_no_hard_delete before delete on school_operation_policy_overrides_v2 for each row execute function public.school_operations_no_hard_delete();
create trigger school_policy_cutover_state_no_hard_delete before delete on school_policy_cutover_state for each row execute function public.school_operations_no_hard_delete();
create trigger school_regulatory_migration_issues_no_hard_delete before delete on school_regulatory_migration_issues for each row execute function public.school_operations_no_hard_delete();

-- Every mutable bridge record retains the standard append-only entity audit.
-- Immutable mappings/snapshots are recorded on insert only.
create trigger school_profile_cutover_identity_versioning after insert on school_profile_cutover_identity for each row execute function public.record_entity_version();
create trigger school_profile_api_projection_versioning after insert on school_institution_profile_api_projection for each row execute function public.record_entity_version();
create trigger school_policy_bindings_v2_versioning after insert or update or delete on school_operation_policy_bindings_v2 for each row execute function public.record_entity_version();
create trigger school_policy_overrides_v2_versioning after insert or update or delete on school_operation_policy_overrides_v2 for each row execute function public.record_entity_version();
create trigger school_policy_input_bindings_versioning after insert on school_operation_policy_input_bindings for each row execute function public.record_entity_version();
create trigger school_policy_evaluation_cutover_identity_versioning after insert on school_policy_evaluation_cutover_identity for each row execute function public.record_entity_version();
create trigger school_policy_cutover_state_versioning after insert or update or delete on school_policy_cutover_state for each row execute function public.record_entity_version();
create trigger school_regulatory_migration_issues_versioning after insert or update or delete on school_regulatory_migration_issues for each row execute function public.record_entity_version();
