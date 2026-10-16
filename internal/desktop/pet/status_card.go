package pet

import (
	_ "embed"
	"net/http"
)

//go:embed assets/pet-status.js
var statusCardScript []byte

func serveStatusCard(w http.ResponseWriter, r *http.Request) bool {
	if r.URL.Path != "/pet-status.js" {
		return false
	}
	w.Header().Set("Content-Type", "text/javascript; charset=utf-8")
	_, _ = w.Write(statusCardScript)
	return true
}
