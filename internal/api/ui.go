package api

import (
	"net/http"

	"github.com/jdebrux/agentic-sim/web"
)

// registerUI mounts the embedded web UI at /ui/, serving its files directly
// from the compiled binary (see web.Assets). A bare /ui redirects to /ui/ so
// relative asset paths in index.html resolve correctly.
func registerUI(mux *http.ServeMux) {
	mux.Handle("/ui/", http.StripPrefix("/ui/", http.FileServerFS(web.Assets())))
	mux.HandleFunc("/ui", func(w http.ResponseWriter, r *http.Request) {
		http.Redirect(w, r, "/ui/", http.StatusMovedPermanently)
	})
}
