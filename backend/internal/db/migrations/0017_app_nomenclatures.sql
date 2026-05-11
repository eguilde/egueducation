create table if not exists app_nomenclatures (
	id uuid primary key default gen_random_uuid(),
	domain text not null,
	code text not null,
	label_ro text not null,
	label_en text not null,
	active boolean not null default true,
	sort_order integer not null default 0,
	created_at timestamptz not null default now(),
	updated_at timestamptz not null default now(),
	unique (domain, code)
);

insert into app_permissions(code, label) values
	('admin.registry_nomenclatures.manage', 'Manage registry nomenclatures'),
	('admin.archive_nomenclatures.manage', 'Manage archive nomenclatures')
on conflict (code) do nothing;

insert into app_user_permissions(user_id, permission_code)
select id, permission_code
from app_users
cross join (values
	('admin.registry_nomenclatures.manage'),
	('admin.archive_nomenclatures.manage')
) as permissions(permission_code)
where sub = 'usr-001'
on conflict do nothing;

insert into app_nomenclatures (domain, code, label_ro, label_en, active, sort_order) values
	('registratura_document_type', 'cerere', 'Cerere', 'Request', true, 10),
	('registratura_document_type', 'adresa', 'Adresă', 'Official letter', true, 20),
	('registratura_document_type', 'decizie', 'Decizie', 'Decision', true, 30),
	('registratura_document_type', 'nota', 'Notă', 'Memo', true, 40),
	('registratura_direction', 'intrare', 'Intrare', 'Incoming', true, 10),
	('registratura_direction', 'iesire', 'Ieșire', 'Outgoing', true, 20),
	('registratura_direction', 'intern', 'Intern', 'Internal', true, 30),
	('registratura_status', 'draft', 'Ciornă', 'Draft', true, 10),
	('registratura_status', 'registered', 'Înregistrat', 'Registered', true, 20),
	('registratura_status', 'in_workflow', 'În workflow', 'In workflow', true, 30),
	('registratura_status', 'archived', 'Arhivat', 'Archived', true, 40),
	('registratura_confidentiality', 'normal', 'Normal', 'Normal', true, 10),
	('registratura_confidentiality', 'internal', 'Intern', 'Internal', true, 20),
	('registratura_confidentiality', 'confidential', 'Confidențial', 'Confidential', true, 30),
	('archive_fond', 'administrativ', 'Administrativ', 'Administrative', true, 10),
	('archive_fond', 'personal', 'Personal', 'Personnel', true, 20),
	('archive_fond', 'elevi', 'Elevi', 'Students', true, 30),
	('archive_series', 'corespondenta', 'Corespondență', 'Correspondence', true, 10),
	('archive_series', 'decizii', 'Decizii', 'Decisions', true, 20),
	('archive_series', 'portofolii', 'Portofolii', 'Portfolios', true, 30),
	('archive_status', 'draft', 'Ciornă', 'Draft', true, 10),
	('archive_status', 'validated', 'Validat', 'Validated', true, 20),
	('archive_status', 'archived', 'Arhivat', 'Archived', true, 30),
	('archive_source_module', 'registratura', 'Registratură', 'Registry', true, 10),
	('archive_source_module', 'education', 'Educație', 'Education', true, 20),
	('archive_source_module', 'gdpr', 'GDPR', 'GDPR', true, 30),
	('archive_source_module', 'earchiva', 'eArhivă', 'eArchive', true, 40)
on conflict (domain, code) do update
set label_ro = excluded.label_ro,
	label_en = excluded.label_en,
	active = excluded.active,
	sort_order = excluded.sort_order,
	updated_at = now();
