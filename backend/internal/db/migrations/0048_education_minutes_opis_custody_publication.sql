create table if not exists education_meeting_minutes (
	id uuid primary key default gen_random_uuid(),
	meeting_id uuid not null references education_meetings(id) on delete cascade,
	agenda_order integer not null check (agenda_order >= 1),
	topic_title text not null,
	discussion_summary text not null,
	decision_summary text not null,
	responsible_party text not null default '',
	due_on date,
	follow_up_status text not null check (follow_up_status in ('de_stabilit', 'in_urmarire', 'realizat', 'amanat', 'inchis')),
	requires_publication boolean not null default false,
	institution_id text not null,
	notes text not null default '',
	created_at timestamptz not null default now(),
	updated_at timestamptz not null default now(),
	unique (meeting_id, agenda_order, topic_title)
);

create index if not exists education_meeting_minutes_meeting_idx
	on education_meeting_minutes(meeting_id, institution_id, agenda_order);

create table if not exists education_portfolio_opis (
	id uuid primary key default gen_random_uuid(),
	portfolio_id uuid not null references education_portfolios(id) on delete cascade,
	section_code text not null,
	component_code text not null,
	entry_title text not null,
	source_scope text not null check (source_scope in ('portofoliu', 'dosar_personal')),
	chronological_index integer not null default 0 check (chronological_index >= 0),
	document_reference text not null,
	included_in_transfer boolean not null default false,
	checked_on date not null,
	checked_by text not null default '',
	institution_id text not null,
	notes text not null default '',
	created_at timestamptz not null default now(),
	updated_at timestamptz not null default now(),
	unique (portfolio_id, document_reference)
);

create index if not exists education_portfolio_opis_portfolio_idx
	on education_portfolio_opis(portfolio_id, institution_id, chronological_index);

create table if not exists education_portfolio_custody (
	id uuid primary key default gen_random_uuid(),
	portfolio_id uuid not null references education_portfolios(id) on delete cascade,
	event_type text not null check (event_type in ('preluare', 'consultare', 'transfer', 'arhivare', 'restituire')),
	holder_name text not null,
	holder_role text not null,
	location_label text not null,
	access_reason text not null,
	started_on date not null,
	ended_on date,
	access_mode text not null check (access_mode in ('fizic', 'digital', 'mixt')),
	sensitive_data_access boolean not null default false,
	institution_id text not null,
	notes text not null default '',
	created_at timestamptz not null default now(),
	updated_at timestamptz not null default now()
);

create index if not exists education_portfolio_custody_portfolio_idx
	on education_portfolio_custody(portfolio_id, institution_id, started_on);

create table if not exists education_publications (
	id uuid primary key default gen_random_uuid(),
	publication_code text not null,
	domain text not null check (domain in ('guvernanta', 'documente_manageriale', 'portofolii', 'regulamente', 'conformitate')),
	entity_type text not null check (entity_type in ('hotarare', 'proces_verbal', 'procedura_portofoliu', 'rof', 'roi', 'pdi_pas', 'raport', 'anunt')),
	entity_label text not null,
	publication_channel text not null check (publication_channel in ('site_public', 'avizier', 'intranet', 'registratura')),
	publication_status text not null check (publication_status in ('pregatit', 'publicat', 'retras')),
	anonymization_status text not null check (anonymization_status in ('necesara', 'finalizata', 'nu_este_necesara')),
	mandatory boolean not null default false,
	published_on date,
	reviewed_by text not null default '',
	institution_id text not null,
	notes text not null default '',
	created_at timestamptz not null default now(),
	updated_at timestamptz not null default now(),
	unique (institution_id, publication_code)
);

create index if not exists education_publications_institution_idx
	on education_publications(institution_id, domain, publication_status, published_on);

insert into education_meeting_minutes (
	meeting_id, agenda_order, topic_title, discussion_summary, decision_summary, responsible_party, due_on, follow_up_status, requires_publication, institution_id, notes
)
select
	em.id,
	minute.agenda_order,
	minute.topic_title,
	minute.discussion_summary,
	minute.decision_summary,
	minute.responsible_party,
	minute.due_on,
	minute.follow_up_status,
	minute.requires_publication,
	em.institution_id,
	minute.notes
from education_meetings em
cross join (
	values
		(1, 'Aprobarea proiectului de buget și a listei de investiții IT', 'S-au analizat prioritățile de investiții și impactul bugetar asupra laboratorului digital.', 'Se aprobă proiectul de buget și se mandatează compartimentul financiar pentru implementare.', 'Director', current_date + 10, 'in_urmarire', true, 'Necesită publicarea hotărârii în formă anonimizată.'),
		(2, 'Validarea planului de achiziții pentru laborator', 'Membrii au confirmat necesarul de echipamente și conformitatea cu planul managerial.', 'Se acordă aviz favorabil fără completări suplimentare.', 'Administrator patrimoniu', current_date + 20, 'de_stabilit', false, 'Se atașează la dosarul anual de achiziții.'),
		(3, 'Propunere de amânare pentru reorganizarea programului', 'S-a constatat lipsa unor documente justificative și necesitatea unei reanalize.', 'Punctul se amână pentru ședința următoare, după completarea documentației.', 'Secretar CA', null, 'amanat', false, 'Se reia pe ordinea de zi după depunerea anexelor lipsă.')
) as minute(agenda_order, topic_title, discussion_summary, decision_summary, responsible_party, due_on, follow_up_status, requires_publication, notes)
where em.title = 'Avizarea proiectului de buget'
on conflict do nothing;

insert into education_portfolio_opis (
	portfolio_id, section_code, component_code, entry_title, source_scope, chronological_index, document_reference,
	included_in_transfer, checked_on, checked_by, institution_id, notes
)
select
	ep.id,
	opis.section_code,
	opis.component_code,
	opis.entry_title,
	opis.source_scope,
	opis.chronological_index,
	opis.document_reference,
	opis.included_in_transfer,
	opis.checked_on,
	opis.checked_by,
	ep.institution_id,
	opis.notes
from education_portfolios ep
cross join (
	values
		('identificare', 'cv', 'CV Europass actualizat', 'portofoliu', 10, 'REG-PORT-0001', true, current_date - 2, 'Secretariat', 'Poziționat primul în opisul portofoliului.'),
		('cariera', 'contracte_incadrare', 'Decizie de încadrare 2025-2026', 'dosar_personal', 20, 'REG-PORT-0002', true, current_date - 2, 'Resurse umane', 'Referință la document din dosarul personal.'),
		('evaluare', 'evaluari_anuale', 'Fișă autoevaluare 2025-2026', 'portofoliu', 30, 'REG-PORT-0003', false, current_date - 1, 'Responsabil CFDCD', 'Document inclus în opis, dar încă nepredat la transfer.'),
		('declaratii', 'autenticitate', 'Declarație de autenticitate', 'portofoliu', 40, 'REG-PORT-0004', true, current_date - 1, 'Secretariat', 'Document obligatoriu verificat la recepție.')
) as opis(section_code, component_code, entry_title, source_scope, chronological_index, document_reference, included_in_transfer, checked_on, checked_by, notes)
where ep.portfolio_code = 'PORT-CD-2026-0001'
on conflict do nothing;

insert into education_portfolio_custody (
	portfolio_id, event_type, holder_name, holder_role, location_label, access_reason, started_on, ended_on,
	access_mode, sensitive_data_access, institution_id, notes
)
select
	ep.id,
	custody.event_type,
	custody.holder_name,
	custody.holder_role,
	custody.location_label,
	custody.access_reason,
	custody.started_on,
	custody.ended_on,
	custody.access_mode,
	custody.sensitive_data_access,
	ep.institution_id,
	custody.notes
from education_portfolios ep
cross join (
	values
		('preluare', 'Secretariat', 'Custode portofolii', 'Arhivă curentă unitate', 'Recepție portofoliu la începutul anului școlar', current_date - 30, current_date - 30, 'fizic', true, 'Preluare inițială cu verificare minimă a conținutului.'),
		('consultare', 'Responsabil CFDCD', 'Evaluator intern', 'Birou metodic', 'Verificare conformitate și opis', current_date - 5, current_date - 4, 'mixt', true, 'Consultare pentru verificarea documentelor și completărilor.'),
		('transfer', 'Secretariat Bălotești', 'Expeditor', 'Registratură ieșiri', 'Predare către unitatea primitoare', current_date - 3, current_date - 3, 'fizic', true, 'Predare consemnată cu confirmare de primire.')
) as custody(event_type, holder_name, holder_role, location_label, access_reason, started_on, ended_on, access_mode, sensitive_data_access, notes)
where ep.portfolio_code = 'PORT-CD-2026-0001'
on conflict do nothing;

insert into education_publications (
	publication_code, domain, entity_type, entity_label, publication_channel, publication_status, anonymization_status,
	mandatory, published_on, reviewed_by, institution_id, notes
)
values
	('PUB-2026-0001', 'guvernanta', 'hotarare', 'Hotărâre privind aprobarea proiectului de buget', 'avizier', 'publicat', 'finalizata', true, current_date - 11, 'DPO unitate', 'inst-001', 'Publicată după anonimizarea datelor personale.'),
	('PUB-2026-0002', 'portofolii', 'procedura_portofoliu', 'Procedura internă pentru portofoliul profesional al cadrului didactic', 'site_public', 'pregatit', 'nu_este_necesara', true, null, 'Director adjunct', 'inst-001', 'În curs de validare pentru publicarea pe site.'),
	('PUB-2026-0003', 'regulamente', 'roi', 'Regulament de ordine interioară 2025-2026', 'site_public', 'publicat', 'nu_este_necesara', true, current_date - 40, 'Secretariat', 'inst-001', 'Versiune aprobată și disponibilă public.')
on conflict do nothing;
