// Package handlers contains all HTTP handlers. Full pages render
// through View.Render, HTMX endpoints through View.RenderPartial.
package handlers

import (
	"database/sql"
	"log/slog"
	"net/http"

	"github.com/maxroth/eumel/internal/view"
)

type Handler struct {
	DB   *sql.DB
	View *view.View
}

// Page returns a handler that renders the named static page template.
// All content comes from locales/, so no per-page data is needed.
func (h *Handler) Page(name string) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		h.View.Render(w, r, http.StatusOK, name, nil)
	}
}

func (h *Handler) NotFound(w http.ResponseWriter, r *http.Request) {
	h.View.Render(w, r, http.StatusNotFound, "error.html", map[string]any{
		"Status": http.StatusNotFound,
	})
}

// Error logs err and renders the 500 page.
func (h *Handler) Error(w http.ResponseWriter, r *http.Request, err error) {
	slog.Error("handler", "path", r.URL.Path, "err", err)
	h.View.Render(w, r, http.StatusInternalServerError, "error.html", map[string]any{
		"Status": http.StatusInternalServerError,
	})
}
