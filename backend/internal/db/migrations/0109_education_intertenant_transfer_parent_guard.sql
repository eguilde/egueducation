-- Version 2 transfers are visible to the receiving tenant, while the source
-- portfolio correctly remains hidden by RLS. A security-definer trigger may
-- inspect only the immutable source route and parent lifecycle flags; it does
-- not return or disclose portfolio data.
do $$
begin
	if not exists (
		select 1 from pg_catalog.pg_roles
		where rolname = current_user and (rolsuper or rolbypassrls)
	) then
		raise exception 'migration owner must be SUPERUSER or BYPASSRLS for the FORCE-RLS transfer parent guard';
	end if;
end;
$$;

create or replace function public.enforce_education_intertenant_transfer_parent_guard()
returns trigger
language plpgsql
security definer
set search_path = pg_catalog, public
as $$
declare
	route_version integer;
	parent_withdrawn_at timestamptz;
	parent_legal_hold boolean;
begin
	if tg_op = 'DELETE' then
		if old.routing_version = 2 then
			raise exception 'inter-tenant portfolio transfer evidence cannot be deleted';
		end if;
		return old;
	end if;

	if tg_op = 'UPDATE' then
		if old.routing_version is distinct from new.routing_version then
			raise exception 'portfolio transfer routing version is immutable';
		end if;
		route_version := old.routing_version;
	else
		route_version := new.routing_version;
	end if;

	if route_version <> 2 then
		return new;
	end if;

	select portfolio.withdrawn_at, portfolio.legal_hold_active
	into parent_withdrawn_at, parent_legal_hold
	from public.education_portfolios portfolio
	join public.app_tenants tenant
		on tenant.code = case when tg_op = 'UPDATE' then old.source_tenant_code else new.source_tenant_code end
		and tenant.institution_id = portfolio.institution_id
	where portfolio.id = case when tg_op = 'UPDATE' then old.portfolio_id else new.portfolio_id end
		and portfolio.institution_id = case when tg_op = 'UPDATE' then old.source_institution_id else new.source_institution_id end
	for update of portfolio;

	if not found then
		raise exception 'inter-tenant transfer source portfolio does not exist in the declared route';
	end if;
	if parent_withdrawn_at is not null then
		raise exception 'withdrawn portfolio transfer records are immutable';
	end if;
	if parent_legal_hold then
		raise exception 'portfolio transfer mutation is blocked by legal hold';
	end if;
	return new;
end;
$$;

revoke all on function public.enforce_education_intertenant_transfer_parent_guard() from public;

-- The generic 0101 trigger must continue to protect local version 1 records.
-- It cannot run for version 2 destination updates because RLS intentionally
-- hides the source portfolio from the destination. The guarded function above
-- provides the same parent checks for v2 without disclosing the parent row.
drop trigger if exists trg_education_portfolio_transfer_lifecycle on public.education_portfolio_transfers;
drop trigger if exists trg_education_portfolio_transfer_lifecycle_insert on public.education_portfolio_transfers;
drop trigger if exists trg_education_portfolio_transfer_lifecycle_update on public.education_portfolio_transfers;
drop trigger if exists trg_education_portfolio_transfer_lifecycle_delete on public.education_portfolio_transfers;

create trigger trg_education_portfolio_transfer_lifecycle_insert
	before insert on public.education_portfolio_transfers
	for each row when (new.routing_version = 1)
	execute function public.enforce_education_portfolio_lifecycle_child();
create trigger trg_education_portfolio_transfer_lifecycle_update
	before update on public.education_portfolio_transfers
	for each row when (old.routing_version = 1 or new.routing_version = 1)
	execute function public.enforce_education_portfolio_lifecycle_child();
create trigger trg_education_portfolio_transfer_lifecycle_delete
	before delete on public.education_portfolio_transfers
	for each row when (old.routing_version = 1)
	execute function public.enforce_education_portfolio_lifecycle_child();

drop trigger if exists trg_education_intertenant_parent_guard on public.education_portfolio_transfers;
create trigger trg_education_intertenant_parent_guard
	before insert or update or delete on public.education_portfolio_transfers
	for each row execute function public.enforce_education_intertenant_transfer_parent_guard();
