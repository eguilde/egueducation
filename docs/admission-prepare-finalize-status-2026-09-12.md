# Admission prepare/finalize — verification status, 2026-09-12

Status: implemented locally, not released and not end-to-end certified.

Current verdict: NOT COMPLETE. The last real browser test fails because ordinary upload does not produce a WORM archive version. The Go negative-verdict contract and Java timestamp/qualification corrections have passing unit coverage (six Java tests); real cryptographic integration is still missing. The immutable storage boundary is being hardened, but its integration into upload, durable recovery, approved retention selection and the generated frontend contract remains required. Nothing in this report claims the new changes are online.

Storage boundary progress: `PutImmutableObject` now accepts an explicit server-derived write request, conditionally creates with `If-None-Match: *`, then HEADs the returned version and verifies observed version, ETag, size, COMPLIANCE mode, retention, requested hold and metadata. Post-write failures retain a typed object identity for reconciliation, with no compensating deletion. Executable S3 HTTP-protocol tests pass for success, provenance mismatches, conditional conflict and duplicate normalized metadata. This is not a real MinIO integration run and the ordinary upload path still does not call this method.

Expanded storage regressions also pass for HEAD failure after PUT, missing PUT version identity, ETag mismatch and missing requested legal hold. The root independently reran all six top-level immutable-storage tests and their mismatch/failure subcases (package 3.188s). These prove the protocol boundary and recoverable errors, not the still-missing durable ingestion implementation.

## Latest preparation-retention verification

Migration 0154 adds immutable, tenant/institution-scoped snapshots of the approved retention policy, rule, source, minimum days, expiry anchor and required deadline. New preparations require a complete matching snapshot; existing history is not backfilled with invented authority. The existing shared DSS binding retention guard remains unchanged. Decision and appeal finalizers additionally check the preparation snapshot, including both favorable-appeal artifacts.

Both preparation responses now expose these server-derived fields through OpenAPI and generated runtime validators. React uses the generated preparation response type and displays the retention requirement as read-only information. Timestamps are serialized as UTC RFC3339; deadline arithmetic in the database explicitly uses UTC.

Latest observed checks against disposable PostgreSQL 17:

- `go test -tags=integration ./internal/db ./internal/education ./internal/admission -count=1 -timeout 600s`: all three packages passed (114.781s, 118.600s, 22.790s respectively).
- After the final UTC migration adjustment, `go test ./internal/admission -run '^TestAdmissionPreparationRetention' -count=1 -timeout 180s`: passed (10.258s).
- Snapshot regressions reject partial/null snapshots, incorrect provenance/deadlines and mutation. The response round-trip test reads real stored preparations in UTC, Europe/Bucharest and Pacific/Auckland, verifies RFC3339 serialization and rejects an archive deadline one microsecond short.
- OpenAPI generation/validation passed for 617 operations, including 430 SchoolAdmission operations. Generated frontend contracts, 14 focused admission tests, API/theme policy checks and production React build passed. CI includes the new PostgreSQL tests and its YAML parses successfully.

Limits: no genuine pre-0154 legacy-upgrade fixture, separately seeded future-effective authority test or cross-DST insertion test has been executed. The cancellation test uses a newly valid snapshot, not legacy history. These checks do not establish a passing browser finalization or real cryptographic verification. Durable ingestion intents and the ordinary-upload-to-WORM integration remain unimplemented. No push or deployment has been performed for this change set.

## Implemented scope

- Decision preparation and appeal-resolution preparation, finalization and cancellation.
- Favorable appeals prepare separate resolution and replacement-decision artifacts.
- Exact canonical payload bytes persisted independently of JSONB and transported as Base64 for downloads.
- Separate WORM archive references for favorable appeal artifacts.
- Capacity reservation/release and legal persistence in the admission transaction.
- Certificate authorization proposal, separate approval, revocation with reason/version, and live RBAC checks.
- Eight new endpoints registered in the actual server entry point, not only the module helper router.
- OpenAPI request/response mappings, generated React client operations and runtime response validation.
- Separate frontend signer-management and signer-approval capabilities.

## Observed validation

- `go test ./... -count=1`: passed. PostgreSQL-dependent tests skip without a database URL.
- OpenAPI validation: 617 actual server routes, including assertions for Base64 signing payloads, conditional replacement archive and idempotent cancellation/finalization.
- React full suite before the final regression test: 48 files, 277 tests passed.
- Final focused admission suite: 14/14 tests passed, including both WORM selectors for a favorable appeal.
- TypeScript check and production React build passed after correcting wizard state and request typing.
- UI/theme policy and API contract policy checks passed.
- Production build still warns about large application/runtime-validator chunks.
- Real PostgreSQL 17 verification now uses a temporary, deny-all-network namespace with no persistent volumes or application credentials. The admission integration, Stage 1B/Stage 2 contracts, policy-v2 upgrade, regulatory Stage 1B, and post-0060 portfolio RLS tests have passed individually.
- Real execution found and led to fixes for the missing scoped compliance-obligation FK key (0136), reserved SQL alias (0138), shared-trigger references to nonexistent NEW fields (0148), and a capacity guard blocking authorization changes without any admission commitments (0149).
- Full database integration also reproduced a signed-artifact provenance regression introduced by 0149; migration 0153 reconciles the original canonical-source and active-state guards with admission support. Real regressions now verify browser provenance replacement and rejection of inactive documents, versions, decisions, publications and managerial documents.
- Final command with a real PostgreSQL 17 DSN: `go test -tags=integration ./internal/db ./internal/education -count=1 -timeout 600s` passed both packages (`internal/db` 94.081s; `internal/education` 102.917s). This includes the new 0152 tests for mismatched canonical bytes/hash, immutable cancellation provenance and expired signer approval. It is not an HTTP/browser/DSS cryptography proof, nor a complete concurrency/hold-expiry proof.
- CI explicitly runs `TestSchoolAdmissionLegalGuardsPostgresIntegration`.
- The admission browser proof has now been executed against three real Go/OIDC instances, React, PostgreSQL 17 with a non-BYPASSRLS application role and disposable MinIO. OCR/antivirus/signature services in this test are contract emulators, not production trust validation. OTP entry, the OIDC token exchange, authenticated UI and the initial permission assertions passed. The run then failed with HTTP 500 on offering-authorization creation: PostgreSQL rejected the reserved SQL alias `authorization` in the read-back query. That alias has been corrected in list/filter/sort/read-back queries; the complete browser workflow still requires a passing rerun.
- Local runner OCR configuration was completed; the CI system job now supplies OCR configuration, requires object lock and builds its PDF validator. Its isolated MinIO image uses the official Quay registry because the equivalent Docker Hub tag failed to pull in the test cluster.
- The new Java DSS adapter compiles against DSS 6.5 and its Maven report records 3 safety/unit tests passing. No positive qualified-signature validation has run. Independent review found concrete Go/Java request-field incompatibilities, missing timestamp evidence and incorrect signature-level mapping; these are implementation blockers, not merely unexecuted tests. Further findings include trust-list freshness, protected outbound fetching, bounded memory, report retention/scoping and exact signed-byte coverage.
- The rerun passed offering-authorization creation and processed an uploaded PDF to `ready`. It failed the legal archive selector timeout because the real upload path always invokes mutable `PutObject` and omits object-version and retention fields from the version INSERT. `PutImmutableObject` currently has no callers. Object Lock bucket verification does not supply per-object retention. This is a production-path gap: the selector must remain strict, and upload needs an authorized server-derived retention policy, immutable write, exact returned provenance persistence and safe orphan reconciliation. The browser proof remains FAILED, not complete.

## Remaining release gates

Additional verified correction: expired preparation cleanup now drains and closes `UPDATE ... RETURNING` rows before updating held allocations on the same pgx transaction. `TestExpireAdmissionPreparationsTxReleasesExpiredHoldPostgres` passed independently against a newly migrated disposable PostgreSQL database (package 4.389s), proving the preparation becomes `expired` and its allocation `released`. This does not establish concurrent finalization/race coverage.

Follow-up DSS patch: the latest Maven report records 4 tests passing, including the Go evidence JSON fixture. Request-field acceptance, signature qualification mapping, actual timestamp extraction, exact ByteRange end, bounded object reads and scoped report retention were corrected locally. These changes still require independent review and a real cryptographic positive/negative integration suite. They do not resolve the archive upload blocker above. The disposable test namespace was deleted after the test runs and the two local port forwards were stopped; application data was untouched.

Independent follow-up review confirms the valid-response transport fixes, but identifies remaining blockers: invalid/error responses omit a payload hash that Go currently requires unconditionally; timestamp selection uses all timestamp types rather than proving signature-timestamp coverage; qualification mapping accepts electronic seals for actor-bound decisions without an artifact-specific policy and omits `ADESIG_QC`. These issues must be resolved deliberately and tested with real cryptographic fixtures. No legal-signature readiness or completion is claimed.

Subsequent corrections: Go now preserves negative verifier verdicts (`invalid`, `indeterminate`, `error`) and their findings without requiring or propagating fabricated trust claims. Six response cases cover absent/present claims across those statuses; the valid-result validation remains strict. Admission binding tests explicitly reject all negative statuses even with otherwise complete proof fields. Focused remote-verifier tests and the education/admission package suites pass (database-dependent cases skip without a DSN in this run). Java now permits only actor-signature qualifications QESIG/ADESIG/ADESIG_QC and qualified signature timestamps of type SIGNATURE_TIMESTAMP covering the signature; six Java safety tests pass. These are unit/compile results, not a positive cryptographic integration proof. The concrete WORM implementation invariants and acceptance gates are recorded in `admission-worm-ingestion-implementation.md`.

1. Expand the real admission service/transaction tests to exercise reservation expiry/release, races, rollback and the complete finalized legal-artifact graph. The current full database and education integration suites pass against PostgreSQL 17. No application database was reset or changed.
2. Implement and validate the production EU DSS adapter. Emulator results do not prove cryptographic validation.
3. Remove remote verification from the transaction using a security-reviewed design. The optional prevalidated-result refactor was rejected by the safety reviewer because it could override configured verification; it was not applied. Current verification remains inside the transaction.
4. Complete real React → OIDC → API → PostgreSQL → MinIO → DSS tests, including two independent signed appeal artifacts and tampering failures.
5. Reconcile the worktree with main, then publish immutable images and verify GitOps rollout. No push or deployment was performed for this change set.

## Latest implementation checkpoint: preparation-bound WORM uploads

This checkpoint supersedes the earlier statement that `PutImmutableObject` has no callers. It does not resolve the separate ordinary archive-upload retention-policy gap.

- Migration 0155 adds durable, scoped ingestion intents, immutable preparation/slot provenance and archive-version adoption guards. The legal finalizer now requires a committed intent belonging to the exact preparation and artifact slot.
- New GET/POST artifact endpoints use server-derived retention snapshots and file-only multipart requests. React uses generated API contracts and runtime validators; decision and favorable-appeal artifacts have separate upload slots.
- Immutable recovery pins the S3 version and ETag, checks COMPLIANCE retention, legal hold, metadata, size and actual SHA-256. OCR source reads and original downloads use the recorded source version. Storage deadlines round upward to SDK millisecond precision without shortening the policy deadline.
- `TestAdmissionLegalArtifactUploadMinIOPostgres` passed against real disposable PostgreSQL 17 and MinIO (package 8.323s). Actual HTTP handlers verified upload, idempotent replay, conflicting bytes, readiness lookup and exact original download. A deliberately failing database adoption after a successful WORM PUT was retried successfully, adopting the same object version without another upload.
- This HTTP test injects authentication context and a clean scanner result. It is not an OIDC, antivirus, OCR, qualified-signature or non-BYPASSRLS proof.
- Real MinIO reconciliation and migrated PostgreSQL intent/binding tests passed separately. React's focused admission suite passed 17 tests across three files, including bounded polling, manual readiness recovery and abort-on-unmount.
- OpenAPI validation passed for 619 registered operations, including 397 School catalog and 39 Admission catalog operations. CI now explicitly runs the real WORM recovery and HTTP adoption tests; its full browser gate is retained.
- Final regression run with real PostgreSQL and MinIO passed all four packages: `internal/db` 94.132s, `internal/education` 102.499s, `internal/admission` 31.925s and `internal/earchiva` 31.263s (`go test -tags=integration ... -count=1 -timeout 600s`). These package results do not extend the individual tests' stated scope.
- Full React unit/component suite passed: 49 files, 281 tests. Production build, UI theme/component policy and API contract policy passed. CI YAML parsed successfully and `git diff --check` reported no whitespace errors. Warnings remain for large build chunks and the test environment's unconfigured PrimeUI license/localStorage; none is being represented as resolved.
- Full browser admission verification is still not passing: the generic archive-series retention-authority/upload path and the existing browser fixture must be completed. A positive cryptographic DSS integration proof and verified GitOps rollout are also outstanding.
- These changes are local and have not been pushed or deployed. The current disposable verification namespace and port forwards remain active while regression tests run; earlier cleanup statements refer to the previous verification run.

This report is not a declaration that the School module or legal signing workflows are complete.

## Follow-up verification: isolation and browser contract alignment

- Added `TestAdmissionWORMIngestionIntentNOBYPASSRLS` with independent foreign-tenant, foreign-institution and combined-scope cases. A real LOGIN role is asserted non-superuser and NOBYPASSRLS. Foreign reads return zero rows, foreign intent transitions affect zero rows, cross-scope INSERTs are rejected, and the original scope can read its intent. This is intent isolation, not proof of the entire archive-version adoption or HTTP authorization graph.
- Root reran the migrated intent test and new isolation test together against the disposable PostgreSQL instance: `go test -tags=integration ./internal/admission -run '^TestAdmissionWORMIngestionIntent(Postgres|NOBYPASSRLS)$' -count=1 -timeout 180s` passed (9.223s). CI now runs both explicitly.
- Updated the browser spec's obsolete decision selector to the actual preparation-bound FileUpload. Favorable appeal artifacts are uploaded by the resolving approver to separate preparation slots, not through another actor's generic archive upload. Typecheck and Playwright discovery passed; the full browser scenario has not passed or been rerun in this checkpoint.
- Removed an advertised HTTP 413 response that the actual legal-upload handler does not emit. OpenAPI generation/validation passed for 619 routes; React contracts were regenerated and typecheck/API contract policy passed.
- The source/retention audit found no archive FK on the regulatory source and therefore no schema-level circular bootstrap. Dedicated source registration and governed general archive-series retention/upload remain genuine missing workflows; see `admission-worm-ingestion-implementation.md` for the implementation boundary.
- No main push, production migration or deployment occurred in this checkpoint.
