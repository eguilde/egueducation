-- archive_documents search_tsv trigger function
create or replace function archive_documents_search_tsv_trigger() returns trigger as $$
begin
    new.search_tsv := to_tsvector('simple', concat_ws(' ',
        new.title,
        new.original_file_name,
        new.source_kind,
        new.source_system,
        new.external_reference,
        coalesce(new.metadata::text, '')
    ));
    return new;
end;
$$ language plpgsql;

-- archive_document_chunks content_tsv trigger function
create or replace function archive_document_chunks_content_tsv_trigger() returns trigger as $$
begin
    new.content_tsv := to_tsvector('simple', new.content);
    return new;
end;
$$ language plpgsql;

-- v3: use triggers instead of generated columns for tsvector

create table if not exists archive_taxonomy_nodes (
	id uuid primary key default gen_random_uuid(),
	institution_id text not null,
	parent_id uuid references archive_taxonomy_nodes(id) on delete cascade,
	code text not null,
	label text not null,
	description text not null default '',
	path text not null default '',
	active boolean not null default true,
	sort_order integer not null default 0,
	created_at timestamptz not null default now(),
	updated_at timestamptz not null default now(),
	unique (institution_id, code)
);

create table if not exists archive_documents (
	id uuid primary key default gen_random_uuid(),
	institution_id text not null,
	title text not null,
	original_file_name text not null,
	mime_type text not null,
	source_kind text not null check (source_kind in ('legacy_pdf', 'upload', 'import')),
	source_system text not null default '',
	external_reference text not null default '',
	taxonomy_node_id uuid references archive_taxonomy_nodes(id) on delete set null,
	status text not null check (status in ('queued', 'processing', 'ready', 'failed', 'archived')),
	original_bucket text not null default '',
	original_object_key text not null default '',
	artifact_bucket text not null default '',
	artifact_object_key text not null default '',
	document_date date,
	metadata jsonb not null default '{}'::jsonb,
	search_tsv tsvector,
	current_version_no integer not null default 1,
	received_at timestamptz not null default now(),
	created_by text not null default '',
	created_at timestamptz not null default now(),
	updated_at timestamptz not null default now()
);

create trigger trg_archive_documents_search_tsv
    before insert or update of title, original_file_name, source_kind, source_system, external_reference, metadata
    on archive_documents
    for each row execute function archive_documents_search_tsv_trigger();

create table if not exists archive_document_versions (
	id uuid primary key default gen_random_uuid(),
	document_id uuid not null references archive_documents(id) on delete cascade,
	institution_id text not null,
	version_no integer not null,
	mime_type text not null,
	title text not null default '',
	bucket_name text not null,
	object_key text not null,
	hash_sha256 text not null,
	size_bytes bigint not null default 0,
	metadata jsonb not null default '{}'::jsonb,
	ocr_text text not null default '',
	status text not null check (status in ('active', 'archived', 'purged')),
	ingested_at timestamptz not null default now(),
	created_at timestamptz not null default now(),
	unique (document_id, version_no)
);

create table if not exists archive_document_chunks (
	id uuid primary key default gen_random_uuid(),
	institution_id text not null,
	version_id uuid not null references archive_document_versions(id) on delete cascade,
	chunk_no integer not null,
	page_no integer not null default 0,
	content text not null,
	char_start integer not null default 0,
	char_end integer not null default 0,
	content_tsv tsvector,
	created_at timestamptz not null default now(),
	unique (version_id, chunk_no)
);

create trigger trg_archive_document_chunks_content_tsv
    before insert or update of content
    on archive_document_chunks
    for each row execute function archive_document_chunks_content_tsv_trigger();

create index if not exists idx_archive_document_versions_document_id on archive_document_versions(document_id);
create index if not exists idx_archive_document_versions_hash on archive_document_versions(hash_sha256);
create index if not exists idx_archive_documents_status on archive_documents(status);
create index if not exists idx_archive_documents_taxonomy_node_id on archive_documents(taxonomy_node_id);
create index if not exists idx_archive_documents_search_tsv on archive_documents using gin (search_tsv);
create index if not exists idx_archive_documents_institution_id on archive_documents(institution_id);
create index if not exists idx_archive_document_versions_institution_id on archive_document_versions(institution_id);
create index if not exists idx_archive_document_chunks_version_id on archive_document_chunks(version_id);
create index if not exists idx_archive_document_chunks_tsv on archive_document_chunks using gin (content_tsv);
create index if not exists idx_archive_document_chunks_institution_id on archive_document_chunks(institution_id);

-- Populate existing search_tsv values if the table was created before this migration
update archive_documents set search_tsv = to_tsvector('simple', concat_ws(' ',
    title, original_file_name, source_kind, source_system, external_reference,
    coalesce(metadata::text, ''))) where search_tsv is null;
update archive_document_chunks set content_tsv = to_tsvector('simple', content) where content_tsv is null;

create or replace function enable_rls_for_tables() returns void as $$
declare
    tbl text;
begin
    for tbl in select unnest(array[
        'archive_taxonomy_nodes',
        'archive_documents',
        'archive_document_versions',
        'archive_document_chunks'
    ])
    loop
        execute format('alter table %I enable row level security', tbl);
        execute format('alter table %I force row level security', tbl);
    end loop;
end;
$$ language plpgsql;

select enable_rls_for_tables();
drop function enable_rls_for_tables();
