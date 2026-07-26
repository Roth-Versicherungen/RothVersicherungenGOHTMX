package server

import (
	"log/slog"
	"net/http"
	"runtime/debug"
	"time"

	"github.com/maxroth/eumel/internal/i18n"
	"github.com/maxroth/eumel/internal/view"
)

// Language resolves the request language, stores it on the context and
// persists an explicit ?lang= choice in a cookie.
func Language(tr *i18n.Translator) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			lang := tr.Resolve(r)
			if q := r.URL.Query().Get("lang"); q == lang && tr.Supported(q) {
				http.SetCookie(w, &http.Cookie{
					Name:     i18n.CookieName,
					Value:    lang,
					Path:     "/",
					MaxAge:   365 * 24 * 60 * 60,
					HttpOnly: true,
					SameSite: http.SameSiteLaxMode,
				})
			}
			next.ServeHTTP(w, r.WithContext(i18n.WithLang(r.Context(), lang)))
		})
	}
}

// Logger logs every request with method, path, status and duration.
func Logger(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()
		rec := &statusRecorder{ResponseWriter: w, status: http.StatusOK}
		next.ServeHTTP(rec, r)
		slog.Info("request",
			"method", r.Method,
			"path", r.URL.Path,
			"status", rec.status,
			"duration", time.Since(start).Round(time.Microsecond),
		)
	})
}

// Recover turns panics into a rendered 500 page instead of a dropped
// connection.
func Recover(v *view.View) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			defer func() {
				if rec := recover(); rec != nil {
					slog.Error("panic", "err", rec, "stack", string(debug.Stack()))
					v.Render(w, r, http.StatusInternalServerError, "error.html", map[string]any{
						"Status": http.StatusInternalServerError,
					})
				}
			}()
			next.ServeHTTP(w, r)
		})
	}
}

type statusRecorder struct {
	http.ResponseWriter
	status int
}

func (r *statusRecorder) WriteHeader(status int) {
	r.status = status
	r.ResponseWriter.WriteHeader(status)
}
