-- Fail closed if historical profile projections claim phone possession without
-- the verified primary global identity required by the current contract.
do $$
declare
	invalid_user_ids text;
begin
	select string_agg(invalid.id::text, ', ' order by invalid.id::text)
	into invalid_user_ids
	from (
		select u.id
		from app_users u
		where u.phone_number_verified
		  and not exists (
			select 1
			from app_user_identities identity
			where identity.user_id = u.id
			  and identity.identity_type = 'phone'
			  and identity.is_primary
			  and identity.verified_at is not null
			  and identity.normalized_value = case
				when regexp_replace(btrim(u.phone_number), '[^0-9+]', '', 'g') like '+%' then regexp_replace(btrim(u.phone_number), '[^0-9+]', '', 'g')
				when regexp_replace(btrim(u.phone_number), '[^0-9]', '', 'g') like '07%' then '+40' || substr(regexp_replace(btrim(u.phone_number), '[^0-9]', '', 'g'), 2)
				else regexp_replace(btrim(u.phone_number), '[^0-9+]', '', 'g')
			  end
		  )
		order by u.id
		limit 25
	) invalid;

	if invalid_user_ids is not null then
		raise exception using
			message = 'phone identity historical validation failed',
			detail = 'Verified profile rows without matching verified primary identity: ' || invalid_user_ids,
			hint = 'Repair or revoke the legacy verification projection explicitly before retrying migration 0094.';
	end if;
end
$$;

alter table app_user_identities
	validate constraint app_user_identities_phone_e164;
