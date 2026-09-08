-- OTP proofs must be bound to the exact immutable identity that received the
-- challenge. This is the expand phase of a rolling deployment: the column
-- stays nullable so an old pod can continue to write during rollout/rollback.
-- New code writes a binding and rejects/removes unbound rows. A later contract
-- migration may enforce NOT NULL only after every old pod has been replaced.
alter table oidc_otp_codes
	add column if not exists identity_id uuid
	references app_user_identities(id) on delete cascade;

create index if not exists idx_oidc_otp_codes_identity
	on oidc_otp_codes (identity_id, expires_at desc);

-- Tenant RLS must not blind the shared-user guard. The function discloses one
-- boolean only, runs with its owner's table privileges, and locks search_path
-- to prevent object-shadowing attacks. It grants no cross-tenant row access.
create or replace function public.user_has_other_active_tenant_membership(
	p_user_id uuid,
	p_current_tenant_code text
)
returns boolean
language sql
stable
security definer
set search_path = pg_catalog, public
as $$
	select exists (
		select 1
		from public.app_memberships membership
		where membership.user_id = p_user_id
			and membership.active
			and membership.tenant_code <> p_current_tenant_code
	)
$$;

comment on function public.user_has_other_active_tenant_membership(uuid, text) is
	'RLS-safe boolean guard preventing a tenant administrator from mutating a global identity shared with another tenant.';
