# Catalog de cerințe — Registratura Costești → eGuEducation React

Sursa de adevăr pentru acest catalog este auditul autentificat, read-only, efectuat la 17 august 2026 pe `https://registratura.costesti.eu/registratura` și pe zona sa de administrare. Nu au fost salvate, modificate sau șterse date în aplicația de referință.

## Ecranul principal

| ID | Cerință verificată în Costești | Criteriu de acceptare eGuEducation |
|---|---|---|
| REG-001 | Toolbar cu Intrare, Ieșire și Multiplu | Butoanele sunt PrimeReact, vizibile și active numai pe permisiuni backend reale; deschid formularele corespunzătoare. |
| REG-002 | Selector de registru | Listează numai registrele vizibile utilizatorului în tenantul curent, păstrează selecția per host/tenant și reîncarcă pagina 1. |
| REG-003 | Filtrare implicit închisă, deschisă prin lupă | Panoul este închis la încărcare, are stare `aria-expanded`, buton de închidere, Resetare și Caută documente. |
| REG-004 | Filtre: nr. document, tip, nr. extern, emitent, destinatar, interval intrare, interval ieșire | Toate filtrele sunt trimise serverului; tenantul/registrul sunt derivate și validate în backend. |
| REG-005 | Tabel server-side | Backend procesează filtrarea, sortarea și pagina; UI ignoră răspunsurile stale. |
| REG-006 | Coloane: expand, Nr. Doc, Tip, Conținut, Emitent, Destinatar, Data intrare, Data ieșire, Status, Acțiuni | Ordinea și sensul datelor sunt identice cu referința. |
| REG-007 | Expansiune inline | Afișează Compartimente, Nr. extern, Data nr. extern și Activitate fără navigare. |
| REG-008 | Paginare | Implicit 20; afișează intervalul și totalul; first/previous/pages/next/last; selector 10/20/50/100. |
| REG-009 | Acțiuni per rând | Istoric, Editare, Anulare, PDF și Flux sunt butoane icon PrimeReact separate, cu etichete accesibile și RBAC. |
| REG-010 | Export PDF pe interval | Dialog cu Data început/Data sfârșit; interval maxim și validare server-side; export doar pentru registrul vizibil curent. |

## Formulare și fluxuri

| ID | Cerință verificată | Criteriu de acceptare |
|---|---|---|
| REG-011 | Intrare | Document/Dosar, Emitent obligatoriu, Conținut obligatoriu, fișier, Destinatar obligatoriu, Data intrare, Compartiment obligatoriu, Nr. extern și Data nr. extern. |
| REG-012 | Ieșire | Aceleași câmpuri, cu Data ieșire și semantica emitent/destinatar corespunzătoare. |
| REG-013 | Multiplu | 1–20 documente, registru, conținut opțional, dată implicită, tip MULTIPLU, corespondent implicit și posibilitate ulterioară de conversie în Intrare/Ieșire. |
| REG-014 | Istoric | Dialog tabelar cu Dată, Nr. Doc, Tip, Conținut, Status și descrierea modificării; versiuni auditate. |
| REG-015 | Editare | Câmpurile documentului, compartimente, tip Document/Dosar, atașamente și notă obligatorie de modificare; statusul este controlat de flux. |
| REG-016 | Anulare | Confirmare separată, avertisment ireversibil, motiv obligatoriu și tranziție terminală auditată. |
| REG-017 | PDF individual | Generează/descarcă PDF numai după verificarea tenantului, registrului și permisiunilor. |
| REG-018 | Flux document | Panou cu document, istoric și numai acțiunile valide pentru starea curentă; atribuire compartiment/utilizator și concurență optimistă. |
| REG-019 | Atașamente | Upload real multipart, limită, validare, scanare antivirus, stare `ready` înainte de succes și download autorizat/auditat. |

## Administrare Registratură

| ID | Zonă verificată | Cerințe |
|---|---|---|
| REG-A01 | Utilizatori | Listă paginată/filterabilă cu identitate, email/telefon verificate, roluri, compartimente, organizație, stare, ultima autentificare și acțiuni. |
| REG-A02 | Compartimente | CRUD, descriere, ierarhie/rol, activ/inactiv și protecția referințelor. |
| REG-A03 | Registre | CRUD, prefix, număr început/curent/următor, compartimente, public/privat, implicit și numerotare tenant-safe. |
| REG-A04 | Persoane fizice | CRUD paginat cu nume/prenume, identificator, contact și date specializate. |
| REG-A05 | Persoane juridice | CRUD paginat cu denumire, CUI, registrul comerțului, email și reprezentant legal. |
| REG-A06 | Instituții publice | CRUD paginat cu denumire, tip, nivel, email, website și instituție implicită. |
| REG-A07 | Organizații | CRUD, descriere, implicit și activ. |
| REG-A08 | Organigramă | Arbore compartimente/utilizatori, zoom, adăugare și atribuiri; toate operațiile tenant-scoped. |
| REG-A09 | Profil utilizator | Date personale/contact, verificări, compartiment principal, passkeys și preferințe de aspect. |

## Reguli multi-tenant și RBAC

- Hostul rezolvă tenantul înainte de autentificare; clientul nu poate alege `institution_id` sau `tenant_code`.
- Orice query și mutație are context de tenant/instituție, RLS și predicate explicite pentru resursele directe.
- Registrele private sunt vizibile numai prin atribuiri de compartiment validate în backend.
- `registratura.read`, `registratura.manage`, `workflow.manage`, `registratura.links.read/manage` și drepturile administrative rămân permisiuni distincte.
- UI ascunde/dezactivează acțiunile pentru UX, dar backend-ul este autoritatea finală.
- Contul automat de test trebuie să fie membru exclusiv al tenantului Balotești și să primească toate permisiunile tenantului pentru testare; nu poate obține acces cross-tenant.

## Verificare obligatorie înainte de declararea parității

1. Teste unitare pentru filtrare, sortare, paginare, race A→B, expansiune și fiecare acțiune.
2. Test Playwright desktop și mobil cu API mock pentru contractele exacte.
3. Test PostgreSQL pentru RLS negativ între doi tenanți, registre private și numerotare.
4. Canary OIDC în producție, apoi `/api/me` cu tenantul și permisiunile exacte.
5. Smoke autentificat în producție: listă, filtru, paginare, detaliu, formulare deschise și RBAC; mutațiile se fac numai pe fixture dedicat.

