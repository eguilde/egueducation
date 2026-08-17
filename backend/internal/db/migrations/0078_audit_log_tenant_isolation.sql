-- Audit rows recorded before tenant isolation cannot be attributed safely.
-- Leave them unassigned: the tenant policies below deliberately never expose
-- null institution IDs to an application session.
alter table app_audit_log
	add column if not exists institution_id text null;

create index if not exists idx_app_audit_log_institution_created_at
	on app_audit_log (institution_id, created_at desc)
	where institution_id is not null;

alter table app_audit_log enable row level security;
alter table app_audit_log force row level security;

drop policy if exists tenant_isolation on app_audit_log;
drop policy if exists app_audit_log_tenant_read on app_audit_log;
drop policy if exists app_audit_log_tenant_append on app_audit_log;

-- Audit records are tenant-scoped and append-only.  In particular, no UPDATE
-- or DELETE policy is created, so ordinary application roles cannot alter or
-- remove recorded events.
create policy tenant_isolation on app_audit_log
	for select
	using (institution_id = public.current_institution_id());

create policy app_audit_log_tenant_append on app_audit_log
	for insert
	with check (institution_id = public.current_institution_id());
