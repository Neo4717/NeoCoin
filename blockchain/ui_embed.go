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

func WalletFileServer() http.Handler {
	sub, err := fs.Sub(embeddedUI, "ui")
	if err != nil {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			http.Error(w, "wallet ui not available", http.StatusInternalServerError)
		})
	}
	fs := http.FS(sub)
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "" || r.URL.Path == "/" {
			r.URL.Path = "/wallet.html"
		}
		http.FileServer(fs).ServeHTTP(w, r)
	})
}
