# EguEducation Frontend And Contract Reset Plan

## Purpose

This plan replaces the current EguEducation frontend direction with a PrimeNG 21, Tailwind-first operational UI based on the audited reference systems:

- `E:\dev\costesti-registratura` for registratura, flux documente, eArhiva, admin tabs, PrimeNG tables, dialogs, drawers, and practical operator UX.
- `E:\dev\eguilde` for OIDC provider behavior, redirect login, callback, DPoP, refresh-token cookie handling, and backend-rendered login/consent/logout.
- Angular 21 MCP guidance for standalone components, lazy routes, signals, modern control flow, accessibility, and maintainable Angular structure.

The current EguEducation implementation has useful backend/domain breadth, but the frontend stack, auth shape, and UI composition went in the wrong direction. This is a reset, not a polish pass.

## Non-Negotiables

- Remove Angular Material from the frontend.
- Install and use PrimeNG 21 as the component framework.
- Install and use `@primeng/themes`, `primeicons`, and `tailwindcss-primeui`.
- Keep Tailwind CSS for layout, flexbox, CSS grid, spacing, responsive behavior, and utility styling.
- Use PrimeNG `p-table` for all operational registers.
- Use PrimeNG `p-tabs` for logical feature grouping.
- Use PrimeNG drawers/dialogs for row details, create/edit forms, workflow panels, attachment/version panels, and high-risk actions.
- Do not create fake dashboard tabs.
- Do not mix statistics, forms, details, and operational tables in one unstructured screen.
- Validate through local browser/Playwright interaction, not only builds or HTTP status codes.

## Reference Findings

### Costesti Registratura

Copy/adapt:

- PrimeNG app config with Aura theme and Tailwind integration.
- Root toolbar plus tabbed operational navigation.
- Registratura workflow:
  - top actions for document creation
  - collapsible search/filter panel
  - lazy `p-table`
  - row expansion
  - action column
  - status tags
  - dialogs for document operations
  - workflow drawer entry point
- Flux documente workflow:
  - `Coada mea`
  - `Mapa semnaturi`
  - `Evidenta completa`
  - role-aware tabs
  - lazy tables per tab
  - shared workflow drawer
- eArhiva workflow:
  - `Cautare`
  - `Dosare`
  - `Ingestie`
  - `Nomenclator`
- Admin console:
  - tabbed sections
  - PrimeNG tables
  - dialogs for editing

Do not copy blindly:

- giant 1500+ line templates and 2000+ line components
- client-side fake paging in some admin screens
- unsafe HTML snippet rendering
- broad `::ng-deep` styling as the normal solution

### Eguilde Auth

Copy/adapt:

- Backend OIDC provider owns login, consent, and logout HTML.
- SPA login means redirect to OIDC authorize, not a custom Angular login page.
- SPA callback route is `/callback`.
- Guard stores intended return URL and starts redirect login.
- Refresh token is moved into an HttpOnly cookie.
- SPA stores only minimal session indicators and access/id token state.
- Token endpoint requests use `credentials: include`.
- DPoP key handling and interceptor behavior follow Eguilde patterns.
- Token claims stay stable identity claims; permissions remain server-side/session-context driven.

Remove/retire:

- EguEducation Angular `/login` as primary login UI.
- EguEducation Angular `/auth/consent` as primary consent UI.
- EguEducation Angular `/auth/logout` as primary logout UI.
- localStorage refresh-token style behavior.

## Target Information Architecture

### Top-Level Workspaces

- `Documente`
- `Educatie`
- `Admin`
- `GDPR`
- `Profil`

### Documente

Tabs:

- `Registratura`
- `Flux documente`
- `eArhiva`

No `Dashboard` tab.

Registratura table columns:

- expander
- numar document
- tip
- continut/subiect
- emitent
- destinatar
- data intrare
- data iesire
- status
- actiuni

Registratura actions:

- view details
- edit
- cancel
- print/export
- attach file
- view versions
- start/open workflow
- archive

Registratura detail surfaces:

- expanded row for compact metadata
- drawer/dialog tabs for:
  - detalii
  - versiuni
  - atasamente
  - flux
  - arhivare

### Flux Documente

Tabs inside the `Flux documente` tab when needed:

- `Coada mea`
- `Mapa semnaturi`
- `Evidenta completa`

Each uses a separate lazy `p-table` and one shared workflow drawer.

### eArhiva

Tabs inside the `eArhiva` tab:

- `Cautare`
- `Dosare`
- `Ingestie`
- `Nomenclator`

Each tab gets its own table/operation surface and contract.

### Admin

Tabbed console, not route sprawl:

- `Identitate si acces`
- `Structura`
- `Fluxuri si nomenclatoare`
- `GDPR si audit`
- `Platforma`
- `Profil`

Every table must be server-side paginated/sorted/filtered.

### Educatie

Before implementation, create a requirement catalog from Romanian education references and the two PDFs.

Initial grouped workspaces:

- `Conducere si conformitate`
  - sedinte CA/CP/CEAC/CFDCD
  - hotarari
  - ROF/ROI
  - dosare manageriale
- `Personal si cariera`
  - cadre
  - evaluari anuale
  - mobilitate
  - gradatii
  - adeverinte punctaj
  - declaratii
- `Portofolii si dosare`
  - portofoliu CD
  - opis/index
  - transfer
  - retentie 3 ani
  - consimtamant
  - declaratie autenticitate

## PrimeNG Register Contract

Every operational register must use the same behavioral contract:

- `p-table`
- `[lazy]="true"`
- server-side paging
- server-side sorting
- server-side filtering
- `scrollable`
- `scrollHeight="flex"` or equivalent full-height table container
- current page report
- rows per page options
- filter row or collapsible filter panel depending on density
- action column always last
- empty state
- loading state
- row expansion or drawer for details
- no client-side load-all filtering for business data

## Frontend-Backend Contract

Create explicit TypeScript and Go contract rules before rebuilding screens.

Paginated response:

```ts
interface Page<T> {
  items: T[];
  total: number;
  page: number;
  pageSize: number;
}
```

Query parameters:

- `page`
- `pageSize`
- `sort`
- `direction`
- typed filters by endpoint

Backend requirements:

- each list endpoint declares allowed sort fields
- each list endpoint declares allowed filter fields
- invalid sort/filter fields return stable error codes
- each mutable endpoint returns stable validation errors
- each mutable aggregate has audit metadata
- each institution-bound query resolves institution from session context, not hardcoded `inst-001`

## Backend-Database Contract

Required cleanup:

- replace hardcoded `inst-001` in services with authenticated institution context
- move OIDC runtime schema toward Eguilde IAM shape where applicable
- define migrations as stable module contracts, not just seed scaffolding
- implement real file upload/storage contract for registratura attachments
- ensure every table used by frontend lists has indexes for filters and sort fields
- define retention/publication/anonymization DB contracts for GDPR-sensitive education flows

## Migration Work Packages

### Package 1: PrimeNG Stack

- remove `@angular/material`
- remove Material theming from `styles.scss`
- add `primeng`, `@primeng/themes`, `primeicons`, `tailwindcss-primeui`
- configure `providePrimeNG`
- import Tailwind and PrimeUI plugin in global styles
- keep Transloco and Angular 21 standalone/lazy route structure

### Package 2: Shell And Auth

- replace Material shell with PrimeNG/Tailwind shell
- remove Angular-owned login/consent/logout screens from primary flow
- port Eguilde auth service/interceptor/callback behavior
- add `/callback`
- add `/api/config` frontend bootstrap contract
- backend-render login/consent/logout from OIDC provider

### Package 3: Documente Reference Module

- build `Documente` shell with PrimeNG tabs
- build registratura PrimeNG table as canonical register
- wire current backend endpoints where adequate
- identify missing backend fields/contracts
- add missing backend endpoints before UI workarounds
- validate with Playwright desktop and mobile

### Package 4: Flux And eArhiva

- build flux role-aware tabs and tables
- build eArhiva tabs and tables
- align backend endpoints to Costesti contract shape where useful
- add workflow drawer and archive dialogs

### Package 5: Admin Console

- rebuild admin as tabbed PrimeNG console
- convert all admin lists to server-side `p-table`
- remove client-side fake paging
- group identity/access, structure, workflows/nomenclatures, GDPR/audit, platform

### Package 6: Education Requirements Catalog

- produce a catalog from:
  - existing PDFs
  - current Romanian education legislation references
  - current EguEducation domain modules
- map each requirement to:
  - entity
  - workflow
  - document/dossier relation
  - retention/GDPR behavior
  - UI register
  - backend endpoint
  - database table

### Package 7: Education UI Rebuild

- implement grouped PrimeNG tabs
- table-first screens
- drawers/dialogs for details/actions
- workflow/document/eArhiva linkage
- GDPR-sensitive visibility and publication controls

### Package 8: Contract Generation And Tests

- introduce OpenAPI or generated contract artifact
- generate Angular API client or enforce typed contract generation
- add backend integration tests for contracts
- add Playwright tests for:
  - auth redirect/callback/logout
  - registratura table filters/sort/paging/actions
  - flux workflow actions
  - eArhiva search/ingest
  - admin tabs
  - responsive mobile tables

## Decisions Needed

These are the only decisions that should stop implementation:

- Whether to port Eguilde OIDC provider code directly into EguEducation or run EguEducation as a client of the existing Eguilde identity service.
- Which object storage target to use for local and production attachment upload.
- Whether to preserve the current EguEducation DB schema with migrations or start a new clean schema after contract definition.
- Whether the frontend should be rewritten in place or under a temporary `frontend-next` folder until parity is reached.

## Default Assumptions If Not Overridden

- Rewrite frontend in place.
- Preserve useful backend modules but refactor contracts.
- Use the existing `scoalabalotesti` database for local development.
- Use Eguilde auth behavior as source of truth.
- Use Costesti document workflows as interaction reference, not copy-paste architecture.
- Keep Romanian as default UI language and English as secondary.

## Validation Standard

A task is not complete because it builds.

It is complete only when:

- code builds
- backend tests pass for touched contracts
- Playwright verifies the local UI workflow
- mobile viewport is checked
- no browser console errors remain for the target flow
- the implementation matches the relevant reference behavior

