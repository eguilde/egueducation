# Catalog canonic de cerințe — funcționalitatea Școală

Data reviziei: 2026-09-09
Statut: audit al codului React/Go/PostgreSQL existent și bază obligatorie pentru implementare

## 1. Reguli de interpretare

Acest document este sursa canonică pentru funcționalitatea Școală. Statusul unei cerințe se stabilește numai din cod executabil și teste, nu din documente de planificare mai vechi.

Valorile de status sunt:

- `acoperit`: contractul DB–API–React există și este demonstrat automat;
- `parțial`: există componente utilizabile, dar lipsește o parte a contractului, autorizării, UX-ului sau testării;
- `lipsă`: comportamentul nu există ori nu poate fi securizat corect;
- `referință`: cerință documentară ori juridică fără implementare directă izolată.

Ordinea autorității surselor este: act normativ în vigoare, act normativ consolidat, document oficial de implementare, cerință explicită de produs. Proiectul din martie 2026 este păstrat pentru trasabilitate, dar Ordinul nr. 3.858/2026 și anexele publicate îl înlocuiesc.

## 2. Surse și valabilitate

| ID | Sursă | Utilizare | Statut la 2026-09-09 |
| --- | --- | --- | --- |
| LEG-001 | [Legea învățământului preuniversitar nr. 198/2023](https://legislatie.just.ro/Public/DetaliiDocumentAfis/271896), în special art. 187 | definiție, utilizări și conținut obligatoriu al portofoliului | în vigoare, se folosește forma consolidată |
| LEG-002 | [Ordinul nr. 3.858/2026 și metodologia-cadru](https://legislatie.just.ro/Public/DetaliiDocument/310372) | structură, gestionare individuală/instituțională, opis, transfer, retenție, procedură internă | în vigoare din 2026-05-08; aplicare pentru portofolii din anul școlar 2026–2027 |
| LEG-003 | [Anexa nr. 1 — structura-cadru](https://www.edu.ro/sites/default/files/Anexa_1_Structura_portofoliu.pdf) | secțiuni și exemple orientative de documente | anexă oficială; lista este minimală, orientativă și adaptabilă |
| LEG-004 | [Ordinul nr. 7.386/2024](https://legislatie.just.ro/Public/DetaliiDocument/291176) și profilul/standardele profesionale | taxonomia competențelor și evaluarea evoluției profesionale | în vigoare; standardele nu sunt fișă de post |
| LEG-005 | [ROFUIP aprobat prin Ordinul nr. 5.726/2024](https://legislatie.just.ro/Public/DetaliiDocument/286829), modificat inclusiv prin [Ordinul nr. 4.261/2026](https://legislatie.just.ro/Public/DetaliiDocumentAfis/312069) | atribuții director, consilii, documente și guvernanță școlară | în vigoare; se aplică numai forma consolidată |
| LEG-006 | GDPR și Legea nr. 190/2018 | minimizare, scop, acces, securitate, trasabilitate și drepturi | în vigoare |
| LEG-007 | Legea Arhivelor Naționale nr. 16/1996, actualizată | integritate, nomenclator, retenție, transfer și export arhivistic | în vigoare |
| LEG-008 | Legea nr. 214/2024 | semnătură, sigiliu și marcă temporală pentru documente cu efect juridic | în vigoare; actele vechi abrogate nu se folosesc ca bază |
| GHID-001 | `docs/fisier-6107.pdf`, ghid ISJ Timiș 2024–2025 | inventar operațional pentru director/director adjunct și documente manageriale | ghid, nu act normativ; cerințele se confruntă cu ROFUIP consolidat |
| DRAFT-001 | `docs/Proiect_Metodologie_portofoliu_CD.pdf` | istoric și comparație cu forma finală | depășit de LEG-002; nu generează cerințe dacă diferă de ordin |

## 3. Modelul de securitate obligatoriu

Toate deciziile de acces se evaluează cumulativ:

`tenant din host și sesiune → instituție activă → membership activ → rol/permisie → relație contextuală cu obiectul → stare/acțiune`

Frontendul poate ascunde acțiuni, dar backendul este singura autoritate. Niciun `tenant`, `institution_id`, `owner_user_id`, rol sau set de permisiuni primit de la browser nu poate extinde sesiunea.

### 3.1 Roluri instituționale

| Rol | Drepturi de bază | Limitări obligatorii |
| --- | --- | --- |
| `teacher` / profesor | propriul portofoliu, propriile documente, opis, declarații, export și cereri | nu listează și nu citește portofoliile altor persoane |
| `director` | vizibilitate instituțională, solicitare completări, verificare/validare, proceduri, rapoarte | numai instituția curentă; fără bypass tenant |
| `director_adjunct` | numai atribuțiile delegate, pe perioadă și domeniu | nu moștenește automat toate drepturile directorului |
| `secretar` / HR | operații administrative și dosar personal conform atribuției | nu modifică materialul didactic al profesorului |
| `portfolio_custodian` / arhivar | custodie, transfer, retenție și arhivare | fără drept implicit de editare pedagogică |
| `committee_member` / CA / CP | operații contextuale pe mandat, ședință sau procedură | fără acces global la portofolii |
| `inspector` / evaluator extern | pachet aprobat, limitat temporal și la scop | implicit read-only, fără acces operațional general |
| `tenant_admin` | utilizatori, roluri și configurare în tenant | nu primește automat acces la conținut pedagogic |
| `platform_admin` | tenanturi și operare tehnică | accesul la conținut cere flux de suport temporar, motivat și auditat |

Permisiunile minime distincte sunt:

- `education.portfolios.read_own`, `education.portfolios.manage_own`, `education.portfolios.export_own`;
- `education.portfolios.read`, `education.portfolios.manage_structure`;
- `education.portfolios.review`, `education.portfolios.verify`, `education.portfolios.transfer`;
- `education.portfolios.custody.manage`, `education.portfolios.archive_grants.manage`, `education.portfolios.audit.read`;
- familiile existente pentru guvernanță, personal, evaluare, comisii și rapoarte.

## 4. Cerințe funcționale — portofoliul profesional

| ID | Cerință și criteriu de acceptare | Contract necesar | Status audit inițial |
| --- | --- | --- | --- |
| PRT-001 | Un portofoliu are proprietar imutabil `app_user_id` și, unde există, legătură cu persoana/încadrarea. Numele este numai proiecție de afișare. | FK DB, model API read-only, ownership derivat din sesiune | lipsă |
| PRT-002 | Profesorul creează, listează, citește și actualizează numai portofoliul propriu din instituția activă. | query scoped user+institution, endpoint `/me`, 403/404 sigur | lipsă |
| PRT-003 | Directorul și rolurile desemnate pot accesa portofoliile instituției numai prin permisiuni administrative explicite. | policy contextuală și audit | parțial |
| PRT-004 | La începutul anului școlar se creează sau completează portofoliul aferent anului, fără duplicate pentru același proprietar/context. | unicitate și validare DB/API | parțial |
| PRT-005 | Structura implicită include cele cinci secțiuni oficiale: identificare; predare–învățare–evaluare; activități complementare; managementul clasei; evoluție/dezvoltare profesională. | catalog versionat de secțiuni | parțial |
| PRT-006 | Școala poate adapta prin procedură proprie categorii, cerințe și caracter obligatoriu/opțional, păstrând versiunea aplicată fiecărui portofoliu. | schemă configurabilă tenant/instituție, versiune, UI admin | parțial |
| PRT-007 | Documentul are secțiune, categorie, titlu, descriere, dată dobândire, dată încărcare, an școlar, disciplină/clasă unde se aplică, competențe, proveniență, hash și versiune. | DB/API/upload/storage | parțial |
| PRT-008 | Documentele sunt ordonate cronologic în secțiune; fiecare adăugare/retragere actualizează și datează opisul. | tranzacție și opis automat | acoperit parțial; testare de ownership lipsă |
| PRT-009 | Înlocuirea/retragerea nu distruge istoricul. Versiunile, actorul, momentul și motivul rămân verificabile. | versionare/soft-delete/audit | lipsă |
| PRT-010 | Profesorul declară separat consimțământul GDPR și autenticitatea documentelor; se păstrează versiunea textului, timpul și dovada. | declarație versionată, nu două booleene editabile arbitrar | parțial |
| PRT-011 | Fluxul minim este `draft → submitted → returned|verified → archived`; transferul este flux separat. Tranzițiile sunt comenzi și respectă rolul/starea. | state machine și endpoints de comandă | lipsă |
| PRT-012 | Directorul/revizorul poate solicita completări, comenta, verifica sau respinge motivat; profesorul vede istoricul și remediază. | review workflow și notificări | parțial |
| PRT-013 | Portofoliul poate fi valorificat distinct pentru licențiere/definitivat, grade, evaluare anuală, mobilitate, dezvoltare, inspecție, evaluare externă și distincții. | tip de valorificare, scop, pachet/raport | parțial |
| PRT-014 | Dosarul personal și portofoliul sunt distincte; duplicarea documentelor administrative urmează decizia CP/procedura instituției. | mapare document-la-document și politică versionată | parțial |
| PRT-015 | Transferul între unități are cerere, expeditor, destinatar, pachet interoperabil, hash, predare, recepție și confirmare, fără pierderea istoricului. | workflow cross-tenant controlat și export | parțial |
| PRT-016 | La încetarea activității, portofoliul se păstrează 3 ani în unitate; termenul se calculează din data încetării, nu din data creării. | eveniment încetare, retention job/legal hold | lipsă |
| PRT-017 | Profesorul poate solicita și primi exportul portofoliului și opisului într-un format interoperabil, cu manifest și integritate verificabilă. | export autenticat, manifest/hash | parțial |
| PRT-018 | Citirea, descărcarea, modificarea, revizia, validarea, partajarea, exportul, transferul și schimbarea drepturilor sunt auditate. | evenimente append-only cu actor/scop/IP/context | parțial |
| PRT-019 | Procedura internă a școlii este versionată, aprobată de CA, publicată și leagă calendarul, accesul, formatele, păstrarea și arhivarea. | registru proceduri + publicare + versiune aplicată | parțial |
| PRT-020 | Conținutul sensibil este minimizat, criptat/protejat în tranzit și repaus, scanat, autorizat la download și separat strict pe tenant. | storage tenant-scoped, signed access, malware/OCR policy | parțial |

## 5. Cerințe funcționale — conducere și celelalte funcții școlare

| ID | Cerință și criteriu de acceptare | Status audit inițial |
| --- | --- | --- |
| SCH-001 | Directorul are cockpit instituțional, nu acces platform-wide, pentru guvernanță, personal, evaluare, portofolii, documente manageriale și raportare. | parțial |
| SCH-002 | Directorul adjunct operează numai prin delegare explicită cu domeniu, început, sfârșit, emitent și revocare. | lipsă |
| SCH-003 | CA și CP au componență/mandate, convocare, cvorum, prezență, vot, minute, hotărâri și registre auditate. | parțial |
| SCH-004 | Comisiile, inclusiv comisia temporară de evaluare, au constituire, mandat, membri, incompatibilități, documente și raport final. | parțial |
| SCH-005 | Profesorul diriginte primește capabilități numai pentru clasa și perioada atribuite. | parțial/lipsă de teste contextuale complete |
| SCH-006 | Portofoliul directorului/directorului adjunct acoperă documentele personale și manageriale aplicabile rolului, fără a amesteca registrele. | parțial |
| SCH-007 | Documentele manageriale de diagnoză, prognoză și evidență au versiune, circuit, aprobare/publicare și retenție. | parțial |
| SCH-008 | Secretariatul, HR, arhivarul, responsabilul de comisie și inspectorul au permisiuni distincte, nu variante ale rolului `admin`. | parțial |
| SCH-009 | Rapoartele sensibile folosesc minimizare și export auditat; rapoartele publice sunt o proiecție separată. | parțial |

## 6. Contract DB → API → React

| ID | Cerință | Dovadă obligatorie | Status audit inițial |
| --- | --- | --- | --- |
| CTR-001 | Fiecare entitate are constrângeri DB, FK, tenant/instituție, versiune concurentă și reguli de ștergere explicite. | migrare + test PostgreSQL | parțial |
| CTR-002 | RLS sau control echivalent acoperă fiecare tabel și operație; serviciul setează contextul numai din sesiune. | teste cross-tenant și cross-institution | parțial |
| CTR-003 | OpenAPI descrie fiecare operație, request/response/error, security, `x-required-permission` și `x-tenant-scope`. | validator CI | parțial |
| CTR-004 | React folosește clientul și tipurile generate per operație; adaptoarele cu cast-uri/rute netipizate sunt eliminate. | compile-time contract test | parțial |
| CTR-005 | Erorile 400/401/403/404/409/422 au schemă comună și nu divulgă existența obiectelor din alt scope. | contract + security tests | parțial |
| CTR-006 | Schimbările incompatibile de schemă/API sunt detectate înainte de merge/deploy. | migration/OpenAPI/client drift checks | parțial |

## 7. Cerințe React și UX

| ID | Cerință | Status audit inițial |
| --- | --- | --- |
| UI-001 | `/scoala/teacher` și `/scoala/portfolio/me` deschid un workspace dedicat propriului portofoliu, nu dashboardul directorului. | lipsă |
| UI-002 | Workspace-ul profesorului prezintă progres, secțiuni, documente, lipsuri, opis, declarații, stare, observații și acțiunea permisă următoare. | lipsă |
| UI-003 | Workspace-ul directorului oferă listă instituțională, filtre de completitudine/stare și flux de review fără editarea conținutului profesorului. | parțial |
| UI-004 | Meniurile, rutele și butoanele sunt derivate numai din sesiunea OIDC; props-urile nu pot suprascrie tenantul sau permisiunile în producție. | parțial |
| UI-005 | Formularele folosesc componente PrimeReact, sunt mobile-first, accesibile, cu loading/error/empty și validări identice contractului. | parțial |
| UI-006 | Uploadul arată progres, scanare, tip/dimensiune, eroare recuperabilă, versiune și rezultat opis. | parțial |

## 8. Catalog minim de teste obligatorii

| Nivel | Scenarii obligatorii |
| --- | --- |
| unit backend | state machine, policy own/admin, validări metadate, retenție, declarații, mapare rol-permisiune |
| PostgreSQL integration | FK ownership, unicitate, RLS, două persoane în aceeași școală, două instituții/tenanturi, audit, concurență |
| OpenAPI/contract | rută–operație–permisiune, schema request/response, client React regenerat și fără drift |
| unit React | capabilități pe sesiune, route guard, view model profesor/director, stări și erori |
| E2E real | OIDC → React → API → PostgreSQL pentru profesor, director, tenant admin și refuz cross-user/cross-tenant |
| E2E portofoliu | creare proprie, upload, opis, submit, return, remediere, verify, export, transfer, retenție/audit |
| securitate | IDOR, permisiune falsă în UI/body, token pentru alt host, obiect inexistent versus obiect interzis, download fără drept |

Un test E2E care mock-uiește API-ul demonstrează numai comportamentul UI și nu poate marca drept `acoperit` un contract end-to-end.

## 9. Gap-uri confirmate din cod la începutul implementării

1. `education_portfolios` are `owner_name`/`owner_role`, dar nu proprietar utilizator imutabil.
2. Endpointurile existente separă numai `education.portfolios.read/manage`; nu există self-service securizat.
3. Profesorul nu primește permisiuni de portofoliu propriu.
4. Starea, transferul, autenticitatea și consimțământul pot fi mutate prin update generic.
5. Rutele React de profesor și `/portfolio/me` folosesc același workspace generic și pot afișa dashboardul directorului.
6. Clientul generat este accesat prin adaptoare dinamice/cast-uri, deci multe contracte nu sunt verificate static per operație.
7. Testul Playwright Education folosește mock-uri și nu dovedește RBAC/persistență în baza reală.
8. Documentele de audit mai vechi conțin referințe Angular și afirmații de „acoperit” care nu mai descriu frontendul React; ele sunt istorice, nu surse de adevăr.

## 10. Definition of done

O cerință poate trece la `acoperit` numai dacă:

1. are migrare și constrângeri adecvate;
2. este autorizată tenant + instituție + utilizator/rol + obiect;
3. are contract OpenAPI valid și client React regenerat;
4. are UI funcțional pentru rolurile aplicabile;
5. are teste unitare și de integrare proporționale cu riscul;
6. pentru fluxurile critice are E2E real cu verificarea efectului în API/DB;
7. produce auditul necesar și nu expune date cross-tenant/cross-user.

## 11. Starea după remedierea P0 — 2026-09-09

Auditul inițial de mai sus rămâne fotografia codului înainte de remediere. Următoarele modificări sunt implementate în aceeași revizie și schimbă evaluarea astfel:

| Cerințe | Stare curentă | Dovadă în cod/teste |
| --- | --- | --- |
| PRT-001, PRT-002 | implementat; `acoperit` după trecerea E2E real în CI | proprietar UUID imutabil, FK, trigger de membership/instituție, endpointuri `/me`, teste own/cross-user/cross-tenant |
| PRT-003 | implementat pentru citire instituțională și comenzile return/verify | permisiuni `school.read`, `school.manage`, `verify`, middleware backend și acțiuni React distincte |
| PRT-004 | implementat pentru portofoliile identity-bound | index unic instituție + proprietar + an școlar, răspuns API 409 |
| PRT-005 | implementat pentru catalogul activ obligatoriu la depunere | submit-ul tranzacțional blochează portofoliul și refuză cu 422 orice componentă obligatorie fără dovadă eArhivă validă |
| PRT-007 | consolidat parțial pentru dovezi deja existente în eArhivă | grant explicit instituție–document–utilizator, versiune curentă activă, stare `ready`, sursă stocată și snapshot imutabil de versiune/hash/obiect la submit |
| PRT-008 | implementat pentru documentele proprii și regenerarea opisului | comenzi owner-scoped, proveniență/autenticitate impuse server-side, listare și ștergere React |
| PRT-011 | implementat pentru `draft/returned → submitted → returned|validated` | endpointuri de comandă, control stare/rol, audit și scenariu E2E real |
| PRT-018 | implementat pentru creare/update/submit/return/validate/document/opis din acest flux | evenimente `app_audit_log` verificate în E2E real |
| CTR-001–CTR-005 | parțial consolidat pentru bounded-context-ul portofoliu propriu | migrare, RLS existent, DTO Go îngust, OpenAPI pentru toate cele 490 de operații router (325 Școală), client și validatori React regenerați, erori 403/409/422 |
| UI-001, UI-002, UI-004, UI-005 | implementat pentru self-service profesor și administrarea granturilor eArhivă | rute dedicate, permisiuni din sesiunea OIDC, componente PrimeReact, documente/checklist/opis/revizuiri, panou director/custode și stări responsive |

Nu sunt declarate finalizate prin această remediere și rămân backlog obligatoriu: uploadul binar autorizat și scanarea fișierelor (restul PRT-007/020, UI-006), versionarea/retragerea fără pierderea istoricului (PRT-009; snapshot-ul de submit există, dar nu înlocuiește istoricul complet), declarațiile juridice versionate (PRT-010), pachetele de valorificare și exportul interoperabil complet (PRT-013/017), transferul cross-tenant complet (PRT-015), retenția calculată de la încetarea activității și legal hold (PRT-016), procedura internă versionată (PRT-019), delegarea directorului adjunct (SCH-002) și acoperirea E2E reală a tuturor acestor fluxuri. Termenul de trei ani din cod este momentan o protecție minimă calculată server-side, nu implementarea completă a PRT-016.
