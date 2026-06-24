# Audit implementare education

Data: 2026-06-24

Document corelat cu:

- [docs/education-management-specifications.md](/E:/dev/egueducation/docs/education-management-specifications.md)
- [docs/education-current-plan.md](/E:/dev/egueducation/docs/education-current-plan.md)
- [docs/rbac-matrix.md](/E:/dev/egueducation/docs/rbac-matrix.md)

## 1. Scop

Acest document este un audit tehnic al implementarii existente pentru modulul `education`, realizat direct in codul backend, frontend si migrari.

Obiective:

- sa identificam ce exista deja efectiv;
- sa separem clar fundatia reala de ceea ce este doar documentat;
- sa evaluam gradul de acoperire fata de specificatiile functionale;
- sa stabilim ce trebuie pastrat, extins sau rescris procedural;
- sa producem o matrice clara `implementat / partial / lipsa`.

## 2. Concluzie executiva

Starea actuala este buna la nivel de fundatie de date si API, dar inca incompleta la nivel de experienta procedurala.

Ce este puternic deja:

- model de date extins pe multe subdomenii educationale;
- endpointuri REST reale, nu mock-uri;
- validari business pe multe subfluxuri;
- audit logging pe operatii importante;
- tenant scoping institutional;
- RBAC destul de bogat pentru module si submodule;
- export CSV/PDF generic;
- PDF-uri dedicate pentru evaluari, mobilitate, gradatii si portofolii;
- dashboarduri agregate simple pe fiecare domeniu;
- frontend functional, cu tabele server-side, filtre, detalii pe subresurse si actiuni CRUD.

Ce lipseste structural:

- wizard-uri procedurale dedicate pe rol si pe flux;
- cockpit managerial real pentru director;
- dashboarduri operationale specializate pe rol;
- fluxuri CA/CP complet conduse procedural, cu cvorum calculat si inchidere asistata;
- flux profesor "my workspace" pentru portofoliu si evaluare;
- roluri contextuale fine pentru membru CA/CP/comisie, nu doar roluri globale;
- documente PDF dedicate pentru guvernanta si documente manageriale;
- consiliul clasei si comisiile ca module explicite;
- rapoarte standard dedicate si pachete de inspectie;
- matrice completa de conformitate cerinta -> document -> status -> dovada.

Verdict:

- backendul este mai avansat decat frontendul;
- UI-ul actual este in principal un strat generic peste resurse bogate;
- proiectul nu trebuie repornit, ci trecut din faza "registre configurabile + CRUD avansat" in faza "procese ghidate si cockpit operational".

## 3. Ce am auditat

### Backend

Fisiere cheie citite:

- `backend/internal/education/service.go`
- `backend/internal/education/models.go`
- `backend/internal/education/detail_records.go`
- `backend/internal/education/procedural_flows.go`
- `backend/internal/education/governance_portfolio_flows.go`
- `backend/internal/education/managerial_regulation_flows.go`
- `backend/internal/education/personnel_detail_flows.go`
- `backend/internal/education/personnel_role_disciplinary_flows.go`
- `backend/internal/education/evaluation_detail_flows.go`
- `backend/internal/education/evaluation_documents.go`
- `backend/internal/education/evaluation_result_issue_flows.go`
- `backend/internal/education/mobility_merit_flows.go`
- `backend/internal/education/mobility_merit_outcomes.go`
- `backend/internal/education/mobility_merit_documents.go`
- `backend/internal/education/portfolio_documents.go`
- `backend/internal/education/export.go`

### Frontend

Fisiere cheie citite:

- `frontend/src/app/features/education/education.routes.ts`
- `frontend/src/app/features/education/education-shell.component.ts`
- `frontend/src/app/features/education/dashboard/education-dashboard.component.ts`
- `frontend/src/app/features/education/shared/education-config.ts`
- `frontend/src/app/features/education/shared/education-domain-workspace.component.ts`
- `frontend/src/app/features/education/governance/education-governance-page.component.ts`
- `frontend/src/app/features/education/personnel/education-personnel-page.component.ts`
- `frontend/src/app/features/education/portfolio/education-portfolio-page.component.ts`
- `frontend/src/app/features/education/compliance/education-compliance-page.component.ts`

### Baza de date si RBAC

Migrari cheie citite:

- `backend/internal/db/migrations/0032_education_decisions_managerial.sql`
- `backend/internal/db/migrations/0033_education_regulations_evaluations.sql`
- `backend/internal/db/migrations/0036_education_requirements_portfolio_sections.sql`
- `backend/internal/db/migrations/0037_education_governance_portfolio_documents.sql`
- `backend/internal/db/migrations/0038_education_votes_checklist.sql`
- `backend/internal/db/migrations/0039_education_memberships_resolutions_transfers_reviews.sql`
- `backend/internal/db/migrations/0048_education_minutes_opis_custody_publication.sql`
- `backend/internal/db/migrations/0049_education_managerial_regulation_flows.sql`
- `backend/internal/db/migrations/0050_education_personnel_files_and_appeals.sql`
- `backend/internal/db/migrations/0051_education_mobility_merit_subflows.sql`
- `backend/internal/db/migrations/0052_education_decision_compliance_rbac.sql`
- `backend/internal/db/migrations/0053_education_mobility_merit_outcomes.sql`
- `backend/internal/db/migrations/0054_education_personnel_assignments_disciplinary.sql`
- `backend/internal/db/migrations/0055_education_evaluation_self_reviews_and_criteria.sql`
- `backend/internal/db/migrations/0056_education_evaluation_qualification_and_result_issues.sql`

## 4. Arhitectura actuala

### 4.1. Backend

Backendul este organizat pe resurse educationale destul de clare:

- guvernanta: sedinte, membri, participanti, documente, voturi, minute, hotarari;
- decizii si publicare;
- dosare manageriale si regulamente;
- personal, dosar personal, acces, atributii, disciplinar;
- evaluari, autoevaluari, criterii, contestatii, comunicari;
- declaratii;
- mobilitate;
- gradatii de merit;
- portofolii, opis, checklist, custodie, transfer, review;
- cerinte si publicari de conformitate.

Modelul folosit este:

- endpointuri CRUD paginated;
- subresurse pe detaliu;
- validari pe request;
- log audit pe create/update/delete;
- unele sincronizari intre module.

### 4.2. Frontend

Frontendul are un singur mecanism principal:

- o pagina pe domeniu;
- o configuratie centrala de resurse;
- un workspace generic care randaza tabele server-side;
- dialoguri generice pentru create/edit/delete;
- taburi pentru subresursele din detaliu;
- export generic CSV/PDF;
- dashboard sumar cu carduri numerice.

Acest design este bun pentru viteza de livrare, dar nu este suficient pentru procesele educationale complexe cerute.

## 5. Ce este implementat bine

### 5.1. Guvernanta institutionala

Status: `partial avansat`

Implementat:

- entitati pentru sedinte: `education_meetings`;
- componenta organismelor: `education_governance_memberships`;
- participanti sedinta: `education_meeting_participants`;
- documente sedinta: `education_meeting_documents`;
- voturi sedinta: `education_meeting_votes`;
- minute: `education_meeting_minutes`;
- hotarari/rezolutii: `education_meeting_resolutions`;
- dashboard sumar pentru sedinte;
- filtre, sortare, paginare si CRUD;
- validari pentru:
  - tip organism;
  - tip sedinta;
  - statusuri;
  - tip participant;
  - prezenta;
  - tip hotarare;
  - status publicare;
  - anonimizare;
- audit logging.

Observatii:

- modelul de date este suficient de bogat pentru a sustine un wizard CA/CP;
- UI-ul actual pentru guvernanta este inca tabelar si generic;
- cvorumul este camp configurabil, nu calcul procedural pe componenta;
- nu exista inchidere de sedinta asistata;
- nu exista agenda formalizata ca entitate separata;
- nu exista motor procedural pentru "deschidere sedinta -> verificare cvorum -> vot -> proces-verbal final".

### 5.2. Decizii si conformitate

Status: `implementat bine, dar orientat pe registre`

Implementat:

- `education_decisions`;
- `education_decision_issuances`;
- `education_decision_publication_steps`;
- `education_publications`;
- RBAC separat pentru emitere si conformitate;
- UI pentru decizii, emitere si publicare;
- filtre si statusuri pentru publicare;
- audit logging.

Observatii:

- domeniul este bine pregatit pentru trasabilitate;
- lipseste flux procedural complet "emitere -> avizare -> anonimizare -> publicare";
- frontendul nu expune inca o harta clara a dependintelor intre cerinte, documente si publicare.

### 5.3. Documente manageriale si regulamente

Status: `implementat bine la nivel de model`

Implementat:

- `education_managerial_dossiers`;
- `education_managerial_documents`;
- `education_managerial_workflow_steps`;
- `education_regulations`;
- `education_regulation_versions`;
- `education_regulation_workflow_steps`;
- dashboarduri sumare;
- workflow-uri cu etape precum `elaborare`, `verificare_secretariat`, `avizare_cp`, `aprobare_ca`, `publicare`, `arhivare`;
- UI configurat pentru dosare, documente, versiuni si workflow.

Observatii:

- exista fundatia corecta pentru PDI/PAS, RAEI, plan managerial si ROF/ROI;
- lipsesc sabloane de document si experienta ghidata pe tip de document;
- lipsesc PDF-uri dedicate pentru aceste documente;
- lipseste publicarea ca experienta cap-coada pentru documentele manageriale.

### 5.4. Personal si dosar personal

Status: `implementat bine`

Implementat:

- `education_personnel`;
- `education_personnel_file_documents`;
- `education_personnel_access_events`;
- `education_personnel_assignments`;
- `education_personnel_disciplinary_cases`;
- distinctie de scope dosar: `dosar_personal`, `dosar_director`, `dosar_director_adjunct`;
- confidentialitate pe documente;
- jurnal de acces;
- UI cu subresurse dedicate.

Observatii:

- domeniul este mai matur decat pare din UI;
- exista o baza buna pentru dosarul personal si acces sensibil;
- lipseste o pagina consolidata de tip "dosar personal" cu vedere coerenta, nu doar resurse separate;
- lipsesc fluxuri ghidate pentru secretariat si director pe acces, arhivare si verificare completitudine.

### 5.5. Evaluari

Status: `implementat avansat`

Implementat:

- `education_evaluations`;
- `education_evaluation_self_reviews`;
- `education_evaluation_criteria`;
- `education_evaluation_appeals`;
- `education_evaluation_result_issues`;
- calificativ derivat din punctaj;
- sincronizare automata a punctajului total din criterii;
- sincronizare evaluare in dosarul personal;
- sincronizare contestatii in dosarul personal, cu confidentialitate ridicata;
- propagare status evaluare in functie de contestatii;
- PDF pentru evaluare, contestatie si comunicare rezultat;
- subresurse frontend pentru autoevaluari, criterii, contestatii si comunicari.

Observatii:

- acesta este unul dintre cele mai solide submodule;
- exista logica reala inter-modul, nu doar CRUD;
- lipseste un wizard de evaluare anuala pentru evaluator si profesor;
- lipseste experienta speciala pentru personal in mai multe unitati;
- exportul Excel specializat nu este implementat distinct, doar CSV generic.

### 5.6. Portofoliul profesional al cadrului didactic

Status: `implementat avansat ca fundatie`

Implementat:

- `education_portfolios`;
- `education_portfolio_sections`;
- `education_portfolio_documents`;
- `education_portfolio_checklist`;
- `education_portfolio_opis`;
- `education_portfolio_custody`;
- `education_portfolio_transfers`;
- `education_portfolio_reviews`;
- dashboard sumar;
- validari pentru opis, custodie, transfer, review;
- PDF principal de portofoliu;
- functii auxiliare PDF pentru opis, checklist, custodie, transfer si review;
- structura-cadru a portofoliului si catalog de cerinte.

Observatii:

- modelul este bun si chiar peste medie pentru acest domeniu;
- exista distinctii utile: autenticitate, consimtamant, retentie, transfer, custodie;
- exista experiența "my portfolio" pentru profesor, cu wizard ghidat de actualizare si trimitere la verificare;
- opisul este regenerat automat si expus explicit in UI;
- lipsesc in continuare reguli institutionale configurabile mai bogate pe unitate si rol;
- review-ul exista ca resursa, dar mai raman extinderi de orchestrare procedurala pe alte contexte.

### 5.7. Mobilitate si gradatii

Status: `implementat bine`

Implementat:

- dosar principal;
- documente;
- punctaje;
- contestatii;
- decizii finale;
- comunicari rezultate;
- sincronizari de outcome;
- PDF-uri dedicate pe dosar si subfluxuri;
- UI cu subresurse complete.

Observatii:

- aceste submodule par printre cele mai "workflow-like" din implementarea actuala;
- in continuare frontendul le trateaza tot ca resurse in workspace generic;
- ar beneficia de wizard-uri, dar nu sunt primul gap critic fata de cerintele discutate.

## 6. Ce este doar partial

### 6.1. Dashboard / cockpit

Status: `partial`

Implementat:

- carduri numerice pe domenii in `education-dashboard.component.ts`;
- endpointuri dashboard pe mai multe domenii.

Lipseste:

- cockpit director cu termene, riscuri, sedinte fara cvorum, documente nepublicate, portofolii incomplete, evaluari contestate;
- dashboard profesor;
- dashboard secretariat/registratura;
- dashboard inspector;
- widgeturi operationale, nu doar KPI-uri brute.

Concluzie:

- exista sumar statistic;
- nu exista cockpit managerial.

### 6.2. RBAC contextual

Status: `partial`

Implementat:

- roluri globale: `director`, `profesor`, `secretar`, `registrator`, `inspector`, `gdpr_officer`, `admin`, `super_admin`;
- permisiuni pe module si submodule;
- mapari rol-permisiune si pozitie-rol.

Lipseste:

- permisiuni contextuale pentru membru CA al unei anumite sedinte;
- membru CP pentru un anumit an scolar;
- responsabil comisie pe o comisie anume;
- profesor care isi vede propriul portofoliu, fara acces la toate portofoliile;
- inspector pe pachet de inspectie limitat;
- actiuni per workflow: inchidere sedinta, validare cvorum, publicare hotarare, verificare portofoliu, transfer portofoliu.

Concluzie:

- RBAC-ul este bun ca baza, dar insuficient pentru experienta pe roluri reale.

### 6.3. Exporturi si PDF

Status: `partial`

Implementat:

- export CSV generic;
- export PDF generic pentru tabele;
- PDF-uri dedicate pentru evaluari, mobilitate, gradatii si portofolii.

Lipseste:

- PDF proces-verbal CA;
- PDF proces-verbal CP;
- PDF hotarare CA;
- PDF convocator CA/CP;
- PDF grafic tematica;
- PDF document managerial;
- pachet ZIP de inspectie.

Concluzie:

- exportul generic este util, dar insuficient pentru documente oficiale.

### 6.4. Conformitate

Status: `partial`

Implementat:

- catalog de cerinte;
- resursa de publicari;
- publicare si anonimizare pe decizii;
- dashboarduri si filtre partiale.

Lipseste:

- matrice operationala completa `cerinta -> document -> flux -> responsabil -> status -> dovada`;
- rapoarte de lipsuri;
- conformitate pe documente manageriale, nu doar pe publicari;
- pachet de audit / inspectie.

## 7. Ce lipseste aproape complet

### 7.1. Wizard-uri educationale

Nu exista in frontend:

- wizard constituire CA;
- wizard sedinta CA;
- wizard sedinta CP;
- wizard procedura interna portofolii;
- wizard evaluare anuala profesor;
- wizard PDI/PAS;
- wizard raport anual al calitatii;
- wizard RAEI;
- wizard consiliul clasei;
- wizard pachet inspectie.

Observatie:

- exista deja constructii de tip `wizard` si `stepper` in modulul `education`, dar numai pentru anumite fluxuri; celelalte ramase in lista de mai sus nu sunt inca implementate.

### 7.2. Consiliul clasei

Status: `lipsa`

Nu am identificat:

- tabele dedicate;
- modele;
- endpointuri;
- UI;
- fluxuri;
- documente generate.

### 7.3. Comisii educationale explicite

Status: `partial spre lipsa`

Exista urme de comisii in:

- raport de comisie;
- `committee_name` in unele subfluxuri;
- assignment-uri de tip `responsabil_comisie`, `membru_comisie`.

Nu exista ca modul explicit:

- `Committee`;
- `CommitteeMember`;
- `CommitteeReport`;
- workflow constituire comisie;
- raportare standard per comisie.

### 7.4. Dashboarduri pe rol

Status: `lipsa`

Nu exista ecrane separate pentru:

- director;
- profesor;
- secretariat;
- inspector.

### 7.5. Flux profesor self-service

Status: `partial`

Profesorul are deja:

- pagina "portofoliul meu";
- wizard de actualizare si validare pentru portofoliu.

Mai lipsesc:

- pagina "evaluarea mea";
- checklist personal generalizat;
- termene personale;
- notificari si completari pentru toate fluxurile personale.

## 8. Matrice pe domenii

| Domeniu | Backend | Frontend | Grad acoperire | Observatie |
| --- | --- | --- | --- | --- |
| Guvernanta CA/CP | Bun | Generic | Partial avansat | Exista modele si subresurse, lipseste wizard procedural |
| Decizii si publicare | Bun | Generic | Bun spre partial | Exista fluxuri de emitere/publicare, lipseste experienta cap-coada |
| Documente manageriale | Bun | Generic | Partial avansat | Exista dosare, documente, workflow; lipsesc sabloane si PDF-uri dedicate |
| Regulamente | Bun | Generic | Partial avansat | Exista versiuni si workflow; lipseste UX procedural dedicat |
| Personal | Bun | Generic | Bun | Domeniul este solid, dar lipseste vedere consolidata de dosar |
| Evaluari | Foarte bun | Generic cu subresurse | Bun | Unul dintre cele mai mature submodule |
| Declaratii | Bun | Generic | Partial | Exista domeniu, dar nu este integrat intr-un flux profesor clar |
| Portofolii CD | Foarte bun | Generic | Bun spre partial | Fundatie puternica, lipseste self-service si opis automatizat real |
| Mobilitate | Bun | Generic | Bun | Fluxurile exista, experienta poate fi rafinata procedural |
| Gradatii | Bun | Generic | Bun | Similar mobilitate |
| Conformitate | Partial bun | Generic | Partial | Exista catalog si publicari, lipseste matrice operationala |
| Consiliul clasei | Lipsa | Lipsa | Lipsa | Necesita modul nou |
| Comisii | Partial | Lipsa | Lipsa functionala | Exista doar referinte indirecte |
| Dashboard pe rol | Partial | Partial | Lipsa functionala | KPI exista, cockpit nu |

## 9. Observatii tehnice importante

### 9.1. Punct forte: sincronizari reale intre module

Evaluarile nu sunt izolate.

Exista logica de:

- recalcul punctaj total din criterii;
- derivare calificativ;
- oglindire document evaluare in dosarul personal;
- oglindire contestatii in dosarul personal;
- ajustare status evaluare in functie de contestatii;
- ajustare status personal.

Aceasta este o baza foarte buna pentru fluxuri procedurale mai mature.

### 9.2. Punct forte: modelul portofoliului este serios

Portofoliul nu este tratat superficial. Exista:

- sectiuni;
- documente;
- checklist;
- opis;
- custodie;
- transfer;
- review;
- retentie;
- consimtamant si autenticitate la nivel de record.

Problema nu este lipsa de model, ci lipsa de experienta ghidata.

### 9.3. Punct slab: frontendul este prea generic pentru domeniu

Practic toate paginile educationale sunt doar:

- `EducationDomainWorkspaceComponent` parametrizat cu configuratie;
- tabele;
- dialoguri generice;
- detalii pe taburi.

Aceasta arhitectura este buna pentru administrare interna, dar nu pentru procese educationale sensibile si frecvente.

### 9.4. Punct slab: PDF-ul generic este prea rudimentar

`backend/internal/education/export.go` genereaza PDF simplu bazat pe linii text.

Este util pentru export rapid, dar nu este suficient pentru:

- procese-verbale;
- hotarari;
- convocatoare;
- rapoarte institutionale;
- documente manageriale formale.

## 10. Gap analysis fata de specificatiile noi

### 10.1. Cerinte deja acoperite substantial

- registre si subregistre pe guvernanta;
- managementul deciziilor;
- publicare si anonimizare de baza;
- dosare manageriale si workflow;
- dosar personal si acces;
- evaluari, contestatii si comunicari;
- portofolii CD cu substructura bogata;
- mobilitate si gradatii;
- RBAC de baza;
- audit logging;
- dashboarduri numerice.

### 10.2. Cerinte acoperite partial

- roluri educationale: exista, dar nu complet contextual;
- dashboard management educational: exista doar sumar statistic;
- portofoliu profesional procedural: exista acum si ca model, si ca self-service / wizard; mai lipsesc reguli institutionale si contexte metodologice suplimentare;
- conformitate: exista partial;
- rapoarte standard: exista generic, nu dedicate;
- documente oficiale generate: exista partial.

### 10.3. Cerinte neacoperite

- wizard-uri pe rol;
- cockpit director;
- dashboard profesor;
- dashboard secretariat;
- dashboard inspector;
- consiliul clasei;
- comisii ca entitati de prim rang;
- pachete inspectie;
- rapoarte standard institutionale complete;
- matrice completa de trasabilitate a cerintelor;
- calcul procedural de cvorum si prag vot din componenta reala.

## 11. Recomandare de evolutie

### 11.1. Ce trebuie pastrat

- schema de date actuala;
- endpointurile actuale;
- logica de audit si sincronizare;
- taxonomiile si filtrele;
- structura pe subresurse;
- seed-urile si RBAC de baza.

### 11.2. Ce trebuie extins

- dashboardurile existente in cockpituri pe rol;
- resursele de guvernanta in wizard-uri;
- resursele de portofoliu in flux "my portfolio" si pe contexte metodologice suplimentare;
- documentele manageriale in template-uri ghidate;
- evaluarile in flux procedural per actor;
- conformitatea in matrice de cerinte si rapoarte.

### 11.3. Ce trebuie adaugat nou

- modul `class council`;
- modul `committees`;
- componente UI `wizard-stepper`, `summary review`, `document preview`, `role dashboard`;
- endpointuri agregate pentru cockpit;
- generatoare PDF oficiale dedicate;
- pachet de inspectie.

## 12. Prioritati recomandate

### Prioritate 1

- audit final al permisiunilor pe actiune;
- cockpit director;
- wizard sedinta CA;
- wizard sedinta CP;
- PDF proces-verbal si hotarare.

### Prioritate 2

- procedura interna portofolii;
- reguli institutionale pentru portofoliu;
- extensie pe contexte metodologice ale portofoliului;
- dashboard profesor;
- dashboard secretariat.

### Prioritate 3

- documente manageriale ghidate;
- PDI/PAS si raport calitate;
- rapoarte standard;
- publicare formala si conformitate extinsa.

### Prioritate 4

- consiliul clasei;
- comisii;
- pachet inspectie;
- dashboard inspector.

## 13. Verdict final

Implementarea actuala nu este un schelet. Este o fundatie functionala si relativ bogata, mai ales in backend.

Problema principala nu este lipsa de date sau de API, ci faptul ca:

- procesele educationale complexe sunt inca operate ca registre;
- rolurile reale nu sunt inca traduse in experiente dedicate;
- managementul educational nu are inca un cockpit operational;
- documentele oficiale si fluxurile de lucru nu sunt inca "primul cetatean" al UX-ului.

Concluzia practica este simpla:

- nu trebuie rescris ce exista;
- trebuie capitalizat ce exista si transformat in fluxuri ghidate, dashboarduri pe rol si documente oficiale dedicate.
