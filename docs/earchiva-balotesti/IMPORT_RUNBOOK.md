# Runbook de import controlat — Balotești

## Gate-uri obligatorii

Importul nu începe până când toate sunt adevărate:

1. credentialul MinIO comunicat prin chat este revocat și înlocuit printr-un canal securizat;
2. bucketul `earhive` are credential dedicat, TLS, policy minimă și protecție de conservare agreată;
3. migrațiile până la 0078 au trecut pe PostgreSQL 17 și schema contract este verde;
4. backendul nou este deployat cu secret references MinIO/Azure și NetworkPolicy permite numai destinațiile aprobate;
5. ClamAV, MinIO și Azure DI au smoke test verde din podul backend;
6. testele negative tenant A → document tenant B sunt 403/404 pentru listă, UUID, versiuni și căutare;
7. manifestul și sidecar-ul SHA-256 sunt verificate față de folderul read-only.

## Valuri

| Val | Conținut | Gate de promovare |
|---|---:|---|
| Canary | 5 PDF-uri: text layer, scan simplu, landscape, scris de mână, document mare | 5 obiecte, 5 documente, 5 versiuni, 5 joburi terminale; hash identic; niciun rezultat cross-tenant |
| Pilot | 25 PDF-uri | retry/latency/cost acceptabile; zero orfani/duplicate |
| Val 1 | 100 PDF-uri | reconciliere completă; review queue funcțională |
| Val 2+ | loturi de 100 | aceleași criterii; oprire automată la abatere |
| Final | restul până la 716 | totalurile exacte din manifest și raport final |

## Reconciliere după fiecare val

- fișiere locale selectate = documente DB = versiuni inițiale = obiecte MinIO;
- suma bytes și SHA-256 coincid;
- fiecare document are exact un job curent și o stare explicabilă;
- `ready` implică OCR/text nenul, chunks indexate și artifact de procesare;
- `failed` are eroare sanitizată și acțiune retry/review;
- interogările FTS pentru termeni canary găsesc documentele corecte;
- utilizator fără `earchiva.manage` nu poate importa/retry;
- alt tenant nu poate lista, căuta sau deschide UUID-urile lotului.

Originalele din `D:\balotesti\scanari dosare Balotesti` nu se șterg după import. Ele rămân dovada sursă până la aprobarea formală a raportului de reconciliere și a politicii de conservare.
