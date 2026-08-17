# EguEducation API contract

`openapi.json` is the generated, versioned OpenAPI 3.1.1 contract. It covers every concrete `GET`, `POST`, `PUT`, `PATCH`, and `DELETE` route registered in `backend/cmd/server/main.go` (currently 463 API operations), plus the public OIDC-provider routes.

The contract is generated from the router, `overrides.json`, and the deterministically merged domain fragments in `domains/`; shared security, error and pagination definitions live in `components/common.json`. Do not edit `openapi.json` by hand.

```powershell
pwsh ./scripts/openapi/generate-openapi.ps1
pwsh ./scripts/openapi/test-openapi.ps1
```

`test-openapi.ps1` gates router coverage, unique operation IDs, component references, operation tag structure, authenticated-route security/tenant/RBAC metadata, the 304-operation Education handler catalog, and the OIDC interaction assets. All 463 concrete router operations are `detailed`; the gate rejects inferred/unknown schema status and any scoped operation that reaches the legacy generic `Entity` schema. Education composite summaries are projected from their named Go DTOs, while the portfolio-OPIS regenerate response is modelled from its explicit handler response map.

All authenticated operations are tenant-scoped by the backend. The active institution is derived from authenticated membership, token/session claims and the request host; APIs do not accept a browser-chosen `X-Institution-ID` header. Permissions in `x-required-permission` are derived from the route registration where determinable and must be enforced server-side; the documentation is not an authorization mechanism.
