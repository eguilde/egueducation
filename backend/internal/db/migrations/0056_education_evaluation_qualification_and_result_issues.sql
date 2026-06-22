alter table education_evaluations
	add column if not exists qualification text not null default 'nesatisfacator';

do $$
begin
	if not exists (
		select 1
		from pg_constraint
		where conname = 'education_evaluations_qualification_check'
	) then
		alter table education_evaluations
			add constraint education_evaluations_qualification_check
			check (qualification in ('foarte_bine', 'bine', 'satisfacator', 'nesatisfacator'));
	end if;
end $$;

update education_evaluations
set qualification = case
	when score >= 90 then 'foarte_bine'
	when score >= 75 then 'bine'
	when score >= 60 then 'satisfacator'
	else 'nesatisfacator'
end
where qualification is null
	or qualification not in ('foarte_bine', 'bine', 'satisfacator', 'nesatisfacator');

create table if not exists education_evaluation_result_issues (
	id uuid primary key default gen_random_uuid(),
	evaluation_id uuid not null references education_evaluations(id) on delete cascade,
	issue_code text not null unique,
	document_type text not null check (document_type in ('fisa_evaluare', 'comunicare', 'decizie', 'raport_final')),
	recipient_name text not null default '',
	recipient_role text not null default '',
	delivery_channel text not null check (delivery_channel in ('registratura', 'email', 'intern', 'posta')),
	delivery_status text not null check (delivery_status in ('pregatit', 'emis', 'transmis', 'confirmat')),
	issued_on date not null,
	delivered_on date,
	acknowledged_on date,
	registry_reference text not null default '',
	attached_to_personnel_file boolean not null default true,
	institution_id text not null,
	notes text not null default '',
	created_at timestamptz not null default now(),
	updated_at timestamptz not null default now()
);

create index if not exists idx_education_evaluation_result_issues_lookup
	on education_evaluation_result_issues(institution_id, evaluation_id, delivery_status, issued_on);

insert into education_evaluation_result_issues (
	evaluation_id,
	issue_code,
	document_type,
	recipient_name,
	recipient_role,
	delivery_channel,
	delivery_status,
	issued_on,
	delivered_on,
	acknowledged_on,
	registry_reference,
	attached_to_personnel_file,
	institution_id,
	notes
)
select
	ee.id,
	'EVRES-2026-0001',
	'fisa_evaluare',
	ee.full_name,
	'Cadru didactic evaluat',
	'registratura',
	'transmis',
	date '2026-06-18',
	date '2026-06-18',
	null,
	'REG-EVAL-2026-0188',
	true,
	ee.institution_id,
	'Comunicarea fișei de evaluare și a punctajului final către persoana evaluată.'
from education_evaluations ee
where ee.evaluation_code = 'EVAL-2026-001'
on conflict (issue_code) do nothing;
