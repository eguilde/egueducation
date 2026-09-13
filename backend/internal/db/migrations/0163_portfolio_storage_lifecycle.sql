-- Durable, tenant-scoped bridge from portfolio lifecycle commands to exact
-- Object Lock versions. Human intent is committed with the portfolio change;
-- storage completion is reported only after an exact-version verification.
create table education_portfolio_lifecycle_operations (
 id uuid primary key default gen_random_uuid(),
 tenant_code text not null,
 institution_id text not null,
 portfolio_id uuid not null,
 operation_type text not null check(operation_type in ('cessation_retention','legal_hold_reconcile')),
 status text not null default 'pending' check(status in ('pending','processing','completed','blocked','dead_letter')),
 activity_ceased_on date,
 retention_through date,
 requested_legal_hold boolean,
 reason text not null,
 requested_by_subject text not null,
 requested_at timestamptz not null default now(),
 completed_at timestamptz,
 last_error text not null default '',
 foreign key(tenant_code,institution_id) references app_tenants(code,institution_id) on delete restrict,
 foreign key(institution_id,portfolio_id) references education_portfolios(institution_id,id) on delete restrict,
 check((operation_type='cessation_retention' and activity_ceased_on is not null and retention_through is not null and requested_legal_hold is null)
    or (operation_type='legal_hold_reconcile' and activity_ceased_on is null and retention_through is null and requested_legal_hold is not null))
);
create unique index education_portfolio_one_cessation_operation
 on education_portfolio_lifecycle_operations(institution_id,portfolio_id)
 where operation_type='cessation_retention';
create index education_portfolio_lifecycle_operations_lookup
 on education_portfolio_lifecycle_operations(institution_id,portfolio_id,requested_at desc);

create unique index if not exists archive_documents_institution_id_id_key on archive_documents(institution_id,id);
create unique index if not exists archive_document_versions_institution_id_id_key on archive_document_versions(institution_id,id);

create table education_portfolio_storage_transitions (
 id uuid primary key default gen_random_uuid(),
 operation_id uuid not null references education_portfolio_lifecycle_operations(id) on delete restrict,
 tenant_code text not null,
 institution_id text not null,
 portfolio_id uuid not null,
 archive_document_id uuid not null,
 archive_version_id uuid not null,
 source_bucket text not null,
 source_object_key text not null,
 source_object_version_id text not null,
 source_object_etag text not null,
 source_sha256 text not null,
 source_size_bytes bigint not null check(source_size_bytes>0),
 required_retention_until timestamptz,
 status text not null default 'pending' check(status in ('pending','processing','completed','blocked','dead_letter')),
 attempts integer not null default 0 check(attempts>=0),
 available_at timestamptz not null default now(),
 locked_at timestamptz,
 locked_by text not null default '',
 storage_verified_at timestamptz,
 observed_retention_until timestamptz,
 observed_hold_active boolean,
 observed_custody_required boolean,
 observed_legal_hold_required boolean,
 completed_at timestamptz,
 dead_lettered_at timestamptz,
 last_error text not null default '',
 created_at timestamptz not null default now(),
 foreign key(tenant_code,institution_id) references app_tenants(code,institution_id) on delete restrict,
 foreign key(institution_id,portfolio_id) references education_portfolios(institution_id,id) on delete restrict,
 foreign key(institution_id,archive_document_id) references archive_documents(institution_id,id) on delete restrict,
 foreign key(institution_id,archive_version_id) references archive_document_versions(institution_id,id) on delete restrict,
 unique(operation_id,archive_version_id),
 check(btrim(source_bucket)<>'' and btrim(source_object_key)<>'' and btrim(source_object_version_id)<>'' and btrim(source_object_etag)<>'' and source_sha256 ~ '^[0-9a-f]{64}$')
);
create index education_portfolio_storage_transition_queue
 on education_portfolio_storage_transitions(status,available_at,created_at);

create table education_portfolio_storage_transition_history (
 id uuid primary key default gen_random_uuid(),
 transition_id uuid not null references education_portfolio_storage_transitions(id) on delete restrict,
 tenant_code text not null,
 institution_id text not null,
 operation_id uuid not null references education_portfolio_lifecycle_operations(id) on delete restrict,
 previous_status text not null check(previous_status in ('blocked','dead_letter')),
 previous_attempts integer not null check(previous_attempts>=0),
 reason text not null check(btrim(reason)<>''),
 requested_by_subject text not null check(btrim(requested_by_subject)<>''),
 created_at timestamptz not null default now(),
 foreign key(tenant_code,institution_id) references app_tenants(code,institution_id) on delete restrict
);

alter table education_portfolio_lifecycle_operations enable row level security;
alter table education_portfolio_lifecycle_operations force row level security;
create policy education_portfolio_lifecycle_operations_scope on education_portfolio_lifecycle_operations
 using(public.can_bypass_tenant_rls() or (tenant_code=public.current_tenant_code() and institution_id=public.current_institution_id()))
 with check(public.can_bypass_tenant_rls() or (tenant_code=public.current_tenant_code() and institution_id=public.current_institution_id()));
alter table education_portfolio_storage_transitions enable row level security;
alter table education_portfolio_storage_transitions force row level security;
create policy education_portfolio_storage_transitions_scope on education_portfolio_storage_transitions
 using(public.can_bypass_tenant_rls() or (tenant_code=public.current_tenant_code() and institution_id=public.current_institution_id()))
 with check(public.can_bypass_tenant_rls() or (tenant_code=public.current_tenant_code() and institution_id=public.current_institution_id()));
alter table education_portfolio_storage_transition_history enable row level security;
alter table education_portfolio_storage_transition_history force row level security;
create policy education_portfolio_storage_transition_history_scope on education_portfolio_storage_transition_history
 using(public.can_bypass_tenant_rls() or (tenant_code=public.current_tenant_code() and institution_id=public.current_institution_id()))
 with check(public.can_bypass_tenant_rls() or (tenant_code=public.current_tenant_code() and institution_id=public.current_institution_id()));

-- Once cessation has been recorded, portfolio content is frozen. Legal-hold
-- intent remains an explicit allowed parent transition; storage reconciliation
-- changes archive-version projections, never the human lifecycle record.
create or replace function public.enforce_education_portfolio_lifecycle()
returns trigger language plpgsql as $$
begin
 if tg_op='DELETE' then raise exception 'education portfolios are evidentiary records and cannot be hard-deleted'; end if;
 if tg_op='UPDATE' then
  if old.withdrawn_at is not null then raise exception 'withdrawn education portfolios are immutable'; end if;
  if old.activity_ceased_on is not null then
   if new.activity_ceased_on is distinct from old.activity_ceased_on then raise exception 'portfolio cessation event is immutable'; end if;
   if (to_jsonb(new)-array['legal_hold_active','legal_hold_reason','legal_hold_set_at','legal_hold_set_by_subject','updated_at'])
      is distinct from (to_jsonb(old)-array['legal_hold_active','legal_hold_reason','legal_hold_set_at','legal_hold_set_by_subject','updated_at']) then
    raise exception 'ceased portfolio content is immutable';
   end if;
  end if;
  if new.withdrawn_at is not null and (old.status not in ('draft','returned') or old.retention_until is not null or old.legal_hold_active) then raise exception 'portfolio withdrawal is blocked after submission, during retention, or under legal hold'; end if;
 end if;
 if new.activity_ceased_on is null then
  if tg_op='INSERT' then new.retention_until:=null; elsif new.retention_until is distinct from old.retention_until then new.retention_until:=null; end if;
 else
  new.retention_until:=greatest((new.activity_ceased_on + interval '3 years')::date,coalesce(old.retention_until,new.activity_ceased_on));
 end if;
 return new;
end $$;

create or replace function public.reject_ceased_portfolio_child_mutation()
returns trigger language plpgsql as $$
declare portfolio_ref uuid; ceased date;
begin
 portfolio_ref:=case when tg_op='DELETE' then old.portfolio_id else new.portfolio_id end;
 select activity_ceased_on into ceased from education_portfolios where id=portfolio_ref for key share;
 if ceased is not null then raise exception 'ceased portfolio evidence and related records are immutable'; end if;
 return case when tg_op='DELETE' then old else new end;
end $$;
do $$ declare table_name text; begin
 foreach table_name in array array['education_portfolio_documents','education_portfolio_checklist','education_portfolio_opis','education_portfolio_custody'] loop
  execute format('drop trigger if exists trg_reject_ceased_portfolio_mutation on %I',table_name);
  execute format('create trigger trg_reject_ceased_portfolio_mutation before insert or update or delete on %I for each row execute function public.reject_ceased_portfolio_child_mutation()',table_name);
 end loop;
end $$;

-- A version whose operational custody hold was released cannot later be
-- attached to a new active portfolio without first going through a separate
-- verified re-hold workflow. Lock the exact version so this check serializes
-- with a concurrent lifecycle release.
create or replace function public.require_active_portfolio_custody_hold()
returns trigger language plpgsql security definer set search_path=pg_catalog,public as $$
declare ceased date; held boolean;
begin
 if new.archive_version_id is null then return new; end if;
 select p.activity_ceased_on into ceased from education_portfolios p
  where p.id=new.portfolio_id and p.institution_id=new.institution_id for key share;
 if ceased is not null then return new; end if;
 select v.custody_hold_active into held from archive_document_versions v
  where v.id=new.archive_version_id and v.institution_id=new.institution_id for key share;
 if not coalesce(held,false) then raise exception 'active portfolio archive attachment requires verified custody hold'; end if;
 return new;
end $$;
create trigger trg_require_active_portfolio_custody_hold
 before insert or update of portfolio_id,institution_id,archive_version_id on education_portfolio_documents
 for each row execute function public.require_active_portfolio_custody_hold();

-- Archive bitstream identity remains immutable. The only lifecycle exception
-- is an exact, storage-verified transition owned by the worker and its durable
-- per-version event. Retention is monotonic and cannot be removed.
create or replace function public.enforce_archive_version_immutability()
returns trigger language plpgsql security definer set search_path=pg_catalog,public as $$
declare step education_portfolio_storage_transitions%rowtype; operation_ref text;
begin
 if tg_op='DELETE' then raise exception 'archive document versions are immutable and cannot be deleted'; end if;
 if row(new.document_id,new.institution_id,new.version_no,new.mime_type,new.bucket_name,new.object_key,new.hash_sha256,new.size_bytes,new.source_bucket,new.source_object_key,new.source_sha256,new.source_size_bytes,new.source_object_version_id,new.source_object_etag,new.created_at,new.portfolio_custody_intent_id)
    is distinct from row(old.document_id,old.institution_id,old.version_no,old.mime_type,old.bucket_name,old.object_key,old.hash_sha256,old.size_bytes,old.source_bucket,old.source_object_key,old.source_sha256,old.source_size_bytes,old.source_object_version_id,old.source_object_etag,old.created_at,old.portfolio_custody_intent_id) then
  raise exception 'archive document version bitstream provenance is immutable';
 end if;
 if row(new.retention_until,new.custody_hold_active,new.legal_hold_active) is not distinct from row(old.retention_until,old.custody_hold_active,old.legal_hold_active) then return new; end if;
 if old.retention_until is not null and (new.retention_until is null or new.retention_until<old.retention_until) then raise exception 'archive retention cannot be removed or shortened'; end if;
 operation_ref:=nullif(current_setting('app.portfolio_storage_lifecycle_operation_id',true),'');
 if operation_ref is null or current_setting('app.actor_subject',true)<>'portfolio-storage-lifecycle-worker' then raise exception 'archive lifecycle projection requires verified worker transition'; end if;
 select s.* into step from education_portfolio_storage_transitions s
 where s.operation_id=operation_ref::uuid and s.archive_version_id=new.id and s.institution_id=new.institution_id
   and s.status='processing' and s.storage_verified_at is not null for key share;
 if not found
    or step.observed_custody_required is distinct from new.custody_hold_active
    or step.observed_legal_hold_required is distinct from new.legal_hold_active
    or step.observed_hold_active is distinct from (new.custody_hold_active or new.legal_hold_active)
    or (step.required_retention_until is not null and (step.observed_retention_until is null or step.observed_retention_until<step.required_retention_until or new.retention_until is null or new.retention_until<step.required_retention_until)) then
  raise exception 'archive lifecycle projection lacks exact storage verification';
 end if;
 return new;
end $$;

-- 0162 binds the immutable upload intent to its adopted version. The link and
-- exact storage identity remain immutable after cessation, while the verified
-- lifecycle event may replace operational custody with COMPLIANCE retention.
create or replace function public.archive_version_custody_intent_guard()
returns trigger language plpgsql security definer set search_path=pg_catalog,public as $$
declare i portfolio_custody_upload_intents%rowtype; step education_portfolio_storage_transitions%rowtype; operation_ref text;
begin
 if tg_op='UPDATE' and old.portfolio_custody_intent_id is not null and new.portfolio_custody_intent_id is distinct from old.portfolio_custody_intent_id then raise exception 'archive custody intent link is immutable'; end if;
 if new.custody_hold_active and new.portfolio_custody_intent_id is null then raise exception 'custody-held archive version requires verified custody intent'; end if;
 if new.portfolio_custody_intent_id is null then return new; end if;
 select * into i from portfolio_custody_upload_intents where id=new.portfolio_custody_intent_id;
 if not found or i.status not in ('stored','committed') or i.institution_id<>new.institution_id or i.reserved_document_id<>new.document_id or i.reserved_version_id<>new.id or i.bucket_name<>new.source_bucket or i.object_key<>new.source_object_key or i.stored_version_id<>new.source_object_version_id or i.stored_etag<>new.source_object_etag or lower(i.expected_sha256)<>lower(new.source_sha256) or i.expected_size_bytes<>new.source_size_bytes or i.expected_mime_type<>new.mime_type or not i.custody_hold_active then raise exception 'archive version must retain matching verified custody intent'; end if;
 if new.custody_hold_active then return new; end if;
 if tg_op<>'UPDATE' or new.retention_until is null then raise exception 'portfolio custody release requires verified retention transition'; end if;
 operation_ref:=nullif(current_setting('app.portfolio_storage_lifecycle_operation_id',true),'');
 if operation_ref is null or current_setting('app.actor_subject',true)<>'portfolio-storage-lifecycle-worker' then raise exception 'portfolio custody release requires lifecycle worker'; end if;
 select s.* into step from education_portfolio_storage_transitions s where s.operation_id=operation_ref::uuid and s.archive_version_id=new.id
   and s.institution_id=new.institution_id and s.status='processing' and s.storage_verified_at is not null and s.observed_custody_required=false for key share;
 if not found or step.observed_retention_until is null or step.observed_retention_until<new.retention_until then raise exception 'portfolio custody release lacks exact storage verification'; end if;
 return new;
end $$;

create or replace function public.portfolio_lifecycle_intent_guard()
returns trigger language plpgsql security definer set search_path=pg_catalog,public as $$
begin
 if tg_op='DELETE' then raise exception 'portfolio lifecycle operations cannot be deleted'; end if;
 if tg_op='UPDATE' and row(new.id,new.tenant_code,new.institution_id,new.portfolio_id,new.operation_type,new.activity_ceased_on,new.retention_through,new.requested_legal_hold,new.reason,new.requested_by_subject,new.requested_at)
   is distinct from row(old.id,old.tenant_code,old.institution_id,old.portfolio_id,old.operation_type,old.activity_ceased_on,old.retention_through,old.requested_legal_hold,old.reason,old.requested_by_subject,old.requested_at) then raise exception 'portfolio lifecycle human intent is immutable'; end if;
 return new;
end $$;
create trigger portfolio_lifecycle_intent_guard before update or delete on education_portfolio_lifecycle_operations for each row execute function public.portfolio_lifecycle_intent_guard();

create or replace function public.portfolio_storage_transition_guard()
returns trigger language plpgsql security definer set search_path=pg_catalog,public as $$
declare actor text:=nullif(current_setting('app.actor_subject',true),''); retry_exists boolean;
begin
 if tg_op='DELETE' then raise exception 'portfolio storage transitions cannot be deleted'; end if;
 if row(new.id,new.operation_id,new.tenant_code,new.institution_id,new.portfolio_id,new.archive_document_id,new.archive_version_id,
   new.source_bucket,new.source_object_key,new.source_object_version_id,new.source_object_etag,new.source_sha256,new.source_size_bytes,new.required_retention_until,new.created_at)
   is distinct from row(old.id,old.operation_id,old.tenant_code,old.institution_id,old.portfolio_id,old.archive_document_id,old.archive_version_id,
   old.source_bucket,old.source_object_key,old.source_object_version_id,old.source_object_etag,old.source_sha256,old.source_size_bytes,old.required_retention_until,old.created_at) then
  raise exception 'portfolio storage transition provenance is immutable';
 end if;
 if old.storage_verified_at is not null and row(new.storage_verified_at,new.observed_retention_until,new.observed_hold_active,new.observed_custody_required,new.observed_legal_hold_required)
   is distinct from row(old.storage_verified_at,old.observed_retention_until,old.observed_hold_active,old.observed_custody_required,old.observed_legal_hold_required) then
  raise exception 'portfolio storage verification evidence is immutable';
 end if;
 if actor='portfolio-storage-lifecycle-worker' then return new; end if;
 select exists(select 1 from education_portfolio_storage_transition_history h where h.transition_id=old.id and h.requested_by_subject=actor and h.created_at>=transaction_timestamp()) into retry_exists;
 if not retry_exists or old.status not in ('blocked','dead_letter') or new.status<>'pending'
    or new.attempts<>0 or new.locked_at is not null or new.locked_by<>'' or new.last_error<>'' or new.dead_lettered_at is not null
    or row(new.storage_verified_at,new.observed_retention_until,new.observed_hold_active,new.observed_custody_required,new.observed_legal_hold_required,new.completed_at)
       is distinct from row(old.storage_verified_at,old.observed_retention_until,old.observed_hold_active,old.observed_custody_required,old.observed_legal_hold_required,old.completed_at) then
  raise exception 'invalid portfolio storage transition mutation';
 end if;
 return new;
end $$;
create trigger portfolio_storage_transition_guard before update or delete on education_portfolio_storage_transitions for each row execute function public.portfolio_storage_transition_guard();

create or replace function public.portfolio_storage_transition_history_guard()
returns trigger language plpgsql security definer set search_path=pg_catalog,public as $$
begin
 if tg_op<>'INSERT' then raise exception 'portfolio storage transition history is append-only'; end if;
 if new.requested_by_subject is distinct from nullif(current_setting('app.actor_subject',true),'') then raise exception 'portfolio storage retry actor mismatch'; end if;
 return new;
end $$;
create trigger portfolio_storage_transition_history_guard before insert or update or delete on education_portfolio_storage_transition_history for each row execute function public.portfolio_storage_transition_history_guard();
