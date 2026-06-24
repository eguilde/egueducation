# Plan complet pentru ce a ramas de facut in managementul educational

Data: 2026-06-24

Documente de referinta:

- [docs/education-management-specifications.md](/E:/dev/egueducation/docs/education-management-specifications.md)
- [docs/education-implementation-audit.md](/E:/dev/egueducation/docs/education-implementation-audit.md)
- [docs/education-delivery-roadmap.md](/E:/dev/egueducation/docs/education-delivery-roadmap.md)
- [docs/education-phase-0-1-execution-plan.md](/E:/dev/egueducation/docs/education-phase-0-1-execution-plan.md)
- [docs/education-current-plan.md](/E:/dev/egueducation/docs/education-current-plan.md)
- [docs/rbac-matrix.md](/E:/dev/egueducation/docs/rbac-matrix.md)

## 1. Scop

Acest document este planul unificat pentru tot ce a ramas de implementat in modulul `education`, dupa fundatia deja construita.

Este gandit pentru:

- prioritizare pe trimestre sau sprinturi;
- executie pe verticale complete;
- aliniere intre backend, frontend, RBAC, rapoarte si dashboarduri;
- urmarirea progresului fara a relua auditul de la zero.

## 2. Ce consideram deja realizat

La data acestui plan, consideram livrate sau aproape livrate:

- workspace-ul generic education;
- configurarea pe domenii si resurse;
- dashboard director si rapoarte standard initiale;
- deep links cu preseturi si export contextual;
- fundatia de wizard-uri reutilizabile;
- wizard-uri initiale pentru:
  - sedinte CA;
  - minute;
  - voturi;
  - hotarari;
  - dosare manageriale;
  - portofolii;
  - personal;
  - evaluari;
  - declaratii;
  - mobilitate;
  - gradatie de merit;
- suita e2e pentru aceste fluxuri.

## 3. Ce ramane de facut

Lucrul ramas intra in sase categorii mari:

1. completare roluri si RBAC contextual;
2. extindere fluxuri educationale pe verticale noi;
3. transformarea CRUD-urilor bune in experiente procedurale complete;
4. dashboarduri si rapoarte pe roluri;
5. backend hardening si documente oficiale;
6. testare pe straturi si operationalizare locala.

## 4. Principii de executie

### 4.1. Livram pe verticale complete

Nu mai facem faze separate de tip:

- doar backend;
- doar tabele;
- doar formulare.

Fiecare verticala noua trebuie sa includa:

- model de date;
- endpointuri reale;
- RBAC;
- wizard sau ecran procedural;
- dashboard hooks;
- rapoarte minime;
- teste.

### 4.2. Refolosim fundatia existenta

Se reutilizeaza in mod implicit:

- `EducationDomainWorkspaceComponent`;
- kitul de wizard-uri;
- exporturile standard;
- componentele de dashboard;
- patternurile de deep link cu preset.

### 4.3. Rolurile conduc experienta

Tot ce urmeaza se construieste pornind de la actor:

- director;
- director adjunct;
- secretariat;
- profesor;
- diriginte;
- evaluator;
- membru CA;
- membru CP;
- inspector;
- responsabil comisie.

## 5. Harta de executie

Ordinea recomandata pentru tot ce a ramas este:

1. RBAC complet si matrice de roluri
2. Verticala guvernanta completa
3. Verticala documente manageriale si conformitate
4. Verticala personal si dosar personal
5. Verticala evaluari, mobilitate, merit - hardening procedural
6. Verticala elevi, clase, diriginte si situatii scolare
7. Dashboarduri pe roluri
8. Rapoarte standard complete
9. Pachet inspectie si export institutional
10. Hardening transversal si suita completa de teste

## 6. Faza A - Roluri si RBAC complet

Status: partial realizata

### Obiectiv

Sa definim modelul complet de acces si de responsabilitati pentru toate rolurile educationale.

### Livrabile

- matrice extinsa rol -> permisiuni -> resurse -> actiuni;
- permisiuni contextuale pentru actiuni sensibile;
- UI gating coerent;
- filtre si dashboarduri diferite pe rol.

### Roluri de inchis explicit

- `director`
- `director_adjunct`
- `secretar`
- `registrator`
- `profesor`
- `diriginte`
- `membru_ca`
- `membru_cp`
- `evaluator`
- `responsabil_comisie`
- `inspector`
- `gdpr_officer`

### Taskuri

1. Extindere [rbac-matrix.md](/E:/dev/egueducation/docs/rbac-matrix.md) cu rolurile lipsa si actiunile educationale detaliate.
2. Introducere permisiuni contextuale finale pentru:
   - acces la propriile dosare;
   - acces la dosarele subordonatilor;
   - participare in sedinte;
   - validare si aprobare;
   - exporturi sensibile;
   - pachet inspectie.
3. Backend:
   - helperi reutilizabili pentru verificari contextuale;
   - aplicare in endpointurile sensibile.
4. Frontend:
   - ascundere actiuni nepermise;
   - redirecturi corecte pentru dashboarduri pe rol.

### Definition of done

- niciun flux major nu depinde doar de rol global;
- actiunile sensibile sunt controlate contextual;
- fiecare rol vede doar ce are nevoie sa opereze.

## 7. Faza B - Guvernanta completa

Status: fundatie buna, dar incompleta procedural

### Obiectiv

Sa inchidem cap-coada guvernanta institutionala pentru CA, CP, comisii si consiliul clasei.

### Subfaza B1 - CA complet

Mai ramane:

- workflow complet de inchidere sedinta;
- validari stricte de cvorum;
- validari pe vot si tip de decizie;
- documente oficiale finale;
- registre consolidate.

### Subfaza B2 - CP complet

De implementat:

- componenta CP;
- convocare;
- prezenta;
- tematica;
- vot;
- proces-verbal;
- registru sedinte CP.

### Subfaza B3 - Comisii

De implementat:

- constituire comisii;
- membri si responsabilitati;
- plan activitate;
- sedinte;
- rapoarte de comisie;
- legaturi cu documente manageriale si conformitate.

### Subfaza B4 - Consiliul clasei

De implementat:

- componenta;
- sedinte;
- masuri;
- urmarire termene;
- procese-verbale;
- raportare pe clasa.

### Livrabile

- ecrane procedurale dedicate pentru sedinte;
- PDF convocator, proces-verbal, hotarare;
- dashboard cards pentru restante de guvernanta;
- rapoarte standard pentru sedinte si hotarari.

### Definition of done

- o sedinta poate fi gestionata de la planificare pana la document final;
- CA si CP sunt trasabile procedural;
- comisiile si consiliul clasei sunt module explicite.

## 8. Faza C - Documente manageriale si conformitate

Status: partial realizata

### Obiectiv

Sa transformam documentele manageriale din registre bune in fluxuri institutionale complete.

### Domenii de implementat sau rafinat

- PDI/PAS;
- RAEI;
- regulament intern;
- regulament de organizare si functionare;
- raport anual asupra calitatii;
- decizii si publicari;
- requirement mapping si dovezi de conformitate.

### Taskuri

1. Tipizare mai stricta pe document.
2. Workflow complet:
   - draft;
   - avizare;
   - aprobare;
   - publicare;
   - revizie;
   - arhivare.
3. Legare intre:
   - document;
   - cerinta;
   - dovada;
   - publicare;
   - istoric versiuni.
4. PDF-uri oficiale dedicate.
5. Dashboard de conformitate pentru conducere.

### Definition of done

- documentele manageriale majore se pot redacta, aviza, aproba si publica din sistem;
- exista trasabilitate intre cerinta si documentul care o satisface.

## 9. Faza D - Personal si dosar personal

Status: avansata, dar cere UX si reguli mai bune

### Obiectiv

Sa facem zona de personal utilizabila operational pentru HR, secretariat si conducere.

### Zone de lucru

- dosar personal complet;
- incadrari;
- roluri si functii;
- documente lipsa;
- acces sensibil si audit;
- cazuri disciplinare;
- declaratii si anexe institutionale.

### Taskuri

1. Ecran procedural de dosar personal, nu doar taburi generice.
2. Grupare documente pe categorii institutionale.
3. Validari pentru completitudine dosar.
4. Audit si restrictii pentru acces sensibil.
5. Rapoarte:
   - dosare incomplete;
   - documente expirate;
   - acces recent;
   - discipline deschise.

### Definition of done

- secretariatul sau HR poate vedea rapid ce lipseste din dosar;
- accesul la date sensibile este auditat si limitat corect.

## 10. Faza E - Evaluari, mobilitate si gradatie de merit

Status: bine implementate ca fundatie, neinchise procedural

### Obiectiv

Sa rafinam fluxurile existente pana la nivel institutional complet.

### Ce mai trebuie

- stari de business stricte;
- tranzitii validate;
- ecrane mai putin generice;
- rapoarte specializate;
- exporturi institutionale;
- scenarii de contestatie mai bine urmarite.

### Taskuri

1. Validari suplimentare pe tranzitii de status.
2. Vederi dedicate pe caz, cu timeline procedural.
3. Dashboard hooks:
   - contestatii deschise;
   - dosare in intarziere;
   - comunicari nefinalizate.
4. Rapoarte standard:
   - ierarhizare;
   - punctaje;
   - contestatii;
   - rezultat final.

### Definition of done

- fluxurile nu mai arata ca simple registre;
- actorii pot opera procedural cap-coada fiecare dosar.

## 11. Faza F - Elevi, clase, diriginte si situatie scolara

Status: de pornit

### Obiectiv

Sa deschidem urmatoarea verticala mare, orientata direct spre operarea scolii de zi cu zi.

### Module de introdus

- elevi;
- clase;
- inscrieri;
- transferuri;
- retrageri;
- repartizari;
- dirigintie;
- situatie scolara;
- absente;
- recompense;
- sanctiuni;
- risc educational.

### Fluxuri prioritare

1. creare si administrare clasa;
2. alocare diriginte;
3. dosar elev;
4. inscriere si transfer;
5. situatie scolara si monitorizare;
6. absente si notificari;
7. masuri de sprijin sau interventie.

### Dashboarduri implicate

- director;
- diriginte;
- secretariat;
- profesor.

### Definition of done

- exista o verticala functionala pentru elev si clasa;
- sistemul poate sustine cazuri reale de administrare scolara de baza.

## 12. Faza G - Dashboarduri pe roluri

Status: doar directorul este avansat

### Obiectiv

Sa avem cockpituri separate si utile pentru rolurile cheie.

### Dashboarduri de construit

- profesor;
- diriginte;
- secretariat;
- HR/personal;
- evaluator;
- inspector;
- responsabil comisie.

### Continut minim pentru fiecare

- sarcini curente;
- termene;
- alerte;
- documente in asteptare;
- fluxuri recomandate;
- scurtaturi catre wizard-uri sau liste filtrate.

### Definition of done

- fiecare rol intra in sistem si vede imediat ce are de facut;
- scade dependenta de navigare manuala prin taburi si tabele.

## 13. Faza H - Rapoarte standard complete

Status: fundatie livrata, catalog incomplet

### Obiectiv

Sa definim si implementam catalogul standard de rapoarte pentru operarea scolii.

### Catalog recomandat

- registru sedinte CA;
- registru sedinte CP;
- registru hotarari;
- participare si cvorum;
- documente manageriale pe status;
- publicari restante;
- dosare personale incomplete;
- evaluari si contestatii;
- mobilitate;
- gradatie de merit;
- portofolii;
- situatie elevi si clase;
- absente si risc;
- conformitate institutionala;
- pachet inspectie.

### Exporturi

- CSV pentru lucru operational;
- PDF pentru prezentare si arhivare;
- XLSX unde raportul este tabelar si necesita prelucrare.

### Definition of done

- toate domeniile majore au cel putin un raport operational si unul managerial;
- filtrele presetate sunt consistente intre dashboard si raport.

## 14. Faza I - Backend hardening si documente oficiale

Status: continua pe tot proiectul

### Obiectiv

Sa inchidem partea nevazuta, dar critica, pentru productie.

### Taskuri

1. Endpointuri reale pentru toate fluxurile nou adaugate.
2. Validari de business centralizate in service layer.
3. Audit logging pentru actiuni sensibile.
4. PDF-uri oficiale dedicate pe fiecare flux unde este nevoie.
5. Performanta:
   - query-uri agregate;
   - indexuri noi;
   - compunere eficienta a dashboardurilor.
6. Consistenta naming pentru fisiere generate.
7. Joburi sau notificari pentru termene si restante, daca se decide activarea lor.

### Definition of done

- fluxurile importante nu se bazeaza pe validari superficiale in UI;
- documentele generate au statut oficial clar si naming consistent.

## 15. Faza J - Testare si operationalizare

Status: pornita bine pe e2e frontend

### Obiectiv

Sa acoperim sistemul pe straturi si sa facem reluarea implementarii sigura.

### Taskuri

1. Backend:
   - teste pe servicii;
   - teste pe handlers;
   - teste pe permisiuni contextuale.
2. Frontend:
   - unit tests unde exista logica locala relevanta;
   - e2e pentru fiecare wizard nou;
   - e2e pentru dashboarduri pe rol.
3. Contracte:
   - payloaduri stabile;
   - mapping clar al exporturilor;
   - scenarii negative.
4. Operationalizare locala:
   - rulare locala backend + frontend;
   - seed minim pentru demo;
   - ghid clar de reluare.

### Definition of done

- fiecare verticala noua vine cu testele ei;
- regresiile se prind devreme;
- rularea locala este predictibila.

## 16. Ordinea recomandata pe sprinturi

### Sprint 1

- finalizare matrice roluri si RBAC contextual;
- hardening guvernanta CA;
- inchidere documente finale CA;
- rapoarte standard guvernanta.

### Sprint 2

- flux CP complet;
- dashboard secretariat minim;
- rapoarte de sedinte si procese-verbale.

### Sprint 3

- documente manageriale si conformitate;
- publicare si requirement mapping;
- dashboard conformitate pentru conducere.

### Sprint 4

- dosar personal procedural;
- rapoarte personal;
- audit acces sensibil.

### Sprint 5

- hardening evaluari, mobilitate, merit;
- dashboard evaluator sau HR;
- exporturi specializate.

### Sprint 6

- elevi, clase, diriginte - fundatie;
- dosar elev;
- transferuri si situatii de baza.

### Sprint 7

- absente, sanctiuni, recompense, risc educational;
- dashboard diriginte;
- rapoarte pe clasa.

### Sprint 8

- pachet inspectie;
- dashboard inspector;
- ultimele rapoarte institutionale.

### Sprint 9

- polishing transversal;
- performanta;
- acoperire teste;
- inchidere restante UX.

## 17. Backlog imediat recomandat

Daca incepem chiar acum, ordinea mea recomandata este:

1. actualizare [docs/rbac-matrix.md](/E:/dev/egueducation/docs/rbac-matrix.md) cu rolurile si actiunile educationale finale;
2. audit tehnic scurt pe endpointurile deja existente vs permisiuni contextuale;
3. completare CA pana la document final oficial;
4. implementare CP complet;
5. dashboard secretariat;
6. documente manageriale si conformitate;
7. dosar personal procedural.

## 18. Criteriu de inchidere pentru intregul program

Putem spune ca modulul de management educational este "complet operational" cand:

- toate rolurile majore au dashboard dedicat;
- toate verticale critice au wizard-uri sau ecrane procedurale;
- documentele oficiale majore se genereaza din sistem;
- rapoartele standard acopera operarea si managementul scolii;
- actiunile sensibile sunt protejate contextual;
- exista acoperire de teste suficienta pentru reluarea in siguranta.

## 19. Concluzie

Nu mai suntem in etapa in care trebuie definita directia generala.

Suntem in etapa in care trebuie:

- sa inchidem verticalele incepute;
- sa adaugam verticalele lipsa mari;
- sa conectam rolurile, dashboardurile, rapoartele si fluxurile;
- sa transformam fundatia existenta intr-un sistem operational complet pentru scoala.
