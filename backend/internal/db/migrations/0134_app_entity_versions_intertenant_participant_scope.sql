-- A version normally belongs to the entity's canonical tenant/institution.
-- The one intentional exception is a routed portfolio transfer: after send,
-- its destination participant is allowed by the transfer table's RLS policy
-- to record receipt even though the row remains owned by the source. Store
-- that receipt version in the destination participant's history boundary.
-- The snapshot retains the complete source/destination route for audit.

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
	request_tenant_value text;
	request_institution_value text;
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
	request_tenant_value := coalesce(nullif(current_setting('app.tenant_id', true), ''), '');
	request_institution_value := coalesce(nullif(current_setting('app.institution_id', true), ''), '');
	tenant_value := coalesce(nullif(snapshot ->> 'tenant_code', ''), request_tenant_value, '');
	institution_value := coalesce(nullif(snapshot ->> 'institution_id', ''), request_institution_value, '');

	if tg_table_name = 'education_portfolio_transfers'
		and coalesce(snapshot ->> 'routing_version', '1') = '2'
		and request_tenant_value = coalesce(snapshot ->> 'destination_tenant_code', '')
		and request_institution_value = coalesce(snapshot ->> 'destination_institution_id', '') then
		tenant_value := request_tenant_value;
		institution_value := request_institution_value;
	end if;

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
