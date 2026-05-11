create table if not exists oidc_consents (
	id uuid primary key default gen_random_uuid(),
	client_id text not null references oidc_clients(client_id) on delete cascade,
	subject text not null,
	scope text not null,
	granted_at timestamptz not null default now(),
	updated_at timestamptz not null default now(),
	unique (client_id, subject, scope)
);

create table if not exists oidc_consent_requests (
	id uuid primary key default gen_random_uuid(),
	client_id text not null references oidc_clients(client_id) on delete cascade,
	subject text not null,
	redirect_uri text not null,
	scope text not null,
	state text not null,
	nonce text null,
	code_challenge text not null,
	code_challenge_method text not null default 'S256',
	status text not null default 'pending',
	expires_at timestamptz not null,
	created_at timestamptz not null default now(),
	updated_at timestamptz not null default now()
);

create index if not exists idx_oidc_consent_requests_subject_expires
	on oidc_consent_requests(subject, expires_at desc);
