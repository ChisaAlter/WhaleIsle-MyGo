package pet

import (
	"embed"
	"fmt"
	"net/http"
)

// Original MIT-licensed images and attribution ship inside every native build.
//
//go:embed assets/pet-rig.js assets/rig/*
var rigResources embed.FS

// Only these compiled files are addressable. The manifest cannot add routes or
// load arbitrary local files, network resources, scripts or model packages.
func serveRigResource(w http.ResponseWriter, r *http.Request) bool {
	if r.URL.Host != "localhost" || r.URL.User != nil || r.URL.RawQuery != "" || r.URL.Fragment != "" || (r.Method != "GET" && r.Method != "HEAD") {
		return false
	}
	switch r.URL.Path {
	case "/pet-rig.js", "/rig/manifest.json", "/rig/shell.png", "/rig/tail.png":
	default:
		return false
	}
	body, err := rigResources.ReadFile("assets" + r.URL.Path)
	if err != nil {
		http.NotFound(w, r)
		return true
	}
	contentType := renderContentType(r.URL.Path)
	if r.URL.Path == "/rig/manifest.json" {
		contentType = "application/json; charset=utf-8"
	}
	w.Header().Set("Content-Type", contentType)
	w.Header().Set("Content-Length", fmt.Sprint(len(body)))
	if r.Method != "HEAD" {
		_, _ = w.Write(body)
	}
	return true
}
