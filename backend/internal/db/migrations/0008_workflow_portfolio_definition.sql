insert into workflow_definitions(code, name, category, initial_step, sla_hours, active)
values
	('portfolio-review', 'Revizuire portofoliu CD', 'education.portfolios', 'Verificare portofoliu', 120, true)
on conflict (code) do update
set name = excluded.name,
	category = excluded.category,
	initial_step = excluded.initial_step,
	sla_hours = excluded.sla_hours,
	active = excluded.active;
