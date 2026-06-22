create table if not exists app_login_sessions (
	session_id text primary key,
	subject text not null,
	expires_at timestamptz not null,
	revoked boolean not null default false,
	last_used_at timestamptz not null default now(),
	created_at timestamptz not null default now(),
	updated_at timestamptz not null default now()
);

create index if not exists idx_app_login_sessions_subject_expires
	on app_login_sessions(subject, expires_at desc);

create table if not exists oidc_signing_keys (
	key_id text primary key,
	private_key_pem text not null,
	active boolean not null default true,
	created_at timestamptz not null default now(),
	updated_at timestamptz not null default now()
);
