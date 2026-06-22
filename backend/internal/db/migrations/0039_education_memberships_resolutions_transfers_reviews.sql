create table if not exists education_governance_memberships (
	id uuid primary key default gen_random_uuid(),
	school_year text not null,
	organism text not null check (organism in ('ca', 'cp', 'ceac', 'cfdcd')),
	full_name text not null,
	role_name text not null,
	mandate_from date not null,
	mandate_to date not null,
	voting_right boolean not null default true,
	status text not null check (status in ('activ', 'suspendat', 'expirat')),
	institution_id text not null,
	notes text not null default '',
	created_at timestamptz not null default now(),
	updated_at timestamptz not null default now(),
	unique (institution_id, school_year, organism, full_name, mandate_from)
);

create index if not exists education_governance_memberships_institution_idx
	on education_governance_memberships(institution_id, school_year, organism);

create table if not exists education_meeting_resolutions (
	id uuid primary key default gen_random_uuid(),
	meeting_id uuid not null references education_meetings(id) on delete cascade,
	vote_id uuid not null references education_meeting_votes(id) on delete cascade,
	resolution_code text not null,
	title text not null,
	resolution_type text not null check (resolution_type in ('hotarare', 'decizie', 'aviz')),
	publication_status text not null check (publication_status in ('intern', 'publicat', 'pregatit_publicare')),
	anonymization_state text not null check (anonymization_state in ('necesara', 'finalizata', 'nu_este_necesara')),
	issued_on date not null,
	signed_by text not null default '',
	institution_id text not null,
	notes text not null default '',
	created_at timestamptz not null default now(),
	updated_at timestamptz not null default now(),
	unique (vote_id)
);

create index if not exists education_meeting_resolutions_meeting_idx
	on education_meeting_resolutions(meeting_id, institution_id, issued_on);

create table if not exists education_portfolio_transfers (
	id uuid primary key default gen_random_uuid(),
	portfolio_id uuid not null references education_portfolios(id) on delete cascade,
	transfer_code text not null,
	transfer_type text not null check (transfer_type in ('predare', 'primire', 'mutare', 'detasare')),
	source_institution text not null,
	destination_institution text not null,
	status text not null check (status in ('pregatit', 'trimis', 'receptionat', 'inchis')),
	handover_on date not null,
	received_on date,
	handover_by text not null default '',
	received_by text not null default '',
	institution_id text not null,
	notes text not null default '',
	created_at timestamptz not null default now(),
	updated_at timestamptz not null default now(),
	unique (portfolio_id, transfer_code)
);

create index if not exists education_portfolio_transfers_portfolio_idx
	on education_portfolio_transfers(portfolio_id, institution_id, handover_on);

create table if not exists education_portfolio_reviews (
	id uuid primary key default gen_random_uuid(),
	portfolio_id uuid not null references education_portfolios(id) on delete cascade,
	review_code text not null,
	review_stage text not null check (review_stage in ('depunere', 'verificare_secretariat', 'validare_manageriala', 'reverificare')),
	outcome text not null check (outcome in ('acceptat', 'completari', 'respins')),
	reviewer_name text not null,
	reviewed_on date not null,
	missing_documents integer not null default 0 check (missing_documents >= 0),
	compliance_score integer not null default 0 check (compliance_score >= 0 and compliance_score <= 100),
	institution_id text not null,
	notes text not null default '',
	created_at timestamptz not null default now(),
	updated_at timestamptz not null default now(),
	unique (portfolio_id, review_code)
);

create index if not exists education_portfolio_reviews_portfolio_idx
	on education_portfolio_reviews(portfolio_id, institution_id, reviewed_on);

insert into education_governance_memberships (
	school_year,
	organism,
	full_name,
	role_name,
	mandate_from,
	mandate_to,
	voting_right,
	status,
	institution_id,
	notes
)
select
	'2025-2026',
	member.organism,
	member.full_name,
	member.role_name,
	member.mandate_from,
	member.mandate_to,
	member.voting_right,
	member.status,
	'inst-001',
	member.notes
from (
	values
		('ca', 'Ioana Marinescu', 'Președinte CA', current_date - 320, current_date + 45, true, 'activ', 'Mandat valid pentru anul școlar curent și semnătură înregistrată.'),
		('cp', 'Adrian Iliescu', 'Secretar CP', current_date - 280, current_date + 90, true, 'activ', 'Coordonează convocările și centralizarea documentelor CP.'),
		('ceac', 'Elena Dumitru', 'Membru CEAC', current_date - 365, current_date - 10, true, 'expirat', 'Mandatul anterior trebuie reînnoit prin decizie internă.'),
		('cfdcd', 'Raluca Dobre', 'Responsabil CFDCD', current_date - 180, current_date + 180, false, 'activ', 'Coordonează portofoliile și activitatea de formare continuă.')
) as member(organism, full_name, role_name, mandate_from, mandate_to, voting_right, status, notes)
on conflict do nothing;

insert into education_meeting_resolutions (
	meeting_id,
	vote_id,
	resolution_code,
	title,
	resolution_type,
	publication_status,
	anonymization_state,
	issued_on,
	signed_by,
	institution_id,
	notes
)
select
	em.id,
	emv.id,
	resolution.resolution_code,
	resolution.title,
	resolution.resolution_type,
	resolution.publication_status,
	resolution.anonymization_state,
	resolution.issued_on,
	resolution.signed_by,
	em.institution_id,
	resolution.notes
from education_meetings em
join education_meeting_votes emv on emv.meeting_id = em.id
join (
	values
		('Aprobarea proiectului de buget și a listei de investiții IT', 'HCA-2025-0001', 'Hotărâre privind aprobarea proiectului de buget', 'hotarare', 'pregatit_publicare', 'finalizata', current_date - 12, 'Director', 'Forma pentru publicare a fost anonimizată și transmisă spre afișare.'),
		('Validarea planului de achiziții pentru laborator', 'AVZ-2025-0002', 'Aviz privind planul de achiziții pentru laborator', 'aviz', 'intern', 'nu_este_necesara', current_date - 12, 'Președinte CA', 'Aviz intern fără date cu caracter personal.')
) as resolution(subject_title, resolution_code, title, resolution_type, publication_status, anonymization_state, issued_on, signed_by, notes)
	on resolution.subject_title = emv.subject_title
where em.title = 'Avizarea proiectului de buget'
on conflict do nothing;

insert into education_portfolio_transfers (
	portfolio_id,
	transfer_code,
	transfer_type,
	source_institution,
	destination_institution,
	status,
	handover_on,
	received_on,
	handover_by,
	received_by,
	institution_id,
	notes
)
select
	ep.id,
	transfer.transfer_code,
	transfer.transfer_type,
	transfer.source_institution,
	transfer.destination_institution,
	transfer.status,
	transfer.handover_on,
	transfer.received_on,
	transfer.handover_by,
	transfer.received_by,
	ep.institution_id,
	transfer.notes
from education_portfolios ep
cross join (
	values
		('TRF-2026-0001', 'predare', 'Școala Bălotești', 'Liceul Teoretic Periam', 'trimis', current_date - 8, current_date - 6, 'Secretariat Bălotești', 'Secretariat Periam', 'Predare interinstituțională cu confirmare de primire și opis semnat.'),
		('TRF-2026-0002', 'primire', 'Inspectoratul Școlar Județean', 'Școala Bălotești', 'receptionat', current_date - 4, current_date - 3, 'Compartiment mobilitate ISJ', 'Responsabil portofolii', 'Portofoliu recepționat pentru reverificare și arhivare locală.')
) as transfer(transfer_code, transfer_type, source_institution, destination_institution, status, handover_on, received_on, handover_by, received_by, notes)
where ep.portfolio_code = 'PORT-CD-2026-0001'
on conflict do nothing;

insert into education_portfolio_reviews (
	portfolio_id,
	review_code,
	review_stage,
	outcome,
	reviewer_name,
	reviewed_on,
	missing_documents,
	compliance_score,
	institution_id,
	notes
)
select
	ep.id,
	review.review_code,
	review.review_stage,
	review.outcome,
	review.reviewer_name,
	review.reviewed_on,
	review.missing_documents,
	review.compliance_score,
	ep.institution_id,
	review.notes
from education_portfolios ep
cross join (
	values
		('REV-2026-0001', 'verificare_secretariat', 'completari', 'Secretariat', current_date - 5, 2, 82, 'Lipsesc două dovezi de formare continuă și un opis numerotat complet.'),
		('REV-2026-0002', 'reverificare', 'acceptat', 'Responsabil CFDCD', current_date - 1, 0, 97, 'Completările au fost depuse, portofoliul poate fi validat și arhivat.')
) as review(review_code, review_stage, outcome, reviewer_name, reviewed_on, missing_documents, compliance_score, notes)
where ep.portfolio_code = 'PORT-CD-2026-0001'
on conflict do nothing;
