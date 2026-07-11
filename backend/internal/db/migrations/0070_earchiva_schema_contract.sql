-- Add missing archive tables required by schema contract

create table if not exists archive_document_entities (
    id uuid primary key default gen_random_uuid(),
    institution_id text not null,
    document_id uuid not null references archive_documents(id) on delete cascade,
    entity_type text not null,
    entity_value text not null,
    confidence real not null default 0,
    metadata jsonb not null default '{}'::jsonb,
    created_at timestamptz not null default now()
);

create table if not exists archive_document_relations (
    id uuid primary key default gen_random_uuid(),
    institution_id text not null,
    source_document_id uuid not null references archive_documents(id) on delete cascade,
    target_document_id uuid not null references archive_documents(id) on delete cascade,
    relation_type text not null,
    metadata jsonb not null default '{}'::jsonb,
    created_at timestamptz not null default now()
);

create table if not exists archive_ingestion_jobs (
    id uuid primary key default gen_random_uuid(),
    institution_id text not null,
    status text not null check (status in ('pending', 'running', 'completed', 'failed')),
    source_bucket text not null,
    source_key text not null,
    document_id uuid references archive_documents(id) on delete set null,
    error_message text,
    started_at timestamptz,
    finished_at timestamptz,
    created_at timestamptz not null default now(),
    updated_at timestamptz not null default now()
);

create index if not exists idx_archive_document_entities_document_id on archive_document_entities(document_id);
create index if not exists idx_archive_document_entities_institution_id on archive_document_entities(institution_id);
create index if not exists idx_archive_document_relations_source on archive_document_relations(source_document_id);
create index if not exists idx_archive_document_relations_target on archive_document_relations(target_document_id);
create index if not exists idx_archive_document_relations_institution_id on archive_document_relations(institution_id);
create index if not exists idx_archive_ingestion_jobs_status on archive_ingestion_jobs(status);
create index if not exists idx_archive_ingestion_jobs_institution_id on archive_ingestion_jobs(institution_id);

-- Enable RLS and create tenant_isolation policies for all archive tables
-- These were missing from migration 0068

do $$
declare
    tbl text;
begin
    for tbl in select unnest(array[
        'archive_taxonomy_nodes',
        'archive_documents',
        'archive_document_versions',
        'archive_document_chunks',
        'archive_document_entities',
        'archive_document_relations',
        'archive_ingestion_jobs'
    ])
    loop
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
