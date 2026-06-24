insert into education_taxonomies (domain, code, label_ro, label_en, active, sort_order)
values
	('managerial_dossier_type', 'director_portfolio', 'Portofoliu director', 'Director portfolio', true, 80),
	('managerial_dossier_type', 'adjunct_director_portfolio', 'Portofoliu director adjunct', 'Deputy director portfolio', true, 90)
on conflict (domain, code) do update
set label_ro = excluded.label_ro,
	label_en = excluded.label_en,
	active = excluded.active,
	sort_order = excluded.sort_order;

insert into education_managerial_dossiers (
	dossier_code,
	school_year,
	dossier_type,
	title,
	status,
	owner_name,
	due_on,
	publication_required,
	institution_id,
	summary
)
values
	(
		'MGR-DIR-2026-001',
		'2025-2026',
		'director_portfolio',
		'Portofoliul directorului unitatii de invatamant',
		'in_review',
		'Raluca Stan',
		date '2026-07-10',
		true,
		'inst-001',
		'Registru managerial explicit pentru portofoliul directorului, cu documente-model si circuit de avizare.'
	),
	(
		'MGR-ADJ-2026-001',
		'2025-2026',
		'adjunct_director_portfolio',
		'Portofoliul directorului adjunct',
		'draft',
		'Mihai Enache',
		date '2026-07-15',
		false,
		'inst-001',
		'Registru managerial explicit pentru portofoliul directorului adjunct, unde unitatea are functie de adjunct.'
	)
on conflict (dossier_code) do nothing;

insert into education_managerial_documents (
	dossier_id, document_code, document_category, title, document_status, version_label,
	mandatory, publication_required, registered_on, approved_on, owner_name, file_reference, institution_id, notes
)
select
	emd.id,
	'MDOC-DIR-2026-0001',
	'evidenta',
	'Opis documente portofoliu director',
	'approved',
	'v1.0',
	true,
	false,
	date '2026-06-20',
	date '2026-06-21',
	emd.owner_name,
	'REG-MGR-DIR-2026-001',
	emd.institution_id,
	'Opis si index procedural pentru portofoliul directorului.'
from education_managerial_dossiers emd
where emd.dossier_code = 'MGR-DIR-2026-001'
on conflict (dossier_id, document_code) do nothing;

insert into education_managerial_documents (
	dossier_id, document_code, document_category, title, document_status, version_label,
	mandatory, publication_required, registered_on, approved_on, owner_name, file_reference, institution_id, notes
)
select
	emd.id,
	'MDOC-DIR-2026-0002',
	'planificare',
	'Plan managerial si documente de baza ale portofoliului directorului',
	'in_review',
	'v0.9',
	true,
	true,
	date '2026-06-22',
	null,
	emd.owner_name,
	'REG-MGR-DIR-2026-002',
	emd.institution_id,
	'Pachetul de baza pentru portofoliul directorului este in curs de avizare.'
from education_managerial_dossiers emd
where emd.dossier_code = 'MGR-DIR-2026-001'
on conflict (dossier_id, document_code) do nothing;

insert into education_managerial_documents (
	dossier_id, document_code, document_category, title, document_status, version_label,
	mandatory, publication_required, registered_on, approved_on, owner_name, file_reference, institution_id, notes
)
select
	emd.id,
	'MDOC-ADJ-2026-0001',
	'evidenta',
	'Opis documente portofoliu director adjunct',
	'draft',
	'v0.1',
	true,
	false,
	date '2026-06-24',
	null,
	emd.owner_name,
	'REG-MGR-ADJ-2026-001',
	emd.institution_id,
	'Opisul initial pentru portofoliul directorului adjunct.'
from education_managerial_dossiers emd
where emd.dossier_code = 'MGR-ADJ-2026-001'
on conflict (dossier_id, document_code) do nothing;

insert into education_managerial_workflow_steps (
	dossier_id, stage_order, stage_type, status, assigned_to, due_on, completed_on,
	requires_signature, decision_reference, institution_id, outcome_note
)
select
	emd.id,
	1,
	'elaborare',
	'completed',
	'Raluca Stan',
	date '2026-06-20',
	date '2026-06-20',
	false,
	'',
	emd.institution_id,
	'Structura portofoliului directorului a fost initializata.'
from education_managerial_dossiers emd
where emd.dossier_code = 'MGR-DIR-2026-001'
on conflict (dossier_id, stage_order, stage_type) do nothing;

insert into education_managerial_workflow_steps (
	dossier_id, stage_order, stage_type, status, assigned_to, due_on, completed_on,
	requires_signature, decision_reference, institution_id, outcome_note
)
select
	emd.id,
	2,
	'avizare_cp',
	'in_progress',
	'Secretar CP',
	date '2026-07-02',
	null,
	true,
	'PV-CP-2026-PORT-01',
	emd.institution_id,
	'Portofoliul directorului este pe circuitul procedural de avizare.'
from education_managerial_dossiers emd
where emd.dossier_code = 'MGR-DIR-2026-001'
on conflict (dossier_id, stage_order, stage_type) do nothing;

insert into education_managerial_workflow_steps (
	dossier_id, stage_order, stage_type, status, assigned_to, due_on, completed_on,
	requires_signature, decision_reference, institution_id, outcome_note
)
select
	emd.id,
	1,
	'elaborare',
	'in_progress',
	'Mihai Enache',
	date '2026-07-01',
	null,
	false,
	'',
	emd.institution_id,
	'Colectarea documentelor de baza este in curs.'
from education_managerial_dossiers emd
where emd.dossier_code = 'MGR-ADJ-2026-001'
on conflict (dossier_id, stage_order, stage_type) do nothing;
