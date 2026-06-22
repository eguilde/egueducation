create table if not exists oidc_clients (
	client_id text primary key,
	client_name text not null,
	public_client boolean not null default true,
	require_pkce boolean not null default true,
	active boolean not null default true,
	data jsonb not null default '{}'::jsonb,
	created_at timestamptz not null default now(),
	updated_at timestamptz not null default now()
);

create table if not exists oidc_client_redirect_uris (
	client_id text not null references oidc_clients(client_id) on delete cascade,
	redirect_uri text not null,
	primary key (client_id, redirect_uri)
);

create table if not exists oidc_authorization_codes (
	code text primary key,
	client_id text not null references oidc_clients(client_id) on delete cascade,
	subject text not null,
	redirect_uri text not null,
	scope text not null,
	nonce text null,
	code_challenge text not null,
	code_challenge_method text not null default 'S256',
	expires_at timestamptz not null,
	used boolean not null default false,
	created_at timestamptz not null default now(),
	updated_at timestamptz not null default now()
);

create index if not exists idx_oidc_authorization_codes_client_expires
	on oidc_authorization_codes (client_id, expires_at desc);

create table if not exists oidc_refresh_tokens (
	token text primary key,
	client_id text not null references oidc_clients(client_id) on delete cascade,
	subject text not null,
	scope text not null,
	cnf_jkt text null,
	expires_at timestamptz not null,
	revoked boolean not null default false,
	created_at timestamptz not null default now(),
	updated_at timestamptz not null default now()
);

create index if not exists idx_oidc_refresh_tokens_client_expires
	on oidc_refresh_tokens (client_id, expires_at desc);

insert into oidc_clients (client_id, client_name, public_client, require_pkce, active)
values
	('egueducation-spa', 'EguEducation SPA', true, true, true),
	('egueducation-desktop', 'EguEducation Desktop', true, true, true)
on conflict (client_id) do update
set client_name = excluded.client_name,
	public_client = excluded.public_client,
	require_pkce = excluded.require_pkce,
	active = excluded.active,
	data = jsonb_build_object(
		'client_id', excluded.client_id,
		'client_name', excluded.client_name,
		'public_client', excluded.public_client,
		'require_pkce', excluded.require_pkce,
		'active', excluded.active
	),
	updated_at = now();

insert into oidc_client_redirect_uris (client_id, redirect_uri)
values
	('egueducation-spa', 'http://localhost:4200/auth/callback'),
	('egueducation-spa', 'https://scoalabalotesti.eguilde.cloud/auth/callback'),
	('egueducation-desktop', 'http://localhost:4200/auth/callback')
on conflict (client_id, redirect_uri) do nothing;
