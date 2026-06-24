create table if not exists education_portfolio_valorifications (
	id uuid primary key default gen_random_uuid(),
	portfolio_id uuid not null references education_portfolios(id) on delete cascade,
	valorification_code text not null,
	scope text not null check (scope in (
		'licentiere',
		'debut',
		'definitivat',
		'grad_ii',
		'grad_i',
		'evaluare_profesionala',
		'mobilitate',
		'dezvoltare_profesionala',
		'inspectie_scolara',
		'evaluare_externa_calitate',
		'gradatie_merit',
		'distinctie_premiu'
	)),
	status text not null check (status in ('planificat', 'in_pregatire', 'transmis', 'validat', 'finalizat')),
	requested_by text not null default '',
	target_institution text not null default '',
	target_reference text not null default '',
	started_on date not null,
	completed_on date,
	institution_id text not null,
	notes text not null default '',
	created_at timestamptz not null default now(),
	updated_at timestamptz not null default now(),
	unique (portfolio_id, valorification_code)
);

create index if not exists education_portfolio_valorifications_portfolio_idx
	on education_portfolio_valorifications(portfolio_id, institution_id, started_on desc);

insert into education_portfolio_valorifications (
	portfolio_id,
	valorification_code,
	scope,
	status,
	requested_by,
	target_institution,
	target_reference,
	started_on,
	completed_on,
	institution_id,
	notes
)
select
	ep.id,
	item.valorification_code,
	item.scope,
	item.status,
	item.requested_by,
	item.target_institution,
	item.target_reference,
	item.started_on,
	item.completed_on,
	ep.institution_id,
	item.notes
from education_portfolios ep
cross join (
	values
		('VAL-2026-0001', 'evaluare_profesionala', 'finalizat', 'Director', 'Scoala Balotesti', 'EVA-2026-0001', current_date - 40, current_date - 30, 'Portofoliul a fost valorificat in evaluarea profesionala anuala.'),
		('VAL-2026-0002', 'mobilitate', 'transmis', 'Secretariat', 'Inspectoratul Scolar Judetean', 'MOB-2026-0003', current_date - 12, null, 'Extrasul de portofoliu a fost transmis pentru etapa de mobilitate.'),
		('VAL-2026-0003', 'gradatie_merit', 'validat', 'Comisia de evaluare', 'Comisia judeteana', 'MERIT-2026-0002', current_date - 20, current_date - 5, 'Dovezile relevante din portofoliu au fost validate pentru gradatie de merit.')
) as item(valorification_code, scope, status, requested_by, target_institution, target_reference, started_on, completed_on, notes)
where ep.portfolio_code = 'PORT-CD-2026-0001'
on conflict do nothing;
