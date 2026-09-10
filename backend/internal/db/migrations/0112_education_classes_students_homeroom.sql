-- SCH-005: institution-scoped classes, pupils, temporal enrolments and
-- homeroom assignments. All personal data is protected by FORCE RLS; a
-- teacher can read only classes to which they are currently assigned.
create extension if not exists btree_gist;

-- These helpers deliberately read FORCE-RLS identity and assignment tables.
-- Their owner therefore must be able to see the authoritative rows; otherwise
-- a deployment under an ordinary app role could silently turn RLS into deny-all
-- or, worse, make its result depend on unrelated caller grants.
do $$
begin
	if not exists (
		select 1 from pg_catalog.pg_roles
		where rolname = current_user and (rolsuper or rolbypassrls)
	) then
		raise exception 'migration owner must be SUPERUSER or BYPASSRLS for School SECURITY DEFINER helpers';
	end if;
end;
$$;

create table if not exists education_school_classes (
	id uuid primary key default gen_random_uuid(),
	tenant_code text not null references app_tenants(code) on delete restrict,
	institution_id text not null,
	class_code text not null,
	class_name text not null,
	school_year text not null,
	grade_level text not null default '',
	study_shift text not null default 'day' check (study_shift in ('day', 'afternoon', 'evening')),
	active boolean not null default true,
	created_at timestamptz not null default now(),
	updated_at timestamptz not null default now(),
	unique (tenant_code, institution_id, class_code),
	unique (institution_id, school_year, class_name)
);

create table if not exists education_students (
	id uuid primary key default gen_random_uuid(),
	tenant_code text not null references app_tenants(code) on delete restrict,
	institution_id text not null,
	student_code text not null,
	first_name text not null,
	last_name text not null,
	status text not null default 'active' check (status in ('active', 'transferred', 'graduated', 'withdrawn')),
	birth_date date null,
	created_at timestamptz not null default now(),
	updated_at timestamptz not null default now(),
	unique (tenant_code, institution_id, student_code)
);

create table if not exists education_student_enrolments (
	id uuid primary key default gen_random_uuid(),
	tenant_code text not null references app_tenants(code) on delete restrict,
	institution_id text not null,
	student_id uuid not null references education_students(id) on delete restrict,
	class_id uuid not null references education_school_classes(id) on delete restrict,
	enrolled_from date not null,
	enrolled_until date null,
	status text not null default 'active' check (status in ('active', 'transferred', 'completed', 'withdrawn')),
	created_at timestamptz not null default now(),
	updated_at timestamptz not null default now(),
	check (enrolled_until is null or enrolled_until >= enrolled_from)
);

alter table education_student_enrolments
	drop constraint if exists education_student_enrolments_student_period_excl;
alter table education_student_enrolments
	add constraint education_student_enrolments_student_period_excl
	exclude using gist (
		institution_id with =,
		student_id with =,
		daterange(enrolled_from, coalesce(enrolled_until, 'infinity'::date), '[]') with &&
	) where (status = 'active');

create table if not exists education_class_homeroom_assignments (
	id uuid primary key default gen_random_uuid(),
	tenant_code text not null references app_tenants(code) on delete restrict,
	institution_id text not null,
	class_id uuid not null references education_school_classes(id) on delete restrict,
	personnel_id uuid not null references education_personnel(id) on delete restrict,
	app_user_id uuid not null references app_users(id) on delete restrict,
	assigned_from date not null,
	assigned_until date null,
	created_at timestamptz not null default now(),
	updated_at timestamptz not null default now(),
	check (assigned_until is null or assigned_until >= assigned_from),
	unique (tenant_code, institution_id, class_id, personnel_id, assigned_from)
);

alter table education_class_homeroom_assignments
	drop constraint if exists education_class_homeroom_assignments_class_period_excl;
alter table education_class_homeroom_assignments
	add constraint education_class_homeroom_assignments_class_period_excl
	exclude using gist (
		institution_id with =,
		class_id with =,
		daterange(assigned_from, coalesce(assigned_until, 'infinity'::date), '[]') with &&
	);

create index if not exists idx_education_student_enrolments_class_period
	on education_student_enrolments (institution_id, class_id, enrolled_from desc);
create index if not exists idx_education_student_enrolments_student_period
	on education_student_enrolments (institution_id, student_id, enrolled_from desc);
create index if not exists idx_education_class_homeroom_assignments_class_period
	on education_class_homeroom_assignments (institution_id, class_id, assigned_from desc);
create index if not exists idx_education_class_homeroom_assignments_user_period
	on education_class_homeroom_assignments (institution_id, app_user_id, assigned_from desc);

create or replace function public.enforce_education_class_scope()
returns trigger language plpgsql security definer set search_path = pg_catalog, public as $$
declare
	tenant_value text := case when tg_op = 'DELETE' then old.tenant_code else new.tenant_code end;
	institution_value text := case when tg_op = 'DELETE' then old.institution_id else new.institution_id end;
begin
	if not exists (select 1 from public.app_tenants where code = tenant_value and institution_id = institution_value and active) then
		raise exception 'School record tenant and institution must identify an active tenant';
	end if;
	if tg_table_name = 'education_student_enrolments' and tg_op <> 'DELETE' and not exists (
		select 1 from public.education_students student join public.education_school_classes class_row on class_row.id = new.class_id
		where student.id = new.student_id and student.tenant_code = new.tenant_code and student.institution_id = new.institution_id
			and class_row.tenant_code = new.tenant_code and class_row.institution_id = new.institution_id
	) then raise exception 'Student enrolment references a different tenant or institution'; end if;
	if tg_table_name = 'education_class_homeroom_assignments' and tg_op <> 'DELETE' then
		if not exists (select 1 from public.education_school_classes where id = new.class_id and tenant_code = new.tenant_code and institution_id = new.institution_id) then
			raise exception 'Homeroom class belongs to a different tenant or institution';
		end if;
		-- A closure is historical evidence and must remain possible after the
		-- teacher's account expires. New/rebound assignments, however, always
		-- prove the canonical pair and current membership.
		if tg_op = 'INSERT' or new.personnel_id is distinct from old.personnel_id or new.app_user_id is distinct from old.app_user_id or new.class_id is distinct from old.class_id then
			if not exists (
				select 1 from public.education_personnel person
				where person.id = new.personnel_id and person.institution_id = new.institution_id and person.app_user_id = new.app_user_id
			) then raise exception 'Homeroom personnel and application user must be a canonical institutional identity pair'; end if;
			if not public.education_membership_is_eligible(new.app_user_id, new.tenant_code, new.institution_id, null) then
				raise exception 'Homeroom teacher requires a currently eligible institutional membership';
			end if;
		end if;
	end if;
	return coalesce(new, old);
end;
$$;
revoke all on function public.enforce_education_class_scope() from public;

do $$ declare target text; begin
	foreach target in array array['education_school_classes','education_students','education_student_enrolments','education_class_homeroom_assignments'] loop
		execute format('drop trigger if exists trg_%s_scope on public.%I', target, target);
		execute format('create trigger trg_%s_scope before insert or update or delete on public.%I for each row execute function public.enforce_education_class_scope()', target, target);
		execute format('alter table public.%I enable row level security', target);
		execute format('alter table public.%I force row level security', target);
	end loop;
end $$;

create or replace function public.education_classes_actor_has_permission(p_permission text)
returns boolean language sql stable security definer set search_path = pg_catalog, public as $$
	with actor as (
		select u.id from public.app_users u where lower(u.sub) = lower(nullif(btrim(current_setting('app.actor_subject', true)), '')) and u.status = 'active' limit 1
	), current_scope as (
		select public.current_tenant_code() tenant_code, public.current_institution_id() institution_id
	)
	select exists (
		select 1 from actor, current_scope scope
		where scope.tenant_code is not null and scope.institution_id is not null
			and public.education_membership_is_eligible(actor.id, scope.tenant_code, scope.institution_id, null)
			and exists (
				select 1 from public.app_user_permissions up where up.user_id = actor.id and up.tenant_code = scope.tenant_code and up.permission_code = p_permission
				union all select 1 from public.app_user_roles ur join public.app_role_permissions rp on rp.role_code = ur.role_code where ur.user_id = actor.id and ur.tenant_code = scope.tenant_code and rp.permission_code = p_permission
				union all select 1 from public.app_memberships membership join public.app_position_permissions pp on pp.position_code = membership.position_code where membership.user_id = actor.id and membership.tenant_code = scope.tenant_code and pp.permission_code = p_permission and public.education_membership_is_eligible(actor.id, scope.tenant_code, scope.institution_id, membership.position_code)
				union all select 1 from public.app_memberships membership join public.app_position_roles pr on pr.position_code = membership.position_code join public.app_role_permissions rp on rp.role_code = pr.role_code where membership.user_id = actor.id and membership.tenant_code = scope.tenant_code and rp.permission_code = p_permission and public.education_membership_is_eligible(actor.id, scope.tenant_code, scope.institution_id, membership.position_code)
			)
	);
$$;
revoke all on function public.education_classes_actor_has_permission(text) from public;
grant execute on function public.education_classes_actor_has_permission(text) to public;

create or replace function public.education_classes_actor_is_assigned(p_class_id uuid)
returns boolean language sql stable security definer set search_path = pg_catalog, public as $$
	select exists (
		select 1 from public.education_class_homeroom_assignments assignment
		join public.app_users actor on actor.id = assignment.app_user_id and actor.status = 'active'
		where assignment.class_id = p_class_id and assignment.tenant_code = public.current_tenant_code() and assignment.institution_id = public.current_institution_id()
			and lower(actor.sub) = lower(nullif(btrim(current_setting('app.actor_subject', true)), ''))
			and assignment.assigned_from <= current_date and (assignment.assigned_until is null or assignment.assigned_until >= current_date)
			and public.education_membership_is_eligible(actor.id, assignment.tenant_code, assignment.institution_id, null)
			and public.education_classes_actor_has_permission('education.classes.read_assigned')
	);
$$;
revoke all on function public.education_classes_actor_is_assigned(uuid) from public;
grant execute on function public.education_classes_actor_is_assigned(uuid) to public;

do $$ declare target text; begin
	foreach target in array array['education_school_classes','education_students','education_student_enrolments','education_class_homeroom_assignments'] loop
		execute format('drop policy if exists education_classes_read on public.%I', target);
		execute format('drop policy if exists education_classes_manage on public.%I', target);
	end loop;
end $$;

create policy education_classes_read on education_school_classes for select using (
	public.can_bypass_tenant_rls() or (tenant_code = public.current_tenant_code() and institution_id = public.current_institution_id() and (
		public.education_classes_actor_has_permission('education.classes.read') or public.education_classes_actor_has_permission('education.classes.manage') or public.education_classes_actor_is_assigned(id)
	))
);
create policy education_classes_manage on education_school_classes for all using (
	public.can_bypass_tenant_rls() or (tenant_code = public.current_tenant_code() and institution_id = public.current_institution_id() and public.education_classes_actor_has_permission('education.classes.manage'))
) with check (
	public.can_bypass_tenant_rls() or (tenant_code = public.current_tenant_code() and institution_id = public.current_institution_id() and public.education_classes_actor_has_permission('education.classes.manage'))
);

create policy education_students_read on education_students for select using (
	public.can_bypass_tenant_rls() or (tenant_code = public.current_tenant_code() and institution_id = public.current_institution_id() and (
		public.education_classes_actor_has_permission('education.classes.read') or public.education_classes_actor_has_permission('education.classes.manage') or exists (select 1 from education_student_enrolments enrolment where enrolment.student_id = education_students.id and public.education_classes_actor_is_assigned(enrolment.class_id))
	))
);
create policy education_students_manage on education_students for all using (
	public.can_bypass_tenant_rls() or (tenant_code = public.current_tenant_code() and institution_id = public.current_institution_id() and public.education_classes_actor_has_permission('education.classes.manage'))
) with check (
	public.can_bypass_tenant_rls() or (tenant_code = public.current_tenant_code() and institution_id = public.current_institution_id() and public.education_classes_actor_has_permission('education.classes.manage'))
);

create policy education_student_enrolments_read on education_student_enrolments for select using (
	public.can_bypass_tenant_rls() or (tenant_code = public.current_tenant_code() and institution_id = public.current_institution_id() and (
		public.education_classes_actor_has_permission('education.classes.read') or public.education_classes_actor_has_permission('education.classes.manage') or public.education_classes_actor_is_assigned(class_id)
	))
);
create policy education_student_enrolments_manage on education_student_enrolments for all using (
	public.can_bypass_tenant_rls() or (tenant_code = public.current_tenant_code() and institution_id = public.current_institution_id() and public.education_classes_actor_has_permission('education.classes.manage'))
) with check (
	public.can_bypass_tenant_rls() or (tenant_code = public.current_tenant_code() and institution_id = public.current_institution_id() and public.education_classes_actor_has_permission('education.classes.manage'))
);

create policy education_homeroom_assignments_read on education_class_homeroom_assignments for select using (
	public.can_bypass_tenant_rls() or (tenant_code = public.current_tenant_code() and institution_id = public.current_institution_id() and (
		public.education_classes_actor_has_permission('education.classes.read') or public.education_classes_actor_has_permission('education.classes.manage') or public.education_classes_actor_is_assigned(class_id)
	))
);
create policy education_homeroom_assignments_manage on education_class_homeroom_assignments for all using (
	public.can_bypass_tenant_rls() or (tenant_code = public.current_tenant_code() and institution_id = public.current_institution_id() and public.education_classes_actor_has_permission('education.classes.manage'))
) with check (
	public.can_bypass_tenant_rls() or (tenant_code = public.current_tenant_code() and institution_id = public.current_institution_id() and public.education_classes_actor_has_permission('education.classes.manage'))
);

insert into app_permissions(code, label) values
	('education.classes.read', 'Read all school classes and pupils'),
	('education.classes.manage', 'Manage school classes, pupils and homeroom assignments'),
	('education.classes.read_assigned', 'Read assigned homeroom classes and pupils')
on conflict (code) do update set label = excluded.label;

insert into app_role_permissions(role_code, permission_code)
select role_code, permission_code from (values
	('super_admin', 'education.classes.read'), ('super_admin', 'education.classes.manage'),
	('admin', 'education.classes.read'), ('admin', 'education.classes.manage'),
	('director', 'education.classes.read'), ('director', 'education.classes.manage'),
	('secretar', 'education.classes.read'),
	('profesor', 'education.classes.read_assigned')
) as mapping(role_code, permission_code)
on conflict do nothing;

do $$ declare target text; begin
	foreach target in array array['education_school_classes','education_students','education_student_enrolments','education_class_homeroom_assignments'] loop
		execute format('drop trigger if exists trg_%s_entity_version on public.%I', target, target);
		execute format('create trigger trg_%s_entity_version after insert or update or delete on public.%I for each row execute function public.record_entity_version()', target, target);
	end loop;
end $$;
