-- Official school records are evidentiary artifacts. Once their legally
-- meaningful lifecycle state is reached, they are immutable at the database
-- boundary. The narrow legal transitions below record server-derived actor and
-- time provenance; every other update or hard delete fails closed.
alter table education_decisions
	add column if not exists published_at timestamptz,
	add column if not exists published_by_subject text not null default '';

alter table education_publications
	add column if not exists withdrawn_at timestamptz,
	add column if not exists withdrawn_by_subject text not null default '',
	add column if not exists withdrawal_reason text not null default '';

alter table education_managerial_documents
	add column if not exists archived_at timestamptz,
	add column if not exists archived_by_subject text not null default '',
	add column if not exists archival_reason text not null default '';

create or replace function public.education_meeting_is_officially_sealed(
	meeting_ref uuid,
	institution_ref text
)
returns boolean
language sql
stable
as $$
	select exists (
		select 1
		from education_meetings meeting
		where meeting.id = meeting_ref
			and meeting.institution_id = institution_ref
			and (
				meeting.status = 'published'
				or exists (
					select 1 from education_meeting_documents document
					where document.meeting_id = meeting.id
						and document.institution_id = meeting.institution_id
						and document.publication_status = 'publicat'
				)
				or exists (
					select 1 from education_meeting_minutes minute
					where minute.meeting_id = meeting.id
						and minute.institution_id = meeting.institution_id
						and minute.follow_up_status = 'inchis'
				)
				or exists (
					select 1 from education_meeting_resolutions resolution
					where resolution.meeting_id = meeting.id
						and resolution.institution_id = meeting.institution_id
						and resolution.publication_status = 'publicat'
				)
			)
	);
$$;

create or replace function public.enforce_education_official_artifact_immutability()
returns trigger
language plpgsql
as $$
declare
	actor_subject text := nullif(btrim(current_setting('app.actor_subject', true)), '');
	meeting_ref uuid;
	institution_ref text;
begin
	if tg_table_name = 'education_meetings' then
		if tg_op in ('UPDATE', 'DELETE') and public.education_meeting_is_officially_sealed(old.id, old.institution_id) then
			if tg_op = 'DELETE' then
				raise exception 'published meeting or meeting with finalized evidence cannot be hard-deleted';
			end if;
			raise exception 'published meeting or meeting with finalized evidence is immutable';
		end if;
		return case when tg_op = 'DELETE' then old else new end;
	end if;

	if tg_table_name in ('education_meeting_participants', 'education_meeting_votes') then
		meeting_ref := case when tg_op = 'DELETE' then old.meeting_id else new.meeting_id end;
		institution_ref := case when tg_op = 'DELETE' then old.institution_id else new.institution_id end;
		if public.education_meeting_is_officially_sealed(meeting_ref, institution_ref)
			or (tg_op = 'UPDATE' and public.education_meeting_is_officially_sealed(old.meeting_id, old.institution_id)) then
			raise exception 'meeting participant or vote mutation is blocked after publication or finalized evidence';
		end if;
		return case when tg_op = 'DELETE' then old else new end;
	end if;

	if tg_table_name = 'education_decisions' then
		if tg_op = 'DELETE' and (old.status in ('approved', 'published') or old.publication_status = 'published') then
			raise exception 'approved or published decision cannot be hard-deleted';
		elsif tg_op = 'UPDATE' and (old.status in ('approved', 'published') or old.publication_status = 'published') then
			if old.status = 'approved'
				and old.publication_status <> 'published'
				and new.status = 'published'
				and new.publication_status = 'published'
				and new.published_at is null
				and btrim(new.published_by_subject) = ''
				and actor_subject is not null
				and (to_jsonb(new) - array['status', 'publication_status', 'published_at', 'published_by_subject', 'updated_at'])
					is not distinct from (to_jsonb(old) - array['status', 'publication_status', 'published_at', 'published_by_subject', 'updated_at']) then
				new.published_at := now();
				new.published_by_subject := actor_subject;
				return new;
			end if;
			raise exception 'approved or published decision is immutable except the provenance-bearing approved-to-published transition';
		end if;
		return case when tg_op = 'DELETE' then old else new end;
	end if;

	if tg_table_name = 'education_meeting_documents' then
		if tg_op = 'DELETE' and old.publication_status = 'publicat' then
			raise exception 'published meeting document cannot be hard-deleted';
		elsif tg_op = 'UPDATE' and old.publication_status = 'publicat' then
			raise exception 'published meeting document is immutable; legal withdrawal or supersession is not modeled';
		end if;
		return case when tg_op = 'DELETE' then old else new end;
	end if;

	if tg_table_name = 'education_managerial_documents' then
		if tg_op = 'DELETE' and old.document_status = 'published' then
			raise exception 'published managerial document cannot be hard-deleted';
		elsif tg_op = 'UPDATE' and old.document_status in ('published', 'archived') then
			if old.document_status = 'published'
				and new.document_status = 'archived'
				and new.archived_at is null
				and btrim(new.archived_by_subject) = ''
				and btrim(new.archival_reason) <> ''
				and actor_subject is not null
				and (to_jsonb(new) - array['document_status', 'archived_at', 'archived_by_subject', 'archival_reason', 'updated_at'])
					is not distinct from (to_jsonb(old) - array['document_status', 'archived_at', 'archived_by_subject', 'archival_reason', 'updated_at']) then
				new.archived_at := now();
				new.archived_by_subject := actor_subject;
				return new;
			end if;
			raise exception 'published or archived managerial document is immutable except the provenance-bearing published-to-archived transition';
		end if;
		return case when tg_op = 'DELETE' then old else new end;
	end if;

	if tg_table_name = 'education_meeting_minutes' then
		if tg_op = 'DELETE' and old.follow_up_status = 'inchis' then
			raise exception 'finalized meeting minute cannot be hard-deleted';
		elsif tg_op = 'UPDATE' and old.follow_up_status = 'inchis' then
			raise exception 'finalized meeting minute is immutable; legal withdrawal or supersession is not modeled';
		end if;
		return case when tg_op = 'DELETE' then old else new end;
	end if;

	if tg_table_name = 'education_meeting_resolutions' then
		if tg_op = 'DELETE' and old.publication_status = 'publicat' then
			raise exception 'published resolution cannot be hard-deleted';
		elsif tg_op = 'UPDATE' and old.publication_status = 'publicat' then
			raise exception 'published resolution is immutable; legal withdrawal or supersession is not modeled';
		end if;
		return case when tg_op = 'DELETE' then old else new end;
	end if;

	if tg_table_name = 'education_publications' then
		if tg_op = 'DELETE' and old.publication_status = 'publicat' then
			raise exception 'published education publication cannot be hard-deleted';
		elsif tg_op = 'UPDATE' and old.publication_status in ('publicat', 'retras') then
			if old.publication_status = 'publicat'
				and new.publication_status = 'retras'
				and new.withdrawn_at is null
				and btrim(new.withdrawn_by_subject) = ''
				and btrim(new.withdrawal_reason) <> ''
				and actor_subject is not null
				and (to_jsonb(new) - array['publication_status', 'withdrawn_at', 'withdrawn_by_subject', 'withdrawal_reason', 'updated_at'])
					is not distinct from (to_jsonb(old) - array['publication_status', 'withdrawn_at', 'withdrawn_by_subject', 'withdrawal_reason', 'updated_at']) then
				new.withdrawn_at := now();
				new.withdrawn_by_subject := actor_subject;
				return new;
			end if;
			raise exception 'published or withdrawn education publication is immutable except the provenance-bearing publicat-to-retras transition';
		end if;
		return case when tg_op = 'DELETE' then old else new end;
	end if;

	raise exception 'official artifact immutability trigger is bound to unsupported table %', tg_table_name;
end;
$$;

drop trigger if exists trg_education_meetings_official_immutability on education_meetings;
create trigger trg_education_meetings_official_immutability
	before update or delete on education_meetings
	for each row execute function public.enforce_education_official_artifact_immutability();

drop trigger if exists trg_education_meeting_participants_official_immutability on education_meeting_participants;
create trigger trg_education_meeting_participants_official_immutability
	before insert or update or delete on education_meeting_participants
	for each row execute function public.enforce_education_official_artifact_immutability();

drop trigger if exists trg_education_meeting_votes_official_immutability on education_meeting_votes;
create trigger trg_education_meeting_votes_official_immutability
	before insert or update or delete on education_meeting_votes
	for each row execute function public.enforce_education_official_artifact_immutability();

drop trigger if exists trg_education_decisions_official_immutability on education_decisions;
create trigger trg_education_decisions_official_immutability
	before update or delete on education_decisions
	for each row execute function public.enforce_education_official_artifact_immutability();

drop trigger if exists trg_education_meeting_documents_official_immutability on education_meeting_documents;
create trigger trg_education_meeting_documents_official_immutability
	before update or delete on education_meeting_documents
	for each row execute function public.enforce_education_official_artifact_immutability();

drop trigger if exists trg_education_managerial_documents_official_immutability on education_managerial_documents;
create trigger trg_education_managerial_documents_official_immutability
	before update or delete on education_managerial_documents
	for each row execute function public.enforce_education_official_artifact_immutability();

drop trigger if exists trg_education_meeting_minutes_official_immutability on education_meeting_minutes;
create trigger trg_education_meeting_minutes_official_immutability
	before update or delete on education_meeting_minutes
	for each row execute function public.enforce_education_official_artifact_immutability();

drop trigger if exists trg_education_meeting_resolutions_official_immutability on education_meeting_resolutions;
create trigger trg_education_meeting_resolutions_official_immutability
	before update or delete on education_meeting_resolutions
	for each row execute function public.enforce_education_official_artifact_immutability();

drop trigger if exists trg_education_publications_official_immutability on education_publications;
create trigger trg_education_publications_official_immutability
	before update or delete on education_publications
	for each row execute function public.enforce_education_official_artifact_immutability();
