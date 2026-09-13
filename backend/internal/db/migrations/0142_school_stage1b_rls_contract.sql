-- Contract phase: scope every Stage 1B aggregate and prohibit deleting final evidence.
do $$ declare t text; begin foreach t in array array['school_regulatory_sources','school_institution_profiles_v2','school_profile_sources','school_institution_party_roles','school_locations','school_education_offerings','school_offering_authorizations','school_funding_instruments','school_procurement_applicability_assessments','school_education_contracts','school_quality_cycles','school_network_memberships','school_operation_policy_inputs','school_operation_policy_evaluations_v2'] loop execute format('alter table %I enable row level security',t); execute format('alter table %I force row level security',t); execute format('create policy tenant_isolation on %I using (public.can_bypass_tenant_rls() or (tenant_code=public.current_tenant_code() and institution_id=public.current_institution_id())) with check (public.can_bypass_tenant_rls() or (tenant_code=public.current_tenant_code() and institution_id=public.current_institution_id()))',t); end loop; end $$;
create or replace function public.school_stage1b_final_no_delete() returns trigger language plpgsql as $$
begin
 if tg_table_name='school_education_contracts' then
  if old.status in ('active','suspended','terminated','archived') then raise exception 'final school regulatory evidence cannot be deleted'; end if;
 elsif tg_table_name='school_offering_authorizations' then
  if old.status in ('accredited','withdrawn','expired') then raise exception 'final school regulatory evidence cannot be deleted'; end if;
 elsif tg_table_name='school_procurement_applicability_assessments' then
  if old.decided_at is not null then raise exception 'final school regulatory evidence cannot be deleted'; end if;
 end if;
 return old;
end $$;
create trigger school_education_contracts_final_no_delete before delete on school_education_contracts for each row execute function public.school_stage1b_final_no_delete();
create trigger school_offering_authorizations_final_no_delete before delete on school_offering_authorizations for each row execute function public.school_stage1b_final_no_delete();
create trigger school_procurement_assessments_final_immutable before update or delete on school_procurement_applicability_assessments for each row execute function public.school_stage1b_final_no_delete();
