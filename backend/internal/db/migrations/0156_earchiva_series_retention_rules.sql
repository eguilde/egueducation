create table archive_series_retention_rules (
 id uuid primary key default gen_random_uuid(), tenant_code text not null, institution_id text not null,
 taxonomy_node_id uuid not null, source_id uuid not null, source_checksum_sha256 text not null check(source_checksum_sha256 ~ '^[0-9a-f]{64}$'),
 anchor_kind text not null check(anchor_kind in ('intake_received_at','event','permanent')), duration_model text not null check(duration_model in ('minimum_days','event_based','permanent')),
 minimum_retention_days integer, effective_from date not null, effective_to date,
 status text not null default 'proposed' check(status in ('proposed','active','retired','revoked')), expected_version integer not null default 1 check(expected_version>0),
 proposed_by_subject text not null, proposed_at timestamptz not null default now(), approved_by_subject text not null default '', approved_at timestamptz, retired_by_subject text not null default '', retired_at timestamptz, retirement_reason text not null default '',
 created_at timestamptz not null default now(), updated_at timestamptz not null default now(),
 unique(tenant_code,institution_id,id),
 foreign key(tenant_code,institution_id) references app_tenants(code,institution_id) on delete restrict,
 foreign key(institution_id,taxonomy_node_id) references archive_taxonomy_nodes(institution_id,id) on delete restrict,
 foreign key(tenant_code,institution_id,source_id) references school_regulatory_sources(tenant_code,institution_id,id) on delete restrict,
 check(effective_to is null or effective_to>=effective_from),
 check((duration_model='minimum_days' and minimum_retention_days between 1 and 36500) or (duration_model<>'minimum_days' and minimum_retention_days is null)),
 check(status not in ('active','retired','revoked') or (approved_by_subject<>'' and approved_at is not null)),
 check(status not in ('retired','revoked') or (retired_by_subject<>'' and retired_at is not null and retirement_reason<>''))
);
alter table archive_series_retention_rules add constraint archive_series_retention_rules_active_window_excl exclude using gist(tenant_code with =,institution_id with =,taxonomy_node_id with =,daterange(effective_from,coalesce(effective_to,'infinity'::date),'[]') with &&) where(status='active');

create or replace function public.archive_series_retention_rule_guard() returns trigger language plpgsql security definer set search_path=pg_catalog,public as $$
declare source_checksum text; source_ok boolean; source_effective_from date; source_effective_to date;
begin
 if tg_op='DELETE' then raise exception 'archive series retention rules are append-only'; end if;
 if not public.can_bypass_tenant_rls() and (new.tenant_code<>public.current_tenant_code() or new.institution_id<>public.current_institution_id()) then raise exception 'archive series retention rule scope mismatch'; end if;
 if tg_op='INSERT' then
  select checksum_sha256,(status='active' and verified_at is not null and verified_by_subject<>'' and source_url<>'' and checksum_sha256<>'') into source_checksum,source_ok from school_regulatory_sources where tenant_code=new.tenant_code and institution_id=new.institution_id and id=new.source_id;
  if not coalesce(source_ok,false) or lower(source_checksum)<>lower(new.source_checksum_sha256) then raise exception 'archive series retention rule requires active verified source checksum'; end if;
  new.proposed_by_subject:=current_setting('app.actor_subject',true); new.proposed_at:=now(); new.approved_by_subject:=''; new.approved_at:=null; new.retired_by_subject:=''; new.retired_at:=null; new.retirement_reason:=''; new.status:='proposed'; new.expected_version:=1; new.created_at:=now(); new.updated_at:=now();
 elsif new.tenant_code<>old.tenant_code or new.institution_id<>old.institution_id or new.taxonomy_node_id<>old.taxonomy_node_id or new.source_id<>old.source_id or new.source_checksum_sha256<>old.source_checksum_sha256 or new.anchor_kind<>old.anchor_kind or new.duration_model<>old.duration_model or new.minimum_retention_days is distinct from old.minimum_retention_days or new.effective_from<>old.effective_from or new.effective_to is distinct from old.effective_to or new.proposed_by_subject<>old.proposed_by_subject or new.proposed_at<>old.proposed_at or new.approved_by_subject<>old.approved_by_subject or new.approved_at is distinct from old.approved_at or new.retired_by_subject<>old.retired_by_subject or new.retired_at is distinct from old.retired_at or new.created_at<>old.created_at or new.updated_at<>old.updated_at or (new.retirement_reason<>old.retirement_reason and not(old.status='active' and new.status in ('retired','revoked'))) then raise exception 'archive series retention rule provenance is immutable';
 elsif old.status='proposed' and new.status='active' then
  if current_setting('app.actor_subject',true)=old.proposed_by_subject then raise exception 'archive series retention approval requires distinct actor'; end if;
  select checksum_sha256,(status='active' and verified_at is not null and verified_by_subject<>'' and source_url<>'' and checksum_sha256<>''),effective_from,effective_to into source_checksum,source_ok,source_effective_from,source_effective_to from school_regulatory_sources where tenant_code=old.tenant_code and institution_id=old.institution_id and id=old.source_id for share;
  if not coalesce(source_ok,false) or lower(source_checksum)<>lower(old.source_checksum_sha256) or source_effective_from is null or source_effective_from>old.effective_from or (old.effective_to is null and source_effective_to is not null) or (old.effective_to is not null and source_effective_to is not null and source_effective_to<old.effective_to) then raise exception 'archive series retention approval requires current source coverage'; end if;
  perform pg_advisory_xact_lock(hashtextextended(old.tenant_code||'/'||old.institution_id||'/'||old.taxonomy_node_id::text,0));
  if exists(select 1 from archive_series_retention_rules r where r.tenant_code=old.tenant_code and r.institution_id=old.institution_id and r.taxonomy_node_id=old.taxonomy_node_id and r.status='active' and daterange(r.effective_from,coalesce(r.effective_to,'infinity'::date),'[]') && daterange(new.effective_from,coalesce(new.effective_to,'infinity'::date),'[]')) then raise exception 'archive series retention active windows overlap'; end if;
  new.approved_by_subject:=current_setting('app.actor_subject',true); new.approved_at:=now(); new.expected_version:=old.expected_version+1;
 elsif old.status='active' and new.status in ('retired','revoked') then new.retired_by_subject:=current_setting('app.actor_subject',true); new.retired_at:=now(); new.expected_version:=old.expected_version+1;
 else raise exception 'invalid archive series retention lifecycle transition'; end if;
 new.updated_at:=now(); return new;
end $$;
revoke all on function public.archive_series_retention_rule_guard() from public;
create trigger archive_series_retention_rule_guard before insert or update or delete on archive_series_retention_rules for each row execute function public.archive_series_retention_rule_guard();
alter table archive_series_retention_rules enable row level security; alter table archive_series_retention_rules force row level security;
create policy archive_series_retention_rules_scope on archive_series_retention_rules using(public.can_bypass_tenant_rls() or (tenant_code=public.current_tenant_code() and institution_id=public.current_institution_id())) with check(public.can_bypass_tenant_rls() or (tenant_code=public.current_tenant_code() and institution_id=public.current_institution_id()));
create trigger archive_series_retention_rules_entity_version after insert or update or delete on archive_series_retention_rules for each row execute function public.record_entity_version();
create trigger archive_series_retention_rules_no_hard_delete before delete on archive_series_retention_rules for each row execute function public.school_operations_no_hard_delete();
create or replace function public.archive_series_retention_source_guard() returns trigger language plpgsql security definer set search_path=pg_catalog,public as $$
begin
 if exists(select 1 from archive_series_retention_rules r where r.tenant_code=old.tenant_code and r.institution_id=old.institution_id and r.source_id=old.id) and (new.source_kind<>old.source_kind or new.citation<>old.citation or new.article_reference<>old.article_reference or new.issuer<>old.issuer or new.source_url<>old.source_url or new.published_on is distinct from old.published_on or new.consolidated_on is distinct from old.consolidated_on or new.effective_from is distinct from old.effective_from or new.effective_to is distinct from old.effective_to or new.checksum_sha256<>old.checksum_sha256 or new.status<>old.status or new.verified_at is distinct from old.verified_at or new.verified_by_subject<>old.verified_by_subject or new.revalidation_owner_subject<>old.revalidation_owner_subject) then raise exception 'archive retention referenced source provenance is immutable'; end if;
 return new;
end $$;
revoke all on function public.archive_series_retention_source_guard() from public;
create trigger archive_series_retention_source_guard before update on school_regulatory_sources for each row execute function public.archive_series_retention_source_guard();
create table archive_series_retention_rule_idempotency (
 tenant_code text not null, institution_id text not null, actor_subject text not null,
 action text not null check(action in ('propose','approve','retire')), idempotency_key text not null check(length(idempotency_key) between 1 and 200),
 request_fingerprint text not null check(request_fingerprint ~ '^[0-9a-f]{64}$'), rule_id uuid not null,
 created_at timestamptz not null default now(),
 primary key(tenant_code,institution_id,actor_subject,action,idempotency_key),
 foreign key(tenant_code,institution_id) references app_tenants(code,institution_id) on delete restrict,
 foreign key(tenant_code,institution_id,rule_id) references archive_series_retention_rules(tenant_code,institution_id,id) on delete restrict
);
alter table archive_series_retention_rule_idempotency enable row level security;
alter table archive_series_retention_rule_idempotency force row level security;
create policy archive_series_retention_rule_idempotency_scope on archive_series_retention_rule_idempotency using(public.can_bypass_tenant_rls() or (tenant_code=public.current_tenant_code() and institution_id=public.current_institution_id())) with check(public.can_bypass_tenant_rls() or (tenant_code=public.current_tenant_code() and institution_id=public.current_institution_id()));
insert into app_permissions(code,label) values ('earchiva.retention.read','Read archive series retention rules'),('earchiva.retention.manage','Propose or retire archive series retention rules'),('earchiva.retention.approve','Approve archive series retention rules') on conflict(code) do update set label=excluded.label;
insert into app_position_permissions(position_code,permission_code) values
 ('arhivar','earchiva.retention.read'),('arhivar','earchiva.retention.manage'),
 ('director','earchiva.retention.read'),('director','earchiva.retention.manage'),('director','earchiva.retention.approve') on conflict do nothing;
insert into app_role_permissions(role_code,permission_code) values
 ('arhivar','earchiva.retention.read'),('arhivar','earchiva.retention.manage'),
 ('director','earchiva.retention.read'),('director','earchiva.retention.manage'),('director','earchiva.retention.approve') on conflict do nothing;
