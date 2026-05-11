alter table app_memberships
	add column if not exists org_unit_code text null references app_org_units(code) on delete set null;

update app_memberships m
set org_unit_code = ou.code
from app_org_units ou
where m.org_unit_code is null
	and lower(m.organization_name) = lower(ou.name);

update app_memberships
set org_unit_code = 'unit-root'
where org_unit_code is null;

alter table app_memberships
	alter column org_unit_code set not null;
