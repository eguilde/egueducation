create table if not exists app_org_units (
	code text primary key,
	name text not null,
	parent_code text null references app_org_units(code) on delete set null,
	active boolean not null default true,
	sort_order integer not null default 100,
	created_at timestamptz not null default now(),
	updated_at timestamptz not null default now()
);

insert into app_permissions(code, label) values
	('admin.org_units.read', 'Read org units'),
	('admin.org_units.manage', 'Manage org units')
on conflict (code) do nothing;

insert into app_org_units (code, name, parent_code, active, sort_order) values
	('unit-root', 'Colegiul Național EguEducation', null, true, 10),
	('unit-management', 'Conducere', 'unit-root', true, 20),
	('unit-secretariat', 'Secretariat', 'unit-root', true, 30),
	('unit-archive', 'Arhivă și registratură', 'unit-root', true, 40),
	('unit-hr', 'Resurse umane', 'unit-root', true, 50),
	('unit-primary', 'Învățământ primar', 'unit-root', true, 60),
	('unit-gymnasium', 'Gimnaziu', 'unit-root', true, 70),
	('unit-highschool', 'Liceu', 'unit-root', true, 80)
on conflict (code) do update
set name = excluded.name,
	parent_code = excluded.parent_code,
	active = excluded.active,
	sort_order = excluded.sort_order,
	updated_at = now();

insert into app_position_permissions(position_code, permission_code)
select 'super_admin', permission_code
from (values ('admin.org_units.read'), ('admin.org_units.manage')) as permissions(permission_code)
on conflict do nothing;
