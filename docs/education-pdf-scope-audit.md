# Audit strict pe scope-ul derivat din cele 2 PDF-uri

Data: 2026-06-24

Acest audit acopera doar scope-ul ramas agreat pentru cele 2 documente-sursa, fara extensii de produs.

Surse de referinta:

- [docs/education-pdf-only-scope.md](/E:/dev/egueducation/docs/education-pdf-only-scope.md)
- [docs/education-implementation-audit.md](/E:/dev/egueducation/docs/education-implementation-audit.md)
- [docs/education-management-specifications.md](/E:/dev/egueducation/docs/education-management-specifications.md)

## 1. Legenda status

- `acoperit` = exista deja implementare suficienta pentru a considera punctul functional, chiar daca mai poate fi polisat
- `partial` = exista fundatie clara, dar lipsesc piese importante pentru a considera punctul inchis
- `lipsa` = nu exista implementare suficienta sau nu exista deloc

## 2. Tabelul canonic de audit

| # | Cerinta ramasa | Status | Ce exista deja | Ce lipseste exact |
| --- | --- | --- | --- | --- |
| 1 | finalizarea completa a portofoliului profesional | `partial` | Exista registru portofolii, sectiuni, documente, checklist, opis, custodie, transfer, review si PDF principal in [backend/internal/education/governance_portfolio_flows.go](/E:/dev/egueducation/backend/internal/education/governance_portfolio_flows.go), [backend/internal/education/procedural_flows.go](/E:/dev/egueducation/backend/internal/education/procedural_flows.go), [backend/internal/education/portfolio_documents.go](/E:/dev/egueducation/backend/internal/education/portfolio_documents.go) si configurare UI in [frontend/src/app/features/education/shared/education-config.ts](/E:/dev/egueducation/frontend/src/app/features/education/shared/education-config.ts) | Lipseste flux procedural complet de depunere/verificare/returnare/validare, lipseste UX dedicat pentru profesor, lipsesc reguli institutionale vizibile in UI |
| 2 | opis + indexare + datare | `partial` | Exista subresursa `education_portfolio_opis`, PDF sumar opis si model de date cu index cronologic, referinta document, verificare si includere in transfer; in plus, opisul se regenereaza acum automat din `education_portfolio_documents` la create/update/delete si exista endpoint explicit de regenerare | Mai lipseste expunerea clara in UI a mecanismului de regenerare/indexare si o prezentare mai vizibila a datarii automate pentru operator |
| 3 | delimitarea completa portofoliu vs dosar personal | `partial` | Exista modele separate pentru portofolii si pentru dosarul/personal file, plus sincronizare a unor documente administrative de evaluare in dosarul personal; vezi [backend/internal/education/personnel_detail_flows.go](/E:/dev/egueducation/backend/internal/education/personnel_detail_flows.go), [backend/internal/education/evaluation_detail_flows.go](/E:/dev/egueducation/backend/internal/education/evaluation_result_issue_flows.go); in plus, exista acum un sumar consolidat `portofoliu - dosar personal` pe fisa cadrului didactic, cu reguli vizibile, trasabilitate si blocaje de delimitare | Mai lipsesc politici institutionale configurabile privind dublarea sau separarea documentelor si automatizari mai puternice pe corelarea document-cu-document intre cele doua registre |
| 4 | transfer digital si mobilitate | `partial` | Exista portofoliu digital, transferuri, custodie, statusuri de transfer si sumar PDF pentru transferuri in [backend/internal/education/portfolio_documents.go](/E:/dev/egueducation/backend/internal/education/portfolio_documents.go) | Lipseste flux procedural complet pentru transfer intre unitati, lipsesc actiuni explicite de solicitare/transmitere/confirmare intre unitati, lipseste clar UX-ul pentru detasare si unitate de baza vs unitate curenta |
| 5 | fluxurile de valorificare a portofoliului | `partial` | Exista legaturi tehnice cu evaluari, mobilitate, gradatii, review si verificare; modelul permite folosirea portofoliului in mai multe contexte | Lipseste maparea explicita in sistem a tuturor contextelor din metodologie: definitivat/licentiere, grade didactice, inspectii, evaluare externa, dezvoltare profesionala; lipsesc filtre, rapoarte si wizard-uri dedicate pe aceste contexte |
| 6 | ROF / ROI | `partial` | Exista regulamente, versiuni si workflow steps in [backend/internal/education/managerial_regulation_flows.go](/E:/dev/egueducation/backend/internal/education/managerial_regulation_flows.go), plus resurse UI pentru regulamente | Lipseste suport procedural complet specific ROF/ROI, lipsesc sabloane/documente-model dedicate, lipsesc PDF-uri oficiale dedicate si UX de avizare/aprobare/publicare direct legat de ghid |
| 7 | CA complet | `partial` | Exista componenta organisme, sedinte, participanti, documente, voturi, minute, hotarari, sumar procedural, tranzitii de status si PDF pentru minute/hotarari; vezi [backend/internal/education/service.go](/E:/dev/egueducation/backend/internal/education/service.go), [backend/internal/education/governance_documents.go](/E:/dev/egueducation/backend/internal/education/governance_documents.go), [frontend/src/app/features/education/governance/ca-meeting-wizard.component.ts](/E:/dev/egueducation/frontend/src/app/features/education/governance/ca-meeting-wizard.component.ts) | Lipsesc convocatorul CA, procesul-verbal oficial CA complet, registrul oficial CA ca document dedicat, semnaturi/participant list operate procedural, calcul procedural complet de cvorum din componenta reala, componenta CA configurata pe tip de unitate |
| 8 | CP complet | `partial` | Suportul generic de guvernanta foloseste si organismul `cp`, iar infrastructura sedinta/vot/minute poate fi reutilizata | Lipseste verticala CP explicita: wizard CP dedicat, componenta CP tratata procedural, proces-verbal CP dedicat, decizie numire secretar CP, convocator CP, rapoarte CP specifice |
| 9 | comisii si comisia de evaluare | `lipsa` | Exista doar referinte indirecte precum `responsabil_comisie`, `membru_comisie`, `committee_name` in unele modele si assignment-uri | Nu exista modul explicit pentru comisii institutionale, nu exista entitati dedicate pentru comisie/membri/rapoarte, nu exista comisia temporara de evaluare modelata procedural |
| 10 | comunicarea rezultatului evaluării | `acoperit` | Exista subflux dedicat pentru comunicarea rezultatului evaluarii, PDF dedicat si integrare in dosarul personal in [backend/internal/education/evaluation_result_issue_flows.go](/E:/dev/egueducation/backend/internal/education/evaluation_result_issue_flows.go) si [backend/internal/education/evaluation_documents.go](/E:/dev/egueducation/backend/internal/education/evaluation_documents.go), plus UI cu subresursa aferenta | Mai poate fi polisat UX-ul, dar functional cerinta este deja acoperita suficient |
| 11 | portofoliul directorului / directorului adjunct | `partial` | Exista distinctii in dosarul personal pentru `dosar_director` si `dosar_director_adjunct`, plus documente manageriale, personal files si acces sensibil | Lipseste un flux sau registru explicit pentru portofoliul directorului/directorului adjunct ca entitate sau vedere proprie, lipsesc documentele-model si checklist-ul specific managerial din ghid |
| 12 | documente-model si PDF-uri oficiale aferente | `partial` | Exista export PDF generic, PDF-uri pentru portofoliu, evaluari, mobilitate, gradatii si unele documente de guvernanta | Lipsesc multe documente oficiale cerute direct de ghid: proces-verbal CA complet, proces-verbal CP, convocatoare CA/CP, decizie numire secretar CP, documente-model manageriale specifice, eventual grafic/tematica oficiala |
| 13 | rapoarte manageriale strict necesare pentru aceste zone | `partial` | Exista pagina de rapoarte standard, deep links, export CSV/PDF contextual si cockpit director initial; vezi [frontend/src/app/features/education/dashboard/education-standard-reports.component.ts](/E:/dev/egueducation/frontend/src/app/features/education/dashboard/education-standard-reports.component.ts) si [backend/internal/education/cockpit_service.go](/E:/dev/egueducation/backend/internal/education/cockpit_service.go) | Lipsesc rapoarte strict aliniate pe scope-ul PDF: registru sedinte CA/CP complet, registru hotarari, raport portofolii pe stadii metodologice, raport ROF/ROI, raport comisii si evaluare temporara, raport transfer portofolii intre unitati |

## 3. Rezumat executiv pe cele 13 puncte

### Acoperit

- 10. comunicarea rezultatului evaluarii

### Partial

- 1. finalizarea completa a portofoliului profesional
- 2. opis + indexare + datare
- 3. delimitarea completa portofoliu vs dosar personal
- 4. transfer digital si mobilitate
- 5. fluxurile de valorificare a portofoliului
- 6. ROF / ROI
- 7. CA complet
- 8. CP complet
- 11. portofoliul directorului / directorului adjunct
- 12. documente-model si PDF-uri oficiale aferente
- 13. rapoarte manageriale strict necesare

### Lipsa

- 9. comisii si comisia de evaluare

## 4. Ce inseamna asta practic

Concluzia reala este:

- fundatia pentru portofoliu, guvernanta si documente manageriale exista;
- lipsa principala nu este de schema sau CRUD, ci de procedura, document oficial si UX ghidat;
- singurul punct care poate fi considerat substantial inchis acum este comunicarea rezultatului evaluarii;
- cel mai slab acoperit punct din lista stricta a PDF-urilor este zona de comisii/comisia de evaluare.

## 5. Ordinea corecta de executie strict pe acest scope

1. finalizarea portofoliului profesional
2. opis automat + indexare + datare
3. delimitare portofoliu vs dosar personal
4. transfer digital si mobilitate portofoliu
5. CA complet
6. CP complet
7. ROF / ROI
8. documente-model si PDF-uri oficiale
9. comisii si comisia de evaluare
10. portofoliul directorului / directorului adjunct
11. rapoarte manageriale strict necesare
