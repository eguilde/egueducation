-- Active hostname directory used by all tenant routing decisions.
create table if not exists app_tenant_hostnames (
	id uuid primary key default gen_random_uuid(),
	tenant_code text not null references app_tenants(code) on delete cascade,
	hostname text not null,
	active boolean not null default true,
	created_at timestamptz not null default now(),
	updated_at timestamptz not null default now(),
	check (hostname = lower(trim(hostname))),
	unique (hostname)
);

create index if not exists idx_app_tenant_hostnames_active_lookup on app_tenant_hostnames (hostname) where active;

-- Provisioning APIs may accept a mixed-case host with an incidental trailing
-- dot. Store its canonical DNS form so unique routing is deterministic.
create or replace function public.normalize_tenant_hostname()
returns trigger language plpgsql as $$
begin
	new.hostname := lower(trim(trailing '.' from trim(new.hostname)));
	return new;
end;
$$;

drop trigger if exists trg_normalize_tenant_hostname on app_tenant_hostnames;
create trigger trg_normalize_tenant_hostname before insert or update of hostname
on app_tenant_hostnames for each row execute function public.normalize_tenant_hostname();

-- Preserve deployed routing while allowing custom domains to be provisioned
-- explicitly in this directory.
insert into app_tenant_hostnames (tenant_code, hostname, active)
select code, subdomain || '.eguilde.cloud', active
from app_tenants
where subdomain in ('egueducation', 'scoalabalotesti')
on conflict (hostname) do update
set tenant_code = excluded.tenant_code, active = excluded.active, updated_at = now();

create or replace function public.sync_tenant_hostname_activity()
returns trigger language plpgsql as $$
begin
	update app_tenant_hostnames set active = new.active, updated_at = now() where tenant_code = new.code;
	return new;
end;
$$;

drop trigger if exists trg_sync_tenant_hostname_activity on app_tenants;
create trigger trg_sync_tenant_hostname_activity after update of active on app_tenants
for each row execute function public.sync_tenant_hostname_activity();
