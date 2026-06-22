create table if not exists app_passkeys (
	id uuid primary key default gen_random_uuid(),
	user_id uuid not null references app_users(id) on delete cascade,
	credential_id text not null unique,
	device_name text not null default 'Passkey',
	credential_payload jsonb not null default '{}'::jsonb,
	created_at timestamptz not null default now(),
	last_used_at timestamptz
);

create table if not exists app_passkey_challenges (
	id uuid primary key default gen_random_uuid(),
	user_id uuid not null references app_users(id) on delete cascade,
	challenge text not null,
	kind text not null default 'registration',
	expires_at timestamptz not null,
	created_at timestamptz not null default now()
);

create table if not exists app_eudi_wallets (
	user_id uuid primary key references app_users(id) on delete cascade,
	status text not null default 'inactive',
	activated_at timestamptz,
	updated_at timestamptz not null default now()
);
