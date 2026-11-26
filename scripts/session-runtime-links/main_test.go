package main

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"os"
	"path/filepath"
	"runtime"
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
	wantPeer, wantErr := resolveDirectoryLink(filepath.Join(final, "peer"))
	if wantErr != nil {
		t.Fatal(wantErr)
	}
	if err != nil || !strings.EqualFold(real, wantPeer) {
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
	if err := operate("verify", root); err == nil {
		t.Fatal("changed physical file accepted by verify")
	}
	// Restore replays links for a tree the installer just wrote; it checks
	// structure, not every payload digest, so the content gate stays verify's.
	if err := operate("restore", root); err != nil {
		t.Fatalf("restore re-read payload content: %v", err)
	}
	if err := operate("detach", root); err != nil {
		t.Fatal(err)
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

func TestRuntimeLinksRestoreRejectsMissingPhysicalTarget(t *testing.T) {
	root, _ := fixtureGraph(t)
	// One physical target is gone before restore ever ran. The retry
	// tolerance for already-removed subtrees is detach's alone: restore must
	// refuse at preflight instead of relinking the surviving half.
	if err := os.RemoveAll(filepath.Join(root, "peer")); err != nil {
		t.Fatal(err)
	}
	if err := operate("restore", root); err == nil {
		t.Fatal("restore accepted a missing physical target")
	}
	for _, name := range []string{"package/node_modules/peer", "peer/node_modules/package"} {
		if _, err := os.Lstat(filepath.Join(root, name)); !os.IsNotExist(err) {
			t.Fatalf("restore left a partial link: %s", name)
		}
	}
	// check refuses the same incomplete tree; only detach tolerates it.
	if err := operate("check", root); err == nil {
		t.Fatal("check accepted a missing physical target")
	}
	if err := operate("detach", root); err != nil {
		t.Fatalf("detach did not tolerate the partial tree: %v", err)
	}
}

func TestRuntimeLinksRejectsMissingOriginAncestor(t *testing.T) {
	root, _ := fixtureGraph(t)
	// Keep both package and peer targets intact, but delete one origin
	// ancestor directory. check, restore and verify must all refuse the
	// half-deleted tree and leave no partial link behind; only detach
	// tolerates the retry cleanup.
	if err := os.RemoveAll(filepath.Join(root, "package", "node_modules")); err != nil {
		t.Fatal(err)
	}
	if err := operate("check", root); err == nil {
		t.Fatal("check accepted a missing origin ancestor")
	}
	if err := operate("restore", root); err == nil {
		t.Fatal("restore accepted a missing origin ancestor")
	}
	if _, err := os.Lstat(filepath.Join(root, "peer", "node_modules", "package")); !os.IsNotExist(err) {
		t.Fatalf("restore left a partial link: peer/node_modules/package (err=%v)", err)
	}
	if err := operate("verify", root); err == nil {
		t.Fatal("verify accepted a missing origin ancestor")
	}
	if err := operate("detach", root); err != nil {
		t.Fatalf("detach did not tolerate missing origin ancestor: %v", err)
	}
}

func TestRuntimeLinksTraceRecordsEntryAndOutcome(t *testing.T) {
	root, _ := fixtureGraph(t)
	trace := filepath.Join(t.TempDir(), "runtime-links.log")
	if code := run([]string{"runtime-links.exe", "restore", root, trace}); code != 0 {
		t.Fatalf("restore exit %d", code)
	}
	recorded, err := os.ReadFile(trace)
	if err != nil {
		t.Fatal(err)
	}
	for _, expected := range []string{"restore entry " + root, "restore done " + root} {
		if !strings.Contains(string(recorded), expected) {
			t.Fatalf("trace lacks %q: %s", expected, recorded)
		}
	}
	if err := os.WriteFile(filepath.Join(root, "package/index.js"), []byte("tamper"), 0600); err != nil {
		t.Fatal(err)
	}
	if code := run([]string{"runtime-links.exe", "verify", root, trace}); code != 1 {
		t.Fatalf("verify exit %d, want 1", code)
	}
	recorded, err = os.ReadFile(trace)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(recorded), "verify failed "+root+": runtime file changed: package/index.js") {
		t.Fatalf("trace lacks the refusal: %s", recorded)
	}
}

func TestRuntimeLinksRejectsUnexpectedArgumentCount(t *testing.T) {
	for _, args := range [][]string{{"runtime-links.exe"}, {"runtime-links.exe", "restore"}, {"runtime-links.exe", "restore", "root", "trace", "extra"}} {
		if code := run(args); code != 2 {
			t.Fatalf("%v exit %d, want 2", args, code)
		}
	}
}

func TestRuntimeLinksLongPathWindows(t *testing.T) {
	if runtime.GOOS != "windows" {
		t.Skip("Windows MAX_PATH (>260 chars) Win32 extended-length path test")
	}
	base := t.TempDir()
	// Build a deep nested vendor root so both the physical target and junction
	// source exceed Windows MAX_PATH (260 chars) even after promotion.
	root := filepath.Join(base, strings.Repeat("segment-a-", 6), strings.Repeat("segment-b-", 6), "vendor")
	targetRel := filepath.Join("deepseek-harness", "node_modules", ".dsh-instances", strings.Repeat("pkg-target-", 6))
	linkParentRel := filepath.Join("deepseek-harness", "node_modules", ".dsh-instances", strings.Repeat("pkg-origin-", 6), "node_modules")
	linkRel := filepath.Join(linkParentRel, "dep-link")

	targetAbs := filepath.Join(root, targetRel)
	linkAbs := filepath.Join(root, linkRel)
	if len(targetAbs) <= 270 || len(linkAbs) <= 270 {
		t.Fatalf("test fixture paths must exceed 270 chars: target=%d link=%d", len(targetAbs), len(linkAbs))
	}
	if err := os.MkdirAll(targetAbs, 0o700); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(filepath.Join(root, linkParentRel), 0o700); err != nil {
		t.Fatal(err)
	}
	payload := []byte("long-path immutable module\n")
	payloadRel := filepath.ToSlash(filepath.Join(targetRel, "index.js"))
	if err := os.WriteFile(filepath.Join(targetAbs, "index.js"), payload, 0o600); err != nil {
		t.Fatal(err)
	}
	sum := sha256.Sum256(payload)
	g := graph{
		SchemaVersion: 1,
		Files:         map[string]string{payloadRel: hex.EncodeToString(sum[:])},
		Links: map[string]linkSpec{
			filepath.ToSlash(linkRel): {Target: filepath.ToSlash(targetRel), Kind: "directory"},
		},
	}
	saveGraph(t, root, g)

	for _, cmd := range []string{"check", "restore", "verify", "detach"} {
		if err := operate(cmd, root); err != nil {
			t.Fatalf("long-path %s failed (target=%d link=%d): %v", cmd, len(targetAbs), len(linkAbs), err)
		}
	}
}
