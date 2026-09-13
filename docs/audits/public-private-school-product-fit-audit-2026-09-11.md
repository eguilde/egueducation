# Audit de adecvare — școli publice, private și confesionale

Data: 2026-09-11
Scop: codul executabil Go/PostgreSQL/React și contractul OpenAPI; documentele istorice nu sunt folosite ca dovadă de implementare

## Verdict

eGuEducation are fundația arhitecturală potrivită pentru a deservi școli publice și private din același produs: tenant din host, instituție din sesiunea OIDC, RBAC, RLS, profil reglementar versionat, policy packs și evaluări imuabile. Nu este necesar și nu este acceptabil un fork public/privat.

Suportul nu poate fi declarat încă complet. Modelul actual aplică policy-ul executabil numai unor verticale înguste — publicațiile de conformitate și contractele operaționale — iar modelul Stage1B normalizat nu este încă sursa runtime. O școală poate avea simultan programe/niveluri cu statute diferite, finanțări diferite pe ani sau beneficiari și o încadrare de achiziții care nu rezultă automat din forma juridică ori din existența banilor publici.

### Re-evaluare după implementarea Stage1B și Stage2 din 2026-09-11

- formele juridice acceptate sunt acum exclusiv `public` și `private`; confesionalul este contractat numai ca overlay tipizat peste `private` și este încă refuzat fail-closed până la comanda atomică de validare/persistență;
- requestul de profil nu mai acceptă `public_funding`, `is_contracting_authority` sau `treasury_required`; UI-ul React nu le mai poate edita;
- sursa profilului este un DTO închis cu tip, citare, articol, emitent, URL oficial, date și SHA-256; pentru profil aprobat/activ backendul cere HTTPS+SHA-256 și completează actorul și momentul verificării;
- scrierea creează în aceeași tranzacție sursa, profilul v2 cu lineage și legătura profil–sursă, dar citirea și `RequireCapability` folosesc încă proiecția legacy 0135; cutover-ul nu este complet;
- Stage2 are acum backend/OpenAPI/client React/UI PrimeReact pentru furnizori, contracte, obligații și lifecycle, cu filtre/sort/paginare server-side; utilitățile, compliance, workerul de arhivare și E2E PostgreSQL real rămân lipsă.

Prin urmare, verdictul rămâne **parțial / neacceptat pentru producție**. Elementele de mai sus sunt progres executabil, nu dovadă de paritate completă.

Decizia de produs este:

`nucleu comun → formă juridică → overlay confesional → ofertă autorizată/acreditată → finanțare/program/an → aplicabilitate pe operație → opțiuni instituționale permise`

## Baza juridică verificată

| Sursă oficială | Consecință pentru produs |
| --- | --- |
| [Legea nr. 198/2023, forma consolidată](https://legislatie.just.ro/Public/DetaliiDocumentAfis/309185), în special art. 19, 27–28 și 135–140 | nucleul educațional comun; autonomie privată; personalitate juridică; finanțare și condiții pentru privat/confesional |
| [ROFUIP — Ordinul nr. 5.726/2024](https://legislatie.just.ro/Public/DetaliiDocument/289484), modificat prin [Ordinul nr. 6.226/2025](https://legislatie.just.ro/Public/DetaliiDocument/302026) și [Ordinul nr. 4.261/2026](https://legislatie.just.ro/Public/DetaliiDocumentAfis/312069) | reguli comune de organizare, contract educațional, catalog electronic, REGES-ONLINE/EDUSAL și diferențe de guvernanță/fondator |
| [HG nr. 993/2020](https://legislatie.just.ro/Public/DetaliiDocument/234510) și [HG nr. 994/2020](https://legislatie.just.ro/Public/DetaliiDocumentAfis/255166) | evaluarea și standardele naționale de calitate se aplică școlilor de stat, private și confesionale |
| [HG nr. 69/2024](https://legislatie.just.ro/Public/DetaliiDocument/295485), cu modificările ulterioare, și [OUG nr. 28/2026](https://legislatie.just.ro/Public/DetaliiDocument/311756) | finanțarea de bază a privatului/confesionalului depinde de acreditare/autorizare, nivel, beneficiar, regim fără taxă și anul aplicabil; dobândirea statutului în cursul anului poate produce finanțare începând cu anul financiar următor |
| [Legea nr. 98/2016](https://legislatie.just.ro/Public/DetaliiDocument/179076), art. 4 | privatul nu devine automat autoritate contractantă prin simpla finanțare publică; încadrarea entității și uneori a contractului trebuie demonstrată separat |
| [ARACIP — hotărâri 2026](https://aracip.eu/categorii-documente/Hot%C4%83r%C3%A2ri%20Consiliul%20ARACIP%202026) | deciziile și denumirea autorității trebuie păstrate ca metadate versionate; logica nu hardcodează denumirea instituției emitente |

Acest audit definește cerințe software, nu o opinie juridică. Policy pack-urile de producție necesită aprobarea unui responsabil juridic/financiar și revalidare la schimbarea formei consolidate.

## Ce există în cod

| Capabilitate | Dovadă | Verdict |
| --- | --- | --- |
| Scope din host/sesiune, membership și RBAC | middleware-ul comun și serviciile nu primesc tenant/instituție din payload | fundație adecvată |
| Profil instituțional versionat | migrațiile 0135 și 0137–0145; `backend/internal/institution/service.go` | dual-write atomic v1/v2, sursă structurată, identity map, proiecție API și bindings v2 implementate pentru scrierile noi; citirea/evaluarea live sunt încă legacy, iar istoricul necesită migrarea controlată 0146 |
| Policy packs, assignments, overrides additive și evaluări imuabile | `school_policy_pack_versions`, `school_policy_assignments`, `school_policy_overrides`, `school_policy_evaluations` | implementat ca fundație |
| Intersecție modul–RBAC–policy | `backend/internal/institution/policy_resolver.go` | implementată pentru catalogul mic de capabilități existent |
| Administrare React a profilului | `frontend-react/src/features/institution/RegulatoryProfileWorkspace.tsx` | folosește public/private și sursă juridică structurată; overlay-ul confesional nu are încă flux executabil |
| Locații, oferte și decizii de autorizare | `backend/internal/institution/offerings.go`, rutele `/api/institution/locations`, `/education-offerings`, `/offering-authorizations` și `EducationOfferingsWorkspace.tsx` | listare cu filtre/sort/paginare server-side, creare idempotentă și actualizare/dezactivare concurentă implementate; guard-urile DB și handler refuză invalidarea autorizărilor dependente, selectorii consumă toate paginile, iar decizia are sursă verificată și înlocuire prospectivă; guard-ul de admitere lipsește |
| Guvernanță, clase/elevi/înscrieri, personal, evaluări, portofolii, rapoarte și dovezi | rutele `/api/education/*`, tabelele și workspace-urile React | fundație funcțională comună public/privat; integrarea policy este incompletă |
| Policy aplicat unei mutații reale | POST/PATCH/DELETE `/api/education/compliance/publications` și comenzile existente `/api/school-operations/contracts*` | dovadă pe două verticale limitate; ambele persistă încă evaluarea legacy până la cutover-ul v2 |
| Test public/privat/overlay | `frontend-react/e2e/system/institution-policy-real-stack.spec.ts` și testele PostgreSQL ale fundației | existent pentru publicații; matricea juridică completă lipsește |

## Gap-uri confirmate

### P0 — verticale juridice încă incomplete

1. Oferta/autorizarea/acreditarea are verticală DB→Go→OpenAPI→React pentru listare, creare și corecție/dezactivare effective-dated pe nivel/program/specializare/locație/capacitate/interval, inclusiv guard-uri pentru dependențe. Nu guvernează încă admiterea/înscrierea și nu are E2E PostgreSQL prin HTTP; migrarea întărită trebuie validată pe un PostgreSQL disposable real.
2. Finanțarea are instrumente/evaluări versionate în DB, iar booleanul legacy este read-only; lipsesc comenzile, UI-ul și integrarea cu operațiile.
3. Achizițiile au evaluare tri-state și `indeterminate` în DB/evaluator, dar nu au comandă API/UI pe entitate și contract/proiect.
4. `confessional` a fost eliminat ca formă juridică și modelat drept overlay tipizat peste privat; comanda este încă fail-closed până la validarea atomică a cultului/protocolului.
5. Rolurile instituționale au schemă `app_parties` effective-dated, dar fondatorul/finanțatorul/ordonatorul legacy nu sunt încă migrate și administrate prin API/UI.
6. Requestul nou are sursă juridică structurată și verificată pentru aprobări; read model-ul și policy pack-urile legacy încă proiectează `source_reference(s)` și trebuie contractate în cutover.

### P0 — aplicare incompletă

1. `backend/internal/institution/service.go` alege pack-uri din forma juridică, booleanul de finanțare și `program_codes`; nu dovedește eligibilitatea finanțării sau aplicabilitatea achizițiilor.
2. `backend/internal/institution/policy_resolver.go` are doar capabilitățile de nucleu/publicare/controale generale.
3. În router, auditul static identifică 198 operații mutabile `/api/education/*`, dintre care numai 3 folosesc `RequireCapability` (POST/PATCH/DELETE publicații). Celelalte 195, inclusiv înscrieri, guvernanță, personal, evaluări și portofolii, nu sunt încă guvernate de profilul public/privat.
4. Nu există contract educațional tipat și verticala privată de taxe/scadențe/reduceri/burse/refunduri.
5. Nu există registru operațional complet pentru ofertă/autorizare/acreditare, rețea școlară și CEAC/RAEI/evaluare externă.

### P1 — validare și demonstrație

1. Backendul verifică enum-uri și sintaxă, dar nu coerența juridică: acreditat fără act, finanțare fără sursă/perioadă, confesional fără cult sau ofertă suspendată folosită la înscriere.
2. React permite administrarea câmpurilor declarative, dar nu oferă wizard cu documente, aprobări, istoric și impact analysis.
3. E2E nu acoperă: privat acreditat, privat autorizat fără taxă, privat autorizat cu taxă, confesional, ofertă suspendată/retrasă, locații diferite ori privat finanțat public care nu este autoritate contractantă.

## Adaptarea aprobată pentru arhitectură

### Persistență

Se păstrează `school_institution_profiles` ca antet versionat și se adaugă agregatele canonice din catalog:

- ofertă educațională + autorizare/acreditare;
- instrument de finanțare;
- evaluare aplicabilitate achiziții;
- profil confesional;
- contract educațional + catalog taxe;
- calitate/evaluare externă;
- apartenență anuală la rețeaua școlară.

Booleenele vechi devin proiecții read-only pentru compatibilitate. Migrarea nu inventează date: valorile neverificate devin `requires_review`, iar scrierile reglementate sunt fail-closed.

### Backend și policy

- resolverul primește contextul operației și, unde este necesar, oferta, anul și sursa de finanțare;
- fiecare comandă salvează evaluarea policy imuabilă;
- catalogul de capabilități include înscriere, finalizare studii, guvernanță, contracte educaționale, finanțare, achiziții și raportare;
- aplicabilitatea achizițiilor se evaluează distinct de finanțare;
- suspendarea sau retragerea produce efect numai prospectiv și nu rescrie documentele istorice.

### OpenAPI și React

- DTO-urile nu acceptă tenant, instituție, rol sau rezultat de policy;
- formularele folosesc selectoare de ofertă/act/sursă și dovezi, nu câmpuri text ori booleene fără proveniență;
- aceeași pagină PrimeReact afișează pașii și acțiunile publicate de server;
- navigația este intersecția modulelor, RBAC și capabilităților, nu condiții `schoolIsPublic`;
- toate tabelele noi respectă filtrarea, sortarea și paginarea server-side, header/paginator sticky și action column.

## Matrice minimă de acceptare

| Scenariu | Rezultat obligatoriu |
| --- | --- |
| școală publică acreditată | nucleu comun + policy public aplicabil |
| privată acreditată, program obligatoriu | nucleu comun + policy privat + instrument de finanțare numai dacă există temei efectiv |
| privată autorizată, fără taxă | eligibilitatea finanțării este evaluată pe ofertă și an |
| privată autorizată, cu taxă | nu moștenește finanțarea destinată unității autorizate fără taxă |
| privată cu finanțare publică, fără criterii Legea 98 | nu este tratată automat ca autoritate contractantă |
| entitate/contract care îndeplinește criteriile Legea 98 | primește overlay-ul procedural și documentar aplicabil |
| confesională | policy privat + overlay confesional, fără pierderea regulilor comune |
| ofertă suspendată/retrasă | operațiile viitoare sunt blocate; istoricul rămâne accesibil și neschimbat |
| două locații cu autorizări diferite | operația este permisă numai în locația/oferta acoperită |
| schimbare de policy pack | tranzacțiile vechi păstrează snapshotul și sursele folosite |

## Concluzie de implementare

Nu schimbăm tehnologia și nu duplicăm modulele. Adaptarea corectă este normalizarea fundației policy și extinderea ei în toate verticale. Ordinea imediată este Etapa 1B din plan, apoi verticale contract-first; orice funcție nouă trebuie demonstrată pentru cel puțin un tenant public, unul privat și un scenariu privat cu overlay condițional.

## Matrice cerință → tip instituție → stare în cod → gap → acceptare

Statusul de mai jos este stabilit prin căutarea rutelor, handlerelor, migrațiilor,
clientului React și testelor executabile. Existența unei migrații fără handler și
test de integrare nu este considerată implementare.

| Domeniu | Tip | Stare verificată în cod | Gap concret | Criteriu de acceptare |
| --- | --- | --- | --- | --- |
| Nucleu clase/elevi/înscrieri | public + privat | parțial funcțional: `education_students`, `education_student_enrolments`, rute și client React cu query server-side | nu există flux de admitere cu ofertă, capacitate, documente, decizie și contestație; înscrierea de clasă nu dovedește admiterea | aceeași API permite admitere numai pe ofertă/locație autorizată și testează public, privat și ofertă expirată |
| Profil și policy | public + privat; confesional overlay | scriere v2, lineage și sursă tipată; request/UI fără booleene juridice; OpenAPI închis | read/capabilities rămân legacy; overlay confesional este fail-closed; integrarea policy în mutațiile education nu este completă | profil versionat, effective-dated, surse tipate, `indeterminate` fail-closed și snapshot policy la fiecare comandă reglementată |
| Contract educațional | public + privat | schemă Stage1B există în `school_education_contracts`, fără verticală API/React demonstrată | lipsesc handler-ele, versiuni semnate, legarea de elev/ofertă/perioadă și eArhivă | creare–aprobare–semnare–încetare prin OpenAPI, audit și E2E real; istoria este imuabilă |
| Taxe, scadențe, reduceri, burse, refunduri | privat/confesional; bursele pot fi publice | nu există flux funcțional API/React; doar câmpuri/proiecții generale și model incipient de funding | lipsesc catalog taxe, ledger/sold, plăți, burse și reguli de gratuitate; publicul nu trebuie să primească taxe comerciale | privat cu taxă calculează server-side soldul și refundul; public fără taxă blochează fluxul comercial; eligibilitatea bursă păstrează dovada |
| Finanțare publică | public + privat eligibil | tabele `school_funding_instruments` și evaluări există, fără comandă/UI completă și integrare cu înscriere/raportare | `public_funding` rămâne proiecție; lipsesc beneficiarul/oferta/an și revalidarea operațională completă | evaluare pe ofertă/an/beneficiar cu sursă, plafon și snapshot; privatul nu primește automat finanțare |
| Achiziții | public + privat condiționat | model de evaluare există, fără flux de achiziții executabil și fără aplicare transversală | booleanul `is_contracting_authority` nu poate decide singur; lipsesc entitate vs contract/proiect, criterii și dovezi | decizie distinctă pentru entitate și contract/proiect; public/privat cu criterii aplicabile primesc procedura, `indeterminate` blochează |
| Contracte operaționale (utilități/SSM/PSI) | public + privat | furnizori + contracte + obligații + lifecycle au Go/OpenAPI/client/UI PrimeReact cu operații tabel server-side | nu există încă E2E React→DB, utility/compliance APIs, archive worker și alerte 180/90/30 | contractul și obligațiile sunt tenant/institution scoped, cu versiune concurentă, audit, outbox idempotent și UI PrimeReact |
| Guvernanță | public + privat/confesional | cockpit, meeting/minutes/votes/decisions și permisiuni există; policy public/privat aplicat limitat | mandat/delegare și overlay confesional nu acoperă toate operațiile; matricea de roluri nu este demonstrată complet | CA/CP/comisii, director și adjunct cu perioade/domenii; aceeași suită E2E testează toate tipurile |
| Personal/RBAC | public + privat | `education_personnel`, assignments și capabilități de bază există | segregarea HR/financiar/achiziții și matricea contextuală public/privat nu sunt complet testate | directorul, profesorul, secretariatul, HR, financiarul, achizițiile, arhivarul și inspectorul primesc numai capabilități explicite |
| Raportări/export | public + privat | catalog de rapoarte și CSV/PDF cu filtrare/sortare/paginare server-side există | lipsesc dimensiunile policy/ofertă/finanțare și controalele distincte pentru date private/sensibile | raportul include snapshotul juridic la data raportării; exportul este minimizat, auditat și refuză scope-ul nepermis |
| Tenant/instituție | public + privat | resolver host + sesiune OIDC, RLS și instituție implicită există | selecția multi-instituție și verificarea tuturor noilor agregate nu sunt închise | hostul stabilește tenantul, sesiunea stabilește instituția; niciun body/header nu poate schimba scope-ul; teste cross-tenant/cross-institution |
| Burse/facilități | public + privat/confesional eligibil | nu există bounded context executabil; finanțarea generală nu este substitut | lipsesc tipuri versionate, eligibilitate pe an/sursă/regim fără taxă, plăți/reversări și documente justificative | valori și calendare vin din policy pack; eligibilitatea tri-state și fiecare plată păstrează sursa/evaluarea |
| REGES-ONLINE/EDUSAL | public + privat | există personal și RBAC general, dar nu desemnare/jurnal de transmitere tipat | lipsesc decizia, mandatul, segregarea și integrarea documentară | responsabilul este desemnat în scris, numai pe intervalul mandatului; transmiterile/erorile sunt auditate fără expunerea salariilor |
| Catalog electronic | public + privat | evaluări și rapoarte există; nu există închidere anuală probatorie completă | lipsesc snapshotul perioadei, tipărirea/înregistrarea, hashul și dovada eArhivă | anul se închide o singură dată prin comandă idempotentă; exportul semnat/hash-uit este arhivat și perioada închisă nu mai poate fi rescrisă |

### Concluzie operațională

În cod există o bază comună reutilizabilă, dar nu există încă paritate funcțională
public/privat. Cea mai mare lipsă este confundarea „profilului instituției” cu
eligibilitatea fiecărei operații. Următoarele implementări trebuie să fie verticale
și contract-first: admitere + contract educațional, taxe/burse, finanțare,
achiziții și apoi integrarea lor cu policy, raportare și eArhivă. Confesionalul se
testează ca `private + overlay`, iar o schimbare de statut nu rescrie istoricul.

### Corecții de valabilitate introduse în catalog

- ROFUIP nr. 4.183/2022 și calendarele 2024–2025 sunt depășite și nu pot alimenta reguli active;
- Ordinul nr. 3.858/2026 este sursa publicată pentru portofoliul profesional; proiectul anterior rămâne numai trasabilitate;
- cuantumurile burselor, costurile standard, calendarele și pragurile de achiziții sunt date effective-dated, nu constante în cod;
- contractul educațional necesită retenție pe durata școlarizării plus 2 ani după plecarea elevului și versiuni append-only pentru actele adiționale;
- autorizarea/acreditarea trebuie urmărită pe combinația efectivă unitate–nivel/program–limbă–formă–locație, nu printr-un status global;
- responsabilitatea REGES-ONLINE/EDUSAL și închiderea anuală a catalogului electronic sunt fluxuri cu actor, mandat, jurnal și dovadă eArhivă.
