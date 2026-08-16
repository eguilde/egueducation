# Arhitectura frontendului React eGuEducation

Status: ADR implementat; promovarea în producție este urmărită separat în
`react-migration-status.md`

## 1. Strategie de migrare

Noua aplicație este creată în `frontend-react/`. Aplicația Angular din
`frontend/` rămâne neschimbată ca referință și opțiune de rollback până când
catalogul funcțional este migrat și verificat integral.

Rescrierea nu se face peste Angular deoarece ar elimina sursa de comparație și
ar combina două schimbări cu risc mare: schimbarea frameworkului și schimbarea
contractelor API.

## 2. Stack

- React 19;
- TypeScript strict;
- Vite;
- React Router, rute lazy;
- PrimeReact 11 Styled ca unică bibliotecă UI;
- Aura ca preset inițial și preset custom numai prin design tokens;
- PrimeIcons;
- Tailwind CSS 4 numai pentru layout;
- `oauth4webapi` pentru OIDC;
- adaptori API injectabili și stare locală race-safe pentru server state;
- formulare TypeScript controlate, validate din contractele backend;
- Vitest, React Testing Library și Playwright;
- client API generat din OpenAPI 3.1.

Nu se adaugă Redux/Zustand până când o nevoie de state global care nu este
server state, auth, theme sau shell este demonstrată.

## 3. Reguli PrimeReact și Tailwind

### PrimeReact

Toate controalele aplicației sunt PrimeReact:

- Button;
- Toolbar;
- Sidebar și Drawer;
- Menu/PanelMenu;
- Popover;
- Dialog și ConfirmDialog;
- Toast/Message;
- DataTable;
- InputText, Select, DatePicker, AutoComplete și celelalte inputuri;
- FileUpload;
- Tabs/Stepper;
- Timeline;
- Avatar, Tag, Badge și ProgressSpinner.

### Tailwind permis

- `display`, flex și grid;
- gap, padding și margin;
- width/height/min/max;
- positioning;
- overflow;
- responsive breakpoints.

### Tailwind interzis

- culori text/fundal/border;
- shadow;
- gradient;
- radius vizual;
- stilizarea controalelor;
- palette utilities.

CSS-ul aplicației poate utiliza exclusiv variabile `var(--p-...)` provenite din
tema PrimeReact. Nu sunt permise valori hex, rgb sau hsl pentru UI.

## 4. Apollo PrimeReact

Apollo este blueprint obligatoriu pentru structura enterprise:

- sidebar ierarhic;
- topbar compact;
- meniu pe grupuri;
- breadcrumb;
- identitate/profil;
- configurator de layout și temă;
- dashboard-uri dense, fără marketing;
- full-page layout separat pentru landing și autentificare.

Showcase-ul public Apollo auditat folosește Next 13, React 18 și PrimeReact 10.
Prin urmare, nu se importă codul, SCSS-ul, theme link switcher-ul sau demo-urile
vechi. Structura este reimplementată pe React 19 + PrimeReact 11 Styled, cu
design tokens moderne. Orice sursă Apollo comercială poate fi incorporată doar
dacă este furnizată separat și licența permite acest lucru.

## 5. Shell responsive

### Toolbar

- stânga: Button PrimeReact cu `pi-bars` când navigația poate fi comutată;
- centru: titlul instituției și titlul paginii, fără badge-uri promoționale;
- dreapta: buton temă care deschide Popover.

### Navigație

- sub 768 px: Drawer modal, închis implicit și închis după navigare;
- de la 768 px: Sidebar persistent, deschis implicit și totuși comutabil;
- schimbarea breakpoint-ului închide corect overlay-ul și restabilește focusul;
- preferința persistentă este tenant + user scoped;
- navigația și route guards consumă același `featureCatalog`.

### Footer navigație

- anonim: text scurt și Login;
- autentificat: Avatar, nume clicabil și Logout pe același rând;
- numele deschide `/profil` într-un flow accesibil.

### Theme Popover

- light, dark, system;
- preset aprobat;
- primary/surface numai din liste controlate de tokens;
- fără selector de culoare arbitrar;
- fără flash de temă la inițializare.

## 6. Rutare

```text
/
/auth/callback
/auth/logout

/registratura
/flux-documente
/earchiva
/scoala
/administrare
/profil
```

URL-urile Angular vechi primesc redirecturi compatibile. Fiecare intrare din
`featureCatalog` definește ruta, eticheta, iconul, modulul, rolurile,
permisiunile, regula all/any și prezența în navigație.

## 7. OIDC React

Se folosește redirect, nu popup. Redirectul este mai robust pentru OTP multi-pas,
mobile browsers, popup blockers și accesibilitate.

### Flux

1. discovery și verificare issuer;
2. generare verifier/challenge S256, state și nonce;
3. tranzacție efemeră în sessionStorage;
4. redirect authorization;
5. validare exactă state;
6. code exchange cu `credentials: include`;
7. validare ID token issuer/audience/signature/nonce prin `oauth4webapi`;
8. access/id token numai în memorie;
9. `/api/me` obligatoriu înainte de `authenticated`;
10. eliminare code/state din URL;
11. refresh single-flight prin cookie HttpOnly;
12. logout server + client.

### DPoP

Auditul a confirmat o incompatibilitate actuală: Angular forțează DPoP, dar
discovery-ul providerului nu declară DPoP și tokenurile nu sunt emise coerent
cu `cnf.jkt`. Clientul React folosește Bearer când discovery nu oferă DPoP.

DPoP poate fi activat numai după ce backendul:

- publică `dpop_signing_alg_values_supported`;
- emite token cu `cnf.jkt`;
- definește nonce behavior;
- trece testele token binding și replay.

### Stări client

```text
anonymous
discovering
redirecting
exchanging
hydrating-profile
authenticated
refreshing
recoverable-error
security-error
```

State, nonce, PKCE și issuer mismatch sunt security errors și impun restart
complet. Eroarea `/api/me` nu poate fi înghițită și nu produce o sesiune falsă.

## 8. API, tenant și RBAC

- OpenAPI 3.1 este contractul unic;
- DTO-urile și clientul sunt generate;
- starea persistată în browser este cheiată cu tenantul acolo unde este necesar;
- tenantul rezultă din host + membership + token, nu din input arbitrar;
- schimbarea contextului golește query cache și state tenant-scoped;
- UI ascunde acțiunile nepermise, backendul autorizează fiecare request;
- 401 declanșează maximum un refresh concurent;
- 403 este afișat explicit și nu produce redirect loop;
- erorile API sunt normalizate la `Problem`.

## 9. Licența PrimeReact

Licența este furnizată providerului prin `VITE_PRIMEUI_LICENSE`. Valoarea reală:

- nu se comite;
- nu apare în `.env.example`;
- nu este transmisă subagenților;
- nu se scrie în loguri/test snapshots;
- este injectată prin secretul controlat al pipeline-ului.

Fiind aplicație statică, valoarea folosită de provider poate ajunge în bundle;
acest lucru trebuie verificat față de termenii licenței PrimeUI. Nu este tratată
ca un secret de autentificare al aplicației.

## 10. MCP

- MCP oficial PrimeReact: `@primereact/mcp@11.1.0`, variantă Styled;
- oferă documentație, exemple, API metadata și `validate_usage`;
- React nu publică încă un MCP oficial stabil;
- nu instalăm un pachet comunitar nevalidat pretinzând că este oficial;
- documentația react.dev, TypeScript, Storybook și browser/CDP acoperă temporar
  verificarea React;
- după introducerea Storybook, addon-ul MCP al Storybook poate expune catalogul
  componentelor proprii, fără a înlocui PrimeReact MCP.

## 11. Testare și quality gates

- typecheck strict;
- Vitest/RTL;
- teste cu adaptori/mocks injectabili;
- OpenAPI drift check;
- PrimeReact usage audit;
- audit care respinge controale native și alte UI libraries;
- audit care respinge culori hardcoded și Tailwind palette classes;
- Playwright desktop 1440×900, tabletă 768×1024 și mobil 360×800;
- controale accesibile și scenarii desktop/mobil Playwright;
- comparație funcțională și responsive Registratură față de Costești;
- E2E OIDC live pentru login, refresh și logout;
- cross-tenant negative tests.

## 12. Ordine de livrare

1. OpenAPI și contract OIDC stabil;
2. scaffold React și quality gates;
3. shell, landing și theme system;
4. OIDC + `/api/me` live;
5. Registratură cu paritate Costești;
6. Flux documente;
7. eArhivă;
8. Școală/Education;
9. Admin, profil și GDPR;
10. CI, imagini imutabile și promovare GitOps controlată;
11. eliminarea Angular numai după paritate verificată.
