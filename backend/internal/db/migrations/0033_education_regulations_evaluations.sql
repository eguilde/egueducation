create table if not exists education_regulations (
	id uuid primary key default gen_random_uuid(),
	regulation_code text not null unique,
	school_year text not null,
	regulation_type text not null,
	title text not null,
	status text not null,
	approval_status text not null,
	owner_name text not null default '',
	review_due_on date not null,
	approved_on date,
	institution_id text not null,
	summary text not null default '',
	created_at timestamptz not null default now(),
	updated_at timestamptz not null default now()
);

create table if not exists education_evaluations (
	id uuid primary key default gen_random_uuid(),
	evaluation_code text not null unique,
	employee_code text not null,
	full_name text not null,
	role_title text not null,
	school_year text not null,
	status text not null,
	score numeric(5,2) not null default 0,
	evaluator_name text not null default '',
	finalized_on date,
	institution_id text not null,
	summary text not null default '',
	created_at timestamptz not null default now(),
	updated_at timestamptz not null default now()
);

create table if not exists education_declarations (
	id uuid primary key default gen_random_uuid(),
	declaration_code text not null unique,
	employee_code text not null,
	full_name text not null,
	declaration_type text not null,
	status text not null,
	school_year text not null,
	submitted_on date not null,
	valid_until date,
	institution_id text not null,
	summary text not null default '',
	created_at timestamptz not null default now(),
	updated_at timestamptz not null default now()
);

insert into education_taxonomies (domain, code, label_ro, label_en, active, sort_order)
values
	('education_regulation_type', 'rof', 'ROF', 'ROF', true, 10),
	('education_regulation_type', 'roi', 'ROI', 'ROI', true, 20),
	('education_regulation_status', 'draft', 'Proiect', 'Draft', true, 10),
	('education_regulation_status', 'consultation', 'În consultare', 'In consultation', true, 20),
	('education_regulation_status', 'endorsed', 'Avizat', 'Endorsed', true, 30),
	('education_regulation_status', 'approved', 'Aprobat', 'Approved', true, 40),
	('education_regulation_status', 'published', 'Publicat', 'Published', true, 50),
	('education_regulation_approval_status', 'working_group', 'Grup de lucru', 'Working group', true, 10),
	('education_regulation_approval_status', 'cp_endorsed', 'Avizat în CP', 'CP endorsed', true, 20),
	('education_regulation_approval_status', 'ca_approved', 'Aprobat în CA', 'CA approved', true, 30),
	('education_regulation_approval_status', 'registered', 'Înregistrat', 'Registered', true, 40),
	('education_evaluation_status', 'draft', 'Proiect', 'Draft', true, 10),
	('education_evaluation_status', 'submitted', 'Depus', 'Submitted', true, 20),
	('education_evaluation_status', 'reviewed', 'Evaluat', 'Reviewed', true, 30),
	('education_evaluation_status', 'approved', 'Aprobat', 'Approved', true, 40),
	('education_evaluation_status', 'contested', 'Contestat', 'Contested', true, 50),
	('education_declaration_type', 'interests', 'Declarație de interese', 'Interest declaration', true, 10),
	('education_declaration_type', 'assets', 'Declarație de avere', 'Asset declaration', true, 20),
	('education_declaration_type', 'gdpr', 'Declarație GDPR', 'GDPR declaration', true, 30),
	('education_declaration_type', 'authenticity', 'Declarație de autenticitate', 'Authenticity declaration', true, 40),
	('education_declaration_status', 'draft', 'Proiect', 'Draft', true, 10),
	('education_declaration_status', 'submitted', 'Depusă', 'Submitted', true, 20),
	('education_declaration_status', 'validated', 'Validată', 'Validated', true, 30),
	('education_declaration_status', 'expired', 'Expirată', 'Expired', true, 40)
on conflict (domain, code) do update
set label_ro = excluded.label_ro,
	label_en = excluded.label_en,
	active = excluded.active,
	sort_order = excluded.sort_order;

insert into app_permissions(code, label) values
	('education.regulations.read', 'Read education regulations'),
	('education.regulations.manage', 'Manage education regulations'),
	('education.evaluations.read', 'Read education evaluations'),
	('education.evaluations.manage', 'Manage education evaluations'),
	('education.declarations.read', 'Read education declarations'),
	('education.declarations.manage', 'Manage education declarations')
on conflict (code) do nothing;

insert into app_user_permissions (user_id, permission_code)
select ur.user_id, permissions.permission_code
from app_user_roles ur
cross join (values
	('education.regulations.read'),
	('education.regulations.manage'),
	('education.evaluations.read'),
	('education.evaluations.manage'),
	('education.declarations.read'),
	('education.declarations.manage')
) as permissions(permission_code)
where ur.role_code = 'super_admin'
on conflict do nothing;

insert into workflow_definitions(code, name, category, initial_step, sla_hours, active)
values
	('education-regulation-review', 'Education regulation review', 'education.regulations', 'working_group', 96, true),
	('personnel-evaluation-review', 'Personnel evaluation review', 'education.evaluations', 'submission', 72, true),
	('personnel-declaration-review', 'Personnel declaration review', 'education.declarations', 'validation', 48, true)
on conflict (code) do update
set name = excluded.name,
	category = excluded.category,
	initial_step = excluded.initial_step,
	sla_hours = excluded.sla_hours,
	active = excluded.active;

insert into education_regulations (
	regulation_code,
	school_year,
	regulation_type,
	title,
	status,
	approval_status,
	owner_name,
	review_due_on,
	approved_on,
	institution_id,
	summary
)
values (
	'REGL-2026-001',
	'2025-2026',
	'roi',
	'ROI actualizat pentru circuitul documentelor educaționale',
	'consultation',
	'working_group',
	'Thomas Galambos',
	'2026-06-30',
	null,
	'inst-001',
	'Regulament intern pentru consultare, avizare CP, aprobare CA și publicare controlată.'
)
on conflict (regulation_code) do nothing;

insert into education_evaluations (
	evaluation_code,
	employee_code,
	full_name,
	role_title,
	school_year,
	status,
	score,
	evaluator_name,
	finalized_on,
	institution_id,
	summary
)
values (
	'EVAL-2026-001',
	'EMP-2026-001',
	'Andreea Popescu',
	'Profesor limba română',
	'2025-2026',
	'submitted',
	96.50,
	'Director unitate',
	null,
	'inst-001',
	'Dosar anual de evaluare cu fișe, rapoarte și anexele obligatorii.'
)
on conflict (evaluation_code) do nothing;

insert into education_declarations (
	declaration_code,
	employee_code,
	full_name,
	declaration_type,
	status,
	school_year,
	submitted_on,
	valid_until,
	institution_id,
	summary
)
values (
	'DECL-2026-001',
	'EMP-2026-001',
	'Andreea Popescu',
	'authenticity',
	'submitted',
	'2025-2026',
	'2026-05-10',
	'2027-05-10',
	'inst-001',
	'Declarație de autenticitate pentru documentele incluse în portofoliul profesional.'
)
on conflict (declaration_code) do nothing;
