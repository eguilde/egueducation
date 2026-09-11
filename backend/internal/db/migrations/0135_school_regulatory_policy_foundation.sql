-- Public/private/confessional school policy foundation. Scope is always
-- derived from the host-bound request session; no regulated profile is
-- inferred from a tenant name or existing school data.
select set_config('app.is_super_admin', 'true', true);

insert into app_permissions(code, label) values
	('institution.regulatory_profile.read', 'Read the active institution regulatory profile'),
	('institution.regulatory_profile.manage', 'Classify and version the institution regulatory profile')
on conflict (code) do update set label = excluded.label;

insert into app_role_permissions(role_code, permission_code)
select role_code, permission_code from (values
	('super_admin', 'institution.regulatory_profile.read'),
	('super_admin', 'institution.regulatory_profile.manage'),
	('admin', 'institution.regulatory_profile.read'),
	('admin', 'institution.regulatory_profile.manage'),
	('director', 'institution.regulatory_profile.read')
) as permission_grants(role_code, permission_code)
on conflict do nothing;

create table school_institution_profiles (
	id uuid primary key default gen_random_uuid(),
	tenant_code text not null,
	institution_id text not null,
	version integer not null check (version > 0),
	status text not null check (status in ('unclassified', 'draft', 'approved', 'active', 'superseded')),
	school_legal_form text check (school_legal_form in ('public', 'private', 'confessional')),
	regulatory_profile text not null default '',
	authorization_status text not null default 'unknown' check (authorization_status in ('unknown', 'provisional', 'authorized', 'accredited', 'suspended', 'withdrawn')),
	accreditation_reference text not null default '',
	authorized_levels text[] not null default '{}',
	has_legal_personality boolean not null default true,
	tax_identifier text not null default '',
	founder_name text not null default '',
	funder_name text not null default '',
	budget_authority_name text not null default '',
	is_contracting_authority boolean not null default false,
	accounting_profile text not null default '',
	procurement_profile text not null default '',
	payroll_profile text not null default '',
	vat_profile text not null default 'not_registered',
	treasury_required boolean not null default false,
	public_funding boolean not null default false,
	program_codes text[] not null default '{}',
	effective_from date,
	effective_to date,
	source_reference text not null default '',
	approved_by_subject text not null default '',
	approved_at timestamptz,
	created_by_subject text not null,
	created_at timestamptz not null default now(),
	updated_by_subject text not null,
	updated_at timestamptz not null default now(),
	constraint school_profiles_tenant_fk foreign key (tenant_code, institution_id) references app_tenants(code, institution_id) on delete restrict,
	constraint school_profiles_scope_id_unique unique (tenant_code, institution_id, id),
	constraint school_profiles_scope_id_version_unique unique (tenant_code, institution_id, id, version),
	constraint school_profiles_scope_version_unique unique (tenant_code, institution_id, version),
	constraint school_profiles_effective_window check (effective_to is null or effective_from is null or effective_to >= effective_from),
	constraint school_profiles_classification check (
		(status = 'unclassified' and school_legal_form is null)
		or (status <> 'unclassified' and school_legal_form is not null and regulatory_profile <> '')
	),
	constraint school_profiles_approval check (
		(status not in ('approved', 'active'))
		or (approved_by_subject <> '' and approved_at is not null and effective_from is not null)
	)
);
create index school_profiles_scope_status
	on school_institution_profiles(tenant_code, institution_id, status, effective_from desc);
alter table school_institution_profiles
	add constraint school_profiles_effective_period_excl
	exclude using gist (
		tenant_code with =,
		institution_id with =,
		daterange(effective_from, coalesce(effective_to, 'infinity'::date), '[]') with &&
	) where (status in ('approved', 'active'));

create table school_policy_pack_versions (
	id uuid primary key default gen_random_uuid(),
	tenant_code text not null,
	institution_id text not null,
	pack_code text not null,
	jurisdiction text not null default 'RO',
	regulatory_profile text not null,
	version integer not null check (version > 0),
	status text not null check (status in ('draft', 'approved', 'withdrawn')),
	effective_from date not null,
	effective_to date,
	rules jsonb not null,
	rules_schema jsonb not null default '{"type":"object","required":["capabilities"],"additionalProperties":false,"properties":{"capabilities":{"type":"array"}}}',
	configurable_keys text[] not null default '{}',
	source_references text[] not null default '{}',
	checksum_sha256 text not null check (checksum_sha256 ~ '^[a-f0-9]{64}$'),
	approved_by_subject text not null default '',
	approved_at timestamptz,
	created_by_subject text not null,
	created_at timestamptz not null default now(),
	updated_by_subject text not null,
	updated_at timestamptz not null default now(),
	constraint school_policy_packs_tenant_fk foreign key (tenant_code, institution_id) references app_tenants(code, institution_id) on delete restrict,
	constraint school_policy_packs_scope_id_unique unique (tenant_code, institution_id, id),
	constraint school_policy_packs_version_unique unique (tenant_code, institution_id, pack_code, version),
	constraint school_policy_packs_window check (effective_to is null or effective_to >= effective_from),
	constraint school_policy_packs_approval check (status <> 'approved' or (approved_by_subject <> '' and approved_at is not null)),
	constraint school_policy_packs_rules_object check (
		jsonb_typeof(rules) = 'object'
		and jsonb_typeof(rules -> 'capabilities') = 'array'
	),
	constraint school_policy_packs_rules_schema_object check (jsonb_typeof(rules_schema) = 'object')
);
create index school_policy_packs_effective
	on school_policy_pack_versions(tenant_code, institution_id, pack_code, status, effective_from, effective_to);

create table school_policy_assignments (
	id uuid primary key default gen_random_uuid(),
	tenant_code text not null,
	institution_id text not null,
	policy_pack_version_id uuid not null,
	profile_id uuid not null,
	profile_version integer not null,
	pack_code text not null,
	assignment_kind text not null check (assignment_kind in ('common', 'legal_form', 'funding', 'program', 'institution')),
	status text not null check (status in ('active', 'superseded', 'revoked')),
	effective_from date not null,
	effective_to date,
	version integer not null default 1 check (version > 0),
	assigned_by_subject text not null,
	created_by_subject text not null,
	created_at timestamptz not null default now(),
	updated_by_subject text not null,
	updated_at timestamptz not null default now(),
	constraint school_policy_assignments_tenant_fk foreign key (tenant_code, institution_id) references app_tenants(code, institution_id) on delete restrict,
	constraint school_policy_assignments_pack_fk foreign key (tenant_code, institution_id, policy_pack_version_id) references school_policy_pack_versions(tenant_code, institution_id, id) on delete restrict,
	constraint school_policy_assignments_profile_fk foreign key (tenant_code, institution_id, profile_id, profile_version) references school_institution_profiles(tenant_code, institution_id, id, version) on delete restrict,
	constraint school_policy_assignments_scope_id_unique unique (tenant_code, institution_id, id),
	constraint school_policy_assignments_window check (effective_to is null or effective_to >= effective_from)
);
create unique index school_policy_assignments_one_active_pack
	on school_policy_assignments(tenant_code, institution_id, profile_id, pack_code)
	where status = 'active';
create index school_policy_assignments_effective
	on school_policy_assignments(tenant_code, institution_id, status, effective_from, effective_to);

create table school_policy_overrides (
	id uuid primary key default gen_random_uuid(),
	tenant_code text not null,
	institution_id text not null,
	policy_assignment_id uuid not null,
	key text not null,
	value jsonb not null,
	justification text not null,
	status text not null check (status in ('draft', 'approved', 'revoked')),
	version integer not null default 1 check (version > 0),
	approved_by_subject text not null default '',
	approved_at timestamptz,
	created_by_subject text not null,
	created_at timestamptz not null default now(),
	updated_by_subject text not null,
	updated_at timestamptz not null default now(),
	constraint school_policy_overrides_tenant_fk foreign key (tenant_code, institution_id) references app_tenants(code, institution_id) on delete restrict,
	constraint school_policy_overrides_assignment_fk foreign key (tenant_code, institution_id, policy_assignment_id) references school_policy_assignments(tenant_code, institution_id, id) on delete restrict,
	constraint school_policy_overrides_scope_key_unique unique (tenant_code, institution_id, policy_assignment_id, key),
	constraint school_policy_overrides_approval check (status <> 'approved' or (approved_by_subject <> '' and approved_at is not null))
);
create index school_policy_overrides_scope_status
	on school_policy_overrides(tenant_code, institution_id, status);

create table school_policy_evaluations (
	id uuid primary key default gen_random_uuid(),
	tenant_code text not null,
	institution_id text not null,
	profile_id uuid not null,
	profile_version integer not null,
	evaluated_at timestamptz not null default now(),
	evaluated_by_subject text not null,
	policy_pack_version_ids uuid[] not null default '{}',
	capabilities jsonb not null,
	warnings text[] not null default '{}',
	blocked boolean not null,
	checksum_sha256 text not null check (checksum_sha256 ~ '^[a-f0-9]{64}$'),
	created_by_subject text not null,
	created_at timestamptz not null default now(),
	updated_by_subject text not null,
	updated_at timestamptz not null default now(),
	constraint school_policy_evaluations_tenant_fk foreign key (tenant_code, institution_id) references app_tenants(code, institution_id) on delete restrict,
	constraint school_policy_evaluations_scope_id_unique unique (tenant_code, institution_id, id),
	constraint school_policy_evaluations_profile_fk foreign key (tenant_code, institution_id, profile_id, profile_version) references school_institution_profiles(tenant_code, institution_id, id, version) on delete restrict,
	constraint school_policy_evaluations_capabilities_array check (jsonb_typeof(capabilities) = 'array')
);
create index school_policy_evaluations_scope_time
	on school_policy_evaluations(tenant_code, institution_id, evaluated_at desc);
create unique index school_policy_evaluations_scope_checksum
	on school_policy_evaluations(tenant_code, institution_id, profile_id, checksum_sha256);

do $$
declare table_name text;
begin
	foreach table_name in array array[
		'school_institution_profiles', 'school_policy_pack_versions',
		'school_policy_assignments', 'school_policy_overrides',
		'school_policy_evaluations'
	]
	loop
		execute format('alter table %I enable row level security', table_name);
		execute format('alter table %I force row level security', table_name);
		execute format(
			'create policy tenant_isolation on %I using (public.can_bypass_tenant_rls() or (tenant_code = public.current_tenant_code() and institution_id = public.current_institution_id())) with check (public.can_bypass_tenant_rls() or (tenant_code = public.current_tenant_code() and institution_id = public.current_institution_id()))',
			table_name
		);
	end loop;
end $$;

create or replace function public.school_policy_evaluation_immutable()
returns trigger language plpgsql as $$
begin
	raise exception 'school policy evaluations are immutable';
end $$;
create trigger school_policy_evaluations_immutable
	before update or delete on school_policy_evaluations
	for each row execute function public.school_policy_evaluation_immutable();

create trigger school_profiles_versioning
	after insert or update or delete on school_institution_profiles
	for each row execute function public.record_entity_version();
create trigger school_policy_packs_versioning
	after insert or update or delete on school_policy_pack_versions
	for each row execute function public.record_entity_version();
create trigger school_policy_assignments_versioning
	after insert or update or delete on school_policy_assignments
	for each row execute function public.record_entity_version();
create trigger school_policy_overrides_versioning
	after insert or update or delete on school_policy_overrides
	for each row execute function public.record_entity_version();
create trigger school_policy_evaluations_versioning
	after insert on school_policy_evaluations
	for each row execute function public.record_entity_version();

-- Existing institutions remain deliberately unclassified. This preserves
-- read access but makes every future regulated command fail closed until an
-- authorised administrator approves an explicit profile.
insert into school_institution_profiles(
	tenant_code, institution_id, version, status,
	created_by_subject, updated_by_subject
)
select code, institution_id, 1, 'unclassified', 'migration:0135', 'migration:0135'
from app_tenants
on conflict (tenant_code, institution_id, version) do nothing;

-- Bootstrap approved, versioned packs per institution. Assignments are not
-- created for unclassified institutions; the service assigns only the packs
-- implied by an explicitly approved profile.
insert into school_policy_pack_versions(
	tenant_code, institution_id, pack_code, regulatory_profile, version,
	status, effective_from, rules, configurable_keys, source_references, checksum_sha256,
	approved_by_subject, approved_at, created_by_subject, updated_by_subject
)
select tenant.code, tenant.institution_id, pack.pack_code, pack.regulatory_profile, 1,
	'approved', date '2026-01-01', pack.rules::jsonb, array[
		'capabilities.education.publication.manage.reason',
		'capabilities.education.publication.manage.required_documents',
		'capabilities.education.publication.manage.required_approvals',
		'capabilities.education.publication.manage.wizard_steps'
	], pack.sources,
	encode(digest(pack.rules || '|' || pack.pack_code || '|1', 'sha256'), 'hex'),
	'migration:0135-reviewed-baseline', now(), 'migration:0135', 'migration:0135'
from app_tenants tenant
cross join (values
	('common.ro', 'ro.preuniversity', '{"capabilities":[{"code":"education.core","enabled":true,"reason":"Nucleu comun pentru învățământul preuniversitar","required_documents":[],"required_approvals":[],"wizard_steps":[]},{"code":"education.publication.manage","enabled":true,"reason":"Publicare instituțională supusă profilului reglementar","required_documents":[],"required_approvals":[],"wizard_steps":["date_publicare","anonimizare","confirmare"]}]}', array['Legea învățământului preuniversitar nr. 198/2023']),
	('legal-form.ro.public', 'ro.public.preuniversity', '{"capabilities":[{"code":"operations.public_controls","enabled":true,"reason":"Controale aplicabile instituției publice","required_documents":["referat_necesitate"],"required_approvals":["ordonator"],"wizard_steps":["fundamentare","control","aprobare"]},{"code":"operations.private_controls","enabled":false,"reason":"Profilul nu este privat","required_documents":[],"required_approvals":[],"wizard_steps":[]},{"code":"education.publication.manage","enabled":true,"reason":"Publicare în regimul instituției publice","required_documents":["versiune_anonimizata"],"required_approvals":["director"],"wizard_steps":["verificare_interes_public"]}]}', array['Legea nr. 544/2001','Legea nr. 198/2023']),
	('legal-form.ro.private', 'ro.private.preuniversity', '{"capabilities":[{"code":"operations.private_controls","enabled":true,"reason":"Politici interne ale instituției private","required_documents":["aprobare_interna"],"required_approvals":["reprezentant_legal"],"wizard_steps":["fundamentare","aprobare"]},{"code":"operations.public_controls","enabled":false,"reason":"Nu există overlay public activ","required_documents":[],"required_approvals":[],"wizard_steps":[]},{"code":"education.publication.manage","enabled":true,"reason":"Publicare conform politicii instituției private","required_documents":["versiune_anonimizata"],"required_approvals":["reprezentant_legal"],"wizard_steps":["verificare_politica_interna"]}]}', array['Legea învățământului preuniversitar nr. 198/2023']),
	('legal-form.ro.confessional', 'ro.private.confessional', '{"capabilities":[{"code":"operations.private_controls","enabled":true,"reason":"Politici interne ale instituției confesionale","required_documents":["aprobare_fondator"],"required_approvals":["fondator"],"wizard_steps":["fundamentare","aviz_fondator","aprobare"]},{"code":"operations.confessional_governance","enabled":true,"reason":"Guvernanță confesională declarată","required_documents":["aviz_cult"],"required_approvals":["fondator"],"wizard_steps":["avizare"]},{"code":"education.publication.manage","enabled":true,"reason":"Publicare conform politicii instituției confesionale","required_documents":["versiune_anonimizata"],"required_approvals":["director"],"wizard_steps":["verificare_politica_interna"]}]}', array['Legea învățământului preuniversitar nr. 198/2023']),
	('funding.public', 'ro.public.funding-overlay', '{"capabilities":[{"code":"operations.public_controls","enabled":true,"reason":"Overlay obligatoriu pentru fonduri publice","required_documents":["sursa_finantare","referat_necesitate"],"required_approvals":["control_financiar","ordonator"],"wizard_steps":["eligibilitate","control_financiar","aprobare"]}]}', array['Legea nr. 98/2016','Legea nr. 500/2002'])
) as pack(pack_code, regulatory_profile, rules, sources)
on conflict (tenant_code, institution_id, pack_code, version) do nothing;

-- Every regulated publication keeps the immutable decision snapshot that
-- authorised its creation. Scope is server-derived and cannot be supplied by
-- the browser.
alter table education_publications add column tenant_code text;
-- This is a provenance-only scope backfill for legacy rows. The official
-- artifact trigger correctly rejects every ordinary update to already
-- published/withdrawn records, so suspend only that trigger for this bounded
-- migration and restore it immediately afterwards.
alter table education_publications disable trigger trg_education_publications_official_immutability;
update education_publications publication
set tenant_code = tenant.code
from app_tenants tenant
where tenant.institution_id = publication.institution_id;
alter table education_publications enable trigger trg_education_publications_official_immutability;
alter table education_publications alter column tenant_code set not null;
alter table education_publications add column policy_evaluation_id uuid;
alter table education_publications add constraint education_publications_tenant_fk
	foreign key (tenant_code, institution_id) references app_tenants(code, institution_id) on delete restrict;
alter table education_publications add constraint education_publications_policy_evaluation_fk
	foreign key (tenant_code, institution_id, policy_evaluation_id)
	references school_policy_evaluations(tenant_code, institution_id, id) on delete restrict;
create index education_publications_policy_evaluation_idx
	on education_publications(tenant_code, institution_id, policy_evaluation_id);
create or replace function public.education_publication_requires_policy_evaluation()
returns trigger language plpgsql as $$
begin
	if new.policy_evaluation_id is null then
		raise exception 'new education publications require a policy evaluation' using errcode='23514';
	end if;
	return new;
end $$;
create trigger education_publications_require_policy_evaluation
	before insert on education_publications
	for each row execute function public.education_publication_requires_policy_evaluation();
drop policy if exists tenant_isolation on education_publications;
create policy tenant_isolation on education_publications
	using (public.can_bypass_tenant_rls() or (tenant_code=public.current_tenant_code() and institution_id=public.current_institution_id()))
	with check (public.can_bypass_tenant_rls() or (tenant_code=public.current_tenant_code() and institution_id=public.current_institution_id()));
