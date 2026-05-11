insert into app_permissions(code, label) values
	('admin.workflow_definitions.manage', 'Manage workflow definitions')
on conflict (code) do nothing;

insert into app_user_permissions(user_id, permission_code)
select id, permission_code
from app_users
cross join (values
	('admin.workflow_definitions.manage')
) as permissions(permission_code)
where sub = 'usr-001'
on conflict do nothing;
