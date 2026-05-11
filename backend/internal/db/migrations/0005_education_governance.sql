create table if not exists education_meetings (
	id uuid primary key default gen_random_uuid(),
	school_year text not null,
	organism text not null,
	title text not null,
	meeting_type text not null check (meeting_type in ('ordinary', 'extraordinary')),
	status text not null check (status in ('draft', 'scheduled', 'held', 'published')),
	quorum_required integer not null check (quorum_required >= 1),
	participants_count integer not null default 0 check (participants_count >= 0),
	meeting_date date not null,
	location text not null default '',
	chairperson text not null default '',
	secretary_name text not null default '',
	institution_id text not null,
	summary text not null default '',
	created_at timestamptz not null default now(),
	updated_at timestamptz not null default now()
);

insert into education_meetings (
	school_year,
	organism,
	title,
	meeting_type,
	status,
	quorum_required,
	participants_count,
	meeting_date,
	location,
	chairperson,
	secretary_name,
	institution_id,
	summary
)
values
	('2025-2026', 'ca', 'Aprobarea programului Școala Altfel', 'ordinary', 'scheduled', 7, 0, current_date + 2, 'Sala profesorală', 'Director', 'Secretar CA', 'inst-001', 'Ordine de zi: program, responsabilități, resurse.'),
	('2025-2026', 'cp', 'Validarea planificărilor pe semestrul II', 'ordinary', 'held', 25, 28, current_date - 5, 'Amfiteatru', 'Director adjunct', 'Secretar CP', 'inst-001', 'Au fost validate planificările și actualizările de curriculum.'),
	('2025-2026', 'ceac', 'Analiza indicatorilor de calitate', 'extraordinary', 'draft', 5, 0, current_date + 7, 'Cabinet CEAC', 'Coordonator CEAC', 'Secretar CEAC', 'inst-001', 'Pregătire raport intermediar și plan de măsuri.'),
	('2025-2026', 'cfdcd', 'Planul de formare continuă', 'ordinary', 'published', 4, 4, current_date - 14, 'Online', 'Responsabil CFDCD', 'Secretar CFDCD', 'inst-001', 'Decizii publicate în portalul intern al școlii.'),
	('2025-2026', 'ca', 'Avizarea proiectului de buget', 'extraordinary', 'held', 7, 8, current_date - 1, 'Sala de consiliu', 'Director', 'Secretar CA', 'inst-001', 'Bugetul a fost avizat cu recomandări privind investițiile IT.')
on conflict do nothing;

insert into app_permissions(code, label) values
	('education.governance.read', 'Read education governance'),
	('education.governance.manage', 'Manage education governance')
on conflict (code) do nothing;

insert into app_user_permissions(user_id, permission_code)
select id, permission_code
from app_users
cross join (values
	('education.governance.read'),
	('education.governance.manage')
) as permissions(permission_code)
where sub = 'usr-001'
on conflict do nothing;
