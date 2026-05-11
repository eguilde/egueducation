create table if not exists workflow_dossier_requirements (
	id uuid primary key default gen_random_uuid(),
	source_module text not null,
	relation_type text not null check (relation_type in ('primary', 'supporting', 'decision', 'archive_basis', 'gdpr_basis')),
	min_count integer not null default 1 check (min_count > 0),
	required_for_readiness boolean not null default true,
	required_for_submit boolean not null default true,
	required_for_approve boolean not null default true,
	created_at timestamptz not null default now(),
	updated_at timestamptz not null default now(),
	unique (source_module, relation_type)
);

insert into app_permissions(code, label) values
	('admin.dossier_requirements.read', 'Read dossier requirements'),
	('admin.dossier_requirements.manage', 'Manage dossier requirements')
on conflict (code) do nothing;

insert into app_user_permissions(user_id, permission_code)
select id, permission_code
from app_users
cross join (values
	('admin.dossier_requirements.read'),
	('admin.dossier_requirements.manage')
) as permissions(permission_code)
where sub = 'usr-001'
on conflict do nothing;

insert into workflow_dossier_requirements (
	source_module,
	relation_type,
	min_count,
	required_for_readiness,
	required_for_submit,
	required_for_approve
) values
	('education.governance', 'decision', 1, true, true, true),
	('education.personnel', 'supporting', 1, true, true, true),
	('education.mobility', 'primary', 1, true, true, true),
	('education.mobility', 'supporting', 1, true, true, true),
	('education.gradatii', 'primary', 1, true, true, true),
	('education.portfolios', 'primary', 1, true, true, true),
	('gdpr.subject_requests', 'gdpr_basis', 1, true, true, true)
on conflict (source_module, relation_type) do update
set min_count = excluded.min_count,
	required_for_readiness = excluded.required_for_readiness,
	required_for_submit = excluded.required_for_submit,
	required_for_approve = excluded.required_for_approve,
	updated_at = now();
