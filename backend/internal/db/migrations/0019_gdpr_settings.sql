create table if not exists gdpr_operational_settings (
	code text primary key,
	value_type text not null,
	value_text text,
	value_bool boolean,
	value_int integer,
	updated_at timestamptz not null default now(),
	check (value_type in ('text', 'bool', 'int'))
);

insert into app_permissions(code, label) values
	('admin.gdpr_settings.read', 'Read GDPR settings'),
	('admin.gdpr_settings.manage', 'Manage GDPR settings')
on conflict (code) do nothing;

insert into app_user_permissions(user_id, permission_code)
select id, permission_code
from app_users
cross join (values
	('admin.gdpr_settings.read'),
	('admin.gdpr_settings.manage')
) as permissions(permission_code)
where sub = 'usr-001'
on conflict do nothing;

insert into app_nomenclatures (domain, code, label_ro, label_en, active, sort_order) values
	('gdpr_domain', 'registratura', 'Registratură', 'Registry', true, 10),
	('gdpr_domain', 'workflow', 'Workflow', 'Workflow', true, 20),
	('gdpr_domain', 'earchiva', 'eArhivă', 'eArchive', true, 30),
	('gdpr_domain', 'education', 'Educație', 'Education', true, 40),
	('gdpr_domain', 'hr', 'Resurse umane', 'Human resources', true, 50),
	('gdpr_policy_status', 'draft', 'Ciornă', 'Draft', true, 10),
	('gdpr_policy_status', 'active', 'Activă', 'Active', true, 20),
	('gdpr_policy_status', 'review', 'În revizuire', 'In review', true, 30),
	('gdpr_policy_status', 'retired', 'Retrasă', 'Retired', true, 40),
	('gdpr_request_type', 'access', 'Acces', 'Access', true, 10),
	('gdpr_request_type', 'rectification', 'Rectificare', 'Rectification', true, 20),
	('gdpr_request_type', 'erasure', 'Ștergere', 'Erasure', true, 30),
	('gdpr_request_type', 'restriction', 'Restricționare', 'Restriction', true, 40),
	('gdpr_request_type', 'portability', 'Portabilitate', 'Portability', true, 50),
	('gdpr_request_type', 'objection', 'Opoziție', 'Objection', true, 60),
	('gdpr_request_status', 'received', 'Primită', 'Received', true, 10),
	('gdpr_request_status', 'identity_check', 'Verificare identitate', 'Identity check', true, 20),
	('gdpr_request_status', 'in_progress', 'În lucru', 'In progress', true, 30),
	('gdpr_request_status', 'waiting_approval', 'Așteaptă aprobare', 'Waiting approval', true, 40),
	('gdpr_request_status', 'completed', 'Finalizată', 'Completed', true, 50),
	('gdpr_request_status', 'rejected', 'Respinsă', 'Rejected', true, 60),
	('gdpr_source_module', 'registratura', 'Registratură', 'Registry', true, 10),
	('gdpr_source_module', 'workflow', 'Workflow', 'Workflow', true, 20),
	('gdpr_source_module', 'earchiva', 'eArhivă', 'eArchive', true, 30),
	('gdpr_source_module', 'education', 'Educație', 'Education', true, 40)
on conflict (domain, code) do update
set label_ro = excluded.label_ro,
	label_en = excluded.label_en,
	active = excluded.active,
	sort_order = excluded.sort_order,
	updated_at = now();

insert into gdpr_operational_settings (code, value_type, value_bool, value_int, value_text) values
	('publication_anonymization_required', 'bool', true, null, null),
	('subject_export_requires_approval', 'bool', true, null, null),
	('default_response_sla_days', 'int', null, 30, null),
	('retention_review_notice_days', 'int', null, 90, null),
	('portfolio_consent_required', 'bool', true, null, null),
	('portfolio_authenticity_required', 'bool', true, null, null)
on conflict (code) do update
set value_type = excluded.value_type,
	value_bool = excluded.value_bool,
	value_int = excluded.value_int,
	value_text = excluded.value_text,
	updated_at = now();
