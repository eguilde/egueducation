create table if not exists education_managerial_documents (
	id uuid primary key default gen_random_uuid(),
	dossier_id uuid not null references education_managerial_dossiers(id) on delete cascade,
	document_code text not null,
	document_category text not null check (document_category in ('diagnoza', 'prognoza', 'evidenta', 'planificare', 'raport', 'anexa', 'hotarare', 'procedura')),
	title text not null,
	document_status text not null check (document_status in ('draft', 'in_review', 'approved', 'published', 'archived')),
	version_label text not null,
	mandatory boolean not null default false,
	publication_required boolean not null default false,
	registered_on date not null,
	approved_on date,
	owner_name text not null default '',
	file_reference text not null default '',
	institution_id text not null,
	notes text not null default '',
	created_at timestamptz not null default now(),
	updated_at timestamptz not null default now(),
	unique (dossier_id, document_code)
);

create index if not exists idx_education_managerial_documents_lookup
	on education_managerial_documents(institution_id, dossier_id, document_status, registered_on);

create table if not exists education_managerial_workflow_steps (
	id uuid primary key default gen_random_uuid(),
	dossier_id uuid not null references education_managerial_dossiers(id) on delete cascade,
	stage_order integer not null check (stage_order > 0),
	stage_type text not null check (stage_type in ('elaborare', 'verificare_secretariat', 'avizare_cp', 'aprobare_ca', 'publicare', 'arhivare')),
	status text not null check (status in ('pending', 'in_progress', 'completed', 'returned', 'waived')),
	assigned_to text not null default '',
	due_on date not null,
	completed_on date,
	requires_signature boolean not null default false,
	decision_reference text not null default '',
	institution_id text not null,
	outcome_note text not null default '',
	created_at timestamptz not null default now(),
	updated_at timestamptz not null default now(),
	unique (dossier_id, stage_order, stage_type)
);

create index if not exists idx_education_managerial_workflow_lookup
	on education_managerial_workflow_steps(institution_id, dossier_id, status, due_on);

create table if not exists education_regulation_versions (
	id uuid primary key default gen_random_uuid(),
	regulation_id uuid not null references education_regulations(id) on delete cascade,
	version_label text not null,
	version_status text not null check (version_status in ('draft', 'consultation', 'endorsed', 'approved', 'published', 'retired')),
	change_summary text not null,
	approved_on date,
	effective_from date not null,
	published_on date,
	prepared_by text not null default '',
	file_reference text not null default '',
	institution_id text not null,
	notes text not null default '',
	created_at timestamptz not null default now(),
	updated_at timestamptz not null default now(),
	unique (regulation_id, version_label)
);

create index if not exists idx_education_regulation_versions_lookup
	on education_regulation_versions(institution_id, regulation_id, version_status, effective_from);

create table if not exists education_regulation_workflow_steps (
	id uuid primary key default gen_random_uuid(),
	regulation_id uuid not null references education_regulations(id) on delete cascade,
	phase_order integer not null check (phase_order > 0),
	phase_type text not null check (phase_type in ('redactare', 'consultare_publica', 'avizare_cp', 'aprobare_ca', 'inregistrare', 'publicare')),
	status text not null check (status in ('pending', 'active', 'completed', 'returned', 'cancelled')),
	audience text not null default '',
	started_on date not null,
	due_on date not null,
	completed_on date,
	feedback_count integer not null default 0 check (feedback_count >= 0),
	decision_reference text not null default '',
	institution_id text not null,
	notes text not null default '',
	created_at timestamptz not null default now(),
	updated_at timestamptz not null default now(),
	unique (regulation_id, phase_order, phase_type)
);

create index if not exists idx_education_regulation_workflow_lookup
	on education_regulation_workflow_steps(institution_id, regulation_id, status, due_on);

insert into education_managerial_documents (
	dossier_id, document_code, document_category, title, document_status, version_label,
	mandatory, publication_required, registered_on, approved_on, owner_name, file_reference, institution_id, notes
)
select
	emd.id,
	'MDOC-2026-0001',
	'planificare',
	'Plan managerial anual - forma de lucru',
	'in_review',
	'v0.9',
	true,
	true,
	date '2026-05-20',
	null,
	'Thomas Galambos',
	'REG-MGR-2026-0154',
	emd.institution_id,
	'Versiunea transmisa pentru verificare si avizare interna.'
from education_managerial_dossiers emd
where emd.dossier_code = 'MGR-2026-001'
on conflict (dossier_id, document_code) do nothing;

insert into education_managerial_documents (
	dossier_id, document_code, document_category, title, document_status, version_label,
	mandatory, publication_required, registered_on, approved_on, owner_name, file_reference, institution_id, notes
)
select
	emd.id,
	'MDOC-2026-0002',
	'diagnoza',
	'Analiza de nevoi institutionale',
	'approved',
	'v1.0',
	true,
	false,
	date '2026-05-12',
	date '2026-05-18',
	'Director unitate',
	'REG-MGR-2026-0121',
	emd.institution_id,
	'Document suport pentru fundamentarea obiectivelor din planul managerial.'
from education_managerial_dossiers emd
where emd.dossier_code = 'MGR-2026-001'
on conflict (dossier_id, document_code) do nothing;

insert into education_managerial_workflow_steps (
	dossier_id, stage_order, stage_type, status, assigned_to, due_on, completed_on,
	requires_signature, decision_reference, institution_id, outcome_note
)
select
	emd.id,
	1,
	'elaborare',
	'completed',
	'Director unitate',
	date '2026-05-18',
	date '2026-05-17',
	false,
	'',
	emd.institution_id,
	'Continutul initial a fost consolidat pe baza documentelor de diagnoza.'
from education_managerial_dossiers emd
where emd.dossier_code = 'MGR-2026-001'
on conflict (dossier_id, stage_order, stage_type) do nothing;

insert into education_managerial_workflow_steps (
	dossier_id, stage_order, stage_type, status, assigned_to, due_on, completed_on,
	requires_signature, decision_reference, institution_id, outcome_note
)
select
	emd.id,
	2,
	'avizare_cp',
	'in_progress',
	'Secretar CP',
	date '2026-06-12',
	null,
	true,
	'PV-CP-2026-07',
	emd.institution_id,
	'Se asteapta observatiile finale din sedinta CP.'
from education_managerial_dossiers emd
where emd.dossier_code = 'MGR-2026-001'
on conflict (dossier_id, stage_order, stage_type) do nothing;

insert into education_regulation_versions (
	regulation_id, version_label, version_status, change_summary, approved_on, effective_from,
	published_on, prepared_by, file_reference, institution_id, notes
)
select
	er.id,
	'v0.8',
	'consultation',
	'Introducerea regulilor de circuit documente, publicare si anonimizare.',
	null,
	date '2026-09-01',
	null,
	'Thomas Galambos',
	'REG-ROI-2026-0008',
	er.institution_id,
	'Varianta pusa in consultare publica pe site-ul unitatii.'
from education_regulations er
where er.regulation_code = 'REGL-2026-001'
on conflict (regulation_id, version_label) do nothing;

insert into education_regulation_versions (
	regulation_id, version_label, version_status, change_summary, approved_on, effective_from,
	published_on, prepared_by, file_reference, institution_id, notes
)
select
	er.id,
	'v0.7',
	'endorsed',
	'Preluare observatii din comisia de redactare si aviz CP.',
	date '2026-05-28',
	date '2026-09-01',
	null,
	'Secretar CA',
	'REG-ROI-2026-0006',
	er.institution_id,
	'Versiune avizata in CP si pregatita pentru aprobare.'
from education_regulations er
where er.regulation_code = 'REGL-2026-001'
on conflict (regulation_id, version_label) do nothing;

insert into education_regulation_workflow_steps (
	regulation_id, phase_order, phase_type, status, audience, started_on, due_on, completed_on,
	feedback_count, decision_reference, institution_id, notes
)
select
	er.id,
	1,
	'redactare',
	'completed',
	'Grup de lucru ROI',
	date '2026-05-01',
	date '2026-05-20',
	date '2026-05-18',
	0,
	'',
	er.institution_id,
	'Textul consolidat a fost finalizat pentru consultare.'
from education_regulations er
where er.regulation_code = 'REGL-2026-001'
on conflict (regulation_id, phase_order, phase_type) do nothing;

insert into education_regulation_workflow_steps (
	regulation_id, phase_order, phase_type, status, audience, started_on, due_on, completed_on,
	feedback_count, decision_reference, institution_id, notes
)
select
	er.id,
	2,
	'consultare_publica',
	'active',
	'Personal, parinti, elevi majori',
	date '2026-06-03',
	date '2026-06-18',
	null,
	14,
	'ANUNT-ROI-2026-03',
	er.institution_id,
	'Observatiile se centralizeaza pentru forma finala ROI.'
from education_regulations er
where er.regulation_code = 'REGL-2026-001'
on conflict (regulation_id, phase_order, phase_type) do nothing;
