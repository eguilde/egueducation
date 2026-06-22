create table if not exists oidc_jwks_keys (
	key_id text primary key,
	use text not null default 'sig',
	alg text not null default 'RS256',
	private_key text not null,
	public_key text not null,
	jwk jsonb not null,
	active boolean not null default true,
	rotated_at timestamptz null,
	created_at timestamptz not null default now(),
	updated_at timestamptz not null default now()
);

create index if not exists idx_oidc_jwks_keys_active_created_at
	on oidc_jwks_keys (active, created_at desc);

create table if not exists oidc_otp_codes (
	user_id uuid not null references app_users(id) on delete cascade,
	purpose text not null,
	code_hash text not null,
	expires_at timestamptz not null,
	attempts integer not null default 0,
	created_at timestamptz not null default now(),
	updated_at timestamptz not null default now(),
	primary key (user_id, purpose)
);

create index if not exists idx_oidc_otp_codes_expires_at
	on oidc_otp_codes (expires_at desc);

create table if not exists oidc_passkey_login_nonces (
	nonce text primary key,
	user_id uuid not null references app_users(id) on delete cascade,
	expires_at timestamptz not null,
	created_at timestamptz not null default now()
);

create index if not exists idx_oidc_passkey_login_nonces_expires_at
	on oidc_passkey_login_nonces (expires_at desc);
