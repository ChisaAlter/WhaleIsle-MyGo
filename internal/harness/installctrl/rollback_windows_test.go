//go:build windows

package installctrl

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestRollbackWithWindowsJunctionsAndDanglingLinks(t *testing.T) {
	sourceRoot := t.TempDir()
	pkgADir := filepath.Join(sourceRoot, "pkg-a")
	scopedDir := filepath.Join(sourceRoot, "scoped-pkg")
	danglingTarget := filepath.Join(sourceRoot, "dangling-target")
	untouchedOther := filepath.Join(sourceRoot, "other-pkg")
	for _, dir := range []string{pkgADir, scopedDir, danglingTarget, untouchedOther} {
		if err := os.MkdirAll(dir, 0o755); err != nil {
			t.Fatal(err)
		}
	}
	pkgASrcFile := filepath.Join(pkgADir, "index.js")
	scopedSrcFile := filepath.Join(scopedDir, "index.js")
	otherSrcFile := filepath.Join(untouchedOther, "keep.txt")
	if err := os.WriteFile(pkgASrcFile, []byte("pkg-a-source"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(scopedSrcFile, []byte("scoped-source"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(otherSrcFile, []byte("keep-other"), 0o644); err != nil {
		t.Fatal(err)
	}

	profileDir := t.TempDir()
	pkgJSONPath := filepath.Join(profileDir, "package.json")
	if err := os.WriteFile(pkgJSONPath, []byte(`{"name":"orig"}`), 0o644); err != nil {
		t.Fatal(err)
	}
	pkgALink := filepath.Join(profileDir, "node_modules", "pkg-a")
	scopedLink := filepath.Join(profileDir, "node_modules", "@scope", "pkg-b")
	danglingLink := filepath.Join(profileDir, "node_modules", "dangling-pkg")
	for link, target := range map[string]string{
		pkgALink:     pkgADir,
		scopedLink:   scopedDir,
		danglingLink: danglingTarget,
	} {
		if err := createWindowsJunction(link, target); err != nil {
			t.Fatalf("createWindowsJunction(%s -> %s): %v", link, target, err)
		}
	}
	// Make danglingLink genuinely dangling before capturing snapshot.
	if err := os.RemoveAll(danglingTarget); err != nil {
		t.Fatal(err)
	}

	snap, err := CaptureInstallSnapshot(profileDir)
	if err != nil {
		t.Fatalf("CaptureInstallSnapshot: %v", err)
	}

	for rel, wantTarget := range map[string]string{
		filepath.Join("node_modules", "pkg-a"):        pkgADir,
		filepath.Join("node_modules", "@scope", "pkg-b"): scopedDir,
		filepath.Join("node_modules", "dangling-pkg"): danglingTarget,
	} {
		snapPath := filepath.Join(snap.Dir, rel)
		st, err := os.Lstat(snapPath)
		if err != nil {
			t.Fatalf("Lstat(%s): %v", snapPath, err)
		}
		if st.Mode()&os.ModeIrregular == 0 || st.Mode()&os.ModeSymlink != 0 {
			t.Fatalf("%s mode = %v, expected Windows junction (ModeIrregular without ModeSymlink)", snapPath, st.Mode())
		}
		gotTarget, err := os.Readlink(snapPath)
		if err != nil {
			t.Fatalf("Readlink(%s): %v", snapPath, err)
		}
		if !strings.EqualFold(filepath.Clean(gotTarget), filepath.Clean(wantTarget)) {
			t.Fatalf("Readlink(%s) = %q, want %q", snapPath, gotTarget, wantTarget)
		}
	}

	// Simulate failed install mutating package.json, removing a scoped junction,
	// and planting both a new directory and a new junction pointing outside.
	if err := os.WriteFile(pkgJSONPath, []byte(`{"name":"corrupt"}`), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.Remove(scopedLink); err != nil {
		t.Fatal(err)
	}
	corruptDir := filepath.Join(profileDir, "node_modules", "corrupt-pkg")
	if err := os.MkdirAll(corruptDir, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(corruptDir, "bad.js"), []byte("bad"), 0o644); err != nil {
		t.Fatal(err)
	}
	badJunction := filepath.Join(profileDir, "node_modules", "bad-junction")
	if err := createWindowsJunction(badJunction, untouchedOther); err != nil {
		t.Fatal(err)
	}

	res := RollbackInstall(snap, Result{Spec: "github:acme/bad", Error: "install failed"})
	if res.OK || !res.RolledBack || res.RollbackError != "" {
		t.Fatalf("RollbackInstall = %+v", res)
	}

	gotPkg, err := os.ReadFile(pkgJSONPath)
	if err != nil || string(gotPkg) != `{"name":"orig"}` {
		t.Fatalf("restored package.json = %q, %v", gotPkg, err)
	}
	if got, err := os.ReadFile(filepath.Join(pkgALink, "index.js")); err != nil || string(got) != "pkg-a-source" {
		t.Fatalf("restored pkg-a = %q, %v", got, err)
	}
	if got, err := os.ReadFile(filepath.Join(scopedLink, "index.js")); err != nil || string(got) != "scoped-source" {
		t.Fatalf("restored scoped pkg-b = %q, %v", got, err)
	}
	if gotTarget, err := os.Readlink(danglingLink); err != nil || !strings.EqualFold(filepath.Clean(gotTarget), filepath.Clean(danglingTarget)) {
		t.Fatalf("restored dangling junction target = %q, %v", gotTarget, err)
	}
	if _, err := os.Lstat(corruptDir); !os.IsNotExist(err) {
		t.Fatalf("corrupt-pkg still exists: %v", err)
	}
	if _, err := os.Lstat(badJunction); !os.IsNotExist(err) {
		t.Fatalf("bad-junction still exists: %v", err)
	}

	// Verify source target directories were never traversed or modified.
	for path, want := range map[string]string{
		pkgASrcFile:   "pkg-a-source",
		scopedSrcFile: "scoped-source",
		otherSrcFile:  "keep-other",
	} {
		got, err := os.ReadFile(path)
		if err != nil || string(got) != want {
			t.Fatalf("source target %s = %q (%v), want %q", path, got, err, want)
		}
	}
}

func TestRollbackAndDiscardTopLevelNodeModulesJunction(t *testing.T) {
	sourceRoot := t.TempDir()
	topNMTarget := filepath.Join(sourceRoot, "shared-node-modules")
	if err := os.MkdirAll(topNMTarget, 0o755); err != nil {
		t.Fatal(err)
	}
	sentinel := filepath.Join(topNMTarget, "sentinel.txt")
	if err := os.WriteFile(sentinel, []byte("keep-top-nm"), 0o644); err != nil {
		t.Fatal(err)
	}

	profileDir := t.TempDir()
	topNMLink := filepath.Join(profileDir, "node_modules")
	if err := createWindowsJunction(topNMLink, topNMTarget); err != nil {
		t.Fatal(err)
	}

	snap1, err := CaptureInstallSnapshot(profileDir)
	if err != nil {
		t.Fatalf("CaptureInstallSnapshot: %v", err)
	}
	// DiscardInstallSnapshot must remove snapshot link without deleting target contents.
	DiscardInstallSnapshot(snap1, Result{OK: true})
	if got, err := os.ReadFile(sentinel); err != nil || string(got) != "keep-top-nm" {
		t.Fatalf("after DiscardInstallSnapshot, sentinel = %q, %v", got, err)
	}

	// Now test full rollback when top-level node_modules junction is replaced by a directory.
	snap2, err := CaptureInstallSnapshot(profileDir)
	if err != nil {
		t.Fatalf("CaptureInstallSnapshot 2: %v", err)
	}
	if err := os.Remove(topNMLink); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(filepath.Join(topNMLink, "extra"), 0o755); err != nil {
		t.Fatal(err)
	}
	res := RollbackInstall(snap2, Result{Spec: "github:acme/top", Error: "failed"})
	if !res.RolledBack || res.RollbackError != "" {
		t.Fatalf("RollbackInstall = %+v", res)
	}
	gotTarget, err := os.Readlink(topNMLink)
	if err != nil || !strings.EqualFold(filepath.Clean(gotTarget), filepath.Clean(topNMTarget)) {
		t.Fatalf("restored top-level node_modules junction target = %q, %v", gotTarget, err)
	}
	if got, err := os.ReadFile(sentinel); err != nil || string(got) != "keep-top-nm" {
		t.Fatalf("after RollbackInstall, sentinel = %q, %v", got, err)
	}
}
