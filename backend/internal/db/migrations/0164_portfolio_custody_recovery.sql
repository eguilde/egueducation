-- Recovery of a held upload is a durable, tenant-scoped adoption workflow.
-- It never changes the custody object's retention or legal-hold state.
alter table portfolio_custody_upload_intents
 add column if not exists final_disposition text,
 add column if not exists recovery_committed_at timestamptz;

alter table portfolio_custody_upload_intents
 add constraint portfolio_custody_upload_intents_final_disposition_check
 check (final_disposition is null or final_disposition in ('teacher_access','institution_archive_only'));

-- Every committed intent created before this migration came exclusively from
-- the normal teacher upload path. Backfill that fact while the migration holds
-- the schema lock; the pre-0164 transition trigger intentionally rejects any
-- same-status application update.
alter table portfolio_custody_upload_intents disable trigger portfolio_custody_upload_intent_guard;
update portfolio_custody_upload_intents
set final_disposition='teacher_access', recovery_committed_at=coalesce(updated_at,created_at)
where status='committed' and final_disposition is null;
alter table portfolio_custody_upload_intents enable trigger portfolio_custody_upload_intent_guard;

create table portfolio_custody_recovery_operations (
 id uuid primary key default gen_random_uuid(),
 intent_id uuid not null references portfolio_custody_upload_intents(id) on delete restrict,
 tenant_code text not null, institution_id text not null, portfolio_id uuid not null,
 requested_by_subject text not null check(btrim(requested_by_subject)<>''),
 disposition text not null check(disposition in ('teacher_access','institution_archive_only')),
 reason text not null check(btrim(reason)<>''), title text not null check(btrim(title)<>''),
 original_file_name text not null check(btrim(original_file_name)<>''), document_date date,
 expected_fingerprint text not null check(expected_fingerprint ~ '^[0-9a-f]{64}$'),
 status text not null default 'queued' check(status in ('queued','leased','committed','blocked','deadletter')),
 attempts integer not null default 0 check(attempts>=0), lease_owner text not null default '', lease_expires_at timestamptz,
 last_error_code text not null default '', committed_at timestamptz, created_at timestamptz not null default now(), updated_at timestamptz not null default now(),
 foreign key(tenant_code,institution_id) references app_tenants(code,institution_id) on delete restrict,
 foreign key(institution_id,portfolio_id) references education_portfolios(institution_id,id) on delete restrict
);
create index portfolio_custody_recovery_operations_queue on portfolio_custody_recovery_operations(status,lease_expires_at,created_at);
create index portfolio_custody_recovery_operations_scope on portfolio_custody_recovery_operations(tenant_code,institution_id,created_at desc);
create unique index portfolio_custody_recovery_one_active_operation on portfolio_custody_recovery_operations(intent_id) where status in ('queued','leased');
alter table portfolio_custody_recovery_operations add constraint portfolio_custody_recovery_operation_state_check check (
 (status='queued' and lease_owner='' and lease_expires_at is null and committed_at is null) or
 (status='leased' and btrim(lease_owner)<>'' and lease_expires_at is not null and attempts>0 and committed_at is null) or
 (status='committed' and lease_owner='' and lease_expires_at is null and committed_at is not null and last_error_code='') or
 (status in ('blocked','deadletter') and lease_owner='' and lease_expires_at is null and committed_at is null and btrim(last_error_code)<>'')
);

create table portfolio_custody_recovery_attempts (
 id uuid primary key default gen_random_uuid(), operation_id uuid not null references portfolio_custody_recovery_operations(id) on delete restrict,
 attempt_no integer not null check(attempt_no>0), worker_id text not null check(btrim(worker_id)<>''), outcome text not null check(outcome in ('claimed','committed','blocked','retry','deadletter')), error_code text not null default '', created_at timestamptz not null default now(), unique(operation_id,attempt_no,outcome)
);
create table portfolio_custody_recovery_deadletters (
 operation_id uuid primary key references portfolio_custody_recovery_operations(id) on delete restrict,
 tenant_code text not null, institution_id text not null, error_code text not null, created_at timestamptz not null default now(),
 foreign key(tenant_code,institution_id) references app_tenants(code,institution_id) on delete restrict
);

alter table portfolio_custody_recovery_operations enable row level security;
alter table portfolio_custody_recovery_operations force row level security;
create policy portfolio_custody_recovery_operations_scope on portfolio_custody_recovery_operations using(public.can_bypass_tenant_rls() or (tenant_code=public.current_tenant_code() and institution_id=public.current_institution_id())) with check(public.can_bypass_tenant_rls() or (tenant_code=public.current_tenant_code() and institution_id=public.current_institution_id()));
alter table portfolio_custody_recovery_attempts enable row level security;
alter table portfolio_custody_recovery_attempts force row level security;
create policy portfolio_custody_recovery_attempts_scope on portfolio_custody_recovery_attempts using(public.can_bypass_tenant_rls() or exists(select 1 from portfolio_custody_recovery_operations o where o.id=operation_id and o.tenant_code=public.current_tenant_code() and o.institution_id=public.current_institution_id())) with check(public.can_bypass_tenant_rls() or exists(select 1 from portfolio_custody_recovery_operations o where o.id=operation_id and o.tenant_code=public.current_tenant_code() and o.institution_id=public.current_institution_id()));
alter table portfolio_custody_recovery_deadletters enable row level security;
alter table portfolio_custody_recovery_deadletters force row level security;
create policy portfolio_custody_recovery_deadletters_scope on portfolio_custody_recovery_deadletters using(public.can_bypass_tenant_rls() or (tenant_code=public.current_tenant_code() and institution_id=public.current_institution_id()));

-- Recovery adoption is a dual-authority operation. The database repeats the
-- authorization check so a raw SQL caller cannot rely on stale token claims.
create or replace function public.portfolio_custody_recovery_actor_has_permission(requested_permission text)
returns boolean language sql stable security definer set search_path=pg_catalog,public as $$
 select nullif(public.current_tenant_code(),'') is not null
  and nullif(public.current_institution_id(),'') is not null
  and nullif(btrim(current_setting('app.actor_subject',true)),'') is not null
  and exists(select 1 from (
   select p.permission_code from app_users u join app_user_permissions p on p.user_id=u.id and p.tenant_code=public.current_tenant_code()
    where lower(u.sub)=lower(nullif(btrim(current_setting('app.actor_subject',true)),'')) and u.status='active' and p.permission_code=requested_permission
      and exists(select 1 from app_memberships m where m.user_id=u.id and m.tenant_code=public.current_tenant_code() and public.education_membership_is_eligible(u.id,public.current_tenant_code(),public.current_institution_id(),m.position_code))
   union
   select rp.permission_code from app_users u join app_user_roles ur on ur.user_id=u.id and ur.tenant_code=public.current_tenant_code() join app_role_permissions rp on rp.role_code=ur.role_code
    where lower(u.sub)=lower(nullif(btrim(current_setting('app.actor_subject',true)),'')) and u.status='active' and rp.permission_code=requested_permission
      and exists(select 1 from app_memberships m where m.user_id=u.id and m.tenant_code=public.current_tenant_code() and public.education_membership_is_eligible(u.id,public.current_tenant_code(),public.current_institution_id(),m.position_code))
   union
   select p.permission_code from app_users u join app_memberships m on m.user_id=u.id and m.tenant_code=public.current_tenant_code() join app_position_permissions p on p.position_code=m.position_code
    where lower(u.sub)=lower(nullif(btrim(current_setting('app.actor_subject',true)),'')) and u.status='active' and p.permission_code=requested_permission and public.education_membership_is_eligible(u.id,public.current_tenant_code(),public.current_institution_id(),m.position_code)
   union
   select rp.permission_code from app_users u join app_memberships m on m.user_id=u.id and m.tenant_code=public.current_tenant_code() join app_position_roles pr on pr.position_code=m.position_code join app_role_permissions rp on rp.role_code=pr.role_code
    where lower(u.sub)=lower(nullif(btrim(current_setting('app.actor_subject',true)),'')) and u.status='active' and rp.permission_code=requested_permission and public.education_membership_is_eligible(u.id,public.current_tenant_code(),public.current_institution_id(),m.position_code)
  ) effective_permissions);
$$;

create or replace function public.portfolio_custody_recovery_guard() returns trigger language plpgsql security definer set search_path=pg_catalog,public as $$
declare actor_subject text := nullif(btrim(current_setting('app.actor_subject',true)), ''); bound_worker_id text:=nullif(btrim(current_setting('app.portfolio_custody_recovery_worker_id',true)),'');
begin
 if tg_op='DELETE' then raise exception 'portfolio custody recovery operations cannot be deleted'; end if;
 if tg_op='INSERT' then
  if public.can_bypass_tenant_rls() then return new; end if;
  if actor_subject is null or new.tenant_code<>public.current_tenant_code() or new.institution_id<>public.current_institution_id() or lower(new.requested_by_subject)<>lower(actor_subject) then raise exception 'portfolio custody recovery requires bound current actor and tenant scope'; end if;
  if not public.portfolio_custody_recovery_actor_has_permission('earchiva.manage') or not public.portfolio_custody_recovery_actor_has_permission('education.portfolios.custody.manage') then raise exception 'portfolio custody recovery requires current dual authority'; end if;
  if not exists(select 1 from portfolio_custody_upload_intents i where i.id=new.intent_id and i.tenant_code=new.tenant_code and i.institution_id=new.institution_id and i.portfolio_id=new.portfolio_id and i.status='stored' and i.final_disposition is null and i.expected_request_fingerprint=new.expected_fingerprint) then raise exception 'portfolio custody recovery provenance does not match stored intent'; end if;
 return new;
 end if;
 if tg_op='UPDATE' and row(new.intent_id,new.tenant_code,new.institution_id,new.portfolio_id,new.requested_by_subject,new.disposition,new.reason,new.title,new.original_file_name,new.document_date,new.expected_fingerprint,new.created_at) is distinct from row(old.intent_id,old.tenant_code,old.institution_id,old.portfolio_id,old.requested_by_subject,old.disposition,old.reason,old.title,old.original_file_name,old.document_date,old.expected_fingerprint,old.created_at) then raise exception 'portfolio custody recovery provenance is immutable'; end if;
 if tg_op='UPDATE' and old.status in ('committed','blocked','deadletter') then raise exception 'terminal portfolio custody recovery is immutable'; end if;
 if actor_subject<>'portfolio-custody-recovery-worker' or bound_worker_id is null then raise exception 'portfolio custody recovery transition requires bound worker'; end if;
 if tg_op='UPDATE' and not (
  (old.status='queued' and new.status='leased' and new.lease_owner=bound_worker_id) or
  (old.status='leased' and old.lease_owner=bound_worker_id and new.status in ('queued','committed','blocked','deadletter')) or
  (old.status='leased' and old.lease_expires_at<now() and new.status='queued')
 ) then raise exception 'invalid portfolio custody recovery state transition'; end if;
 if old.status='leased' and new.status='committed' and not exists(
  select 1 from portfolio_custody_recovery_attempts a join portfolio_custody_upload_intents i on i.id=old.intent_id
  where a.operation_id=old.id and a.attempt_no=old.attempts and a.worker_id=bound_worker_id and a.outcome='committed'
   and i.status='committed' and i.final_disposition=old.disposition and i.recovery_committed_at is not null
 ) then raise exception 'portfolio custody recovery commit lacks exact worker evidence'; end if;
 new.updated_at:=now(); return new;
end $$;
create trigger portfolio_custody_recovery_guard before insert or update or delete on portfolio_custody_recovery_operations for each row execute function public.portfolio_custody_recovery_guard();

create or replace function public.portfolio_custody_recovery_event_immutable() returns trigger language plpgsql security definer set search_path=pg_catalog,public as $$
begin
 if tg_op='INSERT'
  and current_setting('app.actor_subject',true)='portfolio-custody-recovery-worker'
  and nullif(btrim(current_setting('app.portfolio_custody_recovery_worker_id',true)),'') is not null
  and new.worker_id=current_setting('app.portfolio_custody_recovery_worker_id',true)
  and exists(select 1 from portfolio_custody_recovery_operations o where o.id=new.operation_id and o.attempts=new.attempt_no and o.status='leased' and o.lease_owner=new.worker_id)
  then return new;
 end if;
 raise exception 'portfolio custody recovery evidence is append-only';
end $$;
create trigger portfolio_custody_recovery_attempts_immutable before insert or update or delete on portfolio_custody_recovery_attempts for each row execute function public.portfolio_custody_recovery_event_immutable();

create or replace function public.portfolio_custody_recovery_deadletter_immutable() returns trigger language plpgsql security definer set search_path=pg_catalog,public as $$
begin
 if tg_op='INSERT' and current_setting('app.actor_subject',true)='portfolio-custody-recovery-worker'
  and exists(select 1 from portfolio_custody_recovery_operations o where o.id=new.operation_id and o.status='deadletter' and o.tenant_code=new.tenant_code and o.institution_id=new.institution_id)
  then return new;
 end if;
 raise exception 'portfolio custody recovery evidence is append-only';
end $$;
create trigger portfolio_custody_recovery_deadletters_immutable before insert or update or delete on portfolio_custody_recovery_deadletters for each row execute function public.portfolio_custody_recovery_deadletter_immutable();

-- An adopted stored intent has one irreversible disposition. Existing direct
-- upload commit remains valid with teacher_access as its normal disposition.
create or replace function public.portfolio_custody_upload_intent_recovery_guard() returns trigger language plpgsql security definer set search_path=pg_catalog,public as $$
declare actor text:=nullif(btrim(current_setting('app.actor_subject',true)),''); operation_ref text:=nullif(btrim(current_setting('app.portfolio_custody_recovery_operation_id',true)),''); bound_worker_id text:=nullif(btrim(current_setting('app.portfolio_custody_recovery_worker_id',true)),'');
begin
 if tg_op='UPDATE' and old.final_disposition is not null and row(new.final_disposition,new.recovery_committed_at) is distinct from row(old.final_disposition,old.recovery_committed_at) then raise exception 'portfolio custody final disposition is immutable'; end if;
 if new.status='committed' and (new.final_disposition is null or new.recovery_committed_at is null) then raise exception 'committed portfolio custody intent requires final disposition evidence'; end if;
 if new.status<>'committed' and (new.final_disposition is not null or new.recovery_committed_at is not null) then raise exception 'uncommitted portfolio custody intent cannot have final disposition'; end if;
 if tg_op='UPDATE' and old.status='stored' and new.status='committed' then
  if operation_ref is null then
   if new.final_disposition<>'teacher_access' or not (old.actor_subject=actor or lower(old.actor_subject)=lower(actor)) then raise exception 'direct portfolio custody commit requires bound owner'; end if;
  elsif actor<>'portfolio-custody-recovery-worker' or bound_worker_id is null or not exists(
   select 1 from portfolio_custody_recovery_operations o
   join portfolio_custody_recovery_attempts a on a.operation_id=o.id and a.attempt_no=o.attempts and a.worker_id=bound_worker_id and a.outcome='committed'
   join archive_documents d on d.id=old.reserved_document_id and d.institution_id=old.institution_id
   join archive_document_versions v on v.id=old.reserved_version_id and v.document_id=d.id and v.institution_id=d.institution_id
   where o.id=operation_ref::uuid and o.intent_id=old.id and o.status='leased' and o.lease_owner=bound_worker_id and o.disposition=new.final_disposition
    and v.portfolio_custody_intent_id=old.id and v.source_bucket=old.bucket_name and v.source_object_key=old.object_key
    and v.source_object_version_id=old.stored_version_id and v.source_object_etag=old.stored_etag
    and lower(v.source_sha256)=lower(old.expected_sha256) and v.source_size_bytes=old.expected_size_bytes and v.mime_type=old.expected_mime_type
  ) then raise exception 'recovered portfolio custody commit lacks exact worker adoption evidence'; end if;
 end if;
 return new;
end $$;
create trigger portfolio_custody_upload_intent_recovery_guard before update on portfolio_custody_upload_intents for each row execute function public.portfolio_custody_upload_intent_recovery_guard();
