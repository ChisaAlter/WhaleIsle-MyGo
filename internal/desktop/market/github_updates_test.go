package market

import (
	"context"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestGitHubCommitUpdateUsesExactCatalogAndUniqueInstalledProof(t *testing.T) {
	old := strings.Repeat("a", 40)
	latest := strings.Repeat("b", 40)
	item := CatalogItem{ID: "owner/demo", Homepage: "https://github.com/owner/demo", InstallSpec: "github:owner/demo"}
	if InstalledGitCommit(item, "github:owner/demo#"+old, "") != old {
		t.Fatal("pinned current missing")
	}
	dir := t.TempDir()
	lock := filepath.Join(dir, "pnpm-lock.yaml")
	os.WriteFile(lock, []byte("codeload.github.com/owner/demo/tar.gz/"+old), 0600)
	if InstalledGitCommit(item, "github:owner/demo", dir) != old {
		t.Fatal("lock current missing")
	}
	os.WriteFile(lock, []byte("codeload.github.com/owner/demo/tar.gz/"+old+"\ncodeload.github.com/owner/demo/tar.gz/"+latest), 0600)
	if InstalledGitCommit(item, "github:owner/demo", dir) != "" {
		t.Fatal("ambiguous lock accepted")
	}
	if InstalledGitCommit(item, "github:other/demo#"+old, "") != "" {
		t.Fatal("other repo accepted")
	}
	calls := 0
	svc := NewServiceWithClient("", &http.Client{Transport: updateTransport(func(r *http.Request) (*http.Response, error) {
		calls++
		if r.URL.String() != "https://api.github.com/repos/owner/demo/commits/HEAD" || r.Header.Get("Cookie") != "" || r.Header.Get("Authorization") != "" {
			t.Fatal("incorrect public lookup", r.URL)
		}
		return &http.Response{StatusCode: 200, Header: make(http.Header), Body: io.NopCloser(strings.NewReader(`{"sha":"` + latest + `"}`))}, nil
	})})
	svc.known[item.ID] = item
	if got, err := svc.LatestCommit(context.Background(), item); err != nil || got != latest {
		t.Fatal(got, err)
	}
	changed := item
	changed.InstallSpec = "github:other/demo"
	if _, err := svc.LatestCommit(context.Background(), changed); err == nil || calls != 1 {
		t.Fatal("unreviewed lookup")
	}
	if GitHubUpdateSpec(item, latest) != "github:owner/demo#"+latest {
		t.Fatal("commit not pinned")
	}
	item.InstallSpec = "github:owner/demo#path:/packages/plugin"
	if GitHubUpdateSpec(item, latest) != item.InstallSpec {
		t.Fatal("Host path installer contract changed")
	}
}
