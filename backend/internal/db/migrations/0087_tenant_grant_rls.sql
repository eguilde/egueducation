-- Tenant grants are authorization data, not platform-global configuration.
-- Every read and mutation must be bound to the tenant selected by the request
-- transaction. In particular, app.is_super_admin is deliberately absent: a
-- platform role may authorize a backend action, but it must not expose grants
-- for another tenant through an interactive database session.
--
-- Legacy grants with a NULL tenant_code are fail-closed by this policy. The
-- earlier tenant grant migration preserves their historical payload in
-- app_unresolved_legacy_grants; this migration neither assigns nor destroys
-- quarantined data.
do $$
declare
	grant_table text;
	existing_policy record;
begin
	foreach grant_table in array array[
		'app_user_roles',
		'app_user_permissions',
		'app_user_modules'
	]
	loop
		if to_regclass('public.' || grant_table) is null then
			raise exception '0087 preflight failed: required grant table % is missing', grant_table;
		end if;
		if not exists (
			select 1
			from information_schema.columns
			where table_schema = 'public'
				and table_name = grant_table
				and column_name = 'tenant_code'
		) then
			raise exception '0087 preflight failed: grant table % is missing tenant_code', grant_table;
		end if;

		execute format('alter table public.%I enable row level security', grant_table);
		execute format('alter table public.%I force row level security', grant_table);

		-- A permissive legacy policy would OR with a new policy and silently
		-- defeat tenant isolation. Replace every policy on these tables with the
		-- single contract below.
		for existing_policy in
			select policyname
			from pg_policies
			where schemaname = 'public'
				and tablename = grant_table
		loop
			execute format('drop policy if exists %I on public.%I', existing_policy.policyname, grant_table);
		end loop;

		execute format(
			'create policy tenant_grant_isolation on public.%I
				for all
				to public
				using (tenant_code is not null and tenant_code = public.current_tenant_code())
				with check (tenant_code is not null and tenant_code = public.current_tenant_code())',
			grant_table
		);
	end loop;
end;
$$;
