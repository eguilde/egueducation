-- 0086 prevents app_users.phone_number_verified from being enabled without a
-- matching verified primary identity. This reverse projection closes the
-- opposite write path: deleting, demoting, moving or de-verifying that
-- identity immediately revokes the profile projection as well.

create or replace function public.revoke_profile_phone_from_identity_change()
returns trigger
language plpgsql
security definer
set search_path = pg_catalog, public
as $$
declare
	identity_no_longer_primary boolean := false;
	verification_revoked boolean := false;
begin
	if old.identity_type <> 'phone' or not old.is_primary then
		if tg_op = 'DELETE' then
			return old;
		end if;
		return new;
	end if;

	if tg_op = 'DELETE' then
		identity_no_longer_primary := true;
	else
		identity_no_longer_primary :=
			new.user_id is distinct from old.user_id
			or new.identity_type is distinct from old.identity_type
			or new.normalized_value is distinct from old.normalized_value
			or not new.is_primary;
		verification_revoked := old.verified_at is not null and new.verified_at is null;
	end if;

	if identity_no_longer_primary or verification_revoked then
		update public.app_users profile
		set
			phone_number = case when identity_no_longer_primary then '' else profile.phone_number end,
			phone_number_verified = false,
			updated_at = now()
		where profile.id = old.user_id
			and case
				when regexp_replace(btrim(profile.phone_number), '[^0-9+]', '', 'g') like '+%' then regexp_replace(btrim(profile.phone_number), '[^0-9+]', '', 'g')
				when regexp_replace(btrim(profile.phone_number), '[^0-9]', '', 'g') like '07%' then '+40' || substr(regexp_replace(btrim(profile.phone_number), '[^0-9]', '', 'g'), 2)
				else regexp_replace(btrim(profile.phone_number), '[^0-9+]', '', 'g')
			end = old.normalized_value;
	end if;

	if tg_op = 'DELETE' then
		return old;
	end if;
	return new;
end;
$$;

drop trigger if exists trg_phone_identity_reverse_projection on public.app_user_identities;
create trigger trg_phone_identity_reverse_projection
	after delete or update of user_id, identity_type, normalized_value, verified_at, is_primary
	on public.app_user_identities
	for each row execute function public.revoke_profile_phone_from_identity_change();

comment on function public.revoke_profile_phone_from_identity_change() is
	'Revokes app_users phone verification (and stale phone projections) when the matching primary phone identity is de-verified, deleted, demoted, moved or renamed.';

revoke all on function public.revoke_profile_phone_from_identity_change() from public;
