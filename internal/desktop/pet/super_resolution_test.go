package pet

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestEmbeddedAnime4KIdentityAndPublishedRoute(t *testing.T) {
	raw, err := assets.ReadFile("assets/anime4k-provenance.json")
	if err != nil {
		t.Fatal(err)
	}
	var provenance struct {
		Files []struct {
			Path   string
			SHA256 string `json:"sha256"`
		}
	}
	if err := json.Unmarshal(raw, &provenance); err != nil {
		t.Fatal(err)
	}
	if len(provenance.Files) != 2 {
		t.Fatal("missing shader or license identity")
	}
	for _, file := range provenance.Files {
		data, err := assets.ReadFile("assets/" + file.Path)
		if err != nil {
			t.Fatal(err)
		}
		sum := sha256.Sum256(data)
		if hex.EncodeToString(sum[:]) != file.SHA256 {
			t.Fatalf("import changed: %s", file.Path)
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
			t.Fatalf("%s unavailable", path)
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
