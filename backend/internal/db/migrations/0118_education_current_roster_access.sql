-- A holder of education.classes.read_assigned may read only the *current*
-- roster of a class for which that holder is currently the homeroom teacher.
-- Historical pupil/enrolment access requires education.classes.read (or
-- education.classes.manage); it is deliberately not implied by this grant.
--
-- This is a forward migration: 0112 may already have been applied in a tenant
-- database, therefore replace policies/helpers instead of editing its history.

create or replace function public.education_classes_actor_is_assigned(p_class_id uuid)
returns boolean language sql stable security definer set search_path = pg_catalog, public as $$
	select exists (
		select 1
		from public.education_class_homeroom_assignments assignment
		join public.app_users actor on actor.id = assignment.app_user_id and actor.status = 'active'
		where assignment.class_id = p_class_id
			and assignment.tenant_code = public.current_tenant_code()
			and assignment.institution_id = public.current_institution_id()
			and lower(actor.sub) = lower(nullif(btrim(current_setting('app.actor_subject', true)), ''))
			and assignment.assigned_from <= current_date
			and (assignment.assigned_until is null or assignment.assigned_until >= current_date)
			and public.education_membership_is_eligible(actor.id, assignment.tenant_code, assignment.institution_id, null)
			and public.education_classes_actor_has_permission('education.classes.read_assigned')
	);
$$;
revoke all on function public.education_classes_actor_is_assigned(uuid) from public;
grant execute on function public.education_classes_actor_is_assigned(uuid) to public;

-- Canonical temporal predicate for every assigned-teacher pupil/enrolment
-- read. Keep this function as the sole definition of a current roster:
-- active pupil, active enrolment, effective enrolment period and an effective
-- homeroom assignment for the authenticated subject.
create or replace function public.education_classes_current_roster_enrolment(p_enrolment_id uuid)
returns boolean language sql stable security definer set search_path = pg_catalog, public as $$
	select exists (
		select 1
		from public.education_student_enrolments enrolment
		join public.education_students student on student.id = enrolment.student_id
		where enrolment.id = p_enrolment_id
			and enrolment.tenant_code = public.current_tenant_code()
			and enrolment.institution_id = public.current_institution_id()
			and student.tenant_code = enrolment.tenant_code
			and student.institution_id = enrolment.institution_id
			and student.status = 'active'
			and enrolment.status = 'active'
			and enrolment.enrolled_from <= current_date
			and (enrolment.enrolled_until is null or enrolment.enrolled_until >= current_date)
			and public.education_classes_actor_is_assigned(enrolment.class_id)
	);
$$;
revoke all on function public.education_classes_current_roster_enrolment(uuid) from public;
grant execute on function public.education_classes_current_roster_enrolment(uuid) to public;

-- An assigned teacher may see only their own assignment and only while it is
-- effective. This prevents historic/future assignment rows leaking through the
-- homeroom endpoint while retaining them for full readers/managers.
create or replace function public.education_classes_actor_owns_current_homeroom(p_assignment_id uuid)
returns boolean language sql stable security definer set search_path = pg_catalog, public as $$
	select exists (
		select 1
		from public.education_class_homeroom_assignments assignment
		join public.app_users actor on actor.id = assignment.app_user_id and actor.status = 'active'
		where assignment.id = p_assignment_id
			and assignment.tenant_code = public.current_tenant_code()
			and assignment.institution_id = public.current_institution_id()
			and lower(actor.sub) = lower(nullif(btrim(current_setting('app.actor_subject', true)), ''))
			and assignment.assigned_from <= current_date
			and (assignment.assigned_until is null or assignment.assigned_until >= current_date)
			and public.education_classes_actor_is_assigned(assignment.class_id)
	);
$$;
revoke all on function public.education_classes_actor_owns_current_homeroom(uuid) from public;
grant execute on function public.education_classes_actor_owns_current_homeroom(uuid) to public;

do $$ declare target text; begin
	foreach target in array array['education_school_classes','education_students','education_student_enrolments','education_class_homeroom_assignments'] loop
		execute format('drop policy if exists education_classes_read on public.%I', target);
		execute format('drop policy if exists education_classes_manage on public.%I', target);
	end loop;
end $$;

create policy education_classes_read on education_school_classes for select using (
	public.can_bypass_tenant_rls() or (tenant_code = public.current_tenant_code() and institution_id = public.current_institution_id() and (
		public.education_classes_actor_has_permission('education.classes.read')
		or public.education_classes_actor_has_permission('education.classes.manage')
		or public.education_classes_actor_is_assigned(id)
	))
);
create policy education_classes_manage on education_school_classes for all using (
	public.can_bypass_tenant_rls() or (tenant_code = public.current_tenant_code() and institution_id = public.current_institution_id() and public.education_classes_actor_has_permission('education.classes.manage'))
) with check (
	public.can_bypass_tenant_rls() or (tenant_code = public.current_tenant_code() and institution_id = public.current_institution_id() and public.education_classes_actor_has_permission('education.classes.manage'))
);

create policy education_students_read on education_students for select using (
	public.can_bypass_tenant_rls() or (tenant_code = public.current_tenant_code() and institution_id = public.current_institution_id() and (
		public.education_classes_actor_has_permission('education.classes.read')
		or public.education_classes_actor_has_permission('education.classes.manage')
		or exists (select 1 from public.education_student_enrolments enrolment where enrolment.student_id = education_students.id and public.education_classes_current_roster_enrolment(enrolment.id))
	))
);
create policy education_students_manage on education_students for all using (
	public.can_bypass_tenant_rls() or (tenant_code = public.current_tenant_code() and institution_id = public.current_institution_id() and public.education_classes_actor_has_permission('education.classes.manage'))
) with check (
	public.can_bypass_tenant_rls() or (tenant_code = public.current_tenant_code() and institution_id = public.current_institution_id() and public.education_classes_actor_has_permission('education.classes.manage'))
);

create policy education_student_enrolments_read on education_student_enrolments for select using (
	public.can_bypass_tenant_rls() or (tenant_code = public.current_tenant_code() and institution_id = public.current_institution_id() and (
		public.education_classes_actor_has_permission('education.classes.read')
		or public.education_classes_actor_has_permission('education.classes.manage')
		or public.education_classes_current_roster_enrolment(id)
	))
);
create policy education_student_enrolments_manage on education_student_enrolments for all using (
	public.can_bypass_tenant_rls() or (tenant_code = public.current_tenant_code() and institution_id = public.current_institution_id() and public.education_classes_actor_has_permission('education.classes.manage'))
) with check (
	public.can_bypass_tenant_rls() or (tenant_code = public.current_tenant_code() and institution_id = public.current_institution_id() and public.education_classes_actor_has_permission('education.classes.manage'))
);

create policy education_homeroom_assignments_read on education_class_homeroom_assignments for select using (
	public.can_bypass_tenant_rls() or (tenant_code = public.current_tenant_code() and institution_id = public.current_institution_id() and (
		public.education_classes_actor_has_permission('education.classes.read')
		or public.education_classes_actor_has_permission('education.classes.manage')
		or public.education_classes_actor_owns_current_homeroom(id)
	))
);
create policy education_homeroom_assignments_manage on education_class_homeroom_assignments for all using (
	public.can_bypass_tenant_rls() or (tenant_code = public.current_tenant_code() and institution_id = public.current_institution_id() and public.education_classes_actor_has_permission('education.classes.manage'))
) with check (
	public.can_bypass_tenant_rls() or (tenant_code = public.current_tenant_code() and institution_id = public.current_institution_id() and public.education_classes_actor_has_permission('education.classes.manage'))
);

