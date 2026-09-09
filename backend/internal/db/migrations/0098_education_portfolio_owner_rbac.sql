-- A professional portfolio is owned by an authenticated person, not by a
-- display name.  Existing demonstration/legacy rows pre-date identity links,
-- so they remain explicitly unbound until an administrator reconciles them.
alter table education_portfolios
	add column if not exists owner_user_id uuid references app_users(id) on delete restrict,
	add column if not exists owner_personnel_id uuid references education_personnel(id) on delete restrict;

alter table education_portfolios drop constraint if exists education_portfolios_status_check;
alter table education_portfolios add constraint education_portfolios_status_check
	check (status in ('draft', 'submitted', 'returned', 'validated', 'transferred', 'archived'));

create index if not exists idx_education_portfolios_owner_lookup
	on education_portfolios (institution_id, owner_user_id, school_year);

-- One identity-bound professional portfolio per school year and institution.
-- Legacy unbound records remain reconcilable and are intentionally excluded.
create unique index if not exists uq_education_portfolios_owner_school_year
	on education_portfolios (institution_id, owner_user_id, school_year)
	where owner_user_id is not null;

-- A safe best-effort backfill may only link a personnel row in the same
-- institution and only when the display name has one unambiguous match.
with candidates as (
	select ep.id, min(person.id) as personnel_id
	from education_portfolios ep
	join education_personnel person
		on person.institution_id = ep.institution_id
		and lower(btrim(person.full_name)) = lower(btrim(ep.owner_name))
	group by ep.id
	having count(*) = 1
)
update education_portfolios ep
set owner_personnel_id = candidates.personnel_id
from candidates
where ep.id = candidates.id
	and ep.owner_personnel_id is null;

create or replace function public.enforce_education_portfolio_owner()
returns trigger
language plpgsql
as $$
declare
	owner_has_membership boolean;
	personnel_in_institution boolean;
begin
	-- Null ownership is grandfathered for rows created before this migration;
	-- all new application writes are identity-bound.
	if tg_op = 'INSERT' and new.owner_user_id is null then
		raise exception 'education portfolio owner_user_id is required';
	end if;

	if tg_op = 'UPDATE' and old.owner_user_id is not null
		and new.owner_user_id is distinct from old.owner_user_id then
		raise exception 'education portfolio owner_user_id is immutable';
	end if;

	if new.owner_user_id is not null then
		select exists(
			select 1
			from app_memberships membership
			join app_tenants tenant on tenant.code = membership.tenant_code
			where membership.user_id = new.owner_user_id
				and membership.active = true
				and tenant.active = true
				and tenant.institution_id = new.institution_id
		) into owner_has_membership;
		if not owner_has_membership then
			raise exception 'education portfolio owner must have an active membership in the portfolio institution';
		end if;
	end if;

	if new.owner_personnel_id is not null then
		select exists(
			select 1 from education_personnel person
			where person.id = new.owner_personnel_id
				and person.institution_id = new.institution_id
		) into personnel_in_institution;
		if not personnel_in_institution then
			raise exception 'education portfolio personnel owner must belong to the portfolio institution';
		end if;
	end if;
	return new;
end;
$$;

drop trigger if exists trg_education_portfolio_owner on education_portfolios;
create trigger trg_education_portfolio_owner
	before insert or update on education_portfolios
	for each row execute function public.enforce_education_portfolio_owner();

-- A teacher receives a narrow, explicit grant for an eArhiva item selected as
-- portfolio evidence. This avoids treating institution membership as archive
-- read access. Grants are administered by an institution workflow; they do
-- not confer any generic eArhiva capability.
create table if not exists education_portfolio_archive_attachment_grants (
	id uuid primary key default gen_random_uuid(),
	institution_id text not null,
	archive_document_id uuid not null references archive_documents(id) on delete restrict,
	grantee_user_id uuid not null references app_users(id) on delete cascade,
	granted_by_user_id uuid references app_users(id) on delete set null,
	created_at timestamptz not null default now(),
	unique (institution_id, archive_document_id, grantee_user_id)
);
create index if not exists idx_education_portfolio_archive_attachment_grants_grantee
	on education_portfolio_archive_attachment_grants (institution_id, grantee_user_id, archive_document_id);
alter table education_portfolio_archive_attachment_grants
	drop constraint if exists education_portfolio_archive_attachment_grants_archive_document_tenant_fk;
alter table education_portfolio_archive_attachment_grants
	add constraint education_portfolio_archive_attachment_grants_archive_document_tenant_fk
	foreign key (institution_id, archive_document_id)
	references archive_documents (institution_id, id) on delete restrict;

create or replace function public.enforce_education_portfolio_archive_attachment_grant()
returns trigger language plpgsql as $$
begin
	if not exists (
		select 1 from app_memberships membership
		join app_tenants tenant on tenant.code = membership.tenant_code
		where membership.user_id = new.grantee_user_id
			and membership.active and tenant.active
			and tenant.institution_id = new.institution_id
	) then
		raise exception 'portfolio archive attachment grantee must have an active same-institution membership';
	end if;
	return new;
end;
$$;
drop trigger if exists trg_education_portfolio_archive_attachment_grant on education_portfolio_archive_attachment_grants;
create trigger trg_education_portfolio_archive_attachment_grant
	before insert or update on education_portfolio_archive_attachment_grants
	for each row execute function public.enforce_education_portfolio_archive_attachment_grant();
alter table education_portfolio_archive_attachment_grants enable row level security;
alter table education_portfolio_archive_attachment_grants force row level security;
drop policy if exists education_portfolio_archive_attachment_grants_tenant_isolation on education_portfolio_archive_attachment_grants;
create policy education_portfolio_archive_attachment_grants_tenant_isolation
	on education_portfolio_archive_attachment_grants
	using (public.can_bypass_tenant_rls() or institution_id = public.current_institution_id())
	with check (public.can_bypass_tenant_rls() or institution_id = public.current_institution_id());

-- Persist an immutable archive-version snapshot with each newly attached
-- evidence record. Existing legacy references remain nullable until they are
-- reconciled rather than being guessed from mutable text values.
alter table education_portfolio_documents
	add column if not exists archive_document_id uuid references archive_documents(id) on delete restrict,
	add column if not exists archive_version_id uuid references archive_document_versions(id) on delete restrict,
	add column if not exists archive_version_no integer,
	add column if not exists archive_source_bucket text not null default '',
	add column if not exists archive_source_object_key text not null default '',
	add column if not exists archive_sha256 text not null default '';
create index if not exists idx_education_portfolio_documents_archive_snapshot
	on education_portfolio_documents (institution_id, archive_document_id, archive_version_id);

-- Serialize portfolio content mutations with submission. The parent row lock
-- makes an in-flight document update/delete either finish before submit is
-- validated or see the submitted status and fail; submitted evidence is never
-- editable or deletable.
create or replace function public.enforce_education_portfolio_document_state()
returns trigger language plpgsql as $$
declare
	portfolio_status text;
	portfolio_ref uuid;
begin
	portfolio_ref := case when tg_op = 'DELETE' then old.portfolio_id else new.portfolio_id end;
	select status into portfolio_status from education_portfolios where id = portfolio_ref for update;
	if portfolio_status is null then
		raise exception 'portfolio does not exist';
	end if;
	if portfolio_status not in ('draft', 'returned') then
		raise exception 'portfolio evidence is immutable outside draft or returned state';
	end if;
	return case when tg_op = 'DELETE' then old else new end;
end;
$$;
drop trigger if exists trg_education_portfolio_document_state on education_portfolio_documents;
create trigger trg_education_portfolio_document_state
	before insert or update or delete on education_portfolio_documents
	for each row execute function public.enforce_education_portfolio_document_state();

insert into app_permissions(code, label) values
	('education.portfolios.read_own', 'Read own professional portfolio'),
	('education.portfolios.manage_own', 'Manage own professional portfolio'),
	('education.portfolios.school.read', 'Read institution professional portfolios'),
	('education.portfolios.school.manage', 'Manage institution professional portfolios'),
	('education.portfolios.request_corrections', 'Request corrections to professional portfolios'),
	('education.portfolios.archive_grants.manage', 'Manage portfolio archive attachment grants'),
	('education.portfolios.verify', 'Verify professional portfolios'),
	('education.portfolios.transfer', 'Transfer professional portfolios'),
	('education.portfolios.custody.manage', 'Manage professional portfolio custody')
on conflict (code) do update set label = excluded.label;

-- The teacher position is deliberately least-privilege.  It can only see and
-- edit its own portfolio; school-level review and custody stay separate.
insert into app_positions (code, name, scope_module, active, sort_order) values
	('profesor', 'Profesor', 'education', true, 25),
	('director_adjunct', 'Director adjunct', 'education', true, 26),
	('portfolio_reviewer', 'Verificator portofolii', 'education', true, 27),
	('portfolio_custodian', 'Responsabil custodie portofolii', 'education', true, 28)
on conflict (code) do update
set name = excluded.name, scope_module = excluded.scope_module,
	active = excluded.active, sort_order = excluded.sort_order, updated_at = now();

insert into app_position_permissions(position_code, permission_code) values
	('profesor', 'education.portfolios.read_own'),
	('profesor', 'education.portfolios.manage_own'),
	('director', 'education.portfolios.read_own'),
	('director', 'education.portfolios.manage_own'),
	('director', 'education.portfolios.school.read'),
	('director', 'education.portfolios.school.manage'),
	('director', 'education.portfolios.request_corrections'),
	('director', 'education.portfolios.archive_grants.manage'),
	('super_admin', 'education.portfolios.archive_grants.manage'),
	('e2e_canary', 'education.portfolios.archive_grants.manage'),
	('director', 'education.portfolios.verify'),
	('director', 'education.portfolios.transfer'),
	('director', 'education.portfolios.custody.manage'),
	('director_adjunct', 'education.portfolios.read_own'),
	('director_adjunct', 'education.portfolios.manage_own'),
	('director_adjunct', 'education.portfolios.school.read'),
	('portfolio_reviewer', 'education.portfolios.school.read'),
	('portfolio_reviewer', 'education.portfolios.request_corrections'),
	('portfolio_reviewer', 'education.portfolios.verify'),
	('portfolio_custodian', 'education.portfolios.school.read'),
	('portfolio_custodian', 'education.portfolios.archive_grants.manage'),
	('portfolio_custodian', 'education.portfolios.transfer'),
	('portfolio_custodian', 'education.portfolios.custody.manage'),
	('hr', 'education.portfolios.school.read'),
	('hr', 'education.portfolios.school.manage'),
	('hr', 'education.portfolios.transfer'),
	('secretariat', 'education.portfolios.school.read'),
	('secretariat', 'education.portfolios.custody.manage')
on conflict do nothing;

insert into app_role_permissions (role_code, permission_code) values
	('super_admin', 'education.portfolios.archive_grants.manage'),
	('e2e_canary', 'education.portfolios.archive_grants.manage')
on conflict do nothing;

-- Validation is a director/reviewer responsibility. Older catalog data gave
-- the generic secretariat role this authority; remove that over-broad grant.
delete from app_role_permissions
where role_code = 'secretar' and permission_code = 'education.portfolios.verify';
