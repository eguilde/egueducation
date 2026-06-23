create table if not exists app_tenants (
	code text primary key,
	subdomain text not null unique,
	institution_id text not null unique,
	display_name text not null,
	short_name text not null,
	root_org_unit_code text not null references app_org_units(code) on delete restrict,
	active boolean not null default true,
	created_at timestamptz not null default now(),
	updated_at timestamptz not null default now()
);

insert into app_tenants (
	code,
	subdomain,
	institution_id,
	display_name,
	short_name,
	root_org_unit_code,
	active
)
values
	('tenant-egueducation', 'egueducation', 'inst-001', 'EguEducation', 'EguEducation', 'unit-root', true),
	('tenant-balotesti', 'scoalabalotesti', 'inst-balotesti', 'Școala Gimnazială nr. 1 Balotești', 'Balotești', 'unit-balotesti-root', true)
on conflict (code) do update
set subdomain = excluded.subdomain,
	institution_id = excluded.institution_id,
	display_name = excluded.display_name,
	short_name = excluded.short_name,
	root_org_unit_code = excluded.root_org_unit_code,
	active = excluded.active,
	updated_at = now();

insert into app_org_units (code, name, parent_code, active, sort_order) values
	('unit-balotesti-root', 'Școala Gimnazială nr. 1 Balotești', null, true, 15),
	('unit-balotesti-management', 'Conducere', 'unit-balotesti-root', true, 20),
	('unit-balotesti-secretariat', 'Secretariat', 'unit-balotesti-root', true, 30),
	('unit-balotesti-registratura', 'Registratură', 'unit-balotesti-root', true, 40),
	('unit-balotesti-archive', 'Arhivă', 'unit-balotesti-root', true, 50),
	('unit-balotesti-primary', 'Învățământ primar', 'unit-balotesti-root', true, 60),
	('unit-balotesti-gymnasium', 'Gimnaziu', 'unit-balotesti-root', true, 70)
on conflict (code) do update
set name = excluded.name,
	parent_code = excluded.parent_code,
	active = excluded.active,
	sort_order = excluded.sort_order,
	updated_at = now();

insert into app_position_roles (position_code, role_code)
values
	('director', 'admin'),
	('secretariat', 'secretar'),
	('registrator', 'registrator'),
	('super_admin', 'super_admin')
on conflict do nothing;
