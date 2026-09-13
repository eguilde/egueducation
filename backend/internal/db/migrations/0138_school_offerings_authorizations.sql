-- Stage 1B / expand: authorization is evaluated per offer and physical location.
select set_config('app.is_super_admin', 'true', true);
insert into app_permissions(code,label) values
 ('institution.offerings.read','Read institution locations, education offerings and authorization history'),
 ('institution.offerings.manage','Create institution locations, education offerings and authorization decisions')
on conflict(code) do update set label=excluded.label;
insert into app_role_permissions(role_code,permission_code) values
 ('super_admin','institution.offerings.read'),('super_admin','institution.offerings.manage'),
 ('admin','institution.offerings.read'),('admin','institution.offerings.manage'),
 ('director','institution.offerings.read'),('director','institution.offerings.manage'),
 ('secretar','institution.offerings.read'),('secretar','institution.offerings.manage'),
 ('inspector','institution.offerings.read')
on conflict do nothing;
create table school_locations (
 id uuid primary key default gen_random_uuid(), tenant_code text not null, institution_id text not null, code text not null, name text not null, address text not null default '', active boolean not null default true,
 effective_from date not null, effective_to date, expected_version integer not null default 1 check(expected_version>0), source_id uuid, created_by_subject text not null, created_at timestamptz not null default now(), updated_by_subject text not null, updated_at timestamptz not null default now(),
 unique(tenant_code,institution_id,id), unique(tenant_code,institution_id,code), foreign key(tenant_code,institution_id) references app_tenants(code,institution_id) on delete restrict,
 foreign key(tenant_code,institution_id,source_id) references school_regulatory_sources(tenant_code,institution_id,id) on delete restrict, check(effective_to is null or effective_to>=effective_from), check(active or effective_to is not null)
);
create table school_education_offerings (
 id uuid primary key default gen_random_uuid(), tenant_code text not null, institution_id text not null, code text not null, education_level text not null, specialization_code text not null default '', language_code text not null default 'ro', title text not null,
 active boolean not null default true, effective_from date not null, effective_to date, expected_version integer not null default 1 check(expected_version>0), created_by_subject text not null, created_at timestamptz not null default now(), updated_by_subject text not null, updated_at timestamptz not null default now(),
 unique(tenant_code,institution_id,id), unique(tenant_code,institution_id,code), foreign key(tenant_code,institution_id) references app_tenants(code,institution_id) on delete restrict, check(effective_to is null or effective_to>=effective_from), check(active or effective_to is not null)
);
create table school_offering_authorizations (
 id uuid primary key default gen_random_uuid(), tenant_code text not null, institution_id text not null, offering_id uuid not null, location_id uuid not null,
 status text not null check(status in ('provisional','authorized','accredited','suspended','withdrawn','expired')), authority_name text not null, decision_reference text not null, capacity integer check(capacity is null or capacity>=0),
 effective_from date not null, effective_to date, expected_version integer not null default 1 check(expected_version>0), source_id uuid not null, replaces_authorization_id uuid, created_by_subject text not null, created_at timestamptz not null default now(), updated_by_subject text not null, updated_at timestamptz not null default now(),
 unique(tenant_code,institution_id,id), foreign key(tenant_code,institution_id,offering_id) references school_education_offerings(tenant_code,institution_id,id) on delete restrict,
 foreign key(tenant_code,institution_id,location_id) references school_locations(tenant_code,institution_id,id) on delete restrict, foreign key(tenant_code,institution_id,source_id) references school_regulatory_sources(tenant_code,institution_id,id) on delete restrict,
 foreign key(tenant_code,institution_id,replaces_authorization_id) references school_offering_authorizations(tenant_code,institution_id,id) on delete restrict,
 check(effective_to is null or effective_to>=effective_from)
);
create index school_offering_authorizations_effective_idx on school_offering_authorizations(tenant_code,institution_id,offering_id,location_id,status,effective_from desc);

-- Cross-table effective-date invariants are enforced in PostgreSQL, not only
-- by API handlers. Locking both parent rows serializes authorization writes
-- with concurrent parent-window changes.
create or replace function school_offering_authorization_parent_guard() returns trigger
language plpgsql
as $$
declare
 offering_active boolean;
 offering_from date;
 offering_to date;
 location_active boolean;
 location_from date;
 location_to date;
begin
 select active,effective_from,effective_to
 into offering_active,offering_from,offering_to
 from school_education_offerings
 where tenant_code=new.tenant_code and institution_id=new.institution_id and id=new.offering_id
 for update;
 if not found then
  raise exception using errcode='23503',message='offering does not exist in the authorization scope',constraint='school_offering_authorization_offering_scope';
 end if;
 if new.effective_from < offering_from or (offering_to is not null and (new.effective_to is null or new.effective_to > offering_to)) then
  raise exception using errcode='23514',message='authorization is outside the offering effective window',constraint='authorization_outside_offering_window';
 end if;

 select active,effective_from,effective_to
 into location_active,location_from,location_to
 from school_locations
 where tenant_code=new.tenant_code and institution_id=new.institution_id and id=new.location_id
 for update;
 if not found then
  raise exception using errcode='23503',message='location does not exist in the authorization scope',constraint='school_offering_authorization_location_scope';
 end if;
 if new.effective_from < location_from or (location_to is not null and (new.effective_to is null or new.effective_to > location_to)) then
  raise exception using errcode='23514',message='authorization is outside the location effective window',constraint='authorization_outside_location_window';
 end if;
 return new;
end;
$$;

create trigger school_offering_authorization_parent_guard_trigger
before insert or update of tenant_code,institution_id,offering_id,location_id,effective_from,effective_to
on school_offering_authorizations
for each row execute function school_offering_authorization_parent_guard();

create or replace function school_location_authorization_dependency_guard() returns trigger
language plpgsql
as $$
begin
 if not new.active and new.effective_to is null then
  raise exception using errcode='23514',message='an inactive location requires an effective end date',constraint='school_location_inactive_requires_effective_to';
 end if;
 if exists (
  select 1 from school_offering_authorizations authorization_row
  where authorization_row.tenant_code=new.tenant_code
    and authorization_row.institution_id=new.institution_id
    and authorization_row.location_id=new.id
    and (authorization_row.effective_from < new.effective_from
      or (new.effective_to is not null and (authorization_row.effective_to is null or authorization_row.effective_to > new.effective_to)))
 ) then
  raise exception using errcode='23514',message='location update would invalidate a dependent authorization',constraint='school_location_authorization_dependency_conflict';
 end if;
 return new;
end;
$$;

create trigger school_location_authorization_dependency_guard_trigger
before update of active,effective_from,effective_to on school_locations
for each row execute function school_location_authorization_dependency_guard();

create or replace function school_education_offering_authorization_dependency_guard() returns trigger
language plpgsql
as $$
begin
 if not new.active and new.effective_to is null then
  raise exception using errcode='23514',message='an inactive education offering requires an effective end date',constraint='education_offering_inactive_requires_effective_to';
 end if;
 if exists (
  select 1 from school_offering_authorizations authorization_row
  where authorization_row.tenant_code=new.tenant_code
    and authorization_row.institution_id=new.institution_id
    and authorization_row.offering_id=new.id
    and (authorization_row.effective_from < new.effective_from
      or (new.effective_to is not null and (authorization_row.effective_to is null or authorization_row.effective_to > new.effective_to)))
 ) then
  raise exception using errcode='23514',message='education offering update would invalidate a dependent authorization',constraint='education_offering_authorization_dependency_conflict';
 end if;
 return new;
end;
$$;

create trigger school_education_offering_authorization_dependency_guard_trigger
before update of active,effective_from,effective_to on school_education_offerings
for each row execute function school_education_offering_authorization_dependency_guard();
