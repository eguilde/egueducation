# Design tehnic — platforma de management școlar public/privat

Data: 2026-09-11
Decizie: un singur modular monolith, un singur model multi-tenant și diferențe policy-driven

## Context

Backendul actual Go/Chi/PostgreSQL, contractul OpenAPI și frontendul React oferă fundația corectă pentru extindere. Nu există justificare tehnică pentru microservicii sau două produse public/privat înainte de existența unor limite de scalare ori integrare demonstrate.

Designul țintă adaugă bounded contexts coerente, păstrând Registratura, Workflow, eArhiva, identitatea și auditul drept servicii comune.

## Arhitectura logică

```text
Host + sesiune OIDC
        │
        ▼
Tenant / instituție / membership / RBAC
        │
        ▼
Regulatory Profile + Policy Resolver + Capabilities API
        │
        ├── Education / guvernanță / portofolii
        ├── Contracts & Utilities
        ├── Procurement
        ├── Catering
        ├── Logistics & Assets
        ├── SSM / PSI / Security
        ├── HR
        └── Finance & Accounting
                 │
                 ▼
Registratură + Workflow + eArhivă + Audit + Outbox
```

## Reguli de responsabilitate

- Tenant context rezolvă hostul și sesiunea; niciun bounded context nu acceptă scope de securitate din body.
- Policy Resolver decide capabilități și precondiții; bounded context-ul validează din nou regula în tranzacția de scriere.
- Fiecare comandă relevantă salvează snapshotul policy folosit.
- Starea de business aparține modulului; Workflow păstrează taskul și aprobarea, nu o copie concurentă a stării.
- Registratura păstrează înregistrarea oficială, eArhiva păstrează documentul și versiunile, iar modulul de business păstrează relația și metadatele operaționale.
- `app_parties` rămâne identitatea comună pentru furnizori și parteneri.
- `education_personnel` rămâne identitatea comună pentru angajați; HR o extinde fără duplicate.

## Model de politici

Profilul instituțional și policy packs sunt effective-dated. Evaluarea produce un set de capabilități tipate, nu un blob liber interpretat de UI.

```text
common.ro
  + legal-form.ro.public | legal-form.ro.private
  + confessional.<recognized-cult> (dacă este aplicabil)
  + offering.<level/program/location> + authorization.<status/version>
  + funding.<instrument/year> | program.<code/year>
  + procurement-applicability.<entity/contract>
  + approved institutional options
  = immutable policy evaluation snapshot
```

Policy packs conțin reguli validate prin JSON Schema, sursa normativă, checksum, versiune și aprobator. Override-urile sunt permise numai pentru chei declarate configurabile și nu pot diminua o obligație imperativă.

`school_institution_profiles` este antetul juridic versionat, nu întreaga sursă de adevăr. Ofertele educaționale și actele de autorizare/acreditare sunt modelate per nivel/program/specializare/locație; finanțările sunt instrumente per an și beneficiari; aplicabilitatea achizițiilor este o evaluare separată pe entitate și, unde legea o cere, pe contract/proiect. Booleenele de compatibilitate sunt proiecții read-only calculate din aceste agregate și nu autorizează direct operații.

Resolverul primește un `OperationPolicyContext` tipat cu operația, oferta, data efectivă, sursa de finanțare/program și identificatorul resursei. Contextul de securitate rămâne exclusiv server-derived. Lipsa unei clasificări obligatorii, o ofertă suspendată/retrasă, o sursă expirată sau o evaluare ambiguă produce fail-closed și un cod de remediere, fără a afecta citirea istoricului.

## Model de date transversal

Toate agregatele folosesc:

- UUID și cheie naturală scoped unde este necesar;
- `tenant_code` + `institution_id` și FK compozite;
- `version` pentru optimistic concurrency;
- `created_at/by`, `updated_at/by` și audit append-only;
- `policy_evaluation_id` pentru efecte juridice/financiare;
- stare explicită cu tranziții validate;
- `FORCE ROW LEVEL SECURITY` și indexuri tenant/instituție/stare/termen;
- outbox pentru efecte externe și idempotency pentru comenzi/importuri.

Nu se șterg documente finale, recepții semnate, postări contabile sau decizii. Corecțiile folosesc versiune, anulare, stornare ori reversal, după domeniu.

## Capabilities API și frontend

`GET /api/institution/capabilities` întoarce numai capabilitățile efective ale sesiunii: intersecția dintre modulele tenantului, roluri/permisii, relația contextuală, profilul/policy pack-ul și starea obiectului.

React păstrează aceleași pagini PrimeReact pentru public și privat. Policy-ul poate modifica:

- pașii obligatorii ai wizardului;
- documentele și aprobările cerute;
- acțiunile disponibile;
- etichetele determinate legal;
- rapoartele și integrările disponibile.

Policy-ul nu poate modifica arbitrar layoutul, culorile sau contractul de securitate. Toate culorile rămân tokenuri PrimeReact, Tailwind rămâne rezervat layoutului, iar DataTable-urile folosesc filtrare/sortare/paginare server-side, header/paginator sticky și action column completă.

## Fluxuri critice

### Contract-to-renewal

Draft → verificare → aprobare → semnare → înregistrare → activare → obligații/alerte → acte adiționale versionate → expirare/reziliere → arhivare.

### Procure-to-pay

Necesar → rezervare buget → aprobare → procedură policy-driven → evaluare/atribuire → comandă/contract → recepție → factură → three-way match → CFP/aprobare → plată → postare/export.

### Catering

Eligibilitate snapshot → comandă → livrare lot → recepție cantitativă/calității → distribuție → incident/retragere/risipă → reconciliere → factură → raport program.

### HR lifecycle

Selecție/decizie → contract/numire → post/FTE → acces și instruiri → pontaj/concedii → payroll input → evaluare/formare → modificare/încetare → închiderea accesului și retenție.

## Failure modes

- profil absent/ambiguu: citire cu avertisment, scriere reglementată blocată;
- conflict de versiune: `409` cu posibilitate de reload;
- furnizor sau contract expirat: comandă blocată;
- factură duplicată: unicitate furnizor + număr + dată + instituție;
- arhivă indisponibilă: `archive_pending`, retry/dead-letter, fără status fals de arhivare;
- integrare externă indisponibilă: coadă, backoff, operator retry și reconciliere;
- perioadă contabilă închisă: numai reversal într-o perioadă deschisă;
- worker duplicat: idempotency și `SKIP LOCKED`;
- acces cross-scope: `404` nediferențiabil;
- profil privat + fond public: overlay-ul public este derivat server-side și nu poate fi eliminat din request.

## Decizii de etapizare

- MVP Finance este control economic și export verificabil; un ledger statutar complet vine numai după validare profesională.
- Payroll complet, declarații, REGES-ONLINE, SEAP/SICAP, RO e-Factura, Trezorerie și bănci sunt integrări ulterioare, cu contracte și reconciliere proprii.
- CCTV, biometrie și IoT nu intră în nucleul inițial.
- Pilotarea se face cu minimum un tenant public și unul privat, folosind același build.
