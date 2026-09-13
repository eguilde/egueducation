-- Funding eligibility is intentionally independent of procurement applicability.
select set_config('app.is_super_admin', 'true', true);
create table school_funding_instruments (
 id uuid primary key default gen_random_uuid(), tenant_code text not null, institution_id text not null, code text not null, title text not null, funder_party_id uuid, public_funding boolean not null, school_year text not null, eligibility_status text not null check(eligibility_status in ('draft','eligible','ineligible','suspended','closed')),
 amount numeric(16,2) check(amount is null or amount>=0), currency text not null default 'RON', effective_from date not null, effective_to date, expected_version integer not null default 1 check(expected_version>0), source_id uuid not null, created_by_subject text not null, created_at timestamptz not null default now(), updated_by_subject text not null, updated_at timestamptz not null default now(),
 unique(tenant_code,institution_id,id), foreign key(tenant_code,institution_id) references app_tenants(code,institution_id) on delete restrict,
 foreign key(tenant_code,institution_id,funder_party_id) references app_parties(tenant_code,institution_id,id) on delete restrict, foreign key(tenant_code,institution_id,source_id) references school_regulatory_sources(tenant_code,institution_id,id) on delete restrict, check(effective_to is null or effective_to>=effective_from)
);
create table school_procurement_applicability_assessments (
 id uuid primary key default gen_random_uuid(), tenant_code text not null, institution_id text not null, assessment_scope text not null check(assessment_scope in ('institution','funding_instrument','procurement')), funding_instrument_id uuid,
 determination text not null check(determination in ('applicable','not_applicable','indeterminate')), legal_basis text not null, rationale text not null, effective_from date not null, effective_to date, expected_version integer not null default 1 check(expected_version>0), source_id uuid not null, decided_by_subject text not null, decided_at timestamptz not null default now(), created_by_subject text not null, created_at timestamptz not null default now(),
 unique(tenant_code,institution_id,id), foreign key(tenant_code,institution_id) references app_tenants(code,institution_id) on delete restrict,
 foreign key(tenant_code,institution_id,funding_instrument_id) references school_funding_instruments(tenant_code,institution_id,id) on delete restrict, foreign key(tenant_code,institution_id,source_id) references school_regulatory_sources(tenant_code,institution_id,id) on delete restrict, check(effective_to is null or effective_to>=effective_from)
);
