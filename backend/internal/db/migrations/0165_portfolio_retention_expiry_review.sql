-- Retention is a calendar rule, not a monotonic projection of legacy values.
-- 0101's historical deadline remains evidence only and must not be promoted
-- into future lifecycle derivation.
create or replace function public.enforce_education_portfolio_lifecycle()
returns trigger language plpgsql as $$
begin
 if tg_op='DELETE' then raise exception 'education portfolios are evidentiary records and cannot be hard-deleted'; end if;
 if new.activity_ceased_on is not null and new.activity_ceased_on>current_date then
  raise exception 'portfolio cessation cannot be in the future';
 end if;
 if tg_op='UPDATE' then
  if old.withdrawn_at is not null then raise exception 'withdrawn education portfolios are immutable'; end if;
  if old.activity_ceased_on is not null then
   if new.activity_ceased_on is distinct from old.activity_ceased_on then raise exception 'portfolio cessation event is immutable'; end if;
   if (to_jsonb(new)-array['legal_hold_active','legal_hold_reason','legal_hold_set_at','legal_hold_set_by_subject','retention_until','updated_at'])
      is distinct from (to_jsonb(old)-array['legal_hold_active','legal_hold_reason','legal_hold_set_at','legal_hold_set_by_subject','retention_until','updated_at']) then raise exception 'ceased portfolio content is immutable'; end if;
  end if;
  if new.withdrawn_at is not null and (old.status not in ('draft','returned') or old.retention_until is not null or old.legal_hold_active) then raise exception 'portfolio withdrawal is blocked after submission, during retention, or under legal hold'; end if;
 end if;
 if new.activity_ceased_on is null then
  if tg_op='INSERT' then new.retention_until:=null; elsif new.retention_until is distinct from old.retention_until then new.retention_until:=null; end if;
 else
  new.retention_until:=(new.activity_ceased_on + interval '3 years')::date;
 end if;
 return new;
end $$;

alter table education_portfolio_storage_transitions
 add column if not exists protective_extension_provenance jsonb not null default '{}'::jsonb,
 add column if not exists observed_retention_disposition_hold_required boolean;

alter table archive_document_versions
 add column if not exists retention_disposition_hold_active boolean not null default false;

alter table education_portfolio_storage_transitions
 add constraint education_portfolio_storage_transition_extension_provenance_object
 check (jsonb_typeof(protective_extension_provenance)='object');

-- A normal retry must never turn an expired-retention review gate back into
-- executable work. A dedicated immutable disposition is required instead.
create table portfolio_retention_disposition_requests (
 id uuid primary key default gen_random_uuid(),
 transition_id uuid not null references education_portfolio_storage_transitions(id) on delete restrict,
 tenant_code text not null, institution_id text not null, portfolio_id uuid not null,
 archive_document_id uuid not null, archive_version_id uuid not null,
 source_bucket text not null, source_object_key text not null, source_object_version_id text not null, source_object_etag text not null,
 source_sha256 text not null check(source_sha256 ~ '^[0-9a-f]{64}$'), source_size_bytes bigint not null check(source_size_bytes>0), source_mime_type text not null check(btrim(source_mime_type)<>''),
 required_retention_until timestamptz not null,
 evidence jsonb not null check(
  jsonb_typeof(evidence)='object'
  and evidence ? 'statement'
  and jsonb_typeof(evidence->'statement')='string'
  and length(btrim(evidence->>'statement')) between 1 and 4000
  and (not (evidence ? 'reference') or jsonb_typeof(evidence->'reference')='string')
  and length(coalesce(evidence->>'reference',''))<=500
  and evidence-array['statement','reference']='{}'::jsonb
  and octet_length(evidence::text)<=8192
 ),
 requested_by_subject text not null check(btrim(requested_by_subject)<>''),
 requested_by_user_id uuid not null references app_users(id) on delete restrict,
 requested_at timestamptz not null default now(),
 status text not null default 'submitted' check(status in ('submitted','approved','rejected','blocked','closed')),
 foreign key(tenant_code,institution_id) references app_tenants(code,institution_id) on delete restrict,
 foreign key(institution_id,portfolio_id) references education_portfolios(institution_id,id) on delete restrict
);
create unique index portfolio_retention_disposition_one_active_request
 on portfolio_retention_disposition_requests(transition_id)
 where status in ('submitted','approved');
create table portfolio_retention_disposition_decisions (
 id uuid primary key default gen_random_uuid(), request_id uuid not null references portfolio_retention_disposition_requests(id) on delete restrict,
 decision text not null check(decision in ('approved','rejected')), reason text not null check(btrim(reason)<>''),
 decided_by_subject text not null check(btrim(decided_by_subject)<>''),
 decided_by_user_id uuid not null references app_users(id) on delete restrict,
 decided_at timestamptz not null default now(),
 unique(request_id)
);
create table portfolio_retention_disposition_receipts (
 id uuid primary key default gen_random_uuid(), request_id uuid not null references portfolio_retention_disposition_requests(id) on delete restrict,
 transition_id uuid not null references education_portfolio_storage_transitions(id) on delete restrict,
 observed_retention_until timestamptz, observed_hold_active boolean, closed_by_subject text not null check(btrim(closed_by_subject)<>''),
 closed_at timestamptz not null default now(), outcome text not null check(outcome in ('released','retained','blocked')),
 unique(request_id)
);
create table portfolio_retention_disposition_operations (
 id uuid primary key default gen_random_uuid(), request_id uuid not null unique references portfolio_retention_disposition_requests(id) on delete restrict,
 tenant_code text not null, institution_id text not null, status text not null default 'queued' check(status in ('queued','leased','released','blocked','deadletter')),
 attempts integer not null default 0 check(attempts>=0), lease_owner text not null default '', lease_expires_at timestamptz, available_at timestamptz not null default now(), last_error_code text not null default '', created_at timestamptz not null default now(), completed_at timestamptz,
 foreign key(tenant_code,institution_id) references app_tenants(code,institution_id) on delete restrict
);
create index portfolio_retention_disposition_operations_queue on portfolio_retention_disposition_operations(status,available_at,created_at);
alter table portfolio_retention_disposition_operations add constraint portfolio_retention_disposition_operation_state check (
 (status='queued' and lease_owner='' and lease_expires_at is null and completed_at is null) or
 (status='leased' and btrim(lease_owner)<>'' and lease_expires_at is not null and attempts>0 and completed_at is null) or
 (status='released' and lease_owner='' and lease_expires_at is null and completed_at is not null and last_error_code='') or
 (status in ('blocked','deadletter') and lease_owner='' and lease_expires_at is null and completed_at is null and btrim(last_error_code)<>'')
);
create table portfolio_retention_disposition_attempts (
 id uuid primary key default gen_random_uuid(), operation_id uuid not null references portfolio_retention_disposition_operations(id) on delete restrict,
 attempt_no integer not null, worker_id text not null, outcome text not null check(outcome in ('claimed','released','blocked','retry','deadletter')), error_code text not null default '', created_at timestamptz not null default now(), unique(operation_id,attempt_no,outcome)
);
create table education_portfolio_retention_expiry_events (
 transition_id uuid primary key references education_portfolio_storage_transitions(id) on delete restrict,
 tenant_code text not null, institution_id text not null, required_retention_until timestamptz not null, promoted_at timestamptz not null default now(), consumed_at timestamptz, promoted_by_subject text not null check(btrim(promoted_by_subject)<>''),
 foreign key(tenant_code,institution_id) references app_tenants(code,institution_id) on delete restrict
);

alter table portfolio_retention_disposition_requests enable row level security;
alter table portfolio_retention_disposition_requests force row level security;
create policy portfolio_retention_disposition_requests_scope on portfolio_retention_disposition_requests
 using(public.can_bypass_tenant_rls() or (tenant_code=public.current_tenant_code() and institution_id=public.current_institution_id()))
 with check(public.can_bypass_tenant_rls() or (tenant_code=public.current_tenant_code() and institution_id=public.current_institution_id()));
alter table portfolio_retention_disposition_decisions enable row level security;
alter table portfolio_retention_disposition_decisions force row level security;
create policy portfolio_retention_disposition_decisions_scope on portfolio_retention_disposition_decisions using(public.can_bypass_tenant_rls() or exists(select 1 from portfolio_retention_disposition_requests r where r.id=request_id and r.tenant_code=public.current_tenant_code() and r.institution_id=public.current_institution_id())) with check(public.can_bypass_tenant_rls() or exists(select 1 from portfolio_retention_disposition_requests r where r.id=request_id and r.tenant_code=public.current_tenant_code() and r.institution_id=public.current_institution_id()));
alter table portfolio_retention_disposition_receipts enable row level security;
alter table portfolio_retention_disposition_receipts force row level security;
create policy portfolio_retention_disposition_receipts_scope on portfolio_retention_disposition_receipts using(public.can_bypass_tenant_rls() or exists(select 1 from portfolio_retention_disposition_requests r where r.id=request_id and r.tenant_code=public.current_tenant_code() and r.institution_id=public.current_institution_id())) with check(public.can_bypass_tenant_rls() or exists(select 1 from portfolio_retention_disposition_requests r where r.id=request_id and r.tenant_code=public.current_tenant_code() and r.institution_id=public.current_institution_id()));
alter table portfolio_retention_disposition_operations enable row level security;
alter table portfolio_retention_disposition_operations force row level security;
create policy portfolio_retention_disposition_operations_scope on portfolio_retention_disposition_operations using(public.can_bypass_tenant_rls() or (tenant_code=public.current_tenant_code() and institution_id=public.current_institution_id())) with check(public.can_bypass_tenant_rls() or (tenant_code=public.current_tenant_code() and institution_id=public.current_institution_id()));
alter table portfolio_retention_disposition_attempts enable row level security;
alter table portfolio_retention_disposition_attempts force row level security;
create policy portfolio_retention_disposition_attempts_scope on portfolio_retention_disposition_attempts using(public.can_bypass_tenant_rls() or exists(select 1 from portfolio_retention_disposition_operations o where o.id=operation_id and o.tenant_code=public.current_tenant_code() and o.institution_id=public.current_institution_id())) with check(public.can_bypass_tenant_rls() or exists(select 1 from portfolio_retention_disposition_operations o where o.id=operation_id and o.tenant_code=public.current_tenant_code() and o.institution_id=public.current_institution_id()));
alter table education_portfolio_retention_expiry_events enable row level security;
alter table education_portfolio_retention_expiry_events force row level security;
create policy education_portfolio_retention_expiry_events_scope on education_portfolio_retention_expiry_events using(public.can_bypass_tenant_rls() or (tenant_code=public.current_tenant_code() and institution_id=public.current_institution_id())) with check(public.can_bypass_tenant_rls() or (tenant_code=public.current_tenant_code() and institution_id=public.current_institution_id()));

-- Trigger-level authorization is intentionally independent of HTTP middleware.
-- It resolves the current actor from the session and recomputes effective
-- tenant/institution permissions from direct, role, position, and
-- position-role grants using the temporal membership contract.
create or replace function public.portfolio_retention_actor_has_permission(p_permission text)
returns boolean language sql stable security definer set search_path=pg_catalog,public as $$
 with actor as (
  select u.id from public.app_users u
  where u.status='active' and (u.id::text=nullif(btrim(current_setting('app.actor_subject',true)),'') or lower(u.sub)=lower(nullif(btrim(current_setting('app.actor_subject',true)),'')))
  limit 1
 ), scope as (
  select public.current_tenant_code() tenant_code,public.current_institution_id() institution_id
 )
 select exists(
  select 1 from actor,scope
  where scope.tenant_code is not null and scope.institution_id is not null
   and public.education_membership_is_eligible(actor.id,scope.tenant_code,scope.institution_id,null)
   and exists(
    select 1 from public.app_user_permissions up where up.user_id=actor.id and up.tenant_code=scope.tenant_code and up.permission_code=p_permission
    union all select 1 from public.app_user_roles ur join public.app_role_permissions rp on rp.role_code=ur.role_code where ur.user_id=actor.id and ur.tenant_code=scope.tenant_code and rp.permission_code=p_permission
    union all select 1 from public.app_memberships m join public.app_position_permissions pp on pp.position_code=m.position_code where m.user_id=actor.id and m.tenant_code=scope.tenant_code and pp.permission_code=p_permission and public.education_membership_is_eligible(actor.id,scope.tenant_code,scope.institution_id,m.position_code)
    union all select 1 from public.app_memberships m join public.app_position_roles pr on pr.position_code=m.position_code join public.app_role_permissions rp on rp.role_code=pr.role_code where m.user_id=actor.id and m.tenant_code=scope.tenant_code and rp.permission_code=p_permission and public.education_membership_is_eligible(actor.id,scope.tenant_code,scope.institution_id,m.position_code)
   )
 );
$$;
revoke all on function public.portfolio_retention_actor_has_permission(text) from public;

create or replace function public.portfolio_retention_actor_user_id()
returns uuid language sql stable security definer set search_path=pg_catalog,public as $$
 select u.id from public.app_users u
 where u.status='active'
  and (u.id::text=nullif(btrim(current_setting('app.actor_subject',true)),'') or lower(u.sub)=lower(nullif(btrim(current_setting('app.actor_subject',true)),'')))
 limit 1;
$$;
revoke all on function public.portfolio_retention_actor_user_id() from public;

create or replace function public.portfolio_retention_actor_owns_portfolio(p_portfolio_id uuid)
returns boolean language sql stable security definer set search_path=pg_catalog,public as $$
 select exists(
  select 1 from public.education_portfolios p
  join public.app_users u on u.id=p.owner_user_id and u.status='active'
  join public.education_personnel person on person.id=p.owner_personnel_id and person.app_user_id=u.id and person.institution_id=p.institution_id
  where p.id=p_portfolio_id and p.institution_id=public.current_institution_id()
   and (u.id::text=nullif(btrim(current_setting('app.actor_subject',true)),'') or lower(u.sub)=lower(nullif(btrim(current_setting('app.actor_subject',true)),'')))
   and public.education_membership_is_eligible(u.id,public.current_tenant_code(),public.current_institution_id(),null)
 );
$$;
revoke all on function public.portfolio_retention_actor_owns_portfolio(uuid) from public;

create or replace function public.portfolio_retention_disposition_guard()
returns trigger language plpgsql security definer set search_path=pg_catalog,public as $$
declare actor text:=nullif(current_setting('app.actor_subject',true),''); actor_user uuid:=public.portfolio_retention_actor_user_id();
begin
 if tg_op='DELETE' then raise exception 'portfolio retention disposition records cannot be deleted'; end if;
 if tg_op='INSERT' then
  if actor_user is null or (new.requested_by_user_id is not null and new.requested_by_user_id is distinct from actor_user)
   or new.status<>'submitted' or new.requested_by_subject is distinct from actor
   or not public.portfolio_retention_actor_has_permission('education.portfolios.manage_own')
   or not public.portfolio_retention_actor_owns_portfolio(new.portfolio_id)
   or not exists(
   select 1 from education_portfolio_storage_transitions t join archive_document_versions v on v.id=t.archive_version_id and v.institution_id=t.institution_id
    join education_portfolio_retention_expiry_events expiry on expiry.transition_id=t.id and expiry.consumed_at is null
   where t.id=new.transition_id and t.status='blocked' and t.last_error='portfolio_retention_expired_review_required'
     and t.tenant_code=new.tenant_code and t.institution_id=new.institution_id and t.portfolio_id=new.portfolio_id and t.archive_document_id=new.archive_document_id and t.archive_version_id=new.archive_version_id
     and t.source_bucket=new.source_bucket and t.source_object_key=new.source_object_key and t.source_object_version_id=new.source_object_version_id and t.source_object_etag=new.source_object_etag
     and t.source_sha256=new.source_sha256 and t.source_size_bytes=new.source_size_bytes and v.mime_type=new.source_mime_type and t.required_retention_until=new.required_retention_until
  ) or (select count(*) from portfolio_retention_disposition_requests prior where prior.transition_id=new.transition_id)>=20 then raise exception 'portfolio retention disposition request lacks exact expired transition provenance or exceeds review history limit'; end if;
  new.requested_by_user_id:=actor_user;
  return new;
 end if;
 if row(new.transition_id,new.tenant_code,new.institution_id,new.portfolio_id,new.archive_document_id,new.archive_version_id,new.source_bucket,new.source_object_key,new.source_object_version_id,new.source_object_etag,new.source_sha256,new.source_size_bytes,new.source_mime_type,new.required_retention_until,new.evidence,new.requested_by_subject,new.requested_by_user_id,new.requested_at) is distinct from row(old.transition_id,old.tenant_code,old.institution_id,old.portfolio_id,old.archive_document_id,old.archive_version_id,old.source_bucket,old.source_object_key,old.source_object_version_id,old.source_object_etag,old.source_sha256,old.source_size_bytes,old.source_mime_type,old.required_retention_until,old.evidence,old.requested_by_subject,old.requested_by_user_id,old.requested_at) then raise exception 'portfolio retention disposition provenance is immutable'; end if;
 if old.status='submitted' and new.status in ('approved','rejected') then
  if not exists(select 1 from portfolio_retention_disposition_decisions d where d.request_id=old.id and d.decision=new.status and d.decided_by_user_id=actor_user) then raise exception 'retention disposition transition requires actor-bound independent decision'; end if;
  return new;
 end if;
 if old.status='approved' and new.status='blocked' and actor='portfolio-retention-disposition-worker' and exists(select 1 from portfolio_retention_disposition_receipts receipt where receipt.request_id=old.id and receipt.outcome='blocked') then return new; end if;
 if old.status='approved' and new.status='closed' and actor='portfolio-retention-disposition-worker' and exists(select 1 from portfolio_retention_disposition_receipts receipt where receipt.request_id=old.id and receipt.outcome='released' and receipt.observed_hold_active=false) then return new; end if;
 raise exception 'invalid portfolio retention disposition transition';
end $$;
create trigger portfolio_retention_disposition_request_guard before insert or update or delete on portfolio_retention_disposition_requests for each row execute function public.portfolio_retention_disposition_guard();
create or replace function public.portfolio_retention_disposition_decision_guard()
returns trigger language plpgsql security definer set search_path=pg_catalog,public as $$
declare requester uuid; actor text:=nullif(current_setting('app.actor_subject',true),''); actor_user uuid:=public.portfolio_retention_actor_user_id();
begin
 if tg_op<>'INSERT' then raise exception 'portfolio retention disposition decisions are append-only'; end if;
 select requested_by_user_id into requester from portfolio_retention_disposition_requests where id=new.request_id for key share;
 if actor_user is null or requester is null or requester=actor_user or new.decided_by_subject is distinct from actor or (new.decided_by_user_id is not null and new.decided_by_user_id is distinct from actor_user) then raise exception 'portfolio retention disposition requires an actor-bound separate approver'; end if;
 if not public.portfolio_retention_actor_has_permission('education.portfolios.custody.manage') or not public.portfolio_retention_actor_has_permission('earchiva.manage') then raise exception 'portfolio retention disposition approver lacks current permissions'; end if;
 new.decided_by_user_id:=actor_user;
 return new;
end $$;
create trigger portfolio_retention_disposition_decision_guard before insert or update or delete on portfolio_retention_disposition_decisions for each row execute function public.portfolio_retention_disposition_decision_guard();
create or replace function public.portfolio_retention_disposition_receipt_guard()
returns trigger language plpgsql security definer set search_path=pg_catalog,public as $$
declare actor text:=nullif(current_setting('app.actor_subject',true),''); operation_ref text:=nullif(current_setting('app.portfolio_retention_disposition_operation_id',true),'');
begin
 if tg_op<>'INSERT' then raise exception 'portfolio retention disposition receipts are append-only'; end if;
 if new.outcome='released' and actor='portfolio-retention-disposition-worker' and operation_ref is not null and exists(
  select 1 from portfolio_retention_disposition_operations o join portfolio_retention_disposition_requests r on r.id=o.request_id join portfolio_retention_disposition_decisions d on d.request_id=r.id and d.decision='approved'
  where o.id=operation_ref::uuid and o.status='leased' and r.id=new.request_id and r.status='approved' and r.transition_id=new.transition_id
 ) then return new; end if;
 if new.outcome='blocked' and exists(select 1 from portfolio_retention_disposition_decisions d where d.request_id=new.request_id and d.decision='rejected' and d.decided_by_subject=actor) then return new; end if;
 if new.outcome='blocked' and actor='portfolio-retention-disposition-worker' and operation_ref is not null and exists(
  select 1 from portfolio_retention_disposition_operations o join portfolio_retention_disposition_requests r on r.id=o.request_id
  where o.id=operation_ref::uuid and o.status='leased' and r.id=new.request_id and r.status='approved' and r.transition_id=new.transition_id
 ) then return new; end if;
 raise exception 'retention disposition receipt lacks exact authorization';
end $$;
create trigger portfolio_retention_disposition_receipt_guard before insert or update or delete on portfolio_retention_disposition_receipts for each row execute function public.portfolio_retention_disposition_receipt_guard();

create or replace function public.portfolio_retention_disposition_attempt_guard()
returns trigger language plpgsql security definer set search_path=pg_catalog,public as $$
begin
 if tg_op<>'INSERT' then raise exception 'portfolio retention disposition evidence is append-only'; end if;
 if current_setting('app.actor_subject',true)<>'portfolio-retention-disposition-worker' then raise exception 'portfolio retention disposition attempt requires worker'; end if;
 return new;
end $$;
create trigger portfolio_retention_disposition_attempt_guard before insert or update or delete on portfolio_retention_disposition_attempts for each row execute function public.portfolio_retention_disposition_attempt_guard();

create or replace function public.portfolio_retention_expiry_event_guard()
returns trigger language plpgsql security definer set search_path=pg_catalog,public as $$
begin
 if tg_op='DELETE' then raise exception 'portfolio retention expiry evidence cannot be deleted'; end if;
 if tg_op='UPDATE'
    and current_setting('app.actor_subject',true)='portfolio-retention-disposition-worker'
    and old.consumed_at is null and new.consumed_at is not null
    and row(new.transition_id,new.tenant_code,new.institution_id,new.required_retention_until,new.promoted_at,new.promoted_by_subject)
       is not distinct from row(old.transition_id,old.tenant_code,old.institution_id,old.required_retention_until,old.promoted_at,old.promoted_by_subject)
    and exists(select 1 from portfolio_retention_disposition_receipts receipt where receipt.transition_id=old.transition_id and receipt.outcome='released' and receipt.observed_hold_active=false) then return new; end if;
 if tg_op<>'INSERT' then raise exception 'portfolio retention expiry evidence is append-only'; end if;
 if current_setting('app.actor_subject',true)<>'portfolio-storage-lifecycle-worker' then raise exception 'portfolio retention expiry evidence requires lifecycle worker'; end if;
 return new;
end $$;
create trigger education_portfolio_retention_expiry_event_guard before insert or update or delete on education_portfolio_retention_expiry_events for each row execute function public.portfolio_retention_expiry_event_guard();

-- A disposition worker holds this advisory lock from its final reference
-- check through S3 verification. Every mutation that could add a reference or
-- change a portfolio legal hold takes the same lock first.
create or replace function public.portfolio_retention_release_fence()
returns trigger language plpgsql security definer set search_path=pg_catalog,public as $$
declare portfolio_ref uuid; version_ref uuid;
begin
 if tg_table_name='education_portfolio_documents' then
  version_ref:=case when tg_op='DELETE' then old.archive_version_id else new.archive_version_id end;
  if version_ref is not null then perform pg_advisory_xact_lock(hashtextextended(version_ref::text,0)); end if;
 else
  portfolio_ref:=case when tg_op='DELETE' then old.id else new.id end;
  for version_ref in select archive_version_id from education_portfolio_documents where portfolio_id=portfolio_ref and institution_id=(case when tg_op='DELETE' then old.institution_id else new.institution_id end) and archive_version_id is not null order by archive_version_id loop
   perform pg_advisory_xact_lock(hashtextextended(version_ref::text,0));
  end loop;
 end if;
 return case when tg_op='DELETE' then old else new end;
end $$;
create trigger portfolio_retention_release_fence_document before insert or update or delete on education_portfolio_documents for each row execute function public.portfolio_retention_release_fence();
create trigger portfolio_retention_release_fence_hold before update of legal_hold_active on education_portfolios for each row execute function public.portfolio_retention_release_fence();

create or replace function public.portfolio_retention_disposition_operation_guard()
returns trigger language plpgsql security definer set search_path=pg_catalog,public as $$
declare actor text:=nullif(current_setting('app.actor_subject',true),''); actor_user uuid:=public.portfolio_retention_actor_user_id();
begin
 if tg_op='DELETE' then raise exception 'retention disposition operations cannot be deleted'; end if;
 if tg_op='INSERT' then
  if not exists(select 1 from portfolio_retention_disposition_requests r join portfolio_retention_disposition_decisions d on d.request_id=r.id and d.decision='approved' where r.id=new.request_id and r.status='approved' and r.tenant_code=new.tenant_code and r.institution_id=new.institution_id and d.decided_by_user_id=actor_user) then raise exception 'retention disposition operation requires actor-bound approved decision'; end if;
  return new;
 end if;
 if row(new.id,new.request_id,new.tenant_code,new.institution_id,new.created_at) is distinct from row(old.id,old.request_id,old.tenant_code,old.institution_id,old.created_at) then raise exception 'retention disposition operation provenance is immutable'; end if;
 if old.status in ('released','deadletter') then raise exception 'terminal retention disposition operation is immutable'; end if;
 if actor<>'portfolio-retention-disposition-worker' then raise exception 'retention disposition operation requires worker'; end if;
 if not ((old.status='queued' and new.status='leased') or (old.status='leased' and new.status in ('queued','released','blocked','deadletter'))) then raise exception 'invalid retention disposition operation transition'; end if;
 return new;
end $$;
create trigger portfolio_retention_disposition_operation_guard before insert or update or delete on portfolio_retention_disposition_operations for each row execute function public.portfolio_retention_disposition_operation_guard();

-- Reinstall the complete established guards, with one receipt-backed branch.
create or replace function public.archive_version_custody_intent_guard()
returns trigger language plpgsql security definer set search_path=pg_catalog,public as $$
declare i portfolio_custody_upload_intents%rowtype; step education_portfolio_storage_transitions%rowtype; operation_ref text; disposition_operation_ref text; released boolean;
begin
 if tg_op='UPDATE' and old.portfolio_custody_intent_id is not null and new.portfolio_custody_intent_id is distinct from old.portfolio_custody_intent_id then raise exception 'archive custody intent link is immutable'; end if;
 if new.custody_hold_active and new.portfolio_custody_intent_id is null then raise exception 'custody-held archive version requires verified custody intent'; end if;
 if new.portfolio_custody_intent_id is null then return new; end if;
 select * into i from portfolio_custody_upload_intents where id=new.portfolio_custody_intent_id;
 if not found or i.status not in ('stored','committed') or i.institution_id<>new.institution_id or i.reserved_document_id<>new.document_id or i.reserved_version_id<>new.id or i.bucket_name<>new.source_bucket or i.object_key<>new.source_object_key or i.stored_version_id<>new.source_object_version_id or i.stored_etag<>new.source_object_etag or lower(i.expected_sha256)<>lower(new.source_sha256) or i.expected_size_bytes<>new.source_size_bytes or i.expected_mime_type<>new.mime_type or not i.custody_hold_active then raise exception 'archive version must retain matching verified custody intent'; end if;
 if new.custody_hold_active then return new; end if;
 if current_setting('app.actor_subject',true)='portfolio-retention-disposition-worker' then
  disposition_operation_ref:=nullif(current_setting('app.portfolio_retention_disposition_operation_id',true),'');
  if disposition_operation_ref is not null then
   select exists(select 1 from portfolio_retention_disposition_operations o join portfolio_retention_disposition_requests q on q.id=o.request_id join portfolio_retention_disposition_receipts receipt on receipt.request_id=q.id
    where o.id=disposition_operation_ref::uuid and o.status='leased' and q.status='approved' and q.archive_version_id=new.id and q.institution_id=new.institution_id
      and q.source_bucket=new.source_bucket and q.source_object_key=new.source_object_key and q.source_object_version_id=new.source_object_version_id and q.source_object_etag=new.source_object_etag
      and q.source_sha256=new.source_sha256 and q.source_size_bytes=new.source_size_bytes and q.source_mime_type=new.mime_type
      and receipt.transition_id=q.transition_id and receipt.outcome='released' and receipt.observed_hold_active=false) into released;
  end if;
  if released then return new; end if;
  raise exception 'portfolio custody release lacks exact approved disposition receipt';
 end if;
 if tg_op<>'UPDATE' or new.retention_until is null then raise exception 'portfolio custody release requires verified retention transition'; end if;
 operation_ref:=nullif(current_setting('app.portfolio_storage_lifecycle_operation_id',true),'');
 if operation_ref is null or current_setting('app.actor_subject',true)<>'portfolio-storage-lifecycle-worker' then raise exception 'portfolio custody release requires lifecycle worker'; end if;
 select s.* into step from education_portfolio_storage_transitions s where s.operation_id=operation_ref::uuid and s.archive_version_id=new.id
   and s.institution_id=new.institution_id and s.status='processing' and s.storage_verified_at is not null and s.observed_custody_required=false for key share;
 if not found or step.observed_retention_until is null or step.observed_retention_until<new.retention_until then raise exception 'portfolio custody release lacks exact storage verification'; end if;
 return new;
end $$;
create or replace function public.enforce_archive_version_immutability()
returns trigger language plpgsql security definer set search_path=pg_catalog,public as $$
declare step education_portfolio_storage_transitions%rowtype; operation_ref text; disposition_operation_ref text; released boolean;
begin
 if tg_op='DELETE' then raise exception 'archive document versions are immutable and cannot be deleted'; end if;
 if row(new.document_id,new.institution_id,new.version_no,new.mime_type,new.bucket_name,new.object_key,new.hash_sha256,new.size_bytes,new.source_bucket,new.source_object_key,new.source_sha256,new.source_size_bytes,new.source_object_version_id,new.source_object_etag,new.created_at,new.portfolio_custody_intent_id) is distinct from row(old.document_id,old.institution_id,old.version_no,old.mime_type,old.bucket_name,old.object_key,old.hash_sha256,old.size_bytes,old.source_bucket,old.source_object_key,old.source_sha256,old.source_size_bytes,old.source_object_version_id,old.source_object_etag,old.created_at,old.portfolio_custody_intent_id) then raise exception 'archive document version bitstream provenance is immutable'; end if;
 if row(new.retention_until,new.custody_hold_active,new.legal_hold_active,new.retention_disposition_hold_active) is not distinct from row(old.retention_until,old.custody_hold_active,old.legal_hold_active,old.retention_disposition_hold_active) then return new; end if;
 if old.retention_until is not null and (new.retention_until is null or new.retention_until<old.retention_until) then raise exception 'archive retention cannot be removed or shortened'; end if;
 if current_setting('app.actor_subject',true)='portfolio-retention-disposition-worker' then
  disposition_operation_ref:=nullif(current_setting('app.portfolio_retention_disposition_operation_id',true),'');
  if disposition_operation_ref is not null then
   select exists(select 1 from portfolio_retention_disposition_operations o join portfolio_retention_disposition_requests q on q.id=o.request_id join portfolio_retention_disposition_receipts receipt on receipt.request_id=q.id
    where o.id=disposition_operation_ref::uuid and o.status='leased' and q.status='approved' and q.archive_version_id=new.id and q.institution_id=new.institution_id
      and q.source_bucket=new.source_bucket and q.source_object_key=new.source_object_key and q.source_object_version_id=new.source_object_version_id and q.source_object_etag=new.source_object_etag
      and q.source_sha256=new.source_sha256 and q.source_size_bytes=new.source_size_bytes and q.source_mime_type=new.mime_type
      and receipt.transition_id=q.transition_id and receipt.outcome='released' and receipt.observed_hold_active=false
      and new.custody_hold_active=false and new.legal_hold_active=false and new.retention_disposition_hold_active=false) into released;
  end if;
  if released then return new; end if;
  raise exception 'archive lifecycle projection lacks exact approved disposition receipt';
 end if;
 operation_ref:=nullif(current_setting('app.portfolio_storage_lifecycle_operation_id',true),'');
 if operation_ref is null or current_setting('app.actor_subject',true)<>'portfolio-storage-lifecycle-worker' then raise exception 'archive lifecycle projection requires verified worker transition'; end if;
 select s.* into step from education_portfolio_storage_transitions s
 where s.operation_id=operation_ref::uuid and s.archive_version_id=new.id and s.institution_id=new.institution_id
   and s.status='processing' and s.storage_verified_at is not null for key share;
 if not found
    or step.observed_custody_required is distinct from new.custody_hold_active
    or step.observed_legal_hold_required is distinct from new.legal_hold_active
    or step.observed_retention_disposition_hold_required is distinct from new.retention_disposition_hold_active
    or step.observed_hold_active is distinct from (new.custody_hold_active or new.legal_hold_active or new.retention_disposition_hold_active)
    or (step.required_retention_until is not null and (step.observed_retention_until is null or step.observed_retention_until<step.required_retention_until or new.retention_until is null or new.retention_until<step.required_retention_until)) then
  raise exception 'archive lifecycle projection lacks exact storage verification';
 end if;
 return new;
end $$;

create or replace function public.portfolio_storage_transition_guard()
returns trigger language plpgsql security definer set search_path=pg_catalog,public as $$
declare actor text:=nullif(current_setting('app.actor_subject',true),''); retry_exists boolean; released boolean; disposition_operation_ref text:=nullif(current_setting('app.portfolio_retention_disposition_operation_id',true),'');
begin
 if tg_op='DELETE' then raise exception 'portfolio storage transitions cannot be deleted'; end if;
 if row(new.id,new.operation_id,new.tenant_code,new.institution_id,new.portfolio_id,new.archive_document_id,new.archive_version_id,new.source_bucket,new.source_object_key,new.source_object_version_id,new.source_object_etag,new.source_sha256,new.source_size_bytes,new.required_retention_until,new.created_at) is distinct from row(old.id,old.operation_id,old.tenant_code,old.institution_id,old.portfolio_id,old.archive_document_id,old.archive_version_id,old.source_bucket,old.source_object_key,old.source_object_version_id,old.source_object_etag,old.source_sha256,old.source_size_bytes,old.required_retention_until,old.created_at) then raise exception 'portfolio storage transition provenance is immutable'; end if;
 if old.storage_verified_at is not null and row(new.storage_verified_at,new.observed_retention_until,new.observed_hold_active,new.observed_custody_required,new.observed_legal_hold_required,new.observed_retention_disposition_hold_required,new.protective_extension_provenance) is distinct from row(old.storage_verified_at,old.observed_retention_until,old.observed_hold_active,old.observed_custody_required,old.observed_legal_hold_required,old.observed_retention_disposition_hold_required,old.protective_extension_provenance) then raise exception 'portfolio storage verification evidence is immutable'; end if;
 if actor='portfolio-retention-disposition-worker' then
  if disposition_operation_ref is not null then
   select exists(select 1 from portfolio_retention_disposition_operations o join portfolio_retention_disposition_requests r on r.id=o.request_id join portfolio_retention_disposition_receipts receipt on receipt.request_id=r.id
    where o.id=disposition_operation_ref::uuid and o.status='leased' and r.status='approved' and r.transition_id=old.id and receipt.transition_id=old.id and receipt.outcome='released' and receipt.observed_hold_active=false) into released;
  end if;
  if released and old.status='blocked' and old.last_error='portfolio_retention_expired_review_required' and new.status='completed' and new.last_error='' and new.completed_at is not null
     and row(new.attempts,new.available_at,new.locked_at,new.locked_by,new.storage_verified_at,new.observed_retention_until,new.observed_hold_active,new.observed_custody_required,new.observed_legal_hold_required,new.observed_retention_disposition_hold_required,new.protective_extension_provenance,new.dead_lettered_at)
       is not distinct from row(old.attempts,old.available_at,null::timestamptz,''::text,old.storage_verified_at,old.observed_retention_until,old.observed_hold_active,old.observed_custody_required,old.observed_legal_hold_required,old.observed_retention_disposition_hold_required,old.protective_extension_provenance,old.dead_lettered_at) then return new; end if;
  raise exception 'disposition transition lacks exact released receipt';
 end if;
 if actor='portfolio-storage-lifecycle-worker' then return new; end if;
 select exists(select 1 from education_portfolio_storage_transition_history h where h.transition_id=old.id and h.requested_by_subject=actor and h.created_at>=transaction_timestamp()) into retry_exists;
 if old.last_error='portfolio_retention_expired_review_required' then raise exception 'expired portfolio retention requires approved immutable disposition'; end if;
 if not retry_exists or old.status not in ('blocked','dead_letter') or new.status<>'pending' or new.attempts<>0 or new.locked_at is not null or new.locked_by<>'' or new.last_error<>'' or new.dead_lettered_at is not null or row(new.storage_verified_at,new.observed_retention_until,new.observed_hold_active,new.observed_custody_required,new.observed_legal_hold_required,new.observed_retention_disposition_hold_required,new.protective_extension_provenance,new.completed_at) is distinct from row(old.storage_verified_at,old.observed_retention_until,old.observed_hold_active,old.observed_custody_required,old.observed_legal_hold_required,old.observed_retention_disposition_hold_required,old.protective_extension_provenance,old.completed_at) then raise exception 'invalid portfolio storage transition mutation'; end if;
 return new;
end $$;
