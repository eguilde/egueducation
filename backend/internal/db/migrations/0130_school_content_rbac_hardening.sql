-- Tenant administration and pedagogical access are distinct trust domains.
-- Earlier bootstrap migrations made the tenant-scoped `admin` role a
-- convenience superset.  That is unsafe for professional portfolios: an
-- administrator who configures users/RBAC must not thereby read or alter
-- teachers' evidence.  Reconcile already-issued mappings without changing
-- the published migrations that created them.
delete from app_role_permissions role_permission
using app_permissions permission
where role_permission.role_code = 'admin'
	and role_permission.permission_code = permission.code
	and lower(permission.code) like 'education.%';

-- The director has explicit institutional review, verification, correction,
-- transfer and custody permissions.  The legacy broad write permission opens
-- generic evidence CRUD and would let a director edit a teacher's content,
-- so it must not remain on the director role.
delete from app_role_permissions
where role_code = 'director'
	and permission_code = 'education.portfolios.manage';

-- Replace the historical completion trigger.  `admin` still receives future
-- non-pedagogical tenant-administration capabilities, while the technical
-- super-admin and dedicated E2E role retain their established complete
-- catalog behaviour.  No education permission is granted implicitly to
-- either admin or director; an educational role requires an explicit,
-- reviewed role-permission mapping.
create or replace function public.apply_complete_tenant_role_mapping()
returns trigger
language plpgsql
as $$
begin
	insert into app_role_permissions (role_code, permission_code)
	select role.code, new.code
	from app_roles role
	where role.code in ('super_admin', 'e2e_canary')
		or (role.code = 'admin' and lower(new.code) not like 'education.%')
	on conflict do nothing;

	return new;
end;
$$;
