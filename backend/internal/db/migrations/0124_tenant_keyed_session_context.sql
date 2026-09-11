-- A profile is global, while its session bootstrap settings are tenant-local.
-- The original user_id primary key silently made the last writer win whenever
-- one person belonged to more than one tenant.  That could surface the wrong
-- tenant's authentication capabilities during an otherwise valid OIDC flow.
--
-- Preserve every existing row, fail closed if it cannot be bound to a real
-- tenant, then create a context for every already-active membership using the
-- established settings.  This keeps existing users able to sign in while
-- making every subsequent lookup explicit about the host-derived tenant.
select set_config('app.is_super_admin', 'true', true);

alter table app_session_context
	add column if not exists tenant_code text;

-- PostgreSQL may choose an arbitrary source row in UPDATE ... FROM when more
-- than one tenant shares an institution. Refuse that legacy shape before
-- writing anything so a session can never be attached to a guessed tenant.
do $$
begin
	if exists (
		select session_context.institution_id
		from app_session_context session_context
		left join app_tenants tenant
			on tenant.institution_id = session_context.institution_id
		where session_context.tenant_code is null
		group by session_context.institution_id
		having count(distinct tenant.code) <> 1
	) then
		raise exception '0124 session context mapping is ambiguous: every unbound institution must map to exactly one tenant code';
	end if;
end;
$$;

update app_session_context session_context
set tenant_code = tenant.code
from app_tenants tenant
where session_context.tenant_code is null
	and tenant.institution_id = session_context.institution_id;

do $$
begin
	if exists (
		select 1
		from app_session_context session_context
		left join app_tenants tenant on tenant.code = session_context.tenant_code
		where session_context.tenant_code is null
			or tenant.institution_id is distinct from session_context.institution_id
	) then
		raise exception '0124 session context preflight failed: every existing context must map to exactly one tenant institution';
	end if;
end;
$$;

alter table app_session_context
	drop constraint if exists app_session_context_pkey;

-- The historical row is the source of its established authentication policy.
-- Copy it only to active memberships that do not yet have a tenant context.
insert into app_session_context (
	user_id, tenant_code, institution_id, institution_name, auth_methods, gdpr_capabilities
)
select
	source.user_id,
	tenant.code,
	tenant.institution_id,
	tenant.display_name,
	source.auth_methods,
	source.gdpr_capabilities
from app_session_context source
join app_memberships membership
	on membership.user_id = source.user_id
	and membership.active = true
	and membership.start_date <= current_date
	and (membership.end_date is null or membership.end_date >= current_date)
join app_tenants tenant on tenant.code = membership.tenant_code and tenant.active = true
left join app_session_context target
	on target.user_id = source.user_id and target.tenant_code = tenant.code
where target.user_id is null;

alter table app_session_context
	alter column tenant_code set not null;

alter table app_session_context
	add constraint app_session_context_pkey primary key (user_id, tenant_code),
	add constraint app_session_context_tenant_institution_fkey
		foreign key (tenant_code, institution_id)
		references app_tenants (code, institution_id)
		on delete cascade;

create index if not exists idx_app_session_context_tenant_user
	on app_session_context (tenant_code, user_id);
