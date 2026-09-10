-- A professional portfolio belongs to a durable institutional personnel
-- identity. Display names, raw profile attributes, and client payloads are
-- not identity links. Historic rows are linked only from a verified login
-- identifier and only when the relation is one-to-one in this institution.
alter table education_personnel
	add column if not exists app_user_id uuid references app_users(id) on delete restrict;

with verified_matches as (
	select distinct person.id as personnel_id, identity_row.user_id
	from education_personnel person
	join app_tenants tenant on tenant.institution_id = person.institution_id and tenant.active
	join app_user_identities identity_row on identity_row.verified_at is not null
	where person.app_user_id is null
		and public.education_membership_is_eligible(identity_row.user_id, tenant.code, person.institution_id, null)
		and (
			(identity_row.identity_type = 'email'
				and nullif(lower(btrim(person.email)), '') is not null
				and lower(btrim(person.email)) = lower(btrim(identity_row.normalized_value)))
			or (identity_row.identity_type = 'phone'
				and nullif(btrim(person.phone), '') is not null
				and btrim(person.phone) = btrim(identity_row.normalized_value))
		)
), unique_personnel as (
	select personnel_id, min(user_id::text)::uuid as user_id
	from verified_matches
	group by personnel_id
	having count(*) = 1
), unique_users as (
	select user_id
	from verified_matches
	group by user_id
	having count(*) = 1
)
update education_personnel person
set app_user_id = candidate.user_id
from unique_personnel candidate
join unique_users user_candidate on user_candidate.user_id = candidate.user_id
where person.id = candidate.personnel_id
	and person.app_user_id is null;

-- An account may be the canonical personnel identity only once within an
-- institution. The partial index leaves historic unlinked personnel rows
-- visible and reconcilable.
create unique index if not exists uq_education_personnel_institution_app_user
	on education_personnel (institution_id, app_user_id)
	where app_user_id is not null;

-- School registry numbers are institution-local. Before removing a legacy
-- global key, give a deterministic diagnostic if a damaged database already
-- has a duplicate inside the new scope.
do $$
declare
	table_name text;
	code_column text;
	constraint_name text;
	index_name text;
	has_scoped_duplicate boolean;
begin
	for table_name, code_column in
		select * from (values
			('education_personnel', 'employee_code'),
			('education_portfolios', 'portfolio_code'),
			('education_decisions', 'decision_code'),
			('education_managerial_dossiers', 'dossier_code'),
			('education_regulations', 'regulation_code'),
			('education_evaluations', 'evaluation_code'),
			('education_declarations', 'declaration_code'),
			('education_mobility_cases', 'case_code'),
			('education_merit_grants', 'grant_code'),
			('education_personnel_assignments', 'assignment_code'),
			('education_personnel_disciplinary_cases', 'case_code')
		) as scoped_codes(table_name, code_column)
	loop
		if to_regclass('public.' || table_name) is null then
			continue;
		end if;
		execute format(
			'select exists (select 1 from public.%I group by institution_id, %I having count(*) > 1)',
			table_name, code_column
		) into has_scoped_duplicate;
		if has_scoped_duplicate then
			raise exception '0111 cannot scope %.%: duplicate code already exists within an institution', table_name, code_column;
		end if;

		-- Constraints own their index, so drop them first without relying on the
		-- generated constraint name used by an earlier database version.
		for constraint_name in
			select constraint_row.conname
			from pg_constraint constraint_row
			join pg_class relation on relation.oid = constraint_row.conrelid
			join pg_namespace namespace on namespace.oid = relation.relnamespace
			where namespace.nspname = 'public'
				and relation.relname = table_name
				and constraint_row.contype = 'u'
				and constraint_row.conkey = array[(
					select attribute.attnum from pg_attribute attribute
					where attribute.attrelid = relation.oid and attribute.attname = code_column
				)]::smallint[]
		loop
			execute format('alter table public.%I drop constraint %I', table_name, constraint_name);
		end loop;

		-- Some historical environments created a standalone unique index rather
		-- than a table constraint. Remove only a one-column global code index.
		for index_name in
			select index_relation.relname
			from pg_index index_row
			join pg_class relation on relation.oid = index_row.indrelid
			join pg_namespace namespace on namespace.oid = relation.relnamespace
			join pg_class index_relation on index_relation.oid = index_row.indexrelid
			where namespace.nspname = 'public'
				and relation.relname = table_name
				and index_row.indisunique and not index_row.indisprimary
				and index_row.indkey::text = (
					select attribute.attnum::text from pg_attribute attribute
					where attribute.attrelid = relation.oid and attribute.attname = code_column
				)
				and not exists (select 1 from pg_constraint constraint_row where constraint_row.conindid = index_row.indexrelid)
		loop
			execute format('drop index public.%I', index_name);
		end loop;

		execute format(
			'create unique index if not exists %I on public.%I (institution_id, %I)',
			'uq_' || table_name || '_institution_' || code_column,
			table_name,
			code_column
		);
	end loop;
end;
$$;

-- Keep the portfolio owner pair immutable and prove it is the same durable
-- personnel/user identity in the same institution. Historic unbound records
-- remain readable; they cannot be used to create a new portfolio.
create or replace function public.enforce_education_portfolio_owner()
returns trigger
language plpgsql
as $$
declare
	owner_has_membership boolean;
	personnel_matches_owner boolean;
begin
	if tg_op = 'INSERT' and (new.owner_user_id is null or new.owner_personnel_id is null) then
		raise exception 'education portfolio owner user and personnel identities are required';
	end if;
	if tg_op = 'UPDATE' and old.owner_user_id is not null and new.owner_user_id is distinct from old.owner_user_id then
		raise exception 'education portfolio owner_user_id is immutable';
	end if;
	if tg_op = 'UPDATE' and old.owner_personnel_id is not null and new.owner_personnel_id is distinct from old.owner_personnel_id then
		raise exception 'education portfolio owner_personnel_id is immutable';
	end if;

	if new.owner_user_id is not null then
		select exists(
			select 1 from app_tenants tenant
			where tenant.institution_id = new.institution_id
				and public.education_membership_is_eligible(new.owner_user_id, tenant.code, new.institution_id, null)
		) into owner_has_membership;
		if not owner_has_membership then
			raise exception 'education portfolio owner must have an active membership in the portfolio institution';
		end if;
	end if;

	if new.owner_user_id is not null and new.owner_personnel_id is not null then
		select exists(
			select 1 from education_personnel person
			where person.id = new.owner_personnel_id
				and person.institution_id = new.institution_id
				and person.app_user_id = new.owner_user_id
		) into personnel_matches_owner;
		if not personnel_matches_owner then
			raise exception 'education portfolio owner personnel must be canonically linked to owner user in the portfolio institution';
		end if;
	end if;
	return new;
end;
$$;

create or replace function public.prevent_education_personnel_identity_reassignment()
returns trigger
language plpgsql
as $$
begin
	if new.app_user_id is distinct from old.app_user_id
		and exists (select 1 from education_portfolios portfolio where portfolio.owner_personnel_id = old.id) then
		raise exception 'education personnel identity cannot be reassigned while referenced by a portfolio';
	end if;
	return new;
end;
$$;

drop trigger if exists trg_education_personnel_identity_reassignment on education_personnel;
create trigger trg_education_personnel_identity_reassignment
	before update of app_user_id on education_personnel
	for each row execute function public.prevent_education_personnel_identity_reassignment();
