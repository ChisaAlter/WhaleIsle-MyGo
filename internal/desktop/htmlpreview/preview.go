// Package htmlpreview provides the authenticated desktop route's fail-closed
// capture boundary. It deliberately owns no browser, filesystem or credentials.
package htmlpreview

import (
	"encoding/json"
	"io"
	"net/http"
	"strings"
	"time"
	"unicode/utf8"
)

// Limits mirror src/main/html-preview.js and desktop-install-control.js.
// The Host prepares/embeds authorized local images before making this request.
const (
	MaxPreparedHTMLBytes = 25 * 1024 * 1024
	MaxRequestBytes      = 52 * 1024 * 1024
	MinWidth             = 240
	MaxWidth             = 1600
	MaxCaptureHeight     = 4000
	CaptureTimeout       = 20 * time.Second
	UnavailableReason    = "HTML preview rendering service is unavailable: an isolated, network-blocked capture context has not been verified on this platform."
)

// Available is intentionally false on every platform. MyGo's CapturePage is
// real PNG capture, but ordinary windows share browser state and do not provide
// an all-resource deny policy. Neither DisableBindings nor a hidden window
// establishes the isolation required to execute supplied HTML safely.
func Available() bool { return false }

type request struct {
	HTML       string `json:"html"`
	Width      int    `json:"width"`
	Appearance string `json:"appearance"`
}

// Handler must be mounted AFTER the install server's existing Bearer-token
// authentication. Never serve this handler from an unauthenticated listener.
// It validates the existing wire contract but never executes supplied HTML,
// reads local images, opens a window, or returns fabricated screenshot data.
// A future renderer must supply a fresh isolated context, deny network, native
// bindings, navigation, popups, permissions and downloads before loading any
// HTML, capture PNG in memory, and destroy that context on cancellation/20s.
func Handler() http.Handler { return http.HandlerFunc(serve) }

func serve(w http.ResponseWriter, r *http.Request) {
	if r.Context().Err() != nil {
		return
	}
	if r.Method != http.MethodPost {
		w.Header().Set("Allow", http.MethodPost)
		reply(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}
	body, err := io.ReadAll(http.MaxBytesReader(w, r.Body, MaxRequestBytes))
	if r.Context().Err() != nil {
		return
	}
	if err != nil || !utf8.Valid(body) {
		reply(w, http.StatusBadRequest, "invalid or oversized HTML preview request")
		return
	}
	var input request
	if json.Unmarshal(body, &input) != nil {
		reply(w, http.StatusBadRequest, "invalid or oversized HTML preview request")
		return
	}
	if strings.TrimSpace(input.HTML) == "" || len(input.HTML) > MaxPreparedHTMLBytes {
		reply(w, http.StatusBadRequest, "Prepared HTML must contain 1 to 25 MiB of UTF-8 content.")
		return
	}
	if input.Width < MinWidth || input.Width > MaxWidth {
		reply(w, http.StatusBadRequest, "Preview width must be an integer from 240 to 1600.")
		return
	}
	if input.Appearance != "light" && input.Appearance != "dark" {
		reply(w, http.StatusBadRequest, "Preview appearance must be light or dark.")
		return
	}
	// Both entry points keep the production readiness gate closed until the
	// Windows runtime isolation/capture checks have been verified.
	result, err := capturePrepared(r.Context(), input)
	if r.Context().Err() != nil {
		return
	}
	if err != nil {
		reply(w, http.StatusServiceUnavailable, err.Error())
		return
	}
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.Header().Set("Cache-Control", "no-store")
	_ = json.NewEncoder(w).Encode(result)
}

func reply(w http.ResponseWriter, status int, message string) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.Header().Set("Cache-Control", "no-store")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(map[string]string{"error": message})
}
