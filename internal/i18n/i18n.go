// Package i18n loads translation strings from the locales folder and
// resolves the request language (query param > cookie > Accept-Language).
//
// String files are flat JSON maps with dot-separated keys:
//
//	{ "nav.home": "Home", "todos.count": "%d open todos" }
//
// Values are fmt.Sprintf format strings when arguments are passed.
package i18n

import (
	"context"
	"encoding/json"
	"fmt"
	"io/fs"
	"net/http"
	"sort"
	"strings"

	"github.com/maxroth/eumel/locales"
)

const CookieName = "lang"

type Translator struct {
	defaultLang string
	langs       map[string]map[string]string
}

// Load parses every *.json file in the locales folder. The filename
// (without extension) is the language code.
func Load(defaultLang string) (*Translator, error) {
	t := &Translator{defaultLang: defaultLang, langs: map[string]map[string]string{}}

	entries, err := fs.ReadDir(locales.Files, ".")
	if err != nil {
		return nil, fmt.Errorf("i18n: read locales dir: %w", err)
	}
	for _, e := range entries {
		if e.IsDir() || !strings.HasSuffix(e.Name(), ".json") {
			continue
		}
		lang := strings.TrimSuffix(e.Name(), ".json")
		raw, err := fs.ReadFile(locales.Files, e.Name())
		if err != nil {
			return nil, fmt.Errorf("i18n: read %s: %w", e.Name(), err)
		}
		strs := map[string]string{}
		if err := json.Unmarshal(raw, &strs); err != nil {
			return nil, fmt.Errorf("i18n: parse %s: %w", e.Name(), err)
		}
		t.langs[lang] = strs
	}
	if _, ok := t.langs[defaultLang]; !ok {
		return nil, fmt.Errorf("i18n: default language %q has no locales/%s.json", defaultLang, defaultLang)
	}
	return t, nil
}

// T returns the string for key in lang, falling back to the default
// language and finally to the key itself (so missing strings are
// visible instead of silently empty).
func (t *Translator) T(lang, key string, args ...any) string {
	s, ok := t.langs[lang][key]
	if !ok {
		s, ok = t.langs[t.defaultLang][key]
	}
	if !ok {
		return key
	}
	if len(args) > 0 {
		return fmt.Sprintf(s, args...)
	}
	return s
}

// List returns the strings stored under key.1, key.2, ... in order,
// stopping at the first missing index. Lets templates iterate over
// translated lists without knowing their length.
func (t *Translator) List(lang, key string) []string {
	var out []string
	for i := 1; ; i++ {
		k := fmt.Sprintf("%s.%d", key, i)
		s, ok := t.langs[lang][k]
		if !ok {
			s, ok = t.langs[t.defaultLang][k]
		}
		if !ok {
			return out
		}
		out = append(out, s)
	}
}

// Languages returns all loaded language codes, sorted.
func (t *Translator) Languages() []string {
	out := make([]string, 0, len(t.langs))
	for l := range t.langs {
		out = append(out, l)
	}
	sort.Strings(out)
	return out
}

func (t *Translator) Supported(lang string) bool {
	_, ok := t.langs[lang]
	return ok
}

func (t *Translator) DefaultLang() string { return t.defaultLang }

// Resolve determines the language for a request:
// ?lang= query param, then the lang cookie, then Accept-Language.
func (t *Translator) Resolve(r *http.Request) string {
	if lang := r.URL.Query().Get("lang"); t.Supported(lang) {
		return lang
	}
	if c, err := r.Cookie(CookieName); err == nil && t.Supported(c.Value) {
		return c.Value
	}
	for _, part := range strings.Split(r.Header.Get("Accept-Language"), ",") {
		code, _, _ := strings.Cut(strings.TrimSpace(part), ";")
		code, _, _ = strings.Cut(code, "-") // "de-AT" -> "de"
		if t.Supported(code) {
			return code
		}
	}
	return t.defaultLang
}

type ctxKey struct{}

// WithLang stores the resolved language on the request context.
func WithLang(ctx context.Context, lang string) context.Context {
	return context.WithValue(ctx, ctxKey{}, lang)
}

// Lang returns the language stored on the context, or "".
func Lang(ctx context.Context) string {
	lang, _ := ctx.Value(ctxKey{}).(string)
	return lang
}
