package handlers

import "net/http"

func (a *App) manifest(writer http.ResponseWriter, request *http.Request) {
	writer.Header().Set("Content-Type", "application/manifest+json")
	writer.Header().Set("Cache-Control", "no-cache, max-age=0, must-revalidate")
	http.ServeFileFS(writer, request, mustStaticSub(), "manifest.webmanifest")
}

func (a *App) serviceWorker(writer http.ResponseWriter, request *http.Request) {
	writer.Header().Set("Content-Type", "application/javascript; charset=utf-8")
	writer.Header().Set("Cache-Control", "no-cache, max-age=0, must-revalidate")
	writer.Header().Set("Service-Worker-Allowed", "/")
	http.ServeFileFS(writer, request, mustStaticSub(), "js/sw.js")
}
