// Package view renders HTML templates.
//
// Layout: web/templates/layouts/base.html is the page shell, every
// file in web/templates/pages/ fills its "content" block, and files in
// web/templates/partials/ are shared snippets that can also be rendered
// standalone for HTMX responses.
//
// In dev mode templates are re-read from disk on every request so
// edits show up on reload without restarting the server.
package view

import (
	"fmt"
	"html/template"
	"io/fs"
	"log/slog"
	"net/http"
	"os"
	"path"
	"strings"
	"time"

	"github.com/maxroth/eumel/internal/i18n"
	"github.com/maxroth/eumel/web"
)

type View struct {
	dev   bool
	tr    *i18n.Translator
	fsys  fs.FS
	pages map[string]*template.Template // page name -> layout+partials+page
}

// Data is what every template sees. Page-specific values go in .Data.
type Data struct {
	Lang      string
	Languages []string
	Path      string
	Data      any
}

func New(dev bool, tr *i18n.Translator) (*View, error) {
	v := &View{dev: dev, tr: tr}
	if dev {
		v.fsys = os.DirFS("web/templates")
	} else {
		sub, err := fs.Sub(web.Templates, "templates")
		if err != nil {
			return nil, err
		}
		v.fsys = sub
	}
	if !dev {
		// Parse once at startup so template errors fail the boot, not
		// the first request.
		pages, err := parseAll(v.fsys)
		if err != nil {
			return nil, err
		}
		v.pages = pages
	}
	return v, nil
}

// baseFuncs registers every custom template function. The "t" and
// "tlist" functions are placeholders here; they are rebound to the
// request language on render.
func baseFuncs() template.FuncMap {
	return template.FuncMap{
		"t":     func(key string, args ...any) string { return key },
		"tlist": func(key string) []string { return nil },
		// dict builds a map from key/value pairs so partials can take
		// named parameters: {{template "x" dict "Title" "..." "Big" true}}
		"dict": func(pairs ...any) (map[string]any, error) {
			if len(pairs)%2 != 0 {
				return nil, fmt.Errorf("dict: odd number of arguments")
			}
			m := make(map[string]any, len(pairs)/2)
			for i := 0; i < len(pairs); i += 2 {
				key, ok := pairs[i].(string)
				if !ok {
					return nil, fmt.Errorf("dict: key %v is not a string", pairs[i])
				}
				m[key] = pairs[i+1]
			}
			return m, nil
		},
		"hasPrefix": strings.HasPrefix,
		"split":     strings.Split,
		"add":       func(a, b int) int { return a + b },
		"mod":       func(a, b int) int { return a % b },
		"year":      func() int { return time.Now().Year() },
	}
}

func parseAll(fsys fs.FS) (map[string]*template.Template, error) {
	pageFiles, err := fs.Glob(fsys, "pages/*.html")
	if err != nil {
		return nil, err
	}
	pages := make(map[string]*template.Template, len(pageFiles))
	for _, file := range pageFiles {
		name := path.Base(file)
		tmpl, err := template.New(name).Funcs(baseFuncs()).
			ParseFS(fsys, "layouts/*.html", "partials/*.html", file)
		if err != nil {
			return nil, fmt.Errorf("view: parse %s: %w", file, err)
		}
		pages[name] = tmpl
	}
	return pages, nil
}

func (v *View) page(name string) (*template.Template, error) {
	if v.dev {
		pages, err := parseAll(v.fsys)
		if err != nil {
			return nil, err
		}
		if tmpl, ok := pages[name]; ok {
			return tmpl, nil
		}
		return nil, fmt.Errorf("view: unknown page %q", name)
	}
	tmpl, ok := v.pages[name]
	if !ok {
		return nil, fmt.Errorf("view: unknown page %q", name)
	}
	return tmpl, nil
}

// Render writes a full page (layout + page template) with the given
// HTTP status. The page name is the filename in web/templates/pages/.
func (v *View) Render(w http.ResponseWriter, r *http.Request, status int, page string, data any) {
	tmpl, err := v.page(page)
	if err != nil {
		v.fail(w, err)
		return
	}
	v.execute(w, r, tmpl, "base", status, data)
}

// RenderPartial writes a single named template (from partials/) without
// the layout — this is what HTMX endpoints respond with.
func (v *View) RenderPartial(w http.ResponseWriter, r *http.Request, partial string, data any) {
	// Every page set contains all partials; any of them can serve here.
	// In dev this also picks up fresh edits.
	tmpl, err := v.anySet()
	if err != nil {
		v.fail(w, err)
		return
	}
	v.execute(w, r, tmpl, partial, http.StatusOK, data)
}

func (v *View) anySet() (*template.Template, error) {
	if v.dev {
		tmpl, err := template.New("partials").Funcs(baseFuncs()).
			ParseFS(v.fsys, "partials/*.html")
		if err != nil {
			return nil, err
		}
		return tmpl, nil
	}
	for _, tmpl := range v.pages {
		return tmpl, nil
	}
	return nil, fmt.Errorf("view: no templates parsed")
}

func (v *View) execute(w http.ResponseWriter, r *http.Request, tmpl *template.Template, name string, status int, data any) {
	lang := i18n.Lang(r.Context())
	if lang == "" {
		lang = v.tr.DefaultLang()
	}

	// Clone so the per-request "t" binding never leaks between requests.
	tmpl, err := tmpl.Clone()
	if err != nil {
		v.fail(w, err)
		return
	}
	tmpl.Funcs(template.FuncMap{
		"t":     func(key string, args ...any) string { return v.tr.T(lang, key, args...) },
		"tlist": func(key string) []string { return v.tr.List(lang, key) },
	})

	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.WriteHeader(status)
	if err := tmpl.ExecuteTemplate(w, name, Data{
		Lang:      lang,
		Languages: v.tr.Languages(),
		Path:      r.URL.Path,
		Data:      data,
	}); err != nil {
		slog.Error("render", "template", name, "err", err)
	}
}

func (v *View) fail(w http.ResponseWriter, err error) {
	slog.Error("view", "err", err)
	http.Error(w, "internal server error", http.StatusInternalServerError)
}
