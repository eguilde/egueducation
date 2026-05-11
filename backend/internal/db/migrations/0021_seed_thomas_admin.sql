update app_users
set
	sub = 'thomas@eguilde.cloud',
	name = 'Thomas Galambos',
	email = 'thomas@eguilde.cloud',
	phone_number = '+40740000000',
	locale = 'ro',
	status = 'active',
	last_login_at = '2026-05-11T10:15:00Z',
	updated_at = now()
where sub = 'usr-001';

insert into app_users (sub, name, email, phone_number, locale, status, last_login_at)
values ('thomas@eguilde.cloud', 'Thomas Galambos', 'thomas@eguilde.cloud', '+40740000000', 'ro', 'active', '2026-05-11T10:15:00Z')
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
where sub = 'thomas@eguilde.cloud'
on conflict do nothing;

insert into app_user_permissions (user_id, permission_code)
select id, permission_code
from app_users
cross join app_permissions
where sub = 'thomas@eguilde.cloud'
on conflict do nothing;

insert into app_user_modules (user_id, module_code)
select id, code
from app_users
cross join app_modules
where sub = 'thomas@eguilde.cloud'
on conflict do nothing;

insert into app_session_context (user_id, institution_id, institution_name, auth_methods, gdpr_capabilities)
select
	id,
	'inst-001',
	'Colegiul Național EguEducation',
	array['oidc_redirect', 'sms_otp', 'passkey', 'eudi_wallet'],
	array['retention_policies', 'subject_export', 'purpose_limited_access', 'publication_anonymization']
from app_users
where sub = 'thomas@eguilde.cloud'
on conflict (user_id) do update
set institution_id = excluded.institution_id,
	institution_name = excluded.institution_name,
	auth_methods = excluded.auth_methods,
	gdpr_capabilities = excluded.gdpr_capabilities;

insert into app_memberships (user_id, position_code, organization_name, is_primary, active, start_date)
select id, 'super_admin', 'Colegiul Național EguEducation', true, true, '2024-09-01'::date
from app_users
where sub = 'thomas@eguilde.cloud'
	and not exists (
		select 1
		from app_memberships am
		where am.user_id = app_users.id
			and am.position_code = 'super_admin'
			and am.organization_name = 'Colegiul Național EguEducation'
	);
