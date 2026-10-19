// Package htmlpreview provides the authenticated desktop route's fail-closed
// capture boundary. It deliberately owns no browser, filesystem or credentials.
package htmlpreview

import (
	"context"
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

type request struct {
	inertImage bool   // Trusted helper-only mode, never accepted from JSON.
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
type Renderer func(context.Context, string, int, string) (CaptureResult, error)

func Handler() http.Handler { return HandlerWithRenderer(nil) }

// HandlerWithRenderer preserves request validation and readiness reporting.
// The supplied trusted callback owns its separate process isolation gate.
func HandlerWithRenderer(render Renderer) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { serveWithRenderer(w, r, render) })
}

func serveWithRenderer(w http.ResponseWriter, r *http.Request, render Renderer) {
	if r.Context().Err() != nil {
		return
	}
	if r.URL.Path == "/desktop/html-preview/status" && r.Method == http.MethodGet {
		w.Header().Set("Content-Type", "application/json; charset=utf-8")
		w.Header().Set("Cache-Control", "no-store")
		_ = json.NewEncoder(w).Encode(Readiness())
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
	var result CaptureResult
	if render == nil {
		result, err = capturePrepared(r.Context(), input)
	} else {
		result, err = render(r.Context(), input.HTML, input.Width, input.Appearance)
	}
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
