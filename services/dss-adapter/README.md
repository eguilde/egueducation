# eGuEducation EU DSS adapter

This is a deployment-owned Java service for the backend's
`egueducation-signed-artifact-verifier.v1` protocol. It is not an EU DSS demo,
does not create signatures, and fails closed when trusted-list material cannot
be refreshed and validated.

## What `/verify` proves

For every request the service:

1. authenticates the backend bearer token and limits the request body;
2. reads the exact S3/MinIO `(bucket,key,versionId)` requested by the backend;
3. verifies the observed SHA-256, byte length, compliance-mode object lock and
   exact retention instant against the backend evidence;
4. rejects ambiguous/missing PDF signatures and non-whitespace bytes appended
   after the PDF signature ByteRange;
5. extracts exactly one `egueducation-admission-legal-payload.json` embedded
   file and compares its raw SHA-256 with the expected canonical legal payload;
6. validates the PAdES container with EU DSS 6.5 and an EU LOTL/TL certificate
   verifier, including online OCSP, CRL and AIA lookup;
7. maps the leaf certificate SHA-256 to the requested tenant, institution and
   OIDC subject from an operator-provided binding map; and
8. stores diagnostic, detailed, simple and ETSI XML reports as versioned,
   compliance-mode WORM objects and returns only their immutable references.

It returns `valid` only when every step succeeds. `invalid`, `indeterminate`
or unavailable trust material must never be treated as an authorization result.

## Required configuration

| Variable | Meaning |
| --- | --- |
| `DSS_ADAPTER_BEARER_TOKEN` | shared backend-to-adapter secret |
| `DSS_ADAPTER_S3_ENDPOINT`, `DSS_ADAPTER_S3_ACCESS_KEY`, `DSS_ADAPTER_S3_SECRET_KEY` | dedicated MinIO/S3 credentials, least privilege only |
| `DSS_ADAPTER_ALLOWED_BUCKET` | the sole source-artifact bucket |
| `DSS_ADAPTER_REPORT_BUCKET` | versioning + Object Lock **COMPLIANCE** enabled report bucket |
| `DSS_ADAPTER_REPORT_RETENTION_DAYS` | immutable report retention, 365–36500 days |
| `DSS_ADAPTER_EU_LOTL_OJ_KEYSTORE`, `DSS_ADAPTER_EU_LOTL_OJ_KEYSTORE_TYPE`, `DSS_ADAPTER_EU_LOTL_OJ_KEYSTORE_PASSWORD` | Official Journal anchored EU LOTL signing-certificate keystore; never replace this with a downloaded key |
| `DSS_ADAPTER_CERTIFICATE_BINDINGS_JSON` | map of lower-case leaf SHA-256 to `{ "tenantCode", "institutionId", "actorSubject" }` |

Optional values: `DSS_ADAPTER_EU_LOTL_URL`, `DSS_ADAPTER_EU_OJ_URL`,
`DSS_ADAPTER_TRUST_MAX_AGE_HOURS` (1–168), `DSS_ADAPTER_PORT`, and
`DSS_ADAPTER_HTTP_WORKERS`.

Example binding (a secret/config value, not an application user input):

```json
{"0123456789abcdef0123456789abcdef0123456789abcdef0123456789abcdef":{"tenantCode":"scoalabalotesti","institutionId":"...","actorSubject":"oidc-subject"}}
```

`/verify/capabilities` returns 503 until an actual signed EU LOTL refresh is
loaded. A deployment must permit egress to EU LOTL, national TL, OCSP/CRL/AIA
endpoints and configure durable trust-list caching before it can serve legal
signatures. This module intentionally does **not** provide a development
self-signed or mock trust mode.

## Build

```powershell
mvn test
mvn package
docker build -t egueducation-dss-adapter .
```

The module uses `eu.europa.ec.joinup.sd-dss:dss-bom:6.5` and DSS's official
PAdES, validation, policy, TSL and service modules. EU DSS is LGPL-2.1; retain
its notices and supply source/attribution obligations in the distributed image.
