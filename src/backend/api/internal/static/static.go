package static

import (
	"net/http"
	"os"
	"path/filepath"
)

// ServeSPA serves a Single Page Application.
// It serves static files if they exist, otherwise it falls back to serving index.html.
func ServeSPA(dir string) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		path := filepath.Join(dir, r.URL.Path)
		info, err := os.Stat(path)
		if os.IsNotExist(err) || info.IsDir() {
			http.ServeFile(w, r, filepath.Join(dir, "index.html"))
			return
		}
		http.FileServer(http.Dir(dir)).ServeHTTP(w, r)
	}
}
