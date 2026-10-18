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

func TestBridgeFirstInstallFixture(t *testing.T) {
	payload := []byte("fixture bytes, never executable")
	digest := fmt.Sprintf("%x", sha256.Sum256(payload))
	var base string
	bad := false
	hits := 0
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		hits++
		switch r.URL.Path {
		case "/releases":
			json.NewEncoder(w).Encode([]bridgeRelease{{Tag: "whalebridge-v1.2.3", Assets: []bridgeReleaseAsset{{Name: bridgeManifest, URL: base + "/manifest"}, {Name: bridgeAsset, URL: base + "/payload", Size: int64(len(payload))}}}})
		case "/manifest":
			hash := digest
			if bad {
				hash = strings.Repeat("0", 64)
			}
			json.NewEncoder(w).Encode(bridgeReleaseManifest{ID: "whalebridge", Version: "1.2.3", License: "MIT License", Platforms: map[string]bridgeSpec{"win32-x64": {Asset: bridgeAsset, SHA256: hash, Size: int64(len(payload))}}})
		case "/payload":
			w.Write(payload)
		default:
			http.NotFound(w, r)
		}
	}))
	defer srv.Close()
	base = srv.URL
	dir := t.TempDir()
	root := filepath.Join(dir, "components", "whalebridge")
	bad = true
	if err := installBridge(context.Background(), dir, base+"/releases", srv.Client(), true); err == nil {
		t.Fatal("accepted corrupt checksum")
	}
	if _, err := os.Stat(filepath.Join(root, "installed.json")); !os.IsNotExist(err) {
		t.Fatal("published failed install")
	}
	rows := ListUserData(dir)
	if len(rows) != 1 || rows[0].InstalledVersion != "" {
		t.Fatalf("false installed state: %+v", rows)
	}
	bad = false
	if err := installBridge(context.Background(), dir, base+"/releases", srv.Client(), true); err != nil {
		t.Fatal(err)
	}
	got, err := os.ReadFile(filepath.Join(root, "versions", "1.2.3", bridgeAsset))
	if err != nil || string(got) != string(payload) {
		t.Fatalf("binary: %q %v", got, err)
	}
	rows = ListUserData(dir)
	if rows[0].InstalledVersion != "1.2.3" || rows[0].State != "stopped" {
		t.Fatalf("row: %+v", rows[0])
	}
	before := hits
	if err := installBridge(context.Background(), dir, base+"/releases", srv.Client(), true); err == nil {
		t.Fatal("allowed replacement")
	}
	if hits != before {
		t.Fatal("replacement fetched before refusal")
	}
	if _, err := os.Stat(filepath.Join(root, "data", "state.json")); !os.IsNotExist(err) {
		t.Fatal("unexpected runtime execution")
	}
}
