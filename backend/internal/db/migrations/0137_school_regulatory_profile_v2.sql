-- Stage 1B / expand: normalized public/private profile.  Confessional is an
-- overlay, never a third legal form.  Existing 0135 rows stay immutable
-- provenance; applications move to this contract deliberately.
select set_config('app.is_super_admin', 'true', true);

create table school_regulatory_sources (
 id uuid primary key default gen_random_uuid(), tenant_code text not null, institution_id text not null,
 source_kind text not null check(source_kind in ('law','government_decision','ministerial_order','authorization','accreditation','founder_decision','contract','other')),
 citation text not null, article_reference text not null default '', issuer text not null default '', source_url text not null default '', published_on date, consolidated_on date, effective_from date, effective_to date,
	checksum_sha256 text not null default '' check(checksum_sha256='' or checksum_sha256 ~ '^[a-f0-9]{64}$'),
 status text not null default 'active' check(status in ('draft','active','superseded','withdrawn')), expected_version integer not null default 1 check(expected_version>0), verified_at timestamptz, verified_by_subject text not null default '', revalidation_owner_subject text not null default '',
	created_by_subject text not null, created_at timestamptz not null default now(), updated_by_subject text not null, updated_at timestamptz not null default now(),
 unique(tenant_code,institution_id,id), unique(tenant_code,institution_id,citation,expected_version),
 foreign key(tenant_code,institution_id) references app_tenants(code,institution_id) on delete restrict,
 check(effective_to is null or effective_from is null or effective_to >= effective_from),
 check(status <> 'active' or (source_url <> '' and checksum_sha256 <> '' and verified_at is not null and verified_by_subject <> '' and revalidation_owner_subject <> ''))
);
create table school_institution_profiles_v2 (
 id uuid primary key default gen_random_uuid(), tenant_code text not null, institution_id text not null, version integer not null check(version>0),
 status text not null check(status in ('draft','approved','active','superseded')), legal_form text not null check(legal_form in ('public','private')),
 legal_personality boolean not null default true, tax_identifier text not null default '', accounting_basis text not null default '', vat_profile text not null default 'not_registered', treasury_required boolean not null default false,
 effective_from date not null, effective_to date, expected_version integer not null default 1 check(expected_version>0),
 approved_by_subject text not null default '', approved_at timestamptz, created_by_subject text not null, created_at timestamptz not null default now(), updated_by_subject text not null, updated_at timestamptz not null default now(),
 unique(tenant_code,institution_id,id), unique(tenant_code,institution_id,version), unique(tenant_code,institution_id,id,version),
 foreign key(tenant_code,institution_id) references app_tenants(code,institution_id) on delete restrict,
 check(effective_to is null or effective_to >= effective_from), check(status not in ('approved','active') or (approved_by_subject<>'' and approved_at is not null))
);
create table school_profile_sources (
 tenant_code text not null, institution_id text not null, profile_id uuid not null, source_id uuid not null, purpose text not null check(purpose in ('classification','legal_personality','founder','funder','confessional','other')),
 created_by_subject text not null, created_at timestamptz not null default now(), primary key(tenant_code,institution_id,profile_id,source_id,purpose),
 foreign key(tenant_code,institution_id,profile_id) references school_institution_profiles_v2(tenant_code,institution_id,id) on delete restrict,
 foreign key(tenant_code,institution_id,source_id) references school_regulatory_sources(tenant_code,institution_id,id) on delete restrict
);
create table school_institution_party_roles (
 id uuid primary key default gen_random_uuid(), tenant_code text not null, institution_id text not null, party_id uuid not null,
 role_code text not null check(role_code in ('founder','funder','budget_authority','cult','operator','contracting_authority')),
 effective_from date not null, effective_to date, expected_version integer not null default 1 check(expected_version>0), source_id uuid, created_by_subject text not null, created_at timestamptz not null default now(), updated_by_subject text not null, updated_at timestamptz not null default now(),
 unique(tenant_code,institution_id,id), foreign key(tenant_code,institution_id) references app_tenants(code,institution_id) on delete restrict,
 foreign key(tenant_code,institution_id,party_id) references app_parties(tenant_code,institution_id,id) on delete restrict,
 foreign key(tenant_code,institution_id,source_id) references school_regulatory_sources(tenant_code,institution_id,id) on delete restrict,
 check(effective_to is null or effective_to>=effective_from)
);
create index school_profile_v2_effective_idx on school_institution_profiles_v2(tenant_code,institution_id,status,effective_from desc);
