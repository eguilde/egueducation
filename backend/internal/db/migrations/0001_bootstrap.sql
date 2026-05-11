create extension if not exists pgcrypto;

create table if not exists app_users (
	id uuid primary key default gen_random_uuid(),
	sub text not null unique,
	name text not null,
	email text not null unique,
	phone_number text not null default '',
	locale text not null default 'ro',
	created_at timestamptz not null default now(),
	updated_at timestamptz not null default now()
);

create table if not exists app_roles (
	code text primary key,
	label text not null
);

create table if not exists app_user_roles (
	user_id uuid not null references app_users(id) on delete cascade,
	role_code text not null references app_roles(code) on delete cascade,
	primary key (user_id, role_code)
);

create table if not exists app_permissions (
	code text primary key,
	label text not null
);

create table if not exists app_user_permissions (
	user_id uuid not null references app_users(id) on delete cascade,
	permission_code text not null references app_permissions(code) on delete cascade,
	primary key (user_id, permission_code)
);

create table if not exists app_modules (
	code text primary key,
	active boolean not null default true
);

create table if not exists app_user_modules (
	user_id uuid not null references app_users(id) on delete cascade,
	module_code text not null references app_modules(code) on delete cascade,
	primary key (user_id, module_code)
);

create table if not exists app_session_context (
	user_id uuid primary key references app_users(id) on delete cascade,
	institution_id text not null,
	institution_name text not null,
	auth_methods text[] not null default '{}',
	gdpr_capabilities text[] not null default '{}'
);

insert into app_roles(code, label) values
	('super_admin', 'Super Admin'),
	('workflow_admin', 'Workflow Admin'),
	('gdpr_officer', 'GDPR Officer')
on conflict (code) do nothing;

insert into app_permissions(code, label) values
	('dashboard.read', 'Read dashboard'),
	('registratura.read', 'Read registratura'),
	('workflow.read', 'Read workflow'),
	('earchiva.read', 'Read eArhiva'),
	('education.read', 'Read education'),
	('admin.read', 'Read admin'),
	('admin.users.read', 'Read admin users'),
	('admin.users.manage', 'Manage admin users'),
	('admin.memberships.read', 'Read memberships'),
	('admin.positions.read', 'Read positions'),
	('admin.permissions.read', 'Read permissions'),
	('admin.workflow_definitions.read', 'Read workflow definitions'),
	('gdpr.read', 'Read GDPR'),
	('gdpr.export', 'Export GDPR data')
on conflict (code) do nothing;

insert into app_modules(code, active) values
	('dashboard', true),
	('registratura', true),
	('workflow', true),
	('earchiva', true),
	('education', true),
	('admin', true),
	('gdpr', true)
on conflict (code) do nothing;

insert into app_users(sub, name, email, phone_number, locale)
values ('usr-001', 'Ana Ionescu', 'ana.ionescu@egueducation.ro', '+40740100101', 'ro')
on conflict (sub) do update
set name = excluded.name,
	email = excluded.email,
	phone_number = excluded.phone_number,
	locale = excluded.locale,
	updated_at = now();

insert into app_user_roles(user_id, role_code)
select id, role_code
from app_users
cross join (values ('super_admin'), ('workflow_admin'), ('gdpr_officer')) as roles(role_code)
where sub = 'usr-001'
on conflict do nothing;

insert into app_user_permissions(user_id, permission_code)
select id, permission_code
from app_users
cross join (values
	('dashboard.read'),
	('registratura.read'),
	('workflow.read'),
	('earchiva.read'),
	('education.read'),
	('admin.read'),
	('admin.users.read'),
	('admin.users.manage'),
	('admin.memberships.read'),
	('admin.positions.read'),
	('admin.permissions.read'),
	('admin.workflow_definitions.read'),
	('gdpr.read'),
	('gdpr.export')
) as permissions(permission_code)
where sub = 'usr-001'
on conflict do nothing;

insert into app_user_modules(user_id, module_code)
select id, module_code
from app_users
cross join (values
	('dashboard'),
	('registratura'),
	('workflow'),
	('earchiva'),
	('education'),
	('admin'),
	('gdpr')
) as modules(module_code)
where sub = 'usr-001'
on conflict do nothing;

insert into app_session_context(user_id, institution_id, institution_name, auth_methods, gdpr_capabilities)
select
	id,
	'inst-001',
	'Colegiul Național EguEducation',
	array['oidc_redirect', 'sms_otp', 'passkey', 'eudi_wallet'],
	array['retention_policies', 'subject_export', 'purpose_limited_access', 'publication_anonymization']
from app_users
where sub = 'usr-001'
on conflict (user_id) do update
set institution_id = excluded.institution_id,
	institution_name = excluded.institution_name,
	auth_methods = excluded.auth_methods,
	gdpr_capabilities = excluded.gdpr_capabilities;
