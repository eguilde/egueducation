# Plan de executie tehnica - Faza 0 si Faza 1

Data: 2026-06-24

Documente de referinta:

- [docs/education-management-specifications.md](/E:/dev/egueducation/docs/education-management-specifications.md)
- [docs/education-implementation-audit.md](/E:/dev/egueducation/docs/education-implementation-audit.md)
- [docs/education-delivery-roadmap.md](/E:/dev/egueducation/docs/education-delivery-roadmap.md)

## 1. Scop

Acest document detaliaza executia pentru:

- `Faza 0 - Fundatie de livrare`
- `Faza 1 - Cockpit director`

Este gandit ca plan imediat de implementare in cod.

## 2. Rezultatul urmarit

La finalul fazelor 0 si 1 trebuie sa existe:

- o fundatie reutilizabila pentru wizard-uri;
- permisiuni contextuale de baza pentru fluxurile educationale;
- statusuri educationale afisate consistent in UI;
- un cockpit director real, actionabil;
- agregari backend noi pentru management educational;
- primul set de liste restante si shortcut-uri operationale.

## 3. Boundaries

### In scope

- Angular components reutilizabile pentru wizard si cockpit;
- endpointuri agregate pentru dashboard director;
- extinderi RBAC pentru actiuni educationale;
- model de date minim pentru status procedural si liste restante, daca este necesar;
- documentatie de mapping backend/frontend.

### Out of scope

- wizard sedinta CA;
- wizard sedinta CP;
- generatoare PDF oficiale pentru CA/CP;
- consiliul clasei;
- comisii ca module dedicate.

Acestea incep din fazele urmatoare.

## 4. Faza 0 - Fundatie de livrare

## 4.1. Epic 0.1 - Standardizare statusuri si actiuni

### Obiectiv

Sa existe un vocabular comun si stabil pentru:

- statusuri;
- actiuni;
- etichete;
- severitati vizuale;
- naming de documente.

### Taskuri backend

1. Inventariere statusuri educationale existente in:
   - `backend/internal/education/service.go`
   - `backend/internal/education/detail_records.go`
   - `backend/internal/education/procedural_flows.go`
   - `backend/internal/education/governance_portfolio_flows.go`
   - `backend/internal/education/managerial_regulation_flows.go`
   - `backend/internal/education/personnel_detail_flows.go`
   - `backend/internal/education/personnel_role_disciplinary_flows.go`
   - `backend/internal/education/evaluation_detail_flows.go`
   - `backend/internal/education/evaluation_result_issue_flows.go`
   - `backend/internal/education/mobility_merit_flows.go`
   - `backend/internal/education/mobility_merit_outcomes.go`

2. Grupare statusuri pe familii:
   - sedinta;
   - document;
   - publicare;
   - portofoliu;
   - evaluare;
   - contestatie;
   - transfer;
   - review.

3. Definire mapping intern:
   - `code`;
   - `label_ro`;
   - `severity`;
   - `group`;
   - `terminal/non-terminal`.

### Taskuri frontend

1. Extrage logica de etichete/status din `education-domain-workspace.component.ts`.
2. Creeaza utilitar comun:
   - `education-status-map.ts`
   - `education-status.helpers.ts`
3. Standardizeaza tag rendering pentru:
   - culoare;
   - label;
   - fallback.
4. Elimina listele ad hoc de statusuri unde este posibil.

### Deliverables

- registru intern de statusuri;
- helper comun de status;
- rendering consistent in tabele si dashboarduri.

### Fisiere probabile

- `frontend/src/app/features/education/shared/education-status-map.ts`
- `frontend/src/app/features/education/shared/education-status.helpers.ts`
- `frontend/src/app/features/education/shared/education-domain-workspace.component.ts`

## 4.2. Epic 0.2 - RBAC contextual

### Obiectiv

Sa putem controla actiunile educationale nu doar pe rol global, ci si contextual.

### Permisiuni noi recomandate

- `education.governance.meeting.close`
- `education.governance.meeting.vote`
- `education.governance.minutes.publish`
- `education.governance.resolution.publish`
- `education.portfolios.verify`
- `education.portfolios.transfer`
- `education.portfolios.custody.manage`
- `education.managerial.publish`
- `education.reports.export_sensitive`
- `education.inspect.package.read`

### Reguli contextuale recomandate

1. `director`
   - acces complet la cockpit director;
   - inchidere sedinta;
   - publicare documente institutionale;
   - verificare portofolii.

2. `secretar`
   - administrare convocatoare;
   - minute in lucru;
   - emitere/comunicare;
   - acces limitat la documente sensibile conform scope.

3. `profesor`
   - acces doar la propriul portofoliu/evaluare/declaratii;
   - participare in sedinte doar unde este membru activ;
   - nu poate vedea global registrul complet al portofoliilor.

4. `inspector`
   - acces citire controlata la pachete de inspectie;
   - fara editari operationale.

### Taskuri backend

1. Adaugare permisiuni in migrari noi.
2. Extindere mapari rol-permisiune.
3. Introducere helperi pentru verificare contextuala:
   - `canViewOwnPortfolio`
   - `canActOnMeeting`
   - `canInspectPackage`
4. Aplicare initiala in endpointurile noi de cockpit.

### Taskuri frontend

1. Extindere configuratii de permisiuni in `education-config.ts`.
2. Garduri UI pe actiuni:
   - butoane;
   - taburi;
   - shortcut-uri din dashboard.

### Deliverables

- migrare SQL noua pentru permisiuni;
- helper backend pentru reguli contextuale;
- configuratii frontend actualizate.

### Fisiere probabile

- `backend/internal/db/migrations/006x_education_contextual_permissions.sql`
- `backend/internal/education/middleware.go`
- `frontend/src/app/features/education/shared/education-config.ts`

## 4.3. Epic 0.3 - Fundatie frontend pentru wizard-uri

### Obiectiv

Sa existe un kit UI reutilizabil pentru fluxurile educationale urmatoare.

### Componente necesare

1. `WizardStepperComponent`
2. `WizardSummaryPanelComponent`
3. `DocumentPreviewPanelComponent`
4. `DeadlineBadgeComponent`
5. `AuditTrailPanelComponent`

### Cerinte UI

- standalone Angular 21;
- signal-first;
- PrimeNG + Tailwind;
- fara CSS raw inutil;
- responsive;
- capabile sa functioneze in fluxuri diferite.

### Model comun recomandat

```ts
interface WizardStepState {
  key: string;
  label: string;
  description?: string;
  status: 'pending' | 'active' | 'completed' | 'blocked';
  valid: boolean;
  dirty: boolean;
}
```

### Taskuri frontend

1. Creare folder nou:
   - `frontend/src/app/features/education/shared/wizard/`
2. Implementare componente.
3. Implementare modele comune.
4. Implementare exemple demo locale sau stari mock pentru dezvoltare.

### Deliverables

- kit reutilizabil de wizard;
- document preview reutilizabil pentru PDF/documente generate;
- panel audit pentru afisare istoric contextual.

## 5. Faza 1 - Cockpit director

## 5.1. Epic 1.1 - Agregari backend pentru cockpit

### Obiectiv

Sa construim un endpoint agregat nou, separat de dashboardurile sumare existente.

### Endpoint recomandat

- `GET /api/education/director/cockpit`

### Structura recomandata a raspunsului

```json
{
  "schoolYear": "2025-2026",
  "governance": {},
  "portfolios": {},
  "evaluations": {},
  "managerial": {},
  "personnel": {},
  "compliance": {},
  "alerts": [],
  "upcomingDeadlines": [],
  "quickLinks": []
}
```

### Agregari minime

#### Governance

- sedinte planificate;
- sedinte fara cvorum estimat;
- sedinte fara proces-verbal final;
- hotarari nepublicate.

#### Portfolios

- total;
- draft;
- in verificare;
- returnate pentru completari;
- validate;
- cu retentie apropiata;
- cu lipsuri in checklist.

#### Evaluations

- draft;
- depuse;
- evaluate;
- contestate;
- finalizate fara comunicare;
- documente neoglindite in dosarul personal, daca apar.

#### Managerial

- dosare in lucru;
- dosare in avizare;
- dosare in aprobare;
- documente publice nepublicate;
- regulamente in revizie.

#### Personnel

- personal activ;
- dosare incomplete;
- acces sensibil recent;
- lipsuri documente.

#### Compliance

- cerinte planificate;
- cerinte partial implementate;
- documente fara publicare;
- publicari cu anonimizare necesara.

### Alerting rules recomandate

- sedinta in urmatoarele 3 zile fara convocator final;
- sedinta trecuta fara proces-verbal inchis;
- portofoliu returnat de mai mult de 7 zile;
- evaluare contestata deschisa;
- document managerial blocat in avizare;
- document obligatoriu nepublicat;
- acces sensibil recent in dosar personal.

### Taskuri backend

1. Definire model de raspuns in `backend/internal/education/models.go` sau fisier dedicat.
2. Implementare agregari SQL.
3. Implementare reguli de alerting.
4. Expunere endpoint in `backend/cmd/server/main.go`.
5. Audit logging pentru acces cockpit, daca se considera util.

### Fisiere probabile

- `backend/internal/education/cockpit_models.go`
- `backend/internal/education/cockpit_service.go`
- `backend/cmd/server/main.go`

## 5.2. Epic 1.2 - UI dashboard director

### Ruta recomandata

- `education/dashboard/director`

### Structura ecranului

1. Header:
   - an scolar activ;
   - filtre rapide;
   - actiuni rapide.

2. Banda de alerte:
   - carduri mici actionabile.

3. Grid principal:
   - governance;
   - portofolii;
   - evaluari;
   - documente manageriale;
   - conformitate;
   - personal.

4. Deadline lane:
   - termene viitoare;
   - sedinte;
   - documente in risc.

5. Shortcut-uri:
   - deschide sedinta;
   - vezi portofolii incomplete;
   - vezi evaluari contestate;
   - vezi documente nepublicate.

### Taskuri frontend

1. Creare componenta noua:
   - `education-director-dashboard.component.ts`
2. Folosire `httpResource`.
3. Componente suport:
   - `CockpitAlertListComponent`
   - `CockpitMetricCardComponent`
   - `CockpitDeadlineListComponent`
4. Integrare in rute si taburi.

### Fisiere probabile

- `frontend/src/app/features/education/dashboard/education-director-dashboard.component.ts`
- `frontend/src/app/features/education/dashboard/cockpit-alert-list.component.ts`
- `frontend/src/app/features/education/dashboard/cockpit-metric-card.component.ts`
- `frontend/src/app/features/education/dashboard/cockpit-deadline-list.component.ts`
- `frontend/src/app/features/education/education.routes.ts`
- `frontend/src/app/features/education/shared/education-config.ts`

## 5.3. Epic 1.3 - Liste rapide si exporturi din cockpit

### Obiectiv

Fiecare widget important sa poata deschide o lista filtrata sau export.

### Taskuri

1. Deep links catre workspace-ul generic existent cu filtre presetate.
2. Export CSV/PDF pentru listele din cockpit.
3. Butoane:
   - `Sedinte fara proces-verbal`
   - `Portofolii incomplete`
   - `Evaluari contestate`
   - `Documente nepublicate`

### Observatie

In aceasta faza este suficient sa reutilizam workspace-ul generic drept pagina de detaliu. Nu trebuie facute inca ecrane procedurale pentru tot.

## 6. Ordine concreta de implementare

### Pasul 1

- standardizare statusuri in frontend.

### Pasul 2

- migrare permisiuni noi si extindere configuratii RBAC.

### Pasul 3

- componente comune pentru wizard-uri.

### Pasul 4

- model backend pentru director cockpit.

### Pasul 5

- endpoint agregat `director cockpit`.

### Pasul 6

- componenta frontend `education-director-dashboard`.

### Pasul 7

- deep links si exporturi rapide din cockpit.

### Pasul 8

- polish pe acces contextual si etichete.

## 7. Checklist de acceptanta

### Faza 0

- statusurile educationale se afiseaza consistent;
- permisiunile noi exista in sistem;
- exista fundatie UI de wizard;
- se poate construi un wizard nou fara reimplementare a infrastructurii de baza.

### Faza 1

- directorul vede alerte si deadline-uri reale;
- cockpitul foloseste agregari noi, nu doar carduri numerice;
- shortcut-urile deschid liste utile;
- filtrele pe an scolar functioneaza;
- accesul la cockpit este restrictionat corect.

## 8. Riscuri

### Risc 1

Agregarile backend pot deveni grele daca sunt facute intr-un singur query monolitic.

Mitigare:

- query-uri separate pe domenii;
- compunere in service layer;
- optimizare dupa prima versiune.

### Risc 2

Permisiunile contextuale pot complica rapid frontendul.

Mitigare:

- helper centralizat de capability;
- evitare logica duplicata in componente.

### Risc 3

Cockpitul poate deveni doar un dashboard vizual, fara actiune.

Mitigare:

- fiecare card important trebuie sa duca la lista sau flux actionabil.

## 9. Decizii recomandate inainte de cod

1. Cockpitul director va avea ruta separata sau va inlocui dashboardul education generic pentru director.
2. Shortcut-urile din cockpit vor merge catre workspace-ul generic existent sau catre rute noi specializate, unde exista deja.
3. Permisiunile contextuale vor fi aplicate direct in endpointuri sau prin helperi reutilizabili in middleware/service.

Recomandare:

- ruta separata pentru cockpit director;
- shortcut-uri spre workspace-ul generic in Faza 1;
- helperi reutilizabili in service layer pentru contextual RBAC.

## 10. Urmatorul pas dupa aceste faze

Dupa finalizarea fazelor 0 si 1, urmatorul pas optim este:

- `Wizard sedinta CA`

Motiv:

- avem deja componente de wizard;
- avem cockpit care trimite in fluxuri;
- guvernanta este suficient de modelata in backend;
- valoarea operationala pentru scoala este imediata.
