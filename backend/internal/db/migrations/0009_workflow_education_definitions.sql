insert into workflow_definitions(code, name, category, initial_step, sla_hours, active)
values
	('governance-meeting-review', 'Revizuire ședință de guvernanță', 'education.governance', 'Pregătire dosar ședință', 96, true),
	('personnel-evaluation', 'Evaluare personal școlar', 'education.personnel', 'Deschidere evaluare', 120, true),
	('personnel-mobility', 'Mobilitate personal școlar', 'education.personnel', 'Analiză mobilitate', 120, true)
on conflict (code) do update
set name = excluded.name,
	category = excluded.category,
	initial_step = excluded.initial_step,
	sla_hours = excluded.sla_hours,
	active = excluded.active;
