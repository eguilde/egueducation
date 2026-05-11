alter table workflow_instances
	add column if not exists source_record_id uuid;

update workflow_instances wi
set source_record_id = ep.id
from education_portfolios ep
where wi.source_module = 'education.portfolios'
	and wi.source_record_id is null
	and wi.document_number = ep.portfolio_code;

update workflow_instances wi
set source_record_id = em.id
from education_meetings em
where wi.source_module in ('education', 'education.governance')
	and wi.source_record_id is null
	and lower(wi.title) like '%' || lower(em.title) || '%';

update workflow_instances wi
set source_record_id = gsr.id
from gdpr_subject_requests gsr
where wi.source_module = 'gdpr.subject_requests'
	and wi.source_record_id is null
	and wi.document_number = gsr.request_code;
