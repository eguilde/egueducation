create table if not exists education_meeting_participants (
	id uuid primary key default gen_random_uuid(),
	meeting_id uuid not null references education_meetings(id) on delete cascade,
	full_name text not null,
	role_name text not null,
	member_type text not null check (member_type in ('presedinte', 'secretar', 'membru', 'invitat', 'observator')),
	attendance_status text not null check (attendance_status in ('invitat', 'prezent', 'absent_motivat', 'absent_nemotivat')),
	voting_right boolean not null default true,
	signature_present boolean not null default false,
	institution_id text not null,
	notes text not null default '',
	created_at timestamptz not null default now(),
	updated_at timestamptz not null default now()
);

create index if not exists education_meeting_participants_meeting_idx
	on education_meeting_participants(meeting_id, institution_id);

create table if not exists education_meeting_documents (
	id uuid primary key default gen_random_uuid(),
	meeting_id uuid not null references education_meetings(id) on delete cascade,
	document_type text not null check (document_type in ('convocator', 'ordine_de_zi', 'prezenta', 'proces_verbal', 'anexa', 'hotarare', 'material_sedinta', 'delegare')),
	title text not null,
	document_number text not null default '',
	registry_number text not null default '',
	publication_status text not null check (publication_status in ('intern', 'anonimizare_necesara', 'publicat')),
	custody_owner text not null default '',
	signed_by text not null default '',
	issued_on date not null,
	institution_id text not null,
	summary text not null default '',
	created_at timestamptz not null default now(),
	updated_at timestamptz not null default now()
);

create index if not exists education_meeting_documents_meeting_idx
	on education_meeting_documents(meeting_id, institution_id);

create table if not exists education_portfolio_documents (
	id uuid primary key default gen_random_uuid(),
	portfolio_id uuid not null references education_portfolios(id) on delete cascade,
	section_code text not null,
	component_code text not null,
	document_title text not null,
	source_scope text not null check (source_scope in ('portofoliu', 'dosar_personal')),
	evidence_type text not null,
	issued_on date not null,
	added_on date not null,
	chronological_index integer not null default 0 check (chronological_index >= 0),
	sensitive_data boolean not null default false,
	authenticity_status text not null check (authenticity_status in ('declarat', 'verificat', 'respins')),
	file_reference text not null default '',
	institution_id text not null,
	notes text not null default '',
	created_at timestamptz not null default now(),
	updated_at timestamptz not null default now()
);

create index if not exists education_portfolio_documents_portfolio_idx
	on education_portfolio_documents(portfolio_id, institution_id);

insert into education_meeting_participants (
	meeting_id,
	full_name,
	role_name,
	member_type,
	attendance_status,
	voting_right,
	signature_present,
	institution_id,
	notes
)
select
	em.id,
	participant.full_name,
	participant.role_name,
	participant.member_type,
	participant.attendance_status,
	participant.voting_right,
	participant.signature_present,
	em.institution_id,
	participant.notes
from education_meetings em
cross join (
	values
		('Ioana Marinescu', 'Președinte CA', 'presedinte', 'prezent', true, true, 'A condus ședința și a validat ordinea de zi.'),
		('Mihai Stoica', 'Secretar CA', 'secretar', 'prezent', true, true, 'A întocmit procesul-verbal și lista de prezență.'),
		('Elena Dumitru', 'Membru CA', 'membru', 'prezent', true, false, 'A participat la vot pentru aprobarea ROI.')
) as participant(full_name, role_name, member_type, attendance_status, voting_right, signature_present, notes)
where em.title = 'Avizarea proiectului de buget'
on conflict do nothing;

insert into education_meeting_documents (
	meeting_id,
	document_type,
	title,
	document_number,
	registry_number,
	publication_status,
	custody_owner,
	signed_by,
	issued_on,
	institution_id,
	summary
)
select
	em.id,
	document.document_type,
	document.title,
	document.document_number,
	document.registry_number,
	document.publication_status,
	document.custody_owner,
	document.signed_by,
	document.issued_on,
	em.institution_id,
	document.summary
from education_meetings em
cross join (
	values
		('convocator', 'Convocator ședință CA', 'CA-CONV-2025-0001', 'REG-CA-0001', 'intern', 'Secretariat', 'Director', current_date - 15, 'Convocare oficială transmisă membrilor CA.'),
		('ordine_de_zi', 'Ordine de zi ședință CA', 'CA-ODZ-2025-0001', 'REG-CA-0002', 'intern', 'Secretariat', 'Director', current_date - 14, 'Aprobarea orarului și a ROI.'),
		('proces_verbal', 'Proces-verbal ședință CA', 'CA-PV-2025-0001', 'REG-CA-0003', 'anonimizare_necesara', 'Secretariat', 'Secretar CA', current_date - 13, 'Conține dezbateri și voturi privind documentele manageriale.')
) as document(document_type, title, document_number, registry_number, publication_status, custody_owner, signed_by, issued_on, summary)
where em.title = 'Avizarea proiectului de buget'
on conflict do nothing;

insert into education_portfolio_documents (
	portfolio_id,
	section_code,
	component_code,
	document_title,
	source_scope,
	evidence_type,
	issued_on,
	added_on,
	chronological_index,
	sensitive_data,
	authenticity_status,
	file_reference,
	institution_id,
	notes
)
select
	ep.id,
	document.section_code,
	document.component_code,
	document.document_title,
	document.source_scope,
	document.evidence_type,
	document.issued_on,
	document.added_on,
	document.chronological_index,
	document.sensitive_data,
	document.authenticity_status,
	document.file_reference,
	ep.institution_id,
	document.notes
from education_portfolios ep
cross join (
	values
		('identificare', 'cv', 'CV Europass actualizat', 'portofoliu', 'cv', current_date - 180, current_date - 30, 10, false, 'verificat', 'REG-PORT-0001', 'Document verificat la actualizarea anuală.'),
		('cariera', 'contracte_incadrare', 'Decizie de încadrare 2025-2026', 'dosar_personal', 'document_cariera', current_date - 260, current_date - 28, 20, true, 'verificat', 'REG-PORT-0002', 'Document din dosarul personal, referențiat și în portofoliu.'),
		('evaluare', 'evaluari_anuale', 'Fișă autoevaluare 2025-2026', 'portofoliu', 'evaluare', current_date - 20, current_date - 15, 30, true, 'declarat', 'REG-PORT-0003', 'În așteptarea verificării de către evaluator.'),
		('declaratii', 'autenticitate', 'Declarație de autenticitate', 'portofoliu', 'declaratie', current_date - 10, current_date - 10, 40, true, 'verificat', 'REG-PORT-0004', 'Declarație semnată la depunerea portofoliului.')
) as document(section_code, component_code, document_title, source_scope, evidence_type, issued_on, added_on, chronological_index, sensitive_data, authenticity_status, file_reference, notes)
where ep.portfolio_code = 'PORT-CD-2026-0001'
on conflict do nothing;
