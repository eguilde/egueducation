create table if not exists education_mobility_cases (
	id uuid primary key default gen_random_uuid(),
	case_code text not null unique,
	employee_code text not null,
	full_name text not null,
	school_year text not null,
	request_type text not null check (request_type in ('transfer', 'detasare', 'pretransfer', 'restrangere')),
	stage text not null check (stage in ('draft', 'submitted', 'review', 'approved', 'completed')),
	status text not null check (status in ('open', 'pending', 'approved', 'rejected', 'completed')),
	source_school text not null default '',
	destination_school text not null default '',
	submitted_on date not null,
	reviewed_by text not null default '',
	institution_id text not null,
	notes text not null default ''
);

create table if not exists education_merit_grants (
	id uuid primary key default gen_random_uuid(),
	grant_code text not null unique,
	full_name text not null,
	role_title text not null,
	school_year text not null,
	category text not null check (category in ('predare', 'management', 'consiliere', 'auxiliar')),
	status text not null check (status in ('draft', 'submitted', 'evaluated', 'approved', 'funded')),
	score numeric(5,2) not null default 0,
	committee_name text not null default '',
	decision_date date not null,
	funded boolean not null default false,
	institution_id text not null,
	notes text not null default ''
);

create table if not exists gdpr_retention_policies (
	id uuid primary key default gen_random_uuid(),
	policy_code text not null unique,
	domain_code text not null check (domain_code in ('registratura', 'workflow', 'earchiva', 'education', 'hr')),
	record_category text not null,
	retention_years integer not null check (retention_years > 0),
	legal_basis text not null,
	status text not null check (status in ('draft', 'active', 'review', 'retired')),
	review_due_on date not null,
	owner_name text not null default '',
	institution_id text not null,
	notes text not null default ''
);

create table if not exists gdpr_subject_requests (
	id uuid primary key default gen_random_uuid(),
	request_code text not null unique,
	subject_name text not null,
	request_type text not null check (request_type in ('access', 'rectification', 'erasure', 'restriction', 'portability', 'objection')),
	status text not null check (status in ('received', 'identity_check', 'in_progress', 'waiting_approval', 'completed', 'rejected')),
	submitted_on date not null,
	due_on date not null,
	handled_by text not null default '',
	source_module text not null check (source_module in ('registratura', 'workflow', 'earchiva', 'education')),
	anonymization_required boolean not null default false,
	institution_id text not null,
	notes text not null default ''
);

insert into education_mobility_cases (
	case_code,
	employee_code,
	full_name,
	school_year,
	request_type,
	stage,
	status,
	source_school,
	destination_school,
	submitted_on,
	reviewed_by,
	institution_id,
	notes
)
values
	('MOB-2026-0001', 'PER-2026-1001', 'Andreea Munteanu', '2025-2026', 'transfer', 'review', 'pending', 'Liceul Teoretic Balotești', 'Colegiul Național EguEducation', current_date - 8, 'Comisia de mobilitate', 'inst-001', 'Dosar cu aviz ISJ și fișă de evaluare atașată.'),
	('MOB-2026-0002', 'PER-2026-1002', 'Vlad Rădulescu', '2025-2026', 'detasare', 'approved', 'approved', 'Școala Gimnazială nr. 1 Balotești', 'Colegiul Național EguEducation', current_date - 14, 'Director', 'inst-001', 'Detașare aprobată pentru semestrul al II-lea.'),
	('MOB-2026-0003', 'PER-2026-1003', 'Raluca Ene', '2025-2026', 'pretransfer', 'submitted', 'open', 'Colegiul Național EguEducation', 'Școala Europeană Voluntari', current_date - 3, 'Secretariat', 'inst-001', 'Așteaptă validarea comisiei județene.')
on conflict (case_code) do nothing;

insert into education_merit_grants (
	grant_code,
	full_name,
	role_title,
	school_year,
	category,
	status,
	score,
	committee_name,
	decision_date,
	funded,
	institution_id,
	notes
)
values
	('GRM-2026-0001', 'Ioana Stoica', 'Profesor limba română', '2025-2026', 'predare', 'evaluated', 92.50, 'Comisia gradație de merit', current_date - 10, false, 'inst-001', 'Portofoliu complet și validat în comisie.'),
	('GRM-2026-0002', 'Marius Dobre', 'Director adjunct', '2025-2026', 'management', 'approved', 96.25, 'Comisia gradație de merit', current_date - 5, true, 'inst-001', 'Aprobare finală și includere pe listă de finanțare.'),
	('GRM-2026-0003', 'Simona Pavel', 'Consilier școlar', '2025-2026', 'consiliere', 'submitted', 88.00, 'Comisia gradație de merit', current_date - 2, false, 'inst-001', 'În așteptarea evaluării finale.')
on conflict (grant_code) do nothing;

insert into gdpr_retention_policies (
	policy_code,
	domain_code,
	record_category,
	retention_years,
	legal_basis,
	status,
	review_due_on,
	owner_name,
	institution_id,
	notes
)
values
	('RET-2026-0001', 'education', 'portofoliu_cd', 3, 'Metodologie portofoliu CD (proiect) și procedură internă', 'active', current_date + 180, 'Responsabil GDPR', 'inst-001', 'Păstrare 3 ani după încetarea activității cadrului didactic.'),
	('RET-2026-0002', 'registratura', 'cereri_elevi', 5, 'Nomenclator arhivistic intern', 'active', current_date + 120, 'Secretariat', 'inst-001', 'Include cereri, adeverințe și răspunsuri standard.'),
	('RET-2026-0003', 'workflow', 'jurnal_evenimente', 2, 'Politică internă de audit operațional', 'review', current_date + 45, 'Administrator workflow', 'inst-001', 'În curs de revizuire pentru minimizarea datelor.')
on conflict (policy_code) do nothing;

insert into gdpr_subject_requests (
	request_code,
	subject_name,
	request_type,
	status,
	submitted_on,
	due_on,
	handled_by,
	source_module,
	anonymization_required,
	institution_id,
	notes
)
values
	('DSR-2026-0001', 'Mihnea Iacob', 'access', 'in_progress', current_date - 7, current_date + 23, 'Responsabil GDPR', 'education', false, 'inst-001', 'Solicitare export date privind evaluările anuale.'),
	('DSR-2026-0002', 'Elena Pop', 'rectification', 'waiting_approval', current_date - 5, current_date + 10, 'Secretar șef', 'registratura', true, 'inst-001', 'Corectare nume și adresă în cereri istorice.'),
	('DSR-2026-0003', 'Cristian Moga', 'erasure', 'received', current_date - 1, current_date + 29, 'Responsabil GDPR', 'workflow', true, 'inst-001', 'Necesită verificare temei legal de păstrare.')
on conflict (request_code) do nothing;

insert into workflow_definitions(code, name, category, initial_step, sla_hours, active)
values
	('mobility-case-review', 'Mobilitate personal școlar', 'education', 'Verificare dosar', 120, true),
	('merit-grant-review', 'Gradație de merit', 'education', 'Validare portofoliu', 168, true)
on conflict (code) do update
set name = excluded.name,
	category = excluded.category,
	initial_step = excluded.initial_step,
	sla_hours = excluded.sla_hours,
	active = excluded.active;
