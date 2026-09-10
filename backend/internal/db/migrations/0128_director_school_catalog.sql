-- Grant the explicitly approved, tenant-scoped School catalog to directors.
-- This is intentionally a new migration: 0127 may already be recorded in an
-- environment and modifying it would not reconcile previously migrated data.
insert into app_role_permissions (role_code, permission_code)
values
	('director', 'education.governance.read'),
	('director', 'education.governance.manage'),
	('director', 'education.portfolios.read'),
	('director', 'education.portfolios.manage'),
	('director', 'education.personnel.read'),
	('director', 'education.personnel.manage'),
	('director', 'education.mobility.read'),
	('director', 'education.mobility.manage'),
	('director', 'education.gradatii.read'),
	('director', 'education.gradatii.manage'),
	('director', 'education.decisions.read'),
	('director', 'education.decisions.manage'),
	('director', 'education.managerial.read'),
	('director', 'education.managerial.manage'),
	('director', 'education.regulations.read'),
	('director', 'education.regulations.manage'),
	('director', 'education.evaluations.read'),
	('director', 'education.evaluations.manage'),
	('director', 'education.declarations.read'),
	('director', 'education.declarations.manage')
on conflict do nothing;
