-- A legal admission record is never created directly from browser supplied
-- facts.  A short-lived preparation reserves the complete, server-derived
-- aggregate which is later bound to the exact signed WORM PDF.
create table school_admission_legal_preparations (
 id uuid primary key default gen_random_uuid(), tenant_code text not null, institution_id text not null,
 artifact_kind text not null check(artifact_kind in ('admission_decision','admission_appeal_resolution')),
 artifact_id uuid not null, application_id uuid not null, appeal_id uuid,
 resulting_decision_id uuid, capacity_allocation_id uuid,
 policy_evaluation_v2_id uuid not null, aggregate_expected_version integer not null check(aggregate_expected_version>0),
 canonical_payload jsonb not null check(jsonb_typeof(canonical_payload)='object'), canonical_payload_bytes text not null check(btrim(canonical_payload_bytes)<>''), canonical_payload_sha256 text not null check(canonical_payload_sha256 ~ '^[0-9a-f]{64}$'),
 resulting_decision_payload jsonb, resulting_decision_payload_bytes text not null default '', resulting_decision_payload_sha256 text not null default '' check(resulting_decision_payload_sha256='' or resulting_decision_payload_sha256 ~ '^[0-9a-f]{64}$'),
 preparation_snapshot jsonb not null check(jsonb_typeof(preparation_snapshot)='object'),
 prepared_by_subject text not null, prepared_at timestamptz not null default now(), expires_at timestamptz not null,
 status text not null default 'prepared' check(status in ('prepared','finalized','cancelled','expired')),
 finalized_by_subject text not null default '', finalized_at timestamptz,
 cancellation_reason text not null default '', expected_version integer not null default 1 check(expected_version>0),
 unique(tenant_code,institution_id,id), unique(tenant_code,institution_id,artifact_kind,artifact_id),
 foreign key(tenant_code,institution_id) references app_tenants(code,institution_id) on delete restrict,
 foreign key(tenant_code,institution_id,application_id) references school_admission_applications(tenant_code,institution_id,id) on delete restrict,
 foreign key(tenant_code,institution_id,appeal_id) references school_admission_appeals(tenant_code,institution_id,id) on delete restrict,
 foreign key(tenant_code,institution_id,capacity_allocation_id) references school_admission_capacity_allocations(tenant_code,institution_id,id) on delete restrict,
 foreign key(tenant_code,institution_id,policy_evaluation_v2_id) references school_operation_policy_evaluations_v2(tenant_code,institution_id,id) on delete restrict,
 check((status='finalized')=(finalized_at is not null and finalized_by_subject<>'')),
 check((artifact_kind='admission_decision')=(appeal_id is null))
);

create table school_admission_signer_authorizations (
 id uuid primary key default gen_random_uuid(), tenant_code text not null, institution_id text not null,
 certificate_sha256 text not null check(certificate_sha256 ~ '^[0-9a-f]{64}$'), user_id uuid not null, actor_subject text not null,
 permission_code text not null check(permission_code in ('education.admissions.decide','education.admissions.appeals.manage')),
 valid_from timestamptz not null default now(), valid_until timestamptz not null,
 status text not null default 'proposed' check(status in ('proposed','active','revoked','expired')),
 proposed_by_subject text not null, proposed_at timestamptz not null default now(),
 approved_by_subject text not null default '', approved_at timestamptz,
 revoked_by_subject text not null default '', revoked_at timestamptz, revocation_reason text not null default '',
 expected_version integer not null default 1 check(expected_version>0),
 unique(tenant_code,institution_id,id),
 foreign key(tenant_code,institution_id) references app_tenants(code,institution_id) on delete restrict,
 foreign key(user_id) references app_users(id) on delete restrict,
 check(valid_until>valid_from), check(approved_by_subject='' or approved_by_subject<>proposed_by_subject),
 check((status='proposed' and approved_at is null and approved_by_subject='') or (status in ('active','revoked','expired') and approved_at is not null and approved_by_subject<>'')),
 check(status<>'revoked' or (revoked_at is not null and revoked_by_subject<>'' and revocation_reason<>''))
);
create unique index school_admission_signer_authorization_active_unique on school_admission_signer_authorizations(tenant_code,institution_id,certificate_sha256,actor_subject,permission_code) where status='active';

create or replace function public.school_admission_signer_authorization_guard() returns trigger language plpgsql security definer set search_path=pg_catalog,public as $$
declare resolved_subject text;
begin
 if tg_op='DELETE' then raise exception 'admission signer authorization is immutable'; end if;
 if not public.can_bypass_tenant_rls() and (new.tenant_code<>public.current_tenant_code() or new.institution_id<>public.current_institution_id()) then raise exception 'admission signer authorization tenant/institution context mismatch'; end if;
 if tg_op='INSERT' then
  if not public.school_admission_actor_has_position('director') then raise exception using errcode='42501',message='admission signer proposal requires an effective director'; end if;
  select u.sub into resolved_subject from app_users u where u.id=new.user_id and u.status='active' and exists(select 1 from app_memberships m where m.user_id=u.id and m.tenant_code=new.tenant_code and public.education_membership_is_eligible(u.id,new.tenant_code,new.institution_id,m.position_code));
  if resolved_subject is null or resolved_subject<>new.actor_subject then raise exception using errcode='23514',message='signer authorization user and actor subject must match'; end if;
   new.proposed_by_subject:=current_setting('app.actor_subject',true); new.status:='proposed'; new.expected_version:=1;
 elsif tg_op='UPDATE' then
  if new.tenant_code<>old.tenant_code or new.institution_id<>old.institution_id or new.certificate_sha256<>old.certificate_sha256 or new.user_id<>old.user_id or new.actor_subject<>old.actor_subject or new.permission_code<>old.permission_code or new.valid_from<>old.valid_from or new.valid_until<>old.valid_until or new.proposed_by_subject<>old.proposed_by_subject or new.proposed_at<>old.proposed_at then raise exception 'admission signer authorization legal facts are immutable'; end if;
   if new.status=old.status then raise exception 'admission signer authorization lifecycle records are immutable'; end if;
   if old.status<>'proposed' and (new.approved_by_subject<>old.approved_by_subject or new.approved_at is distinct from old.approved_at) then raise exception 'admission signer approval provenance is immutable'; end if;
   if old.status<>'active' and (new.revoked_by_subject<>old.revoked_by_subject or new.revoked_at is distinct from old.revoked_at or new.revocation_reason<>old.revocation_reason) then raise exception 'admission signer revocation provenance is immutable'; end if;
   new.expected_version:=old.expected_version+1;
   if old.status='proposed' and new.status='active' then
    if new.valid_until<=now() then raise exception using errcode='23514',message='expired signer authorization cannot be approved'; end if;
    if current_setting('app.actor_subject',true)=old.proposed_by_subject or not public.school_admission_actor_has_position('director') then raise exception using errcode='42501',message='signer authorization requires a distinct director approver'; end if;
   new.approved_by_subject:=current_setting('app.actor_subject',true); new.approved_at:=now();
  elsif old.status='active' and new.status='revoked' then
   if not public.school_admission_actor_has_position('director') then raise exception using errcode='42501',message='signer revocation requires an effective director'; end if;
   new.revoked_by_subject:=current_setting('app.actor_subject',true); new.revoked_at:=now();
  elsif old.status='active' and new.status='expired' and now()>=old.valid_until then null;
  elsif new.status<>old.status then raise exception 'invalid signer authorization lifecycle transition'; end if;
 end if;
 return new;
end $$;
revoke all on function public.school_admission_signer_authorization_guard() from public;
create trigger school_admission_signer_authorization_guard before insert or update or delete on school_admission_signer_authorizations for each row execute function public.school_admission_signer_authorization_guard();

create or replace function public.school_admission_legal_preparation_guard() returns trigger language plpgsql security definer set search_path=pg_catalog,public as $$
begin
 if tg_op='DELETE' then raise exception 'admission legal preparations are append-only'; end if;
 if not public.can_bypass_tenant_rls() and (new.tenant_code<>public.current_tenant_code() or new.institution_id<>public.current_institution_id()) then raise exception 'admission legal preparation tenant/institution context mismatch'; end if;
 if tg_op='INSERT' then
  if nullif(btrim(current_setting('app.actor_subject',true)),'') is null then raise exception 'admission legal preparation requires actor'; end if;
   new.prepared_by_subject:=current_setting('app.actor_subject',true); new.status:='prepared'; new.expected_version:=1;
   if new.expires_at<=now() or new.expires_at>now()+interval '30 minutes' then raise exception 'legal preparation expiry must be within thirty minutes'; end if;
   if new.canonical_payload_bytes::jsonb<>new.canonical_payload or encode(digest(convert_to(new.canonical_payload_bytes,'UTF8'),'sha256'),'hex')<>new.canonical_payload_sha256 then raise exception 'canonical legal preparation payload bytes and hash must match'; end if;
   if new.resulting_decision_id is null then
    if new.resulting_decision_payload is not null or new.resulting_decision_payload_bytes<>'' or new.resulting_decision_payload_sha256<>'' then raise exception 'non-resulting appeal preparation cannot carry a resulting decision payload'; end if;
   elsif new.resulting_decision_payload is null or new.resulting_decision_payload_bytes='' or new.resulting_decision_payload_sha256='' or new.resulting_decision_payload_bytes::jsonb<>new.resulting_decision_payload or encode(digest(convert_to(new.resulting_decision_payload_bytes,'UTF8'),'sha256'),'hex')<>new.resulting_decision_payload_sha256 then raise exception 'resulting decision payload bytes and hash must match'; end if;
 elsif tg_op='UPDATE' then
  if new.tenant_code<>old.tenant_code or new.institution_id<>old.institution_id or new.artifact_kind<>old.artifact_kind or new.artifact_id<>old.artifact_id or new.application_id<>old.application_id or new.appeal_id is distinct from old.appeal_id or new.resulting_decision_id is distinct from old.resulting_decision_id or new.capacity_allocation_id is distinct from old.capacity_allocation_id or new.policy_evaluation_v2_id<>old.policy_evaluation_v2_id or new.canonical_payload<>old.canonical_payload or new.canonical_payload_bytes<>old.canonical_payload_bytes or new.canonical_payload_sha256<>old.canonical_payload_sha256 or new.resulting_decision_payload is distinct from old.resulting_decision_payload or new.resulting_decision_payload_bytes<>old.resulting_decision_payload_bytes or new.resulting_decision_payload_sha256<>old.resulting_decision_payload_sha256 or new.preparation_snapshot<>old.preparation_snapshot or new.prepared_by_subject<>old.prepared_by_subject or new.prepared_at<>old.prepared_at or new.expires_at<>old.expires_at then raise exception 'admission legal preparation facts are immutable'; end if;
  if new.status=old.status then raise exception 'admission legal preparation lifecycle records are immutable'; end if;
  new.expected_version:=old.expected_version+1;
  if old.status='prepared' and new.status='finalized' then new.finalized_by_subject:=current_setting('app.actor_subject',true); new.finalized_at:=now(); if new.cancellation_reason<>'' then raise exception 'finalized preparation cannot have a cancellation reason'; end if;
  elsif old.status='prepared' and new.status='cancelled' then if btrim(new.cancellation_reason)='' or new.finalized_by_subject<>'' or new.finalized_at is not null then raise exception 'cancelled preparation requires only a reason'; end if;
  elsif old.status='prepared' and new.status='expired' then if new.cancellation_reason<>'expired' or new.finalized_by_subject<>'' or new.finalized_at is not null then raise exception 'expired preparation requires the server expiry reason'; end if;
  elsif new.status<>old.status then raise exception 'invalid admission preparation transition'; end if;
 end if;
 return new;
end $$;
revoke all on function public.school_admission_legal_preparation_guard() from public;
create trigger school_admission_legal_preparation_guard before insert or update or delete on school_admission_legal_preparations for each row execute function public.school_admission_legal_preparation_guard();

alter table school_admission_legal_preparations enable row level security; alter table school_admission_legal_preparations force row level security;
alter table school_admission_signer_authorizations enable row level security; alter table school_admission_signer_authorizations force row level security;
create policy school_admission_legal_preparation_tenant_isolation on school_admission_legal_preparations using(public.can_bypass_tenant_rls() or (tenant_code=public.current_tenant_code() and institution_id=public.current_institution_id())) with check(public.can_bypass_tenant_rls() or (tenant_code=public.current_tenant_code() and institution_id=public.current_institution_id()));
create policy school_admission_signer_authorization_tenant_isolation on school_admission_signer_authorizations using(public.can_bypass_tenant_rls() or (tenant_code=public.current_tenant_code() and institution_id=public.current_institution_id())) with check(public.can_bypass_tenant_rls() or (tenant_code=public.current_tenant_code() and institution_id=public.current_institution_id()));
create trigger school_admission_legal_preparation_entity_version after insert or update or delete on school_admission_legal_preparations for each row execute function public.record_entity_version();
create trigger school_admission_signer_authorization_entity_version after insert or update or delete on school_admission_signer_authorizations for each row execute function public.record_entity_version();
create trigger school_admission_legal_preparation_no_hard_delete before delete on school_admission_legal_preparations for each row execute function public.school_operations_no_hard_delete();
create trigger school_admission_signer_authorization_no_hard_delete before delete on school_admission_signer_authorizations for each row execute function public.school_operations_no_hard_delete();

insert into app_permissions(code,label) values
 ('education.admissions.signer.manage','Propose and revoke admission signer authorizations'),
 ('education.admissions.signer.approve','Approve admission signer authorizations') on conflict(code) do update set label=excluded.label;
insert into app_position_permissions(position_code,permission_code) values
 ('director','education.admissions.signer.manage'),('director','education.admissions.signer.approve') on conflict do nothing;
