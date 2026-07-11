create or replace function immutable_jsonb_to_text(jb jsonb) returns text
    language sql immutable strict parallel safe
    return jb::text;

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
	search_tsv tsvector generated always as (
		to_tsvector(
			'simple',
			concat_ws(
				' ',
				title,
				original_file_name,
				source_kind,
				source_system,
				external_reference,
				coalesce(immutable_jsonb_to_text(metadata), '')
			)
		)
	) stored,
	current_version_no integer not null default 1,
	received_at timestamptz not null default now(),
	created_by text not null default '',
	created_at timestamptz not null default now(),
	updated_at timestamptz not null default now()
);

create table if not exists archive_document_versions (
	id uuid primary key default gen_random_uuid(),
	institution_id text not null,
	document_id uuid not null references archive_documents(id) on delete cascade,
	version_no integer not null,
	source_bucket text not null default '',
	source_object_key text not null default '',
	artifact_bucket text not null default '',
	artifact_object_key text not null default '',
	source_sha256 text not null default '',
	source_size_bytes bigint not null default 0,
	page_count integer not null default 0,
	text_status text not null check (text_status in ('pending', 'processed', 'failed')),
	extracted_text text not null default '',
	extracted_metadata jsonb not null default '{}'::jsonb,
	created_by text not null default '',
	created_at timestamptz not null default now(),
	updated_at timestamptz not null default now(),
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
	content_tsv tsvector generated always as (to_tsvector('simple', content)) stored,
	created_at timestamptz not null default now(),
	unique (version_id, chunk_no)
);

create table if not exists archive_document_entities (
	id uuid primary key default gen_random_uuid(),
	institution_id text not null,
	version_id uuid not null references archive_document_versions(id) on delete cascade,
	entity_type text not null,
	entity_value text not null,
	normalized_value text not null default '',
	confidence numeric(4,3) not null default 1,
	chunk_no integer not null default 0,
	page_no integer not null default 0,
	created_at timestamptz not null default now()
);

create table if not exists archive_document_relations (
	id uuid primary key default gen_random_uuid(),
	institution_id text not null,
	source_document_id uuid not null references archive_documents(id) on delete cascade,
	target_document_id uuid references archive_documents(id) on delete set null,
	relation_type text not null,
	relation_value text not null default '',
	confidence numeric(4,3) not null default 1,
	metadata jsonb not null default '{}'::jsonb,
	created_at timestamptz not null default now()
);

create table if not exists archive_ingestion_jobs (
	id uuid primary key default gen_random_uuid(),
	institution_id text not null,
	document_id uuid not null references archive_documents(id) on delete cascade,
	version_id uuid not null references archive_document_versions(id) on delete cascade,
	job_type text not null check (job_type in ('extract_text', 'reprocess')),
	status text not null check (status in ('pending', 'running', 'succeeded', 'failed')),
	attempts integer not null default 0,
	available_at timestamptz not null default now(),
	locked_at timestamptz,
	locked_by text not null default '',
	last_error text not null default '',
	payload jsonb not null default '{}'::jsonb,
	created_by text not null default '',
	created_at timestamptz not null default now(),
	updated_at timestamptz not null default now()
);

create index if not exists idx_archive_documents_institution_status on archive_documents (institution_id, status, updated_at desc);
create index if not exists idx_archive_documents_search_tsv on archive_documents using gin (search_tsv);
create index if not exists idx_archive_documents_taxonomy on archive_documents (taxonomy_node_id);
create index if not exists idx_archive_taxonomy_nodes_institution_code on archive_taxonomy_nodes (institution_id, code);
create index if not exists idx_archive_document_versions_document on archive_document_versions (document_id, version_no desc);
create index if not exists idx_archive_document_versions_bucket on archive_document_versions (source_bucket, source_object_key);
create index if not exists idx_archive_document_chunks_version on archive_document_chunks (version_id, chunk_no);
create index if not exists idx_archive_document_chunks_tsv on archive_document_chunks using gin (content_tsv);
create index if not exists idx_archive_document_entities_version on archive_document_entities (version_id, entity_type);
create index if not exists idx_archive_document_relations_source on archive_document_relations (source_document_id, relation_type);
create index if not exists idx_archive_ingestion_jobs_status_available on archive_ingestion_jobs (status, available_at);

do $$
declare
	tbl text;
begin
	foreach tbl in array array[
		'archive_taxonomy_nodes',
		'archive_documents',
		'archive_document_versions',
		'archive_document_chunks',
		'archive_document_entities',
		'archive_document_relations',
		'archive_ingestion_jobs'
	] loop
		execute format('alter table %I enable row level security', tbl);
		execute format('alter table %I force row level security', tbl);
		execute format('drop policy if exists tenant_isolation on %I', tbl);
		execute format(
			'create policy tenant_isolation on %I using (public.can_bypass_tenant_rls() or institution_id = public.current_institution_id()) with check (public.can_bypass_tenant_rls() or institution_id = public.current_institution_id())',
			tbl
		);
	end loop;
end;
$$;
