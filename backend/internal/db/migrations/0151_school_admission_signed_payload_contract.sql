-- Bind every admission legal artifact to the exact canonical payload and
-- authenticated application subject carried by the validated signature.
-- Defaults preserve upgradeability for pre-existing non-admission evidence;
-- new admission bindings require non-empty, matching values in the guard.
alter table education_signed_artifact_evidence
    add column if not exists expected_canonical_legal_payload_sha256 text not null default '',
    add column if not exists expected_actor_subject text not null default '';

alter table education_signed_artifact_evidence
    drop constraint if exists education_signed_artifact_evidence_expected_payload_sha256_check,
    add constraint education_signed_artifact_evidence_expected_payload_sha256_check
        check (expected_canonical_legal_payload_sha256 = '' or expected_canonical_legal_payload_sha256 ~ '^[0-9a-f]{64}$');

alter table education_signed_artifact_validations
    add column if not exists signed_payload_sha256 text not null default '',
    add column if not exists certificate_sha256 text not null default '',
    add column if not exists signed_actor_subject text not null default '';

alter table education_signed_artifact_validations
    drop constraint if exists education_signed_artifact_validations_signed_payload_sha256_check,
    drop constraint if exists education_signed_artifact_validations_certificate_sha256_check,
    add constraint education_signed_artifact_validations_signed_payload_sha256_check
        check (signed_payload_sha256 = '' or signed_payload_sha256 ~ '^[0-9a-f]{64}$'),
    add constraint education_signed_artifact_validations_certificate_sha256_check
        check (certificate_sha256 = '' or certificate_sha256 ~ '^[0-9a-f]{64}$');

alter table school_admission_signed_artifact_bindings
    add column if not exists canonical_legal_payload_sha256 text not null default '',
    add column if not exists expected_actor_subject text not null default '';

alter table school_admission_signed_artifact_bindings
    drop constraint if exists school_admission_binding_canonical_payload_sha256_check,
    add constraint school_admission_binding_canonical_payload_sha256_check
        check (canonical_legal_payload_sha256 ~ '^[0-9a-f]{64}$'),
    drop constraint if exists school_admission_binding_expected_actor_subject_check,
    add constraint school_admission_binding_expected_actor_subject_check
        check (btrim(expected_actor_subject) <> '');

create or replace function public.school_admission_dss_binding_guard() returns trigger language plpgsql security definer set search_path=pg_catalog,public as $$
declare actor text:=nullif(btrim(current_setting('app.actor_subject',true)), ''); ok boolean;
begin
 if tg_op in ('UPDATE','DELETE') then raise exception 'admission DSS binding is append-only'; end if;
 if actor is null then raise exception 'admission DSS binding requires authenticated actor provenance'; end if;
 new.bound_by_subject:=actor;
 if not public.can_bypass_tenant_rls() and (new.tenant_code<>public.current_tenant_code() or new.institution_id<>public.current_institution_id()) then raise exception 'admission DSS binding tenant/institution context mismatch'; end if;
 select exists(
  select 1
  from education_signed_artifact_evidence e
  join education_signed_artifact_validations v
    on v.tenant_code=e.tenant_code and v.institution_id=e.institution_id
   and v.evidence_id=e.id and v.id=new.validation_id
  where e.tenant_code=new.tenant_code and e.institution_id=new.institution_id
    and e.id=new.evidence_id and e.artifact_type=new.artifact_kind and e.artifact_id=new.artifact_id
    and e.document_sha256=new.archive_sha256
    and e.expected_canonical_legal_payload_sha256=new.canonical_legal_payload_sha256
    and e.expected_actor_subject=new.expected_actor_subject
    and e.storage_document_id=new.archive_document_id and e.storage_version_id=new.archive_version_id
    and e.storage_bucket=new.archive_source_bucket and e.storage_object_key=new.archive_source_object_key
    and e.signature_format='PAdES'
    and v.validation_status='valid' and btrim(v.trusted_list_provider)<>''
    and btrim(v.validator_provider)<>'' and btrim(v.validator_version)<>'' and btrim(v.validation_policy)<>''
    and v.observed_sha256=new.archive_sha256 and v.observed_size_bytes=new.archive_size_bytes
    and v.signed_payload_sha256=new.canonical_legal_payload_sha256
    and v.certificate_sha256~'^[0-9a-f]{64}$'
    and v.signed_actor_subject=new.expected_actor_subject
    and v.timestamp_token_sha256~'^[0-9a-f]{64}$' and v.timestamp_at is not null and btrim(v.timestamp_authority)<>''
    and v.diagnostic_data<>'{}'::jsonb and v.detailed_report<>'{}'::jsonb
    and v.simple_report<>'{}'::jsonb and v.etsi_validation_report<>'{}'::jsonb
    and e.submitted_by_subject=actor and v.validated_by_subject=actor
    and new.expected_actor_subject=actor
 ) into ok;
 if not ok then raise exception 'admission DSS binding must reference the exact canonical payload, signer, PAdES/DSS evidence and actor'; end if;
 return new;
end $$;
revoke all on function public.school_admission_dss_binding_guard() from public;
