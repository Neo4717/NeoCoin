package http

import (
	"embed"
	"io/fs"
	"net/http"
)

//go:embed public
var publicFS embed.FS

func ExplorerFileServer() http.Handler {
	subFS, err := fs.Sub(publicFS, "public/explorer")
	if err != nil {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			http.Error(w, "explorer UI not available", http.StatusNotFound)
		})
	}
	return http.FileServer(http.FS(subFS))
}

func WalletFileServer() http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.Error(w, "wallet UI not available", http.StatusNotFound)
	})
}
