package api

import (
	"net/http"
	"os"
	"path/filepath"
)

// DashboardRoutes serves the compiled React SPA.
func (s *Server) DashboardRoutes() http.Handler {
	distDir := "./web-client/dist"

	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		path := filepath.Join(distDir, r.URL.Path)

		// If root, serve index.html
		if r.URL.Path == "/" {
			http.ServeFile(w, r, filepath.Join(distDir, "index.html"))
			return
		}

		// Try to serve the exact file
		if info, err := os.Stat(path); err == nil && !info.IsDir() {
			http.ServeFile(w, r, path)
			return
		}

		// Fallback to index.html for SPA routing (e.g. /app, /login, /register)
		http.ServeFile(w, r, filepath.Join(distDir, "index.html"))
	})
}
