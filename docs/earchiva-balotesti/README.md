# Inventar tehnic al arhivei PDF - Scoala Balotesti

Acest inventar a fost executat read-only asupra sursei locale:
`D:\balotesti\scanari dosare Balotesti`.

Originalele nu au fost redenumite, mutate, deschise pentru scriere sau modificate. Nu a fost salvat continut OCR, text extras, imagini randate sau metadate documentare cu caracter personal. Manifestul operational pastreaza numai calea relativa si hash-ul necesare pentru mapping si verificarea integritatii.

## Rezultate

- 716 fisiere PDF, 3,019,724,566 bytes (aprox. 2.81 GiB).
- 42,342 pagini in total; minim 1, maxim 848, median 34.5 pagini/document.
- 716/716 valide la deschidere cu pypdf 6.10.0; 0 erori.
- 0 fisiere criptate/parolate.
- 11 documente au text layer detectabil; 705 nu au text layer detectabil si trebuie tratate ca scan-uri pentru OCR.
- 0 duplicate exacte SHA-256; toate 716 hash-uri sunt distincte.

## Verificari vizuale si de scan

Un esantion determinist de 17 documente a acoperit inceputul listei, extremele de marime, quantile de marime si quantile de numar de pagini. Prima pagina a fost inspectata prin randare Poppler pentru patru fisiere reprezentative. Au fost observate atat pagini portret, cat si landscape, precum si scan-uri cu scris de mana, zgomot/urme de scanare si variatii de contrast. Aceasta nu este o certificare de calitate OCR: fiecare pagina trebuie procesata si, pentru scoruri mici, revizuita manual.

Detaliile tehnice ale eșantionului rămân într-un artefact local ignorat de Git. Temporarele de randare rămân numai în `tmp/pdfs/` și nu fac parte din arhiva operațională.

## Naming si mapping

Schema propusa este stabila, fara date personale in numele nou:

`balotesti-archive-0001.pdf` ... `balotesti-archive-0716.pdf`

Inventarul tehnic păstrează ordinea lexicografică inițială. Manifestul operațional local aplică o sortare naturală, case-insensitive, identică regulii din importatorul UI, apoi atribuie numele canonic. Redenumirea efectivă nu a fost executată în sursă, deoarece aceasta trebuie păstrată ca evidență, iar ingestia în storage trebuie făcută idempotent, cu verificare hash și jurnal de audit.

## Blocaje si recomandari pentru ingestie

1. Nu se poate deduce cu certitudine categoria, data, numarul de inregistrare sau durata de pastrare numai din denumirile actuale. OCR si clasificarea trebuie sa produca propuneri cu scor, apoi o etapa de validare umana.
2. Cele 705 documente fara text layer necesita OCR. Pentru documentele cu scris de mana, OCR automat trebuie marcat ca nesigur si nu trebuie sa suprascrie originalul.
3. Variatia portret/landscape trebuie pastrata ca proprietate de pagina; nu se va roti sau recomprima originalul la ingestie.
4. Cheia de storage include instituția derivată server-side și UUID-ul documentului, nu numele sursă: `archive/{institution_id}/{document_id}/original/document-{uuid}.pdf`. Indexul de căutare este filtrat server-side prin instituția sesiunii.
5. Inainte de executia in cluster sunt necesare validarea secrets Azure (endpoint, credential/key, container/index names), test de conectivitate cu permisiuni minimale, dead-letter/retry si reconciliere hash-count.

## Artefacte

- manifestele reale CSV/JSON, sidecar-ul SHA, eșantionul de calitate și lista duplicatelor sunt artefacte operaționale locale ignorate de Git; ele conțin căi, nume și hash-uri ale arhivei reale și nu se publică în repository;
- manifestul operațional nu conține/selectează tenantul; verifică doar hostul API așteptat, iar backendul derivă instituția din sesiunea OIDC și host;
- `REQUIREMENTS.md` - catalogul de cerinte si starea implementarii.
- `IMPORT_RUNBOOK.md` - gate-urile, valurile si reconcilierea importului.
