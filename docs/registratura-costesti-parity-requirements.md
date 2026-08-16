# Registratură Costești parity requirements catalog

Status: audited and implemented for parity release
Audit date: 2026-08-16
Primary target: the eguEducation `Registratură` workspace and its supporting administration
Reference: the authenticated Costești Registratură application and the local `D:\dev\costesti-registratura` source tree

## 1. Objective and scope

The objective is behavioral and visual parity with the Costești `Registratură` tab while preserving eguEducation's tenant isolation, UUID-based identities, education modules, and current security controls.

Functionality has priority over appearance. A visually similar control is not considered complete until its server-side behavior, authorization, validation, persistence, history, and error states match the reference behavior.

In scope:

- the main Registratură document grid;
- Intrare, Ieșire, and Multiplu registration flows;
- registry selection, numbering, search, paging, sorting, history, edit, cancel, print, and PDF export;
- inline creation and administration of correspondents;
- per-document workflow operations needed from Registratură;
- the administration areas needed to configure users, departments, registries, parties, organizations, and the organization chart;
- the user-profile settings that affect the Registratură experience;
- responsive, dark/light, keyboard, and accessibility behavior.

Out of scope for this catalog:

- copying Costești tenant data, personal data, credentials, or production identifiers;
- changing or testing data in the Costești application;
- reproducing the entire Costești Dashboard, Flux, and Archive products beyond the capabilities required by Registratură;
- replacing eguEducation's authentication, RBAC, tenancy, education modules, or stable public API conventions with Costești internals.

## 2. Audit method and evidence boundary

The reference application was inspected read-only in an authenticated browser session. Navigation, tabs, filters, selectors, row expansion, and dialogs were opened. No document was created, edited, cancelled, claimed, assigned, approved, rejected, uploaded, deleted, or exported. Destructive and workflow-transition buttons were not submitted.

Behavior that could not safely be executed was corroborated from the local Costești frontend and backend source. The catalog deliberately contains no live personal data or document content.

Evidence levels used below:

- **Observed**: visible in the authenticated reference UI without changing data.
- **Source-confirmed**: action and constraints verified in the local Costești source.
- **Not executed**: the final mutating or file-producing operation was not invoked against production.

## 3. Current eguEducation baseline

Status legend:

- **Implemented**: the principal behavior and backend contract already exist.
- **Partial**: a related implementation exists, but parity behavior, validation, data, authorization, or presentation is incomplete.
- **Missing**: no working end-to-end capability exists; a placeholder may exist.
- **Verify**: implementation exists but needs an authenticated integration test against a disposable environment.

| Capability | eguEducation status | Current evidence | Main parity gap |
|---|---|---|---|
| Registratură tab and dense document table | Implemented | `documente-workspace.component.ts` | Exact Costești sizing, toolbar order, row expansion, conditional actions |
| Intrare creation | Partial | UI and `POST /api/registratura/documents` | Department, external number/date, activity, file upload, exact defaults and validation |
| Ieșire creation | Partial | Same creation contract with direction | Same gaps; default institution behavior must be tested end to end |
| Multiplu | Partial | Batch UI and server-side unique numbering | Reference limit is 1–20; reference creates `MULTIPLU` placeholders with simpler defaults |
| Registry selector | Implemented | Selected registry is persisted and filters the list | Authorization by registry/department and exact selector presentation |
| Server paging and sorting | Implemented | Lazy table with 10/20/50/100 sizes | Verify every visible Costești column has a stable server sort |
| Advanced search | Partial | Right-side filter drawer | Missing external number and exact entry/exit date semantics; layout differs |
| Row detail | Partial | Right drawer and separate detail page | Reference uses inline expansion with department, external reference, activity, and cancellation reason |
| History | Partial | Document versions in detail drawer | Reference history columns and change descriptions must be matched |
| Edit document | Partial | Edit dialog and page exist | Department, external fields, activity, attachments, exact MULTIPLU conversion rules, workflow-managed status |
| Cancel document | Partial | Cancel endpoint and dialog exist | Minimum reason length, exact immutable cancelled state, warning copy, action gating; current UI says “archived” |
| Single-document print | Missing | Row print opens range-export dialog | Reference produces a document-specific client PDF |
| Date-range PDF export | Implemented | PDF export endpoint and dialog | Default last-30-day range and output equivalence need verification |
| Inline party creation | Partial | Generic physical/legal/institution party dialog | Specialized Costești fields and validation are incomplete |
| Create/edit attachments | Missing | Attachment metadata endpoints and detail view exist | Real upload/select UX and create/edit integration |
| Per-document workflow panel | Missing | Navigation to generic `/workflow` exists | Reference contextual side panel, audit timeline, and state-specific actions |
| Workflow state machine | Partial | Generic workflow tasks exist | Reference document states and transitions are not represented exactly |
| Registry CRUD/default | Implemented | Registry workspace and endpoints | Department assignments, public/private semantics, reset/number constraints, action confirmation |
| Party CRUD API | Implemented | Generic party list/lookup/create/update/delete APIs | No complete administration workspaces; specialized schemas missing |
| User administration | Partial | User and RBAC management exist | Department/organization assignment actions and Costești-specific account controls |
| Departments | Missing | Admin tab is marked `contract-missing` | Tables, APIs, assignments, permissions, and UI |
| Organizations | Missing | Admin tab is marked `contract-missing` | CRUD, default/active, department membership |
| Organization chart | Missing | Admin tab is marked `contract-missing` | Hierarchy, role tag, zoom, user assignment/removal |
| Profile edit and passkeys | Implemented | `/profile` supports profile update and passkey registration | Exact visual grouping, assigned departments, accent palette |
| Light/dark mode | Implemented | Theme service and global theme panel | Costești provides an accent palette and places appearance controls in profile |

## 4. Functional requirements: main Registratură workspace

### 4.1 Shell, toolbar, and registry context

| ID | Requirement | Acceptance criteria | Current status |
|---|---|---|---|
| REG-001 | Registratură must be a first-class authenticated workspace. | The active navigation item is visibly selected; loading or changing registry never leaves the workspace; browser back/forward retains a valid state. | Implemented |
| REG-002 | The primary toolbar must expose `Intrare`, `Ieșire`, and `Multiplu` as distinct actions. | Each opens its own correctly initialized flow; keyboard focus moves into the dialog; unavailable actions are hidden or disabled by permission. | Implemented, permission gating missing |
| REG-003 | The toolbar must contain advanced search, registry selector, PDF export, administration, theme/profile, and logout access in the same discoverable hierarchy as the reference. | Every icon-only action has an accessible name and tooltip; admin is only available to authorized users. | Partial |
| REG-004 | A selected registry is mandatory context for list, create, batch, export, and numbering operations. | Selection is persisted per user/tenant; inaccessible or removed selections fall back to the permitted default registry; all API requests enforce the same registry access server-side. | Partial |
| REG-005 | Changing registry reloads data without a full-page navigation. | Table, total count, filters, numbering preview, and export context update together; stale requests cannot overwrite the latest selection. | Implemented; race test required |
| REG-006 | The registry selector must show only registries permitted by tenant, user, and department policy. | An unauthorized registry cannot be queried by changing a URL or request body. | Missing department policy |

### 4.2 Document table

| ID | Requirement | Acceptance criteria | Current status |
|---|---|---|---|
| REG-010 | Display the reference columns: expand, document number, type, content, sender, recipient, entry date, exit date, status, actions. | Values use consistent Romanian labels, date formatting, truncation, and tooltips; absent values render predictably. | Partial |
| REG-011 | Paging must be server-side with 10, 20, 50, and 100 rows per page. | Total and current range are accurate after filters, registry changes, edits, and cancellations. | Implemented |
| REG-012 | Sorting must be server-side for every sortable visible column. | Ascending/descending results are stable and deterministic, including ties. | Verify |
| REG-013 | Type and status must have distinct, theme-safe visual treatments. | Intrare/Ieșire/Multiplu and all workflow states remain distinguishable in light and dark modes without relying only on color. | Partial |
| REG-014 | Content truncation must preserve access to the full text. | Mouse tooltip and keyboard/screen-reader equivalent expose full content; the table does not expand unpredictably. | Partial |
| REG-015 | Each row must expose only valid actions for its state and the caller's permissions. | Edit/cancel/workflow controls are absent or disabled when prohibited; the server independently rejects forged requests. | Missing in main UI |
| REG-016 | Expanding a row must reveal department(s), external number/date, activity, and cancellation details where applicable. | Expansion is inline, preserves table scroll, and shows an empty-value convention. | Missing; drawer is not equivalent |
| REG-017 | Documents are never hard-deleted through Registratură. | There is no delete control or document delete endpoint; cancellation is immutable and audited. | Implemented structurally |

### 4.3 Intrare and Ieșire

| ID | Requirement | Acceptance criteria | Current status |
|---|---|---|---|
| REG-020 | Intrare initializes sender as selectable and recipient as the tenant's default public institution. | Default is tenant-scoped, visible, and server-validated; no cross-tenant party ID can be submitted. | Partial |
| REG-021 | Ieșire initializes sender as the tenant's default public institution and recipient as selectable. | Same isolation and validation as REG-020. | Partial |
| REG-022 | Both flows support `Document` and `Dosar` record kinds. | Document accepts one primary file; Dosar accepts multiple files; size/type/count limits are visible and server-enforced. | Missing |
| REG-023 | Required data includes content, registry, applicable parties, event date, and at least one department. | Save remains disabled until valid; server returns field-specific errors; the assigned registry is permitted. | Missing department/file semantics |
| REG-024 | Optional data includes external number, external-number date, and activity. | Fields persist, appear in list detail/history/PDF, and can be edited when allowed. | Missing |
| REG-025 | Parties can be searched by normalized name and identifier. | Results are tenant-scoped, debounced, keyboard-selectable, and distinguish party type. | Partial |
| REG-026 | A new party can be created inline without losing document form state. | The new party is selected after save; cancelling returns to the unchanged document form; duplicates trigger a clear warning. | Partial |
| REG-027 | Successful creation allocates the number atomically on the server. | Concurrent requests never receive the same number; failed transactions do not silently consume or duplicate a number according to the approved numbering policy. | Implemented; concurrency test required |
| REG-028 | File upload is integrated into the same creation transaction or a recoverable staged flow. | Partial failure is visible and retryable; orphaned files are cleaned up; malware/type/size controls are enforced. | Missing |

### 4.4 Multiplu

| ID | Requirement | Acceptance criteria | Current status |
|---|---|---|---|
| REG-030 | Multiplu creates between 1 and 20 consecutively numbered placeholder records. | Client and server enforce the range; allocation is atomic under concurrency. | Partial; current egu UI allows up to 100 |
| REG-031 | The flow requires a permitted registry and uses the current entry date by default. | Summary clearly states how many records and which registry will be used. | Partial |
| REG-032 | New batch records start as type `MULTIPLU` and are later convertible to Intrare or Ieșire. | Conversion applies the correct default institution, date field, party editability, and history entry. | Missing exact conversion semantics |
| REG-033 | Batch failure behavior is explicit. | Either the whole batch succeeds atomically or the response identifies committed items and safe retry rules; duplicate numbering is impossible. | Verify/design decision |

### 4.5 Search and export

| ID | Requirement | Acceptance criteria | Current status |
|---|---|---|---|
| REG-040 | Advanced search supports number, type, external number, sender, recipient, entry-date range, and exit-date range. | Search is server-side, composable, resettable, and scoped to selected registry. | Partial |
| REG-041 | Search state must be visibly separable from the results and easy to reset. | Costești-style full-width reveal is reproduced; applied filters remain visible or summarized after collapse. | Missing visual parity |
| REG-042 | Registry PDF export supports a start and end date and defaults to the last 30 days. | Invalid ranges are blocked; the selected registry and inclusive date semantics are shown; output filename is meaningful. | Partial |
| REG-043 | Single-document print/export is available per row. | Output contains exactly the selected document and its approved metadata, independent of the date-range export. | Missing |
| REG-044 | PDF generation is authorized and audited. | Export cannot include another tenant/registry; audit records include user, registry/document, range, and result without storing sensitive file contents. | Partial |

### 4.6 History, editing, cancellation, and workflow

| ID | Requirement | Acceptance criteria | Current status |
|---|---|---|---|
| REG-050 | History shows date, document number, type, content, status, and change description. | Every significant change produces an immutable, actor-stamped version/audit record. | Partial |
| REG-051 | Edit supports content, parties, department(s), external fields, activity, record kind, and new/existing attachments. | Fields are enabled only when allowed by document type/state; save requires a change description where policy requires it. | Partial |
| REG-052 | Workflow-owned status is read-only in the general edit dialog. | Users cannot bypass workflow by directly selecting a target status. | Missing; current egu edit exposes status |
| REG-053 | Cancellation requires an explicit reason of at least 10 characters. | Confirmation shows document number/content and irreversible warning; submission is idempotent; cancelled documents cannot be edited. | Partial |
| REG-054 | Cancellation retains the record and all attachments/history. | Status becomes `ANULAT`, reason/date/actor are visible in expanded detail and audit; UI never describes this as ordinary archiving. | Partial; copy is currently incorrect |
| REG-055 | A contextual workflow side panel opens from a document row. | Panel shows document summary, current state, assignments, and chronological action history without losing list context. | Missing |
| REG-056 | Supported workflow states match the reference: `INCOMING`, `ALOCAT_COMPARTIMENT`, `IN_LUCRU`, `FLUX_APROBARE`, `FINALIZAT`, `ANULAT`. | Database constraint, API enum, UI labels, colors, filters, and transition tests use one canonical state model. | Missing exact model |
| REG-057 | Supported actions include department assignment, optional user assignment, claim, send for approval, approve, and reject with notes. | Only valid actions appear; transitions are atomic and actor-stamped; rejection requires notes. | Missing exact model |
| REG-058 | Workflow authorization is permission- and assignment-based. | Mutable display names are never authorization inputs; tenant, department, assigned user, and approver IDs are immutable references. | Partial; security hardening now denies mutable-name fallback |

## 5. Administration and settings requirements

### 5.1 Users

| ID | Requirement | Acceptance criteria | Current status |
|---|---|---|---|
| ADM-001 | List users with identity, contact verification, roles, departments, organization, state, and last login. | Server paging/filtering/sorting; tenant-scoped values; no cross-tenant PII. | Partial |
| ADM-002 | Create/edit first name, last name, email, phone, address, active state, and verification flags according to policy. | Required fields and uniqueness are validated; changing phone clears verification; shared identities cannot be modified by a tenant admin. | Partial |
| ADM-003 | Provide separate actions for activate/deactivate, edit, role assignment, department assignment, and organization assignment. | Every mutation has a permission, audit event, confirmation where destructive, and tenant boundary. | Partial |
| ADM-004 | Tenant admins cannot assign platform `super_admin` or mutate global RBAC catalogs. | UI omits forbidden options and backend rejects forged requests. | Implemented in local hardening; migration deployment pending |
| ADM-005 | User deletion must follow an approved retention model. | Prefer deactivation where records reference the user; destructive deletion is blocked when audit integrity would be broken. | Design required |

### 5.2 Departments

| ID | Requirement | Acceptance criteria | Current status |
|---|---|---|---|
| ADM-010 | CRUD departments with required name and optional description. | Tenant-scoped, filterable, pageable; deletion is blocked or safely migrated when referenced. | Missing |
| ADM-011 | Departments can be assigned to users, registries, documents, and organizations. | Join records are tenant-bound and unique; stale assignments are visible and repairable. | Missing |
| ADM-012 | One user-department assignment can be designated primary if required by registration defaults. | Primary is unique per user/tenant; removing it selects no silent cross-tenant fallback. | Missing |

### 5.3 Registries

| ID | Requirement | Acceptance criteria | Current status |
|---|---|---|---|
| ADM-020 | Registry form supports name, prefix, starting/current/next number, reset date, public/private type, default flag, and departments. | Numeric invariants and unique prefix/default rules are enforced in the database transaction. | Partial |
| ADM-021 | A registry can be edited and deleted only when safe. | Registries with documents cannot be destructively removed; default transfer is explicit; confirmation identifies impact. | Partial |
| ADM-022 | Public/private and department rules determine visibility and use. | List, selector, create, export, and direct API access enforce identical policy. | Missing |

### 5.4 Parties

| ID | Requirement | Acceptance criteria | Current status |
|---|---|---|---|
| ADM-030 | Separate pageable workspaces exist for physical persons, legal entities, and public institutions. | Add/edit/delete, filters, empty states, validation, and tenant scoping match the reference information architecture. | Missing UI; generic API exists |
| ADM-031 | Physical person data supports first/last name, CNP, birth date/place, contact, and address. | CNP is validated and protected as sensitive data; search and display reveal only necessary information. | Partial data model |
| ADM-032 | Legal entity data supports name, CUI, trade-register number, legal representative, legal form, share capital, contact, and address. | Identifier validation and duplicate detection are server-side. | Partial data model |
| ADM-033 | Public institution data supports name, type, level, website, contact, address, hierarchy, and default institution. | Exactly one active default institution per tenant; the default cannot be casually edited/deleted. | Partial data model |
| ADM-034 | Party deletion preserves referential and audit integrity. | Referenced parties are deactivated or deletion is rejected; historical document rendering remains stable. | Partial |

### 5.5 Organizations and organization chart

| ID | Requirement | Acceptance criteria | Current status |
|---|---|---|---|
| ADM-040 | CRUD organizations with name, description, active state, default state, and department assignments. | One default per tenant; inactive organizations cannot receive new assignments. | Missing |
| ADM-041 | Organization chart renders the department hierarchy and assigned users. | Large hierarchies support horizontal scrolling, zoom in/out, percentage, and reset. | Missing |
| ADM-042 | Department nodes can edit parent and role tag. | Cycles, self-parenting, and cross-tenant parents are rejected. | Missing |
| ADM-043 | Users can be added to or removed from a department from the chart. | Mutations are permission-checked, audited, reversible, and update all relevant selectors. | Missing |

### 5.6 User profile and appearance

| ID | Requirement | Acceptance criteria | Current status |
|---|---|---|---|
| ADM-050 | Profile displays identity, verification, account dates/state, roles, and assigned departments. | Sensitive identifiers are not exposed unnecessarily; values are tenant-contextual. | Partial |
| ADM-051 | Profile editing supports first/last display data, phone, and address while email remains policy-controlled. | Phone change clears verification; save errors are field-specific. | Partial |
| ADM-052 | Passkeys can be listed and added safely. | Attested credential ID is server-bound, challenges are one-time, and duplicate credentials fail. | Implemented locally; deployment pending |
| ADM-053 | Appearance supports light/dark mode and a Costești-equivalent primary-color palette. | Choice persists per user or device, has sufficient contrast, and applies consistently to dialogs/tables/actions. | Partial; accent palette missing |

## 6. Visual parity requirements

| ID | Requirement | Reference characteristic | Current status |
|---|---|---|---|
| UI-001 | Typography | Inter/system sans, approximately 14 px base text, compact labels and table content. | Partial; egu uses system sans without explicit Inter |
| UI-002 | Dark surfaces | Page around `#18181b`, dense row surface around `#09090b`, borders around `#3f3f46`, high-contrast text. | Partial |
| UI-003 | Primary accent | Indigo primary close to `#818cf8`; Intrare blue, Ieșire green, destructive actions red. | Partial |
| UI-004 | Toolbar composition | Search at the left, central Intrare/Ieșire/Multiplu group, registry and PDF at the right; no wrapping at normal desktop widths. | Partial |
| UI-005 | Table density | Compact striped rows around 64 px, sticky paginator, horizontal and vertical scroll without page overflow. | Partial |
| UI-006 | Dialog geometry | Rounded 12 px dark/light surface, restrained border, approximately 780 px reference width where appropriate, scrollable body, stable footer. | Partial; egu dialogs are generally wider |
| UI-007 | Advanced search layout | Full-width reveal beneath toolbar with grouped fields and right-aligned Reset/Search controls. | Missing; egu uses a right drawer |
| UI-008 | Inline row expansion | Detail appears immediately below the selected row and keeps the table context visible. | Missing |
| UI-009 | Workflow panel | Contextual right-side panel with document header, timeline, and a visually separated action area. | Missing |
| UI-010 | Status/type styling | Compact tags with readable Romanian labels and consistent semantic palette in both themes. | Partial |
| UI-011 | Responsive behavior | Toolbar collapses deliberately, dialogs fit 94vw, tables retain horizontal scroll, touch targets remain usable. | Verify |
| UI-012 | Accessibility | All icon buttons have `aria-label`, tooltip, visible focus, keyboard activation, and non-color status text. | Partial; the Costești reference itself has unlabeled row buttons, which egu should improve rather than copy |

Visual parity means the same hierarchy, spacing density, control placement, and interaction model. It does not require copying tenant logos, personal data, or inaccessible defects.

## 7. Cross-cutting security, integrity, and operational requirements

| ID | Requirement | Acceptance criteria |
|---|---|---|
| SEC-001 | Every document, registry, party, department, organization, workflow action, and assignment is tenant-bound. | Cross-tenant identifiers fail closed at the database and service layer. |
| SEC-002 | Registry and department access is enforced on every read, write, export, and lookup. | UI filtering is never the only control. |
| SEC-003 | Permissions are granular: read, manage documents, cancel, export, manage workflow, manage registries, manage parties, manage structure. | Tests cover allowed and denied cases for each role. |
| SEC-004 | Sensitive identifiers and uploaded documents are protected at rest and in logs. | CNP/CUI/contact data are redacted from routine logs; object storage is private and malware-scanned. |
| SEC-005 | All mutations and exports create immutable audit events. | Actor, tenant, action, target, result, and timestamp are captured without plaintext secrets or file contents. |
| SEC-006 | Concurrency-sensitive numbering and workflow transitions are transactional. | Race tests prove uniqueness and reject stale transitions. |
| SEC-007 | The locally added tenant/RBAC migration is rehearsed before deployment. | Backup, retained/removed grant preview, migration test, and rollback procedure are approved. |
| OPS-001 | Liveness and readiness are separate. | `/healthz` checks process health; `/readyz` fails when the database is unavailable. |
| OPS-002 | Deployment uses immutable images and non-root restricted pods. | Digests, probes, resource policy, PDB, and manifest validation pass before promotion. |

### 7.1 Mandatory multi-tenant adaptation

Costești is a single-tenant behavioral reference. eguEducation must never reproduce its implicit global scope. Every parity feature must be adapted to an explicit tenant context.

| ID | Requirement | Acceptance criteria |
|---|---|---|
| TEN-001 | Tenant context is resolved from a trusted host/domain mapping and then checked against the authenticated user's active membership. | Knowing or supplying another tenant's hostname/ID does not grant access; an inactive or absent membership fails closed. |
| TEN-002 | Every Registratură-owned record carries a non-null tenant/institution key. | Database constraints and RLS cover documents, registries, parties, departments, organizations, assignments, workflow actions, versions, attachments, exports, and audit events. |
| TEN-003 | Registry numbering is independent per tenant, registry, and configured reset period. | Concurrent numbering in tenant A cannot block, consume, or collide with numbering in tenant B; uniqueness includes the tenant key. |
| TEN-004 | Defaults are tenant-specific. | Default registry, public institution, organization, departments, locale/time zone, and appearance/branding cannot leak or fall back across tenants. |
| TEN-005 | Shared identities have separate memberships and privileges per tenant. | The same person may be an operator in one tenant and read-only in another; roles, direct permissions, departments, organizations, and registry access never bleed between memberships. |
| TEN-006 | Platform administration is distinct from tenant administration. | Tenant administrators can manage only their tenant and cannot grant `super_admin`, mutate platform catalogs, or bypass RLS. |
| TEN-007 | All lookup, autocomplete, filter-option, count, and dashboard queries include tenant context. | No names, counts, identifier matches, or existence signals from another tenant appear, including through empty-state and timing behavior. |
| TEN-008 | File storage is partitioned and authorized by tenant. | Object keys include an immutable tenant namespace; download/upload/signing endpoints re-check tenant ownership; signed URLs are short-lived. |
| TEN-009 | Background work retains tenant context. | PDF generation, notifications, file scanning, retention, indexing, and workflow jobs record and enforce tenant identity rather than running in an unscoped global context. |
| TEN-010 | Caches and persisted UI state are tenant-keyed. | Registry selections, filter state, query caches, DPoP/session state, and generated artifacts cannot be reused after switching tenants unless explicitly safe. |
| TEN-011 | Audit and observability include tenant identity without exposing tenant data to other tenants. | Operational staff can diagnose by tenant; tenant admins see only their own audit records; logs redact sensitive party/document data. |
| TEN-012 | Automated isolation tests cover every resource family and role. | Tests create two tenants with deliberately overlapping names/numbers and attempt cross-tenant list, lookup, get-by-ID, mutation, export, workflow, file, and admin access. |

The local tenant/RBAC hardening already moves authentication and administration toward this model, but parity work must preserve these guarantees in every new table, endpoint, job, and UI cache. A Costești code path may be reused only after its implicit global assumptions are removed.

## 8. Required API and data-model work

The existing eguEducation document, registry, party, version, attachment-metadata, and generic workflow contracts should be evolved rather than replaced.

Required additions:

1. Departments and assignment tables:
   - tenant-keyed departments;
   - tenant-keyed user-department memberships, including optional primary flag;
   - tenant-keyed registry-department access;
   - tenant-keyed document-department assignments;
   - tenant-keyed organization-department assignments.
2. Organizations and hierarchy:
   - tenant-keyed organizations with active/default state;
   - tenant-keyed user-organization membership;
   - department `parent_id` and `role_tag`, with cycle protection.
3. Document parity fields:
   - external number and external-number date;
   - separate entry and exit timestamps;
   - activity;
   - record kind (`document`/`dosar`);
   - canonical workflow status and workflow assignment IDs;
   - cancellation actor/date/reason.
4. Party specialization:
   - protected physical-person fields;
   - legal-entity identifiers and representative data;
   - public-institution type, level, website, hierarchy, and default constraint.
5. Files:
   - staged or multipart upload contract;
   - tenant ownership, checksum, size, MIME, scan state, retention, and tenant-namespaced object-storage key;
   - one-primary-file versus multi-file rules.
6. Workflow:
   - document-specific actions and transition table;
   - department/user/approver assignments;
   - immutable action history and optimistic concurrency/version checks.
7. Read models:
   - one server-paged grid response with display-ready party and department names;
   - exact advanced filters and sortable fields;
   - dashboard/workflow counts only where needed by Registratură.

Compatibility decisions:

- Keep eguEducation's UUID identifiers where already established; do not import Costești numeric IDs.
- Preserve the eguEducation cancellation route unless a versioned public API requires change; behavior matters more than URL shape.
- Provide migration/backfill rules for existing documents whose new fields are absent.
- Do not expose arbitrary status changes through general document edit after the workflow state machine is introduced.

## 9. Implementation plan

### Phase 0 — freeze contracts and test fixtures

- Convert this catalog into executable acceptance cases.
- Capture sanitized desktop and responsive reference screenshots for layout only.
- Define canonical Romanian labels, status enum, transition matrix, permission matrix, and registry visibility policy.
- Create disposable tenant fixtures covering two tenants, two registries, departments, parties, and users with different permissions.

Exit gate: product owner approves the transition matrix, field dictionary, and non-copying treatment of inaccessible reference behavior.

### Phase 1 — data model and backend foundations

- Add departments, organizations, hierarchy, assignments, parity document fields, and specialized party storage.
- Add database constraints for tenant identity, default records, hierarchy cycles, numbering, and workflow versioning.
- Extend document list/filter/read models and registry access enforcement.
- Add file staging/upload and scan-state contracts.

Exit gate: migration rehearses successfully on a production-shaped disposable database; tenant-negative, numbering-concurrency, and referential-integrity tests pass.

### Phase 2 — Registratură administration

- Replace the current `contract-missing` admin tabs with working departments, registries, three party workspaces, organizations, and organization chart.
- Extend user administration with role, department, and organization assignments.
- Complete profile department display and appearance controls.

Exit gate: an authorized admin can configure every prerequisite for document registration; an unauthorized or other-tenant user cannot read or change them.

### Phase 3 — main registration flows

- Align toolbar and registry context.
- Complete Intrare/Ieșire fields, defaults, validation, department selection, record kind, and attachments.
- Align Multiplu to the 1–20 placeholder workflow and conversion rules.
- Complete inline specialized party creation while preserving form state.

Exit gate: browser tests create Intrare, Ieșire, Document, Dosar, and Multiplu records in a disposable environment and validate numbering, fields, files, and history.

### Phase 4 — grid, search, history, edit, cancel, and print

- Implement inline row expansion and exact reference columns.
- Implement the full-width advanced-search reveal and missing filters.
- Make row actions state- and permission-aware.
- Complete reference history/edit/cancel rules.
- Separate single-document print from date-range registry export.

Exit gate: the complete main-grid acceptance suite passes for all document states and roles, including direct-API negative tests.

### Phase 5 — document workflow parity

- Implement the canonical states and atomic transitions.
- Add the contextual workflow side panel and action timeline.
- Add assign, claim, approval, approve, and reject interactions.
- Reconcile the existing generic workflow task feature with document workflow: adapt it behind a compatibility layer or keep it as a separate non-Registratură workflow rather than creating two sources of truth.

Exit gate: transition-table tests and multi-user browser scenarios pass; stale or unauthorized transitions fail predictably.

### Phase 6 — visual parity and accessibility

- Apply Inter-compatible typography, reference spacing, toolbar geometry, table density, dialog sizes, dark surfaces, and semantic colors.
- Add responsive toolbar behavior and test common desktop/tablet/mobile widths.
- Add accessible names, focus management, keyboard coverage, contrast checks, and screen-reader status text.

Exit gate: sanitized visual-regression diffs are within approved tolerances and automated accessibility checks have no serious/critical violations.

### Phase 7 — release validation and deployment

- Run backend unit/integration/race tests, frontend unit tests, browser end-to-end tests, dependency scans, manifest validation, and database migration rehearsal.
- Backup the database and preview legacy RBAC grant effects before applying the tenant-grant migration.
- Publish immutable images, promote through GitOps, verify readiness, smoke-test login and Registratură with non-production test records, and monitor errors/latency/audit events.
- Use a documented rollback that preserves any documents created after deployment.

Exit gate: production login and read paths work, approved smoke writes succeed in a designated test registry, monitoring is clean, and rollback remains viable.

## 10. Minimum browser acceptance scenarios

1. A permitted operator selects each accessible registry and sees only its documents.
2. An unauthorized registry ID is rejected through direct API use.
3. Intrare and Ieșire initialize the correct default party and persist all parity fields.
4. Document accepts one primary file; Dosar accepts multiple; failed upload is recoverable.
5. Multiplu accepts 1 and 20, rejects 0 and 21, and allocates unique numbers under concurrency.
6. A MULTIPLU item converts to Intrare and Ieșire with correct party/date rules and history.
7. All advanced filters combine correctly and reset cleanly.
8. Every sortable column produces stable server ordering; paging totals remain correct.
9. Edit availability follows permission and status; direct forged updates are rejected.
10. Cancellation rejects short reasons, is idempotent, remains visible in history, and prevents later editing.
11. Single-document print contains only the chosen record; registry export respects registry/date scope.
12. Workflow assignment, claim, approval, rejection, completion, and cancellation follow the transition matrix.
13. Two concurrent users cannot double-claim or apply stale workflow transitions.
14. Tenant admins can configure local users/structure but cannot assign platform super-admin.
15. Keyboard-only users can operate toolbar, grid, expansion, dialogs, autocomplete, and workflow panel.

## 11. Definition of parity complete

Parity is complete only when:

- every **Partial**, **Missing**, and **Verify** item in the baseline has an accepted disposition;
- all required Costești Registratură behaviors are backed by tenant-safe server contracts and automated tests;
- the supporting administration can configure the behavior without database intervention;
- authenticated visual and functional browser tests pass in both light and dark themes;
- deployment migration, rollback, observability, login, and production smoke checks are complete;
- no production Costești data or credentials have been copied into eguEducation.
