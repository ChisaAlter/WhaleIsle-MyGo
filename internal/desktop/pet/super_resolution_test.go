package pet

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestEmbeddedAnime4KIdentityAndPublishedRoute(t *testing.T) {
	// anime4k.js must NOT be embedded in assets to keep binary footprint lean.
	if f, err := assets.Open("assets/anime4k.js"); err == nil {
		_ = f.Close()
		t.Fatal("assets/anime4k.js must not be go:embedded")
	}

	raw, err := assets.ReadFile("assets/anime4k-provenance.json")
	if err != nil {
		t.Fatal(err)
	}
	var provenance struct {
		Files []struct {
			Path     string
			SHA256   string `json:"sha256"`
			LFSHA256 string `json:"lfSha256"`
		}
	}
	if err := json.Unmarshal(raw, &provenance); err != nil {
		t.Fatal(err)
	}
	if len(provenance.Files) != 2 {
		t.Fatal("missing shader or license identity")
	}
	for _, file := range provenance.Files {
		data, readErr := os.ReadFile(filepath.Join("assets", file.Path))
		if readErr != nil {
			t.Fatal(readErr)
		}
		sum := sha256.Sum256(data)
		got := hex.EncodeToString(sum[:])
		if got != file.SHA256 && (file.LFSHA256 == "" || got != file.LFSHA256) {
			t.Fatalf("import changed on disk: %s (got %s)", file.Path, got)
		}
	}
	license, _ := assets.ReadFile("assets/anime4k.LICENSE")
	if !strings.Contains(string(license), "Copyright (c) 2023") || !strings.Contains(string(license), "MIT License") {
		t.Fatal("upstream license absent")
	}

	p := &Companion{}
	for _, path := range []string{"/anime4k.js", "/pet-super-resolution.js", "/pet-layout.js"} {
		res := httptest.NewRecorder()
		p.ServeHTTP(res, httptest.NewRequest("GET", "whale-pet://localhost"+path, nil))
		if res.Code != 200 || res.Body.Len() == 0 || res.Header().Get("Content-Type") != "text/javascript; charset=utf-8" {
			t.Fatalf("%s unavailable: code=%d len=%d ct=%q", path, res.Code, res.Body.Len(), res.Header().Get("Content-Type"))
		}
		head := httptest.NewRecorder()
		p.ServeHTTP(head, httptest.NewRequest("HEAD", "whale-pet://localhost"+path, nil))
		if head.Code != 200 || head.Body.Len() != 0 || head.Header().Get("Content-Length") == "" {
			t.Fatalf("%s HEAD", path)
		}
	}
	res := httptest.NewRecorder()
	p.ServeHTTP(res, httptest.NewRequest("GET", "whale-pet://localhost/anime4k.js?script=other", nil))
	if res.Code != 404 {
		t.Fatal("query expanded executable routes")
	}
}

func TestOptionalAnime4KRuntimeLookupAndGracefulDegradation(t *testing.T) {
	prevRoot := configuredRenderRoot.Load()
	t.Cleanup(func() { configuredRenderRoot.Store(prevRoot) })

	wantSize, wantSHA, ok := expectedAnime4KIdentity()
	if !ok {
		t.Fatal("provenance identity missing")
	}
	diskBytes, err := os.ReadFile(filepath.Join("assets", "anime4k.js"))
	if err != nil {
		t.Fatal(err)
	}

	// Empty temporary environment + working dir: must gracefully degrade without error.
	isolated := t.TempDir()
	emptyCfg := filepath.Join(t.TempDir(), "user-data")
	t.Setenv("WHALEISLE_PET_RENDER_ASSETS", "")
	t.Setenv("WHALEISLE_USER_DATA", emptyCfg)
	emptySelected := ""
	configuredRenderRoot.Store(&emptySelected)

	origWD, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	if err := os.Chdir(isolated); err != nil {
		t.Fatal(err)
	}
	defer func() { _ = os.Chdir(origWD) }()

	p := &Companion{}
	res := httptest.NewRecorder()
	p.ServeHTTP(res, httptest.NewRequest("GET", "whale-pet://localhost/anime4k.js", nil))
	if res.Code != 200 || res.Header().Get("Content-Type") != "text/javascript; charset=utf-8" {
		t.Fatalf("degraded response unexpected: code=%d ct=%q", res.Code, res.Header().Get("Content-Type"))
	}
	if string(res.Body.Bytes()) != anime4kFallbackScript {
		t.Fatalf("expected fallback stub when absent, got %d bytes", res.Body.Len())
	}

	// Tampered file in userData/pet-render-assets must be rejected by SHA-256 verification.
	userPetAssets := filepath.Join(emptyCfg, "pet-render-assets")
	if err := os.MkdirAll(userPetAssets, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(userPetAssets, "anime4k.js"), []byte("alert('tampered')"), 0o600); err != nil {
		t.Fatal(err)
	}
	resTampered := httptest.NewRecorder()
	p.ServeHTTP(resTampered, httptest.NewRequest("GET", "whale-pet://localhost/anime4k.js", nil))
	if string(resTampered.Body.Bytes()) != anime4kFallbackScript {
		t.Fatal("accepted tampered anime4k.js")
	}

	// Valid verified copy in userData/pet-render-assets is served verbatim.
	if err := os.WriteFile(filepath.Join(userPetAssets, "anime4k.js"), diskBytes, 0o600); err != nil {
		t.Fatal(err)
	}
	resUserData := httptest.NewRecorder()
	p.ServeHTTP(resUserData, httptest.NewRequest("GET", "whale-pet://localhost/anime4k.js", nil))
	if int64(resUserData.Body.Len()) != wantSize {
		t.Fatalf("served size=%d want %d", resUserData.Body.Len(), wantSize)
	}
	sum := sha256.Sum256(resUserData.Body.Bytes())
	if hex.EncodeToString(sum[:]) != wantSHA {
		t.Fatal("served checksum mismatch")
	}

	// LF-normalized download bytes (matching GitHub raw) are also verified & served.
	lfBytes := bytes.ReplaceAll(diskBytes, []byte("\r\n"), []byte("\n"))
	if err := os.WriteFile(filepath.Join(userPetAssets, "anime4k.js"), lfBytes, 0o600); err != nil {
		t.Fatal(err)
	}
	resLF := httptest.NewRecorder()
	p.ServeHTTP(resLF, httptest.NewRequest("GET", "whale-pet://localhost/anime4k.js", nil))
	if resLF.Body.Len() != 9398501 {
		t.Fatalf("LF served size=%d want 9398501", resLF.Body.Len())
	}
	lfSum := sha256.Sum256(resLF.Body.Bytes())
	if hex.EncodeToString(lfSum[:]) != "b4b6fcbe537f4a1479854fcedb470f782c696d46f6534323814bec0f3245b240" {
		t.Fatal("LF served checksum mismatch")
	}

	// WHALEISLE_PET_RENDER_ASSETS takes priority over userData when present & valid.
	envAssets := filepath.Join(t.TempDir(), "render-assets")
	if err := os.MkdirAll(envAssets, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(envAssets, "anime4k.js"), diskBytes, 0o600); err != nil {
		t.Fatal(err)
	}
	_ = os.Remove(filepath.Join(userPetAssets, "anime4k.js"))
	t.Setenv("WHALEISLE_PET_RENDER_ASSETS", envAssets)
	resEnv := httptest.NewRecorder()
	p.ServeHTTP(resEnv, httptest.NewRequest("GET", "whale-pet://localhost/anime4k.js", nil))
	if int64(resEnv.Body.Len()) != wantSize {
		t.Fatalf("env override served size=%d want %d", resEnv.Body.Len(), wantSize)
	}
}
