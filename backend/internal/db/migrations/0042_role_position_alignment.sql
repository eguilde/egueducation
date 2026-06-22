insert into app_roles (code, label) values
	('arhivar', 'Arhivar'),
	('hr', 'Resurse umane')
on conflict (code) do update
set label = excluded.label;

insert into app_role_permissions (role_code, permission_code)
select role_code, permission_code
from (
	values
		('arhivar', 'dashboard.read'),
		('arhivar', 'earchiva.read'),
		('hr', 'dashboard.read'),
		('hr', 'education.read'),
		('hr', 'gdpr.read'),
		('gdpr_officer', 'dashboard.read'),
		('gdpr_officer', 'gdpr.read'),
		('gdpr_officer', 'gdpr.export'),
		('workflow_admin', 'dashboard.read'),
		('workflow_admin', 'workflow.read'),
		('workflow_admin', 'admin.workflow_definitions.read'),
		('super_admin', 'dashboard.read'),
		('super_admin', 'registratura.read'),
		('super_admin', 'workflow.read'),
		('super_admin', 'earchiva.read'),
		('super_admin', 'education.read'),
		('super_admin', 'admin.read'),
		('super_admin', 'admin.users.read'),
		('super_admin', 'admin.users.manage'),
		('super_admin', 'admin.memberships.read'),
		('super_admin', 'admin.positions.read'),
		('super_admin', 'admin.permissions.read'),
		('super_admin', 'admin.roles.read'),
		('super_admin', 'admin.roles.manage'),
		('super_admin', 'admin.workflow_definitions.read'),
		('super_admin', 'gdpr.read'),
		('super_admin', 'gdpr.export'),
		('super_admin', 'registratura.manage'),
		('super_admin', 'education.mobility.manage'),
		('super_admin', 'education.gradatii.manage'),
		('super_admin', 'gdpr.policies.manage'),
		('super_admin', 'gdpr.requests.manage')
) as mapping(role_code, permission_code)
on conflict do nothing;

insert into app_position_roles (position_code, role_code)
values
	('gdpr_officer', 'gdpr_officer'),
	('workflow_admin', 'workflow_admin'),
	('super_admin', 'super_admin'),
	('arhivar', 'arhivar'),
	('hr', 'hr')
on conflict do nothing;
