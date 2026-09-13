-- Admission DSS retention authority is deliberately separate from the
-- operational policy used by the signing path.  The source checksum is a
-- captured legal fact, rather than a mutable lookup at enforcement time.
create table school_admission_retention_rule_versions (
 id uuid primary key default gen_random_uuid(), tenant_code text not null, institution_id text not null,
 artifact_kind text not null check(artifact_kind in ('admission_dss')),
 jurisdiction text not null default 'RO' check(btrim(jurisdiction)<>''),
 status text not null check(status in ('proposed','active','superseded','revoked')),
 minimum_retention_days integer not null check(minimum_retention_days between 1 and 36500),
 effective_from date not null, effective_to date,
 source_id uuid not null, source_checksum_sha256 text not null check(source_checksum_sha256 ~ '^[a-f0-9]{64}$'),
 proposed_by_subject text not null, proposed_at timestamptz not null default now(),
 approved_by_subject text not null default '', approved_at timestamptz,
 revoked_by_subject text not null default '', revoked_at timestamptz, revocation_reason text not null default '',
 expected_version integer not null default 1 check(expected_version>0),
 unique(tenant_code,institution_id,id),
 foreign key(tenant_code,institution_id) references app_tenants(code,institution_id) on delete restrict,
 foreign key(tenant_code,institution_id,source_id) references school_regulatory_sources(tenant_code,institution_id,id) on delete restrict,
 check(effective_to is null or effective_to>=effective_from),
 check((status='proposed' and approved_by_subject='' and approved_at is null) or (status in ('active','superseded','revoked') and approved_by_subject<>'' and approved_at is not null)),
 check(approved_by_subject='' or approved_by_subject<>proposed_by_subject),
 check(status<>'revoked' or (revoked_by_subject<>'' and revoked_at is not null and revocation_reason<>''))
);
create unique index school_admission_retention_rule_one_active on school_admission_retention_rule_versions(tenant_code,institution_id,artifact_kind) where status='active';

create or replace function public.school_admission_actor_has_position(required_position text) returns boolean language sql stable security definer set search_path=pg_catalog,public as $$
 select exists(select 1 from app_users u join app_memberships m on m.user_id=u.id and m.tenant_code=public.current_tenant_code() where (u.id::text=current_setting('app.actor_subject',true) or lower(u.sub)=lower(current_setting('app.actor_subject',true))) and u.status='active' and m.position_code=required_position and public.education_membership_is_eligible(u.id,public.current_tenant_code(),public.current_institution_id(),m.position_code));
$$;
revoke all on function public.school_admission_actor_has_position(text) from public;

create or replace function public.school_admission_retention_rule_guard() returns trigger language plpgsql security definer set search_path=pg_catalog,public as $$
declare source_checksum text; source_ok boolean;
begin
 perform pg_advisory_xact_lock(hashtextextended('school_admission_retention_rule:'||new.tenant_code||':'||new.institution_id||':'||new.artifact_kind,0));
 if tg_op='DELETE' then raise exception 'admission retention rule versions are immutable'; end if;
 if not public.can_bypass_tenant_rls() and (new.tenant_code<>public.current_tenant_code() or new.institution_id<>public.current_institution_id()) then raise exception 'admission retention rule tenant/institution context mismatch'; end if;
 if tg_op='INSERT' then
  if nullif(btrim(current_setting('app.actor_subject',true)),'') is null then raise exception 'admission retention rule requires authenticated actor provenance'; end if;
  if not public.school_admission_actor_has_position('director') then raise exception using errcode='42501',message='admission retention authority requires an effective director position'; end if;
  new.proposed_by_subject:=current_setting('app.actor_subject',true); new.status:='proposed'; new.expected_version:=1;
  select checksum_sha256, status='active' and effective_from<=new.effective_from and (effective_to is null or (new.effective_to is not null and effective_to>=new.effective_to)) into source_checksum,source_ok from school_regulatory_sources where tenant_code=new.tenant_code and institution_id=new.institution_id and id=new.source_id;
  if not coalesce(source_ok,false) or source_checksum='' then raise exception using errcode='23514',message='admission retention rule requires an active source covering its effective window'; end if;
  new.source_checksum_sha256:=source_checksum;
 elsif tg_op='UPDATE' then
  if new.tenant_code<>old.tenant_code or new.institution_id<>old.institution_id or new.id<>old.id or new.artifact_kind<>old.artifact_kind or new.jurisdiction<>old.jurisdiction or new.minimum_retention_days<>old.minimum_retention_days or new.effective_from<>old.effective_from or new.effective_to is distinct from old.effective_to or new.source_id<>old.source_id or new.source_checksum_sha256<>old.source_checksum_sha256 or new.proposed_by_subject<>old.proposed_by_subject or new.proposed_at<>old.proposed_at then raise exception 'admission retention rule legal facts are immutable'; end if;
  new.expected_version:=old.expected_version+1;
  if old.status='proposed' and new.status='active' then
   if nullif(btrim(current_setting('app.actor_subject',true)),'') is null or current_setting('app.actor_subject',true)=old.proposed_by_subject then raise exception 'admission retention rule requires a distinct approving actor'; end if;
   if not public.school_admission_actor_has_position('director') then raise exception using errcode='42501',message='admission retention approval requires an effective director position'; end if;
   select checksum_sha256, status='active' and checksum_sha256=old.source_checksum_sha256 and effective_from<=old.effective_from and (effective_to is null or (old.effective_to is not null and effective_to>=old.effective_to)) into source_checksum,source_ok from school_regulatory_sources where tenant_code=old.tenant_code and institution_id=old.institution_id and id=old.source_id for share;
   if not coalesce(source_ok,false) then raise exception using errcode='23514',message='admission retention rule source is no longer active or checksum-matched'; end if;
   new.approved_by_subject:=current_setting('app.actor_subject',true); new.approved_at:=now();
  elsif old.status='active' and new.status in ('superseded','revoked') then
   if new.status='revoked' then new.revoked_by_subject:=current_setting('app.actor_subject',true); new.revoked_at:=now(); end if;
  elsif new.status<>old.status then raise exception 'invalid admission retention rule lifecycle transition'; end if;
 end if;
 return new;
end $$;
revoke all on function public.school_admission_retention_rule_guard() from public;
create trigger school_admission_retention_rule_guard before insert or update or delete on school_admission_retention_rule_versions for each row execute function public.school_admission_retention_rule_guard();

alter table school_admission_dss_retention_policies add column rule_version_id uuid;
alter table school_admission_dss_retention_policies add constraint school_admission_dss_retention_policy_rule_fk foreign key(tenant_code,institution_id,rule_version_id) references school_admission_retention_rule_versions(tenant_code,institution_id,id) on delete restrict;
create or replace function public.school_admission_dss_retention_policy_authority_guard() returns trigger language plpgsql security definer set search_path=pg_catalog,public as $$
declare rule record;
begin
 perform pg_advisory_xact_lock(hashtextextended('school_admission_retention_policy:'||new.tenant_code||':'||new.institution_id,0));
 if tg_op='INSERT' then
  if not public.school_admission_actor_has_position('director') then raise exception using errcode='42501',message='admission retention policy activation requires an effective director position'; end if;
  select * into rule from school_admission_retention_rule_versions where tenant_code=new.tenant_code and institution_id=new.institution_id and id=new.rule_version_id and artifact_kind='admission_dss' and status='active' and effective_from<=new.effective_from and (effective_to is null or effective_to>=new.effective_from) for share;
  if not found then raise exception using errcode='23514',message='admission retention policy requires an approved active rule version'; end if;
  new.minimum_retention_days:=rule.minimum_retention_days; new.source_id:=rule.source_id;
 elsif tg_op='UPDATE' and (new.rule_version_id is distinct from old.rule_version_id or new.minimum_retention_days<>old.minimum_retention_days or new.source_id<>old.source_id) then raise exception 'admission retention policy authority facts are server-derived';
 end if;
 return new;
end $$;
revoke all on function public.school_admission_dss_retention_policy_authority_guard() from public;
create trigger school_admission_dss_retention_policy_authority_guard before insert or update on school_admission_dss_retention_policies for each row execute function public.school_admission_dss_retention_policy_authority_guard();

alter table school_admission_retention_rule_versions enable row level security;
alter table school_admission_retention_rule_versions force row level security;
create policy school_admission_retention_rule_tenant_isolation on school_admission_retention_rule_versions using(public.can_bypass_tenant_rls() or (tenant_code=public.current_tenant_code() and institution_id=public.current_institution_id())) with check(public.can_bypass_tenant_rls() or (tenant_code=public.current_tenant_code() and institution_id=public.current_institution_id()));
create trigger school_admission_retention_rule_no_hard_delete before delete on school_admission_retention_rule_versions for each row execute function public.school_operations_no_hard_delete();
create trigger school_admission_retention_rule_entity_version after insert or update or delete on school_admission_retention_rule_versions for each row execute function public.record_entity_version();
create trigger school_admission_dss_retention_policy_entity_version after insert or update or delete on school_admission_dss_retention_policies for each row execute function public.record_entity_version();

insert into app_permissions(code,label) values
 ('education.admissions.retention.manage','Propose and configure admission retention authority'),
 ('education.admissions.retention.approve','Approve admission retention authority') on conflict(code) do update set label=excluded.label;
insert into app_position_permissions(position_code,permission_code) values ('director','education.admissions.retention.manage'),('director','education.admissions.retention.approve') on conflict do nothing;
delete from app_position_permissions where position_code='director_adjunct' and permission_code in ('education.admissions.retention.manage','education.admissions.retention.approve');

-- A binding is legal only while its selected operational policy still points
-- at the approved authority version.  This replaces the legacy 0149 guard.
create or replace function public.school_admission_requires_dss_binding() returns trigger language plpgsql security definer set search_path=pg_catalog,public as $$
declare kind text:=case when tg_table_name='school_admission_decisions' then 'admission_decision' else 'admission_appeal_resolution' end; artifact_actor text:=case when tg_table_name='school_admission_decisions' then new.decided_by_subject else new.resolved_by_subject end; artifact_at timestamptz:=case when tg_table_name='school_admission_decisions' then new.decided_at else new.resolved_at end; ok boolean;
begin
 select exists(select 1 from school_admission_signed_artifact_bindings b join school_admission_dss_retention_policies p on p.tenant_code=b.tenant_code and p.institution_id=b.institution_id and p.id=b.retention_policy_id and p.status='active' and p.effective_from<=artifact_at::date and (p.effective_to is null or p.effective_to>=artifact_at::date) join school_admission_retention_rule_versions rule on rule.tenant_code=p.tenant_code and rule.institution_id=p.institution_id and rule.id=p.rule_version_id and rule.artifact_kind='admission_dss' and rule.status='active' and rule.effective_from<=artifact_at::date and (rule.effective_to is null or rule.effective_to>=artifact_at::date) join school_operation_policy_evaluations_v2 evaluation on evaluation.tenant_code=b.tenant_code and evaluation.institution_id=b.institution_id and evaluation.id=b.policy_evaluation_v2_id and evaluation.allowed join school_operation_policy_inputs input on input.tenant_code=evaluation.tenant_code and input.institution_id=evaluation.institution_id and input.id=evaluation.input_id where b.tenant_code=new.tenant_code and b.institution_id=new.institution_id and b.artifact_kind=kind and b.artifact_id=new.id and b.policy_evaluation_v2_id=new.policy_evaluation_v2_id and b.bound_by_subject=artifact_actor and evaluation.evaluated_by_subject=artifact_actor and input.created_by_subject=artifact_actor and b.archive_document_id=new.archive_document_id and b.archive_version_id=new.archive_version_id and b.archive_version_no=new.archive_version_no and b.archive_source_bucket=new.archive_source_bucket and b.archive_source_object_key=new.archive_source_object_key and b.archive_sha256=new.archive_sha256 and new.archive_retention_until >= artifact_at + make_interval(days=>rule.minimum_retention_days)) into ok;
 if not ok then raise exception using errcode='23514',message='admission decision/resolution requires exact valid DSS binding and approved legal retention authority'; end if;
 return null;
end $$;
revoke all on function public.school_admission_requires_dss_binding() from public;
