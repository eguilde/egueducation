# EguEducation Architecture Notes

## Current implemented baseline

- Angular 21 standalone frontend scaffold
- Angular Material 21 + Tailwind integration
- Transloco with `ro` default and `en` support
- Responsive app shell with persistent desktop sidebar and mobile drawer behavior
- Material 3 expressive red/rose theme with light/dark switching
- Frontend auth runtime scaffold for redirect-based OIDC + DPoP
- Go backend scaffold with app metadata, auth metadata, admin dashboard metadata, and SMS transport integration

## Planned donor sources

- `E:\dev\egudoc-rs` for registratura, workflow, archive, and education modules
- `E:\dev\egudoc` for Go workflow/admin/RBAC patterns
- `E:\dev\eguilde` for OIDC, DPoP, passkeys, and wallet architecture
- `E:\dev\costesti-registratura` for SMSAPI transport code

## Next implementation layers

1. Real OIDC provider and token/session endpoints
2. Position-based RBAC and admin CRUD
3. Shared server-side table API/query contract
4. Registratura domain and document lifecycle
5. Workflow runtime/admin
6. eArhiva
7. Education governance and personnel modules
8. GDPR exports, anonymization, retention, and audit hardening
