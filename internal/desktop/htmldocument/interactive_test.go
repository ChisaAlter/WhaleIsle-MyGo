package htmldocument

import (
	"context"
	"encoding/base64"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestInteractiveResourceModeIsSeparateFromStatic(t *testing.T) {
	dir := t.TempDir()
	script := "document.body.dataset.loaded='yes';"
	if err := os.WriteFile(filepath.Join(dir, "page.js"), []byte(script), 0600); err != nil {
		t.Fatal(err)
	}
	root, err := os.OpenRoot(dir)
	if err != nil {
		t.Fatal(err)
	}
	defer root.Close()
	source := "<button onclick=\"this.textContent='yes'\">Run</button><script>window.inline=true</script><script defer src=\"page.js\"></script>"
	valid := func() bool { return true }
	static, _, err := prepareResources(root, "page.html", source, valid)
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(static, "<script") || strings.Contains(static, "onclick") {
		t.Fatal("static mode retained scripts")
	}
	dynamic, _, err := prepareResourceMode(root, "page.html", source, valid, true)
	if err != nil {
		t.Fatal(err)
	}
	for _, want := range []string{"onclick=", "window.inline=true", "data:text/javascript;base64," + base64.StdEncoding.EncodeToString([]byte(script)), "defer"} {
		if !strings.Contains(dynamic, want) {
			t.Fatalf("interactive snapshot missing %q", want)
		}
	}
}

func TestScriptGraphRejectsRemoteAndWorkspaceEscape(t *testing.T) {
	dir := t.TempDir()
	root, err := os.OpenRoot(dir)
	if err != nil {
		t.Fatal(err)
	}
	defer root.Close()
	g, err := newResourceGraph(root, 0, func() bool { return true })
	if err != nil {
		t.Fatal(err)
	}
	for _, reference := range []string{"https://example.invalid/a.js", "../outside.js", ".git/private.js", "file:///private.js"} {
		if _, err := g.script("index.html", reference); err == nil {
			t.Fatalf("accepted %q", reference)
		}
	}
	if resourceKindAllows("asset", ".js") || resourceKindMIME("asset", "text/javascript") {
		t.Fatal("static asset loader gained script access")
	}
}

func TestInteractiveSnapshotBindsOriginalFileAndLease(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "index.html")
	source := "<script>window.original=true</script>"
	if err := os.WriteFile(path, []byte(source), 0600); err != nil {
		t.Fatal(err)
	}
	rootInfo, err := os.Stat(dir)
	if err != nil {
		t.Fatal(err)
	}
	fileInfo, err := os.Stat(path)
	if err != nil {
		t.Fatal(err)
	}
	d := &Document{source: source, root: dir, relative: "index.html", rootIdentity: rootInfo, identity: fileInfo, valid: func() bool { return true }, done: make(chan struct{})}
	got, _, err := d.InteractiveSnapshot(context.Background())
	if err != nil || !strings.Contains(got, "window.original") {
		t.Fatalf("snapshot: %q %v", got, err)
	}
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	if _, _, err := d.InteractiveSnapshot(ctx); err == nil {
		t.Fatal("accepted cancelled gesture")
	}
	if err := os.WriteFile(path, []byte(source+"changed"), 0600); err != nil {
		t.Fatal(err)
	}
	if _, _, err := d.InteractiveSnapshot(context.Background()); err == nil {
		t.Fatal("accepted changed source")
	}
	d.Close()
	select {
	case <-d.Done():
	default:
		t.Fatal("lease was not revoked")
	}
	if d.source != "" {
		t.Fatal("closed snapshot retained source bytes")
	}
	if _, _, err := d.InteractiveSnapshot(context.Background()); err == nil {
		t.Fatal("accepted closed snapshot")
	}
}

func TestInteractiveSnapshotNeverAddsResourceRoutes(t *testing.T) {
	dir := t.TempDir()
	root, err := os.OpenRoot(dir)
	if err != nil {
		t.Fatal(err)
	}
	defer root.Close()
	source := "<script src='https://example.invalid/a.js'></script><iframe src='private.html'></iframe>"
	got, report, err := prepareResourceMode(root, "index.html", source, func() bool { return true }, true)
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(got, "example.invalid") || strings.Contains(got, "private.html") || report.Skipped == 0 {
		t.Fatalf("unexpected external dependencies: %q %#v", got, report)
	}
}
