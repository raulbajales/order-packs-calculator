package http

import (
	"io/fs"
	nethttp "net/http"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
)

// Creates a new router with all routes and middleware
func NewRouter(h *Handlers, webFS fs.FS) nethttp.Handler {
	r := chi.NewRouter()
	r.Use(middleware.Logger)
	r.Use(middleware.Recoverer) // For error 500 on panic

	r.Get("/", h.Index)
	r.Get("/app.css", func(w nethttp.ResponseWriter, req *nethttp.Request) {
		nethttp.FileServer(nethttp.FS(webFS)).ServeHTTP(w, req)
	})

	// API routes
	r.Get("/api/health", h.Health)
	r.Get("/api/packs", h.ListPacksAPI)
	r.Post("/api/packs", h.UpdatePacksAPI)
	r.Get("/api/calculate", h.CalculateAPI)

	return r
}
