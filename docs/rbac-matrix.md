# Matrice RBAC education

Data audit: 2026-06-24

Acest document descrie modelul de roluri pentru modulul `education` si traseaza regulile care trebuie aplicate in cod. Scopul nu este doar UI gating, ci control real pe actiuni, resurse si context institutional.

## 1. Principii

- Rolurile globale dau acces la module si la operare larga in unitatea curenta.
- Rolurile contextuale dau drepturi doar pe o resursa concreta: sedinta, portofoliu, evaluare, comisie, clasa sau pachet de inspectie.
- Actiunile sensibile trebuie verificate in backend chiar daca butonul este ascuns in frontend.
- `admin` si `super_admin` pot administra platforma, dar fluxurile educationale trebuie sa ramana auditate ca actiuni institutionale.
- `profesor`, `diriginte`, `membru_ca`, `membru_cp`, `evaluator` si `responsabil_comisie` nu trebuie transformati automat in roluri globale cu acces la toate registrele.

## 2. Roluri globale

| Rol | Tip | Scop operational | Observatii de implementare |
| --- | --- | --- | --- |
| `super_admin` | global platforma | acces complet tehnic si operational | Folosit pentru suport si administrare platforma. |
| `admin` | global institutie | administrare operationala a institutiei | Poate primi toate permisiunile educationale, dar actiunile sensibile raman auditate. |
| `director` | global institutie | conducere, aprobare, inchidere fluxuri | Rol principal pentru cockpit, CA, documente manageriale, rapoarte sensibile. |
| `director_adjunct` | global institutie sau pozitie mapata | delegare conducere pe domenii | Trebuie seed-uit explicit daca devine rol global; altfel se modeleaza prin pozitie + permisiuni. |
| `secretar` | global compartiment | registratura, convocatoare, minute, dosare personal, publicari | Nu primeste automat drept de aprobare finala. |
| `registrator` | global compartiment | inregistrare si legare documente | Citeste documente operationale, gestioneaza registratura, nu aproba fluxuri educationale. |
| `hr` | global compartiment | personal, dosar personal, evaluari administrative | Exista ca rol seed-uit; trebuie legat in UI de zona personal/HR. |
| `profesor` | global limitat | self-service profesor si participare contextuala | Acces implicit la propriile fluxuri; acces la sedinte doar contextual. |
| `inspector` | global extern/control | citire controlata si pachet inspectie | Fara editare operationala in unitate. |
| `gdpr_officer` | global conformitate | GDPR, anonimizare, exporturi sensibile | Acces pe date personale si rapoarte sensibile cu audit. |
| `workflow_admin` | global platforma | configurare workflow | Nu inseamna automat operare education. |
| `arhivar` | global compartiment | e-archiva si retentie | Relevanta pentru portofolii, personal si documente finale dupa arhivare. |

## 3. Roluri contextuale

Acestea trebuie calculate din datele resursei, nu doar din `app_user_roles`.

| Actor contextual | Context sursa | Permite | Nu permite |
| --- | --- | --- | --- |
| `diriginte` | clasa activa / an scolar | dashboard diriginte, consiliul clasei, situatie clasa, masuri si rapoarte pe clasa | acces la toate clasele sau la dosare personale globale |
| `membru_ca` | `education_governance_memberships` pentru organism `CA` | citire sedinte CA unde este membru, confirmare prezenta, vot contextual | inchidere sedinta, publicare hotarare, editare componenta |
| `membru_cp` | componenta CP / personal didactic activ | citire sedinte CP, prezenta, vot contextual | acces la CA sau documente manageriale sensibile |
| `evaluator` | evaluare atribuita / comisie evaluare | completare criterii, propunere calificativ, observatii | citire globala evaluari neatribuite |
| `responsabil_comisie` | comisie activa / mandat | plan activitate, sedinte comisie, raport comisie | aprobari CA/CP sau acces la alte comisii |
| `custode_portofoliu` | custodie portofoliu | preluare, predare, verificare custodie | modificare continut profesor fara workflow |
| `auditor_inspectie` | pachet inspectie aprobat | citire pachet, descarcare inventar | export ad-hoc din toate registrele |

## 4. Scopuri de actiune

| Scop | Definitie | Exemple permisiuni |
| --- | --- | --- |
| `read_all` | citire registru la nivel de institutie | `education.governance.read`, `education.personnel.read` |
| `read_own` | citire inregistrari proprii sau atribuite | necesita helper contextual, nu doar permisiune globala |
| `manage` | creare/editare operationala in registru | `education.governance.manage`, `education.portfolios.manage` |
| `participate` | prezenta, vot, confirmare, observatii pe context | `education.governance.meeting.vote` + membership activ |
| `validate` | verificare, returnare, marcare completitudine | `education.portfolios.verify`, `education.personnel.files.manage` |
| `approve` | avizare/aprobare/inchidere etapa | `education.governance.meeting.close`, `education.managerial.publish` |
| `publish` | publicare minute, hotarari, documente | `education.governance.minutes.publish`, `education.governance.resolution.publish` |
| `export_standard` | export operational fara date sensibile speciale | permisiunea read pe domeniu |
| `export_sensitive` | export cu date personale, inspectie sau arhiva sensibila | `education.reports.export_sensitive` |
| `inspect` | citire controlata pentru inspectie | `education.inspect.package.read` |

## 5. Acces pe workspace

| Workspace / ruta | Roluri principale | Conditie minima |
| --- | --- | --- |
| autentificare, profil, consimtamant OIDC | orice utilizator autentificat | sesiune valida |
| `documente` / registratura | `admin`, `super_admin`, `director`, `secretar`, `registrator` | `registratura.read` |
| `education/dashboard` generic | toate rolurile educationale | cel putin o permisiune `education.*.read` sau `education.read` |
| `education/dashboard/director` | `director`, `director_adjunct`, `admin`, `super_admin` | `education.read` sau una dintre permisiunile de citire majore; recomandat filtru pe rol conducere in UI |
| `education/governance` | `director`, `director_adjunct`, `secretar`, `profesor` contextual, `inspector` read-only | `education.governance.read` sau context sedinta |
| `education/personnel` | `director`, `director_adjunct`, `secretar`, `hr`, `gdpr_officer` | `education.personnel.read`; datele sensibile cer context/audit |
| `education/portfolio` | `director`, `secretar`, `profesor` propriu, `custode_portofoliu`, `inspector` read-only | `education.portfolios.read` sau `read_own` |
| `education/compliance` | `director`, `secretar`, `gdpr_officer`, `inspector` | `education.compliance.read` |
| `admin` | `admin`, `super_admin`, partial `director` | permisiuni `admin.*` |
| `gdpr` | `admin`, `super_admin`, `director`, `gdpr_officer` | `gdpr.read` |

## 6. Matrice rol -> domenii

Legenda: `R` citire, `M` gestionare, `C` contextual, `A` aprobare/publicare, `S` sensibil/export, `-` fara acces implicit.

| Domeniu | super_admin/admin | director | director_adjunct | secretar | registrator | hr | profesor | inspector | gdpr_officer |
| --- | --- | --- | --- | --- | --- | --- | --- | --- | --- |
| Guvernanta CA/CP | R/M/A/S | R/M/A/S | R/M/A delegat | R/M, A pe minute/publicari delegate | R legaturi registratura | - | C participare/vot | R inspectie | R pe anonimizare |
| Decizii si comunicari | R/M/A/S | R/M/A/S | R/M/A delegat | R/M publicare/comunicare | R/M registratura | R legat personal | C destinatar | R inspectie | R/S anonimizare |
| Documente manageriale | R/M/A/S | R/M/A/S | R/M/A delegat | R/M publicare | R legaturi | - | C consultare unde public | R inspectie | R/S publicare date |
| Regulamente | R/M/A/S | R/M/A/S | R/M/A delegat | R/M publicare | R legaturi | - | R publicat | R inspectie | R/S |
| Personal si dosar personal | R/M/S | R/M/S | R/M delegat | R/M operational | R limitat | R/M/S | C propriu | R inspectie controlata | R/S |
| Evaluari | R/M/S | R/M/A/S | R/M/A delegat | R/M suport | - | R/M suport | C propriu/evaluator | R inspectie | R/S |
| Declaratii | R/M/S | R/M/S | R/M delegat | R/M suport | - | R/M suport | C propriu | R inspectie | R/S |
| Portofolii | R/M/S | R/M/A/S | R/M/A delegat | R/M verificare/custodie | R legaturi | - | C propriu | R inspectie | R/S |
| Mobilitate | R/M/S | R/M/A/S | R/M/A delegat | R/M suport | - | R/M suport | C propriu | R inspectie | R/S |
| Gradatie merit | R/M/S | R/M/A/S | R/M/A delegat | R/M suport | - | R/M suport | C propriu | R inspectie | R/S |
| Clase/elevi/diriginte | R/M/S | R/M/A/S | R/M delegat | R/M administrativ | R legaturi | - | C clasa/disciplina | R inspectie | R/S limitat |
| Comisii | R/M/A/S | R/M/A/S | R/M/A delegat | R/M suport | - | - | C membru/responsabil | R inspectie | R/S unde date personale |
| Rapoarte standard | toate | toate | delegate | operationale | registratura | personal | proprii/context | inspectie | GDPR |
| Export sensibil | toate | da | delegat | doar operational aprobat | nu implicit | personal aprobat | nu implicit | doar pachet inspectie | da |

## 7. Matrice actiuni sensibile

| Actiune | Permisiune globala | Regula contextuala obligatorie | Roluri care pot trece regula |
| --- | --- | --- | --- |
| Inchide sedinta CA/CP | `education.governance.meeting.close` | sedinta in institutia curenta, status inchidabil, cvorum si documente minime validate | `director`, `director_adjunct` delegat, `admin`, `super_admin` |
| Inregistreaza vot | `education.governance.meeting.vote` | utilizatorul este membru activ, participant sau secretar desemnat pentru sedinta | `membru_ca`, `membru_cp`, `secretar`, `director`, `profesor` contextual |
| Publica minute | `education.governance.minutes.publish` | minute finale, sedinta inchisa sau in etapa publicabila, actor desemnat | `director`, `secretar`, `director_adjunct` delegat |
| Publica hotarare | `education.governance.resolution.publish` | hotarare aprobata si anonimizare marcata unde este cazul | `director`, `secretar`, `gdpr_officer` pentru aviz anonimizare |
| Emite/comunica decizie | `education.decisions.issuance.manage` | decizia aprobata, canal valid, destinatar si registratura completate | `director`, `secretar` |
| Publica document managerial | `education.managerial.publish` | workflow finalizat, versiune aprobata, cerinta de conformitate legata | `director`, `director_adjunct` delegat, `secretar` operational |
| Verifica portofoliu | `education.portfolios.verify` | actorul este secretariat, director, evaluator desemnat sau responsabil procedura | `director`, `secretar`, `responsabil_comisie`, `custode_portofoliu` |
| Transfera portofoliu | `education.portfolios.transfer` | cerere valida, destinatie si custodie completate, audit obligatoriu | `director`, `secretar`, `custode_portofoliu` |
| Gestioneaza custodie | `education.portfolios.custody.manage` | portofoliu existent, termen de retentie, responsabil nominalizat | `secretar`, `arhivar`, `director` |
| Acceseaza dosar personal sensibil | `education.personnel.files.read` sau `education.personnel.access.read` | motiv acces, scope dosar, jurnal de acces scris | `director`, `secretar`, `hr`, `gdpr_officer`, subiectul pentru propriul dosar |
| Exporta raport sensibil | `education.reports.export_sensitive` | raport marcat sensibil, scop export, log audit | `director`, `gdpr_officer`, `inspector` doar pachet, `admin`, `super_admin` |
| Citeste pachet inspectie | `education.inspect.package.read` | pachet aprobat pentru perioada/unitatea curenta | `inspector`, `director`, `admin`, `super_admin` |

## 8. Reguli contextuale de implementat

### 8.1. Proprietate proprie

Un utilizator cu `profesor` poate citi si opera doar:

- portofoliile unde `owner_user_id`, emailul sau codul personal se potriveste cu utilizatorul curent;
- evaluarile proprii unde este subiect al evaluarii;
- declaratiile proprii;
- sarcinile sau workflow-urile atribuite lui.

Nu este suficient ca utilizatorul sa aiba `education.read`.

### 8.2. Sedinte si organisme

Pentru `meeting.vote`, `meeting.read_contextual` si prezenta:

- sedinta trebuie sa apartina institutiei curente;
- utilizatorul trebuie sa fie in componenta organismului sau participant nominalizat;
- mandatul trebuie sa acopere data sedintei;
- `voting_right = true` este necesar pentru vot decizional;
- secretarul poate inregistra voturi doar daca este secretar desemnat sau are delegare operationala.

### 8.3. Delegare director adjunct

`director_adjunct` trebuie tratat ca delegare limitata:

- fie rol global seed-uit cu permisiuni explicite;
- fie pozitie in `app_position_roles` plus permisiuni directe;
- fie relatie de delegare pe domeniu, daca se introduce tabel dedicat.

Nu se recomanda maparea implicita `director_adjunct -> director` fara limite, pentru ca ar supradimensiona accesul la dosare sensibile si exporturi.

### 8.4. Inspector

Inspectorul nu lucreaza in registrele operationale. Accesul corect este:

- citire agregata;
- pachet de inspectie aprobat;
- export controlat;
- fara create/update/delete pe fluxurile scolii.

### 8.5. GDPR si exporturi

`gdpr_officer` poate vedea si aviza date sensibile pentru anonimizare, export si drepturi persoana vizata. Nu devine automat operator de flux educational.

## 9. Permisiuni existente relevante

Permisiuni de domeniu:

- `education.read`
- `education.governance.read`
- `education.governance.manage`
- `education.decisions.read`
- `education.decisions.manage`
- `education.decisions.issuance.read`
- `education.decisions.issuance.manage`
- `education.managerial.read`
- `education.managerial.manage`
- `education.managerial.publish`
- `education.regulations.read`
- `education.regulations.manage`
- `education.compliance.read`
- `education.compliance.manage`
- `education.personnel.read`
- `education.personnel.manage`
- `education.personnel.files.read`
- `education.personnel.files.manage`
- `education.personnel.access.read`
- `education.personnel.access.manage`
- `education.evaluations.read`
- `education.evaluations.manage`
- `education.declarations.read`
- `education.declarations.manage`
- `education.portfolios.read`
- `education.portfolios.manage`
- `education.portfolios.verify`
- `education.portfolios.transfer`
- `education.portfolios.custody.manage`
- `education.mobility.read`
- `education.mobility.manage`
- `education.gradatii.read`
- `education.gradatii.manage`
- `education.governance.meeting.close`
- `education.governance.meeting.vote`
- `education.governance.minutes.publish`
- `education.governance.resolution.publish`
- `education.reports.export_sensitive`
- `education.inspect.package.read`

Permisiuni care trebuie adaugate cand pornesc verticalele noi:

- `education.classes.read`
- `education.classes.manage`
- `education.students.read`
- `education.students.manage`
- `education.class_council.read`
- `education.class_council.manage`
- `education.class_council.measures.manage`
- `education.committees.read`
- `education.committees.manage`
- `education.committees.report.publish`
- `education.inspection.package.manage`

## 10. Mapari recomandate rol -> permisiuni

| Rol | Permisiuni recomandate |
| --- | --- |
| `director` | toate permisiunile `education.*.read`, manage pe guvernanta/documente/personal/portofolii/evaluari, `*.publish`, `meeting.close`, `reports.export_sensitive`, `inspect.package.read` |
| `director_adjunct` | aceleasi familii ca directorul, dar doar pentru domenii delegate; preferat prin pozitie/delegare, nu grant global nelimitat |
| `secretar` | `registratura.*`, `education.governance.read/manage`, `decisions.issuance.*`, `compliance.*`, `personnel.files.*`, `portfolios.verify`, `portfolios.transfer`, `portfolios.custody.manage`, publicari delegate |
| `registrator` | `registratura.read/manage`, citire limitata pentru documente educationale legate de inregistrari |
| `hr` | `education.personnel.*`, `education.evaluations.read/manage` operational, `education.declarations.read/manage`, acces sensibil auditat |
| `profesor` | `education.read`, `read_own` pentru portofoliu/evaluare/declaratii, `meeting.vote` doar contextual |
| `inspector` | `education.read`, citire domenii necesare, `education.inspect.package.read`, `education.reports.export_sensitive` doar prin pachet |
| `gdpr_officer` | `gdpr.*`, `education.compliance.*`, `education.personnel.access.read/manage`, `education.reports.export_sensitive` |
| `responsabil_comisie` | permisiuni contextuale pe comisia activa; global doar daca este rol administrativ real |
| `diriginte` | permisiuni contextuale pe clasa si consiliul clasei |
| `membru_ca` / `membru_cp` | participare/vot contextual, fara acces global la registru |

## 11. File touchpoints pentru implementare

Backend:

- `backend/internal/db/migrations/0064_education_contextual_permissions.sql`
- urmatoarea migrare pentru roluri lipsa si verticalele noi
- `backend/internal/education/middleware.go`
- fisier nou recomandat: `backend/internal/education/contextual_rbac.go`
- `backend/internal/education/service.go`
- `backend/internal/education/governance_portfolio_flows.go`
- `backend/internal/education/managerial_regulation_flows.go`
- `backend/internal/education/personnel_detail_flows.go`
- `backend/internal/education/evaluation_detail_flows.go`
- `backend/internal/education/export.go`
- `backend/cmd/server/main.go`

Frontend:

- `frontend/src/app/core/authz/authz.guard.ts`
- `frontend/src/app/features/education/education.routes.ts`
- `frontend/src/app/features/education/shared/education-config.ts`
- `frontend/src/app/features/education/shared/education-domain-workspace.component.ts`
- fisier nou recomandat: `frontend/src/app/features/education/shared/education-capabilities.ts`
- dashboardurile si wizard-urile pe rol care ascund actiunile nepermise

## 12. Checklist de acceptanta RBAC

- Un profesor nu poate lista toate portofoliile, evaluarile sau declaratiile altor profesori.
- Un membru CA/CP poate vota doar in sedinte unde are mandat activ si drept de vot.
- Secretarul poate pregati/publica documente delegate, dar nu poate inchide sedinta fara delegare explicita.
- Inspectorul vede doar pachete sau rapoarte aprobate pentru inspectie.
- Accesul la dosar personal sensibil scrie eveniment de audit cu actor, motiv si resursa.
- Exporturile sensibile cer `education.reports.export_sensitive` si jurnalizare.
- UI-ul ascunde actiunile nepermise, iar backendul returneaza `403` cand regula contextuala nu trece.
- Fiecare permisiune noua are seed in `app_permissions` si mapare intentionata in `app_role_permissions` sau in regulile contextuale.
