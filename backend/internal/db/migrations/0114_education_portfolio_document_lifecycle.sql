-- Portfolio evidence is retained as a legal/audit record.  A user-facing
-- DELETE is a withdrawal transition, never a physical delete.  Operational
-- projections consume only active evidence; entity versions retain the
-- withdrawn row and its server-derived provenance.
alter table education_portfolio_documents
	add column if not exists status text not null default 'active',
	add column if not exists withdrawn_at timestamptz,
	add column if not exists withdrawn_by_subject text not null default '',
	add column if not exists withdrawal_reason text not null default '';

update education_portfolio_documents
set status = 'active'
where status is null or status = '';

alter table education_portfolio_documents
	drop constraint if exists education_portfolio_documents_status_check;
alter table education_portfolio_documents
	add constraint education_portfolio_documents_status_check
	check (status in ('active', 'withdrawn'));
alter table education_portfolio_documents
	drop constraint if exists education_portfolio_documents_withdrawal_provenance_check;
alter table education_portfolio_documents
	add constraint education_portfolio_documents_withdrawal_provenance_check
	check (
		(status = 'active' and withdrawn_at is null and withdrawn_by_subject = '' and withdrawal_reason = '')
		or
		(status = 'withdrawn' and withdrawn_at is not null and btrim(withdrawn_by_subject) <> '' and btrim(withdrawal_reason) <> '')
	);

create index if not exists idx_education_portfolio_documents_operational
	on education_portfolio_documents (portfolio_id, institution_id, status, chronological_index, issued_on);

-- This intentionally replaces the function introduced by 0098 while keeping
-- its trigger and parent-row serialization.  It strengthens, rather than
-- bypasses, the existing editable-parent rule.
create or replace function public.enforce_education_portfolio_document_state()
returns trigger language plpgsql as $$
declare
	portfolio_status text;
	portfolio_withdrawn_at timestamptz;
	portfolio_legal_hold boolean;
	portfolio_ref uuid;
	actor_subject text := nullif(btrim(current_setting('app.actor_subject', true)), '');
	session_can_bypass boolean := false;
begin
	portfolio_ref := case when tg_op = 'DELETE' then old.portfolio_id else new.portfolio_id end;
	select status, withdrawn_at, legal_hold_active
	into portfolio_status, portfolio_withdrawn_at, portfolio_legal_hold
	from education_portfolios where id = portfolio_ref for update;
	if portfolio_status is null then
		raise exception 'portfolio does not exist';
	end if;
	if portfolio_status not in ('draft', 'returned') or portfolio_withdrawn_at is not null or coalesce(portfolio_legal_hold, false) then
		raise exception 'portfolio evidence is immutable outside editable portfolio state';
	end if;

	if tg_op = 'DELETE' then
		raise exception 'portfolio evidence hard-delete is forbidden; withdraw the evidence instead';
	end if;

	select coalesce(role_row.rolsuper or role_row.rolbypassrls, false)
	into session_can_bypass
	from pg_catalog.pg_roles role_row
	where role_row.rolname = session_user;

	if tg_op = 'INSERT' then
		if new.status <> 'active' or new.withdrawn_at is not null or new.withdrawn_by_subject <> '' or new.withdrawal_reason <> '' then
			raise exception 'portfolio evidence must be created active without withdrawal provenance';
		end if;
		return new;
	end if;

	if old.status = 'withdrawn' then
		raise exception 'withdrawn portfolio evidence is immutable';
	end if;
	if new.portfolio_id <> old.portfolio_id or new.institution_id <> old.institution_id then
		raise exception 'portfolio evidence cannot be reassigned';
	end if;

	if new.status = 'active' then
		if new.withdrawn_at is not null or new.withdrawn_by_subject <> '' or new.withdrawal_reason <> '' then
			raise exception 'active portfolio evidence cannot carry withdrawal provenance';
		end if;
		return new;
	end if;

	if new.status <> 'withdrawn' then
		raise exception 'invalid portfolio evidence lifecycle status';
	end if;
	if (to_jsonb(new) - array['status', 'withdrawn_at', 'withdrawn_by_subject', 'withdrawal_reason', 'updated_at'])
		is distinct from (to_jsonb(old) - array['status', 'withdrawn_at', 'withdrawn_by_subject', 'withdrawal_reason', 'updated_at']) then
		raise exception 'portfolio evidence withdrawal cannot alter evidence content';
	end if;
	if actor_subject is null and not session_can_bypass then
		raise exception 'portfolio evidence withdrawal requires authenticated actor provenance';
	end if;
	new.withdrawn_at := now();
	new.withdrawn_by_subject := coalesce(actor_subject, 'migration-or-privileged-session');
	new.withdrawal_reason := 'withdrawn_by_authorized_user';
	return new;
end;
$$;

-- 0098 owns this trigger name.  It remains attached to the same table and now
-- records the strengthened append-audit lifecycle above.
drop trigger if exists trg_education_portfolio_document_state on education_portfolio_documents;
create trigger trg_education_portfolio_document_state
	before insert or update or delete on education_portfolio_documents
	for each row execute function public.enforce_education_portfolio_document_state();
