-- Registratura records carry both tenant_code and institution_id. Enforce both
-- dimensions at the database boundary so a connection can never substitute an
-- identifier from another tenant, even if an institution mapping is changed.
select set_config('app.is_super_admin', 'true', true);

do $$
declare
    tbl text;
begin
    foreach tbl in array array[
        'registratura_documents',
        'registre',
        'registratura_departments',
        'registratura_organizations',
        'registratura_user_departments',
        'registratura_user_organizations',
        'registratura_registry_departments',
        'registratura_organization_departments',
        'registratura_document_departments',
        'registratura_document_workflow_events',
        'registratura_document_versions',
        'registratura_document_attachments',
        'registratura_archive_outbox'
    ] loop
        execute format('alter table %I enable row level security', tbl);
        execute format('alter table %I force row level security', tbl);
        execute format('drop policy if exists tenant_isolation on %I', tbl);
        execute format(
            'create policy tenant_isolation on %I using (public.can_bypass_tenant_rls() or (tenant_code = public.current_tenant_code() and institution_id = public.current_institution_id())) with check (public.can_bypass_tenant_rls() or (tenant_code = public.current_tenant_code() and institution_id = public.current_institution_id()))',
            tbl
        );
    end loop;
end $$;
