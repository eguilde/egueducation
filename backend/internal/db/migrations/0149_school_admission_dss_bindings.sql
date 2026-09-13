-- Admission outcomes are legal artifacts. A decision/resolution can commit
-- only with one immutable, successful DSS validation bound to its exact WORM
-- archive snapshot and the policy evaluation/actor that authorized it.

alter table education_signed_artifact_evidence
    drop constraint if exists education_signed_artifact_evidence_artifact_type_check,
    add constraint education_signed_artifact_evidence_artifact_type_check
    check (artifact_type in ('decision','publication','managerial_document','meeting_document','meeting_minute','meeting_resolution','admission_decision','admission_appeal_resolution'));

create unique index if not exists education_signed_artifact_evidence_scope_id_key
    on education_signed_artifact_evidence(tenant_code,institution_id,id);
create unique index if not exists education_signed_artifact_validations_scope_id_key
    on education_signed_artifact_validations(tenant_code,institution_id,id);

-- Replaces the older artifact resolver solely to add the two admission legal
-- aggregates; the storage and authenticated-subject checks remain identical.
create or replace function public.enforce_education_signed_artifact_evidence()
returns trigger language plpgsql security definer set search_path=pg_catalog,public as $$
declare actor_subject text:=nullif(btrim(current_setting('app.actor_subject',true)), ''); artifact_found boolean:=false; document_in_scope boolean:=false; session_can_bypass boolean:=false;
begin
 if tg_op in ('UPDATE','DELETE') then raise exception 'signed artifact evidence is append-only'; end if;
 select coalesce(rolsuper or rolbypassrls,false) into session_can_bypass from pg_catalog.pg_roles where rolname=session_user;
 if actor_subject is not null then new.submitted_by_subject:=actor_subject; elsif not session_can_bypass then raise exception 'signed artifact evidence requires authenticated actor provenance'; end if;
 if not public.can_bypass_tenant_rls() and (new.tenant_code<>public.current_tenant_code() or new.institution_id<>public.current_institution_id()) then raise exception 'signed artifact evidence tenant/institution context mismatch'; end if;
 case new.artifact_type
  when 'decision' then select exists(select 1 from education_decisions where id=new.artifact_id and institution_id=new.institution_id) into artifact_found;
  when 'publication' then select exists(select 1 from education_publications where id=new.artifact_id and institution_id=new.institution_id) into artifact_found;
  when 'managerial_document' then select exists(select 1 from education_managerial_documents where id=new.artifact_id and institution_id=new.institution_id) into artifact_found;
  when 'meeting_document' then select exists(select 1 from education_meeting_documents where id=new.artifact_id and institution_id=new.institution_id) into artifact_found;
  when 'meeting_minute' then select exists(select 1 from education_meeting_minutes where id=new.artifact_id and institution_id=new.institution_id) into artifact_found;
  when 'meeting_resolution' then select exists(select 1 from education_meeting_resolutions where id=new.artifact_id and institution_id=new.institution_id) into artifact_found;
  when 'admission_decision' then select exists(select 1 from school_admission_decisions where tenant_code=new.tenant_code and institution_id=new.institution_id and id=new.artifact_id) into artifact_found;
  when 'admission_appeal_resolution' then select exists(select 1 from school_admission_appeal_resolutions where tenant_code=new.tenant_code and institution_id=new.institution_id and id=new.artifact_id) into artifact_found;
 end case;
 if not artifact_found then raise exception 'signed artifact evidence must reference an artifact in the active institution'; end if;
 select exists(select 1 from archive_documents d join archive_document_versions v on v.id=new.storage_version_id and v.document_id=d.id where d.id=new.storage_document_id and d.institution_id=new.institution_id and v.institution_id=new.institution_id) into document_in_scope;
 if not document_in_scope then raise exception 'signed artifact evidence storage references must belong to active institution'; end if;
 return new;
end $$;
revoke all on function public.enforce_education_signed_artifact_evidence() from public;

create table school_admission_dss_retention_policies (
 id uuid primary key default gen_random_uuid(), tenant_code text not null, institution_id text not null,
 status text not null check(status in ('draft','active','superseded','revoked')),
 minimum_retention_days integer not null check(minimum_retention_days>=1 and minimum_retention_days<=36500), source_id uuid not null,
 effective_from date not null, effective_to date, created_by_subject text not null default '', created_at timestamptz not null default now(),
 unique(tenant_code,institution_id,id), foreign key(tenant_code,institution_id) references app_tenants(code,institution_id) on delete restrict,
 foreign key(tenant_code,institution_id,source_id) references school_regulatory_sources(tenant_code,institution_id,id) on delete restrict,
 check(effective_to is null or effective_to>=effective_from)
);
create unique index school_admission_dss_retention_one_active on school_admission_dss_retention_policies(tenant_code,institution_id) where status='active';
create or replace function public.school_admission_dss_retention_source_guard() returns trigger language plpgsql security definer set search_path=pg_catalog,public as $$ begin if not exists(select 1 from school_regulatory_sources source where source.tenant_code=new.tenant_code and source.institution_id=new.institution_id and source.id=new.source_id and source.status='active' and source.effective_from<=new.effective_from and (source.effective_to is null or source.effective_to>=new.effective_from)) then raise exception using errcode='23514',message='admission DSS retention policy requires an active regulatory source'; end if; return new; end $$;
revoke all on function public.school_admission_dss_retention_source_guard() from public;
create trigger school_admission_dss_retention_source_guard before insert or update on school_admission_dss_retention_policies for each row execute function public.school_admission_dss_retention_source_guard();

create table school_admission_signed_artifact_bindings (
 id uuid primary key default gen_random_uuid(), tenant_code text not null, institution_id text not null,
 artifact_kind text not null check(artifact_kind in ('admission_decision','admission_appeal_resolution')), artifact_id uuid not null,
 evidence_id uuid not null, validation_id uuid not null, policy_evaluation_v2_id uuid not null, retention_policy_id uuid not null,
 archive_document_id uuid not null, archive_version_id uuid not null, archive_version_no integer not null check(archive_version_no>0),
 archive_source_bucket text not null check(btrim(archive_source_bucket)<>''), archive_source_object_key text not null check(btrim(archive_source_object_key)<>''), archive_sha256 text not null check(archive_sha256~'^[0-9a-f]{64}$'), archive_size_bytes bigint not null check(archive_size_bytes>0),
 bound_by_subject text not null default '', bound_at timestamptz not null default now(),
 unique(tenant_code,institution_id,id), unique(tenant_code,institution_id,artifact_kind,artifact_id), unique(tenant_code,institution_id,evidence_id), unique(tenant_code,institution_id,validation_id),
 foreign key(tenant_code,institution_id) references app_tenants(code,institution_id) on delete restrict,
 foreign key(tenant_code,institution_id,evidence_id) references education_signed_artifact_evidence(tenant_code,institution_id,id) on delete restrict,
 foreign key(tenant_code,institution_id,validation_id) references education_signed_artifact_validations(tenant_code,institution_id,id) on delete restrict,
 foreign key(tenant_code,institution_id,retention_policy_id) references school_admission_dss_retention_policies(tenant_code,institution_id,id) on delete restrict,
 foreign key(tenant_code,institution_id,policy_evaluation_v2_id) references school_operation_policy_evaluations_v2(tenant_code,institution_id,id) on delete restrict
);

create or replace function public.school_admission_dss_binding_guard() returns trigger language plpgsql security definer set search_path=pg_catalog,public as $$
declare actor text:=nullif(btrim(current_setting('app.actor_subject',true)), ''); ok boolean;
begin
 if tg_op in ('UPDATE','DELETE') then raise exception 'admission DSS binding is append-only'; end if;
 if actor is null then raise exception 'admission DSS binding requires authenticated actor provenance'; end if;
 new.bound_by_subject:=actor;
 if not public.can_bypass_tenant_rls() and (new.tenant_code<>public.current_tenant_code() or new.institution_id<>public.current_institution_id()) then raise exception 'admission DSS binding tenant/institution context mismatch'; end if;
 select exists(select 1 from education_signed_artifact_evidence e join education_signed_artifact_validations v on v.tenant_code=e.tenant_code and v.institution_id=e.institution_id and v.evidence_id=e.id and v.id=new.validation_id where e.tenant_code=new.tenant_code and e.institution_id=new.institution_id and e.id=new.evidence_id and e.artifact_type=new.artifact_kind and e.artifact_id=new.artifact_id and e.document_sha256=new.archive_sha256 and e.storage_document_id=new.archive_document_id and e.storage_version_id=new.archive_version_id and e.storage_bucket=new.archive_source_bucket and e.storage_object_key=new.archive_source_object_key and e.signature_format='PAdES' and v.validation_status='valid' and btrim(v.trusted_list_provider)<>'' and btrim(v.validator_provider)<>'' and btrim(v.validator_version)<>'' and btrim(v.validation_policy)<>'' and v.observed_sha256=new.archive_sha256 and v.observed_size_bytes=new.archive_size_bytes and v.timestamp_token_sha256~'^[0-9a-f]{64}$' and v.timestamp_at is not null and btrim(v.timestamp_authority)<>'' and v.diagnostic_data<>'{}'::jsonb and v.detailed_report<>'{}'::jsonb and v.simple_report<>'{}'::jsonb and v.etsi_validation_report<>'{}'::jsonb and e.submitted_by_subject=actor and v.validated_by_subject=actor) into ok;
 if not ok then raise exception 'admission DSS binding must reference exact valid PAdES/DSS evidence and actor'; end if;
 return new;
end $$;
revoke all on function public.school_admission_dss_binding_guard() from public;
create trigger school_admission_dss_binding_append_only before insert or update or delete on school_admission_signed_artifact_bindings for each row execute function public.school_admission_dss_binding_guard();

create or replace function public.school_admission_requires_dss_binding() returns trigger language plpgsql security definer set search_path=pg_catalog,public as $$
declare kind text:=case when tg_table_name='school_admission_decisions' then 'admission_decision' else 'admission_appeal_resolution' end; artifact_actor text:=case when tg_table_name='school_admission_decisions' then new.decided_by_subject else new.resolved_by_subject end; artifact_at timestamptz:=case when tg_table_name='school_admission_decisions' then new.decided_at else new.resolved_at end; ok boolean;
begin
 select exists(select 1 from school_admission_signed_artifact_bindings b join school_admission_dss_retention_policies p on p.tenant_code=b.tenant_code and p.institution_id=b.institution_id and p.id=b.retention_policy_id and p.status='active' and p.effective_from<=artifact_at::date and (p.effective_to is null or p.effective_to>=artifact_at::date) join school_operation_policy_evaluations_v2 evaluation on evaluation.tenant_code=b.tenant_code and evaluation.institution_id=b.institution_id and evaluation.id=b.policy_evaluation_v2_id and evaluation.allowed join school_operation_policy_inputs input on input.tenant_code=evaluation.tenant_code and input.institution_id=evaluation.institution_id and input.id=evaluation.input_id where b.tenant_code=new.tenant_code and b.institution_id=new.institution_id and b.artifact_kind=kind and b.artifact_id=new.id and b.policy_evaluation_v2_id=new.policy_evaluation_v2_id and b.bound_by_subject=artifact_actor and evaluation.evaluated_by_subject=artifact_actor and input.created_by_subject=artifact_actor and b.archive_document_id=new.archive_document_id and b.archive_version_id=new.archive_version_id and b.archive_version_no=new.archive_version_no and b.archive_source_bucket=new.archive_source_bucket and b.archive_source_object_key=new.archive_source_object_key and b.archive_sha256=new.archive_sha256 and new.archive_retention_until >= artifact_at + make_interval(days=>p.minimum_retention_days)) into ok;
 if not ok then raise exception using errcode='23514',message='admission decision/resolution requires exact valid DSS binding and minimum legal retention'; end if;
 return null;
end $$;
revoke all on function public.school_admission_requires_dss_binding() from public;
create constraint trigger school_admission_decision_dss_required after insert on school_admission_decisions deferrable initially deferred for each row execute function public.school_admission_requires_dss_binding();
create constraint trigger school_admission_resolution_dss_required after insert on school_admission_appeal_resolutions deferrable initially deferred for each row execute function public.school_admission_requires_dss_binding();

alter table school_admission_dss_retention_policies enable row level security; alter table school_admission_dss_retention_policies force row level security;
alter table school_admission_signed_artifact_bindings enable row level security; alter table school_admission_signed_artifact_bindings force row level security;
create policy school_admission_dss_tenant_isolation on school_admission_dss_retention_policies using(public.can_bypass_tenant_rls() or (tenant_code=public.current_tenant_code() and institution_id=public.current_institution_id())) with check(public.can_bypass_tenant_rls() or (tenant_code=public.current_tenant_code() and institution_id=public.current_institution_id()));
create policy school_admission_dss_tenant_isolation on school_admission_signed_artifact_bindings using(public.can_bypass_tenant_rls() or (tenant_code=public.current_tenant_code() and institution_id=public.current_institution_id())) with check(public.can_bypass_tenant_rls() or (tenant_code=public.current_tenant_code() and institution_id=public.current_institution_id()));
create trigger school_admission_dss_retention_no_hard_delete before delete on school_admission_dss_retention_policies for each row execute function public.school_operations_no_hard_delete();
create trigger school_admission_dss_binding_no_hard_delete before delete on school_admission_signed_artifact_bindings for each row execute function public.school_operations_no_hard_delete();
create trigger school_admission_dss_binding_entity_version after insert or update or delete on school_admission_signed_artifact_bindings for each row execute function public.record_entity_version();

insert into app_permissions(code,label) values ('education.admissions.retention.manage','Manage admission DSS retention policy') on conflict(code) do update set label=excluded.label;
insert into app_position_permissions(position_code,permission_code) values ('director','education.admissions.retention.manage'),('director_adjunct','education.admissions.retention.manage') on conflict do nothing;

-- 0148 serialized capacity only within one campaign. Capacity belongs to an
-- authorization, shift and school year: allocations from an older school year
-- must not block a new cohort, while sequential rounds in the same school year
-- must not each consume the full authorization.
create or replace function public.school_admission_campaign_guard() returns trigger language plpgsql security definer set search_path=pg_catalog,public as $$
declare authorization_capacity integer; authorization_capacity_unit text; authorization_shift text; context_match boolean; active_allocations integer; annual_claim integer; new_claim integer;
begin
 perform pg_advisory_xact_lock(hashtextextended('school_admission_capacity:'||new.tenant_code||':'||new.institution_id||':'||new.authorization_id::text||':'||new.shift||':'||new.school_year,0));
 if not exists(select 1 from school_regulatory_sources source where source.tenant_code=new.tenant_code and source.institution_id=new.institution_id and source.id=new.source_id and source.status='active' and source.effective_from<=new.opens_on and (source.effective_to is null or source.effective_to>=new.closes_on)) then raise exception using errcode='23514',message='admission campaign requires an active regulatory source covering its window'; end if;
 select a.capacity,a.capacity_unit,a.shift into authorization_capacity,authorization_capacity_unit,authorization_shift from school_offering_authorizations a where a.tenant_code=new.tenant_code and a.institution_id=new.institution_id and a.id=new.authorization_id for update;
 if authorization_capacity is null or authorization_capacity_unit<>new.capacity_unit or authorization_shift<>new.shift or new.capacity_limit>authorization_capacity then raise exception using errcode='23514', message='admission campaign capacity exceeds or mismatches authorization capacity'; end if;
 select exists(select 1 from school_admission_class_offering_contexts c where c.tenant_code=new.tenant_code and c.institution_id=new.institution_id and c.id=new.class_offering_context_id and c.offering_id=new.offering_id and c.location_id=new.location_id and c.authorization_id=new.authorization_id and c.shift=new.shift and c.effective_from<=new.opens_on and (c.effective_to is null or c.effective_to>=new.closes_on)) into context_match;
 if not context_match then raise exception using errcode='23514', message='admission campaign context does not match authorization'; end if;
 if not public.school_admission_authorization_eligible(new.tenant_code,new.institution_id,new.authorization_id,new.opens_on,new.closes_on,new.capacity_unit,new.shift) then raise exception using errcode='23514', message='admission campaign requires eligible provisional or accredited authorization'; end if;
 new_claim:=case when new.capacity_unit='students' then new.student_place_limit else new.capacity_limit end;
 select coalesce(sum(case when campaign.capacity_unit='students' then campaign.student_place_limit else campaign.capacity_limit end),0) into annual_claim from school_admission_campaigns campaign where campaign.tenant_code=new.tenant_code and campaign.institution_id=new.institution_id and campaign.authorization_id=new.authorization_id and campaign.shift=new.shift and campaign.school_year=new.school_year and campaign.status<>'cancelled' and (campaign.status<>'archived' or exists(select 1 from school_admission_capacity_allocations allocation where allocation.tenant_code=campaign.tenant_code and allocation.institution_id=campaign.institution_id and allocation.campaign_id=campaign.id and allocation.status in ('held','consumed'))) and (tg_op='INSERT' or campaign.id<>new.id);
 if annual_claim+new_claim>authorization_capacity then raise exception using errcode='23514',message='school-year admission campaigns exceed authorization capacity'; end if;
 if tg_op='UPDATE' then
  select count(*) into active_allocations from school_admission_capacity_allocations allocation where allocation.tenant_code=new.tenant_code and allocation.institution_id=new.institution_id and allocation.campaign_id=new.id and allocation.status in ('held','consumed');
  if active_allocations>new.student_place_limit then raise exception using errcode='23514', message='admission campaign student place limit cannot be reduced below active allocations'; end if;
 end if;
 return new;
end $$;
revoke all on function public.school_admission_campaign_guard() from public;

create or replace function public.school_admission_capacity_guard() returns trigger language plpgsql security definer set search_path=pg_catalog,public as $$
declare campaign_student_place_limit integer; capacity_school_year text; authorization_capacity integer; authorization_capacity_unit text; authorization_shift text; valid_match boolean; campaign_allocated integer; authorization_allocated integer;
begin
 select c.student_place_limit,c.school_year into campaign_student_place_limit,capacity_school_year from school_admission_campaigns c where c.tenant_code=new.tenant_code and c.institution_id=new.institution_id and c.id=new.campaign_id for update;
 if capacity_school_year is null then raise exception using errcode='23514', message='admission capacity allocation requires a campaign school year'; end if;
 perform pg_advisory_xact_lock(hashtextextended('school_admission_capacity:'||new.tenant_code||':'||new.institution_id||':'||new.authorization_id::text||':'||new.shift||':'||capacity_school_year,0));
 select a.capacity,a.capacity_unit,a.shift into authorization_capacity,authorization_capacity_unit,authorization_shift from school_offering_authorizations a where a.tenant_code=new.tenant_code and a.institution_id=new.institution_id and a.id=new.authorization_id for update;
 select exists(select 1 from school_admission_campaigns c join school_admission_class_offering_contexts x on x.tenant_code=c.tenant_code and x.institution_id=c.institution_id and x.id=new.class_offering_context_id join school_admission_applications app on app.tenant_code=c.tenant_code and app.institution_id=c.institution_id and app.id=new.application_id and app.campaign_id=c.id where c.tenant_code=new.tenant_code and c.institution_id=new.institution_id and c.id=new.campaign_id and c.authorization_id=new.authorization_id and x.authorization_id=new.authorization_id and c.capacity_unit=new.capacity_unit and x.shift=new.shift and c.shift=new.shift and authorization_capacity_unit=new.capacity_unit and authorization_shift=new.shift) into valid_match;
 if campaign_student_place_limit is null or authorization_capacity is null or not valid_match then raise exception using errcode='23514', message='admission capacity allocation has incompatible application, authorization or context'; end if;
 if new.status not in ('held','consumed') then return new; end if;
 select coalesce(sum(a.allocated_capacity),0) into campaign_allocated from school_admission_capacity_allocations a where a.tenant_code=new.tenant_code and a.institution_id=new.institution_id and a.campaign_id=new.campaign_id and a.status in ('held','consumed') and (tg_op='INSERT' or a.id<>new.id);
 if campaign_allocated+new.allocated_capacity>campaign_student_place_limit then raise exception using errcode='23514', message='admission capacity allocation exceeds campaign student place limit'; end if;
 if authorization_capacity_unit='students' then
  select coalesce(sum(a.allocated_capacity),0) into authorization_allocated from school_admission_capacity_allocations a join school_admission_campaigns campaign on campaign.tenant_code=a.tenant_code and campaign.institution_id=a.institution_id and campaign.id=a.campaign_id where a.tenant_code=new.tenant_code and a.institution_id=new.institution_id and a.authorization_id=new.authorization_id and a.shift=new.shift and campaign.school_year=capacity_school_year and a.status in ('held','consumed') and (tg_op='INSERT' or a.id<>new.id);
  if authorization_allocated+new.allocated_capacity>authorization_capacity then raise exception using errcode='23514',message='admission capacity allocation exceeds authorization student capacity'; end if;
 end if;
 return new;
end $$;
revoke all on function public.school_admission_capacity_guard() from public;

create or replace function public.school_admission_authorization_campaign_dependency_guard() returns trigger language plpgsql security definer set search_path=pg_catalog,public as $$
declare invalid_campaign boolean; yearly_claim integer;
begin
 select exists(select 1 from school_admission_campaigns campaign where campaign.tenant_code=new.tenant_code and campaign.institution_id=new.institution_id and campaign.authorization_id=new.id and (new.status not in ('provisional','accredited') or new.capacity is null or new.capacity_unit<>campaign.capacity_unit or new.shift<>campaign.shift or new.effective_from>campaign.opens_on or (new.effective_to is not null and new.effective_to<campaign.closes_on))) into invalid_campaign;
 if invalid_campaign then raise exception using errcode='23514',message='authorization update would invalidate an admission campaign',constraint='school_admission_authorization_campaign_dependency_conflict'; end if;
 select coalesce(max(claim),0) into yearly_claim from (select campaign.school_year,sum(case when campaign.capacity_unit='students' then campaign.student_place_limit else campaign.capacity_limit end) claim from school_admission_campaigns campaign where campaign.tenant_code=new.tenant_code and campaign.institution_id=new.institution_id and campaign.authorization_id=new.id and campaign.status<>'cancelled' and (campaign.status<>'archived' or exists(select 1 from school_admission_capacity_allocations allocation where allocation.tenant_code=campaign.tenant_code and allocation.institution_id=campaign.institution_id and allocation.campaign_id=campaign.id and allocation.status in ('held','consumed'))) group by campaign.school_year) claims;
 -- An authorization with no admission commitments may have no capacity yet.
 -- Preserve the non-null/minimum capacity requirement whenever places are claimed.
 if yearly_claim>0 and (new.capacity is null or yearly_claim>new.capacity) then raise exception using errcode='23514',message='authorization capacity is below school-year admission commitments',constraint='school_admission_authorization_campaign_dependency_conflict'; end if;
 return new;
end $$;
revoke all on function public.school_admission_authorization_campaign_dependency_guard() from public;

-- An appeal cannot make the latest legal outcome non-admitted while an
-- enrolment created from that application is still active. The API rejects
-- this transition early; this deferred guard protects every database writer.
create or replace function public.school_admission_decision_enrolment_guard() returns trigger language plpgsql security definer set search_path=pg_catalog,public as $$
begin
 if new.outcome<>'admitted' and exists(select 1 from education_student_enrolments enrolment where enrolment.tenant_code=new.tenant_code and enrolment.institution_id=new.institution_id and enrolment.admission_application_id=new.application_id and enrolment.status='active') then
  raise exception using errcode='23514',message='non-admitted decision conflicts with active admission enrolment';
 end if;
 return null;
end $$;
revoke all on function public.school_admission_decision_enrolment_guard() from public;
create constraint trigger school_admission_decision_enrolment_consistency after insert on school_admission_decisions deferrable initially deferred for each row execute function public.school_admission_decision_enrolment_guard();
