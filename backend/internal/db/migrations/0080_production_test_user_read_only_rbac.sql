-- The production browser-test identity is a normal tenant member with one
-- explicit read-only role. It may inspect every enabled module in its tenant,
-- but it receives no manage, transition, approval, upload or delete grant.
delete from app_role_permissions
where role_code = 'e2e_canary';

insert into app_role_permissions (role_code, permission_code)
select 'e2e_canary', code
from app_permissions
where code = 'dashboard.read'
   or code = 'admin.read'
   or code like '%.read'
on conflict do nothing;
