# Admission and archive WORM ingestion: required implementation

Status: implementation in progress; the existing legal archive browser proof fails. Bucket Object Lock alone does not make `UploadDocument` immutable. No production release is authorized by passing storage unit tests alone.

## Invariants

- Tenant, institution and actor are resolved from host/OIDC/DB session, never from upload metadata supplied by the browser.
- Retention is derived from an approved, effective legal rule. The admission DSS rule is not an authority for unrelated archive series.
- A prepared decision or appeal snapshots the retention policy/rule, minimum days, preparation expiry and required deadline. Its deadline is preparation expiry plus the applicable minimum period: every permitted finalization occurs before expiry. Finalization must reject policy changes that invalidate that bound; no arbitrary grace period is introduced.
- Existing preparations without a genuine policy snapshot cannot be finalized under a fabricated snapshot. They must be cancelled/re-prepared through the normal lifecycle, releasing capacity.
- General archive series need their own versioned, approved source-backed retention policy and a defined retention anchor. No fallback number of years or implicit taxonomy creation supplies legal authority.
- Browser finalization references only server-persisted archive IDs. The legal version must come from an ingestion intent bound to the same preparation/artifact, not merely any WORM document in the tenant.

## Database-to-storage transaction boundary

1. Reserve an ingestion intent in PostgreSQL before writing to MinIO. Persist immutable scope, actor, idempotency key, request fingerprint, purpose, preparation/series reference, policy snapshot, expected SHA/size, reserved document/version IDs and deterministic object key.
2. Write a new object conditionally (`If-None-Match: *`) with COMPLIANCE retention and server-owned intent/scope metadata. Never overwrite or delete an existing WORM version to make a retry succeed.
3. HEAD the exact returned version. Validate version ID, size, lock mode, retention deadline, requested legal hold and metadata. Return observed values for persistence. A post-PUT error must retain the known object identity for reconciliation.
4. Lock the intent in a new transaction. Adopt the verified source into the archive document/version, including exact S3 version, ETag and observed retention. Commit archive job, audit and intent completion atomically.
5. On failure after PUT, retain the intent and reconcile. Verify the deterministic object against its intent before adoption; no guessed version or metadata and no untracked compensating delete. Replays after response loss return the existing resource; changed request fingerprints return 409.

## Required schema and contracts

- Preparation fields: retention policy/rule IDs, anchor, minimum days, required deadline, immutable after creation.
- `archive_ingestion_intents`: forced RLS, composite scope FKs, unique scoped idempotency key, immutable request/provenance, controlled reserved/stored/committed/failure transitions.
- Archive version references its intent; exact object identity cannot be assigned to unrelated versions.
- Source-backed archive-series retention rules have distinct proposal/approval permissions and provenance.
- Multipart legal upload is preparation-bound with operation-specific admission permission plus archive access. General uploads select a governed series, not a retention deadline.
- OpenAPI declares each input, status and replay/error outcome; generated React client and response validators are used by the UI. The UI displays server-derived retention and recovery state, never an editable legal deadline.

## Acceptance evidence

- Real PostgreSQL migration, lifecycle, forced-RLS and cross-tenant/institution rejection tests.
- Real MinIO exact-version/COMPLIANCE/retention/hold verification, not only an HTTP emulator.
- Fault injection after PUT and around DB commit, response loss, concurrent retries and fingerprint conflicts, without duplicate adopted versions.
- Preparation-expiry and policy-tightening tests; no moving `now + days` threshold that invalidates a correct upload seconds later.
- Browser prepare, download exact canonical payload, signed upload, OCR/readiness, DSS validation and finalize; favorable appeals use two independently bound signed artifacts.
- Positive and negative cryptographic fixtures cover actor/certificate authorization, timestamp coverage, document and payload tampering, trust-list freshness and revocation.

Storage-boundary implementation alone does not satisfy these gates. The whole original School objective remains active.

## General archive authority: follow-up boundary

The general upload gap must not be repaired by copying the admission DSS retention rule into every archive series. Article 8 of [Law 16/1996](https://legislatie.just.ro/Public/DetaliiDocument/284305) ties document grouping and retention to the creator's archive nomenclature. The legal text was located through the official portal on 2026-09-12; direct page retrieval returned HTTP 502, so this lookup is not a completed consolidated-law audit.

Implementation requirements derived for this boundary (engineering design, not a claim of statutory certification):

- A series classification and an arbitrary `retention_years` value are not proof of an approved retention authority. Preserve the source, applicable institutional scope, approval evidence, effective version and retention anchor separately.
- Internal RBAC approval must not be represented as external archival approval or accreditation. Record the latter's evidence where applicable.
- The initial policy/source registration must not depend on an already finalized admission decision. Do not manufacture a legally finalized preparation solely to upload unrelated supporting evidence.
- Bootstrap tests must create the governed series through its published workflow, then upload application/appeal evidence under that authority. Preparation-bound decision and resolution uploads remain separate.
- Unknown or event-based retention anchors cannot silently become the upload date. Do not invent fixed retention periods to unblock a browser test.

The follow-up code audit confirmed that `school_regulatory_sources` has no archive FK and admission rule approval does not require a WORM source. The browser fixture currently creates its source through offering authorization, then prematurely waits for a generic upload to become WORM. There is no schema-level circular dependency. The actual missing workflows are dedicated source registration/verification and governed archive-series retention/upload. Source registration must not require an unrelated offering-authorization mutation.

Implementation order: standalone regulatory source lifecycle; versioned series authority with distinct proposal/approval permissions; a new migration extending intents with mutually exclusive preparation/series bindings; generic upload using the durable WORM saga; generated API and administrative UI; real generic-upload/RLS tests and rerun of the unchanged evidence requirements in the admission browser workflow.

### Source provenance blocker discovered during implementation

The current `institution/service.go` profile mutation and `institution/offerings.go` authorization mutation accept the browser's `checksum_sha256`, then create active source rows with `verified_at`/`verified_by_subject`. The 0137 constraint validates field presence and digest syntax, not that downloaded bytes were hashed. Consequently, an existing `active`/`verified_at` source is only a legacy metadata assertion; the new series-rule configuration must not turn that into a claim of verified legal content.

Required correction before connecting the general WORM upload to this authority:

1. Standalone source draft registration, without writable verification timestamps or computed hashes in its request DTO.
2. Server-side bounded evidence hashing with immutable, scoped verification records. For URL verification, defend against SSRF, DNS rebinding, private/link-local destinations and unsafe redirects. Operator-upload evidence must be explicitly distinguished from independently retrieved publisher content.
3. Activation bound to a successful verification for the unchanged source/version. Rechecking only the presence of legacy verification fields is insufficient.
4. Existing profile/offering commands select an activated source rather than self-certifying browser metadata. Legacy rows require a genuine verification workflow; no migration may synthesize evidence.
5. Tests reject activation without evidence, cross-scope evidence reuse and changes to activated provenance. Real-stack fixtures create sources through the same new public workflow, never privileged inserts masquerading as user behavior.

Event-triggered and permanent retention remain required by the full School scope. Initial intake-anchored policy configuration is an intermediate implementation only; unsupported anchors must stay explicit and fail closed until their corresponding lifecycle is implemented.
