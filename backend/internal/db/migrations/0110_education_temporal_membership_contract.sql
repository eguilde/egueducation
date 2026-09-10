-- Keep the database authorization boundary aligned with OIDC claim issuance:
-- a tenant membership is usable only while the account, tenant and dated
-- membership are all active. Security-definer callers need a migration owner
-- which can inspect FORCE-RLS identity tables.
do $$
begin
	if not exists (
		select 1 from pg_catalog.pg_roles
		where rolname = current_user and (rolsuper or rolbypassrls)
	) then
		raise exception 'migration owner must be SUPERUSER or BYPASSRLS for the FORCE-RLS education membership contract';
	end if;
end;
$$;

create or replace function public.education_membership_is_eligible(
	p_user_id uuid,
	p_tenant_code text default null,
	p_institution_id text default null,
	p_position_code text default null
)
returns boolean
language plpgsql
stable
security definer
set search_path = pg_catalog, public
as $$
declare
	context_tenant text := public.current_tenant_code();
	context_institution text := public.current_institution_id();
	session_can_bypass boolean;
begin
	select coalesce(role_row.rolsuper or role_row.rolbypassrls, false)
	into session_can_bypass
	from pg_catalog.pg_roles role_row
	where role_row.rolname = session_user;

	-- This function may be called by RLS, but it must not become a directory
	-- oracle for another tenant or institution.
	if not coalesce(session_can_bypass, false) and not public.can_bypass_tenant_rls() then
		if context_tenant is null
			or (p_tenant_code is not null and p_tenant_code is distinct from context_tenant)
			or (p_institution_id is not null and p_institution_id is distinct from context_institution) then
			return false;
		end if;
	end if;

	return exists (
		select 1
		from public.app_memberships membership
		join public.app_users user_row
			on user_row.id = membership.user_id
		join public.app_tenants tenant
			on tenant.code = membership.tenant_code
		where membership.user_id = p_user_id
			and user_row.status = 'active'
			and tenant.active
			and membership.active
			and membership.start_date <= current_date
			and (membership.end_date is null or membership.end_date >= current_date)
			and (p_tenant_code is null or membership.tenant_code = p_tenant_code)
			and (p_institution_id is null or tenant.institution_id = p_institution_id)
			and (p_position_code is null or membership.position_code = p_position_code)
			and (
				coalesce(session_can_bypass, false)
				or public.can_bypass_tenant_rls()
				or membership.tenant_code = context_tenant
			)
	);
end;
$$;

revoke all on function public.education_membership_is_eligible(uuid, text, text, text) from public;
grant execute on function public.education_membership_is_eligible(uuid, text, text, text) to public;

comment on function public.education_membership_is_eligible(uuid, text, text, text) is
	'Canonical dated School membership predicate shared by database RBAC boundaries.';

-- This trigger deliberately runs before the detailed 0106 evidentiary state
-- machine (the trigger name sorts first). It closes temporal gaps without
-- rewriting an already-published migration.
create or replace function public.enforce_education_role_delegation_temporal_eligibility()
returns trigger
language plpgsql
security definer
set search_path = pg_catalog, public
as $$
declare
	actor_subject text := nullif(btrim(current_setting('app.actor_subject', true)), '');
	actor_id uuid;
	scope_tenant text := case when tg_op = 'DELETE' then old.tenant_code else new.tenant_code end;
	scope_institution text := case when tg_op = 'DELETE' then old.institution_id else new.institution_id end;
begin
	if tg_op = 'DELETE' then
		return old;
	end if;

	select user_row.id into actor_id
	from public.app_users user_row
	where lower(user_row.sub) = lower(actor_subject)
		and user_row.status = 'active'
	limit 1;
	if actor_id is null
		or not public.education_membership_is_eligible(actor_id, scope_tenant, scope_institution, null) then
		raise exception 'delegation actor has no currently eligible tenant membership';
	end if;

	if tg_op = 'INSERT' then
		if actor_id <> new.delegator_user_id
			or not public.education_membership_is_eligible(new.delegator_user_id, new.tenant_code, new.institution_id, 'director') then
			raise exception 'only a currently eligible director can offer a delegation';
		end if;
		if not public.education_membership_is_eligible(new.delegate_user_id, new.tenant_code, new.institution_id, 'director_adjunct') then
			raise exception 'delegation recipient must be a currently eligible director adjunct';
		end if;
		return new;
	end if;

	if old.status = 'offered' and new.status = 'accepted' then
		if actor_id <> old.delegate_user_id
			or not public.education_membership_is_eligible(old.delegate_user_id, old.tenant_code, old.institution_id, 'director_adjunct')
			or not public.education_membership_is_eligible(old.delegator_user_id, old.tenant_code, old.institution_id, 'director') then
			raise exception 'delegation parties are no longer eligible for acceptance';
		end if;
	elsif old.status in ('offered', 'accepted') and new.status = 'revoked' then
		if actor_id <> old.delegator_user_id
			or not public.education_membership_is_eligible(old.delegator_user_id, old.tenant_code, old.institution_id, 'director') then
			raise exception 'only a currently eligible delegating director can revoke a delegation';
		end if;
	elsif old.status = 'accepted' and new.status = 'expired' then
		if actor_id not in (old.delegator_user_id, old.delegate_user_id)
			or not public.education_membership_is_eligible(actor_id, old.tenant_code, old.institution_id, null) then
			raise exception 'only a currently eligible delegation party can record expiry';
		end if;
	end if;
	return new;
end;
$$;

revoke all on function public.enforce_education_role_delegation_temporal_eligibility() from public;
drop trigger if exists trg_education_role_delegations_00_temporal_guard on public.education_role_delegations;
create trigger trg_education_role_delegations_00_temporal_guard
	before insert or update or delete on public.education_role_delegations
	for each row execute function public.enforce_education_role_delegation_temporal_eligibility();

drop policy if exists tenant_isolation on public.education_role_delegations;
create policy tenant_isolation on public.education_role_delegations
	using (
		public.can_bypass_tenant_rls()
		or (
			tenant_code = public.current_tenant_code()
			and institution_id = public.current_institution_id()
			and exists (
				select 1
				from public.app_users actor
				where lower(actor.sub) = lower(nullif(btrim(current_setting('app.actor_subject', true)), ''))
					and actor.status = 'active'
					and public.education_membership_is_eligible(actor.id, tenant_code, institution_id, null)
					and (
						actor.id = delegator_user_id
						or actor.id = delegate_user_id
						or public.education_membership_is_eligible(actor.id, tenant_code, institution_id, 'director')
					)
			)
		)
	)
	with check (
		public.can_bypass_tenant_rls()
		or (tenant_code = public.current_tenant_code() and institution_id = public.current_institution_id())
	);

-- Replace the 0105 permission predicate with the same dated membership gate
-- used by authentication and delegation. Direct roles/permissions remain
-- valid only while the account has an eligible membership in this institution.
create or replace function public.education_actor_can_transfer_portfolio()
returns boolean
language sql
stable
security definer
set search_path = pg_catalog, public
as $$
	with actor as (
		select user_row.id
		from public.app_users user_row
		where lower(user_row.sub) = lower(nullif(btrim(current_setting('app.actor_subject', true)), ''))
			and user_row.status = 'active'
		limit 1
	)
	select nullif(public.current_tenant_code(), '') is not null
		and nullif(public.current_institution_id(), '') is not null
		and exists (
			select 1 from actor
			where public.education_membership_is_eligible(
				actor.id,
				public.current_tenant_code(),
				public.current_institution_id(),
				null
			)
		)
		and exists (
			select 1
			from actor
			join lateral (
				select up.permission_code
				from public.app_user_permissions up
				where up.user_id = actor.id and up.tenant_code = public.current_tenant_code()
				union
				select rp.permission_code
				from public.app_user_roles ur
				join public.app_role_permissions rp on rp.role_code = ur.role_code
				where ur.user_id = actor.id and ur.tenant_code = public.current_tenant_code()
				union
				select pp.permission_code
				from public.app_memberships membership
				join public.app_position_permissions pp on pp.position_code = membership.position_code
				where membership.user_id = actor.id
					and membership.tenant_code = public.current_tenant_code()
					and public.education_membership_is_eligible(
						actor.id, membership.tenant_code, public.current_institution_id(), membership.position_code
					)
				union
				select rp.permission_code
				from public.app_memberships membership
				join public.app_position_roles pr on pr.position_code = membership.position_code
				join public.app_role_permissions rp on rp.role_code = pr.role_code
				where membership.user_id = actor.id
					and membership.tenant_code = public.current_tenant_code()
					and public.education_membership_is_eligible(
						actor.id, membership.tenant_code, public.current_institution_id(), membership.position_code
					)
			) effective_permissions on true
			where effective_permissions.permission_code = 'education.portfolios.transfer'
		)
$$;

