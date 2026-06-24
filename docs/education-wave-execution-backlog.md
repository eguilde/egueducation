# Backlog executabil pe valuri pentru management educational

Data: 2026-06-24

Document companion pentru:

- [docs/education-remaining-implementation-plan.md](/E:/dev/egueducation/docs/education-remaining-implementation-plan.md)
- [docs/education-delivery-roadmap.md](/E:/dev/egueducation/docs/education-delivery-roadmap.md)
- [docs/rbac-matrix.md](/E:/dev/egueducation/docs/rbac-matrix.md)

## 1. Scop

Acest document transforma planul ramas intr-un backlog de executie pe valuri si verticale. Fiecare val trebuie sa livreze functionalitate utilizabila, nu doar tabele sau componente izolate.

Starea observata in cod la 2026-06-24:

- exista deja permisiuni contextuale seed-uite in `0064_education_contextual_permissions.sql`;
- exista kit de wizard-uri frontend si mai multe wizard-uri initiale;
- exista endpoint si modele pentru `DirectorCockpit`;
- exista dashboard director si pagina de rapoarte standard;
- lipsesc inca aplicarea contextuala completa in backend, rolurile educationale fine, verticalele CP/comisii/consiliul clasei/elevi si pachetul de inspectie.

## 2. Reguli de lucru

- Fiecare val include backend, frontend, RBAC, audit si minimum un raport sau export daca procesul produce documente.
- Nu se adauga rol global doar ca sa treaca un buton in UI; se prefera context calculat din resursa.
- Orice actiune sensibila are verificare in backend si ascundere in frontend.
- Workspace-ul generic ramane util pentru liste si administrare, dar fluxurile principale trebuie sa devina procedurale.
- Testarea automata nu se ruleaza fara cerere explicita in sesiune; criteriile de acceptanta raman totusi scrise pentru implementatori.

## 3. Valul 0 - Aliniere RBAC si inventar tehnic

Status: partial implementat, de inchis inainte de verticale noi.

### Obiectiv

Sa existe contract clar pentru roluri, permisiuni si reguli contextuale, aplicabil in endpointuri si in UI.

### Stories

| ID | Story | Acceptanta | File touchpoints probabile |
| --- | --- | --- | --- |
| W0-01 | Seed roluri educationale lipsa sau decide explicit ca raman contextuale | `director_adjunct`, `diriginte`, `membru_ca`, `membru_cp`, `evaluator`, `responsabil_comisie` sunt fie in `app_roles`, fie documentate ca roluri contextuale fara grant global | `backend/internal/db/migrations/006x_education_roles_alignment.sql`, `docs/rbac-matrix.md` |
| W0-02 | Helper backend de contextual RBAC | Exista functii reutilizabile pentru `canReadOwnPortfolio`, `canActOnMeeting`, `canReadPersonnelFile`, `canExportSensitiveReport`; endpointurile sensibile le folosesc | `backend/internal/education/contextual_rbac.go`, `backend/internal/education/middleware.go`, `backend/internal/education/service.go` |
| W0-03 | Capabilities frontend centralizate | Butoanele si shortcut-urile folosesc helper comun, nu conditii duplicate pe roluri | `frontend/src/app/features/education/shared/education-capabilities.ts`, `education-config.ts`, wizard/dashboard components |
| W0-04 | Audit actiuni sensibile | Inchidere sedinta, publicare, export sensibil, acces dosar personal si transfer portofoliu scriu audit cu actor si resursa | `backend/internal/education/*flows.go`, `backend/internal/education/export.go`, infrastructura audit existenta |
| W0-05 | Contract 403 contextual | Cand regula contextuala pica, backendul raspunde consistent cu `403` si cod de eroare util | `backend/internal/education/contextual_rbac.go`, handlers sensibile |

### Decizii de produs

- `membru_ca`, `membru_cp`, `diriginte`, `evaluator` si `responsabil_comisie` pornesc ca roluri contextuale.
- `director_adjunct` poate deveni rol global doar daca se implementeaza delegarea pe domeniu.
- `profesor` primeste self-service, nu citire globala.

## 4. Valul 1 - Guvernanta CA/CP inchisa procedural

Status: CA are fundatie si wizard-uri initiale; CP este inca incomplet ca verticala explicita.

### Obiectiv

Sedinta CA/CP se gestioneaza cap-coada: planificare, componenta, convocator, prezenta, cvorum, vot, minute, hotarari, publicare si registru.

### Stories

| ID | Story | Acceptanta | File touchpoints probabile |
| --- | --- | --- | --- |
| W1-01 | Calcul cvorum din componenta reala | Cvorumul se calculeaza din membri activi si participanti, nu doar din camp manual; inchiderea este blocata daca nu trece | `backend/internal/education/governance_portfolio_flows.go`, `backend/internal/education/service.go`, `frontend/.../ca-meeting-wizard.component.ts` |
| W1-02 | Inchidere sedinta CA | Statusul trece doar prin tranzitii valide; minutele si voturile necesare sunt prezente | `backend/internal/education/procedural_flows.go`, `backend/internal/education/contextual_rbac.go`, `backend/cmd/server/main.go` |
| W1-03 | PDF oficial CA | Exista PDF convocator, proces-verbal, hotarare si registru CA cu naming stabil | fisier nou `backend/internal/education/governance_documents.go`, `backend/internal/education/export.go`, `education-config.ts` |
| W1-04 | CP ca organism procedural | CP are componenta, sedinte, prezenta, vot, minute si registru distincte de CA | migrari noi pentru CP daca modelul curent nu ajunge, `governance_portfolio_flows.go`, ruta/componenta CP noua |
| W1-05 | UI sedinta procedurala | Directorul/secretarul opereaza sedinta din ecran dedicat; workspace-ul generic ramane fallback | `frontend/src/app/features/education/governance/*wizard.component.ts`, componenta noua `governance-meeting-detail.component.ts` |
| W1-06 | Rapoarte guvernanta | Registru sedinte CA, registru sedinte CP, registru hotarari si raport cvorum sunt exportabile | `backend/internal/education/cockpit_service.go`, `backend/internal/education/export.go`, `frontend/.../education-standard-reports.component.ts` |

### Acceptanta de val

- Un utilizator fara context de membru nu poate vota.
- O sedinta fara cvorum sau fara minute finale nu poate fi inchisa.
- PDF-urile oficiale se pot descarca din UI.
- Cockpitul director arata sedintele fara minute, fara vot sau cu publicare restanta.

## 5. Valul 2 - Documente manageriale, decizii si conformitate

Status: modelul de date este bun, dar experienta este inca prea apropiata de registru.

### Obiectiv

Documentele institutionale majore devin fluxuri ghidate cu versiuni, avizare, aprobare, publicare si dovada de conformitate.

### Stories

| ID | Story | Acceptanta | File touchpoints probabile |
| --- | --- | --- | --- |
| W2-01 | Workflow document managerial pe tip | PDI/PAS, RAEI, raport calitate, ROF/ROI au pasi si campuri obligatorii diferite | `backend/internal/education/managerial_regulation_flows.go`, `frontend/.../managerial-dossier-wizard.component.ts` |
| W2-02 | Publicare document managerial | Publicarea cere versiune aprobata, canal, anonimizare unde este cazul si legatura cu cerinta | `backend/internal/education/decision_compliance_flows.go`, `managerial_regulation_flows.go`, `education-compliance-page.component.ts` |
| W2-03 | Matrice cerinta -> document -> dovada | Exista vedere operationala cu cerinte fara document, documente fara publicare si dovezi expirate | `backend/internal/education/cockpit_service.go`, `frontend/.../education-compliance-page.component.ts` |
| W2-04 | PDF-uri manageriale | PDI/PAS, raport calitate, RAEI si regulament/versiune au PDF dedicat, nu doar export generic | fisier nou `backend/internal/education/managerial_documents.go`, `education-config.ts` |
| W2-05 | Decizie si comunicare cap-coada | Emiterea/comunicarea deciziei se leaga de registratura si are statusuri clare | `backend/internal/education/decision_compliance_flows.go`, `frontend/.../education-config.ts` |
| W2-06 | Dashboard conformitate conducere | Directorul vede publicari restante, cerinte partiale si anonimizari in asteptare | `backend/internal/education/cockpit_service.go`, `frontend/.../education-director-dashboard.component.ts` |

### Acceptanta de val

- O cerinta de conformitate poate fi urmarita pana la documentul si publicarea aferenta.
- Documentele aprobate devin versiuni oficiale nemodificabile.
- Publicarea sensibila cere regula GDPR si audit.

## 6. Valul 3 - Personal, portofoliu profesor si evaluari self-service

Status: backendul este matur; lipsesc experiente dedicate pe rol si reguli contextuale stricte.

### Obiectiv

Profesorul isi opereaza propriile fluxuri, iar secretariatul/HR/conducerea au dosar personal procedural si verificari clare.

### Stories

| ID | Story | Acceptanta | File touchpoints probabile |
| --- | --- | --- | --- |
| W3-01 | `My portfolio` profesor (livrat) | Profesorul vede portofoliul propriu si actiunile de actualizare / verificare pe pagina dedicata | ruta `frontend/.../portfolio/education-portfolio-self-service.component.ts`, `frontend/.../portfolio/education-portfolio-workflow-wizard.component.ts`, `portfolio_documents.go` |
| W3-02 | Opis automat (livrat) | Opisul se genereaza din documentele portofoliului si poate fi exportat PDF | `backend/internal/education/portfolio_documents.go`, `governance_portfolio_flows.go`, `portfolio-record-wizard.component.ts` |
| W3-03 | Review portofoliu procedural | Secretariatul verifica, returneaza, valideaza; directorul vede exceptiile | `backend/internal/education/governance_portfolio_flows.go`, `education-director-dashboard.component.ts` |
| W3-04 | Dosar personal consolidat | HR/secretariat vad completitudine, lipsuri, acces recent si documente expirate intr-o vedere unica | `backend/internal/education/personnel_detail_flows.go`, `personnel_role_disciplinary_flows.go`, componenta noua `personnel-file-detail.component.ts` |
| W3-05 | Acces sensibil auditat | Fiecare acces la dosar sensibil creeaza eveniment cu motiv; lipsa motivului blocheaza accesul | `backend/internal/education/contextual_rbac.go`, `personnel_detail_flows.go` |
| W3-06 | Evaluare profesor procedurala | Evaluatorul si profesorul au pasi separati: autoevaluare, criterii, calificativ, contestatie, comunicare | `backend/internal/education/evaluation_detail_flows.go`, `evaluation_result_issue_flows.go`, `evaluation-record-wizard.component.ts` |
| W3-07 | Dashboard profesor si HR | Profesorul vede portofoliu/evaluare/sedinte; HR vede dosare incomplete si acces sensibil | componente noi in `frontend/src/app/features/education/dashboard/`, endpointuri noi sau extindere cockpit |

### Acceptanta de val

- Profesorul nu poate lista sau deschide dosarele altor profesori.
- Dosarul personal sensibil este accesibil doar cu permisiune, context si audit.
- Evaluarile si portofoliile au timeline procedural, nu doar taburi CRUD.

## 7. Valul 4 - Elevi, clase, diriginte, consiliul clasei

Status: verticala mare de pornit.

### Obiectiv

Sa existe model operational pentru elevi, clase, diriginte, consiliul clasei si situatie scolara de baza.

### Stories

| ID | Story | Acceptanta | File touchpoints probabile |
| --- | --- | --- | --- |
| W4-01 | Model clase/elevi | Clasele, elevii, inscrierile, transferurile si retragerile au tabele, endpointuri si permisiuni | migrari noi `006x_education_students_classes.sql`, fisiere noi `student_models.go`, `student_flows.go` |
| W4-02 | Alocare diriginte | Dirigintele este context pe clasa/an scolar si vede doar clasa lui | `contextual_rbac.go`, migrari rol/pozitie, dashboard diriginte |
| W4-03 | Dosar elev | Dosarul elevului are documente, status, responsabil si acces sensibil | backend nou `student_file_flows.go`, frontend nou `students/` |
| W4-04 | Consiliul clasei | Sedinte, componenta, masuri, termene si proces-verbal pe clasa | migrari noi `education_class_councils`, backend nou `class_council_flows.go`, frontend nou `class-council/` |
| W4-05 | Absente, sanctiuni, recompense, risc | Exista inregistrari minime, agregari si dashboard diriginte | backend nou, `education-director-dashboard.component.ts`, dashboard nou diriginte |
| W4-06 | Rapoarte pe clasa | Situatie clasa, absente, masuri, risc educational exportabile | `backend/internal/education/export.go` sau fisier raport dedicat, `education-standard-reports.component.ts` |

### Acceptanta de val

- Dirigintele vede si opereaza doar clasa alocata.
- Directorul vede agregat riscurile si absentele, fara sa caute manual in registre.
- Consiliul clasei produce proces-verbal si masuri urmaribile.

## 8. Valul 5 - Comisii si responsabilitati institutionale

Status: exista referinte dispersate, dar nu modul explicit.

### Obiectiv

Comisiile devin entitati de prim rang cu membri, plan, sedinte, documente si raportare.

### Stories

| ID | Story | Acceptanta | File touchpoints probabile |
| --- | --- | --- | --- |
| W5-01 | Model comisii | Comisie, membri, mandat, responsabil, domeniu si status au schema proprie | migrari noi `006x_education_committees.sql`, backend nou `committee_models.go`, `committee_flows.go` |
| W5-02 | Constituire comisie | Directorul/secretariatul pot constitui comisie si genera decizie/registru | `committee_flows.go`, `governance_documents.go`, frontend `committees/` |
| W5-03 | Plan activitate si sedinte | Responsabilul comisiei opereaza planul si sedintele doar pentru comisia sa | `contextual_rbac.go`, `committee_flows.go`, dashboard responsabil comisie |
| W5-04 | Raport comisie | Raportul se genereaza si se leaga de documente manageriale/conformitate | `managerial_regulation_flows.go`, `decision_compliance_flows.go`, `committee_documents.go` |
| W5-05 | Dashboard responsabil comisie | Responsabilul vede termene, rapoarte restante si sedinte ale comisiei sale | frontend dashboard nou, endpoint agregat nou |

### Acceptanta de val

- Un responsabil comisie nu poate edita alta comisie.
- Comisia are traseu documentar complet: constituire, plan, sedinte, raport.
- Rapoartele pot fi folosite ca dovezi de conformitate.

## 9. Valul 6 - Rapoarte standard, inspectie si hardening final

Status: exista export generic, dashboard director si pagina initiala de rapoarte; lipseste pachetul complet.

### Obiectiv

Directorul si inspectorul pot produce rapoarte standard si pachet de inspectie auditabil fara colectare manuala.

### Stories

| ID | Story | Acceptanta | File touchpoints probabile |
| --- | --- | --- | --- |
| W6-01 | Catalog rapoarte standard | Fiecare domeniu major are raport operational si managerial cu filtre comune | `backend/internal/education/report_models.go`, `report_service.go`, `education-standard-reports.component.ts` |
| W6-02 | XLSX pentru rapoarte tabelare | Rapoartele tabelare importante au export XLSX, nu doar CSV/PDF | backend export nou, frontend action buttons |
| W6-03 | Pachet inspectie | Selectare domenii/perioada, generare ZIP, inventar PDF, jurnal export | `backend/internal/education/inspection_package.go`, migrari noi pentru jurnal, frontend `inspection-package.component.ts` |
| W6-04 | Dashboard inspector | Inspectorul vede pachetele aprobate, lipsurile si observatiile deschise | endpoint agregat inspector, dashboard component nou |
| W6-05 | Performanta agregari | Dashboardurile si rapoartele folosesc query-uri agregate si indexuri unde este nevoie | migrari indexuri, `cockpit_service.go`, `report_service.go` |
| W6-06 | Contracte negative | Scenariile `403`, status invalid, export sensibil fara scop si acces cross-institution sunt acoperite in teste cand se cere rularea | teste backend/frontend viitoare, fara rulare automata in aceasta sesiune |

### Acceptanta de val

- Pachetul de inspectie este limitat, aprobat si auditat.
- Rapoartele standard au aceleasi filtre ca dashboardurile.
- Exporturile sensibile nu se pot face fara permisiune si scop.

## 10. Ordine recomandata de executie

1. W0-01 pana la W0-05, pentru a evita rework RBAC.
2. W1-01 si W1-02, pentru ca guvernanta blocheaza multe documente finale.
3. W1-03 si W1-06, ca directorul sa primeasca documente oficiale si rapoarte.
4. W2-01 pana la W2-04, pentru conformitate si documente manageriale.
5. W3-01 pana la W3-05, pentru self-service profesor si dosar personal.
6. W4, apoi W5, deoarece clasele/dirigintii sunt mai fundamentale operational decat comisiile.
7. W6, dupa ce verticalele produc datele necesare pachetului de inspectie.

## 11. Definition of ready pentru o story

- Actorul principal este clar.
- Permisiunea globala si regula contextuala sunt precizate.
- Endpointurile sau fisierele principale sunt identificate.
- Exista cel putin un criteriu de acceptanta observabil.
- Daca story-ul produce document, formatul si naming-ul sunt definite.
- Daca story-ul atinge date sensibile, auditul este in scope.

## 12. Definition of done pentru o verticala

- Fluxul principal poate fi parcurs cap-coada din UI.
- Backendul aplica aceleasi reguli ca UI-ul.
- Actiunile sensibile sunt auditate.
- Dashboardul relevant are minim o alerta sau lista actionabila.
- Exista raport/export minim pentru operare.
- Rolurile contextuale nu primesc acces global accidental.

## 13. Actualizare de progres - 2026-06-24

Progres adaugat in aceasta iteratie:

- guvernanta:
  - sumar procedural in dialogul de detaliu pentru sedinte;
  - actiuni recomandate pe blocaje;
  - tranzitii procedurale `scheduled -> held -> published`;
  - export PDF pentru sumar procedural;
  - PDF dedicat pentru elemente de proces-verbal si hotarari;
- dashboarduri:
  - cockpit secretariat livrat;
  - cockpit conformitate livrat;
  - cockpit profesor livrat;
- testare:
  - E2E pentru cockpit secretariat;
  - E2E pentru cockpit conformitate;
  - E2E pentru cockpit profesor;
  - E2E pentru sumar procedural, tranzitii si PDF-uri de guvernanta.

Backlog ramas imediat, in ordinea recomandata:

1. helperi backend comuni pentru RBAC contextual si aplicare uniforma;
2. cockpituri noi pentru evaluator si diriginte;
3. PDF-uri oficiale pentru documente manageriale si regulamente;
4. timeline procedural dedicat pentru evaluari, mobilitate si portofolii;
5. dosar personal procedural cu completitudine si audit acces sensibil.
