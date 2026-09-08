-- OTP challenges are authentication-session capabilities. Use a new table so
-- rolling deployments cannot let an older pod consume challenges created by
-- the session-bound implementation through the legacy `(user_id, purpose)`
-- lookup.
create table if not exists oidc_otp_challenges (
	authn_session_id text not null,
	purpose text not null,
	user_id uuid not null references app_users(id) on delete cascade,
	tenant_code text not null references app_tenants(code) on delete cascade,
	identity_id uuid not null references app_user_identities(id) on delete cascade,
	code_hash text not null,
	expires_at timestamptz not null,
	attempts integer not null default 0,
	created_at timestamptz not null default now(),
	updated_at timestamptz not null default now(),
	primary key (authn_session_id, purpose),
	constraint oidc_otp_challenges_session_nonempty check (btrim(authn_session_id) <> ''),
	constraint oidc_otp_challenges_tenant_nonempty check (btrim(tenant_code) <> ''),
	constraint oidc_otp_challenges_attempts_nonnegative check (attempts >= 0)
);

create index if not exists idx_oidc_otp_challenges_user_scope
	on oidc_otp_challenges (user_id, tenant_code, purpose, expires_at desc);

create index if not exists idx_oidc_otp_challenges_expiry
	on oidc_otp_challenges (expires_at desc);

comment on table oidc_otp_challenges is
	'One-time authentication proofs bound to immutable identity, tenant, purpose, and exact OIDC authentication session.';

-- Rows in the pre-session-bound table are deliberately invalidated. They
-- cannot prove which browser authorization transaction requested the code.
delete from oidc_otp_codes;
