alter table oidc_clients
	add column if not exists data jsonb not null default '{}'::jsonb;

update oidc_clients c
set data = jsonb_build_object(
	'client_id', c.client_id,
	'client_name', c.client_name,
	'public_client', c.public_client,
	'require_pkce', c.require_pkce,
	'active', c.active
)
where coalesce(c.data, '{}'::jsonb) = '{}'::jsonb;
