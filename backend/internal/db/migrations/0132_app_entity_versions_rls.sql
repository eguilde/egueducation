-- Entity snapshots are shared infrastructure, but never shared tenant data.
-- Rows created before a complete request context may have blank scope fields;
-- they intentionally remain inaccessible to ordinary application sessions
-- rather than being guessed into an institution during this security repair.
alter table public.app_entity_versions enable row level security;
alter table public.app_entity_versions force row level security;

drop policy if exists app_entity_versions_tenant_read on public.app_entity_versions;
drop policy if exists app_entity_versions_tenant_append on public.app_entity_versions;

create policy app_entity_versions_tenant_read on public.app_entity_versions
	for select
	using (
		public.can_bypass_tenant_rls()
		or (
			tenant_code = public.current_tenant_code()
			and institution_id = public.current_institution_id()
		)
	);

create policy app_entity_versions_tenant_append on public.app_entity_versions
	for insert
	with check (
		public.can_bypass_tenant_rls()
		or (
			tenant_code = public.current_tenant_code()
			and institution_id = public.current_institution_id()
		)
	);

-- History is append-only for ordinary application roles. There intentionally
-- is no UPDATE or DELETE policy; corrections create a compensating version.
