package auth

import (
	"bytes"
	"encoding/json"
	"html/template"
	"io"
	"net/http"
	"net/url"
	"strings"

	"github.com/eguilde/egueducation/internal/config"
)

func serveOIDCUIScript(w http.ResponseWriter, r *http.Request, script string) {
	w.Header().Set("Content-Type", "text/javascript; charset=utf-8")
	// Authentication behavior must change atomically with the provider HTML.
	// A cached script can otherwise leave OTP input handling on an older release.
	w.Header().Set("Cache-Control", "no-store")
	w.Header().Set("X-Content-Type-Options", "nosniff")
	if r.Method == http.MethodHead {
		return
	}
	_, _ = io.WriteString(w, script)
}

func (s *Service) OIDCLoginScript(w http.ResponseWriter, r *http.Request) {
	serveOIDCUIScript(w, r, oidcLoginScript)
}

func (s *Service) OIDCLogoutScript(w http.ResponseWriter, r *http.Request) {
	serveOIDCUIScript(w, r, oidcLogoutScript)
}

func wrapRefreshTokenCookie(next http.Handler, cfg *config.Config) http.Handler {
	secure := cfg.TLSEnabled()
	cookiePath := "/api/oidc"

	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		isTokenEndpoint := r.Method == http.MethodPost && strings.HasSuffix(r.URL.Path, "/token")
		if !isTokenEndpoint {
			next.ServeHTTP(w, r)
			return
		}

		_ = r.ParseForm()
		if r.FormValue("grant_type") == "refresh_token" {
			value := r.FormValue("refresh_token")
			if value == "" || value == "cookie" {
				if cookie, err := r.Cookie("egueducation_rt"); err == nil && cookie.Value != "" {
					r.Form.Set("refresh_token", cookie.Value)
					r.PostForm.Set("refresh_token", cookie.Value)
				}
			}
		}
		recorder := &tokenResponseRecorder{
			header: make(http.Header),
			buf:    &bytes.Buffer{},
			status: http.StatusOK,
		}
		next.ServeHTTP(recorder, r)

		for key, values := range recorder.header {
			for _, value := range values {
				w.Header().Add(key, value)
			}
		}

		body := recorder.buf.Bytes()
		if isTokenEndpoint && strings.Contains(recorder.header.Get("Content-Type"), "application/json") && bytes.Contains(body, []byte(`"refresh_token"`)) {
			var payload map[string]any
			if err := json.Unmarshal(body, &payload); err == nil {
				if refreshToken, ok := payload["refresh_token"].(string); ok && refreshToken != "" {
					http.SetCookie(w, &http.Cookie{
						Name:     "egueducation_rt",
						Value:    refreshToken,
						Path:     cookiePath,
						HttpOnly: true,
						Secure:   secure,
						SameSite: http.SameSiteLaxMode,
						MaxAge:   86400,
					})
					delete(payload, "refresh_token")
					if rewritten, err := json.Marshal(payload); err == nil {
						body = rewritten
					}
				}
			}
		}

		w.WriteHeader(recorder.status)
		_, _ = w.Write(body)
	})
}

type tokenResponseRecorder struct {
	header http.Header
	buf    *bytes.Buffer
	status int
}

func (r *tokenResponseRecorder) Header() http.Header { return r.header }
func (r *tokenResponseRecorder) WriteHeader(statusCode int) {
	r.status = statusCode
}
func (r *tokenResponseRecorder) Write(data []byte) (int, error) {
	return r.buf.Write(data)
}
func (r *tokenResponseRecorder) ReadFrom(src io.Reader) (int64, error) {
	return io.Copy(r.buf, src)
}

func wrapRegisterPage(next http.Handler, cfg *config.Config) http.Handler {
	tmpl := template.Must(template.New("oidc_register").Parse(oidcRegisterHTML))
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/register" && r.URL.Path != "/new-account" {
			next.ServeHTTP(w, r)
			return
		}
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		w.WriteHeader(http.StatusOK)
		_ = tmpl.Execute(w, map[string]string{
			"CustomerName":   cfg.CustomerName,
			"FrontendOrigin": strings.TrimRight(cfg.FrontendOrigin, "/"),
		})
	})
}

// allowedLogoutReturnTo remains a narrow same-origin helper for callers that
// render an application-owned logout result. It is deliberately not wired to
// the OIDC provider: RP-initiated logout uses exact client metadata matching.
func allowedLogoutReturnTo(candidate, frontendOrigin string) bool {
	candidateURL, err := url.Parse(strings.TrimSpace(candidate))
	if err != nil || !candidateURL.IsAbs() || candidateURL.User != nil {
		return false
	}
	frontendURL, err := url.Parse(strings.TrimSpace(frontendOrigin))
	if err != nil || !frontendURL.IsAbs() {
		return false
	}
	return strings.EqualFold(candidateURL.Scheme, frontendURL.Scheme) && strings.EqualFold(candidateURL.Host, frontendURL.Host)
}

const oidcRegisterHTML = `<!DOCTYPE html>
<html lang="ro">
<head>
  <meta charset="UTF-8">
  <meta name="viewport" content="width=device-width, initial-scale=1.0">
  <title>{{.CustomerName}} - Cont nou</title>
  <style>
    :root{color-scheme:light;--bg:#fff7f8;--card:#fff;--card-2:#fff8fa;--soft:#fff1f5;--border:#e2e8f0;--text:#0f172a;--muted:#64748b;--primary:#e11d48;--shadow:0 28px 72px rgba(15,23,42,.16)}
    *{box-sizing:border-box}body{margin:0;min-height:100vh;display:flex;align-items:center;justify-content:center;padding:24px;background:radial-gradient(circle at top left,rgba(225,29,72,.16),transparent 28rem),linear-gradient(135deg,var(--bg),#fff 48%,#ffe4ec 100%);font-family:Inter,system-ui,sans-serif;color:var(--text)}
    .shell{display:grid;gap:20px;grid-template-columns:1.05fr .95fr;width:min(1040px,100%)}@media(max-width:860px){.shell{grid-template-columns:1fr}}
    .hero,.panel{border:1px solid var(--border);border-radius:30px;background:linear-gradient(180deg,var(--card),var(--card-2));box-shadow:var(--shadow)}.hero{padding:34px}.panel{padding:30px}
    .eyebrow{display:inline-flex;padding:9px 14px;border-radius:999px;background:var(--soft);color:var(--primary);font-size:.76rem;font-weight:800;letter-spacing:.14em;text-transform:uppercase}
    h1{margin:16px 0 14px;font-size:2.2rem;letter-spacing:-.04em}h2{margin:0 0 12px;font-size:1.5rem;letter-spacing:-.03em}
    p{color:var(--muted);line-height:1.8}.actions{display:grid;gap:14px;margin-top:18px}.btn{display:inline-flex;align-items:center;justify-content:center;padding:14px 16px;border-radius:16px;text-decoration:none;font-weight:800}
    .primary{background:linear-gradient(180deg,var(--primary),#be123c);color:#fff}.secondary{border:1px solid var(--border);color:var(--text);background:#fff}
  </style>
</head>
<body>
  <main class="shell">
    <section class="hero">
      <span class="eyebrow">Înregistrare</span>
      <h1>{{.CustomerName}}</h1>
      <p>Crearea contului rămâne separată de providerul OIDC, dar experiența trebuie să rămână coerentă cu autentificarea principală și cu platforma de referință.</p>
    </section>
    <section class="panel">
      <h2>Continuă în aplicație</h2>
      <p>Pentru a crea un cont nou, folosește parcursul de înregistrare din frontend. Providerul OIDC va prelua autentificarea după finalizarea onboarding-ului.</p>
      <div class="actions">
        <a class="btn primary" href="{{.FrontendOrigin}}/auth/register">Deschide înregistrarea</a>
        <a class="btn secondary" href="{{.FrontendOrigin}}/">Înapoi la autentificare</a>
      </div>
    </section>
  </main>
</body>
</html>`

const oidcLogoutScript = `(function(){
  var closeButton=document.getElementById('closeWindow');
  if(closeButton){closeButton.addEventListener('click',function(){window.close();});}
})();`

const oidcLoginScript = `(function(){
  function b64u(value){
    var normalized=value.replace(/-/g,'+').replace(/_/g,'/');
    var decoded=atob(normalized);
    return new Uint8Array([].map.call(decoded,function(character){return character.charCodeAt(0);}));
  }
  function u8b64(value){
    return btoa(String.fromCharCode.apply(null,value)).replace(/\+/g,'-').replace(/\//g,'_').replace(/=+$/g,'');
  }

  var biometricButton=document.getElementById('biometricBtn');
  if(biometricButton){
    if(!window.PublicKeyCredential){
      biometricButton.disabled=true;
    }else{
      var passkeyBanner=document.getElementById('passkey-banner');
      if(passkeyBanner){passkeyBanner.style.display='flex';}
      biometricButton.addEventListener('click',function(){
        fetch('/api/passkeys/login-options',{method:'POST',headers:{'Content-Type':'application/json'},body:'{}'})
          .then(function(response){if(!response.ok)throw new Error('options failed');return response.json();})
          .then(function(payload){
            var options=payload.options||payload;
            var challenge=options.challenge;
            options.challenge=b64u(options.challenge);
            if(options.allowCredentials){options.allowCredentials=options.allowCredentials.map(function(item){return Object.assign({},item,{id:b64u(item.id)});});}
            return navigator.credentials.get({publicKey:options}).then(function(credential){return {credential:credential,challenge:challenge};});
          })
          .then(function(result){
            var credential=result.credential;
            var assertion={clientDataJSON:u8b64(new Uint8Array(credential.response.clientDataJSON)),authenticatorData:u8b64(new Uint8Array(credential.response.authenticatorData)),signature:u8b64(new Uint8Array(credential.response.signature)),type:credential.type};
            return fetch('/api/passkeys/login-finish',{method:'POST',headers:{'Content-Type':'application/json'},credentials:'include',body:JSON.stringify({challenge:result.challenge,credential_id:credential.id,response:assertion})});
          })
          .then(function(response){if(!response.ok)throw new Error('finish failed');return response.json();})
          .then(function(data){
            var form=document.createElement('form');
            form.method='POST';
            form.action=document.body.dataset.oidcAction||'';
            var method=document.createElement('input');method.type='hidden';method.name='method';method.value='passkey_done';
            var nonce=document.createElement('input');nonce.type='hidden';nonce.name='nonce';nonce.value=data.nonce||'';
            form.appendChild(method);form.appendChild(nonce);document.body.appendChild(form);form.submit();
          })
          .catch(function(error){if(error&&error.name!=='NotAllowedError'){alert(error.message||'Autentificarea cu passkey a eșuat');}});
      });
    }
  }

  var otpBoxes=document.querySelectorAll('.otp-box');
  var otpCode=document.getElementById('code');
  var verifyButton=document.getElementById('verifyBtn');
  var otpForm=document.getElementById('otpForm');
  if(otpBoxes.length===6&&otpCode&&verifyButton&&otpForm){
    var syncOTP=function(){
      var value=Array.prototype.map.call(otpBoxes,function(box){return box.value;}).join('');
      otpCode.value=value;
      verifyButton.disabled=value.length<6;
      Array.prototype.forEach.call(otpBoxes,function(box){box.classList.toggle('filled',box.value!=='');});
      if(value.length===6){verifyButton.focus();}
    };
    Array.prototype.forEach.call(otpBoxes,function(box,index){
      box.addEventListener('paste',function(event){
        event.preventDefault();
        var value=(event.clipboardData||window.clipboardData).getData('text').replace(/\D/g,'').slice(0,6);
        if(value){value.split('').forEach(function(digit,digitIndex){if(otpBoxes[digitIndex])otpBoxes[digitIndex].value=digit;});(otpBoxes[Math.min(value.length,5)]||otpBoxes[5]).focus();syncOTP();}
      });
      box.addEventListener('input',function(event){
        var value=event.target.value.replace(/\D/g,'');
        event.target.value=value.slice(-1);
        if(value&&index<5){otpBoxes[index+1].focus();}
        syncOTP();
      });
      box.addEventListener('keydown',function(event){
        if(event.key==='Backspace'&&!box.value&&index>0){otpBoxes[index-1].value='';otpBoxes[index-1].focus();syncOTP();}
        if(event.key==='ArrowLeft'&&index>0){otpBoxes[index-1].focus();}
        if(event.key==='ArrowRight'&&index<5){otpBoxes[index+1].focus();}
        if(event.key==='Enter'&&otpCode.value.length===6){otpForm.requestSubmit();}
      });
    });
    otpForm.addEventListener('submit',syncOTP);
  }

  var selectAll=document.getElementById('selectAll');
  if(selectAll){selectAll.addEventListener('change',function(){document.querySelectorAll('#consentForm input[name=granted_scope]').forEach(function(input){input.checked=selectAll.checked;});});}
  document.documentElement.dataset.oidcUiReady='true';
})();`
