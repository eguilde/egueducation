# Audit de conformitate - modulul Scoala

Data auditului: 2026-09-09

Verdict: **PARTIAL - aplicatia nu poate fi declarata inca 100% conforma sau complet implementata.**

Auditul a verificat codul executabil, migrarile, rutele HTTP, contractul OpenAPI, clientul React si testele. Documentele Markdown existente nu au fost folosite drept dovada a implementarii.

## 1. Surse normative si ghiduri verificate

- [Legea invatamantului preuniversitar nr. 198/2023](https://legislatie.just.ro/Public/DetaliiDocumentAfis/271896), in forma consolidata aplicabila;
- [Ordinul MEC nr. 3.858/2026 si metodologia-cadru privind portofoliul profesional](https://legislatie.just.ro/Public/DetaliiDocumentAfis/310372), Monitorul Oficial nr. 392/08.05.2026;
- Anexele Ordinului nr. 3.858/2026: structura-cadru cu cinci sectiuni si modelul declaratiei;
- [Ordinul ME nr. 7.386/2024](https://legislatie.just.ro/Public/DetaliiDocument/291176) privind profilul si standardele profesionale ale cadrului didactic;
- [ROFUIP aprobat prin Ordinul nr. 5.726/2024](https://legislatie.just.ro/Public/DetaliiDocumentAfis/301726), cu modificarile ulterioare, inclusiv [Ordinul nr. 4.261/2026](https://legislatie.just.ro/Public/DetaliiDocumentAfis/312069);
- [Legea nr. 214/2024](https://legislatie.just.ro/Public/DetaliiDocument/285178) privind semnatura electronica, marca temporala si serviciile de incredere;
- GDPR, Legea nr. 190/2018 si legislatia arhivistica aplicabila;
- Ghidul ISJ Timis pentru directorii unitatilor de invatamant 2024-2025, utilizat ca ghid operational si confruntat cu actele consolidate;
- proiectul metodologiei din martie 2026, folosit numai comparativ; forma finala aprobata prevaleaza.

## 2. Matrice de acoperire a codului

| Domeniu | Baza de date si izolare | Backend si OpenAPI | React | Teste | Verdict |
| --- | --- | --- | --- | --- | --- |
| CA, CP, CEAC, sedinte, membri, voturi, minute si hotarari | modele si scope institutional existente | operatii si relatii existente | liste, relatii si wizard | integrare limitata; E2E cu API simulat | PARTIAL |
| Decizii si publicare | modele si RLS existente | fluxuri si OpenAPI existente | lista si relatii | lipseste fluxul E2E real | PARTIAL |
| Management institutional | modele si RLS existente | documente manageriale si operatii | relatii si wizard | lipsesc teste de flux complete | PARTIAL |
| Regulamente | modele, versiuni si RLS existente | operatii existente | lista si relatii; fara wizard dedicat | lipsesc integrarea si E2E | PARTIAL |
| Comisii | tabelele exista; remediere RLS adaugata in acest lot | operatii existente | lista, membri si completitudine | necesita E2E real pe roluri | PARTIAL |
| Personal, dosar si disciplinar | modele si RLS existente | operatii detaliate | wizard si relatii | lipseste cazul complet DB-API-UI | PARTIAL |
| Evaluarea cadrelor didactice | modele si RLS existente | autoevaluare, criterii, apeluri si rezultate | wizard si relatii | acoperire statica/unitara insuficienta | PARTIAL |
| Declaratii generale | modele si RLS existente | CRUD existent | UI generic | lipsesc teste dedicate | PARTIAL |
| Mobilitate | modele si RLS existente | documente, scoruri, apeluri si decizii | wizard si relatii | lipseste cazul complet | PARTIAL |
| Gradatie de merit | modele si RLS existente | documente, scoruri, apeluri si decizii | wizard si relatii | lipseste cazul complet | PARTIAL |
| Portofoliul profesorului | proprietar UUID, institutie, unicitate si RLS | endpointuri `/me`, procedura, declaratii, OPIS si lifecycle | workspace dedicat profesorului | unit/integration si E2E real React-OIDC-API-PostgreSQL trecute in CI | PARTIAL avansat |
| RBAC si tenant host-based | tenant/institutie in scope; RLS extins pentru tabelele tardive | middleware si permisiuni pe rute | navigatie si actiuni conditionate de sesiune | dovada reala profesor/director/platform admin si refuz cross-user/cross-tenant; matricea tuturor rolurilor ramane incompleta | PARTIAL |
| Contract DB-API-React | constrangeri si OpenAPI generate | 506/506 rute, 341 Education | adaptoare manuale inca accepta rute dinamice | control de drift existent, dar nu complet tipizat | PARTIAL |

## 3. Portofoliul profesional - cerinte confirmate in cod

Acoperite in implementarea curenta:

1. proprietar de portofoliu imutabil si derivat din utilizatorul autentificat;
2. izolare cumulativa tenant, institutie, membership si proprietar;
3. unicitate pentru proprietar, institutie si an scolar;
4. cele cinci sectiuni obligatorii din art. 6 si anexa nr. 1;
5. procedura institutionala versionata, cu stari draft, aprobata, publicata, inlocuita si retrasa;
6. legarea portofoliului de procedura publicata aplicabila;
7. reguli de sectiune obligatorii stabilite de procedura, verificate de server la depunere;
8. atasarea numai a documentelor eArhiva eligibile si snapshot al versiunii/hash-ului la depunere;
9. OPIS regenerat atomic si ordonare cronologica;
10. declaratii emise de server, versionate, confirmate explicit si legate de proprietarul real;
11. eliminarea afirmatiilor client-controlled despre autenticitate si consimtamant;
12. fluxul draft/returned -> submitted -> returned/validated;
13. incetarea activitatii, termen de pastrare calculat de baza de date si legal hold;
14. retragere logica fara hard-delete al dovezii;
15. actiuni administrative si self-service separate in UI.
16. export probatoriu autoritativ, generat exclusiv de server, cu manifest versionat, hash SHA-256, snapshot de versiune si provenienta storage; manifestul este imuabil, tenant-scoped si accesul self-service verifica proprietarul.

## 4. Blocante pentru conformitate 100%

| Prioritate | Lipsa | Criteriu de inchidere |
| --- | --- | --- |
| P0 | Transfer real intre unitati/tenanti | destinatia este FK tenant+institutie, pachetul este imuabil, receptorul autentificat accepta explicit in tenantul destinatie, iar predarea/receptia sunt auditate |
| P0 | RBAC institutional coerent | permisiunea moderna pentru administrarea portofoliilor scolii este aliniata explicit cu operatiile legacy; modificarea necesita decizie explicita de autorizare |
| P1 | Pachete de valorificare | snapshot separat si limitat la scop pentru licentiere/definitivat, grade, evaluare, mobilitate, dezvoltare, inspectie, evaluare externa si distinctii |
| P1 | Delegarea directorului adjunct | mandat explicit cu emitent, domeniu, inceput, sfarsit, revocare si audit; fara mostenire globala automata |
| P1 | Metadate si istoric uniforme | aceleasi reguli de versionare, retragere, motiv, actor si timp pentru toate tipurile de dovezi si documente manageriale |
| P1 | Semnatura/sigiliu/marca temporala | integrare verificabila pentru documentele cu efect juridic, conform clasei de semnatura cerute de actul aplicabil |
| P1 | Contract TypeScript complet | eliminarea rutelor string si a obiectului generic `EducationRecord`; request/response generate per operatie |
| P1 | E2E real complet | OIDC -> React -> API -> PostgreSQL/storage pentru fiecare rol, domeniu si refuz cross-user/cross-tenant |
| P2 | Sticky footer uniform | paginatorul face parte din cadrul fix al fiecarui tabel, iar numai randurile au scroll |

## 5. Dovezi de validare ale lotului curent

- `go test ./...`: trecut;
- `go test -tags integration ./internal/education`: compilat si trecut local; scenariile PostgreSQL cu rol non-bypass/RLS au trecut in CI;
- Vitest: 27 fisiere, 112 teste trecute;
- E2E browser pentru fisierul Education: 3/3 trecute dupa alinierea fixture-urilor la declaratiile server-issued si dialogul de confirmare;
- E2E sistemic real React -> OIDC -> RBAC -> API -> PostgreSQL/storage: trecut in [GitHub Actions run 34391818214](https://github.com/eguilde/egueducation/actions/runs/34391818214), inclusiv cele cinci sectiuni, OPIS, submit/return/remediere/validate si refuz cross-user/cross-tenant;
- TypeScript: trecut;
- build productie React: trecut;
- politica PrimeReact si culori exclusiv din tema: trecuta;
- politica de contract frontend: trecuta;
- OpenAPI: 506/506 operatii router, 506 operationId unice, 341 operatii Education cu schema completa.
- build-ul imaginilor, promovarea GitOps si verificarea reviziei publice: trecute in [GitHub Actions run 34392098849](https://github.com/eguilde/egueducation/actions/runs/34392098849);
- canary OIDC de productie PKCE + OTP: trecut in [GitHub Actions run 34392646149](https://github.com/eguilde/egueducation/actions/runs/34392646149).

## 6. Regula de declarare a conformitatii

Nicio cerinta nu trece la `PASS/100%` numai pentru ca exista un ecran sau un endpoint. Sunt obligatorii cumulativ: constrangere si izolare DB, autorizare server-side, contract OpenAPI, client React tipizat, UI functionala, audit si teste reale proportionale cu riscul. Pana la inchiderea blocantelor din sectiunea 4, verdictul ramane `PARTIAL`.
