package handler

import (
	"io/fs"
	"net/http"
	"strings"

	"go02-neighbor-help/internal/respond"
)

func (h *Handler) registerPageRoutes(mux *http.ServeMux) {
	if h.assets == nil {
		return
	}
	staticFS, err := fs.Sub(h.assets, "static")
	if err == nil {
		mux.Handle("GET /static/", http.StripPrefix("/static/", http.FileServer(http.FS(staticFS))))
	}
	mux.HandleFunc("GET /", h.servePage("index.html"))
	mux.HandleFunc("GET /login", h.servePage("login.html"))
	mux.HandleFunc("GET /app", h.servePage("app.html"))
	mux.HandleFunc("GET /me", h.servePage("me.html"))
	mux.HandleFunc("GET /admin", h.servePage("admin.html"))
	mux.HandleFunc("GET /posts/{id}", h.servePage("post.html"))
}

func (h *Handler) servePage(name string) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/" && name == "index.html" {
			http.Redirect(w, r, "/login", http.StatusFound)
			return
		}
		data, err := fs.ReadFile(h.assets, name)
		if err != nil {
			respond.Error(w, http.StatusNotFound, "not_found", "页面不存在")
			return
		}
		w.Header().Set("Content-Type", contentTypeFor(name))
		_, _ = w.Write(data)
	}
}

func contentTypeFor(name string) string {
	switch {
	case strings.HasSuffix(name, ".html"):
		return "text/html; charset=utf-8"
	case strings.HasSuffix(name, ".css"):
		return "text/css; charset=utf-8"
	case strings.HasSuffix(name, ".js"):
		return "application/javascript; charset=utf-8"
	default:
		return "text/plain; charset=utf-8"
	}
}
