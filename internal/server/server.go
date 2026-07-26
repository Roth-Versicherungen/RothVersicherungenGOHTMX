// Package server wires routes, middleware and handlers together.
package server

import (
	"database/sql"
	"io/fs"
	"net/http"

	"github.com/maxroth/eumel/internal/config"
	"github.com/maxroth/eumel/internal/handlers"
	"github.com/maxroth/eumel/internal/i18n"
	"github.com/maxroth/eumel/internal/view"
	"github.com/maxroth/eumel/web"
)

// Pages maps every URL path to its page template. New registers a route
// per entry; cmd/export renders each one into the static site.
var Pages = map[string]string{
	"/": "home.html",

	"/roth-versicherungen":                                      "versicherungen.html",
	"/roth-versicherungen/firmenkunden":                         "firmenkunden.html",
	"/roth-versicherungen/firmenkunden/cyber-police":            "cyber.html",
	"/roth-versicherungen/privatkunden":                         "privatkunden.html",
	"/roth-versicherungen/privatkunden/tierkrankenversicherung": "tier.html",
	"/roth-versicherungen/wichtige-hinweise":                    "hinweise.html",
	"/roth-versicherungen/jobs":                                 "jobs.html",
	"/roth-versicherungen/erstinformation":                      "vers-erstinformation.html",
	"/roth-versicherungen/datenschutz":                          "vers-datenschutz.html",
	"/roth-versicherungen/impressum":                            "vers-impressum.html",

	"/roth-finanz":                        "finanz.html",
	"/roth-finanz/altersversorgung":       "altersversorgung.html",
	"/roth-finanz/sterbegeldversicherung": "sterbegeld.html",
	"/roth-finanz/erstinformation":        "finanz-erstinformation.html",
	"/roth-finanz/datenschutz":            "finanz-datenschutz.html",
	"/roth-finanz/impressum":              "finanz-impressum.html",

	"/team":            "team.html",
	"/kontakt-anfahrt": "kontakt.html",
	"/sitemap":         "sitemap.html",
}

func New(cfg *config.Config, database *sql.DB, v *view.View, tr *i18n.Translator) http.Handler {
	h := &handlers.Handler{DB: database, View: v}

	mux := http.NewServeMux()

	// Pages — every route renders a static page template; all content
	// comes from locales/de.json.
	for route, page := range Pages {
		pattern := "GET " + route
		if route == "/" {
			pattern = "GET /{$}"
		}
		mux.HandleFunc(pattern, h.Page(page))
	}

	// Static assets: from disk in dev (live reload), embedded in prod.
	var staticFS http.FileSystem
	if cfg.Dev {
		staticFS = http.Dir("web/static")
	} else {
		sub, _ := fs.Sub(web.Static, "static")
		staticFS = http.FS(sub)
	}
	mux.Handle("GET /static/", http.StripPrefix("/static/", cacheStatic(cfg, http.FileServer(staticFS))))

	// Operational
	mux.HandleFunc("GET /healthz", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("ok"))
	})

	// Everything else is a 404 rendered through the error page.
	mux.HandleFunc("/", h.NotFound)

	var handler http.Handler = mux
	handler = Language(tr)(handler)
	handler = Logger(handler)
	handler = Recover(v)(handler)
	return handler
}

func cacheStatic(cfg *config.Config, next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if !cfg.Dev {
			w.Header().Set("Cache-Control", "public, max-age=86400")
		}
		next.ServeHTTP(w, r)
	})
}
