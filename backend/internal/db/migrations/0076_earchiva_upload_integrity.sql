-- Tenant-pair keys make an accidental privileged cross-tenant reference fail
-- at the database boundary, independently of RLS policy evaluation.

alter table archive_documents add column if not exists idempotency_key text not null default '';

alter table archive_taxonomy_nodes add constraint archive_taxonomy_nodes_institution_id_id_key unique (institution_id, id);
alter table archive_documents add constraint archive_documents_institution_id_id_key unique (institution_id, id);
alter table archive_document_versions add constraint archive_document_versions_institution_id_id_key unique (institution_id, id);
alter table archive_document_chunks add constraint archive_document_chunks_institution_id_id_key unique (institution_id, id);
alter table archive_document_entities add constraint archive_document_entities_institution_id_id_key unique (institution_id, id);
alter table archive_document_relations add constraint archive_document_relations_institution_id_id_key unique (institution_id, id);
alter table archive_ingestion_jobs add constraint archive_ingestion_jobs_institution_id_id_key unique (institution_id, id);

alter table archive_taxonomy_nodes add constraint archive_taxonomy_parent_tenant_fk
    foreign key (institution_id, parent_id) references archive_taxonomy_nodes (institution_id, id) on delete cascade;
alter table archive_documents add constraint archive_documents_taxonomy_tenant_fk
    foreign key (institution_id, taxonomy_node_id) references archive_taxonomy_nodes (institution_id, id) on delete set null (taxonomy_node_id);
alter table archive_document_versions add constraint archive_versions_document_tenant_fk
    foreign key (institution_id, document_id) references archive_documents (institution_id, id) on delete cascade;
alter table archive_document_chunks add constraint archive_chunks_version_tenant_fk
    foreign key (institution_id, version_id) references archive_document_versions (institution_id, id) on delete cascade;
alter table archive_document_entities add constraint archive_entities_document_tenant_fk
    foreign key (institution_id, document_id) references archive_documents (institution_id, id) on delete cascade;
alter table archive_document_entities add constraint archive_entities_version_tenant_fk
    foreign key (institution_id, version_id) references archive_document_versions (institution_id, id) on delete cascade;
alter table archive_document_relations add constraint archive_relations_source_tenant_fk
    foreign key (institution_id, source_document_id) references archive_documents (institution_id, id) on delete cascade;
alter table archive_document_relations add constraint archive_relations_target_tenant_fk
    foreign key (institution_id, target_document_id) references archive_documents (institution_id, id) on delete cascade;
alter table archive_ingestion_jobs add constraint archive_jobs_document_tenant_fk
    foreign key (institution_id, document_id) references archive_documents (institution_id, id) on delete set null (document_id);
alter table archive_ingestion_jobs add constraint archive_jobs_version_tenant_fk
    foreign key (institution_id, version_id) references archive_document_versions (institution_id, id) on delete set null (version_id);

create unique index archive_documents_institution_idempotency_key
    on archive_documents (institution_id, idempotency_key)
    where idempotency_key <> '';

-- Viewing original bitstreams is intentionally separate from searching their
-- metadata/OCR text. Existing archive operational roles receive the explicit
-- capability, while tenant-specific direct grants remain unchanged.
insert into app_permissions (code, label) values
    ('earchiva.content.read', 'Read original eArchive document content')
on conflict (code) do update set label = excluded.label;

insert into app_role_permissions (role_code, permission_code)
select code, 'earchiva.content.read'
from app_roles
where code in ('super_admin', 'admin', 'arhivar')
on conflict do nothing;
