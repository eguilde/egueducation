# Catalog de cerințe — eArhivă Școala Balotești

Acest catalog descrie comportamentul verificabil al importului celor 716 PDF-uri și al platformei eArhivă multi-tenant. „Implementat în worktree” nu înseamnă „deja disponibil în cluster”; funcția devine operațională numai după build, migrare, rollout și test live.

## Integritate, proveniență și storage

| ID | Cerință | Criteriu de acceptare | Stare |
|---|---|---|---|
| ARC-001 | Originalul este păstrat bit-cu-bit în MinIO | SHA-256 local = SHA-256 versiune DB = obiect verificat; sursa locală nu este modificată | Implementat în manifest și upload; verificare live necesară |
| ARC-002 | Numele original și cel canonic sunt distincte | `original_filename` rămâne în proveniență; numele canonic este `balotesti-archive-NNNN.pdf`; cheia storage nu conține PII | Implementat în worktree |
| ARC-003 | Import idempotent | aceeași cheie de idempotency în aceeași instituție nu creează document/obiect/job duplicat | Implementat în worktree; test PostgreSQL în curs |
| ARC-004 | Storage izolat | credential dedicat bucketului, TLS, prefix instituție/document, fără reutilizarea credentialelor altei aplicații | Secret și GitOps pregătite; credentialul comunicat în chat trebuie rotit |
| ARC-005 | Conservare | versioning, immutability/WORM, retenție și legal hold sunt impuse în storage, nu doar în UI | Lipsă; obligatoriu înainte de declarație de conformitate |
| ARC-006 | Fixity periodic | job periodic recalculează/verifică hash și produce eveniment de conservare | Lipsă |

## Ingestie și securitate

| ID | Cerință | Criteriu de acceptare | Stare |
|---|---|---|---|
| ARC-010 | Validare PDF | limită 100 MiB, magic PDF, structură și 1–2000 pagini, PDF criptat/corupt respins | Implementat în worktree |
| ARC-011 | Malware gate | ClamAV trebuie să dea verdict clean înainte de MinIO; indisponibilitatea scannerului este fail-closed | Implementat în worktree |
| ARC-012 | Control resurse | maximum două uploaduri simultane; validare structurală PDF serializată într-un proces separat cu timeout/memorie; worker asincron; retry exponențial și reclaim lease | Implementat în worktree; izolarea validatorului într-un container separat rămâne hardening recomandat |
| ARC-013 | Manifest verificabil | manifest tipizat, hash sidecar, 716 fișiere/3.019.724.566 bytes/42.342 pagini | Implementat |
| ARC-014 | Reconciliere | după fiecare val: număr, bytes, SHA, joburi, erori și obiecte orfane = reconciliate | Planificat în runbook |

## Tenant și autorizare

| ID | Cerință | Criteriu de acceptare | Stare |
|---|---|---|---|
| ARC-020 | Tenant derivat server-side | clientul nu trimite/selectează tenant; hostul, tokenul și membershipul stabilesc tenant+instituție | Implementat |
| ARC-021 | RLS + integritate DB | FORCE RLS și FK compuse pe `institution_id` împiedică citiri și legături cross-tenant | Implementat în worktree; test negativ DB în curs |
| ARC-022 | Drepturi distincte | `earchiva.read` pentru căutare/detaliu; `earchiva.manage` pentru import/admin; `earchiva.content.read` pentru original; `earchiva.review` pentru clasificare | Implementat în worktree |
| ARC-023 | Acces pe fond/confidențialitate | documentele sensibile pot fi limitate pe fond, compartiment și nivel de acces | Lipsă; tenant-wide read este insuficient pentru arhivă sensibilă |
| ARC-024 | Bypass platformă | nicio cerere normală, inclusiv superadmin tenant, nu citește alt tenant prin UUID | Predicate explicite și RLS implementate; test negativ în curs |

## OCR, metadate și clasificare

| ID | Cerință | Criteriu de acceptare | Stare |
|---|---|---|---|
| ARC-030 | OCR scan real | Azure Document Intelligence procesează cele 705 PDF-uri fără text; OCR gol nu devine `ready` | Implementat în worktree; test live necesar |
| ARC-031 | Proveniență OCR | provider, model, versiune API, pagini și moment procesare sunt păstrate | Parțial implementat |
| ARC-032 | Coordonate/confidence pe pagină | text, liniile și scorurile se păstrează cu număr de pagină și confidence | Lipsă; adaptorul păstrează momentan textul agregat |
| ARC-033 | Clasificare asistată | fond/serie/tip/datǎ/număr sunt propuse cu confidence și sursă, fără publicare automată la scor mic | Implementat în worktree; reguli deterministe, validare live necesară |
| ARC-034 | Human review | operatorul aprobă/corectează clasificarea prin DTO structurat; optimistic revision și audit atomic | Implementat în backend și React; review OCR la nivel de pagină rămâne gap |

## Căutare și acces document

| ID | Cerință | Criteriu de acceptare | Stare |
|---|---|---|---|
| ARC-040 | Full-text tenant-safe | căutarea returnează exclusiv documentele instituției active | Implementat; test negativ în curs |
| ARC-041 | Căutare semantică | embedding real și index scalabil, cu izolare server-side per tenant | Implementarea curentă este feature hashing; necesită înlocuire sau acceptare explicită ca non-semantică |
| ARC-042 | Vizualizare și versiuni | listă, filtre, detaliu, versiuni, status OCR și taxonomie responsive | Implementat în frontend worktree |
| ARC-043 | Download controlat | endpoint separat `earchiva.content.read`; tentativa este auditată înainte de primul byte și outcome-ul după stream | Original implementat; derivat/DIP lipsesc |

## Administrare și operare

| ID | Cerință | Criteriu de acceptare | Stare |
|---|---|---|---|
| ARC-050 | Health administrativ | admin vede storage/OCR configurat, documente, bytes/pagini și coada, fără secrete | Implementat în worktree; fără probe externe active |
| ARC-051 | Job management | listare, filtrare, retry confirmat și audit atomic pentru joburi failed | Implementat în worktree |
| ARC-052 | Observabilitate | metrici pentru throughput, latență OCR, retry, DLQ, cost și alerte | Lipsă |
| ARC-053 | Taxonomie tenant | nomenclatoarele și politicile de retenție nu pot fi modificate cross-tenant | Taxonomia eArhivă este tenant-scoped; nomenclatorul global legacy rămâne gap |

## Conservare eIDAS 2.0 și interoperabilitate

| ID | Cerință | Criteriu de acceptare | Stare |
|---|---|---|---|
| ARC-060 | Durabilitate, lizibilitate și integritate | proceduri tehnice și dovezi verificabile protejează originea, integritatea și accesibilitatea pe termen lung | Parțial; hash/versioning DB nu este suficient |
| ARC-061 | Raport automat semnat/sigilat | ingestia produce raport de evidență semnat/sigilat și timestamp de la servicii de încredere adecvate | Lipsă |
| ARC-062 | SIP/AIP/DIP | export/import E-ARK CSIP/SIP/AIP/DIP cu METS/PREMIS și validare | Lipsă |
| ARC-063 | Retenție/dispoziție | legal hold, expirare, aprobare în patru ochi și dovadă de eliminare | Lipsă |
| ARC-064 | Declarație de conformitate | produsul nu este numit serviciu calificat până la QTSP, evaluare și trusted-list aplicabile | Regula de produs acceptată; certificarea nu există |

Referințe normative principale: [Regulamentul eIDAS consolidat](https://eur-lex.europa.eu/eli/reg/2014/910/2024-05-20/eng), [Regulamentul de punere în aplicare (UE) 2025/2532](https://eur-lex.europa.eu/eli/reg_impl/2025/2532/oj/eng/pdf), [E-ARK CSIP/SIP/AIP](https://dilcis.eu/specifications) și [ISO 14721:2025](https://www.iso.org/standard/87471.html).
