create table if not exists education_portfolios (
	id uuid primary key default gen_random_uuid(),
	portfolio_code text not null unique,
	owner_name text not null,
	owner_role text not null,
	school_year text not null,
	status text not null check (status in ('draft', 'submitted', 'validated', 'transferred', 'archived')),
	section_count integer not null default 0 check (section_count >= 0),
	last_updated_on date not null,
	retention_until date not null,
	transfer_status text not null check (transfer_status in ('none', 'prepared', 'sent', 'received')),
	authenticity_declared boolean not null default false,
	consent_captured boolean not null default false,
	custodian text not null default '',
	institution_id text not null,
	notes text not null default '',
	created_at timestamptz not null default now(),
	updated_at timestamptz not null default now()
);

insert into education_portfolios (
	portfolio_code,
	owner_name,
	owner_role,
	school_year,
	status,
	section_count,
	last_updated_on,
	retention_until,
	transfer_status,
	authenticity_declared,
	consent_captured,
	custodian,
	institution_id,
	notes
)
values
	('PORT-CD-2026-0001', 'Maria Popescu', 'Profesor limba română', '2025-2026', 'validated', 14, current_date - 5, current_date + 1095, 'none', true, true, 'Secretar șef', 'inst-001', 'Portofoliu complet pentru evaluarea anuală.'),
	('PORT-CD-2026-0002', 'Andrei Ionescu', 'Profesor matematică', '2025-2026', 'submitted', 11, current_date - 2, current_date + 1095, 'prepared', true, true, 'Responsabil resurse umane', 'inst-001', 'Se pregătește transferul către noua unitate.'),
	('PORT-CD-2026-0003', 'Elena Dumitru', 'Învățător', '2025-2026', 'draft', 6, current_date - 1, current_date + 1095, 'none', false, true, 'Secretariat', 'inst-001', 'Lipsesc documente justificative pentru formare.'),
	('PORT-CD-2026-0004', 'Ioana Marinescu', 'Profesor fizică', '2025-2026', 'transferred', 13, current_date - 12, current_date + 1095, 'received', true, true, 'Secretariat', 'inst-001', 'Portofoliu recepționat după detașare.'),
	('PORT-CD-2026-0005', 'Sorin Pavel', 'Secretar șef', '2024-2025', 'archived', 9, current_date - 90, current_date + 730, 'none', true, true, 'Arhivar', 'inst-001', 'Păstrat conform procedurii interne și regulii de 3 ani.')
on conflict do nothing;

insert into app_permissions(code, label) values
	('education.portfolios.read', 'Read education portfolios'),
	('education.portfolios.manage', 'Manage education portfolios')
on conflict (code) do nothing;

insert into app_user_permissions(user_id, permission_code)
select id, permission_code
from app_users
cross join (values
	('education.portfolios.read'),
	('education.portfolios.manage')
) as permissions(permission_code)
where sub = 'usr-001'
on conflict do nothing;
