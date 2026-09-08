-- Migration 0083 was published before its legacy-identity adoption was
-- hardened. Environments which already recorded that version cannot replay
-- its corrected body. Reconcile the only security-sensitive legacy outcome:
-- operator rows created by the old migration were incorrectly marked as
-- phone-verified without an OTP possession proof.
select set_config('app.tenant_id', 'tenant-balotesti', true);
select set_config('app.institution_id', 'inst-balotesti', true);
select set_config('app.is_super_admin', 'false', true);

do $$
declare
	operator_row record;
	operator_user_id uuid;
	has_sms_possession_proof boolean;
begin
	if not exists (
		select 1 from schema_migrations
		where version = '0083_identity_contract_foundation.sql'
	) then
		raise exception '0095 reconciliation requires the 0083 migration ledger entry';
	end if;

	for operator_row in
		select * from (values
			('user-thomas-galambos', '+40771364169'),
			('user-stelian-fedorca', '+40744652476'),
			('user-diana-ilhan', '+40735091230')
		) expected(subject, phone)
	loop
		operator_user_id := null;
		select profile.id
		into operator_user_id
		from app_users profile
		join app_user_identities identity
		  on identity.user_id = profile.id
		 and identity.identity_type = 'phone'
		 and identity.normalized_value = operator_row.phone
		 and identity.is_primary
		join app_memberships membership
		  on membership.user_id = profile.id
		 and membership.tenant_code = 'tenant-balotesti'
		 and membership.active
		join app_user_roles role_grant
		  on role_grant.user_id = profile.id
		 and role_grant.tenant_code = 'tenant-balotesti'
		 and role_grant.role_code = 'admin'
		where profile.sub = operator_row.subject;

		-- The corrected 0083 may adopt an older immutable subject (for example
		-- Thomas's established account). Only exact subjects used by the old
		-- published bootstrap are candidates for this reconciliation.
		if operator_user_id is null then
			continue;
		end if;

		select exists (
			select 1
			from app_audit_log audit
			where audit.institution_id = 'inst-balotesti'
			  and audit.status = 'success'
			  and audit.action in ('identity.phone.enrollment_verified', 'identity.phone.login_authenticated')
			  and audit.details->>'user_id' = operator_user_id::text
			  and audit.details->>'method' = 'sms_otp'
		) into has_sms_possession_proof;

		if not has_sms_possession_proof then
			update app_user_identities
			set verified_at = null,
				updated_at = now()
			where user_id = operator_user_id
			  and identity_type = 'phone'
			  and normalized_value = operator_row.phone;

			update app_users
			set phone_number_verified = false,
				updated_at = now()
			where id = operator_user_id;
		end if;
	end loop;
end
$$;

-- Whichever safe path produced the operator accounts (new subject or adopted
-- immutable legacy subject), each supplied phone must resolve to exactly one
-- active Balotesti administrator. This is validation only; ambiguous identity
-- ownership is never repaired by guessing.
do $$
declare
	operator_phone text;
	matching_users integer;
	matching_admins integer;
begin
	foreach operator_phone in array array['+40771364169', '+40744652476', '+40735091230']
	loop
		select count(distinct identity.user_id)
		into matching_users
		from app_user_identities identity
		where identity.identity_type = 'phone'
		  and identity.normalized_value = operator_phone
		  and identity.is_primary;

		select count(distinct identity.user_id)
		into matching_admins
		from app_user_identities identity
		join app_memberships membership
		  on membership.user_id = identity.user_id
		 and membership.tenant_code = 'tenant-balotesti'
		 and membership.active
		join app_user_roles role_grant
		  on role_grant.user_id = identity.user_id
		 and role_grant.tenant_code = 'tenant-balotesti'
		 and role_grant.role_code = 'admin'
		where identity.identity_type = 'phone'
		  and identity.normalized_value = operator_phone
		  and identity.is_primary;

		if matching_users <> 1 or matching_admins <> 1 then
			raise exception using
				message = '0095 Balotesti operator reconciliation failed',
				detail = format('phone %s has %s primary owners and %s active tenant administrators', operator_phone, matching_users, matching_admins),
				hint = 'Resolve the global identity ownership explicitly; this migration will not select an owner automatically.';
		end if;
	end loop;
end
$$;
