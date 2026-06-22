create table if not exists education_meeting_votes (
	id uuid primary key default gen_random_uuid(),
	meeting_id uuid not null references education_meetings(id) on delete cascade,
	subject_title text not null,
	agenda_order integer not null check (agenda_order >= 1),
	decision_type text not null check (decision_type in ('hotarare', 'aviz', 'informare', 'delegare', 'aprobare')),
	votes_for integer not null default 0 check (votes_for >= 0),
	votes_against integer not null default 0 check (votes_against >= 0),
	abstentions integer not null default 0 check (abstentions >= 0),
	outcome text not null check (outcome in ('adoptat', 'respins', 'amanat')),
	requires_follow_up boolean not null default false,
	legal_basis text not null default '',
	institution_id text not null,
	notes text not null default '',
	created_at timestamptz not null default now(),
	updated_at timestamptz not null default now()
);

create index if not exists education_meeting_votes_meeting_idx
	on education_meeting_votes(meeting_id, institution_id);

create table if not exists education_portfolio_checklist (
	id uuid primary key default gen_random_uuid(),
	portfolio_id uuid not null references education_portfolios(id) on delete cascade,
	requirement_code text not null,
	requirement_label text not null,
	section_code text not null,
	source_scope text not null check (source_scope in ('portofoliu', 'dosar_personal')),
	mandatory boolean not null default true,
	status text not null check (status in ('complet', 'partial', 'lipsa', 'in_verificare')),
	document_count integer not null default 0 check (document_count >= 0),
	last_checked_on date not null,
	checked_by text not null default '',
	institution_id text not null,
	notes text not null default '',
	created_at timestamptz not null default now(),
	updated_at timestamptz not null default now()
);

create index if not exists education_portfolio_checklist_portfolio_idx
	on education_portfolio_checklist(portfolio_id, institution_id);

insert into education_meeting_votes (
	meeting_id,
	subject_title,
	agenda_order,
	decision_type,
	votes_for,
	votes_against,
	abstentions,
	outcome,
	requires_follow_up,
	legal_basis,
	institution_id,
	notes
)
select
	em.id,
	vote.subject_title,
	vote.agenda_order,
	vote.decision_type,
	vote.votes_for,
	vote.votes_against,
	vote.abstentions,
	vote.outcome,
	vote.requires_follow_up,
	vote.legal_basis,
	em.institution_id,
	vote.notes
from education_meetings em
cross join (
	values
		('Aprobarea proiectului de buget și a listei de investiții IT', 1, 'aprobare', 7, 1, 0, 'adoptat', true, 'Legea 198/2023, ROFUIP, buget anual', 'Necesită emiterea hotărârii și comunicarea către compartimentul financiar.'),
		('Validarea planului de achiziții pentru laborator', 2, 'aviz', 6, 1, 1, 'adoptat', false, 'Plan managerial și necesar logistic', 'Aviz favorabil fără măsuri suplimentare.'),
		('Propunere de amânare pentru reorganizarea programului', 3, 'informare', 3, 3, 2, 'amanat', true, 'Analiză internă CA', 'Se reia la ședința următoare după completarea documentelor.')
) as vote(subject_title, agenda_order, decision_type, votes_for, votes_against, abstentions, outcome, requires_follow_up, legal_basis, notes)
where em.title = 'Avizarea proiectului de buget'
on conflict do nothing;

insert into education_portfolio_checklist (
	portfolio_id,
	requirement_code,
	requirement_label,
	section_code,
	source_scope,
	mandatory,
	status,
	document_count,
	last_checked_on,
	checked_by,
	institution_id,
	notes
)
select
	ep.id,
	item.requirement_code,
	item.requirement_label,
	item.section_code,
	item.source_scope,
	item.mandatory,
	item.status,
	item.document_count,
	item.last_checked_on,
	item.checked_by,
	ep.institution_id,
	item.notes
from education_portfolios ep
cross join (
	values
		('opis_001', 'Opis și ordonare cronologică documente', 'identificare', 'portofoliu', true, 'partial', 4, current_date - 3, 'Secretariat', 'Există documentele principale, dar opisul nu este complet numerotat.'),
		('autenticitate_001', 'Declarație de autenticitate semnată', 'declaratii', 'portofoliu', true, 'complet', 1, current_date - 2, 'Secretariat', 'Declarația este prezentă și verificată.'),
		('gdpr_001', 'Informare și consimțământ GDPR', 'declaratii', 'portofoliu', true, 'complet', 1, current_date - 2, 'DPO unitate', 'Informarea este atașată la dosar.'),
		('dosar_personal_001', 'Documente de carieră preluate din dosarul personal', 'cariera', 'dosar_personal', true, 'complet', 2, current_date - 2, 'Resurse umane', 'Documentele de încadrare sunt referențiate corect.'),
		('formare_001', 'Dovezi de formare continuă pentru anul școlar curent', 'formare', 'portofoliu', false, 'lipsa', 0, current_date - 1, 'Responsabil CFDCD', 'Nu au fost încă încărcate certificatele de formare.')
) as item(requirement_code, requirement_label, section_code, source_scope, mandatory, status, document_count, last_checked_on, checked_by, notes)
where ep.portfolio_code = 'PORT-CD-2026-0001'
on conflict do nothing;
