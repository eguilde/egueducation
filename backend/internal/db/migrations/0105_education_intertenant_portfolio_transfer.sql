-- A portfolio transfer crosses an institutional trust boundary.  The legacy
-- table kept free-text institution labels and allowed the source institution
-- to mark its own record as received.  Version 2 routing binds both ends to
-- active tenants, seals an authoritative export manifest before sending and
-- requires the destination tenant to record receipt with its own actor.

create unique index if not exists uq_app_tenants_code_institution
	on app_tenants (code, institution_id);

alter table education_portfolio_export_manifests
	drop constraint if exists uq_education_portfolio_export_manifest_route;
alter table education_portfolio_export_manifests
	add constraint uq_education_portfolio_export_manifest_route
	unique (id, tenant_code, institution_id, portfolio_id);

alter table education_portfolio_transfers
	add column if not exists routing_version integer not null default 1,
	add column if not exists source_tenant_code text,
	add column if not exists source_institution_id text,
	add column if not exists destination_tenant_code text,
	add column if not exists destination_institution_id text,
	add column if not exists export_manifest_id uuid,
	add column if not exists sent_at timestamptz,
	add column if not exists sent_by_subject text not null default '',
	add column if not exists received_at timestamptz,
	add column if not exists received_by_subject text not null default '',
	add column if not exists closed_at timestamptz,
	add column if not exists closed_by_subject text not null default '';

update education_portfolio_transfers transfer
set source_tenant_code = tenant.code,
	source_institution_id = tenant.institution_id
from app_tenants tenant
where transfer.institution_id = tenant.institution_id
	and (transfer.source_tenant_code is null or transfer.source_institution_id is null);

update education_portfolio_transfers transfer
set destination_tenant_code = tenant.code,
	destination_institution_id = tenant.institution_id
from app_tenants tenant
where transfer.destination_tenant_code is null
	and transfer.destination_institution_id is null
	and (
		lower(btrim(transfer.destination_institution)) = lower(btrim(tenant.display_name))
		or lower(btrim(transfer.destination_institution)) = lower(btrim(tenant.short_name))
		or lower(btrim(transfer.destination_institution)) = lower(btrim(tenant.institution_id))
	);

alter table education_portfolio_transfers alter column routing_version set default 2;
alter table education_portfolio_transfers drop constraint if exists education_portfolio_transfers_routing_version_check;
alter table education_portfolio_transfers add constraint education_portfolio_transfers_routing_version_check
	check (routing_version in (1, 2));
alter table education_portfolio_transfers drop constraint if exists education_portfolio_transfers_v2_route_check;
alter table education_portfolio_transfers add constraint education_portfolio_transfers_v2_route_check check (
	routing_version = 1 or (
		source_tenant_code is not null and btrim(source_tenant_code) <> ''
		and source_institution_id is not null and btrim(source_institution_id) <> ''
		and destination_tenant_code is not null and btrim(destination_tenant_code) <> ''
		and destination_institution_id is not null and btrim(destination_institution_id) <> ''
		and source_tenant_code <> destination_tenant_code
		and source_institution_id <> destination_institution_id
		and institution_id = source_institution_id
	)
);
alter table education_portfolio_transfers drop constraint if exists education_portfolio_transfers_v2_evidence_check;
alter table education_portfolio_transfers add constraint education_portfolio_transfers_v2_evidence_check check (
	routing_version = 1 or (
		(status = 'pregatit' or (export_manifest_id is not null and sent_at is not null and btrim(sent_by_subject) <> ''))
		and (status not in ('receptionat', 'inchis') or (received_at is not null and received_on is not null and btrim(received_by_subject) <> ''))
		and (status <> 'inchis' or (closed_at is not null and btrim(closed_by_subject) <> ''))
	)
);

alter table education_portfolio_transfers drop constraint if exists education_portfolio_transfers_source_tenant_fk;
alter table education_portfolio_transfers add constraint education_portfolio_transfers_source_tenant_fk
	foreign key (source_tenant_code, source_institution_id)
	references app_tenants (code, institution_id) on delete restrict;
alter table education_portfolio_transfers drop constraint if exists education_portfolio_transfers_destination_tenant_fk;
alter table education_portfolio_transfers add constraint education_portfolio_transfers_destination_tenant_fk
	foreign key (destination_tenant_code, destination_institution_id)
	references app_tenants (code, institution_id) on delete restrict;
alter table education_portfolio_transfers drop constraint if exists education_portfolio_transfers_export_manifest_fk;
alter table education_portfolio_transfers add constraint education_portfolio_transfers_export_manifest_fk
	foreign key (export_manifest_id, source_tenant_code, source_institution_id, portfolio_id)
	references education_portfolio_export_manifests (id, tenant_code, institution_id, portfolio_id) on delete restrict;

create index if not exists idx_education_portfolio_transfers_destination_inbox
	on education_portfolio_transfers (destination_tenant_code, destination_institution_id, status, sent_at desc)
	where routing_version = 2 and withdrawn_at is null;

-- Keep a minimal routing projection separate from the protected tenant
-- configuration table. The application role cannot select this table
-- directly; only the permission-checked function below can disclose rows.
create table if not exists public.education_portfolio_transfer_destination_directory (
	tenant_code text primary key references public.app_tenants(code) on delete cascade,
	institution_id text not null,
	display_name text not null,
	short_name text not null,
	active boolean not null,
	updated_at timestamptz not null default now()
);

-- Migrations execute transactionally as the table owner. Temporarily remove
-- FORCE (not RLS itself) so the owner can backfill the projection, then
-- restore FORCE before the migration can commit.
alter table public.app_tenants no force row level security;
insert into public.education_portfolio_transfer_destination_directory (
	tenant_code, institution_id, display_name, short_name, active, updated_at
)
select code, institution_id, display_name, short_name, active, now()
from public.app_tenants
on conflict (tenant_code) do update set
	institution_id = excluded.institution_id,
	display_name = excluded.display_name,
	short_name = excluded.short_name,
	active = excluded.active,
	updated_at = excluded.updated_at;
alter table public.app_tenants force row level security;

create or replace function public.sync_education_portfolio_transfer_destination_directory()
returns trigger
language plpgsql
security definer
set search_path = pg_catalog, public
as $$
begin
	insert into public.education_portfolio_transfer_destination_directory (
		tenant_code, institution_id, display_name, short_name, active, updated_at
	) values (new.code, new.institution_id, new.display_name, new.short_name, new.active, statement_timestamp())
	on conflict (tenant_code) do update set
		institution_id = excluded.institution_id,
		display_name = excluded.display_name,
		short_name = excluded.short_name,
		active = excluded.active,
		updated_at = excluded.updated_at;
	return new;
end;
$$;

drop trigger if exists trg_sync_education_portfolio_transfer_destination_directory on public.app_tenants;
create trigger trg_sync_education_portfolio_transfer_destination_directory
	after insert or update of institution_id, display_name, short_name, active on public.app_tenants
	for each row execute function public.sync_education_portfolio_transfer_destination_directory();

revoke all on public.education_portfolio_transfer_destination_directory from public;
revoke all on function public.sync_education_portfolio_transfer_destination_directory() from public;

-- The security-definer boundary performs the same effective-permission union
-- used by OIDC token issuance. It is also the RLS predicate, so even an
-- explicit table grant cannot bypass the portfolio-transfer permission.
create or replace function public.education_actor_can_transfer_portfolio()
returns boolean
language sql
stable
security definer
set search_path = pg_catalog, public
as $$
	select nullif(public.current_tenant_code(), '') is not null
		and nullif(btrim(current_setting('app.actor_subject', true)), '') is not null
		and exists (
			select 1
			from (
				select up.permission_code
				from public.app_user_permissions up
				join public.app_users u on u.id = up.user_id
				where lower(u.sub) = lower(nullif(btrim(current_setting('app.actor_subject', true)), ''))
					and up.tenant_code = public.current_tenant_code()
				union
				select rp.permission_code
				from public.app_user_roles ur
				join public.app_users u on u.id = ur.user_id
				join public.app_role_permissions rp on rp.role_code = ur.role_code
				where lower(u.sub) = lower(nullif(btrim(current_setting('app.actor_subject', true)), ''))
					and ur.tenant_code = public.current_tenant_code()
				union
				select pp.permission_code
				from public.app_memberships membership
				join public.app_users u on u.id = membership.user_id
				join public.app_position_permissions pp on pp.position_code = membership.position_code
				where lower(u.sub) = lower(nullif(btrim(current_setting('app.actor_subject', true)), ''))
					and membership.active and membership.tenant_code = public.current_tenant_code()
				union
				select rp.permission_code
				from public.app_memberships membership
				join public.app_users u on u.id = membership.user_id
				join public.app_position_roles pr on pr.position_code = membership.position_code
				join public.app_role_permissions rp on rp.role_code = pr.role_code
				where lower(u.sub) = lower(nullif(btrim(current_setting('app.actor_subject', true)), ''))
					and membership.active and membership.tenant_code = public.current_tenant_code()
			) effective_permissions
			where permission_code = 'education.portfolios.transfer'
		)
$$;

alter table public.education_portfolio_transfer_destination_directory enable row level security;
alter table public.education_portfolio_transfer_destination_directory force row level security;
drop policy if exists education_portfolio_transfer_destination_read on public.education_portfolio_transfer_destination_directory;
create policy education_portfolio_transfer_destination_read on public.education_portfolio_transfer_destination_directory
	for select
	to public
	using (active and public.education_actor_can_transfer_portfolio());
drop policy if exists education_portfolio_transfer_destination_maintenance on public.education_portfolio_transfer_destination_directory;
create policy education_portfolio_transfer_destination_maintenance on public.education_portfolio_transfer_destination_directory
	for all
	to public
	using (public.can_bypass_tenant_rls())
	with check (public.can_bypass_tenant_rls());

create or replace function public.education_portfolio_transfer_destinations()
returns table (tenant_code text, institution_id text, display_name text, short_name text)
language plpgsql
stable
security definer
set search_path = pg_catalog, public
as $$
declare
	context_tenant text := public.current_tenant_code();
begin
	if not public.education_actor_can_transfer_portfolio() then
		raise insufficient_privilege using message = 'portfolio transfer destination directory requires education.portfolios.transfer';
	end if;

	return query
	select tenant.tenant_code, tenant.institution_id, tenant.display_name, tenant.short_name
	from public.education_portfolio_transfer_destination_directory tenant
	where tenant.active and tenant.tenant_code <> context_tenant
	order by lower(tenant.display_name), tenant.tenant_code;
end;
$$;

comment on function public.education_portfolio_transfer_destinations() is
	'Permission-checked minimal active-tenant routing directory for inter-tenant portfolio transfer.';

create or replace function public.enforce_education_intertenant_portfolio_transfer()
returns trigger
language plpgsql
as $$
declare
	actor_subject text := nullif(btrim(current_setting('app.actor_subject', true)), '');
	context_tenant text := public.current_tenant_code();
	context_institution text := public.current_institution_id();
	is_bypass boolean := public.can_bypass_tenant_rls();
begin
	if coalesce(case when tg_op = 'DELETE' then old.routing_version else new.routing_version end, 1) = 1 then
		return case when tg_op = 'DELETE' then old else new end;
	end if;

	if tg_op = 'DELETE' then
		raise exception 'inter-tenant portfolio transfer evidence cannot be deleted';
	end if;
	if actor_subject is null and not is_bypass then
		raise exception 'authenticated actor is required for an inter-tenant portfolio transfer';
	end if;

	if tg_op = 'INSERT' then
		if new.status <> 'pregatit' then
			raise exception 'inter-tenant portfolio transfer must start as prepared';
		end if;
		if not is_bypass and (context_tenant <> new.source_tenant_code or context_institution <> new.source_institution_id) then
			raise exception 'only the source institution can prepare a portfolio transfer';
		end if;
		select source.display_name, destination.display_name
		into new.source_institution, new.destination_institution
		from app_tenants source
		join app_tenants destination
			on destination.code = new.destination_tenant_code
			and destination.institution_id = new.destination_institution_id
			and destination.active
		where source.code = new.source_tenant_code
			and source.institution_id = new.source_institution_id
			and source.active;
		if not found then
			raise exception 'inter-tenant portfolio transfer route does not identify active tenants';
		end if;
		if not is_bypass then
			new.handover_by := actor_subject;
		end if;
		new.export_manifest_id := null;
		new.sent_at := null;
		new.sent_by_subject := '';
		new.received_on := null;
		new.received_by := '';
		new.received_at := null;
		new.received_by_subject := '';
		new.closed_at := null;
		new.closed_by_subject := '';
		new.withdrawn_at := null;
		new.withdrawn_by_subject := '';
		new.withdrawal_reason := '';
		new.updated_at := statement_timestamp();
		return new;
	end if;

	if row(new.portfolio_id, new.routing_version, new.source_tenant_code, new.source_institution_id,
		new.destination_tenant_code, new.destination_institution_id, new.transfer_code)
		is distinct from
		row(old.portfolio_id, old.routing_version, old.source_tenant_code, old.source_institution_id,
		old.destination_tenant_code, old.destination_institution_id, old.transfer_code) then
		raise exception 'inter-tenant portfolio transfer route is immutable';
	end if;

	if old.status = 'pregatit' and new.status = 'pregatit' then
		if not is_bypass and (context_tenant <> old.source_tenant_code or context_institution <> old.source_institution_id) then
			raise exception 'only the source institution can edit a prepared portfolio transfer';
		end if;
		if row(new.transfer_code, new.source_institution, new.destination_institution, new.received_on,
			new.handover_by, new.received_by, new.institution_id, new.created_at,
			new.export_manifest_id, new.sent_at, new.sent_by_subject, new.received_at,
			new.received_by_subject, new.closed_at, new.closed_by_subject)
			is distinct from row(old.transfer_code, old.source_institution, old.destination_institution, old.received_on,
			old.handover_by, old.received_by, old.institution_id, old.created_at,
			old.export_manifest_id, old.sent_at, old.sent_by_subject, old.received_at,
			old.received_by_subject, old.closed_at, old.closed_by_subject) then
			raise exception 'prepared transfer identity and evidence are server controlled';
		end if;
		if new.withdrawn_at is distinct from old.withdrawn_at then
			if old.withdrawn_at is not null or new.withdrawn_at is null or btrim(new.withdrawal_reason) = '' then
				raise exception 'invalid prepared transfer withdrawal';
			end if;
			if not is_bypass then
				new.withdrawn_by_subject := actor_subject;
			end if;
		elsif row(new.withdrawn_by_subject, new.withdrawal_reason)
			is distinct from row(old.withdrawn_by_subject, old.withdrawal_reason) then
			raise exception 'withdrawal provenance is server controlled';
		end if;
		new.updated_at := statement_timestamp();
		return new;
	end if;

	if old.status = 'pregatit' and new.status = 'trimis' then
		if not is_bypass and (context_tenant <> old.source_tenant_code or context_institution <> old.source_institution_id) then
			raise exception 'only the source institution can send a portfolio transfer';
		end if;
		if row(new.transfer_type, new.source_institution, new.destination_institution, new.handover_on,
			new.received_on, new.handover_by, new.received_by, new.institution_id, new.notes,
			new.created_at, new.withdrawn_at, new.withdrawn_by_subject, new.withdrawal_reason,
			new.received_at, new.received_by_subject, new.closed_at, new.closed_by_subject)
			is distinct from row(old.transfer_type, old.source_institution, old.destination_institution, old.handover_on,
			old.received_on, old.handover_by, old.received_by, old.institution_id, old.notes,
			old.created_at, old.withdrawn_at, old.withdrawn_by_subject, old.withdrawal_reason,
			old.received_at, old.received_by_subject, old.closed_at, old.closed_by_subject) then
			raise exception 'prepared portfolio transfer package is immutable while sending';
		end if;
		if new.export_manifest_id is null then
			raise exception 'sent portfolio transfer requires sealed manifest and sender provenance';
		end if;
		if not is_bypass then
			new.sent_at := statement_timestamp();
			new.sent_by_subject := actor_subject;
		elsif new.sent_at is null or btrim(new.sent_by_subject) = '' then
			raise exception 'sent portfolio transfer requires sealed manifest and sender provenance';
		end if;
		new.updated_at := statement_timestamp();
		return new;
	end if;

	if old.status = 'trimis' and new.status = 'receptionat' then
		if not is_bypass and (context_tenant <> old.destination_tenant_code or context_institution <> old.destination_institution_id) then
			raise exception 'only the destination institution can receive a portfolio transfer';
		end if;
		if row(new.transfer_type, new.source_institution, new.destination_institution, new.handover_on,
			new.handover_by, new.institution_id, new.notes, new.created_at,
			new.withdrawn_at, new.withdrawn_by_subject, new.withdrawal_reason,
			new.export_manifest_id, new.sent_at, new.sent_by_subject, new.closed_at, new.closed_by_subject)
			is distinct from row(old.transfer_type, old.source_institution, old.destination_institution, old.handover_on,
			old.handover_by, old.institution_id, old.notes, old.created_at,
			old.withdrawn_at, old.withdrawn_by_subject, old.withdrawal_reason,
			old.export_manifest_id, old.sent_at, old.sent_by_subject, old.closed_at, old.closed_by_subject) then
			raise exception 'sent portfolio transfer package is immutable';
		end if;
		if not is_bypass then
			new.received_at := statement_timestamp();
			new.received_on := current_date;
			new.received_by := actor_subject;
			new.received_by_subject := actor_subject;
		elsif new.received_at is null or new.received_on is null or btrim(new.received_by_subject) = '' then
			raise exception 'received portfolio transfer requires receiver provenance';
		end if;
		new.updated_at := statement_timestamp();
		return new;
	end if;

	if old.status = 'receptionat' and new.status = 'inchis' then
		if not is_bypass and (context_tenant <> old.source_tenant_code or context_institution <> old.source_institution_id) then
			raise exception 'only the source institution can close a received portfolio transfer';
		end if;
		if row(new.transfer_type, new.source_institution, new.destination_institution, new.handover_on,
			new.received_on, new.handover_by, new.received_by, new.institution_id, new.notes,
			new.created_at, new.withdrawn_at, new.withdrawn_by_subject, new.withdrawal_reason,
			new.export_manifest_id, new.sent_at, new.sent_by_subject, new.received_at, new.received_by_subject)
			is distinct from row(old.transfer_type, old.source_institution, old.destination_institution, old.handover_on,
			old.received_on, old.handover_by, old.received_by, old.institution_id, old.notes,
			old.created_at, old.withdrawn_at, old.withdrawn_by_subject, old.withdrawal_reason,
			old.export_manifest_id, old.sent_at, old.sent_by_subject, old.received_at, old.received_by_subject) then
			raise exception 'received portfolio transfer evidence is immutable';
		end if;
		if not is_bypass then
			new.closed_at := statement_timestamp();
			new.closed_by_subject := actor_subject;
		elsif new.closed_at is null or btrim(new.closed_by_subject) = '' then
			raise exception 'closed portfolio transfer requires closing provenance';
		end if;
		new.updated_at := statement_timestamp();
		return new;
	end if;

	raise exception 'invalid inter-tenant portfolio transfer transition % -> %', old.status, new.status;
end;
$$;

drop trigger if exists trg_education_intertenant_portfolio_transfer on education_portfolio_transfers;
create trigger trg_education_intertenant_portfolio_transfer
	before insert or update or delete on education_portfolio_transfers
	for each row execute function public.enforce_education_intertenant_portfolio_transfer();

alter table education_portfolio_transfers enable row level security;
alter table education_portfolio_transfers force row level security;
drop policy if exists tenant_isolation on education_portfolio_transfers;
drop policy if exists education_portfolio_transfer_participants on education_portfolio_transfers;
create policy education_portfolio_transfer_participants on education_portfolio_transfers
	using (
		public.can_bypass_tenant_rls()
		or (source_tenant_code = public.current_tenant_code() and source_institution_id = public.current_institution_id())
		or (destination_tenant_code = public.current_tenant_code() and destination_institution_id = public.current_institution_id() and status <> 'pregatit')
		or (routing_version = 1 and institution_id = public.current_institution_id())
	)
	with check (
		public.can_bypass_tenant_rls()
		or (source_tenant_code = public.current_tenant_code() and source_institution_id = public.current_institution_id())
		or (destination_tenant_code = public.current_tenant_code() and destination_institution_id = public.current_institution_id() and status in ('receptionat', 'inchis'))
		or (routing_version = 1 and institution_id = public.current_institution_id())
	);

insert into app_permissions (code, label) values
	('education.portfolios.transfer.receive', 'Receive inter-institution professional portfolio transfers')
on conflict (code) do update set label=excluded.label;

insert into app_position_permissions (position_code, permission_code) values
	('director', 'education.portfolios.transfer.receive'),
	('portfolio_custodian', 'education.portfolios.transfer.receive'),
	('secretariat', 'education.portfolios.transfer.receive'),
	('super_admin', 'education.portfolios.transfer.receive'),
	('e2e_canary', 'education.portfolios.transfer.receive')
on conflict do nothing;

insert into app_role_permissions (role_code, permission_code) values
	('super_admin', 'education.portfolios.transfer.receive'),
	('e2e_canary', 'education.portfolios.transfer.receive')
on conflict do nothing;
