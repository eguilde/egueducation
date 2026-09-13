# Portofoliul profesorului — verificare curentă

Prioritatea confirmată de utilizator este portofoliul profesional al profesorului.
Admiterea și autorizările nu sunt continuate în această etapă. Acest raport nu
declară finalizat modulul Școală și nu înlocuiește catalogul canonic.

## Dovezi executate

### Checkpoint suplimentar — flux profesor/director

Auditul independent al codului confirmă că portofoliul NU este încă complet:

- PRT-017: exportul propriu este un manifest JSON, nu un pachet care include
  opisul și dovezile autorizate. Accesul UI trebuie aliniat și pentru export/read own.
- PRT-009: backendul permite PATCH pentru documentul propriu și păstrează istoricul,
  dar clientul React și dialogul de corectare a documentului lipsesc.
- PRT-019: profesorul nu poate consulta procedura publicată aplicată portofoliului.
  Este necesară o proiecție proprie, fără enumerarea procedurilor altor instituții.
- PRT-015: calculul `ready_for_transfer` precedă încărcarea numărului de transferuri
  în `portfolio_transfer_summary.go`; calculul și un test pozitiv rămân de corectat.

Scenariul browser extins a ajuns la validare, confirmând în DB cinci documente,
cinci intrări în opis și două revizii. Rularea a fost totuși FAIL: ultima aserție
căuta un buton dezactivat, deși UI ascunde corect acțiunile după validare.
Asserția verifică acum mesajul de validare și absența acțiunilor de modificare.
O rulare ulterioară a expus o problemă reală la liste mai lungi: opțiunile
selectorului eArhivă ieșeau din viewport. Lista are acum înălțime limitată și scroll.
Nu se folosește click forțat și nu se șterg datele pentru a masca acest defect.

Au fost corectate și:
- formularul de editare a ciornei trimite exclusiv câmpurile permise, fără ID,
  proprietar sau stare primite de la server (anterior PATCH 400 în browser);
- etichetele din pagina profesorului transmit textul drept children, conform
  implementării PrimeReact instalate, în loc de proprietatea veche `value`;
- fixture-urile testelor de revizie respectă toate câmpurile contractului generat.

Verificare locală după aceste corecții: 43 teste PASS în patru suite
(TeacherPortfolioWorkspace, PortfolioReviewWorkspace, PortfolioOwnUpload, API).
Rerularea E2E extinsă a trecut: `1 passed (2.0m)`, exit code 0, la 16:29
în 12 septembrie 2026. Fluxul verifică cele cinci secțiuni, declarațiile,
regenerarea opisului, transmiterea, returnarea de către director, corectarea
observațiilor ciornei, retransmiterea, validarea și interfața read-only finală.
Corectarea unui document individual nu este acoperită și rămâne lipsa PRT-009.
Runnerul lansat cu permisiune pentru administrarea propriilor procese a închis
automat serverele locale și a terminat fără intervenție. `npx tsc -b`,
`quality:ui` și `quality:contracts` au trecut și ele după corecții.
OIDC, Go, PostgreSQL și MinIO sunt reale în mediul izolat; OCR, antivirusul și
verificatorul de semnături rămân emulatoare. Această probă nu este una live.
Nicio publicare/push nu este revendicată în acest checkpoint.

### PRT-009 — verificarea corectării documentului

Testul backend
`TestPortfolioDocumentVersionHistoryIsTenantScopedAndIncludesImmutableArchiveProvenance`
a fost rerulat separat cu PostgreSQL temporar: PASS în 9,95 secunde.
Harness-ul creează o bază și un rol restricționat dedicate testului, aplică
migrările, verifică schema și elimină fixture-ul la final. Prima încercare,
cu rolul migrator fără CREATEROLE, a eșuat la pregătirea harness-ului; rerularea
a folosit administratorul exclusiv al instanței PostgreSQL de test, nu a extins
drepturile aplicației. Operațiunile testate rulează prin rolul restricționat.
Sunt verificate editarea, istoricul cu instantanee, proveniența arhivei,
filtrarea istoricului, respingerea unei referințe de arhivă indisponibile și
respingerea accesului din alt tenant.

Interfața are acum acțiunea de editare în coloana de acțiuni, dialog precompletat
și adaptorul OpenAPI PATCH propriu. Se transmit exclusiv câmpurile cererii
OwnPortfolioDocumentRequest; nu se inventează un câmp de motiv absent din contract.
Acțiunea este ascunsă în stările needitabile. Verificarea integrată React:
46 teste în patru suite PASS, politicile UI și API PASS.

Testul browser a fost extins pentru a corecta descrierea unui document returnat,
a verifica istoricul vechi/nou și hash-ul arhivei, apoi a confirma răspunsul 409
la tentativa de editare după validare. Rularea finală a extinderii a trecut:
`1 passed (2.0m)`, exit code 0, la 16:50 în 12 septembrie 2026.
Sunt confirmate în browser editarea și închiderea istoricului; în API și DB,
instantaneele vechi/noi, hash-ul arhivei păstrat și refuzul 409 după validare,
fără suprascrierea descrierii. Serverele de test s-au închis automat.
Prima execuție a extinderii a confirmat PATCH 200, metadatele corectate în DB,
hash-ul neschimbat și ambele instantanee în istoric, dar a eșuat la închiderea
dialogului: Dialog.Close nu avea conținut vizibil. A fost adăugată pictograma
și un test de regresie. Rerularea a demonstrat că pictograma font `pi-times`
nu era randată în configurația instalată: butonul rămânea invizibil. Corecția
folosește acum `Times` din `@primeicons/react`, aceeași bibliotecă SVG utilizată
de shell. Testul verifică SVG-ul și închiderea (PASS); proba browser finală a trecut.
`go test ./internal/education -count=1` a trecut în 3,604 secunde; buildul React
a trecut. Celelalte utilizări de pictograme font din modul trebuie reverificate.

### PRT-019 — procedura aplicată portofoliului propriu

Ruta proprie `GET /api/education/portfolios/me/{recordID}/procedure` și panoul
React sunt implementate local. Răspunsul conține procedura exact asociată și
regulile sale, nu cea mai recentă procedură din instituție. Permisiunea este
`education.portfolios.read_own`, cu verificarea proprietarului și instituției.
Panoul afișează versiunea, sursa, perioada, calendarul, regulile de acces,
formatele, păstrarea și transferul. Lipsa asocierii într-un portofoliu vechi este
explicată fără fallback la o procedură aleasă arbitrar.

OpenAPI: PASS după regenerare, 630 operațiuni totale și 399 Școală. Un defect al
generatorului pentru structuri declarate în alt fișier producea string în locul
obiectului procedurii/regulilor; răspunsul reutilizează acum schemele canonice
prin referințe explicite, verificate de testul OpenAPI. Nu se duplică manual
câmpurile procedurii. Testul backend PostgreSQL pentru proprietar, tenant străin,
alt proprietar, procedură nouă fără legătură, lipsa asocierii și versiune istorică
a trecut (4,60 secunde în test, 6,648 secunde pachetul). Au trecut și testele
de pachet education/server. Clientul include acum un validator runtime generat
pentru această operațiune; un test negativ verifică respingerea scalarilor în
loc de obiecte. Rularea browser pornită înainte de finalizarea regenerării a
eșuat în timpul reîncărcării Vite; nu este o dovadă de funcționare a acestei rute.
Se reia după compilarea integrată, fără modificări de sursă în timpul probei.
Rularea stabilă a identificat și o cursă la crearea portofoliului: butonul putea
deschide panoul vechii selecții înainte de terminarea salvării. Butonul este acum
dezactivat în timpul salvării/reîncărcării, iar E2E așteaptă codul noului portofoliu.
Testul de regresie și celelalte teste ale paginii/panoului au trecut: 12 PASS.
Clientul API: 33 teste PASS; TypeScript, build React, politici UI/API: PASS.

Proba browser finală: `1 passed (2.0m)`, exit code 0, 12 septembrie 2026,
17:20. Confirmă exact ID-ul procedurii aplicate și cele cinci reguli, afișarea
titlului și închiderea dialogului, apoi întregul flux de upload, transmitere,
returnare, corectare a documentului, istoric, validare și blocare a editării.
Aceasta este verificare locală izolată, nu publicare în producție. Exportul complet
PRT-017 și restul auditului canonic rămân obligatorii; modulul nu este declarat complet.
Pictogramele paginii profesorului sunt acum SVG PrimeIcons, nu clase de font
fără resursă încărcată; testul verifică SVG în butoanele de adăugare/istoric/editare/ștergere.

### PRT-015 — dovada calculului

Calculul disponibilității a fost mutat după încărcarea statisticilor de transfer.
Patru cazuri unitare PASS: portofoliu complet cu transfer, blocaje de verificare,
documente absente și transfer absent. Această probă unitară nu înlocuiește încă
o probă pozitivă a sumarului prin HTTP și PostgreSQL și nu dovedește transferul complet.

Pe baza PostgreSQL de verificare izolată, comanda
`go test -tags=integration ./internal/education -count=1 -timeout 240s`
a trecut în 87,808 secunde. Sunt incluse testele pentru identitate și scope,
declarații, istoric documente, opis atomic, procedură și lifecycle, export,
transfer între tenanturi și pachete de valorificare. Rezultatul nu demonstrează
interfața publicată, OIDC în browser sau conformitatea integrală.

## Lipsuri confirmate în interfața profesorului

- PRT-012 / UI-002: tabelul de revizii omitea `notes`, data și numărul de
  documente lipsă, deși API-ul propriu le returnează după verificarea
  proprietarului. Profesorul nu avea instrucțiuni vizibile pentru remediere.
- PRT-006 / PRT-019: versiunea procedurii aplicate nu este explicată în
  workspace-ul profesorului; existența câmpului în API nu dovedește UX complet.
- PRT-015: lipsesc accesul și istoricul transferului din workspace-ul propriu.
- UI-006: formularul atașează documente deja existente în eArhivă. Ruta proprie
  `GET /education/portfolios/me/archive-documents` doar listează; uploadul
  `POST /earchiva/documents` cere `earchiva.manage`. Lipsește traseul de upload
  autonom limitat la portofoliul propriu. Remedierea nu trebuie să acorde
  profesorului administrarea întregii arhive.
- Testul real de portofoliu este inclus într-un scenariu mare cu alte module;
  este necesară o probă E2E independentă profesor → director → remediere.

## Referință normativă reverificată

### Upload propriu — E2E local verificat, publicare încă neconfirmată

Au fost adăugate parserul închis `earchiva/portfolio_upload.go` și testele sale:
fișier unic, titlu obligatoriu, dată opțională validă, cheie de idempotentă în
header și respingerea câmpurilor tenant/proprietar/grantee/taxonomie/metadata.
Ruta POST /api/education/portfolios/me/{recordID}/archive-documents este
înregistrată, cu contract OpenAPI și adaptor React validat la runtime.

Fluxul reverifică autorizarea și starea portofoliului înainte de persistare;
documentul, versiunea, jobul, grantul proprietarului și auditul sunt tranzacționale.
Nu se acordă `earchiva.manage` profesorului.

Verificare locală din 12 septembrie 2026:
- TestPortfolioOwnArchiveUploadIntegration: PASS, PostgreSQL și MinIO reale de
  test, inclusiv replay identic, conflict de conținut, acces al altui profesor,
  câmpuri nepermise, schimbarea stării și revocarea apartenenței în timpul scanării.
  Scannerul antivirus este simulat; testul nu dovedește scanarea antivirus reală.
- Build React și compilare TypeScript: PASS.
- TeacherPortfolioWorkspace: 3 teste PASS; PortfolioOwnUpload: 1 test PASS.
  Testul de componentă verifică cheia stabilă la retry și disponibilitatea
  documentului numai după confirmarea procesării. API-ul este simulat în acest test.
- Politicile UI PrimeReact/culori și contracte API: PASS.

### Proba browser de upload și atașare

La 12 septembrie 2026, scenariul independent
`frontend-react/e2e/system/teacher-portfolio-real-stack.spec.ts` a trecut:
`1 passed (5.3m)`, exit code 0.

- Directorul de test publică procedura prin OIDC și API; procedura nu este
  inserată direct prin SQL. SQL pregătește identitățile/încadrările fixture.
- Profesorul obișnuit se autentifică prin OIDC/OTP, fără `earchiva.manage`,
  creează portofoliul, încarcă PDF-ul prin FileUpload, așteaptă procesarea și
  atașează documentul prin formularul React.
- PostgreSQL și MinIO sunt reale, izolate și temporare. OCR Azure, antivirusul
  și verificatorul de semnături sunt emulatoare; acest test NU dovedește
  integrarea cu serviciile externe reale și NU testează o semnătură juridică.
- Asserțiile DB verifică documentul, subiectul creatorului, hash-ul sursei,
  grantul profesorului și relația cu portofoliul său.
- Finalizarea runnerului pe Windows a necesitat închiderea controlată a celor
  opt arbori de servere de test după terminarea workerului. Automatizarea
  închiderii locale fără intervenție rămâne de verificat; CI nu a fost rulat.

Proba a descoperit un defect real: generatorul transforma Go `any` în
`string`, respingând metadatele numerice ale uploadului. Generatorul, schema,
clientul și validatorii au fost corectați; fixture-ul răspunsului real este
acum verificat de testele adaptorului. După corecție: 29 teste API, 4 teste
workspace profesor și testul uploadului au trecut, ca și buildul,
verificarea OpenAPI (629 operațiuni) și politicile UI/contracte.

Rămân obligatorii trimitere → returnare → corectare → verificare, verificarea
celorlalte cerințe canonice și legislative și publicarea/verificarea live.
Nu s-a făcut push/deploy în acest checkpoint și nu se declară portofoliul complet.

### Remediere locală PRT-012

Workspace-ul profesorului afișează acum observațiile ultimei revizii, data,
rezultatul, documentele lipsă și scorul returnate de API. Interogarea este
separată de filtrarea/paginarea tabelului, folosește ruta proprie și ignoră
răspunsurile întârziate după schimbarea portofoliului. Testul React pentru
portofoliu returnat și buildul au trecut. Publicarea și proba E2E reală de
remediere nu au fost încă realizate.

[Metodologia-cadru din 30 aprilie 2026](https://legislatie.just.ro/Public/FormaPrintabila/00000G0OKWK5FMC46SU0J6HUUOD3SDUB)
distinge gestionarea individuală și instituțională, cere opis datat și ordine
cronologică și permite adaptarea listei de documente. Verificarea finală trebuie
să includă și regulile de depunere pentru personalul detașat, cu mai multe
încadrări, pensionar sau asociat (art. 10), fără a impune aceeași obligație tuturor.
Aceste cazuri nu sunt declarate acoperite prin testele executate mai sus.

### Checkpoint PRT-017 — 12 septembrie, 17:45 (local, nepublicat)

Interfața profesorului folosește acum `PortfolioDownloadPanel` și contractul
POST `/api/education/portfolios/me/{recordID}/export`, fără corp, cu răspuns ZIP.
Accesul la descărcare nu este condiționat de dreptul de modificare; ruta UI
existentă cere consultarea propriului portofoliu. Nu se acordă acces general
la eArhivă. Permisiunea alternativă `export_own` nu este declarată provisionată
independent până la verificarea migrațiilor RBAC.

Dovezi executate în acest checkpoint:

- 48 teste frontend PASS: API, componenta de descărcare și workspace profesor.
- Compilare TypeScript fără erori raportate.
- Regenerare și validare OpenAPI PASS: 631 operațiuni, inclusiv 400 Școală.
- Guard dedicat pentru export fără corp, media type `application/zip` și
  răspunsuri de eroare 401/403/404/413/422/500/503.

Exportul NU este declarat complet: review-ul backend și testele PostgreSQL,
concurență, integritate ZIP și E2E sunt încă în curs. Au fost identificate
riscuri privind consistența snapshotului complet și reproducerea exactă a
checksumului sidecar; corectarea lor este obligatorie înainte de publicare.
Nu s-a făcut push sau deploy în acest checkpoint.

### PRT-017 — proba E2E reală, 18:03

Rularea prin browser/OIDC/Go/PostgreSQL/MinIO a parcurs încărcarea a cinci
PDF-uri, procedura aplicată, trimiterea, returnarea, corectarea și validarea.
Refuzul corpului de export controlat de client a fost verificat (400, fără
manifest persistat). Descărcarea validă a eșuat cu 422
`education_portfolio_export_provenance_incomplete`; exportul rămâne neterminat.
În baza de date, toate cele cinci documente au versiuni active, arhive ready,
bucket/cheie/hash concordante și 552 bytes PDF. Verificarea inițială `IS NOT
NULL` a fost insuficientă: diagnosticul ulterior a arătat ID-uri de versiune
storage goale. Încărcarea generică eArhivă nu persista VersionId returnat de
MinIO. Exportul refuză corect aceste surse; se corectează încărcarea, fără
fallback la obiectul curent și fără inventarea versiunilor pentru datele vechi.

Prima încercare a identificat o regresie React de chei identice între panoul
procedurii și cel al descărcării. Cheile au fost separate pe tipul panoului.
Testul de regresie comută repetat portofoliile și cere un singur buton pentru
fiecare panou; cele 11 teste ale workspace-ului au trecut, iar a doua rulare
E2E a depășit etapa care eșuase. Nu există rezultat PASS pentru exportul E2E.

### Custodia portofoliului activ — checkpoint după aprobarea explicită

Utilizatorul a aprobat protecția portofoliilor active fără inventarea unei date
de încetare, precum și tranziția către retenție la încetarea activității.
Aceasta este o alegere tehnică de protecție, nu o afirmație că legea impune
Object Lock. Blocajele juridice reale nu trebuie eliberate de această tranziție.

Review-ul independent al patch-ului curent a identificat blocaje de publicare:

- Migrația `0162` eșuează pe PostgreSQL nou cu SQLSTATE `42830`: cheia externă
  compusă către portofoliu nu are o constrângere unică echivalentă în părinte.
  Toate cele patru teste de securitate a exportului se opresc la această migrație
  (rulare de 17,944 s). Rezultatul PASS anterior migrației nu validează patch-ul nou.
- Intenția de încărcare este rezervată înainte de atribuirea bucketului/cheii;
  identitatea versiunii stocate se pierde la rollback-ul tranzacției finale.
- Reluarea idempotentă, legătura verificabilă dintre intenție și versiunea arhivei,
  verificarea ETag/metadatelor și protecția tranzițiilor trebuie completate.
- Încetarea activității și blocajele juridice au încă modificări doar în DB;
  aplicarea și verificarea retenției în storage înainte de eliberarea custodiei
  nu sunt implementate cap-coadă.

Testele unitare ale pachetului eArhiva au trecut în snapshotul verificat de
reviewer, dar nu demonstrează aceste scenarii. E2E și publicarea rămân blocate
până la corectare și reverificare. Nu s-a făcut push sau deploy.

Verificare frontend după acest checkpoint:

- `npm test -- --reporter=dot`: 57 fișiere / 328 teste PASS, 96,00 s.
- După această rulare au fost adăugate patru teste pentru erorile 413/503/403
  și anularea descărcării la unmount; componenta de descărcare: 10 teste PASS,
  5,48 s. Nu este declarată o rerulare integrală de 332 teste.
- `npx tsc -b`: PASS înainte de cele patru teste suplimentare.
- `npm run quality:ui` și `npm run quality:contracts`: PASS.
- Proba E2E cere acum și egalitatea semantică dintre manifestul public și
  reprezentarea canonică (cu câmpul propriei sume gol), nu doar potrivirea
  checksumului unui fișier potențial fără legătură. Proba nu a fost încă rulată
  cu migrația de custodie corectată.

Aceste rezultate nu demonstrează încă exportul complet, tranzițiile WORM,
conformitatea întregului modul Școală sau funcționarea lor în mediul publicat.

### Proba integrată reușită — 12 septembrie, 19:45 (local)

Migrația `0162` corectată se aplică pe PostgreSQL nou. Cele patru regresii
de securitate a exportului au trecut din nou (20,318 s): corp nepermis,
proprietar diferit/acces între tenanți, grant revocat și citire sub blocaj
juridic fără modificarea acestuia.

`scripts/ci/run-local-school-system.ps1 -StorageEndpoint http://127.0.0.1:59001`
a terminat cu exit 0: **1 E2E PASS, 2,3 minute**. Proba a folosit browserul,
OIDC-ul aplicației, backendul Go, PostgreSQL și MinIO reale, exclusiv cu date
sintetice. OCR, antivirusul și verificatorul de semnături sunt emulatoare;
serviciile externe din producție nu sunt validate prin această rulare.

Au fost verificate încărcarea celor cinci PDF-uri și versiunea exactă din
MinIO cu hold activ și fără termen inventat pentru portofoliul activ,
procedura aplicată, trimiterea, returnarea directorului, corectarea, istoricul,
retrimiterea și validarea. Modificarea ulterioară nepermisă este refuzată.
Descărcarea prin UI returnează ZIP cu portofoliu, opis și cele cinci PDF-uri;
octeții, dimensiunile, SHA-256, versiunea manifestului, reprezentarea canonică
și persistența manifestului sunt verificate în test.

Buildul frontend de producție și TypeScript au trecut; buildul avertizează
în continuare asupra chunkurilor mari. Acest rezultat nu reprezintă deploy.

Rămân deschise: persistența post-upload la anularea cererii, protecția tuturor
tranzițiilor intenției și legătura cu versiunea arhivei, recuperarea idempotentă
a întreruperilor, retenția fizică înainte de eliberarea custodiei, sincronizarea
blocajelor juridice și legarea criptografică a fișierelor generate de manifest.
Modulul complet și publicarea nu sunt declarate finalizate.

### Reverificare pe bază nouă — 19:51

După corecturile suplimentare pentru contextul post-upload, imutabilitatea
intenției și asocierea verificată dintre intenție și versiunea arhivei,
același E2E a trecut din nou: **1 PASS, 2,3 minute**, exit 0. Această rulare
a folosit baza sintetică nouă `egueducation_system_e2e_custody_v2`, creată în
PostgreSQL-ul izolat, cu migrațiile curente aplicate de la zero. Baza de date
a aplicației publicate nu a fost modificată.

Runnerul acceptă acum opțional `-DatabaseName`, limitat prin validare la
nume `egueducation_system_e2e...`, păstrând conexiunile locale de test și
rolul aplicației separat de migrator. Nu creează și nu șterge automat baze.
Review-ul negativ pentru noile constrângeri DB este încă în curs; succesul
fluxului normal nu închide recuperarea la întreruperi sau ciclul de retenție.

### Verificări negative finale ale acestui checkpoint — 19:55

Review-ul a reprodus un defect SQL: comparația `<>` permitea trecerea unei
dimensiuni verificate `NULL`. A fost înlocuită cu `IS DISTINCT FROM`.
După corecție, pe baze PostgreSQL noi, cu migrațiile curente:

- Șase scenarii pentru intențiile de custodie: PASS (5,990 s). Sunt refuzate
  modificarea provenienței odată cu tranziția, schimbarea identității verificate
  la commit, dimensiunea NULL, adoptarea neconcordantă, eliminarea legăturii
  și versiunile sub custodie fără intenție. Adoptarea exactă reușește;
  izolarea tenantului este verificată prin invizibilitate și zero rânduri modificate.
- Patru teste de securitate a exportului: PASS (17,381 s), rerulate după
  ultima corecție. Comanda combinată a încheiat cu exit 0.

Cele două E2E reușite preced ultima corecție SQL pentru respingerea NULL;
aceasta este acoperită de noua regresie pe PostgreSQL, nu de un al treilea E2E.
Recuperarea/reluarea încărcărilor întrerupte și ciclul complet de retenție
rămân neînchise. Nu s-a făcut push/deploy și nu se declară modulul terminat.

### Upload reluat și fișiere generate verificate — 20:18

Frontendul păstrează cheia de reîncercare, transmite cauza erorii API și nu
aplică răspunsuri vechi după schimbarea portofoliului. După refuzul explicit
al dreptului de încărcare, formularul nu mai lansează reîncercări. Suita a
trecut cu 57 fișiere/336 teste (98,51 s); cele 41 de teste upload/API au fost
rerulate cu succes după întărirea UI. TypeScript, buildul și politicile
PrimeReact/contracte au trecut. Avertizarea de chunkuri mari rămâne.

Manifestul ZIP include acum opțional `generated_files` cu cale, SHA-256 și
număr de octeți pentru `portfolio.json`, `opis.json`, `opis.txt`. Backendul
verifică aceiași octeți la staging și revalidează snapshotul final. Testele
unitare education au trecut (3,457 s), nouă teste PostgreSQL de export au
trecut (38,203 s), iar `go vet` a trecut. OpenAPI și clientul React au fost
regenerate și validate. Manifestele istorice fără câmp rămân compatibile.

Instanțele temporare PostgreSQL/MinIO au expirat la limita de patru ore.
Au fost create `postgres-r2` și `minio-r2` în același namespace izolat, cu
stocare efemeră nouă și acces local prin aceleași porturi. Datele publicate
nu au fost atinse.

E2E actualizat: **1 PASS, 2,4 minute**, exit 0. Browserul pierde intenționat
răspunsul primului upload numai după ce backendul real l-a salvat; reîncercarea
returnează 200, aceeași cheie și același ID. MinIO are exact o versiune și
niciun delete marker pentru obiect. Fluxul profesor/director și ZIP-ul final
trec, inclusiv descriptorii și octeții celor trei fișiere generate.

Acest test demonstrează reluarea după commit, nu încă scenariul storage
reușit/DB rollback sau recuperarea unui PUT cu răspuns necunoscut. Acestea
rămân în implementare/testare separată, împreună cu retenția fizică la încetare.

### Operații persistente și revizie independentă — după confirmarea 0163

Confirmarea explicită a utilizatorului acoperă migrația 0163, operațiile
persistente și workerul de aplicare/verificare a retenției, inclusiv riscul
prelungirii ireversibile WORM. Implementarea rămâne în checkoutul de verificare;
nu s-a făcut push sau deploy.

- Contractele HTTP pentru încetare/blocaj juridic returnează 202 cu ID și stare
  persistentă; GET citește starea, iar POST retry cere motiv și drept de
  administrare. OpenAPI: 633 operații concrete, 402 în catalogul Școală,
  validare PASS; clientul și validatorii React au fost regenerați.
- UI PrimeReact afișează progresul, nu confundă acceptarea cu finalizarea,
  oprește pollingul la eroare și permite reîncărcarea explicită. Retry este
  disponibil pentru administrator, numai cu motiv. Cele 82 de teste focalizate
  au trecut; suita completă: **59 fișiere, 345 teste PASS, 122,62 s**.
  TypeScript, build, quality:ui și quality:contracts PASS. Avertizarea privind
  dimensiunea chunkurilor rămâne.
- `go test ./internal/education ./internal/earchiva ./cmd/server`: PASS
  (3,573 s / 6,811 s / 3,248 s). Agentul backend raportează și teste reale PG
  pentru starea persistentă/retry (5,80 s), inclusiv respingerea retry-ului
  duplicat cu 409. Testele adapterului verifică faptul că eroarea de retenție
  nu este urmată de eliberarea holdului.
- Revizia independentă a identificat atașarea unui document deja eliberat la
  un portofoliu activ. Triggerul din 0163 blochează această atașare și se
  serializează cu workerul pe versiunea exactă. Regresia PG pentru partajare,
  OR/MAX, eliberare și respingere după eliberare: PASS (5,22 s, agent backend).
  Reviewerul a confirmat închiderea problemei. Intercalarea concurentă forțată
  rămâne datorie de acoperire, nu un defect demonstrat.
- Recuperarea uploadului are teste zero/una/mai multe versiuni și verificare
  rollback fără document sau audit parțial. Testul real PG/MinIO trece
  (7,802 s); suita PG de custodie trece (6,185 s). Testul de tranziție verifică
  și cazul în care actualizarea condițională pierde și reîncarcă identitatea
  exactă, nu reprezintă un test complet cu cereri HTTP simultane.

Limite încă deschise: descoperirea istoricului operațiilor după reîncărcarea
paginii; reconcilierea administrativă a unui intent deja stocat când
autorizarea este revocată înainte de commit. Obiectul rămâne protejat, dar nu
există încă fluxul administrativ complet de rezolvare. Nu se șterge și nu se
ridică holdul pentru a ascunde această situație.

E2E extins pentru blocaj juridic → încetare → eliberare este în curs de
verificare pe baza nouă `egueducation_system_e2e_lifecycle_20260912`. Prima
încercare a fost oprită de portul 4173 ocupat de un server demo. Runnerul permite
acum un port primar alternativ; serverul demo nu a fost oprit. Acest checkpoint
nu declară noul E2E trecut.

Prima probă extinsă pe port alternativ a trecut fluxul normal până la export,
dar a eșuat înainte de prima operație de retenție: ruta publicată pentru
director folosește `PortfolioReviewWorkspace`, nu lista generică din
`EducationWorkspace`. Operațiile au fost conectate și pe ruta reală, cu
permisiunea explicită `education.portfolios.school.manage`. Regresia de acces
și testele dialog/stare: **14 PASS, 6,57 s**; TypeScript, build și politicile
UI/contracte au trecut după corecție. Proba extinsă este rerulată; suita de
345 de teste de mai sus precede această ultimă integrare de rută.

A doua probă pe port alternativ a ajuns la dialogul de blocaj juridic, dar
locatorul exact al dialogului nu corespundea numelui accesibil compus din
întregul header. Trace-ul confirmă dialogul deschis și formularul prezent,
fără operație persistentă încă înregistrată. `Dialog.Popup` este acum legat
explicit prin `aria-labelledby` de `Dialog.Title`, nu de headerul care include
butonul Închide. Testele dialogului au trecut după corecție. Probele eșuate
sunt păstrate ca limite ale verificării, nu numărate ca E2E reușite.

Proba următoare a executat blocajul juridic și l-a finalizat, apoi a expus
eroarea reală MinIO `MissingContentMD5` pentru `PutObjectRetention`. Operația
de încetare a ajuns în `dead_letter`, nu în `completed`. Verificarea directă
HEAD a tuturor celor cinci versiuni exacte a confirmat **hold ON**, fără
retenție pretinsă în storage. Protecția nu a fost eliberată la eșec.
Adapterul a fost corectat să calculeze Content-MD5 pentru XML-ul exact atât
la retenție, cât și la hold; cele trei teste de protocol verifică headerul
față de corpul real și au trecut (5,18 s, agent backend).

Suita frontend completă de după integrarea rutei și corecția numelui accesibil:
**59 fișiere, 346 teste PASS, 101,83 s**. Ulterior, cazul revocării permisiunii
cu dialogul deschis a fost corectat și verificat prin cele 14 teste focalizate
(9,23 s). Rerularea E2E cu checksumul corect este în curs.

Rerularea cu checksum a demonstrat aplicarea retenției COMPLIANCE pe cele
cinci versiuni exacte și păstrarea blocajului juridic după încetare. A eșuat
apoi la acțiunea de ridicare a blocajului: UI deducea starea juridică din
rezumatul listei, care nu o conținea. Pagina de verificare încarcă acum și
detaliul tipizat al portofoliului selectat; acțiunile de retenție devin
disponibile numai după încărcarea lui. Regresia pentru detaliu cu hold activ
și încetare înregistrată a trecut împreună cu celelalte teste ale paginii
(8 PASS, 8,91 s, înainte de adăugarea gardului de încărcare). Testul complet
este rerulat pentru a verifica și ultimul pas de eliberare.

Clarificare după trace-ul rerulării: încărcarea detaliului UI nu a fost
suficientă. GET-ul generic al portofoliului întorcea efectiv
`legal_hold_active:false`, `retention_period_days:0` și omitea încetarea,
în timp ce GET-ul operației persistente întorcea valorile corecte din DB.
Prin urmare, cauza include o proiecție/scannare backend incompletă, nu doar
rezumatul UI. Corecția backend și o regresie PostgreSQL pentru paritatea
listei/detaliului cu datele de ciclu de viață sunt în lucru. Ultima probă
nu este declarată reușită.

Proiecția generică backend a fost corectată: `PortfolioRecords` și
`PortfolioRecordDetail` folosesc acum aceeași listă de coloane și același
scanner canonic ca operațiile/portofoliul propriu. Regresia pe PostgreSQL
compară owner IDs, încetare, retenție, blocaj juridic/motiv și procedura
aplicată pentru listă și detaliu cu datele persistate: PASS (6,16 s, agent
backend). Testele/vet education și server au trecut; validarea OpenAPI
rămâne PASS. Cele 16 teste UI focalizate au trecut (9,55 s), inclusiv
indisponibilitatea detaliului. Proba completă rulează pe această corecție.

### Rezultat final al probei curente

**E2E: 1 passed (7,7 minute), exit 0.** Comandă:

```powershell
& scripts/ci/run-local-school-system.ps1 -StorageEndpoint 'http://127.0.0.1:59001' -DatabaseName 'egueducation_system_e2e_lifecycle_20260912' -FrontendPort 4183
```

Proba verifică browser → OIDC → API → PostgreSQL → MinIO: cinci uploaduri,
reîncercare după răspuns pierdut, corectare/validare profesor-director, export
ZIP și hashuri, blocaj juridic, încetare, retenție COMPLIANCE și ridicarea
blocajului. După încetare, cele cinci versiuni exacte au hold ON cât timp
blocajul juridic este activ. După ridicarea lui, au hold OFF și păstrează
retenția COMPLIANCE. Toate cele trei operații au 5/5 tranziții finalizate.

**Limită de automatizare:** închiderea proceselor webServer pe Windows a
rămas blocată după terminarea workerului de test. Au fost identificate și
închise exclusiv procesele backend/Vite/emulatoare din arborele acestei probe,
după care runnerul a returnat PASS/exit 0. Nu este încă o probă complet
nesupravegheată a teardownului Windows. Serverul demo, PostgreSQL și MinIO
nu au fost oprite, iar niciun obiect de arhivă nu a fost șters.

Suita frontend finală: **59 fișiere, 348 teste PASS, 121,04 s**. Nu există
push/deploy în acest checkpoint. OCR/AV/verificatorul de semnături rămân
emulate; rezultatul demonstrează fluxul cu storage și DB reale, nu serviciile
externe publicate. Cerințele și limitele rămase din catalog sunt păstrate.

## Istoric persistent al operațiilor — checkpoint ulterior

GET `portfolios/records/{recordID}/lifecycle-operations` are contract generat
pentru paginare, sortare și filtre de stare/tip/zi UTC. Interfața React este
accesibilă din portofoliul profesorului și din pagina de verificare; profesorul
nu primește acțiunea de reluare. Starea selectată actualizează lista printr-o
nouă citire de server, iar erorile de citire elimină rândurile/detaliile vechi.

Dovezi executate în acest checkpoint:

- Suita frontend completă: 60 fișiere / 355 teste PASS (129,85 s).
- După adăugarea testului de integrare în pagina de verificare și curățarea
  detaliului la refuzul citirii: 3 fișiere / 19 teste focalizate PASS (13,71 s).
- TypeScript, build React și politicile UI/contracte API: PASS.
- OpenAPI: 634 operații concrete, dintre care 403 în catalogul Școală; validare PASS.
- Testele PostgreSQL `TestPortfolioLifecycleOperations*`: PASS (13,557 s),
  cu acces propriu/instituțional, refuz pentru alt profesor/tenant, 404 pentru
  resursă absentă cu rol privilegiat, filtre UTC/stare/tip, sortare stabilă,
  paginare, numărătoarea tranzițiilor și parametri invalizi/overflow/spații.
  Pagina este limitată explicit la 1.000.000 inclusiv în OpenAPI; dimensiunea
  paginii păstrează politica comună (implicit 25, plafon 100).
- Revizie backend independentă: aceeași suită PASS (10,731 s), compilare
  education/server PASS, fără constatări blocante de autorizare/injecție.
  Limită: proba cross-tenant refuză actorul înainte de autorizarea în tenantul
  alternativ; izolarea după autorizare acolo necesită o probă distinctă.
  Totalul și pagina sunt citite READ COMMITTED și pot diferi tranzitoriu când
  workerul schimbă stările între cele două interogări; nu se pretinde snapshot atomic.

### Completarea probei de izolare și diagnosticul runnerului Windows

Proba suplimentară acordă actorului drept real `education.portfolios.school.read`
în tenantul/instituția B, apoi cere portofoliul din A: răspuns 404 scoped.
Cazul anterior fără drept în B rămâne 403. Suita PG: PASS (11,660 s).
Aceasta închide limita de testare cross-tenant menționată în checkpointul anterior.

Încercarea browserului pentru istoric a eșuat la pornire, înainte de aserțiuni:
portul 4174 era ocupat de un server demo separat, păstrat intact. Runnerul și
configurația acceptă acum porturi distincte pentru cele trei frontenduri;
proba profesorului utilizează aceleași origini configurabile.

Diagnosticul teardownului: sandboxul Windows refuză `taskkill`, iar Playwright
1.62.1 nu verifică statusul comenzii și așteaptă închiderea procesului rămas viu.
Au fost închise numai rădăcinile copil identificate ale probei eșuate. Nu s-au
oprit procese prin căutarea unui port. Remedierea mediului de execuție este
rularea autorizată cu permisiuni pentru oprirea propriilor procese; nu o
rescriere a serviciilor aplicației. Confirmarea teardownului automat necesită
rezultatul noii probe, nu doar acest diagnostic.

### Proba browser finală pentru istoric și teardown — PASS

Comanda executată cu permisiuni pentru propriile procese Windows:

```powershell
./scripts/ci/run-local-school-system.ps1 -StorageEndpoint http://127.0.0.1:59001 -DatabaseName egueducation_system_e2e_lifecycle_20260912 -FrontendPort 4183 -TeacherFrontendPort 4184 -AlternateFrontendPort 4185
```

Rezultat: **1 passed (2.8m), exit 0**, fără intervenție de cleanup. Încercarea
precedentă a detectat nume duplicate în test, corectate și verificate cu
`playwright test --list`; runnerul include acum acest preflight înaintea
serverelor. Configurația și runnerul refuză porturi duplicate înainte de start.

Proba finală include fluxul complet anterior (cinci uploaduri, retrimitere,
verificare director, ZIP/hashuri, trei operații WORM cu cinci versiuni fiecare),
apoi reload, redeschiderea istoricului, trei operații persistente finalizate,
filtru de zi UTC transmis serverului și starea 5/5 deschisă din dialog.
Portofoliul fixture 2037–2038 este `validated`, cu cinci documente și trei operații.

După exit 0, controlul read-only nu găsește listeneri pe 4183–4185,
8080–8082 sau 9090–9091. Serverul demo separat de pe 4174 este încă activ.
Această probă închide limita de teardown manual din checkpointurile precedente,
în mediul autorizat. Nu demonstrează publicarea; OCR/AV/semnăturile externe sunt
în continuare emulate. Warningul licenței PrimeUI neconfigurate în harnessul
izolat rămâne vizibil. Nu s-au șters obiecte sau scurtat retenții.

### Următorul gap confirmat prin cod: intenții stocate fără finalizare

Revizia read-only confirmă că după marcarea versiunii exacte ca `stored`,
reverificarea autorității sau persistarea grantului poate refuza finalizarea.
Rollbackul păstrează versiunea S3 sub hold, dar fără document/versiune/job/grant
în graful arhivei. Retry-ul joburilor de ingestie și cel al operațiilor de
retenție nu se aplică deoarece acele entități nu există încă. Comportamentul
actual este fail-closed, însă lipsește recuperarea administrativă.

Implementarea următoare trebuie să ofere inventar și detaliu scoped pe
tenant/instituție, urmate de reconciliere motivată și auditată. Orice reluare
a finalizării trebuie să verifice din nou proprietarul, statutul portofoliului
și drepturile actuale; administrarea recuperării nu conferă implicit acces
profesorului și nu ocolește blocajele. Verificarea utilizează aceeași versiune
exactă, hash, metadate și hold, fără PUT nou, ștergere sau hold OFF.
Trebuie acoperite revocarea după PUT, schimbarea statutului, izolarea/RBAC,
idempotenta după crash/concurență și refuzul oricărei neconcordanțe de proveniență.
Acest flux nu este implementat sau declarat verificat în checkpointul curent.

### Politica pentru încetări cu termen deja expirat — rezultat de audit

Data istorică reală a încetării trebuie acceptată și păstrată; mutarea ei în
viitor ar falsifica ancora termenului legal. Expirarea celor trei ani produce
eligibilitate pentru revizia dispoziției, nu autoritate automată de ștergere,
transfer sau ridicare a holdului. Fluxul actual eșuează fail-closed deoarece
storage refuză o retenție COMPLIANCE în trecut, însă o clasifică generic și o
reîncearcă până la dead-letter.

Corecția necesară: înaintea oricărei mutații storage, termenul exclusiv deja
expirat trebuie să producă starea permanentă și sigură
`portfolio_retention_expired_review_required`, fără PutRetention sau hold OFF.
Revizia ulterioară trebuie să fie append-only, scoped, să conțină dovada cererii
scrise și să reverifice referințele partajate, blocajele juridice și versiunea
exactă. Transferul poate autoriza schimbarea custodiei numai după recepția
finalizată în fluxul PRT-015. Acest flux de revizie nu este încă implementat.

Corecție critică a auditului: comportamentul existent nu este fail-closed în
toate cazurile. Dacă storage are deja o retenție COMPLIANCE expirată, dar egală
sau ulterioară termenului istoric solicitat, adaptorul poate considera retenția
suficientă și poate continua cu hold OFF. Bariera `requiredRetention > now`
trebuie verificată înaintea citirii sau mutării holdului, chiar dacă retenția
observată există. Regresia obligatorie pornește cu COMPLIANCE expirată și hold ON
și demonstrează zero apeluri de ridicare a holdului.

O a doua neconcordanță: migrațiile 0161/0163 folosesc maximul dintre termenul
canonic și vechiul `retention_until`, deși 0101 declară valorile istorice
pre-încetare ca informație de audit, nu autoritate de retenție. Aceasta poate
promova o dată arbitrară și extinde ireversibil WORM. Termenul de bază trebuie
să fie exact încetare + 3 ani calendaristici; extensiile ulterioare pot proveni
numai dintr-o autoritate separată, aprobată și imuabilă. Corecția și regresia
pentru o dată veche neautorizată nu sunt încă implementate.

Istoricul este verificat în browserul cu backend real prin proba PASS de 2,8
minute documentată mai sus, dar nu este încă publicat. Nu au fost eliminate
blocaje juridice sau obiecte pentru a facilita testele.

### Închiderea fluxului pentru retenția expirată — implementat și verificat

Constatările fail-open de mai sus au fost remediate în migrația 0165 și în
workerul persistent. Fluxul normal menține un hold distinct de dispoziție pe
toată durata retenției COMPLIANCE. La expirare, scannerul promovează versiunea
o singură dată în `portfolio_retention_expired_review_required`, fără a modifica
storage. Profesorul trimite dovezi prin contractul propriu, iar o altă persoană
poate decide numai dacă are simultan `education.portfolios.custody.manage` și
`earchiva.manage`; separarea actorilor și permisiunile curente sunt verificate
atât HTTP, cât și în trigger-ele PostgreSQL.

Workerul reverifică sub advisory lock toate referințele și blocajele juridice,
bucketul serverului, VersionId, ETag, SHA-256, dimensiunea și MIME. Hold OFF este
urmat de verificare și de un receipt append-only. Eșecurile tranzitorii nu mai
ajung în dead-letter și sunt reluate până când receipt-ul și proiecțiile DB
converg; o versiune deja OFF după întrerupere este tratată idempotent. Un blocaj
nou închide aprobarea curentă ca `blocked`, lasă expiry event neconsumat și cere
o solicitare și o decizie independentă nouă după remediere.

Probe executate pe PostgreSQL și MinIO Object Lock reale:

- `TestPortfolioRetentionDisposition*`: PASS, 3 teste de integrare, 16,555 s;
  include eliberarea workerului și toate cele trei flaguri DB false, exact un
  receipt, RBAC DB, lipsa GUC, SHA/size/MIME/version greșite fără mutație,
  restart cu hold deja OFF și fresh review după un legal hold temporar;
- lifecycle pre-expiry agregat: PASS, 6,738 s, cu hold-ul de dispoziție activ;
- OpenAPI: 640 operații concrete, 406 operații Școală, validare PASS;
- React: 62 fișiere / 365 teste PASS; typecheck, policy UI PrimeReact/theme și
  contractele API generate PASS.

UI React oferă acum lista server-side paginată/sortată/filtrată, formularul
profesorului pentru dovezi și dialogul deciziei independente pentru utilizatorii
cu ambele drepturi. Tabelul și dialogul deciziei afișează explicit identitatea
solicitantului, justificarea și referința documentului înainte ca aprobatorul să
poată confirma operația ireversibilă. Coordonatele storage nu sunt expuse.

Proba browser finală după toate corecțiile a trecut: **1 test PASS (12,4 minute)** prin OIDC, React,
API Go, PostgreSQL și MinIO Object Lock reale. Ea verifică solicitarea
profesorului, vizibilitatea solicitantului și a ambelor dovezi pentru aprobator,
decizia independentă, închiderea operației, receipt-ul unic, cele trei proiecții
DB false și hold-ul exact al versiunii WORM în starea OFF. Regresia separată de
recuperare verifică și cazul întreruperii după hold OFF: dacă între timp apare
un blocaj nou, workerul restaurează și verifică hold ON înainte de a închide
aprobarea ca blocată. Publicarea rămâne singura poartă deschisă a acestui flux.
