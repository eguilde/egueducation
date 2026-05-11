create table if not exists app_audit_log (
	id uuid primary key default gen_random_uuid(),
	actor_subject text not null,
	action text not null,
	target_type text not null,
	target_id text not null default '',
	status text not null default 'success',
	summary text not null default '',
	details jsonb not null default '{}'::jsonb,
	created_at timestamptz not null default now()
);

create index if not exists idx_app_audit_log_created_at
	on app_audit_log (created_at desc);

create index if not exists idx_app_audit_log_actor_subject
	on app_audit_log (actor_subject);

create index if not exists idx_app_audit_log_action
	on app_audit_log (action);

create index if not exists idx_app_audit_log_target_type
	on app_audit_log (target_type);

insert into app_permissions(code, label) values
	('admin.audit.read', 'Read administrative audit log')
on conflict (code) do nothing;

insert into app_user_permissions (user_id, permission_code)
select ur.user_id, 'admin.audit.read'
from app_user_roles ur
where ur.role_code = 'super_admin'
on conflict do nothing;
