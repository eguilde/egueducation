alter table app_users
	add column if not exists email_verified boolean not null default false,
	add column if not exists phone_number_verified boolean not null default false,
	add column if not exists preferred_otp_channel text not null default 'sms';

update app_users
set
	phone_number = '010000001',
	email_verified = true,
	phone_number_verified = true,
	preferred_otp_channel = 'sms',
	updated_at = now()
where sub = 'platform-admin@example.test';

insert into app_auth_methods (code, enabled, primary_method, sort_order)
values ('sms_otp', true, false, 20)
on conflict (code) do update
set enabled = true,
	sort_order = excluded.sort_order,
	updated_at = now();
