insert into app_org_units (code, name, parent_code, active, sort_order)
values ('unit-balotesti-root', 'Școala Gimnazială nr. 1 Balotești', null, true, 15)
on conflict (code) do update
set name = excluded.name,
	parent_code = excluded.parent_code,
	active = excluded.active,
	sort_order = excluded.sort_order,
	updated_at = now();

insert into app_users (
	sub,
	name,
	email,
	phone_number,
	locale,
	status,
	email_verified,
	phone_number_verified,
	preferred_otp_channel,
	last_login_at
)
values (
	'dianailhan',
	'DianaMaria Ilhan',
	'dianailhan@eguilde.cloud',
	'+40735091230',
	'ro',
	'active',
	true,
	true,
	'sms',
	'2026-05-11T10:15:00Z'
)
on conflict (sub) do update
set name = excluded.name,
	email = excluded.email,
	phone_number = excluded.phone_number,
	locale = excluded.locale,
	status = excluded.status,
	email_verified = excluded.email_verified,
	phone_number_verified = excluded.phone_number_verified,
	preferred_otp_channel = excluded.preferred_otp_channel,
	last_login_at = excluded.last_login_at,
	updated_at = now();

insert into app_user_roles (user_id, role_code)
select id, role_code
from app_users
cross join (values ('admin')) as roles(role_code)
where sub = 'dianailhan'
on conflict do nothing;

insert into app_user_permissions (user_id, permission_code)
select app_users.id, app_permissions.code
from app_users
cross join app_permissions
where sub = 'dianailhan'
on conflict do nothing;

insert into app_user_modules (user_id, module_code)
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
where sub = 'dianailhan'
on conflict do nothing;

insert into app_session_context (user_id, institution_id, institution_name, auth_methods, gdpr_capabilities)
select
	id,
	'inst-balotesti',
	'Școala Gimnazială nr. 1 Balotești',
	array['oidc_redirect', 'sms_otp', 'passkey', 'eudi_wallet'],
	array['retention_policies', 'subject_export', 'purpose_limited_access', 'publication_anonymization']
from app_users
where sub = 'dianailhan'
on conflict (user_id) do update
set institution_id = excluded.institution_id,
	institution_name = excluded.institution_name,
	auth_methods = excluded.auth_methods,
	gdpr_capabilities = excluded.gdpr_capabilities;

insert into app_memberships (user_id, position_code, org_unit_code, organization_name, is_primary, active, start_date)
select
	u.id,
	'director',
	'unit-balotesti-root',
	'Școala Gimnazială nr. 1 Balotești',
	true,
	true,
	'2024-09-01'::date
from app_users u
where u.sub = 'dianailhan'
on conflict do nothing;
