insert into app_roles(code, label) values
	('hr_manager', 'Responsabil resurse umane'),
	('committee_responsible', 'Responsabil comisie')
on conflict (code) do update set label = excluded.label;

insert into app_permissions(code, label) values
	('education.cockpit.secretariat.read', 'Read secretariat operational cockpit'),
	('education.cockpit.hr.read', 'Read human resources operational cockpit'),
	('education.cockpit.committee.read', 'Read committee operational cockpit'),
	('education.cockpit.inspector.read', 'Read inspector operational cockpit')
on conflict (code) do update set label = excluded.label;

insert into app_role_permissions(role_code, permission_code)
select role_code, permission_code from (values
	('super_admin','education.cockpit.secretariat.read'), ('admin','education.cockpit.secretariat.read'), ('director','education.cockpit.secretariat.read'), ('secretar','education.cockpit.secretariat.read'),
	('super_admin','education.cockpit.hr.read'), ('admin','education.cockpit.hr.read'), ('director','education.cockpit.hr.read'), ('hr_manager','education.cockpit.hr.read'),
	('super_admin','education.cockpit.committee.read'), ('admin','education.cockpit.committee.read'), ('director','education.cockpit.committee.read'), ('committee_responsible','education.cockpit.committee.read'),
	('super_admin','education.cockpit.inspector.read'), ('admin','education.cockpit.inspector.read'), ('director','education.cockpit.inspector.read'), ('inspector','education.cockpit.inspector.read')
) as mapping(role_code, permission_code)
on conflict do nothing;
