package handlers

import (
	"net/http"
	"os"
	"path/filepath"
	"strings"
)

type SPAHandler struct {
	staticPath string
	indexPath  string
	fileServer http.Handler
}

func NewSPAHandler(staticPath string, fileServer http.Handler) *SPAHandler {
	return &SPAHandler{
		staticPath: staticPath,
		indexPath:  filepath.Join(staticPath, "index.html"),
		fileServer: fileServer,
	}
}

func (h *SPAHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	// Clean the URL path to prevent directory traversal
	cleanPath := filepath.Clean(r.URL.Path)
	path := filepath.Join(h.staticPath, cleanPath)

	// Ensure the resulting path is still within staticPath
	if !strings.HasPrefix(path, h.staticPath) {
		http.ServeFile(w, r, h.indexPath)
		return
	}

	_, err := os.Stat(path)
	if os.IsNotExist(err) {
		http.ServeFile(w, r, h.indexPath)
		return
	} else if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	h.fileServer.ServeHTTP(w, r)
}
