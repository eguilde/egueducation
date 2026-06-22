create table if not exists app_passkey_login_challenges (
	id uuid primary key default gen_random_uuid(),
	challenge text not null unique,
	expires_at timestamptz not null,
	created_at timestamptz not null default now()
);

create index if not exists idx_app_passkey_login_challenges_expires_at
	on app_passkey_login_challenges(expires_at desc);
