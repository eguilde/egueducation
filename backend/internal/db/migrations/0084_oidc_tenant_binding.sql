-- Bind transient OIDC protocol state to the host-resolved tenant. The legacy
-- tenant_id UUID is retained only as a compatibility attribute; tenant_code
-- is the actual primary/conflict boundary. Unbound legacy state is quarantined
-- rather than guessed into a tenant, so in-flight pre-migration logins fail
-- closed instead of being replayable from another host.
alter table oidc_authn_sessions
    add column if not exists tenant_code text references app_tenants(code) on delete cascade;

alter table oidc_grant_sessions
    add column if not exists tenant_code text references app_tenants(code) on delete cascade;

create table if not exists oidc_legacy_unbound_authn_sessions (
    tenant_id uuid not null,
    id text not null,
    data jsonb not null,
    expires_at timestamptz not null,
    quarantined_at timestamptz not null default now(),
    quarantine_reason text not null,
    primary key (tenant_id, id)
);

create table if not exists oidc_legacy_unbound_grant_sessions (
    tenant_id uuid not null,
    id text not null,
    data jsonb not null,
    expires_at timestamptz not null,
    quarantined_at timestamptz not null default now(),
    quarantine_reason text not null,
    primary key (tenant_id, id)
);

insert into oidc_legacy_unbound_authn_sessions (tenant_id, id, data, expires_at, quarantine_reason)
select tenant_id, id, data, expires_at, 'missing host-resolved tenant_code during 0084 migration'
from oidc_authn_sessions
where tenant_code is null
on conflict (tenant_id, id) do nothing;

insert into oidc_legacy_unbound_grant_sessions (tenant_id, id, data, expires_at, quarantine_reason)
select tenant_id, id, data, expires_at, 'missing host-resolved tenant_code during 0084 migration'
from oidc_grant_sessions
where tenant_code is null
on conflict (tenant_id, id) do nothing;

delete from oidc_authn_sessions where tenant_code is null;
delete from oidc_grant_sessions where tenant_code is null;

-- A manually pre-populated tenant_code column may still contain duplicate
-- tenant/id pairs under the old (tenant_id,id) key. Keep the newest expiry as
-- the only usable record and quarantine every ambiguous predecessor.
with duplicates as (
    select ctid, tenant_id, id, data, expires_at,
           row_number() over (partition by tenant_code, id order by expires_at desc, ctid desc) as row_number
    from oidc_authn_sessions
)
insert into oidc_legacy_unbound_authn_sessions (tenant_id, id, data, expires_at, quarantine_reason)
select tenant_id, id, data, expires_at, 'duplicate tenant_code/id under legacy storage key during 0084 migration'
from duplicates
where row_number > 1
on conflict (tenant_id, id) do nothing;

with duplicates as (
    select ctid,
           row_number() over (partition by tenant_code, id order by expires_at desc, ctid desc) as row_number
    from oidc_authn_sessions
)
delete from oidc_authn_sessions session
using duplicates
where session.ctid = duplicates.ctid and duplicates.row_number > 1;

with duplicates as (
    select ctid, tenant_id, id, data, expires_at,
           row_number() over (partition by tenant_code, id order by expires_at desc, ctid desc) as row_number
    from oidc_grant_sessions
)
insert into oidc_legacy_unbound_grant_sessions (tenant_id, id, data, expires_at, quarantine_reason)
select tenant_id, id, data, expires_at, 'duplicate tenant_code/id under legacy storage key during 0084 migration'
from duplicates
where row_number > 1
on conflict (tenant_id, id) do nothing;

with duplicates as (
    select ctid,
           row_number() over (partition by tenant_code, id order by expires_at desc, ctid desc) as row_number
    from oidc_grant_sessions
)
delete from oidc_grant_sessions session
using duplicates
where session.ctid = duplicates.ctid and duplicates.row_number > 1;

alter table oidc_authn_sessions
    alter column tenant_code set not null;

alter table oidc_grant_sessions
    alter column tenant_code set not null;

alter table oidc_authn_sessions
    drop constraint if exists oidc_authn_sessions_pkey;

alter table oidc_grant_sessions
    drop constraint if exists oidc_grant_sessions_pkey;

alter table oidc_authn_sessions
    add primary key (tenant_code, id);

alter table oidc_grant_sessions
    add primary key (tenant_code, id);

create index if not exists idx_oidc_authn_sessions_tenant_code_expires
    on oidc_authn_sessions (tenant_code, expires_at desc);

create index if not exists idx_oidc_grant_sessions_tenant_code_expires
    on oidc_grant_sessions (tenant_code, expires_at desc);

comment on column oidc_authn_sessions.tenant_code is
    'Host-resolved tenant that owns this OIDC authorization transaction.';

comment on column oidc_grant_sessions.tenant_code is
    'Host-resolved tenant that owns this authorization code or refresh grant.';
