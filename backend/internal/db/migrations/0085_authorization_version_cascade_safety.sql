-- Authorization-version writes are tenant-bound and FORCE RLS remains the
-- authority. A membership delete caused by cascading app_users deletion has
-- no surviving authorization subject to invalidate, however;
-- do not recreate an authorization row that would violate the parent FK.
create or replace function public.bump_tenant_authorization_version(
	p_tenant_code text,
	p_user_id uuid
)
returns void
language plpgsql
as $$
begin
	if p_tenant_code is null or p_user_id is null then
		return;
	end if;

	-- Preserve invalidation for ordinary membership/role/module deletes. Only
	-- cascading deletion of a parent tenant or profile is a no-op because the
	-- corresponding authorization state is being removed in the same command.
	if not exists (select 1 from app_users where id = p_user_id) then
		return;
	end if;

	begin
		insert into app_tenant_authorization_versions (tenant_code, user_id, version)
		values (p_tenant_code, p_user_id, 1)
		on conflict (tenant_code, user_id) do update
		set version = app_tenant_authorization_versions.version + 1,
			updated_at = now();
	exception when foreign_key_violation then
		-- A concurrently cascading parent delete can occur after the checks
		-- above. Swallow only that terminal case; all other FK failures remain
		-- visible and fail closed.
		if not exists (select 1 from app_users where id = p_user_id) then
			return;
		end if;
		raise;
	end;
end;
$$;
