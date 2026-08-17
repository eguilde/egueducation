# Importator manifest eArhivă — Balotești

`scripts/archive/import-balotesti-manifest.ps1` este un client desktop OIDC și trimite fiecare PDF numai la `POST /api/earchiva/documents`. Nu comunică direct cu MinIO. Backendul aplică validarea PDF, ClamAV, auditul, stocarea și coada OCR/indexare.

## Model de securitate

- Nu se copiază, nu se acceptă și nu se persistă tokenuri de acces. Importatorul pornește Authorization Code cu PKCE S256 pentru clientul public `egueducation-desktop`.
- Discovery-ul OIDC este verificat, iar callback-ul este exclusiv loopback `http://localhost:4300/callback`, redirect preînregistrat pentru clientul desktop. Browserul se deschide pentru OTP/consimțământ; codul este acceptat numai dacă `state` și calea callback sunt exacte.
- Tokenurile nu sunt scrise în checkpoint/log. Un `CookieContainer` ține refresh cookie-ul HttpOnly pentru procesul curent și reînnoiește accesul înainte de expirare. Cookie-ul nu este exportat pe disc.
- Importatorul folosește `Authorization: Bearer` numai pentru un token desktop nelegat. Dacă endpointul de token livrează `token_type=DPoP` sau un JWT cu `cnf.jkt`, importatorul oprește; nu degradează un token legat DPoP.
- Tenantul și instituția nu sunt parametri sau metadata client. Backendul le deduce din token și host. Cheia de idempotență este doar `archive-import-v1:<sha256>`.
- Manifestul verifică local că API-ul este hostul Balotești; această gardă previne importul accidental în alt tenant și nu este transmisă ca tenant/instituție.
- Înainte de fiecare upload sunt recalculate path-ul sigur, dimensiunea și SHA-256. Originalele sunt numai citite.

## Verificare și canary

```powershell
pwsh -File scripts/archive/import-balotesti-manifest.ps1 -SelfTest
pwsh -File scripts/archive/import-balotesti-manifest.ps1 -DryRun -MaxFiles 5 -ApiBaseUrl https://scoalabalotesti.eguilde.cloud
pwsh -File scripts/archive/import-balotesti-manifest.ps1 -ApiBaseUrl https://scoalabalotesti.eguilde.cloud -MaxFiles 5 -PollUntilTerminal
```

Ultima comandă deschide browserul; operatorul se autentifică normal. Nu sunt necesare sau permise tokenuri lipite în shell. Pentru reluare după un val aprobat, se folosește `-Resume`; checkpoint-ul este asociat hashului exact al manifestului. Implicit, eroarea oprește valul după checkpoint. `-ContinueOnError` se folosește numai într-un val aprobat.
