# Specificatii pentru management educational in EguEducation

Data: 2026-06-24

Surse analizate:

- Proiect de Metodologie-cadru privind elaborarea si gestionarea portofoliului profesional al cadrului didactic: https://www.edu.ro/sites/default/files/Proiect_Metodologie_portofoliu_CD.pdf
- Ghid pentru directorii unitatilor de invatamant in anul scolar 2024-2025, ISJ Timis: https://www.isj.tm.edu.ro/public/data_files/specializari/fisier-6107.pdf

## 1. Scopul documentului

Acest document defineste cerintele functionale si operationale pentru rafinarea modulului `education` din EguEducation, pe baza documentelor normative si metodologice analizate.

Documentul trebuie folosit ca baza pentru:

- definirea completa a rolurilor educationale;
- rafinarea RBAC pe actiuni sensibile;
- construirea fluxurilor ghidate de tip wizard pentru fiecare rol si proces;
- proiectarea cockpitului de management educational pentru director;
- definirea rapoartelor standard;
- completarea modelelor de date, validari, documente generate si UX procedural.

Repo-ul are deja fundatie implementata pentru guvernanta, decizii, conformitate, documente manageriale, personal, evaluari, mobilitate, gradatie de merit si portofolii. Cerintele de mai jos nu pornesc de la zero, ci descriu ce trebuie consolidat pentru ca platforma sa devina un instrument operational coerent pentru managementul scolii.

## 2. Principii produs

### 2.1. Platforma trebuie sa functioneze procedural

Utilizatorii nu trebuie sa lucreze cu tabele generice atunci cand procesul are pasi formali. Pentru procese precum sedinte CA/CP, constituire CA, portofolii, evaluari, mobilitate, documente manageriale, rapoarte si publicari, sistemul trebuie sa ofere wizard-uri cu pasi, validari si documente generate.

### 2.2. Fiecare document trebuie sa aiba trasabilitate

Orice document managerial, hotarare, decizie, raport, convocator, proces-verbal, opis sau document din portofoliu trebuie sa poata fi legat de:

- sursa cerintei sau temeiul legal;
- rolul emitent;
- fluxul in care a fost creat;
- sedinta sau aprobarea aferenta, daca exista;
- numarul de inregistrare;
- versiunea curenta;
- statusul de lucru, aprobare, publicare si arhivare;
- actorii care l-au creat, verificat, aprobat, semnat sau publicat.

### 2.3. Sistemul trebuie sa diferentieze clar portofoliul profesional de dosarul personal

Portofoliul profesional al cadrului didactic este o colectie de dovezi profesionale, materiale didactice, documente de activitate si evolutie profesionala. Dosarul personal este dosarul administrativ gestionat de angajator.

EguEducation trebuie sa trateze cele doua entitati ca distincte, complementare si legabile, cu reguli diferite de acces, arhivare, continut si responsabilitate.

### 2.4. Directorul are nevoie de cockpit, nu doar de registre

Dashboardul directorului trebuie sa ofere o vedere sintetica asupra scolii: sedinte, hotarari, portofolii, evaluari, documente manageriale, comisii, personal, rapoarte, riscuri, termene si conformitate.

### 2.5. Profesorul are nevoie de flux personal simplu

Profesorul trebuie sa vada clar ce are de completat, ce este in verificare, ce lipseste, ce a fost aprobat si ce termene are.

### 2.6. Secretariatul si registratura trebuie sa reduca munca repetitiva

Fluxurile pentru numere de inregistrare, convocatoare, decizii, solicitari, minute, procese-verbale, comunicari si arhivare trebuie sa fie generate din date deja introduse, cu cat mai putina reintroducere manuala.

## 3. Domenii functionale majore

### 3.1. Guvernanta institutionala

Acopera:

- Consiliul de administratie (CA);
- Consiliul profesoral (CP);
- Consiliul clasei;
- comisii permanente si temporare;
- sedinte ordinare si extraordinare;
- convocatoare;
- ordine de zi;
- participanti, observatori si invitati;
- cvorum;
- voturi;
- hotarari;
- procese-verbale;
- anexe;
- registre oficiale.

### 3.2. Management strategic si documente manageriale

Acopera:

- PDI/PAS;
- plan managerial anual;
- planuri operationale;
- rapoarte anuale;
- raport asupra calitatii educatiei;
- RAEI;
- documente de diagnoza;
- documente de prognoza;
- documente de evidenta;
- regulamente interne si ROF;
- programul de control intern managerial;
- publicarea documentelor de interes public.

### 3.3. Personal si roluri institutionale

Acopera:

- director;
- director adjunct;
- profesor;
- profesor diriginte;
- secretar/secretar-sef;
- registrator;
- contabil/administrator financiar;
- administrator patrimoniu;
- responsabil comisie;
- membru CA;
- membru CP;
- membru consiliu clasa;
- inspector;
- observator;
- invitat;
- parinte/reprezentant legal, unde este relevant;
- elev major sau reprezentant al elevilor, unde este relevant.

### 3.4. Portofoliul profesional al cadrului didactic

Acopera:

- elaborare initiala;
- actualizare;
- opis;
- organizare pe sectiuni;
- incarcari documente;
- verificare;
- custodie;
- transfer intre unitati;
- solicitari de acces;
- eliberare la cerere;
- arhivare 3 ani dupa incetarea activitatii;
- consimtamant GDPR si declaratie pe propria raspundere.

### 3.5. Evaluarea personalului

Acopera:

- autoevaluare;
- fisa de evaluare;
- criterii si punctaje;
- validare in CP, unde este cazul;
- acordare calificativ;
- contestatii;
- comunicare rezultat;
- adeverinte pentru personal care functioneaza in mai multe unitati;
- introducerea documentelor administrative in dosarul personal.

### 3.6. Conformitate, publicare si arhivare

Acopera:

- cerinte legale si metodologice;
- maparea cerinta-document-flux;
- verificari periodice;
- documente publice;
- documente confidentiale;
- jurnale de acces;
- arhiva digitala;
- retentie;
- exporturi.

## 4. Roluri si responsabilitati

### 4.1. Director

Directorul este actorul principal pentru guvernanta, decizii, validari, publicari si cockpitul managerial.

Permisiuni principale:

- configureaza anul scolar operational;
- initiaza si aproba calendare CA/CP;
- initiaza sedinte CA/CP;
- propune ordinea de zi;
- convoaca sau deleaga convocarea;
- verifica cvorum;
- conduce voturi;
- emite hotarari CA;
- emite decizii;
- aproba documente manageriale;
- aproba planul managerial;
- aproba procedura interna pentru portofolii;
- desemneaza responsabili;
- verifica stadiul portofoliilor;
- acceseaza rapoarte agregate;
- vede riscuri si termene;
- gestioneaza publicarea documentelor de interes public;
- asigura introducerea in dosarul personal a documentelor administrative rezultate din evaluare sau formare.

Dashboard director:

- stadiu sedinte CA/CP;
- hotarari in asteptare;
- documente manageriale in lucru;
- portofolii incomplete;
- evaluari in termen, intarziate sau contestate;
- comisii fara componenta completa;
- documente care trebuie publicate;
- termene calendaristice importante;
- indicatori de participare, cvorum si vot;
- indicatori privind calitatea educatiei.

### 4.2. Director adjunct

Directorul adjunct are flux propriu de portofoliu managerial si poate primi delegari operationale.

Permisiuni principale:

- gestioneaza propriul plan managerial in concordanta cu PDI/PAS si planul directorului;
- gestioneaza documente privind activitatea educativă;
- inregistreaza asistente la ore;
- urmareste graficul serviciului pe scoala;
- gestioneaza situatii nominale privind sanctiuni disciplinare elevi;
- gestioneaza situatii privind insertia absolventilor sau continuarea studiilor;
- poate pregati documente pentru CA/CP, daca este delegat;
- poate verifica fluxuri operationale pe domeniile alocate.

### 4.3. Profesor

Profesorul este responsabil de portofoliul profesional si de participarea la organismele unde are obligatii.

Permisiuni principale:

- creeaza si actualizeaza portofoliul profesional;
- incarca documente si materiale;
- mentine opisul;
- completeaza autoevaluari;
- primeste notificari privind termene si lipsuri;
- participa la CP;
- voteaza in CP, unde este membru cu drept de vot;
- participa la CA, consiliul clasei sau comisii daca este desemnat;
- poate solicita eliberarea portofoliului profesional;
- poate raspunde solicitarilor de clarificare.

Restrictii:

- nu poate modifica documente institutionale aprobate;
- nu poate vedea portofoliile altor profesori, exceptand roluri desemnate explicit;
- nu poate accesa dosare personale ale altor angajati.

### 4.4. Profesor diriginte / invatator / profesor pentru invatamant primar / prescolar

Rol specializat pentru consiliul clasei si managementul clasei.

Permisiuni principale:

- initiaza sau pregateste documente pentru consiliul clasei;
- gestioneaza componenta clasei din perspectiva consiliului clasei;
- consemneaza sau propune masuri privind progresul elevilor;
- gestioneaza documente de management al clasei in portofoliul propriu;
- contribuie la rapoarte privind rezultate, absente, comportament, risc educational si relatie cu parintii.

### 4.5. Secretar / secretar-sef

Secretariatul gestioneaza documente oficiale, registre, convocari si siguranta unor documente scolare.

Permisiuni principale:

- aloca sau propune numere de inregistrare;
- gestioneaza convocatoare;
- gestioneaza registre de procese-verbale;
- genereaza procese-verbale pe baza datelor din sedinte;
- gestioneaza lista de participanti si semnaturi;
- pastreaza evidenta cataloagelor si condicilor, daca modulul este extins in aceasta directie;
- gestioneaza solicitari catre primarie, consiliu local, parinti, operatori economici si alte entitati pentru constituirea CA;
- gestioneaza comunicari oficiale;
- ajuta la arhivare.

### 4.6. Registrator

Registratorul gestioneaza circuitul formal al documentelor.

Permisiuni principale:

- inregistreaza documente primite si emise;
- conecteaza documentele de intrare la sedinte, decizii, fluxuri sau dosare;
- urmareste statusul documentelor;
- marcheaza documente pentru publicare sau arhivare;
- exporta registre.

### 4.7. Inspector

Inspectorul are rol de control si verificare, cu acces limitat la contexte justificate.

Permisiuni principale:

- vizualizeaza rapoarte institutionale;
- verifica documente de guvernanta;
- verifica portofolii in contexte de inspectie;
- verifica trasabilitatea hotararilor, deciziilor si documentelor manageriale;
- exporta pachete de inspectie;
- nu modifica date operationale, exceptand observatii sau rapoarte de control, daca modulul prevede explicit.

### 4.8. Responsabil comisie

Permisiuni principale:

- gestioneaza componenta comisiei;
- elaboreaza planuri si rapoarte ale comisiei;
- transmite rapoarte catre conducere;
- pregateste documente pentru CA/CP;
- urmareste sarcini, termene si indicatori.

### 4.9. Membru CA

Permisiuni principale:

- vede convocatorul si ordinea de zi;
- vede documentele anexate sedintei;
- confirma participarea;
- voteaza;
- poate formula observatii;
- semneaza procesul-verbal sau confirma participarea, in functie de semnatura folosita.

### 4.10. Membru CP

Permisiuni principale:

- vede convocatorul CP;
- vede ordinea de zi si materialele;
- confirma participarea;
- voteaza cand punctul de pe ordinea de zi necesita vot;
- poate formula interventii;
- semneaza lista de prezenta.

### 4.11. Observator si invitat

Permisiuni principale:

- primeste acces doar la sedinta si documentele explicit marcate pentru rolul sau;
- poate avea interventii consemnate;
- nu voteaza, daca nu are drept de vot conform componentei organismului.

## 5. Cerinte pentru portofoliul profesional al cadrului didactic

### 5.1. Entitate principala

`TeacherProfessionalPortfolio`

Campuri minime:

- id;
- tenant_id;
- school_year_id;
- teacher_user_id;
- employment_unit_id;
- base_unit_flag;
- teacher_status: titular, viabilitate_post, anual, detasat, mai_multe_unitati, pensionar, asociat;
- submission_unit_id;
- format: fizic, digital, mixt;
- status: draft, in_completare, depus, in_verificare, verificat, returnat_pentru_completari, arhivat, eliberat;
- initial_created_at;
- last_updated_at;
- submitted_at;
- verified_at;
- archived_at;
- retention_until;
- custodian_role_id;
- custodian_user_id;
- consent_status;
- declaration_status;
- transfer_status.

### 5.2. Sectiuni obligatorii

Portofoliul trebuie sa fie structurat in cel putin urmatoarele sectiuni:

1. Date personale si de identificare profesionala.
2. Activitate specifica normei didactice de predare-invatare-evaluare.
3. Activitati complementare procesului de invatamant.
4. Activitati de management al clasei.
5. Evolutia in cariera didactica si dezvoltarea profesionala.

Fiecare sectiune trebuie sa permita:

- documente;
- materiale;
- linkuri catre resurse digitale;
- metadate;
- data dobandirii documentului;
- data incarcarii;
- status verificare;
- observatii;
- marcaj daca documentul este relevant si pentru dosarul personal.

### 5.3. Opisul portofoliului

Opisul trebuie sa poata fi generat automat din documentele incarcate.

Campuri opis:

- numar curent;
- sectiune;
- titlu document/material;
- tip document;
- data documentului;
- data incarcarii;
- sursa;
- link intern;
- status;
- observatii;
- data ultimei actualizari.

Reguli:

- opisul se actualizeaza la fiecare adaugare, modificare, eliminare sau transfer;
- documentele sunt ordonate cronologic in cadrul fiecarei sectiuni;
- sistemul trebuie sa pastreze istoric pentru versiuni si eliminari;
- pentru portofoliul digital, opisul poate fi inlocuit operational de indexare automata, dar trebuie sa existe export PDF/CSV al opisului.

### 5.4. Flux wizard - portofoliu profesor

Pasi:

1. Confirmare date profesionale.
2. Selectare unitate de depunere si statut curent.
3. Confirmare format portofoliu: digital, fizic, mixt.
4. Completare sectiuni portofoliu.
5. Incarcare documente si materiale.
6. Generare opis.
7. Confirmare consimtamant GDPR.
8. Declaratie pe propria raspundere privind autenticitatea documentelor.
9. Depunere.
10. Verificare institutionala.
11. Corectii, daca este cazul.
12. Validare/confirmare custodie.

Validari:

- nu se permite depunerea fara consimtamant si declaratie;
- nu se permite depunerea fara sectiunile minime create;
- pentru fiecare document trebuie completata data dobandirii sau data relevanta;
- pentru personal in mai multe unitati se cere marcarea unitatii de depunere si notificarea celorlalte unitati;
- pentru detasati se permite solicitarea portofoliului digital de catre unitatea de baza;
- pentru pensionari si asociati sistemul trebuie sa poata marca exceptia de la obligatia depunerii.

### 5.5. Flux wizard - procedura interna pentru portofolii

Actor principal: director.

Pasi:

1. Configurare termen intern de elaborare/actualizare.
2. Alegere modalitati de depunere: fizic, digital, mixt.
3. Definire formate digitale acceptate.
4. Definire persoane cu acces.
5. Definire custode si responsabili.
6. Definire calendare intermediare.
7. Definire integrare cu platforme digitale institutionale.
8. Definire reguli de arhivare si transfer.
9. Generare procedura.
10. Aprobare in CA.
11. Publicare interna.

### 5.6. Portofoliu si dosar personal

Cerinta principala: sistemul trebuie sa permita marcarea documentelor care sunt:

- doar in portofoliu;
- doar in dosarul personal;
- in ambele, prin copie sau referinta;
- document administrativ rezultat din evaluare/formare;
- material didactic sau dovada profesionala.

Reguli:

- fisa de evaluare semnata si comunicarea rezultatului se introduc in dosarul personal;
- diplomele/certificatele/adeverintele de credite pot exista in portofoliu si copii in dosarul personal;
- materialele didactice, proiectele de lectie, instrumentele de evaluare raman in portofoliu;
- decizia CA/CP privind introducerea documentelor obligatorii si in portofoliu trebuie configurabila la nivelul unitatii.

## 6. Cerinte pentru Consiliul de administratie

### 6.1. Componenta CA

Sistemul trebuie sa permita configurarea CA in functie de tipul unitatii:

- unitate standard;
- unitate cu invatamant tehnologic;
- unitate exclusiv tehnologica;
- liceu agricol/silvic;
- unitate vocationala teologica;
- unitate postliceala;
- unitate cu elevi majori reprezentati.

Campuri membru CA:

- persoana;
- rol in CA;
- sursa desemnarii: director, cadre didactice, primar, consiliu local, parinti, elevi, operator economic, cult, minister etc.;
- drept de vot;
- statut: membru, observator, invitat;
- mandat_start;
- mandat_end;
- document desemnare;
- inlocuitor, daca exista;
- activ/inactiv.

### 6.2. Calcul cvorum si prag vot

Sistemul trebuie sa calculeze automat:

- jumatate plus unu din numarul total de membri;
- doua treimi din numarul total de membri;
- o treime din numarul total de membri;
- cvorum minim de sedinta;
- voturi minime pentru adoptare in functie de tipul hotararii.

Cerinta UX:

- in sedinta, directorul/secretarul vede in timp real daca sedinta are cvorum;
- la fiecare punct de vot se afiseaza pragul necesar;
- sistemul blocheaza adoptarea hotararii daca pragul nu este indeplinit;
- sistemul permite consemnarea motivelor si amanarea punctului.

### 6.3. Flux wizard - constituire CA

Pasi:

1. Selectare an scolar.
2. Selectare tip unitate.
3. Calcul structura CA.
4. Generare hotarare privind declansarea procedurii.
5. Generare solicitari catre primar, consiliu local, consiliul reprezentativ al parintilor, operatori economici sau alte entitati relevante.
6. Inregistrare raspunsuri si desemnari.
7. Alegere reprezentanti cadre didactice in CP, daca este cazul.
8. Verificare completitudine.
9. Generare decizie constituire CA.
10. Aprobare/publicare interna.

Documente generate:

- HCA declansare constituire;
- solicitare desemnare reprezentant primar;
- solicitare desemnare reprezentant consiliu local;
- solicitare desemnare reprezentanti parinti;
- solicitare desemnare reprezentanti operatori economici;
- decizie constituire CA;
- lista componenta CA;
- atributii membri CA.

### 6.4. Flux wizard - sedinta CA

Pasi:

1. Alegere tip sedinta: ordinara/extraordinara.
2. Alegere data, ora, locatie sau link online.
3. Selectare tematica din grafic sau introducere puncte noi.
4. Atasare documente suport.
5. Generare convocator.
6. Transmitere convocator.
7. Confirmare participare.
8. Deschidere sedinta.
9. Verificare cvorum.
10. Aprobare ordine de zi.
11. Parcurgere puncte de pe ordinea de zi.
12. Consemnare interventii.
13. Vot pe fiecare punct.
14. Generare hotarari CA.
15. Generare proces-verbal.
16. Semnare/confirmare participanti.
17. Inchidere sedinta.
18. Publicare sau arhivare documente.

Date obligatorii in proces-verbal:

- data sedintei;
- componenta CA;
- membri prezenti nominal;
- membri absenti;
- invitati si observatori;
- ordine de zi;
- vot pentru ordinea de zi;
- documente discutate si numere de inregistrare unde exista;
- interventii;
- rezultat vot pentru fiecare punct;
- nume membri care au votat impotriva sau s-au abtinut;
- anexe;
- lista participanti;
- semnaturi/confirmari;
- mentiune privind absentele.

### 6.5. Tematica standard CA

Sistemul trebuie sa ofere sablon configurabil pentru graficul si tematica sedintelor ordinare CA.

Teme recurente:

- alegerea secretarului CA;
- declansarea constituirii CA;
- acordarea punctajelor si calificativelor;
- aprobare tematica sedinte ordinare;
- aprobare documente manageriale;
- aprobare rapoarte;
- aprobare/actualizare PDI/PAS;
- aprobare plan managerial;
- aprobare ROF/ROI;
- aprobare comisii;
- avizare plan de scolarizare;
- aprobari privind incadrare si mobilitate;
- aprobari privind CDȘ/CDL/CDEOS;
- masuri privind absenteism, violenta, abandon scolar;
- buget, achizitii, patrimoniu, investitii;
- rapoarte CEAC/RAEI;
- situatii curente si diverse.

## 7. Cerinte pentru Consiliul profesoral

### 7.1. Componenta CP

CP este alcatuit din totalitatea cadrelor didactice din unitate. Directorul este presedinte.

Campuri:

- membru;
- norma de baza in unitate;
- drept de vot;
- participare obligatorie;
- statut participare: prezent, absent motivat, absent nemotivat;
- unitati suplimentare unde activeaza.

Reguli:

- CP se intruneste lunar sau ori de cate ori este nevoie;
- poate fi convocat de director sau la solicitarea a minimum o treime din cadrele didactice;
- cvorumul este de doua treimi din numarul total al membrilor cu norma de baza;
- hotararile se adopta cu cel putin jumatate plus unu din numarul total al membrilor cu norma de baza;
- absenta nemotivata de la CP in unitatea unde cadrul didactic are norma de baza trebuie marcata ca risc disciplinar.

### 7.2. Documente CP

Sistemul trebuie sa gestioneze:

- tematica si graficul sedintelor CP;
- convocatoare si dovezi ale convocarii;
- registrul de procese-verbale;
- dosarul cu anexele proceselor-verbale;
- hotarari sau aprobari CP;
- decizie numire secretar CP.

### 7.3. Flux wizard - sedinta CP

Pasi:

1. Alegere sedinta ordinara/extraordinara.
2. Selectare tematica din grafic.
3. Generare convocator.
4. Atasare documente.
5. Confirmare participanti.
6. Verificare cvorum.
7. Aprobare ordine de zi.
8. Parcurgere puncte.
9. Consemnare interventii.
10. Vot pentru punctele decizionale.
11. Marcare puncte informative fara vot.
12. Generare proces-verbal.
13. Generare anexe.
14. Semnare.
15. Inchidere si arhivare.

Date obligatorii:

- prezenta si cvorum;
- absenti si motive;
- ordine de zi;
- voturi pentru/impotriva/abtineri;
- nume voturi impotriva si abtineri;
- interventii ale observatorilor;
- semnaturi;
- anexe numerotate.

### 7.4. Tematica standard CP

Sistemul trebuie sa ofere sablon configurabil pentru:

- alegerea secretarului CP;
- validarea fiselor de autoevaluare;
- alegerea coordonatorului pentru proiecte si programe educative;
- alegerea consilierului de etica;
- prezentarea dirigintilor;
- constituirea comisiei pentru curriculum;
- completarea documentelor scolare;
- alegerea reprezentantilor CP in CA;
- analiza raportului privind calitatea educatiei;
- dezbaterea si avizarea ROF/ROI;
- actualizarea PDI;
- activitati extrascolare si extracurriculare;
- programe de formare continua;
- alegerea reprezentantilor CP in CEAC;
- analiza RAEI;
- rezultate testari initiale;
- masuri pentru examene nationale;
- plan de scolarizare;
- mobilitatea personalului didactic;
- prevenirea violentei;
- prevenirea abandonului;
- incadrare personal didactic;
- CDȘ/CDL/CDEOS;
- analiza situatiei scolare pe module.

## 8. Cerinte pentru Consiliul clasei

### 8.1. Componenta

Consiliul clasei include:

- cadrele didactice care predau la clasa;
- cel putin un parinte delegat, cu exceptii pentru postliceal;
- reprezentantul elevilor, pentru nivelurile unde este aplicabil;
- presedinte: invatator/institutor/profesor invatamant primar/profesor diriginte.

### 8.2. Functionalitati

Sistemul trebuie sa permita:

- definirea consiliului clasei pentru fiecare clasa;
- planificarea sedintelor;
- gestionarea prezentei;
- documentarea problemelor clasei;
- masuri privind progresul elevilor;
- masuri privind disciplina, absenteism, risc educational;
- export raport consiliul clasei;
- legatura cu portofoliul dirigintelui.

### 8.3. Flux wizard - consiliul clasei

Pasi:

1. Selectare clasa si an scolar.
2. Verificare componenta.
3. Stabilire ordine de zi.
4. Atasare rapoarte: rezultate, absente, comportament, risc.
5. Sedinta si consemnare discutii.
6. Masuri stabilite si responsabili.
7. Termene.
8. Generare proces-verbal/raport.
9. Urmarire masuri.

## 9. Cerinte pentru documente manageriale

### 9.1. Tipuri de documente

Sistemul trebuie sa gestioneze cel putin:

Documente de diagnoza:

- rapoartele anuale ale comisiilor si compartimentelor;
- raportul anual asupra calitatii educatiei;
- RAEI;
- alte rapoarte dedicate unor domenii specifice.

Documente de prognoza:

- PDI;
- PAS, pentru invatamant profesional si tehnic;
- plan managerial;
- plan operational;
- program de dezvoltare a sistemului de control intern managerial;
- alte documente de prognoza.

Documente de evidenta:

- stat de functii;
- organigrama;
- schema orara/program zilnic;
- plan de scolarizare.

Regulamente si proceduri:

- ROF;
- ROI/regulament intern;
- procedura portofolii;
- proceduri de control intern managerial;
- proceduri operationale pe fluxuri.

### 9.2. Flux wizard - document managerial

Pasi:

1. Selectare tip document.
2. Selectare an scolar si perioada.
3. Selectare surse de diagnoza sau documente anterioare.
4. Completare structura ghidata.
5. Atasare anexe.
6. Revizuire interna.
7. Avizare CP, daca este necesar.
8. Aprobare CA, daca este necesar.
9. Inregistrare.
10. Publicare interna/externa, daca documentul este public.
11. Arhivare.

Statusuri:

- draft;
- in lucru;
- in revizie;
- avizat;
- respins la avizare;
- aprobat;
- publicat;
- arhivat;
- inlocuit de versiune noua.

### 9.3. Structura PDI/PAS

PDI/PAS trebuie sa permita completarea:

- diagnoza mediului intern si extern;
- context local;
- resurse umane;
- resurse materiale;
- rezultate educationale;
- analiza SWOT;
- analiza PEST/PESTEL;
- relatia cu comunitatea;
- misiune;
- viziune;
- valori;
- tinte strategice;
- obiective/optiuni strategice;
- modalitati de realizare;
- planificare activitati;
- termene;
- responsabili;
- resurse;
- indicatori de performanta;
- stadii de realizare;
- evaluare.

### 9.4. Structura raport anual asupra calitatii educatiei

Raportul trebuie sa permita capitole pentru:

- misiune, viziune, valori;
- istoric scurt;
- SWOT;
- PEST/PESTEL;
- prioritati si tinte strategice;
- resurse umane si materiale;
- evolutia efectivelor de elevi;
- cadre didactice;
- incidente de violenta;
- rezultate la olimpiade si concursuri;
- infrastructura;
- rezultate evaluari nationale;
- rata absolvire;
- rezultate bacalaureat;
- rezultate postliceale;
- formare continua personal;
- activitate instructiv-educativa;
- programe internationale;
- activitate educativa;
- parteneriate;
- comunicare institutionala;
- servicii financiar-contabil, secretariat, administrativ;
- indicatori europeni;
- aspecte pozitive;
- aspecte negative;
- concluzii si directii de imbunatatire.

## 10. Cerinte pentru evaluarea personalului

### 10.1. Evaluare didactica

Functionalitati:

- autoevaluare profesor;
- criterii configurabile;
- punctaje pe domenii;
- fisa evaluare;
- validare;
- calificativ anual/partial;
- contestatie;
- comunicare rezultat;
- generare documente PDF;
- export Excel;
- raport agregat pe unitate.

Regula pentru personal in mai multe unitati:

- fiecare unitate cu personalitate juridica realizeaza evaluarea pentru activitatea din unitate;
- cadrul didactic completeaza fisa in fiecare unitate;
- unitatea elibereaza adeverinta cu punctajul pe fiecare domeniu;
- unitatea unde este functia de baza calculeaza media pe domenii si acorda calificativul final.

### 10.2. Evaluare personal administrativ

Functionalitati viitoare:

- evaluare pe an calendaristic;
- perioada implicita 1-31 ianuarie;
- fisa evaluare personal administrativ;
- calificativ;
- comunicare rezultat;
- arhivare in dosar personal.

## 11. Cerinte pentru secretariat, registratura si documente oficiale

### 11.1. Registratura

Functionalitati:

- registru intrari;
- registru iesiri;
- alocare numar;
- documente interne generate dupa aprobari;
- legatura document-sedinta;
- legatura document-dosar personal;
- legatura document-portofoliu;
- legatura document-flux managerial;
- scanare/incarcare document;
- status: primit, in lucru, raspuns, inchis, arhivat.

### 11.2. Modele de documente

Sistemul trebuie sa genereze:

- decizie constituire CA;
- decizie constituire CP;
- decizie numire secretar CP;
- hotarare declansare constituire CA;
- solicitari desemnare membri CA;
- atributii membri CA;
- proces-verbal CA;
- proces-verbal CP;
- grafic tematica CA;
- grafic tematica CP;
- convocator CA;
- convocator CP;
- lista prezenta;
- comunicari rezultate evaluare;
- adeverinte punctaj evaluare;
- opis portofoliu;
- declaratie portofoliu si consimtamant.

## 12. Cockpit de management educational

### 12.1. Dashboard director

Widgeturi obligatorii:

- Calendar operational: sedinte CA/CP, termene portofolii, evaluari, rapoarte, publicari.
- Guvernanta: sedinte planificate, sedinte fara cvorum, hotarari nepublicate, procese-verbale nesemnate.
- Portofolii: total cadre didactice, depuse, incomplete, in verificare, returnate, arhivate, exceptate.
- Evaluari: autoevaluari neincepute, in lucru, validate, contestate, finalizate.
- Documente manageriale: PDI/PAS, plan managerial, RAEI, raport calitate, ROF/ROI, status aprobare.
- Comisii: comisii constituite, comisii fara raport, rapoarte restante.
- Personal: incadrari, functii, roluri, documente lipsa in dosar personal.
- Conformitate: cerinte fara document asociat, documente expirate, documente care trebuie publicate.
- Riscuri educationale: absenteism, violenta, abandon, rezultate scazute, unde exista date.
- Indicatori institutionali: elevi, clase, cadre didactice, participare, rezultate.

### 12.2. Dashboard profesor

Widgeturi:

- portofoliu personal;
- documente lipsa;
- autoevaluare;
- sedinte CP viitoare;
- sedinte consiliu clasa;
- sarcini din comisii;
- notificari si termene;
- documente returnate pentru completari.

### 12.3. Dashboard secretariat/registratura

Widgeturi:

- documente de inregistrat;
- documente generate fara numar;
- convocatoare de trimis;
- procese-verbale in lucru;
- documente de arhivat;
- solicitari externe fara raspuns;
- termene de raspuns.

### 12.4. Dashboard inspector

Widgeturi:

- unitati verificate;
- pachete de inspectie;
- documente lipsa;
- sedinte si hotarari;
- portofolii disponibile;
- rapoarte institutionale;
- observatii deschise.

## 13. Rapoarte standard

### 13.1. Rapoarte guvernanta

- Registru sedinte CA.
- Registru sedinte CP.
- Registru hotarari CA.
- Raport cvorum si participare CA.
- Raport cvorum si participare CP.
- Raport voturi pe hotarari.
- Raport sedinte fara documente complete.
- Raport procese-verbale nesemnate.
- Raport anexe lipsa.

### 13.2. Rapoarte portofolii

- Situatie portofolii pe an scolar.
- Portofolii depuse/incomplete/in verificare/verificate.
- Portofolii cu documente lipsa pe sectiuni.
- Opis portofoliu profesor.
- Portofolii transferate intre unitati.
- Portofolii arhivate si termen retentie.
- Exceptii de depunere: pensionari/asociati.
- Solicitari de eliberare portofoliu.

### 13.3. Rapoarte personal

- Registru personal pe roluri.
- Incadrari si functii.
- Cadre didactice cu norma de baza.
- Cadre didactice in mai multe unitati.
- Dosare personale incomplete.
- Documente administrative rezultate din evaluare/formare.
- Participare CP si absente nemotivate.

### 13.4. Rapoarte evaluari

- Stadiu autoevaluari.
- Punctaje pe domenii.
- Calificative anuale.
- Contestatii.
- Adeverinte punctaj pentru personal in mai multe unitati.
- Evaluari finalizate si comunicate.

### 13.5. Rapoarte documente manageriale

- Stadiu PDI/PAS.
- Stadiu plan managerial.
- Stadiu RAEI.
- Stadiu raport calitate.
- Documente publice publicate/nepublicate.
- Documente in avizare CP.
- Documente in aprobare CA.
- Versiuni documente si istoric.

### 13.6. Rapoarte calitate si indicatori

- Evolutia efectivelor de elevi.
- Rezultate evaluari nationale.
- Rezultate bacalaureat.
- Rezultate olimpiade si concursuri.
- Incidente de violenta.
- Absenteism si abandon.
- Formare continua cadre didactice.
- Parteneriate si relatie comunitate.
- Indicatori europeni relevanti.

### 13.7. Exporturi

Fiecare raport standard trebuie sa poata fi exportat:

- PDF;
- CSV;
- XLSX, pentru rapoarte tabelare importante;
- pachet ZIP pentru inspectie sau arhivare, unde este cazul.

## 14. Notificari si termene

Sistemul trebuie sa permita notificari pentru:

- sedinte CA/CP programate;
- convocatoare trimise;
- confirmare participare;
- documente de sedinta lipsa;
- cvorum insuficient estimat;
- documente manageriale cu termen apropiat;
- portofolii nedepuse;
- portofolii returnate pentru completari;
- autoevaluari nefinalizate;
- contestatii primite;
- documente care trebuie publicate;
- documente care intra in arhivare;
- solicitari externe fara raspuns in termen.

Canale:

- notificari in aplicatie;
- email, daca este configurat;
- export lista sarcini;
- badge-uri in cockpit.

## 15. Securitate, GDPR si audit

### 15.1. Acces diferentiat

Accesul trebuie acordat pe baza:

- rolului;
- unitatii scolare;
- anului scolar;
- apartenentei la organism;
- relatiei cu documentul;
- statusului documentului;
- scopului accesului.

### 15.2. Jurnal de audit

Sistemul trebuie sa pastreze audit pentru:

- vizualizare document sensibil;
- incarcare;
- descarcare;
- modificare metadate;
- schimbare status;
- aprobare;
- respingere;
- publicare;
- arhivare;
- transfer;
- stergere logica;
- export.

### 15.3. Retentie si arhivare

Reguli:

- portofoliul profesional se arhiveaza fizic sau digital timp de 3 ani de la incetarea activitatii cadrului didactic in unitate;
- documentele oficiale trebuie sa ramana in registre si arhive conform procedurilor interne;
- sistemul trebuie sa poata marca documente ca arhivate fara pierderea trasabilitatii;
- eliberarea portofoliului la cererea scrisa a cadrului didactic trebuie inregistrata.

### 15.4. Consimtamant si declaratii

Pentru portofoliu, sistemul trebuie sa includa:

- consimtamant pentru prelucrarea datelor cu caracter personal;
- declaratie pe propria raspundere privind autenticitatea documentelor si apartenenta lor cadrului didactic;
- data si versiunea declaratiei;
- semnatura/confirmare electronica, daca este disponibila.

## 16. Cerinte UX pentru wizard-uri

Fiecare wizard trebuie sa includa:

- indicator clar de progres;
- pasi salvabili;
- validari pe pas;
- rezumat inainte de finalizare;
- documente generate previzualizabile;
- posibilitate de revenire la pas anterior pana la aprobare;
- istoric schimbari;
- actiuni contextuale in functie de rol;
- mesaje in limba romana;
- status vizibil;
- explicatii scurte pentru campurile obligatorii;
- butoane pentru export/printare unde documentul are caracter oficial.

Wizard-uri prioritare:

1. Constituire CA.
2. Sedinta CA.
3. Sedinta CP.
4. Procedura interna portofolii.
5. Portofoliu profesor.
6. Evaluare anuala profesor.
7. Document managerial PDI/PAS.
8. Raport anual asupra calitatii educatiei.
9. RAEI.
10. Constituire comisie si raport comisie.
11. Consiliul clasei.
12. Publicare document oficial.
13. Pachet inspectie.

## 17. Cerinte API si modelare

### 17.1. Entitati recomandate

- `SchoolYear`
- `EducationUnit`
- `EducationRoleAssignment`
- `GovernanceBody`
- `GovernanceBodyMember`
- `GovernanceMeeting`
- `MeetingAgendaItem`
- `MeetingParticipant`
- `MeetingVote`
- `MeetingDecision`
- `MeetingMinutes`
- `MeetingAttachment`
- `ManagerialDocument`
- `ManagerialDocumentVersion`
- `DocumentWorkflowStep`
- `DocumentPublication`
- `TeacherProfessionalPortfolio`
- `PortfolioSection`
- `PortfolioDocument`
- `PortfolioIndexEntry`
- `PortfolioConsent`
- `PortfolioTransfer`
- `PersonnelFile`
- `PersonnelFileDocument`
- `EvaluationCase`
- `EvaluationCriterionScore`
- `EvaluationCommunication`
- `EvaluationAppeal`
- `Committee`
- `CommitteeMember`
- `CommitteeReport`
- `ClassCouncil`
- `ClassCouncilMeeting`
- `ComplianceRequirement`
- `RequirementDocumentMapping`
- `AuditEvent`
- `Notification`
- `ReportDefinition`
- `ReportRun`

### 17.2. Statusuri comune

Status document:

- draft;
- in_lucru;
- in_revizie;
- avizat;
- aprobat;
- respins;
- publicat;
- arhivat;
- retras;
- inlocuit.

Status sedinta:

- planificata;
- convocata;
- in_desfasurare;
- fara_cvorum;
- inchisa;
- proces_verbal_in_lucru;
- finalizata;
- arhivata.

Status portofoliu:

- draft;
- in_completare;
- depus;
- in_verificare;
- returnat_pentru_completari;
- verificat;
- transfer_solicitat;
- transferat;
- arhivat;
- eliberat.

### 17.3. Reguli de consistenta

- O hotarare CA nu poate fi adoptata fara sedinta valida, cvorum si vot suficient.
- Un proces-verbal CA/CP nu poate fi finalizat fara prezenta, ordine de zi si rezultat pe fiecare punct decizional.
- Un document managerial aprobat trebuie sa aiba versiune imutabila.
- Un document publicat trebuie sa aiba referinta la versiunea aprobata.
- Un portofoliu depus trebuie sa aiba consimtamant si declaratie.
- Documentele portofoliului trebuie ordonate cronologic in sectiuni.
- O evaluare finalizata trebuie sa aiba comunicare rezultat.
- Documentele administrative rezultate din evaluare trebuie sa fie legabile la dosarul personal.

## 18. Integrare cu implementarea existenta

Conform `docs/education-current-plan.md`, exista deja fundatie pentru:

- guvernanta: sedinte, participanti, documente sedinta, voturi, minute, apartenente, hotarari;
- decizii si conformitate;
- documente manageriale si workflow;
- personal si dosar personal;
- evaluari;
- mobilitate;
- gradatie de merit;
- portofolii;
- export CSV/PDF;
- RBAC si tenant scoping.

Directia recomandata este:

1. Sa nu se inlocuiasca registrele existente.
2. Sa se adauge ecrane procedurale peste fundatia existenta.
3. Sa se standardizeze statusurile si documentele generate.
4. Sa se adauge cockpit si rapoarte agregate.
5. Sa se rafineze RBAC pe actiune, nu doar pe modul.

Fisiere deja relevante:

- `frontend/src/app/features/education/shared/education-config.ts`
- `frontend/src/app/features/education/shared/education-domain-workspace.component.ts`
- `frontend/src/app/features/education/education-shell.component.ts`
- `backend/internal/education/service.go`
- `backend/internal/education/governance_portfolio_flows.go`
- `backend/internal/education/managerial_regulation_flows.go`
- `backend/internal/education/decision_compliance_flows.go`
- `backend/internal/education/personnel_detail_flows.go`
- `backend/internal/education/personnel_role_disciplinary_flows.go`
- `backend/internal/education/evaluation_documents.go`
- `backend/internal/education/portfolio_documents.go`

## 19. Plan de implementare recomandat

### Faza 1 - Consolidare guvernanta CA/CP

Obiective:

- wizard sedinta CA;
- wizard sedinta CP;
- calcul cvorum si prag vot;
- proces-verbal dedicat;
- hotarare CA dedicata;
- grafic tematica CA/CP;
- convocatoare;
- registru sedinte si hotarari.

Livrabile:

- API pentru status procedural sedinta;
- UI dedicat sedinta;
- PDF-uri CA/CP;
- rapoarte guvernanta.

### Faza 2 - Portofoliu profesional CD complet

Obiective:

- self-service portofoliu profesor;
- procedura interna portofolii;
- opis automat;
- consimtamant si declaratie;
- distinctie portofoliu/dosar personal;
- transfer si arhivare.

Livrabile:

- UI portofoliu tip dosar;
- verificari si checklist;
- PDF opis si declaratii;
- rapoarte portofolii.

### Faza 3 - Cockpit director si notificari

Obiective:

- dashboard director;
- dashboard profesor;
- dashboard secretariat;
- notificari si termene;
- indicatori risc.

Livrabile:

- endpoint-uri agregate;
- widgeturi PrimeNG;
- badge-uri si filtre;
- export cockpit.

### Faza 4 - Documente manageriale si rapoarte calitate

Obiective:

- PDI/PAS wizard;
- raport calitate wizard;
- RAEI workflow;
- publicare documente;
- versiuni aprobate.

Livrabile:

- template-uri documente;
- PDF-uri;
- statusuri document;
- rapoarte manageriale.

### Faza 5 - Personal, evaluari si dosar personal

Obiective:

- evaluari pentru personal in mai multe unitati;
- adeverinte punctaj;
- dosar personal consolidat;
- roluri si incadrari;
- audit acces sensibil.

Livrabile:

- flux evaluare extins;
- raport personal;
- documente administrative;
- RBAC granular.

### Faza 6 - Inspectie, conformitate si arhivare

Obiective:

- pachet inspectie;
- matrice cerinta-document;
- rapoarte lipsuri;
- arhivare digitala;
- retentie.

Livrabile:

- export ZIP/PDF;
- dashboard inspector;
- raport conformitate;
- flux arhivare.

## 20. Backlog initial detaliat

### Backend

- Adaugare/rafinare campuri pentru tip organism, prag cvorum si prag vot.
- Endpoint pentru calcul cvorum si eligibilitate vot.
- Endpoint pentru rezumat sedinta.
- Endpoint pentru cockpit director.
- Endpoint pentru cockpit profesor.
- Endpoint pentru rapoarte standard.
- Generator PDF pentru proces-verbal CA.
- Generator PDF pentru proces-verbal CP.
- Generator PDF pentru hotarare CA.
- Generator PDF pentru opis portofoliu.
- Generator PDF pentru procedura portofolii.
- Generator PDF pentru grafic tematica CA/CP.
- Model pentru consimtamant si declaratie portofoliu.
- Audit pentru acces documente sensibile.
- Reguli de retentie portofoliu.

### Frontend

- Pagina dedicata `education/dashboard/director`.
- Pagina dedicata `education/dashboard/profesor`.
- Pagina dedicata `education/governance/ca/:id/meeting-wizard`.
- Pagina dedicata `education/governance/cp/:id/meeting-wizard`.
- Pagina dedicata `education/portfolio/me`.
- Pagina dedicata `education/portfolio/procedure`.
- Componenta comuna `WizardStepper`.
- Componenta comuna `DocumentPreviewPanel`.
- Componenta comuna `MeetingCvorumPanel`.
- Componenta comuna `VoteResultPanel`.
- Componenta comuna `ReportExportToolbar`.
- Romanizare completa etichete.
- Filtre rapide pe statusuri educationale.

### RBAC

- Permisiuni pe actiuni: `meeting.create`, `meeting.close`, `vote.cast`, `decision.publish`, `portfolio.verify`, `portfolio.transfer`, `document.approve`, `document.publish`, `personnel_file.view_sensitive`, `report.export`.
- Mapare pozitii la roluri educationale extinsa.
- Acces contextual pentru membru CA/CP, nu doar rol global.
- Acces inspector pe pachet de inspectie, cu jurnal.

### Rapoarte

- Registru sedinte CA.
- Registru sedinte CP.
- Registru hotarari.
- Situatie portofolii.
- Situatie evaluari.
- Documente manageriale pe status.
- Conformitate documente obligatorii.
- Participare CP.
- Pachet inspectie.

## 21. Criterii de acceptanta

### Guvernanta

- Directorul poate crea o sedinta CA/CP din wizard, poate convoca participantii, poate verifica cvorumul, poate consemna voturile si poate genera procesul-verbal.
- Sistemul nu permite adoptarea unei hotarari fara prag de vot indeplinit.
- Procesul-verbal include participantii, absentele, ordinea de zi, voturile si anexele.

### Portofoliu

- Profesorul poate depune portofoliul prin wizard.
- Opisul se genereaza automat.
- Documentele sunt organizate cronologic pe sectiuni.
- Depunerea nu este posibila fara consimtamant si declaratie.
- Directorul sau responsabilul poate vedea status agregat pe unitate.

### Documente manageriale

- PDI/PAS si raportul calitatii pot fi create ca documente versionate.
- Documentele pot fi avizate, aprobate, publicate si arhivate.
- Versiunea publicata este imutabila.

### Dashboard

- Directorul vede intr-o singura pagina termene, riscuri, sedinte, documente, portofolii si evaluari.
- Profesorul vede sarcinile personale si stadiul portofoliului/evaluarii.
- Secretariatul vede documentele de inregistrat, procesele-verbale si convocatoarele.

### Audit si securitate

- Orice acces la document sensibil este jurnalizat.
- Exporturile sunt jurnalizate.
- Accesul la portofolii si dosare personale este limitat contextual.

## 22. Riscuri si decizii deschise

Riscuri:

- Diferente intre procedurile proprii ale fiecarei unitati si modelele generale.
- Necesitatea semnaturii electronice pentru validare completa.
- Integrarea cu platforme externe de stocare sau identitate.
- Actualizari legislative ulterioare.
- Complexitate ridicata in calculul componentei CA pentru tipuri speciale de unitati.

Decizii de produs necesare:

- Daca EguEducation va stoca efectiv documentele portofoliului sau doar metadate/linkuri catre stocare institutionala.
- Daca semnatura electronica este obligatorie in MVP sau optionala.
- Care rapoarte XLSX sunt prioritare fata de PDF.
- Daca parintii/elevii vor avea conturi in platforma pentru roluri de CA/consiliul clasei sau vor fi gestionati ca participanti externi.
- Nivelul de integrare cu registratura generala.

## 23. Recomandare de MVP

MVP-ul operational pentru management educational ar trebui sa includa:

1. Dashboard director.
2. Wizard sedinta CA.
3. Wizard sedinta CP.
4. Self-service portofoliu profesor.
5. Opis automat si PDF portofoliu.
6. Proces-verbal CA/CP PDF.
7. Registru hotarari CA.
8. Rapoarte: sedinte, hotarari, portofolii, evaluari, documente manageriale.
9. RBAC contextual pentru director, profesor, secretar, registrator, inspector.
10. Audit pentru documente sensibile si exporturi.

Acest MVP se potriveste cu fundatia deja existenta si produce cel mai rapid castig operational pentru scoala: directorul vede controlul institutional, profesorul are flux clar pentru portofoliu, iar secretariatul reduce munca repetitiva cu documente formale.
