package main

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func fixtureGraph(t *testing.T) (string, graph) {
	t.Helper()
	root := filepath.Join(t.TempDir(), "vendor")
	if err := os.MkdirAll(filepath.Join(root, "package", "node_modules"), 0700); err != nil {
		t.Fatal(err)
	}
	bytes := []byte("fixture immutable code")
	if err := os.WriteFile(filepath.Join(root, "package", "index.js"), bytes, 0600); err != nil {
		t.Fatal(err)
	}
	sum := sha256.Sum256(bytes)
	g := graph{SchemaVersion: 1, Files: map[string]string{"package/index.js": hex.EncodeToString(sum[:])}, Links: map[string]linkSpec{"package/node_modules/self": {Target: "package", Kind: "directory"}}}
	// A dependency may refer to the containing package, as pnpm/workspace
	// dependency cycles do. The alias itself must never target an ancestor of
	// the alias path: use two sibling packages to model a valid cyclic graph.
	if err := os.MkdirAll(filepath.Join(root, "peer", "node_modules"), 0700); err != nil {
		t.Fatal(err)
	}
	g.Links = map[string]linkSpec{"package/node_modules/peer": {Target: "peer", Kind: "directory"}, "peer/node_modules/package": {Target: "package", Kind: "directory"}}
	saveGraph(t, root, g)
	return root, g
}
func saveGraph(t *testing.T, root string, g graph) {
	t.Helper()
	raw, err := json.Marshal(g)
	if err != nil {
		t.Fatal(err)
	}
	if err = os.WriteFile(filepath.Join(root, "runtime-graph.json"), raw, 0600); err != nil {
		t.Fatal(err)
	}
}
func TestRuntimeLinksRestoreDetachRenameRollback(t *testing.T) {
	root, _ := fixtureGraph(t)
	for _, command := range []string{"check", "restore", "restore", "verify", "detach", "detach"} {
		if err := operate(command, root); err != nil {
			t.Fatalf("%s: %v", command, err)
		}
	}
	final := root + "-final"
	if err := os.Rename(root, final); err != nil {
		t.Fatal(err)
	}
	if err := operate("restore", final); err != nil {
		t.Fatal(err)
	}
	real, err := resolveDirectoryLink(filepath.Join(final, "package/node_modules/peer"))
	if err != nil || !strings.EqualFold(real, filepath.Join(final, "peer")) {
		t.Fatalf("bad rebased target %s: %v", real, err)
	}
	if err = operate("detach", final); err != nil {
		t.Fatal(err)
	}
	if err = os.Rename(final, root); err != nil {
		t.Fatal(err)
	}
	if err = operate("restore", root); err != nil {
		t.Fatal(err)
	}
	if err = operate("verify", root); err != nil {
		t.Fatal(err)
	}
}
func TestRuntimeLinksUnknownOrChangedTargetNeverMutates(t *testing.T) {
	root, g := fixtureGraph(t)
	if err := operate("restore", root); err != nil {
		t.Fatal(err)
	}
	unknown := filepath.Join(root, "unknown")
	if err := createDirectoryLink(filepath.Join(root, "peer"), unknown); err != nil {
		t.Fatal(err)
	}
	if err := operate("detach", root); err == nil {
		t.Fatal("unknown junction accepted")
	}
	if _, err := os.Lstat(filepath.Join(root, "package/node_modules/peer")); err != nil {
		t.Fatal("preflight removed a good link")
	}
	if err := removeDirectoryLink(unknown); err != nil {
		t.Fatal(err)
	}
	changed := filepath.Join(root, "package/node_modules/peer")
	if err := removeDirectoryLink(changed); err != nil {
		t.Fatal(err)
	}
	external := t.TempDir()
	sentinel := filepath.Join(external, "keep.txt")
	if err := os.WriteFile(sentinel, []byte("keep"), 0600); err != nil {
		t.Fatal(err)
	}
	if err := createDirectoryLink(external, changed); err != nil {
		t.Fatal(err)
	}
	if err := operate("detach", root); err == nil {
		t.Fatal("changed external target accepted")
	}
	if bytes, err := os.ReadFile(sentinel); err != nil || string(bytes) != "keep" {
		t.Fatal("external target changed")
	}
	if err := removeDirectoryLink(changed); err != nil {
		t.Fatal(err)
	}
	g.Links["package/node_modules/peer"] = linkSpec{Target: "../escape", Kind: "directory"}
	saveGraph(t, root, g)
	if err := operate("restore", root); err == nil {
		t.Fatal("unsafe target accepted")
	}
}
func TestRuntimeLinksChangedFileAndStaleJunction(t *testing.T) {
	root, _ := fixtureGraph(t)
	if err := os.WriteFile(filepath.Join(root, "package/index.js"), []byte("tamper"), 0600); err != nil {
		t.Fatal(err)
	}
	if err := operate("restore", root); err == nil {
		t.Fatal("changed physical file accepted")
	}
	root, _ = fixtureGraph(t)
	if err := operate("restore", root); err != nil {
		t.Fatal(err)
	}
	moved := root + "-stale"
	if err := os.Rename(root, moved); err != nil {
		t.Fatal(err)
	}
	if err := operate("detach", moved); err == nil {
		t.Fatal("stale absolute junction accepted")
	}
}
