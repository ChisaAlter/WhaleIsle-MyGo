package webview

import (
	"errors"
	"html"
	"regexp"

	"github.com/egoist/mygo"
)

// This is the actual release identifier in mygo.json, the macOS bundle, and
// scripts/nsis/whaleisle.nsi. It identifies this app, never the current user.
// Keep it aligned with those packaging contracts if the application is renamed.
const youtubeApplicationOrigin = "https://ai.deepseek.harness.gui"

var youtubeVideoID = regexp.MustCompile(`^[A-Za-z0-9_-]{11}$`)

// NewYouTubeHost creates only the fixed player document in a fresh private
// browser. The HTTPS base supplies the app identity as the iframe's Referer:
// https://developers.google.com/youtube/terms/required-minimum-functionality
//
// The existing LoadHTML API supplies local document bytes at that base URL:
// WKWebView uses loadHTMLString:baseURL:, and WebView2 answers the initial
// navigation from its local HTML map. No app-origin server or proxy is used.
func NewYouTubeHost(parent *mygo.Window, videoID, title string, onChange, onClosed func()) (*Host, error) {
	if !youtubeVideoID.MatchString(videoID) {
		return nil, errors.New("invalid YouTube video ID")
	}
	embed := "https://www.youtube-nocookie.com/embed/" + videoID + "?autoplay=0&playsinline=1&rel=0&fs=0"
	base := youtubeApplicationOrigin + "/openui/youtube/" + videoID
	// A fixed script-free parent gives the cross-origin player no native
	// bindings or top-level navigation, popup, form, download or fullscreen
	// capability. Only the video ID and escaped display title are variable.
	document := `<!doctype html><html><head><meta charset="utf-8"><meta name="referrer" content="strict-origin-when-cross-origin"><meta http-equiv="Content-Security-Policy" content="default-src 'none'; frame-src https://www.youtube-nocookie.com; style-src 'unsafe-inline'; base-uri 'none'; form-action 'none'"><title>` + html.EscapeString(title) + `</title><style>html,body{margin:0;width:100%;height:100%;overflow:hidden;background:#000}iframe{display:block;width:100%;height:100%;border:0}</style></head><body><iframe title="` + html.EscapeString(title) + `" src="` + html.EscapeString(embed) + `" referrerpolicy="strict-origin-when-cross-origin" sandbox="allow-scripts allow-same-origin allow-presentation" allow="encrypted-media"></iframe></body></html>`
	return newHost(HostOptions{
		Parent: parent, URL: base, DisplayURL: embed, Title: title,
		TransientProfile: true, Untrusted: true,
		// The local parent never needs to leave its fixed document. The remote
		// iframe is separately sandboxed; ordinary browser hosts are unchanged.
		AllowNavigation: func(string) bool { return false },
		OnChange:        onChange, OnClosed: onClosed,
	}, document)
}
