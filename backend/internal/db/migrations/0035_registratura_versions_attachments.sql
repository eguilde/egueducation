create table if not exists registratura_document_versions (
	id uuid primary key default gen_random_uuid(),
	document_id uuid not null references registratura_documents (id) on delete cascade,
	version_no integer not null,
	subject text not null,
	document_type text not null,
	direction text not null,
	status text not null,
	correspondent text not null,
	assigned_to text not null default '',
	confidentiality text not null,
	summary text not null default '',
	due_date date,
	change_notes text not null default '',
	created_by text not null default '',
	created_at timestamptz not null default now(),
	unique (document_id, version_no)
);

create table if not exists registratura_document_attachments (
	id uuid primary key default gen_random_uuid(),
	document_id uuid not null references registratura_documents (id) on delete cascade,
	title text not null,
	file_name text not null,
	mime_type text not null,
	storage_key text not null,
	size_bytes bigint not null default 0,
	category text not null,
	status text not null,
	uploaded_by text not null default '',
	uploaded_at timestamptz not null default now()
);

insert into registratura_document_versions (
	document_id,
	version_no,
	subject,
	document_type,
	direction,
	status,
	correspondent,
	assigned_to,
	confidentiality,
	summary,
	due_date,
	change_notes,
	created_by,
	created_at
)
select
	d.id,
	1,
	d.subject,
	d.document_type,
	d.direction,
	d.status,
	d.correspondent,
	d.assigned_to,
	d.confidentiality,
	d.summary,
	d.due_date,
	'Inițializare istoric versiuni',
	'migration',
	d.created_at
from registratura_documents d
where not exists (
	select 1
	from registratura_document_versions v
	where v.document_id = d.id
);

insert into registratura_document_attachments (
	document_id,
	title,
	file_name,
	mime_type,
	storage_key,
	size_bytes,
	category,
	status,
	uploaded_by,
	uploaded_at
)
select
	d.id,
	case d.document_type
		when 'cerere' then 'Cerere scanată'
		when 'decizie' then 'Document decizie semnat'
		else 'Document suport'
	end,
	case d.document_type
		when 'cerere' then replace(lower(d.registry_number), '-', '_') || '_cerere.pdf'
		when 'decizie' then replace(lower(d.registry_number), '-', '_') || '_decizie.pdf'
		else replace(lower(d.registry_number), '-', '_') || '_suport.pdf'
	end,
	'application/pdf',
	'registratura/' || lower(d.registry_number) || '/v1/document.pdf',
	245760,
	case d.document_type
		when 'cerere' then 'incoming_scan'
		when 'decizie' then 'signed_decision'
		else 'supporting'
	end,
	case
		when d.status = 'archived' then 'archived'
		else 'active'
	end,
	'seed',
	d.created_at
from registratura_documents d
where d.registry_number in ('REG-2026-0001', 'REG-2026-0003', 'REG-2026-0007')
	and not exists (
		select 1
		from registratura_document_attachments a
		where a.document_id = d.id
	);
