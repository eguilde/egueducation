-- Admission foundation.  This is schema-only: runtime/API cutover remains
-- explicitly disabled until the policy-v2 and admission workflows are proven.
-- Every aggregate is scoped by the exact tenant+institution pair.
select set_config('app.is_super_admin', 'true', true);
create extension if not exists btree_gist;

insert into app_permissions(code,label) values
 ('education.admissions.read','Read admissions campaigns, applications and decisions'),
 ('education.admissions.manage','Manage admissions campaigns, criteria, documents and applications'),
 ('education.admissions.decide','Issue admission decisions and capacity allocations'),
 ('education.admissions.appeals.manage','Manage admission appeals and resolutions'),
 ('education.admissions.exports.generate','Generate sealed admission export manifests')
on conflict(code) do update set label=excluded.label;
insert into app_role_permissions(role_code,permission_code) values
 ('super_admin','education.admissions.read'),('super_admin','education.admissions.manage'),('super_admin','education.admissions.decide'),('super_admin','education.admissions.appeals.manage'),('super_admin','education.admissions.exports.generate'),
 ('admin','education.admissions.read'),('admin','education.admissions.manage'),('admin','education.admissions.decide'),('admin','education.admissions.appeals.manage'),('admin','education.admissions.exports.generate'),
 ('director','education.admissions.read'),('director','education.admissions.manage'),('director','education.admissions.decide'),('director','education.admissions.appeals.manage'),('director','education.admissions.exports.generate'),
 ('secretar','education.admissions.read'),('secretar','education.admissions.manage'),('secretar','education.admissions.appeals.manage'),
 ('inspector','education.admissions.read')
on conflict do nothing;

-- The old authorization capacity had no declared unit or shift.  The default
-- preserves existing numeric values as student places; future API work must
-- never infer either dimension from a free-text campaign label.
alter table school_offering_authorizations
 add column capacity_unit text not null default 'students',
 add column shift text not null default 'day';
alter table school_offering_authorizations
 add constraint school_offering_authorizations_capacity_unit_check check (capacity_unit in ('students','study_groups')),
 add constraint school_offering_authorizations_shift_check check (shift in ('day','afternoon','evening'));

-- Earlier education aggregates predate composite scope keys.  Add them before
-- any admission FK so a student, class or enrolment cannot be rebound across
-- tenant/institution boundaries.
alter table education_school_classes add constraint education_school_classes_scope_id_unique unique (tenant_code,institution_id,id);
alter table education_students add constraint education_students_scope_id_unique unique (tenant_code,institution_id,id);
alter table education_student_enrolments add constraint education_student_enrolments_scope_id_unique unique (tenant_code,institution_id,id);
alter table education_students add column party_id uuid;
alter table education_students add constraint education_students_party_scope_fk foreign key(tenant_code,institution_id,party_id) references app_parties(tenant_code,institution_id,id) on delete restrict;
alter table education_students add constraint education_students_party_scope_unique unique(tenant_code,institution_id,party_id);
create or replace function public.school_admission_student_party_guard() returns trigger language plpgsql security definer set search_path=pg_catalog,public as $$
begin
 if new.party_id is not null and not exists(select 1 from app_parties party where party.tenant_code=new.tenant_code and party.institution_id=new.institution_id and party.id=new.party_id and party.party_type='physical') then raise exception using errcode='23514',message='student party must be an in-scope physical person'; end if;
 return new;
end $$;
revoke all on function public.school_admission_student_party_guard() from public;
create trigger school_admission_student_party_guard before insert or update of tenant_code,institution_id,party_id on education_students for each row execute function public.school_admission_student_party_guard();

create table school_admission_class_offering_contexts (
 id uuid primary key default gen_random_uuid(), tenant_code text not null, institution_id text not null,
 class_id uuid not null, offering_id uuid not null, location_id uuid not null, authorization_id uuid not null,
 school_year text not null, shift text not null check (shift in ('day','afternoon','evening')),
 active boolean not null default true, effective_from date not null, effective_to date,
 expected_version integer not null default 1 check(expected_version>0), created_by_subject text not null, created_at timestamptz not null default now(), updated_by_subject text not null, updated_at timestamptz not null default now(),
 unique(tenant_code,institution_id,id), unique(tenant_code,institution_id,class_id,school_year,effective_from),
 foreign key(tenant_code,institution_id) references app_tenants(code,institution_id) on delete restrict,
 foreign key(tenant_code,institution_id,class_id) references education_school_classes(tenant_code,institution_id,id) on delete restrict,
 foreign key(tenant_code,institution_id,offering_id) references school_education_offerings(tenant_code,institution_id,id) on delete restrict,
 foreign key(tenant_code,institution_id,location_id) references school_locations(tenant_code,institution_id,id) on delete restrict,
 foreign key(tenant_code,institution_id,authorization_id) references school_offering_authorizations(tenant_code,institution_id,id) on delete restrict,
 check(effective_to is null or effective_to>=effective_from)
);

create table school_admission_campaigns (
 id uuid primary key default gen_random_uuid(), tenant_code text not null, institution_id text not null,
 source_id uuid not null, code text not null, title text not null, school_year text not null,
 offering_id uuid not null, location_id uuid not null, authorization_id uuid not null, class_offering_context_id uuid not null,
 capacity_limit integer not null check(capacity_limit>0), capacity_unit text not null check(capacity_unit in ('students','study_groups')), student_place_limit integer not null check(student_place_limit>0), capacity_basis jsonb not null default '{}'::jsonb, shift text not null check(shift in ('day','afternoon','evening')),
 opens_on date not null, closes_on date not null, decision_due_on date, status text not null default 'draft' check(status in ('draft','published','open','closed','cancelled','archived')),
 expected_version integer not null default 1 check(expected_version>0), created_by_subject text not null, created_at timestamptz not null default now(), updated_by_subject text not null, updated_at timestamptz not null default now(),
 unique(tenant_code,institution_id,id), unique(tenant_code,institution_id,code),
 foreign key(tenant_code,institution_id) references app_tenants(code,institution_id) on delete restrict,
 foreign key(tenant_code,institution_id,source_id) references school_regulatory_sources(tenant_code,institution_id,id) on delete restrict,
 foreign key(tenant_code,institution_id,offering_id) references school_education_offerings(tenant_code,institution_id,id) on delete restrict,
 foreign key(tenant_code,institution_id,location_id) references school_locations(tenant_code,institution_id,id) on delete restrict,
 foreign key(tenant_code,institution_id,authorization_id) references school_offering_authorizations(tenant_code,institution_id,id) on delete restrict,
 foreign key(tenant_code,institution_id,class_offering_context_id) references school_admission_class_offering_contexts(tenant_code,institution_id,id) on delete restrict,
 check(closes_on>=opens_on), check(decision_due_on is null or decision_due_on>=closes_on),
 check(jsonb_typeof(capacity_basis)='object'),
 check((capacity_unit='students' and student_place_limit=capacity_limit) or (capacity_unit='study_groups' and capacity_basis ? 'students_per_group' and jsonb_typeof(capacity_basis->'students_per_group')='number' and (capacity_basis->>'students_per_group') ~ '^[1-9][0-9]*$' and student_place_limit<=capacity_limit*case when (capacity_basis->>'students_per_group') ~ '^[1-9][0-9]*$' then (capacity_basis->>'students_per_group')::integer else 0 end))
);

create table school_admission_criteria (
 id uuid primary key default gen_random_uuid(), tenant_code text not null, institution_id text not null, campaign_id uuid not null,
 code text not null, title text not null, criterion_kind text not null check(criterion_kind in ('eligibility','priority','ranking','tie_breaker')),
 required boolean not null default true, weight numeric(12,4) not null default 0 check(weight>=0), ordinal integer not null check(ordinal>0), rule_snapshot jsonb not null default '{}'::jsonb,
 expected_version integer not null default 1 check(expected_version>0), created_by_subject text not null, created_at timestamptz not null default now(), updated_by_subject text not null, updated_at timestamptz not null default now(),
 unique(tenant_code,institution_id,id), unique(tenant_code,institution_id,campaign_id,code), unique(tenant_code,institution_id,campaign_id,ordinal),
 foreign key(tenant_code,institution_id) references app_tenants(code,institution_id) on delete restrict,
 foreign key(tenant_code,institution_id,campaign_id) references school_admission_campaigns(tenant_code,institution_id,id) on delete restrict,
 check(jsonb_typeof(rule_snapshot)='object')
);

create table school_admission_document_requirements (
 id uuid primary key default gen_random_uuid(), tenant_code text not null, institution_id text not null, campaign_id uuid not null,
 code text not null, title text not null, required boolean not null default true, allowed_mime_types text[] not null default '{}', ordinal integer not null check(ordinal>0),
 expected_version integer not null default 1 check(expected_version>0), created_by_subject text not null, created_at timestamptz not null default now(), updated_by_subject text not null, updated_at timestamptz not null default now(),
 unique(tenant_code,institution_id,id), unique(tenant_code,institution_id,campaign_id,code), unique(tenant_code,institution_id,campaign_id,ordinal),
 foreign key(tenant_code,institution_id) references app_tenants(code,institution_id) on delete restrict,
 foreign key(tenant_code,institution_id,campaign_id) references school_admission_campaigns(tenant_code,institution_id,id) on delete restrict,
 check(array_position(allowed_mime_types,'') is null)
);

create table school_admission_applications (
 id uuid primary key default gen_random_uuid(), tenant_code text not null, institution_id text not null, campaign_id uuid not null,
 application_no text not null, candidate_party_id uuid not null, student_id uuid, submitted_at timestamptz, status text not null default 'draft' check(status in ('draft','submitted','under_review','waitlisted','admitted','rejected','withdrawn','cancelled')),
 consent_snapshot jsonb not null default '{}'::jsonb, expected_version integer not null default 1 check(expected_version>0),
 created_by_subject text not null, created_at timestamptz not null default now(), updated_by_subject text not null, updated_at timestamptz not null default now(),
 unique(tenant_code,institution_id,id), unique(tenant_code,institution_id,campaign_id,application_no),
 foreign key(tenant_code,institution_id) references app_tenants(code,institution_id) on delete restrict,
 foreign key(tenant_code,institution_id,campaign_id) references school_admission_campaigns(tenant_code,institution_id,id) on delete restrict,
 foreign key(tenant_code,institution_id,candidate_party_id) references app_parties(tenant_code,institution_id,id) on delete restrict,
 foreign key(tenant_code,institution_id,student_id) references education_students(tenant_code,institution_id,id) on delete restrict,
 check(jsonb_typeof(consent_snapshot)='object'), check(status<>'submitted' or submitted_at is not null)
);

create table school_admission_capacity_allocations (
 id uuid primary key default gen_random_uuid(), tenant_code text not null, institution_id text not null,
 application_id uuid not null, campaign_id uuid not null, class_offering_context_id uuid not null, authorization_id uuid not null,
 allocated_capacity integer not null default 1 check(allocated_capacity=1), capacity_unit text not null check(capacity_unit in ('students','study_groups')), shift text not null check(shift in ('day','afternoon','evening')),
 status text not null default 'held' check(status in ('held','consumed','released','expired')),
 expected_version integer not null default 1 check(expected_version>0), created_by_subject text not null, created_at timestamptz not null default now(), updated_by_subject text not null, updated_at timestamptz not null default now(),
 unique(tenant_code,institution_id,id),
 foreign key(tenant_code,institution_id) references app_tenants(code,institution_id) on delete restrict,
 foreign key(tenant_code,institution_id,application_id) references school_admission_applications(tenant_code,institution_id,id) on delete restrict,
 foreign key(tenant_code,institution_id,campaign_id) references school_admission_campaigns(tenant_code,institution_id,id) on delete restrict,
 foreign key(tenant_code,institution_id,class_offering_context_id) references school_admission_class_offering_contexts(tenant_code,institution_id,id) on delete restrict,
 foreign key(tenant_code,institution_id,authorization_id) references school_offering_authorizations(tenant_code,institution_id,id) on delete restrict
);
create unique index school_admission_capacity_allocations_one_active_application on school_admission_capacity_allocations(tenant_code,institution_id,application_id) where status in ('held','consumed');

-- Historical enrolments remain nullable. Admission commands introduced after
-- this expand migration must set the exact application reference.
alter table education_student_enrolments add column admission_application_id uuid;
alter table education_student_enrolments add constraint education_student_enrolments_admission_application_scope_fk foreign key(tenant_code,institution_id,admission_application_id) references school_admission_applications(tenant_code,institution_id,id) on delete restrict;

create table school_admission_application_representatives (
 id uuid primary key default gen_random_uuid(), tenant_code text not null, institution_id text not null, application_id uuid not null, representative_party_id uuid not null,
 relationship_type text not null check(relationship_type in ('parent','guardian','legal_representative','proxy')), authority_reference text not null default '', primary_contact boolean not null default false,
 expected_version integer not null default 1 check(expected_version>0), created_by_subject text not null, created_at timestamptz not null default now(), updated_by_subject text not null, updated_at timestamptz not null default now(),
 unique(tenant_code,institution_id,id), unique(tenant_code,institution_id,application_id,representative_party_id),
 foreign key(tenant_code,institution_id) references app_tenants(code,institution_id) on delete restrict,
 foreign key(tenant_code,institution_id,application_id) references school_admission_applications(tenant_code,institution_id,id) on delete restrict,
 foreign key(tenant_code,institution_id,representative_party_id) references app_parties(tenant_code,institution_id,id) on delete restrict
);

create table school_admission_application_documents (
 id uuid primary key default gen_random_uuid(), tenant_code text not null, institution_id text not null, application_id uuid not null, document_requirement_id uuid,
 document_kind text not null, status text not null default 'requested' check(status in ('requested','submitted','accepted','rejected','waived','withdrawn')),
 archive_document_id uuid, archive_version_id uuid, archive_version_no integer, archive_source_bucket text not null default '', archive_source_object_key text not null default '', archive_source_object_version_id text not null default '', archive_retention_until timestamptz, archive_sha256 text not null default '',
 reviewed_at timestamptz, reviewed_by_subject text not null default '', review_note text not null default '', expected_version integer not null default 1 check(expected_version>0),
 created_by_subject text not null, created_at timestamptz not null default now(), updated_by_subject text not null, updated_at timestamptz not null default now(),
 unique(tenant_code,institution_id,id),
 foreign key(tenant_code,institution_id) references app_tenants(code,institution_id) on delete restrict,
 foreign key(tenant_code,institution_id,application_id) references school_admission_applications(tenant_code,institution_id,id) on delete restrict,
 foreign key(tenant_code,institution_id,document_requirement_id) references school_admission_document_requirements(tenant_code,institution_id,id) on delete restrict,
 foreign key(institution_id,archive_document_id) references archive_documents(institution_id,id) on delete restrict,
 foreign key(institution_id,archive_document_id,archive_version_id) references archive_document_versions(institution_id,document_id,id) on delete restrict,
 check((status in ('requested','waived')) or (archive_document_id is not null and archive_version_id is not null and archive_version_no is not null and archive_version_no>0 and btrim(archive_source_bucket)<>'' and btrim(archive_source_object_key)<>'' and btrim(archive_source_object_version_id)<>'' and archive_retention_until is not null and archive_sha256 ~ '^[0-9a-f]{64}$')),
 check((archive_document_id is null and archive_version_id is null and archive_version_no is null and archive_source_bucket='' and archive_source_object_key='' and archive_source_object_version_id='' and archive_retention_until is null and archive_sha256='') or (archive_document_id is not null and archive_version_id is not null and archive_version_no is not null))
);

create table school_admission_criterion_assessments (
 id uuid primary key default gen_random_uuid(), tenant_code text not null, institution_id text not null, application_id uuid not null, criterion_id uuid not null,
 outcome text not null check(outcome in ('pending','met','not_met','not_applicable','indeterminate')), score numeric(12,4), rationale text not null default '', evidence_snapshot jsonb not null default '{}'::jsonb,
 assessed_by_subject text not null default '', assessed_at timestamptz, expected_version integer not null default 1 check(expected_version>0), created_by_subject text not null, created_at timestamptz not null default now(), updated_by_subject text not null, updated_at timestamptz not null default now(),
 unique(tenant_code,institution_id,id), unique(tenant_code,institution_id,application_id,criterion_id),
 foreign key(tenant_code,institution_id) references app_tenants(code,institution_id) on delete restrict,
 foreign key(tenant_code,institution_id,application_id) references school_admission_applications(tenant_code,institution_id,id) on delete restrict,
 foreign key(tenant_code,institution_id,criterion_id) references school_admission_criteria(tenant_code,institution_id,id) on delete restrict,
 check(jsonb_typeof(evidence_snapshot)='object'), check((outcome='pending' and assessed_at is null) or (outcome<>'pending' and assessed_at is not null and assessed_by_subject<>''))
);

create table school_admission_decisions (
 id uuid primary key default gen_random_uuid(), tenant_code text not null, institution_id text not null, application_id uuid not null, capacity_allocation_id uuid, enrolment_id uuid, policy_evaluation_v2_id uuid not null,
 decision_no text not null, outcome text not null check(outcome in ('admitted','waitlisted','rejected','withdrawn','cancelled')), rationale text not null, ranking_value numeric(16,4), appeal_deadline date, decided_at timestamptz not null default now(), decided_by_subject text not null,
 supersedes_decision_id uuid, decision_snapshot jsonb not null default '{}'::jsonb, archive_document_id uuid not null, archive_version_id uuid not null, archive_version_no integer not null check(archive_version_no>0), archive_source_bucket text not null, archive_source_object_key text not null, archive_source_object_version_id text not null, archive_retention_until timestamptz not null, archive_sha256 text not null check(archive_sha256 ~ '^[0-9a-f]{64}$'), created_at timestamptz not null default now(),
 unique(tenant_code,institution_id,id), unique(tenant_code,institution_id,decision_no), unique(tenant_code,institution_id,policy_evaluation_v2_id),
 foreign key(tenant_code,institution_id) references app_tenants(code,institution_id) on delete restrict,
 foreign key(tenant_code,institution_id,application_id) references school_admission_applications(tenant_code,institution_id,id) on delete restrict,
 foreign key(tenant_code,institution_id,capacity_allocation_id) references school_admission_capacity_allocations(tenant_code,institution_id,id) on delete restrict,
 foreign key(tenant_code,institution_id,enrolment_id) references education_student_enrolments(tenant_code,institution_id,id) on delete restrict,
 foreign key(tenant_code,institution_id,policy_evaluation_v2_id) references school_operation_policy_evaluations_v2(tenant_code,institution_id,id) on delete restrict,
 foreign key(tenant_code,institution_id,supersedes_decision_id) references school_admission_decisions(tenant_code,institution_id,id) on delete restrict,
 foreign key(institution_id,archive_document_id) references archive_documents(institution_id,id) on delete restrict,
 foreign key(institution_id,archive_document_id,archive_version_id) references archive_document_versions(institution_id,document_id,id) on delete restrict,
 check(jsonb_typeof(decision_snapshot)='object'), check((outcome='admitted' and capacity_allocation_id is not null) or outcome<>'admitted')
);

create table school_admission_decision_deliveries (
 id uuid primary key default gen_random_uuid(), tenant_code text not null, institution_id text not null, decision_id uuid not null,
 channel text not null check(channel in ('portal','email','sms','postal','in_person')), status text not null check(status in ('queued','sent','delivered','failed','acknowledged')),
 destination_snapshot text not null default '', provider_reference text not null default '', sent_at timestamptz, delivered_at timestamptz, delivery_snapshot jsonb not null default '{}'::jsonb,
 created_by_subject text not null, created_at timestamptz not null default now(),
 unique(tenant_code,institution_id,id),
 foreign key(tenant_code,institution_id) references app_tenants(code,institution_id) on delete restrict,
 foreign key(tenant_code,institution_id,decision_id) references school_admission_decisions(tenant_code,institution_id,id) on delete restrict,
 check(jsonb_typeof(delivery_snapshot)='object'), check((status in ('sent','delivered','acknowledged')) = (sent_at is not null)), check(status not in ('delivered','acknowledged') or delivered_at is not null)
);

create table school_admission_appeals (
 id uuid primary key default gen_random_uuid(), tenant_code text not null, institution_id text not null, application_id uuid not null, decision_id uuid not null,
 appeal_no text not null, status text not null default 'draft' check(status in ('draft','submitted','under_review','resolved','withdrawn','dismissed','rejected_late')), submitted_at timestamptz, submitted_by_party_id uuid, expected_version integer not null default 1 check(expected_version>0),
 created_by_subject text not null, created_at timestamptz not null default now(), updated_by_subject text not null, updated_at timestamptz not null default now(),
 unique(tenant_code,institution_id,id), unique(tenant_code,institution_id,appeal_no),
 foreign key(tenant_code,institution_id) references app_tenants(code,institution_id) on delete restrict,
 foreign key(tenant_code,institution_id,application_id) references school_admission_applications(tenant_code,institution_id,id) on delete restrict,
 foreign key(tenant_code,institution_id,decision_id) references school_admission_decisions(tenant_code,institution_id,id) on delete restrict,
 foreign key(tenant_code,institution_id,submitted_by_party_id) references app_parties(tenant_code,institution_id,id) on delete restrict,
 check(status<>'submitted' or (submitted_at is not null and submitted_by_party_id is not null))
);

create table school_admission_appeal_submissions (
 id uuid primary key default gen_random_uuid(), tenant_code text not null, institution_id text not null, appeal_id uuid not null,
 submission_no integer not null check(submission_no>0), submitted_at timestamptz not null default now(), submitted_by_party_id uuid, statement text not null,
 archive_document_id uuid, archive_version_id uuid, archive_version_no integer, archive_source_bucket text not null default '', archive_source_object_key text not null default '', archive_source_object_version_id text not null default '', archive_retention_until timestamptz, archive_sha256 text not null default '',
 created_by_subject text not null, created_at timestamptz not null default now(),
 unique(tenant_code,institution_id,id), unique(tenant_code,institution_id,appeal_id,submission_no),
 foreign key(tenant_code,institution_id) references app_tenants(code,institution_id) on delete restrict,
 foreign key(tenant_code,institution_id,appeal_id) references school_admission_appeals(tenant_code,institution_id,id) on delete restrict,
 foreign key(tenant_code,institution_id,submitted_by_party_id) references app_parties(tenant_code,institution_id,id) on delete restrict,
 foreign key(institution_id,archive_document_id) references archive_documents(institution_id,id) on delete restrict,
 foreign key(institution_id,archive_document_id,archive_version_id) references archive_document_versions(institution_id,document_id,id) on delete restrict,
 check((archive_document_id is null and archive_version_id is null and archive_version_no is null and archive_source_bucket='' and archive_source_object_key='' and archive_source_object_version_id='' and archive_retention_until is null and archive_sha256='') or (archive_document_id is not null and archive_version_id is not null and archive_version_no is not null and archive_version_no>0 and archive_source_bucket<>'' and archive_source_object_key<>'' and archive_source_object_version_id<>'' and archive_retention_until is not null and archive_sha256 ~ '^[0-9a-f]{64}$'))
);

create table school_admission_appeal_resolutions (
 id uuid primary key default gen_random_uuid(), tenant_code text not null, institution_id text not null, appeal_id uuid not null, resulting_decision_id uuid, resulting_outcome text check(resulting_outcome in ('admitted','waitlisted','rejected','withdrawn','cancelled')), capacity_allocation_id uuid, policy_evaluation_v2_id uuid not null,
 outcome text not null check(outcome in ('upheld','partially_upheld','dismissed','withdrawn')), rationale text not null, resolved_at timestamptz not null default now(), resolved_by_subject text not null, resolution_snapshot jsonb not null default '{}'::jsonb, archive_document_id uuid not null, archive_version_id uuid not null, archive_version_no integer not null check(archive_version_no>0), archive_source_bucket text not null, archive_source_object_key text not null, archive_source_object_version_id text not null, archive_retention_until timestamptz not null, archive_sha256 text not null check(archive_sha256 ~ '^[0-9a-f]{64}$'), created_at timestamptz not null default now(),
 unique(tenant_code,institution_id,id), unique(tenant_code,institution_id,appeal_id), unique(tenant_code,institution_id,policy_evaluation_v2_id),
 foreign key(tenant_code,institution_id) references app_tenants(code,institution_id) on delete restrict,
 foreign key(tenant_code,institution_id,appeal_id) references school_admission_appeals(tenant_code,institution_id,id) on delete restrict,
 foreign key(tenant_code,institution_id,resulting_decision_id) references school_admission_decisions(tenant_code,institution_id,id) on delete restrict,
 foreign key(tenant_code,institution_id,capacity_allocation_id) references school_admission_capacity_allocations(tenant_code,institution_id,id) on delete restrict,
 foreign key(tenant_code,institution_id,policy_evaluation_v2_id) references school_operation_policy_evaluations_v2(tenant_code,institution_id,id) on delete restrict,
 foreign key(institution_id,archive_document_id) references archive_documents(institution_id,id) on delete restrict,
 foreign key(institution_id,archive_document_id,archive_version_id) references archive_document_versions(institution_id,document_id,id) on delete restrict,
 check(jsonb_typeof(resolution_snapshot)='object'), check((resulting_outcome='admitted') = (capacity_allocation_id is not null)), check(resulting_outcome is not null)
);

create table school_admission_export_manifests (
 id uuid primary key default gen_random_uuid(), tenant_code text not null, institution_id text not null, campaign_id uuid, application_id uuid,
 manifest_version text not null check(btrim(manifest_version)<>''), manifest_sha256 text not null check(manifest_sha256 ~ '^[0-9a-f]{64}$'), manifest jsonb not null check(jsonb_typeof(manifest)='object'), archive_document_id uuid not null, archive_version_id uuid not null, archive_version_no integer not null check(archive_version_no>0), archive_source_bucket text not null, archive_source_object_key text not null, archive_source_object_version_id text not null, archive_retention_until timestamptz not null, archive_sha256 text not null check(archive_sha256 ~ '^[0-9a-f]{64}$'),
 generated_by_subject text not null, generated_at timestamptz not null default now(),
 unique(tenant_code,institution_id,id), unique(tenant_code,institution_id,manifest_sha256),
 foreign key(tenant_code,institution_id) references app_tenants(code,institution_id) on delete restrict,
 foreign key(tenant_code,institution_id,campaign_id) references school_admission_campaigns(tenant_code,institution_id,id) on delete restrict,
 foreign key(tenant_code,institution_id,application_id) references school_admission_applications(tenant_code,institution_id,id) on delete restrict,
 foreign key(institution_id,archive_document_id) references archive_documents(institution_id,id) on delete restrict,
 foreign key(institution_id,archive_document_id,archive_version_id) references archive_document_versions(institution_id,document_id,id) on delete restrict,
 check(campaign_id is not null or application_id is not null)
);

create table school_admission_idempotency (
 id uuid primary key default gen_random_uuid(), tenant_code text not null, institution_id text not null,
 actor_subject text not null, operation_code text not null, idempotency_key text not null, request_fingerprint text not null check(request_fingerprint ~ '^[0-9a-f]{64}$'), response_status integer, response_snapshot jsonb not null default '{}'::jsonb,
 created_at timestamptz not null default now(), expires_at timestamptz,
 unique(tenant_code,institution_id,id), unique(tenant_code,institution_id,actor_subject,operation_code,idempotency_key),
 foreign key(tenant_code,institution_id) references app_tenants(code,institution_id) on delete restrict,
 check(jsonb_typeof(response_snapshot)='object')
);

create table school_admission_outbox (
 id uuid primary key default gen_random_uuid(), tenant_code text not null, institution_id text not null,
 aggregate_type text not null, aggregate_id uuid not null, event_type text not null, payload jsonb not null default '{}'::jsonb, occurred_by_subject text not null, occurred_at timestamptz not null default now(),
 delivery_status text not null default 'pending' check(delivery_status in ('pending','delivered','failed','dead_letter')), delivered_at timestamptz, attempt_count integer not null default 0 check(attempt_count>=0), last_error text not null default '',
 unique(tenant_code,institution_id,id),
 foreign key(tenant_code,institution_id) references app_tenants(code,institution_id) on delete restrict,
 check(jsonb_typeof(payload)='object'), check((delivery_status='delivered') = (delivered_at is not null))
);

-- Only an authorization with a defined legal interval, a positive bounded
-- capacity and an explicit canonical status can create admission places.
-- Law 198/2023 art.237(1): provisional and accredited are positive statuses.
-- The legacy/ambiguous 'authorized' status is deliberately fail-closed until a
-- source-document migration maps it to a canonical statutory status.
create or replace function public.school_admission_authorization_eligible(
 p_tenant_code text, p_institution_id text, p_authorization_id uuid, p_from date, p_to date, p_capacity_unit text, p_shift text
) returns boolean language sql stable security definer set search_path=pg_catalog,public as $$
 select exists(
  select 1 from public.school_offering_authorizations a
  join public.school_education_offerings o on o.tenant_code=a.tenant_code and o.institution_id=a.institution_id and o.id=a.offering_id
  join public.school_locations l on l.tenant_code=a.tenant_code and l.institution_id=a.institution_id and l.id=a.location_id
  where a.tenant_code=p_tenant_code and a.institution_id=p_institution_id and a.id=p_authorization_id
   and a.status in ('provisional', 'accredited')
   and a.capacity is not null and a.capacity>0 and a.capacity_unit=p_capacity_unit and a.shift=p_shift
   and a.effective_from<=p_from and (a.effective_to is null or a.effective_to>=p_to)
   and o.effective_from<=p_from and (o.effective_to is null or o.effective_to>=p_to)
   and l.effective_from<=p_from and (l.effective_to is null or l.effective_to>=p_to)
 );
$$;
revoke all on function public.school_admission_authorization_eligible(text,text,uuid,date,date,text,text) from public;

create or replace function public.school_admission_class_context_guard() returns trigger language plpgsql security definer set search_path=pg_catalog,public as $$
begin
 if not exists(select 1 from school_offering_authorizations a join education_school_classes class_row on class_row.tenant_code=new.tenant_code and class_row.institution_id=new.institution_id and class_row.id=new.class_id and class_row.school_year=new.school_year where a.tenant_code=new.tenant_code and a.institution_id=new.institution_id and a.id=new.authorization_id and a.offering_id=new.offering_id and a.location_id=new.location_id and a.shift=new.shift and a.effective_from<=new.effective_from and (a.effective_to is null or (new.effective_to is not null and a.effective_to>=new.effective_to))) then
  raise exception using errcode='23514', message='class offering context is outside authorization window or scope';
 end if;
 return new;
end $$;
revoke all on function public.school_admission_class_context_guard() from public;
create trigger school_admission_class_context_authorization_guard before insert or update of offering_id,location_id,authorization_id,shift,effective_from,effective_to on school_admission_class_offering_contexts for each row execute function public.school_admission_class_context_guard();

create or replace function public.school_admission_campaign_guard() returns trigger language plpgsql security definer set search_path=pg_catalog,public as $$
declare authorization_capacity integer; context_match boolean; active_allocations integer;
begin
 if not exists(select 1 from school_regulatory_sources source where source.tenant_code=new.tenant_code and source.institution_id=new.institution_id and source.id=new.source_id and source.status='active' and source.effective_from<=new.opens_on and (source.effective_to is null or source.effective_to>=new.closes_on)) then raise exception using errcode='23514',message='admission campaign requires an active regulatory source covering its window'; end if;
 select a.capacity into authorization_capacity from school_offering_authorizations a where a.tenant_code=new.tenant_code and a.institution_id=new.institution_id and a.id=new.authorization_id for update;
 if authorization_capacity is null or new.capacity_limit>authorization_capacity then raise exception using errcode='23514', message='admission campaign capacity exceeds authorization capacity'; end if;
 select exists(select 1 from school_admission_class_offering_contexts c where c.tenant_code=new.tenant_code and c.institution_id=new.institution_id and c.id=new.class_offering_context_id and c.offering_id=new.offering_id and c.location_id=new.location_id and c.authorization_id=new.authorization_id and c.shift=new.shift and c.effective_from<=new.opens_on and (c.effective_to is null or c.effective_to>=new.closes_on)) into context_match;
 if not context_match then raise exception using errcode='23514', message='admission campaign context does not match authorization'; end if;
 if not public.school_admission_authorization_eligible(new.tenant_code,new.institution_id,new.authorization_id,new.opens_on,new.closes_on,new.capacity_unit,new.shift) then raise exception using errcode='23514', message='admission campaign requires eligible provisional or accredited authorization'; end if;
 if tg_op='UPDATE' then
  select count(*) into active_allocations from school_admission_capacity_allocations allocation where allocation.tenant_code=new.tenant_code and allocation.institution_id=new.institution_id and allocation.campaign_id=new.id and allocation.status in ('held','consumed');
  if active_allocations>new.student_place_limit then raise exception using errcode='23514', message='admission campaign student place limit cannot be reduced below active allocations'; end if;
 end if;
 return new;
end $$;
revoke all on function public.school_admission_campaign_guard() from public;
create trigger school_admission_campaign_authorization_guard before insert or update of source_id,authorization_id,offering_id,location_id,class_offering_context_id,capacity_limit,capacity_unit,student_place_limit,capacity_basis,shift,opens_on,closes_on on school_admission_campaigns for each row execute function public.school_admission_campaign_guard();

create or replace function public.school_admission_authorization_campaign_dependency_guard() returns trigger language plpgsql security definer set search_path=pg_catalog,public as $$
begin
 if exists(select 1 from school_admission_campaigns campaign where campaign.tenant_code=new.tenant_code and campaign.institution_id=new.institution_id and campaign.authorization_id=new.id and (new.status not in ('provisional','accredited') or new.capacity is null or new.capacity<campaign.capacity_limit or new.capacity_unit<>campaign.capacity_unit or new.shift<>campaign.shift or new.effective_from>campaign.opens_on or (new.effective_to is not null and new.effective_to<campaign.closes_on))) then raise exception using errcode='23514',message='authorization update would invalidate an admission campaign',constraint='school_admission_authorization_campaign_dependency_conflict'; end if;
 return new;
end $$;
revoke all on function public.school_admission_authorization_campaign_dependency_guard() from public;
create trigger school_admission_authorization_campaign_dependency_guard before update of status,capacity,capacity_unit,shift,effective_from,effective_to on school_offering_authorizations for each row execute function public.school_admission_authorization_campaign_dependency_guard();

create or replace function public.school_admission_capacity_guard() returns trigger language plpgsql security definer set search_path=pg_catalog,public as $$
declare campaign_student_place_limit integer; valid_match boolean; allocated integer;
begin
 perform pg_advisory_xact_lock(hashtextextended('school_admission_capacity:'||new.tenant_code||':'||new.institution_id||':'||new.campaign_id::text,0));
 select c.student_place_limit into campaign_student_place_limit from school_admission_campaigns c where c.tenant_code=new.tenant_code and c.institution_id=new.institution_id and c.id=new.campaign_id for update;
 select exists(select 1 from school_admission_campaigns c join school_admission_class_offering_contexts x on x.tenant_code=c.tenant_code and x.institution_id=c.institution_id and x.id=new.class_offering_context_id join school_admission_applications app on app.tenant_code=c.tenant_code and app.institution_id=c.institution_id and app.id=new.application_id and app.campaign_id=c.id where c.tenant_code=new.tenant_code and c.institution_id=new.institution_id and c.id=new.campaign_id and c.authorization_id=new.authorization_id and x.authorization_id=new.authorization_id and c.capacity_unit=new.capacity_unit and x.shift=new.shift and c.shift=new.shift) into valid_match;
 if campaign_student_place_limit is null or not valid_match then raise exception using errcode='23514', message='admission capacity allocation has incompatible application, authorization or context'; end if;
 if new.status not in ('held','consumed') then return new; end if;
 select count(*) into allocated from school_admission_capacity_allocations a where a.tenant_code=new.tenant_code and a.institution_id=new.institution_id and a.campaign_id=new.campaign_id and a.status in ('held','consumed') and (tg_op='INSERT' or a.id<>new.id);
 if allocated+1>campaign_student_place_limit then raise exception using errcode='23514', message='admission capacity allocation exceeds campaign student place limit'; end if;
 return new;
end $$;
revoke all on function public.school_admission_capacity_guard() from public;
create trigger school_admission_capacity_allocation_guard before insert or update of application_id,campaign_id,class_offering_context_id,authorization_id,allocated_capacity,capacity_unit,shift,status on school_admission_capacity_allocations for each row execute function public.school_admission_capacity_guard();

create or replace function public.school_admission_archive_snapshot_guard() returns trigger language plpgsql security definer set search_path=pg_catalog,public as $$
declare snapshot_ok boolean;
begin
 if tg_op='UPDATE' and row(new.archive_document_id,new.archive_version_id,new.archive_version_no,new.archive_source_bucket,new.archive_source_object_key,new.archive_source_object_version_id,new.archive_retention_until,new.archive_sha256) is distinct from row(old.archive_document_id,old.archive_version_id,old.archive_version_no,old.archive_source_bucket,old.archive_source_object_key,old.archive_source_object_version_id,old.archive_retention_until,old.archive_sha256) then raise exception 'admission archive evidence snapshot is immutable'; end if;
 if new.archive_document_id is null then return new; end if;
 select exists(select 1 from archive_documents d join archive_document_versions v on v.institution_id=d.institution_id and v.document_id=d.id where d.institution_id=new.institution_id and d.id=new.archive_document_id and d.status='ready' and v.id=new.archive_version_id and v.status='active' and v.version_no=new.archive_version_no and v.source_bucket=new.archive_source_bucket and v.source_object_key=new.archive_source_object_key and v.source_object_version_id=new.archive_source_object_version_id and v.retention_until=new.archive_retention_until and lower(v.source_sha256)=new.archive_sha256 and v.source_object_version_id<>'' and v.retention_until>now()) into snapshot_ok;
 if not coalesce(snapshot_ok,false) then raise exception using errcode='23514', message='admission evidence requires ready WORM archive version snapshot'; end if;
 return new;
end $$;
revoke all on function public.school_admission_archive_snapshot_guard() from public;
create trigger school_admission_application_document_archive_guard before insert or update on school_admission_application_documents for each row execute function public.school_admission_archive_snapshot_guard();
create trigger school_admission_appeal_submission_archive_guard before insert or update on school_admission_appeal_submissions for each row execute function public.school_admission_archive_snapshot_guard();
create trigger school_admission_decision_archive_guard before insert or update on school_admission_decisions for each row execute function public.school_admission_archive_snapshot_guard();
create trigger school_admission_appeal_resolution_archive_guard before insert or update on school_admission_appeal_resolutions for each row execute function public.school_admission_archive_snapshot_guard();
create trigger school_admission_export_manifest_archive_guard before insert or update on school_admission_export_manifests for each row execute function public.school_admission_archive_snapshot_guard();

create or replace function public.school_admission_cross_aggregate_guard() returns trigger language plpgsql security definer set search_path=pg_catalog,public as $$
begin
 if tg_table_name='school_admission_applications' then
  if not exists(select 1 from app_parties party where party.tenant_code=new.tenant_code and party.institution_id=new.institution_id and party.id=new.candidate_party_id and party.party_type='physical' and party.active) then raise exception using errcode='23514',message='admission candidate must be an active in-scope physical person'; end if;
 elsif tg_table_name='school_admission_application_documents' then
  if exists(select 1 from school_admission_applications app where app.tenant_code=new.tenant_code and app.institution_id=new.institution_id and app.id=new.application_id and (app.status in ('admitted','rejected','withdrawn','cancelled') or exists(select 1 from school_admission_decisions decision where decision.tenant_code=app.tenant_code and decision.institution_id=app.institution_id and decision.application_id=app.id))) then raise exception using errcode='23514',message='application documents are immutable after final state or decision'; end if;
  if new.document_requirement_id is not null and not exists(select 1 from school_admission_applications a join school_admission_document_requirements r on r.tenant_code=a.tenant_code and r.institution_id=a.institution_id and r.campaign_id=a.campaign_id where a.tenant_code=new.tenant_code and a.institution_id=new.institution_id and a.id=new.application_id and r.id=new.document_requirement_id) then raise exception using errcode='23514',message='document requirement does not belong to application campaign'; end if;
 elsif tg_table_name='school_admission_criterion_assessments' then
  if exists(select 1 from school_admission_applications app where app.tenant_code=new.tenant_code and app.institution_id=new.institution_id and app.id=new.application_id and (app.status in ('admitted','rejected','withdrawn','cancelled') or exists(select 1 from school_admission_decisions decision where decision.tenant_code=app.tenant_code and decision.institution_id=app.institution_id and decision.application_id=app.id))) then raise exception using errcode='23514',message='criterion assessments are immutable after final state or decision'; end if;
  if not exists(select 1 from school_admission_applications a join school_admission_criteria c on c.tenant_code=a.tenant_code and c.institution_id=a.institution_id and c.campaign_id=a.campaign_id where a.tenant_code=new.tenant_code and a.institution_id=new.institution_id and a.id=new.application_id and c.id=new.criterion_id) then raise exception using errcode='23514',message='criterion does not belong to application campaign'; end if;
 elsif tg_table_name='school_admission_decisions' then
  if new.capacity_allocation_id is not null and not exists(select 1 from school_admission_capacity_allocations ca where ca.tenant_code=new.tenant_code and ca.institution_id=new.institution_id and ca.id=new.capacity_allocation_id and ca.application_id=new.application_id and ca.status in ('held','consumed')) then raise exception using errcode='23514',message='decision capacity allocation is not active for application'; end if;
  if new.outcome='admitted' and exists(select 1 from school_admission_applications app join school_admission_criteria criterion on criterion.tenant_code=app.tenant_code and criterion.institution_id=app.institution_id and criterion.campaign_id=app.campaign_id left join school_admission_criterion_assessments assessment on assessment.tenant_code=criterion.tenant_code and assessment.institution_id=criterion.institution_id and assessment.application_id=app.id and assessment.criterion_id=criterion.id where app.tenant_code=new.tenant_code and app.institution_id=new.institution_id and app.id=new.application_id and criterion.required and coalesce(assessment.outcome,'pending')<>'met') then raise exception using errcode='23514',message='admission decision fails closed while a required criterion is not met'; end if;
  if new.outcome='admitted' and exists(select 1 from school_admission_applications app join school_admission_document_requirements requirement on requirement.tenant_code=app.tenant_code and requirement.institution_id=app.institution_id and requirement.campaign_id=app.campaign_id where app.tenant_code=new.tenant_code and app.institution_id=new.institution_id and app.id=new.application_id and requirement.required and not exists(select 1 from school_admission_application_documents document where document.tenant_code=app.tenant_code and document.institution_id=app.institution_id and document.application_id=app.id and document.document_requirement_id=requirement.id and document.status in ('accepted','waived'))) then raise exception using errcode='23514',message='admission decision fails closed while a required document is not accepted or waived'; end if;
  if not exists(select 1 from school_admission_applications app join school_admission_campaigns campaign on campaign.tenant_code=app.tenant_code and campaign.institution_id=app.institution_id and campaign.id=app.campaign_id join school_operation_policy_evaluations_v2 evaluation on evaluation.tenant_code=app.tenant_code and evaluation.institution_id=app.institution_id and evaluation.id=new.policy_evaluation_v2_id and evaluation.allowed join school_operation_policy_inputs input on input.tenant_code=evaluation.tenant_code and input.institution_id=evaluation.institution_id and input.id=evaluation.input_id where app.tenant_code=new.tenant_code and app.institution_id=new.institution_id and app.id=new.application_id and input.operation_code='admission.decision.issue' and input.offering_id=campaign.offering_id and input.location_id=campaign.location_id and input.effective_on=new.decided_at::date and input.context @> jsonb_build_object('authorization_id',campaign.authorization_id::text,'campaign_id',campaign.id::text,'application_id',app.id::text,'outcome',new.outcome)) then raise exception using errcode='23514',message='admission decision policy evaluation is not the exact allowed operation context'; end if;
  if new.enrolment_id is not null and not exists(select 1 from school_admission_applications a join school_admission_campaigns c on c.tenant_code=a.tenant_code and c.institution_id=a.institution_id and c.id=a.campaign_id join school_admission_class_offering_contexts x on x.tenant_code=c.tenant_code and x.institution_id=c.institution_id and x.id=c.class_offering_context_id join education_student_enrolments e on e.tenant_code=a.tenant_code and e.institution_id=a.institution_id and e.class_id=x.class_id where a.tenant_code=new.tenant_code and a.institution_id=new.institution_id and a.id=new.application_id and e.id=new.enrolment_id and (a.student_id is null or e.student_id=a.student_id)) then raise exception using errcode='23514',message='decision enrolment does not belong to application student and class context'; end if;
 elsif tg_table_name='school_admission_appeals' then
  if not exists(select 1 from school_admission_decisions d where d.tenant_code=new.tenant_code and d.institution_id=new.institution_id and d.id=new.decision_id and d.application_id=new.application_id) then raise exception using errcode='23514',message='appeal decision does not belong to application'; end if;
  if new.submitted_by_party_id is not null and not exists(select 1 from school_admission_applications application where application.tenant_code=new.tenant_code and application.institution_id=new.institution_id and application.id=new.application_id and (application.candidate_party_id=new.submitted_by_party_id or exists(select 1 from school_admission_application_representatives representative where representative.tenant_code=application.tenant_code and representative.institution_id=application.institution_id and representative.application_id=application.id and representative.representative_party_id=new.submitted_by_party_id))) then raise exception using errcode='23514',message='appeal appellant must be candidate or registered legal representative'; end if;
 elsif tg_table_name='school_admission_appeal_resolutions' then
  if new.resulting_decision_id is not null and not exists(select 1 from school_admission_appeals a join school_admission_decisions d on d.tenant_code=a.tenant_code and d.institution_id=a.institution_id and d.application_id=a.application_id where a.tenant_code=new.tenant_code and a.institution_id=new.institution_id and a.id=new.appeal_id and d.id=new.resulting_decision_id) then raise exception using errcode='23514',message='appeal resolution decision belongs to another application'; end if;
  if not exists(select 1 from school_admission_appeals a join school_admission_decisions issued on issued.tenant_code=a.tenant_code and issued.institution_id=a.institution_id and issued.id=a.decision_id where a.tenant_code=new.tenant_code and a.institution_id=new.institution_id and a.id=new.appeal_id and issued.decided_by_subject<>new.resolved_by_subject) then raise exception using errcode='23514',message='appeal resolver must differ from decision issuer'; end if;
  if new.capacity_allocation_id is not null and not exists(select 1 from school_admission_appeals a join school_admission_capacity_allocations ca on ca.tenant_code=a.tenant_code and ca.institution_id=a.institution_id and ca.application_id=a.application_id where a.tenant_code=new.tenant_code and a.institution_id=new.institution_id and a.id=new.appeal_id and ca.id=new.capacity_allocation_id and ca.status in ('held','consumed')) then raise exception using errcode='23514',message='appeal resolution capacity allocation is not active for application'; end if;
  if new.resulting_decision_id is not null and not exists(select 1 from school_admission_decisions d where d.tenant_code=new.tenant_code and d.institution_id=new.institution_id and d.id=new.resulting_decision_id and d.outcome=new.resulting_outcome and (new.resulting_outcome<>'admitted' or d.capacity_allocation_id=new.capacity_allocation_id)) then raise exception using errcode='23514',message='appeal resolution resulting decision must preserve exact outcome and allocation'; end if;
  if not exists(select 1 from school_admission_appeals appeal join school_admission_applications app on app.tenant_code=appeal.tenant_code and app.institution_id=appeal.institution_id and app.id=appeal.application_id join school_admission_campaigns campaign on campaign.tenant_code=app.tenant_code and campaign.institution_id=app.institution_id and campaign.id=app.campaign_id join school_operation_policy_evaluations_v2 evaluation on evaluation.tenant_code=appeal.tenant_code and evaluation.institution_id=appeal.institution_id and evaluation.id=new.policy_evaluation_v2_id and evaluation.allowed join school_operation_policy_inputs input on input.tenant_code=evaluation.tenant_code and input.institution_id=evaluation.institution_id and input.id=evaluation.input_id where appeal.tenant_code=new.tenant_code and appeal.institution_id=new.institution_id and appeal.id=new.appeal_id and input.operation_code='admission.appeal.resolve' and input.offering_id=campaign.offering_id and input.location_id=campaign.location_id and input.effective_on=new.resolved_at::date and input.context @> jsonb_build_object('authorization_id',campaign.authorization_id::text,'campaign_id',campaign.id::text,'application_id',app.id::text,'appeal_id',appeal.id::text,'outcome',new.outcome)) then raise exception using errcode='23514',message='appeal resolution policy evaluation is not the exact allowed operation context'; end if;
 elsif tg_table_name='school_admission_export_manifests' then
  if new.campaign_id is not null and new.application_id is not null and not exists(select 1 from school_admission_applications a where a.tenant_code=new.tenant_code and a.institution_id=new.institution_id and a.id=new.application_id and a.campaign_id=new.campaign_id) then raise exception using errcode='23514',message='admission export application does not belong to campaign'; end if;
 end if;
 return new;
end $$;
revoke all on function public.school_admission_cross_aggregate_guard() from public;
create trigger school_admission_application_guard before insert or update on school_admission_applications for each row execute function public.school_admission_cross_aggregate_guard();
create trigger school_admission_application_document_cross_guard before insert or update on school_admission_application_documents for each row execute function public.school_admission_cross_aggregate_guard();
create trigger school_admission_assessment_cross_guard before insert or update on school_admission_criterion_assessments for each row execute function public.school_admission_cross_aggregate_guard();
create trigger school_admission_decision_cross_guard before insert on school_admission_decisions for each row execute function public.school_admission_cross_aggregate_guard();
create trigger school_admission_appeal_cross_guard before insert or update on school_admission_appeals for each row execute function public.school_admission_cross_aggregate_guard();
create trigger school_admission_appeal_resolution_cross_guard before insert on school_admission_appeal_resolutions for each row execute function public.school_admission_cross_aggregate_guard();
create trigger school_admission_export_cross_guard before insert on school_admission_export_manifests for each row execute function public.school_admission_cross_aggregate_guard();

create or replace function public.school_admission_immutable_evidence() returns trigger language plpgsql as $$ begin raise exception 'admission evidentiary record is immutable'; end $$;
revoke all on function public.school_admission_immutable_evidence() from public;
do $$ declare t text; begin
 foreach t in array array['school_admission_decisions','school_admission_decision_deliveries','school_admission_appeal_submissions','school_admission_appeal_resolutions','school_admission_export_manifests'] loop
  execute format('create trigger %I_immutable before update or delete on %I for each row execute function public.school_admission_immutable_evidence()',t,t);
 end loop;
 foreach t in array array['school_admission_class_offering_contexts','school_admission_campaigns','school_admission_criteria','school_admission_document_requirements','school_admission_capacity_allocations','school_admission_applications','school_admission_application_representatives','school_admission_application_documents','school_admission_criterion_assessments','school_admission_decisions','school_admission_decision_deliveries','school_admission_appeals','school_admission_appeal_submissions','school_admission_appeal_resolutions','school_admission_export_manifests','school_admission_idempotency','school_admission_outbox'] loop
  execute format('alter table %I enable row level security',t); execute format('alter table %I force row level security',t);
  execute format('create policy school_admission_tenant_isolation on %I using (public.can_bypass_tenant_rls() or (tenant_code=public.current_tenant_code() and institution_id=public.current_institution_id())) with check (public.can_bypass_tenant_rls() or (tenant_code=public.current_tenant_code() and institution_id=public.current_institution_id()))',t);
  execute format('create trigger %I_no_hard_delete before delete on %I for each row execute function public.school_operations_no_hard_delete()',t,t);
  execute format('create trigger %I_entity_version after insert or update or delete on %I for each row execute function public.record_entity_version()',t,t);
 end loop;
end $$;

create index school_admission_campaigns_window_idx on school_admission_campaigns(tenant_code,institution_id,status,opens_on,closes_on);
create index school_admission_applications_queue_idx on school_admission_applications(tenant_code,institution_id,campaign_id,status,submitted_at);
create index school_admission_documents_application_idx on school_admission_application_documents(tenant_code,institution_id,application_id,status);
create index school_admission_decisions_application_idx on school_admission_decisions(tenant_code,institution_id,application_id,decided_at desc);
create index school_admission_appeals_queue_idx on school_admission_appeals(tenant_code,institution_id,status,submitted_at);
create index school_admission_capacity_allocations_auth_idx on school_admission_capacity_allocations(tenant_code,institution_id,authorization_id,status);
create index school_admission_outbox_delivery_idx on school_admission_outbox(tenant_code,institution_id,delivery_status,occurred_at);
