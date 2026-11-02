package update

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"
)

func nativeFixture() NativeRelease {
	return NativeRelease{Feed: Feed{Version: "1.2.3", URL: "https://github.com/ChisaAlter/WhaleIsle-MyGo/releases/download/v1.2.3/WhaleIsle-Setup-1.2.3.exe", SHA256: strings.Repeat("a", 64), Size: 1024},
		Tag: "v1.2.3", Platform: "windows", Arch: "amd64", Format: "nsis", RuntimeIncluded: true,
		HostSourceCommit: NativeHostSourceCommit, HostReadinessGuard: NativeHostReadinessGuard}
}

func TestNativeCatalogRejectsLegacyAndDuplicates(t *testing.T) {
	for _, body := range []string{
		`{"version":"1.2.3","url":"https://github.com/ChisaAlter/WhaleIsle/releases/download/v1.2.3/setup.exe"}`,
		`{"schemaVersion":1,"application":"electron","releases":[]}`,
	} {
		if _, err := DecodeNativeCatalog([]byte(body), "windows", "amd64"); err == nil {
			t.Fatal("accepted a non-MyGo catalog")
		}
	}
	r := nativeFixture()
	raw, _ := json.Marshal(NativeCatalog{SchemaVersion: 1, Application: NativeApplication, Releases: []NativeRelease{r, r}})
	if _, err := DecodeNativeCatalog(raw, "windows", "amd64"); err == nil {
		t.Fatal("accepted an ambiguous tag")
	}
}

func TestNativeReleaseCompatibilityAndRuntime(t *testing.T) {
	r := nativeFixture()
	if err := r.Validate("windows", "amd64", true); err != nil {
		t.Fatal(err)
	}
	r.RuntimeIncluded = false
	if err := r.Validate("windows", "amd64", true); err == nil {
		t.Fatal("shell-only release accepted for runtime install")
	}
	if err := r.Validate("windows", "amd64", false); err != nil {
		t.Fatal(err)
	}
	r = nativeFixture()
	r.HostSourceCommit = "old"
	if err := r.Validate("windows", "amd64", true); err == nil {
		t.Fatal("old Host accepted")
	}
	r = nativeFixture()
	if err := r.Validate("linux", "amd64", false); err == nil {
		t.Fatal("Windows installer accepted on Linux")
	}
	r.URL = "https://github.com/ChisaAlter/WhaleIsle/releases/download/v1.2.3/setup.exe"
	if err := r.Validate("windows", "amd64", false); err == nil {
		t.Fatal("Electron source accepted")
	}
}

func TestStageProgressCancellationRemovesPartial(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		_, _ = w.Write([]byte(strings.Repeat("x", 128<<10)))
	}))
	defer server.Close()
	seen := false
	_, err := DownloadAndStageOpts(ctx, StageOptions{ReleaseURL: server.URL + "/setup.exe", DestDir: t.TempDir(), OnProgress: func(n, total int64) { seen = true; cancel() }})
	if !seen || err == nil {
		t.Fatal("cancellation must prevent staging success")
	}
}

func TestNativeReleaseInstalledRC2ProvenancePair(t *testing.T) {
	r := nativeFixture()
	r.HostSourceCommit = NativeHostRC2SourceCommit
	r.HostReadinessGuard = NativeHostRC2ReadinessGuard
	if err := r.Validate("windows", "amd64", true); err != nil {
		t.Fatal(err)
	}
	for _, guard := range []string{"", NativeHostReadinessGuard, "unknown"} {
		r.HostReadinessGuard = guard
		if err := r.Validate("windows", "amd64", true); err == nil {
			t.Fatalf("installed rc2 accepted mismatched readiness guard %q", guard)
		}
	}
	r.HostSourceCommit, r.HostReadinessGuard = NativeHostSourceCommit, NativeHostRC2ReadinessGuard
	if err := r.Validate("windows", "amd64", true); err == nil {
		t.Fatal("source build accepted installed rc2 guard")
	}
}

// Explicit local release output only: this never downloads, installs or starts
// the artifact. The publisher has already hashed the actual staged installer.
func TestNativeReleaseCatalogBuiltFixture(t *testing.T) {
	filename := os.Getenv("WHALEISLE_NATIVE_RELEASE_FIXTURE_CATALOG")
	if filename == "" {
		t.Skip("set WHALEISLE_NATIVE_RELEASE_FIXTURE_CATALOG to the actual emitted local catalog")
	}
	body, err := os.ReadFile(filename)
	if err != nil {
		t.Fatal(err)
	}
	rows, err := DecodeNativeCatalog(body, "windows", "amd64")
	if err != nil {
		t.Fatal(err)
	}
	if len(rows) != 1 {
		t.Fatalf("want one real installer, got %d", len(rows))
	}
	row := rows[0]
	if err := row.Validate("windows", "amd64", true); err != nil {
		t.Fatal(err)
	}
	if !row.RuntimeIncluded || row.HostSourceCommit != NativeHostRC2SourceCommit || row.HostReadinessGuard != NativeHostRC2ReadinessGuard || row.Size <= 0 {
		t.Fatalf("unexpected real installed runtime release: %+v", row)
	}
}

func TestNativeReleaseExactSourcePins(t *testing.T) {
	for _, pin := range []string{NativeHostSourceCommit, "e9f584be6d3b491ce777bb6823f30bc2bbc6f195"} {
		r := nativeFixture()
		r.HostSourceCommit = pin
		if err := r.Validate("windows", "amd64", true); err != nil {
			t.Fatalf("pin %s: %v", pin, err)
		}
		r.HostReadinessGuard = ""
		if err := r.Validate("windows", "amd64", true); err == nil {
			t.Fatal("missing guard accepted")
		}
	}
	r := nativeFixture()
	r.HostSourceCommit = strings.Repeat("f", 40)
	if err := r.Validate("windows", "amd64", true); err == nil {
		t.Fatal("unknown source accepted")
	}
}
