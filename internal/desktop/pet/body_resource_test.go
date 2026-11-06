package pet

import (
	"bytes"
	"net/http"
	"net/http/httptest"
	"regexp"
	"strings"
	"testing"
)

func TestBodyResourceAndConsumerOrder(t *testing.T) {
	p := &Companion{}
	const csp = "default-src 'none'; img-src 'self'; style-src 'unsafe-inline'; script-src 'self' 'wasm-unsafe-eval'; connect-src 'self'; worker-src 'none'; frame-ancestors 'none'"
	get := func(method, target string) *httptest.ResponseRecorder {
		t.Helper()
		r := httptest.NewRecorder()
		p.ServeHTTP(r, httptest.NewRequest(method, target, nil))
		return r
	}
	r := get(http.MethodGet, "whale-pet://localhost/pet-body.js")
	if r.Code != http.StatusOK || r.Header().Get("Content-Type") != "text/javascript; charset=utf-8" || !bytes.Contains(r.Body.Bytes(), []byte("WhalePetBody")) {
		t.Fatalf("body resource status=%d MIME=%q body=%q", r.Code, r.Header().Get("Content-Type"), r.Body.String())
	}
	if r.Header().Get("Content-Security-Policy") != csp || r.Header().Get("X-Content-Type-Options") != "nosniff" || r.Header().Get("Cache-Control") != "no-store" {
		t.Fatalf("body resource weakened headers: %v", r.Header())
	}
	bodyLength := r.Header().Get("Content-Length")
	head := get(http.MethodHead, "whale-pet://localhost/pet-body.js")
	if head.Code != http.StatusOK || head.Body.Len() != 0 || head.Header().Get("Content-Length") != bodyLength || head.Header().Get("Content-Security-Policy") != csp {
		t.Fatalf("HEAD changed resource contract: %d %v %q", head.Code, head.Header(), head.Body.String())
	}
	t.Logf("GET /pet-body.js status=%d MIME=%s bytes=%d CSP=%s; HEAD body=%d", r.Code, r.Header().Get("Content-Type"), r.Body.Len(), r.Header().Get("Content-Security-Policy"), head.Body.Len())
	for _, target := range []string{"whale-pet://localhost/pet-body.js?extra=1", "whale-pet://elsewhere/pet-body.js", "whale-pet://localhost/pet-missing.js"} {
		if denied := get(http.MethodGet, target); denied.Code != http.StatusNotFound {
			t.Fatalf("unexpected resource %s status=%d", target, denied.Code)
		}
	}
	page := get(http.MethodGet, "whale-pet://localhost/index.html")
	if page.Code != http.StatusOK || page.Header().Get("Content-Security-Policy") != csp {
		t.Fatalf("page status=%d headers=%v", page.Code, page.Header())
	}
	scripts := regexp.MustCompile(`<script\s+src="([^"]+)"\s+defer></script>`).FindAllStringSubmatch(page.Body.String(), -1)
	position := map[string]int{}
	for i, entry := range scripts {
		if _, exists := position[entry[1]]; exists {
			t.Fatalf("duplicate script %s", entry[1])
		}
		position[entry[1]] = i
		script := get(http.MethodGet, "whale-pet://localhost/"+entry[1])
		if script.Code != http.StatusOK || script.Body.Len() == 0 || script.Header().Get("Content-Type") != "text/javascript; charset=utf-8" {
			t.Fatalf("packaged page script %s unavailable: status=%d bytes=%d MIME=%q", entry[1], script.Code, script.Body.Len(), script.Header().Get("Content-Type"))
		}
	}
	bodyPosition, ok := position["pet-body.js"]
	if !ok || strings.Count(page.Body.String(), `src="pet-body.js"`) != 1 {
		t.Fatal("body is not a single deferred packaged script")
	}
	for _, name := range []string{"pet-sleep.js", "pet-care-animation.js", "pet-drop.js", "pet-status.js", "pet-interaction.js", "pet-behavior.js", "pet-layout.js", "pet-runtime.js"} {
		if index, exists := position[name]; !exists || index <= bodyPosition {
			t.Fatalf("%s must execute after body: %v", name, position)
		}
	}
	if position["pet-sprite.js"] >= position["pet-runtime.js"] {
		t.Fatal("sprite cell selector must execute before the runtime consumes it")
	}
	t.Logf("served defer order=%v; unavailable path returns404 without changing CSP", position)
}
