-- Art. 11(3): three calendar years from cessation, never a fixed-day proxy.
create or replace function public.enforce_education_portfolio_lifecycle()
returns trigger language plpgsql as $$
begin
 if tg_op='DELETE' then raise exception 'education portfolios are evidentiary records and cannot be hard-deleted'; end if;
 if tg_op='UPDATE' then
  if old.withdrawn_at is not null then raise exception 'withdrawn education portfolios are immutable'; end if;
  if old.activity_ceased_on is not null and new.activity_ceased_on is distinct from old.activity_ceased_on then raise exception 'portfolio cessation event is immutable'; end if;
  if new.withdrawn_at is not null and (old.status not in ('draft','returned') or old.retention_until is not null or old.legal_hold_active) then raise exception 'portfolio withdrawal is blocked after submission, during retention, or under legal hold'; end if;
 end if;
 if new.activity_ceased_on is null then
  if tg_op='INSERT' then new.retention_until:=null; elsif new.retention_until is distinct from old.retention_until then new.retention_until:=null; end if;
 else
  new.retention_until:=greatest((new.activity_ceased_on + interval '3 years')::date, coalesce(old.retention_until, new.activity_ceased_on));
 end if;
 return new;
end $$;
