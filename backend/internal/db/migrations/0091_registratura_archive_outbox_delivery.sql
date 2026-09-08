-- Delivery state for the durable Registratura -> eArhiva bridge.  A failed
-- outbox event is a dead-letter: it is retained for operator inspection and
-- never silently retried forever.
alter table registratura_archive_outbox
    add column if not exists locked_at timestamptz,
    add column if not exists locked_by text not null default '',
    add column if not exists dead_lettered_at timestamptz;

create index if not exists idx_registratura_archive_outbox_claim
    on registratura_archive_outbox (status, available_at, created_at)
    where status = 'pending';

-- Archive rows created by the bridge remain regular eArhiva documents; the
-- source kind only makes their provenance explicit and searchable.
alter table archive_documents
    drop constraint if exists archive_documents_source_kind_check;
alter table archive_documents
    add constraint archive_documents_source_kind_check
    check (source_kind in ('legacy_pdf', 'upload', 'import', 'registratura_finalized'));

comment on column registratura_archive_outbox.dead_lettered_at is
    'Terminal delivery failure timestamp. Failed rows are dead letters and require an explicit operator retry.';
