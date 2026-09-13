-- Stage 1B architecture enrichment. Confessional status is a typed overlay
-- over an exact private profile version, never a third legal form.
select set_config('app.is_super_admin', 'true', true);
alter table school_institution_profiles_v2 add column profile_series_id uuid;
alter table school_institution_profiles_v2 add column supersedes_profile_id uuid;
update school_institution_profiles_v2 set profile_series_id=id where profile_series_id is null;
alter table school_institution_profiles_v2 alter column profile_series_id set not null;
alter table school_institution_profiles_v2 add constraint school_profiles_v2_series_fk foreign key(tenant_code,institution_id,profile_series_id) references school_institution_profiles_v2(tenant_code,institution_id,id) on delete restrict;
alter table school_institution_profiles_v2 add constraint school_profiles_v2_supersedes_fk foreign key(tenant_code,institution_id,supersedes_profile_id) references school_institution_profiles_v2(tenant_code,institution_id,id) on delete restrict;

create table school_confessional_profiles (
 id uuid primary key default gen_random_uuid(), tenant_code text not null, institution_id text not null, profile_id uuid not null, profile_version integer not null, cult_party_id uuid not null, cult_code text not null, protocol_reference text not null, status text not null check(status in ('draft','approved','active','superseded','withdrawn')), effective_from date not null, effective_to date, expected_version integer not null default 1 check(expected_version>0), source_id uuid not null, approved_by_subject text not null default '', approved_at timestamptz, created_by_subject text not null, created_at timestamptz not null default now(), updated_by_subject text not null, updated_at timestamptz not null default now(),
 unique(tenant_code,institution_id,id), foreign key(tenant_code,institution_id) references app_tenants(code,institution_id) on delete restrict, foreign key(tenant_code,institution_id,profile_id,profile_version) references school_institution_profiles_v2(tenant_code,institution_id,id,version) on delete restrict, foreign key(tenant_code,institution_id,cult_party_id) references app_parties(tenant_code,institution_id,id) on delete restrict, foreign key(tenant_code,institution_id,source_id) references school_regulatory_sources(tenant_code,institution_id,id) on delete restrict, check(effective_to is null or effective_to>=effective_from), check(status not in ('approved','active') or (approved_by_subject<>'' and approved_at is not null))
);
create table school_confessional_protocols (
 id uuid primary key default gen_random_uuid(), tenant_code text not null, institution_id text not null, confessional_profile_id uuid not null, protocol_type text not null check(protocol_type in ('governance','curriculum','personnel','facility','other')), reference text not null, status text not null check(status in ('active','superseded','withdrawn')), effective_from date not null, effective_to date, source_id uuid not null, created_by_subject text not null, created_at timestamptz not null default now(), unique(tenant_code,institution_id,id), foreign key(tenant_code,institution_id,confessional_profile_id) references school_confessional_profiles(tenant_code,institution_id,id) on delete restrict, foreign key(tenant_code,institution_id,source_id) references school_regulatory_sources(tenant_code,institution_id,id) on delete restrict, check(effective_to is null or effective_to>=effective_from)
);
create or replace function public.school_confessional_private_only() returns trigger language plpgsql as $$ begin if not exists(select 1 from school_institution_profiles_v2 p where p.tenant_code=new.tenant_code and p.institution_id=new.institution_id and p.id=new.profile_id and p.version=new.profile_version and p.legal_form='private') then raise exception 'confessional overlay requires exact private profile version'; end if; return new; end $$;
create trigger school_confessional_private_only before insert or update on school_confessional_profiles for each row execute function public.school_confessional_private_only();

alter table school_funding_instruments add column offering_id uuid;
alter table school_funding_instruments add column beneficiary_party_id uuid;
alter table school_funding_instruments add column tuition_eligible boolean;
alter table school_funding_instruments add constraint school_funding_offering_fk foreign key(tenant_code,institution_id,offering_id) references school_education_offerings(tenant_code,institution_id,id) on delete restrict;
alter table school_funding_instruments add constraint school_funding_beneficiary_fk foreign key(tenant_code,institution_id,beneficiary_party_id) references app_parties(tenant_code,institution_id,id) on delete restrict;
alter table school_funding_instruments add constraint school_funding_subject_unique unique nulls not distinct(tenant_code,institution_id,code,school_year,offering_id,beneficiary_party_id);
create table school_funding_eligibility_evaluations (
 id uuid primary key default gen_random_uuid(), tenant_code text not null, institution_id text not null, funding_instrument_id uuid not null, evaluated_for_date date not null, status text not null check(status in ('eligible','ineligible','indeterminate')), criteria jsonb not null, evidence jsonb not null, warnings jsonb not null default '[]'::jsonb, checksum_sha256 text not null check(checksum_sha256 ~ '^[a-f0-9]{64}$'), evaluated_by_subject text not null, evaluated_at timestamptz not null default now(), unique(tenant_code,institution_id,id), foreign key(tenant_code,institution_id,funding_instrument_id) references school_funding_instruments(tenant_code,institution_id,id) on delete restrict, check(jsonb_typeof(criteria)='array' and jsonb_typeof(evidence)='array' and jsonb_typeof(warnings)='array')
);

alter table school_procurement_applicability_assessments add column subject_type text not null default 'entity' check(subject_type in ('entity','contract','project'));
alter table school_procurement_applicability_assessments add column subject_reference text not null default '';
alter table school_procurement_applicability_assessments add column criteria jsonb not null default '[]'::jsonb;
alter table school_procurement_applicability_assessments add column evidence jsonb not null default '[]'::jsonb;
alter table school_institution_profiles_v2 add constraint school_profiles_v2_effective_excl exclude using gist(tenant_code with =,institution_id with =,daterange(effective_from,coalesce(effective_to,'infinity'::date),'[]') with &&) where(status in ('approved','active'));
alter table school_offering_authorizations add constraint school_offering_authorization_effective_excl exclude using gist(tenant_code with =,institution_id with =,offering_id with =,location_id with =,daterange(effective_from,coalesce(effective_to,'infinity'::date),'[]') with &&);
alter table school_procurement_applicability_assessments add constraint school_procurement_effective_excl exclude using gist(tenant_code with =,institution_id with =,subject_type with =,subject_reference with =,daterange(effective_from,coalesce(effective_to,'infinity'::date),'[]') with &&);

alter table school_operation_policy_inputs add column revalidates_evaluation_id uuid;
alter table school_operation_policy_inputs add column profile_series_id uuid;
alter table school_operation_policy_inputs add column funding_eligibility_evaluation_id uuid;
update school_operation_policy_inputs i set profile_series_id=p.profile_series_id from school_institution_profiles_v2 p where p.tenant_code=i.tenant_code and p.institution_id=i.institution_id and p.id=i.profile_id and p.version=i.profile_version and i.profile_series_id is null;
alter table school_operation_policy_inputs alter column profile_series_id set not null;
create or replace function public.school_policy_input_bind_profile_series() returns trigger language plpgsql as $$
begin
 select p.profile_series_id into new.profile_series_id
 from school_institution_profiles_v2 p
 where p.tenant_code=new.tenant_code and p.institution_id=new.institution_id and p.id=new.profile_id and p.version=new.profile_version;
 if new.profile_series_id is null then
  raise exception using errcode='23503', message='policy input requires an exact profile version', constraint='school_policy_input_exact_profile_version';
 end if;
 return new;
end $$;
create trigger school_policy_input_bind_profile_series before insert on school_operation_policy_inputs for each row execute function public.school_policy_input_bind_profile_series();
alter table school_operation_policy_inputs add constraint school_policy_inputs_revalidation_fk foreign key(tenant_code,institution_id,revalidates_evaluation_id) references school_operation_policy_evaluations_v2(tenant_code,institution_id,id) on delete restrict;
alter table school_operation_policy_inputs add constraint school_policy_inputs_series_fk foreign key(tenant_code,institution_id,profile_series_id) references school_institution_profiles_v2(tenant_code,institution_id,id) on delete restrict;
alter table school_operation_policy_inputs add constraint school_policy_inputs_funding_evaluation_fk foreign key(tenant_code,institution_id,funding_eligibility_evaluation_id) references school_funding_eligibility_evaluations(tenant_code,institution_id,id) on delete restrict;

do $$ declare t text; begin foreach t in array array['school_confessional_profiles','school_confessional_protocols','school_funding_eligibility_evaluations'] loop execute format('alter table %I enable row level security',t); execute format('alter table %I force row level security',t); execute format('create policy tenant_isolation on %I using (public.can_bypass_tenant_rls() or (tenant_code=public.current_tenant_code() and institution_id=public.current_institution_id())) with check (public.can_bypass_tenant_rls() or (tenant_code=public.current_tenant_code() and institution_id=public.current_institution_id()))',t); end loop; end $$;
create trigger school_funding_eligibility_immutable before update or delete on school_funding_eligibility_evaluations for each row execute function public.school_stage1b_snapshot_immutable();
