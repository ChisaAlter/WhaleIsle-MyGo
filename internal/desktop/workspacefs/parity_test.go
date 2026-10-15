package workspacefs

import (
	"context"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"slices"
	"strings"
	"testing"
)

func TestVisibleDotfilesAndGitIgnoreParity(t *testing.T) {
	root := t.TempDir()
	if out, err := exec.Command("git", "init", root).CombinedOutput(); err != nil {
		t.Skipf("git unavailable: %s %v", out, err)
	}
	for name, text := range map[string]string{".env": "ok", ".gitignore": "*.log\ncache/\n!keep.log\n", "drop.log": "ignored", "keep.log": "kept", "normal.txt": "ok"} {
		if err := os.WriteFile(filepath.Join(root, name), []byte(text), 0600); err != nil {
			t.Fatal(err)
		}
	}
	os.Mkdir(filepath.Join(root, "cache"), 0700)
	ents, err := ListWorkspaceDir(context.Background(), root, "")
	if err != nil {
		t.Fatal(err)
	}
	var names []string
	for _, e := range ents {
		names = append(names, e.Name)
	}
	for _, name := range []string{".env", ".gitignore", "keep.log", "normal.txt"} {
		if !slices.Contains(names, name) {
			t.Errorf("missing %s in %v", name, names)
		}
	}
	for _, name := range []string{".git", "drop.log", "cache"} {
		if slices.Contains(names, name) {
			t.Errorf("unexpected %s in %v", name, names)
		}
	}
	os.Mkdir(filepath.Join(root, "nested"), 0700)
	os.WriteFile(filepath.Join(root, "nested", ".gitignore"), []byte("hide.txt\n"), 0600)
	os.WriteFile(filepath.Join(root, "nested", "hide.txt"), []byte("x"), 0600)
	os.WriteFile(filepath.Join(root, "nested", ".visible"), []byte("x"), 0600)
	paths, err := SearchRelPaths(context.Background(), root, "", nil)
	if err != nil {
		t.Fatal(err)
	}
	if slices.Contains(paths, "nested/hide.txt") || !slices.Contains(paths, "nested/.visible") {
		t.Fatalf("nested ignore mismatch: %v", paths)
	}
}

func TestSearchHasNoIndexOrMatchTruncation(t *testing.T) {
	root := t.TempDir()
	for i := 0; i < 2105; i++ {
		name := fmt.Sprintf("target-%04d.txt", i)
		if err := os.WriteFile(filepath.Join(root, name), nil, 0600); err != nil {
			t.Fatal(err)
		}
	}
	paths, err := SearchRelPaths(context.Background(), root, "target", nil)
	if err != nil {
		t.Fatal(err)
	}
	if len(paths) != 2105 || !slices.Contains(paths, "target-2104.txt") {
		t.Fatalf("search truncated: %d", len(paths))
	}
	all, err := IndexRelPaths(root, 2000)
	if err != nil || len(all) != 2105 {
		t.Fatalf("index truncated: %d, %v", len(all), err)
	}
}

func TestSearchCancellationAndRemoteTraversal(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	if _, err := SearchRelPaths(ctx, t.TempDir(), "", nil); !errors.Is(err, context.Canceled) {
		t.Fatalf("cancel = %v", err)
	}
	calls := 0
	remote := func(ctx context.Context, rel string) ([]Entry, error) {
		calls++
		if rel == "" {
			return []Entry{{Name: "nested", IsDir: true}, {Name: ".env"}, {Name: ".git", IsDir: true}}, nil
		}
		if rel == "nested" {
			return []Entry{{Name: "target.go"}}, nil
		}
		t.Fatalf("unexpected remote traversal: %q", rel)
		return nil, nil
	}
	paths, err := SearchRelPaths(context.Background(), filepath.Join(t.TempDir(), "unavailable"), "", remote)
	if err != nil || calls != 2 || !slices.Contains(paths, "nested/target.go") || !slices.Contains(paths, ".env") {
		t.Fatalf("remote paths = %v, calls=%d err=%v", paths, calls, err)
	}
	bad := func(context.Context, string) ([]Entry, error) { return []Entry{{Name: "../escape", IsDir: true}}, nil }
	if _, err := SearchRelPaths(context.Background(), "missing", "", bad); err == nil {
		t.Fatal("accepted untrusted remote path")
	}
}

func TestWorkspaceBoundaryAndSymlinkTraversal(t *testing.T) {
	root, outside := t.TempDir(), t.TempDir()
	os.WriteFile(filepath.Join(outside, "private.txt"), []byte("private"), 0600)
	if err := os.Symlink(outside, filepath.Join(root, "escape")); err != nil {
		t.Skip(err)
	}
	if _, err := JoinRoot(root, "escape/private.txt"); err == nil {
		t.Fatal("joined outside symlink")
	}
	os.Mkdir(filepath.Join(root, ".git"), 0700)
	os.WriteFile(filepath.Join(root, ".git", "config"), nil, 0600)
	os.Symlink(filepath.Join(root, ".git"), filepath.Join(root, "git-alias"))
	if _, err := JoinRoot(root, "git-alias/config"); err == nil {
		t.Fatal("joined git alias")
	}
	for _, rel := range []string{"../private.txt", ".git/config", "/absolute", "dir/../file"} {
		if _, err := JoinRoot(root, rel); err == nil {
			t.Fatalf("accepted %q", rel)
		}
	}
	if _, err := JoinRoot(root, "version..txt"); err != nil {
		t.Fatalf("legal filename rejected: %v", err)
	}
	paths, err := SearchRelPaths(context.Background(), root, "", nil)
	if err != nil {
		t.Fatal(err)
	}
	for _, p := range paths {
		if strings.Contains(p, "private") || strings.Contains(p, "config") {
			t.Fatalf("followed symlink: %v", paths)
		}
	}
}

func TestSearchRejectsLateBindingAndQueryResults(t *testing.T) {
	var search Search
	firstStarted := make(chan struct{})
	release := make(chan struct{})
	firstDone := make(chan struct{})
	first := func(context.Context, string) ([]Entry, error) {
		close(firstStarted)
		<-release
		defer close(firstDone)
		return []Entry{{Name: "old-result"}}, nil
	}
	search.Ensure("host-one/session-one", "missing-old", "old", first, nil)
	<-firstStarted
	done := make(chan struct{}, 1)
	second := func(context.Context, string) ([]Entry, error) { return []Entry{{Name: "new-result"}}, nil }
	search.Ensure("host-two/session-two", "missing-new", "new", second, func() { done <- struct{}{} })
	<-done
	close(release)
	<-firstDone
	snap := search.Snapshot()
	if snap.Busy || snap.Total != 1 || snap.Paths[0] != "new-result" {
		t.Fatalf("stale result applied: %+v", snap)
	}
	search.Ensure("host-two/session-two", "missing-new", "", second, nil)
	if snap = search.Snapshot(); snap.Busy || snap.Total != 0 {
		t.Fatalf("clear retained results: %+v", snap)
	}
}

func TestSearchPagingRetainsEveryMatch(t *testing.T) {
	var search Search
	done := make(chan struct{}, 1)
	remote := func(context.Context, string) ([]Entry, error) {
		var out []Entry
		for i := 0; i < 451; i++ {
			out = append(out, Entry{Name: fmt.Sprintf("target-%04d", i)})
		}
		return out, nil
	}
	search.Ensure("host/session", "missing", "target", remote, func() { done <- struct{}{} })
	<-done
	snap := search.Snapshot()
	if len(snap.Paths) != 200 || snap.Total != 451 {
		t.Fatalf("first page: %+v", snap)
	}
	search.More()
	search.More()
	snap = search.Snapshot()
	if len(snap.Paths) != 451 || snap.Paths[450] != "target-0450" {
		t.Fatalf("last page: %d", len(snap.Paths))
	}
}
