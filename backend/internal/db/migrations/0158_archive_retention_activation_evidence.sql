-- Preserve historical rules, but never authorize a new rule from legacy
-- metadata-only sources. Apply the same boundary to direct SQL and HTTP writes.
create function public.archive_retention_activation_evidence_guard()
returns trigger language plpgsql security definer
set search_path=pg_catalog,public as $$
declare v_source_version integer; v_source_hash text; v_source_url text; v_source_status text;
begin
 if tg_op='UPDATE' then
  if not (old.status='proposed' and new.status='active') then return new; end if;
 end if;

 if not public.can_bypass_tenant_rls() and
    (new.tenant_code<>public.current_tenant_code() or new.institution_id<>public.current_institution_id()) then
  raise exception 'archive retention evidence scope mismatch';
 end if;

 select s.expected_version,s.checksum_sha256,s.source_url,s.status
 into v_source_version,v_source_hash,v_source_url,v_source_status
 from school_regulatory_sources s
 where s.tenant_code=new.tenant_code and s.institution_id=new.institution_id and s.id=new.source_id
 for share;

 if not found or v_source_status<>'active' or v_source_hash<>new.source_checksum_sha256 then
  raise exception 'archive retention requires activated source evidence';
 end if;
 if not exists (
  select 1 from school_regulatory_source_activations a
  join school_regulatory_source_evidence e
   on e.tenant_code=a.tenant_code and e.institution_id=a.institution_id
   and e.id=a.evidence_id and e.source_id=a.source_id
  where a.tenant_code=new.tenant_code and a.institution_id=new.institution_id
   and a.source_id=new.source_id and e.sha256=v_source_hash
   and e.requested_url=v_source_url and e.source_version=v_source_version-1
 ) then
  raise exception 'archive retention requires activated source evidence';
 end if;
 return new;
end $$;
revoke all on function public.archive_retention_activation_evidence_guard() from public;
create trigger archive_retention_activation_evidence_guard
before insert or update on archive_series_retention_rules
for each row execute function public.archive_retention_activation_evidence_guard();
