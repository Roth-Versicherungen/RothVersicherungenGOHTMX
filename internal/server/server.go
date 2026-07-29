// Package server wires routes, middleware and handlers together.
package server

import (
	"fmt"
	"io/fs"
	"net/http"
	"sort"

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

func New(cfg *config.Config, v *view.View, tr *i18n.Translator) http.Handler {
	h := &handlers.Handler{View: v}

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

	// Crawler files, generated from the Pages map.
	mux.HandleFunc("GET /robots.txt", robotsTxt(cfg))
	mux.HandleFunc("GET /sitemap.xml", sitemapXML(cfg))

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

func robotsTxt(cfg *config.Config) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/plain; charset=utf-8")
		fmt.Fprintf(w, "User-agent: *\nAllow: /\n\nSitemap: %s/sitemap.xml\n", cfg.BaseURL)
	}
}

func sitemapXML(cfg *config.Config) http.HandlerFunc {
	routes := make([]string, 0, len(Pages))
	for route := range Pages {
		routes = append(routes, route)
	}
	sort.Strings(routes)

	return func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/xml; charset=utf-8")
		fmt.Fprint(w, `<?xml version="1.0" encoding="UTF-8"?>`+"\n")
		fmt.Fprint(w, `<urlset xmlns="http://www.sitemaps.org/schemas/sitemap/0.9">`+"\n")
		for _, route := range routes {
			fmt.Fprintf(w, "  <url><loc>%s%s</loc></url>\n", cfg.BaseURL, route)
		}
		fmt.Fprint(w, "</urlset>\n")
	}
}

func cacheStatic(cfg *config.Config, next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if !cfg.Dev {
			w.Header().Set("Cache-Control", "public, max-age=86400")
		}
		next.ServeHTTP(w, r)
	})
}
