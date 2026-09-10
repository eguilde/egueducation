-- Tenant administrator roles receive the complete tenant catalog, but the
-- School director role is intentionally least-privilege. New education
-- permissions must be reviewed and granted explicitly instead of becoming
-- director capabilities merely because of their namespace.
create or replace function public.apply_complete_tenant_role_mapping()
returns trigger
language plpgsql
as $$
begin
	insert into app_role_permissions (role_code, permission_code)
	select role.code, new.code
	from app_roles role
	where role.code in ('admin', 'super_admin', 'e2e_canary')
	on conflict do nothing;

	return new;
end;
$$;

-- 0125 temporarily granted every education.* permission to director. Remove
-- only permissions outside the reviewed director catalog. The explicit list
-- below is the union of the deliberate director grants from earlier schema
-- migrations.
with reviewed(permission_code) as (
	values
		('education.read'),
		('education.compliance.read'),
		('education.compliance.manage'),
		('education.decisions.issuance.read'),
		('education.decisions.issuance.manage'),
		('education.personnel.files.read'),
		('education.personnel.files.manage'),
		('education.personnel.access.read'),
		('education.personnel.access.manage'),
		('education.governance.meeting.close'),
		('education.governance.meeting.vote'),
		('education.governance.minutes.publish'),
		('education.governance.resolution.publish'),
		('education.portfolios.verify'),
		('education.portfolios.transfer'),
		('education.portfolios.custody.manage'),
		('education.managerial.publish'),
		('education.reports.export_sensitive'),
		('education.inspect.package.read'),
		('education.portfolios.read_own'),
		('education.portfolios.manage_own'),
		('education.portfolios.school.read'),
		('education.portfolios.school.manage'),
		('education.portfolios.request_corrections'),
		('education.portfolios.archive_grants.manage'),
		('education.portfolios.transfer.receive'),
		('education.delegations.read'),
		('education.delegations.offer'),
		('education.delegations.revoke'),
		('education.classes.read'),
		('education.classes.manage'),
		('education.signatures.read'),
		('education.signatures.manage'),
		('education.signatures.validate'),
		('education.cockpit.secretariat.read'),
		('education.cockpit.hr.read'),
		('education.cockpit.committee.read'),
		('education.cockpit.inspector.read')
)
delete from app_role_permissions role_permission
where role_permission.role_code = 'director'
	and role_permission.permission_code like 'education.%'
	and not exists (
		select 1 from reviewed
		where reviewed.permission_code = role_permission.permission_code
	);

-- A school director is a professional position, not a tenant administrator.
-- The legacy mapping to the all-powerful admin role would otherwise make every
-- future permission granted to admin effective for every director as well.
delete from app_position_roles
where position_code = 'director' and role_code = 'admin';

insert into app_position_roles (position_code, role_code)
values ('director', 'director')
on conflict do nothing;
