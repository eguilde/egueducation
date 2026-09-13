-- 0149 expanded signed evidence to admission artifacts but accidentally
-- replaced the server-owned archive provenance derivation introduced by 0117
-- with a scope-only existence check. Reconcile both contracts in the final
-- trigger definition: all supported artifact kinds remain available, while
-- browser-supplied digest and storage coordinates are never authoritative.
create or replace function public.enforce_education_signed_artifact_evidence()
returns trigger
language plpgsql
security definer
set search_path = pg_catalog, public
as $$
declare
    actor_subject text := nullif(btrim(current_setting('app.actor_subject', true)), '');
    artifact_found boolean := false;
    canonical_sha256 text;
    canonical_bucket text;
    canonical_object_key text;
    session_can_bypass boolean := false;
begin
    if tg_op in ('UPDATE', 'DELETE') then
        raise exception 'signed artifact evidence is append-only';
    end if;

    select coalesce(role_row.rolsuper or role_row.rolbypassrls, false)
    into session_can_bypass
    from pg_catalog.pg_roles role_row
    where role_row.rolname = session_user;

    if actor_subject is not null then
        new.submitted_by_subject := actor_subject;
    elsif not session_can_bypass then
        raise exception 'signed artifact evidence requires authenticated actor provenance';
    end if;

    if not public.can_bypass_tenant_rls() and (
        new.tenant_code <> public.current_tenant_code()
        or new.institution_id <> public.current_institution_id()
    ) then
        raise exception 'signed artifact evidence tenant/institution context mismatch';
    end if;

    case new.artifact_type
        when 'decision' then
            select exists(
                select 1 from education_decisions
                where id = new.artifact_id
                  and institution_id = new.institution_id
                  and status <> 'blocked'
            ) into artifact_found;
        when 'publication' then
            select exists(
                select 1 from education_publications
                where id = new.artifact_id
                  and institution_id = new.institution_id
                  and publication_status <> 'retras'
            ) into artifact_found;
        when 'managerial_document' then
            select exists(
                select 1 from education_managerial_documents
                where id = new.artifact_id
                  and institution_id = new.institution_id
                  and document_status <> 'archived'
            ) into artifact_found;
        when 'meeting_document' then
            select exists(
                select 1 from education_meeting_documents
                where id = new.artifact_id
                  and institution_id = new.institution_id
            ) into artifact_found;
        when 'meeting_minute' then
            select exists(
                select 1 from education_meeting_minutes
                where id = new.artifact_id
                  and institution_id = new.institution_id
            ) into artifact_found;
        when 'meeting_resolution' then
            select exists(
                select 1 from education_meeting_resolutions
                where id = new.artifact_id
                  and institution_id = new.institution_id
            ) into artifact_found;
        when 'admission_decision' then
            select exists(
                select 1 from school_admission_decisions
                where tenant_code = new.tenant_code
                  and institution_id = new.institution_id
                  and id = new.artifact_id
            ) into artifact_found;
        when 'admission_appeal_resolution' then
            select exists(
                select 1 from school_admission_appeal_resolutions
                where tenant_code = new.tenant_code
                  and institution_id = new.institution_id
                  and id = new.artifact_id
            ) into artifact_found;
    end case;

    if not artifact_found then
        raise exception 'signed artifact evidence must reference an active artifact in the same tenant and institution';
    end if;

    if new.storage_document_id is null or new.storage_version_id is null then
        raise exception 'signed artifact evidence requires an archive document version';
    end if;

    select lower(version.source_sha256), version.source_bucket, version.source_object_key
    into canonical_sha256, canonical_bucket, canonical_object_key
    from archive_documents document
    join archive_document_versions version on version.document_id = document.id
    where document.id = new.storage_document_id
      and version.id = new.storage_version_id
      and document.institution_id = new.institution_id
      and version.institution_id = new.institution_id
      and document.status = 'ready'
      and version.status = 'active'
      and version.source_sha256 ~ '^[0-9a-fA-F]{64}$'
      and nullif(btrim(version.source_bucket), '') is not null
      and nullif(btrim(version.source_object_key), '') is not null;

    if not found then
        raise exception 'signed artifact evidence requires a ready archive version in the same institution';
    end if;

    new.document_sha256 := canonical_sha256;
    new.storage_bucket := canonical_bucket;
    new.storage_object_key := canonical_object_key;
    return new;
end
$$;

revoke all on function public.enforce_education_signed_artifact_evidence() from public;
