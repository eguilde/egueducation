create table if not exists education_mobility_documents (
	id uuid primary key default gen_random_uuid(),
	mobility_case_id uuid not null references education_mobility_cases(id) on delete cascade,
	document_code text not null,
	document_type text not null check (document_type in ('cerere', 'adeverinta', 'aviz', 'fisa_evaluare', 'decizie', 'anexa')),
	stage_scope text not null check (stage_scope in ('depunere', 'verificare', 'sedinta', 'aprobare', 'emitere')),
	document_title text not null,
	registered_on date not null,
	submitted_by text not null default '',
	verified_by text not null default '',
	validation_status text not null check (validation_status in ('draft', 'submitted', 'validated', 'rejected')),
	mandatory boolean not null default false,
	institution_id text not null,
	notes text not null default '',
	created_at timestamptz not null default now(),
	updated_at timestamptz not null default now(),
	unique (mobility_case_id, document_code)
);

create index if not exists idx_education_mobility_documents_lookup
	on education_mobility_documents(institution_id, mobility_case_id, document_type, validation_status, registered_on);

create table if not exists education_mobility_scores (
	id uuid primary key default gen_random_uuid(),
	mobility_case_id uuid not null references education_mobility_cases(id) on delete cascade,
	criterion_code text not null,
	criterion_label text not null,
	criterion_category text not null check (criterion_category in ('studii', 'vechime', 'performanta', 'social', 'administrativ')),
	max_score numeric(6,2) not null check (max_score > 0),
	awarded_score numeric(6,2) not null check (awarded_score >= 0),
	evidence_reference text not null default '',
	validated_by text not null default '',
	contested boolean not null default false,
	institution_id text not null,
	notes text not null default '',
	created_at timestamptz not null default now(),
	updated_at timestamptz not null default now(),
	unique (mobility_case_id, criterion_code)
);

create index if not exists idx_education_mobility_scores_lookup
	on education_mobility_scores(institution_id, mobility_case_id, criterion_category, contested);

create table if not exists education_mobility_appeals (
	id uuid primary key default gen_random_uuid(),
	mobility_case_id uuid not null references education_mobility_cases(id) on delete cascade,
	appeal_code text not null unique,
	submitted_by text not null default '',
	submitted_on date not null,
	status text not null check (status in ('submitted', 'review', 'accepted', 'rejected', 'resolved')),
	grounds text not null default '',
	hearing_on date,
	resolved_on date,
	decision_summary text not null default '',
	institution_id text not null,
	notes text not null default '',
	created_at timestamptz not null default now(),
	updated_at timestamptz not null default now()
);

create index if not exists idx_education_mobility_appeals_lookup
	on education_mobility_appeals(institution_id, mobility_case_id, status, submitted_on);

create table if not exists education_merit_documents (
	id uuid primary key default gen_random_uuid(),
	grant_id uuid not null references education_merit_grants(id) on delete cascade,
	document_code text not null,
	document_type text not null check (document_type in ('cerere', 'declaratie', 'autoevaluare', 'adeverinta', 'portofoliu', 'anexa')),
	document_title text not null,
	registered_on date not null,
	submitted_by text not null default '',
	validation_status text not null check (validation_status in ('draft', 'submitted', 'validated', 'rejected')),
	mandatory boolean not null default false,
	institution_id text not null,
	notes text not null default '',
	created_at timestamptz not null default now(),
	updated_at timestamptz not null default now(),
	unique (grant_id, document_code)
);

create index if not exists idx_education_merit_documents_lookup
	on education_merit_documents(institution_id, grant_id, document_type, validation_status, registered_on);

create table if not exists education_merit_scores (
	id uuid primary key default gen_random_uuid(),
	grant_id uuid not null references education_merit_grants(id) on delete cascade,
	criterion_code text not null,
	criterion_label text not null,
	criterion_category text not null check (criterion_category in ('performanta', 'impact', 'dezvoltare', 'management', 'incluziune')),
	panel_stage text not null check (panel_stage in ('autoevaluare', 'evaluare_comisie', 'validare_finala')),
	max_score numeric(6,2) not null check (max_score > 0),
	awarded_score numeric(6,2) not null check (awarded_score >= 0),
	reviewer_name text not null default '',
	evidence_reference text not null default '',
	contested boolean not null default false,
	institution_id text not null,
	notes text not null default '',
	created_at timestamptz not null default now(),
	updated_at timestamptz not null default now(),
	unique (grant_id, criterion_code, panel_stage)
);

create index if not exists idx_education_merit_scores_lookup
	on education_merit_scores(institution_id, grant_id, criterion_category, panel_stage, contested);

create table if not exists education_merit_appeals (
	id uuid primary key default gen_random_uuid(),
	grant_id uuid not null references education_merit_grants(id) on delete cascade,
	appeal_code text not null unique,
	submitted_by text not null default '',
	submitted_on date not null,
	status text not null check (status in ('submitted', 'review', 'accepted', 'rejected', 'resolved')),
	grounds text not null default '',
	resolved_on date,
	decision_summary text not null default '',
	institution_id text not null,
	notes text not null default '',
	created_at timestamptz not null default now(),
	updated_at timestamptz not null default now()
);

create index if not exists idx_education_merit_appeals_lookup
	on education_merit_appeals(institution_id, grant_id, status, submitted_on);

insert into education_mobility_documents (
	mobility_case_id, document_code, document_type, stage_scope, document_title, registered_on,
	submitted_by, verified_by, validation_status, mandatory, institution_id, notes
)
select
	emc.id, 'MOBDOC-2026-0001', 'cerere', 'depunere', 'Cerere de transfer semnata', current_date - 8,
	emc.full_name, 'Secretariat', 'validated', true, emc.institution_id, 'Document principal din dosarul de mobilitate.'
from education_mobility_cases emc
where emc.case_code = 'MOB-2026-0001'
on conflict (mobility_case_id, document_code) do nothing;

insert into education_mobility_documents (
	mobility_case_id, document_code, document_type, stage_scope, document_title, registered_on,
	submitted_by, verified_by, validation_status, mandatory, institution_id, notes
)
select
	emc.id, 'MOBDOC-2026-0002', 'aviz', 'aprobare', 'Aviz comisie judeteana', current_date - 2,
	'Comisia de mobilitate', 'Inspector HR', 'submitted', true, emc.institution_id, 'In asteptarea aprobarii finale.'
from education_mobility_cases emc
where emc.case_code = 'MOB-2026-0003'
on conflict (mobility_case_id, document_code) do nothing;

insert into education_mobility_scores (
	mobility_case_id, criterion_code, criterion_label, criterion_category, max_score, awarded_score,
	evidence_reference, validated_by, contested, institution_id, notes
)
select
	emc.id, 'MOB-CRT-01', 'Studii de specialitate', 'studii', 20, 20,
	'DIP-TRANSFER-2026', 'Comisia de mobilitate', false, emc.institution_id, 'Documentele de studii sunt complete.'
from education_mobility_cases emc
where emc.case_code = 'MOB-2026-0001'
on conflict (mobility_case_id, criterion_code) do nothing;

insert into education_mobility_scores (
	mobility_case_id, criterion_code, criterion_label, criterion_category, max_score, awarded_score,
	evidence_reference, validated_by, contested, institution_id, notes
)
select
	emc.id, 'MOB-CRT-02', 'Vechime la catedra', 'vechime', 25, 18,
	'ADVR-VECHIME-2026', 'Comisia de mobilitate', true, emc.institution_id, 'Candidatul contesta incadrarea la transa de vechime.'
from education_mobility_cases emc
where emc.case_code = 'MOB-2026-0001'
on conflict (mobility_case_id, criterion_code) do nothing;

insert into education_mobility_appeals (
	mobility_case_id, appeal_code, submitted_by, submitted_on, status, grounds, hearing_on, resolved_on, decision_summary, institution_id, notes
)
select
	emc.id, 'MOBAPL-2026-0001', emc.full_name, current_date - 1, 'review',
	'Solicita reanalizarea punctajului pentru vechime si activitate metodica.', current_date + 5, null,
	'', emc.institution_id, 'Contestatia este inregistrata si repartizata comisiei.'
from education_mobility_cases emc
where emc.case_code = 'MOB-2026-0001'
on conflict (appeal_code) do nothing;

insert into education_merit_documents (
	grant_id, document_code, document_type, document_title, registered_on, submitted_by, validation_status, mandatory, institution_id, notes
)
select
	emg.id, 'GRDOC-2026-0001', 'cerere', 'Cerere inscriere gradatie de merit', current_date - 12,
	emg.full_name, 'validated', true, emg.institution_id, 'Cererea este inregistrata si validata formal.'
from education_merit_grants emg
where emg.grant_code = 'GRM-2026-0001'
on conflict (grant_id, document_code) do nothing;

insert into education_merit_documents (
	grant_id, document_code, document_type, document_title, registered_on, submitted_by, validation_status, mandatory, institution_id, notes
)
select
	emg.id, 'GRDOC-2026-0002', 'autoevaluare', 'Fisa de autoevaluare management', current_date - 8,
	emg.full_name, 'submitted', true, emg.institution_id, 'Forma depusa pentru evaluarea comisiei.'
from education_merit_grants emg
where emg.grant_code = 'GRM-2026-0002'
on conflict (grant_id, document_code) do nothing;

insert into education_merit_scores (
	grant_id, criterion_code, criterion_label, criterion_category, panel_stage, max_score, awarded_score,
	reviewer_name, evidence_reference, contested, institution_id, notes
)
select
	emg.id, 'GRM-CRT-01', 'Rezultate la catedra si impact educational', 'performanta', 'evaluare_comisie', 40, 36,
	'Comisia gradatie de merit', 'PORT-2026-01', false, emg.institution_id, 'Dovezi validate integral.'
from education_merit_grants emg
where emg.grant_code = 'GRM-2026-0001'
on conflict (grant_id, criterion_code, panel_stage) do nothing;

insert into education_merit_scores (
	grant_id, criterion_code, criterion_label, criterion_category, panel_stage, max_score, awarded_score,
	reviewer_name, evidence_reference, contested, institution_id, notes
)
select
	emg.id, 'GRM-CRT-02', 'Leadership institutional si dezvoltare', 'management', 'evaluare_comisie', 35, 33,
	'Comisia gradatie de merit', 'PORT-MGMT-2026', true, emg.institution_id, 'Exista observatii privind documentele justificative pentru un subcriteriu.'
from education_merit_grants emg
where emg.grant_code = 'GRM-2026-0002'
on conflict (grant_id, criterion_code, panel_stage) do nothing;

insert into education_merit_appeals (
	grant_id, appeal_code, submitted_by, submitted_on, status, grounds, resolved_on, decision_summary, institution_id, notes
)
select
	emg.id, 'GRAPL-2026-0001', emg.full_name, current_date - 1, 'submitted',
	'Solicita reevaluarea subcriteriului privind impactul institutional.', null,
	'', emg.institution_id, 'Contestatia a fost depusa in termen si urmeaza repartizarea.'
from education_merit_grants emg
where emg.grant_code = 'GRM-2026-0002'
on conflict (appeal_code) do nothing;
