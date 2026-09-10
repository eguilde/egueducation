-- Evidence of an electronic signature is an append-only legal record.  This
-- migration intentionally stores verification facts and references only: never
-- private keys, certificate private material, or timestamp tokens themselves.
do $$
begin
    if not exists (
        select 1 from pg_catalog.pg_roles
        where rolname = current_user and (rolsuper or rolbypassrls)
    ) then
        raise exception 'migration owner must be SUPERUSER or BYPASSRLS for signed artifact FORCE-RLS validation';
    end if;
end;
$$;

create table if not exists education_signed_artifact_evidence (
    id uuid primary key default gen_random_uuid(),
    tenant_code text not null,
    institution_id text not null,
    artifact_type text not null check (artifact_type in ('decision', 'publication', 'managerial_document', 'meeting_document', 'meeting_minute', 'meeting_resolution')),
    artifact_id uuid not null,
    document_sha256 text not null check (document_sha256 ~ '^[0-9a-f]{64}$'),
    signature_format text not null check (signature_format in ('PAdES', 'XAdES', 'CAdES')),
    signature_level text not null check (signature_level in ('advanced', 'qualified')),
    signature_subject text not null check (btrim(signature_subject) <> ''),
    certificate_issuer text not null check (btrim(certificate_issuer) <> ''),
    certificate_serial text not null check (btrim(certificate_serial) <> ''),
    certificate_valid_from timestamptz not null,
    certificate_valid_until timestamptz not null,
    storage_document_id uuid references archive_documents(id) on delete restrict,
    storage_version_id uuid references archive_document_versions(id) on delete restrict,
    storage_bucket text not null default '',
    storage_object_key text not null default '',
    submitted_by_subject text not null default '',
    submitted_at timestamptz not null default now(),
    constraint education_signed_artifact_evidence_tenant_fk foreign key (tenant_code, institution_id) references app_tenants(code, institution_id) on delete restrict,
    constraint education_signed_artifact_evidence_certificate_window check (certificate_valid_until >= certificate_valid_from),
    constraint education_signed_artifact_evidence_storage_reference check ((storage_document_id is null and storage_version_id is null) or (storage_document_id is not null and storage_version_id is not null))
);

create table if not exists education_signed_artifact_validations (
    id uuid primary key default gen_random_uuid(),
    evidence_id uuid not null references education_signed_artifact_evidence(id) on delete restrict,
    tenant_code text not null,
    institution_id text not null,
    validation_status text not null check (validation_status in ('pending', 'valid', 'invalid', 'error')),
    trusted_list_provider text not null default '',
    validated_at timestamptz not null default now(),
    timestamp_token_sha256 text not null default '' check (timestamp_token_sha256 = '' or timestamp_token_sha256 ~ '^[0-9a-f]{64}$'),
    timestamp_at timestamptz,
    timestamp_authority text not null default '',
    findings jsonb not null default '{}'::jsonb,
    validated_by_subject text not null default '',
    constraint education_signed_artifact_validations_tenant_fk foreign key (tenant_code, institution_id) references app_tenants(code, institution_id) on delete restrict,
    constraint education_signed_artifact_validations_timestamp_provenance check ((timestamp_token_sha256 = '' and timestamp_at is null and timestamp_authority = '') or (timestamp_token_sha256 <> '' and timestamp_at is not null and btrim(timestamp_authority) <> '')),
    constraint education_signed_artifact_validations_qualified_needs_trust check (validation_status <> 'valid' or btrim(trusted_list_provider) <> '')
);

create index if not exists idx_education_signed_artifact_evidence_artifact on education_signed_artifact_evidence (tenant_code, institution_id, artifact_type, artifact_id, submitted_at desc);
create index if not exists idx_education_signed_artifact_validations_latest on education_signed_artifact_validations (evidence_id, validated_at desc, id desc);

create or replace function public.enforce_education_signed_artifact_evidence()
returns trigger language plpgsql security definer set search_path = pg_catalog, public as $$
declare
    actor_subject text := nullif(btrim(current_setting('app.actor_subject', true)), '');
    artifact_found boolean := false;
    document_in_scope boolean := false;
    session_can_bypass boolean := false;
begin
    if tg_op in ('UPDATE', 'DELETE') then
        raise exception 'signed artifact evidence is append-only';
    end if;
    select coalesce(role_row.rolsuper or role_row.rolbypassrls, false)
    into session_can_bypass
    from pg_catalog.pg_roles role_row where role_row.rolname = session_user;
    if actor_subject is not null then
        new.submitted_by_subject := actor_subject;
    elsif not session_can_bypass then
        raise exception 'signed artifact evidence requires authenticated actor provenance';
    end if;
    if not public.can_bypass_tenant_rls() then
        if new.tenant_code <> public.current_tenant_code() or new.institution_id <> public.current_institution_id() then
            raise exception 'signed artifact evidence tenant/institution context mismatch';
        end if;
    end if;
    case new.artifact_type
        when 'decision' then select exists(select 1 from education_decisions where id=new.artifact_id and institution_id=new.institution_id) into artifact_found;
        when 'publication' then select exists(select 1 from education_publications where id=new.artifact_id and institution_id=new.institution_id) into artifact_found;
        when 'managerial_document' then select exists(select 1 from education_managerial_documents where id=new.artifact_id and institution_id=new.institution_id) into artifact_found;
        when 'meeting_document' then select exists(select 1 from education_meeting_documents where id=new.artifact_id and institution_id=new.institution_id) into artifact_found;
        when 'meeting_minute' then select exists(select 1 from education_meeting_minutes where id=new.artifact_id and institution_id=new.institution_id) into artifact_found;
        when 'meeting_resolution' then select exists(select 1 from education_meeting_resolutions where id=new.artifact_id and institution_id=new.institution_id) into artifact_found;
    end case;
    if not artifact_found then raise exception 'signed artifact evidence must reference an artifact in the active institution'; end if;
    if new.storage_document_id is not null then
        select exists(select 1 from archive_documents document join archive_document_versions version on version.id=new.storage_version_id and version.document_id=document.id where document.id=new.storage_document_id and document.institution_id=new.institution_id and version.institution_id=new.institution_id) into document_in_scope;
        if not document_in_scope then raise exception 'signed artifact evidence storage references must belong to active institution'; end if;
    end if;
    return new;
end $$;

revoke all on function public.enforce_education_signed_artifact_evidence() from public;

create or replace function public.enforce_education_signed_artifact_validation()
returns trigger language plpgsql security definer set search_path = pg_catalog, public as $$
declare actor_subject text := nullif(btrim(current_setting('app.actor_subject', true)), ''); evidence_in_scope boolean := false; session_can_bypass boolean := false;
begin
    if tg_op in ('UPDATE', 'DELETE') then raise exception 'signed artifact validation is append-only'; end if;
    select coalesce(role_row.rolsuper or role_row.rolbypassrls, false)
    into session_can_bypass
    from pg_catalog.pg_roles role_row where role_row.rolname = session_user;
    if actor_subject is not null then
        new.validated_by_subject := actor_subject;
    elsif not session_can_bypass then
        raise exception 'signed artifact validation requires authenticated actor provenance';
    end if;
    if not public.can_bypass_tenant_rls() then
        if new.tenant_code <> public.current_tenant_code() or new.institution_id <> public.current_institution_id() then raise exception 'signed artifact validation tenant/institution context mismatch'; end if;
    end if;
    select exists(select 1 from education_signed_artifact_evidence where id=new.evidence_id and tenant_code=new.tenant_code and institution_id=new.institution_id) into evidence_in_scope;
    if not evidence_in_scope then raise exception 'signed artifact validation must reference evidence in same tenant/institution'; end if;
    return new;
end $$;

revoke all on function public.enforce_education_signed_artifact_validation() from public;

create trigger trg_education_signed_artifact_evidence_append_only before insert or update or delete on education_signed_artifact_evidence for each row execute function public.enforce_education_signed_artifact_evidence();
create trigger trg_education_signed_artifact_validation_append_only before insert or update or delete on education_signed_artifact_validations for each row execute function public.enforce_education_signed_artifact_validation();

alter table education_signed_artifact_evidence enable row level security;
alter table education_signed_artifact_evidence force row level security;
alter table education_signed_artifact_validations enable row level security;
alter table education_signed_artifact_validations force row level security;
create policy tenant_isolation on education_signed_artifact_evidence using (public.can_bypass_tenant_rls() or (tenant_code=public.current_tenant_code() and institution_id=public.current_institution_id())) with check (public.can_bypass_tenant_rls() or (tenant_code=public.current_tenant_code() and institution_id=public.current_institution_id()));
create policy tenant_isolation on education_signed_artifact_validations using (public.can_bypass_tenant_rls() or (tenant_code=public.current_tenant_code() and institution_id=public.current_institution_id())) with check (public.can_bypass_tenant_rls() or (tenant_code=public.current_tenant_code() and institution_id=public.current_institution_id()));

insert into app_permissions(code, label) values
    ('education.signatures.read', 'Read School signed artifact evidence'),
    ('education.signatures.manage', 'Submit School signed artifact evidence'),
    ('education.signatures.validate', 'Revalidate School signed artifact evidence')
on conflict (code) do update set label=excluded.label;
insert into app_position_permissions(position_code, permission_code) values
    ('director', 'education.signatures.read'), ('director', 'education.signatures.manage'), ('director', 'education.signatures.validate'),
    ('director_adjunct', 'education.signatures.read'), ('director_adjunct', 'education.signatures.manage'),
    ('portfolio_custodian', 'education.signatures.read'), ('portfolio_custodian', 'education.signatures.manage')
on conflict do nothing;
