create table if not exists education_personnel_assignments (
	id uuid primary key default gen_random_uuid(),
	personnel_id uuid not null references education_personnel(id) on delete cascade,
	assignment_code text not null unique,
	assignment_type text not null check (assignment_type in ('diriginte', 'coordonator_proiect', 'responsabil_comisie', 'mentor', 'membru_comisie', 'administrator_structura')),
	assignment_title text not null default '',
	status text not null check (status in ('propus', 'activ', 'suspendat', 'incetat')),
	assigned_on date not null,
	ended_on date,
	weekly_hours integer not null default 0 check (weekly_hours >= 0),
	decision_reference text not null default '',
	institution_id text not null,
	notes text not null default '',
	created_at timestamptz not null default now(),
	updated_at timestamptz not null default now()
);

create index if not exists education_personnel_assignments_personnel_idx
	on education_personnel_assignments(personnel_id, institution_id, assigned_on, status);

create table if not exists education_personnel_disciplinary_cases (
	id uuid primary key default gen_random_uuid(),
	personnel_id uuid not null references education_personnel(id) on delete cascade,
	case_code text not null unique,
	case_type text not null check (case_type in ('sesizare', 'cercetare', 'sanctiune', 'contestatie')),
	status text not null check (status in ('deschis', 'in_cercetare', 'solutionat', 'contestat', 'inchis')),
	reported_on date not null,
	hearing_on date,
	resolved_on date,
	committee_name text not null default '',
	sanction text not null default '',
	legal_basis text not null default '',
	institution_id text not null,
	notes text not null default '',
	created_at timestamptz not null default now(),
	updated_at timestamptz not null default now()
);

create index if not exists education_personnel_disciplinary_cases_personnel_idx
	on education_personnel_disciplinary_cases(personnel_id, institution_id, reported_on, status);

insert into education_personnel_assignments (
	personnel_id, assignment_code, assignment_type, assignment_title, status,
	assigned_on, ended_on, weekly_hours, decision_reference, institution_id, notes
)
select
	ep.id,
	'ATR-2026-0001',
	'diriginte',
	'Diriginte clasa a IX-a B',
	'activ',
	current_date - 30,
	null,
	1,
	'DEC-PERS-2026-0012',
	ep.institution_id,
	'Atribuire anuală pentru coordonarea colectivului de elevi și relația cu părinții.'
from education_personnel ep
where ep.employee_code = 'PER-2026-1001'
on conflict (assignment_code) do nothing;

insert into education_personnel_disciplinary_cases (
	personnel_id, case_code, case_type, status, reported_on, hearing_on,
	resolved_on, committee_name, sanction, legal_basis, institution_id, notes
)
select
	ep.id,
	'DISC-2026-0001',
	'sesizare',
	'in_cercetare',
	current_date - 12,
	current_date - 5,
	null,
	'Comisia de cercetare disciplinară',
	'',
	'Legea nr. 198/2023 și regulamentul intern al unității',
	ep.institution_id,
	'Sesizare privind nerespectarea programului și întocmirea referatului de constatare.'
from education_personnel ep
where ep.employee_code = 'PER-2026-1002'
on conflict (case_code) do nothing;
