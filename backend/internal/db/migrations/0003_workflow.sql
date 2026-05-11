create table if not exists workflow_definitions (
	code text primary key,
	name text not null,
	category text not null,
	initial_step text not null,
	sla_hours integer not null default 72,
	active boolean not null default true
);

create table if not exists workflow_instances (
	id uuid primary key default gen_random_uuid(),
	definition_code text not null references workflow_definitions(code) on delete restrict,
	title text not null,
	document_number text not null default '',
	source_module text not null default 'registratura',
	status text not null check (status in ('new', 'in_progress', 'waiting_approval', 'approved', 'archived')),
	priority text not null check (priority in ('low', 'medium', 'high', 'urgent')),
	assigned_to text not null default '',
	current_step text not null,
	due_at date,
	started_at timestamptz not null default now(),
	updated_at timestamptz not null default now(),
	institution_id text not null,
	summary text not null default ''
);

insert into workflow_definitions(code, name, category, initial_step, sla_hours, active)
values
	('incoming-document', 'Document de intrare', 'registratura', 'Înregistrare', 48, true),
	('board-decision', 'Decizie CA / CP', 'education', 'Pregătire dosar', 120, true),
	('gdpr-request', 'Solicitare GDPR', 'gdpr', 'Validare identitate', 72, true),
	('archive-transfer', 'Transfer la eArhivă', 'earchiva', 'Verificare dosar', 96, true)
on conflict (code) do update
set name = excluded.name,
	category = excluded.category,
	initial_step = excluded.initial_step,
	sla_hours = excluded.sla_hours,
	active = excluded.active;

insert into workflow_instances (
	definition_code,
	title,
	document_number,
	source_module,
	status,
	priority,
	assigned_to,
	current_step,
	due_at,
	started_at,
	updated_at,
	institution_id,
	summary
)
values
	('incoming-document', 'Cerere bursă socială - cls. a IX-a B', 'REG-2026-0001', 'registratura', 'new', 'high', 'Secretariat liceu', 'Înregistrare', current_date + 1, now() - interval '1 hour', now() - interval '1 hour', 'inst-001', 'Document nou pentru bursă socială, necesită verificarea anexelor.'),
	('incoming-document', 'Solicitare adeverință elev transferat', 'REG-2026-0002', 'registratura', 'in_progress', 'medium', 'Compartiment elevi', 'Verificare documente', current_date + 2, now() - interval '6 hours', now() - interval '2 hours', 'inst-001', 'Verificare acte de studii și emitere răspuns.'),
	('board-decision', 'Hotărâre CA privind programul Școala Altfel', 'REG-2026-0003', 'education', 'waiting_approval', 'urgent', 'Director adjunct', 'Aprobare finală', current_date, now() - interval '1 day', now() - interval '40 minutes', 'inst-001', 'Dosarul conține convocator, proces verbal și anexe pentru vot.'),
	('gdpr-request', 'Cerere export date profesor titular', 'REG-2026-0004', 'gdpr', 'in_progress', 'high', 'Responsabil GDPR', 'Colectare date', current_date - 1, now() - interval '2 days', now() - interval '3 hours', 'inst-001', 'Se pregătește exportul conform art. 15 GDPR.'),
	('archive-transfer', 'Predare dosare finalizate semestrul I', 'REG-2026-0005', 'earchiva', 'approved', 'medium', 'Arhivar', 'Pregătit pentru arhivare', current_date + 5, now() - interval '3 days', now() - interval '1 day', 'inst-001', 'Lot de dosare validat și pregătit pentru transfer.'),
	('incoming-document', 'Adresă ISJ privind simulare examene', 'REG-2026-0006', 'registratura', 'archived', 'low', 'Secretariat liceu', 'Arhivat', current_date - 7, now() - interval '10 days', now() - interval '8 days', 'inst-001', 'Flux finalizat și arhivat.'),
	('board-decision', 'Aprobarea planului managerial anual', 'REG-2026-0007', 'education', 'new', 'urgent', 'Director', 'Pregătire dosar', current_date + 4, now() - interval '30 minutes', now() - interval '30 minutes', 'inst-001', 'Necesită încărcarea raportului și avizelor.'),
	('gdpr-request', 'Rectificare date personale elev', 'REG-2026-0008', 'gdpr', 'waiting_approval', 'medium', 'Secretar șef', 'Validare răspuns', current_date + 1, now() - interval '9 hours', now() - interval '90 minutes', 'inst-001', 'Răspunsul este pregătit și așteaptă validarea finală.'),
	('archive-transfer', 'Transfer cataloge promoția 2025', 'REG-2026-0009', 'earchiva', 'in_progress', 'high', 'Arhivar', 'Inventariere', current_date + 7, now() - interval '5 hours', now() - interval '70 minutes', 'inst-001', 'Inventariere și etichetare serii arhivistice.'),
	('incoming-document', 'Cerere concediu fără plată personal auxiliar', 'REG-2026-0010', 'registratura', 'approved', 'medium', 'Resurse umane', 'Finalizat', current_date + 3, now() - interval '14 hours', now() - interval '2 hours', 'inst-001', 'Circuit intern finalizat, pregătit pentru arhivare.')
on conflict do nothing;

insert into app_permissions(code, label) values
	('workflow.manage', 'Manage workflow runtime'),
	('workflow.transition', 'Transition workflow runtime')
on conflict (code) do nothing;

insert into app_user_permissions(user_id, permission_code)
select id, permission_code
from app_users
cross join (values
	('workflow.manage'),
	('workflow.transition')
) as permissions(permission_code)
where sub = 'usr-001'
on conflict do nothing;
