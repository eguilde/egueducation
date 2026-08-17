-- Reconcile the original eArhiva schema with the document ingestion runtime.
-- This migration is additive and maps legacy values before tightening the new
-- runtime contract; it deliberately does not delete archive material.

alter table archive_document_versions
    add column if not exists source_bucket text,
    add column if not exists source_object_key text,
    add column if not exists artifact_bucket text not null default '',
    add column if not exists artifact_object_key text not null default '',
    add column if not exists source_sha256 text,
    add column if not exists source_size_bytes bigint not null default 0,
    add column if not exists page_count integer not null default 0,
    add column if not exists text_status text not null default 'pending',
    add column if not exists extracted_text text not null default '',
    add column if not exists extracted_metadata jsonb not null default '{}'::jsonb,
    add column if not exists created_by text not null default '',
    add column if not exists updated_at timestamptz not null default now();

update archive_document_versions
set source_bucket = coalesce(nullif(source_bucket, ''), nullif(bucket_name, '')),
    source_object_key = coalesce(nullif(source_object_key, ''), nullif(object_key, '')),
    source_sha256 = coalesce(nullif(source_sha256, ''), nullif(hash_sha256, '')),
    source_size_bytes = case when source_size_bytes = 0 then coalesce(size_bytes, 0) else source_size_bytes end,
    extracted_text = coalesce(nullif(extracted_text, ''), ocr_text, ''),
    text_status = case
        when status = 'active' and coalesce(ocr_text, '') <> '' then 'processed'
        when status = 'active' then 'pending'
        else 'failed'
    end
where source_bucket is null or source_object_key is null or source_sha256 is null
   or extracted_text = '' or text_status = 'pending';

alter table archive_document_versions
    alter column source_bucket set not null,
    alter column source_bucket set default '',
    alter column source_object_key set not null,
    alter column source_object_key set default '',
    alter column source_sha256 set not null,
    alter column source_sha256 set default '';

alter table archive_document_versions drop constraint if exists archive_document_versions_text_status_check;
alter table archive_document_versions add constraint archive_document_versions_text_status_check
    check (text_status in ('pending', 'processing', 'processed', 'failed'));

alter table archive_document_entities
    add column if not exists version_id uuid references archive_document_versions(id) on delete cascade,
    add column if not exists normalized_value text not null default '',
    add column if not exists chunk_no integer not null default 0,
    add column if not exists page_no integer not null default 0;

update archive_document_entities e
set version_id = v.id
from archive_document_versions v
where e.version_id is null and v.document_id = e.document_id and v.version_no = 1;

alter table archive_document_relations alter column target_document_id drop not null;
alter table archive_document_relations
    add column if not exists relation_value text not null default '',
    add column if not exists confidence double precision not null default 0;

alter table archive_ingestion_jobs
    add column if not exists version_id uuid references archive_document_versions(id) on delete set null,
    add column if not exists job_type text not null default 'extract_text',
    add column if not exists available_at timestamptz not null default now(),
    add column if not exists attempts integer not null default 0,
    add column if not exists locked_at timestamptz,
    add column if not exists locked_by text not null default '',
    add column if not exists last_error text not null default '',
    add column if not exists created_by text not null default '';

update archive_ingestion_jobs j
set version_id = v.id
from archive_documents d
join archive_document_versions v
  on v.document_id = d.id and v.version_no = d.current_version_no
where j.version_id is null and j.document_id = d.id;

update archive_ingestion_jobs
set available_at = coalesce(available_at, created_at, now()),
    last_error = coalesce(nullif(last_error, ''), error_message, '');

-- The legacy check permits `completed`, not the runtime's `succeeded`; remove
-- it before translating values or an existing completed row aborts migration.
alter table archive_ingestion_jobs drop constraint if exists archive_ingestion_jobs_status_check;

update archive_ingestion_jobs
set status = case status when 'completed' then 'succeeded' else status end;

-- A legacy queue row without a resolvable immutable version cannot be safely
-- processed. Quarantine it instead of letting a NULL scan poison all claims.
update archive_ingestion_jobs
set status = 'failed',
    last_error = case when last_error = '' then 'legacy ingestion job has no resolvable archive version' else last_error end,
    finished_at = coalesce(finished_at, now()),
    updated_at = now()
where version_id is null and status in ('pending', 'running');

alter table archive_ingestion_jobs alter column source_bucket drop not null;
alter table archive_ingestion_jobs alter column source_key drop not null;
alter table archive_ingestion_jobs add constraint archive_ingestion_jobs_status_check
    check (status in ('pending', 'running', 'succeeded', 'failed'));

create index if not exists idx_archive_ingestion_jobs_claim
    on archive_ingestion_jobs (status, available_at, created_at)
    where status = 'pending';
create index if not exists idx_archive_document_entities_version on archive_document_entities (version_id);
