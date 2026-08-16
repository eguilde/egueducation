# Starea rescrierii React și a livrării

Data verificării: 2026-08-16

Acest document separă codul implementat de validarea locală, validarea CI și
starea deploymentului. Frontendul Angular din `frontend/` rămâne în repository
ca referință și fallback istoric; imaginea nouă de producție este construită din
`frontend-react/` numai după trecerea tuturor gate-urilor.

## Rezumat

| Domeniu | Implementat în React | Verificare disponibilă | Stare release |
|---|---|---|---|
| Shell și landing | Toolbar, drawer dreapta responsive, navigație RBAC, profil/login în footer, selector Aura light/dark/system, landing minimal | unit, policy UI, Playwright desktop/mobil | așteaptă CI și smoke live |
| OIDC și sesiune | Authorization Code, PKCE S256, state, nonce, validare protocol, tokenuri numai în memorie, refresh HttpOnly, `/api/me` fail-closed, RFC 7009 și RP-Initiated Logout | unit + test PostgreSQL efemer pentru provider/token/refresh/logout/RLS | testul PostgreSQL trebuie să treacă în CI |
| Registratură | paritate Costești pe selector registru, Intrare/Ieșire/MULTIPLU, filtre, CRUD/versiuni, workflow, PDF/export, atașamente scanate, părți, structuri, atribuiri și linkuri cu RBAC separat | unit, race tests, E2E responsive, policy UI, build | așteaptă smoke live pe date tenant reale |
| Flux documente | dashboard, definiții, filtre, taskuri, creare, detaliu, dosar și tranziții permise | unit/E2E existente, typecheck/build | așteaptă CI/live |
| eArhivă | dashboard, dosare, filtre, documente, upload, detaliu, versiuni și taxonomie pe toate rutele backend disponibile | unit/E2E existente, typecheck/build | așteaptă CI/live |
| Școală | toate cele 304 operații Education documentate sunt expuse: CRUD, subresurse, guvernanță, dosare, PDF/CSV, comenzi, dashboarduri, filtre, cataloge și sumarizări | 11 teste focalizate, policy UI, typecheck/build | așteaptă CI/live pe rolurile principale |
| Administrare/GDPR | toate resursele read și toate mutațiile backend disponibile, formulare PrimeReact și gate per permisiune | 43 teste unitare, 3 Playwright responsive, policy UI, build | așteaptă CI/live |
| Profil | editare profil, verificări, listare/înrolare passkeys, activare EUDI când serverul permite | unit + Playwright | nu există endpoint backend pentru ștergere passkey; UI nu îl simulează |
| OpenAPI/Swagger | acoperire router, operații standard OIDC, Swagger UI, embed backend și client TypeScript generat | 455 rute, generare deterministă, coverage/drift, Redocly și gate-uri semantice fără fallback-uri generice | așteaptă CI/live |

## Decizii aplicate

- React 19, TypeScript strict și Vite;
- PrimeReact 11 Styled cu Aura și PrimeIcons; licența este injectată ca secret de
  build și nu este stocată în Git;
- Tailwind este folosit numai pentru layout, spațiere, dimensiuni și breakpoint-uri;
- auditul UI respinge controalele native și culorile hardcodate/Tailwind;
- Apollo PrimeReact este referință structurală enterprise, nu sursă de cod;
- tenantul este derivat exclusiv de backend din host, token și membership; clientul
  nu trimite un tenant arbitrar;
- accesul `super_admin` de tenant nu mai activează bypass RLS sau privilegii globale;
- fixture-urile migrațiilor sunt sintetice; provisioningul administratorilor reali
  este o operație externă, auditată;
- endpointurile care nu pot păstra integritatea datelor sunt fail-closed. Stagingul
  metadata-only pentru atașamente Registratură răspunde `410`; uploadul multipart
  scanat este unica rută de scriere suportată.

## Autentificare

Providerul publică discovery, JWKS, Authorization Code + PKCE S256, refresh,
revocare RFC 7009 și `end_session_endpoint`. Clientul React:

1. generează PKCE, state și nonce efemere;
2. validează issuer/audience/semnătură/nonce prin biblioteca OIDC;
3. nu persistă access, ID sau refresh token în storage;
4. acceptă starea `authenticated` numai după `/api/me` verificat de backend;
5. folosește cookie HttpOnly pentru refresh;
6. revocă sesiunea internă și finalizează RP logout cu `id_token_hint`, redirect
   înregistrat și state de unică folosință.

JavaScript-ul OTP este livrat ca asset extern compatibil CSP. Auto-avansarea,
paste, Backspace, săgețile și Enter sunt testate; nu este necesar `unsafe-inline`.

## OpenAPI

Documentul este disponibil la `/api/openapi.json`, iar Swagger UI la `/api/docs`.
CI regenerează determinist contractul, verifică driftul față de router și clientul
TypeScript, validează OpenAPI 3.1 cu Redocly și respinge:

- rute lipsă sau operationId duplicate;
- request body generic ori cu `additionalProperties` nelimitat;
- scheme request goale sau cu câmpuri din alt DTO;
- pagini cu elemente anonime/nemodelate, obiecte goale ori response fallback-uri
  comune unor endpointuri cu forme diferite;
- securitate, tenant scope sau RBAC lipsă pe operațiile protejate.

## Multi-tenancy și securitate

- toate conexiunile de request primesc context tenant determinist;
- flagul privilegiat este resetat primul, iar o conexiune cu bind/clear eșuat este
  distrusă, nu returnată în pool;
- încărcarea sesiunii citește memberships/roluri prin conexiunea tenant-scoped;
- rolul tenant `super_admin` nu poate activa bypass RLS sau operații globale;
- testul PostgreSQL folosește pool cu o singură conexiune și verifică lipsa scurgerii
  contextului între doi tenants;
- atașamentele Registratură sunt bounded, scanate cu clamd și stocate cu chei unice;
- migrațiile nu mai conțin identități reale ori administratori bootstrap.

## Condiții rămase înainte de producție

1. rularea testului PostgreSQL efemer în GitHub Actions;
2. PR, CI verde și merge controlat în `main`;
3. build multi-arhitectură cu licența PrimeUI ca secret;
4. promovare simultană backend/frontend prin digest în GitOps;
5. rollout Kubernetes, verificarea revision/digest, health, OIDC discovery și SPA;
6. test live de login OTP/callback/refresh/logout și audit vizual/funcțional pe tenant.

Până la îndeplinirea acestor puncte, clusterul live rulează versiunea anterioară și
nu trebuie descris ca remediat.
