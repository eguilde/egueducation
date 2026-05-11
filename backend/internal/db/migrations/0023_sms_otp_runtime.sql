create table if not exists sms_queue (
	id bigserial primary key,
	to_phone text not null,
	message text not null,
	status text not null,
	provider_id text,
	error text,
	sent_at timestamptz,
	created_at timestamptz not null default now(),
	updated_at timestamptz not null default now()
);

create table if not exists sms_otp_codes (
	id bigserial primary key,
	identifier text not null,
	code_hash text not null,
	purpose text not null,
	expires_at timestamptz not null,
	used boolean not null default false,
	attempts integer not null default 0,
	created_at timestamptz not null default now(),
	updated_at timestamptz not null default now()
);

create index if not exists idx_sms_otp_codes_identifier_created_at on sms_otp_codes(identifier, created_at desc);
create index if not exists idx_sms_queue_created_at on sms_queue(created_at desc);
