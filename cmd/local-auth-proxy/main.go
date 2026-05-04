package main

import (
	"flag"
	"fmt"
	"html/template"
	"log"
	"net/http"
	"net/http/httputil"
	"net/url"
	"strings"
	"time"
)

const roleCookieName = "checkycheck_local_role"

type proxyConfig struct {
	listen         string
	upstream       *url.URL
	defaultRole    string
	userHeader     string
	emailHeader    string
	groupsHeader   string
	viewerGroup    string
	operatorGroup  string
	adminGroup     string
	usernamePrefix string
	emailDomain    string
}

type roleProfile struct {
	Name    string
	Label   string
	User    string
	Email   string
	Groups  []string
	Auth    bool
	Comment string
}

type chooserData struct {
	CurrentRole string
	Roles       []roleProfile
}

var chooserTemplate = template.Must(template.New("chooser").Funcs(template.FuncMap{
	"join": strings.Join,
}).Parse(`<!doctype html>
<html lang="en">
<head>
  <meta charset="utf-8">
  <meta name="viewport" content="width=device-width, initial-scale=1">
  <title>CheckyCheck Local Auth Proxy</title>
  <style>
    body { font-family: ui-sans-serif, system-ui, sans-serif; margin: 0; background: #f5f7fb; color: #182235; }
    main { max-width: 820px; margin: 0 auto; padding: 40px 20px; }
    .card { background: #fff; border: 1px solid #d8dfe9; border-radius: 18px; padding: 24px; box-shadow: 0 12px 30px rgba(15, 23, 42, 0.06); }
    h1 { margin: 0 0 12px; font-size: 2rem; }
    p { line-height: 1.6; color: #5f6c82; }
    .role-grid { display: grid; grid-template-columns: repeat(auto-fit, minmax(220px, 1fr)); gap: 16px; margin-top: 24px; }
    .role-card { border: 1px solid #d8dfe9; border-radius: 14px; padding: 16px; background: #fbfcfe; }
    .role-card.is-active { border-color: #6c83ff; background: #eef3ff; }
    .role-title { display: flex; align-items: center; justify-content: space-between; gap: 8px; margin-bottom: 10px; }
    .role-title strong { font-size: 1rem; }
    .badge { display: inline-flex; align-items: center; justify-content: center; padding: 0.25rem 0.55rem; border-radius: 999px; font-size: 0.74rem; font-weight: 700; border: 1px solid #d8dfe9; background: #fff; color: #182235; }
    .badge.is-active { border-color: #6c83ff; color: #3557d4; }
    code { display: block; margin-top: 8px; padding: 8px 10px; border-radius: 10px; background: #f5f7fb; color: #182235; font-size: 0.82rem; word-break: break-all; }
    .actions { display: flex; flex-wrap: wrap; gap: 10px; margin-top: 14px; }
    a.button { display: inline-flex; align-items: center; justify-content: center; min-height: 2.5rem; padding: 0.5rem 0.9rem; border-radius: 999px; border: 1px solid #d8dfe9; color: #182235; background: #fff; text-decoration: none; font-weight: 700; }
    a.button.primary { background: #182235; color: #fff; border-color: #182235; }
  </style>
</head>
<body>
  <main>
    <div class="card">
      <h1>Local Auth Proxy</h1>
      <p>Switch the simulated role below, then open CheckyCheck through this proxy. The proxy injects the same auth headers your production reverse proxy would normally send.</p>
      <p><strong>Current role:</strong> {{.CurrentRole}}</p>
      <div class="role-grid">
        {{range .Roles}}
        <section class="role-card{{if eq $.CurrentRole .Name}} is-active{{end}}">
          <div class="role-title">
            <strong>{{.Label}}</strong>
            {{if eq $.CurrentRole .Name}}<span class="badge is-active">active</span>{{else}}<span class="badge">inactive</span>{{end}}
          </div>
          <p>{{.Comment}}</p>
          <code>{{if .Auth}}{{.User}} / {{.Email}} / {{join .Groups ", "}}{{else}}No auth headers{{end}}</code>
          <div class="actions">
            <a class="button" href="/__auth__/set-role?role={{.Name}}">Use {{.Label}}</a>
            <a class="button primary" href="/__auth__/set-role?role={{.Name}}&redirect=/">Use and open app</a>
          </div>
        </section>
        {{end}}
      </div>
    </div>
  </main>
</body>
</html>`))

func main() {
	listen := flag.String("listen", ":8080", "Local listen address")
	upstreamRaw := flag.String("upstream", "http://127.0.0.1:8070", "Upstream CheckyCheck URL")
	defaultRole := flag.String("default-role", "unauthenticated", "Default simulated role: unauthenticated, viewer, operator, admin")
	userHeader := flag.String("user-header", "X-Forwarded-User", "Header name used for the authenticated username")
	emailHeader := flag.String("email-header", "X-Forwarded-Email", "Header name used for the authenticated email")
	groupsHeader := flag.String("groups-header", "X-Forwarded-Groups", "Header name used for the authenticated groups")
	viewerGroup := flag.String("viewer-group", "checkycheck-viewer", "Viewer group to inject")
	operatorGroup := flag.String("operator-group", "checkycheck-operator", "Operator group to inject")
	adminGroup := flag.String("admin-group", "checkycheck-admin", "Admin group to inject")
	usernamePrefix := flag.String("username-prefix", "local", "Prefix for generated usernames")
	emailDomain := flag.String("email-domain", "example.test", "Email domain for generated users")
	flag.Parse()

	upstreamURL, err := url.Parse(*upstreamRaw)
	if err != nil {
		log.Fatalf("invalid upstream URL: %v", err)
	}

	cfg := proxyConfig{
		listen:         *listen,
		upstream:       upstreamURL,
		defaultRole:    normalizeRole(*defaultRole),
		userHeader:     *userHeader,
		emailHeader:    *emailHeader,
		groupsHeader:   *groupsHeader,
		viewerGroup:    *viewerGroup,
		operatorGroup:  *operatorGroup,
		adminGroup:     *adminGroup,
		usernamePrefix: *usernamePrefix,
		emailDomain:    *emailDomain,
	}

	if !isSupportedRole(cfg.defaultRole) {
		log.Fatalf("unsupported default role %q", cfg.defaultRole)
	}

	proxy := httputil.NewSingleHostReverseProxy(cfg.upstream)
	originalDirector := proxy.Director
	proxy.Director = func(req *http.Request) {
		originalDirector(req)
		role := roleFromRequest(req, cfg.defaultRole)
		applyRoleHeaders(req, cfg, role)
	}
	proxy.ErrorHandler = func(w http.ResponseWriter, r *http.Request, err error) {
		log.Printf("proxy error for %s: %v", r.URL.Path, err)
		http.Error(w, "Local auth proxy could not reach CheckyCheck.", http.StatusBadGateway)
	}

	mux := http.NewServeMux()
	mux.HandleFunc("/__auth__", func(w http.ResponseWriter, r *http.Request) {
		currentRole := roleFromRequest(r, cfg.defaultRole)
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		if err := chooserTemplate.Execute(w, chooserData{
			CurrentRole: currentRole,
			Roles:       buildProfiles(cfg),
		}); err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
		}
	})
	mux.HandleFunc("/__auth__/set-role", func(w http.ResponseWriter, r *http.Request) {
		role := normalizeRole(r.URL.Query().Get("role"))
		if !isSupportedRole(role) {
			http.Error(w, fmt.Sprintf("unsupported role %q", role), http.StatusBadRequest)
			return
		}

		http.SetCookie(w, &http.Cookie{
			Name:     roleCookieName,
			Value:    role,
			Path:     "/",
			HttpOnly: true,
			SameSite: http.SameSiteLaxMode,
			Expires:  time.Now().Add(24 * time.Hour),
		})

		redirectTarget := strings.TrimSpace(r.URL.Query().Get("redirect"))
		if redirectTarget == "" {
			redirectTarget = "/__auth__"
		}
		http.Redirect(w, r, redirectTarget, http.StatusSeeOther)
	})
	mux.HandleFunc("/__auth__/logout", func(w http.ResponseWriter, r *http.Request) {
		http.SetCookie(w, &http.Cookie{
			Name:     roleCookieName,
			Value:    "",
			Path:     "/",
			HttpOnly: true,
			SameSite: http.SameSiteLaxMode,
			MaxAge:   -1,
		})
		http.Redirect(w, r, "/__auth__", http.StatusSeeOther)
	})
	mux.Handle("/", proxy)

	log.Printf("Local auth proxy listening on %s and forwarding to %s", cfg.listen, cfg.upstream)
	log.Printf("Open http://127.0.0.1%s/__auth__ to switch simulated roles", cfg.listen)
	if err := http.ListenAndServe(cfg.listen, mux); err != nil {
		log.Fatalf("proxy failed: %v", err)
	}
}

func normalizeRole(value string) string {
	role := strings.ToLower(strings.TrimSpace(value))
	if role == "guest" {
		return "unauthenticated"
	}
	return role
}

func isSupportedRole(role string) bool {
	switch normalizeRole(role) {
	case "unauthenticated", "viewer", "operator", "admin":
		return true
	default:
		return false
	}
}

func roleFromRequest(r *http.Request, fallback string) string {
	if cookie, err := r.Cookie(roleCookieName); err == nil {
		if role := normalizeRole(cookie.Value); isSupportedRole(role) {
			return role
		}
	}
	return fallback
}

func buildProfiles(cfg proxyConfig) []roleProfile {
	return []roleProfile{
		{Name: "unauthenticated", Label: "Unauthenticated", Comment: "No auth headers are sent. Useful for checking the sign-in-required state."},
		{Name: "viewer", Label: "Viewer", User: cfg.usernamePrefix + ".viewer", Email: cfg.usernamePrefix + ".viewer@" + cfg.emailDomain, Groups: []string{cfg.viewerGroup}, Auth: true, Comment: "Authenticated, but read-only. Run and rerun actions should stay hidden."},
		{Name: "operator", Label: "Operator", User: cfg.usernamePrefix + ".operator", Email: cfg.usernamePrefix + ".operator@" + cfg.emailDomain, Groups: []string{cfg.operatorGroup}, Auth: true, Comment: "Authenticated with operate rights. Manual runs and reruns should be available."},
		{Name: "admin", Label: "Admin", User: cfg.usernamePrefix + ".admin", Email: cfg.usernamePrefix + ".admin@" + cfg.emailDomain, Groups: []string{cfg.adminGroup}, Auth: true, Comment: "Authenticated with admin rights. Currently behaves like operator plus admin flag."},
	}
}

func applyRoleHeaders(req *http.Request, cfg proxyConfig, role string) {
	req.Header.Del(cfg.userHeader)
	req.Header.Del(cfg.emailHeader)
	req.Header.Del(cfg.groupsHeader)
	req.Header.Del("X-Auth-Request-User")
	req.Header.Del("X-Auth-Request-Email")
	req.Header.Del("X-Auth-Request-Groups")

	profiles := buildProfiles(cfg)
	for _, profile := range profiles {
		if profile.Name != role || !profile.Auth {
			continue
		}
		req.Header.Set(cfg.userHeader, profile.User)
		req.Header.Set(cfg.emailHeader, profile.Email)
		req.Header.Set(cfg.groupsHeader, strings.Join(profile.Groups, ","))
		req.Header.Set("X-Auth-Request-User", profile.User)
		req.Header.Set("X-Auth-Request-Email", profile.Email)
		req.Header.Set("X-Auth-Request-Groups", strings.Join(profile.Groups, ","))
		return
	}
}
