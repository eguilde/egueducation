-- Declaration wording is a versioned legal artefact.  It intentionally lives
-- in persistence, rather than in an HTTP handler, so an acknowledgement always
-- preserves precisely the text that was presented to the teacher.
create table if not exists education_portfolio_declaration_templates (
	id uuid primary key default gen_random_uuid(),
	declaration_type text not null check (declaration_type in ('gdpr_information', 'authenticity')),
	declaration_version text not null check (btrim(declaration_version) <> ''),
	declaration_text text not null check (btrim(declaration_text) <> ''),
	source_ref text not null check (btrim(source_ref) <> ''),
	lifecycle_status text not null default 'published' check (lifecycle_status in ('draft', 'published', 'superseded', 'withdrawn')),
	effective_from date not null,
	effective_to date,
	created_at timestamptz not null default now(),
	updated_at timestamptz not null default now(),
	unique (declaration_type, declaration_version),
	check (effective_to is null or effective_to >= effective_from)
);

create unique index if not exists uq_education_portfolio_declaration_template_current
	on education_portfolio_declaration_templates (declaration_type)
	where lifecycle_status = 'published' and effective_to is null;

-- The active wording is seeded as database data, never reconstructed by an
-- API client or handler. A later legal revision is inserted as a new version.
insert into education_portfolio_declaration_templates (
	declaration_type, declaration_version, declaration_text, source_ref,
	lifecycle_status, effective_from, effective_to
) values
	(
		'gdpr_information',
		'gdpr-information-v1-2026-09-01',
		'Am luat cunoștință de informarea privind prelucrarea datelor cu caracter personal din portofoliul profesional, inclusiv scopurile, accesul autorizat, perioadele de păstrare și drepturile persoanei vizate.',
		'Regulamentul (UE) 2016/679; Ordinul nr. 3.858/2026, metodologia-cadru',
		'published', date '2026-09-01', null
	),
	(
		'authenticity',
		'authenticity-v1-2026-09-01',
		'Declar pe propria răspundere că documentele și informațiile pe care le includ în portofoliul profesional sunt complete, corecte și corespund activității mele profesionale.',
		'Ordinul nr. 3.858/2026, metodologia-cadru, Anexa nr. 1',
		'published', date '2026-09-01', null
	)
on conflict (declaration_type, declaration_version) do nothing;

create or replace function public.prevent_education_portfolio_declaration_template_rewrite()
returns trigger language plpgsql as $$
begin
	if tg_op = 'DELETE' then
		raise exception 'portfolio declaration templates are legal records and cannot be deleted';
	end if;
	if old.declaration_type is distinct from new.declaration_type
		or old.declaration_version is distinct from new.declaration_version
		or old.declaration_text is distinct from new.declaration_text
		or old.source_ref is distinct from new.source_ref then
		raise exception 'published portfolio declaration wording is immutable; create a new version';
	end if;
	new.updated_at = now();
	return new;
end;
$$;

drop trigger if exists trg_education_portfolio_declaration_template_immutable on education_portfolio_declaration_templates;
create trigger trg_education_portfolio_declaration_template_immutable
	before update or delete on education_portfolio_declaration_templates
	for each row execute function public.prevent_education_portfolio_declaration_template_rewrite();
