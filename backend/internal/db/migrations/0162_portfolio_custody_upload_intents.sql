create unique index if not exists education_portfolios_institution_id_id_key on education_portfolios(institution_id,id);

create table portfolio_custody_upload_intents (
 id uuid primary key default gen_random_uuid(), tenant_code text not null, institution_id text not null,
 portfolio_id uuid not null, actor_subject text not null, idempotency_key text not null,
 expected_sha256 text not null check(expected_sha256 ~ '^[0-9a-f]{64}$'), expected_size_bytes bigint not null check(expected_size_bytes>0),
 expected_mime_type text not null default 'application/pdf', expected_request_fingerprint text not null default '', expected_metadata jsonb not null default '{}'::jsonb,
 bucket_name text not null, object_key text not null, reserved_document_id uuid not null, reserved_version_id uuid not null,
 status text not null default 'reserved' check(status in ('reserved','stored','committed','failed')),
 stored_version_id text not null default '', stored_etag text not null default '', stored_size_bytes bigint,
 custody_hold_active boolean not null default true, failure_code text not null default '', created_at timestamptz not null default now(), updated_at timestamptz not null default now(),
 unique(tenant_code,institution_id,actor_subject,idempotency_key), unique(institution_id,reserved_document_id), unique(institution_id,reserved_version_id),
 foreign key(tenant_code,institution_id) references app_tenants(code,institution_id) on delete restrict,
 foreign key(institution_id,portfolio_id) references education_portfolios(institution_id,id) on delete restrict
);
alter table portfolio_custody_upload_intents enable row level security;
alter table portfolio_custody_upload_intents force row level security;
create policy portfolio_custody_upload_intents_scope on portfolio_custody_upload_intents using(public.can_bypass_tenant_rls() or (tenant_code=public.current_tenant_code() and institution_id=public.current_institution_id())) with check(public.can_bypass_tenant_rls() or (tenant_code=public.current_tenant_code() and institution_id=public.current_institution_id()));

create or replace function public.portfolio_custody_upload_intent_guard() returns trigger language plpgsql security definer set search_path=pg_catalog,public as $$
begin
 if tg_op='DELETE' then raise exception 'portfolio custody upload intents cannot be deleted'; end if;
 if tg_op='UPDATE' and row(new.tenant_code,new.institution_id,new.portfolio_id,new.actor_subject,new.idempotency_key,new.expected_sha256,new.expected_size_bytes,new.expected_mime_type,new.expected_request_fingerprint,new.expected_metadata,new.bucket_name,new.object_key,new.reserved_document_id,new.reserved_version_id,new.custody_hold_active) is distinct from row(old.tenant_code,old.institution_id,old.portfolio_id,old.actor_subject,old.idempotency_key,old.expected_sha256,old.expected_size_bytes,old.expected_mime_type,old.expected_request_fingerprint,old.expected_metadata,old.bucket_name,old.object_key,old.reserved_document_id,old.reserved_version_id,old.custody_hold_active) then raise exception 'portfolio custody intent provenance is immutable'; end if;
 if tg_op='UPDATE' and old.status<>'reserved' and row(new.stored_version_id,new.stored_etag,new.stored_size_bytes) is distinct from row(old.stored_version_id,old.stored_etag,old.stored_size_bytes) then raise exception 'verified custody storage identity is immutable'; end if;
 if tg_op='INSERT' then
  if new.actor_subject<>current_setting('app.actor_subject',true) or new.status<>'reserved' or new.stored_version_id<>'' or new.stored_etag<>'' or new.stored_size_bytes is not null or not new.custody_hold_active then raise exception 'invalid portfolio custody upload reservation'; end if;
 elsif old.status='reserved' and new.status='stored' then
  if new.stored_version_id='' or new.stored_etag='' or new.stored_size_bytes is distinct from new.expected_size_bytes or not new.custody_hold_active then raise exception 'portfolio custody upload requires verified exact held version'; end if;
 elsif old.status='stored' and new.status='committed' then null;
 elsif old.status='reserved' and new.status='failed' then null;
 elsif row(new.tenant_code,new.institution_id,new.portfolio_id,new.actor_subject,new.idempotency_key,new.expected_sha256,new.expected_size_bytes,new.expected_mime_type,new.expected_request_fingerprint,new.expected_metadata,new.bucket_name,new.object_key,new.reserved_document_id,new.reserved_version_id,new.custody_hold_active) is distinct from row(old.tenant_code,old.institution_id,old.portfolio_id,old.actor_subject,old.idempotency_key,old.expected_sha256,old.expected_size_bytes,old.expected_mime_type,old.expected_request_fingerprint,old.expected_metadata,old.bucket_name,old.object_key,old.reserved_document_id,old.reserved_version_id,old.custody_hold_active) then raise exception 'portfolio custody intent provenance is immutable';
 else raise exception 'invalid portfolio custody upload intent transition'; end if;
 new.updated_at:=now(); return new;
end $$;
create trigger portfolio_custody_upload_intent_guard before insert or update or delete on portfolio_custody_upload_intents for each row execute function public.portfolio_custody_upload_intent_guard();

alter table archive_document_versions add column if not exists portfolio_custody_intent_id uuid references portfolio_custody_upload_intents(id) on delete restrict;
create unique index if not exists archive_versions_portfolio_custody_intent_key on archive_document_versions(portfolio_custody_intent_id) where portfolio_custody_intent_id is not null;
create or replace function public.archive_version_custody_intent_guard() returns trigger language plpgsql security definer set search_path=pg_catalog,public as $$
declare i portfolio_custody_upload_intents%rowtype;
begin
 if new.custody_hold_active and new.portfolio_custody_intent_id is null then raise exception 'custody-held archive version requires verified custody intent'; end if;
 if tg_op='UPDATE' and old.portfolio_custody_intent_id is not null and new.portfolio_custody_intent_id is distinct from old.portfolio_custody_intent_id then raise exception 'archive custody intent link is immutable'; end if;
 if new.portfolio_custody_intent_id is null then return new; end if;
 select * into i from portfolio_custody_upload_intents where id=new.portfolio_custody_intent_id;
 if not found or i.status not in ('stored','committed') or i.institution_id<>new.institution_id or i.reserved_document_id<>new.document_id or i.reserved_version_id<>new.id or i.bucket_name<>new.source_bucket or i.object_key<>new.source_object_key or i.stored_version_id<>new.source_object_version_id or i.stored_etag<>new.source_object_etag or lower(i.expected_sha256)<>lower(new.source_sha256) or i.expected_size_bytes<>new.source_size_bytes or i.expected_mime_type<>new.mime_type or not i.custody_hold_active or not new.custody_hold_active then raise exception 'archive version must adopt matching verified custody intent'; end if;
 return new;
end $$;
create trigger archive_version_custody_intent_guard before insert or update on archive_document_versions for each row execute function public.archive_version_custody_intent_guard();
