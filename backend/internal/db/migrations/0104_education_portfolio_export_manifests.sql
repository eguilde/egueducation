-- An interoperable portfolio export is an evidentiary manifest, not a
-- browser-composed table export.  The SHA-256 is calculated over the stable
-- manifest payload by the application and the immutable JSON snapshot keeps
-- the exact archive version/provenance that was released.
create table if not exists education_portfolio_export_manifests (
	id uuid primary key default gen_random_uuid(),
	portfolio_id uuid not null references education_portfolios(id) on delete restrict,
	institution_id text not null,
	tenant_code text not null references app_tenants(code) on delete restrict,
	manifest_version text not null check (btrim(manifest_version) <> ''),
	manifest_sha256 text not null check (manifest_sha256 ~ '^[0-9a-f]{64}$'),
	manifest jsonb not null check (jsonb_typeof(manifest) = 'object'),
	generated_by_subject text not null default '',
	generated_at timestamptz not null default now(),
	unique (portfolio_id, manifest_sha256)
);

create index if not exists idx_education_portfolio_export_manifests_scope
	on education_portfolio_export_manifests (tenant_code, institution_id, portfolio_id, generated_at desc);

create or replace function public.enforce_education_portfolio_export_manifest_scope()
returns trigger language plpgsql as $$
begin
	perform public.enforce_education_portfolio_tenant_scope(new.institution_id, new.tenant_code, null, 'portfolio export manifest');
	if not exists (
		select 1 from education_portfolios portfolio
		where portfolio.id = new.portfolio_id and portfolio.institution_id = new.institution_id
	) then
		raise exception 'portfolio export manifest must belong to its portfolio institution';
	end if;
	if tg_op = 'UPDATE' and row(new.portfolio_id, new.institution_id, new.tenant_code, new.manifest_version, new.manifest_sha256, new.manifest)
		is distinct from row(old.portfolio_id, old.institution_id, old.tenant_code, old.manifest_version, old.manifest_sha256, old.manifest) then
		raise exception 'portfolio export manifest evidence is immutable';
	end if;
	return new;
end;
$$;

drop trigger if exists trg_education_portfolio_export_manifest_scope on education_portfolio_export_manifests;
create trigger trg_education_portfolio_export_manifest_scope
	before insert or update on education_portfolio_export_manifests
	for each row execute function public.enforce_education_portfolio_export_manifest_scope();

create or replace function public.prevent_education_portfolio_export_manifest_delete()
returns trigger language plpgsql as $$
begin
	raise exception 'portfolio export manifests are evidentiary records and cannot be deleted';
end;
$$;

drop trigger if exists trg_education_portfolio_export_manifest_immutable on education_portfolio_export_manifests;
create trigger trg_education_portfolio_export_manifest_immutable
	before delete on education_portfolio_export_manifests
	for each row execute function public.prevent_education_portfolio_export_manifest_delete();

alter table education_portfolio_export_manifests enable row level security;
alter table education_portfolio_export_manifests force row level security;
drop policy if exists tenant_isolation on education_portfolio_export_manifests;
create policy tenant_isolation on education_portfolio_export_manifests
	using (public.can_bypass_tenant_rls() or (institution_id = public.current_institution_id() and tenant_code = public.current_tenant_code()))
	with check (public.can_bypass_tenant_rls() or (institution_id = public.current_institution_id() and tenant_code = public.current_tenant_code()));

drop trigger if exists trg_education_portfolio_export_manifests_entity_version on education_portfolio_export_manifests;
create trigger trg_education_portfolio_export_manifests_entity_version
	after insert on education_portfolio_export_manifests
	for each row execute function public.record_entity_version();
