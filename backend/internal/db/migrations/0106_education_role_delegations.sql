-- Delegation is an institution-scoped, evidentiary authorization grant.  It
-- deliberately augments request-time authorization only: delegated rights are
-- never materialized into a user's tenant roles or JWT claims.

create table if not exists education_role_delegations (
	id uuid primary key default gen_random_uuid(),
	tenant_code text not null,
	institution_id text not null,
	delegator_user_id uuid not null references app_users(id) on delete restrict,
	delegate_user_id uuid not null references app_users(id) on delete restrict,
	permission_code text not null references app_permissions(code) on delete restrict,
	resource_type text not null default 'institution',
	resource_id uuid,
	status text not null default 'offered',
	valid_from date not null default current_date,
	valid_until date,
	offered_by_user_id uuid not null references app_users(id) on delete restrict,
	offered_at timestamptz not null default now(),
	accepted_by_user_id uuid references app_users(id) on delete restrict,
	accepted_at timestamptz,
	revoked_by_user_id uuid references app_users(id) on delete restrict,
	revoked_at timestamptz,
	expired_by_user_id uuid references app_users(id) on delete restrict,
	expired_at timestamptz,
	notes text not null default '',
	updated_at timestamptz not null default now(),
	constraint education_role_delegations_tenant_institution_fk
		foreign key (tenant_code, institution_id)
		references app_tenants(code, institution_id) on delete restrict,
	constraint education_role_delegations_not_self
		check (delegator_user_id <> delegate_user_id),
	constraint education_role_delegations_education_permission
		check (permission_code like 'education.%'),
	constraint education_role_delegations_resource_scope
		check (
			(resource_type = 'institution' and resource_id is null)
			or (resource_type in ('portfolio', 'meeting', 'decision', 'regulation', 'personnel') and resource_id is not null)
		),
	constraint education_role_delegations_validity
		check (valid_until is null or valid_until >= valid_from),
	constraint education_role_delegations_status
		check (status in ('offered', 'accepted', 'revoked', 'expired')),
	constraint education_role_delegations_provenance
		check (
			(status = 'offered' and accepted_at is null and accepted_by_user_id is null and revoked_at is null and revoked_by_user_id is null and expired_at is null and expired_by_user_id is null)
			or (status = 'accepted' and accepted_at is not null and accepted_by_user_id = delegate_user_id and revoked_at is null and revoked_by_user_id is null and expired_at is null and expired_by_user_id is null)
			or (status = 'revoked' and revoked_at is not null and revoked_by_user_id is not null and expired_at is null and expired_by_user_id is null)
			or (status = 'expired' and expired_at is not null and expired_by_user_id is not null and revoked_at is null and revoked_by_user_id is null)
		)
);

create unique index if not exists uq_education_role_delegations_open_scope
	on education_role_delegations (
		tenant_code,
		institution_id,
		delegator_user_id,
		delegate_user_id,
		permission_code,
		resource_type,
		coalesce(resource_id, '00000000-0000-0000-0000-000000000000'::uuid)
	)
	where status in ('offered', 'accepted');

create index if not exists idx_education_role_delegations_delegate_active
	on education_role_delegations (tenant_code, institution_id, delegate_user_id, permission_code, valid_from, valid_until)
	where status = 'accepted';

-- This is intentionally security-definer because app_tenants is FORCE RLS;
-- it is only invoked by the trigger below and derives identity from the
-- request session rather than from caller-provided columns.
create or replace function public.enforce_education_role_delegation()
returns trigger
language plpgsql
security definer
set search_path = pg_catalog, public
as $$
declare
	context_tenant text := public.current_tenant_code();
	context_institution text := public.current_institution_id();
	actor_subject text := nullif(btrim(current_setting('app.actor_subject', true)), '');
	actor_id uuid;
	actor_is_director boolean;
	delegate_is_adjunct boolean;
	actor_has_permission boolean;
begin
	if tg_op = 'DELETE' then
		raise exception 'education role delegations are evidentiary records and cannot be hard-deleted';
	end if;
	if actor_subject is null or context_tenant is null or context_institution is null then
		raise exception 'authenticated tenant and institution context is required for role delegation';
	end if;

	select u.id
	into actor_id
	from public.app_users u
	join public.app_memberships m on m.user_id = u.id
	join public.app_tenants tenant on tenant.code = m.tenant_code
	where lower(u.sub) = lower(actor_subject)
		and m.tenant_code = context_tenant
		and tenant.institution_id = context_institution
		and tenant.active
		and m.active
		and (m.end_date is null or m.end_date >= current_date)
	limit 1;
	if actor_id is null then
		raise exception 'delegation actor has no active tenant membership';
	end if;

	if tg_op = 'INSERT' then
		if new.status <> 'offered' then
			raise exception 'role delegation must start as offered';
		end if;
		if new.tenant_code is distinct from context_tenant or new.institution_id is distinct from context_institution then
			raise exception 'role delegation must remain in the authenticated tenant and institution';
		end if;
		if new.delegator_user_id <> actor_id then
			raise exception 'only the authenticated director can offer a delegation';
		end if;
		select exists(
			select 1 from public.app_memberships m
			where m.user_id = actor_id
				and m.tenant_code = context_tenant
				and m.position_code = 'director'
				and m.active
				and (m.end_date is null or m.end_date >= current_date)
		) into actor_is_director;
		if not actor_is_director then
			raise exception 'only an active director can offer a delegation';
		end if;
		select exists(
			select 1 from public.app_memberships m
			where m.user_id = new.delegate_user_id
				and m.tenant_code = context_tenant
				and m.position_code = 'director_adjunct'
				and m.active
				and (m.end_date is null or m.end_date >= current_date)
		) into delegate_is_adjunct;
		if not delegate_is_adjunct then
			raise exception 'delegation recipient must be an active director adjunct in the same tenant';
		end if;
		select exists(
			select 1
			from (
				select up.permission_code
				from public.app_user_permissions up
				where up.user_id = actor_id and up.tenant_code = context_tenant
				union
				select rp.permission_code
				from public.app_user_roles ur
				join public.app_role_permissions rp on rp.role_code = ur.role_code
				where ur.user_id = actor_id and ur.tenant_code = context_tenant
				union
				select pp.permission_code
				from public.app_memberships m
				join public.app_position_permissions pp on pp.position_code = m.position_code
				where m.user_id = actor_id and m.tenant_code = context_tenant and m.active
					and (m.end_date is null or m.end_date >= current_date)
				union
				select rp.permission_code
				from public.app_memberships m
				join public.app_position_roles pr on pr.position_code = m.position_code
				join public.app_role_permissions rp on rp.role_code = pr.role_code
				where m.user_id = actor_id and m.tenant_code = context_tenant and m.active
					and (m.end_date is null or m.end_date >= current_date)
			) effective_permissions
			where permission_code = new.permission_code
		) into actor_has_permission;
		if not actor_has_permission then
			raise exception 'director can delegate only a permission currently held';
		end if;
		-- Resource scope is a database contract, not merely an HTTP validation.
		-- This prevents direct SQL callers from manufacturing a grant for an
		-- object that is absent or belongs to another institution.
		if new.resource_type = 'portfolio' and not exists (
			select 1 from public.education_portfolios resource
			where resource.id = new.resource_id and resource.institution_id = context_institution
		) then
			raise exception 'delegated portfolio does not exist in the active institution';
		elsif new.resource_type = 'meeting' and not exists (
			select 1 from public.education_meetings resource
			where resource.id = new.resource_id and resource.institution_id = context_institution
		) then
			raise exception 'delegated meeting does not exist in the active institution';
		elsif new.resource_type = 'decision' and not exists (
			select 1 from public.education_decisions resource
			where resource.id = new.resource_id and resource.institution_id = context_institution
		) then
			raise exception 'delegated decision does not exist in the active institution';
		elsif new.resource_type = 'regulation' and not exists (
			select 1 from public.education_regulations resource
			where resource.id = new.resource_id and resource.institution_id = context_institution
		) then
			raise exception 'delegated regulation does not exist in the active institution';
		elsif new.resource_type = 'personnel' and not exists (
			select 1 from public.education_personnel resource
			where resource.id = new.resource_id and resource.institution_id = context_institution
		) then
			raise exception 'delegated personnel record does not exist in the active institution';
		end if;
		new.offered_by_user_id := actor_id;
		new.offered_at := statement_timestamp();
		new.accepted_by_user_id := null;
		new.accepted_at := null;
		new.revoked_by_user_id := null;
		new.revoked_at := null;
		new.expired_by_user_id := null;
		new.expired_at := null;
		new.updated_at := statement_timestamp();
		return new;
	end if;

	if row(new.tenant_code, new.institution_id, new.delegator_user_id, new.delegate_user_id,
		new.permission_code, new.resource_type, new.resource_id, new.valid_from, new.valid_until,
		new.offered_by_user_id, new.offered_at, new.notes)
		is distinct from row(old.tenant_code, old.institution_id, old.delegator_user_id, old.delegate_user_id,
		old.permission_code, old.resource_type, old.resource_id, old.valid_from, old.valid_until,
		old.offered_by_user_id, old.offered_at, old.notes) then
		raise exception 'role delegation scope and offer evidence are immutable';
	end if;
	if old.status = 'offered' and new.status = 'accepted' then
		if actor_id <> old.delegate_user_id then
			raise exception 'only the designated director adjunct can accept a delegation';
		end if;
		if old.valid_until is not null and old.valid_until < current_date then
			raise exception 'an expired delegation offer cannot be accepted';
		end if;
		select exists(
			select 1 from public.app_memberships m
			where m.user_id = actor_id and m.tenant_code = context_tenant
				and m.position_code = 'director_adjunct' and m.active
				and (m.end_date is null or m.end_date >= current_date)
		) into delegate_is_adjunct;
		if not delegate_is_adjunct then
			raise exception 'only an active director adjunct can accept a delegation';
		end if;
		new.accepted_by_user_id := actor_id;
		new.accepted_at := statement_timestamp();
		new.revoked_by_user_id := null;
		new.revoked_at := null;
		new.expired_by_user_id := null;
		new.expired_at := null;
	elsif old.status in ('offered', 'accepted') and new.status = 'revoked' then
		if actor_id <> old.delegator_user_id then
			raise exception 'only the delegating director can revoke a delegation';
		end if;
		select exists(
			select 1 from public.app_memberships m
			where m.user_id = actor_id and m.tenant_code = context_tenant
				and m.position_code = 'director' and m.active
				and (m.end_date is null or m.end_date >= current_date)
		) into actor_is_director;
		if not actor_is_director then
			raise exception 'only an active director can revoke a delegation';
		end if;
		new.accepted_by_user_id := old.accepted_by_user_id;
		new.accepted_at := old.accepted_at;
		new.revoked_by_user_id := actor_id;
		new.revoked_at := statement_timestamp();
		new.expired_by_user_id := null;
		new.expired_at := null;
	elsif old.status = 'accepted' and new.status = 'expired' then
		if old.valid_until is null or old.valid_until >= current_date then
			raise exception 'a delegation can be marked expired only after its validity end';
		end if;
		if actor_id <> old.delegator_user_id and actor_id <> old.delegate_user_id then
			raise exception 'only a delegation party can record expiry';
		end if;
		new.accepted_by_user_id := old.accepted_by_user_id;
		new.accepted_at := old.accepted_at;
		new.expired_by_user_id := actor_id;
		new.expired_at := statement_timestamp();
		new.revoked_by_user_id := null;
		new.revoked_at := null;
	else
		raise exception 'invalid role delegation state transition';
	end if;
	new.updated_at := statement_timestamp();
	return new;
end;
$$;

drop trigger if exists trg_education_role_delegations_guard on education_role_delegations;
create trigger trg_education_role_delegations_guard
	before insert or update or delete on education_role_delegations
	for each row execute function public.enforce_education_role_delegation();

alter table education_role_delegations enable row level security;
alter table education_role_delegations force row level security;
drop policy if exists tenant_isolation on education_role_delegations;
create policy tenant_isolation on education_role_delegations
	using (
		public.can_bypass_tenant_rls()
		or (
			tenant_code = public.current_tenant_code()
			and institution_id = public.current_institution_id()
			and (
				delegator_user_id = (select id from app_users where lower(sub) = lower(nullif(btrim(current_setting('app.actor_subject', true)), '')) limit 1)
				or delegate_user_id = (select id from app_users where lower(sub) = lower(nullif(btrim(current_setting('app.actor_subject', true)), '')) limit 1)
				or exists (
					select 1 from app_memberships membership
					join app_users actor on actor.id = membership.user_id
					where lower(actor.sub) = lower(nullif(btrim(current_setting('app.actor_subject', true)), ''))
						and membership.tenant_code = public.current_tenant_code()
						and membership.position_code = 'director' and membership.active
						and (membership.end_date is null or membership.end_date >= current_date)
				)
			)
		)
	)
	with check (
		public.can_bypass_tenant_rls()
		or (tenant_code = public.current_tenant_code() and institution_id = public.current_institution_id())
	);

insert into app_permissions(code, label) values
	('education.delegations.read', 'Read own or institution education delegations'),
	('education.delegations.offer', 'Offer director authority to a director adjunct'),
	('education.delegations.accept', 'Accept offered director authority'),
	('education.delegations.revoke', 'Revoke or record expiry of delegated director authority')
on conflict (code) do update set label=excluded.label;

insert into app_position_permissions(position_code, permission_code) values
	('director', 'education.delegations.read'),
	('director', 'education.delegations.offer'),
	('director', 'education.delegations.revoke'),
	('director_adjunct', 'education.delegations.read'),
	('director_adjunct', 'education.delegations.accept'),
	('director_adjunct', 'education.delegations.revoke')
on conflict do nothing;

drop trigger if exists trg_education_role_delegations_entity_version on education_role_delegations;
create trigger trg_education_role_delegations_entity_version
	after insert or update or delete on education_role_delegations
	for each row execute function public.record_entity_version();
