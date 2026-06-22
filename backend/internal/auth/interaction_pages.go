package auth

import (
	"html/template"
	"net/http"
	"net/url"
	"strings"
	"time"
)

type oidcLoginPageData struct {
	CustomerName string
	ReturnURL    string
	Error        string
	Phone        string
	MaskedPhone  string
}

type oidcConsentPageData struct {
	CustomerName string
	RequestID     string
	ClientName    string
	Scopes        []ConsentScope
	ExpiresAt     string
	Error         string
}

type oidcLogoutPageData struct {
	CustomerName string
	ReturnTo     string
}

func (s *Service) LoginPage(w http.ResponseWriter, r *http.Request) {
	returnURL := strings.TrimSpace(r.URL.Query().Get("returnUrl"))
	if returnURL == "" {
		returnURL = s.cfg.OIDCIssuer + "/authorize"
	}
	if subject := s.currentSubject(r); subject != "" {
		http.Redirect(w, r, returnURL, http.StatusFound)
		return
	}

	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	_ = template.Must(template.New("oidc-login").Parse(oidcLoginHTML)).Execute(w, oidcLoginPageData{
		CustomerName: "EguEducation",
		ReturnURL:    returnURL,
	})
}

func (s *Service) ConsentPage(w http.ResponseWriter, r *http.Request) {
	subject := s.currentSubject(r)
	if subject == "" {
		http.Redirect(w, r, s.cfg.OIDCIssuer+"/login?returnUrl="+url.QueryEscape(s.cfg.OIDCIssuer+"/authorize?"+r.URL.RawQuery), http.StatusFound)
		return
	}

	data, err := s.loadConsentPageData(r, subject)
	if err != nil {
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		w.WriteHeader(http.StatusNotFound)
		_ = template.Must(template.New("oidc-consent-error").Parse(oidcConsentHTML)).Execute(w, oidcConsentPageData{
			CustomerName: "EguEducation",
			Error:        "Cererea de consimțământ nu a fost găsită sau a expirat.",
		})
		return
	}

	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	_ = template.Must(template.New("oidc-consent").Parse(oidcConsentHTML)).Execute(w, data)
}

func (s *Service) LogoutPage(w http.ResponseWriter, r *http.Request) {
	returnTo := strings.TrimSpace(r.URL.Query().Get("returnTo"))
	if returnTo == "" {
		returnTo = s.cfg.FrontendOrigin
	}

	if cookie, err := r.Cookie(sessionCookieName); err == nil {
		_ = s.revokeLoginSession(r.Context(), strings.TrimSpace(cookie.Value))
	}
	http.SetCookie(w, &http.Cookie{
		Name:     sessionCookieName,
		Value:    "",
		Path:     "/",
		HttpOnly: true,
		SameSite: http.SameSiteLaxMode,
		Secure:   s.cfg.Environment == "production",
		MaxAge:   -1,
		Expires:  time.Unix(0, 0),
	})

	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	_ = template.Must(template.New("oidc-logout").Parse(oidcLogoutHTML)).Execute(w, oidcLogoutPageData{
		CustomerName: "EguEducation",
		ReturnTo:     returnTo,
	})
}

func (s *Service) loadConsentPageData(r *http.Request, subject string) (oidcConsentPageData, error) {
	requestID := strings.TrimSpace(r.URL.Query().Get("request"))
	if requestID == "" {
		return oidcConsentPageData{}, http.ErrNoCookie
	}

	var (
		clientName string
		scope      string
		expiresAt  time.Time
	)
	if err := s.db.QueryRow(r.Context(), `
		select c.client_name, cr.scope, cr.expires_at
		from oidc_consent_requests cr
		join oidc_clients c on c.client_id = cr.client_id
		where cr.id::text = $1
			and lower(cr.subject) = lower($2)
			and cr.status = 'pending'
			and cr.expires_at > now()
	`, requestID, subject).Scan(&clientName, &scope, &expiresAt); err != nil {
		return oidcConsentPageData{}, err
	}

	return oidcConsentPageData{
		CustomerName: "EguEducation",
		RequestID:     requestID,
		ClientName:    clientName,
		Scopes:        consentScopes(scope),
		ExpiresAt:     expiresAt.UTC().Format(time.RFC3339),
	}, nil
}

const oidcLoginHTML = `<!DOCTYPE html>
<html lang="ro">
<head>
  <meta charset="UTF-8">
  <meta name="viewport" content="width=device-width, initial-scale=1.0">
  <title>{{.CustomerName}} - Autentificare</title>
  <style>
    *,*::before,*::after{box-sizing:border-box;margin:0;padding:0}
    html,body{height:100%}
    :root{color-scheme:light;--bg:#fff7f8;--card:#fff;--soft:#fff1f2;--border:#e2e8f0;--text:#0f172a;--muted:#64748b;--primary:#e11d48;--primary-contrast:#fff;--shadow:0 24px 60px rgba(15,23,42,.14)}
    html.dark,html.app-dark{color-scheme:dark;--bg:#020617;--card:#0f172a;--soft:#1e293b;--border:#334155;--text:#f8fafc;--muted:#cbd5e1;--primary:#f43f5e;--primary-contrast:#fff;--shadow:0 24px 60px rgba(0,0,0,.35)}
    body{font-family:Inter,system-ui,-apple-system,sans-serif;background:radial-gradient(circle at top left,rgba(225,29,72,.16),transparent 28rem),linear-gradient(135deg,var(--bg),#fff 44%,#ffe4e9);color:var(--text);min-height:100%;overflow:hidden}
    .shell{min-height:100vh;display:grid;grid-template-columns:minmax(360px,480px) minmax(0,1fr)}
    .visual{display:flex;align-items:flex-end;min-height:100vh;padding:clamp(40px,6vw,80px);background:linear-gradient(135deg,rgba(15,23,42,.9),rgba(225,29,72,.72) 48%,rgba(8,47,73,.46));color:#fff}
    .visual h2{font-size:clamp(34px,4.5vw,64px);line-height:1.02;margin:14px 0 18px}
    .visual p{max-width:460px;font-size:16px;line-height:1.6;color:rgba(255,255,255,.86)}
    .panel{display:flex;align-items:center;justify-content:center;padding:clamp(24px,4vw,64px);overflow-y:auto}
    .card{width:100%;max-width:420px;padding:24px;border:1px solid var(--border);border-radius:18px;background:var(--card);box-shadow:var(--shadow)}
    .header{text-align:center;margin-bottom:18px}
    .badge{display:inline-flex;align-items:center;gap:8px;padding:8px 12px;border-radius:999px;border:1px solid var(--border);background:var(--soft);color:var(--primary);font-size:12px;font-weight:700;letter-spacing:.08em;text-transform:uppercase}
    h1{font-size:20px;margin-top:10px}
    .subtitle{margin-top:4px;font-size:13px;color:var(--muted);line-height:1.6}
    .grid{display:grid;gap:10px;margin-bottom:14px}
    .method{display:flex;flex-direction:column;gap:8px;padding:12px;border-radius:12px;border:1px solid var(--border);background:var(--card);cursor:pointer;transition:transform .2s,box-shadow .2s,border-color .2s}
    .method:hover{transform:translateY(-1px);box-shadow:0 14px 30px rgba(15,23,42,.12);border-color:var(--primary)}
    .method strong{font-size:14px}
    .method span{font-size:12px;color:var(--muted);line-height:1.5}
    .field{margin-bottom:14px}
    .field label{display:block;font-size:13px;font-weight:600;margin-bottom:6px}
    .field input{width:100%;padding:10px 12px;border:1px solid var(--border);border-radius:10px;font:inherit;color:var(--text);background:var(--soft)}
    .field input:focus{outline:none;border-color:var(--primary);box-shadow:0 0 0 3px rgba(225,29,72,.15)}
    .btn{display:inline-flex;align-items:center;justify-content:center;gap:6px;width:100%;padding:11px 16px;border:none;border-radius:10px;background:var(--primary);color:var(--primary-contrast);font:inherit;font-weight:700;cursor:pointer;text-decoration:none}
    .btn.secondary{background:transparent;border:1px solid var(--border);color:var(--primary)}
    .row{display:grid;gap:10px;grid-template-columns:repeat(2,minmax(0,1fr))}
    .muted{font-size:12px;color:var(--muted);line-height:1.5}
    .notice{margin-top:10px;padding:10px 12px;border-radius:12px;background:var(--soft);border:1px solid var(--border);font-size:12px;line-height:1.5}
    .error{margin-bottom:14px;padding:10px 12px;border-radius:12px;background:#fef2f2;border:1px solid #fecaca;color:#b91c1c;font-size:12px}
    @media(max-width:900px){body{overflow-y:auto}.shell{display:block}.visual{display:none}.panel{min-height:100vh;padding:20px}.card{max-width:440px}}
    @media(max-width:420px){.panel{padding:0}.card{min-height:100vh;max-width:none;border:0;border-radius:0;box-shadow:none}}
  </style>
</head>
<body>
  <div class="shell">
    <section class="visual">
      <div>
        <div class="badge">OIDC Provider</div>
        <h2>Autentificare sigură pentru EguEducation</h2>
        <p>Acces cu SMS OTP, passkey și EUDI wallet, cu consimțământ și sesiuni gestionate exclusiv în backend-ul OIDC.</p>
      </div>
    </section>
    <section class="panel">
      <div class="card">
        <div class="header">
          <div class="badge">Autentificare</div>
          <h1>{{.CustomerName}}</h1>
          <p class="subtitle">Alege metoda de autentificare și continuă în fluxul OIDC.</p>
        </div>

        <div id="error" class="error" hidden></div>
        <div class="grid">
          <button class="method" type="button" onclick="showSms()">
            <strong>SMS OTP</strong>
            <span>Autentificare cu cod trimis pe telefonul verificat.</span>
          </button>
          <button class="method" type="button" onclick="loginPasskey()">
            <strong>Passkey</strong>
            <span>Cheie de acces fără parolă, pentru autentificare rapidă și sigură.</span>
          </button>
          <button class="method" type="button" onclick="showError('EUDI wallet login este disponibil după configurarea serviciului de portofel digital.')">
            <strong>EUDI wallet</strong>
            <span>Fluxul de portofel digital este servit de providerul OIDC.</span>
          </button>
        </div>

        <form id="smsForm" class="grid" onsubmit="return requestSms(event)" hidden>
          <div class="field">
            <label for="phone">Număr de telefon</label>
            <input id="phone" name="phone" autocomplete="tel" inputmode="tel" placeholder="+40..." />
          </div>
          <button class="btn" type="submit">Trimite codul</button>
          <button class="btn secondary" type="button" onclick="hideSms()">Înapoi</button>
        </form>

        <form id="codeForm" class="grid" onsubmit="return verifySms(event)" hidden>
          <div class="notice" id="smsNotice">Am trimis codul pe telefonul introdus.</div>
          <div class="field">
            <label for="code">Cod OTP</label>
            <input id="code" name="code" inputmode="numeric" maxlength="6" placeholder="••••••" />
          </div>
          <button class="btn" type="submit">Verifică și continuă</button>
          <button class="btn secondary" type="button" onclick="hideCode()">Schimbă telefonul</button>
        </form>

        <div class="notice">
          {{if .ReturnURL}}La autentificare vei reveni la fluxul OIDC original.{{end}}
        </div>
      </div>
    </section>
  </div>
  <script>
    const returnUrl = {{printf "%q" .ReturnURL}};
    const errorBox = document.getElementById('error');
    const smsForm = document.getElementById('smsForm');
    const codeForm = document.getElementById('codeForm');
    const smsNotice = document.getElementById('smsNotice');
    let phoneValue = '';
    function showError(message) { errorBox.hidden = !message; errorBox.textContent = message || ''; }
    function showSms() { showError(''); smsForm.hidden = false; codeForm.hidden = true; document.getElementById('phone').focus(); }
    function hideSms() { smsForm.hidden = true; showError(''); }
    function hideCode() { codeForm.hidden = true; smsForm.hidden = false; showError(''); }
    async function requestSms(event) {
      event.preventDefault();
      showError('');
      phoneValue = document.getElementById('phone').value.trim();
      if (!phoneValue) { showError('Introdu telefonul.'); return false; }
      const response = await fetch('/api/auth/request-sms', {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({ phone_number: phoneValue })
      });
      if (!response.ok) {
        showError('Nu am putut trimite codul. Încearcă din nou.');
        return false;
      }
      smsNotice.textContent = 'Codul a fost trimis pe ' + phoneValue + '.';
      smsForm.hidden = true;
      codeForm.hidden = false;
      document.getElementById('code').focus();
      return false;
    }
    async function verifySms(event) {
      event.preventDefault();
      showError('');
      const code = document.getElementById('code').value.trim();
      if (!phoneValue || !code) { showError('Completează telefonul și codul.'); return false; }
      const response = await fetch('/api/auth/verify-sms', {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({ phone_number: phoneValue, code: code })
      });
      if (!response.ok) {
        showError('Cod invalid sau expirat.');
        return false;
      }
      window.location.assign(returnUrl);
      return false;
    }
    async function loginPasskey() {
      showError('');
      if (!navigator.credentials || !window.PublicKeyCredential) {
        showError('Browserul nu suportă passkey.');
        return;
      }
      const response = await fetch('/api/passkeys/login-options', { method: 'POST' });
      if (!response.ok) { showError('Backendul nu a putut genera challenge-ul passkey.'); return; }
      const payload = await response.json();
      const opts = payload.options;
      const credential = await navigator.credentials.get({
        publicKey: {
          challenge: b64urlToBuf(opts.challenge),
          rpId: opts.rp.id,
          timeout: opts.timeout,
          userVerification: opts.userVerification,
          allowCredentials: (opts.allowCredentials || []).map((entry) => ({ type: entry.type, id: b64urlToBuf(entry.id) })),
        }
      });
      if (!(credential instanceof PublicKeyCredential)) { showError('Autentificarea passkey a fost anulată.'); return; }
      const assertion = credential.response;
      const finish = await fetch('/api/passkeys/login-finish', {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({
          challenge: opts.challenge,
          credential_id: credential.id,
          response: {
            clientDataJSON: bufToB64url(assertion.clientDataJSON),
            authenticatorData: bufToB64url(assertion.authenticatorData),
            signature: bufToB64url(assertion.signature),
            userHandle: assertion.userHandle ? bufToB64url(assertion.userHandle) : '',
            type: credential.type,
          }
        })
      });
      if (!finish.ok) { showError('Passkey respins sau invalid.'); return; }
      window.location.assign(returnUrl);
    }
    function b64urlToBuf(value) {
      const base64 = value.replace(/-/g, '+').replace(/_/g, '/').padEnd(Math.ceil(value.length / 4) * 4, '=');
      const binary = atob(base64);
      const bytes = new Uint8Array(binary.length);
      for (let i = 0; i < binary.length; i++) bytes[i] = binary.charCodeAt(i);
      return bytes.buffer;
    }
    function bufToB64url(buffer) {
      const bytes = new Uint8Array(buffer);
      let binary = '';
      for (const byte of bytes) binary += String.fromCharCode(byte);
      return btoa(binary).replace(/\+/g, '-').replace(/\//g, '_').replace(/=+$/g, '');
    }
  </script>
</body>
</html>`

const oidcConsentHTML = `<!DOCTYPE html>
<html lang="ro">
<head>
  <meta charset="UTF-8">
  <meta name="viewport" content="width=device-width, initial-scale=1.0">
  <title>{{.CustomerName}} - Consimțământ</title>
  <style>
    *,*::before,*::after{box-sizing:border-box;margin:0;padding:0}
    html,body{height:100%}
    :root{color-scheme:light;--bg:#fff7f8;--card:#fff;--soft:#fff1f2;--border:#e2e8f0;--text:#0f172a;--muted:#64748b;--primary:#e11d48;--primary-contrast:#fff;--shadow:0 24px 60px rgba(15,23,42,.14)}
    html.dark,html.app-dark{color-scheme:dark;--bg:#020617;--card:#0f172a;--soft:#1e293b;--border:#334155;--text:#f8fafc;--muted:#cbd5e1;--primary:#f43f5e;--primary-contrast:#fff;--shadow:0 24px 60px rgba(0,0,0,.35)}
    body{font-family:Inter,system-ui,-apple-system,sans-serif;background:radial-gradient(circle at top left,rgba(225,29,72,.16),transparent 28rem),linear-gradient(135deg,var(--bg),#fff 44%,#ffe4e9);color:var(--text);min-height:100%;display:grid;place-items:center;padding:24px}
    .card{width:100%;max-width:560px;padding:24px;border:1px solid var(--border);border-radius:18px;background:var(--card);box-shadow:var(--shadow)}
    .header{text-align:center;margin-bottom:18px}
    .badge{display:inline-flex;padding:8px 12px;border-radius:999px;border:1px solid var(--border);background:var(--soft);color:var(--primary);font-size:12px;font-weight:700;letter-spacing:.08em;text-transform:uppercase}
    h1{margin-top:10px;font-size:20px}
    .subtitle{margin-top:4px;font-size:13px;color:var(--muted);line-height:1.6}
    .grid{display:grid;gap:8px}
    .client{margin:16px 0;padding:12px;border-radius:12px;border:1px solid var(--border);background:var(--soft)}
    .scope{display:flex;align-items:flex-start;gap:10px;padding:12px;border:1px solid var(--border);border-radius:12px;margin-bottom:8px}
    .scope input{accent-color:var(--primary);margin-top:4px}
    .scope strong{display:block;font-size:14px}
    .scope span{display:block;font-size:12px;color:var(--muted);margin-top:2px}
    .notice{margin-top:12px;padding:10px 12px;border-radius:12px;background:var(--soft);border:1px solid var(--border);font-size:12px;line-height:1.5}
    .row{display:grid;gap:10px;grid-template-columns:repeat(2,minmax(0,1fr));margin-top:16px}
    .btn{display:inline-flex;align-items:center;justify-content:center;width:100%;padding:11px 16px;border:none;border-radius:10px;background:var(--primary);color:var(--primary-contrast);font:inherit;font-weight:700;cursor:pointer}
    .btn.secondary{background:transparent;border:1px solid var(--border);color:var(--primary)}
    .error{margin-bottom:14px;padding:10px 12px;border-radius:12px;background:#fef2f2;border:1px solid #fecaca;color:#b91c1c;font-size:12px}
  </style>
</head>
<body>
  <div class="card">
    <div class="header">
      <div class="badge">Consimțământ OIDC</div>
      <h1>{{.CustomerName}}</h1>
      <p class="subtitle">Confirmă ce date și ce acces acordi aplicației solicitante.</p>
    </div>
    {{if .Error}}<div class="error">{{.Error}}</div>{{end}}
    <div class="client"><strong>{{.ClientName}}</strong><div class="subtitle">Solicită acces prin OIDC pentru acest cont.</div></div>
    <form method="post" action="/api/oidc/consent/decision">
      <input type="hidden" name="request_id" value="{{.RequestID}}">
      <div class="grid">
        {{range .Scopes}}
        <label class="scope">
          <input type="checkbox" name="granted_scopes" value="{{.Code}}" {{if .Required}}checked disabled{{else}}checked{{end}}>
          <span>
            <strong>{{.Label}}</strong>
            <span>{{.Code}}</span>
          </span>
        </label>
        {{end}}
      </div>
      <div class="notice">Cererea este valabilă până la {{.ExpiresAt}} și este gestionată de providerul OIDC, nu de aplicația frontend.</div>
      <div class="row">
        <button class="btn secondary" type="submit" name="decision" value="deny">Refuză</button>
        <button class="btn" type="submit" name="decision" value="allow">Permite</button>
      </div>
    </form>
  </div>
</body>
</html>`

const oidcLogoutHTML = `<!DOCTYPE html>
<html lang="ro">
<head>
  <meta charset="UTF-8">
  <meta name="viewport" content="width=device-width, initial-scale=1.0">
  <title>{{.CustomerName}} - Deconectare</title>
  <style>
    *,*::before,*::after{box-sizing:border-box;margin:0;padding:0}
    html,body{height:100%}
    :root{color-scheme:light;--bg:#fff7f8;--card:#fff;--border:#e2e8f0;--text:#0f172a;--muted:#64748b;--primary:#e11d48;--shadow:0 24px 60px rgba(15,23,42,.14)}
    html.dark,html.app-dark{color-scheme:dark;--bg:#020617;--card:#0f172a;--border:#334155;--text:#f8fafc;--muted:#cbd5e1;--primary:#f43f5e;--shadow:0 24px 60px rgba(0,0,0,.35)}
    body{font-family:Inter,system-ui,-apple-system,sans-serif;background:radial-gradient(circle at top left,rgba(225,29,72,.16),transparent 28rem),linear-gradient(135deg,var(--bg),#fff 44%,#ffe4e9);color:var(--text);min-height:100%;display:grid;place-items:center;padding:24px}
    .card{width:100%;max-width:420px;padding:24px;border:1px solid var(--border);border-radius:18px;background:var(--card);box-shadow:var(--shadow);text-align:center}
    .icon{width:64px;height:64px;border-radius:50%;margin:0 auto 16px;background:color-mix(in srgb,var(--primary) 12%,var(--card));display:flex;align-items:center;justify-content:center;color:var(--primary)}
    h1{font-size:20px;margin-bottom:8px}
    p{font-size:14px;line-height:1.6;color:var(--muted);margin-bottom:20px}
    .actions{display:grid;gap:10px}
    .btn{display:flex;align-items:center;justify-content:center;width:100%;padding:11px 16px;border:none;border-radius:10px;background:var(--primary);color:#fff;font:inherit;font-weight:700;cursor:pointer;text-decoration:none}
    .btn.secondary{background:transparent;border:1px solid var(--border);color:var(--primary)}
  </style>
</head>
<body>
  <div class="card">
    <div class="icon">⟲</div>
    <h1>Sesiune închisă</h1>
    <p>Te-ai deconectat din providerul OIDC. Poți reveni în aplicație sau închide această pagină.</p>
    <div class="actions">
      <a class="btn" href="{{.ReturnTo}}">Înapoi în aplicație</a>
      <button type="button" class="btn secondary" onclick="window.close()">Închide pagina</button>
    </div>
  </div>
</body>
</html>`
