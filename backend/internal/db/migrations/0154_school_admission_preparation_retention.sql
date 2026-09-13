-- A legal preparation fixes the retention authority before the signing
-- ceremony. Existing preparations intentionally remain without this snapshot
-- and cannot be finalized; manufacturing legal provenance later is forbidden.
alter table school_admission_legal_preparations
 add column retention_policy_id uuid,
 add column retention_rule_version_id uuid,
 add column retention_source_id uuid,
 add column retention_anchor_at timestamptz,
 add column minimum_retention_days integer,
 add column required_retention_until timestamptz;
alter table school_admission_legal_preparations
 add constraint school_admission_preparation_retention_policy_fk foreign key(tenant_code,institution_id,retention_policy_id) references school_admission_dss_retention_policies(tenant_code,institution_id,id) on delete restrict,
 add constraint school_admission_preparation_retention_rule_fk foreign key(tenant_code,institution_id,retention_rule_version_id) references school_admission_retention_rule_versions(tenant_code,institution_id,id) on delete restrict,
 add constraint school_admission_preparation_retention_source_fk foreign key(tenant_code,institution_id,retention_source_id) references school_regulatory_sources(tenant_code,institution_id,id) on delete restrict,
 add constraint school_admission_preparation_retention_values_check check((retention_policy_id is null and retention_rule_version_id is null and retention_source_id is null and retention_anchor_at is null and minimum_retention_days is null and required_retention_until is null) or (retention_policy_id is not null and retention_rule_version_id is not null and retention_source_id is not null and retention_anchor_at is not null and minimum_retention_days is not null and minimum_retention_days between 1 and 36500 and required_retention_until is not null));

create or replace function public.school_admission_preparation_retention_guard() returns trigger language plpgsql security definer set search_path=pg_catalog,public as $$
declare rule_id uuid; source_id uuid; days integer;
begin
 if tg_op='INSERT' then
  select p.rule_version_id,r.source_id,r.minimum_retention_days into rule_id,source_id,days
  from school_admission_dss_retention_policies p join school_admission_retention_rule_versions r on r.tenant_code=p.tenant_code and r.institution_id=p.institution_id and r.id=p.rule_version_id
  where p.tenant_code=new.tenant_code and p.institution_id=new.institution_id and p.id=new.retention_policy_id and p.status='active' and p.effective_from<=timezone('UTC',now())::date and (p.effective_to is null or p.effective_to>=timezone('UTC',now())::date) and p.effective_from<=(new.expires_at at time zone 'UTC')::date and (p.effective_to is null or p.effective_to>=(new.expires_at at time zone 'UTC')::date) and r.status='active' and r.artifact_kind='admission_dss' and r.effective_from<=timezone('UTC',now())::date and (r.effective_to is null or r.effective_to>=timezone('UTC',now())::date) and r.effective_from<=(new.expires_at at time zone 'UTC')::date and (r.effective_to is null or r.effective_to>=(new.expires_at at time zone 'UTC')::date);
  if rule_id is null or new.retention_policy_id is null or new.retention_rule_version_id is distinct from rule_id or new.retention_source_id is distinct from source_id or new.minimum_retention_days is distinct from days or new.retention_anchor_at is distinct from new.expires_at or new.required_retention_until is distinct from ((new.expires_at at time zone 'UTC')+make_interval(days=>days)) at time zone 'UTC' then raise exception using errcode='23514',message='admission preparation requires exact active retention authority snapshot'; end if;
 elsif row(new.retention_policy_id,new.retention_rule_version_id,new.retention_source_id,new.retention_anchor_at,new.minimum_retention_days,new.required_retention_until) is distinct from row(old.retention_policy_id,old.retention_rule_version_id,old.retention_source_id,old.retention_anchor_at,old.minimum_retention_days,old.required_retention_until) then
  raise exception 'admission preparation retention snapshot is immutable';
 end if;
 return new;
end $$;
revoke all on function public.school_admission_preparation_retention_guard() from public;
create trigger school_admission_preparation_retention_guard before insert or update on school_admission_legal_preparations for each row execute function public.school_admission_preparation_retention_guard();
