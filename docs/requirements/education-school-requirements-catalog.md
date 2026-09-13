# Catalog canonic de cerințe — funcționalitatea Școală

Data reviziei: 2026-09-12
Statut: audit al codului React/Go/PostgreSQL existent și bază obligatorie pentru implementare

## 1. Reguli de interpretare

Acest document este sursa canonică pentru funcționalitatea Școală. Statusul unei cerințe se stabilește numai din cod executabil și teste, nu din documente de planificare mai vechi.

### Corecție de stare bazată pe verificările din 12 septembrie

Această precizare înlocuiește afirmațiile de finalizare pentru PRT-016/017 din
checkpointurile istorice de mai jos; nu elimină nicio cerință din catalog.

- **PRT-016: parțial.** Termenul DB este calculat de la încetare, iar documentele
  active pot avea custodie fără o dată de retenție inventată. Aplicarea și
  verificarea retenției pe versiunea exactă din storage înainte de eliberarea
  custodiei și sincronizarea blocajelor juridice sunt demonstrate acum pe
  fluxul normal prin E2E cu PostgreSQL/MinIO reale. Istoricul operațiilor este
  implementat prin API paginat/filtrat/sortat pe server și UI React pentru
  profesor și verificator; integrarea instituțională după refresh este demonstrată
  și prin proba browser/OIDC/API/PostgreSQL/MinIO (PASS, 2,8 minute), inclusiv
  filtrul de zi UTC și citirea stării din dialog. Politica pentru încetări cu
  termen deja expirat este implementată prin 0165: hold de dispoziție, cerere
  proprie, decizie separată cu RBAC dual, receipt append-only și worker exact-version;
  PostgreSQL + MinIO verifică inclusiv restartul și fresh review după blocker.
  Rămân deschise reconcilierea intențiilor blocate și publicarea/verificarea în mediul țintă.
  Existența DB/API/UI și proba izolată nu închid singure întreaga cerință.
- **PRT-017: parțial, cu flux normal demonstrat.** Două probe browser/OIDC/Go/
  PostgreSQL/MinIO au verificat exportul ZIP al portofoliului, opisului și
  PDF-urilor, inclusiv octeți/hashuri/manifest. O probă suplimentară la 20:18
  a demonstrat și hashurile fișierelor generate și reîncercarea după pierderea
  răspunsului unui upload deja salvat. Testele PG/MinIO ulterioare demonstrează
  reluarea după storage reușit/rollback DB și lipsa auditului parțial. Reconcilierea
  administrativă după revocarea autorizării înainte de commit și verificarea
  publicării rămân deschise. Un manifest fără fișierele cerute nu este echivalentul
  exportului complet.
- Rezultatele sunt din mediu izolat cu date sintetice; OCR, antivirusul și
  verificarea semnăturilor folosesc emulatoare în E2E. Nu demonstrează
  serviciile externe de producție și nu reprezintă publicare.

Dovezile, comenzile și limitele verificării sunt în
`teacher-portfolio-verification-2026-09-12.md`. Celelalte cerințe rămân în
scop și necesită verificare individuală înainte de declararea modulului complet.

Valorile de status sunt:

- `acoperit`: contractul DB–API–React există și este demonstrat automat;
- `parțial`: există componente utilizabile, dar lipsește o parte a contractului, autorizării, UX-ului sau testării;
- `lipsă`: comportamentul nu există ori nu poate fi securizat corect;
- `referință`: cerință documentară ori juridică fără implementare directă izolată.

Ordinea autorității surselor este: act normativ în vigoare, act normativ consolidat, document oficial de implementare, cerință explicită de produs. Proiectul din martie 2026 este păstrat pentru trasabilitate, dar Ordinul nr. 3.858/2026 și anexele publicate îl înlocuiesc.

## 2. Surse și valabilitate

| ID | Sursă | Utilizare | Statut la 2026-09-11 |
| --- | --- | --- | --- |
| LEG-001 | [Legea învățământului preuniversitar nr. 198/2023, forma consolidată](https://legislatie.just.ro/Public/DetaliiDocument/309185), în special art. 27–28, 108, 137–138 și 187 | nucleu public/privat/confesional, finanțare/burse și portofoliu | în vigoare; regulile temporale sunt versionate, nu hardcodate |
| LEG-002 | [Ordinul nr. 3.858/2026 și metodologia-cadru](https://legislatie.just.ro/Public/DetaliiDocument/310372) | structură, gestionare individuală/instituțională, opis, transfer, retenție, procedură internă | în vigoare din 2026-05-08; aplicare pentru portofolii din anul școlar 2026–2027 |
| LEG-003 | [Anexa nr. 1 — structura-cadru](https://www.edu.ro/sites/default/files/Anexa_1_Structura_portofoliu.pdf) | secțiuni și exemple orientative de documente | anexă oficială; lista este minimală, orientativă și adaptabilă |
| LEG-004 | [Ordinul nr. 7.386/2024](https://legislatie.just.ro/Public/DetaliiDocument/291176) și profilul/standardele profesionale | taxonomia competențelor și evaluarea evoluției profesionale | în vigoare; standardele nu sunt fișă de post |
| LEG-005 | [ROFUIP aprobat prin Ordinul nr. 5.726/2024](https://legislatie.just.ro/Public/DetaliiDocument/289484), modificat prin [Ordinul nr. 6.226/2025](https://legislatie.just.ro/Public/DetaliiDocument/302026) și [Ordinul nr. 4.261/2026](https://legislatie.just.ro/Public/DetaliiDocumentAfis/312069) | atribuții director, consilii, contract educațional, catalog și guvernanță școlară | în vigoare; se aplică numai forma consolidată, nu ROFUIP 4.183/2022 |
| LEG-006 | GDPR și Legea nr. 190/2018 | minimizare, scop, acces, securitate, trasabilitate și drepturi | în vigoare |
| LEG-007 | Legea Arhivelor Naționale nr. 16/1996, actualizată | integritate, nomenclator, retenție, transfer și export arhivistic | în vigoare |
| LEG-008 | Legea nr. 214/2024 | semnătură, sigiliu și marcă temporală pentru documente cu efect juridic | în vigoare; actele vechi abrogate nu se folosesc ca bază |
| LEG-009 | [HG nr. 69/2024](https://legislatie.just.ro/Public/DetaliiDocument/295485), cu modificările ulterioare | finanțarea de bază a unităților particulare/confesionale acreditate și a celor autorizate fără taxă | în vigoare; condițiile și valorile sunt versionate pe anul financiar |
| LEG-010 | [HG nr. 993/2020](https://legislatie.just.ro/Public/DetaliiDocument/234510) și [HG nr. 994/2020](https://legislatie.just.ro/Public/DetaliiDocumentAfis/255166) | autorizare, acreditare, evaluare periodică și standarde naționale de calitate | în vigoare cu modificările ulterioare; se folosesc registrele oficiale ARACIP |
| LEG-011 | [Legea nr. 98/2016, forma consolidată](https://legislatie.just.ro/Public/DetaliiDocument/257213), art. 4 și 6 | determinarea calității de autoritate contractantă și aplicabilitatea pentru anumite contracte finanțate public | în vigoare; aplicabilitatea se evaluează pe entitate și, când este cazul, pe contract/proiect |
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

## 10A. Cerințe de produs public/privat

Această secțiune este obligatorie pentru orice verticală Școală. „Privat” include
școala confesională ca overlay (`private + confessional`), nu ca ramură separată
a produsului. Forma juridică, autorizarea/acreditarea, oferta, finanțarea și
aplicabilitatea achizițiilor sunt dimensiuni independente și trebuie evaluate la
data/contextul operației. Niciun boolean trimis de browser nu este dovadă.

| ID | Cerință | Tip instituție | Criteriu de acceptare |
| --- | --- | --- | --- |
| PUB-PRV-001 | Nucleul comun (elevi, clase, înscrieri, personal, portofoliu, guvernanță, documente, rapoarte) are același contract și aceeași izolare tenant/instituție. | public, privat, confesional | Aceeași operație OpenAPI este executată cu date sintetice în cel puțin un tenant public și unul privat; RLS/RBAC refuză cross-tenant și cross-institution. |
| PUB-PRV-002 | Profilul instituției este versionat și separă forma `public`/`private` de overlay-ul confesional, cu fondator/finanțator/ordonator identificați prin părți și surse verificabile. | toate | Profilul activ poate fi reconstruit la o dată; confesional fără profil privat, cult sau act verificabil este respins fail-closed. |
| PUB-PRV-003 | Oferta educațională și autorizarea/acreditarea sunt effective-dated pe nivel/program/specializare/locație/capacitate, nu doar pe instituție. | toate | Înscrierea și raportarea sunt permise numai pentru oferta și locația active la data operației; suspendarea afectează doar operațiile viitoare. |
| PUB-PRV-004 | Contractul educațional este tipat, versionat, aprobat și legat de elev/beneficiar, ofertă și perioada școlară. | public și privat | Crearea/semnarea/rezilierea are actor, stare, versiune, document eArhivă și audit; istoria nu se șterge. |
| PUB-PRV-005 | Taxele private au catalog versionat, scadențe, reduceri, burse, refunduri, plăți și creanțe, cu reguli distincte de gratuitate/finanțare publică. | privat (inclusiv confesional) | O școală publică nu vede fluxul de taxe comerciale; o școală privată nu moștenește automat gratuitatea sau finanțarea; soldul este calculat server-side și auditat. |
| PUB-PRV-006 | Admiterea/înscrierea reflectă oferta autorizată, criteriile, capacitatea, documentele, decizia și contestația; înscrierea de clasă existentă nu este echivalentă cu admiterea. | toate | Un candidat nu poate fi admis pe ofertă/locație expirată; decizia și motivarea sunt versionate, iar contestația are termen și rezultat. |
| PUB-PRV-007 | Bursele și facilitățile sunt modelate separat de taxe și finanțare, cu eligibilitate, perioadă, sursă legală, aprobare și beneficiar minimizat. | public și privat, după program/sursă | Eligibilitatea este evaluată pentru an/ofertă/beneficiar; lipsa dovezii produce `indeterminate`, nu acord automat. |
| PUB-PRV-008 | Finanțarea publică este instrument versionat pe an, ofertă, beneficiar, regim cu/fără taxă, sursă, plafon și dovadă; nu derivă din `public_funding=true`. | public și privat eligibil | Fluxul păstrează evaluarea și snapshotul folosit; privatul primește finanțare numai dacă sunt îndeplinite condițiile aplicabile. |
| PUB-PRV-009 | Achizițiile se evaluează separat la nivel de entitate și contract/proiect; finanțarea publică nu transformă automat o școală privată în autoritate contractantă. | public și privat | Pentru Legea 98/2016, decizia conține criterii, dovezi, sursă și motivare; `indeterminate` blochează procedura reglementată. |
| PUB-PRV-010 | Guvernanța permite diferențe de fondator/CA/CP/comisii și delegări, păstrând regulile comune și perioadele de mandat. | toate | Directorul/administratorul operează numai în instituția activă; delegarea adjunctului are domeniu/perioadă/revocare; confesionalul are overlay fără bypass. |
| PUB-PRV-011 | Rapoartele și exporturile separă datele operaționale sensibile de proiecțiile publice și includ forma juridică, oferta, finanțarea și sursele la data raportului. | toate | Exportul este permis pe rol/scop, auditat și repetabil; nu divulga taxe/private către roluri publice fără drept. |
| PUB-PRV-012 | Catalogul de permisiuni este pe capabilitate și context, nu pe un singur rol `admin`; tenant admin nu primește implicit conținut pedagogic/financiar. | toate | Matricea RBAC este testată pentru director, profesor, secretariat/HR, financiar, achiziții, arhivar și inspector, inclusiv refuz cross-user. |
| PUB-PRV-013 | Finanțarea de bază distinge unitatea acreditată de cea autorizată provizoriu, regimul cu/fără taxă, nivelul, beneficiarul și anul financiar; dobândirea statutului în cursul anului produce efect de la anul financiar următor când legea o cere. | privat/confesional eligibil | Evaluarea folosește profilul și oferta efective, nu un flag; păstrează actul, anul, costul standard/plafonul și snapshotul beneficiarilor folosit la calcul. |
| PUB-PRV-014 | Bursele și facilitățile folosesc reguli versionate pe tip, an școlar, sursă, beneficiar și condiții; cuantumurile ori calendarele tranzitorii nu sunt hardcodate. | public și privat/confesional eligibil | Social/tehnologic și orice alt tip activ sunt configurate din policy pack; pentru privat/confesional se verifică explicit regimul fără taxă unde este cerut, iar rezultatul este auditat. |
| PUB-PRV-015 | Contractul educațional se încheie la înscriere cu reprezentantul legal sau elevul major, are versiuni/acte adiționale aprobate și retenția legală calculată. | toate | Două exemplare și semnăturile sunt dovedite; contractul se păstrează pe durata școlarizării și încă 2 ani după plecarea elevului; actele adiționale nu rescriu originalul. |
| PUB-PRV-016 | Autorizarea/acreditarea și evaluarea periodică se urmăresc pe unitate, nivel/program, limbă, formă de învățământ și locație, cu decizia și registrul oficial de origine. | toate | O schimbare ARACIP este importată/revalidată cu impact report; nicio ofertă nu moștenește statutul altei combinații, iar admiterea pe statut suspendat/retras este refuzată. |
| PUB-PRV-017 | Guvernanța privată/confesională păstrează fondatorul, persoana juridică finanțatoare, actul de numire, mandatul și avizul CA; diferențele nu elimină controalele comune. | privat/confesional | Numirea/revocarea directorului și deciziile fondatorului sunt effective-dated și arhivate; salariile și datele HR sunt accesibile numai rolurilor distincte autorizate. |
| PUB-PRV-018 | Responsabilul REGES-ONLINE/EDUSAL este desemnat în scris, iar catalogul electronic are închidere anuală, înregistrare, export probator și arhivare. | toate | Desemnarea, mandatul, transmiterile și erorile sunt auditate; închiderea catalogului produce document/hash/versiune și dovadă eArhivă fără posibilitatea rescrierii perioadei închise. |
| PUB-PRV-019 | Orice regulă dependentă de an școlar/financiar, cuantum, calendar sau formă consolidată are interval și sursă versionată; ghidurile și proiectele nu pot suprascrie actul normativ. | toate | Revalidarea marchează regula expirată ori `indeterminate`, emite impact report și nu modifică retroactiv evaluările/operațiile istorice. |
| PUB-PRV-020 | Schimbarea formei, fondatorului, acreditării, regimului de taxă ori finanțării este prospectivă și declanșează evaluarea impactului asupra ofertelor, contractelor, burselor, achizițiilor și raportărilor. | toate | Operațiile istorice păstrează profilul/evaluarea originale; operațiile viitoare folosesc noua versiune numai după aprobare și fără intervale ambigue. |

Starea de implementare se publică în auditul public/privat asociat; cerințele de
mai sus nu pot fi marcate `acoperit` prin existența unei tabele sau a unei
componente UI, ci numai prin contract DB–API–React și test real.

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
| CTR-001–CTR-005 | parțial consolidat pentru bounded-context-ul portofoliu propriu | migrare, RLS existent, DTO Go îngust, OpenAPI pentru toate cele 506 operații router (341 Școală), client și validatori React regenerați, erori 403/409/422 |
| UI-001, UI-002, UI-004, UI-005 | implementat pentru self-service profesor și administrarea granturilor eArhivă | rute dedicate, permisiuni din sesiunea OIDC, componente PrimeReact, documente/checklist/opis/revizuiri, panou director/custode și stări responsive |

Nu sunt declarate finalizate prin această remediere și rămân backlog obligatoriu: metadatele pedagogice complete și controalele uniforme de storage (restul PRT-007/020, UI-006), versionarea uniformă fără pierderea istoricului (PRT-009), pachetele distincte de valorificare (PRT-013), transferul cross-tenant complet (PRT-015), delegarea directorului adjunct (SCH-002) și acoperirea E2E reală a tuturor acestor fluxuri. Exportul probatoriu propriu PRT-017, declarațiile juridice versionate PRT-010, retenția calculată de la încetarea activității și legal hold PRT-016, precum și procedura internă versionată PRT-019 sunt implementate; fluxul critic al portofoliului a trecut E2E real în CI, fără a demonstra încă matricea completă a tuturor rolurilor și domeniilor Școală.

## 12. Reaudit juridic și tehnic după remedierea fluxului legal

Verdictul din această secțiune este bazat pe cod executabil, contractul generat și teste, nu pe afirmațiile documentelor de proiect. La data reviziei nu există încă dovadă pentru conformitate 100% a întregului modul Școală.

| Cerință | Verdict actual | Dovadă executabilă |
| --- | --- | --- |
| PRT-001, PRT-002, PRT-004 | implementat și demonstrat în CI pe stiva reală | proprietar și instituție derivate server-side, unicitate, endpointuri `/me`, teste cross-owner/cross-tenant |
| PRT-005, PRT-006, PRT-019 | implementat pentru structura legală și aplicarea procedurii; UI admin în validare | catalogul final cu 5 secțiuni, procedură versionată draft/approved/published/superseded/withdrawn, reguli obligatorii, `applied_procedure_id`, submit guvernat de procedura aplicată |
| PRT-008 | implementat pentru actualizarea atomică document–opis | o singură tranzacție DB și test de rollback |
| PRT-010 | implementat și demonstrat în E2E real cu PostgreSQL | șabloane juridice server-issued versionate, acknowledgement imuabil și idempotent, submit fără booleene controlate de browser |
| PRT-016 | implementat la nivel DB/API/UI; lifecycle-ul de retenție rămâne fără scenariu browser dedicat | eveniment explicit de încetare, retenție calculată în DB, legal hold și interdicție de hard-delete |
| PRT-017 | implementat pentru exportul probatoriu propriu | manifest server-issued, versionat și imuabil, hash SHA-256, snapshot de proveniență și control owner-scoped |
| CTR-003 | acoperit pentru inventarul curent | OpenAPI verifică 506/506 rute, inclusiv 341 operații Școală; clientul React și validatorii sunt regenerați |
| UI-002, UI-003, UI-005 | consolidat | tabele cu filtrare/sortare/paginare server-side, header și paginator sticky, action column frozen, dialoguri PrimeReact și declarații afișate cu text/versiune |

Blocante care rămân înainte de o declarație de acoperire 100%:

1. PRT-013: pachete distincte, limitate la scop, pentru valorificare; exportul probatoriu propriu PRT-017 este implementat, dar nu înlocuiește aceste pachete.
2. PRT-015: transfer complet cross-tenant cu instituție țintă identificată, expediere, recepție și confirmare.
3. SCH-002: delegarea directorului adjunct limitată pe domeniu și perioadă.
4. PRT-007, PRT-009 și PRT-020: metadate pedagogice complete, versionare/retragere uniformă și integrarea verificabilă a tuturor controalelor de stocare.
5. CTR-001, CTR-002, CTR-004 și catalogul de teste: optimist concurrency uniform, eliminarea adaptoarelor cu rute string și rularea scenariilor PostgreSQL/E2E reale pentru fiecare rol și tenant.
6. Alinierea rolului instituțional `education.portfolios.school.manage` cu operațiile legacy `education.portfolios.manage` necesită o decizie explicită de autorizare; auditul nu extinde implicit drepturile.

## 13. Reaudit complet al codului și închiderea gap-urilor — 2026-09-11

Această reevaluare nu modifică fotografia istorică din secțiunile 9–12. Ea inventariază exclusiv comportamentele demonstrate de codul Go/PostgreSQL, contractul OpenAPI generat, clientul React și testele automate din candidatul curent de release.

| Cerințe | Verdict candidat release | Dovadă executabilă |
| --- | --- | --- |
| PRT-007, PRT-009, PRT-020 | implementat pentru documentele portofoliului | metadate pedagogice complete; document eArhivă `ready` obligatoriu; snapshot imuabil tenant-scoped de versiune/hash; versiuni append-only cu actor, moment și motiv; proiecția publică a istoricului exclude bucket-ul, cheia obiectului și identificatori interni sensibili |
| PRT-013 | implementat | pachete de valorificare limitate la scop și test PostgreSQL `TestPortfolioValorificationPackagesIntegration` |
| PRT-015 | implementat | rutare și dovezi de predare/recepție cross-tenant, cu test PostgreSQL `TestIntertenantPortfolioTransferRoutingAndEvidenceContractIntegration` |
| SCH-002, SCH-005 | implementat | delegare pe domeniu și perioadă, plus acces contextual la clase; teste de delegare și paritate pentru clase |
| Model RBAC | consolidat least-privilege | migrarea `0130_school_content_rbac_hardening.sql` elimină accesul implicit la conținut pentru `tenant_admin` și permisiunea legacy prea largă a directorului; superadminul și utilizatorul E2E rămân explicit separați |
| CTR-001, CTR-002, CTR-005 | implementat pentru modificările auditate | migrarea `0131_education_portfolio_document_contract.sql`, constrângeri și trigger-e tenant/instituție/arhivă, răspunsuri 404/422 nedivulgative și teste cross-tenant |
| CTR-003, CTR-006 | acoperit pentru inventarul curent | 562/562 rute concrete mapate la operații unice, metadate security/tenant/RBAC și 396 operații Education cu scheme complete; artefactele OpenAPI backend/canonic sunt verificate fără drift |
| CTR-004 | acoperit pentru operațiile documentelor portofoliului | rute literale `openapi-fetch` și tipuri generate pentru listare, creare, actualizare și istoric instituțional/propriu; fără `transport.request`, `as never` sau adaptoare de rezultat nevalidate pe aceste operații |
| UI-002, UI-003, UI-005 | implementat pentru fluxurile auditate | filtre și sortare server-side, metadate și proveniență eArhivă, dialoguri read-only de istoric, acțiuni de transfer/valorificare, componente PrimeReact și scenariu mobil la 390×844 |

Validarea locală obligatorie a candidatului include `go test ./...`, `go vet ./...`, validarea OpenAPI, typecheck React, auditul contractelor/UI, testele Vitest și build-ul de producție. Marcarea release-ului ca publicat și verificat în producție se face numai după trecerea pipeline-ului GitHub Actions și confirmarea reviziei servite de cluster; acest document nu substituie acea dovadă operațională.

## 14. Extindere obligatorie pentru unități publice și private

Funcționalitatea Școală și extensiile de management operațional trebuie să ruleze din același cod pentru unități publice, private și confesionale. Diferențele nu se exprimă prin fork-uri sau condiții UI, ci prin profil instituțional și policy packs versionate, evaluate în backend.

Forma juridică este independentă de:

- statutul de autorizare/acreditare;
- sursa finanțării și participarea la programe publice;
- calitatea de autoritate contractantă;
- profilul contabil, de salarizare, TVA și Trezorerie;
- opțiunile instituționale permise de lege.

Modulul existent oferă fundația tenant/instituție/RBAC/OIDC, procesele educaționale comune și un resolver public/privat incipient, aplicat numai unor verticale. Resolverul nu folosește încă modelul Stage1B ca sursă runtime și nu există bounded contexts complete pentru utilități, achiziții, catering, patrimoniu, logistică, SSM/PSI, HR, economic și contabil; verticala de contracte operaționale este doar parțială.

Catalogul detaliat, matricea existent–lipsă, designul și planul de implementare sunt documentate în:

- [Catalogul managementului operațional public/privat](school-operations-management-catalog.md);
- [Designul platformei de management școlar](../design/school-operations-platform-design.md);
- [Planul de implementare](../plans/school-operations-implementation-plan.md).

Testul de paritate obligatoriu demonstrează aceeași operație de nucleu în minimum un tenant public și unul privat, obligațiile suplimentare pentru public, politica internă pentru privat și overlay-ul public pentru o operațiune privată finanțată ori guvernată public.

Revizia din 2026-09-11 stabilește suplimentar că forma juridică nu este suficientă pentru autorizarea unei operații. Contractele School trebuie să evalueze cumulativ oferta educațională și locația, autorizarea/acreditarea, intervalul, finanțarea și regimul cu/fără taxă, aplicabilitatea achizițiilor și eventualul overlay confesional. Cerințele normative sunt APP-001–APP-012 din [catalogul managementului operațional public/privat](school-operations-management-catalog.md), iar gap-urile din cod și strategia de migrare sunt consemnate în [auditul de adecvare public/privat](../audits/public-private-school-product-fit-audit-2026-09-11.md).
