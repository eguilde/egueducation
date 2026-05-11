create table if not exists app_position_permissions (
	position_code text not null references app_positions(code) on delete cascade,
	permission_code text not null references app_permissions(code) on delete cascade,
	primary key (position_code, permission_code)
);

insert into app_position_permissions (position_code, permission_code)
select 'super_admin', code
from app_permissions
on conflict do nothing;

insert into app_position_permissions (position_code, permission_code)
values
	('director', 'dashboard.read'),
	('director', 'education.read'),
	('director', 'workflow.read'),
	('director', 'admin.read'),
	('director', 'admin.users.read'),
	('director', 'admin.memberships.read'),
	('director', 'admin.positions.read'),
	('director', 'admin.permissions.read'),
	('registrator', 'dashboard.read'),
	('registrator', 'registratura.read'),
	('registrator', 'workflow.read'),
	('secretariat', 'dashboard.read'),
	('secretariat', 'registratura.read'),
	('secretariat', 'workflow.read'),
	('gdpr_officer', 'dashboard.read'),
	('gdpr_officer', 'gdpr.read'),
	('gdpr_officer', 'gdpr.export'),
	('workflow_admin', 'dashboard.read'),
	('workflow_admin', 'workflow.read'),
	('workflow_admin', 'admin.workflow_definitions.read'),
	('arhivar', 'dashboard.read'),
	('arhivar', 'earchiva.read'),
	('inspector', 'dashboard.read'),
	('inspector', 'education.read'),
	('hr', 'dashboard.read'),
	('hr', 'education.read'),
	('hr', 'gdpr.read')
on conflict do nothing;
