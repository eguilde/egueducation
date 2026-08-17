-- The production functional-test identity is intentionally a complete tenant
-- operator. Its membership remains bound to exactly one tenant/institution;
-- this role does not grant the platform-superadmin RLS bypass.
insert into app_role_permissions (role_code, permission_code)
select 'e2e_canary', code
from app_permissions
on conflict do nothing;
