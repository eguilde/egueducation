-- Ordinul nr. 3.858/2026 / metodologia-cadru, Anexa nr. 1, applies to
-- portfolios starting with school year 2026-2027.  The former catalog was
-- assembled from a draft and must remain traceable, but must no longer be the
-- active default.
create table if not exists education_portfolio_section_catalog_versions (
	code text primary key,
	source_ref text not null,
	effective_from date,
	effective_to date,
	status text not null check (status in ('active', 'superseded', 'retired')),
	created_at timestamptz not null default now(),
	retired_at timestamptz
);

insert into education_portfolio_section_catalog_versions (
	code, source_ref, effective_from, effective_to, status, retired_at
) values
	('legacy-unversioned', 'Configurare istorică anterioară formei finale a Ordinului nr. 3.858/2026', null, date '2026-08-31', 'superseded', now()),
	('ome-3858-2026-annexa-1-v1', 'Ordinul nr. 3.858/2026, metodologia-cadru, Anexa nr. 1', date '2026-09-01', null, 'active', null)
on conflict (code) do update
set source_ref = excluded.source_ref,
	effective_from = excluded.effective_from,
	effective_to = excluded.effective_to,
	status = excluded.status,
	retired_at = excluded.retired_at;

alter table education_portfolio_sections
	add column if not exists catalog_version text not null default 'legacy-unversioned',
	add column if not exists source_ref text not null default '';

alter table education_portfolio_sections
	drop constraint if exists education_portfolio_sections_catalog_version_fkey;
alter table education_portfolio_sections
	add constraint education_portfolio_sections_catalog_version_fkey
	foreign key (catalog_version)
	references education_portfolio_section_catalog_versions(code) on delete restrict;

update education_portfolio_sections
set catalog_version = 'legacy-unversioned',
	source_ref = coalesce(nullif(source_ref, ''), 'Configurare istorică anterioară formei finale a Ordinului nr. 3.858/2026'),
	active = false
where catalog_version = 'legacy-unversioned';

-- The five rows below deliberately model only the mandatory top-level
-- structure from Anexa nr. 1.  Categories and example documents are set by a
-- school procedure; they are not inferred here.
insert into education_portfolio_sections (
	section_code,
	component_code,
	label_ro,
	label_en,
	example_documents,
	required,
	sensitive_data,
	retention_rule,
	sort_order,
	active,
	catalog_version,
	source_ref
) values
	('identificare_profesionala', 'structura_cadru', 'Date personale și de identificare profesională', 'Personal data and professional identification', array[]::text[], true, false, 'portfolio_retention_3_years_after_activity_end', 10, true, 'ome-3858-2026-annexa-1-v1', 'Ordinul nr. 3.858/2026, metodologia-cadru, Anexa nr. 1'),
	('predare_invatare_evaluare', 'structura_cadru', 'Activitate specifică normei didactice de predare-învățare-evaluare', 'Teaching-learning-assessment activity specific to the teaching workload', array[]::text[], true, false, 'portfolio_retention_3_years_after_activity_end', 20, true, 'ome-3858-2026-annexa-1-v1', 'Ordinul nr. 3.858/2026, metodologia-cadru, Anexa nr. 1'),
	('activitati_complementare', 'structura_cadru', 'Activități complementare procesului de învățământ', 'Activities complementary to the educational process', array[]::text[], true, false, 'portfolio_retention_3_years_after_activity_end', 30, true, 'ome-3858-2026-annexa-1-v1', 'Ordinul nr. 3.858/2026, metodologia-cadru, Anexa nr. 1'),
	('managementul_clasei', 'structura_cadru', 'Activități de management al clasei', 'Classroom management activities', array[]::text[], true, false, 'portfolio_retention_3_years_after_activity_end', 40, true, 'ome-3858-2026-annexa-1-v1', 'Ordinul nr. 3.858/2026, metodologia-cadru, Anexa nr. 1'),
	('evolutie_dezvoltare_profesionala', 'structura_cadru', 'Evoluția în cariera didactică și dezvoltarea profesională', 'Teaching career evolution and professional development', array[]::text[], true, false, 'portfolio_retention_3_years_after_activity_end', 50, true, 'ome-3858-2026-annexa-1-v1', 'Ordinul nr. 3.858/2026, metodologia-cadru, Anexa nr. 1')
on conflict (section_code, component_code) do update
set label_ro = excluded.label_ro,
	label_en = excluded.label_en,
	example_documents = excluded.example_documents,
	required = excluded.required,
	sensitive_data = excluded.sensitive_data,
	retention_rule = excluded.retention_rule,
	sort_order = excluded.sort_order,
	active = true,
	catalog_version = excluded.catalog_version,
	source_ref = excluded.source_ref;

-- A procedure is an institution-owned, versioned legal artefact.  Content is
-- editable while it is a draft only; once approved it is frozen and a change
-- requires a new version.
create table if not exists education_portfolio_procedure_versions (
	id uuid primary key default gen_random_uuid(),
	institution_id text not null,
	tenant_code text not null references app_tenants(code) on delete restrict,
	procedure_code text not null check (btrim(procedure_code) <> ''),
	version_no integer not null check (version_no > 0),
	title text not null check (btrim(title) <> ''),
	source_ref text not null default 'Ordinul nr. 3.858/2026, metodologia-cadru',
	lifecycle_status text not null default 'draft' check (lifecycle_status in ('draft', 'approved', 'published', 'superseded', 'withdrawn')),
	effective_from date,
	effective_to date,
	calendar_rules jsonb not null default '{}'::jsonb check (jsonb_typeof(calendar_rules) = 'object'),
	access_rules jsonb not null default '{}'::jsonb check (jsonb_typeof(access_rules) = 'object'),
	accepted_formats jsonb not null default '{}'::jsonb check (jsonb_typeof(accepted_formats) = 'object'),
	retention_rules jsonb not null default '{}'::jsonb check (jsonb_typeof(retention_rules) = 'object'),
	transfer_rules jsonb not null default '{}'::jsonb check (jsonb_typeof(transfer_rules) = 'object'),
	approval_evidence jsonb not null default '{}'::jsonb check (jsonb_typeof(approval_evidence) = 'object'),
	publication_evidence jsonb not null default '{}'::jsonb check (jsonb_typeof(publication_evidence) = 'object'),
	approved_at timestamptz,
	approved_by_user_id uuid references app_users(id) on delete restrict,
	published_at timestamptz,
	published_by_user_id uuid references app_users(id) on delete restrict,
	created_by_user_id uuid not null references app_users(id) on delete restrict,
	created_at timestamptz not null default now(),
	updated_at timestamptz not null default now(),
	superseded_at timestamptz,
	withdrawn_at timestamptz,
	unique (institution_id, procedure_code, version_no),
	unique (id, institution_id, tenant_code),
	check (effective_to is null or effective_from is null or effective_to >= effective_from),
	check (lifecycle_status not in ('approved', 'published', 'superseded') or (approved_at is not null and approved_by_user_id is not null)),
	check (lifecycle_status <> 'published' or (published_at is not null and published_by_user_id is not null))
);

create unique index if not exists uq_education_portfolio_procedure_published
	on education_portfolio_procedure_versions (institution_id, procedure_code)
	where lifecycle_status = 'published';
create index if not exists idx_education_portfolio_procedure_versions_scope
	on education_portfolio_procedure_versions (tenant_code, institution_id, lifecycle_status, effective_from desc);

create table if not exists education_portfolio_procedure_section_rules (
	id uuid primary key default gen_random_uuid(),
	procedure_id uuid not null,
	institution_id text not null,
	tenant_code text not null,
	section_code text not null check (btrim(section_code) <> ''),
	label_ro text not null check (btrim(label_ro) <> ''),
	label_en text not null default '',
	source_catalog_version text not null references education_portfolio_section_catalog_versions(code) on delete restrict,
	required boolean not null default true,
	sort_order integer not null check (sort_order > 0),
	active boolean not null default true,
	created_at timestamptz not null default now(),
	updated_at timestamptz not null default now(),
	unique (procedure_id, section_code),
	foreign key (procedure_id, institution_id, tenant_code)
		references education_portfolio_procedure_versions (id, institution_id, tenant_code) on delete restrict
);
create index if not exists idx_education_portfolio_procedure_section_rules_scope
	on education_portfolio_procedure_section_rules (tenant_code, institution_id, procedure_id, sort_order);

-- The selected procedure is intentionally nullable for historical portfolios.
-- New command handling will set it at creation once the procedure workflow is
-- exposed; the trigger makes cross-institution binding impossible.
alter table education_portfolios
	add column if not exists applied_procedure_id uuid references education_portfolio_procedure_versions(id) on delete restrict;
create index if not exists idx_education_portfolios_applied_procedure
	on education_portfolios (institution_id, applied_procedure_id);

-- Acknowledgements are evidence, rather than editable booleans.  The exact
-- text and its version are copied at acceptance so later template changes can
-- never rewrite a teacher's declaration.
create table if not exists education_portfolio_declaration_acknowledgements (
	id uuid primary key default gen_random_uuid(),
	portfolio_id uuid not null references education_portfolios(id) on delete restrict,
	institution_id text not null,
	tenant_code text not null references app_tenants(code) on delete restrict,
	declaration_type text not null check (declaration_type in ('gdpr_information', 'authenticity')),
	declaration_version text not null check (btrim(declaration_version) <> ''),
	declaration_text text not null check (btrim(declaration_text) <> ''),
	accepted_at timestamptz not null default now(),
	accepted_by_user_id uuid not null references app_users(id) on delete restrict,
	attestation_method text not null check (btrim(attestation_method) <> ''),
	attestation_evidence jsonb not null check (jsonb_typeof(attestation_evidence) = 'object' and attestation_evidence <> '{}'::jsonb),
	signature_evidence jsonb not null default '{}'::jsonb check (jsonb_typeof(signature_evidence) = 'object'),
	created_at timestamptz not null default now(),
	unique (portfolio_id, declaration_type, declaration_version)
);
create index if not exists idx_education_portfolio_declarations_scope
	on education_portfolio_declaration_acknowledgements (tenant_code, institution_id, portfolio_id, declaration_type, accepted_at desc);

create or replace function public.enforce_education_portfolio_tenant_scope(
	p_institution_id text,
	p_tenant_code text,
	p_user_id uuid,
	p_entity_name text
)
returns void
language plpgsql
as $$
begin
	if not exists (
		select 1 from app_tenants tenant
		where tenant.code = p_tenant_code
			and tenant.institution_id = p_institution_id
	) then
		raise exception '% tenant_code must belong to institution_id', p_entity_name;
	end if;
	if p_user_id is not null and not exists (
		select 1 from app_memberships membership
		where membership.user_id = p_user_id
			and membership.tenant_code = p_tenant_code
			and membership.active
	) then
		raise exception '% actor must have an active membership in tenant', p_entity_name;
	end if;
end;
$$;

create or replace function public.enforce_education_portfolio_procedure_scope()
returns trigger language plpgsql as $$
begin
	perform public.enforce_education_portfolio_tenant_scope(new.institution_id, new.tenant_code, new.created_by_user_id, 'portfolio procedure');
	if new.approved_by_user_id is not null then
		perform public.enforce_education_portfolio_tenant_scope(new.institution_id, new.tenant_code, new.approved_by_user_id, 'portfolio procedure approval');
	end if;
	if new.published_by_user_id is not null then
		perform public.enforce_education_portfolio_tenant_scope(new.institution_id, new.tenant_code, new.published_by_user_id, 'portfolio procedure publication');
	end if;
	return new;
end;
$$;
drop trigger if exists trg_education_portfolio_procedure_scope on education_portfolio_procedure_versions;
create trigger trg_education_portfolio_procedure_scope
	before insert or update on education_portfolio_procedure_versions
	for each row execute function public.enforce_education_portfolio_procedure_scope();

create or replace function public.enforce_education_portfolio_procedure_lifecycle()
returns trigger language plpgsql as $$
begin
	if tg_op = 'DELETE' then
		raise exception 'portfolio procedure versions are evidentiary records and cannot be hard-deleted';
	end if;
	if tg_op = 'UPDATE' then
		if old.institution_id is distinct from new.institution_id
			or old.tenant_code is distinct from new.tenant_code
			or old.procedure_code is distinct from new.procedure_code
			or old.version_no is distinct from new.version_no then
			raise exception 'portfolio procedure identity and scope are immutable';
		end if;
		if old.lifecycle_status = 'draft' then
			if new.lifecycle_status not in ('draft', 'approved', 'withdrawn') then
				raise exception 'invalid portfolio procedure lifecycle transition from draft';
			end if;
		elsif old.lifecycle_status = 'approved' then
			if new.lifecycle_status not in ('published', 'withdrawn') then
				raise exception 'approved portfolio procedure can only be published or withdrawn';
			end if;
			if row(new.title, new.source_ref, new.effective_from, new.effective_to, new.calendar_rules, new.access_rules, new.accepted_formats, new.retention_rules, new.transfer_rules, new.approval_evidence, new.approved_at, new.approved_by_user_id)
				is distinct from row(old.title, old.source_ref, old.effective_from, old.effective_to, old.calendar_rules, old.access_rules, old.accepted_formats, old.retention_rules, old.transfer_rules, old.approval_evidence, old.approved_at, old.approved_by_user_id) then
				raise exception 'approved portfolio procedure content is immutable; create a new version';
			end if;
		elsif old.lifecycle_status = 'published' then
			if new.lifecycle_status not in ('superseded', 'withdrawn') then
				raise exception 'published portfolio procedure can only be superseded or withdrawn';
			end if;
			if row(new.title, new.source_ref, new.effective_from, new.effective_to, new.calendar_rules, new.access_rules, new.accepted_formats, new.retention_rules, new.transfer_rules, new.approval_evidence, new.approved_at, new.approved_by_user_id, new.publication_evidence, new.published_at, new.published_by_user_id)
				is distinct from row(old.title, old.source_ref, old.effective_from, old.effective_to, old.calendar_rules, old.access_rules, old.accepted_formats, old.retention_rules, old.transfer_rules, old.approval_evidence, old.approved_at, old.approved_by_user_id, old.publication_evidence, old.published_at, old.published_by_user_id) then
				raise exception 'published portfolio procedure content is immutable; create a new version';
			end if;
		else
			raise exception 'superseded or withdrawn portfolio procedure is immutable';
		end if;
	end if;
	return new;
end;
$$;
drop trigger if exists trg_education_portfolio_procedure_lifecycle on education_portfolio_procedure_versions;
create trigger trg_education_portfolio_procedure_lifecycle
	before update or delete on education_portfolio_procedure_versions
	for each row execute function public.enforce_education_portfolio_procedure_lifecycle();

create or replace function public.enforce_education_portfolio_procedure_section_rule_mutability()
returns trigger language plpgsql as $$
declare
	procedure_status text;
	procedure_id_value uuid;
begin
	procedure_id_value := case when tg_op = 'DELETE' then old.procedure_id else new.procedure_id end;
	select lifecycle_status into procedure_status
	from education_portfolio_procedure_versions
	where id = procedure_id_value
	for update;
	if procedure_status is null then
		raise exception 'portfolio procedure does not exist';
	end if;
	if procedure_status <> 'draft' then
		raise exception 'portfolio procedure section rules are immutable outside draft';
	end if;
	return case when tg_op = 'DELETE' then old else new end;
end;
$$;
drop trigger if exists trg_education_portfolio_procedure_section_rule_mutability on education_portfolio_procedure_section_rules;
create trigger trg_education_portfolio_procedure_section_rule_mutability
	before insert or update or delete on education_portfolio_procedure_section_rules
	for each row execute function public.enforce_education_portfolio_procedure_section_rule_mutability();

create or replace function public.enforce_education_portfolio_applied_procedure_scope()
returns trigger language plpgsql as $$
begin
	if new.applied_procedure_id is not null and not exists (
		select 1 from education_portfolio_procedure_versions procedure_row
		where procedure_row.id = new.applied_procedure_id
			and procedure_row.institution_id = new.institution_id
			and procedure_row.lifecycle_status in ('approved', 'published')
	) then
		raise exception 'portfolio applied procedure must be approved or published in the same institution';
	end if;
	return new;
end;
$$;
drop trigger if exists trg_education_portfolio_applied_procedure_scope on education_portfolios;
create trigger trg_education_portfolio_applied_procedure_scope
	before insert or update on education_portfolios
	for each row execute function public.enforce_education_portfolio_applied_procedure_scope();

create or replace function public.enforce_education_portfolio_declaration_acknowledgement()
returns trigger language plpgsql as $$
declare
	portfolio_owner uuid;
	portfolio_institution text;
begin
	perform public.enforce_education_portfolio_tenant_scope(new.institution_id, new.tenant_code, new.accepted_by_user_id, 'portfolio declaration acknowledgement');
	select owner_user_id, institution_id into portfolio_owner, portfolio_institution
	from education_portfolios
	where id = new.portfolio_id;
	if portfolio_institution is null or portfolio_institution <> new.institution_id then
		raise exception 'portfolio declaration acknowledgement must belong to the portfolio institution';
	end if;
	if portfolio_owner is null or portfolio_owner <> new.accepted_by_user_id then
		raise exception 'portfolio declaration acknowledgement must be accepted by the portfolio owner';
	end if;
	return new;
end;
$$;
drop trigger if exists trg_education_portfolio_declaration_acknowledgement on education_portfolio_declaration_acknowledgements;
create trigger trg_education_portfolio_declaration_acknowledgement
	before insert on education_portfolio_declaration_acknowledgements
	for each row execute function public.enforce_education_portfolio_declaration_acknowledgement();

create or replace function public.prevent_education_portfolio_declaration_acknowledgement_mutation()
returns trigger language plpgsql as $$
begin
	raise exception 'portfolio declaration acknowledgements are immutable';
end;
$$;
drop trigger if exists trg_education_portfolio_declaration_acknowledgement_immutable on education_portfolio_declaration_acknowledgements;
create trigger trg_education_portfolio_declaration_acknowledgement_immutable
	before update or delete on education_portfolio_declaration_acknowledgements
	for each row execute function public.prevent_education_portfolio_declaration_acknowledgement_mutation();

alter table education_portfolio_procedure_versions enable row level security;
alter table education_portfolio_procedure_versions force row level security;
drop policy if exists tenant_isolation on education_portfolio_procedure_versions;
create policy tenant_isolation on education_portfolio_procedure_versions
	using (public.can_bypass_tenant_rls() or (institution_id = public.current_institution_id() and tenant_code = public.current_tenant_code()))
	with check (public.can_bypass_tenant_rls() or (institution_id = public.current_institution_id() and tenant_code = public.current_tenant_code()));

alter table education_portfolio_procedure_section_rules enable row level security;
alter table education_portfolio_procedure_section_rules force row level security;
drop policy if exists tenant_isolation on education_portfolio_procedure_section_rules;
create policy tenant_isolation on education_portfolio_procedure_section_rules
	using (public.can_bypass_tenant_rls() or (institution_id = public.current_institution_id() and tenant_code = public.current_tenant_code()))
	with check (public.can_bypass_tenant_rls() or (institution_id = public.current_institution_id() and tenant_code = public.current_tenant_code()));

alter table education_portfolio_declaration_acknowledgements enable row level security;
alter table education_portfolio_declaration_acknowledgements force row level security;
drop policy if exists tenant_isolation on education_portfolio_declaration_acknowledgements;
create policy tenant_isolation on education_portfolio_declaration_acknowledgements
	using (public.can_bypass_tenant_rls() or (institution_id = public.current_institution_id() and tenant_code = public.current_tenant_code()))
	with check (public.can_bypass_tenant_rls() or (institution_id = public.current_institution_id() and tenant_code = public.current_tenant_code()));

drop trigger if exists trg_education_portfolio_procedure_versions_entity_version on education_portfolio_procedure_versions;
create trigger trg_education_portfolio_procedure_versions_entity_version
	after insert or update or delete on education_portfolio_procedure_versions
	for each row execute function public.record_entity_version();
drop trigger if exists trg_education_portfolio_procedure_section_rules_entity_version on education_portfolio_procedure_section_rules;
create trigger trg_education_portfolio_procedure_section_rules_entity_version
	after insert or update or delete on education_portfolio_procedure_section_rules
	for each row execute function public.record_entity_version();
drop trigger if exists trg_education_portfolio_declaration_acknowledgements_entity_version on education_portfolio_declaration_acknowledgements;
create trigger trg_education_portfolio_declaration_acknowledgements_entity_version
	after insert on education_portfolio_declaration_acknowledgements
	for each row execute function public.record_entity_version();
