create table if not exists oidc_production_e2e_challenges (
    jti text primary key,
    authn_session_id text not null unique,
    user_id uuid not null references app_users(id) on delete cascade,
    tenant_code text not null references app_tenants(code) on delete cascade,
    expires_at timestamptz not null,
    consumed_at timestamptz,
    created_at timestamptz not null default now()
);

create index if not exists idx_oidc_production_e2e_challenges_expiry
    on oidc_production_e2e_challenges (expires_at)
    where consumed_at is null;

insert into app_roles (code, label)
values ('e2e_canary', 'Production E2E Canary')
on conflict (code) do update set label=excluded.label;

insert into app_positions (code, name, scope_module, active, sort_order)
values ('e2e_canary', 'Production E2E Canary', 'platform', true, 999)
on conflict (code) do update
set name=excluded.name, scope_module=excluded.scope_module, active=true, sort_order=excluded.sort_order, updated_at=now();

delete from app_position_roles where position_code='e2e_canary';

insert into app_position_roles (position_code, role_code)
values ('e2e_canary', 'e2e_canary')
on conflict do nothing;

delete from app_role_permissions where role_code='e2e_canary';

insert into app_role_permissions (role_code, permission_code)
select 'e2e_canary', code
from app_permissions
where code in ('dashboard.read', 'admin.read')
on conflict do nothing;
