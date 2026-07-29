// Package config loads all runtime configuration from environment
// variables so the binary can be configured without code changes.
package config

import "os"

type Config struct {
	// Addr is the listen address, e.g. ":8080".
	Addr string
	// Env is "dev" or "prod". In dev mode templates and static files
	// are read from disk on every request (live reload); in prod they
	// are served from the embedded filesystem.
	Env string
	// BaseURL is the site's public origin (no trailing slash), used
	// for canonical links, Open Graph URLs and the sitemap.
	BaseURL string
	// DefaultLang is the fallback language code, e.g. "de".
	DefaultLang string
	// Dev is true when Env == "dev".
	Dev bool
}

func Load() *Config {
	cfg := &Config{
		Addr:        getenv("ADDR", ":8080"),
		Env:         getenv("ENV", "dev"),
		BaseURL:     getenv("BASE_URL", "https://www.roth-makler.de"),
		DefaultLang: getenv("DEFAULT_LANG", "de"),
	}
	cfg.Dev = cfg.Env == "dev"
	return cfg
}

func getenv(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}
