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
	('PER-2026-0101', 'Raluca Stan', 'Director', 'titular', 'active', 'finalized', 'none', '2025-2026', 'Conducere', '+40740111991', 'raluca.stan@egueducation.ro', true, 'inst-001', 'Dosar personal distinct pentru director.'),
	('PER-2026-0102', 'Mihai Enache', 'Director adjunct', 'titular', 'active', 'in_review', 'none', '2025-2026', 'Conducere', '+40740111992', 'mihai.enache@egueducation.ro', true, 'inst-001', 'Dosar personal distinct pentru director adjunct.')
on conflict (employee_code) do nothing;

create table if not exists education_personnel_file_documents (
	id uuid primary key default gen_random_uuid(),
	personnel_id uuid not null references education_personnel(id) on delete cascade,
	document_code text not null,
	document_category text not null check (document_category in ('identificare', 'studii', 'cariera', 'evaluare', 'declaratie', 'medical', 'disciplina', 'management')),
	document_title text not null,
	file_scope text not null check (file_scope in ('dosar_personal', 'dosar_director', 'dosar_director_adjunct')),
	confidentiality_level text not null check (confidentiality_level in ('intern', 'confidential', 'strict_confidential')),
	issued_on date not null,
	expires_on date,
	file_reference text not null default '',
	sensitive_data boolean not null default false,
	included_in_portfolio boolean not null default false,
	institution_id text not null,
	notes text not null default '',
	created_at timestamptz not null default now(),
	updated_at timestamptz not null default now(),
	unique (personnel_id, document_code)
);

create index if not exists idx_education_personnel_file_documents_lookup
	on education_personnel_file_documents(institution_id, personnel_id, document_category, confidentiality_level, issued_on);

create table if not exists education_personnel_access_events (
	id uuid primary key default gen_random_uuid(),
	personnel_id uuid not null references education_personnel(id) on delete cascade,
	event_type text not null check (event_type in ('consultare', 'predare', 'actualizare', 'arhivare', 'export')),
	actor_name text not null default '',
	actor_role text not null default '',
	purpose text not null default '',
	access_channel text not null check (access_channel in ('fizic', 'digital', 'mixt')),
	accessed_on date not null,
	closed_on date,
	sensitive_scope boolean not null default false,
	institution_id text not null,
	notes text not null default '',
	created_at timestamptz not null default now(),
	updated_at timestamptz not null default now()
);

create index if not exists idx_education_personnel_access_events_lookup
	on education_personnel_access_events(institution_id, personnel_id, event_type, accessed_on);

create table if not exists education_evaluation_appeals (
	id uuid primary key default gen_random_uuid(),
	evaluation_id uuid not null references education_evaluations(id) on delete cascade,
	appeal_code text not null unique,
	submitted_by text not null default '',
	submitted_on date not null,
	status text not null check (status in ('submitted', 'review', 'accepted', 'rejected', 'resolved')),
	grounds text not null default '',
	hearing_on date,
	resolved_on date,
	decision_summary text not null default '',
	committee_note text not null default '',
	attached_to_personnel_file boolean not null default false,
	institution_id text not null,
	created_at timestamptz not null default now(),
	updated_at timestamptz not null default now()
);

create index if not exists idx_education_evaluation_appeals_lookup
	on education_evaluation_appeals(institution_id, evaluation_id, status, submitted_on);

insert into education_personnel_file_documents (
	personnel_id, document_code, document_category, document_title, file_scope, confidentiality_level,
	issued_on, expires_on, file_reference, sensitive_data, included_in_portfolio, institution_id, notes
)
select
	ep.id,
	'PFD-2026-0001',
	'management',
	'Decizie numire director',
	'dosar_director',
	'confidential',
	date '2025-08-28',
	null,
	'REG-PERS-2025-0008',
	true,
	false,
	ep.institution_id,
	'Document de cariera pastrat exclusiv in dosarul personal al directorului.'
from education_personnel ep
where ep.employee_code = 'PER-2026-0101'
on conflict (personnel_id, document_code) do nothing;

insert into education_personnel_file_documents (
	personnel_id, document_code, document_category, document_title, file_scope, confidentiality_level,
	issued_on, expires_on, file_reference, sensitive_data, included_in_portfolio, institution_id, notes
)
select
	ep.id,
	'PFD-2026-0002',
	'evaluare',
	'Fisa evaluare anuala director adjunct',
	'dosar_director_adjunct',
	'strict_confidential',
	date '2026-05-30',
	null,
	'REG-PERS-2026-0042',
	true,
	false,
	ep.institution_id,
	'Rezultatul evaluarii se arhiveaza separat fata de portofoliul profesional.'
from education_personnel ep
where ep.employee_code = 'PER-2026-0102'
on conflict (personnel_id, document_code) do nothing;

insert into education_personnel_file_documents (
	personnel_id, document_code, document_category, document_title, file_scope, confidentiality_level,
	issued_on, expires_on, file_reference, sensitive_data, included_in_portfolio, institution_id, notes
)
select
	ep.id,
	'PFD-2026-0003',
	'studii',
	'Diploma licenta',
	'dosar_personal',
	'confidential',
	date '2016-07-10',
	null,
	'REG-PERS-2016-0109',
	true,
	true,
	ep.institution_id,
	'Document de studii preluat si ca referinta in portofoliul profesional.'
from education_personnel ep
where ep.employee_code = 'PER-2026-0001'
on conflict (personnel_id, document_code) do nothing;

insert into education_personnel_access_events (
	personnel_id, event_type, actor_name, actor_role, purpose, access_channel, accessed_on, closed_on,
	sensitive_scope, institution_id, notes
)
select
	ep.id,
	'consultare',
	'DPO unitate',
	'Responsabil protectia datelor',
	'Verificare acces si minimizare date pentru publicare procedura evaluare.',
	'digital',
	date '2026-06-05',
	date '2026-06-05',
	true,
	ep.institution_id,
	'Acces punctual la dosarul directorului pentru verificare conformitate.'
from education_personnel ep
where ep.employee_code = 'PER-2026-0101'
on conflict do nothing;

insert into education_personnel_access_events (
	personnel_id, event_type, actor_name, actor_role, purpose, access_channel, accessed_on, closed_on,
	sensitive_scope, institution_id, notes
)
select
	ep.id,
	'actualizare',
	'Secretar sef',
	'Secretariat',
	'Actualizare documente de evaluare si anexe pentru dosarul director adjunct.',
	'fizic',
	date '2026-06-10',
	null,
	true,
	ep.institution_id,
	'Documentele au fost depuse in mapa securizata a compartimentului.'
from education_personnel ep
where ep.employee_code = 'PER-2026-0102'
on conflict do nothing;

insert into education_evaluation_appeals (
	evaluation_id, appeal_code, submitted_by, submitted_on, status, grounds,
	hearing_on, resolved_on, decision_summary, committee_note, attached_to_personnel_file, institution_id
)
select
	ee.id,
	'APEL-2026-0001',
	'Andreea Popescu',
	date '2026-06-02',
	'review',
	'Solicita reanalizarea punctajului pentru activitatea metodica si formare continua.',
	date '2026-06-12',
	null,
	'',
	'Comisia solicita completarea dovezilor pentru activitatile metodice invocate.',
	true,
	ee.institution_id
from education_evaluations ee
where ee.evaluation_code = 'EVAL-2026-001'
on conflict (appeal_code) do nothing;
