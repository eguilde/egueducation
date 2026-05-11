insert into app_permissions(code, label) values
	('admin.roles.read', 'Read administrative roles'),
	('admin.roles.manage', 'Manage administrative roles')
on conflict (code) do nothing;

insert into app_user_permissions (user_id, permission_code)
select ur.user_id, permissions.permission_code
from app_user_roles ur
cross join (values
	('admin.roles.read'),
	('admin.roles.manage')
) as permissions(permission_code)
where ur.role_code = 'super_admin'
on conflict do nothing;
