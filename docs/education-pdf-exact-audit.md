# Audit exact pe scope-ul celor 2 PDF-uri

Data: 2026-06-24

Acest document este varianta exacta a auditului pentru scope-ul derivat strict din:

1. `Proiect_Metodologie_portofoliu_CD.pdf`
2. `Ghid pentru directorii unitatilor de invatamant in anul scolar 2024-2025`

Regula de marcare:

- `acoperit` = exista flux suficient pentru a considera cerinta inchisa functional;
- `partial` = exista baza clara, dar lipsesc inca elemente procedurale, documente oficiale sau UX dedicat;
- `lipsa` = nu exista inca o implementare relevanta.

## Rezumat exact

- `acoperit`: 2
- `partial`: 11
- `lipsa`: 0

## 1. Portofoliul profesional al cadrului didactic

Status exact: `partial`

Ce este acoperit:

- registru de portofolii;
- structura pe sectiuni;
- documente, checklist, custodie, transfer, review;
- PDF principal pentru portofoliu;
- sumar de relatie cu dosarul personal;
- ecran self-service pentru profesor, cu flux ghidat de actualizare si trimitere la verificare;
- wizard procedural pentru completare, verificare, revenire si validare.

Unde se vede:

- [backend/internal/education/governance_portfolio_flows.go](E:/dev/egueducation/backend/internal/education/governance_portfolio_flows.go)
- [backend/internal/education/portfolio_documents.go](E:/dev/egueducation/backend/internal/education/portfolio_documents.go)
- [frontend/src/app/features/education/shared/education-config.ts](E:/dev/egueducation/frontend/src/app/features/education/shared/education-config.ts)

Ce mai lipseste exact:

- reguli institutionale vizibile si configurabile in UI.

## 2. Opis + indexare + datare

Status exact: `acoperit`

Ce este acoperit:

- opis dedicat;
- index cronologic;
- referinte document;
- includere in transfer;
- regenerare automata din documentele portofoliului;
- endpoint explicit de regenerare;
- afisare explicita in UI a ultimei datari si a sincronizarii opisului;
- actiune directa in UI pentru regenerarea opisului din sumarul portofoliului.

Unde se vede:

- [backend/internal/education/portfolio_opis_sync.go](E:/dev/egueducation/backend/internal/education/portfolio_opis_sync.go)
- [backend/internal/education/portfolio_documents.go](E:/dev/egueducation/backend/internal/education/portfolio_documents.go)

Ce mai lipseste exact:

- nimic esential in plus pentru cerinta curenta.

## 3. Delimitarea portofoliu vs dosar personal

Status exact: `partial`

Ce este acoperit:

- modele separate pentru portofoliu si dosar personal;
- sumar consolidat de relatie intre cele doua;
- blocaje si trasabilitate;
- sincronizare a unor documente administrative in dosarul personal.

Unde se vede:

- [backend/internal/education/personnel_portfolio_relation_summary.go](E:/dev/egueducation/backend/internal/education/personnel_portfolio_relation_summary.go)
- [backend/internal/education/personnel_detail_flows.go](E:/dev/egueducation/backend/internal/education/personnel_detail_flows.go)
- [frontend/src/app/features/education/shared/education-domain-workspace.component.ts](E:/dev/egueducation/frontend/src/app/features/education/shared/education-domain-workspace.component.ts)

Ce mai lipseste exact:

- politici institutionale configurabile pentru dublare sau separare documente;
- corelare mai puternica document-cu-document intre cele doua registre;
- UX mai clar pentru decizia institutională.

## 4. Transfer digital si mobilitate

Status exact: `partial`

Ce este acoperit:

- portofoliu digital;
- statusuri de transfer;
- custodie;
- sumar PDF pentru transferuri;
- legaturi cu mobilitate.

Unde se vede:

- [backend/internal/education/governance_portfolio_flows.go](E:/dev/egueducation/backend/internal/education/governance_portfolio_flows.go)
- [backend/internal/education/portfolio_documents.go](E:/dev/egueducation/backend/internal/education/portfolio_documents.go)

Ce mai lipseste exact:

- solicitare / transmitere / confirmare intre unitati ca flux procedural explicit;
- UX clar pentru unitatea de baza vs unitatea curenta;
- traseu mai ghidat pentru cazurile de detasare si mobilitate.

## 5. Fluxurile de valorificare a portofoliului

Status exact: `partial`

Ce este acoperit:

- legaturi tehnice cu evaluare, mobilitate, gradatii, review;
- sumar de transfer si valorificare;
- existenta unor trasee de export si analiza.

Unde se vede:

- [backend/internal/education/portfolio_transfer_summary.go](E:/dev/egueducation/backend/internal/education/portfolio_transfer_summary.go)
- [backend/internal/education/governance_portfolio_flows.go](E:/dev/egueducation/backend/internal/education/governance_portfolio_flows.go)

Ce mai lipseste exact:

- maparea explicita a tuturor contextelor din metodologie:
  - debut / licentiere / definitivat;
  - grad didactic II;
  - grad didactic I;
  - inspectii;
  - evaluare externa;
  - dezvoltare profesionala;
- filtre, rapoarte si wizard-uri dedicate pe aceste contexte.

## 6. ROF / ROI

Status exact: `partial`

Ce este acoperit:

- modele de regulament;
- versiuni;
- workflow procedural;
- sumar de pregatire pentru avizare si aprobare.

Unde se vede:

- [backend/internal/education/regulation_procedural_summary.go](E:/dev/egueducation/backend/internal/education/regulation_procedural_summary.go)
- [backend/internal/education/managerial_regulation_flows.go](E:/dev/egueducation/backend/internal/education/managerial_regulation_flows.go)

Ce mai lipseste exact:

- suport procedural complet separat pentru ROF si ROI;
- sabloane sau documente-model dedicate;
- PDF-uri oficiale dedicate pentru circuitul complet.

## 7. CA complet

Status exact: `partial`

Ce este acoperit:

- sedinte CA;
- membri si participanti;
- documente si modele oficiale pentru convocator, proces-verbal si registru;
- voturi;
- minute;
- hotarari;
- sumar procedural;
- tranzitii de status;
- wizard CA.

Unde se vede:

- [backend/internal/education/service.go](E:/dev/egueducation/backend/internal/education/service.go)
- [backend/internal/education/governance_documents.go](E:/dev/egueducation/backend/internal/education/governance_documents.go)
- [frontend/src/app/features/education/governance/ca-meeting-wizard.component.ts](E:/dev/egueducation/frontend/src/app/features/education/governance/ca-meeting-wizard.component.ts)

Ce mai lipseste exact:

- semnaturi / lista participanti operate procedural;
- calcul procedural complet de cvorum din componenta reala;
- model CA mai specific pe tip de unitate.

## 8. CP complet

Status exact: `partial`

Ce este acoperit:

- resource explicit CP;
- listare filtrata CP;
- wizard CP prefabricat;
- documente de sedinta cu PDF, inclusiv modele oficiale explicite pentru CP;
- deschidere directa din tabul CP.

Unde se vede:

- [frontend/src/app/features/education/shared/education-config.ts](E:/dev/egueducation/frontend/src/app/features/education/shared/education-config.ts)
- [frontend/src/app/features/education/governance/ca-meeting-wizard.component.ts](E:/dev/egueducation/frontend/src/app/features/education/governance/ca-meeting-wizard.component.ts)
- [backend/internal/education/meeting_documents_pdf.go](E:/dev/egueducation/backend/internal/education/meeting_documents_pdf.go)

Ce mai lipseste exact:

- rapoarte CP specifice.

## 9. Comisii si comisia de evaluare

Status exact: `partial`

Ce este acoperit:

- modul explicit de comisii;
- membri de comisie;
- detaliu si completitudine;
- comisie de evaluare marcata procedural.

Unde se vede:

- [backend/internal/education/committee_flows.go](E:/dev/egueducation/backend/internal/education/committee_flows.go)
- [frontend/src/app/features/education/shared/education-config.ts](E:/dev/egueducation/frontend/src/app/features/education/shared/education-config.ts)
- [frontend/src/app/features/education/shared/education-domain-workspace.component.ts](E:/dev/egueducation/frontend/src/app/features/education/shared/education-domain-workspace.component.ts)

Ce mai lipseste exact:

- flux procedural complet de constituire si mandat;
- documente oficiale sau PDF-uri dedicate comisiei de evaluare;
- rapoarte de comisie mai bogate procedural.

## 10. Comunicarea rezultatului evaluarii

Status exact: `acoperit`

Ce este acoperit:

- flux dedicat;
- PDF dedicat;
- integrare in dosarul personal;
- expunere in UI.

Unde se vede:

- [backend/internal/education/evaluation_result_issue_flows.go](E:/dev/egueducation/backend/internal/education/evaluation_result_issue_flows.go)
- [backend/internal/education/evaluation_documents.go](E:/dev/egueducation/backend/internal/education/evaluation_documents.go)

## 11. Portofoliul directorului / directorului adjunct

Status exact: `partial`

Ce este acoperit:

- distinctii in dosarul personal pentru director si adjunct;
- documente manageriale;
- PDF pentru documente manageriale;
- sumar de management.
- verticale explicite pentru portofoliul directorului si al directorului adjunct;
- wizard contextual preselectat pentru fiecare verticala;
- listare separata in cockpitul de guvernanta.

Unde se vede:

- [backend/internal/education/managerial_portfolio_summary.go](E:/dev/egueducation/backend/internal/education/managerial_portfolio_summary.go)
- [backend/internal/education/managerial_documents_pdf.go](E:/dev/egueducation/backend/internal/education/managerial_documents_pdf.go)
- [backend/internal/education/personnel_portfolio_relation_summary.go](E:/dev/egueducation/backend/internal/education/personnel_portfolio_relation_summary.go)

Ce mai lipseste exact:

- checklist specific managerial;
- documente-model dedicate acestui rol.

## 12. Documente-model si PDF-uri oficiale aferente

Status exact: `partial`

Ce este acoperit:

- export PDF generic;
- PDF portofoliu;
- PDF-uri pentru evaluari, mobilitate si gradatii;
- PDF-uri pentru unele documente de guvernanta;
- PDF pentru documente de sedinta.

Unde se vede:

- [backend/internal/education/portfolio_documents.go](E:/dev/egueducation/backend/internal/education/portfolio_documents.go)
- [backend/internal/education/evaluation_documents.go](E:/dev/egueducation/backend/internal/education/evaluation_documents.go)
- [backend/internal/education/meeting_documents_pdf.go](E:/dev/egueducation/backend/internal/education/meeting_documents_pdf.go)
- [backend/internal/education/managerial_documents_pdf.go](E:/dev/egueducation/backend/internal/education/managerial_documents_pdf.go)

Ce mai lipseste exact:

- convocatoare CA si CP ca documente-model oficiale complete;
- proces-verbal CA complet;
- proces-verbal CP complet;
- decizie de numire secretar CP;
- documente-model manageriale specifice ghidului.

## 13. Rapoarte manageriale strict necesare

Status exact: `partial`

Ce este acoperit:

- dashboard director;
- dashboarduri pe roluri;
- rapoarte standard;
- rapoarte explicite pentru CA, CP, portofoliu director, comisii si ROF / ROI;
- export contextual CSV / PDF;
- cockpit cu agregari operationale.

Unde se vede:

- [backend/internal/education/cockpit_service.go](E:/dev/egueducation/backend/internal/education/cockpit_service.go)
- [frontend/src/app/features/education/dashboard/education-standard-reports.component.ts](E:/dev/egueducation/frontend/src/app/features/education/dashboard/education-standard-reports.component.ts)
- [frontend/src/app/features/education/dashboard/education-director-dashboard.component.ts](E:/dev/egueducation/frontend/src/app/features/education/dashboard/education-director-dashboard.component.ts)

Ce mai lipseste exact:

- registru sedinte CA complet;
- registru sedinte CP complet;
- registru hotarari;
- raport ROF / ROI;
- raport comisii si evaluare temporara;
- raport transfer portofolii intre unitati;
- raport portofolii pe stadii metodologice.

## Concluzie exacta

Din perspectiva celor 2 PDF-uri:

- nu exista cerinte complet neatinse in zona principala;
- cea mai mare parte este deja construita ca fondatie functionala;
- ce ramane de facut este in principal inchiderea procedurala, documentele oficiale si UI-ul ghidat;
- cerintele complet inchise sunt comunicarea rezultatului evaluarii si opis/indexare/datare.

## Ce trebuie inchis in continuare, in ordinea cea mai corecta

1. portofoliul profesional, ca flux complet;
2. delimitarea portofoliu vs dosar personal;
3. transfer digital si mobilitate;
4. valorificarea portofoliului pe toate cazurile metodologice;
5. ROF / ROI;
6. CA complet;
7. CP complet;
8. comisii si comisia de evaluare;
9. portofoliul directorului si al adjunctului;
10. documente-model si PDF-uri oficiale;
11. rapoarte manageriale strict necesare.
