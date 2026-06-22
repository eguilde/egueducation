create table if not exists education_requirement_catalog (
	id uuid primary key default gen_random_uuid(),
	domain text not null,
	code text not null,
	title_ro text not null,
	title_en text not null,
	source_ref text not null,
	requirement_type text not null,
	implementation_status text not null check (implementation_status in ('implemented', 'partial', 'planned')),
	priority integer not null default 100,
	notes text not null default '',
	unique(domain, code)
);

create table if not exists education_portfolio_sections (
	id uuid primary key default gen_random_uuid(),
	section_code text not null,
	component_code text not null,
	label_ro text not null,
	label_en text not null,
	example_documents text[] not null default '{}',
	required boolean not null default false,
	sensitive_data boolean not null default false,
	retention_rule text not null default 'portfolio_retention_3_years_after_activity_end',
	sort_order integer not null default 100,
	active boolean not null default true,
	unique(section_code, component_code)
);

insert into education_requirement_catalog (
	domain,
	code,
	title_ro,
	title_en,
	source_ref,
	requirement_type,
	implementation_status,
	priority,
	notes
) values
	('governance', 'school_leadership_bodies', 'Conducerea unității conlucrează cu CA, CEAC, CFDCD, CP, autorități locale, părinți, sindicate și consiliul elevilor.', 'School leadership cooperates with CA, CEAC, CFDCD, teachers council and institutional partners.', 'Legea 198/2023 art. 128', 'legal_structure', 'partial', 10, 'UI must model organism memberships and meeting workflows.'),
	('governance', 'meeting_file', 'Ședințele trebuie să păstreze convocator, prezență, cvorum, agendă, minute, anexe, voturi, semnături și custodie.', 'Meetings must preserve convocation, attendance, quorum, agenda, minutes, annexes, votes, signatures and custody.', 'ROFUIP / school governance practice', 'workflow_dossier', 'partial', 20, 'Document links and workflow dossiers are required for each meeting.'),
	('decisions', 'publication_anonymization', 'Deciziile publicabile necesită verificare legală și anonimizare când includ date personale.', 'Publishable decisions need legal review and anonymization when they contain personal data.', 'GDPR + Legea 198/2023 governance publication practice', 'gdpr_control', 'partial', 30, 'Publication status must include pending anonymization.'),
	('portfolio', 'methodology_draft', 'Portofoliul profesional al cadrului didactic este configurabil și versionat până la publicarea textului final.', 'Teacher professional portfolio is configurable and versioned until final legal text.', 'MEC consultare publică 16.03.2026 - Proiect Metodologie portofoliu CD', 'draft_methodology', 'implemented', 40, 'Catalog is seeded from project and annexes, but must remain configurable.'),
	('portfolio', 'structure_annex_1', 'Structura portofoliului include date personale, identificare profesională, studii, documente de carieră, activitate didactică, formare și dovezi profesionale.', 'Portfolio structure includes personal/professional identification, studies, career records, teaching activity, training and professional evidence.', 'Anexa 1 - Structura portofoliu', 'portfolio_structure', 'implemented', 50, 'Sections/components are seeded in education_portfolio_sections.'),
	('portfolio', 'authenticity_declaration', 'Portofoliul trebuie să poată captura declarația de autenticitate.', 'Portfolio must capture authenticity declaration.', 'Anexa 2 - Declarație portofoliu', 'declaration', 'implemented', 60, 'Existing portfolio fields track authenticity_declared.'),
	('portfolio', 'transfer_retention', 'Transferul digital între școli și păstrarea 3 ani după încetarea activității trebuie urmărite explicit.', 'Digital transfer between schools and 3-year retention after activity end must be tracked explicitly.', 'Proiect Metodologie portofoliu CD', 'retention_transfer', 'partial', 70, 'Existing fields track transfer_status and retention_until; lifecycle actions still needed.'),
	('personnel', 'annual_evaluation', 'Evaluarea anuală a personalului didactic include autoevaluare și evaluare pe baza fișei postului/fișei de evaluare.', 'Annual evaluation includes self-assessment and job-description/evaluation-sheet based evaluation.', 'Metodologii evaluare anuală personal didactic', 'personnel_workflow', 'partial', 80, 'Need scoring criteria and contestation workflow.'),
	('personnel', 'mobility_2025_2026', 'Mobilitatea personalului didactic 2025-2026 folosește criterii și punctaje pe studii, grade, vechime și activitate profesională.', 'Teacher mobility uses criteria and scores for studies, grades, seniority and professional activity.', 'OME 7495/2024 mobilitate 2025-2026', 'personnel_workflow', 'partial', 90, 'Need configurable scoring criteria.'),
	('personnel', 'merit_grant_2025', 'Gradația de merit 2025 are dosar, cerere, declarație, autoevaluare, evaluare și contestații.', '2025 merit grant has dossier, request, declaration, self-assessment, evaluation and appeals.', 'Ordin 3745/2025 gradații merit', 'personnel_workflow', 'partial', 100, 'Need dossier checklist and appeal states.')
on conflict (domain, code) do update
set title_ro = excluded.title_ro,
	title_en = excluded.title_en,
	source_ref = excluded.source_ref,
	requirement_type = excluded.requirement_type,
	implementation_status = excluded.implementation_status,
	priority = excluded.priority,
	notes = excluded.notes;

insert into education_portfolio_sections (
	section_code,
	component_code,
	label_ro,
	label_en,
	example_documents,
	required,
	sensitive_data,
	sort_order
) values
	('identificare', 'cv', 'Curriculum Vitae', 'Curriculum Vitae', array['CV Europass semnat și datat'], true, false, 10),
	('identificare', 'date_identificare', 'Date personale și identificare', 'Personal and identification data', array['Carte de identitate', 'Certificat de naștere / schimbare nume'], true, true, 20),
	('identificare', 'studii', 'Studii și calificări', 'Studies and qualifications', array['Diplome', 'Foi matricole', 'Certificate competențe'], true, true, 30),
	('cariera', 'grade_didactice', 'Definitivat / grade didactice', 'Teacher certification / grades', array['Certificat definitivat', 'Grad didactic II', 'Grad didactic I', 'Doctorat echivalat'], false, true, 40),
	('cariera', 'contracte_incadrare', 'Încadrare și fișa postului', 'Employment assignment and job description', array['Decizie încadrare', 'Fișa postului', 'Contract / acte adiționale'], true, true, 50),
	('activitate_didactica', 'planificari', 'Planificări și proiectare didactică', 'Planning and teaching design', array['Planificări calendaristice', 'Proiecte unități de învățare'], false, false, 60),
	('activitate_didactica', 'resurse_materiale', 'Resurse educaționale și materiale', 'Educational resources and materials', array['Materiale didactice', 'Resurse educaționale deschise'], false, false, 70),
	('evaluare', 'evaluari_anuale', 'Evaluări anuale', 'Annual evaluations', array['Fișă autoevaluare', 'Fișă evaluare', 'Calificativ anual'], false, true, 80),
	('formare', 'formare_continua', 'Formare continuă', 'Continuous professional development', array['Certificate cursuri', 'Adeverințe formare', 'Credite profesionale transferabile'], false, true, 90),
	('declaratii', 'autenticitate', 'Declarație de autenticitate', 'Authenticity declaration', array['Declarație pe propria răspundere privind autenticitatea documentelor'], true, true, 100),
	('declaratii', 'consimtamant', 'Consimțământ și informare GDPR', 'Consent and GDPR information', array['Informare prelucrare date', 'Consimțământ unde este necesar'], true, true, 110),
	('transfer_retinere', 'transfer_digital', 'Transfer digital între unități', 'Digital transfer between schools', array['Proces-verbal transfer', 'Confirmare primire'], false, true, 120)
on conflict (section_code, component_code) do update
set label_ro = excluded.label_ro,
	label_en = excluded.label_en,
	example_documents = excluded.example_documents,
	required = excluded.required,
	sensitive_data = excluded.sensitive_data,
	sort_order = excluded.sort_order,
	active = true;
