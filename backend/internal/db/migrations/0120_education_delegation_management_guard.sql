-- Delegations may grant operational School permissions only. Administration of
-- delegations remains direct RBAC, otherwise a delegate could recursively
-- manufacture or broaden authority. Fail fast rather than silently changing
-- any existing evidentiary row: an operator must remediate invalid history.
do $$
begin
	if exists (
		select 1
		from public.education_role_delegations
		where permission_code like 'education.delegations.%'
	) then
		raise exception 'migration 0120 refused: education_role_delegations contains delegation-management permissions; remediate rows explicitly before retrying';
	end if;
end;
$$;

alter table public.education_role_delegations
	add constraint education_role_delegations_no_management_permission
	check (permission_code not like 'education.delegations.%');
