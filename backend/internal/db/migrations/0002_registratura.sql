create table if not exists registratura_documents (
	id uuid primary key default gen_random_uuid(),
	registry_number text not null unique,
	subject text not null,
	document_type text not null,
	direction text not null,
	status text not null,
	correspondent text not null,
	assigned_to text not null default '',
	institution_id text not null,
	confidentiality text not null default 'normal',
	summary text not null default '',
	registered_at timestamptz not null default now(),
	due_date date,
	created_at timestamptz not null default now(),
	updated_at timestamptz not null default now(),
	check (direction in ('intrare', 'iesire', 'intern')),
	check (status in ('draft', 'registered', 'in_workflow', 'archived')),
	check (confidentiality in ('normal', 'internal', 'confidential'))
);

insert into registratura_documents (
	registry_number,
	subject,
	document_type,
	direction,
	status,
	correspondent,
	assigned_to,
	institution_id,
	confidentiality,
	summary,
	registered_at,
	due_date
) values
	('REG-2026-0001', 'Cerere transfer elev', 'cerere', 'intrare', 'registered', 'Ionescu Maria', 'Secretariat', 'inst-001', 'normal', 'Cerere de transfer pentru anul scolar 2026-2027.', '2026-05-02T08:15:00Z', '2026-05-16'),
	('REG-2026-0002', 'Raspuns solicitare ISJ', 'adresa', 'iesire', 'registered', 'Inspectoratul Scolar Timis', 'Director', 'inst-001', 'internal', 'Transmitere raspuns privind situatia incadrarilor.', '2026-05-02T11:30:00Z', '2026-05-09'),
	('REG-2026-0003', 'Decizie constituire comisie', 'decizie', 'intern', 'in_workflow', 'Conducerea unitatii', 'Workflow Admin', 'inst-001', 'internal', 'Document intern pentru constituirea comisiei de mobilitate.', '2026-05-03T09:00:00Z', '2026-05-20'),
	('REG-2026-0004', 'Solicitare adeverinta vechime', 'cerere', 'intrare', 'registered', 'Popa Andrei', 'HR', 'inst-001', 'confidential', 'Solicitare adeverinta de vechime pentru dosarul de mobilitate.', '2026-05-04T07:45:00Z', '2026-05-12'),
	('REG-2026-0005', 'Transmitere convocator CA', 'convocator', 'iesire', 'archived', 'Membrii Consiliului de Administratie', 'Secretariat', 'inst-001', 'internal', 'Convocator pentru sedinta CA din 10 mai 2026.', '2026-05-04T12:20:00Z', '2026-05-10'),
	('REG-2026-0006', 'Raport activitate semestrial', 'raport', 'intern', 'registered', 'Comisia pentru curriculum', 'Director', 'inst-001', 'normal', 'Raport intern privind activitatea semestriala.', '2026-05-05T10:05:00Z', '2026-05-24'),
	('REG-2026-0007', 'Cerere acces date personale', 'cerere', 'intrare', 'in_workflow', 'Stan Roxana', 'GDPR Officer', 'inst-001', 'confidential', 'Cerere in temei GDPR pentru export date personale.', '2026-05-06T08:50:00Z', '2026-05-13'),
	('REG-2026-0008', 'Adresa DSP privind avize', 'adresa', 'intrare', 'registered', 'Directia de Sanatate Publica', 'Secretariat', 'inst-001', 'normal', 'Solicitare actualizare evidenta avizelor medicale.', '2026-05-06T13:10:00Z', '2026-05-22'),
	('REG-2026-0009', 'Planificare evaluare anuala', 'plan', 'intern', 'draft', 'Compartiment resurse umane', 'HR', 'inst-001', 'internal', 'Planificare preliminara pentru evaluarile anuale ale personalului.', '2026-05-07T09:35:00Z', '2026-05-28'),
	('REG-2026-0010', 'Transmitere situatie burse', 'raport', 'iesire', 'registered', 'Primaria Balotesti', 'Secretariat', 'inst-001', 'normal', 'Situatia burselor scolare pentru luna aprilie.', '2026-05-07T14:40:00Z', '2026-05-14'),
	('REG-2026-0011', 'Sesizare parinte clasa a VII-a', 'sesizare', 'intrare', 'in_workflow', 'Dumitrescu Elena', 'Director', 'inst-001', 'confidential', 'Sesizare privind incident disciplinar la clasa a VII-a.', '2026-05-08T07:20:00Z', '2026-05-11'),
	('REG-2026-0012', 'Nota interna inventar arhiva', 'nota', 'intern', 'archived', 'Arhiva scolii', 'Arhivar', 'inst-001', 'internal', 'Inventar documente transferate catre eArhiva.', '2026-05-08T16:10:00Z', null)
on conflict (registry_number) do nothing;
