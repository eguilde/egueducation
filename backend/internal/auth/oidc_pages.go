package auth

import (
	"bytes"
	"encoding/json"
	"html/template"
	"io"
	"net/http"
	"strings"

	"github.com/eguilde/egueducation/internal/config"
)

func wrapRefreshTokenCookie(next http.Handler, cfg *config.Config) http.Handler {
	secure := cfg.TLSEnabled()
	cookiePath := "/api/oidc"

	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		isTokenEndpoint := r.Method == http.MethodPost && strings.HasSuffix(r.URL.Path, "/token")
		isRevocationEndpoint := r.Method == http.MethodPost && strings.HasSuffix(r.URL.Path, "/revoke")
		if !isTokenEndpoint && !isRevocationEndpoint {
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
		if isRevocationEndpoint {
			value := r.FormValue("token")
			if value == "" || value == "cookie" {
				if cookie, err := r.Cookie("egueducation_rt"); err == nil && cookie.Value != "" {
					r.Form.Set("token", cookie.Value)
					r.PostForm.Set("token", cookie.Value)
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

func wrapLogoutPage(next http.Handler, cfg *config.Config) http.Handler {
	tmpl := template.Must(template.New("oidc_logout").Parse(oidcLogoutHTML))
	secure := cfg.TLSEnabled()

	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/logout" {
			next.ServeHTTP(w, r)
			return
		}

		returnTo := strings.TrimSpace(r.URL.Query().Get("returnTo"))
		if returnTo == "" {
			returnTo = cfg.FrontendOrigin
		}

		http.SetCookie(w, &http.Cookie{
			Name:     "egueducation_rt",
			Value:    "",
			Path:     "/api/oidc",
			HttpOnly: true,
			Secure:   secure,
			SameSite: http.SameSiteLaxMode,
			MaxAge:   -1,
		})

		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		_ = tmpl.Execute(w, map[string]string{
			"CustomerName": cfg.CustomerName,
			"ReturnTo":     returnTo,
		})
	})
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

const oidcLogoutHTML = `<!DOCTYPE html>
<html lang="ro">
<head>
  <meta charset="UTF-8">
  <meta name="viewport" content="width=device-width, initial-scale=1.0">
  <title>{{.CustomerName}} - Logout</title>
  <style>
    :root{color-scheme:light;--bg:#fff7f8;--card:#fff;--soft:#fff1f5;--border:#e2e8f0;--text:#0f172a;--muted:#64748b;--primary:#e11d48;--shadow:0 28px 72px rgba(15,23,42,.16)}
    *{box-sizing:border-box}body{margin:0;min-height:100vh;display:flex;align-items:center;justify-content:center;padding:24px;background:radial-gradient(circle at top left,rgba(225,29,72,.16),transparent 28rem),linear-gradient(135deg,var(--bg),#fff 48%,#ffe4ec 100%);font-family:Inter,system-ui,sans-serif;color:var(--text)}
    .card{width:min(440px,100%);border:1px solid var(--border);border-radius:28px;background:linear-gradient(180deg,var(--card),#fff8fa);box-shadow:var(--shadow);padding:30px;text-align:center}
    .eyebrow{display:inline-flex;padding:8px 12px;border-radius:999px;background:var(--soft);color:var(--primary);font-size:11px;font-weight:800;letter-spacing:.12em;text-transform:uppercase}
    h1{margin:14px 0 12px;font-size:1.8rem;letter-spacing:-.03em}.msg{color:var(--muted);line-height:1.7;margin-bottom:18px}.actions{display:grid;gap:12px}
    .btn{display:inline-flex;align-items:center;justify-content:center;padding:14px 16px;border-radius:16px;text-decoration:none;font-weight:800}.primary{background:linear-gradient(180deg,var(--primary),#be123c);color:#fff}.secondary{border:1px solid var(--border);color:var(--text);background:#fff}
  </style>
</head>
<body>
  <main class="card">
    <span class="eyebrow">OIDC Provider</span>
    <h1>Deconectare finalizată</h1>
    <p class="msg">Sesiunea OIDC a fost închisă și refresh token-ul din cookie a fost eliminat.</p>
    <div class="actions">
      <a class="btn primary" href="{{.ReturnTo}}">Înapoi în aplicație</a>
      <button class="btn secondary" type="button" onclick="window.close()">Închide</button>
    </div>
  </main>
</body>
</html>`
