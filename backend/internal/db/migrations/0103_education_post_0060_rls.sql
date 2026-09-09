-- Education tables introduced after the 0060 tenant-isolation baseline must
-- never rely on handler predicates alone.  Each tenant has a unique
-- institution_id (app_tenants.institution_id is UNIQUE), so institution scope
-- is the canonical database discriminator for these legacy-shaped tables.
do $$
declare
	tables text[] := array[
		'education_portfolio_valorifications',
		'education_committees',
		'education_committee_members'
	];
	tbl text;
	has_institution_id boolean;
begin
	foreach tbl in array tables loop
		select exists (
			select 1 from information_schema.columns
			where table_schema = 'public'
				and table_name = tbl
				and column_name = 'institution_id'
		) into has_institution_id;
		if not has_institution_id then
			raise exception 'post-0060 education table % is missing institution_id', tbl;
		end if;

		execute format('alter table public.%I enable row level security', tbl);
		execute format('alter table public.%I force row level security', tbl);
		execute format('drop policy if exists tenant_isolation on public.%I', tbl);
		execute format(
			'create policy tenant_isolation on public.%I
			 using (public.can_bypass_tenant_rls() or institution_id = public.current_institution_id())
			 with check (public.can_bypass_tenant_rls() or institution_id = public.current_institution_id())',
			tbl
		);
		execute format('drop trigger if exists trg_%s_entity_version on public.%I', tbl, tbl);
		execute format(
			'create trigger trg_%s_entity_version
			 after insert or update or delete on public.%I
			 for each row execute function public.record_entity_version()',
			tbl, tbl
		);
	end loop;
end;
$$;
