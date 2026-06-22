# Plan Curent Education

## Scop

Acest document este singura referinta activa pentru reluarea lucrului pe modulul `education`.

Documentele vechi de audit, reset frontend si design direction au fost eliminate pentru a evita confuzia. Documentatia operationala care ramane valida si trebuie pastrata este:

- [docs/local-development.md](/E:/dev/egueducation/docs/local-development.md)
- [docs/rbac-matrix.md](/E:/dev/egueducation/docs/rbac-matrix.md)
- [backend/docs/oidc-provider-data-model.md](/E:/dev/egueducation/backend/docs/oidc-provider-data-model.md)

## Stare Curenta

### Backend

Exista implementare reala pentru:

- guvernanta institutionala: sedinte, participanti, documente sedinta, voturi, minute, apartenente, hotarari
- decizii si conformitate: emitere, publicare, requirement mapping
- documente manageriale si regulamente: documente, versiuni, workflow steps
- personal: registre, roluri, incadrari, cazuri disciplinare, dosar personal, acces dosar
- evaluari: autoevaluari, criterii, contestatii, comunicare rezultat, PDF-uri
- mobilitate: documente, punctaje, contestatii, decizie finala, comunicare rezultat, PDF-uri
- gradatie de merit: documente, punctaje, contestatii, decizie finala, comunicare rezultat, PDF-uri
- portofolii CD: documente, checklist, opis, custodie, transferuri, verificari, PDF principal
- export generic CSV/PDF
- RBAC si tenant scoping

### Frontend

Exista implementare Angular 21 + PrimeNG + Tailwind pentru:

- shell education pe module
- workspace generic cu taburi
- tabele server-side cu sortare, filtrare in header, paginare si actiuni
- dialoguri CRUD
- toasts
- export CSV/PDF
- PDF row actions pentru evaluari, mobilitate, gradatii si portofolii

### Ultima verificare tehnica

Verificat local la data `2026-06-22`:

- `frontend`: `npm run build` OK
- `backend`: `go build ./...` OK

## Ce Este Gata vs Ce Mai Lipseste

### 1. Guvernanta

Implementat partial-avansat:

- registre si subregistre exista
- modelul de date este deja mult peste un CRUD simplu

Mai lipseste:

- PDF-uri dedicate pentru sedinte, hotarari, minute, voturi si pachete de sedinta
- validari business mai stricte pentru cvorum, tipuri de vot si coerenta participantilor
- workflow complet de inchidere sedinta si emitere document final
- UX specializat, nu doar generic table

### 2. Decizii, conformitate, publicare

Implementat partial-avansat:

- registre de decizii
- publicare
- requirement/compliance flows

Mai lipseste:

- PDF-uri dedicate pentru decizii si publicari
- trasabilitate cap-coada intre cerinta, document, aprobare si publicare
- ecrane dedicate pentru conformitate, nu doar tabel generic

### 3. Documente manageriale si regulamente

Implementat partial-avansat:

- dosare manageriale
- documente
- versiuni
- workflow steps

Mai lipseste:

- modele business mai stricte pe tip de document
- PDF-uri dedicate pe dosar/document/versiune
- flux complet de avizare, aprobare si publicare
- UI dedicat pentru versiuni si workflow

### 4. Personal

Implementat partial-avansat:

- registre personal
- incadrari
- disciplinar
- dosar personal
- jurnal acces

Mai lipseste:

- consolidare completa a dosarului personal in UI
- PDF-uri dedicate pentru roluri, incadrari, disciplinar si dosar
- polish pe RBAC de actiuni sensibile

### 5. Evaluari

Implementat bine:

- evaluare principala
- autoevaluari
- criterii
- contestatii
- comunicare rezultat
- PDF evaluare, contestatie, comunicare rezultat

Mai lipseste:

- UI dedicat pe fisa de evaluare, mai putin generic
- export Excel specializat
- validari business suplimentare pe status transitions

### 6. Mobilitate

Implementat bine:

- dosar principal
- documente
- punctaje
- contestatii
- decizii finale
- comunicari rezultat
- PDF principal si PDF-uri pe subfluxuri

Mai lipseste:

- validari mai stricte pe etape de mobilitate
- UI procedural dedicat pe caz
- export specializat si rapoarte

### 7. Gradatie de merit

Implementat bine:

- dosar principal
- documente
- punctaje
- contestatii
- decizii finale
- comunicari rezultat
- PDF principal si PDF-uri pe subfluxuri

Mai lipseste:

- validari mai stricte pe finantare si ierarhizare
- UI procedural dedicat pe dosar
- export specializat si rapoarte

### 8. Portofoliu profesional CD

Implementat bine la nivel de fundatie:

- registru principal
- structura sectiuni
- documente
- checklist
- opis
- custodie
- transfer
- review
- PDF principal

Mai lipseste:

- PDF-uri dedicate pe subfluxuri relevante daca devine necesar
- flux explicit de procedura interna aprobata si publicata
- UI dedicat pentru citire tip dosar, nu doar taburi generice
- retentie si arhivare mai bine expuse in UX

### 9. Frontend transversal

Mai lipseste:

- trecere mai larga la abordare signal-first / `httpResource` unde merita
- eliminarea treptata a logicii imperative bazate pe `subscribe`
- romanizare completa si consistenta a tuturor etichetelor
- verificare fina de responsive pentru toate modulele
- reducerea scroll-urilor secundare in dialoguri mari

### 10. Calitate si contracte

Mai lipseste:

- testare automata pe fluxurile education
- contracte API mai formalizate
- smoke tests UI pentru fluxurile majore

## Prioritate Recomandata De Reluare

Ordinea recomandata pentru reluare este:

1. `Guvernanta`
   Finalizare PDF-uri, validari de cvorum/vot si UX procedural pentru sedinte/hotarari.

2. `Documente manageriale + regulamente`
   Workflow complet de versiuni, aprobare, publicare si document final.

3. `Conformitate + publicare`
   Legare clara intre requirement, document, emitere, aprobare si publicare.

4. `Personal`
   Polish pe dosar personal, actiuni sensibile si documente generate.

5. `Frontend hardening`
   Signal-first, romanizare completa, responsive pass, exporturi specializate.

## Plan De Continuare

### Etapa 1. Guvernanta si documente finale

- adaugare PDF-uri pentru sedinte, hotarari, minute si voturi
- ecran dedicat de sedinta cu tabs pentru participanti, documente, voturi, minute
- validari de business pentru cvorum si tipuri de decizie

### Etapa 2. Documente manageriale si regulamente

- PDF-uri dedicate pentru dosare/documente/versiuni
- workflow vizibil in UI pentru draft, avizare, aprobare, publicare
- taburi specializate pentru versiuni si pasi de workflow

### Etapa 3. Conformitate si publicare

- UX dedicat pentru requirement mapping
- raport de conformitate pe document si pe domeniu
- publicare cu status clar si documente generate

### Etapa 4. Personal si dosar personal

- pagini dedicate pentru dosar personal si acces
- documente generate pentru disciplinar / incadrari daca sunt necesare operational
- verificare completa RBAC pe toate actiunile sensibile

### Etapa 5. Frontend modernization pass

- unde este fezabil, inlocuire progresiva cu signals si `httpResource`
- consistenta completa PrimeNG
- polish responsive si romanizare finala

## Fisiere Cheie Pentru Reluare

### Frontend

- [frontend/src/app/features/education/shared/education-config.ts](/E:/dev/egueducation/frontend/src/app/features/education/shared/education-config.ts)
- [frontend/src/app/features/education/shared/education-domain-workspace.component.ts](/E:/dev/egueducation/frontend/src/app/features/education/shared/education-domain-workspace.component.ts)
- [frontend/src/app/features/education/education-shell.component.ts](/E:/dev/egueducation/frontend/src/app/features/education/education-shell.component.ts)

### Backend

- [backend/cmd/server/main.go](/E:/dev/egueducation/backend/cmd/server/main.go)
- [backend/internal/education/service.go](/E:/dev/egueducation/backend/internal/education/service.go)
- [backend/internal/education/governance_portfolio_flows.go](/E:/dev/egueducation/backend/internal/education/governance_portfolio_flows.go)
- [backend/internal/education/managerial_regulation_flows.go](/E:/dev/egueducation/backend/internal/education/managerial_regulation_flows.go)
- [backend/internal/education/decision_compliance_flows.go](/E:/dev/egueducation/backend/internal/education/decision_compliance_flows.go)
- [backend/internal/education/personnel_detail_flows.go](/E:/dev/egueducation/backend/internal/education/personnel_detail_flows.go)
- [backend/internal/education/personnel_role_disciplinary_flows.go](/E:/dev/egueducation/backend/internal/education/personnel_role_disciplinary_flows.go)
- [backend/internal/education/evaluation_documents.go](/E:/dev/egueducation/backend/internal/education/evaluation_documents.go)
- [backend/internal/education/mobility_merit_documents.go](/E:/dev/egueducation/backend/internal/education/mobility_merit_documents.go)
- [backend/internal/education/portfolio_documents.go](/E:/dev/egueducation/backend/internal/education/portfolio_documents.go)

## Comenzi De Reluare

### Frontend

```powershell
cd E:\dev\egueducation\frontend
npm run build
```

### Backend

```powershell
cd E:\dev\egueducation\backend
go build ./...
```

## Nota Finala

Modulul `education` nu mai este intr-o faza de mock sau schelet. Exista deja fundatie serioasa si fluxuri reale pe mai multe subdomenii.

Ce a ramas nu este "sa incepem de la zero", ci:

- finalizare de fluxuri procedurale complexe
- documente finale si PDF-uri dedicate
- UX specializat pe procese
- hardening tehnic si functional
