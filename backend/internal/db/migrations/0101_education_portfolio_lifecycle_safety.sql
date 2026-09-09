-- Portfolio evidence has an evidentiary lifecycle. Retention starts only
-- after an explicit cessation-of-activity event; it must never be derived
-- from a create/update timestamp or supplied by an API client.
alter table education_portfolios
	alter column retention_until drop not null,
	add column if not exists activity_ceased_on date,
	add column if not exists activity_cessation_reason text not null default '',
	add column if not exists retention_period_days integer not null default 1095 check (retention_period_days > 0),
	add column if not exists legal_hold_active boolean not null default false,
	add column if not exists legal_hold_reason text not null default '',
	add column if not exists legal_hold_set_at timestamptz,
	add column if not exists legal_hold_set_by_subject text not null default '',
	add column if not exists withdrawn_at timestamptz,
	add column if not exists withdrawn_by_subject text not null default '',
	add column if not exists withdrawal_reason text not null default '';

-- Historic retention values remain preserved for audit, but are not a valid
-- lifecycle deadline until activity_ceased_on is explicitly recorded.
alter table education_portfolios drop constraint if exists education_portfolios_status_check;
alter table education_portfolios add constraint education_portfolios_status_check
	check (status in ('draft', 'submitted', 'returned', 'validated', 'transferred', 'archived', 'withdrawn'));

create or replace function public.enforce_education_portfolio_lifecycle()
returns trigger
language plpgsql
as $$
begin
	if tg_op = 'DELETE' then
		raise exception 'education portfolios are evidentiary records and cannot be hard-deleted';
	end if;
	if tg_op = 'UPDATE' then
		if old.withdrawn_at is not null then
			raise exception 'withdrawn education portfolios are immutable';
		end if;
		if old.activity_ceased_on is not null
			and new.activity_ceased_on is distinct from old.activity_ceased_on then
			raise exception 'portfolio cessation event is immutable';
		end if;
		if new.withdrawn_at is not null and (
			old.status not in ('draft', 'returned')
			or old.retention_until is not null
			or old.legal_hold_active
		) then
			raise exception 'portfolio withdrawal is blocked after submission, during retention, or under legal hold';
		end if;
	end if;
	if new.activity_ceased_on is null then
		-- A new or modified record without a cessation event cannot have a
		-- server-recognized retention deadline. Existing untouched records are
		-- preserved above for historical review.
		if tg_op = 'INSERT' then
			new.retention_until := null;
		elsif new.retention_until is distinct from old.retention_until then
			new.retention_until := null;
		end if;
	else
		new.retention_until := new.activity_ceased_on + new.retention_period_days;
	end if;
	return new;
end;
$$;

drop trigger if exists trg_education_portfolio_lifecycle on education_portfolios;
create trigger trg_education_portfolio_lifecycle
	before insert or update or delete on education_portfolios
	for each row execute function public.enforce_education_portfolio_lifecycle();

-- The existing document-state trigger already locks submitted evidence. Add
-- legal-hold and withdrawal checks to that database backstop as well.
create or replace function public.enforce_education_portfolio_document_state()
returns trigger
language plpgsql
as $$
declare
	portfolio_status text;
	portfolio_ref uuid;
	portfolio_withdrawn_at timestamptz;
	portfolio_legal_hold boolean;
begin
	portfolio_ref := case when tg_op = 'DELETE' then old.portfolio_id else new.portfolio_id end;
	select status, withdrawn_at, legal_hold_active
	into portfolio_status, portfolio_withdrawn_at, portfolio_legal_hold
	from education_portfolios where id = portfolio_ref for update;
	if portfolio_status is null then
		raise exception 'portfolio does not exist';
	end if;
	if portfolio_withdrawn_at is not null or portfolio_legal_hold then
		raise exception 'portfolio evidence mutation is blocked by withdrawal or legal hold';
	end if;
	if portfolio_status not in ('draft', 'returned') then
		raise exception 'portfolio evidence is immutable outside draft or returned state';
	end if;
	return case when tg_op = 'DELETE' then old else new end;
end;
$$;

-- Checklist, opis, custody and review rows are also blocked while a legal
-- hold applies. Their established lifecycle rules remain otherwise intact.
create or replace function public.enforce_education_portfolio_related_hold()
returns trigger
language plpgsql
as $$
declare
	portfolio_ref uuid;
	portfolio_withdrawn_at timestamptz;
	portfolio_legal_hold boolean;
begin
	portfolio_ref := case when tg_op = 'DELETE' then old.portfolio_id else new.portfolio_id end;
	select withdrawn_at, legal_hold_active into portfolio_withdrawn_at, portfolio_legal_hold
	from education_portfolios where id = portfolio_ref for key share;
	if not found then
		raise exception 'portfolio does not exist';
	end if;
	if portfolio_withdrawn_at is not null or portfolio_legal_hold then
		raise exception 'portfolio related record mutation is blocked by withdrawal or legal hold';
	end if;
	return case when tg_op = 'DELETE' then old else new end;
end;
$$;
drop trigger if exists trg_education_portfolio_checklist_hold on education_portfolio_checklist;
create trigger trg_education_portfolio_checklist_hold before insert or update or delete on education_portfolio_checklist
	for each row execute function public.enforce_education_portfolio_related_hold();
drop trigger if exists trg_education_portfolio_opis_hold on education_portfolio_opis;
create trigger trg_education_portfolio_opis_hold before insert or update or delete on education_portfolio_opis
	for each row execute function public.enforce_education_portfolio_related_hold();
drop trigger if exists trg_education_portfolio_custody_hold on education_portfolio_custody;
create trigger trg_education_portfolio_custody_hold before insert or update or delete on education_portfolio_custody
	for each row execute function public.enforce_education_portfolio_related_hold();
drop trigger if exists trg_education_portfolio_reviews_hold on education_portfolio_reviews;
create trigger trg_education_portfolio_reviews_hold before insert or update or delete on education_portfolio_reviews
	for each row execute function public.enforce_education_portfolio_related_hold();

alter table education_portfolio_transfers
	add column if not exists withdrawn_at timestamptz,
	add column if not exists withdrawn_by_subject text not null default '',
	add column if not exists withdrawal_reason text not null default '';
alter table education_portfolio_valorifications
	add column if not exists withdrawn_at timestamptz,
	add column if not exists withdrawn_by_subject text not null default '',
	add column if not exists withdrawal_reason text not null default '';

create or replace function public.enforce_education_portfolio_lifecycle_child()
returns trigger
language plpgsql
as $$
declare
	portfolio_ref uuid;
	is_withdrawn timestamptz;
	is_held boolean;
begin
	if tg_op = 'DELETE' then
		raise exception 'portfolio lifecycle records cannot be hard-deleted';
	end if;
	portfolio_ref := case when tg_op = 'UPDATE' then old.portfolio_id else new.portfolio_id end;
	select withdrawn_at, legal_hold_active into is_withdrawn, is_held
	from education_portfolios where id = portfolio_ref for key share;
	if not found then
		raise exception 'portfolio does not exist';
	end if;
	if is_withdrawn is not null then
		raise exception 'withdrawn portfolio lifecycle records are immutable';
	end if;
	if is_held then
		raise exception 'portfolio lifecycle record mutation is blocked by legal hold';
	end if;
	if tg_op = 'UPDATE' and old.withdrawn_at is not null then
		raise exception 'withdrawn portfolio lifecycle record is immutable';
	end if;
	return new;
end;
$$;

drop trigger if exists trg_education_portfolio_transfer_lifecycle on education_portfolio_transfers;
create trigger trg_education_portfolio_transfer_lifecycle
	before insert or update or delete on education_portfolio_transfers
	for each row execute function public.enforce_education_portfolio_lifecycle_child();
drop trigger if exists trg_education_portfolio_valorification_lifecycle on education_portfolio_valorifications;
create trigger trg_education_portfolio_valorification_lifecycle
	before insert or update or delete on education_portfolio_valorifications
	for each row execute function public.enforce_education_portfolio_lifecycle_child();

create index if not exists idx_education_portfolios_lifecycle
	on education_portfolios (institution_id, activity_ceased_on, legal_hold_active, withdrawn_at);
create index if not exists idx_education_portfolio_transfers_active
	on education_portfolio_transfers (portfolio_id, institution_id, handover_on desc)
	where withdrawn_at is null;
create index if not exists idx_education_portfolio_valorifications_active
	on education_portfolio_valorifications (portfolio_id, institution_id, started_on desc)
	where withdrawn_at is null;
