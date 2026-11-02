package main

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestRuntimeLinksDetachRetryAfterPhysicalSubtreesRemoved(t *testing.T) {
	root, g := fixtureGraph(t)
	g.Links["package/entry.js"] = linkSpec{Target: "package/index.js", Kind: "file"}
	saveGraph(t, root, g)
	if err := operate("restore", root); err != nil {
		t.Fatal(err)
	}
	if err := operate("detach", root); err != nil {
		t.Fatal(err)
	}
	// All aliases were already detached by the successful first pre-delete phase.
	// Manifest-owned cleanup may then remove targets and alias parent subtrees.
	for _, relative := range []string{"package", "peer"} {
		if err := os.RemoveAll(filepath.Join(root, relative)); err != nil {
			t.Fatal(err)
		}
	}
	keep := filepath.Join(root, "user-kept.txt")
	if err := os.WriteFile(keep, []byte("unowned bytes"), 0600); err != nil {
		t.Fatal(err)
	}
	if err := operate("detach", root); err != nil {
		t.Fatalf("partial cleanup retry blocked: %v", err)
	}
	if bytes, err := os.ReadFile(keep); err != nil || string(bytes) != "unowned bytes" {
		t.Fatal("retry changed unowned bytes")
	}
	for _, command := range []string{"verify", "restore", "check"} {
		if err := operate(command, root); err == nil {
			t.Fatalf("%s accepted incomplete graph", command)
		}
	}
}
func TestRuntimeLinksDetachRetryRejectsReplacedAncestor(t *testing.T) {
	root, _ := fixtureGraph(t)
	if err := operate("restore", root); err != nil {
		t.Fatal(err)
	}
	if err := operate("detach", root); err != nil {
		t.Fatal(err)
	}
	if err := os.RemoveAll(filepath.Join(root, "package")); err != nil {
		t.Fatal(err)
	}
	external := t.TempDir()
	sentinel := filepath.Join(external, "keep.txt")
	if err := os.WriteFile(sentinel, []byte("safe"), 0600); err != nil {
		t.Fatal(err)
	}
	if err := createDirectoryLink(external, filepath.Join(root, "package")); err != nil {
		t.Fatal(err)
	}
	if err := operate("detach", root); err == nil || !strings.Contains(err.Error(), "ancestor") {
		t.Fatalf("planted ancestor not rejected: %v", err)
	}
	if bytes, err := os.ReadFile(sentinel); err != nil || string(bytes) != "safe" {
		t.Fatal("external bytes changed")
	}
	if err := removeDirectoryLink(filepath.Join(root, "package")); err != nil {
		t.Fatal(err)
	}
}
func TestRuntimeLinksDetachExistingAliasRequiresItsTarget(t *testing.T) {
	root, _ := fixtureGraph(t)
	if err := operate("restore", root); err != nil {
		t.Fatal(err)
	}
	// Keep one alias present while its exact physical target disappears.
	if err := removeDirectoryLink(filepath.Join(root, "peer/node_modules/package")); err != nil {
		t.Fatal(err)
	}
	if err := os.RemoveAll(filepath.Join(root, "peer")); err != nil {
		t.Fatal(err)
	}
	if err := operate("detach", root); err == nil {
		t.Fatal("present alias with absent target accepted")
	}
	if _, err := os.Lstat(filepath.Join(root, "package/node_modules/peer")); err != nil {
		t.Fatal("rejected retry mutated existing alias")
	}
}
