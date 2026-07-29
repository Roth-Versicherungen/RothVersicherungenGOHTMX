# Roth Versicherungen — Go + HTMX

Server-rendered website for Roth Versicherungen Maklergesellschaft m.b.H. and Roth Finanz Maklergesellschaft m.b.H. in Langen, converted from the original React SPA.

**Stack:** Go stdlib (router + `html/template`) · [HTMX](https://htmx.org) (vendored) · [Tailwind CSS v4](https://tailwindcss.com) (standalone CLI, no Node.js) · SQLite (pure-Go driver, no CGO) · JSON-based i18n

## Quick start

```sh
make dev        # downloads the Tailwind binary on first run, builds CSS, starts the server
```

Open http://localhost:8080. In dev mode, templates and static files are re-read from disk on every request — edit and reload, no restart needed. Run `make css-watch` in a second terminal if you're changing Tailwind classes.

## Project layout

```
cmd/server/            Entrypoint: config, i18n, renderer, graceful shutdown
cmd/export/            Pre-renders the whole site into public/ (static deploys, Vercel)
internal/
  config/              Env-var configuration (ADDR, ENV, BASE_URL, DEFAULT_LANG)
  server/              Pages route map + middleware + robots.txt/sitemap.xml
  handlers/            HTTP handlers — every page renders a static template
  db/                  SQLite setup + migrations (dormant; see package doc to revive)
  i18n/                String loading + per-request language resolution
  view/                Template renderer (dev: live reload, prod: parsed once, embedded)
locales/               All site content and UI strings: de.json (flat JSON, dot keys)
web/
  templates/layouts/   base.html — the page shell (header, footer, meta/SEO tags)
  templates/pages/     One file per page, fills the "content" block
  templates/partials/  nav, footer, page-hero, section-head, cta, link-card, legal helpers
  static/              css/ (Tailwind in+out), js/ (vendored htmx + nav.js), img/ (site images)
```

## Pages

| Route | Template |
| --- | --- |
| `/` | home.html |
| `/roth-versicherungen` (+ firmenkunden, cyber-police, privatkunden, tierkrankenversicherung, wichtige-hinweise, jobs, erstinformation, datenschutz, impressum) | versicherungen.html … |
| `/roth-finanz` (+ altersversorgung, sterbegeldversicherung, erstinformation, datenschutz, impressum) | finanz.html … |
| `/team`, `/kontakt-anfahrt`, `/sitemap` | team.html, kontakt.html, sitemap.html |

## How to…

### Add a page

1. Create `web/templates/pages/name.html` with `{{define "content"}}` (plus optional `title`/`description` blocks).
2. Add the route to the `Pages` map in `internal/server/server.go` — that single entry drives the route, the sitemap and the static export.
3. Add its strings to `locales/de.json`.

`go test ./...` verifies every page renders and that no translation key is missing.

### Edit content

All visible text lives in `locales/de.json`, grouped per page by blank lines. Lists use numbered keys (`x.items.1`, `x.items.2`, …) and are rendered with the `tlist` template function; changing text needs no template edits.

### Use the database

`internal/db` is dormant — nothing opens it today. To revive it, call `db.Open` + `db.Migrate` in `cmd/server/main.go` and pass the `*sql.DB` where needed; migrations are sequential `internal/db/migrations/NNNN_name.sql` files, tracked in `schema_migrations`.

### Deploy

```sh
make build      # builds CSS, compiles bin/server with templates/static/locales/migrations embedded
ENV=prod ./bin/server
```

The binary is fully self-contained — copy it to the server and run it. Configuration via env vars (see `.env.example`).

### Deploy to Vercel (static export)

The site is pure content, so on Vercel it deploys as prerendered static files instead of a Go serverless function: the build command in `vercel.json` runs `scripts/vercel-build.sh`, which downloads the Go toolchain (Vercel's build image has none) and runs `go run ./cmd/export`. That renders every route in `internal/server.Pages` to `public/<route>/index.html` (plus `404.html` and the static assets), and Vercel serves that directory from its CDN. Just push — no env vars needed. Remember to commit `web/static/css/output.css` after running `make css`, since the export embeds it as-is.

## Make targets

Run `make help` to list them: `dev`, `run`, `build`, `css`, `css-watch`, `tailwind`, `htmx` (update vendored htmx), `test`, `tidy`, `clean`.
