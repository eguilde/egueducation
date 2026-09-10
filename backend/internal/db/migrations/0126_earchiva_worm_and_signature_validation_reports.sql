-- Storage WORM and signature-validation provenance.
-- Object Lock is enforced and verified by the S3/MinIO storage adapter. These
-- triggers independently prevent the database from rewriting or deleting the
-- identity and digest of an archived bitstream after it has been recorded.

alter table archive_document_versions
    add column if not exists source_object_version_id text not null default '',
    add column if not exists source_object_etag text not null default '',
    add column if not exists retention_until timestamptz,
    add column if not exists legal_hold_active boolean not null default false;

alter table archive_document_versions
    drop constraint if exists archive_document_versions_retention_contract,
    add constraint archive_document_versions_retention_contract
        check ((source_object_version_id = '' and retention_until is null)
            or (btrim(source_object_version_id) <> '' and retention_until is not null));

create or replace function public.enforce_archive_version_immutability()
returns trigger
language plpgsql
security definer
set search_path = pg_catalog, public
as $$
begin
    if tg_op = 'DELETE' then
        raise exception 'archive document versions are immutable and cannot be deleted';
    end if;
    if row(new.document_id, new.institution_id, new.version_no, new.mime_type,
           new.bucket_name, new.object_key, new.hash_sha256, new.size_bytes,
           new.source_bucket, new.source_object_key, new.source_sha256,
           new.source_size_bytes, new.source_object_version_id,
           new.source_object_etag, new.retention_until, new.created_at)
       is distinct from
       row(old.document_id, old.institution_id, old.version_no, old.mime_type,
           old.bucket_name, old.object_key, old.hash_sha256, old.size_bytes,
           old.source_bucket, old.source_object_key, old.source_sha256,
           old.source_size_bytes, old.source_object_version_id,
           old.source_object_etag, old.retention_until, old.created_at) then
        raise exception 'archive document version bitstream provenance is immutable';
    end if;
    return new;
end
$$;

revoke all on function public.enforce_archive_version_immutability() from public;
drop trigger if exists trg_archive_version_immutability on archive_document_versions;
create trigger trg_archive_version_immutability
before update or delete on archive_document_versions
for each row execute function public.enforce_archive_version_immutability();

create unique index if not exists archive_versions_institution_document_id_key
    on archive_document_versions(institution_id, document_id, id);

alter table education_signed_artifact_evidence
    drop constraint if exists education_signed_artifact_evidence_archive_version_fk,
    add constraint education_signed_artifact_evidence_archive_version_fk
        foreign key (institution_id, storage_document_id, storage_version_id)
        references archive_document_versions(institution_id, document_id, id)
        on delete restrict
        not valid;

alter table education_signed_artifact_validations
    add column if not exists validator_provider text not null default '',
    add column if not exists validator_version text not null default '',
    add column if not exists validation_policy text not null default '',
    add column if not exists observed_sha256 text not null default '',
    add column if not exists observed_size_bytes bigint not null default 0,
    add column if not exists diagnostic_data jsonb not null default '{}'::jsonb,
    add column if not exists detailed_report jsonb not null default '{}'::jsonb,
    add column if not exists simple_report jsonb not null default '{}'::jsonb,
    add column if not exists etsi_validation_report jsonb not null default '{}'::jsonb;

alter table education_signed_artifact_validations
    drop constraint if exists education_signed_artifact_validations_observed_sha256_check,
    drop constraint if exists education_signed_artifact_validations_observed_size_check,
    add constraint education_signed_artifact_validations_observed_sha256_check
        check (observed_sha256 = '' or observed_sha256 ~ '^[0-9a-f]{64}$'),
    add constraint education_signed_artifact_validations_observed_size_check
        check (observed_size_bytes >= 0);

comment on column education_signed_artifact_validations.diagnostic_data is 'Immutable DSS Diagnostic Data report snapshot.';
comment on column education_signed_artifact_validations.detailed_report is 'Immutable DSS Detailed Report snapshot.';
comment on column education_signed_artifact_validations.simple_report is 'Immutable DSS Simple Report snapshot.';
comment on column education_signed_artifact_validations.etsi_validation_report is 'Immutable DSS ETSI Validation Report snapshot.';
