# Regulatory source content verification contract

Status: fetch primitive and dedicated registration/verification/activation/evidence API, migration 0157 and React administration are implemented locally. Existing regulatory-profile/offering writes have not yet been migrated to activated source references. This is not a deployment or school-module completion claim.

## Evidence semantics

The checksum is SHA-256 of the exact HTTP representation body accepted by the server, not browser metadata, normalized HTML, extracted PDF text or a legal interpretation. Content encoding is restricted to identity so compressed and decompressed representations cannot be confused. Preserve the exact bytes, final URL, content type, retrieval timestamp and computed hash in one immutable scoped evidence record. Retrieval alone does not prove statutory applicability, completeness of a consolidated law, or validity of a signature. An approver must examine the retrieved evidence and its applicability; a login or error page returning HTTP 200 is not an approved law merely because it has a hash.

## Publisher retrieval boundary

`backend/internal/regulatorysource/fetch.go` permits only exact, operator-configured DNS hostnames using HTTPS/443, without browser-controlled allowlists, credentials, proxy environment, cookies or custom TLS trust overrides. Every connection resolves and validates every returned address, then dials a validated numeric address to prevent a second DNS resolution. Each redirect must pass the same URL allowlist and connection checks. Fetches are bounded to 30 seconds, five redirect attempts and 20 MiB. Invalid/unconfigured policy fails closed.

The implementation uses the standard Go [HTTP transport](https://pkg.go.dev/net/http#Transport) and [address classification APIs](https://pkg.go.dev/net/netip#Addr.IsGlobalUnicast), plus explicit special-use exclusions; global-unicast classification alone is not a public-network guarantee.

## Lifecycle and remaining integration requirements

1. Register a draft with citation, publisher URL, issuer, applicable dates and source kind. Scope/actor come from host/session. Hashes, verification timestamps and activation flags are not writable request fields.
2. Reserve an actor/scoped, idempotent verification operation against the unchanged source version in a short DB transaction. Fetch outside the DB transaction. Lock and compare the source version before committing immutable bytes and evidence; reject stale verification rather than attach it to changed metadata.
3. Activate through a separate live-RBAC command with explicit evidence ID and expected source version. Persist approval actor and applicability assessment. A legacy `active` row without an evidence record is not verified by migration or by metadata presence.
4. Replace nested `RegulatorySourceRequest` self-certification in `PutRegulatoryProfile` and `CreateOfferingAuthorization` with an activated source reference, preserving scoped composite FKs and historical provenance. Update generated OpenAPI/React contracts and real-stack fixtures in the same change set.
5. Offer archive administrators source selection without requiring unrelated admission permissions. Add dedicated source permissions, administrative UI and a read-only evidence viewer. Publisher allowlist configuration is operator authority, not a tenant-editable SSRF bypass.
6. Non-public institution documents need a separately labeled operator-upload evidence workflow. It must never assert that bytes were independently retrieved from the publisher. Neither workflow bypasses signature verification where signatures are required.
7. Retain immutable cited sources. Source withdrawal/replacement must be represented through a separately versioned lifecycle record and checked by new approvals without rewriting historical citations frozen by migration 0156.

## Required acceptance tests

DNS mixed public/private answers, rebinding, mapped and special-use addresses; redirects outside the allowlist or to private addresses; TLS failures; body size/read failures; request cancellation; exact byte hash; concurrent verification versus edits; activation without evidence; stale source versions; cross-tenant/institution evidence references; replay/conflict; revoked live permission; source selection and approval through generated-contract React UI and real PostgreSQL.

## Verification checkpoint, 2026-09-12

Focused database integration audit confirmed both existing source inserts trust browser checksums; it found no obvious parameterization/context/row-cleanup defects in the inspected institution paths. Independent fetcher review found no SSRF bypass in the inspected implementation, but identified synthesized Referer disclosure across redirects; the redirect hook now removes Referer before validation.

Unit suite and `go vet ./internal/regulatorysource` passed locally. The first complete suite covered 76% of statements; this is not proof of full transport behavior. Local `-race` could not execute because CGO/C compiler support is unavailable; a dedicated Linux CI race command was added, not yet run on GitHub. CI YAML parses with 12 jobs.

The lifecycle now rejects query-bearing publisher and final retrieval URLs rather than persisting them as public provenance. Real TLS/redirect/proxy/slow-body transport tests remain required. No main push or deployment is claimed.

### Integrated local checkpoint

Migration 0157 adds forced-RLS immutable evidence and activation records, server-computed content hashes, scoped idempotency and source metadata snapshots. Verification reserves and commits in separate short transactions around network retrieval, rechecking live permission and metadata before adoption. Activation requires a distinct approver. Legacy active rows receive no fabricated evidence.

Real PostgreSQL HTTP tests passed for registration/replay, verification/replay without refetch, conflicting reuse against another source, exact evidence download, rejection of self-approval, distinct activation/replay, permission revocation and source mutation during retrieval. Retrieval in these tests uses an injected fetcher; this does not prove TLS or OIDC behavior. Archive-retention HTTP and cutover-upgrade integration tests also passed.

OpenAPI validation passed for 628 operations. Generated React contracts, production build, UI policy and contract policy passed. Build reports large JavaScript chunks requiring further performance work. The full React suite initially exposed a test synchronization defect: the PDF section was present before its file input mounted. The test now waits for that input; the complete rerun passed 53 files and 296 tests. Focused Go regulatory-source and server unit suites also passed after correcting typed-nil handling for an unconfigured fetcher.

Remaining release gates include existing profile/offering source-reference cutover, backend enforcement of activation evidence for retention approvals, complete signed admission/appeal E2E with real DSS/WORM, cross-scope adversarial coverage, operator configuration, main reconciliation and cluster deployment verification.
