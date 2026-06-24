do $$
declare
	previous_institution_id text := nullif(current_setting('app.institution_id', true), '');
	previous_tenant_id text := nullif(current_setting('app.tenant_id', true), '');
	previous_is_super_admin text := nullif(current_setting('app.is_super_admin', true), '');
begin
	perform set_config('app.institution_id', 'inst-balotesti', false);
	perform set_config('app.tenant_id', 'tenant-balotesti', false);
	perform set_config('app.is_super_admin', 'true', false);

	insert into registratura_documents (
		registry_number,
		subject,
		document_type,
		direction,
		status,
		correspondent,
		assigned_to,
		institution_id,
		confidentiality,
		summary,
		registered_at,
		due_date,
		registru_id
	)
	select
		v.registry_number,
		v.subject,
		v.document_type,
		v.direction,
		v.status,
		v.correspondent,
		v.assigned_to,
		'inst-balotesti',
		v.confidentiality,
		v.summary,
		v.registered_at,
		v.due_date,
		r.id
	from (
		values
			('BAL-2026-0001', 'Cerere înscriere elev', 'cerere', 'intrare', 'registered', 'DianaMaria Ilhan', 'Secretariat', 'normal', 'Cerere de înscriere pentru anul școlar 2026-2027.', '2026-05-02T08:10:00Z'::timestamptz, '2026-05-16'::date),
			('BAL-2026-0002', 'Răspuns solicitare primărie', 'adresă', 'iesire', 'registered', 'Școala Gimnazială nr. 1 Balotești', 'Primăria Balotești', 'normal', 'Transmitere răspuns oficial către autoritatea locală.', '2026-05-02T11:20:00Z'::timestamptz, '2026-05-09'::date),
			('BAL-2026-0003', 'Proces-verbal consiliu profesoral', 'proces-verbal', 'intern', 'in_workflow', 'Conducerea unității', 'Director', 'internal', 'Proces-verbal pentru ședința consiliului profesoral.', '2026-05-03T09:00:00Z'::timestamptz, '2026-05-20'::date),
			('BAL-2026-0004', 'Notă internă inventar', 'notă', 'intern', 'draft', 'Compartiment administrativ', 'Arhivă', 'internal', 'Notă privind pregătirea inventarului anual al arhivei.', '2026-05-04T10:00:00Z'::timestamptz, null),
			('BAL-2026-0005', 'Solicitare adeverință vechime', 'cerere', 'intrare', 'archived', 'Thomas Galambos', 'Resurse umane', 'confidential', 'Solicitare pentru emiterea adeverinței de vechime.', '2026-05-05T08:45:00Z'::timestamptz, '2026-05-18'::date)
	) as v(
		registry_number,
		subject,
		document_type,
		direction,
		status,
		correspondent,
		assigned_to,
		confidentiality,
		summary,
		registered_at,
		due_date
	)
	cross join lateral (
		select id
		from registre
		order by is_default desc, id asc
		limit 1
	) r
	on conflict (registry_number) do update
	set subject = excluded.subject,
		document_type = excluded.document_type,
		direction = excluded.direction,
		status = excluded.status,
		correspondent = excluded.correspondent,
		assigned_to = excluded.assigned_to,
		institution_id = excluded.institution_id,
		confidentiality = excluded.confidentiality,
		summary = excluded.summary,
		registered_at = excluded.registered_at,
		due_date = excluded.due_date,
		registru_id = excluded.registru_id,
		updated_at = now();

	perform set_config('app.institution_id', coalesce(previous_institution_id, ''), false);
	perform set_config('app.tenant_id', coalesce(previous_tenant_id, ''), false);
	perform set_config('app.is_super_admin', coalesce(previous_is_super_admin, 'false'), false);
end;
$$;
