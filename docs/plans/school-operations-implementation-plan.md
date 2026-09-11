# Plan de implementare — management școlar public/privat

Data: 2026-09-11
Dependență: [catalogul funcțional](../requirements/school-operations-management-catalog.md) și [designul tehnic](../design/school-operations-platform-design.md)

## Principiu de livrare

Fiecare etapă livrează vertical DB → serviciu → OpenAPI → client React → UI → teste reale. Nu se adaugă meniuri sau formulare care nu au contract backend și persistență securizată. `main` rămâne publicabil numai când toate gate-urile existente și testele noii verticale sunt verzi.

## Etapa 0 — catalog, reguli și acceptanță

Livrabile:

- matrice juridică public/privat/confesional și overlay-uri de finanțare/program;
- responsabilități școală–fondator/UAT–ordonator–furnizor;
- registru oficial de surse și proprietar uman pentru validarea fiecărui policy pack;
- scenarii de acceptanță semnate pentru un tenant public și unul privat;
- decizia de graniță dintre control economic și contabilitate statutară.

Gate: nicio regulă schimbătoare nu este hardcodată și fiecare cerință are sursă, proprietar și dată de revizuire.

## Etapa 1 — foundation public/privat

Implementare:

- migrații pentru profil instituțional, policy packs, assignments, overrides și evaluations;
- resolver policy și endpointuri `/api/institution/regulatory-profile` și `/api/institution/capabilities`;
- admin UI PrimeReact pentru clasificare, aprobare și impact analysis;
- migrare inițială `unclassified`, fără presupunere publică;
- integrarea resolverului cu navigația și action guards;
- RLS, RBAC, audit și versioning.

Teste: public, privat, confesional, privat cu fond public; policy expirat/ambiguu; falsificare în body; schimbare fără efect retroactiv; cross-tenant/institution.

## Etapa 2 — contracte, utilități și conformitate

Implementare:

- furnizori peste `app_parties`;
- contracte, versiuni, acte adiționale, obligații, SLA, garanții și alerte 180/90/30;
- puncte de consum, contoare, citiri și reconciliere factură;
- calendar comun SSM/PSI/securitate și acțiuni corective;
- legături obligatorii Registratură–Workflow–eArhivă;
- dashboard operațional pe rol.

Teste: contract expirat, reînnoire, act adițional, alertă idempotentă, utility anomaly, arhivă indisponibilă, permisiuni pe responsabil.

## Etapa 3 — achiziții MVP

Implementare:

- plan anual, necesar, buget, aprobare, dosar, loturi, oferte/evaluare, atribuire, comandă și recepție;
- procedură și documente obligatorii din policy;
- CPV și controale de agregare/fragmentare;
- conflict de interese și segregarea atribuțiilor;
- overlay public pentru instituții private/operațiuni cu fond public.

Teste: flux public complet, flux privat comercial, privat cu overlay public, inițiator≠CFP≠aprobator, concurență și idempotency.

## Etapa 4 — catering, stocuri și patrimoniu

Implementare:

- program alimentar generic și template-uri versionate;
- eligibilitate, meniu/alergeni, comandă, livrare/lot, recepție, distribuție, incidente și reconciliere;
- depozite, loturi, stoc, inventar, bunuri, custodie, QR/barcode și mentenanță;
- integrarea recepției din Procurement și a facturii în Finance.

Teste: diferențe public/privat, absențe/porții nedistribuite, lot retras, temperatură neconformă, expirare, reconciliere cantitativă și inventar mobil.

## Etapa 5 — HR operațional

Implementare:

- extinderea fără duplicare a `education_personnel`;
- stat de funcții, posturi, raporturi de muncă, acte adiționale, FTE/normă;
- pontaj, programe, concedii, absențe și substituții;
- calificări, formări, aptitudine ca status, SSM/PSI și payroll inputs;
- închiderea automată a asignărilor/accesului la încetare.

Teste: public/privat, delegări temporale, date salariale/medicale least-privilege, încetare și retenție, export/reconciliere payroll.

## Etapa 6 — control economic și financiar

Implementare:

- exerciții/perioade, surse, bugete, centre de cost și angajamente;
- facturi, documente justificative, three-way match, plăți și reconciliere;
- public: clasificație, CFP, ordonanțare, Trezorerie/export;
- privat: taxe, contracte cu părinții, scadențare, reduceri/burse și creanțe;
- active și amortizare ca integrare Finance–Logistics;
- export contabil verificabil și audit complet.

Teste: buget insuficient, factură duplicată, 3-way mismatch, perioadă închisă/reversal, segregare, profil TVA și tenant isolation.

## Etapa 7 — contabilitate nativă și integrări

Numai după validarea etapelor anterioare:

- ledger dublă partidă, planuri de conturi versionate și period close;
- SEAP/SICAP, RO e-Factura/ANAF, Trezorerie/bănci și REGES-ONLINE;
- payroll complet și declarații numai cu expertiză de specialitate;
- portal furnizori și importuri/reconcilieri controlate.

Fiecare integrare folosește outbox, idempotency, retry/dead-letter, observabilitate și operator reconciliation.

## Etapa 8 — pilot și rollout

- seed-uri și date sintetice pentru minimum un tenant public și unul privat;
- import/reconciliere și shadow mode;
- testare UAT pe roluri și validare juridică/contabilă/SSM;
- activare graduală pe module și rollback documentat;
- runbook, alerte și indicatori operaționali.

## Definition of Done pentru fiecare verticală

- policy/sursă versionată și validată;
- migrare, constrângeri, FK compozite, `FORCE RLS` și schema contract;
- teste cross-tenant, cross-institution, cross-role și public/privat/overlay;
- handlers tranzacționali, optimistic concurrency, audit și idempotency;
- OpenAPI closed cu metadate RBAC/tenant/policy și zero drift;
- client și DTO React generate, fără rute dinamice sau obiecte generice;
- PrimeReact mobile-first, accesibilitate și toate stările operaționale;
- E2E real React → OIDC → API → PostgreSQL/storage;
- failure/retry/dead-letter și reconciliere;
- documentație de migrare, rollback și operare;
- aceeași imagine de aplicație deservește ambele tipuri de școli.
