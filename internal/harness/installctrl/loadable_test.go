package installctrl

import (
	"os"
	"path/filepath"
	"testing"
)

func TestHasLoadableEntryAndIdentity(t *testing.T) {
	profile := t.TempDir()
	plantLoadablePackage(t, profile, "widget")
	if !HasLoadableEntry(profile, "widget") {
		t.Fatal("expected loadable")
	}
	if !HasInventoryIdentity(profile, "widget") {
		t.Fatal("expected identity")
	}
	if HasLoadableEntry(profile, "missing") {
		t.Fatal("missing should not be loadable")
	}
}

func TestResolveAddedPackageNames(t *testing.T) {
	before := []string{"a"}
	after := []string{"a", "widget"}
	names := ResolveAddedPackageNames("github:acme/widget", before, after)
	if len(names) != 1 || names[0] != "widget" {
		t.Fatalf("names = %v", names)
	}
}

func TestListNodeModuleNames_scoped(t *testing.T) {
	profile := t.TempDir()
	dir := filepath.Join(profile, "node_modules", "@acme", "pkg")
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatal(err)
	}
	names := ListNodeModuleNames(profile)
	if len(names) != 1 || names[0] != "@acme/pkg" {
		t.Fatalf("names = %v", names)
	}
}
