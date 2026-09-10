-- Tenant administrators and the approved production test identity operate
-- entirely inside the tenant selected from the request host. Platform roles
-- remain a separate, explicit grant and are never implied by these mappings.
insert into app_role_permissions (role_code, permission_code)
select role.code, permission.code
from app_roles role
cross join app_permissions permission
where role.code in ('admin', 'super_admin', 'e2e_canary')
on conflict do nothing;

insert into app_role_permissions (role_code, permission_code)
select 'director', permission.code
from app_permissions permission
where permission.code like 'education.%'
on conflict do nothing;

-- Preserve these invariants when later migrations extend the tenant
-- permission catalog. This trigger grants no app_platform_roles entry and
-- therefore cannot create the cross-tenant RLS bypass.
create or replace function public.apply_complete_tenant_role_mapping()
returns trigger
language plpgsql
as $$
begin
	insert into app_role_permissions (role_code, permission_code)
	select role.code, new.code
	from app_roles role
	where role.code in ('admin', 'super_admin', 'e2e_canary')
	on conflict do nothing;

	if new.code like 'education.%' then
		insert into app_role_permissions (role_code, permission_code)
		values ('director', new.code)
		on conflict do nothing;
	end if;
	return new;
end;
$$;

drop trigger if exists trg_app_permissions_complete_tenant_roles on app_permissions;
create trigger trg_app_permissions_complete_tenant_roles
	after insert on app_permissions
	for each row execute function public.apply_complete_tenant_role_mapping();

-- This UUID is permanently reserved by the application for the approved
-- production fixed-OTP test identity. Provisioning also enforces this cleanup on
-- every startup, but the migration removes any historical platform grant
-- before the fixed-OTP feature is enabled.
delete from app_user_platform_roles
where user_id = '20c36b31-d7e9-4a4b-b6df-42adc5b2913d'::uuid;

-- The rejected activation-capability mechanism is no longer part of the
-- login contract. Its rows were short-lived and carry no business records.
drop table if exists oidc_production_e2e_challenges;
