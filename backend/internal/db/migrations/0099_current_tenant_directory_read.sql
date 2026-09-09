-- Request-scoped roles need the current tenant directory row for same-tenant
-- membership validation. This policy is read-only: tenant directory writes
-- remain restricted by the existing maintenance-only policy.
alter table app_tenants enable row level security;
alter table app_tenants force row level security;
drop policy if exists tenant_current_read on app_tenants;
create policy tenant_current_read on app_tenants
	for select
	to public
	using (code = public.current_tenant_code());
