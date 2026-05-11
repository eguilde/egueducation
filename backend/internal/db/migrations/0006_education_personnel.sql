create table if not exists education_personnel (
	id uuid primary key default gen_random_uuid(),
	employee_code text not null unique,
	full_name text not null,
	role_title text not null,
	employment_type text not null check (employment_type in ('titular', 'suplinitor', 'plata_cu_ora', 'auxiliar')),
	status text not null check (status in ('active', 'on_leave', 'vacant', 'inactive')),
	evaluation_status text not null check (evaluation_status in ('draft', 'in_review', 'finalized')),
	mobility_stage text not null check (mobility_stage in ('none', 'transfer', 'detasare', 'restrangere')),
	school_year text not null,
	assigned_unit text not null default '',
	phone text not null default '',
	email text not null default '',
	has_portfolio boolean not null default false,
	institution_id text not null,
	notes text not null default '',
	created_at timestamptz not null default now(),
	updated_at timestamptz not null default now()
);

insert into education_personnel (
	employee_code,
	full_name,
	role_title,
	employment_type,
	status,
	evaluation_status,
	mobility_stage,
	school_year,
	assigned_unit,
	phone,
	email,
	has_portfolio,
	institution_id,
	notes
)
values
	('PER-2026-0001', 'Maria Popescu', 'Profesor limba română', 'titular', 'active', 'in_review', 'none', '2025-2026', 'Catedra limba și comunicare', '+40740111001', 'maria.popescu@egueducation.ro', true, 'inst-001', 'Dosar evaluare anuală în curs.'),
	('PER-2026-0002', 'Andrei Ionescu', 'Profesor matematică', 'titular', 'active', 'finalized', 'transfer', '2025-2026', 'Catedra matematică-informatică', '+40740111002', 'andrei.ionescu@egueducation.ro', true, 'inst-001', 'Solicitare transfer interjudețean.'),
	('PER-2026-0003', 'Elena Dumitru', 'Învățător', 'suplinitor', 'active', 'draft', 'none', '2025-2026', 'Învățământ primar', '+40740111003', 'elena.dumitru@egueducation.ro', false, 'inst-001', 'Portofoliu în curs de completare.'),
	('PER-2026-0004', 'Sorin Pavel', 'Secretar șef', 'auxiliar', 'on_leave', 'finalized', 'none', '2025-2026', 'Secretariat', '+40740111004', 'sorin.pavel@egueducation.ro', false, 'inst-001', 'Concediu medical până la final de lună.'),
	('PER-2026-0005', 'Ioana Marinescu', 'Profesor fizică', 'plata_cu_ora', 'active', 'in_review', 'detasare', '2025-2026', 'Catedra științe', '+40740111005', 'ioana.marinescu@egueducation.ro', true, 'inst-001', 'Detașare în interesul învățământului.')
on conflict do nothing;

insert into app_permissions(code, label) values
	('education.personnel.read', 'Read education personnel'),
	('education.personnel.manage', 'Manage education personnel')
on conflict (code) do nothing;

insert into app_user_permissions(user_id, permission_code)
select id, permission_code
from app_users
cross join (values
	('education.personnel.read'),
	('education.personnel.manage')
) as permissions(permission_code)
where sub = 'usr-001'
on conflict do nothing;
