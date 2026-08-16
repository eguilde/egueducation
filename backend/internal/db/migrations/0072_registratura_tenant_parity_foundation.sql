-- Tenant-safe Registratura parity foundation.  Costesti is a single-tenant
-- reference; none of these records may be shared implicitly between schools.

alter table registre add column if not exists tenant_code text;
alter table registre add column if not exists institution_id text;
alter table registre add column if not exists visibility text not null default 'public'
    check (visibility in ('public', 'private'));
alter table registre add column if not exists active boolean not null default true;
alter table app_tenants add constraint app_tenants_code_institution_unique unique (code, institution_id);

alter table registratura_documents add column if not exists tenant_code text;
update registratura_documents d set tenant_code=t.code from app_tenants t where t.institution_id=d.institution_id and d.tenant_code is null;
alter table registratura_documents alter column tenant_code set not null;
alter table registratura_documents add constraint registratura_documents_tenant_fk foreign key (tenant_code,institution_id) references app_tenants(code,institution_id) on delete restrict;
alter table registratura_documents add constraint registratura_documents_tenant_id_unique unique (tenant_code,id);
alter table registratura_documents drop constraint if exists registratura_documents_registry_number_key;
alter table registratura_documents add constraint registratura_documents_tenant_registry_number_unique unique (tenant_code,registry_number);

alter table registratura_document_versions add column if not exists tenant_code text;
alter table registratura_document_versions add column if not exists institution_id text;
update registratura_document_versions v set tenant_code=d.tenant_code,institution_id=d.institution_id from registratura_documents d where d.id=v.document_id and (v.tenant_code is null or v.institution_id is null);
alter table registratura_document_versions alter column tenant_code set not null;
alter table registratura_document_versions alter column institution_id set not null;
alter table registratura_document_versions alter column tenant_code set default public.current_tenant_code();
alter table registratura_document_versions alter column institution_id set default public.current_institution_id();
alter table registratura_document_versions add constraint registratura_versions_document_tenant_fk foreign key (tenant_code,document_id) references registratura_documents(tenant_code,id) on delete cascade;

-- Legacy registries were global.  Split each configuration per active tenant;
-- documents retain their own tenant's clone.  Empty configurations are cloned
-- to every tenant, avoiding an implicit primary-tenant fallback.
do $$
declare rec record; clone_id bigint; target record; first_clone boolean;
begin
  for rec in select r.* from registre r where r.tenant_code is null loop
    first_clone := true;
    for target in select t.code,t.institution_id from app_tenants t where t.active order by t.code
    loop
      if first_clone then
        clone_id := rec.id;
        update registre set tenant_code=target.code,institution_id=target.institution_id where id=rec.id;
        first_clone := false;
      else
        insert into registre(nume,prefix_nr,nr_inceput,nr_curent,nr_urmator,data_resetare,tip_registru,is_default,created_at,updated_at,tenant_code,institution_id,visibility,active)
        values(rec.nume,rec.prefix_nr,rec.nr_inceput,rec.nr_curent,rec.nr_urmator,rec.data_resetare,rec.tip_registru,rec.is_default,rec.created_at,rec.updated_at,target.code,target.institution_id,rec.visibility,rec.active) returning id into clone_id;
      end if;
      update registratura_documents set registru_id=clone_id where registru_id=rec.id and institution_id=target.institution_id;
    end loop;
    if first_clone then raise exception '0072 requires at least one active tenant'; end if;
  end loop;
end $$;
alter table registre alter column tenant_code set not null;
alter table registre alter column institution_id set not null;
alter table registre add constraint registre_tenant_fk foreign key (tenant_code) references app_tenants(code) on delete restrict;
alter table registre add constraint registre_tenant_institution_fk foreign key (tenant_code, institution_id) references app_tenants(code, institution_id) on delete restrict;
alter table registre add constraint registre_tenant_id_unique unique (tenant_code, id);
with ranked as (select id,row_number() over(partition by tenant_code order by is_default desc,id) as rn from registre where active) update registre r set is_default=(ranked.rn=1) from ranked where r.id=ranked.id;
create unique index if not exists uq_registre_tenant_name on registre(tenant_code, lower(nume));
create unique index if not exists uq_registre_tenant_default on registre(tenant_code) where is_default and active;
create index if not exists idx_registre_institution_visibility on registre(institution_id, visibility, active);
alter table registratura_documents add constraint registratura_documents_registry_tenant_fk foreign key (tenant_code,registru_id) references registre(tenant_code,id) on delete restrict;

create table if not exists registratura_departments (
    id uuid primary key default gen_random_uuid(),
    tenant_code text not null references app_tenants(code) on delete restrict,
    institution_id text not null,
    name text not null,
    description text not null default '',
    parent_id uuid references registratura_departments(id) on delete restrict,
    role_tag text not null default '',
    active boolean not null default true,
    created_at timestamptz not null default now(),
    updated_at timestamptz not null default now(),
    check (parent_id is null or parent_id <> id)
);

create table if not exists registratura_organizations (
    id uuid primary key default gen_random_uuid(),
    tenant_code text not null references app_tenants(code) on delete restrict,
    institution_id text not null,
    name text not null,
    description text not null default '',
    active boolean not null default true,
    is_default boolean not null default false,
    created_at timestamptz not null default now(),
    updated_at timestamptz not null default now()
);
create unique index if not exists uq_registratura_departments_tenant_name on registratura_departments(tenant_code, lower(name));
create unique index if not exists uq_registratura_organizations_tenant_name on registratura_organizations(tenant_code, lower(name));
alter table registratura_departments add constraint registratura_departments_tenant_id_unique unique (tenant_code,id);
alter table registratura_organizations add constraint registratura_organizations_tenant_id_unique unique (tenant_code,id);
create unique index if not exists uq_registratura_organizations_default on registratura_organizations(tenant_code) where is_default and active;

create table if not exists registratura_user_departments (
    tenant_code text not null references app_tenants(code) on delete restrict,
    institution_id text not null,
    user_id uuid not null references app_users(id) on delete restrict,
    department_id uuid not null references registratura_departments(id) on delete restrict,
    is_primary boolean not null default false,
    created_at timestamptz not null default now(),
    primary key (tenant_code, user_id, department_id)
);
create unique index if not exists uq_registratura_user_primary_department on registratura_user_departments(tenant_code, user_id) where is_primary;

create table if not exists registratura_user_organizations (
    tenant_code text not null references app_tenants(code) on delete restrict,
    institution_id text not null,
    user_id uuid not null references app_users(id) on delete restrict,
    organization_id uuid not null references registratura_organizations(id) on delete restrict,
    created_at timestamptz not null default now(),
    primary key (tenant_code, user_id)
);

alter table registratura_departments add constraint registratura_departments_tenant_institution_fk foreign key (tenant_code,institution_id) references app_tenants(code,institution_id) on delete restrict;
alter table registratura_organizations add constraint registratura_organizations_tenant_institution_fk foreign key (tenant_code,institution_id) references app_tenants(code,institution_id) on delete restrict;
alter table registratura_user_departments add constraint registratura_user_departments_department_tenant_fk foreign key (tenant_code,department_id) references registratura_departments(tenant_code,id) on delete restrict;
alter table registratura_user_organizations add constraint registratura_user_organizations_organization_tenant_fk foreign key (tenant_code,organization_id) references registratura_organizations(tenant_code,id) on delete restrict;

create table if not exists registratura_registry_departments (
    tenant_code text not null references app_tenants(code) on delete restrict,
    institution_id text not null,
    registry_id bigint not null references registre(id) on delete cascade,
    department_id uuid not null references registratura_departments(id) on delete restrict,
    primary key (tenant_code, registry_id, department_id)
);

create table if not exists registratura_organization_departments (
    tenant_code text not null references app_tenants(code) on delete restrict,
    institution_id text not null,
    organization_id uuid not null references registratura_organizations(id) on delete cascade,
    department_id uuid not null references registratura_departments(id) on delete restrict,
    primary key (tenant_code, organization_id, department_id)
);
alter table registratura_registry_departments add constraint registratura_registry_departments_registry_tenant_fk foreign key (tenant_code,registry_id) references registre(tenant_code,id) on delete cascade;
alter table registratura_registry_departments add constraint registratura_registry_departments_department_tenant_fk foreign key (tenant_code,department_id) references registratura_departments(tenant_code,id) on delete restrict;
alter table registratura_organization_departments add constraint registratura_organization_departments_organization_tenant_fk foreign key (tenant_code,organization_id) references registratura_organizations(tenant_code,id) on delete cascade;
alter table registratura_organization_departments add constraint registratura_organization_departments_department_tenant_fk foreign key (tenant_code,department_id) references registratura_departments(tenant_code,id) on delete restrict;

alter table registratura_documents add column if not exists external_number text not null default '';
alter table registratura_documents add column if not exists external_number_date date;
alter table registratura_documents add column if not exists entry_at timestamptz;
alter table registratura_documents add column if not exists exit_at timestamptz;
alter table registratura_documents add column if not exists activity text not null default '';
alter table registratura_documents add column if not exists record_kind text not null default 'document'
    check (record_kind in ('document', 'dosar'));
alter table registratura_documents add column if not exists cancelled_at timestamptz;
alter table registratura_documents add column if not exists cancelled_by text not null default '';
alter table registratura_documents add column if not exists cancellation_reason text not null default '';
alter table registratura_documents add column if not exists workflow_version integer not null default 1;
alter table registratura_documents drop constraint if exists registratura_documents_status_check;
alter table registratura_documents add constraint registratura_documents_status_check
    check (status in ('INCOMING', 'ALOCAT_COMPARTIMENT', 'IN_LUCRU', 'FLUX_APROBARE', 'FINALIZAT', 'ANULAT')) not valid;
update registratura_documents set status = case status
    when 'archived' then 'FINALIZAT'
    when 'in_workflow' then 'IN_LUCRU'
    when 'registered' then 'INCOMING'
    else 'INCOMING' end
where status not in ('INCOMING','ALOCAT_COMPARTIMENT','IN_LUCRU','FLUX_APROBARE','FINALIZAT','ANULAT');
alter table registratura_documents validate constraint registratura_documents_status_check;
update registratura_documents set entry_at = registered_at where entry_at is null and direction = 'intrare';
update registratura_documents set exit_at = registered_at where exit_at is null and direction = 'iesire';

create table if not exists registratura_document_departments (
    tenant_code text not null references app_tenants(code) on delete restrict,
    institution_id text not null,
    document_id uuid not null references registratura_documents(id) on delete restrict,
    department_id uuid not null references registratura_departments(id) on delete restrict,
    assigned_at timestamptz not null default now(),
    assigned_by text not null default '',
    primary key (tenant_code, document_id, department_id)
);

create table if not exists registratura_document_workflow_events (
    id uuid primary key default gen_random_uuid(),
    tenant_code text not null references app_tenants(code) on delete restrict,
    institution_id text not null,
    document_id uuid not null references registratura_documents(id) on delete restrict,
    action text not null check (action in ('assign_department','assign_user','claim','send_for_approval','approve','reject','cancel')),
    from_status text not null,
    to_status text not null,
    department_id uuid references registratura_departments(id) on delete restrict,
    assigned_user_id uuid references app_users(id) on delete restrict,
    note text not null default '',
    actor_subject text not null,
    created_at timestamptz not null default now()
);
alter table registratura_document_departments add constraint registratura_document_departments_document_tenant_fk foreign key (tenant_code,document_id) references registratura_documents(tenant_code,id) on delete restrict;
alter table registratura_document_departments add constraint registratura_document_departments_department_tenant_fk foreign key (tenant_code,department_id) references registratura_departments(tenant_code,id) on delete restrict;
alter table registratura_document_workflow_events add constraint registratura_workflow_events_document_tenant_fk foreign key (tenant_code,document_id) references registratura_documents(tenant_code,id) on delete restrict;
alter table registratura_document_workflow_events add constraint registratura_workflow_events_department_tenant_fk foreign key (tenant_code,department_id) references registratura_departments(tenant_code,id) on delete restrict;
create index if not exists idx_registratura_workflow_events_document on registratura_document_workflow_events(institution_id, document_id, created_at desc);

alter table registratura_document_attachments add column if not exists tenant_code text;
alter table registratura_document_attachments add column if not exists institution_id text;
alter table registratura_document_attachments add column if not exists checksum_sha256 text not null default '';
alter table registratura_document_attachments add column if not exists scan_status text not null default 'pending'
    check (scan_status in ('pending','clean','infected','failed'));
alter table registratura_document_attachments add column if not exists storage_state text not null default 'staged'
    check (storage_state in ('staged','ready','deleted'));
update registratura_document_attachments a set institution_id = d.institution_id
from registratura_documents d where a.document_id = d.id and a.institution_id is null;
update registratura_document_attachments a set tenant_code = t.code
from app_tenants t where t.institution_id = a.institution_id and a.tenant_code is null;
alter table registratura_document_attachments alter column institution_id set not null;
alter table registratura_document_attachments alter column tenant_code set not null;
alter table registratura_document_attachments alter column tenant_code set default public.current_tenant_code();
alter table registratura_document_attachments alter column institution_id set default public.current_institution_id();
alter table registratura_document_attachments add constraint registratura_attachments_document_tenant_fk foreign key (tenant_code,document_id) references registratura_documents(tenant_code,id) on delete cascade;

alter table app_parties add column if not exists birth_date date;
alter table app_parties add column if not exists birth_place text not null default '';
alter table app_parties add column if not exists trade_register_no text not null default '';
alter table app_parties add column if not exists legal_representative text not null default '';
alter table app_parties add column if not exists legal_form text not null default '';
alter table app_parties add column if not exists share_capital numeric(18,2);
alter table app_parties add column if not exists institution_type text not null default '';
alter table app_parties add column if not exists institution_level text not null default '';
alter table app_parties add column if not exists website text not null default '';

-- All new and upgraded registratura tables are institution-isolated through the
-- request-scoped PostgreSQL setting.  Tenant_code is retained for uniqueness and auditing.
do $$ declare tbl text; begin
  foreach tbl in array array['registre','registratura_departments','registratura_organizations','registratura_user_departments','registratura_user_organizations','registratura_registry_departments','registratura_organization_departments','registratura_document_departments','registratura_document_workflow_events','registratura_document_versions','registratura_document_attachments'] loop
    execute format('alter table %I enable row level security', tbl);
    execute format('alter table %I force row level security', tbl);
    execute format('drop policy if exists tenant_isolation on %I', tbl);
    execute format('create policy tenant_isolation on %I using (public.can_bypass_tenant_rls() or institution_id = public.current_institution_id()) with check (public.can_bypass_tenant_rls() or institution_id = public.current_institution_id())', tbl);
  end loop;
end $$;

create or replace function public.registratura_department_parent_guard()
returns trigger language plpgsql as $$
begin
  if new.parent_id is null then return new; end if;
  if exists (
    with recursive ancestors as (
      select id, parent_id from registratura_departments where id = new.parent_id
      union all
      select d.id, d.parent_id from registratura_departments d join ancestors a on d.id = a.parent_id
    ) select 1 from ancestors where id = new.id
  ) then raise exception 'department hierarchy cycle'; end if;
  if not exists (select 1 from registratura_departments where id = new.parent_id and tenant_code = new.tenant_code and institution_id = new.institution_id) then
    raise exception 'department parent must belong to same tenant';
  end if;
  return new;
end $$;
drop trigger if exists trg_registratura_department_parent_guard on registratura_departments;
create trigger trg_registratura_department_parent_guard before insert or update of parent_id, tenant_code, institution_id on registratura_departments for each row execute function public.registratura_department_parent_guard();

-- Replace legacy UI filter codes with the document-workflow canonical enum.
delete from app_nomenclatures where domain='registratura_status';
insert into app_nomenclatures(domain,code,label_ro,label_en,active,sort_order) values
 ('registratura_status','INCOMING','Înregistrat','Incoming',true,10),
 ('registratura_status','ALOCAT_COMPARTIMENT','Alocat compartiment','Assigned to department',true,20),
 ('registratura_status','IN_LUCRU','În lucru','In progress',true,30),
 ('registratura_status','FLUX_APROBARE','Flux aprobare','Approval flow',true,40),
 ('registratura_status','FINALIZAT','Finalizat','Finalized',true,50),
 ('registratura_status','ANULAT','Anulat','Cancelled',true,60);
