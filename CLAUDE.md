# Roth Versicherungen — Go + HTMX

Server-rendered website for Roth Versicherungen / Roth Finanz (Langen), built on the Eumel template: stdlib router + `html/template`, HTMX (vendored in `web/static/js/`), Tailwind v4 standalone CLI, SQLite (modernc, no CGO), JSON i18n.

## Commands

- `make dev` — build CSS + run server in dev mode (templates/static re-read from disk per request)
- `make test` — run tests; `make build` — production binary with everything embedded
- `make css` — rebuild Tailwind output (needed after adding new utility classes)

## Conventions

- **Never hardcode UI text** in templates or handlers. Every string lives in `locales/de.json` (one flat JSON file, dot keys like `nav.team`, grouped per page by blank lines) and is used via `{{t "key"}}` in templates. Args make it a Sprintf: `{{t "footer.copyright" year}}`. Lists use numbered keys (`x.items.1`, `x.items.2`, …) iterated with `{{range tlist "x.items"}}`. `internal/server`'s `TestPagesRender` fails on any key a template references but de.json doesn't define.
- The site is German-only; `DEFAULT_LANG` defaults to `de`.
- Pages live in `web/templates/pages/` and define a `content` block (plus optional `title`/`description` blocks) rendered inside `layouts/base.html`. All pages are static content: add the route to the `Pages` map in `internal/server/server.go` — that one entry drives routing, the sitemap and the static export.
- Shared building blocks are partials in `web/templates/partials/`: `page-hero`, `section-head`, `cta`, `link-card`, `nav`, `footer`, `legal-address`, `legal-text-section`. They take named parameters via the `dict` template function. HTMX endpoints (none yet) would render partials via `View.RenderPartial`.
- Brand theme (colors `brand-red`/`brand-page`/…, `shadow-card`, `rounded-4xl`, Inter font) is defined in `@theme` in `web/static/css/input.css`. Header dropdowns/mobile menu are progressive enhancement in `web/static/js/nav.js`.
- `internal/db` is dormant: nothing opens the database. To use it, wire `db.Open` + `db.Migrate` back into `cmd/server/main.go`; migrations are sequential `internal/db/migrations/NNNN_name.sql`.
- Config is env-vars only (`internal/config`); defaults must keep the server runnable with zero setup. `BASE_URL` feeds canonical/OG tags, `robots.txt` and `sitemap.xml`.
- Deploys: any host can run the `make build` binary; Vercel serves the static export (`cmd/export` → `public/`, driven by `vercel.json` + `scripts/vercel-build.sh`). CI (`.github/workflows/ci.yml`) runs gofmt/vet/test/export.
