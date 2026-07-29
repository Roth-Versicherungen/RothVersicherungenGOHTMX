// Command export pre-renders the whole site into public/ as plain
// HTML files plus the static assets, ready for any static host. The
// Vercel deploy runs this as its build command (see vercel.json).
package main

import (
	"fmt"
	"io/fs"
	"log"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"

	"github.com/maxroth/eumel/internal/config"
	"github.com/maxroth/eumel/internal/i18n"
	"github.com/maxroth/eumel/internal/server"
	"github.com/maxroth/eumel/internal/view"
	"github.com/maxroth/eumel/web"
)

const outDir = "public"

func main() {
	if err := run(); err != nil {
		log.Fatal(err)
	}
}

func run() error {
	cfg := config.Load()
	// Render from the embedded filesystem so the export never depends
	// on the working directory, and skip the unused SQLite database.
	cfg.Env = "prod"
	cfg.Dev = false

	tr, err := i18n.Load(cfg.DefaultLang)
	if err != nil {
		return err
	}
	v, err := view.New(cfg.Dev, cfg.BaseURL, tr)
	if err != nil {
		return err
	}
	handler := server.New(cfg, v, tr)

	if err := os.RemoveAll(outDir); err != nil {
		return err
	}

	for route := range server.Pages {
		out := filepath.Join(outDir, route, "index.html")
		if err := render(handler, route, http.StatusOK, out); err != nil {
			return err
		}
	}
	// 404 page, picked up automatically by static hosts like Vercel.
	if err := render(handler, "/this-page-does-not-exist", http.StatusNotFound, filepath.Join(outDir, "404.html")); err != nil {
		return err
	}
	// Crawler files.
	for _, name := range []string{"robots.txt", "sitemap.xml"} {
		if err := render(handler, "/"+name, http.StatusOK, filepath.Join(outDir, name)); err != nil {
			return err
		}
	}

	static, err := fs.Sub(web.Static, "static")
	if err != nil {
		return err
	}
	if err := os.CopyFS(filepath.Join(outDir, "static"), static); err != nil {
		return err
	}

	fmt.Printf("exported %d pages + 404 + static assets to %s/\n", len(server.Pages), outDir)
	return nil
}

// render requests route from the handler and writes the response body
// to out, failing if the status is not the expected one.
func render(h http.Handler, route string, wantStatus int, out string) error {
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, route, nil))
	if rec.Code != wantStatus {
		return fmt.Errorf("render %s: status %d, want %d", route, rec.Code, wantStatus)
	}
	if err := os.MkdirAll(filepath.Dir(out), 0o755); err != nil {
		return err
	}
	return os.WriteFile(out, rec.Body.Bytes(), 0o644)
}
