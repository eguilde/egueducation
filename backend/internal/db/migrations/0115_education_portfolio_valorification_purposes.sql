-- Separate the legal purpose of portfolio evidence from the authoritative
-- source domain. A package still points to a real evaluation, mobility case or
-- merit record, while its use covers every purpose in the statutory catalog.
alter table education_portfolio_valorification_packages
	add column if not exists purpose text;

update education_portfolio_valorification_packages
set purpose = scope
where purpose is null or btrim(purpose) = '';

alter table education_portfolio_valorification_packages
	alter column purpose set not null,
	drop constraint if exists education_portfolio_valorification_packages_purpose_check,
	drop constraint if exists education_portfolio_valorification_packages_scope_purpose_check;

alter table education_portfolio_valorification_packages
	add constraint education_portfolio_valorification_packages_purpose_check check (purpose in (
		'licentiere', 'debut', 'definitivat', 'grad_ii', 'grad_i',
		'evaluare_profesionala', 'mobilitate', 'dezvoltare_profesionala',
		'inspectie_scolara', 'evaluare_externa_calitate', 'gradatie_merit',
		'distinctie_premiu'
	)),
	add constraint education_portfolio_valorification_packages_scope_purpose_check check (
		(scope = 'mobilitate' and purpose = 'mobilitate')
		or (scope = 'gradatie_merit' and purpose in ('gradatie_merit', 'distinctie_premiu'))
		or (scope = 'evaluare_profesionala' and purpose in (
			'licentiere', 'debut', 'definitivat', 'grad_ii', 'grad_i',
			'evaluare_profesionala', 'dezvoltare_profesionala',
			'inspectie_scolara', 'evaluare_externa_calitate', 'distinctie_premiu'
		))
	);

drop index if exists uq_education_portfolio_valorification_package_evaluation;
create unique index uq_education_portfolio_valorification_package_evaluation
	on education_portfolio_valorification_packages (portfolio_id, purpose, source_evaluation_id)
	where source_evaluation_id is not null;
drop index if exists uq_education_portfolio_valorification_package_mobility;
create unique index uq_education_portfolio_valorification_package_mobility
	on education_portfolio_valorification_packages (portfolio_id, purpose, source_mobility_case_id)
	where source_mobility_case_id is not null;
drop index if exists uq_education_portfolio_valorification_package_merit;
create unique index uq_education_portfolio_valorification_package_merit
	on education_portfolio_valorification_packages (portfolio_id, purpose, source_merit_grant_id)
	where source_merit_grant_id is not null;

create or replace function public.protect_education_portfolio_valorification_purpose()
returns trigger language plpgsql as $$
begin
	if new.purpose is distinct from old.purpose then
		raise exception 'portfolio valorification purpose is immutable';
	end if;
	return new;
end;
$$;
revoke all on function public.protect_education_portfolio_valorification_purpose() from public;
drop trigger if exists trg_education_portfolio_valorification_purpose on education_portfolio_valorification_packages;
create trigger trg_education_portfolio_valorification_purpose
	before update on education_portfolio_valorification_packages
	for each row execute function public.protect_education_portfolio_valorification_purpose();
