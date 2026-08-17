-- OCR classification is a suggestion-only workflow.  It deliberately lives
-- beside the immutable bitstream/version, rather than mutating document
-- metadata, taxonomy, or OCR output.

create table if not exists archive_document_classification_reviews (
    id uuid primary key default gen_random_uuid(),
    institution_id text not null,
    document_id uuid not null,
    version_id uuid not null,
    state text not null check (state in ('pending_review', 'needs_review', 'approved', 'corrected')),
    revision integer not null default 1 check (revision > 0),
    suggestion jsonb not null default '{}'::jsonb,
    suggestion_confidence double precision not null default 0 check (suggestion_confidence >= 0 and suggestion_confidence <= 1),
    suggestion_source text not null default '',
    requires_human_review boolean not null default true,
    generated_at timestamptz not null default now(),
    reviewed_at timestamptz,
    reviewed_by text not null default '',
    final_classification jsonb not null default '{}'::jsonb,
    review_note text not null default '',
    created_at timestamptz not null default now(),
    updated_at timestamptz not null default now(),
    unique (institution_id, document_id, version_id),
    foreign key (institution_id, document_id)
        references archive_documents (institution_id, id) on delete cascade,
    foreign key (institution_id, version_id)
        references archive_document_versions (institution_id, id) on delete cascade
);

create index if not exists idx_archive_classification_reviews_pending
    on archive_document_classification_reviews (institution_id, state, generated_at asc)
    where state in ('pending_review', 'needs_review');

alter table archive_document_classification_reviews enable row level security;
alter table archive_document_classification_reviews force row level security;
drop policy if exists tenant_isolation on archive_document_classification_reviews;
create policy tenant_isolation on archive_document_classification_reviews
    using (public.can_bypass_tenant_rls() or institution_id = public.current_institution_id())
    with check (public.can_bypass_tenant_rls() or institution_id = public.current_institution_id());

insert into app_permissions (code, label) values
    ('earchiva.review', 'Review eArchive OCR classification suggestions')
on conflict (code) do update set label = excluded.label;

insert into app_role_permissions (role_code, permission_code)
select role_code, 'earchiva.review'
from app_roles
where role_code in ('super_admin', 'admin', 'arhivar')
on conflict do nothing;
