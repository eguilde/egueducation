-- Preserve historic login identifiers without weakening tenant membership checks.
-- Platform operators still need an explicit membership and role in every tenant
-- they administer; the alias only resolves the global identity.
select set_config('app.is_super_admin', 'true', true);

create table if not exists app_user_login_aliases (
    user_id uuid not null references app_users(id) on delete cascade,
    alias text not null check (length(trim(alias)) > 0),
    created_at timestamptz not null default now(),
    primary key (user_id, alias)
);

create unique index if not exists uq_app_user_login_aliases_normalized
    on app_user_login_aliases (lower(alias));

insert into app_user_login_aliases (user_id, alias)
select id, 'thomas@eguilde.cloud'
from app_users
where sub = 'thomasgalambos'
on conflict do nothing;

insert into app_memberships (
    user_id,
    tenant_code,
    position_code,
    org_unit_code,
    organization_name,
    is_primary,
    active,
    start_date
)
select
    u.id,
    'tenant-balotesti',
    'super_admin',
    'unit-balotesti-root',
    t.display_name,
    true,
    true,
    current_date
from app_users u
join app_tenants t on t.code = 'tenant-balotesti' and t.active
where u.sub = 'thomasgalambos'
  and not exists (
      select 1
      from app_memberships m
      where m.user_id = u.id
        and m.tenant_code = 'tenant-balotesti'
        and m.active
  );

insert into app_user_roles (tenant_code, user_id, role_code)
select 'tenant-balotesti', id, 'super_admin'
from app_users
where sub = 'thomasgalambos'
on conflict do nothing;

do $$
begin
    if not exists (
        select 1
        from app_users u
        join app_user_login_aliases a on a.user_id = u.id
        join app_memberships m on m.user_id = u.id
        join app_user_roles r on r.user_id = u.id and r.tenant_code = m.tenant_code
        where u.sub = 'thomasgalambos'
          and lower(a.alias) = 'thomas@eguilde.cloud'
          and m.tenant_code = 'tenant-balotesti'
          and m.active
          and r.role_code = 'super_admin'
    ) then
        raise exception '0073 failed to provision the Balotesti platform superadmin login';
    end if;
end $$;
