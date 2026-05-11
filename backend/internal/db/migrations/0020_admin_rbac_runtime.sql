alter table app_users
	add column if not exists status text not null default 'active',
	add column if not exists last_login_at timestamptz;

create table if not exists app_positions (
	code text primary key,
	name text not null,
	scope_module text not null,
	active boolean not null default true,
	sort_order integer not null default 0,
	updated_at timestamptz not null default now()
);

create table if not exists app_memberships (
	id uuid primary key default gen_random_uuid(),
	user_id uuid not null references app_users(id) on delete cascade,
	position_code text not null references app_positions(code) on delete cascade,
	organization_name text not null,
	is_primary boolean not null default false,
	active boolean not null default true,
	start_date date not null,
	end_date date,
	created_at timestamptz not null default now(),
	updated_at timestamptz not null default now()
);

insert into app_permissions(code, label) values
	('admin.memberships.manage', 'Manage memberships'),
	('admin.positions.manage', 'Manage positions'),
	('admin.permissions.manage', 'Manage permissions')
on conflict (code) do nothing;

insert into app_user_permissions(user_id, permission_code)
select id, permission_code
from app_users
cross join (values
	('admin.memberships.manage'),
	('admin.positions.manage'),
	('admin.permissions.manage')
) as permissions(permission_code)
where sub = 'usr-001'
on conflict do nothing;

insert into app_positions (code, name, scope_module, active, sort_order) values
	('super_admin', 'Super Admin', 'admin', true, 10),
	('director', 'Director', 'education', true, 20),
	('secretariat', 'Secretariat', 'registratura', true, 30),
	('registrator', 'Registrator', 'registratura', true, 40),
	('gdpr_officer', 'GDPR Officer', 'gdpr', true, 50),
	('workflow_admin', 'Workflow Admin', 'workflow', true, 60),
	('arhivar', 'Arhivar', 'earchiva', true, 70),
	('inspector', 'Inspector', 'education', true, 80),
	('hr', 'HR', 'education', true, 90)
on conflict (code) do update
set name = excluded.name,
	scope_module = excluded.scope_module,
	active = excluded.active,
	sort_order = excluded.sort_order,
	updated_at = now();

insert into app_users (sub, name, email, phone_number, locale, status, last_login_at) values
	('usr-002', 'Mihai Popa', 'mihai.popa@egueducation.ro', '+40740100102', 'ro', 'active', '2026-05-11T09:30:00Z'),
	('usr-003', 'Ioana Dumitrescu', 'ioana.dumitrescu@egueducation.ro', '+40740100103', 'ro', 'active', '2026-05-10T15:48:00Z'),
	('usr-004', 'Daniel Georgescu', 'daniel.georgescu@egueducation.ro', '+40740100104', 'ro', 'pending', '2026-05-09T11:20:00Z'),
	('usr-005', 'Roxana Stan', 'roxana.stan@egueducation.ro', '+40740100105', 'en', 'active', '2026-05-10T08:17:00Z'),
	('usr-006', 'Carmen Pavel', 'carmen.pavel@egueducation.ro', '+40740100106', 'ro', 'active', '2026-05-08T16:22:00Z'),
	('usr-007', 'Cristian Matei', 'cristian.matei@egueducation.ro', '+40740100107', 'ro', 'suspended', '2026-05-06T09:45:00Z'),
	('usr-008', 'Andreea Nistor', 'andreea.nistor@egueducation.ro', '+40740100108', 'ro', 'active', '2026-05-11T07:05:00Z'),
	('usr-009', 'Elena Marinescu', 'elena.marinescu@egueducation.ro', '+40740100109', 'ro', 'active', '2026-05-11T06:52:00Z'),
	('usr-010', 'Victor Tudor', 'victor.tudor@egueducation.ro', '+40740100110', 'en', 'pending', '2026-05-05T14:14:00Z'),
	('usr-011', 'Larisa Ene', 'larisa.ene@egueducation.ro', '+40740100111', 'ro', 'active', '2026-05-10T10:00:00Z'),
	('usr-012', 'George Sandu', 'george.sandu@egueducation.ro', '+40740100112', 'ro', 'active', '2026-05-08T12:00:00Z')
on conflict (sub) do update
set name = excluded.name,
	email = excluded.email,
	phone_number = excluded.phone_number,
	locale = excluded.locale,
	status = excluded.status,
	last_login_at = excluded.last_login_at,
	updated_at = now();

update app_users
set status = 'active',
	last_login_at = '2026-05-11T10:15:00Z'
where sub = 'usr-001';

insert into app_user_roles(user_id, role_code)
select u.id, v.role_code
from app_users u
join (values
	('usr-002', 'super_admin'),
	('usr-005', 'gdpr_officer'),
	('usr-007', 'workflow_admin')
) as v(sub, role_code) on v.sub = u.sub
on conflict do nothing;

insert into app_user_modules(user_id, module_code)
select u.id, m.module_code
from app_users u
cross join (values
	('dashboard'),
	('registratura'),
	('workflow'),
	('earchiva'),
	('education'),
	('gdpr')
) as m(module_code)
where u.sub in ('usr-002','usr-003','usr-004','usr-005','usr-006','usr-007','usr-008','usr-009','usr-010','usr-011','usr-012')
on conflict do nothing;

insert into app_memberships (user_id, position_code, organization_name, is_primary, active, start_date, end_date)
select u.id, v.position_code, v.organization_name, v.is_primary, v.active, v.start_date::date, v.end_date::date
from app_users u
join (values
	('usr-001', 'super_admin', 'Colegiul Național EguEducation', true, true, '2024-09-01', null),
	('usr-002', 'director', 'Colegiul Național EguEducation', true, true, '2024-09-01', null),
	('usr-003', 'secretariat', 'Colegiul Național EguEducation', true, true, '2025-01-15', null),
	('usr-004', 'registrator', 'Colegiul Național EguEducation', true, true, '2025-02-01', null),
	('usr-005', 'gdpr_officer', 'Colegiul Național EguEducation', true, true, '2024-10-01', null),
	('usr-006', 'hr', 'Colegiul Național EguEducation', true, true, '2025-01-10', null),
	('usr-007', 'workflow_admin', 'Colegiul Național EguEducation', true, false, '2024-09-15', '2026-04-30'),
	('usr-008', 'arhivar', 'Colegiul Național EguEducation', true, true, '2024-11-01', null),
	('usr-009', 'inspector', 'Inspectoratul Școlar Demo', true, true, '2025-03-01', null),
	('usr-010', 'secretariat', 'Colegiul Național EguEducation', true, false, '2025-03-10', null),
	('usr-011', 'director', 'Liceul Teoretic Balotești', true, true, '2024-09-01', null),
	('usr-012', 'registrator', 'Liceul Teoretic Balotești', true, true, '2025-02-15', null)
) as v(sub, position_code, organization_name, is_primary, active, start_date, end_date) on v.sub = u.sub
where not exists (
	select 1
	from app_memberships am
	where am.user_id = u.id
		and am.position_code = v.position_code
		and am.organization_name = v.organization_name
		and am.start_date = v.start_date::date
);
