package server

import (
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/maxroth/eumel/internal/config"
	"github.com/maxroth/eumel/internal/i18n"
	"github.com/maxroth/eumel/internal/view"
)

func newTestHandler(t *testing.T, tr *i18n.Translator) *httptest.Server {
	t.Helper()
	cfg := &config.Config{Env: "prod", BaseURL: "https://example.test", DefaultLang: "de"}
	v, err := view.New(false, cfg.BaseURL, tr)
	if err != nil {
		t.Fatalf("view.New: %v", err)
	}
	return httptest.NewServer(New(cfg, v, tr))
}

// TestPagesRender requests every registered page and fails if any page
// errors or renders an untranslated key. This is the safety net for
// locale edits: a typo in a key or a missing entry in locales/de/
// fails here instead of shipping as raw key text.
func TestPagesRender(t *testing.T) {
	tr, err := i18n.Load("de")
	if err != nil {
		t.Fatalf("i18n.Load: %v", err)
	}
	missing := map[string]bool{}
	tr.OnMissing = func(lang, key string) { missing[key] = true }

	srv := newTestHandler(t, tr)
	defer srv.Close()

	for route := range Pages {
		res, err := srv.Client().Get(srv.URL + route)
		if err != nil {
			t.Fatalf("GET %s: %v", route, err)
		}
		body := readBody(t, res)
		if res.StatusCode != 200 {
			t.Errorf("GET %s = %d, want 200", route, res.StatusCode)
		}
		if !strings.Contains(body, "</html>") {
			t.Errorf("GET %s: response is not a complete page", route)
		}
	}

	// The 404 page renders through the same template machinery.
	res, err := srv.Client().Get(srv.URL + "/does-not-exist")
	if err != nil {
		t.Fatal(err)
	}
	readBody(t, res)
	if res.StatusCode != 404 {
		t.Errorf("unknown route = %d, want 404", res.StatusCode)
	}

	for key := range missing {
		t.Errorf("missing translation: %q", key)
	}
}

func TestCrawlerFiles(t *testing.T) {
	tr, err := i18n.Load("de")
	if err != nil {
		t.Fatalf("i18n.Load: %v", err)
	}
	srv := newTestHandler(t, tr)
	defer srv.Close()

	res, err := srv.Client().Get(srv.URL + "/sitemap.xml")
	if err != nil {
		t.Fatal(err)
	}
	body := readBody(t, res)
	if !strings.Contains(body, "<loc>https://example.test/team</loc>") {
		t.Error("sitemap.xml is missing the /team URL")
	}
	if got := strings.Count(body, "<url>"); got != len(Pages) {
		t.Errorf("sitemap.xml has %d URLs, want %d", got, len(Pages))
	}

	res, err = srv.Client().Get(srv.URL + "/robots.txt")
	if err != nil {
		t.Fatal(err)
	}
	if body := readBody(t, res); !strings.Contains(body, "Sitemap: https://example.test/sitemap.xml") {
		t.Error("robots.txt is missing the sitemap reference")
	}
}

func readBody(t *testing.T, res *http.Response) string {
	t.Helper()
	defer res.Body.Close()
	b, err := io.ReadAll(res.Body)
	if err != nil {
		t.Fatalf("read body: %v", err)
	}
	return string(b)
}
