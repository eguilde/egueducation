-- Evidence-backed lifecycle for existing school_regulatory_sources. Existing
-- source identity and all historical foreign keys remain untouched.
select set_config('app.is_super_admin', 'true', true);

alter table school_regulatory_sources drop constraint if exists school_regulatory_sources_status_check;
alter table school_regulatory_sources add constraint school_regulatory_sources_status_check check(status in ('draft','verified','active','superseded','withdrawn'));

create table school_regulatory_source_evidence (
 requested_url text not null check(position('?' in requested_url)=0), source_snapshot jsonb not null,
 id uuid primary key default gen_random_uuid(), tenant_code text not null, institution_id text not null, source_id uuid not null,
 source_version integer not null check(source_version>0), retrieved_url text not null check(position('?' in retrieved_url)=0), content_type text not null default '', content bytea not null check(octet_length(content)>0 and octet_length(content)<=20971520), sha256 text not null check(sha256 ~ '^[0-9a-f]{64}$' and encode(digest(content,'sha256'),'hex')=sha256),
 retrieved_at timestamptz not null, retrieved_by_subject text not null, created_at timestamptz not null default now(),
 unique(tenant_code,institution_id,id), foreign key(tenant_code,institution_id,source_id) references school_regulatory_sources(tenant_code,institution_id,id) on delete restrict
);
create index school_regulatory_source_evidence_latest_idx on school_regulatory_source_evidence(tenant_code,institution_id,source_id,retrieved_at desc,id desc);

create table school_regulatory_source_activations (
 id uuid primary key default gen_random_uuid(), tenant_code text not null, institution_id text not null, source_id uuid not null, evidence_id uuid not null,
 assessment text not null check(length(trim(assessment))>0), activated_by_subject text not null, activated_at timestamptz not null default now(),
 unique(tenant_code,institution_id,id), foreign key(tenant_code,institution_id,source_id) references school_regulatory_sources(tenant_code,institution_id,id) on delete restrict,
 foreign key(tenant_code,institution_id,evidence_id) references school_regulatory_source_evidence(tenant_code,institution_id,id) on delete restrict
);
create unique index school_regulatory_source_activations_evidence_once on school_regulatory_source_activations(tenant_code,institution_id,source_id,evidence_id);

create table school_regulatory_source_idempotency (
 source_snapshot jsonb,
 tenant_code text not null, institution_id text not null, actor_subject text not null, action text not null check(action in ('register','verify','activate')),
 idempotency_key text not null check(length(idempotency_key) between 1 and 200), request_fingerprint text not null check(request_fingerprint ~ '^[0-9a-f]{64}$'), source_id uuid not null, evidence_id uuid,
 created_at timestamptz not null default now(), primary key(tenant_code,institution_id,actor_subject,action,idempotency_key),
 foreign key(tenant_code,institution_id,source_id) references school_regulatory_sources(tenant_code,institution_id,id) on delete restrict,
 foreign key(tenant_code,institution_id,evidence_id) references school_regulatory_source_evidence(tenant_code,institution_id,id) on delete restrict
);

-- Evidence is immutable; no trigger backfills it for old active rows.
create or replace function public.school_regulatory_source_evidence_guard() returns trigger language plpgsql security definer set search_path=pg_catalog,public as $$
begin
 if tg_op<>'INSERT' then raise exception 'regulatory source evidence is immutable'; end if;
 if not public.can_bypass_tenant_rls() and (new.tenant_code<>public.current_tenant_code() or new.institution_id<>public.current_institution_id()) then raise exception 'regulatory source evidence scope mismatch'; end if;
 return new;
end $$;
revoke all on function public.school_regulatory_source_evidence_guard() from public;
create trigger school_regulatory_source_evidence_guard before insert or update or delete on school_regulatory_source_evidence for each row execute function public.school_regulatory_source_evidence_guard();
create or replace function public.school_regulatory_source_activation_guard() returns trigger language plpgsql security definer set search_path=pg_catalog,public as $$
declare creator text; current_version integer; current_status text; current_hash text; current_url text;
begin
 if tg_op<>'INSERT' then raise exception 'regulatory source activation is immutable'; end if;
 if not public.can_bypass_tenant_rls() and (new.tenant_code<>public.current_tenant_code() or new.institution_id<>public.current_institution_id()) then raise exception 'regulatory source activation scope mismatch'; end if;
 select created_by_subject,expected_version,status,checksum_sha256,source_url into creator,current_version,current_status,current_hash,current_url from school_regulatory_sources where tenant_code=new.tenant_code and institution_id=new.institution_id and id=new.source_id;
 if creator=new.activated_by_subject then raise exception 'regulatory source approval requires distinct actor'; end if;
 if current_status<>'active' or not exists(select 1 from school_regulatory_source_evidence where tenant_code=new.tenant_code and institution_id=new.institution_id and id=new.evidence_id and source_id=new.source_id and source_version=current_version-1 and sha256=current_hash and requested_url=current_url) then raise exception 'activation evidence scope mismatch'; end if;
 return new;
end $$;
revoke all on function public.school_regulatory_source_activation_guard() from public;
create trigger school_regulatory_source_activation_guard before insert or update or delete on school_regulatory_source_activations for each row execute function public.school_regulatory_source_activation_guard();
create or replace function public.school_regulatory_source_lifecycle_guard() returns trigger language plpgsql security definer set search_path=pg_catalog,public as $$
begin
 if old.status='active' and new.status='verified' then raise exception 'active regulatory source cannot be reverified'; end if;
 if exists(select 1 from school_regulatory_source_evidence e where e.tenant_code=old.tenant_code and e.institution_id=old.institution_id and e.source_id=old.id) then
  if (to_jsonb(new)-array['status','checksum_sha256','expected_version','verified_at','verified_by_subject','revalidation_owner_subject','updated_by_subject','updated_at']) is distinct from
     (to_jsonb(old)-array['status','checksum_sha256','expected_version','verified_at','verified_by_subject','revalidation_owner_subject','updated_by_subject','updated_at']) then
   raise exception 'evidence-backed regulatory source metadata is immutable';
  end if;
  if new.expected_version<>old.expected_version+1 or not ((old.status='draft' and new.status='verified') or (old.status='verified' and new.status='active')) then
   raise exception 'invalid evidence-backed source transition';
  end if;
  if old.status='verified' and (new.checksum_sha256<>old.checksum_sha256 or new.verified_at is distinct from old.verified_at or new.verified_by_subject<>old.verified_by_subject or new.revalidation_owner_subject<>old.revalidation_owner_subject) then
   raise exception 'source verification provenance is immutable';
  end if;
 end if;
 return new;
end $$;
revoke all on function public.school_regulatory_source_lifecycle_guard() from public;
create trigger school_regulatory_source_lifecycle_guard before update on school_regulatory_sources for each row execute function public.school_regulatory_source_lifecycle_guard();

create or replace function public.school_regulatory_source_completion_guard() returns trigger language plpgsql set search_path=pg_catalog,public as $$
begin
 if new.status='active' and old.status='verified' and not exists(
  select 1 from school_regulatory_source_activations a where a.tenant_code=new.tenant_code and a.institution_id=new.institution_id and a.source_id=new.id
 ) then raise exception 'source activation requires persisted approval'; end if;
 return new;
end $$;
revoke all on function public.school_regulatory_source_completion_guard() from public;
create constraint trigger school_regulatory_source_completion_guard after update on school_regulatory_sources deferrable initially deferred for each row execute function public.school_regulatory_source_completion_guard();

do $$ declare t text; begin foreach t in array array['school_regulatory_source_evidence','school_regulatory_source_activations','school_regulatory_source_idempotency'] loop execute format('alter table %I enable row level security',t); execute format('alter table %I force row level security',t); execute format('create policy %I on %I using(public.can_bypass_tenant_rls() or (tenant_code=public.current_tenant_code() and institution_id=public.current_institution_id())) with check(public.can_bypass_tenant_rls() or (tenant_code=public.current_tenant_code() and institution_id=public.current_institution_id()))',t||'_scope',t); end loop; end $$;

insert into app_permissions(code,label) values ('school.regulatory_sources.read','Read regulatory sources'),('school.regulatory_sources.manage','Register and verify regulatory sources'),('school.regulatory_sources.approve','Activate regulatory sources') on conflict(code) do update set label=excluded.label;
insert into app_position_permissions(position_code,permission_code) values ('arhivar','school.regulatory_sources.read'),('arhivar','school.regulatory_sources.manage'),('director','school.regulatory_sources.read'),('director','school.regulatory_sources.manage'),('director','school.regulatory_sources.approve') on conflict do nothing;
insert into app_role_permissions(role_code,permission_code) values ('arhivar','school.regulatory_sources.read'),('arhivar','school.regulatory_sources.manage'),('director','school.regulatory_sources.read'),('director','school.regulatory_sources.manage'),('director','school.regulatory_sources.approve') on conflict do nothing;
