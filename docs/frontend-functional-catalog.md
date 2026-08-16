# Catalog funcțional frontend eGuEducation

Data auditului: 2026-08-16

Acest document este sursa de adevăr pentru rescrierea React. Frontendul Angular
din `frontend/` rămâne în repository exclusiv ca referință până când fiecare
capabilitate de mai jos are contract OpenAPI, implementare React și teste de
acceptanță echivalente.

## Legendă

- **Implementat**: codul React și adaptorul API există;
- **Verificat local**: există teste automate și build local reușit;
- **Verificat CI/live**: rezultatul a trecut baza efemeră, buildul de container și
  deploymentul public; această stare se acordă numai după promovare;
- **Planificat**: backendul nu oferă încă operația și UI nu o simulează.

## Situația generală

| Zonă | React implementat | Verificare locală | Verificare CI/live |
|---|---|---|---|
| OIDC și sesiune | flux standard PKCE/state/nonce, refresh HttpOnly, RFC 7009, RP logout, `/api/me` fail-closed | unit + integrare PostgreSQL obligatorie în CI | nepromovat încă |
| Shell / navigație | PrimeReact 11, responsive, structură Apollo | unit, UI policy, Playwright | nepromovat încă |
| Landing | minimal și operațional | unit/Playwright/build | nepromovat încă |
| Registratură | toate contractele utilizabile, inclusiv paritate Costești și admin | unit, race, E2E responsive/build | nepromovat încă |
| Flux documente | toate cele 7 operații backend | unit/E2E/build | nepromovat încă |
| eArhivă | toate cele 10 operații backend | unit/E2E/build | nepromovat încă |
| Școală / Education | toate cele 304 operații documentate | 11 teste focalizate/UI/build | nepromovat încă |
| Profil / passkeys/EUDI | toate operațiile backend existente | unit + Playwright/build | nepromovat încă |
| Admin / RBAC / GDPR | toate citirile și mutațiile backend existente | 43 unit totale + 3 Playwright focalizate/UI/build | nepromovat încă |
| OpenAPI/Swagger | 455 rute router + OIDC standard + client generat | generare deterministă, coverage/drift/Redocly, zero fallback-uri ori obiecte unrestricted | nepromovat încă |

Inventarul final de operații este calculat de generator și verificat în CI; nu se
menține manual în acest catalog. Starea de release este detaliată separat în
`react-migration-status.md`.

## 1. OIDC, autentificare și sesiune

### Implementat în React

- discovery OIDC;
- Authorization Code;
- PKCE S256;
- `state` și `nonce`;
- redirect `/auth/start` și callback `/auth/callback`;
- schimb de cod cu `oauth4webapi`;
- refresh cookie HttpOnly;
- access token ținut în memorie după hardening;
- revocare RFC 7009 și RP-Initiated Logout cu redirect înregistrat și `state`;
- passkey/WebAuthn;
- roluri, permisiuni și module obținute din `/api/me`.

### Defecte istorice confirmate și închise în rescriere

- clientul React nu activează DPoP deoarece providerul nu îl publică;
- starea autentificată este fail-closed dacă schimbul de cod ori `/api/me` eșuează;
- testul PostgreSQL efemer acoperă provider -> PKCE/OTP -> token -> `/api/me` ->
  refresh -> logout, inclusiv izolarea RLS pe o conexiune reutilizată;
- discovery, token, revoke, end-session, erorile și assetul OTP sunt documentate
  în contractul OpenAPI;
- scriptul OTP este extern și compatibil CSP, cu auto-avansare, paste,
  Backspace, săgeți și Enter.

### Țintă React

- Authorization Code + PKCE S256, `state` și `nonce` obligatorii;
- validare issuer, audience, semnătură, nonce și tip de token;
- tokenurile nu se scriu în localStorage/sessionStorage;
- PKCE/state/nonce sunt efemere și curățate după o singură utilizare;
- refresh exclusiv prin cookie HttpOnly `Secure`;
- `/api/me` este obligatoriu înainte de starea `authenticated`;
- DPoP este activat numai dacă discovery și providerul îl suportă complet;
- callback idempotent, cu timeout și erori explicite;
- login redirect implicit; popup doar dacă este demonstrată interoperabilitatea;
- teste mock și integrare live pentru happy path, state invalid, nonce invalid,
  token endpoint error, `/api/me` 401 și refresh expirat.

## 2. Shell responsive și navigație

### Țintă obligatorie

- componente UI exclusiv PrimeReact;
- `Toolbar` sus, cu titlu contextual;
- buton PrimeReact `Button` cu icon `pi-bars` în stânga când drawer-ul poate fi
  comutat;
- sidebar/drawer deschis implicit pe desktop și tabletă;
- sidebar/drawer închis implicit pe mobil;
- comutarea breakpoint-ului nu trebuie să lase overlay sau focus blocat;
- navigația în partea de sus a drawer-ului;
- identitatea și acțiunile login/logout/profil în footer-ul drawer-ului;
- click pe numele utilizatorului deschide profilul;
- configurator temă în dreapta toolbar-ului, inspirat din primereact.dev;
- light, dark și system;
- meniul este filtrat central după roluri, permisiuni și module;
- backendul rămâne autoritatea finală pentru acces;
- breadcrumb contextual pentru paginile operaționale.

## 3. Landing page

### De eliminat

- carduri decorative;
- statistici de autentificare/rol fără valoare operațională;
- texte promoționale lungi;
- griduri de module care repetă meniul principal;
- accente și culori hardcoded.

### Țintă

- numele instituției;
- o singură descriere scurtă;
- login dacă utilizatorul nu este autentificat;
- acces direct la activitatea principală dacă este autentificat;
- maximum trei legături operaționale, numai dacă sunt permise;
- layout fluid, lizibil pe 320 px fără scroll orizontal.

## 4. Registratură — paritate Costești

Ruta React canonică este `/registratura`. Aliasurile istorice pot redirecționa,
dar nu trebuie să fragmenteze starea sau breadcrumb-ul.

### Catalog documente

| Funcție | Stare actuală | Criteriu React |
|---|---|---|
| Selector registru | Funcțional | PrimeReact Select; persistat per tenant |
| Intrare | Funcțional | Formular și nomenclatură echivalente Costești |
| Ieșire | Funcțional | Formular și nomenclatură echivalente Costești |
| Intern | Funcțional | Disponibil numai unde registrul permite |
| Multiplu | Funcțional | 1–20 documente, validare înainte de submit |
| Listare | Funcțional | DataTable server-side, paginare/sortare/filtre |
| Filtru număr extern | Funcțional | Filtru vizibil și query documentat |
| Creare | Funcțional | Validare, loading, retry și conflict handling |
| Editare | Funcțional | `expected_version` și conflict 409 vizibil |
| Anulare | Funcțional | ConfirmDialog PrimeReact și motiv obligatoriu |
| Detalii inline | Funcțional | Panel/Drawer responsive |
| Print document | Funcțional | Blob PDF și nume fișier corect |
| Export interval | Funcțional | Maximum 30 zile și rezultat PDF |
| Părți | Funcțional | Persoană fizică/juridică/instituție discriminată |
| Atașamente | Funcțional | Upload real, progres, scanare, retry și download |
| Versiuni | Funcțional | Timeline/DataTable fără schimbare directă de status |
| Istoric workflow | Funcțional | Timeline cu actor, stare, dată și observație |
| Legături documente | Parțial | Creare/listare/ștergere testate |

### Registre

- listare, creare, editare și ștergere;
- registru implicit;
- prefix, număr inițial, număr curent și număr următor;
- tipuri: GENERAL, INTRARI, IESIRI, INTERN, PETITII, CONTRACTE,
  DECIZII, HOTARARI, DISPOZITII;
- politici de vizibilitate pe compartiment și utilizator;
- confirmări exclusiv PrimeReact `ConfirmDialog`;
- numerotare tenant-scoped și teste de coliziune.

### Părți și structuri administrative

- persoane fizice: nume, identificatori, adresă, contact, data/locul nașterii;
- persoane juridice: CUI, registrul comerțului, reprezentant, formă, capital;
- instituții: tip, nivel, site și date de contact;
- departamente;
- organizații;
- organigramă;
- atribuiri utilizator-compartiment;
- politici registru-compartiment.

### Workflow Registratură

Stări canonice:

- `INCOMING`;
- `ALOCAT_COMPARTIMENT`;
- `IN_LUCRU`;
- `FLUX_APROBARE`;
- `FINALIZAT`;
- `ANULAT`.

Acțiuni:

- alocare compartiment;
- alocare utilizator;
- preluare (`claim`);
- trimitere la aprobare;
- aprobare;
- respingere;
- anulare terminal-safe.

UI afișează numai acțiunile permise de stare și permisiune. Serverul validează
aceleași reguli, actorul, compartimentul și versiunea optimistă.

### Teste obligatorii Registratură

- paritate vizuală desktop/tabletă/mobil cu referința Costești;
- Intrare, Ieșire și Multiplu;
- selector registru și registru implicit;
- CRUD document;
- upload curat, infectat, prea mare, scanner ocupat, storage indisponibil;
- workflow complet și tranziții interzise;
- concurență `expected_version`;
- permisiuni read/manage/workflow;
- acces cross-tenant negativ pentru documente, registre, părți și atașamente.

## 5. Flux documente

### Existent

- dashboard;
- taskuri;
- listare paginată, filtre și sortare;
- creare task;
- selectare și tranziție task;
- status, prioritate, responsabil și termen;
- taskuri întârziate;
- aprobări;
- definiții workflow active.

### Limitări determinate de backend, nesimulate

- backendul nu expune o rută de istoric pentru taskurile Workflow;
- administrarea definițiilor este disponibilă separat în Admin numai pentru
  operațiile și permisiunile publicate de backend;
- UI nu inventează tranziții sau istoric care nu există în contract.

## 6. eArhivă

### Implementat în React

- dashboard;
- listare records;
- paginare, sortare și filtre;
- creare record;
- listare și detaliu documente;
- upload, versiuni și taxonomie;
- toate cele 10 operații backend curente, cu loading/error/empty și RBAC.

### Planificat numai dacă backendul va publica operații noi

- transfer, restaurare și eliminare, dacă politica și API-ul le vor permite;
- relații suplimentare cu Registratura și audit arhivistic dedicat, dacă vor fi
  adăugate ca rute backend;
- UI nu afișează controale inactive pentru aceste capabilități absente.

Funcțiile fără suport backend rămân marcate **planificate**, nu sunt simulate în
UI și nu primesc butoane inactive în producție.

## 7. Școală / Education

Zona existentă este extinsă și trebuie migrată fără pierderi:

### Dashboard-uri

- general;
- director;
- rapoarte director;
- secretariat;
- conformitate;
- profesor.

### Guvernanță

- ședințe CA;
- convocare și agendă;
- minute/procese-verbale;
- membri și completitudine;
- voturi;
- hotărâri și rezoluții;
- semnături, finalizare și publicare;
- dosar managerial.

### Personal și carieră

- evidență personal;
- evaluări;
- declarații;
- mobilitate;
- merit/gradații;
- decizii și reglementări;
- comisii;
- rapoarte și exporturi.

### Portofoliu

- portofoliu profesor;
- portofoliu personal;
- documente și cerințe;
- workflow și transfer;
- validare/finalizare;
- dashboard-uri contextuale.

### Criterii transversale Education

- fiecare wizard Angular devine Stepper/Dialog/Drawer PrimeReact după context;
- matrice rol × modul × acțiune centralizată;
- request/response generate din OpenAPI;
- funcțiile ascunse în lipsa permisiunii;
- stări loading/empty/error consistente;
- testele cu mocks sunt completate cu contract tests și happy paths live.

## 8. Profil și metode de autentificare

- afișare profil și instituție curentă;
- editarea numai a câmpurilor permise;
- schimbarea telefonului invalidează verificarea până la reverificare;
- passkeys: listare și înregistrare; eliminarea este planificată deoarece
  backendul nu expune încă endpoint de ștergere;
- preferință OTP;
- EUDI numai dacă backendul o declară disponibilă;
- logout și terminare sesiune;
- setările nu pot modifica roluri sau tenant membership;
- profilul se deschide din identitatea din footer-ul drawer-ului.

## 9. Admin, RBAC și GDPR

### Admin platformă

- utilizatori;
- roluri și permisiuni;
- module;
- poziții;
- membership-uri tenant;
- clienți OIDC;
- invitații și provisioning, dacă backendul le implementează;
- audit și evenimente de securitate.

### Admin Registratură

- registre;
- părți;
- departamente;
- organizații;
- organigramă;
- atribuiri utilizatori;
- politici de vizibilitate.

### GDPR

- catalog operații existent;
- drepturi și exporturi;
- retenție;
- audit;
- acces strict pe permisiuni.

Niciun ecran admin nu presupune că rolul din frontend autorizează operația;
toate răspunsurile 403 sunt tratate explicit și fără redirect loops.

## 10. Multi-tenancy

- tenantul rezultă din host + membership + claim instituție;
- UI nu trimite un tenant arbitrar în locul celui rezolvat de backend;
- cache keys includ contextul instituției;
- schimbarea contextului golește toate cache-urile și selecțiile tenant-scoped;
- selectorul de tenant apare numai dacă backendul oferă explicit această opțiune;
- registrul selectat este persistat separat per tenant;
- testele negative cross-tenant sunt obligatorii pentru toate domeniile.

## 11. Reguli UI și design

- PrimeReact Styled este unica bibliotecă de componente;
- Aura/custom preset cu design tokens primitive, semantic și component;
- culorile nu se scriu în clase Tailwind, CSS sau JSX;
- Tailwind se folosește numai pentru layout, spacing, grid, flex, dimensionare și
  breakpoint-uri;
- PrimeIcons este sistemul standard de iconuri;
- Apollo este referință obligatorie de structură, nu sursă de cod vechi;
- toate dialogurile, confirmările, mesajele, tabelele, meniurile, inputurile,
  uploadurile și overlay-urile sunt PrimeReact;
- accesibilitate WCAG AA, tastatură completă și focus management;
- densitate enterprise, fără carduri decorative sau text promoțional inutil.

## 12. Definiția finalizării unei capabilități

O capabilitate poate fi marcată migrată numai când:

1. are operația și schema documentate OpenAPI;
2. clientul React este generat sau tipat din OpenAPI;
3. UI folosește exclusiv componente/tokens conforme;
4. loading, empty, validation, error și success sunt implementate;
5. RBAC și tenant behavior sunt testate;
6. există test unit/contract și E2E pentru fluxul critic;
7. este verificată la 320 px, tabletă și desktop;
8. nu introduce tokenuri, licențe sau PII în repository/loguri;
9. Angularul poate fi eliminat pentru acea rută fără pierdere funcțională.
