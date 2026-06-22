create table if not exists education_decision_issuances (
	id uuid primary key default gen_random_uuid(),
	decision_id uuid not null references education_decisions(id) on delete cascade,
	issuance_code text not null unique,
	document_type text not null check (document_type in ('decizie', 'extras', 'comunicare', 'adeverinta', 'dispozitie')),
	recipient_name text not null default '',
	recipient_role text not null default '',
	delivery_channel text not null check (delivery_channel in ('intern', 'email', 'registratura', 'avizier', 'site')),
	delivery_status text not null check (delivery_status in ('draft', 'semnat', 'transmis', 'confirmat', 'returnat')),
	signed_on date,
	delivered_on date,
	acknowledged_on date,
	file_reference text not null default '',
	institution_id text not null,
	notes text not null default '',
	created_at timestamptz not null default now(),
	updated_at timestamptz not null default now()
);

create index if not exists education_decision_issuances_decision_idx
	on education_decision_issuances(decision_id, institution_id, delivery_status, signed_on);

create table if not exists education_decision_publication_steps (
	id uuid primary key default gen_random_uuid(),
	decision_id uuid not null references education_decisions(id) on delete cascade,
	step_order integer not null check (step_order > 0),
	step_type text not null check (step_type in ('analiza_juridica', 'anonimizare', 'aprobare_publicare', 'publicare', 'retragere')),
	status text not null check (status in ('pending', 'in_progress', 'completed', 'blocked')),
	responsible_name text not null default '',
	publication_channel text not null default '',
	due_on date not null,
	completed_on date,
	publication_reference text not null default '',
	institution_id text not null,
	notes text not null default '',
	created_at timestamptz not null default now(),
	updated_at timestamptz not null default now()
);

create index if not exists education_decision_publication_steps_decision_idx
	on education_decision_publication_steps(decision_id, institution_id, step_order, status);

insert into app_permissions(code, label) values
	('education.compliance.read', 'Read education compliance publication workflows'),
	('education.compliance.manage', 'Manage education compliance publication workflows'),
	('education.decisions.issuance.read', 'Read education decision issuance flows'),
	('education.decisions.issuance.manage', 'Manage education decision issuance flows'),
	('education.personnel.files.read', 'Read education personnel file documents'),
	('education.personnel.files.manage', 'Manage education personnel file documents'),
	('education.personnel.access.read', 'Read education personnel access events'),
	('education.personnel.access.manage', 'Manage education personnel access events')
on conflict (code) do nothing;

insert into app_role_permissions (role_code, permission_code)
select role_code, permission_code
from (values
	('super_admin', 'education.compliance.read'),
	('super_admin', 'education.compliance.manage'),
	('super_admin', 'education.decisions.issuance.read'),
	('super_admin', 'education.decisions.issuance.manage'),
	('super_admin', 'education.personnel.files.read'),
	('super_admin', 'education.personnel.files.manage'),
	('super_admin', 'education.personnel.access.read'),
	('super_admin', 'education.personnel.access.manage'),
	('admin', 'education.compliance.read'),
	('admin', 'education.compliance.manage'),
	('admin', 'education.decisions.issuance.read'),
	('admin', 'education.decisions.issuance.manage'),
	('admin', 'education.personnel.files.read'),
	('admin', 'education.personnel.files.manage'),
	('admin', 'education.personnel.access.read'),
	('admin', 'education.personnel.access.manage'),
	('director', 'education.compliance.read'),
	('director', 'education.compliance.manage'),
	('director', 'education.decisions.issuance.read'),
	('director', 'education.decisions.issuance.manage'),
	('director', 'education.personnel.files.read'),
	('director', 'education.personnel.files.manage'),
	('director', 'education.personnel.access.read'),
	('director', 'education.personnel.access.manage'),
	('secretar', 'education.compliance.read'),
	('secretar', 'education.compliance.manage'),
	('secretar', 'education.decisions.issuance.read'),
	('secretar', 'education.decisions.issuance.manage'),
	('secretar', 'education.personnel.files.read'),
	('secretar', 'education.personnel.files.manage'),
	('secretar', 'education.personnel.access.read'),
	('secretar', 'education.personnel.access.manage'),
	('gdpr_officer', 'education.compliance.read'),
	('gdpr_officer', 'education.compliance.manage'),
	('gdpr_officer', 'education.personnel.files.read'),
	('gdpr_officer', 'education.personnel.access.read'),
	('gdpr_officer', 'education.personnel.access.manage'),
	('inspector', 'education.compliance.read'),
	('inspector', 'education.decisions.issuance.read'),
	('inspector', 'education.personnel.files.read'),
	('inspector', 'education.personnel.access.read')
) as mapping(role_code, permission_code)
on conflict do nothing;

insert into education_decision_issuances (
	decision_id,
	issuance_code,
	document_type,
	recipient_name,
	recipient_role,
	delivery_channel,
	delivery_status,
	signed_on,
	delivered_on,
	acknowledged_on,
	file_reference,
	institution_id,
	notes
)
select
	ed.id,
	'DEC-OUT-2026-0001',
	'decizie',
	'Compartiment resurse umane',
	'Executare măsuri',
	'registratura',
	'confirmat',
	current_date - 41,
	current_date - 40,
	current_date - 39,
	'REG-DEC-2026-0001',
	ed.institution_id,
	'Exemplar comunicat pentru punerea în aplicare și confirmarea de primire.'
from education_decisions ed
where ed.decision_code = 'DEC-2026-001'
on conflict (issuance_code) do nothing;

insert into education_decision_publication_steps (
	decision_id,
	step_order,
	step_type,
	status,
	responsible_name,
	publication_channel,
	due_on,
	completed_on,
	publication_reference,
	institution_id,
	notes
)
select
	ed.id,
	step.step_order,
	step.step_type,
	step.status,
	step.responsible_name,
	step.publication_channel,
	step.due_on,
	step.completed_on,
	step.publication_reference,
	ed.institution_id,
	step.notes
from education_decisions ed
cross join (values
	(1, 'analiza_juridica', 'completed', 'Consilier juridic', 'intranet', current_date - 44, current_date - 43, 'AVJ-2026-001', 'Verificare bază legală și necesitate publicare.'),
	(2, 'anonimizare', 'completed', 'Responsabil GDPR', 'site_public', current_date - 42, current_date - 41, 'ANON-2026-001', 'Au fost eliminate datele personale din anexele publicabile.'),
	(3, 'aprobare_publicare', 'completed', 'Director', 'site_public', current_date - 41, current_date - 40, 'APUB-2026-001', 'Aprobarea finală pentru publicare controlată.'),
	(4, 'publicare', 'completed', 'Secretariat', 'site_public', current_date - 40, current_date - 40, 'PUB-DEC-2026-001', 'Documentul a fost publicat în secțiunea de transparență.')
) as step(step_order, step_type, status, responsible_name, publication_channel, due_on, completed_on, publication_reference, notes)
where ed.decision_code = 'DEC-2026-001'
on conflict do nothing;
