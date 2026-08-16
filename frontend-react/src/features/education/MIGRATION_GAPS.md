# Școală — starea migrării React

Fundația React implementează navigație responsive bazată pe module și permisiuni, context instituțional derivat exclusiv de backend din sesiunea/tokenul autentificat și apartenența la tenant, dashboardul de guvernanță și un tipar reutilizabil pentru liste paginate. Clientul nu trimite și backendul nu consumă un antet `X-Institution-ID`, evitând alegerea arbitrară a instituției din browser.

Catalogul backend conține 304 operații Education. Toate cele 12 domenii funcționale au acum liste reale conectate la API. Ședințele de guvernanță, precum și înregistrările principale de decizii, management, regulamente, comisii, personal, evaluări, declarații, mobilitate, gradații de merit, portofolii și conformitate au listă, detaliu, creare, editare și ștergere cu formulare specifice domeniului.

Implementat și pentru guvernanță: memberships, participanți, documente, voturi, minute și hotărâri, fiecare cu listă, detaliu, creare, editare și ștergere; PDF protejat pentru documente, minute și hotărâri.

Implementat pentru dosare: documente și pași workflow management; încadrări, documente, disciplinar și acces personal; autoevaluări, criterii, contestații și comunicări evaluări; documente, punctaje, contestații și decizii pentru mobilitate și gradații; documente, checklist, opis, custodie și revizuiri portofolii. Exporturile PDF/CSV sunt descărcate prin fetch autentificat.

Toate rutele Education documentate în contractul OpenAPI sunt expuse prin interfață: liste, detalii, CRUD pentru resursele modificabile, documente/PDF, comenzi de portofoliu, exporturi, dashboarduri, filtre, cataloage și sumarizări. Nu există rute Education documentate care să fie lăsate intenționat neexpuse sau contracte backend invalide identificate în această migrare.
