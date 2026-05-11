create table if not exists education_taxonomies (
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
	('admin.education_taxonomies.read', 'Read education taxonomies'),
	('admin.education_taxonomies.manage', 'Manage education taxonomies')
on conflict (code) do nothing;

insert into app_user_permissions(user_id, permission_code)
select id, permission_code
from app_users
cross join (values
	('admin.education_taxonomies.read'),
	('admin.education_taxonomies.manage')
) as permissions(permission_code)
where sub = 'usr-001'
on conflict do nothing;

insert into education_taxonomies (domain, code, label_ro, label_en, active, sort_order) values
	('school_year', '2025-2026', 'An școlar 2025-2026', 'School year 2025-2026', true, 10),
	('school_year', '2026-2027', 'An școlar 2026-2027', 'School year 2026-2027', true, 20),
	('governance_organism', 'ca', 'Consiliu de administrație', 'Administrative council', true, 10),
	('governance_organism', 'cp', 'Consiliu profesoral', 'Teachers council', true, 20),
	('governance_organism', 'ceac', 'CEAC', 'Quality assurance committee', true, 30),
	('governance_organism', 'cfdcd', 'CFDCD', 'Continuous development committee', true, 40),
	('governance_meeting_type', 'ordinary', 'Ordinară', 'Ordinary', true, 10),
	('governance_meeting_type', 'extraordinary', 'Extraordinară', 'Extraordinary', true, 20),
	('governance_status', 'draft', 'Ciornă', 'Draft', true, 10),
	('governance_status', 'scheduled', 'Programată', 'Scheduled', true, 20),
	('governance_status', 'held', 'Desfășurată', 'Held', true, 30),
	('governance_status', 'published', 'Publicată', 'Published', true, 40),
	('personnel_employment_type', 'titular', 'Titular', 'Tenured', true, 10),
	('personnel_employment_type', 'suplinitor', 'Suplinitor', 'Substitute', true, 20),
	('personnel_employment_type', 'plata_cu_ora', 'Plata cu ora', 'Hourly', true, 30),
	('personnel_employment_type', 'auxiliar', 'Auxiliar', 'Support staff', true, 40),
	('personnel_status', 'active', 'Activ', 'Active', true, 10),
	('personnel_status', 'on_leave', 'În concediu', 'On leave', true, 20),
	('personnel_status', 'vacant', 'Vacant', 'Vacant', true, 30),
	('personnel_status', 'inactive', 'Inactiv', 'Inactive', true, 40),
	('personnel_evaluation_status', 'draft', 'Ciornă', 'Draft', true, 10),
	('personnel_evaluation_status', 'in_review', 'În evaluare', 'In review', true, 20),
	('personnel_evaluation_status', 'finalized', 'Finalizată', 'Finalized', true, 30),
	('personnel_mobility_stage', 'none', 'Fără mobilitate', 'No mobility', true, 10),
	('personnel_mobility_stage', 'transfer', 'Transfer', 'Transfer', true, 20),
	('personnel_mobility_stage', 'detasare', 'Detașare', 'Secondment', true, 30),
	('personnel_mobility_stage', 'restrangere', 'Restrângere', 'Reduction', true, 40),
	('mobility_request_type', 'transfer', 'Transfer', 'Transfer', true, 10),
	('mobility_request_type', 'detasare', 'Detașare', 'Secondment', true, 20),
	('mobility_request_type', 'pretransfer', 'Pretransfer', 'Pre-transfer', true, 30),
	('mobility_request_type', 'restrangere', 'Restrângere', 'Reduction', true, 40),
	('mobility_stage', 'draft', 'Ciornă', 'Draft', true, 10),
	('mobility_stage', 'submitted', 'Depus', 'Submitted', true, 20),
	('mobility_stage', 'review', 'În verificare', 'In review', true, 30),
	('mobility_stage', 'approved', 'Aprobat', 'Approved', true, 40),
	('mobility_stage', 'completed', 'Finalizat', 'Completed', true, 50),
	('mobility_status', 'open', 'Deschis', 'Open', true, 10),
	('mobility_status', 'pending', 'În așteptare', 'Pending', true, 20),
	('mobility_status', 'approved', 'Aprobat', 'Approved', true, 30),
	('mobility_status', 'rejected', 'Respins', 'Rejected', true, 40),
	('mobility_status', 'completed', 'Finalizat', 'Completed', true, 50),
	('merit_category', 'predare', 'Predare', 'Teaching', true, 10),
	('merit_category', 'management', 'Management', 'Management', true, 20),
	('merit_category', 'consiliere', 'Consiliere', 'Counselling', true, 30),
	('merit_category', 'auxiliar', 'Auxiliar', 'Support staff', true, 40),
	('merit_status', 'draft', 'Ciornă', 'Draft', true, 10),
	('merit_status', 'submitted', 'Depus', 'Submitted', true, 20),
	('merit_status', 'evaluated', 'Evaluat', 'Evaluated', true, 30),
	('merit_status', 'approved', 'Aprobat', 'Approved', true, 40),
	('merit_status', 'funded', 'Finanțat', 'Funded', true, 50),
	('portfolio_status', 'draft', 'Ciornă', 'Draft', true, 10),
	('portfolio_status', 'submitted', 'Depus', 'Submitted', true, 20),
	('portfolio_status', 'validated', 'Validat', 'Validated', true, 30),
	('portfolio_status', 'transferred', 'Transferat', 'Transferred', true, 40),
	('portfolio_status', 'archived', 'Arhivat', 'Archived', true, 50),
	('portfolio_transfer_status', 'none', 'Fără transfer', 'No transfer', true, 10),
	('portfolio_transfer_status', 'prepared', 'Pregătit', 'Prepared', true, 20),
	('portfolio_transfer_status', 'sent', 'Trimis', 'Sent', true, 30),
	('portfolio_transfer_status', 'received', 'Recepționat', 'Received', true, 40)
on conflict (domain, code) do update
set label_ro = excluded.label_ro,
	label_en = excluded.label_en,
	active = excluded.active,
	sort_order = excluded.sort_order,
	updated_at = now();
