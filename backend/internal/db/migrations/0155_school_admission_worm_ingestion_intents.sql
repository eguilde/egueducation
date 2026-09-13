-- Preparation-bound, durable WORM ingestion.  This is deliberately separate
-- from the general archive upload path: its retention authority is the legal
-- preparation snapshot, never browser metadata or an archive default.
create table archive_ingestion_intents (
 id uuid primary key default gen_random_uuid(),
 tenant_code text not null, institution_id text not null,
 actor_subject text not null, idempotency_key text not null,
 request_fingerprint text not null check(request_fingerprint ~ '^[0-9a-f]{64}$'),
 purpose text not null check(purpose='admission_legal_preparation'),
 preparation_id uuid not null, artifact_slot text not null check(artifact_slot in ('primary','resulting_decision')),
 expected_sha256 text not null check(expected_sha256 ~ '^[0-9a-f]{64}$'), expected_size_bytes bigint not null check(expected_size_bytes>=0),
 reserved_document_id uuid not null, reserved_version_id uuid not null,
 bucket_name text not null, object_key text not null,
 retention_until timestamptz not null, legal_hold_active boolean not null default true,
 status text not null default 'reserved' check(status in ('reserved','stored','committed','failed')),
 stored_version_id text not null default '', stored_etag text not null default '', stored_size_bytes bigint, stored_retention_until timestamptz,
 failure_code text not null default '', committed_at timestamptz, created_at timestamptz not null default now(), updated_at timestamptz not null default now(),
 unique(tenant_code,institution_id,id),
 unique(tenant_code,institution_id,actor_subject,idempotency_key),
 unique(tenant_code,institution_id,preparation_id,artifact_slot),
 unique(institution_id,reserved_document_id), unique(institution_id,reserved_version_id),
 foreign key(tenant_code,institution_id) references app_tenants(code,institution_id) on delete restrict,
 foreign key(tenant_code,institution_id,preparation_id) references school_admission_legal_preparations(tenant_code,institution_id,id) on delete restrict,
 check((status in ('stored','committed')) = (btrim(stored_version_id)<>'' and btrim(stored_etag)<>'' and stored_size_bytes is not null and stored_retention_until is not null and stored_retention_until>=retention_until)),
 check((status='committed') = (committed_at is not null))
);

alter table archive_document_versions add column ingestion_intent_id uuid;
alter table archive_document_versions add constraint archive_document_versions_ingestion_intent_fk
 foreign key(ingestion_intent_id) references archive_ingestion_intents(id) on delete restrict;
create unique index archive_document_versions_ingestion_intent_unique on archive_document_versions(ingestion_intent_id) where ingestion_intent_id is not null;

create or replace function public.archive_ingestion_intent_guard() returns trigger language plpgsql security definer set search_path=pg_catalog,public as $$
declare preparation record; authority_ok boolean;
begin
 if tg_op='DELETE' then raise exception 'archive ingestion intents cannot be deleted'; end if;
 if not public.can_bypass_tenant_rls() and (new.tenant_code<>public.current_tenant_code() or new.institution_id<>public.current_institution_id()) then raise exception 'archive ingestion intent tenant/institution context mismatch'; end if;
 if tg_op='INSERT' then
 if nullif(btrim(current_setting('app.actor_subject',true)),'') is null or new.actor_subject<>current_setting('app.actor_subject',true) then raise exception 'archive ingestion intent actor context mismatch'; end if;
	if new.status<>'reserved' or new.stored_version_id<>'' or new.stored_etag<>'' or new.stored_size_bytes is not null or new.stored_retention_until is not null or new.committed_at is not null or new.expected_size_bytes<=0 or new.expected_size_bytes>104857600 then raise exception 'archive ingestion intent must reserve a bounded unstored artifact'; end if;
	select p.status,p.prepared_by_subject,p.expires_at,p.artifact_kind,p.resulting_decision_id,p.retention_policy_id,p.retention_rule_version_id,p.retention_source_id,p.minimum_retention_days,p.required_retention_until into preparation from school_admission_legal_preparations p where p.tenant_code=new.tenant_code and p.institution_id=new.institution_id and p.id=new.preparation_id for key share;
	if not found or preparation.status<>'prepared' or preparation.prepared_by_subject<>new.actor_subject or preparation.expires_at<=now() or preparation.retention_policy_id is null or preparation.retention_rule_version_id is null or preparation.retention_source_id is null or preparation.minimum_retention_days is null or preparation.required_retention_until is null or new.retention_until<preparation.required_retention_until or (new.artifact_slot='resulting_decision' and (preparation.artifact_kind<>'admission_appeal_resolution' or preparation.resulting_decision_id is null)) then raise exception 'archive ingestion intent requires an active matching legal preparation'; end if;
	select exists(select 1 from school_admission_dss_retention_policies policy join school_admission_retention_rule_versions rule on rule.tenant_code=policy.tenant_code and rule.institution_id=policy.institution_id and rule.id=policy.rule_version_id where policy.tenant_code=new.tenant_code and policy.institution_id=new.institution_id and policy.id=preparation.retention_policy_id and policy.rule_version_id=preparation.retention_rule_version_id and policy.status='active' and policy.effective_from<=timezone('UTC',now())::date and (policy.effective_to is null or policy.effective_to>=timezone('UTC',now())::date) and rule.status='active' and rule.source_id=preparation.retention_source_id and rule.minimum_retention_days=preparation.minimum_retention_days) into authority_ok;
	if not authority_ok then raise exception 'archive ingestion intent retention authority is no longer approved'; end if;
 elsif row(new.id,new.tenant_code,new.institution_id,new.actor_subject,new.idempotency_key,new.request_fingerprint,new.purpose,new.preparation_id,new.artifact_slot,new.expected_sha256,new.expected_size_bytes,new.reserved_document_id,new.reserved_version_id,new.bucket_name,new.object_key,new.retention_until,new.legal_hold_active,new.stored_version_id,new.stored_etag,new.stored_size_bytes,new.stored_retention_until,new.committed_at) is distinct from row(old.id,old.tenant_code,old.institution_id,old.actor_subject,old.idempotency_key,old.request_fingerprint,old.purpose,old.preparation_id,old.artifact_slot,old.expected_sha256,old.expected_size_bytes,old.reserved_document_id,old.reserved_version_id,old.bucket_name,old.object_key,old.retention_until,old.legal_hold_active,case when old.status='reserved' then new.stored_version_id else old.stored_version_id end,case when old.status='reserved' then new.stored_etag else old.stored_etag end,case when old.status='reserved' then new.stored_size_bytes else old.stored_size_bytes end,case when old.status='reserved' then new.stored_retention_until else old.stored_retention_until end,case when old.status='stored' then new.committed_at else old.committed_at end) then raise exception 'archive ingestion intent provenance is immutable';
 elsif old.status='reserved' and new.status in ('stored','failed') then null;
 elsif old.status='stored' and new.status='committed' then null;
 elsif new.status<>old.status then raise exception 'invalid archive ingestion intent transition'; end if;
 new.updated_at:=now();
 return new;
end $$;
revoke all on function public.archive_ingestion_intent_guard() from public;
create trigger archive_ingestion_intent_guard before insert or update or delete on archive_ingestion_intents for each row execute function public.archive_ingestion_intent_guard();

create or replace function public.archive_version_ingestion_intent_guard() returns trigger language plpgsql security definer set search_path=pg_catalog,public as $$
declare i archive_ingestion_intents%rowtype;
begin
 if tg_op='UPDATE' and new.ingestion_intent_id is distinct from old.ingestion_intent_id then raise exception 'archive version ingestion intent is immutable'; end if;
 if new.ingestion_intent_id is null then return new; end if;
 select * into i from archive_ingestion_intents where id=new.ingestion_intent_id;
 if not found or i.status not in ('stored','committed') or i.tenant_code is distinct from public.current_tenant_code() or i.institution_id is distinct from new.institution_id or i.reserved_document_id is distinct from new.document_id or i.reserved_version_id is distinct from new.id or i.bucket_name is distinct from new.source_bucket or i.object_key is distinct from new.source_object_key or i.stored_version_id is distinct from new.source_object_version_id or i.stored_etag is distinct from new.source_object_etag or lower(i.expected_sha256) is distinct from lower(new.source_sha256) or i.expected_size_bytes is distinct from new.source_size_bytes or i.stored_retention_until is distinct from new.retention_until or not new.legal_hold_active then raise exception 'archive version must adopt its matching verified ingestion intent'; end if;
 return new;
end $$;
revoke all on function public.archive_version_ingestion_intent_guard() from public;
create trigger archive_version_ingestion_intent_guard before insert or update on archive_document_versions for each row execute function public.archive_version_ingestion_intent_guard();

alter table archive_ingestion_intents enable row level security; alter table archive_ingestion_intents force row level security;
create policy archive_ingestion_intent_tenant_isolation on archive_ingestion_intents using(public.can_bypass_tenant_rls() or (tenant_code=public.current_tenant_code() and institution_id=public.current_institution_id())) with check(public.can_bypass_tenant_rls() or (tenant_code=public.current_tenant_code() and institution_id=public.current_institution_id()));
create trigger archive_ingestion_intent_entity_version after insert or update or delete on archive_ingestion_intents for each row execute function public.record_entity_version();
create trigger archive_ingestion_intent_no_hard_delete before delete on archive_ingestion_intents for each row execute function public.school_operations_no_hard_delete();
