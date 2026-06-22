create table if not exists app_role_permissions (
	role_code text not null references app_roles(code) on delete cascade,
	permission_code text not null references app_permissions(code) on delete cascade,
	primary key (role_code, permission_code)
);

create table if not exists app_position_roles (
	position_code text not null references app_positions(code) on delete cascade,
	role_code text not null references app_roles(code) on delete cascade,
	primary key (position_code, role_code)
);

insert into app_roles (code, label) values
	('admin', 'Administrator'),
	('director', 'Director'),
	('profesor', 'Profesor'),
	('secretar', 'Secretar'),
	('registrator', 'Registrator'),
	('inspector', 'Inspector'),
	('gdpr_officer', 'Responsabil GDPR'),
	('workflow_admin', 'Administrator workflow'),
	('super_admin', 'Super Admin')
on conflict (code) do update
set label = excluded.label;

insert into app_permissions (code, label) values
	('admin.roles.read', 'Read roles'),
	('admin.roles.manage', 'Manage roles')
on conflict (code) do nothing;

insert into app_role_permissions (role_code, permission_code)
select role_code, permission_code
from (
	values
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
		('super_admin', 'gdpr.requests.manage'),
		('admin', 'dashboard.read'),
		('admin', 'registratura.read'),
		('admin', 'workflow.read'),
		('admin', 'earchiva.read'),
		('admin', 'education.read'),
		('admin', 'admin.read'),
		('admin', 'admin.users.read'),
		('admin', 'admin.users.manage'),
		('admin', 'admin.memberships.read'),
		('admin', 'admin.positions.read'),
		('admin', 'admin.permissions.read'),
		('admin', 'admin.roles.read'),
		('admin', 'admin.roles.manage'),
		('admin', 'admin.workflow_definitions.read'),
		('admin', 'gdpr.read'),
		('admin', 'gdpr.export'),
		('admin', 'registratura.manage'),
		('admin', 'education.mobility.manage'),
		('admin', 'education.gradatii.manage'),
		('admin', 'gdpr.policies.manage'),
		('admin', 'gdpr.requests.manage'),
		('director', 'dashboard.read'),
		('director', 'education.read'),
		('director', 'workflow.read'),
		('director', 'registratura.read'),
		('director', 'admin.read'),
		('director', 'admin.users.read'),
		('director', 'admin.memberships.read'),
		('director', 'admin.positions.read'),
		('director', 'admin.permissions.read'),
		('director', 'admin.roles.read'),
		('director', 'gdpr.read'),
		('profesor', 'dashboard.read'),
		('profesor', 'education.read'),
		('profesor', 'workflow.read'),
		('secretar', 'dashboard.read'),
		('secretar', 'registratura.read'),
		('secretar', 'workflow.read'),
		('registrator', 'dashboard.read'),
		('registrator', 'registratura.read'),
		('registrator', 'workflow.read'),
		('inspector', 'dashboard.read'),
		('inspector', 'education.read'),
		('inspector', 'workflow.read'),
		('gdpr_officer', 'dashboard.read'),
		('gdpr_officer', 'gdpr.read'),
		('gdpr_officer', 'gdpr.export'),
		('workflow_admin', 'dashboard.read'),
		('workflow_admin', 'workflow.read'),
		('workflow_admin', 'admin.workflow_definitions.read')
) as mapping(role_code, permission_code)
on conflict do nothing;

insert into app_positions (code, name, scope_module, active, sort_order) values
	('profesor', 'Profesor', 'education', true, 95)
on conflict (code) do update
set name = excluded.name,
	scope_module = excluded.scope_module,
	active = excluded.active,
	sort_order = excluded.sort_order,
	updated_at = now();

insert into app_position_roles (position_code, role_code)
values
	('director', 'director'),
	('secretariat', 'secretar'),
	('registrator', 'registrator'),
	('inspector', 'inspector'),
	('profesor', 'profesor')
on conflict do nothing;

insert into app_user_roles (user_id, role_code)
select id, role_code
from app_users
cross join (values ('admin')) as roles(role_code)
where sub = 'thomas@eguilde.cloud'
on conflict do nothing;
