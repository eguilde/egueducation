# Catalog canonic — managementul operațional al școlii

Data reviziei: 2026-09-11
Statut: cerințe aprobabile și audit al codului existent; modulele marcate `lipsă` sau `planificat` nu sunt declarate implementate

## 1. Scop și regulă de produs

eGuEducation trebuie să deservească, din același cod și aceeași arhitectură multi-tenant, atât unități publice, cât și unități private sau confesionale autorizate/acreditate. Nu se creează fork-uri public/privat și frontendul nu decide regimul juridic prin condiții hardcodate.

Diferențele sunt evaluate de backend printr-un profil instituțional și policy packs versionate:

`nucleu comun RO → formă juridică → statut autorizare/acreditare → sursă de finanțare/program → configurație instituțională permisă`

Forma juridică nu este suficientă singură. De exemplu, o școală privată poate primi finanțare publică ori poate participa la un program public, iar o operațiune poate dobândi obligații suplimentare fără ca întreaga instituție să fie tratată ca școală publică.

Statusurile din acest catalog sunt:

- `existent`: comportament identificat în cod executabil și contracte;
- `parțial`: fundație reutilizabilă, dar contractul vertical nu este complet;
- `lipsă`: bounded context-ul sau regula nu există în cod;
- `planificat`: cerință țintă care trebuie implementată și demonstrată.

## 2. Surse oficiale și limitele conformității

| Domeniu | Surse principale |
| --- | --- |
| Învățământ public/privat | [Legea învățământului preuniversitar nr. 198/2023, forma actualizată](https://legislatie.just.ro/Public/DetaliiDocument/309185) |
| Organizare și funcționare | [ROFUIP aprobat prin Ordinul nr. 5.726/2024](https://legislatie.just.ro/Public/DetaliiDocument/289484), modificat prin [Ordinul nr. 6.226/2025](https://legislatie.just.ro/Public/DetaliiDocument/302026) și [Ordinul nr. 4.261/2026](https://legislatie.just.ro/Public/DetaliiDocumentAfis/312069) |
| Autorizare, acreditare și calitate | [HG nr. 993/2020](https://legislatie.just.ro/Public/DetaliiDocument/234510), [HG nr. 994/2020](https://legislatie.just.ro/Public/DetaliiDocumentAfis/255166) și [ARACIP](https://aracip.eu/) |
| Finanțare particular/confesional | [HG nr. 69/2024](https://legislatie.just.ro/Public/DetaliiDocument/295485), cu modificările ulterioare, inclusiv [HG nr. 381/2026](https://legislatie.just.ro/Public/DetaliiDocument/310287) și [OUG nr. 28/2026](https://legislatie.just.ro/Public/DetaliiDocument/311756) |
| Finanțe publice | [Legea nr. 500/2002](https://legislatie.just.ro/Public/DetaliiDocument/37954), [Legea nr. 273/2006](https://legislatie.just.ro/Public/DetaliiDocument/293380) |
| Achiziții publice | [Legea nr. 98/2016, forma actualizată](https://legislatie.just.ro/Public/DetaliiDocument/257213), în special art. 4 și 6, [HG nr. 395/2016](https://legislatie.just.ro/Public/DetaliiDocument/179009), [OG nr. 119/1999](https://legislatie.just.ro/Public/DetaliiDocument/286006) |
| Contabilitate | [Legea contabilității nr. 82/1991](https://legislatie.just.ro/Public/DetaliiDocument/55046), [OMFP nr. 1.917/2005](https://legislatie.just.ro/Public/DetaliiDocument/211573), [OMFP nr. 1.802/2014](https://legislatie.just.ro/Public/DetaliiDocumentAfis/250771) |
| HR și salarizare | [Codul muncii — Legea nr. 53/2003](https://legislatie.just.ro/Public/DetaliiDocument/128647), [Legea-cadru nr. 153/2017](https://legislatie.just.ro/Public/DetaliiDocument/190446) |
| Programe alimentare | [HG nr. 652/2023](https://legislatie.just.ro/Public/DetaliiDocumentAfis/273596), [Ordinul nr. 346/2023](https://legislatie.just.ro/Public/DetaliiDocumentAfis/274249), [Ordinul nr. 1.718/2023](https://legislatie.just.ro/Public/DetaliiDocument/303074), [HG nr. 24/2024](https://legislatie.just.ro/Public/FormaPrintabila/00000G0K2BJJ4RB6AK621QJ6ZTQ5PTS6) |
| SSM, PSI și pază | [Legea nr. 319/2006](https://legislatie.just.ro/Public/DetaliiDocument/73772), [Legea nr. 307/2006](https://legislatie.just.ro/Public/DetaliiDocument/307734), [Legea nr. 333/2003](https://legislatie.just.ro/Public/DetaliiDocument/172774), [HG nr. 301/2012](https://legislatie.just.ro/Public/DetaliiDocument/174736) |

Pragurile, formularele, planurile de conturi, programele și regulile anuale nu se hardcodează. Ele sunt versionate, au interval de valabilitate și sursă oficială. Declararea conformității unui policy pack necesită validarea unui expert juridic, contabil și, după caz, SSM/PSI; testele software demonstrează implementarea regulii aprobate, nu emit o opinie juridică.

## 3. Auditul funcționalităților existente

Auditul a fost realizat asupra codului Go/PostgreSQL/React și a contractului OpenAPI, nu pe baza afirmațiilor din documentele istorice.

| Capabilitate existentă | Reutilizare pentru public și privat | Verdict |
| --- | --- | --- |
| Tenant derivat din host, instituție activă și membership | aceeași izolare pentru orice formă juridică | existent |
| PostgreSQL RLS/FORCE RLS și teste cross-tenant | se extinde obligatoriu la fiecare tabel nou | existent ca fundație |
| RBAC tenant/instituție și sesiune OIDC | se extinde cu permisiuni operaționale și segregarea atribuțiilor | parțial |
| Registratură, Flux documente, eArhivă și audit | suport comun pentru documente, aprobări, păstrare și dovezi | existent ca fundație |
| Modul Education: guvernanță, personal, dosar, evaluări, portofolii | identitatea `education_personnel` se păstrează; se completează HR operațional | parțial |
| OpenAPI 3.1.1 și client React generat | contract unic DB–API–UI pentru noile verticale | existent ca fundație |
| Profil juridic și de reglementare al instituției | scrierea v2, sursele structurate și administrarea PrimeReact există; citirea și evaluarea live folosesc încă proiecția v1 | parțial; necesită cutover expand–migrate–contract |
| Policy packs versionate și endpoint de capabilități | fundația profil/packs/assignments/evaluations și resolverul comun există; extinderea catalogului de capabilități rămâne incrementală | implementat parțial |
| Contracte, utilități, achiziții, catering, patrimoniu, logistică, SSM/PSI | furnizorii, contractele operaționale, obligațiile și lifecycle-ul au DB/Go/OpenAPI/React; restul verticalelor nu sunt complete | parțial pentru contracte; lipsă pentru restul |
| HR complet, economic și contabil | dosarul personal existent nu acoperă aceste domenii | lipsă |

Concluzie: arhitectura de bază se păstrează, însă eGuEducation nu poate fi prezentat astăzi ca suită completă de management public/privat. Extinderea trebuie realizată vertical, contract-first, fără tabele sau formulare generice care simulează procese inexistente.

### 3.1 Limită structurală curentă și adaptare obligatorie

Modelul curent este multi-tenant, dar fiecare înregistrare `app_tenants` are exact un singur `institution_id` (`UNIQUE`). În această etapă produsul suportă corect mai multe școli izolate, câte o instituție principală pe tenant/subdomeniu. Nu se declară încă suport pentru un operator privat, fundație sau autoritate care administrează mai multe școli în același tenant.

Adaptarea public/privat nu necesită două produse și nu schimbă această izolare. Ea introduce profilul juridic independent de tenant și policy overlays effective-dated. Suportul viitor „mai multe instituții într-un tenant” necesită un agregat `app_institutions`, o mapare tenant–instituție și revizuirea membershipurilor, sesiunilor OIDC și a tuturor politicilor RLS; acesta este un prerechizit explicit înaintea comercializării unui tenant-grup.

## 4. Profil instituțional și politici public/privat

### 4.1 Date canonice

`school_institution_profiles` trebuie să conțină cel puțin:

- `tenant_code`, `institution_id` și interval de valabilitate;
- `school_legal_form`: `public` sau `private`; caracterul confesional este un overlay separat, deoarece nu înlocuiește forma juridică privată;
- `regulatory_profile`: de exemplu `ro.public.preuniversity`, `ro.private.preuniversity`, `ro.private.confessional`;
- rezumatul instituțional al autorizării/acreditării; adevărul operațional se păstrează separat pe nivel/program/specializare/locație;
- personalitate juridică, CUI, fondator, cult recunoscut unde este cazul și entitate finanțatoare/ordonator, legate de `app_parties`, nu doar denumiri libere;
- rezultatul determinării calității de autoritate contractantă, cu temei, dovezi, aprobator și valabilitate, fără a o deduce automat din `school_legal_form` sau din existența unei finanțări publice;
- `accounting_profile`, `procurement_profile`, `payroll_profile`, profil TVA și obligația Trezorerie;
- stare `unclassified`, `draft`, `approved`, `active`, `superseded`;
- numai proiecții/sumarizări derivate pentru finanțare, taxe și autorizare; acestea nu pot înlocui agregatele effective-dated de la secțiunea 4.4.

La migrare, instituțiile existente devin `unclassified`; nu se face backfill implicit `public`. Citirile rămân disponibile cu avertisment, iar scrierile reglementate sunt fail-closed până la aprobarea profilului.

### 4.2 Policy packs

Sunt necesare:

- `school_policy_pack_versions`: jurisdicție, profil, versiune, `effective_from/to`, reguli tipate validate JSON Schema, surse, checksum și aprobare;
- `school_policy_assignments`: asocierea tenant/instituție–versiune, fără intervale active suprapuse;
- `school_policy_overrides`: numai valori declarate configurabile, fără dezactivarea obligațiilor imperative;
- `school_policy_evaluations`: snapshot imuabil al regulilor folosite de o decizie sau tranzacție;
- `/api/institution/capabilities`: pașii, acțiunile, documentele și permisiunile aplicabile contextului curent.

Frontendul redă capabilitățile publicate de server. Nu are ramuri `if schoolIsPublic` și nu permite payloadului să aleagă tenantul, instituția ori profilul juridic.

### 4.3 Starea implementării fundației

Verticala inițială folosește `school_institution_profiles`, `school_policy_pack_versions`, `school_policy_assignments`, `school_policy_overrides` și `school_policy_evaluations`, cu scope compozit tenant–instituție, RLS forțat, versionare și evaluări imuabile. Fiecare assignment este legat prin FK de versiunea exactă a profilului, astfel încât un draft sau profil viitor nu dezactivează profilul efectiv și nu rescrie istoricul. Endpointurile canonice sunt `GET/PUT /api/institution/regulatory-profile` și `GET /api/institution/capabilities`; clasificarea este administrativă, iar capabilitățile sunt accesibile utilizatorului autentificat numai pentru propriul context. Policy packs de bază sunt instalate prin migrare controlată, nu editate arbitrar de administratorul tenantului.

Prima integrare verticală este `education.publication.manage`: POST/PATCH/DELETE pentru publicațiile de conformitate sunt evaluate server-side, create păstrează `policy_evaluation_id`, iar UI ascunde mutațiile când intersecția modul–RBAC–policy nu le permite. Contractele operaționale Stage2 folosesc apoi aceeași fundație pentru furnizori, contracte, obligații și lifecycle. OpenAPI publică `x-required-policy-capability`. Aceste dovezi nu fac singure modulele OPS-CON…OPS-FIN complete; fiecare operațiune reglementată următoare trebuie conectată prin același model, iar simpla afișare a unui guard în React nu este control de securitate.

### 4.4 Dimensiuni juridice și agregate obligatorii

Aplicabilitatea nu este o alegere binară public/privat. Resolverul trebuie să evalueze, effective-dated, cel puțin:

`formă juridică × caracter confesional × autorizare/acreditare × nivel/program/specializare/locație × finanțare/an × regim cu/fără taxă × statut achiziții × program public`

Modelul țintă adaugă următoarele agregate, toate tenant/institution-scoped, versionate, cu `FORCE RLS` și dovezi eArhivă:

- `school_education_offerings`: nivel, filieră/profil/specializare/calificare, limbă, locație, capacitate aprobată și stare;
- `school_authorization_accreditation_records`: tip decizie, emitent, act/număr, ofertă educațională afectată, interval, suspendare/retragere și termen de reevaluare;
- `school_funding_instruments`: sursă, temei, an fiscal/școlar, ofertă și beneficiari eligibili, cost standard, regim fără taxă, sumă/plafon și autoritate de validare;
- `school_procurement_applicability_assessments`: încadrarea entității și, separat, încadrarea contractului/proiectului, criterii Legea nr. 98/2016 art. 4, dovezi, aprobator și termen de reevaluare;
- `school_confessional_profiles`: cult recunoscut, persoană juridică fondatoare, protocoale și aprobări specifice, numai dacă sunt aplicabile;
- `school_education_contract_templates`, `school_education_contracts` și `school_tuition_schedules`: șablon legal/instituțional versionat, contract pe beneficiar, taxe/rate/reduceri/burse/refunduri și acte adiționale;
- `school_quality_evaluations`: autoevaluare/RAEI, evaluare externă, constatări, planuri de îmbunătățire, dovezi și publicare;
- `school_network_memberships`: includerea anuală în rețeaua școlară, unități arondate/structuri, act, capacitate și valabilitate.

Valorile `public_funding` și `is_contracting_authority` existente devin numai proiecții de compatibilitate, calculate din înregistrări aprobate. Nu pot fi folosite singure pentru autorizarea unei operații și nu mai pot fi declarate liber de frontend.

## 5. Matrice funcțională public/privat

| Domeniu | Nucleu comun | Policy public | Policy privat/confesional |
| --- | --- | --- | --- |
| Guvernanță școlară | structuri, mandate, ședințe, decizii, documente | transparență și controale instituționale aplicabile | fondator, regulament intern, contract educațional |
| Finanțare | surse, bugete, centre de cost, dovezi | credite/classificație, angajamente, CFP, ordonanțare, Trezorerie | taxe, rate, reduceri/burse private, buget managerial, creanțe |
| Achiziții | necesar–aprobare–ofertă–contract–recepție | Legea 98/HG 395, PAAP/SEAP și segregare; overlay și pentru privat când este aplicabil | achiziții comerciale și reguli interne; overlay public dacă sursa/calitatea o impune |
| Contabilitate | documente, perioade, note, reconciliere, active | plan de conturi și raportare instituții publice | reglementarea entității private, fiscalitate și TVA unde se aplică |
| HR/salarizare | posturi, raport de muncă, pontaj, concedii, calificări | normare și grile/reguli pentru fonduri publice | politici contractuale, beneficii și salarizare privată |
| Catering | programe, eligibilitate, comenzi, recepții, incidente, reconciliere | programe naționale/locale activate prin eligibilitate și finanțare | contract/abonament/cantină; program public numai dacă este eligibil |
| Patrimoniu | active, custodie, inventar, mentenanță | bunuri publice/private UAT și fluxuri de transfer/casare | active proprii/închiriate și politica entității |
| Autorizare și calitate | standarde naționale, CEAC, autoevaluare/RAEI, evaluare externă, îmbunătățire | autorizare/acreditare și rețea școlară pe oferta efectivă | aceleași standarde naționale; personalitatea juridică și dreptul de operare derivă din actele aplicabile |
| Ofertă educațională și acte de studii | niveluri/programe autorizate, capacitate, înscriere, evidențe și contract educațional | regim public și efectele actelor conform legii | diplomele unității private acreditate au aceleași efecte juridice; programul neacreditat nu moștenește statutul altui program |
| Taxe și contract educațional | contractul educațional-tip și clauze instituționale compatibile cu interesul superior al elevului | fără taxă de școlarizare în regimul public obișnuit | taxe stabilite de CA, scadențe, reduceri/burse/refunduri și acte adiționale; starea fără taxă influențează eligibilitatea anumitor finanțări |
| Conducere | mandate, incompatibilități, ședințe, hotărâri și transparență | structura și desemnarea prevăzute pentru unitățile de stat | fondatorul stabilește structura permisă și desemnează conducerea CA/directorul în limitele legii; overlay de cult unde este cazul |

### 5.1 Catalog de cerințe pentru aplicabilitate public/privat

| ID | Cerință verificabilă | Starea codului la 2026-09-11 | Adaptare obligatorie |
| --- | --- | --- | --- |
| APP-001 | Același build și aceleași contracte deservește public, privat și confesional; UI nu decide regimul juridic. | fundație existentă | păstrarea modelului policy-driven și interzicerea ramurilor hardcodate în React |
| APP-002 | Tenantul și instituția sunt derivate din host/sesiune, iar toate datele sunt izolate prin scope compozit și RLS. | existent ca fundație | toate agregatele noi primesc FK compozite, `FORCE RLS` și teste de falsificare/cross-tenant |
| APP-003 | Autorizarea/acreditarea se gestionează per ofertă educațională și locație, nu printr-un singur statut instituțional. | parțial avansat: schema 0138, permisiuni dedicate, listare/creare idempotentă și actualizare/dezactivare cu optimistic concurrency, contract OpenAPI, client generat și UI PrimeReact există; intervalele părinte–autorizare sunt protejate în DB și handler, scrierile concurente sunt serializate, selectorii încarcă toate paginile, iar deciziile se înlocuiesc prospectiv cu lineage | integrare tranzacțională cu admiterea și capacitatea și E2E HTTP→PostgreSQL pentru public/privat/status expirat; invariantul DB nou necesită încă rularea pe PostgreSQL disposable real |
| APP-004 | Standardele de calitate și evaluarea periodică se aplică identic tuturor formelor de școală. | parțial în guvernanță/publicații | registru CEAC/RAEI/evaluări/planuri și pachet comun obligatoriu public/privat/confesional |
| APP-005 | Dreptul la finanțare publică pentru privat/confesional se determină pe ofertă, beneficiar și perioadă, inclusiv condiția fără taxă unde legea o cere. | parțial: schema 0139/0143 și evaluatorul tri-state există; nu există API/UI, calcul sau reconciliere | comenzi contract-first, cost standard/plafon, snapshot de beneficiari și reconciliere anuală |
| APP-006 | Primirea unei finanțări publice nu transformă automat școala privată în autoritate contractantă. | parțial: assessment entity/contract/project există în DB și evaluator; nu există flux API/UI executabil | evaluare juridică tipată cu dovezi, aprobare, expirare și aplicare în aceeași tranzacție cu achiziția |
| APP-007 | Guvernanța folosește nucleul comun, cu reguli de componență/desemnare publice sau ale fondatorului și, unde este cazul, ale cultului. | parțial; motor de guvernanță existent, policy aplicat doar publicațiilor | capabilități și validări la constituire, mandat, director, cvorum, aprobare și publicare |
| APP-008 | Contractul educațional există pentru fiecare beneficiar; privatul gestionează suplimentar taxele și actele adiționale. | parțial: tabela 0140 există; nu are API/React, versiuni semnate, eArhivă ori ledger de taxe | șabloane versionate, semnare, eArhivă, taxe/scadențe/reduceri/refunduri și separare față de finanțarea publică |
| APP-009 | Regimul contabil, salarial, TVA și Trezorerie este determinat separat de forma juridică. | doar câmpuri declarative | clasificări cu temei și verticale HR/FIN; nicio regulă financiară derivată numai din `school_legal_form` |
| APP-010 | Caracterul confesional este overlay peste regimul privat, nu o formă juridică ce elimină regulile private comune. | parțial: v2/0143 îl modelează ca overlay private-only și API-ul respinge forma directă; schema legacy 0135 păstrează încă enum-ul exclusiv | cutover v1→v2, comandă atomică de validare/persistență și compunere `private + confessional` cu cult/protocoale/dovezi |
| APP-011 | Înscrierea, școlarizarea, documentele și diplomele sunt permise numai în oferta autorizată/acreditată și în capacitatea aprobată. | clase/elevi/înscrieri există, dar fără această intersecție policy | guard tranzacțional și snapshot policy pe creare/mutare/finalizare; teste pe statut suspendat/retras |
| APP-012 | Fiecare decizie policy păstrează surse structurate: act, articol, formă consolidată, URL oficial și interval. | parțial: profilul v2 cere sursă tipată, HTTPS și SHA-256; citirea/evaluarea live și pack-urile legacy folosesc încă proiecții libere | cutover la surse v2, job de revalidare și impact report fără rescrierea istoricului |
| APP-013 | Contractul educațional are retenție calculată pe durata școlarizării și încă 2 ani după plecarea elevului; actele adiționale sunt aprobate și append-only. | schemă incipientă, fără lifecycle/retention executabil | stări, semnături, versiuni, două exemplare, eveniment de plecare și politică eArhivă verificată automat |
| APP-014 | Bursele/facilitățile sunt separate de taxele comerciale și folosesc reguli pe an, sursă și statut cu/fără taxă. | lipsă | catalog versionat, eligibilitate tri-state, dovezi, aprobare, plăți/reversări și reguli nehardcodate |
| APP-015 | Desemnarea responsabilului REGES-ONLINE/EDUSAL și închiderea anuală a catalogului electronic produc dovezi auditate și arhivate. | lipsă ca flux integrat | decizie/mandat, jurnal transmiteri, export probator, hash, semnare și legătură eArhivă |
| APP-016 | Reîncadrarea juridică, acreditarea, finanțarea și regulile anuale au efect prospectiv și produc impact report. | parțial la nivel de intervale/snapshot policy | comandă administrativă versionată, revalidare automată și dovadă că istoricul nu este rescris |

Verdict: arhitectura comună este adecvată, dar suportul public/privat nu este complet. Fundația de profil și policy trebuie normalizată conform APP-003/005/006/010/012, iar motorul trebuie conectat la operațiile educaționale și operaționale, nu doar la publicații.

## 6. Catalogul bounded contexts

### OPS-REG — profil și conformitate

- administrare profil juridic/acreditare/finanțare;
- registru de surse și policy packs versionate;
- calendar de intrare în vigoare și analiză de impact;
- resolver de capabilități și snapshot al regulii aplicate;
- raport de operațiuni blocate sau afectate de schimbarea normei.

### OPS-CON — parteneri, contracte și utilități

- furnizori peste `app_parties`, fără registru duplicat;
- contracte, versiuni/acte adiționale, obligații, SLA, garanții, termene și notificări;
- categorii: încălzire, electricitate, gaz, apă, salubritate, pază, internet, DDD, mentenanță, asigurări și catering;
- puncte de consum, contoare, citiri, consum și reconciliere cu factura;
- flux `draft → verificat → aprobat → semnat → activ → suspendat/expirat/reziliat → arhivat`;
- contractul expirat blochează comenzile noi, exceptând o procedură explicită de urgență.

### OPS-PROC — achiziții

- plan anual și linii, necesar/referat, disponibil bugetar;
- dosar, loturi, CPV, criterii, invitații/oferte, evaluări și conflicte de interese;
- atribuire, comandă/contract, recepție și neconformități;
- procedură și documente determinate de policy, inclusiv prevenirea fragmentării artificiale;
- separarea inițiator–achiziții–CFP–ordonator–recepție–plată când este aplicabilă.

### OPS-CAT — catering și programe alimentare

- model generic de program; „cornul și laptele” nu este hardcodat ca produse fixe;
- eligibilitate și snapshot elevi/clase, calendar, meniu, produse și alergeni;
- comandă zilnică, livrare/lot, temperatură, ambalaj, termen și recepție;
- distribuție agregată, absențe, porții nedistribuite, refuz, retragere lot, incident și risipă;
- reconciliere comandat–livrat–acceptat–distribuit–facturat și raportarea programului;
- diagnosticul medical rămâne într-un scope separat; cateringul primește numai restricția operațională minimă autorizată.

### OPS-LOG — logistică, stocuri și patrimoniu

- campus, clădire, spațiu și depozit;
- articol, lot, termen, mișcare stoc și inventar;
- bun patrimonial, număr inventar, locație, custode și predare-primire;
- mentenanță preventivă/corectivă, ordin de lucru, transfer și casare;
- QR/barcode și inventariere mobilă;
- Finance deține valoarea contabilă și amortizarea; Logistics deține existența și custodia fizică.

### OPS-COM — SSM, PSI și securitate

- obligații/termene, evaluări de risc, autorizații și certificate;
- instruiri și dovezi, simulări, inspecții, constatări și acțiuni corective;
- incidente, plan de securitate, zone și responsabilități;
- legătură cu contractele de pază, monitorizare și mentenanță;
- CCTV nu este inclus în nucleul inițial; se păstrează doar referințe, temei, acces și retenție dacă va fi integrat.

### OPS-HR — resurse umane

- `education_personnel` rămâne identitatea canonică;
- organigramă, stat de funcții, post, raport de muncă/numire, versiuni contractuale și FTE/normă;
- pontaj, program, concedii, absențe, substituții, calificări și formare;
- aptitudine medicală numai ca stare/valabilitate, cu datele medicale separate;
- instruiri SSM/PSI, payroll inputs și integrare/export către sistemele oficiale;
- încetarea închide asignările și accesul, dar păstrează dosarul conform retenției.

### OPS-FIN — economic și contabil

- exerciții/perioade, bugete, surse de finanțare și centre de cost;
- angajamente, facturi, documente justificative, încasări, plăți și reconciliere;
- plan de conturi versionat, note debit/credit, perioade închise și reversal;
- active/amortizare și pachete de raportare;
- profil public: clasificație, CFP, ordonanțare, Trezorerie și raportare publică;
- profil privat: taxe școlare, contracte cu părinții, creanțe, fiscalitate/TVA și raportarea entității;
- prima versiune poate fi control economic cu export verificabil; contabilitatea statutară și salarizarea completă necesită validare profesională separată.

## 7. RBAC și segregarea atribuțiilor

Roluri funcționale: `responsabil_contract`, `responsabil_achizitii`, `responsabil_catering`, `receptioner`, `gestionar`, `administrator_patrimoniu`, `responsabil_ssm`, `responsabil_psi`, `security_officer`, `administrator_financiar`, `contabil`, `casier`, `cfp`, `auditor`, `external_accountant`.

Familiile de permisiuni sunt distincte: `operations.contracts.*`, `operations.utilities.*`, `procurement.*`, `catering.*`, `logistics.stock.*`, `assets.*`, `maintenance.*`, `compliance.ssm.*`, `compliance.psi.*`, `compliance.security.*`, `hr.*` și `finance.*`.

Reguli obligatorii:

- rolurile sunt tenant-scoped; responsabilitățile pot fi resource-scoped și temporale;
- administratorul tehnic nu primește automat acces HR, salarial, financiar sau medical;
- aceeași persoană nu poate iniția, viza CFP și aproba plata când segregarea este obligatorie;
- introducerea facturii și autorizarea plății sunt permisiuni diferite;
- `super_admin` nu devine implicit actor operațional;
- accesul extern este limitat temporal, la scop, read-only implicit și complet auditat;
- refuzul cross-tenant/cross-institution este nediferențiabil de obiect inexistent.

## 8. Contractul DB → OpenAPI → React

Fiecare tabel nou are UUID, `tenant_code`, `institution_id`, FK compozite, `version`, actor/timestamps, `FORCE RLS`, indexuri scoped, audit/versioning și reguli explicite de ștergere. Înregistrările cu efect juridic/financiar se corectează prin anulare sau reversal, nu prin hard delete, și păstrează `policy_evaluation_id`.

Namespace-uri canonice:

- `/api/institution/regulatory-profile` și `/api/institution/capabilities`;
- `/api/school-operations/...`;
- `/api/procurement/...`;
- `/api/catering/...`;
- `/api/logistics/...`;
- `/api/compliance/...`;
- `/api/hr/...`;
- `/api/finance/...`.

Fiecare operație are DTO închis, erori comune, `x-required-permission`, `x-tenant-scope` și `x-required-policy-capability`. Comenzile acceptă `expected_version`, iar create/import folosesc idempotency key. React utilizează exclusiv rute și tipuri generate; serverul livrează capabilitățile, iar meniul/acțiunile sunt intersecția modul–permisiune–policy.

## 9. Integrări între module

- Procure-to-pay: necesar → buget → aprobare → procedură → atribuire → contract/comandă → recepție → factură → three-way match → CFP/aprobare → plată → postare/export.
- Catering: eligibilitate → comandă → livrare/recepție → distribuție/incidente → reconciliere → factură → raport.
- HR: decizie/contract → Registratură/eArhivă → raport de muncă → post/program → instruiri → pontaj/concedii → payroll input → Finance.
- Documentele finale intră în eArhivă prin outbox idempotent, retry/dead-letter, hash și proveniență; bloburile nu se duplică în tabelele operaționale.
- Workflow coordonează taskurile, însă starea domeniului rămâne autoritatea.

## 10. Criterii de acceptare public/privat

O verticală nu este completă până când testele automate demonstrează cumulativ:

1. aceeași operație de nucleu funcționează într-un tenant public și unul privat;
2. publicul primește pașii/documentele/aprobările suplimentare ale policy pack-ului;
3. privatul folosește politica internă fără a moșteni automat fluxuri publice;
4. privatul cu finanțare/program public primește overlay-ul obligatoriu;
5. profilul, tenantul și instituția nu pot fi falsificate în payload;
6. schimbarea policy pack-ului nu rescrie tranzacțiile istorice;
7. există teste cross-role, cross-institution și cross-tenant;
8. OpenAPI/clientul React nu au drift, iar UI are loading/empty/error/conflict/retry;
9. E2E real trece prin React → OIDC → API → PostgreSQL/storage;
10. UI PrimeReact este funcțional la 320 px, tabletă și desktop.
11. aceeași ofertă educațională nu poate fi folosită în afara nivelului, locației, capacității și intervalului autorizat/acreditat;
12. un privat acreditat, un privat autorizat fără taxă, un privat autorizat cu taxă și un confesional primesc rezultate de policy distincte și corecte;
13. o școală privată cu fonduri publice, dar fără încadrare conform Legii nr. 98/2016, nu este tratată automat drept autoritate contractantă;
14. suspendarea/retragerea unei autorizări blochează operațiile viitoare fără a modifica tranzacțiile și actele istorice;
15. revalidarea anuală a finanțării, rețelei școlare și policy pack-urilor produce avertizări și raport de impact înainte de expirare.
