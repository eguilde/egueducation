alter table oidc_refresh_tokens
	add column if not exists cnf_jkt text null;
