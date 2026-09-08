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

-- Production contains one historical audit fixture that accidentally reuses
-- Thomas's verified SMS number.  The tenant owner explicitly authorized this
-- narrow repair.  It is deliberately not a general de-duplication rule:
-- it can affect only this exact synthetic account, only when the immutable
-- real Thomas subject owns the same normalized number, and it preserves the
-- synthetic profile for audit traceability.
do $$
declare
	synthetic_subject constant text := 'audit.profesor.1782256488896' || '@example.com';
	canonical_phone text;
	canonical_user_id uuid;
	synthetic_user_id uuid;
	synthetic_count integer;
begin
	select u.id, case
		when regexp_replace(btrim(u.phone_number), '[^0-9+]', '', 'g') like '+%' then
			regexp_replace(btrim(u.phone_number), '[^0-9+]', '', 'g')
		when regexp_replace(btrim(u.phone_number), '[^0-9]', '', 'g') like '07%' then
			'+40' || substr(regexp_replace(btrim(u.phone_number), '[^0-9]', '', 'g'), 2)
		else regexp_replace(btrim(u.phone_number), '[^0-9+]', '', 'g')
	into canonical_user_id, canonical_phone
	from app_users u
	where u.sub = 'thomasgalambos';

	select count(*)
	into synthetic_count
	from app_users u
	where u.sub = synthetic_subject
		and lower(btrim(u.email)) = synthetic_subject
		and u.name = 'Audit Profesor 1782256488896';

	if synthetic_count > 0 then
		if canonical_phone is distinct from '+40771364169' then
			raise exception '0083 identity remediation failed: canonical thomasgalambos does not own the approved phone';
		end if;

		update app_users u
		set phone_number = '',
			phone_number_verified = false,
			updated_at = now()
		where u.sub = synthetic_subject
			and lower(btrim(u.email)) = synthetic_subject
			and u.name = 'Audit Profesor 1782256488896'
			and case
				when regexp_replace(btrim(u.phone_number), '[^0-9+]', '', 'g') like '+%' then
					regexp_replace(btrim(u.phone_number), '[^0-9+]', '', 'g')
				when regexp_replace(btrim(u.phone_number), '[^0-9]', '', 'g') like '07%' then
					'+40' || substr(regexp_replace(btrim(u.phone_number), '[^0-9]', '', 'g'), 2)
				else regexp_replace(btrim(u.phone_number), '[^0-9+]', '', 'g')
			end = '+40771364169'
		returning u.id into synthetic_user_id;

		-- app_audit_log is tenant-scoped and append-only. The event records
		-- provenance for the authorized quarantine without OTP data or secrets.
		if synthetic_user_id is not null then
			perform set_config('app.institution_id', 'inst-balotesti', true);
			insert into app_audit_log (
				institution_id, actor_subject, action, target_type, target_id, status, summary, details
			)
			values (
				'inst-balotesti',
				'migration/0083',
				'identity.phone.quarantined',
				'app_user',
				synthetic_user_id::text,
				'success',
				'Quarantined duplicate legacy phone from the approved synthetic audit profile.',
				jsonb_build_object(
					'identity_type', 'phone',
					'canonical_subject', 'thomasgalambos',
					'canonical_user_id', canonical_user_id,
					'remediation', 'approved_synthetic_duplicate_quarantine'
				)
			);
		end if;
	end if;
end;
$$;

-- Never let INSERT ... ON CONFLICT choose the owner of a login credential.
-- Legacy profile phones must already have one unambiguous owner before they
-- are promoted into the global identity directory. Any repair is a separate,
-- explicitly reviewed data decision.
do $$
begin
	if exists (
		select 1
		from (
			select
				u.id,
				case
					when regexp_replace(btrim(u.phone_number), '[^0-9+]', '', 'g') like '+%' then
						regexp_replace(btrim(u.phone_number), '[^0-9+]', '', 'g')
					when regexp_replace(btrim(u.phone_number), '[^0-9]', '', 'g') like '07%' then
						'+40' || substr(regexp_replace(btrim(u.phone_number), '[^0-9]', '', 'g'), 2)
					else regexp_replace(btrim(u.phone_number), '[^0-9+]', '', 'g')
				end as normalized_phone
			from app_users u
			where btrim(u.phone_number) <> ''
		) legacy_phone
		where btrim(legacy_phone.normalized_phone) <> ''
		group by legacy_phone.normalized_phone
		having count(distinct legacy_phone.id) > 1
	) then
		raise exception '0083 identity preflight failed: duplicate normalized phone in app_users requires explicit repair';
	end if;
end;
$$;

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
where btrim(u.email) <> '';

insert into app_user_identities (
	user_id, identity_type, normalized_value, display_value, verified_at, is_primary
)
select
	u.id,
	'phone',
	case
		when regexp_replace(btrim(u.phone_number), '[^0-9+]', '', 'g') like '+%' then
			regexp_replace(btrim(u.phone_number), '[^0-9+]', '', 'g')
		when regexp_replace(btrim(u.phone_number), '[^0-9]', '', 'g') like '07%' then
			'+40' || substr(regexp_replace(btrim(u.phone_number), '[^0-9]', '', 'g'), 2)
		else regexp_replace(btrim(u.phone_number), '[^0-9+]', '', 'g')
	end,
	btrim(u.phone_number),
	case when u.phone_number_verified then now() else null end,
	true
from app_users u
where btrim(u.phone_number) <> '';

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

-- Bind each named operator to a unique, already-verified global identity when
-- one exists. Historical OIDC subjects are immutable: an adopted profile
-- keeps its existing sub; the supplied subject is used only for a new profile.
-- Any unverified, ambiguous, or cross-person identity match aborts the whole
-- migration rather than attaching authority to the wrong person.
create temporary table _0083_balotesti_operator_bindings (
	operator_key text primary key,
	desired_sub text not null,
	user_id uuid not null unique references app_users(id) on delete cascade
) on commit drop;

do $$
declare
	operator_row record;
	candidate_user_ids uuid[];
	reserved_subject_user_id uuid;
	bound_user_id uuid;
	has_unverified_identity boolean;
begin
	for operator_row in
		select *
		from (values
			('thomas', 'user-thomas-galambos', 'Thomas Galambos', 'thomas@eguilde.cloud', '+40771364169'),
			('stelian', 'user-stelian-fedorca', 'Stelian Fedorca', '', '+40744652476'),
			('diana', 'user-diana-ilhan', 'Diana Ilhan', '', '+40735091230')
		) as expected(operator_key, desired_sub, display_name, email, phone)
	loop
		select
			array_agg(distinct identity_row.user_id) filter (where identity_row.verified_at is not null),
			coalesce(bool_or(identity_row.verified_at is null), false)
		into candidate_user_ids, has_unverified_identity
		from app_user_identities identity_row
		where (operator_row.email <> ''
				and identity_row.identity_type = 'email'
				and identity_row.normalized_value = lower(operator_row.email))
			or (identity_row.identity_type = 'phone'
				and identity_row.normalized_value = operator_row.phone);

		if has_unverified_identity then
			raise exception '0083 identity preflight failed: % has an unverified reserved login identity', operator_row.operator_key;
		end if;
		if coalesce(cardinality(candidate_user_ids), 0) > 1 then
			raise exception '0083 identity preflight failed: % identities map to multiple existing users', operator_row.operator_key;
		end if;

		select id into reserved_subject_user_id
		from app_users
		where sub = operator_row.desired_sub;
		bound_user_id := candidate_user_ids[1];
		if reserved_subject_user_id is not null
			and (bound_user_id is null or reserved_subject_user_id <> bound_user_id) then
			raise exception '0083 identity preflight failed: reserved subject for % belongs to a different user', operator_row.operator_key;
		end if;

		if bound_user_id is null then
			insert into app_users (
				sub, name, email, phone_number, locale, status,
				email_verified, phone_number_verified, preferred_otp_channel
			)
			values (
				operator_row.desired_sub, operator_row.display_name, operator_row.email, operator_row.phone,
				'ro', 'active', false, false, 'sms'
			)
			returning id into bound_user_id;
		else
			-- Preserve the existing immutable OIDC subject and any email belonging
			-- to a phone-only operator. Only the supplied verified identity and
			-- basic profile state are normalized for the adopted account.
			update app_users
			set name = operator_row.display_name,
				phone_number = operator_row.phone,
				-- A changed phone is only a pending login identifier. It must be
				-- verified by the normal SMS flow; legacy verification survives only
				-- when the profile already owns this exact normalized number.
				phone_number_verified = case
					when case
						when regexp_replace(btrim(app_users.phone_number), '[^0-9+]', '', 'g') like '+%' then
							regexp_replace(btrim(app_users.phone_number), '[^0-9+]', '', 'g')
						when regexp_replace(btrim(app_users.phone_number), '[^0-9]', '', 'g') like '07%' then
							'+40' || substr(regexp_replace(btrim(app_users.phone_number), '[^0-9]', '', 'g'), 2)
						else regexp_replace(btrim(app_users.phone_number), '[^0-9+]', '', 'g')
					end = operator_row.phone then app_users.phone_number_verified
					else false
				end,
				email = case when operator_row.email <> '' then operator_row.email else app_users.email end,
				email_verified = case when operator_row.email <> '' then true else app_users.email_verified end,
				status = 'active',
				preferred_otp_channel = 'sms',
				updated_at = now()
			where id = bound_user_id;
		end if;
		if exists (
			select 1
			from _0083_balotesti_operator_bindings binding
			where binding.user_id = bound_user_id
		) then
			raise exception '0083 identity preflight failed: % shares an existing user with another named operator', operator_row.operator_key;
		end if;

		insert into _0083_balotesti_operator_bindings (operator_key, desired_sub, user_id)
		values (operator_row.operator_key, operator_row.desired_sub, bound_user_id);
	end loop;
end;
$$;

-- The version table deliberately has no super-admin bypass. Bind the migration
-- itself to the target tenant before authorization triggers begin writing it.
select set_config('app.tenant_id', 'tenant-balotesti', true);

-- An adopted account may already have another primary identity of the same
-- type. Keep it as a verified secondary alias and make the supplied identity
-- primary, so the partial primary-by-type uniqueness invariant still holds.
update app_user_identities identity_row
set is_primary = false,
	updated_at = now()
from _0083_balotesti_operator_bindings binding
join (
	values
		('thomas', 'email', 'thomas@eguilde.cloud'),
		('thomas', 'phone', '+40771364169'),
		('stelian', 'phone', '+40744652476'),
		('diana', 'phone', '+40735091230')
) as expected(operator_key, identity_type, normalized_value)
	on expected.operator_key = binding.operator_key
where identity_row.user_id = binding.user_id
	and identity_row.identity_type = expected.identity_type
	and identity_row.normalized_value <> expected.normalized_value
	and identity_row.is_primary;

insert into app_user_identities (
	user_id, identity_type, normalized_value, display_value, verified_at, is_primary
)
select
	binding.user_id,
	expected.identity_type,
	expected.normalized_value,
	expected.display_value,
	case
		when expected.identity_type = 'phone'
			and u.phone_number_verified
			and case
				when regexp_replace(btrim(u.phone_number), '[^0-9+]', '', 'g') like '+%' then
					regexp_replace(btrim(u.phone_number), '[^0-9+]', '', 'g')
				when regexp_replace(btrim(u.phone_number), '[^0-9]', '', 'g') like '07%' then
					'+40' || substr(regexp_replace(btrim(u.phone_number), '[^0-9]', '', 'g'), 2)
				else regexp_replace(btrim(u.phone_number), '[^0-9+]', '', 'g')
			end = expected.normalized_value then now()
		when expected.identity_type = 'email'
			and u.email_verified
			and lower(btrim(u.email)) = expected.normalized_value then now()
		else null
	end,
	true
from _0083_balotesti_operator_bindings binding
join app_users u on u.id = binding.user_id
join (
	values
		('thomas', 'email', 'thomas@eguilde.cloud', 'thomas@eguilde.cloud'),
		('thomas', 'phone', '+40771364169', '+40771364169'),
		('stelian', 'phone', '+40744652476', '+40744652476'),
		('diana', 'phone', '+40735091230', '+40735091230')
) as expected(operator_key, identity_type, normalized_value, display_value)
	on expected.operator_key = binding.operator_key
on conflict (identity_type, normalized_value) do update
set display_value = excluded.display_value,
	verified_at = coalesce(app_user_identities.verified_at, excluded.verified_at),
	is_primary = true,
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
		join _0083_balotesti_operator_bindings binding on binding.desired_sub = expected.subject
		left join app_user_identities i
			on i.user_id = binding.user_id
			and i.identity_type = expected.identity_type
			and i.normalized_value = expected.normalized_value
		where i.id is null
	) then
		raise exception '0083 identity provision failed: a requested identity is already bound to another subject';
	end if;
end;
$$;

-- Preserve auditable provenance when the existing verified Thomas phone is
-- adopted as the canonical login identity. This is a migration event, not an
-- OTP verification substitute; any later phone change remains unverified
-- until the ordinary SMS challenge completes.
insert into app_audit_log (
	institution_id, actor_subject, action, target_type, target_id, status, summary, details
)
select
	'inst-balotesti',
	'migration/0083',
	'identity.phone.legacy_adopted',
	'app_user',
	binding.user_id::text,
	'success',
	'Adopted the existing verified Thomas SMS identity without changing its immutable subject.',
	jsonb_build_object(
		'identity_type', 'phone',
		'legacy_subject', 'thomasgalambos',
		'verification_provenance', 'legacy_profile_verified',
		'remediation', 'approved_synthetic_duplicate_quarantine'
	)
from _0083_balotesti_operator_bindings binding
join app_users u on u.id = binding.user_id
join app_user_identities identity_row
	on identity_row.user_id = binding.user_id
	and identity_row.identity_type = 'phone'
	and identity_row.normalized_value = '+40771364169'
	and identity_row.verified_at is not null
where binding.operator_key = 'thomas'
	and u.sub = 'thomasgalambos';

insert into app_user_platform_roles (user_id, role_code)
select user_id, 'platform_super_admin'
from _0083_balotesti_operator_bindings
where operator_key = 'thomas'
on conflict do nothing;

insert into app_memberships (
	user_id, tenant_code, position_code, org_unit_code, organization_name,
	is_primary, active, start_date
)
select u.id, 'tenant-balotesti', 'director', 'unit-balotesti-root',
	'Școala Gimnazială nr. 1 Balotești', true, true, current_date
from _0083_balotesti_operator_bindings binding
join app_users u on u.id = binding.user_id
where binding.operator_key in ('thomas', 'stelian', 'diana')
	and not exists (
		select 1 from app_memberships m
		where m.user_id = u.id
			and m.tenant_code = 'tenant-balotesti'
			and m.position_code = 'director'
			and m.org_unit_code = 'unit-balotesti-root'
	);

insert into app_user_roles (tenant_code, user_id, role_code)
select 'tenant-balotesti', u.id, 'admin'
from _0083_balotesti_operator_bindings binding
join app_users u on u.id = binding.user_id
where binding.operator_key in ('thomas', 'stelian', 'diana')
on conflict do nothing;

insert into app_user_permissions (tenant_code, user_id, permission_code)
select 'tenant-balotesti', u.id, p.code
from app_users u
cross join app_permissions p
join _0083_balotesti_operator_bindings binding on binding.user_id = u.id
where binding.operator_key in ('thomas', 'stelian', 'diana')
on conflict do nothing;

insert into app_user_modules (tenant_code, user_id, module_code)
select 'tenant-balotesti', u.id, m.code
from app_users u
cross join app_modules m
join _0083_balotesti_operator_bindings binding on binding.user_id = u.id
where m.active
	and binding.operator_key in ('thomas', 'stelian', 'diana')
on conflict do nothing;

insert into app_session_context (
	user_id, institution_id, institution_name, auth_methods, gdpr_capabilities
)
select u.id, 'inst-balotesti', 'Școala Gimnazială nr. 1 Balotești',
	array['oidc_redirect', 'sms_otp', 'passkey'],
	array['retention_policies', 'subject_export', 'purpose_limited_access', 'publication_anonymization']
from app_users u
join _0083_balotesti_operator_bindings binding on binding.user_id = u.id
where binding.operator_key in ('thomas', 'stelian', 'diana')
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
		from _0083_balotesti_operator_bindings binding
		where binding.operator_key in ('thomas', 'stelian', 'diana')
		and not exists (
			select 1 from app_memberships m
			where m.user_id = binding.user_id and m.tenant_code = 'tenant-balotesti' and m.active
		)
	) then
		raise exception '0083 tenant provision failed: an operator has no active Balotesti membership';
	end if;
end;
$$;
