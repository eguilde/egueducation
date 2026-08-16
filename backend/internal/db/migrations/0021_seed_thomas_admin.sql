update app_users
set
	sub = 'platform-admin@example.test',
	name = 'Platform Administrator',
	email = 'platform-admin@example.test',
	phone_number = '+40100000000',
	locale = 'ro',
	status = 'active',
	last_login_at = '2026-05-11T10:15:00Z',
	updated_at = now()
where sub = 'usr-001';

insert into app_users (sub, name, email, phone_number, locale, status, last_login_at)
values ('platform-admin@example.test', 'Platform Administrator', 'platform-admin@example.test', '+40100000000', 'ro', 'active', '2026-05-11T10:15:00Z')
on conflict (sub) do update
set name = excluded.name,
	email = excluded.email,
	phone_number = excluded.phone_number,
	locale = excluded.locale,
	status = excluded.status,
	last_login_at = excluded.last_login_at,
	updated_at = now();

insert into app_user_roles (user_id, role_code)
select id, role_code
from app_users
cross join (values ('super_admin'), ('workflow_admin'), ('gdpr_officer')) as roles(role_code)
where sub = 'platform-admin@example.test'
on conflict do nothing;

insert into app_user_permissions (user_id, permission_code)
select app_users.id, app_permissions.code
from app_users
cross join app_permissions
where sub = 'platform-admin@example.test'
on conflict do nothing;

insert into app_user_modules (user_id, module_code)
select id, code
from app_users
cross join app_modules
where sub = 'platform-admin@example.test'
on conflict do nothing;

insert into app_session_context (user_id, institution_id, institution_name, auth_methods, gdpr_capabilities)
select
	id,
	'inst-001',
	'Colegiul Național EguEducation',
	array['oidc_redirect', 'sms_otp', 'passkey', 'eudi_wallet'],
	array['retention_policies', 'subject_export', 'purpose_limited_access', 'publication_anonymization']
from app_users
where sub = 'platform-admin@example.test'
on conflict (user_id) do update
set institution_id = excluded.institution_id,
	institution_name = excluded.institution_name,
	auth_methods = excluded.auth_methods,
	gdpr_capabilities = excluded.gdpr_capabilities;

insert into app_memberships (user_id, position_code, organization_name, is_primary, active, start_date)
select id, 'super_admin', 'Colegiul Național EguEducation', true, true, '2024-09-01'::date
from app_users
where sub = 'platform-admin@example.test'
	and not exists (
		select 1
		from app_memberships am
		where am.user_id = app_users.id
			and am.position_code = 'super_admin'
			and am.organization_name = 'Colegiul Național EguEducation'
	);
