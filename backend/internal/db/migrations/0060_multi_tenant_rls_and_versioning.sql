create or replace function public.current_institution_id()
returns text
language sql
stable
as $$
	select nullif(current_setting('app.institution_id', true), '');
$$;

create or replace function public.current_tenant_code()
returns text
language sql
stable
as $$
	select nullif(current_setting('app.tenant_id', true), '');
$$;

create or replace function public.can_bypass_tenant_rls()
returns boolean
language sql
stable
as $$
	select coalesce(nullif(current_setting('app.is_super_admin', true), ''), 'false') = 'true';
$$;

alter table app_org_units
	add column if not exists tenant_code text null references app_tenants(code) on delete restrict;

update app_org_units
set tenant_code = case
	when code like 'unit-balotesti-%' then 'tenant-balotesti'
	when code in ('unit-root', 'unit-management', 'unit-secretariat', 'unit-archive', 'unit-hr', 'unit-primary', 'unit-gymnasium', 'unit-root') then 'tenant-egueducation'
	else coalesce(tenant_code, 'tenant-egueducation')
end
where tenant_code is null or tenant_code = '';

alter table app_org_units
	alter column tenant_code set not null;

create index if not exists idx_app_org_units_tenant_code
	on app_org_units (tenant_code, code);

alter table app_memberships
	add column if not exists tenant_code text null references app_tenants(code) on delete restrict;

update app_memberships m
set tenant_code = coalesce(ou.tenant_code, 'tenant-egueducation')
from app_org_units ou
where m.org_unit_code = ou.code
	and (m.tenant_code is null or m.tenant_code = '');

alter table app_memberships
	alter column tenant_code set not null;

create index if not exists idx_app_memberships_tenant_user
	on app_memberships (tenant_code, user_id, active);

create table if not exists app_entity_versions (
	id uuid primary key default gen_random_uuid(),
	entity_table text not null,
	entity_id uuid not null,
	version_no integer not null,
	change_type text not null check (change_type in ('insert', 'update', 'delete')),
	tenant_code text not null default '',
	institution_id text not null default '',
	snapshot jsonb not null,
	changed_by text not null default '',
	changed_at timestamptz not null default now(),
	unique (entity_table, entity_id, version_no)
);

create index if not exists idx_app_entity_versions_lookup
	on app_entity_versions (tenant_code, entity_table, entity_id, version_no desc);

create or replace function public.record_entity_version()
returns trigger
language plpgsql
as $$
declare
	snapshot jsonb;
	entity_id_value uuid;
	next_version integer;
	tenant_value text;
	institution_value text;
	changed_by_value text;
begin
	if tg_op = 'DELETE' then
		snapshot := to_jsonb(old);
	elsif tg_op = 'UPDATE' then
		snapshot := to_jsonb(new);
	else
		snapshot := to_jsonb(new);
	end if;

	if coalesce(snapshot ->> 'id', '') = '' then
		return coalesce(new, old);
	end if;

	entity_id_value := (snapshot ->> 'id')::uuid;
	tenant_value := coalesce(nullif(snapshot ->> 'tenant_code', ''), nullif(current_setting('app.tenant_id', true), ''), '');
	institution_value := coalesce(nullif(snapshot ->> 'institution_id', ''), nullif(current_setting('app.institution_id', true), ''), '');
	changed_by_value := coalesce(nullif(current_setting('app.actor_subject', true), ''), '');

	select coalesce(max(version_no), 0) + 1
	into next_version
	from app_entity_versions
	where entity_table = tg_table_name
		and entity_id = entity_id_value;

	insert into app_entity_versions (
		entity_table,
		entity_id,
		version_no,
		change_type,
		tenant_code,
		institution_id,
		snapshot,
		changed_by,
		changed_at
	) values (
		tg_table_name,
		entity_id_value,
		next_version,
		lower(tg_op),
		tenant_value,
		institution_value,
		snapshot,
		changed_by_value,
		now()
	);

	return coalesce(new, old);
end;
$$;

do $$
declare
	tables text[] := array[
		'registratura_documents',
		'archive_records',
		'workflow_definitions',
		'workflow_instances',
		'education_meetings',
		'education_personnel',
		'education_portfolios',
		'education_mobility_cases',
		'education_merit_grants',
		'gdpr_retention_policies',
		'gdpr_subject_requests',
		'education_regulations',
		'education_evaluations',
		'education_declarations',
		'education_decisions',
		'education_managerial_dossiers',
		'gdpr_subject_exports',
		'gdpr_publication_reviews',
		'education_meeting_participants',
		'education_meeting_documents',
		'education_portfolio_documents',
		'education_meeting_votes',
		'education_portfolio_checklist',
		'education_governance_memberships',
		'education_meeting_resolutions',
		'education_portfolio_transfers',
		'education_portfolio_reviews',
		'education_meeting_minutes',
		'education_portfolio_opis',
		'education_portfolio_custody',
		'education_publications',
		'education_managerial_documents',
		'education_managerial_workflow_steps',
		'education_regulation_versions',
		'education_regulation_workflow_steps',
		'education_decision_issuances',
		'education_decision_publication_steps',
		'education_mobility_documents',
		'education_mobility_scores',
		'education_mobility_appeals',
		'education_merit_documents',
		'education_merit_scores',
		'education_merit_appeals',
		'education_mobility_final_decisions',
		'education_mobility_result_issues',
		'education_merit_final_decisions',
		'education_merit_result_issues',
		'education_personnel_assignments',
		'education_personnel_disciplinary_cases',
		'education_personnel_file_documents',
		'education_personnel_access_events',
		'education_evaluation_appeals',
		'education_evaluation_self_reviews',
		'education_evaluation_criteria',
		'education_evaluation_result_issues'
	];
	tbl text;
	has_institution_id boolean;
	begin
	foreach tbl in array tables loop
		if tbl = 'workflow_definitions' then
			execute format('alter table %I enable row level security', tbl);
			execute format('alter table %I force row level security', tbl);
			execute format('drop policy if exists tenant_isolation on %I', tbl);
			execute format('drop policy if exists tenant_read on %I', tbl);
			execute format('drop policy if exists tenant_write on %I', tbl);
			execute format(
				'create policy tenant_isolation on %I using (true) with check (true)',
				tbl
			);
			execute format('drop trigger if exists trg_%s_entity_version on %I', tbl, tbl);
			execute format('create trigger trg_%s_entity_version after insert or update or delete on %I for each row execute function public.record_entity_version()', tbl, tbl);
			continue;
		end if;

		select exists (
			select 1
			from information_schema.columns
			where table_schema = 'public'
				and table_name = tbl
				and column_name = 'institution_id'
		)
		into has_institution_id;

		if not has_institution_id then
			raise exception 'table % is missing institution_id and is not handled by multi-tenant RLS migration', tbl;
		end if;

		execute format('alter table %I enable row level security', tbl);
		execute format('alter table %I force row level security', tbl);
		execute format('drop policy if exists tenant_isolation on %I', tbl);
		execute format('drop policy if exists tenant_read on %I', tbl);
		execute format('drop policy if exists tenant_write on %I', tbl);
		execute format(
			'create policy tenant_isolation on %I using (public.can_bypass_tenant_rls() or institution_id = public.current_institution_id()) with check (public.can_bypass_tenant_rls() or institution_id = public.current_institution_id())',
			tbl
		);
		execute format('drop trigger if exists trg_%s_entity_version on %I', tbl, tbl);
		execute format('create trigger trg_%s_entity_version after insert or update or delete on %I for each row execute function public.record_entity_version()', tbl, tbl);
	end loop;
end;
$$;

do $$
begin
	alter table app_tenants enable row level security;
	alter table app_tenants force row level security;
	drop policy if exists tenant_isolation on app_tenants;
	create policy tenant_isolation on app_tenants
		using (public.can_bypass_tenant_rls())
		with check (public.can_bypass_tenant_rls());

	alter table app_org_units enable row level security;
	alter table app_org_units force row level security;
	drop policy if exists tenant_isolation on app_org_units;
	create policy tenant_isolation on app_org_units
		using (public.can_bypass_tenant_rls() or tenant_code = public.current_tenant_code())
		with check (public.can_bypass_tenant_rls() or tenant_code = public.current_tenant_code());

	alter table app_memberships enable row level security;
	alter table app_memberships force row level security;
	drop policy if exists tenant_isolation on app_memberships;
	create policy tenant_isolation on app_memberships
		using (public.can_bypass_tenant_rls() or tenant_code = public.current_tenant_code())
		with check (public.can_bypass_tenant_rls() or tenant_code = public.current_tenant_code());
end;
$$;
