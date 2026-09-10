-- Reconcile the published shared class-scope trigger. PostgreSQL may resolve
-- fields referenced by a compound boolean expression even when the table-name
-- predicate is false. Keep table-specific NEW/OLD fields inside nested blocks
-- so inserts into education_school_classes and education_students remain valid.
create or replace function public.enforce_education_class_scope()
returns trigger
language plpgsql
security definer
set search_path = pg_catalog, public
as $$
declare
	tenant_value text := case when tg_op = 'DELETE' then old.tenant_code else new.tenant_code end;
	institution_value text := case when tg_op = 'DELETE' then old.institution_id else new.institution_id end;
	identity_changed boolean := false;
begin
	-- Tenant and institution are part of the immutable record identity. Moving
	-- any parent or relation row in place could break the scoped relationship
	-- guarantees because the published foreign keys intentionally use UUIDs.
	if tg_op = 'UPDATE' then
		if new.tenant_code is distinct from old.tenant_code
			or new.institution_id is distinct from old.institution_id then
			raise exception 'School record tenant and institution are immutable';
		end if;
	end if;

	if not exists (
		select 1
		from public.app_tenants
		where code = tenant_value and institution_id = institution_value and active
	) then
		raise exception 'School record tenant and institution must identify an active tenant';
	end if;

	if tg_table_name = 'education_student_enrolments' then
		if tg_op <> 'DELETE' then
			if not exists (
				select 1
				from public.education_students student
				join public.education_school_classes class_row on class_row.id = new.class_id
				where student.id = new.student_id
					and student.tenant_code = new.tenant_code
					and student.institution_id = new.institution_id
					and class_row.tenant_code = new.tenant_code
					and class_row.institution_id = new.institution_id
			) then
				raise exception 'Student enrolment references a different tenant or institution';
			end if;
		end if;
	end if;

	if tg_table_name = 'education_class_homeroom_assignments' then
		if tg_op <> 'DELETE' then
			if not exists (
				select 1
				from public.education_school_classes
				where id = new.class_id
					and tenant_code = new.tenant_code
					and institution_id = new.institution_id
			) then
				raise exception 'Homeroom class belongs to a different tenant or institution';
			end if;

			if tg_op = 'INSERT' then
				identity_changed := true;
			else
				identity_changed := new.personnel_id is distinct from old.personnel_id
					or new.app_user_id is distinct from old.app_user_id
					or new.class_id is distinct from old.class_id;
			end if;

			if identity_changed then
				if not exists (
					select 1
					from public.education_personnel person
					where person.id = new.personnel_id
						and person.institution_id = new.institution_id
						and person.app_user_id = new.app_user_id
				) then
					raise exception 'Homeroom personnel and application user must be a canonical institutional identity pair';
				end if;
				if not public.education_membership_is_eligible(new.app_user_id, new.tenant_code, new.institution_id, null) then
					raise exception 'Homeroom teacher requires a currently eligible institutional membership';
				end if;
			end if;
		end if;
	end if;

	return coalesce(new, old);
end;
$$;

revoke all on function public.enforce_education_class_scope() from public;
