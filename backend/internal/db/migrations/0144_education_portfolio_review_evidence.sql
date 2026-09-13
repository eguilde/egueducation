-- Portfolio review decisions are immutable evidence. Managerial rejection is
-- a terminal, explicit state and must never be represented as a correction.
select set_config('app.is_super_admin', 'true', true);

alter table education_portfolios drop constraint if exists education_portfolios_status_check;
alter table education_portfolios add constraint education_portfolios_status_check
 check (status in ('draft', 'submitted', 'returned', 'validated', 'rejected', 'transferred', 'archived', 'withdrawn'));

insert into education_taxonomies(domain, code, label_ro, label_en, active, sort_order)
values ('portfolio_status', 'rejected', 'Respins', 'Rejected', true, 35)
on conflict (domain, code) do update
set label_ro=excluded.label_ro, label_en=excluded.label_en, active=true, sort_order=excluded.sort_order;

create or replace function public.education_portfolio_review_immutable()
returns trigger
language plpgsql
as $$
begin
 raise exception 'portfolio review evidence is immutable';
end
$$;

create trigger education_portfolio_reviews_immutable
before update or delete on education_portfolio_reviews
for each row execute function public.education_portfolio_review_immutable();
