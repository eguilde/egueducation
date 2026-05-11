insert into app_permissions(code, label) values
	('admin.identity.read', 'Read OIDC identity administration'),
	('admin.identity.manage', 'Manage OIDC identity administration'),
	('admin.nomenclatures.read', 'Read application nomenclatures'),
	('admin.nomenclatures.manage', 'Manage application nomenclatures'),
	('registratura.manage', 'Manage registratura'),
	('education.mobility.manage', 'Manage education mobility'),
	('education.gradatii.manage', 'Manage education merit grants'),
	('gdpr.policies.manage', 'Manage GDPR retention policies'),
	('gdpr.requests.manage', 'Manage GDPR subject requests')
on conflict (code) do nothing;

insert into app_user_permissions (user_id, permission_code)
select ur.user_id, permissions.permission_code
from app_user_roles ur
cross join (values
	('admin.identity.read'),
	('admin.identity.manage'),
	('admin.nomenclatures.read'),
	('admin.nomenclatures.manage'),
	('registratura.manage'),
	('education.mobility.manage'),
	('education.gradatii.manage'),
	('gdpr.policies.manage'),
	('gdpr.requests.manage')
) as permissions(permission_code)
where ur.role_code = 'super_admin'
on conflict do nothing;
