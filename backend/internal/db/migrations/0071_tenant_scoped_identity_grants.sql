-- Identity is global, but every authorization and module grant belongs to one tenant.
-- This migration intentionally fails closed.  It must never turn an RLS visibility
-- issue or an incomplete tenant bootstrap into a deployment that silently strips
-- every administrator's grants.
select set_config('app.is_super_admin', 'true', true);
do $$
begin
	if not exists (select 1 from app_tenants where active) then
		raise exception '0071 preflight failed: no active tenants are available';
	end if;
	if not exists (select 1 from app_memberships where active) then
		raise exception '0071 preflight failed: no active tenant memberships are available';
	end if;
end $$;

alter table app_user_roles add column if not exists tenant_code text references app_tenants(code) on delete cascade;
alter table app_user_permissions add column if not exists tenant_code text references app_tenants(code) on delete cascade;
alter table app_user_modules add column if not exists tenant_code text references app_tenants(code) on delete cascade;

alter table app_user_roles drop constraint if exists app_user_roles_pkey;
alter table app_user_permissions drop constraint if exists app_user_permissions_pkey;
alter table app_user_modules drop constraint if exists app_user_modules_pkey;

-- Preserve existing grants only in the user's established home tenant. Replicating
-- a formerly global grant to every membership would preserve the vulnerability.
insert into app_user_roles (user_id, role_code, tenant_code)
select distinct ur.user_id, ur.role_code, t.code
from app_user_roles ur
join app_session_context sc on sc.user_id = ur.user_id
join app_tenants t on t.institution_id = sc.institution_id
join app_memberships m on m.user_id = ur.user_id and m.tenant_code = t.code and m.active = true
where ur.tenant_code is null
on conflict do nothing;

-- A user with no session context can be migrated only when exactly one active
-- membership makes the destination unambiguous.  Never replicate grants.
insert into app_user_roles(user_id,role_code,tenant_code)
select ur.user_id,ur.role_code,min(m.tenant_code)
from app_user_roles ur join app_memberships m on m.user_id=ur.user_id and m.active
where ur.tenant_code is null and not exists(select 1 from app_session_context sc join app_tenants t on t.institution_id=sc.institution_id and t.active join app_memberships sm on sm.user_id=ur.user_id and sm.tenant_code=t.code and sm.active where sc.user_id=ur.user_id)
group by ur.user_id,ur.role_code having count(distinct m.tenant_code)=1;

insert into app_user_permissions (user_id, permission_code, tenant_code)
select distinct up.user_id, up.permission_code, t.code
from app_user_permissions up
join app_session_context sc on sc.user_id = up.user_id
join app_tenants t on t.institution_id = sc.institution_id
join app_memberships m on m.user_id = up.user_id and m.tenant_code = t.code and m.active = true
where up.tenant_code is null
on conflict do nothing;
insert into app_user_permissions(user_id,permission_code,tenant_code)
select up.user_id,up.permission_code,min(m.tenant_code)
from app_user_permissions up join app_memberships m on m.user_id=up.user_id and m.active
where up.tenant_code is null and not exists(select 1 from app_session_context sc join app_tenants t on t.institution_id=sc.institution_id and t.active join app_memberships sm on sm.user_id=up.user_id and sm.tenant_code=t.code and sm.active where sc.user_id=up.user_id)
group by up.user_id,up.permission_code having count(distinct m.tenant_code)=1;

insert into app_user_modules (user_id, module_code, tenant_code)
select distinct um.user_id, um.module_code, t.code
from app_user_modules um
join app_session_context sc on sc.user_id = um.user_id
join app_tenants t on t.institution_id = sc.institution_id
join app_memberships m on m.user_id = um.user_id and m.tenant_code = t.code and m.active = true
where um.tenant_code is null
on conflict do nothing;
insert into app_user_modules(user_id,module_code,tenant_code)
select um.user_id,um.module_code,min(m.tenant_code)
from app_user_modules um join app_memberships m on m.user_id=um.user_id and m.active
where um.tenant_code is null and not exists(select 1 from app_session_context sc join app_tenants t on t.institution_id=sc.institution_id and t.active join app_memberships sm on sm.user_id=um.user_id and sm.tenant_code=t.code and sm.active where sc.user_id=um.user_id)
group by um.user_id,um.module_code having count(distinct m.tenant_code)=1;

do $$
begin
	if exists (select 1 from app_user_roles where tenant_code is null and role_code in ('super_admin','admin','workflow_admin'))
		and not exists (select 1 from app_user_roles where tenant_code is not null and role_code in ('super_admin','admin','workflow_admin')) then
		raise exception '0071 preflight failed: privileged role coverage would become zero';
	end if;
	if exists (select 1 from app_user_permissions where tenant_code is null and permission_code in ('admin.users.manage','registratura.manage'))
		and not exists (select 1 from app_user_permissions where tenant_code is not null and permission_code in ('admin.users.manage','registratura.manage')) then
		raise exception '0071 preflight failed: administrative permission coverage would become zero';
	end if;
end $$;

-- A grant without an active tenant membership must not remain authoritative.
create table if not exists app_unresolved_legacy_grants (
 id uuid primary key default gen_random_uuid(), user_id uuid not null, grant_kind text not null,
 grant_code text not null, payload jsonb not null default '{}'::jsonb,
 reason text not null, migrated_at timestamptz not null default now(),
 unique(user_id,grant_kind,grant_code)
);
insert into app_unresolved_legacy_grants(user_id,grant_kind,grant_code,payload,reason)
select user_id,'role',role_code,jsonb_build_object('role_code',role_code),'no unambiguous active tenant membership' from app_user_roles where tenant_code is null
on conflict(user_id,grant_kind,grant_code) do nothing;
insert into app_unresolved_legacy_grants(user_id,grant_kind,grant_code,payload,reason)
select user_id,'permission',permission_code,jsonb_build_object('permission_code',permission_code),'no unambiguous active tenant membership' from app_user_permissions where tenant_code is null
on conflict(user_id,grant_kind,grant_code) do nothing;
insert into app_unresolved_legacy_grants(user_id,grant_kind,grant_code,payload,reason)
select user_id,'module',module_code,jsonb_build_object('module_code',module_code),'no unambiguous active tenant membership' from app_user_modules where tenant_code is null
on conflict(user_id,grant_kind,grant_code) do nothing;
do $$ begin
 if (select count(*) from app_user_roles where tenant_code is null) <> (select count(*) from app_unresolved_legacy_grants where grant_kind='role') then raise exception '0071 role quarantine verification failed'; end if;
 if (select count(*) from app_user_permissions where tenant_code is null) <> (select count(*) from app_unresolved_legacy_grants where grant_kind='permission') then raise exception '0071 permission quarantine verification failed'; end if;
 if (select count(*) from app_user_modules where tenant_code is null) <> (select count(*) from app_unresolved_legacy_grants where grant_kind='module') then raise exception '0071 module quarantine verification failed'; end if;
end $$;
delete from app_user_roles where tenant_code is null;
delete from app_user_permissions where tenant_code is null;
delete from app_user_modules where tenant_code is null;

alter table app_user_roles alter column tenant_code set not null;
alter table app_user_permissions alter column tenant_code set not null;
alter table app_user_modules alter column tenant_code set not null;

alter table app_user_roles add primary key (tenant_code, user_id, role_code);
alter table app_user_permissions add primary key (tenant_code, user_id, permission_code);
alter table app_user_modules add primary key (tenant_code, user_id, module_code);

create index if not exists idx_app_user_roles_user_tenant on app_user_roles (user_id, tenant_code);
create index if not exists idx_app_user_permissions_user_tenant on app_user_permissions (user_id, tenant_code);
create index if not exists idx_app_user_modules_user_tenant on app_user_modules (user_id, tenant_code);
