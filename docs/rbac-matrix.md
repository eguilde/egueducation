# RBAC Matrix

This document describes the default education role model used by EguEducation.

## Roles

- `admin`: operational school administrator
- `super_admin`: platform operator with full access
- `director`: school director / principal
- `profesor`: teacher
- `secretar`: school secretary
- `registrator`: registry clerk
- `inspector`: school inspector
- `gdpr_officer`: data protection officer
- `workflow_admin`: workflow administrator

## Default access

| Functionality | Required roles |
| --- | --- |
| Login / registration / OIDC consent | Any authenticated user |
| `documente` / registratura | `admin`, `super_admin`, `director`, `secretar`, `registrator` |
| `education` workspace | `admin`, `super_admin`, `director`, `profesor`, `inspector` |
| `admin` workspace | `admin`, `super_admin`, `director` |
| `gdpr` workspace | `admin`, `super_admin`, `director`, `gdpr_officer` |
| profile page | Any authenticated user |

## Admin RBAC management

| Admin function | Required roles |
| --- | --- |
| Role catalog | `admin`, `super_admin`, `director` |
| User-role assignments | `admin`, `super_admin`, `director` |
| Role-permission mappings | `admin`, `super_admin` |
| Position-role mappings | `admin`, `super_admin`, `director` |
| OIDC clients / consents / sessions | `admin`, `super_admin` |
| Auth method settings | `admin`, `super_admin` |
| GDPR settings | `admin`, `super_admin` |

## Permission model

Roles are mapped to permissions in `app_role_permissions`.
Membership positions are mapped to roles in `app_position_roles`.
Session permissions are derived from:

- direct user permissions
- permissions granted by direct user roles
- permissions granted by membership positions
- permissions granted by roles mapped from membership positions

This keeps the system extensible while allowing role-based UI gating.

## Default position mapping

The starter lookup table seeds the most common education positions:

- `director` -> `director`
- `secretariat` -> `secretar`
- `registrator` -> `registrator`
- `inspector` -> `inspector`
- `profesor` -> `profesor`
