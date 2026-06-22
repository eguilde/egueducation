create table if not exists oidc_authn_sessions (
	tenant_id uuid not null,
	id text not null,
	data jsonb not null,
	expires_at timestamptz not null,
	primary key (tenant_id, id)
);

create table if not exists oidc_grant_sessions (
	tenant_id uuid not null,
	id text not null,
	data jsonb not null,
	expires_at timestamptz not null,
	primary key (tenant_id, id)
);

create table if not exists oidc_models (
	tenant_id uuid not null,
	id bigserial not null,
	model_id varchar(255) not null,
	model_name varchar(64) not null,
	payload jsonb not null,
	grant_id varchar(255),
	user_code varchar(255),
	uid varchar(255),
	expires_at timestamptz,
	consumed_at timestamptz,
	created_at timestamptz not null default now(),
	primary key (tenant_id, id)
);

create index if not exists idx_oidc_authn_sessions_expires on oidc_authn_sessions (tenant_id, expires_at desc);
create index if not exists idx_oidc_grant_sessions_expires on oidc_grant_sessions (tenant_id, expires_at desc);
create index if not exists idx_oidc_models_expires on oidc_models (tenant_id, expires_at desc);
