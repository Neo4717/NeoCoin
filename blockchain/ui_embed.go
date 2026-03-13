package main

import (
	"embed"
	"io/fs"
	"net/http"
)

//go:embed ui/*
var embeddedUI embed.FS

func ExplorerFileServer() http.Handler {
	sub, err := fs.Sub(embeddedUI, "ui")
	if err != nil {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			http.Error(w, "explorer ui not available", http.StatusInternalServerError)
		})
	}
	return http.FileServer(http.FS(sub))
}
