# Retention implementation verification — 2026-09-12

The approved admission decision and appeal prepare/finalize work remains in progress. This checkpoint is not a completion or production deployment declaration.

## Verified in this continuation

- Removed unreachable duplicate retention UI scaffolding, retaining the complete prior source in `reference/archive-retention-ui-before-cleanup-2026-09-12.tsx.txt` (content compared before editing).
- Corrected React event typing, closed status-filter values and invalid Testing Library options.
- `npm run build`: PASS, including TypeScript compilation. Vite still warns about large chunks.
- Targeted retention UI and API tests: 2 files, 7 tests PASS.
- `npm run quality:ui`: PASS.
- `npm run quality:contracts`: PASS for 7 production adapters and generated validators.
- `go test -tags=integration ./internal/earchiva -run '^TestArchiveSeriesRetentionHTTPPostgresIntegration$' -count=1 -timeout 180s -v`: PASS, 5.559s, using disposable PostgreSQL on localhost:55438. This tests HTTP/database behavior with integration authentication, not real OIDC.
- Added this HTTP integration test to CI alongside the retention database lifecycle test.

## Remaining release gates

- Server-verified regulatory source provenance rather than trusting a browser-supplied checksum.
- Connect approved retention authority to general archive ingestion and durable WORM recovery.
- Extend retention beyond the currently supported intake date/minimum-days model where required.
- Demonstrate admission and appeal browser E2E with actual signed artifacts, certificate authorization and DSS validation; existing partial tests do not establish this.
- Review and reconcile the complete working tree before push directly to main, then verify CI and deployed revision. No push or deployment was performed in this continuation.

## Portfolio retention expiry disposition checkpoint

Migration 0165 now implements the fail-closed expired-retention review path.
The exact WORM version remains held until an owner request and a distinct,
dual-authorized custody/eArhiva decision have both been persisted. The durable
worker verifies VersionId, ETag, SHA-256, size, MIME, COMPLIANCE expiry and all
current portfolio/legal blockers under the shared release fence. It then
verifies hold OFF and atomically appends the receipt and DB projections.

Real PostgreSQL + MinIO tests pass for exact release, wrong provenance with zero
mutation, DB-level RBAC, missing worker context, fresh review after a blocker,
hold-already-OFF restart convergence, three cleared hold projections and one
receipt. React exposes the owner request and independent decision flow using
PrimeReact and generated OpenAPI types. The full React result is 62 files / 365
tests PASS; the OpenAPI catalogue is 640 routes / 406 School operations PASS.

The final real-stack browser proof after all fixes passes: **1 test in 12.4 minutes** across OIDC,
React, Go API, PostgreSQL and MinIO Object Lock. Before approval, the decision UI
now displays the immutable requester identity, evidence statement and document
reference. A focused recovery regression also covers the post-hold-off rollback
window: when a new legal/reference blocker appears before retry, the worker
restores and verifies the exact version's hold ON before recording a blocked
outcome. Release/deployment verification remains pending.
