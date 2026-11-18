package components

import (
	"archive/tar"
	"bytes"
	"compress/gzip"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
	"time"
)

func TestWriteJSONAtomicPreservesExistingFileOnRenameFailure(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "installed.json")
	orig := WhaleBridgeInstalled{Version: "1.0.0"}
	if err := writeJSONAtomic(path, orig); err != nil {
		t.Fatalf("initial writeJSONAtomic: %v", err)
	}

	// Destination turned into a non-empty directory so os.Rename cannot replace it.
	dirTarget := filepath.Join(dir, "locked-dir")
	if err := os.MkdirAll(filepath.Join(dirTarget, "child"), 0o700); err != nil {
		t.Fatal(err)
	}
	if err := writeJSONAtomic(dirTarget, orig); err == nil {
		t.Fatal("expected writeJSONAtomic onto directory to fail")
	}
	leftover, err := filepath.Glob(filepath.Join(dir, ".*.tmp"))
	if err != nil || len(leftover) != 0 {
		t.Fatalf("expected temp files cleaned up, got %v err=%v", leftover, err)
	}
}

func TestAcquireOperationLockDeadAndLivePIDs(t *testing.T) {
	root := t.TempDir()
	svc := NewWhaleBridgeService(ServiceOptions{Root: root, DshHome: filepath.Join(t.TempDir(), "dsh")})

	lockPath := filepath.Join(root, "operation.lock")
	deadLock := operationLockPayload{
		PID:       99999999,
		Token:     "dead-token",
		CreatedAt: time.Now().Add(-time.Minute).UTC().Format(time.RFC3339Nano),
	}
	rawDead, _ := json.Marshal(deadLock)
	if err := os.WriteFile(lockPath, rawDead, 0o600); err != nil {
		t.Fatal(err)
	}

	release, err := svc.acquireOperationLock()
	if err != nil {
		t.Fatalf("expected stale dead-PID lock to be reclaimed, got %v", err)
	}
	if _, err := svc.acquireOperationLock(); err == nil || !strings.Contains(err.Error(), "busy") {
		t.Fatalf("expected live same-process lock to block second acquire, got %v", err)
	}
	release()
	if _, err := os.Stat(lockPath); !os.IsNotExist(err) {
		t.Fatalf("expected release to remove lock file, err=%v", err)
	}
}

func TestWhaleBridgeRollbackVerifiesManifestAndChecksum(t *testing.T) {
	root := t.TempDir()
	dsh := filepath.Join(t.TempDir(), "dsh")
	svc := NewWhaleBridgeService(ServiceOptions{
		Root:         root,
		DshHome:      dsh,
		StartRuntime: func(context.Context, string, string) error { return nil },
		StopRuntime:  func(context.Context, string) error { return nil },
		Disconnect:   func(context.Context, string, string) error { return nil },
	})

	prevPayload := []byte("valid-whalebridge-1.0.0")
	prevSum := sha256.Sum256(prevPayload)
	prevDir := filepath.Join(root, "versions", "1.0.0")
	if err := os.MkdirAll(prevDir, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(prevDir, whaleBridgeAsset), prevPayload, 0o700); err != nil {
		t.Fatal(err)
	}
	manifest := WhaleBridgeManifest{
		ID:      whaleBridgeID,
		Version: "1.0.0",
		License: "MIT License",
		Platforms: map[string]PlatformSpec{
			"win32-x64": {
				Asset:  whaleBridgeAsset,
				Size:   int64(len(prevPayload)),
				SHA256: hex.EncodeToString(prevSum[:]),
			},
		},
	}
	if err := writeJSONAtomic(filepath.Join(prevDir, whaleBridgeManifest), manifest); err != nil {
		t.Fatal(err)
	}
	if err := writeJSONAtomic(filepath.Join(root, "installed.json"), WhaleBridgeInstalled{
		Version:  "1.1.0",
		Previous: "1.0.0",
	}); err != nil {
		t.Fatal(err)
	}

	// Corrupt previous binary and verify rollback refuses to switch.
	if err := os.WriteFile(filepath.Join(prevDir, whaleBridgeAsset), []byte("tampered-binary-1.0.0"), 0o700); err != nil {
		t.Fatal(err)
	}
	if _, err := svc.Rollback(context.Background(), nil); err == nil {
		t.Fatal("expected Rollback to reject corrupted previous payload")
	}
	rec, err := svc.Installed()
	if err != nil || rec == nil || rec.Version != "1.1.0" {
		t.Fatalf("expected installed version unchanged at 1.1.0, got %+v err=%v", rec, err)
	}

	// Restore valid binary and verify Rollback succeeds.
	if err := os.WriteFile(filepath.Join(prevDir, whaleBridgeAsset), prevPayload, 0o700); err != nil {
		t.Fatal(err)
	}
	if _, err := svc.Rollback(context.Background(), nil); err != nil {
		t.Fatalf("expected verified Rollback to succeed, got %v", err)
	}
	rec, err = svc.Installed()
	if err != nil || rec == nil || rec.Version != "1.0.0" {
		t.Fatalf("expected version rolled back to 1.0.0, got %+v err=%v", rec, err)
	}
}

func TestWhaleBridgeUninstallTempDirAndComponentDataRoot(t *testing.T) {
	root := t.TempDir()
	dataFile := filepath.Join(root, "data", "config.json")
	if err := os.MkdirAll(filepath.Dir(dataFile), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(dataFile, []byte(`{"keep":true}`), 0o600); err != nil {
		t.Fatal(err)
	}
	vDir := filepath.Join(root, "versions", "1.0.0")
	if err := os.MkdirAll(vDir, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(vDir, whaleBridgeAsset), []byte("bin"), 0o700); err != nil {
		t.Fatal(err)
	}
	if err := writeJSONAtomic(filepath.Join(root, "installed.json"), WhaleBridgeInstalled{Version: "1.0.0"}); err != nil {
		t.Fatal(err)
	}

	var seenDisconnectRoot string
	svc := NewWhaleBridgeService(ServiceOptions{
		Root:    root,
		DshHome: filepath.Join(t.TempDir(), "dsh"),
		Disconnect: func(_ context.Context, stagedRoot, version string) error {
			seenDisconnectRoot = stagedRoot
			if version != "1.0.0" {
				t.Fatalf("unexpected version %q", version)
			}
			return nil
		},
	})
	if _, err := svc.Uninstall(context.Background(), false, nil); err != nil {
		t.Fatalf("Uninstall failed: %v", err)
	}
	if got := disconnectComponentRoot(seenDisconnectRoot); got != root {
		t.Fatalf("disconnectComponentRoot(%q) = %q, want %q", seenDisconnectRoot, got, root)
	}
	if _, err := os.Stat(dataFile); err != nil {
		t.Fatalf("expected component data retained on removeData=false: %v", err)
	}
	leftover, err := filepath.Glob(filepath.Join(root, ".uninstall-*"))
	if err != nil || len(leftover) != 0 {
		t.Fatalf("expected .uninstall-* cleaned up, got %v err=%v", leftover, err)
	}
}

func TestSanitizeComponentEnv(t *testing.T) {
	base := []string{
		"PATH=/usr/bin",
		"DSH_DESKTOP_INSTALL_SECRET=top-secret",
		"NODE_OPTIONS=--require=/evil.js",
		"NODE_EXTRA_CA_CERTS=/bad.pem",
		"ELECTRON_RUN_AS_NODE=1",
		"MAGPIE_WEB_KEY=abc",
		"MAGPIE_WEB_RUNKEY=def",
		"WHALEISLE_CONTROL_TOKEN=tok1",
		"DSH_AUTH_TOKEN=tok2",
		"MY_CUSTOM_CONTROL_TOKEN=tok3",
		"SAFE_VAR=keep-me",
	}
	got := sanitizeComponentEnv(base, "LAUNCHER_COMPONENT_ID=whalebridge", "SAFE_VAR=overridden")
	joined := "\n" + strings.Join(got, "\n") + "\n"
	for _, forbidden := range []string{
		"DSH_DESKTOP_INSTALL_SECRET=",
		"NODE_OPTIONS=",
		"NODE_EXTRA_CA_CERTS=",
		"ELECTRON_RUN_AS_NODE=",
		"MAGPIE_WEB_KEY=",
		"MAGPIE_WEB_RUNKEY=",
		"WHALEISLE_CONTROL_TOKEN=",
		"DSH_AUTH_TOKEN=",
		"MY_CUSTOM_CONTROL_TOKEN=",
	} {
		if strings.Contains(joined, "\n"+forbidden) {
			t.Fatalf("sensitive env key %q leaked into child env: %v", forbidden, got)
		}
	}
	if !strings.Contains(joined, "\nPATH=/usr/bin\n") ||
		!strings.Contains(joined, "\nLAUNCHER_COMPONENT_ID=whalebridge\n") ||
		!strings.Contains(joined, "\nSAFE_VAR=overridden\n") ||
		strings.Contains(joined, "\nSAFE_VAR=keep-me\n") {
		t.Fatalf("unexpected sanitized env: %v", got)
	}
}

func TestBridgeHardenedHTTPClientRedirects(t *testing.T) {
	httpTarget := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte("plain"))
	}))
	defer httpTarget.Close()

	tlsDowngrade := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.Redirect(w, r, httpTarget.URL+"/asset.exe", http.StatusFound)
	}))
	defer tlsDowngrade.Close()

	client := bridgeHardenedHTTPClient(tlsDowngrade.Client(), 5*time.Second)
	if _, err := client.Get(tlsDowngrade.URL + "/asset.exe"); err == nil || !strings.Contains(err.Error(), "downgrade") {
		t.Fatalf("expected redirect downgrade error, got %v", err)
	}
}

func TestHostBundleExtractionLimitsAndJunctionRejection(t *testing.T) {
	buildTarGz := func(entries []struct {
		name     string
		typeflag byte
		body     []byte
		linkname string
	}) string {
		t.Helper()
		archive := filepath.Join(t.TempDir(), "bundle.tar.gz")
		f, err := os.Create(archive)
		if err != nil {
			t.Fatal(err)
		}
		defer f.Close()
		gz := gzip.NewWriter(f)
		tw := tar.NewWriter(gz)
		for _, e := range entries {
			hdr := &tar.Header{
				Name:     e.name,
				Typeflag: e.typeflag,
				Mode:     0o644,
				Size:     int64(len(e.body)),
				Linkname: e.linkname,
			}
			if e.typeflag == tar.TypeDir {
				hdr.Mode = 0o755
			}
			if err := tw.WriteHeader(hdr); err != nil {
				t.Fatal(err)
			}
			if len(e.body) > 0 {
				if _, err := tw.Write(e.body); err != nil {
					t.Fatal(err)
				}
			}
		}
		if err := tw.Close(); err != nil {
			t.Fatal(err)
		}
		if err := gz.Close(); err != nil {
			t.Fatal(err)
		}
		return archive
	}

	oversizeArchive := buildTarGz([]struct {
		name     string
		typeflag byte
		body     []byte
		linkname string
	}{
		{name: "runtime", typeflag: tar.TypeDir},
		{name: "runtime/big.bin", typeflag: tar.TypeReg, body: bytes.Repeat([]byte("A"), 2048)},
	})
	if err := extractHostBundleWithLimits(context.Background(), oversizeArchive, t.TempDir(), 64, 1024, 4096); err == nil {
		t.Fatal("expected single-file size limit rejection")
	}

	totalBombArchive := buildTarGz([]struct {
		name     string
		typeflag byte
		body     []byte
		linkname string
	}{
		{name: "runtime", typeflag: tar.TypeDir},
		{name: "runtime/a.bin", typeflag: tar.TypeReg, body: bytes.Repeat([]byte("A"), 600)},
		{name: "runtime/b.bin", typeflag: tar.TypeReg, body: bytes.Repeat([]byte("B"), 600)},
	})
	if err := extractHostBundleWithLimits(context.Background(), totalBombArchive, t.TempDir(), 64, 1024, 1000); err == nil {
		t.Fatal("expected total extracted size limit rejection")
	}

	manyFilesArchive := buildTarGz([]struct {
		name     string
		typeflag byte
		body     []byte
		linkname string
	}{
		{name: "runtime", typeflag: tar.TypeDir},
		{name: "runtime/1.txt", typeflag: tar.TypeReg, body: []byte("1")},
		{name: "runtime/2.txt", typeflag: tar.TypeReg, body: []byte("2")},
	})
	if err := extractHostBundleWithLimits(context.Background(), manyFilesArchive, t.TempDir(), 2, 1024, 4096); err == nil {
		t.Fatal("expected file count limit rejection")
	}

	escapeLinkArchive := buildTarGz([]struct {
		name     string
		typeflag byte
		body     []byte
		linkname string
	}{
		{name: "runtime", typeflag: tar.TypeDir},
		{name: "runtime/bad-link", typeflag: tar.TypeSymlink, linkname: "../../outside"},
	})
	if err := extractHostBundleWithLimits(context.Background(), escapeLinkArchive, t.TempDir(), 64, 1024, 4096); err == nil {
		t.Fatal("expected escaping symlink rejection")
	}

	// Verify Windows NTFS junction detection in validateBridgeRemovalPaths.
	if runtime.GOOS == "windows" {
		base := t.TempDir()
		realDir := filepath.Join(base, "real-versions")
		if err := os.MkdirAll(realDir, 0o755); err != nil {
			t.Fatal(err)
		}
		comp := filepath.Join(base, "components", "whalebridge")
		if err := os.MkdirAll(comp, 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(filepath.Join(comp, "installed.json"), []byte(`{"version":"1.0.0"}`), 0o600); err != nil {
			t.Fatal(err)
		}
		junction := filepath.Join(comp, "versions")
		if out, err := exec.Command("cmd", "/c", "mklink", "/J", junction, realDir).CombinedOutput(); err == nil {
			if err := validateBridgeRemovalPaths(comp); err == nil {
				t.Fatalf("expected validateBridgeRemovalPaths to reject NTFS junction")
			}
		} else {
			t.Logf("mklink /J unavailable: %v (%s)", err, out)
		}
	}
}
