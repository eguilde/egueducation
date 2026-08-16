-- Historical compatibility migration. It intentionally does not create an
-- operator, credential, cross-tenant membership, or privileged grant. Those
-- must be provisioned through the audited administration workflow after a
-- deployment. Existing production rows are left unchanged because this file is
-- already recorded in schema_migrations there.
select set_config('app.is_super_admin', 'true', true);

create table if not exists app_user_login_aliases (
    user_id uuid not null references app_users(id) on delete cascade,
    alias text not null check (length(trim(alias)) > 0),
    created_at timestamptz not null default now(),
    primary key (user_id, alias)
);

create unique index if not exists uq_app_user_login_aliases_normalized
    on app_user_login_aliases (lower(alias));

-- The legacy alias is safe only when an explicitly-created synthetic fixture
-- already exists. No row is inserted on a fresh production install.
insert into app_user_login_aliases (user_id, alias)
select id, 'platform-admin@example.test'
from app_users
where sub = 'platform-admin'
	and exists (
		select 1 from app_memberships m
		where m.user_id = app_users.id
			and m.tenant_code = 'tenant-balotesti'
			and m.active
	)
on conflict do nothing;
