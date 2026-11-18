//go:build windows

package updater

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	"whaleisle/internal/desktop/update"
	"whaleisle/internal/paths"
)

// Uses an actual unsigned NSIS artifact finalized by the production script.
// The caller may supply the complete release artifact as well as a synthetic
// NSIS fixture. This test downloads the same bytes but never launches them.
func TestUnsignedNativeReleaseFinalizedArtifact(t *testing.T) {
	installer := os.Getenv("WHALEISLE_UNSIGNED_RELEASE_FIXTURE")
	if installer == "" {
		t.Skip("set WHALEISLE_UNSIGNED_RELEASE_FIXTURE to the owned finalized NSIS artifact")
	}
	if !filepath.IsAbs(installer) {
		t.Fatal("fixture must be an absolute owned path")
	}
	report := map[string]any{"scope": "actual unsigned NSIS final bytes -> local TLS -> default native updater -> pause/reopen/resume -> durable ready", "installerExecuted": false}
	checks := []string{}
	defer func() {
		if output := os.Getenv("WHALEISLE_UNSIGNED_RELEASE_REPORT"); output != "" {
			report["passed"], report["checks"] = !t.Failed(), checks
			raw, err := json.MarshalIndent(report, "", "  ")
			if err == nil {
				err = os.WriteFile(output, append(raw, '\n'), 0600)
			}
			if err != nil {
				t.Error(err)
			}
		}
	}()
	for _, key := range []string{"WHALEISLE_UPDATE_REQUIRE_SIGNATURE", "WHALEISLE_UPDATE_PUBLISHER"} {
		t.Setenv(key, "")
		if err := os.Unsetenv(key); err != nil {
			t.Fatal(err)
		}
	}
	t.Setenv(paths.EnvVendor, "")
	t.Setenv(paths.EnvUserData, t.TempDir())
	payload, err := os.ReadFile(installer)
	if err != nil || len(payload) < 2 || string(payload[:2]) != "MZ" {
		t.Fatalf("real NSIS artifact: %v", err)
	}
	info, err := update.VerifyAuthenticode(context.Background(), installer)
	if err != nil || info.Status != update.SignatureNone {
		t.Fatalf("expected real unsigned artifact, got %+v %v", info, err)
	}
	report["authenticode"], report["bytes"] = info.Status, len(payload)
	digest := sha256.Sum256(payload)
	sum := hex.EncodeToString(digest[:])
	manifest, err := os.ReadFile(filepath.Join(filepath.Dir(installer), "SHA256SUMS.txt"))
	if err != nil || strings.TrimSpace(string(manifest)) != sum+"  "+filepath.Base(installer) {
		t.Fatalf("final metadata does not identify artifact: %v", err)
	}
	blockRaw, err := os.ReadFile(installer + ".blockmap.json")
	if err != nil {
		t.Fatal(err)
	}
	blocks, err := update.ParseBlockMap(blockRaw)
	if err != nil || blocks.FileSize != int64(len(payload)) {
		t.Fatalf("final blockmap: %+v %v", blocks, err)
	}
	if err := update.VerifyBlockMap(installer, blocks); err != nil {
		t.Fatal(err)
	}
	checks = append(checks, "default finalizer emitted exact size/SHA256/blockmap for unsigned NSIS")

	var interrupt atomic.Bool
	interrupt.Store(true)
	var resumed atomic.Bool
	server := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/fixture.exe" {
			http.NotFound(w, r)
			return
		}
		if interrupt.CompareAndSwap(true, false) {
			w.Header().Set("Content-Length", fmt.Sprint(len(payload)))
			_, _ = w.Write(payload[:65536])
			w.(http.Flusher).Flush()
			<-r.Context().Done()
			return
		}
		if r.Header.Get("Range") != "" {
			resumed.Store(true)
		}
		http.ServeContent(w, r, "fixture.exe", time.Time{}, bytes.NewReader(payload))
	}))
	defer server.Close()
	defer server.CloseClientConnections()
	previousTransport := http.DefaultTransport
	http.DefaultTransport = server.Client().Transport
	t.Cleanup(func() { http.DefaultTransport = previousTransport })
	release := updateCenterFixture()
	currentVersion := "1.2.2"
	if version := os.Getenv("WHALEISLE_UNSIGNED_RELEASE_VERSION"); version != "" {
		release.Version, release.Tag = version, "v"+version
		currentVersion = "0.0.0"
		report["declaredReleaseVersion"] = version
	}
	release.URL, release.Size, release.SHA256 = server.URL+"/fixture.exe", int64(len(payload)), sum
	c := NewCenterForTest(nil, currentVersion, "github")
	c.SetVendorReady(true)
	c.RestoreReleaseState(ReleaseState{CenterLoaded: true, Initialized: true, Rows: []update.NativeRelease{release}, Route: "github", CatalogRoute: "github"})
	wait := func(label string, done func() bool) {
		t.Helper()
		deadline := time.Now().Add(15 * time.Second)
		for time.Now().Before(deadline) {
			if done() {
				return
			}
			time.Sleep(10 * time.Millisecond)
		}
		t.Fatalf("%s timed out: %+v", label, c.ReleaseSnapshot())
	}
	c.Download(nil, release.PackageKey(), false)
	// Progress is throttled. TLS can deliver the first 64 KiB in several reads,
	// so its last published count need not equal the server's flushed prefix.
	// A positive count proves bytes reached the staging writer; Range below
	// independently verifies that reopening actually resumes a persisted prefix.
	wait("partial download", func() bool { return c.ReleaseSnapshot().Received > 0 })
	c.Pause(nil)
	wait("persisted pause", func() bool {
		r, e := c.ReadRecord()
		return !c.ReleaseSnapshot().Busy && e == nil && r.Phase == "paused" && r.Active.PackageKey() == release.PackageKey()
	})
	c = NewCenterForTest(nil, currentVersion, "github")
	c.SetVendorReady(true)
	c.EnsureLoaded()
	if !c.ReleaseSnapshot().Paused {
		t.Fatal("new updater instance did not restore paused download")
	}
	c.Resume(nil)
	wait("durable ready", func() bool {
		r, e := c.ReadRecord()
		return !c.ReleaseSnapshot().Busy && e == nil && r.Phase == "ready"
	})
	record, err := c.ReadRecord()
	if err != nil || record.Staged.SHA256 != sum || record.Staged.Size != int64(len(payload)) || record.StagedTag != release.Tag || record.Staged.Signature != update.SignatureSkipped || record.Staged.Publisher != "" || !resumed.Load() {
		t.Fatalf("default unsigned ready identity: %+v resumed=%v error=%v", record, resumed.Load(), err)
	}
	staged, err := os.ReadFile(record.Staged.Path)
	if err != nil || !bytes.Equal(staged, payload) {
		t.Fatalf("staged final bytes changed: %v", err)
	}
	if err := update.VerifyBlockMap(record.Staged.Path, blocks); err != nil {
		t.Fatal(err)
	}
	reopened := NewCenterForTest(nil, currentVersion, "github")
	reopened.EnsureLoaded()
	if state := reopened.ReleaseSnapshot(); state.Phase != "ready" || state.Staged.Path != record.Staged.Path || state.Staged.Signature != update.SignatureSkipped {
		t.Fatalf("unsigned ready could not be restored: %+v", state)
	}
	checks = append(checks, "default HTTPS staging paused, reopened and resumed exact bytes", "complete durable ready restored with signature skipped and no publisher claim")

	stage := func(candidate update.NativeRelease) error {
		ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
		defer cancel()
		_, err := update.StageNativeReleaseWithOptions(ctx, candidate, update.NativeStageOptions{CurrentVersion: currentVersion, DestDir: t.TempDir()})
		return err
	}
	for _, key := range []string{"WHALEISLE_UPDATE_REQUIRE_SIGNATURE", "WHALEISLE_UPDATE_PUBLISHER"} {
		value := "1"
		if key == "WHALEISLE_UPDATE_PUBLISHER" {
			value = "CN=Owned unsigned fixture"
		}
		_ = os.Setenv(key, value)
		err := stage(release)
		_ = os.Unsetenv(key)
		if err == nil || !strings.Contains(strings.ToLower(err.Error()), "authenticode") {
			t.Fatalf("explicit %s did not reject actual unsigned artifact: %v", key, err)
		}
	}
	checks = append(checks, "explicit strict signature and publisher policies reject actual unsigned NSIS")
	bad := release
	bad.SHA256 = strings.Repeat("0", 64)
	if err := stage(bad); err == nil || !strings.Contains(err.Error(), "sha256") {
		t.Fatalf("bad SHA accepted: %v", err)
	}
	bad = release
	bad.Size++
	if err := stage(bad); err == nil {
		t.Fatal("bad exact size accepted")
	}
	bad = release
	bad.URL = strings.Replace(bad.URL, "https://", "http://", 1)
	if err := stage(bad); err == nil || !strings.Contains(err.Error(), "HTTPS") {
		t.Fatalf("native HTTP source accepted: %v", err)
	}
	checks = append(checks, "SHA256, exact size and native HTTPS source rejection retained")
	t.Logf("PASS actual unsigned NSIS %d bytes: finalized metadata, TLS pause/reopen/resume, durable ready, strict policy and integrity rejection; installer never executed", len(payload))
}
