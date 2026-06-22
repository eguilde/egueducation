# EguEducation

Greenfield education platform scaffold with:

- Go backend
- Angular 21 frontend
- Angular Material 21 + Tailwind CSS
- Transloco i18n (`ro` default, `en` secondary)
- Red/rose Material 3 expressive theme
- Responsive shell/sidebar foundation
- Auth and admin foundations for OIDC, OTP, passkeys, wallet, RBAC, workflow, archive, and education modules

## Structure

- `backend/` Go API and service foundations
- `frontend/` Angular SPA
- `docs/` architecture and implementation notes
- `ops/` local/dev deployment assets

## Current baseline

This repository currently includes the initial frontend shell/theme/i18n/auth runtime scaffold and a compile-ready backend skeleton with configuration, database bootstrap migrations, admin/auth metadata endpoints, and SMS transport integration points.

## Database

The backend now expects a `DATABASE_URL` and runs embedded bootstrap migrations on startup.

Cluster database wiring currently targets:

- namespace: `education`
- secret: `egueducation-scoalabalotesti-db`
- database: `scoalabalotesti`
- user: `egueducation_scoalabalotesti_app`

## Kubernetes

Kubernetes manifests are in `ops/k8s/` and currently assume:

- frontend host: `scoalabalotesti.eguilde.cloud`
- image registry: `registry.eguilde.cloud/eguilde`
- backend image: `registry.eguilde.cloud/eguilde/egueducation-backend:latest`
- frontend image: `registry.eguilde.cloud/eguilde/egueducation-frontend:latest`
- image pull secret: `ghcr-pull-secret`
- TLS via `letsencrypt-godaddy`
