package pet

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"image/png"
	"net/http/httptest"
	"testing"
)

func TestPackagedRigResources(t *testing.T) {
	for _, tc := range []struct{ path, hash string }{{"/rig/shell.png", "8a30393afc9f64914c8f5f51389c827c8e2dee6702c17b0ccdaf6d294258edb3"}, {"/rig/tail.png", "2e2b60beb5552f3040cf9daa8473c9ed488b3897fa83beff5330964d48bb1836"}} {
		r := httptest.NewRecorder()
		if !serveRigResource(r, httptest.NewRequest("GET", "whale-pet://localhost"+tc.path, nil)) {
			t.Fatal("missing packaged route")
		}
		sum := sha256.Sum256(r.Body.Bytes())
		if hex.EncodeToString(sum[:]) != tc.hash {
			t.Fatal("original asset changed", tc.path)
		}
		image, err := png.DecodeConfig(bytes.NewReader(r.Body.Bytes()))
		if err != nil || image.Width != 502 || image.Height != 695 {
			t.Fatal("invalid original PNG", image, err)
		}
		head := httptest.NewRecorder()
		serveRigResource(head, httptest.NewRequest("HEAD", "whale-pet://localhost"+tc.path, nil))
		if head.Body.Len() != 0 || head.Header().Get("Content-Length") == "" {
			t.Fatal("HEAD exposes body or omits length")
		}
	}
	for _, url := range []string{"whale-pet://localhost/rig/other.png", "whale-pet://localhost/rig/../shell.png", "whale-pet://localhost/rig/shell.png?extra=1", "whale-pet://evil/rig/shell.png"} {
		r := httptest.NewRecorder()
		if serveRigResource(r, httptest.NewRequest("GET", url, nil)) {
			t.Fatal("unlisted route admitted", url)
		}
	}
}

func TestRigPageIntegration(t *testing.T) {
	p := &Companion{}
	r := httptest.NewRecorder()
	p.ServeHTTP(r, httptest.NewRequest("GET", "whale-pet://localhost/index.html", nil))
	if !bytes.Contains(r.Body.Bytes(), []byte(`data-rig="true"`)) {
		t.Fatal("page does not opt into bundled rig")
	}
	r = httptest.NewRecorder()
	p.ServeHTTP(r, httptest.NewRequest("GET", "whale-pet://localhost/rig/shell.png", nil))
	if r.Code != 200 || r.Header().Get("Content-Type") != "image/png" {
		t.Fatal("compositor asset route not connected")
	}
	p.sheet = []byte("selected sprite")
	r = httptest.NewRecorder()
	p.ServeHTTP(r, httptest.NewRequest("GET", "whale-pet://localhost/index.html", nil))
	if !bytes.Contains(r.Body.Bytes(), []byte(`data-rig="false"`)) {
		t.Fatal("selected skin must retain its renderer")
	}
}
