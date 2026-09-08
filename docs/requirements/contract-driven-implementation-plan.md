# EGU Education — plan de implementare bazat pe cerințe

Status: aprobat pentru începerea implementării  
Baseline sursă: `origin/main` la `e901456`  
Referință pentru Registratură, Flux și eArhivă: codul din `D:\dev\costesti-registratura`

## 1. Obiectiv și reguli

EGU Education trebuie să aibă contracte explicite și verificabile de la PostgreSQL până la React și înapoi. Registratură, Flux documente, eArhivă și administrarea asociată reproduc funcționalitatea și interfața Costești; diferența admisă este izolarea host-based multi-tenant și RBAC tenant-scoped.

1. React/PrimeReact este frontendul canonic; Angular și Costești rămân referințe read-only.
2. Nu se introduc simplificări sau reinterpretări ale comportamentului Costești.
3. Tenantul este rezolvat server-side din hostname; browserul nu îl selectează prin header/body.
4. Backendul este autoritatea RBAC; frontendul folosește aceleași drepturi pentru rutare și prezentare.
5. OpenAPI 3.1.1 este contractul HTTP canonic; toate operațiile business React folosesc clientul generat.
6. Modelele DB, domeniu și DTO public sunt separate și mapate explicit.
7. Cerințele se închid prin teste pe implementarea reală, nu doar prin mock-uri.
8. Se lucrează din checkout curat al `main`; modificările istorice locale nu sunt suprascrise.
9. Publicarea se face direct pe `main`, fără PR, după gate-urile aplicabile.

### 1.1 Stadiul implementării

| Increment | Stare | Acoperire |
|---|---|---|
| Fundație identitate și RBAC | Implementat, în validare | Identități telefon/e-mail normalizate, rol global separat, versiune de autorizare tenant-scoped, bootstrap Thomas/Stelian/Diana. |
| OIDC tenant și claim-uri | Implementat, în validare | Stare tranzitorie izolată prin cheia `(tenant_code, id)`, access token cu autorizație, ID Token/UserInfo cu PII limitat prin scope. |
| OpenAPI executabil în React | Implementat pentru primul slice | `/api/me` și Registratură list/create folosesc transport și validatori generați; extinderea la restul operațiilor rămâne obligatorie. |
| Registratură contract list/create | Implementat pentru primul slice | Filtre, căutare globală, sortare și paginare server-side; POST snake_case conform DTO-ului real. |
| Flux, eArhivă, Admin și paritate UI completă | Planificat | Se implementează după închiderea Gate B și extinderea clientului generat. |

„Implementat pentru primul slice” nu înseamnă paritate completă. Cerințele rămân deschise până când toate criteriile de acceptanță din secțiunile de mai jos au dovezi automate.

## 2. Lanțul contractual țintă

```text
Migrații PostgreSQL + constraints + RLS
  -> interogări SQL tipizate
  -> modele și servicii de domeniu
  -> DTO-uri și operații OpenAPI 3.1.1
  -> interfețe server Go + validare request/response
  -> client TypeScript și validatori runtime generați
  -> formulare, tabele și acțiuni React
  -> request autentificat și tenant-scoped
  -> tranzacție și constrângeri DB
  -> răspuns validat prin același contract
```

Niciun strat nu expune direct structura internă a stratului precedent.

## 3. Cerințe de sistem

| ID | Cerință | Acceptanță |
|---|---|---|
| SYS-001 | Fiecare funcționalitate leagă cerința, codul, endpointul, migrația și testul. | CI validează trasabilitatea și nu acceptă cerințe implementate fără teste. |
| SYS-002 | Erorile API folosesc un singur contract Problem Details. | Orice 4xx/5xx documentat are `application/problem+json`, cod stabil și request ID. |
| SYS-003 | Audit obligatoriu pentru modificări, autentificare și acces sensibil. | Evenimentul include tenant, subiect, actor, acțiune, rezultat și correlation ID. |
| SYS-004 | Timpul, UUID-urile și enumurile au reprezentări canonice. | OpenAPI, Go, SQL și TypeScript folosesc aceleași formate și enumuri. |

## 4. Contractul bazei de date

| ID | Cerință | Acceptanță |
|---|---|---|
| DB-001 | Schema se modifică exclusiv prin migrații versionate append-only. | Baza goală și upgrade-ul produc aceeași schemă verificată. |
| DB-002 | Datele tenant-owned conțin `tenant_code` și, unde este necesar, `institution_id`. | Schema check nu găsește tabele operaționale fără tenant. |
| DB-003 | Relațiile tenant-owned nu permit chei externe cross-tenant. | Inserarea unei relații A -> B între tenanți eșuează în PostgreSQL. |
| DB-004 | Unicitatea operațională este tenant-scoped. | Numerele/codurile se pot repeta numai între tenanți diferiți. |
| DB-005 | RLS este activat și forțat pe datele tenant-owned. | Rolul aplicației nu poate citi sau modifica tenantul greșit. |
| DB-006 | Numerotarea registrelor este atomică. | Test concurent fără duplicate sau numere cross-tenant. |
| DB-007 | Interogările critice sunt tipizate și verificate la build. | SQL invalid sau mapping incompatibil oprește CI. |

## 5. Multi-tenancy

| ID | Cerință | Acceptanță |
|---|---|---|
| TEN-001 | Tenantul se rezolvă din directorul activ de hostname-uri. | `scoalabalotesti.eguilde.cloud` rezolvă exclusiv `tenant-balotesti`; host necunoscut este respins. |
| TEN-002 | Host/forwarded host sunt acceptate numai de la ingress de încredere. | Host spoofing și custom domain inactiv sunt respinse. |
| TEN-003 | Tenantul tokenului coincide cu hostname-ul și membershipul activ. | Token A folosit pe B primește 401 înaintea serviciului de domeniu. |
| TEN-004 | Conexiunea PostgreSQL este tenant-bound pe durata requestului și curățată la release. | Pool test demonstrează lipsa scurgerii contextului. |
| TEN-005 | Storage, OCR, cozi și search păstrează tenantul end-to-end. | Obiectele și rezultatele A nu sunt vizibile/adresabile din B. |

## 6. Identitate și OIDC

| ID | Cerință | Acceptanță |
|---|---|---|
| IAM-001 | OIDC Provider implementează Authorization Code cu PKCE S256. | Discovery, authorize, token, JWKS, userinfo, revocation și logout trec testele de protocol. |
| IAM-002 | Issuer, audience, redirect URI, state, nonce și verifier sunt validate strict. | Testele de issuer/audience/redirect/replay/PKCE eșuează închis. |
| IAM-003 | Access tokenul conține tenant, roluri și permisiuni efective. | Claimurile coincid cu `/api/me` și hostname-ul. |
| IAM-004 | Schimbarea drepturilor invalidează autorizarea veche. | Un token cu autorizare depășită este respins sau reîmprospătat. |
| IAM-005 | SMS OTP este metoda primară, one-time, limitată și auditată. | TTL, retry limit, resend throttling și replay tests trec. |
| IAM-006 | E-mail OTP este permis numai pentru adrese verificate. | Add/verify/login e-mail funcționează end-to-end. |
| IAM-007 | Passkey se înrolează ulterior din profil prin WebAuthn. | Register/assert/delete validează challenge, origin, RP ID și sign counter. |
| IAM-008 | UI OTP suportă auto-advance, paste, Backspace și mobil. | Testele browser trec. |
| IAM-009 | UI OIDC folosește tema și brandingul tenantului. | Light/dark, paleta și numele rămân identice prin redirect. |
| IAM-010 | Refresh tokenul nu este accesibil JavaScriptului. | Cookie HttpOnly/Secure/SameSite, rotație și reuse detection. |
| IAM-011 | Un telefon normalizat aparține unei singure identități globale. | Constrângerea DB și serviciile resping orice al doilea proprietar; migrarea se oprește înainte de acordarea drepturilor dacă găsește un duplicat. |
| IAM-012 | Telefonul devine verificat numai după dovada posesiei prin OTP, iar verificarea este documentată. | Evenimentul append-only păstrează identitatea, metoda, momentul, tenantul și rezultatul fără OTP sau secret; schimbarea telefonului revocă verificarea și cere o verificare nouă. |

## 7. Bootstrap utilizatori Școala Balotești

| ID | Utilizator | Contract |
|---|---|---|
| USR-001 | Thomas Galambos, `+40771364169`, `thomas@eguilde.cloud` | Utilizator global, membership activ `tenant-balotesti`, rol de platformă `platform_super_admin` și rol `admin` în tenant. |
| USR-002 | Stelian Fedorca, `+40744652476` | Membership activ `tenant-balotesti`, rol `admin`, SMS primar; e-mail/passkey ulterior din profil. |
| USR-003 | Diana Ilhan, `+40735091230` | Membership activ `tenant-balotesti`, rol `admin`, SMS primar; e-mail/passkey ulterior din profil. |
| USR-004 | `test@eguilde.cloud` | Identitate E2E izolată la `tenant-balotesti`, toate permisiunile tenantului, OTP fix numai din secret de deployment. |

Bootstrapul este idempotent, nu dublează identități după telefon/e-mail/subiect și nu introduce OTP-uri sau secrete în Git. Nu se inventează adrese de e-mail pentru Stelian sau Diana. Orice adopție a unui telefon verificat existent păstrează o înregistrare auditabilă a provenienței; valorile noi sau modificate nu sunt declarate verificate administrativ, ci numai după OTP reușit.

## 8. RBAC unitar

| ID | Cerință | Acceptanță |
|---|---|---|
| RBAC-001 | Rolurile, permisiunile, rol-permisiune și membership-rol sunt tenant-scoped. | Administrarea unui rol nu modifică alt tenant. |
| RBAC-002 | Drepturile efective provin din grant direct, rol și poziție printr-un singur resolver. | Tokenul, `/api/me` și middleware-ul returnează același set sortat. |
| RBAC-003 | Fiecare endpoint protejat declară permisiunea în router și OpenAPI. | CI eșuează fără `x-required-permission` și `x-tenant-scope`. |
| RBAC-004 | Frontendul construiește meniurile, rutele și acțiunile din sesiune. | Rolurile testate văd exact modulele și acțiunile permise. |
| RBAC-005 | Ascunderea UI nu reprezintă autorizare. | Apelul manual la un endpoint interzis primește 403. |
| RBAC-006 | Superadminul interactiv nu primește bypass RLS implicit. | Cross-tenant folosește flux separat, explicit și auditat. |

## 9. OpenAPI și clientul React

| ID | Cerință | Acceptanță |
|---|---|---|
| API-001 | OpenAPI 3.1.1 acoperă fiecare rută concretă. | Routerul și specificația au aceeași mulțime method/path și operationId unic. |
| API-002 | Requesturile/răspunsurile folosesc DTO-uri închise. | Nu există fallback generic sau `additionalProperties: true` nejustificat pe operații. |
| API-003 | Contractul declară auth, tenant și RBAC. | Fiecare operație nepublică are security, tenant scope și permisiune exactă. |
| API-004 | Backendul validează requesturile, iar contract tests validează răspunsurile. | Payloadurile invalide sunt respinse înaintea handlerului; răspunsurile incompatibile opresc testele. |
| API-005 | TypeScript paths/DTOs și clientul executabil sunt generate. | Regenerarea este deterministă și `git diff --exit-code` rămâne curat. |
| API-006 | Toate operațiile business React folosesc clientul generat. | CI interzice `fetch('/api/...')`, URL-uri și DTO-uri API duplicate în features. |
| API-007 | OIDC protocol rămâne în biblioteca standard OAuth/OIDC. | `/me`, profilul, passkeys și business API folosesc OpenAPI. |
| API-008 | Upload, download și PDF sunt tipizate. | Multipart și răspunsurile binare funcționează prin același contract. |

## 10. Registratură — paritate Costești

| ID | Cerință | Acceptanță |
|---|---|---|
| REG-001 | Lista are aceleași coloane, filtre header, sortare și paginare server-side. | Același fixture produce aceleași rânduri, totaluri și ordine. |
| REG-002 | Headerul și paginatorul sunt sticky; doar rândurile fac scroll. | Teste și capturi desktop/tabletă/mobil. |
| REG-003 | Search este închis implicit și se deschide din lupa coloanei Acțiuni. | Nu există buton search redundant. |
| REG-004 | Intrare/Ieșire/Multiplu au aceleași formulare, validări și efecte. | Scenarii reale, batch și numerotare concurentă. |
| REG-005 | Detalii, istoric, editare, anulare, imprimare, export și Flux au paritate. | Fiecare acțiune persistă corect și respectă stare/RBAC. |
| REG-006 | Persoanele, instituțiile și organizațiile au DTO-uri specializate. | Dialogurile și validările coincid cu Costești. |
| REG-007 | Documentele/dosarele au atașamente, legături, versiuni și audit tenant-safe. | Upload/list/download și cross-tenant tests. |
| REG-008 | Registrele au CRUD, default și secvențe tenant-scoped. | Paritate de câmpuri și acțiuni. |
| REG-009 | Dashboardul expune stats, trend și attention. | Indicatorii coincid pe același set de date. |

## 11. Flux documente

| ID | Cerință | Acceptanță |
|---|---|---|
| FLX-001 | UI conține Coada mea, Mapă Semnături și Evidență completă. | Tabelele, filtrele, totalurile și acțiunile coincid cu Costești. |
| FLX-002 | Documentele și dosarele folosesc tranziții explicite. | Assign compartment/user, claim, send, approve și reject au aceleași reguli. |
| FLX-003 | Tranzițiile sunt concurent-safe și auditate. | `expected_version` previne update-uri pierdute. |
| FLX-004 | Acțiunile depind de rol, asignare și stare. | Matricea rol x stare x acțiune trece în backend/frontend. |

## 12. eArhivă

| ID | Cerință | Acceptanță |
|---|---|---|
| ARC-001 | Nomenclatorul are CRUD și arbore ierarhic. | Câmpurile, validările și comportamentul coincid cu Costești. |
| ARC-002 | Dosarele au create/detail/tree/close și asignare documente. | Regulile de închidere și permisiunile sunt testate. |
| ARC-003 | Ingestia este idempotentă și tenant-scoped. | Upload duplicat detectat; joburile nu schimbă tenantul. |
| ARC-004 | OCR, clasificarea și metadatele au contracte versionate. | Rezultatul păstrează proveniența, confidence și review uman. |
| ARC-005 | Search are full-text, metadate, filtre, sortare și paginare. | Documentul devine căutabil numai în tenantul său. |
| ARC-006 | Preview/download folosește autorizare și URL presemnat scurt. | Object key din alt tenant nu este accesibil. |
| ARC-007 | Adminul vede health, stats, jobs, erori și retry. | Jobul poate fi relansat fără duplicarea arhivei. |

## 13. Admin, profil și UI

| ID | Cerință | Acceptanță |
|---|---|---|
| ADM-001 | Adminul reproduce taburile Costești pentru utilizatori, compartimente, registre, entități, organizații și organigramă. | Coloanele, dialogurile și acțiunile au paritate. |
| ADM-002 | Include administrarea rolurilor, permisiunilor și membershipurilor. | Tenant admin nu poate acorda rol global sau modifica alt tenant. |
| ADM-003 | Profilul se deschide prin numele din partea de jos a drawerului stâng. | Utilizatorul administrează datele, e-mailul, passkeys și sesiunile. |
| ADM-004 | Schimbările de identitate cer autentificare recentă. | Telefon/e-mail/passkey sunt protejate și auditate. |
| UI-001 | Drawer stânga, permanent pe ecrane mari/medii, overlay pe mobil. | Bars apare numai când drawerul nu este permanent vizibil. |
| UI-002 | Top toolbar afișează numele tenantului. | Balotești afișează numele configurat al școlii. |
| UI-003 | Selectorul reproduce tema PrimeReact/Apollo. | Light/dark, preset, primary și surface funcționează și persistă. |
| UI-004 | PrimeReact furnizează componentele/culorile; Tailwind este layout-only. | Auditul nu găsește culori hardcodate nejustificate. |
| UI-005 | Landing page neautentificată este simplă. | Branding, mesaj scurt și login, fără elemente decorative inutile. |

## 14. Gate-uri și ordine

### Gate A — contracte

- migrațiile și schema contract trec;
- OpenAPI este valid și complet;
- clientul generat este sincronizat;
- nu există API business manual nou.

### Gate B — identitate și izolare

- OIDC real Authorization Code + PKCE trece;
- tokenul conține tenant/roles/permissions;
- SMS/e-mail/passkey trec scenariile pozitive și negative;
- matricea RBAC și testele cross-tenant trec.

### Gate C — paritate

- Registratură, Flux, eArhivă și Admin sunt validate separat;
- scenariile golden rulează pe PostgreSQL și storage reale de test;
- capturile desktop/tabletă/mobil sunt comparate cu Costești.

### Gate D — release

- unit, integration, contract, React și Playwright trec;
- fiecare cerință funcțională are un test automat care demonstrează comportamentul, nu doar prezența codului;
- integrarea PostgreSQL/API folosește date realiste pentru minimum doi tenanți și verifică atât răspunsul HTTP, cât și starea persistentă și RLS;
- Playwright parcurge frontendul React și OIDC real, iar efectele sunt validate prin API/DB pentru Registratură, Flux, eArhivă, Admin, profil și RBAC;
- fluxurile obligatorii nu pot fi declarate prin teste `skip`, backend mock-uit în E2E sau rezultate tolerate;
- buildurile sunt reproductibile și scanarea secretelor este curată;
- commitul este publicat direct pe `main`;
- revision-ul din cluster coincide cu commitul și smoke testul public trece.

Ordinea implementării:

1. Contract comun tenant/roluri/permisiuni în token și `/api/me`.
2. Bootstrap utilizatori Balotești și contractele de profil.
3. SMS/e-mail/passkey și OIDC conformance.
4. Client OpenAPI executabil, validatori runtime și blocarea apelurilor manuale.
5. Contracte DB/API și React Registratură 1:1.
6. Flux 1:1.
7. eArhivă 1:1, inclusiv storage/OCR/search.
8. Admin Dashboard și management RBAC.
9. Modulul Școală pe același lanț contractual.
10. Testare integrală, publicare și verificare în cluster.

## 15. Definition of Done

Sistemul este finalizat când fiecare cerință are implementare, contract DB/API, test automat, dovadă de izolare tenant și, pentru UI, dovadă de paritate/responsive. Existența unui endpoint, buton sau tip generat fără comportament real verificat nu închide cerința.
