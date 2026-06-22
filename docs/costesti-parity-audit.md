# Costesti Registratura Parity Audit

This document is the implementation checklist for replacing guessed EguEducation UI with reference-driven functionality from `E:\dev\costesti-registratura`.

## Admin Dashboard

Reference files:
- `frontend/src/app/admin/admin-dashboard/admin-dashboard.component.html`
- `frontend/src/app/admin/user-management/user-management.component.html`
- `frontend/src/app/admin/registre-management/registre-management.component.html`
- `frontend/src/app/admin/compartimente-management`
- `frontend/src/app/admin/persoane-fizice-management`
- `frontend/src/app/admin/persoane-juridice-management`
- `frontend/src/app/admin/institutii-publice-management`
- `frontend/src/app/admin/organisation-management`
- `frontend/src/app/admin/organigrama`

Costesti behavior:
- One admin dashboard component with tabs.
- Admin tabs: Utilizatori, Compartimente, Registre, Persoane Fizice, Persoane Juridice, Institutii Publice, Organizatii, Organigrama.
- Profile tab available in same dashboard.
- Each management tab has a header action, lazy PrimeNG table, server-side paging/sorting/filtering, action column, and modal create/edit dialogs.

EguEducation status:
- Partial: `/admin` exists but had generic placeholder tabs.
- Missing: Costesti-style admin tabs and management surfaces.
- Partial backend: users, org units, positions, permissions, roles, modules, auth methods, OIDC, GDPR settings exist.
- Missing backend contracts: Costesti-style compartimente, registre, persoane fizice, persoane juridice, institutii publice, organizatii, organigrama as first-class admin contracts.

Implementation target:
- Replace generic admin tabs with Costesti-compatible admin dashboard grouping.
- Use existing backend contracts where available.
- Show explicit "backend contract missing" state for missing Costesti contracts until implemented.

## Registratura Toolbar and Dialogs

Reference files:
- `frontend/src/app/app.component.html`
- `frontend/src/app/registratura/registratura.component.html`
- `frontend/src/app/registratura/multiplu-dialog/multiplu-dialog.component.ts`
- `frontend/src/app/registratura/date-interval-dialog/date-interval-dialog.component.ts`

Costesti behavior:
- Search icon button is on the left.
- Intrare, Iesire, Multiplu button group is centered.
- Export PDF is on the right.
- Intrare/Iesire opens new document dialog with mode selector and full form.
- Multiplu opens dedicated generation dialog with count, registru, continut, data intrare, validation, info panel.
- Export PDF opens date interval dialog with default last-30-days and validates start/end.

EguEducation status:
- Partial: toolbar now has search left, button group center, Export PDF right.
- Partial: dialogs exist but not yet backed by real registratura API.
- Missing: registru selector, real entity autocomplete, real save/generate/export actions.

Implementation target:
- Wire dialogs to registratura API.
- Add selected registru contract.
- Add batch creation and PDF export backend endpoints or explicitly reuse generated API when available.

## Flux Documente

Reference files:
- `frontend/src/app/flux/flux.component.ts`
- `frontend/src/app/flux/flux.component.html`
- `frontend/src/app/services/workflow.service.ts`

Costesti behavior:
- Tabs: Coada mea, Mapa semnaturi, Evidenta completa.
- Server-side lazy tables, filters, sort, paging.
- Pipeline stats cards.
- Workflow action side panel.

EguEducation status:
- Partial UI shell exists.
- Missing real API binding and action panel parity.

## Arhiva

Reference files:
- `frontend/src/app/arhiva/arhiva.component.html`
- `frontend/src/app/arhiva/cautare/cautare.component.html`
- `frontend/src/app/arhiva/dosare/dosare.component.html`
- `frontend/src/app/arhiva/ingestie`
- `frontend/src/app/arhiva/nomenclator`
- `frontend/src/app/services/archive.service.ts`

Costesti behavior:
- Tabs: Cautare, Dosare, Ingestie, Nomenclator.
- Cautare uses search bar, filters, p-table, file preview drawer.
- Dosare uses tree + detail file list + create dialog.
- Ingestie and nomenclator are real workflows.

EguEducation status:
- Partial visual shell exists.
- Missing real archive API binding and full dialogs/actions.
