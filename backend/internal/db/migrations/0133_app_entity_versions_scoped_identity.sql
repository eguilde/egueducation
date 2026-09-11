-- Make version identity match the tenant/institution RLS boundary introduced
-- by 0132. This is deliberately a forward migration because 0132 may already
-- be recorded in deployed schema_migrations ledgers.

alter table public.app_entity_versions
	drop constraint if exists app_entity_versions_entity_table_entity_id_version_no_key;
alter table public.app_entity_versions
	drop constraint if exists app_entity_versions_scope_entity_version_key;
alter table public.app_entity_versions
	add constraint app_entity_versions_scope_entity_version_key
	unique (tenant_code, institution_id, entity_table, entity_id, version_no);

-- SECURITY INVOKER is intentional: history writes remain subject to the same
-- RLS policies as the caller. The transaction advisory lock prevents two
-- writers from allocating the same version number for one scoped entity.
create or replace function public.record_entity_version()
returns trigger
language plpgsql
security invoker
set search_path = pg_catalog, public
as $$
declare
	snapshot jsonb;
	entity_id_value uuid;
	next_version integer;
	tenant_value text;
	institution_value text;
	changed_by_value text;
begin
	if tg_op = 'DELETE' then
		snapshot := to_jsonb(old);
	elsif tg_op = 'UPDATE' then
		snapshot := to_jsonb(new);
	else
		snapshot := to_jsonb(new);
	end if;

	if coalesce(snapshot ->> 'id', '') = '' then
		return coalesce(new, old);
	end if;

	entity_id_value := (snapshot ->> 'id')::uuid;
	tenant_value := coalesce(nullif(snapshot ->> 'tenant_code', ''), nullif(current_setting('app.tenant_id', true), ''), '');
	institution_value := coalesce(nullif(snapshot ->> 'institution_id', ''), nullif(current_setting('app.institution_id', true), ''), '');
	changed_by_value := coalesce(nullif(current_setting('app.actor_subject', true), ''), '');

	perform pg_advisory_xact_lock(hashtextextended(
		concat_ws(chr(31), tenant_value, institution_value, tg_table_name, entity_id_value::text),
		0
	));

	select coalesce(max(version_no), 0) + 1
	into next_version
	from public.app_entity_versions
	where tenant_code = tenant_value
		and institution_id = institution_value
		and entity_table = tg_table_name
		and entity_id = entity_id_value;

	insert into public.app_entity_versions (
		entity_table,
		entity_id,
		version_no,
		change_type,
		tenant_code,
		institution_id,
		snapshot,
		changed_by,
		changed_at
	) values (
		tg_table_name,
		entity_id_value,
		next_version,
		lower(tg_op),
		tenant_value,
		institution_value,
		snapshot,
		changed_by_value,
		now()
	);

	return coalesce(new, old);
end;
$$;
