create table if not exists education_mobility_final_decisions (
	id uuid primary key default gen_random_uuid(),
	mobility_case_id uuid not null references education_mobility_cases(id) on delete cascade,
	decision_code text not null unique,
	decision_type text not null check (decision_type in ('validare_dosar', 'repartizare', 'transfer', 'detasare', 'solutionare_contestatie')),
	outcome text not null check (outcome in ('admis', 'respins', 'redistribuit', 'rezerva')),
	approved_on date not null,
	effective_from date not null,
	panel_name text not null default '',
	legal_basis text not null default '',
	destination_unit text not null default '',
	institution_id text not null,
	notes text not null default '',
	created_at timestamptz not null default now(),
	updated_at timestamptz not null default now()
);

create index if not exists education_mobility_final_decisions_case_idx
	on education_mobility_final_decisions(mobility_case_id, institution_id, approved_on, outcome);

create table if not exists education_mobility_result_issues (
	id uuid primary key default gen_random_uuid(),
	mobility_case_id uuid not null references education_mobility_cases(id) on delete cascade,
	issue_code text not null unique,
	document_type text not null check (document_type in ('decizie', 'comunicare', 'adeverinta', 'raport_final')),
	recipient_name text not null default '',
	recipient_role text not null default '',
	delivery_channel text not null check (delivery_channel in ('registratura', 'email', 'intern', 'posta')),
	delivery_status text not null check (delivery_status in ('pregatit', 'emis', 'transmis', 'confirmat')),
	issued_on date not null,
	delivered_on date,
	registry_reference text not null default '',
	institution_id text not null,
	notes text not null default '',
	created_at timestamptz not null default now(),
	updated_at timestamptz not null default now()
);

create index if not exists education_mobility_result_issues_case_idx
	on education_mobility_result_issues(mobility_case_id, institution_id, issued_on, delivery_status);

create table if not exists education_merit_final_decisions (
	id uuid primary key default gen_random_uuid(),
	grant_id uuid not null references education_merit_grants(id) on delete cascade,
	decision_code text not null unique,
	decision_stage text not null check (decision_stage in ('evaluare_initiala', 'solutionare_contestatie', 'validare_finala', 'finantare')),
	outcome text not null check (outcome in ('admis', 'respins', 'rezerva', 'finantat')),
	approved_on date not null,
	effective_from date not null,
	panel_name text not null default '',
	funded boolean not null default false,
	legal_basis text not null default '',
	institution_id text not null,
	notes text not null default '',
	created_at timestamptz not null default now(),
	updated_at timestamptz not null default now()
);

create index if not exists education_merit_final_decisions_grant_idx
	on education_merit_final_decisions(grant_id, institution_id, approved_on, outcome);

create table if not exists education_merit_result_issues (
	id uuid primary key default gen_random_uuid(),
	grant_id uuid not null references education_merit_grants(id) on delete cascade,
	issue_code text not null unique,
	document_type text not null check (document_type in ('decizie', 'comunicare', 'extras_punctaj', 'adeverinta')),
	recipient_name text not null default '',
	recipient_role text not null default '',
	delivery_channel text not null check (delivery_channel in ('registratura', 'email', 'intern', 'posta')),
	delivery_status text not null check (delivery_status in ('pregatit', 'emis', 'transmis', 'confirmat')),
	issued_on date not null,
	delivered_on date,
	registry_reference text not null default '',
	institution_id text not null,
	notes text not null default '',
	created_at timestamptz not null default now(),
	updated_at timestamptz not null default now()
);

create index if not exists education_merit_result_issues_grant_idx
	on education_merit_result_issues(grant_id, institution_id, issued_on, delivery_status);

insert into education_mobility_final_decisions (
	mobility_case_id, decision_code, decision_type, outcome, approved_on, effective_from,
	panel_name, legal_basis, destination_unit, institution_id, notes
)
select
	emc.id,
	'MOB-DEC-2026-0001',
	'transfer',
	'admis',
	current_date - 3,
	current_date + 30,
	'Comisia judeteana de mobilitate',
	'Metodologia mobilitatii personalului didactic 2025-2026',
	emc.destination_school,
	emc.institution_id,
	'Cererea a fost admisa dupa validarea dosarului si punctajului final.'
from education_mobility_cases emc
where emc.case_code = 'MOB-2026-0001'
on conflict (decision_code) do nothing;

insert into education_mobility_result_issues (
	mobility_case_id, issue_code, document_type, recipient_name, recipient_role, delivery_channel,
	delivery_status, issued_on, delivered_on, registry_reference, institution_id, notes
)
select
	emc.id,
	'MOB-OUT-2026-0001',
	'decizie',
	emc.full_name,
	'Cadru didactic solicitant',
	'registratura',
	'transmis',
	current_date - 2,
	current_date - 1,
	'REG-MOB-2026-0001',
	emc.institution_id,
	'Rezultatul final a fost comunicat oficial catre solicitant.'
from education_mobility_cases emc
where emc.case_code = 'MOB-2026-0001'
on conflict (issue_code) do nothing;

insert into education_merit_final_decisions (
	grant_id, decision_code, decision_stage, outcome, approved_on, effective_from,
	panel_name, funded, legal_basis, institution_id, notes
)
select
	emg.id,
	'MER-DEC-2026-0001',
	'validare_finala',
	'finantat',
	current_date - 4,
	current_date + 60,
	'Comisia de evaluare gradatie de merit',
	true,
	'Metodologia privind acordarea gradatiei de merit 2025-2026',
	emg.institution_id,
	'Candidatura a fost validata final si inclusa la finantare.'
from education_merit_grants emg
where emg.grant_code = 'MER-2026-0001'
on conflict (decision_code) do nothing;

insert into education_merit_result_issues (
	grant_id, issue_code, document_type, recipient_name, recipient_role, delivery_channel,
	delivery_status, issued_on, delivered_on, registry_reference, institution_id, notes
)
select
	emg.id,
	'MER-OUT-2026-0001',
	'comunicare',
	emg.full_name,
	emg.role_title,
	'email',
	'confirmat',
	current_date - 3,
	current_date - 2,
	'REG-MER-2026-0001',
	emg.institution_id,
	'Comunicarea rezultatului final si a incadrarii la finantare a fost transmisa si confirmata.'
from education_merit_grants emg
where emg.grant_code = 'MER-2026-0001'
on conflict (issue_code) do nothing;
