-- School operations: contracts, utilities and statutory compliance. Every
-- business record is bound to the host-selected tenant+institution scope.
select set_config('app.is_super_admin', 'true', true);

insert into app_permissions(code, label) values
 ('school_operations.read','Read school operational contracts, utilities and compliance'),
 ('school_operations.manage','Create and amend school operational records'),
 ('school_operations.approve','Approve, sign and close school operational records'),
 ('school_operations.compliance.manage','Manage statutory compliance controls')
on conflict (code) do update set label=excluded.label;
insert into app_roles(code,label) values
 ('economist','Economist'),('responsabil_ssm','Responsabil SSM/PSI')
on conflict (code) do update set label=excluded.label;
insert into app_role_permissions(role_code,permission_code)
select role_code,permission_code from (values
 ('admin','school_operations.read'),('admin','school_operations.manage'),('admin','school_operations.approve'),('director','school_operations.read'),('director','school_operations.approve'),
 ('economist','school_operations.read'),('economist','school_operations.manage'),('responsabil_ssm','school_operations.read'),('responsabil_ssm','school_operations.compliance.manage')
) x(role_code,permission_code) on conflict do nothing;

-- app_parties predates composite scope keys. Add the referenced key once so
-- suppliers cannot be substituted across a tenant/institution boundary.
alter table app_parties add constraint app_parties_scope_id_unique unique (tenant_code,institution_id,id);
alter table app_parties add constraint app_parties_tenant_institution_fk
 foreign key (tenant_code,institution_id) references app_tenants(code,institution_id) on delete restrict;

create table school_operation_idempotency (
 tenant_code text not null, institution_id text not null, operation text not null, idempotency_key text not null,
 response_id uuid not null, request_fingerprint text not null, created_by_subject text not null, created_at timestamptz not null default now(),
 primary key(tenant_code,institution_id,operation,idempotency_key),
 foreign key(tenant_code,institution_id) references app_tenants(code,institution_id) on delete restrict
);
create table school_contracts (
 id uuid primary key default gen_random_uuid(), tenant_code text not null, institution_id text not null,
 supplier_party_id uuid not null, contract_number text not null, title text not null, category text not null default 'general',
 lifecycle_status text not null default 'draft' check(lifecycle_status in ('draft','verified','approved','signed','active','suspended','expired','terminated','archived')),
 starts_on date not null, ends_on date, currency text not null default 'RON' check(currency ~ '^[A-Z]{3}$'), total_value numeric(16,2) not null default 0 check(total_value>=0),
 expected_version integer not null default 1 check(expected_version>0), policy_evaluation_id uuid not null,
 source_registratura_document_id uuid, archive_status text not null default 'archive_pending' check(archive_status in ('archive_pending','queued','archived','failed')),
 archive_document_id uuid, archive_version_id uuid,
 created_by_subject text not null, created_at timestamptz not null default now(), updated_by_subject text not null, updated_at timestamptz not null default now(),
 unique(tenant_code,institution_id,id), unique(tenant_code,institution_id,contract_number),
 foreign key(tenant_code,institution_id) references app_tenants(code,institution_id) on delete restrict,
 foreign key(tenant_code,institution_id,supplier_party_id) references app_parties(tenant_code,institution_id,id) on delete restrict,
 foreign key(tenant_code,institution_id,policy_evaluation_id) references school_policy_evaluations(tenant_code,institution_id,id) on delete restrict,
 check(ends_on is null or ends_on >= starts_on)
);
create table school_contract_versions (
 id uuid primary key default gen_random_uuid(), tenant_code text not null, institution_id text not null, contract_id uuid not null, version integer not null,
 change_kind text not null check(change_kind in ('initial','amendment','lifecycle')), snapshot jsonb not null, created_by_subject text not null, created_at timestamptz not null default now(),
 unique(tenant_code,institution_id,contract_id,version), foreign key(tenant_code,institution_id,contract_id) references school_contracts(tenant_code,institution_id,id) on delete restrict
);
create table school_contract_obligations (
 id uuid primary key default gen_random_uuid(), tenant_code text not null, institution_id text not null, contract_id uuid not null, title text not null, due_on date, status text not null default 'open' check(status in ('open','fulfilled','waived','overdue')), sla_hours integer check(sla_hours is null or sla_hours>=0), guarantee_value numeric(16,2) check(guarantee_value is null or guarantee_value>=0), expected_version integer not null default 1 check(expected_version>0), policy_evaluation_id uuid not null, created_by_subject text not null, created_at timestamptz not null default now(), updated_at timestamptz not null default now(),
 unique(tenant_code,institution_id,id), foreign key(tenant_code,institution_id,contract_id) references school_contracts(tenant_code,institution_id,id) on delete restrict, foreign key(tenant_code,institution_id,policy_evaluation_id) references school_policy_evaluations(tenant_code,institution_id,id) on delete restrict
);
create table school_utility_points (
 id uuid primary key default gen_random_uuid(), tenant_code text not null, institution_id text not null, supplier_party_id uuid, utility_type text not null check(utility_type in ('electricity','gas','heating','water','waste','security','telecom')), name text not null, address text not null default '', active boolean not null default true, expected_version integer not null default 1, policy_evaluation_id uuid not null, created_by_subject text not null, created_at timestamptz not null default now(), updated_at timestamptz not null default now(), unique(tenant_code,institution_id,id), foreign key(tenant_code,institution_id) references app_tenants(code,institution_id) on delete restrict, foreign key(tenant_code,institution_id,supplier_party_id) references app_parties(tenant_code,institution_id,id) on delete restrict, foreign key(tenant_code,institution_id,policy_evaluation_id) references school_policy_evaluations(tenant_code,institution_id,id) on delete restrict
);
create table school_utility_meters (
 id uuid primary key default gen_random_uuid(), tenant_code text not null, institution_id text not null, point_id uuid not null, serial_number text not null, unit text not null, active boolean not null default true, expected_version integer not null default 1, created_by_subject text not null, created_at timestamptz not null default now(), unique(tenant_code,institution_id,id), unique(tenant_code,institution_id,serial_number), foreign key(tenant_code,institution_id,point_id) references school_utility_points(tenant_code,institution_id,id) on delete restrict
);
create table school_utility_readings (
 id uuid primary key default gen_random_uuid(), tenant_code text not null, institution_id text not null, meter_id uuid not null, reading_on date not null, value numeric(16,3) not null check(value>=0), source text not null default 'manual', expected_version integer not null default 1, policy_evaluation_id uuid not null, created_by_subject text not null, created_at timestamptz not null default now(), unique(tenant_code,institution_id,meter_id,reading_on), foreign key(tenant_code,institution_id,meter_id) references school_utility_meters(tenant_code,institution_id,id) on delete restrict, foreign key(tenant_code,institution_id,policy_evaluation_id) references school_policy_evaluations(tenant_code,institution_id,id) on delete restrict
);
create table school_utility_invoices (
 id uuid primary key default gen_random_uuid(), tenant_code text not null, institution_id text not null, point_id uuid not null, invoice_number text not null, invoice_date date not null, amount numeric(16,2) not null check(amount>=0), currency text not null default 'RON', reconciliation_status text not null default 'pending' check(reconciliation_status in ('pending','matched','disputed','paid')), expected_version integer not null default 1, policy_evaluation_id uuid not null, created_by_subject text not null, created_at timestamptz not null default now(), updated_at timestamptz not null default now(), unique(tenant_code,institution_id,invoice_number), foreign key(tenant_code,institution_id,point_id) references school_utility_points(tenant_code,institution_id,id) on delete restrict, foreign key(tenant_code,institution_id,policy_evaluation_id) references school_policy_evaluations(tenant_code,institution_id,id) on delete restrict
);
create table school_compliance_obligations (
 id uuid primary key default gen_random_uuid(), tenant_code text not null, institution_id text not null, domain text not null check(domain in ('ssm','psi','security')), code text not null, title text not null, due_on date, status text not null default 'open' check(status in ('open','compliant','overdue','waived')), expected_version integer not null default 1, policy_evaluation_id uuid not null, created_by_subject text not null, created_at timestamptz not null default now(), updated_at timestamptz not null default now(), unique(tenant_code,institution_id,code), foreign key(tenant_code,institution_id) references app_tenants(code,institution_id) on delete restrict, foreign key(tenant_code,institution_id,policy_evaluation_id) references school_policy_evaluations(tenant_code,institution_id,id) on delete restrict
);
-- Composite identity is required by the tenant-bound inspection foreign key.
-- The global primary key alone cannot back a three-column PostgreSQL FK.
alter table school_compliance_obligations
 add constraint school_compliance_obligations_scope_id_key unique(tenant_code,institution_id,id);

create table school_compliance_inspections (
 id uuid primary key default gen_random_uuid(), tenant_code text not null, institution_id text not null, obligation_id uuid not null, inspected_on date not null, outcome text not null check(outcome in ('compliant','non_compliant','recommendation')), notes text not null default '', expected_version integer not null default 1, policy_evaluation_id uuid not null, created_by_subject text not null, created_at timestamptz not null default now(), unique(tenant_code,institution_id,id), foreign key(tenant_code,institution_id,obligation_id) references school_compliance_obligations(tenant_code,institution_id,id) on delete restrict, foreign key(tenant_code,institution_id,policy_evaluation_id) references school_policy_evaluations(tenant_code,institution_id,id) on delete restrict
);
create table school_compliance_corrective_actions (
 id uuid primary key default gen_random_uuid(), tenant_code text not null, institution_id text not null, inspection_id uuid not null, title text not null, due_on date, status text not null default 'open' check(status in ('open','completed','overdue','cancelled')), expected_version integer not null default 1, policy_evaluation_id uuid not null, created_by_subject text not null, created_at timestamptz not null default now(), updated_at timestamptz not null default now(), unique(tenant_code,institution_id,id), foreign key(tenant_code,institution_id,inspection_id) references school_compliance_inspections(tenant_code,institution_id,id) on delete restrict, foreign key(tenant_code,institution_id,policy_evaluation_id) references school_policy_evaluations(tenant_code,institution_id,id) on delete restrict
);
create table school_operations_outbox (
 id uuid primary key default gen_random_uuid(), tenant_code text not null, institution_id text not null, aggregate_type text not null, aggregate_id uuid not null, event_type text not null, status text not null default 'pending' check(status in ('pending','delivered','failed')), payload jsonb not null, created_at timestamptz not null default now(), delivered_at timestamptz, unique(tenant_code,institution_id,id), foreign key(tenant_code,institution_id) references app_tenants(code,institution_id) on delete restrict
);
alter table school_operations_outbox
 add column attempt_count integer not null default 0 check(attempt_count >= 0),
 add column next_attempt_at timestamptz not null default now(),
 add column last_error text not null default '';
do $$ declare t text; begin foreach t in array array['school_operation_idempotency','school_contracts','school_contract_versions','school_contract_obligations','school_utility_points','school_utility_meters','school_utility_readings','school_utility_invoices','school_compliance_obligations','school_compliance_inspections','school_compliance_corrective_actions','school_operations_outbox'] loop execute format('alter table %I enable row level security',t); execute format('alter table %I force row level security',t); execute format('create policy tenant_isolation on %I using (public.can_bypass_tenant_rls() or (tenant_code=public.current_tenant_code() and institution_id=public.current_institution_id())) with check (public.can_bypass_tenant_rls() or (tenant_code=public.current_tenant_code() and institution_id=public.current_institution_id()))',t); end loop; end $$;
create or replace function public.school_operations_final_no_delete() returns trigger language plpgsql as $$ begin if old.lifecycle_status in ('signed','active','suspended','expired','terminated','archived') then raise exception 'legally relevant contract cannot be deleted'; end if; return old; end $$;
create trigger school_contracts_no_final_delete before delete on school_contracts for each row execute function public.school_operations_final_no_delete();
create or replace function public.school_operations_no_hard_delete() returns trigger language plpgsql as $$
begin
 raise exception 'school operations evidence cannot be hard deleted';
end
$$;
do $$
declare t text;
begin
 foreach t in array array[
  'school_operation_idempotency','school_contract_versions','school_contract_obligations',
  'school_utility_points','school_utility_meters','school_utility_readings','school_utility_invoices',
  'school_compliance_obligations','school_compliance_inspections','school_compliance_corrective_actions',
  'school_operations_outbox'
 ] loop
  execute format('create trigger %I_no_hard_delete before delete on %I for each row execute function public.school_operations_no_hard_delete()', t, t);
 end loop;
end
$$;
create or replace function public.school_contract_history_append() returns trigger language plpgsql as $$ begin insert into school_contract_versions(tenant_code,institution_id,contract_id,version,change_kind,snapshot,created_by_subject) values(new.tenant_code,new.institution_id,new.id,new.expected_version,case when tg_op='INSERT' then 'initial' when new.lifecycle_status is distinct from old.lifecycle_status then 'lifecycle' else 'amendment' end,to_jsonb(new),new.updated_by_subject); return new; end $$;
create trigger school_contract_history after insert or update on school_contracts for each row execute function public.school_contract_history_append();
