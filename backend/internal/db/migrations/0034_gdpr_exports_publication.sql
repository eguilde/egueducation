create table if not exists gdpr_subject_exports (
	id uuid primary key default gen_random_uuid(),
	export_code text not null unique,
	request_id uuid references gdpr_subject_requests(id) on delete set null,
	subject_name text not null,
	source_module text not null,
	status text not null,
	export_format text not null,
	approved_by text not null default '',
	approved_on date,
	generated_on date,
	package_summary text not null default '',
	institution_id text not null,
	notes text not null default '',
	created_at timestamptz not null default now(),
	updated_at timestamptz not null default now()
);

create table if not exists gdpr_publication_reviews (
	id uuid primary key default gen_random_uuid(),
	review_code text not null unique,
	source_module text not null,
	source_record_id text not null,
	source_label text not null,
	anonymization_status text not null,
	publication_status text not null,
	reviewed_by text not null default '',
	reviewed_on date,
	legal_basis text not null default '',
	institution_id text not null,
	notes text not null default '',
	created_at timestamptz not null default now(),
	updated_at timestamptz not null default now()
);

insert into app_nomenclatures (domain, code, label_ro, label_en, active, sort_order)
values
	('gdpr_export_status', 'draft', 'Proiect', 'Draft', true, 10),
	('gdpr_export_status', 'pending_approval', 'În așteptarea aprobării', 'Pending approval', true, 20),
	('gdpr_export_status', 'approved', 'Aprobat', 'Approved', true, 30),
	('gdpr_export_status', 'generated', 'Generat', 'Generated', true, 40),
	('gdpr_export_status', 'delivered', 'Livrat', 'Delivered', true, 50),
	('gdpr_export_format', 'pdf_bundle', 'Pachet PDF', 'PDF bundle', true, 10),
	('gdpr_export_format', 'zip_archive', 'Arhivă ZIP', 'ZIP archive', true, 20),
	('gdpr_export_format', 'structured_json', 'JSON structurat', 'Structured JSON', true, 30),
	('gdpr_anonymization_status', 'not_required', 'Nu este necesară', 'Not required', true, 10),
	('gdpr_anonymization_status', 'pending', 'În așteptare', 'Pending', true, 20),
	('gdpr_anonymization_status', 'completed', 'Finalizată', 'Completed', true, 30),
	('gdpr_publication_status', 'internal', 'Intern', 'Internal', true, 10),
	('gdpr_publication_status', 'blocked', 'Blocată', 'Blocked', true, 20),
	('gdpr_publication_status', 'ready', 'Pregătită', 'Ready', true, 30),
	('gdpr_publication_status', 'published', 'Publicată', 'Published', true, 40),
	('gdpr_publication_source_module', 'education.decisions', 'Decizii educaționale', 'Education decisions', true, 10),
	('gdpr_publication_source_module', 'education.managerial', 'Dosare manageriale', 'Managerial dossiers', true, 20),
	('gdpr_publication_source_module', 'education.regulations', 'Regulamente educaționale', 'Education regulations', true, 30)
on conflict (domain, code) do update
set label_ro = excluded.label_ro,
	label_en = excluded.label_en,
	active = excluded.active,
	sort_order = excluded.sort_order;

insert into app_permissions(code, label) values
	('gdpr.exports.read', 'Read GDPR subject exports'),
	('gdpr.exports.manage', 'Manage GDPR subject exports'),
	('gdpr.publication.read', 'Read GDPR publication reviews'),
	('gdpr.publication.manage', 'Manage GDPR publication reviews')
on conflict (code) do nothing;

insert into app_user_permissions (user_id, permission_code)
select ur.user_id, permissions.permission_code
from app_user_roles ur
cross join (values
	('gdpr.exports.read'),
	('gdpr.exports.manage'),
	('gdpr.publication.read'),
	('gdpr.publication.manage')
) as permissions(permission_code)
where ur.role_code = 'super_admin'
on conflict do nothing;

insert into workflow_definitions(code, name, category, initial_step, sla_hours, active)
values
	('gdpr-subject-export', 'GDPR subject export', 'gdpr.exports', 'approval', 48, true),
	('gdpr-publication-review', 'GDPR publication review', 'gdpr.publication', 'anonymization', 24, true)
on conflict (code) do update
set name = excluded.name,
	category = excluded.category,
	initial_step = excluded.initial_step,
	sla_hours = excluded.sla_hours,
	active = excluded.active;

insert into gdpr_subject_exports (
	export_code,
	request_id,
	subject_name,
	source_module,
	status,
	export_format,
	approved_by,
	approved_on,
	generated_on,
	package_summary,
	institution_id,
	notes
)
select
	'EXP-2026-0001',
	gsr.id,
	gsr.subject_name,
	gsr.source_module,
	'pending_approval',
	'pdf_bundle',
	'Responsabil GDPR',
	null,
	null,
	'Pachet export pentru evaluări, dosare manageriale și decizii asociate.',
	'inst-001',
	'Export pregătit pentru aprobare înainte de livrarea către persoana vizată.'
from gdpr_subject_requests gsr
where gsr.request_code = 'DSR-2026-0001'
on conflict (export_code) do nothing;

insert into gdpr_publication_reviews (
	review_code,
	source_module,
	source_record_id,
	source_label,
	anonymization_status,
	publication_status,
	reviewed_by,
	reviewed_on,
	legal_basis,
	institution_id,
	notes
)
select
	'PUB-2026-0001',
	'education.decisions',
	ed.id::text,
	ed.decision_code || ' - ' || ed.title,
	'pending',
	'blocked',
	'Responsabil GDPR',
	null,
	'Regulamentul (UE) 2016/679 și procedura internă de publicare',
	'inst-001',
	'Publicarea este blocată până la finalizarea anonimizării.'
from education_decisions ed
where ed.decision_code = 'DEC-2026-001'
on conflict (review_code) do nothing;
