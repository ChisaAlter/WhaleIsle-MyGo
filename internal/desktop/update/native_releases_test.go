package update

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
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
