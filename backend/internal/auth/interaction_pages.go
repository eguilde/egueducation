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
	ProjectTitle string
	ReturnURL    string
	Error        string
	Phone        string
	MaskedPhone  string
	SMSAvailable bool
	PasskeyAvailable bool
	WalletAvailable bool
}

type oidcConsentPageData struct {
	CustomerName string
	ProjectTitle string
	RequestID     string
	ClientName    string
	Scopes        []ConsentScope
	ExpiresAt     string
	Error         string
}

type oidcLogoutPageData struct {
	CustomerName string
	ProjectTitle string
	ReturnTo     string
}

func projectTitleFromHostname(host string) string {
	host = strings.TrimSpace(strings.ToLower(host))
	if host == "" {
		return "EguEducation"
	}
	if i := strings.Index(host, ":"); i >= 0 {
		host = host[:i]
	}
	if host == "localhost" || host == "127.0.0.1" || host == "[::1]" {
		return "EguEducation"
	}

	labels := strings.Split(host, ".")
	tenant := ""
	for _, label := range labels {
		label = strings.TrimSpace(label)
		if label == "" || label == "www" || label == "app" {
			continue
		}
		tenant = label
		break
	}
	if tenant == "" {
		return "EguEducation"
	}

	parts := strings.FieldsFunc(tenant, func(r rune) bool {
		return r == '-' || r == '_'
	})
	if len(parts) == 0 {
		parts = []string{tenant}
	}
	for i, part := range parts {
		if part == "" {
			continue
		}
		parts[i] = strings.ToUpper(part[:1]) + strings.ToLower(part[1:])
	}
	return strings.Join(parts, " ")
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

	projectTitle := projectTitleFromHostname(r.Host)
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	_ = template.Must(template.New("oidc-login").Parse(oidcLoginHTML)).Execute(w, oidcLoginPageData{
		CustomerName: "EguEducation",
		ProjectTitle: projectTitle,
		ReturnURL:    returnURL,
		SMSAvailable: s.cfg.EnableSMSOTP && s.smsService != nil && s.smsService.Configured(),
		PasskeyAvailable: s.cfg.EnablePasskeys,
		WalletAvailable: s.cfg.EnableWallet,
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
			ProjectTitle: projectTitleFromHostname(r.Host),
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
		ProjectTitle: projectTitleFromHostname(r.Host),
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
		ProjectTitle: projectTitleFromHostname(r.Host),
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
  <title>{{.ProjectTitle}} - Autentificare</title>
  <style>
    *,*::before,*::after{box-sizing:border-box;margin:0;padding:0}
    html,body{height:100%}
    :root{color-scheme:light;--bg:#fff7f8;--card:#fff;--card-2:#fff8fa;--soft:#fff1f5;--border:#e2e8f0;--text:#0f172a;--muted:#64748b;--muted-2:#94a3b8;--primary:#e11d48;--primary-contrast:#fff;--shadow:0 28px 72px rgba(15,23,42,.16);--shadow-2:0 18px 36px rgba(225,29,72,.14)}
    html.dark,html.app-dark{color-scheme:dark;--bg:#020617;--card:#0f172a;--card-2:#111c31;--soft:#1e293b;--border:#334155;--text:#f8fafc;--muted:#cbd5e1;--muted-2:#94a3b8;--primary:#f43f5e;--primary-contrast:#fff;--shadow:0 28px 72px rgba(0,0,0,.38);--shadow-2:0 18px 36px rgba(244,63,94,.18)}
    body{font-family:'Inter Variable','Inter',ui-sans-serif,system-ui,-apple-system,sans-serif;background:radial-gradient(circle at top left,rgba(225,29,72,.16),transparent 28rem),radial-gradient(circle at bottom right,rgba(248,113,113,.10),transparent 24rem),linear-gradient(135deg,var(--bg),#fff 45%,#ffe4ec);color:var(--text);min-height:100%;overflow:hidden}
    .shell{min-height:100vh;display:grid;grid-template-columns:minmax(380px,1fr) minmax(420px,680px)}
    .visual{position:relative;display:flex;align-items:flex-end;min-height:100vh;padding:clamp(40px,6vw,84px);background:linear-gradient(135deg,rgba(15,23,42,.94),rgba(225,29,72,.84) 42%,rgba(8,47,73,.56));color:#fff;overflow:hidden}
    .visual::before,.visual::after{content:'';position:absolute;border-radius:999px;pointer-events:none}
    .visual::before{width:24rem;height:24rem;right:-6rem;top:-5rem;background:radial-gradient(circle,rgba(255,255,255,.18),transparent 68%)}
    .visual::after{width:18rem;height:18rem;left:-4rem;bottom:-4rem;background:radial-gradient(circle,rgba(255,255,255,.12),transparent 70%)}
    .visual-inner{position:relative;z-index:1;max-width:560px}
    .eyebrow{display:inline-flex;align-items:center;gap:8px;padding:9px 14px;border-radius:999px;border:1px solid rgba(255,255,255,.18);background:rgba(255,255,255,.08);backdrop-filter:blur(14px);color:#fff;font-size:12px;font-weight:800;letter-spacing:.12em;text-transform:uppercase}
    .hero-title{font-size:clamp(34px,4.8vw,64px);line-height:1.02;margin:18px 0 18px;letter-spacing:-.045em}
    .hero-copy{max-width:48ch;font-size:16px;line-height:1.7;color:rgba(255,255,255,.88)}
    .feature-row{display:flex;flex-wrap:wrap;gap:10px;margin-top:28px}
    .feature{display:inline-flex;align-items:center;gap:8px;padding:10px 14px;border-radius:999px;background:rgba(255,255,255,.10);border:1px solid rgba(255,255,255,.12);backdrop-filter:blur(10px);font-size:13px;font-weight:700;color:#fff}
    .panel{display:flex;align-items:center;justify-content:center;padding:clamp(18px,3vw,48px);overflow-y:auto}
    .card{width:100%;max-width:720px;padding:clamp(18px,2.8vw,28px);border:1px solid var(--border);border-radius:28px;background:linear-gradient(180deg,var(--card),var(--card-2));box-shadow:var(--shadow)}
    .card-shell{display:grid;grid-template-columns:1.02fr .98fr;gap:18px;align-items:start}
    .card-hero{padding:22px;border-radius:22px;background:radial-gradient(circle at top right,rgba(225,29,72,.12),transparent 45%),linear-gradient(180deg,var(--card),var(--soft));border:1px solid var(--border)}
    .card-header{display:grid;gap:10px}
    .brand-line{display:flex;align-items:center;justify-content:space-between;gap:12px;flex-wrap:wrap}
    .brand-pill{display:inline-flex;align-items:center;gap:8px;padding:8px 12px;border-radius:999px;border:1px solid var(--border);background:var(--soft);color:var(--primary);font-size:11px;font-weight:800;letter-spacing:.11em;text-transform:uppercase}
    .status-pill{display:inline-flex;align-items:center;gap:8px;padding:8px 12px;border-radius:999px;background:rgba(225,29,72,.08);border:1px solid rgba(225,29,72,.16);font-size:12px;font-weight:700;color:var(--primary)}
    .title{font-size:22px;line-height:1.15;letter-spacing:-.03em}
    .subtitle{font-size:14px;color:var(--muted);line-height:1.65}
    .summary{display:grid;gap:10px;margin-top:18px}
    .summary-item{display:flex;align-items:flex-start;gap:10px;padding:12px 14px;border-radius:16px;background:rgba(255,255,255,.65);border:1px solid var(--border)}
    html.dark .summary-item,html.app-dark .summary-item{background:rgba(15,23,42,.72)}
    .summary-icon{display:inline-flex;align-items:center;justify-content:center;width:30px;height:30px;border-radius:10px;background:var(--soft);color:var(--primary);flex:0 0 auto}
    .summary-item strong{display:block;font-size:13px;margin-bottom:3px}
    .summary-item span{display:block;font-size:12px;color:var(--muted)}
    .summary-item small{display:block;font-size:12px;color:var(--muted)}
    .card-form{padding:22px;border-radius:22px;background:var(--card);border:1px solid var(--border);box-shadow:var(--shadow-2)}
    .section-title{font-size:12px;font-weight:800;letter-spacing:.12em;text-transform:uppercase;color:var(--primary);margin-bottom:8px}
    .grid{display:grid;gap:12px}
    .method{display:flex;align-items:flex-start;gap:12px;width:100%;padding:14px;border-radius:18px;border:1px solid var(--border);background:linear-gradient(180deg,var(--card),var(--card-2));cursor:pointer;transition:transform .18s ease,box-shadow .18s ease,border-color .18s ease,background .18s ease;text-align:left}
    .method:hover{transform:translateY(-1px);box-shadow:0 18px 32px rgba(15,23,42,.10);border-color:rgba(225,29,72,.42)}
    .method.active{border-color:rgba(225,29,72,.56);box-shadow:0 18px 32px rgba(225,29,72,.12);background:linear-gradient(180deg,rgba(225,29,72,.06),rgba(225,29,72,.02))}
    .method:disabled{cursor:not-allowed;opacity:.55;transform:none;box-shadow:none}
    .method-icon{display:inline-flex;align-items:center;justify-content:center;width:42px;height:42px;border-radius:14px;background:var(--soft);color:var(--primary);flex:0 0 auto}
    .method-body{display:grid;gap:4px}
    .method strong{font-size:15px}
    .method span{font-size:12px;color:var(--muted);line-height:1.55}
    .method-kicker{font-size:11px;font-weight:800;letter-spacing:.11em;text-transform:uppercase;color:var(--primary)}
    .field{display:grid;gap:6px}
    .field label{font-size:13px;font-weight:700}
    .field input{width:100%;padding:14px 14px;border:1px solid var(--border);border-radius:14px;font:inherit;color:var(--text);background:var(--soft);transition:border-color .18s ease,box-shadow .18s ease,background .18s ease}
    .field input::placeholder{color:var(--muted-2)}
    .field input:focus{outline:none;border-color:var(--primary);box-shadow:0 0 0 4px rgba(225,29,72,.14);background:var(--card)}
    .field-help{font-size:12px;color:var(--muted);line-height:1.55}
    .toolbar{display:flex;gap:10px;flex-wrap:wrap}
    .btn{display:inline-flex;align-items:center;justify-content:center;gap:8px;width:100%;padding:13px 16px;border:none;border-radius:14px;background:linear-gradient(180deg,var(--primary),#be123c);color:var(--primary-contrast);font:inherit;font-weight:800;cursor:pointer;text-decoration:none;box-shadow:0 16px 26px rgba(225,29,72,.22);transition:transform .18s ease,box-shadow .18s ease,filter .18s ease}
    .btn:hover{transform:translateY(-1px);filter:saturate(1.02)}
    .btn.secondary{background:transparent;border:1px solid var(--border);color:var(--primary);box-shadow:none}
    .btn:disabled{cursor:not-allowed;opacity:.55;transform:none;box-shadow:none}
    .row{display:grid;gap:10px;grid-template-columns:repeat(2,minmax(0,1fr))}
    .helper-card{margin-top:12px;padding:12px 14px;border-radius:16px;background:var(--soft);border:1px solid var(--border);font-size:12px;line-height:1.6;color:var(--muted)}
    .error{margin-bottom:14px;padding:12px 14px;border-radius:16px;background:#fef2f2;border:1px solid #fecaca;color:#b91c1c;font-size:13px;line-height:1.6}
    .success{margin-bottom:14px;padding:12px 14px;border-radius:16px;background:#f0fdf4;border:1px solid #bbf7d0;color:#166534;font-size:13px;line-height:1.6}
    .step{display:grid;gap:12px;margin-top:14px}
    .step-header{display:flex;align-items:center;justify-content:space-between;gap:12px}
    .step-header strong{font-size:14px}
    .step-header span{font-size:12px;color:var(--muted)}
    .otp{display:grid;gap:12px}
    .otp-grid{display:grid;grid-template-columns:repeat(6,minmax(0,1fr));gap:8px}
    .otp-box{width:100%;min-width:0;padding:14px 10px;border:1px solid var(--border);border-radius:14px;font:inherit;font-size:18px;letter-spacing:.2em;text-align:center;color:var(--text);background:var(--soft)}
    .otp-box:focus{outline:none;border-color:var(--primary);box-shadow:0 0 0 4px rgba(225,29,72,.14);background:var(--card)}
    .otp-box.filled{border-color:rgba(225,29,72,.35)}
    .footer-note{margin-top:14px;padding:12px 14px;border-radius:16px;background:rgba(2,6,23,.03);border:1px solid var(--border);font-size:12px;line-height:1.6;color:var(--muted)}
    .footer-note strong{color:var(--text)}
    .method-state{display:none}
    .method-state.is-active{display:grid}
    @media(max-width:1120px){.shell{grid-template-columns:1fr}.visual{min-height:auto}.panel{padding:18px 16px 24px}.card{max-width:760px}}
    @media(max-width:860px){body{overflow-y:auto}.shell{display:block}.visual{display:none}.card-shell{grid-template-columns:1fr}.card{border-radius:24px}.card-form,.card-hero{padding:18px}}
    @media(max-width:520px){.panel{padding:0}.card{min-height:100vh;max-width:none;border:0;border-radius:0;box-shadow:none}.row{grid-template-columns:1fr}.method{padding:13px}.title{font-size:20px}.hero-title{font-size:32px}.card-hero,.card-form{border-radius:20px}.otp-grid{gap:6px}.otp-box{padding:12px 8px;font-size:16px}}
  </style>
</head>
<body>
  <div class="shell">
    <section class="visual">
      <div class="visual-inner">
        <div class="eyebrow">OIDC Provider</div>
        <h2 class="hero-title">Autentificare sigură pentru {{.ProjectTitle}}</h2>
        <p class="hero-copy">Accesul este gestionat complet de providerul OIDC, cu SMS OTP, passkey, consimțământ separat și sesiuni securizate în backend.</p>
        <div class="feature-row" aria-label="Capabilități autentificare">
          <span class="feature">SMS OTP</span>
          <span class="feature">Passkey</span>
          <span class="feature">EUDI Wallet</span>
          <span class="feature">Consent flow</span>
        </div>
      </div>
    </section>
    <section class="panel">
      <div class="card">
        <div class="card-shell">
          <aside class="card-hero">
            <div class="card-header">
              <div class="brand-line">
                <div class="brand-pill">Autentificare</div>
                <span class="status-pill">Sesiune OIDC activă</span>
              </div>
              <h1 class="title">{{.ProjectTitle}}</h1>
              <p class="subtitle">Alege metoda de autentificare și continuă în aplicație fără parolă.</p>
            </div>
            <div class="summary" aria-label="Avantaje autentificare">
              <div class="summary-item"><span class="summary-icon">1</span><div><strong>SMS OTP și passkey</strong><span>Metode moderne, simple și fără parolă.</span></div></div>
              <div class="summary-item"><span class="summary-icon">2</span><div><strong>Consimțământ în OIDC</strong><span>Fluxul de aprobare rămâne în providerul backend.</span></div></div>
              <div class="summary-item"><span class="summary-icon">3</span><div><strong>Întoarcere sigură</strong><span>Revii la exact cererea inițială după autentificare.</span></div></div>
              <div class="summary-item"><span class="summary-icon">4</span><div><strong>Canale disponibile</strong><span>SMS: {{if .SMSAvailable}}activ{{else}}indisponibil{{end}} · Passkey: {{if .PasskeyAvailable}}activ{{else}}indisponibil{{end}} · Wallet: {{if .WalletAvailable}}activ{{else}}indisponibil{{end}}</span></div></div>
            </div>
          </aside>
          <section class="card-form">
            <div class="section-title">Metode de acces</div>
            <div id="error" class="error" hidden role="alert" aria-live="polite"></div>
            <div id="success" class="success" hidden role="status" aria-live="polite"></div>
            <div class="grid" id="methodGrid">
              <button class="method" type="button" onclick="showSms(this)" {{if not .SMSAvailable}}disabled{{end}}><span class="method-icon">✉</span><span class="method-body"><span class="method-kicker">Recomandat</span><strong>SMS OTP</strong><span>Introdu numele de utilizator, apoi confirmă codul primit prin SMS.</span></span></button>
              <button class="method" type="button" onclick="loginPasskey(this)" {{if not .PasskeyAvailable}}disabled{{end}}><span class="method-icon">◈</span><span class="method-body"><span class="method-kicker">Fără cod</span><strong>Passkey</strong><span>Folosește cheia de acces din dispozitiv pentru autentificare rapidă și sigură.</span></span></button>
              <button class="method" type="button" onclick="showWalletMessage(this)" {{if not .WalletAvailable}}disabled{{end}}><span class="method-icon">⌁</span><span class="method-body"><span class="method-kicker">Identitate digitală</span><strong>EUDI Wallet</strong><span>{{if .WalletAvailable}}Continuă cu wallet-ul digital compatibil eIDAS.{{else}}Fluxul de wallet digital nu este încă activ în această instanță.{{end}}</span></span></button>
            </div>
            <div class="step">
              <div id="methodIntro" class="helper-card">Alege una dintre cele 3 metode de autentificare pentru a continua fluxul OIDC.</div>
              <form id="smsForm" class="grid" onsubmit="return requestSms(event)" hidden>
                <div class="step-header"><strong>Autentificare SMS</strong><span>Pasul 1 din 2</span></div>
                <div class="field"><label for="identifier">Nume de utilizator</label><input id="identifier" name="identifier" autocomplete="username" inputmode="text" placeholder="thomasgalambos" /><span class="field-help">Introdu numele de utilizator al contului. Codul SMS va fi trimis către numărul verificat asociat contului.</span></div>
                <div class="toolbar"><button class="btn secondary" type="button" onclick="resetToMethods()">Alege altă metodă</button><button class="btn" type="submit">Trimite codul OTP</button></div>
              </form>
              <form id="codeForm" class="grid otp" onsubmit="return verifySms(event)" hidden>
                <div class="step-header"><strong>Confirmă codul OTP</strong><span>Pasul 2 din 2</span></div>
                <div class="helper-card" id="smsNotice">Am trimis codul SMS către numărul verificat din cont.</div>
                <input type="hidden" id="code" name="code" />
                <div class="otp-grid" aria-label="Cod OTP">
                  <input class="otp-box" type="text" inputmode="numeric" maxlength="1" autocomplete="one-time-code" autofocus />
                  <input class="otp-box" type="text" inputmode="numeric" maxlength="1" />
                  <input class="otp-box" type="text" inputmode="numeric" maxlength="1" />
                  <input class="otp-box" type="text" inputmode="numeric" maxlength="1" />
                  <input class="otp-box" type="text" inputmode="numeric" maxlength="1" />
                  <input class="otp-box" type="text" inputmode="numeric" maxlength="1" />
                </div>
                <span class="field-help">Introdu cele 6 cifre primite prin SMS.</span>
                <div class="row"><button class="btn secondary" type="button" onclick="hideCode()">Schimbă identificatorul</button><button class="btn" type="submit" id="verifyBtn" disabled>Verifică și continuă</button></div>
              </form>
            </div>
            <div class="footer-note">{{if .ReturnURL}}După autentificare vei reveni automat la cererea OIDC inițială.{{else}}Sesiunea este gestionată exclusiv de providerul OIDC.{{end}}</div>
          </section>
        </div>
      </div>
    </section>
  </div>
  <script>
    const returnUrl = {{printf "%q" .ReturnURL}};
    const errorBox = document.getElementById('error');
    const successBox = document.getElementById('success');
    const methodGrid = document.getElementById('methodGrid');
    const methodIntro = document.getElementById('methodIntro');
    const smsForm = document.getElementById('smsForm');
    const codeForm = document.getElementById('codeForm');
    const smsNotice = document.getElementById('smsNotice');
    const identifierInput = document.getElementById('identifier');
    const codeInput = document.getElementById('code');
    const verifyBtn = document.getElementById('verifyBtn');
    const otpBoxes = Array.from(document.querySelectorAll('.otp-box'));
    let identifierValue = '';
    function showError(message){errorBox.hidden=!message;errorBox.textContent=message||'';if(message){successBox.hidden=true;successBox.textContent='';}}
    function showSuccess(message){successBox.hidden=!message;successBox.textContent=message||'';if(message){errorBox.hidden=true;errorBox.textContent='';}}
    function setActiveMethod(button){for(const item of methodGrid.querySelectorAll('.method')){item.classList.remove('active');}if(button){button.classList.add('active');}}
    function showSms(button){setActiveMethod(button);showError('');showSuccess('');methodIntro.hidden=true;smsForm.hidden=false;codeForm.hidden=true;identifierInput.focus();}
    function resetToMethods(){smsForm.hidden=true;codeForm.hidden=true;methodIntro.hidden=false;showError('');showSuccess('');for(const item of methodGrid.querySelectorAll('.method')){item.classList.remove('active');}}
    function hideCode(){resetToMethods();identifierValue='';codeInput.value='';otpBoxes.forEach((box)=>{box.value='';box.classList.remove('filled');});verifyBtn.disabled=true;}
    function showWalletMessage(button){setActiveMethod(button);smsForm.hidden=true;codeForm.hidden=true;methodIntro.hidden=false;showSuccess('Continuă cu EUDI Wallet după activarea fluxului dedicat pentru acest tenant.');showError('');}
    function syncOtp(){const code=otpBoxes.map((box)=>box.value.replace(/\D/g,'').slice(0,1)).join('');codeInput.value=code;verifyBtn.disabled=code.length!==6;otpBoxes.forEach((box)=>box.classList.toggle('filled',box.value!=='');)}
    async function requestSms(event){event.preventDefault();showError('');showSuccess('');identifierValue=document.getElementById('identifier').value.trim();if(!identifierValue){showError('Introdu numele de utilizator.');return false;}if(!{{if .SMSAvailable}}true{{else}}false{{end}}){showError('Autentificarea SMS nu este activată în această instanță.');return false;}const response=await fetch('/api/auth/request-sms',{method:'POST',headers:{'Content-Type':'application/json'},body:JSON.stringify({identifier:identifierValue})});if(!response.ok){showError('Nu am putut trimite codul. Încearcă din nou.');return false;}smsNotice.textContent='Am trimis codul SMS către contul asociat identificatorului introdus.';smsForm.hidden=true;codeForm.hidden=false;otpBoxes[0].focus();showSuccess('Codul a fost trimis. Verifică mesajul SMS și continuă.');return false;}
    async function verifySms(event){event.preventDefault();showError('');showSuccess('');const code=document.getElementById('code').value.trim();if(!identifierValue||!code){showError('Completează identificatorul și codul.');return false;}const response=await fetch('/api/auth/verify-sms',{method:'POST',headers:{'Content-Type':'application/json'},body:JSON.stringify({identifier:identifierValue,code:code})});if(!response.ok){showError('Cod invalid sau expirat.');return false;}showSuccess('Autentificare reușită. Se redirecționează către aplicație...');window.location.assign(returnUrl);return false;}
    async function loginPasskey(button){setActiveMethod(button);showError('');showSuccess('');if(!{{if .PasskeyAvailable}}true{{else}}false{{end}}){showError('Autentificarea cu passkey nu este activată în această instanță.');return;}if(!navigator.credentials||!window.PublicKeyCredential){showError('Browserul nu suportă passkey.');return;}const response=await fetch('/api/passkeys/login-options',{method:'POST'});if(!response.ok){showError('Backendul nu a putut genera challenge-ul passkey.');return;}const payload=await response.json();const opts=payload.options;const credential=await navigator.credentials.get({publicKey:{challenge:b64urlToBuf(opts.challenge),rpId:opts.rp.id,timeout:opts.timeout,userVerification:opts.userVerification,allowCredentials:(opts.allowCredentials||[]).map((entry)=>({type:entry.type,id:b64urlToBuf(entry.id)}))}});if(!(credential instanceof PublicKeyCredential)){showError('Autentificarea passkey a fost anulată.');return;}const assertion=credential.response;const finish=await fetch('/api/passkeys/login-finish',{method:'POST',headers:{'Content-Type':'application/json'},body:JSON.stringify({challenge:opts.challenge,credential_id:credential.id,response:{clientDataJSON:bufToB64url(assertion.clientDataJSON),authenticatorData:bufToB64url(assertion.authenticatorData),signature:bufToB64url(assertion.signature),userHandle:assertion.userHandle?bufToB64url(assertion.userHandle):'',type:credential.type}})});if(!finish.ok){showError('Passkey respins sau invalid.');return;}showSuccess('Passkey verificat. Se redirecționează către aplicație...');window.location.assign(returnUrl);}
    function b64urlToBuf(value){const base64=value.replace(/-/g,'+').replace(/_/g,'/').padEnd(Math.ceil(value.length/4)*4,'=');const binary=atob(base64);const bytes=new Uint8Array(binary.length);for(let i=0;i<binary.length;i++)bytes[i]=binary.charCodeAt(i);return bytes.buffer;}
    function bufToB64url(buffer){const bytes=new Uint8Array(buffer);let binary='';for(const byte of bytes)binary+=String.fromCharCode(byte);return btoa(binary).replace(/\+/g,'-').replace(/\//g,'_').replace(/=+$/g,'');}
    otpBoxes.forEach((box,index)=>{box.addEventListener('input',(event)=>{const value=event.target.value.replace(/\D/g,'').slice(0,1);event.target.value=value;if(value&&index<otpBoxes.length-1){otpBoxes[index+1].focus();}syncOtp();});box.addEventListener('keydown',(event)=>{if(event.key==='Backspace'&&!box.value&&index>0){otpBoxes[index-1].value='';otpBoxes[index-1].focus();syncOtp();}if(event.key==='ArrowLeft'&&index>0){otpBoxes[index-1].focus();}if(event.key==='ArrowRight'&&index<otpBoxes.length-1){otpBoxes[index+1].focus();}if(event.key==='Enter'&&codeInput.value.length===6){codeForm.requestSubmit();}});box.addEventListener('paste',(event)=>{event.preventDefault();const paste=(event.clipboardData||window.clipboardData).getData('text').replace(/\D/g,'').slice(0,6);if(!paste){return;}paste.split('').forEach((digit,digitIndex)=>{if(otpBoxes[digitIndex]){otpBoxes[digitIndex].value=digit;}});otpBoxes[Math.min(paste.length,otpBoxes.length-1)].focus();syncOtp();});});
    syncOtp();
  </script>
</body>
</html>`
const oidcConsentHTML = `<!DOCTYPE html>
<html lang="ro">
<head>
  <meta charset="UTF-8">
  <meta name="viewport" content="width=device-width, initial-scale=1.0">
  <title>{{.ProjectTitle}} - Consimțământ</title>
  <style>
    *,*::before,*::after{box-sizing:border-box;margin:0;padding:0}
    html,body{height:100%}
    :root{color-scheme:light;--bg:#fff7f8;--card:#fff;--card-2:#fff8fa;--soft:#fff1f5;--soft-2:#ffe4ec;--border:#e2e8f0;--text:#0f172a;--muted:#64748b;--muted-2:#94a3b8;--primary:#e11d48;--primary-contrast:#fff;--shadow:0 28px 72px rgba(15,23,42,.16);--shadow-2:0 18px 36px rgba(225,29,72,.14)}
    html.dark,html.app-dark{color-scheme:dark;--bg:#020617;--card:#0f172a;--card-2:#111c31;--soft:#1e293b;--soft-2:#31243b;--border:#334155;--text:#f8fafc;--muted:#cbd5e1;--muted-2:#94a3b8;--primary:#f43f5e;--primary-contrast:#fff;--shadow:0 28px 72px rgba(0,0,0,.38);--shadow-2:0 18px 36px rgba(244,63,94,.18)}
    body{font-family:Inter,system-ui,-apple-system,sans-serif;background:radial-gradient(circle at top left,rgba(225,29,72,.16),transparent 28rem),radial-gradient(circle at bottom right,rgba(248,113,113,.10),transparent 24rem),linear-gradient(135deg,var(--bg),#fff 45%,#ffe4ec);color:var(--text);min-height:100%;overflow:hidden}
    .shell{min-height:100vh;display:grid;grid-template-columns:minmax(360px,1fr) minmax(420px,680px)}
    .visual{display:flex;align-items:flex-end;min-height:100vh;padding:clamp(40px,6vw,84px);background:linear-gradient(135deg,rgba(15,23,42,.94),rgba(225,29,72,.84) 42%,rgba(8,47,73,.56));color:#fff;position:relative;overflow:hidden}
    .visual::before,.visual::after{content:"";position:absolute;border-radius:999px;pointer-events:none}
    .visual::before{width:24rem;height:24rem;right:-6rem;top:-5rem;background:radial-gradient(circle,rgba(255,255,255,.18),transparent 68%)}
    .visual::after{width:18rem;height:18rem;left:-4rem;bottom:-4rem;background:radial-gradient(circle,rgba(255,255,255,.12),transparent 72%)}
    .visual-inner{position:relative;z-index:1;max-width:560px}
    .eyebrow{display:inline-flex;align-items:center;gap:8px;padding:9px 14px;border-radius:999px;border:1px solid rgba(255,255,255,.18);background:rgba(255,255,255,.08);backdrop-filter:blur(14px);color:#fff;font-size:12px;font-weight:800;letter-spacing:.12em;text-transform:uppercase}
    .hero-title{font-size:clamp(34px,4.8vw,64px);line-height:1.02;margin:18px 0 18px;letter-spacing:-.045em}
    .hero-copy{max-width:48ch;font-size:16px;line-height:1.7;color:rgba(255,255,255,.88)}
    .feature-row{display:flex;flex-wrap:wrap;gap:10px;margin-top:28px}
    .feature{display:inline-flex;align-items:center;gap:8px;padding:10px 14px;border-radius:999px;background:rgba(255,255,255,.10);border:1px solid rgba(255,255,255,.12);backdrop-filter:blur(10px);font-size:13px;font-weight:700;color:#fff}
    .panel{display:flex;align-items:center;justify-content:center;padding:clamp(18px,3vw,48px);overflow-y:auto}
    .card{width:100%;max-width:720px;padding:clamp(18px,2.8vw,28px);border:1px solid var(--border);border-radius:28px;background:linear-gradient(180deg,var(--card),var(--card-2));box-shadow:var(--shadow)}
    .card-shell{display:grid;grid-template-columns:1fr 1.02fr;gap:18px;align-items:start}
    .card-hero{padding:22px;border-radius:22px;background:radial-gradient(circle at top right,rgba(225,29,72,.12),transparent 45%),linear-gradient(180deg,var(--card),var(--soft));border:1px solid var(--border)}
    .card-header{display:grid;gap:10px}
    .brand-line{display:flex;align-items:center;justify-content:space-between;gap:12px;flex-wrap:wrap}
    .brand-pill{display:inline-flex;align-items:center;gap:8px;padding:8px 12px;border-radius:999px;border:1px solid var(--border);background:var(--soft);color:var(--primary);font-size:11px;font-weight:800;letter-spacing:.11em;text-transform:uppercase}
    .status-pill{display:inline-flex;align-items:center;gap:8px;padding:8px 12px;border-radius:999px;background:rgba(225,29,72,.08);border:1px solid rgba(225,29,72,.16);font-size:12px;font-weight:700;color:var(--primary)}
    .title{font-size:22px;line-height:1.15;letter-spacing:-.03em}
    .subtitle{font-size:14px;color:var(--muted);line-height:1.65}
    .summary{display:grid;gap:10px;margin-top:18px}
    .summary-item{display:flex;align-items:flex-start;gap:10px;padding:12px 14px;border-radius:16px;background:rgba(255,255,255,.65);border:1px solid var(--border)}
    html.dark .summary-item,html.app-dark .summary-item{background:rgba(15,23,42,.72)}
    .summary-icon{display:inline-flex;align-items:center;justify-content:center;width:30px;height:30px;border-radius:10px;background:var(--soft);color:var(--primary);flex:0 0 auto}
    .summary-item strong{display:block;font-size:13px;margin-bottom:3px}
    .summary-item span{display:block;font-size:12px;color:var(--muted)}
    .card-form{padding:22px;border-radius:22px;background:var(--card);border:1px solid var(--border);box-shadow:var(--shadow-2)}
    .section-title{font-size:12px;font-weight:800;letter-spacing:.12em;text-transform:uppercase;color:var(--primary);margin-bottom:8px}
    .grid{display:grid;gap:12px}
    .scope-list{display:grid;gap:10px}
    .scope{display:flex;align-items:flex-start;gap:12px;padding:14px;border:1px solid var(--border);border-radius:18px;background:linear-gradient(180deg,var(--card),var(--card-2))}
    .scope input{accent-color:var(--primary);margin-top:4px}
    .scope strong{display:block;font-size:14px}
    .scope span{display:block;font-size:12px;color:var(--muted);margin-top:2px;line-height:1.55}
    .scope .scope-code{font-size:11px;letter-spacing:.08em;text-transform:uppercase;color:var(--muted-2)}
    .client{margin:16px 0 18px;padding:14px;border-radius:18px;border:1px solid var(--border);background:var(--soft)}
    .client-top{display:flex;align-items:flex-start;justify-content:space-between;gap:12px;flex-wrap:wrap}
    .client-name{font-size:16px;font-weight:800}
    .client-meta{font-size:12px;color:var(--muted);line-height:1.6;max-width:42ch}
    .notice{margin-top:14px;padding:12px 14px;border-radius:16px;background:var(--soft);border:1px solid var(--border);font-size:12px;line-height:1.6;color:var(--muted)}
    .row{display:grid;gap:10px;grid-template-columns:repeat(2,minmax(0,1fr));margin-top:18px}
    .btn{display:inline-flex;align-items:center;justify-content:center;width:100%;padding:12px 16px;border:none;border-radius:14px;background:linear-gradient(180deg,var(--primary),#be123c);color:var(--primary-contrast);font:inherit;font-weight:800;cursor:pointer;text-decoration:none;box-shadow:0 16px 26px rgba(225,29,72,.22);transition:transform .18s ease,filter .18s ease}
    .btn:hover{transform:translateY(-1px);filter:saturate(1.02)}
    .btn.secondary{background:transparent;border:1px solid var(--border);color:var(--primary);box-shadow:none}
    .error{margin-bottom:14px;padding:12px 14px;border-radius:16px;background:#fef2f2;border:1px solid #fecaca;color:#b91c1c;font-size:13px;line-height:1.6}
    @media(max-width:1120px){body{overflow-y:auto}.shell{grid-template-columns:1fr}.visual{min-height:auto}.panel{padding:18px 16px 24px}.card{max-width:760px}}
    @media(max-width:860px){.visual{display:none}.card-shell{grid-template-columns:1fr}.card,.card-hero,.card-form{border-radius:24px}}
    @media(max-width:480px){.panel{padding:0}.card{min-height:100vh;max-width:none;border:0;border-radius:0;box-shadow:none}.row{grid-template-columns:1fr}.client-top{display:grid}.title{font-size:20px}.hero-title{font-size:32px}.card-hero,.card-form{padding:18px}}
  </style>
</head>
<body>
  <div class="shell">
    <section class="visual">
      <div class="visual-inner">
        <div class="eyebrow">OIDC Consent</div>
        <h2 class="hero-title">Confirmă accesul cerut de aplicație</h2>
        <p class="hero-copy">Consimțământul rămâne în providerul OIDC. Poți vedea exact ce date sunt cerute și poți aproba sau refuza cererea înainte să revii în aplicație.</p>
        <div class="feature-row" aria-label="Capabilități consimțământ">
          <span class="feature">Scope review</span>
          <span class="feature">User control</span>
          <span class="feature">OIDC native</span>
        </div>
      </div>
    </section>
    <section class="panel">
      <div class="card">
        <div class="card-shell">
          <aside class="card-hero">
            <div class="card-header">
              <div class="brand-line">
                <div class="brand-pill">Consimțământ OIDC</div>
                <span class="status-pill">Cerere activă</span>
              </div>
              <h1 class="title">{{.ProjectTitle}}</h1>
              <p class="subtitle">Verifică ce scope-uri sunt cerute și aprobă doar ce este necesar pentru această sesiune.</p>
            </div>

            <div class="summary" aria-label="Rezumat cerere">
              <div class="summary-item">
                <span class="summary-icon">1</span>
                <div>
                  <strong>Aplicația solicitantă</strong>
                  <span>{{.ClientName}}</span>
                </div>
              </div>
              <div class="summary-item">
                <span class="summary-icon">2</span>
                <div>
                  <strong>Ce va primi aplicația</strong>
                  <span>Poți aproba doar scope-urile relevante pentru această sesiune.</span>
                </div>
              </div>
              <div class="summary-item">
                <span class="summary-icon">3</span>
                <div>
                  <strong>Control complet</strong>
                  <span>Decizia este validată în backend și revii apoi în aplicația inițială.</span>
                </div>
              </div>
              {{if .Scopes}}
              <div class="summary-scopes" aria-label="Scope-uri cerute">
                {{range .Scopes}}
                <div class="summary-scope">
                  <span class="summary-icon">{{if .Required}}✓{{else}}•{{end}}</span>
                  <div>
                    <strong>{{.Label}}</strong>
                    <span>{{.Description}}</span>
                    <code>{{.Code}}</code>
                  </div>
                </div>
                {{end}}
              </div>
              {{end}}
            </div>
          </aside>

          <section class="card-form">
            <div class="section-title">Detalii cerere</div>
            {{if .Error}}<div class="error" role="alert">{{.Error}}</div>{{end}}

            <div class="client">
              <div class="client-top">
                <div>
                  <div class="client-name">{{.ClientName}}</div>
                  <div class="client-meta">Solicită acces prin OIDC pentru acest cont. Confirmarea se aplică doar acestei cereri active.</div>
                </div>
                <div class="badge">OIDC</div>
              </div>
            </div>

            <form method="post" action="/api/oidc/consent/decision">
              <input type="hidden" name="request_id" value="{{.RequestID}}">
              <div class="scope-list">
                {{range .Scopes}}
                <label class="scope">
                  <input type="checkbox" name="granted_scopes" value="{{.Code}}" {{if .Required}}checked disabled{{else}}checked{{end}}>
                  <span>
                    <strong>{{.Label}}</strong>
                    <span>{{.Description}}</span>
                    <span class="scope-code">{{.Code}}</span>
                  </span>
                </label>
                {{end}}
              </div>
              <div class="notice">Prin aprobare, aplicația primește doar scope-urile selectate aici. Cererea este valabilă până la {{.ExpiresAt}} și este gestionată de providerul OIDC, nu de aplicația frontend.</div>
              <div class="row">
                <button class="btn secondary" type="submit" name="decision" value="deny">Refuză accesul</button>
                <button class="btn" type="submit" name="decision" value="allow">Permite accesul</button>
              </div>
            </form>
          </section>
        </div>
      </div>
    </section>
  </div>
</body>
</html>`

const oidcLogoutHTML = `<!DOCTYPE html>
<html lang="ro">
<head>
  <meta charset="UTF-8">
  <meta name="viewport" content="width=device-width, initial-scale=1.0">
  <title>{{.ProjectTitle}} — Deconectare</title>
  <style>
    *,*::before,*::after{box-sizing:border-box;margin:0;padding:0}
    html,body{height:100%}
    :root{color-scheme:light;--oidc-primary-50:#fff1f2;--oidc-primary-100:#ffe4e6;--oidc-primary-500:#f43f5e;--oidc-primary-600:#e11d48;--oidc-surface-0:#fff;--oidc-surface-50:#f8fafc;--oidc-surface-200:#e2e8f0;--oidc-surface-300:#cbd5e1;--oidc-surface-500:#64748b;--oidc-surface-700:#334155;--oidc-surface-800:#1e293b;--oidc-surface-900:#0f172a;--oidc-surface-950:#020617;--oidc-bg:var(--oidc-surface-50);--oidc-card:var(--oidc-surface-0);--oidc-card-soft:var(--oidc-surface-50);--oidc-border:var(--oidc-surface-200);--oidc-text:var(--oidc-surface-900);--oidc-muted:var(--oidc-surface-500);--oidc-soft:color-mix(in srgb,var(--oidc-primary-500) 12%,var(--oidc-surface-0));--oidc-shadow:0 24px 60px rgba(15,23,42,.14)}
    html.dark,html.app-dark{color-scheme:dark;--oidc-bg:var(--oidc-surface-950);--oidc-card:var(--oidc-surface-900);--oidc-card-soft:var(--oidc-surface-800);--oidc-border:var(--oidc-surface-700);--oidc-text:var(--oidc-surface-50);--oidc-muted:var(--oidc-surface-300);--oidc-soft:color-mix(in srgb,var(--oidc-primary-500) 18%,var(--oidc-surface-900));--oidc-shadow:0 24px 60px rgba(0,0,0,.35)}
    body{font-family:'Inter Variable','Inter',ui-sans-serif,system-ui,-apple-system,sans-serif;font-size:14px;background:var(--oidc-bg);color:var(--oidc-text);display:flex;align-items:center;justify-content:center;min-height:100%;padding:20px}
    .card{width:100%;max-width:420px;padding:24px 24px 20px;border:1px solid var(--oidc-border);border-radius:18px;background:var(--oidc-card);box-shadow:var(--oidc-shadow);text-align:center}
    .icon{width:64px;height:64px;border-radius:50%;background:var(--oidc-soft);color:var(--oidc-primary-600);display:flex;align-items:center;justify-content:center;margin:0 auto 16px}
    h1{font-size:20px;font-weight:700;color:var(--oidc-text);margin-bottom:8px}
    p{font-size:14px;line-height:1.6;color:var(--oidc-muted);margin-bottom:20px}
    .actions{display:grid;gap:10px}
    .btn{display:flex;align-items:center;justify-content:center;gap:6px;width:100%;padding:11px 16px;background:var(--oidc-primary-500);color:#fff;border:none;border-radius:10px;font-size:14px;font-weight:600;font-family:inherit;cursor:pointer;transition:background .15s,box-shadow .15s;text-decoration:none}
    .btn:hover{background:var(--oidc-primary-600);box-shadow:0 10px 20px rgba(15,23,42,.16)}
    .btn-secondary{background:transparent;border:1px solid var(--oidc-border);color:var(--oidc-primary-600)}
    .btn-secondary:hover{background:var(--oidc-soft);box-shadow:none}
    @media(prefers-color-scheme:dark){
      :root:not(.light){color-scheme:dark;--oidc-bg:var(--oidc-surface-950);--oidc-card:var(--oidc-surface-900);--oidc-card-soft:var(--oidc-surface-800);--oidc-border:var(--oidc-surface-700);--oidc-text:var(--oidc-surface-50);--oidc-muted:var(--oidc-surface-300);--oidc-soft:color-mix(in srgb,var(--oidc-primary-500) 18%,var(--oidc-surface-900));--oidc-shadow:0 24px 60px rgba(0,0,0,.35)}
    }
  </style>
  <script>
    (function(){function cookie(n){return document.cookie.split(';').map(function(c){return c.trim();}).filter(function(c){return c.indexOf(n+'=')===0;}).map(function(c){return decodeURIComponent(c.slice(n.length+1));})[0]||'';}function stored(k){try{return localStorage.getItem(k)||'';}catch(_){return '';}}var darkCookie=cookie('eguilde_dark'),savedDark=stored('app-dark-mode');var dark=darkCookie==='1'||savedDark==='true'||(!darkCookie&&!savedDark&&window.matchMedia&&window.matchMedia('(prefers-color-scheme: dark)').matches);document.documentElement.classList.toggle('dark',dark);document.documentElement.classList.toggle('app-dark',dark);document.documentElement.classList.toggle('light',!dark);})();
  </script>
</head>
<body>
  <div class="card">
    <div class="icon">⟲</div>
    <h1>Deconectare finalizata</h1>
    <p>Sesiunea curenta a fost inchisa. Puteti reveni in platforma sau inchide aceasta pagina.</p>
    <div class="actions">
      <a class="btn" href="{{.ReturnTo}}">Inapoi in eGuilde</a>
      <button type="button" class="btn btn-secondary" onclick="window.close()">Inchide pagina</button>
    </div>
  </div>
</body>
</html>`

