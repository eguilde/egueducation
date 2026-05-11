create table if not exists app_auth_methods (
	code text primary key,
	enabled boolean not null default true,
	primary_method boolean not null default false,
	sort_order integer not null default 0,
	updated_at timestamptz not null default now()
);

insert into app_permissions(code, label) values
	('admin.auth_methods.read', 'Read auth methods'),
	('admin.auth_methods.manage', 'Manage auth methods'),
	('admin.modules.read', 'Read modules'),
	('admin.modules.manage', 'Manage modules')
on conflict (code) do nothing;

insert into app_user_permissions(user_id, permission_code)
select id, permission_code
from app_users
cross join (values
	('admin.auth_methods.read'),
	('admin.auth_methods.manage'),
	('admin.modules.read'),
	('admin.modules.manage')
) as permissions(permission_code)
where sub = 'usr-001'
on conflict do nothing;

insert into app_auth_methods (code, enabled, primary_method, sort_order) values
	('oidc_redirect', true, true, 10),
	('sms_otp', true, false, 20),
	('passkey', true, false, 30),
	('eudi_wallet', true, false, 40)
on conflict (code) do update
set enabled = excluded.enabled,
	primary_method = excluded.primary_method,
	sort_order = excluded.sort_order,
	updated_at = now();
