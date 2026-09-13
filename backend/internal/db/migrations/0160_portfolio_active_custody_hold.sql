-- Active portfolios have no cessation-derived retention deadline. Their exact
-- original is nevertheless custody-protected with Object Lock; this is not a
-- judicial/legal hold on the portfolio.
alter table archive_document_versions
  add column if not exists custody_hold_active boolean not null default false;

alter table archive_document_versions
  drop constraint if exists archive_document_versions_retention_contract,
  add constraint archive_document_versions_retention_contract check (
    (source_object_version_id='' and retention_until is null and not custody_hold_active)
    or (btrim(source_object_version_id)<>'' and (retention_until is not null or custody_hold_active))
  );

comment on column archive_document_versions.custody_hold_active is
  'Storage Object Lock custody hold for an active portfolio; distinct from portfolio legal hold.';

create or replace function public.enforce_archive_version_immutability()
returns trigger language plpgsql security definer set search_path=pg_catalog,public as $$
begin
 if tg_op='DELETE' then raise exception 'archive document versions are immutable and cannot be deleted'; end if;
 if row(new.document_id,new.institution_id,new.version_no,new.mime_type,new.bucket_name,new.object_key,new.hash_sha256,new.size_bytes,new.source_bucket,new.source_object_key,new.source_sha256,new.source_size_bytes,new.source_object_version_id,new.source_object_etag,new.retention_until,new.custody_hold_active,new.created_at)
    is distinct from
    row(old.document_id,old.institution_id,old.version_no,old.mime_type,old.bucket_name,old.object_key,old.hash_sha256,old.size_bytes,old.source_bucket,old.source_object_key,old.source_sha256,old.source_size_bytes,old.source_object_version_id,old.source_object_etag,old.retention_until,old.custody_hold_active,old.created_at) then
  raise exception 'archive document version bitstream provenance is immutable';
 end if;
 return new;
end $$;
