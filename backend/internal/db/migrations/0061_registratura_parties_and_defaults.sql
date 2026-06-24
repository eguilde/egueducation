create table if not exists app_parties (
	id uuid primary key default gen_random_uuid(),
	tenant_code text not null references app_tenants(code) on delete restrict,
	institution_id text not null,
	code text not null,
	party_type text not null check (party_type in ('physical', 'legal', 'institution')),
	display_name text not null,
	short_name text not null default '',
	first_name text not null default '',
	last_name text not null default '',
	legal_name text not null default '',
	identifier_code text not null default '',
	tax_id text not null default '',
	phone_number text not null default '',
	email text not null default '',
	address_line1 text not null default '',
	address_line2 text not null default '',
	locality text not null default '',
	county text not null default '',
	country text not null default 'RO',
	notes text not null default '',
	is_default_organization boolean not null default false,
	active boolean not null default true,
	created_at timestamptz not null default now(),
	updated_at timestamptz not null default now(),
	unique (tenant_code, code)
);

create index if not exists idx_app_parties_institution_type_name
	on app_parties (institution_id, party_type, display_name);

alter table app_parties
	enable row level security;
alter table app_parties
	force row level security;
drop policy if exists tenant_isolation on app_parties;
create policy tenant_isolation on app_parties
	using (public.can_bypass_tenant_rls() or institution_id = public.current_institution_id())
	with check (public.can_bypass_tenant_rls() or institution_id = public.current_institution_id());

do $$
begin
	if exists (
		select 1
		from pg_proc
		where proname = 'record_entity_version'
			and pronamespace = 'public'::regnamespace
	) then
		drop trigger if exists trg_app_parties_entity_version on app_parties;
		execute 'create trigger trg_app_parties_entity_version after insert or update or delete on app_parties for each row execute function public.record_entity_version()';
	end if;
end;
$$;

insert into app_parties (
	tenant_code,
	institution_id,
	code,
	party_type,
	display_name,
	short_name,
	legal_name,
	is_default_organization,
	active,
	created_at,
	updated_at
)
select
	t.code,
	t.institution_id,
	'organization',
	'institution',
	t.display_name,
	t.short_name,
	t.display_name,
	true,
	true,
	now(),
	now()
from app_tenants t
where t.active = true
on conflict (tenant_code, code) do update
set institution_id = excluded.institution_id,
	party_type = excluded.party_type,
	display_name = excluded.display_name,
	short_name = excluded.short_name,
	legal_name = excluded.legal_name,
	is_default_organization = excluded.is_default_organization,
	active = excluded.active,
	updated_at = now();

alter table registratura_documents
	add column if not exists correspondent_party_id uuid null references app_parties(id) on delete set null;

alter table registratura_documents
	add column if not exists assigned_party_id uuid null references app_parties(id) on delete set null;

create index if not exists idx_registratura_documents_correspondent_party
	on registratura_documents (institution_id, correspondent_party_id);

create index if not exists idx_registratura_documents_assigned_party
	on registratura_documents (institution_id, assigned_party_id);

update registratura_documents d
set assigned_party_id = p.id,
	assigned_to = case when d.assigned_to = '' then p.display_name else d.assigned_to end,
	updated_at = now()
from app_parties p
where p.institution_id = d.institution_id
	and p.is_default_organization = true
	and d.direction = 'intrare'
	and d.assigned_party_id is null;

update registratura_documents d
set correspondent_party_id = p.id,
	correspondent = case when d.correspondent = '' then p.display_name else d.correspondent end,
	updated_at = now()
from app_parties p
where p.institution_id = d.institution_id
	and p.is_default_organization = true
	and d.direction = 'iesire'
	and d.correspondent_party_id is null;

