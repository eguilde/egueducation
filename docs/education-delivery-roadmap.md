# Roadmap de livrare pentru management educational

Data: 2026-06-24

Documente sursa:

- [docs/education-management-specifications.md](/E:/dev/egueducation/docs/education-management-specifications.md)
- [docs/education-implementation-audit.md](/E:/dev/egueducation/docs/education-implementation-audit.md)
- [docs/education-current-plan.md](/E:/dev/egueducation/docs/education-current-plan.md)

## 1. Scop

Acest document transforma specificatiile si auditul in plan executabil de livrare.

Este gandit pentru:

- prioritizare;
- spargere pe epics si stories;
- organizarea implementarii pe faze;
- clarificarea dependentelor;
- definirea criteriilor de ready si done.

## 2. Principii de executie

### 2.1. Nu rescriem fundatia existenta

Se reutilizeaza:

- schema de date actuala;
- endpointurile education existente;
- subresursele deja modelate;
- PDF-urile si exporturile existente unde sunt utile;
- taxonomiile, RBAC-ul si auditul.

### 2.2. Mutam valoarea in UX procedural

Prioritatea nu este sa mai adaugam inca 20 de tabele generice, ci:

- wizard-uri;
- cockpituri;
- rapoarte standard;
- documente oficiale generate;
- roluri contextuale.

### 2.3. Livram vertical

Fiecare faza trebuie sa produca un flux utilizabil cap-coada:

- backend;
- frontend;
- RBAC;
- PDF/documente;
- audit;
- raportare minima.

### 2.4. Limitam schimbarea structurala

Adaugam entitati noi doar unde lipsa este reala:

- `ClassCouncil`;
- `Committee`;
- `CommitteeReport`;
- agregari dashboard;
- pachete inspectie.

## 3. Imagine generala

Ordinea recomandata:

1. Fundatie UX si RBAC contextual
2. Cockpit director
3. Wizard sedinta CA
4. Wizard sedinta CP
5. Portofoliu profesor
6. Procedura interna portofolii
7. Documente manageriale ghidate
8. Dashboard profesor si secretariat
9. Rapoarte standard si pachet inspectie
10. Consiliul clasei si comisii

## 4. Faza 0 - Fundatie de livrare

Scop:

- sa pregatim infrastructura de produs pentru restul fazelor.

### Epic 0.1 - Standardizare statusuri si actiuni

Stories:

- definire vocabular comun pentru statusuri educationale folosite in UI;
- mapare statusuri backend -> taguri si texte consistente in frontend;
- inventar actiuni sensibile pe fiecare domeniu;
- standardizare naming pentru documente generate.

Deliverables:

- lista unica de statusuri pentru guvernanta, portofolii, documente, evaluari;
- utilitare frontend pentru afisare consistenta;
- document intern de naming.

### Epic 0.2 - RBAC contextual

Stories:

- definire permisiuni pe actiune: `meeting.close`, `meeting.vote`, `meeting.publish_minutes`, `portfolio.verify`, `portfolio.transfer`, `document.publish`, `report.export_sensitive`;
- definire model contextual pentru membru CA/CP;
- definire reguli pentru profesor pe propriul portofoliu/evaluare;
- definire reguli pentru inspector pe pachet inspectie.

Deliverables:

- lista permisiuni noi;
- mapari roluri globale -> permisiuni;
- reguli contextuale in backend;
- garduri UI noi.

### Epic 0.3 - Fundatie frontend pentru wizard-uri

Stories:

- componenta `WizardStepper`;
- componenta `WizardSummaryPanel`;
- componenta `DocumentPreviewPanel`;
- componenta `DeadlineBadge`;
- componenta `AuditTrailPanel`.

Deliverables:

- componente reutilizabile Angular 21 standalone;
- modele comune pentru pasi, status, erori si rezumat.

Definition of done pentru faza 0:

- exista baza reutilizabila pentru wizard-uri;
- permisiunile contextuale de baza sunt definite;
- statusurile educationale sunt consistente in UI.

## 5. Faza 1 - Cockpit director

Scop:

- directorul sa poata vedea dintr-un loc datele relevante pentru managementul scolii.

### Epic 1.1 - Agregari backend pentru cockpit

Stories:

- endpoint agregat `director cockpit`;
- agregari pentru sedinte, hotarari, publicari;
- agregari pentru portofolii;
- agregari pentru evaluari;
- agregari pentru documente manageriale;
- agregari pentru personal;
- agregari pentru conformitate.

Deliverables:

- endpointuri noi de sumar;
- filtre pe an scolar si unitate;
- semnalare intarzieri si riscuri.

### Epic 1.2 - UI dashboard director

Stories:

- ecran `education/dashboard/director`;
- widget calendar operational;
- widget guvernanta;
- widget portofolii;
- widget evaluari;
- widget documente manageriale;
- widget conformitate;
- shortcut-uri catre fluxuri.

Deliverables:

- dashboard operational, nu doar KPI-uri;
- filtre rapide pe status;
- deep link-uri catre inregistrarile relevante.

### Epic 1.3 - Rapoarte rapide din cockpit

Stories:

- export liste restante;
- export sedinte in asteptare;
- export portofolii incomplete;
- export documente nepublicate.

Definition of done pentru faza 1:

- directorul poate vedea situatia operationala reala a scolii dintr-un singur ecran;
- fiecare widget duce catre un flux sau raport actionabil.

## 6. Faza 2 - Wizard sedinta CA

Scop:

- flux complet pentru constituire si operare sedinta CA.

### Epic 2.1 - Constituire CA

Stories:

- modelare completa componenta CA pe tip de unitate;
- calcul automat structura minima;
- generare solicitari de desemnare;
- inregistrare raspunsuri;
- generare decizie de constituire;
- publicare interna a componentei.

Deliverables:

- wizard constituire CA;
- documente generate;
- componenta CA valida pe an scolar.

### Epic 2.2 - Sedinta CA

Stories:

- planificare sedinta;
- ordine de zi;
- atasare documente suport;
- convocator;
- confirmare participare;
- calcul cvorum din membrii activi;
- vot pe puncte;
- generare hotarari;
- generare proces-verbal;
- inchidere sedinta.

Deliverables:

- wizard sedinta CA;
- status procedural sedinta;
- validari de cvorum si prag vot;
- hotarari legate de voturi.

### Epic 2.3 - Documente CA

Stories:

- PDF convocator CA;
- PDF proces-verbal CA;
- PDF hotarare CA;
- registru sedinte CA;
- registru hotarari CA.

Definition of done pentru faza 2:

- directorul sau secretariatul poate duce o sedinta CA de la planificare la proces-verbal final din sistem;
- sistemul blocheaza inchiderea invalida.

## 7. Faza 3 - Wizard sedinta CP

Scop:

- flux procedural complet pentru CP, cu accent pe prezenta si vot.

### Epic 3.1 - Componenta CP

Stories:

- constituire CP din cadre didactice cu norma de baza;
- marcarea dreptului de vot;
- evidenta absentelor motivate si nemotivate.

### Epic 3.2 - Sedinta CP

Stories:

- convocare;
- tematica si ordine de zi;
- atasare materiale;
- confirmare prezenta;
- calcul cvorum;
- vot pe puncte decizionale;
- minute si proces-verbal.

### Epic 3.3 - Documente CP

Stories:

- PDF convocator CP;
- PDF proces-verbal CP;
- registru sedinte CP;
- raport absente CP.

Definition of done pentru faza 3:

- CP poate fi condus procedural din sistem;
- absentele si voturile sunt trasabile.

## 8. Faza 4 - Portofoliu profesor

Scop:

- profesorul sa aiba un flux clar si simplu pentru propriul portofoliu.

### Epic 4.1 - Workspace personal profesor

Stories:

- pagina `education/portfolio/me`;
- rezumat status portofoliu;
- documente lipsa;
- checklist personal;
- termene;
- notificari.

### Epic 4.2 - Self-service portofoliu profesor

Stories:

- confirmare date;
- alegere unitate si statut;
- format digital/fizic/mixt;
- completare sectiuni;
- incarcari documente;
- consimtamant GDPR;
- declaratie autenticitate;
- depunere;
- retrimitere dupa completari.

### Epic 4.3 - Opis automat

Stories:

- generare opis din documente;
- ordonare cronologica;
- export PDF opis;
- istoric modificari opis.

### Epic 4.4 - Review institutional

Stories:

- flux verificare secretariat;
- flux validare manageriala;
- returnare pentru completari;
- scor de conformitate;
- jurnal verificare.

Definition of done pentru faza 4:

- profesorul isi poate gestiona portofoliul fara sa opereze registre generic;
- secretariatul si directorul pot verifica si returna portofoliul procedural.

## 9. Faza 5 - Procedura interna portofolii

Scop:

- directorul sa poata configura, aproba si publica procedura interna.

### Epic 5.1 - Configurator procedura

Stories:

- termene interne;
- mod de depunere;
- formate acceptate;
- reguli de acces;
- custode si responsabili;
- calendare intermediare;
- arhivare si transfer.

### Epic 5.2 - Aprobare si publicare

Stories:

- trimitere la CA;
- aprobare;
- document final;
- publicare interna;
- versiuni procedura.

Definition of done pentru faza 5:

- exista procedura interna operationala direct in sistem;
- setarile procedurii guverneaza comportamentul wizard-ului de portofoliu.

## 10. Faza 6 - Documente manageriale ghidate

Scop:

- sa transformam dosarele manageriale din resurse generice in documente ghidate.

### Epic 6.1 - PDI/PAS wizard

Stories:

- structura ghidata pe capitole;
- SWOT;
- PEST/PESTEL;
- tinte strategice;
- indicatori;
- responsabili si termene;
- versiuni;
- avizare/aprobare.

### Epic 6.2 - Raport anual asupra calitatii

Stories:

- structura ghidata;
- captare indicatori;
- sectiuni obligatorii;
- atasare dovezi;
- aprobare.

### Epic 6.3 - RAEI si regulamente

Stories:

- workflow dedicat pentru RAEI;
- workflow regulamente;
- versiune aprobata imutabila;
- publicare.

### Epic 6.4 - PDF-uri manageriale

Stories:

- PDF PDI/PAS;
- PDF raport calitate;
- PDF RAEI;
- PDF regulament/versiune.

Definition of done pentru faza 6:

- documentele manageriale majore pot fi redactate si aprobate ghidat;
- exista versiune oficiala generata din sistem.

## 11. Faza 7 - Dashboard profesor si secretariat

Scop:

- fiecare rol operational sa primeasca o experienta care il ajuta direct.

### Epic 7.1 - Dashboard profesor

Stories:

- portofoliu;
- evaluarea proprie;
- sedinte CP;
- sarcini din comisii;
- termene si notificari.

### Epic 7.2 - Dashboard secretariat

Stories:

- documente de inregistrat;
- convocatoare de trimis;
- procese-verbale in lucru;
- documente de arhivat;
- termene de raspuns.

### Epic 7.3 - Dashboard inspector

Stories:

- pachete inspectie;
- documente lipsa;
- unitati verificate;
- rapoarte institutionale;
- observatii deschise.

Definition of done pentru faza 7:

- exista dashboarduri diferite, centrate pe sarcinile reale ale fiecarui rol.

## 12. Faza 8 - Rapoarte standard

Scop:

- rapoarte institutionale standard, exportabile, repetabile.

### Epic 8.1 - Rapoarte guvernanta

Stories:

- registru sedinte CA;
- registru sedinte CP;
- registru hotarari;
- raport cvorum si participare;
- proces-verbal incomplet / nesemnat.

### Epic 8.2 - Rapoarte portofolii

Stories:

- situatie portofolii pe an scolar;
- portofolii incomplete;
- opis pe profesor;
- transferuri;
- arhivare si retentie.

### Epic 8.3 - Rapoarte evaluari si personal

Stories:

- stadiu evaluari;
- contestatii;
- punctaje pe domenii;
- dosare personale incomplete;
- participare CP.

### Epic 8.4 - Rapoarte manageriale si conformitate

Stories:

- stadiu PDI/PAS;
- documente publicate/nepublicate;
- cerinte fara dovada;
- versiuni si istoric.

### Epic 8.5 - Exporturi

Stories:

- PDF;
- CSV;
- XLSX pentru rapoarte tabelare importante;
- filtre salvabile.

Definition of done pentru faza 8:

- directorul si inspectorul pot obtine rapoarte standard fara operare manuala extinsa.

## 13. Faza 9 - Pachet inspectie

Scop:

- sa pregatim un export controlat si auditabil pentru inspectie.

### Epic 9.1 - Selecare pachet

Stories:

- selectare domenii;
- selectare perioada;
- selectare unitate;
- selectare documente incluse.

### Epic 9.2 - Construire pachet

Stories:

- export ZIP;
- inventar inclus;
- PDF sumar;
- jurnal export;
- control acces inspector.

Definition of done pentru faza 9:

- inspectorul poate primi un pachet complet si trasabil, fara colectare manuala dispersata.

## 14. Faza 10 - Consiliul clasei si comisii

Scop:

- acoperirea zonelor lipsa din modelul educational operational.

### Epic 10.1 - Consiliul clasei

Stories:

- entitati;
- componenta;
- sedinte;
- masuri;
- termene;
- raport/proces-verbal.

### Epic 10.2 - Comisii

Stories:

- constituire comisie;
- membri;
- plan activitate;
- raport comisie;
- relatii cu CA/CP si documentele manageriale.

Definition of done pentru faza 10:

- consiliul clasei si comisiile devin module explicite, nu doar referinte dispersate.

## 15. Dependente critice

### Dependente de produs

- RBAC contextual este necesar inainte de dashboarduri pe rol si wizard-uri sensibile.
- Standardizarea statusurilor este necesara inainte de cockpit si rapoarte.
- Componenta reutilizabila de wizard este necesara inainte de CA/CP/portofolii.

### Dependente backend

- agregarile dashboard trebuie facute inainte de cockpit;
- calculul procedural de cvorum trebuie facut inainte de wizard sedinta;
- PDF-urile oficiale trebuie facute inainte de inchiderea completa a fluxurilor CA/CP;
- raportarea standard cere endpointuri agregate dedicate, nu doar export generic.

### Dependente frontend

- `WizardStepper` si `DocumentPreviewPanel` trebuie sa existe inainte de primele fluxuri ghidate;
- dashboardurile pe rol cer un strat de carduri si liste actionabile separat de workspace-ul generic;
- unele ecrane noi trebuie sa stea alaturi de `EducationDomainWorkspaceComponent`, nu in interiorul lui.

## 16. Definition of ready

Un epic este ready cand:

- exista scop clar;
- exista rol principal;
- exista model de date suficient sau decizia de extindere;
- exista lista minimala de endpointuri;
- exista lista de documente generate;
- exista criterii de acceptanta;
- exista decizie de prioritate.

O story este ready cand:

- are rezultat observabil;
- are actor clar;
- are campuri si validari clare;
- are dependente identificate;
- poate fi verificata independent.

## 17. Definition of done

O story este done cand:

- backendul este implementat;
- UI-ul este utilizabil;
- RBAC-ul este aplicat;
- auditul exista pentru actiunile sensibile;
- documentele sau exporturile promise functioneaza, daca sunt in scope;
- criteriul de acceptanta este verificabil.

Un epic este done cand:

- fluxul este utilizabil cap-coada;
- nu mai necesita tabele generice pentru scenariul principal;
- actorul principal isi poate duce sarcina in sistem;
- exista minim un raport sau sumar relevant;
- documentele oficiale aferente exista daca procesul le cere.

## 18. MVP recomandat

MVP-ul de livrare recomandat pentru cea mai mare valoare operationala este:

1. Faza 0
2. Faza 1
3. Faza 2
4. Faza 3
5. Faza 4

Acest MVP produce:

- cockpit director;
- sedinte CA/CP procedurale;
- portofoliu profesor usor de folosit;
- documente oficiale esentiale;
- baza clara pentru restul extensiilor.

## 19. Primul sprint recomandat

Sprintul 1 ar trebui sa atace doar infrastructura care deblocheaza restul:

- standardizare statusuri education;
- permisiuni contextuale noi;
- componente frontend de wizard;
- endpoint agregat pentru cockpit director;
- schelet UI `dashboard/director`;
- design tehnic pentru cvorum CA/CP.

Rezultatul sprintului 1 trebuie sa fie:

- fara functionalitate spectaculoasa inca;
- dar cu fundatia corecta pentru a livra rapid fazele 1-4 fara rework.

## 20. Al doilea sprint recomandat

Sprintul 2:

- widgeturi cockpit director;
- calcul cvorum CA;
- wizard sedinta CA;
- PDF convocator CA;
- PDF proces-verbal CA;
- raport sedinte CA.

## 21. Al treilea sprint recomandat

Sprintul 3:

- wizard sedinta CP;
- dashboard profesor minimal;
- inceput `my portfolio`;
- opis automat generatie 1;
- notificari de baza.

## 22. Concluzie

Planul optim nu este sa extindem inca un strat de CRUD generic.

Planul optim este:

- sa folosim fundatia buna deja existenta;
- sa o imbracam in experiente procedurale pe rol;
- sa livram cockpituri, wizard-uri, rapoarte si documente oficiale;
- sa tratam managementul educational ca sistem de operare al scolii, nu doar ca registru digital.
