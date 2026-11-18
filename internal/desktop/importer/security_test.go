package importer

import (
	"bytes"
	"context"
	"errors"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"testing"

	"whaleisle/internal/harness"
)

func TestCompressedMetadataScriptRejectsDecompressionBomb(t *testing.T) {
	node, err := harness.ResolveComponentNode("")
	if err != nil {
		t.Skipf("node runtime unavailable: %v", err)
	}
	tmp := t.TempDir()
	bomb := filepath.Join(tmp, "bomb.zst")
	valid := filepath.Join(tmp, "valid.zst")

	// Craft a highly compressible >3 MiB payload that fits in a tiny zstd file (< 1 KiB)
	// and must trip MAX_FRAME_OUT (2 MiB) inside compressedMetadataScript.
	gen := exec.Command(node, "-e", `
		const fs = require('node:fs'), zlib = require('node:zlib');
		const pad = ' '.repeat(3 * 1024 * 1024);
		fs.writeFileSync(process.argv[1], zlib.zstdCompressSync(Buffer.from(pad)));
		const row = JSON.stringify({type:'session', cwd:'/work', title:'Safe Title'}) + '\n';
		fs.writeFileSync(process.argv[2], zlib.zstdCompressSync(Buffer.from(row)));
	`, bomb, valid)
	if out, err := gen.CombinedOutput(); err != nil {
		t.Fatalf("generate zstd fixtures: %v\n%s", err, out)
	}

	if _, err := runCompressedMetadataScript(context.Background(), node, bomb); err == nil {
		t.Fatal("expected compressedMetadataScript to reject >2MiB decompressed frame")
	}
	title, cwd := readCompressedSessionMetaWithNode(context.Background(), node, bomb)
	if title != "" || cwd != "" {
		t.Fatalf("expected fail-closed empty metadata on bomb, got title=%q cwd=%q", title, cwd)
	}

	title, cwd = readCompressedSessionMetaWithNode(context.Background(), node, valid)
	if title != "Safe Title" || cwd != "/work" {
		t.Fatalf("expected valid metadata extraction, got title=%q cwd=%q", title, cwd)
	}
}

func TestImportRejectsSymlinkAndWindowsJunction(t *testing.T) {
	tmp := t.TempDir()
	outside := filepath.Join(tmp, "outside")
	if err := os.MkdirAll(filepath.Join(outside, "sessions", "s1"), 0o700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(outside, "sessions", "s1", "log.jsonl"), []byte(`{"type":"session","title":"secret"}`+"\n"), 0o600); err != nil {
		t.Fatal(err)
	}

	srcJunction := filepath.Join(tmp, "src-junction")
	createdLink := false
	if runtime.GOOS == "windows" {
		cmd := exec.Command("cmd", "/c", "mklink", "/J", srcJunction, outside)
		if out, err := cmd.CombinedOutput(); err == nil {
			createdLink = true
		} else {
			t.Logf("mklink /J unavailable: %v (%s)", err, out)
		}
	}
	if !createdLink {
		if err := os.Symlink(outside, srcJunction); err == nil {
			createdLink = true
		} else {
			t.Skipf("symlinks and junctions unavailable: %v", err)
		}
	}

	dst := filepath.Join(tmp, "dst")
	if err := os.MkdirAll(dst, 0o700); err != nil {
		t.Fatal(err)
	}

	// BuildCatalog must reject a symlinked/junctioned source home directly.
	if _, err := BuildCatalog(srcJunction, dst); err == nil {
		t.Fatal("expected BuildCatalog to reject symlink/junction source root")
	}

	// Also place a junction inside an otherwise ordinary source tree and verify
	// copyTree, treeFingerprint, and copyDir all reject it instead of traversing.
	srcNormal := filepath.Join(tmp, "src-normal")
	if err := os.MkdirAll(srcNormal, 0o700); err != nil {
		t.Fatal(err)
	}
	nestedLink := filepath.Join(srcNormal, "linked")
	if runtime.GOOS == "windows" {
		if out, err := exec.Command("cmd", "/c", "mklink", "/J", nestedLink, outside).CombinedOutput(); err != nil {
			t.Fatalf("mklink /J nested: %v (%s)", err, out)
		}
	} else if err := os.Symlink(outside, nestedLink); err != nil {
		t.Fatal(err)
	}

	if _, err := treeFingerprint(context.Background(), srcNormal); err == nil {
		t.Fatal("expected treeFingerprint to reject symlink/junction directory")
	}
	if err := copyTree(context.Background(), srcNormal, filepath.Join(tmp, "copy1"), true); err == nil {
		t.Fatal("expected copyTree to reject symlink/junction directory")
	}
	if err := copyDir(srcNormal, filepath.Join(tmp, "copy2"), true); err == nil {
		t.Fatal("expected copyDir to reject symlink/junction directory")
	}
}

func TestDirectoryTransactionCommitRollbackAndCleanupBoundaries(t *testing.T) {
	home := t.TempDir()
	src := filepath.Join(t.TempDir(), "session-src")
	if err := os.MkdirAll(src, 0o700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(src, "log.jsonl"), []byte("new-session-data\n"), 0o600); err != nil {
		t.Fatal(err)
	}

	dst := filepath.Join(home, "sessions", "s1")
	if err := os.MkdirAll(dst, 0o700); err != nil {
		t.Fatal(err)
	}
	origBytes := []byte("original-session-data\n")
	if err := os.WriteFile(filepath.Join(dst, "log.jsonl"), origBytes, 0o600); err != nil {
		t.Fatal(err)
	}

	// Normal atomic commit cleans up .mygo-import control directory.
	if err := publishDirectory(context.Background(), home, src, dst, true, false); err != nil {
		t.Fatalf("publishDirectory failed: %v", err)
	}
	if _, err := os.Stat(filepath.Join(home, transactionDirectory)); !os.IsNotExist(err) {
		t.Fatalf("expected %s cleaned after commit, stat err=%v", transactionDirectory, err)
	}
	got, err := os.ReadFile(filepath.Join(dst, "log.jsonl"))
	if err != nil || !bytes.Equal(got, []byte("new-session-data\n")) {
		t.Fatalf("unexpected committed content %q err=%v", got, err)
	}

	// Rollback recovery: simulate crash during swap after dst -> old backup move.
	control := filepath.Join(home, transactionDirectory)
	if err := os.MkdirAll(filepath.Join(control, "new"), 0o700); err != nil {
		t.Fatal(err)
	}
	beforeFP, err := treeFingerprint(context.Background(), dst)
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(control, "new", "log.jsonl"), []byte("staged-only\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	afterFP, err := treeFingerprint(context.Background(), filepath.Join(control, "new"))
	if err != nil {
		t.Fatal(err)
	}
	if err := os.Rename(dst, filepath.Join(control, "old")); err != nil {
		t.Fatal(err)
	}
	if err := saveDirectoryTransaction(control, directoryTransaction{
		Version: 1,
		Target:  "sessions/s1",
		Phase:   "ready",
		Before:  beforeFP,
		After:   afterFP,
	}); err != nil {
		t.Fatal(err)
	}
	recovered, err := RecoverInterrupted(home)
	if err != nil || !recovered {
		t.Fatalf("expected RecoverInterrupted rollback success, got recovered=%v err=%v", recovered, err)
	}
	if _, err := os.Stat(control); !os.IsNotExist(err) {
		t.Fatalf("expected control dir removed after rollback, got err=%v", err)
	}
	restored, err := os.ReadFile(filepath.Join(dst, "log.jsonl"))
	if err != nil || !bytes.Equal(restored, []byte("new-session-data\n")) {
		t.Fatalf("expected original tree restored on rollback, got %q err=%v", restored, err)
	}

	// Unrecognized artifact inside .mygo-import must block cleanup rather than blind RemoveAll.
	foreign := filepath.Join(control, "foreign-user-file.txt")
	if err := os.MkdirAll(control, 0o700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(foreign, []byte("keep-me"), 0o600); err != nil {
		t.Fatal(err)
	}
	_, recErr := RecoverInterrupted(home)
	if !errors.Is(recErr, ErrRecoveryBlocked) || !strings.Contains(recErr.Error(), "unrecognized recovery artifact") {
		t.Fatalf("expected ErrRecoveryBlocked on unknown artifact, got %v", recErr)
	}
	if _, err := os.Stat(foreign); err != nil {
		t.Fatalf("expected unrecognized file preserved, got %v", err)
	}
}
