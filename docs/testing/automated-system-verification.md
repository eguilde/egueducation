# Verificarea automată a sistemului

Acest catalog definește dovezile obligatorii pentru candidatul de release. Un
test de browser cu API mock nu este considerat dovadă de sistem; scenariile
`system` pornesc backendul Go real, OIDC Provider-ul real și PostgreSQL 17.

## Niveluri obligatorii

| Nivel | Comandă / job CI | Dovadă |
|---|---|---|
| Backend unit și contract | `go test ./...`, `go vet ./...` | servicii, validări, paginare, workflow, RBAC, OTP, SMS, outbox, migrații și schema contract |
| React unit și contract | `npm test`, `quality:contracts`, `quality:ui`, `typecheck`, `build` | componente, adaptoare exclusiv OpenAPI, validare runtime, PrimeReact și teme |
| Browser responsive | `npm run e2e` | 17 scenarii desktop/mobil pentru shell, temă, Registratură, Flux, eArhivă, Școală, Admin și profil |
| PostgreSQL integrare | `oidc-postgres-integration`, `archive-postgres-integration` | migrații reale, RLS, identități globale, SMS/OTP, workflow, audit și outbox |
| Sistem UI→DB | `react-real-stack-system-e2e` | React → OIDC/PKCE → RBAC → API → PostgreSQL/MinIO pentru doi tenanți și trei utilizatori |
| Sistem eArhivă | `react-earchiva-real-pipeline-e2e` | upload PrimeReact → PDF/ClamAV → MinIO → Azure OCR emulator → clasificare → metadata → FTS/vector → search izolat |
| Producție | `Production OIDC canary` | activare securizată → OTP fix secret → OIDC real → `/api/me` → Școală/Registratură → logout |

## Scenarii critice acoperite

- OTP cu auto-advance este one-time, HMAC, legat de identitate, tenant și
  sesiunea OIDC exactă; cross-session, cross-host, cod greșit și replay eșuează.
- Telefonul este unic global, nu poate fi marcat verificat de administrator și
  schimbarea lui revocă verificarea și toate challenge-urile active.
- Tokenul conține tenant, roluri, permisiuni și `authz_version`; grant/revoke
  invalidează tokenul vechi, iar replay-ul pe alt hostname primește 401.
- Administrarea prin UI creează utilizator, membership, rol, permisiune și
  verifică atât răspunsul HTTP, cât și rândurile PostgreSQL tenant-scoped.
- Registratura creează Intrare, Ieșire și MULTIPLU, editează, anulează,
  filtrează/sortează/paginează server-side, generează PDF și persistă auditul.
- Aprobarea Flux cere actor independent, versiune optimistă și PDF curat;
  finalizarea produce outbox livrat și un job eArhivă idempotent.
- eArhivă păstrează originalul și artifactul în MinIO, persistă OCR, chunks,
  entities, classification review și embedding și nu expune documentul celuilalt tenant.
- WebAuthn este executat de un authenticator virtual CTAP2 real: profilul
  înregistrează cheia, OIDC verifică assertion-ul și `last_used_at` este persistat.

## Reguli anti-fals-pozitiv

1. Configurațiile `playwright.system.config.ts` și
   `playwright.earchiva-system.config.ts` selectează explicit câte un scenariu;
   suitele cu mock nu pot intra în joburile de sistem.
2. `TEST_DATABASE_URL` este obligatoriu; testele de sistem se opresc dacă DB nu
   există și verifică direct starea persistentă după acțiunea din UI.
3. MinIO pornește explicit cu o versiune fixată și health check; ClamAV și Azure
   au emulatoare de protocol deterministe, nu răspunsuri interceptate în browser.
4. Orice eșec oprește CI și blochează build-ul/promotion. Imaginile sunt
   publicate numai după succesul workflow-ului `Validate application` pe `main`.
5. După rollout, revision-ul public trebuie să coincidă cu commitul sursă;
   eșecul smoke testului produce rollback GitOps automat.
