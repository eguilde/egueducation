-- Scope-bound, evidentiary valorification packages.  A package is tied to one
-- authoritative source record; human-entered target references are therefore
-- never the source of truth for evaluation, mobility, or merit use-cases.
-- Every source is first bound to a personnel identity.  A package may only
-- relate that source to the portfolio of the same person and school year.
alter table education_evaluations
	add column if not exists personnel_id uuid references education_personnel(id) on delete restrict;
alter table education_mobility_cases
	add column if not exists personnel_id uuid references education_personnel(id) on delete restrict;
alter table education_merit_grants
	add column if not exists personnel_id uuid references education_personnel(id) on delete restrict;

update education_evaluations evaluation
set personnel_id = personnel.id
from education_personnel personnel
where evaluation.personnel_id is null
	and personnel.institution_id = evaluation.institution_id
	and personnel.employee_code = evaluation.employee_code;
update education_mobility_cases mobility
set personnel_id = personnel.id
from education_personnel personnel
where mobility.personnel_id is null
	and personnel.institution_id = mobility.institution_id
	and personnel.employee_code = mobility.employee_code;
-- Merit grants have no employee code in the legacy schema.  Backfill only an
-- unambiguous institutional name/year match; ambiguous historic rows remain
-- deliberately ineligible until an administrator resolves their identity.
update education_merit_grants merit
set personnel_id = (
	select (array_agg(personnel.id))[1]
	from education_personnel personnel
	where personnel.institution_id = merit.institution_id
		and personnel.school_year = merit.school_year
		and lower(btrim(personnel.full_name)) = lower(btrim(merit.full_name))
)
where merit.personnel_id is null
	and 1 = (
		select count(*)
		from education_personnel personnel
		where personnel.institution_id = merit.institution_id
			and personnel.school_year = merit.school_year
			and lower(btrim(personnel.full_name)) = lower(btrim(merit.full_name))
	);

create index if not exists idx_education_evaluations_personnel_scope on education_evaluations (institution_id, personnel_id, school_year);
create index if not exists idx_education_mobility_cases_personnel_scope on education_mobility_cases (institution_id, personnel_id, school_year);
create index if not exists idx_education_merit_grants_personnel_scope on education_merit_grants (institution_id, personnel_id, school_year);

-- Existing source handlers still supply legacy employee/name fields.  Bind
-- those writes to personnel at the database boundary so a later package does
-- not depend on a client remembering an additional identity field.
create or replace function public.bind_education_evaluation_personnel()
returns trigger language plpgsql as $$
begin
	if new.personnel_id is null then
		select personnel.id into new.personnel_id
		from education_personnel personnel
		where personnel.institution_id = new.institution_id and personnel.employee_code = new.employee_code;
	end if;
	if new.personnel_id is not null and not exists (
		select 1 from education_personnel personnel
		where personnel.id = new.personnel_id and personnel.institution_id = new.institution_id and personnel.employee_code = new.employee_code
	) then raise exception 'evaluation personnel identity must match its institution and employee code'; end if;
	return new;
end $$;
create or replace function public.bind_education_mobility_personnel()
returns trigger language plpgsql as $$
begin
	if new.personnel_id is null then
		select personnel.id into new.personnel_id
		from education_personnel personnel
		where personnel.institution_id = new.institution_id and personnel.employee_code = new.employee_code;
	end if;
	if new.personnel_id is not null and not exists (
		select 1 from education_personnel personnel
		where personnel.id = new.personnel_id and personnel.institution_id = new.institution_id and personnel.employee_code = new.employee_code
	) then raise exception 'mobility personnel identity must match its institution and employee code'; end if;
	return new;
end $$;
create or replace function public.bind_education_merit_personnel()
returns trigger language plpgsql as $$
begin
	if new.personnel_id is null then
		select (array_agg(personnel.id))[1] into new.personnel_id
		from education_personnel personnel
		where personnel.institution_id = new.institution_id
			and personnel.school_year = new.school_year
			and lower(btrim(personnel.full_name)) = lower(btrim(new.full_name))
		having count(*) = 1;
	end if;
	if new.personnel_id is not null and not exists (
		select 1 from education_personnel personnel
		where personnel.id = new.personnel_id and personnel.institution_id = new.institution_id
			and personnel.school_year = new.school_year
			and lower(btrim(personnel.full_name)) = lower(btrim(new.full_name))
	) then raise exception 'merit personnel identity must match its institution, school year, and person'; end if;
	return new;
end $$;
drop trigger if exists trg_education_evaluations_personnel_identity on education_evaluations;
create trigger trg_education_evaluations_personnel_identity before insert or update on education_evaluations for each row execute function public.bind_education_evaluation_personnel();
drop trigger if exists trg_education_mobility_personnel_identity on education_mobility_cases;
create trigger trg_education_mobility_personnel_identity before insert or update on education_mobility_cases for each row execute function public.bind_education_mobility_personnel();
drop trigger if exists trg_education_merit_personnel_identity on education_merit_grants;
create trigger trg_education_merit_personnel_identity before insert or update on education_merit_grants for each row execute function public.bind_education_merit_personnel();

create table if not exists education_portfolio_valorification_packages (
	id uuid primary key default gen_random_uuid(),
	tenant_code text not null references app_tenants(code) on delete restrict,
	institution_id text not null,
	portfolio_id uuid not null references education_portfolios(id) on delete restrict,
	scope text not null check (scope in ('evaluare_profesionala', 'mobilitate', 'gradatie_merit')),
	source_evaluation_id uuid references education_evaluations(id) on delete restrict,
	source_mobility_case_id uuid references education_mobility_cases(id) on delete restrict,
	source_merit_grant_id uuid references education_merit_grants(id) on delete restrict,
	status text not null default 'draft' check (status in ('draft', 'submitted', 'validated', 'completed')),
	created_by_subject text not null default '',
	created_at timestamptz not null default now(),
	submitted_by_subject text not null default '',
	submitted_at timestamptz,
	validated_by_subject text not null default '',
	validated_at timestamptz,
	completed_by_subject text not null default '',
	completed_at timestamptz,
	updated_at timestamptz not null default now(),
	check (
		(scope = 'evaluare_profesionala' and source_evaluation_id is not null and source_mobility_case_id is null and source_merit_grant_id is null)
		or (scope = 'mobilitate' and source_evaluation_id is null and source_mobility_case_id is not null and source_merit_grant_id is null)
		or (scope = 'gradatie_merit' and source_evaluation_id is null and source_mobility_case_id is null and source_merit_grant_id is not null)
	)
);

create unique index if not exists uq_education_portfolio_valorification_package_evaluation
	on education_portfolio_valorification_packages (portfolio_id, scope, source_evaluation_id)
	where source_evaluation_id is not null;
create unique index if not exists uq_education_portfolio_valorification_package_mobility
	on education_portfolio_valorification_packages (portfolio_id, scope, source_mobility_case_id)
	where source_mobility_case_id is not null;
create unique index if not exists uq_education_portfolio_valorification_package_merit
	on education_portfolio_valorification_packages (portfolio_id, scope, source_merit_grant_id)
	where source_merit_grant_id is not null;

create table if not exists education_portfolio_valorification_package_documents (
	id uuid primary key default gen_random_uuid(),
	package_id uuid not null references education_portfolio_valorification_packages(id) on delete restrict,
	institution_id text not null,
	archive_document_id uuid not null references archive_documents(id) on delete restrict,
	archive_version_id uuid not null references archive_document_versions(id) on delete restrict,
	archive_version_no integer not null check (archive_version_no > 0),
	archive_source_bucket text not null,
	archive_source_object_key text not null,
	archive_sha256 text not null check (archive_sha256 ~ '^[0-9a-f]{64}$'),
	created_by_subject text not null default '',
	created_at timestamptz not null default now(),
	unique (package_id, archive_version_id)
);

create or replace function public.enforce_education_portfolio_valorification_package()
returns trigger language plpgsql as $$
declare
	actor text := nullif(btrim(current_setting('app.actor_subject', true)), '');
	bypass boolean := public.can_bypass_tenant_rls();
	portfolio_personnel_id uuid;
	portfolio_school_year text;
begin
	if tg_op = 'DELETE' then raise exception 'portfolio valorification packages cannot be hard-deleted'; end if;
	if not bypass and actor is null then raise exception 'authenticated actor is required for valorification package'; end if;
	if not exists (select 1 from app_tenants t where t.code=new.tenant_code and t.institution_id=new.institution_id) then
		raise exception 'valorification package tenant and institution do not match';
	end if;
	select p.owner_personnel_id, p.school_year
	into portfolio_personnel_id, portfolio_school_year
	from education_portfolios p
	where p.id=new.portfolio_id and p.institution_id=new.institution_id;
	if portfolio_school_year is null then
		raise exception 'valorification package portfolio does not belong to institution';
	end if;
	if portfolio_personnel_id is null then
		raise exception 'valorification package portfolio must have an identity-bound personnel owner';
	end if;
	if new.scope='evaluare_profesionala' and not exists (select 1 from education_evaluations e where e.id=new.source_evaluation_id and e.institution_id=new.institution_id and e.personnel_id=portfolio_personnel_id and e.school_year=portfolio_school_year) then raise exception 'valorification package evaluation source must match portfolio personnel and school year'; end if;
	if new.scope='mobilitate' and not exists (select 1 from education_mobility_cases m where m.id=new.source_mobility_case_id and m.institution_id=new.institution_id and m.personnel_id=portfolio_personnel_id and m.school_year=portfolio_school_year) then raise exception 'valorification package mobility source must match portfolio personnel and school year'; end if;
	if new.scope='gradatie_merit' and not exists (select 1 from education_merit_grants g where g.id=new.source_merit_grant_id and g.institution_id=new.institution_id and g.personnel_id=portfolio_personnel_id and g.school_year=portfolio_school_year) then raise exception 'valorification package merit source must match portfolio personnel and school year'; end if;
	if tg_op='INSERT' then
		if new.status <> 'draft' then raise exception 'valorification package must start as draft'; end if;
		if not bypass then new.created_by_subject:=actor; end if;
		new.submitted_by_subject:=''; new.submitted_at:=null;
		new.validated_by_subject:=''; new.validated_at:=null;
		new.completed_by_subject:=''; new.completed_at:=null;
		new.created_at:=now(); new.updated_at:=new.created_at;
		return new;
	end if;
	if row(new.tenant_code,new.institution_id,new.portfolio_id,new.scope,new.source_evaluation_id,new.source_mobility_case_id,new.source_merit_grant_id,new.created_by_subject,new.created_at)
		is distinct from row(old.tenant_code,old.institution_id,old.portfolio_id,old.scope,old.source_evaluation_id,old.source_mobility_case_id,old.source_merit_grant_id,old.created_by_subject,old.created_at) then
		raise exception 'valorification package route and source evidence are immutable';
	end if;
	if old.status='draft' and new.status='draft' then
		if row(new.submitted_by_subject,new.submitted_at,new.validated_by_subject,new.validated_at,new.completed_by_subject,new.completed_at)
			is distinct from row(old.submitted_by_subject,old.submitted_at,old.validated_by_subject,old.validated_at,old.completed_by_subject,old.completed_at) then raise exception 'draft valorification package lifecycle provenance is server controlled'; end if;
		new.updated_at:=now(); return new;
	end if;
	if old.status='draft' and new.status='submitted' then
		if row(old.submitted_by_subject,old.submitted_at,old.validated_by_subject,old.validated_at,old.completed_by_subject,old.completed_at)
			is distinct from row(''::text,null::timestamptz,''::text,null::timestamptz,''::text,null::timestamptz) then
			raise exception 'draft valorification package lifecycle provenance is invalid';
		end if;
		if not exists (select 1 from education_portfolio_valorification_package_documents document where document.package_id=old.id) then
			raise exception 'submitted valorification package requires archive-version evidence';
		end if;
		if not bypass then new.submitted_by_subject:=actor; end if;
		new.submitted_at:=now(); new.validated_by_subject:=''; new.validated_at:=null;
		new.completed_by_subject:=''; new.completed_at:=null; new.updated_at:=new.submitted_at; return new;
	end if;
	if old.status='submitted' and new.status='validated' then
		if row(new.submitted_by_subject,new.submitted_at)
			is distinct from row(old.submitted_by_subject,old.submitted_at) then
			raise exception 'submitted valorification package provenance is immutable';
		end if;
		if not bypass then new.validated_by_subject:=actor; end if;
		new.validated_at:=now(); new.completed_by_subject:=''; new.completed_at:=null;
		new.updated_at:=new.validated_at; return new;
	end if;
	if old.status='validated' and new.status='completed' then
		if row(new.submitted_by_subject,new.submitted_at,new.validated_by_subject,new.validated_at)
			is distinct from row(old.submitted_by_subject,old.submitted_at,old.validated_by_subject,old.validated_at) then
			raise exception 'validated valorification package provenance is immutable';
		end if;
		if not bypass then new.completed_by_subject:=actor; end if;
		new.completed_at:=now(); new.updated_at:=new.completed_at; return new;
	end if;
	raise exception 'invalid valorification package transition % -> %', old.status, new.status;
end $$;

create or replace function public.enforce_education_portfolio_valorification_package_document()
returns trigger language plpgsql as $$
declare actor text := nullif(btrim(current_setting('app.actor_subject', true)), '');
	package_status text;
	package_institution text;
	version_document uuid;
	archive_version_number integer;
	source_bucket text;
	source_key text;
	source_hash text;
begin
	if tg_op='DELETE' then raise exception 'valorification package documents cannot be hard-deleted'; end if;
	if tg_op='UPDATE' then raise exception 'valorification package document evidence is immutable'; end if;
	if actor is null and not public.can_bypass_tenant_rls() then raise exception 'authenticated actor is required for valorification package document'; end if;
	select status,institution_id into package_status,package_institution from education_portfolio_valorification_packages where id=new.package_id;
	if not found or package_institution <> new.institution_id then raise exception 'valorification package document must belong to its package institution'; end if;
	if package_status <> 'draft' then raise exception 'valorification package documents are immutable after submission'; end if;
	select version.document_id,version.version_no,version.source_bucket,version.source_object_key,version.source_sha256
	into version_document,archive_version_number,source_bucket,source_key,source_hash
	from archive_document_versions version
	where version.id=new.archive_version_id and version.institution_id=new.institution_id and version.status='active';
	if not found or version_document <> new.archive_document_id then raise exception 'valorification package document must reference an archive version in institution'; end if;
	if btrim(source_bucket) = '' or btrim(source_key) = '' or lower(btrim(source_hash)) !~ '^[0-9a-f]{64}$' then raise exception 'valorification package document requires an active archive version with complete source provenance'; end if;
	if not public.can_bypass_tenant_rls() then new.created_by_subject:=actor; end if;
	new.archive_version_no:=archive_version_number; new.archive_source_bucket:=source_bucket; new.archive_source_object_key:=source_key; new.archive_sha256:=source_hash; new.created_at:=now();
	return new;
end $$;

drop trigger if exists trg_education_portfolio_valorification_package on education_portfolio_valorification_packages;
create trigger trg_education_portfolio_valorification_package before insert or update or delete on education_portfolio_valorification_packages for each row execute function public.enforce_education_portfolio_valorification_package();
drop trigger if exists trg_education_portfolio_valorification_package_document on education_portfolio_valorification_package_documents;
create trigger trg_education_portfolio_valorification_package_document before insert or update or delete on education_portfolio_valorification_package_documents for each row execute function public.enforce_education_portfolio_valorification_package_document();
drop trigger if exists trg_education_portfolio_valorification_package_entity_version on education_portfolio_valorification_packages;
create trigger trg_education_portfolio_valorification_package_entity_version after insert or update or delete on education_portfolio_valorification_packages for each row execute function public.record_entity_version();
drop trigger if exists trg_education_portfolio_valorification_package_document_entity_version on education_portfolio_valorification_package_documents;
create trigger trg_education_portfolio_valorification_package_document_entity_version after insert or update or delete on education_portfolio_valorification_package_documents for each row execute function public.record_entity_version();

alter table education_portfolio_valorification_packages enable row level security;
alter table education_portfolio_valorification_packages force row level security;
alter table education_portfolio_valorification_package_documents enable row level security;
alter table education_portfolio_valorification_package_documents force row level security;
create policy tenant_isolation on education_portfolio_valorification_packages using (public.can_bypass_tenant_rls() or (tenant_code=public.current_tenant_code() and institution_id=public.current_institution_id())) with check (public.can_bypass_tenant_rls() or (tenant_code=public.current_tenant_code() and institution_id=public.current_institution_id()));
create policy tenant_isolation on education_portfolio_valorification_package_documents using (public.can_bypass_tenant_rls() or institution_id=public.current_institution_id()) with check (public.can_bypass_tenant_rls() or institution_id=public.current_institution_id());
