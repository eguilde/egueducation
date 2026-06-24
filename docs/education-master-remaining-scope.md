# Master scope ramas de implementat - management educational

Data consolidarii: 2026-06-24

Scopul acestui document este sa fie lista canonica a tot ce mai avem de facut in modulul educational.

Regula de lucru de aici inainte:

- cand intrebi "ce mai avem de facut", raspunsul trebuie raportat la acest document;
- nu mai introducem categorii noi in raspunsuri ad-hoc;
- orice element nou aparut trebuie marcat explicit ca:
  - `bug descoperit in implementarea existenta`, sau
  - `dependenta tehnica necesara`, sau
  - `extindere de scope aprobata explicit`.

## 1. Ce este deja livrat

Acestea nu mai fac parte din scope-ul ramas:

- workspace-urile education pe domenii;
- configurarea pe registre si tab-uri;
- wizard kit reutilizabil;
- wizard-uri initiale pentru guvernanta, personal, evaluari, declaratii, mobilitate, gradatii, portofolii;
- cockpit `director`;
- cockpit `secretariat`;
- cockpit `conformitate`;
- cockpit `profesor`;
- rapoarte standard initiale;
- sumar procedural pentru sedinte;
- tranzitii procedurale pentru sedinte;
- exporturi PDF initiale, inclusiv minute si hotarari;
- suita E2E curenta.

## 2. Scope ramas - lista completa actuala

Tot ce urmeaza reprezinta scope-ul ramas cunoscut in acest moment.

### A. Cockpituri pe roluri

1. cockpit `evaluator`
2. cockpit `diriginte`
3. cockpit `inspector`
4. cockpit `responsabil_comisie`

### B. RBAC contextual si control acces

5. helperi backend comuni pentru RBAC contextual
6. aplicare uniforma a regulilor contextuale pe endpointurile sensibile
7. contracte coerente `403` pentru refuz contextual
8. audit complet pentru actiuni sensibile:
   - acces dosar personal
   - export sensibil
   - publicare
   - aprobare
   - inchidere flux

### C. Guvernanta institutionala

9. hardening complet pentru CA
10. verticala completa pentru CP
11. verticala pentru comisii
12. verticala pentru consiliul clasei
13. documente oficiale finale suplimentare pentru guvernanta
14. rapoarte extinse pentru sedinte, hotarari, cvorum, restante

### D. Documente manageriale si regulamente

15. workflow procedural complet pentru documente manageriale
16. workflow procedural complet pentru regulamente
17. versiuni, avizare, aprobare, publicare, arhivare
18. PDF-uri oficiale dedicate pentru:
   - PDI/PAS
   - RAEI
   - regulamente
   - rapoarte institutionale
19. legatura cerinta -> document -> dovada -> publicare

### E. Dosar personal si personal educational

20. dosar personal procedural complet
21. completitudine dosar
22. documente lipsa sau expirate
23. acces sensibil cu audit si motivatie
24. rapoarte HR/secretariat pentru dosare si acces

### F. Evaluari, mobilitate, gradatii, portofolii

25. timeline procedural pentru evaluari
26. timeline procedural pentru mobilitate
27. timeline procedural pentru gradatii
28. timeline procedural pentru portofolii
29. reguli de business suplimentare pe tranzitii
30. rapoarte specializate suplimentare pentru aceste fluxuri

### G. Conformitate institutionala

31. urmarire completa cerinta -> dovada
32. gap analysis operational
33. publicari restante si anonimizare cu urmarire mai adanca
34. trasee de remediere pentru cerinte partial implementate

### H. Verticala elevi si clase

35. registru elevi
36. registru clase
37. alocare diriginte
38. dosar elev
39. inscrieri, transferuri, retrageri
40. situatie scolara de baza
41. absente
42. recompense si sanctiuni
43. risc educational
44. consiliul clasei integrat cu elevi/clase

### I. Rapoarte si inspectie

45. extindere catalog rapoarte standard
46. rapoarte comparative pe ani scolari
47. rapoarte executive pe roluri
48. pachet inspectie
49. exporturi institutionale suplimentare
50. XLSX unde este necesar

### J. Hardening final si operationalizare

51. optimizari query si indexuri unde este nevoie
52. consistenta naming pentru documente generate
53. seed minim sau scenarii demo mai bune
54. testare suplimentara backend pentru RBAC si validari
55. extindere E2E pe fluxurile noi

## 3. Ce NU trebuie considerat "scope nou"

Urmatoarele NU sunt scope nou. Sunt doar subtaskuri, defecte sau detalii interne ale punctelor de mai sus:

- ajustari de selector in teste;
- extinderi de mock pentru roluri deja in scope;
- o noua ruta pentru un cockpit deja inclus in lista;
- un PDF necesar pentru o verticala deja listata;
- un helper tehnic necesar ca sa implementam un punct deja asumat;
- un bug descoperit in ceea ce este deja implementat.

## 4. Cum trebuie raportat progresul

De aici inainte, progresul trebuie raportat doar asa:

- `livrat`
- `partial`
- `neinceput`
- `blocat`

si doar raportat la punctele 1-55 de mai sus.

## 5. Stare actuala agregata

### Livrat

- cockpit `director`
- cockpit `secretariat`
- cockpit `conformitate`
- cockpit `profesor`
- wizard-uri principale
- self-service portofoliu profesor cu wizard ghidat de completare / validare si datare vizibila
- sumar procedural sedinte
- PDF-uri initiale de guvernanta
- rapoarte standard initiale
- suite E2E curente

### Partial

- RBAC contextual
- guvernanta CA
- documente manageriale
- conformitate
- dosar personal
- evaluari / mobilitate / gradatii / portofolii
- rapoarte executive

### Neinceput sau aproape neinceput

- cockpit `evaluator`
- cockpit `diriginte`
- cockpit `inspector`
- cockpit `responsabil_comisie`
- CP complet
- comisii
- consiliul clasei
- verticala elevi/clase
- pachet inspectie complet

## 6. Ordinea recomandata de executie

Ordinea stabila recomandata este:

1. cockpit `evaluator`
2. cockpit `diriginte`
3. helperi backend comuni pentru RBAC contextual
4. dosar personal procedural
5. workflow + PDF-uri pentru documente manageriale si regulamente
6. timeline procedural pentru evaluari / mobilitate / gradatii / portofolii
7. CP complet
8. comisii
9. verticala elevi/clase
10. rapoarte extinse + pachet inspectie
