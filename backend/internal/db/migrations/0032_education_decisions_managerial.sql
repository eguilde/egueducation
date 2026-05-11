create table if not exists education_decisions (
	id uuid primary key default gen_random_uuid(),
	decision_code text not null unique,
	school_year text not null,
	organism text not null,
	title text not null,
	status text not null,
	publication_status text not null,
	decision_date date not null,
	legal_basis text not null default '',
	signed_by text not null default '',
	institution_id text not null,
	summary text not null default '',
	created_at timestamptz not null default now(),
	updated_at timestamptz not null default now()
);

create table if not exists education_managerial_dossiers (
	id uuid primary key default gen_random_uuid(),
	dossier_code text not null unique,
	school_year text not null,
	dossier_type text not null,
	title text not null,
	status text not null,
	owner_name text not null default '',
	due_on date not null,
	publication_required boolean not null default false,
	institution_id text not null,
	summary text not null default '',
	created_at timestamptz not null default now(),
	updated_at timestamptz not null default now()
);

insert into education_taxonomies (domain, code, label_ro, label_en, active, sort_order)
values
	('governance_decision_status', 'draft', 'Proiect', 'Draft', true, 10),
	('governance_decision_status', 'endorsed', 'Avizat', 'Endorsed', true, 20),
	('governance_decision_status', 'approved', 'Aprobat', 'Approved', true, 30),
	('governance_decision_status', 'published', 'Publicat', 'Published', true, 40),
	('governance_decision_status', 'blocked', 'Blocat juridic', 'Legally blocked', true, 50),
	('governance_publication_status', 'internal', 'Intern', 'Internal', true, 10),
	('governance_publication_status', 'pending_anonymization', 'În așteptarea anonimizării', 'Pending anonymization', true, 20),
	('governance_publication_status', 'published', 'Publicat', 'Published', true, 30),
	('managerial_dossier_type', 'pdi_pas', 'PDI / PAS', 'PDI / PAS', true, 10),
	('managerial_dossier_type', 'annual_plan', 'Plan managerial anual', 'Annual managerial plan', true, 20),
	('managerial_dossier_type', 'raei', 'RAEI', 'RAEI', true, 30),
	('managerial_dossier_type', 'organigram', 'Organigramă', 'Organigram', true, 40),
	('managerial_dossier_type', 'staffing_plan', 'Plan de încadrare', 'Staffing plan', true, 50),
	('managerial_dossier_type', 'timetable', 'Orar', 'Timetable', true, 60),
	('managerial_dossier_type', 'commission_report', 'Raport de comisie', 'Commission report', true, 70),
	('managerial_dossier_status', 'draft', 'Proiect', 'Draft', true, 10),
	('managerial_dossier_status', 'in_review', 'În avizare', 'In review', true, 20),
	('managerial_dossier_status', 'approved', 'Aprobat', 'Approved', true, 30),
	('managerial_dossier_status', 'published', 'Publicat', 'Published', true, 40),
	('managerial_dossier_status', 'archived', 'Arhivat', 'Archived', true, 50)
on conflict (domain, code) do update
set label_ro = excluded.label_ro,
	label_en = excluded.label_en,
	active = excluded.active,
	sort_order = excluded.sort_order;

insert into app_permissions(code, label) values
	('education.decisions.read', 'Read education governance decisions'),
	('education.decisions.manage', 'Manage education governance decisions'),
	('education.managerial.read', 'Read education managerial dossiers'),
	('education.managerial.manage', 'Manage education managerial dossiers')
on conflict (code) do nothing;

insert into app_user_permissions (user_id, permission_code)
select ur.user_id, permissions.permission_code
from app_user_roles ur
cross join (values
	('education.decisions.read'),
	('education.decisions.manage'),
	('education.managerial.read'),
	('education.managerial.manage')
) as permissions(permission_code)
where ur.role_code = 'super_admin'
on conflict do nothing;

insert into workflow_definitions(code, name, category, initial_step, sla_hours, active)
values
	('governance-decision-review', 'Governance decision review', 'education.decisions', 'legal_review', 72, true),
	('managerial-dossier-review', 'Managerial dossier review', 'education.managerial', 'drafting', 120, true)
on conflict (code) do update
set name = excluded.name,
	category = excluded.category,
	initial_step = excluded.initial_step,
	sla_hours = excluded.sla_hours,
	active = excluded.active;

insert into education_decisions (
	decision_code,
	school_year,
	organism,
	title,
	status,
	publication_status,
	decision_date,
	legal_basis,
	signed_by,
	institution_id,
	summary
)
values (
	'DEC-2026-001',
	'2025-2026',
	'ca',
	'Aprobarea calendarului intern pentru portofolii CD',
	'approved',
	'pending_anonymization',
	'2026-05-10',
	'ROI și hotărâre CA',
	'Director unitate',
	'inst-001',
	'Decizie pentru aprobarea calendarului de depunere, validare și păstrare a portofoliilor CD.'
)
on conflict (decision_code) do nothing;

insert into education_managerial_dossiers (
	dossier_code,
	school_year,
	dossier_type,
	title,
	status,
	owner_name,
	due_on,
	publication_required,
	institution_id,
	summary
)
values (
	'MGR-2026-001',
	'2025-2026',
	'annual_plan',
	'Plan managerial anual 2025-2026',
	'in_review',
	'Thomas Galambos',
	'2026-06-15',
	true,
	'inst-001',
	'Dosar managerial pentru planificarea anuală, circuit de avizare și publicare controlată.'
)
on conflict (dossier_code) do nothing;
