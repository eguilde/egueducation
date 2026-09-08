-- Phone numbers are globally-owned login credentials.  The unique identity
-- index introduced in 0083 supplies ownership; this migration makes the
-- profile projection impossible to mark verified unless its primary identity
-- has already been verified by the SMS OTP transaction.

create or replace function public.assert_profile_phone_identity_contract()
returns trigger
language plpgsql
as $$
declare
	normalized_phone text;
begin
	if btrim(coalesce(new.phone_number, '')) = '' then
		if new.phone_number_verified then
			raise exception 'phone verification requires an assigned phone identity';
		end if;
		return new;
	end if;

	normalized_phone := case
		when regexp_replace(btrim(new.phone_number), '[^0-9+]', '', 'g') like '+%' then regexp_replace(btrim(new.phone_number), '[^0-9+]', '', 'g')
		when regexp_replace(btrim(new.phone_number), '[^0-9]', '', 'g') like '07%' then '+40' || substr(regexp_replace(btrim(new.phone_number), '[^0-9]', '', 'g'), 2)
		else regexp_replace(btrim(new.phone_number), '[^0-9+]', '', 'g')
	end;

	-- Every non-empty profile phone must be represented by the sole primary
	-- phone identity. This removes stale aliases from the login resolver when a
	-- user changes their own number.
	if not exists (
		select 1 from app_user_identities i
		where i.user_id = new.id
		  and i.identity_type = 'phone'
		  and i.is_primary
		  and i.normalized_value = normalized_phone
	) then
		raise exception 'profile phone requires a matching primary phone identity';
	end if;

	if new.phone_number_verified and not exists (
		select 1 from app_user_identities i
		where i.user_id = new.id
		  and i.identity_type = 'phone'
		  and i.is_primary
		  and i.normalized_value = normalized_phone
		  and i.verified_at is not null
	) then
		raise exception 'phone verification requires successful SMS OTP identity verification';
	end if;
	return new;
end;
$$;

alter table app_user_identities
	add constraint app_user_identities_phone_e164
	check (identity_type <> 'phone' or normalized_value ~ '^\+[1-9][0-9]{7,14}$') not valid;

-- The NOT VALID form protects every future insert/update at the database
-- boundary without guessing how to repair historical data during deployment.

drop trigger if exists trg_app_users_phone_identity_contract on app_users;
create constraint trigger trg_app_users_phone_identity_contract
	after insert or update of phone_number, phone_number_verified on app_users
	deferrable initially deferred
	for each row execute function public.assert_profile_phone_identity_contract();

comment on function public.assert_profile_phone_identity_contract() is
	'Ensures app_users.phone_number is the primary global phone identity and phone_number_verified follows OTP verification only.';
