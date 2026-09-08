-- Costesti document-flux parity, expressed on the tenant-scoped Registratura
-- aggregate.  Assignment is stored on the aggregate (not reconstructed from
-- a mutable UI projection), while the event stream remains the audit record.
select set_config('app.is_super_admin', 'true', true);

alter table registratura_documents
  add column if not exists workflow_department_id uuid references registratura_departments(id) on delete restrict,
  add column if not exists workflow_assigned_user_id uuid references app_users(id) on delete restrict,
  add column if not exists workflow_target_approver_id uuid references app_users(id) on delete restrict,
  add column if not exists rejection_count integer not null default 0 check (rejection_count >= 0),
  add column if not exists workflow_locked_until timestamptz;

alter table registratura_documents
  add constraint registratura_documents_workflow_department_tenant_fk
  foreign key (tenant_code, workflow_department_id)
  references registratura_departments(tenant_code, id) on delete restrict;

create index if not exists idx_registratura_documents_flux_queue
  on registratura_documents(institution_id, status, workflow_assigned_user_id, workflow_department_id, updated_at desc);
create index if not exists idx_registratura_documents_flux_approvals
  on registratura_documents(institution_id, status, workflow_target_approver_id, updated_at desc);

-- A completed approval publishes a durable intent only.  The eArhiva worker
-- owns ingestion and is the sole component allowed to mark this outbox row
-- delivered; an HTTP workflow response never claims archive success.
create table if not exists registratura_archive_outbox (
  id uuid primary key default gen_random_uuid(),
  tenant_code text not null references app_tenants(code) on delete restrict,
  institution_id text not null,
  document_id uuid not null,
  event_type text not null check (event_type = 'document_finalized'),
  payload jsonb not null default '{}'::jsonb,
  status text not null default 'pending' check (status in ('pending','processing','delivered','failed')),
  attempts integer not null default 0 check (attempts >= 0),
  available_at timestamptz not null default now(),
  delivered_at timestamptz,
  last_error text not null default '',
  created_at timestamptz not null default now(),
  unique (tenant_code, document_id, event_type),
  foreign key (tenant_code, document_id) references registratura_documents(tenant_code, id) on delete restrict,
  foreign key (tenant_code, institution_id) references app_tenants(code, institution_id) on delete restrict
);
alter table registratura_archive_outbox enable row level security;
alter table registratura_archive_outbox force row level security;
drop policy if exists tenant_isolation on registratura_archive_outbox;
create policy tenant_isolation on registratura_archive_outbox
  using (public.can_bypass_tenant_rls() or institution_id = public.current_institution_id())
  with check (public.can_bypass_tenant_rls() or institution_id = public.current_institution_id());
