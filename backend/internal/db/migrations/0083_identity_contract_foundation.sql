-- Identity is global; authorization is always tenant-bound. This migration is
-- intentionally additive and may be applied repeatedly only through the
-- migration ledger. The three named operators below were explicitly supplied
-- by the tenant owner for the Balotești production tenant.
select set_config('app.is_super_admin', 'true', true);

-- app_users remains the profile directory. Login identifiers are moved into a
-- dedicated table so a user may be phone-only and the same profile can later
-- hold independently verified phone, email and passkey credentials.
alter table app_users drop constraint if exists app_users_email_key;

create unique index if not exists uq_app_users_normalized_nonempty_email
	 on app_users ((lower(btrim(email))))
	 where btrim(email) <> '';

create table if not exists app_user_identities (
	id uuid primary key default gen_random_uuid(),
	user_id uuid not null references app_users(id) on delete cascade,
	identity_type text not null check (identity_type in ('phone', 'email')),
	normalized_value text not null check (btrim(normalized_value) <> ''),
	display_value text not null check (btrim(display_value) <> ''),
	verified_at timestamptz,
	is_primary boolean not null default false,
	created_at timestamptz not null default now(),
	updated_at timestamptz not null default now(),
	unique (identity_type, normalized_value)
);

create unique index if not exists uq_app_user_identities_primary_by_type
	on app_user_identities (user_id, identity_type)
	where is_primary;

create index if not exists idx_app_user_identities_user
	on app_user_identities (user_id, identity_type);

-- Existing profile values are retained as aliases. New writes must populate
-- normalized_value before authentication can use this table as the resolver.
insert into app_user_identities (
	user_id, identity_type, normalized_value, display_value, verified_at, is_primary
)
select
	u.id,
	'email',
	lower(btrim(u.email)),
	btrim(u.email),
	case when u.email_verified then now() else null end,
	true
from app_users u
where btrim(u.email) <> ''
on conflict (identity_type, normalized_value) do nothing;

insert into app_user_identities (
	user_id, identity_type, normalized_value, display_value, verified_at, is_primary
)
select
	u.id,
	'phone',
	case
		when btrim(u.phone_number) like '+%' then btrim(u.phone_number)
		when regexp_replace(btrim(u.phone_number), '[^0-9]', '', 'g') like '07%' then
			'+40' || substr(regexp_replace(btrim(u.phone_number), '[^0-9]', '', 'g'), 2)
		else regexp_replace(btrim(u.phone_number), '[^0-9+]', '', 'g')
	end,
	btrim(u.phone_number),
	case when u.phone_number_verified then now() else null end,
	true
from app_users u
where btrim(u.phone_number) <> ''
on conflict (identity_type, normalized_value) do nothing;

-- Platform authority is deliberately separate from tenant roles. It is never
-- inferred from a tenant role such as admin or super_admin.
create table if not exists app_platform_roles (
	code text primary key,
	label text not null,
	created_at timestamptz not null default now(),
	updated_at timestamptz not null default now()
);

create table if not exists app_user_platform_roles (
	user_id uuid not null references app_users(id) on delete cascade,
	role_code text not null references app_platform_roles(code) on delete cascade,
	granted_at timestamptz not null default now(),
	primary key (user_id, role_code)
);

insert into app_platform_roles (code, label)
values ('platform_super_admin', 'Platform Super Administrator')
on conflict (code) do update
set label = excluded.label,
	updated_at = now();

-- Version values are the database authority for tenant-scoped token/session
-- invalidation. The issuing service will later place this value in its token.
create table if not exists app_tenant_authorization_versions (
	tenant_code text not null references app_tenants(code) on delete cascade,
	user_id uuid not null references app_users(id) on delete cascade,
	version bigint not null default 1 check (version > 0),
	updated_at timestamptz not null default now(),
	primary key (tenant_code, user_id)
);

alter table app_tenant_authorization_versions enable row level security;
alter table app_tenant_authorization_versions force row level security;
drop policy if exists tenant_isolation on app_tenant_authorization_versions;
create policy tenant_isolation on app_tenant_authorization_versions
	using (tenant_code = public.current_tenant_code())
	with check (tenant_code = public.current_tenant_code());

create or replace function public.bump_tenant_authorization_version(
	p_tenant_code text,
	p_user_id uuid
)
returns void
language plpgsql
as $$
begin
	if p_tenant_code is null or p_user_id is null then
		return;
	end if;

	insert into app_tenant_authorization_versions (tenant_code, user_id, version)
	values (p_tenant_code, p_user_id, 1)
	on conflict (tenant_code, user_id) do update
	set version = app_tenant_authorization_versions.version + 1,
		updated_at = now();
end;
$$;

create or replace function public.record_tenant_authorization_change()
returns trigger
language plpgsql
as $$
begin
	if tg_op = 'DELETE' then
		perform public.bump_tenant_authorization_version(old.tenant_code, old.user_id);
	else
		perform public.bump_tenant_authorization_version(new.tenant_code, new.user_id);
		if tg_op = 'UPDATE'
			and (old.tenant_code, old.user_id) is distinct from (new.tenant_code, new.user_id) then
			perform public.bump_tenant_authorization_version(old.tenant_code, old.user_id);
		end if;
	end if;
	return coalesce(new, old);
end;
$$;

drop trigger if exists trg_app_user_roles_authz_version on app_user_roles;
create trigger trg_app_user_roles_authz_version
	after insert or update or delete on app_user_roles
	for each row execute function public.record_tenant_authorization_change();

drop trigger if exists trg_app_user_permissions_authz_version on app_user_permissions;
create trigger trg_app_user_permissions_authz_version
	after insert or update or delete on app_user_permissions
	for each row execute function public.record_tenant_authorization_change();

drop trigger if exists trg_app_user_modules_authz_version on app_user_modules;
create trigger trg_app_user_modules_authz_version
	after insert or update or delete on app_user_modules
	for each row execute function public.record_tenant_authorization_change();

drop trigger if exists trg_app_memberships_authz_version on app_memberships;
create trigger trg_app_memberships_authz_version
	after insert or update or delete on app_memberships
	for each row execute function public.record_tenant_authorization_change();

create or replace function public.record_platform_authorization_change()
returns trigger
language plpgsql
as $$
declare
	changed_user_id uuid;
	membership_row record;
	original_tenant_code text;
begin
	changed_user_id := case when tg_op = 'DELETE' then old.user_id else new.user_id end;
	original_tenant_code := current_setting('app.tenant_id', true);
	for membership_row in
		select distinct tenant_code
		from app_memberships
		where user_id = changed_user_id and active
	loop
		-- The authorization-version table is FORCE RLS. Switch to the exact
		-- affected tenant for each write and restore the caller context below.
		perform set_config('app.tenant_id', membership_row.tenant_code, true);
		perform public.bump_tenant_authorization_version(membership_row.tenant_code, changed_user_id);
	end loop;
	perform set_config('app.tenant_id', coalesce(original_tenant_code, ''), true);
	return coalesce(new, old);
exception when others then
	perform set_config('app.tenant_id', coalesce(original_tenant_code, ''), true);
	raise;
end;
$$;

drop trigger if exists trg_app_user_platform_roles_authz_version on app_user_platform_roles;
create trigger trg_app_user_platform_roles_authz_version
	after insert or update or delete on app_user_platform_roles
	for each row execute function public.record_platform_authorization_change();

-- The preflight rejects a collision rather than attaching a real person's
-- login identifier to the wrong historical account.
do $$
begin
	if exists (select 1 from app_users where lower(btrim(email)) = 'thomas@eguilde.cloud' and sub <> 'user-thomas-galambos') then
		raise exception '0083 identity preflight failed: Thomas email belongs to a different subject';
	end if;
	if exists (select 1 from app_users where phone_number in ('+40771364169', '+40744652476', '+40735091230') and sub not in ('user-thomas-galambos', 'user-stelian-fedorca', 'user-diana-ilhan')) then
		raise exception '0083 identity preflight failed: a Balotesti phone belongs to a different subject';
	end if;
end;
$$;

-- The version table deliberately has no super-admin bypass. Bind the migration
-- itself to the target tenant before authorization triggers begin writing it.
select set_config('app.tenant_id', 'tenant-balotesti', true);

insert into app_users (
	sub, name, email, phone_number, locale, status,
	email_verified, phone_number_verified, preferred_otp_channel
)
values
	('user-thomas-galambos', 'Thomas Galambos', 'thomas@eguilde.cloud', '+40771364169', 'ro', 'active', true, true, 'sms'),
	('user-stelian-fedorca', 'Stelian Fedorca', '', '+40744652476', 'ro', 'active', false, true, 'sms'),
	('user-diana-ilhan', 'Diana Ilhan', '', '+40735091230', 'ro', 'active', false, true, 'sms')
on conflict (sub) do update
set name = excluded.name,
	email = excluded.email,
	phone_number = excluded.phone_number,
	locale = excluded.locale,
	status = excluded.status,
	email_verified = excluded.email_verified,
	phone_number_verified = excluded.phone_number_verified,
	preferred_otp_channel = excluded.preferred_otp_channel,
	updated_at = now();

insert into app_user_identities (
	user_id, identity_type, normalized_value, display_value, verified_at, is_primary
)
select u.id, expected.identity_type, expected.normalized_value, expected.display_value, now(), true
from app_users u
join (
	values
		('user-thomas-galambos', 'email', 'thomas@eguilde.cloud', 'thomas@eguilde.cloud'),
		('user-thomas-galambos', 'phone', '+40771364169', '+40771364169'),
		('user-stelian-fedorca', 'phone', '+40744652476', '+40744652476'),
		('user-diana-ilhan', 'phone', '+40735091230', '+40735091230')
) as expected(subject, identity_type, normalized_value, display_value)
	on expected.subject = u.sub
on conflict (identity_type, normalized_value) do update
set display_value = excluded.display_value,
	verified_at = coalesce(app_user_identities.verified_at, excluded.verified_at),
	updated_at = now();

do $$
begin
	if exists (
		select 1
		from (values
			('user-thomas-galambos', 'email', 'thomas@eguilde.cloud'),
			('user-thomas-galambos', 'phone', '+40771364169'),
			('user-stelian-fedorca', 'phone', '+40744652476'),
			('user-diana-ilhan', 'phone', '+40735091230')
		) as expected(subject, identity_type, normalized_value)
		join app_users u on u.sub = expected.subject
		left join app_user_identities i
			on i.user_id = u.id
			and i.identity_type = expected.identity_type
			and i.normalized_value = expected.normalized_value
		where i.id is null
	) then
		raise exception '0083 identity provision failed: a requested identity is already bound to another subject';
	end if;
end;
$$;

insert into app_user_platform_roles (user_id, role_code)
select id, 'platform_super_admin'
from app_users
where sub = 'user-thomas-galambos'
on conflict do nothing;

insert into app_memberships (
	user_id, tenant_code, position_code, org_unit_code, organization_name,
	is_primary, active, start_date
)
select u.id, 'tenant-balotesti', 'director', 'unit-balotesti-root',
	'Școala Gimnazială nr. 1 Balotești', true, true, current_date
from app_users u
where u.sub in ('user-thomas-galambos', 'user-stelian-fedorca', 'user-diana-ilhan')
	and not exists (
		select 1 from app_memberships m
		where m.user_id = u.id
			and m.tenant_code = 'tenant-balotesti'
			and m.position_code = 'director'
			and m.org_unit_code = 'unit-balotesti-root'
	);

insert into app_user_roles (tenant_code, user_id, role_code)
select 'tenant-balotesti', u.id, 'admin'
from app_users u
where u.sub in ('user-thomas-galambos', 'user-stelian-fedorca', 'user-diana-ilhan')
on conflict do nothing;

insert into app_user_permissions (tenant_code, user_id, permission_code)
select 'tenant-balotesti', u.id, p.code
from app_users u
cross join app_permissions p
where u.sub in ('user-thomas-galambos', 'user-stelian-fedorca', 'user-diana-ilhan')
on conflict do nothing;

insert into app_user_modules (tenant_code, user_id, module_code)
select 'tenant-balotesti', u.id, m.code
from app_users u
cross join app_modules m
where m.active
	and u.sub in ('user-thomas-galambos', 'user-stelian-fedorca', 'user-diana-ilhan')
on conflict do nothing;

insert into app_session_context (
	user_id, institution_id, institution_name, auth_methods, gdpr_capabilities
)
select u.id, 'inst-balotesti', 'Școala Gimnazială nr. 1 Balotești',
	array['oidc_redirect', 'sms_otp', 'passkey'],
	array['retention_policies', 'subject_export', 'purpose_limited_access', 'publication_anonymization']
from app_users u
where u.sub in ('user-thomas-galambos', 'user-stelian-fedorca', 'user-diana-ilhan')
on conflict (user_id) do update
set institution_id = excluded.institution_id,
	institution_name = excluded.institution_name,
	auth_methods = excluded.auth_methods,
	gdpr_capabilities = excluded.gdpr_capabilities;

-- Ensure a version exists even where a previous migration inserted all grants
-- before these triggers were installed. FORCE RLS requires each tenant's rows
-- to be inserted under that tenant's own context.
do $$
declare
	tenant_row record;
	original_tenant_code text;
begin
	original_tenant_code := current_setting('app.tenant_id', true);
	for tenant_row in
		select distinct membership.tenant_code
		from app_memberships membership
		where membership.active
	loop
		perform set_config('app.tenant_id', tenant_row.tenant_code, true);
		insert into app_tenant_authorization_versions (tenant_code, user_id, version)
		select distinct membership.tenant_code, membership.user_id, 1
		from app_memberships membership
		where membership.active
			and membership.tenant_code = tenant_row.tenant_code
		on conflict do nothing;
	end loop;
	perform set_config('app.tenant_id', coalesce(original_tenant_code, ''), true);
exception when others then
	perform set_config('app.tenant_id', coalesce(original_tenant_code, ''), true);
	raise;
end;
$$;

do $$
begin
	if not exists (select 1 from app_tenants where code = 'tenant-balotesti' and institution_id = 'inst-balotesti' and active) then
		raise exception '0083 tenant preflight failed: tenant-balotesti/inst-balotesti is missing or inactive';
	end if;
	if exists (
		select 1
		from app_users u
		where u.sub in ('user-thomas-galambos', 'user-stelian-fedorca', 'user-diana-ilhan')
		and not exists (
			select 1 from app_memberships m
			where m.user_id = u.id and m.tenant_code = 'tenant-balotesti' and m.active
		)
	) then
		raise exception '0083 tenant provision failed: an operator has no active Balotesti membership';
	end if;
end;
$$;
