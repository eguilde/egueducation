do $$
begin
	alter table app_tenants
		alter column root_org_unit_code drop not null;
	exception
		when undefined_table then null;
end;
$$;

do $$
begin
	alter table app_org_units
		alter column tenant_code drop not null;
exception
	when undefined_table then null;
end;
$$;

do $$
begin
	alter table app_tenants disable row level security;
exception
	when undefined_table then null;
end;
$$;

do $$
begin
	alter table app_org_units disable row level security;
exception
	when undefined_table then null;
end;
$$;

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
	('tenant-egueducation', 'egueducation', 'inst-001', 'Colegiul Național EguEducation', 'EguEducation', null, true),
	('tenant-balotesti', 'scoalabalotesti', 'inst-balotesti', 'Școala Gimnazială nr. 1 Balotești', 'Balotești', null, true)
on conflict (code) do update
set subdomain = excluded.subdomain,
	institution_id = excluded.institution_id,
	display_name = excluded.display_name,
	short_name = excluded.short_name,
	active = excluded.active,
	updated_at = now();

insert into app_org_units (code, name, parent_code, active, sort_order, tenant_code)
values
	('unit-root', 'Colegiul Național EguEducation', null, true, 10, null),
	('unit-management', 'Conducere', 'unit-root', true, 20, null),
	('unit-secretariat', 'Secretariat', 'unit-root', true, 30, null),
	('unit-archive', 'Arhivă și registratură', 'unit-root', true, 40, null),
	('unit-hr', 'Resurse umane', 'unit-root', true, 50, null),
	('unit-primary', 'Învățământ primar', 'unit-root', true, 60, null),
	('unit-gymnasium', 'Gimnaziu', 'unit-root', true, 70, null),
	('unit-highschool', 'Liceu', 'unit-root', true, 80, null),
	('unit-balotesti-root', 'Școala Gimnazială nr. 1 Balotești', null, true, 15, null),
	('unit-balotesti-management', 'Conducere', 'unit-balotesti-root', true, 20, null),
	('unit-balotesti-secretariat', 'Secretariat', 'unit-balotesti-root', true, 30, null),
	('unit-balotesti-registratura', 'Registratură', 'unit-balotesti-root', true, 40, null),
	('unit-balotesti-archive', 'Arhivă', 'unit-balotesti-root', true, 50, null),
	('unit-balotesti-primary', 'Învățământ primar', 'unit-balotesti-root', true, 60, null),
	('unit-balotesti-gymnasium', 'Gimnaziu', 'unit-balotesti-root', true, 70, null)
on conflict (code) do update
set name = excluded.name,
	parent_code = excluded.parent_code,
	active = excluded.active,
	sort_order = excluded.sort_order,
	updated_at = now();

update app_org_units
set tenant_code = case
	when code like 'unit-balotesti-%' then 'tenant-balotesti'
	else 'tenant-egueducation'
end
where tenant_code is null;

update app_tenants
set root_org_unit_code = case code
	when 'tenant-balotesti' then 'unit-balotesti-root'
	else 'unit-root'
end,
	updated_at = now();

do $$
begin
	alter table app_tenants
		alter column root_org_unit_code set not null;
exception
	when undefined_table then null;
end;
$$;

do $$
begin
	alter table app_org_units
		alter column tenant_code set not null;
exception
	when undefined_table then null;
end;
$$;

do $$
begin
	alter table app_tenants disable row level security;
exception
	when undefined_table then null;
end;
$$;

do $$
begin
	alter table app_org_units disable row level security;
exception
	when undefined_table then null;
end;
$$;

do $$
begin
	alter table app_parties disable row level security;
exception
	when undefined_table then null;
end;
$$;

do $$
begin
	alter table app_memberships disable row level security;
exception
	when undefined_table then null;
end;
$$;

insert into app_parties (
	tenant_code,
	institution_id,
	code,
	party_type,
	display_name,
	short_name,
	first_name,
	last_name,
	legal_name,
	phone_number,
	email,
	country,
	notes,
	is_default_organization,
	active
)
values
	('tenant-egueducation', 'inst-001', 'organization', 'institution', 'Colegiul Național EguEducation', 'EguEducation', '', '', 'Colegiul Național EguEducation', '', '', 'RO', '', true, true),
	('tenant-balotesti', 'inst-balotesti', 'organization', 'institution', 'Școala Gimnazială nr. 1 Balotești', 'Balotești', '', '', 'Școala Gimnazială nr. 1 Balotești', '', '', 'RO', '', true, true),
	('tenant-egueducation', 'inst-001', 'thomas-galambos', 'physical', 'Thomas Galambos', 'Thomas', 'Thomas', 'Galambos', '', '+40771364169', 'thomas.galambos@eguilde.cloud', 'RO', '', false, true),
	('tenant-balotesti', 'inst-balotesti', 'diana-maria-ilhan', 'physical', 'DianaMaria Ilhan', 'Diana', 'DianaMaria', 'Ilhan', '', '+40735091230', 'dianailhan@eguilde.cloud', 'RO', '', false, true),
	('tenant-balotesti', 'inst-balotesti', 'primaria-balotesti', 'institution', 'Primăria Balotești', 'Primăria Balotești', '', '', 'Primăria Balotești', '', '', 'RO', '', false, true)
on conflict (tenant_code, code) do update
set institution_id = excluded.institution_id,
	party_type = excluded.party_type,
	display_name = excluded.display_name,
	short_name = excluded.short_name,
	first_name = excluded.first_name,
	last_name = excluded.last_name,
	legal_name = excluded.legal_name,
	phone_number = excluded.phone_number,
	email = excluded.email,
	country = excluded.country,
	notes = excluded.notes,
	is_default_organization = excluded.is_default_organization,
	active = excluded.active,
	updated_at = now();

insert into app_memberships (
	user_id,
	tenant_code,
	position_code,
	org_unit_code,
	organization_name,
	is_primary,
	active,
	start_date
)
select
	u.id,
	'tenant-egueducation',
	'super_admin',
	'unit-root',
	'Colegiul Național EguEducation',
	true,
	true,
	'2024-09-01'::date
from app_users u
where u.sub = 'thomasgalambos'
	and not exists (
		select 1
		from app_memberships m
		where m.user_id = u.id
			and m.tenant_code = 'tenant-egueducation'
			and m.position_code = 'super_admin'
			and m.org_unit_code = 'unit-root'
	);

insert into app_memberships (
	user_id,
	tenant_code,
	position_code,
	org_unit_code,
	organization_name,
	is_primary,
	active,
	start_date
)
select
	u.id,
	'tenant-balotesti',
	'director',
	'unit-balotesti-root',
	'Școala Gimnazială nr. 1 Balotești',
	true,
	true,
	'2024-09-01'::date
from app_users u
where u.sub = 'dianailhan'
	and not exists (
		select 1
		from app_memberships m
	where m.user_id = u.id
			and m.tenant_code = 'tenant-balotesti'
			and m.position_code = 'director'
			and m.org_unit_code = 'unit-balotesti-root'
	);

do $$
begin
	alter table app_tenants enable row level security;
	alter table app_tenants force row level security;
exception
	when undefined_table then null;
end;
$$;

do $$
begin
	alter table app_org_units enable row level security;
	alter table app_org_units force row level security;
exception
	when undefined_table then null;
end;
$$;

do $$
begin
	alter table app_parties enable row level security;
	alter table app_parties force row level security;
exception
	when undefined_table then null;
end;
$$;

do $$
begin
	alter table app_memberships enable row level security;
	alter table app_memberships force row level security;
exception
	when undefined_table then null;
end;
$$;
