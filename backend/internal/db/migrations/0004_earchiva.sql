create table if not exists archive_records (
	id uuid primary key default gen_random_uuid(),
	record_number text not null unique,
	title text not null,
	fond text not null,
	series text not null,
	source_module text not null,
	source_reference text not null default '',
	status text not null check (status in ('draft', 'validated', 'archived')),
	retention_years integer not null check (retention_years >= 1),
	assigned_archivist text not null default '',
	box_number text not null default '',
	location_code text not null default '',
	archived_at date not null,
	institution_id text not null,
	notes text not null default '',
	created_at timestamptz not null default now(),
	updated_at timestamptz not null default now()
);

insert into archive_records (
	record_number,
	title,
	fond,
	series,
	source_module,
	source_reference,
	status,
	retention_years,
	assigned_archivist,
	box_number,
	location_code,
	archived_at,
	institution_id,
	notes
)
values
	('ARH-2026-0001', 'Dosare bursă socială semestrul I', 'Fond elevi', 'Seria burse', 'registratura', 'REG-2026-0001', 'validated', 5, 'Maria Arhivar', 'BX-11', 'R1-S2-P03', current_date - 3, 'inst-001', 'Lot pregătit pentru predare finală.'),
	('ARH-2026-0002', 'Hotărâri CA 2025-2026', 'Fond conducere', 'Seria decizii CA', 'education', 'REG-2026-0003', 'archived', 10, 'Maria Arhivar', 'BX-04', 'R1-S1-P02', current_date - 12, 'inst-001', 'Conține procese verbale și anexe.'),
	('ARH-2026-0003', 'Cereri GDPR soluționate trimestrul I', 'Fond GDPR', 'Seria cereri persoane vizate', 'gdpr', 'REG-2026-0004', 'validated', 3, 'Andrei Pop', 'BX-19', 'R2-S3-P01', current_date - 1, 'inst-001', 'Necesită confirmarea registrului de evidență.'),
	('ARH-2026-0004', 'Cataloge promoția 2025', 'Fond școlaritate', 'Seria cataloage finale', 'earchiva', 'REG-2026-0009', 'draft', 50, 'Maria Arhivar', 'BX-22', 'R3-S1-P07', current_date, 'inst-001', 'În curs de inventariere finală.'),
	('ARH-2026-0005', 'Plan managerial anual aprobat', 'Fond conducere', 'Seria planuri manageriale', 'education', 'REG-2026-0007', 'validated', 10, 'Director adjunct', 'BX-08', 'R1-S4-P04', current_date - 6, 'inst-001', 'Urmează semnarea procesului verbal de predare.')
on conflict do nothing;

insert into app_permissions(code, label) values
	('earchiva.manage', 'Manage archive records')
on conflict (code) do nothing;

insert into app_user_permissions(user_id, permission_code)
select id, permission_code
from app_users
cross join (values ('earchiva.manage')) as permissions(permission_code)
where sub = 'usr-001'
on conflict do nothing;
