package components

import (
	"context"
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestBridgeGitHubPinnedReleaseFixture(t *testing.T) {
	payload := []byte("fixed version fixture only")
	digest := fmt.Sprintf("%x", sha256.Sum256(payload))
	paths := []string{}
	version := "1.3.7"
	bad := false
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		paths = append(paths, r.URL.Path)
		switch r.URL.Path {
		case "/releases/latest/download/" + bridgeManifest:
			hash := digest
			if bad {
				hash = strings.Repeat("0", 64)
			}
			json.NewEncoder(w).Encode(bridgeReleaseManifest{ID: "whalebridge", Version: version, License: "MIT License", Platforms: map[string]bridgeSpec{"win32-x64": {Asset: bridgeAsset, SHA256: hash, Size: int64(len(payload))}}})
		case "/releases/download/whalebridge-v1.3.7/" + bridgeAsset:
			w.Write(payload)
		default:
			t.Errorf("unexpected unpinned/API request: %s", r.URL.Path)
			http.NotFound(w, r)
		}
	}))
	defer srv.Close()
	endpoint := srv.URL + "/releases/latest/download/" + bridgeManifest
	dir := t.TempDir()
	if err := installBridge(context.Background(), dir, endpoint, srv.Client(), true); err != nil {
		t.Fatal(err)
	}
	if len(paths) != 2 || !strings.Contains(paths[1], "/whalebridge-v1.3.7/") {
		t.Fatalf("requests: %v", paths)
	}
	got, err := os.ReadFile(filepath.Join(dir, "components", "whalebridge", "versions", "1.3.7", bridgeAsset))
	if err != nil || string(got) != string(payload) {
		t.Fatalf("payload %q: %v", got, err)
	}
	bad = true
	other := t.TempDir()
	if err := installBridge(context.Background(), other, endpoint, srv.Client(), true); err == nil {
		t.Fatal("bad checksum accepted")
	}
	if readInstalled(filepath.Join(other, "components", "whalebridge", "installed.json")).Version != "" {
		t.Fatal("bad checksum published")
	}
	version = "../invalid"
	before := len(paths)
	if err := installBridge(context.Background(), t.TempDir(), endpoint, srv.Client(), true); err == nil {
		t.Fatal("bad version accepted")
	}
	if len(paths) != before+1 {
		t.Fatal("downloaded invalid manifest payload")
	}
}
