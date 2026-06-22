create table if not exists education_evaluation_self_reviews (
	id uuid primary key default gen_random_uuid(),
	evaluation_id uuid not null references education_evaluations(id) on delete cascade,
	review_code text not null unique,
	section_title text not null default '',
	narrative_type text not null check (narrative_type in ('autoevaluare', 'performanta', 'dezvoltare', 'impact')),
	status text not null check (status in ('draft', 'submitted', 'validated', 'returned')),
	completed_on date not null,
	evidence_summary text not null default '',
	strengths text not null default '',
	improvement_needs text not null default '',
	assumed_score numeric(5,2) not null default 0 check (assumed_score >= 0 and assumed_score <= 100),
	institution_id text not null,
	notes text not null default '',
	created_at timestamptz not null default now(),
	updated_at timestamptz not null default now()
);

create index if not exists education_evaluation_self_reviews_eval_idx
	on education_evaluation_self_reviews(evaluation_id, institution_id, completed_on, status);

create table if not exists education_evaluation_criteria (
	id uuid primary key default gen_random_uuid(),
	evaluation_id uuid not null references education_evaluations(id) on delete cascade,
	criterion_code text not null unique,
	criterion_category text not null check (criterion_category in ('proiectare', 'predare', 'evaluare', 'management_clasa', 'dezvoltare', 'parteneriat')),
	criterion_label text not null default '',
	max_score numeric(5,2) not null default 0 check (max_score >= 0 and max_score <= 100),
	self_score numeric(5,2) not null default 0 check (self_score >= 0),
	reviewer_score numeric(5,2) not null default 0 check (reviewer_score >= 0),
	final_score numeric(5,2) not null default 0 check (final_score >= 0),
	status text not null check (status in ('draft', 'reviewed', 'validated', 'contested')),
	evidence_summary text not null default '',
	institution_id text not null,
	notes text not null default '',
	created_at timestamptz not null default now(),
	updated_at timestamptz not null default now(),
	check (self_score <= max_score),
	check (reviewer_score <= max_score),
	check (final_score <= max_score)
);

create index if not exists education_evaluation_criteria_eval_idx
	on education_evaluation_criteria(evaluation_id, institution_id, criterion_category, status);

insert into education_evaluation_self_reviews (
	evaluation_id, review_code, section_title, narrative_type, status, completed_on,
	evidence_summary, strengths, improvement_needs, assumed_score, institution_id, notes
)
select
	ee.id,
	'AUTO-2026-0001',
	'Reflecție anuală asupra activității la clasă',
	'autoevaluare',
	'submitted',
	current_date - 14,
	'Sinteză a rezultatelor la clasă, participării la formare și activităților extracurriculare.',
	'Relație bună cu elevii și documentare constantă a progresului.',
	'Necesită consolidarea evaluării diferențiate și a feedbackului individualizat.',
	88.50,
	ee.institution_id,
	'Autoevaluare depusă înainte de ședința comisiei.'
from education_evaluations ee
where ee.evaluation_code = 'EVAL-2026-0001'
on conflict (review_code) do nothing;

insert into education_evaluation_criteria (
	evaluation_id, criterion_code, criterion_category, criterion_label, max_score,
	self_score, reviewer_score, final_score, status, evidence_summary, institution_id, notes
)
select
	ee.id,
	'CRIT-2026-0001',
	'predare',
	'Planificarea și organizarea activităților de predare',
	25,
	23,
	22,
	22,
	'validated',
	'Planificări, proiecte didactice și feedback din asistențe.',
	ee.institution_id,
	'Criteriu validat de evaluator.'
from education_evaluations ee
where ee.evaluation_code = 'EVAL-2026-0001'
on conflict (criterion_code) do nothing;

update education_evaluations ee
set score = totals.total_score
from (
	select evaluation_id, coalesce(sum(final_score), 0) as total_score
	from education_evaluation_criteria
	group by evaluation_id
) as totals
where ee.id = totals.evaluation_id;

insert into education_evaluation_criteria (
	evaluation_id, criterion_code, criterion_category, criterion_label, max_score,
	self_score, reviewer_score, final_score, status, evidence_summary, institution_id, notes
)
select
	ee.id,
	'CRIT-2026-0002',
	'evaluare',
	'Utilizarea instrumentelor de evaluare și feedback',
	25,
	22,
	21,
	21,
	'validated',
	'Teste, rubrici de evaluare și exemple de feedback individual.',
	ee.institution_id,
	'Criteriu validat de evaluator.'
from education_evaluations ee
where ee.evaluation_code = 'EVAL-2026-0001'
on conflict (criterion_code) do nothing;
