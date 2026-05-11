create table if not exists registratura_document_links (
	id uuid primary key default gen_random_uuid(),
	document_id uuid not null references registratura_documents(id) on delete cascade,
	source_module text not null,
	source_record_id uuid not null,
	relation_type text not null check (relation_type in ('primary', 'supporting', 'decision', 'archive_basis', 'gdpr_basis')),
	created_at timestamptz not null default now(),
	unique (document_id, source_module, source_record_id, relation_type)
);

insert into app_permissions(code, label) values
	('registratura.links.read', 'Read document links'),
	('registratura.links.manage', 'Manage document links')
on conflict (code) do nothing;

insert into app_user_permissions(user_id, permission_code)
select id, permission_code
from app_users
cross join (values
	('registratura.links.read'),
	('registratura.links.manage')
) as permissions(permission_code)
where sub = 'usr-001'
on conflict do nothing;

insert into registratura_document_links (document_id, source_module, source_record_id, relation_type)
select d.id, 'education.portfolios', p.id, 'primary'
from registratura_documents d
join education_portfolios p on p.portfolio_code = 'PORT-CD-2026-1001'
where d.registry_number = 'REG-2026-0003'
on conflict do nothing;

insert into registratura_document_links (document_id, source_module, source_record_id, relation_type)
select d.id, 'gdpr.subject_requests', g.id, 'gdpr_basis'
from registratura_documents d
join gdpr_subject_requests g on g.request_code = 'DSR-2026-0001'
where d.registry_number = 'REG-2026-0004'
on conflict do nothing;
