insert into app_permissions(code, label) values
	('education.mobility.read', 'Read education mobility'),
	('education.gradatii.read', 'Read education merit grants'),
	('gdpr.policies.read', 'Read GDPR retention policies'),
	('gdpr.requests.read', 'Read GDPR subject requests')
on conflict (code) do nothing;

insert into app_user_permissions(user_id, permission_code)
select id, permission_code
from app_users
cross join (values
	('education.mobility.read'),
	('education.gradatii.read'),
	('gdpr.policies.read'),
	('gdpr.requests.read')
) as permissions(permission_code)
where sub = 'usr-001'
on conflict do nothing;
