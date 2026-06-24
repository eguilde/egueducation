create table if not exists education_committees (
	id uuid primary key default gen_random_uuid(),
	committee_code text not null,
	school_year text not null,
	committee_type text not null check (committee_type in ('permanenta', 'temporara', 'evaluare_personal_didactic', 'curriculum', 'mentorat', 'securitate', 'burse', 'alta')),
	title text not null,
	status text not null check (status in ('draft', 'active', 'completed', 'archived')),
	decision_reference text not null default '',
	starts_on date not null,
	ends_on date,
	evaluation_scope boolean not null default false,
	institution_id text not null,
	notes text not null default '',
	created_at timestamptz not null default now(),
	updated_at timestamptz not null default now(),
	unique (institution_id, committee_code)
);

create index if not exists idx_education_committees_lookup
	on education_committees(institution_id, school_year, committee_type, status, starts_on);

create table if not exists education_committee_members (
	id uuid primary key default gen_random_uuid(),
	committee_id uuid not null references education_committees(id) on delete cascade,
	full_name text not null,
	role_name text not null,
	member_type text not null check (member_type in ('presedinte', 'secretar', 'membru', 'observator', 'invitat')),
	voting_right boolean not null default true,
	status text not null check (status in ('active', 'inactive', 'replaced')),
	appointed_on date not null,
	released_on date,
	institution_id text not null,
	notes text not null default '',
	created_at timestamptz not null default now(),
	updated_at timestamptz not null default now()
);

create index if not exists idx_education_committee_members_lookup
	on education_committee_members(institution_id, committee_id, status, member_type, appointed_on);

insert into education_committees (
	committee_code, school_year, committee_type, title, status, decision_reference, starts_on, ends_on, evaluation_scope, institution_id, notes
)
select
	'COM-EVAL-2026-001',
	'2025-2026',
	'evaluare_personal_didactic',
	'Comisia temporara de evaluare a personalului didactic',
	'active',
	'DEC-2026-CP-014',
	date '2026-06-01',
	date '2026-08-31',
	true,
	'school-1',
	'Constituita pentru evaluarea personalului didactic conform calendarului anual.'
where not exists (
	select 1 from education_committees where institution_id = 'school-1' and committee_code = 'COM-EVAL-2026-001'
);

insert into education_committee_members (
	committee_id, full_name, role_name, member_type, voting_right, status, appointed_on, released_on, institution_id, notes
)
select
	ec.id,
	'Director Demo',
	'Director',
	'presedinte',
	true,
	'active',
	date '2026-06-01',
	null,
	ec.institution_id,
	'Presedinte al comisiei de evaluare.'
from education_committees ec
where ec.institution_id = 'school-1' and ec.committee_code = 'COM-EVAL-2026-001'
and not exists (
	select 1 from education_committee_members ecm
	where ecm.committee_id = ec.id and ecm.full_name = 'Director Demo' and ecm.member_type = 'presedinte'
);

insert into education_committee_members (
	committee_id, full_name, role_name, member_type, voting_right, status, appointed_on, released_on, institution_id, notes
)
select
	ec.id,
	'Secretariat Demo',
	'Secretar unitate',
	'secretar',
	true,
	'active',
	date '2026-06-01',
	null,
	ec.institution_id,
	'Secretar al comisiei.'
from education_committees ec
where ec.institution_id = 'school-1' and ec.committee_code = 'COM-EVAL-2026-001'
and not exists (
	select 1 from education_committee_members ecm
	where ecm.committee_id = ec.id and ecm.full_name = 'Secretariat Demo' and ecm.member_type = 'secretar'
);
