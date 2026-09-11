-- Complete the statutory pedagogical metadata and make every newly created or
-- edited active portfolio document a verified snapshot of one immutable
-- eArhiva version. Existing legacy rows remain readable and withdrawable; the
-- NOT VALID constraints are enforced for all future writes without inventing
-- provenance for historical data.
alter table education_portfolio_documents
	add column if not exists description text not null default '',
	add column if not exists school_year text not null default '',
	add column if not exists subject_discipline text not null default '',
	add column if not exists applicable_class text not null default '',
	add column if not exists competencies text[] not null default '{}',
	add column if not exists last_change_reason text not null default '';

alter table education_portfolio_documents
	drop constraint if exists education_portfolio_documents_pedagogical_metadata_check;
alter table education_portfolio_documents
	add constraint education_portfolio_documents_pedagogical_metadata_check
	check (
		status = 'withdrawn'
		or (
			btrim(description) <> ''
			and btrim(school_year) <> ''
			and btrim(subject_discipline) <> ''
			and btrim(applicable_class) <> ''
			and cardinality(competencies) > 0
			and array_position(competencies, '') is null
		)
	) not valid;

alter table education_portfolio_documents
	drop constraint if exists education_portfolio_documents_archive_snapshot_check;
alter table education_portfolio_documents
	add constraint education_portfolio_documents_archive_snapshot_check
	check (
		status = 'withdrawn'
		or (
			archive_document_id is not null
			and archive_version_id is not null
			and archive_version_no is not null and archive_version_no > 0
			and btrim(archive_source_bucket) <> ''
			and btrim(archive_source_object_key) <> ''
			and archive_sha256 ~ '^[0-9a-f]{64}$'
			and file_reference = 'archive://' || archive_document_id::text
		)
	) not valid;

create or replace function public.enforce_education_portfolio_document_archive_contract()
returns trigger
language plpgsql
as $$
declare
	valid_snapshot boolean;
begin
	if new.status = 'withdrawn' then
		return new;
	end if;

	select exists (
		select 1
		from archive_documents document
		join archive_document_versions version
			on version.id = new.archive_version_id
			and version.document_id = document.id
		where document.id = new.archive_document_id
			and document.institution_id = new.institution_id
			and document.status = 'ready'
			and version.institution_id = new.institution_id
			and version.status = 'active'
			and version.version_no = new.archive_version_no
			and version.source_bucket = new.archive_source_bucket
			and version.source_object_key = new.archive_source_object_key
			and lower(version.source_sha256) = new.archive_sha256
	) into valid_snapshot;

	if not coalesce(valid_snapshot, false) then
		raise exception 'portfolio evidence requires a ready immutable eArhiva version snapshot';
	end if;
	return new;
end;
$$;

drop trigger if exists trg_education_portfolio_document_archive_contract on education_portfolio_documents;
create trigger trg_education_portfolio_document_archive_contract
	before insert or update on education_portfolio_documents
	for each row execute function public.enforce_education_portfolio_document_archive_contract();

create index if not exists idx_education_portfolio_documents_competencies
	on education_portfolio_documents using gin (competencies);
