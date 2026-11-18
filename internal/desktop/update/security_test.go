package update

import (
	"context"
	"net/http"
	"net/http/httptest"
	"net/url"
	"os"
	"runtime"
	"strings"
	"testing"
	"time"
)

func TestRejectsNonLoopbackHTTPAndUserinfoURLs(t *testing.T) {
	ctx := context.Background()
	dest := t.TempDir()

	badURLs := []string{
		"http://example.com/setup.exe",
		"http://8.8.8.8/setup.exe",
		"https://user:pass@example.com/setup.exe",
		"http://user:pass@127.0.0.1:8080/setup.exe",
		"file:///C:/temp/setup.exe",
	}
	for _, raw := range badURLs {
		if _, err := validateDownloadURL(raw); err == nil {
			t.Fatalf("validateDownloadURL(%q) unexpectedly succeeded", raw)
		}
		if _, err := DownloadAndStageOpts(ctx, StageOptions{ReleaseURL: raw, DestDir: dest}); err == nil {
			t.Fatalf("DownloadAndStageOpts(%q) unexpectedly succeeded", raw)
		}
		if _, err := FetchRange(ctx, raw, 0, 16, nil); err == nil {
			t.Fatalf("FetchRange(%q) unexpectedly succeeded", raw)
		}
		if _, err := DownloadBlockMap(ctx, raw, nil); err == nil {
			t.Fatalf("DownloadBlockMap(%q) unexpectedly succeeded", raw)
		}
		if _, err := ApplyBlockPatchOpts(ctx, PatchOptions{
			Feed:    Feed{Version: "1.2.0", URL: raw},
			DestDir: dest,
		}); err == nil {
			t.Fatalf("ApplyBlockPatchOpts(%q) unexpectedly succeeded", raw)
		}
	}
}

func TestDownloadRedirectsBlockDowngradeAndCrossHost(t *testing.T) {
	ctx := context.Background()

	httpTarget := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte("plain-http-payload"))
	}))
	defer httpTarget.Close()

	tlsDowngrade := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.Redirect(w, r, httpTarget.URL+"/setup.exe", http.StatusFound)
	}))
	defer tlsDowngrade.Close()

	tlsSameHost := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/redirect" {
			http.Redirect(w, r, "/final.exe", http.StatusFound)
			return
		}
		_, _ = w.Write([]byte("ok-tls-payload"))
	}))
	defer tlsSameHost.Close()

	client := tlsDowngrade.Client()
	dest := t.TempDir()

	if _, err := DownloadAndStageOpts(ctx, StageOptions{
		ReleaseURL: tlsDowngrade.URL + "/setup.exe",
		DestDir:    dest,
		HTTPClient: client,
	}); err == nil || !strings.Contains(err.Error(), "downgrade") {
		t.Fatalf("expected HTTPS->HTTP downgrade rejection, got %v", err)
	}
	if _, err := FetchRange(ctx, tlsDowngrade.URL+"/setup.exe", 0, 10, client); err == nil {
		t.Fatalf("FetchRange accepted HTTPS->HTTP downgrade")
	}
	if _, err := DownloadBlockMap(ctx, tlsDowngrade.URL+"/map.json", client); err == nil {
		t.Fatalf("DownloadBlockMap accepted HTTPS->HTTP downgrade")
	}

	// Same-host HTTPS redirect must continue to work cleanly.
	staged, err := DownloadAndStageOpts(ctx, StageOptions{
		ReleaseURL: tlsSameHost.URL + "/redirect",
		DestDir:    dest,
		HTTPClient: tlsSameHost.Client(),
	})
	if err != nil {
		t.Fatalf("same-host HTTPS redirect failed: %v", err)
	}
	if got, err := os.ReadFile(staged.Path); err != nil || string(got) != "ok-tls-payload" {
		t.Fatalf("unexpected staged payload %q err=%v", got, err)
	}

	// Redirect policy unit checks for CDN allowlist, cross-host denial, and internal SSRF denial.
	parse := func(raw string) *url.URL {
		u, err := url.Parse(raw)
		if err != nil {
			t.Fatal(err)
		}
		return u
	}
	ghInit := parse("https://github.com/org/repo/releases/download/v1.0.0/setup.exe")
	if err := validateRedirectTarget(ghInit, ghInit, parse("https://objects.githubusercontent.com/abc/setup.exe")); err != nil {
		t.Fatalf("expected GitHub CDN redirect allowed, got %v", err)
	}
	if err := validateRedirectTarget(ghInit, ghInit, parse("https://evil.example/setup.exe")); err == nil {
		t.Fatal("expected unknown cross-host redirect rejected")
	}
	if err := validateRedirectTarget(ghInit, ghInit, parse("http://objects.githubusercontent.com/abc/setup.exe")); err == nil {
		t.Fatal("expected GitHub CDN HTTPS->HTTP downgrade rejected")
	}
	pubInit := parse("https://updates.example.com/setup.exe")
	if err := validateRedirectTarget(pubInit, pubInit, parse("https://127.0.0.1/setup.exe")); err == nil {
		t.Fatal("expected public->loopback redirect rejected")
	}
	if err := validateRedirectTarget(parse("http://127.0.0.1:8080/a"), parse("http://127.0.0.1:8080/a"), parse("http://localhost:8080/b")); err == nil {
		t.Fatal("expected cross-host loopback redirect rejected")
	}

	// Cross-host header stripping verification via hardened client.
	hardened := hardenDownloadClient(nil, time.Second)
	req := httptest.NewRequest(http.MethodGet, "https://objects.githubusercontent.com/asset", nil)
	req.Header.Set("Authorization", "Bearer secret")
	req.Header.Set("Cookie", "sid=123")
	prevReq := httptest.NewRequest(http.MethodGet, "https://github.com/org/repo/releases/download/v1/asset", nil)
	if err := hardened.CheckRedirect(req, []*http.Request{prevReq}); err != nil {
		t.Fatalf("expected allowed GitHub CDN hop, got %v", err)
	}
	if req.Header.Get("Authorization") != "" || req.Header.Get("Cookie") != "" {
		t.Fatalf("expected sensitive headers stripped on cross-host redirect")
	}
}

func TestPublisherRDNStrictMatching(t *testing.T) {
	positives := []struct {
		publisher string
		expected  string
	}{
		{"CN=WhaleIsle Inc, O=WhaleIsle", "whaleisle"},
		{"CN=WhaleIsle Inc, O=WhaleIsle", "CN=WhaleIsle Inc"},
		{"CN=WhaleIsle Inc, O=WhaleIsle", "O=whaleisle"},
		{"CN=Acme", "Acme"},
		{`CN="WhaleIsle, Inc.", O=Other`, "WhaleIsle, Inc."},
		{"Authority=Developer ID Application: WhaleIsle Inc (TEAM123); TeamID=TEAM123", "TEAM123"},
		{"Authority=Developer ID Application: WhaleIsle Inc (TEAM123); TeamID=TEAM123", "Developer ID Application: WhaleIsle Inc (TEAM123)"},
	}
	for _, tc := range positives {
		if !publisherMatches(tc.publisher, tc.expected) {
			t.Fatalf("expected publisher %q to match %q", tc.publisher, tc.expected)
		}
	}

	spoofs := []struct {
		publisher string
		expected  string
	}{
		{"CN=Evil (WhaleIsle)", "whaleisle"},
		{"CN=WhaleIsle Fake, O=Fake Corp", "whaleisle"},
		{"CN=Attacker, OU=WhaleIsle", "whaleisle"},
		{`CN=Evil\, O=WhaleIsle`, "whaleisle"},
		{`CN="Evil, O=WhaleIsle"`, "whaleisle"},
		{"CN=Evil, O=WhaleIsle", "CN=WhaleIsle"},
		{"CN=Evil, O=WhaleIsleCorp", "whaleisle"},
		{"Authority=Developer ID Application: Evil (WhaleIsle); TeamID=EVIL1", "whaleisle"},
	}
	for _, tc := range spoofs {
		if publisherMatches(tc.publisher, tc.expected) {
			t.Fatalf("spoofed publisher %q matched %q", tc.publisher, tc.expected)
		}
	}
}

func TestNativeReleasesRejectDowngradeAndHonorSignaturePolicy(t *testing.T) {
	rows := []NativeRelease{
		{Feed: Feed{Version: "1.2.1", URL: "https://github.com/ChisaAlter/WhaleIsle-MyGo/releases/download/v1.2.1/setup.exe", SHA256: strings.Repeat("a", 64), Size: 16}, Tag: "v1.2.1", Platform: runtime.GOOS, Arch: runtime.GOARCH, Format: "nsis", DownloadRoute: "github"},
		{Feed: Feed{Version: "1.2.2", URL: "https://github.com/ChisaAlter/WhaleIsle-MyGo/releases/download/v1.2.2/setup.exe", SHA256: strings.Repeat("b", 64), Size: 16}, Tag: "v1.2.2", Platform: runtime.GOOS, Arch: runtime.GOARCH, Format: "nsis", DownloadRoute: "github"},
		{Feed: Feed{Version: "1.2.3", URL: "https://github.com/ChisaAlter/WhaleIsle-MyGo/releases/download/v1.2.3/setup.exe", SHA256: strings.Repeat("c", 64), Size: 16}, Tag: "v1.2.3", Platform: runtime.GOOS, Arch: runtime.GOARCH, Format: "nsis", DownloadRoute: "github"},
	}

	prepared := PrepareNativeReleases(rows, "1.2.2", t.TempDir())
	if prepared[0].UnavailableReason == "" {
		t.Fatal("expected downgrade release 1.2.1 < 1.2.2 to have UnavailableReason")
	}
	if prepared[1].UnavailableReason != "" || prepared[2].UnavailableReason != "" {
		t.Fatalf("unexpected UnavailableReason on same/newer versions: %+v", prepared)
	}
	if _, err := SelectNativeReleaseForVersion(rows, "v1.2.1", "1.2.2", runtime.GOOS, runtime.GOARCH, false); err == nil {
		t.Fatal("expected SelectNativeReleaseForVersion to reject downgrade")
	}
	if _, err := SelectNativeReleaseForVersion(rows, "v1.2.3", "1.2.2", runtime.GOOS, runtime.GOARCH, false); err != nil {
		t.Fatalf("expected newer release allowed, got %v", err)
	}

	if _, err := StageNativeReleaseWithOptions(context.Background(), rows[0], NativeStageOptions{
		CurrentVersion: "1.2.2",
		DestDir:        t.TempDir(),
	}); err == nil || !strings.Contains(strings.ToLower(err.Error()), "downgrade") {
		t.Fatalf("expected downgrade rejection error, got %v", err)
	}

	// Windows unsigned releases are normal production artifacts. Other platform
	// defaults remain strict; explicit policy and publisher constraints still win.
	t.Setenv("WHALEISLE_UPDATE_REQUIRE_SIGNATURE", "1")
	_ = os.Unsetenv("WHALEISLE_UPDATE_REQUIRE_SIGNATURE")
	if got := nativeSignatureRequired(""); got != (runtime.GOOS != "windows") {
		t.Fatalf("unexpected platform default: %v", got)
	}
	for _, raw := range []string{"0", "false", "no"} {
		t.Setenv("WHALEISLE_UPDATE_REQUIRE_SIGNATURE", raw)
		if nativeSignatureRequired("") {
			t.Fatalf("explicit opt-out %q ignored", raw)
		}
	}
	for _, raw := range []string{"1", "true", "yes", "invalid"} {
		t.Setenv("WHALEISLE_UPDATE_REQUIRE_SIGNATURE", raw)
		if !nativeSignatureRequired("") {
			t.Fatalf("explicit strict policy %q ignored", raw)
		}
	}
	t.Setenv("WHALEISLE_UPDATE_REQUIRE_SIGNATURE", "")
	if got := nativeSignatureRequired(""); got != (runtime.GOOS != "windows") {
		t.Fatal("empty policy differs between tests and production")
	}
	t.Setenv("WHALEISLE_UPDATE_REQUIRE_SIGNATURE", "0")
	if !nativeSignatureRequired("WhaleIsle") {
		t.Fatal("expected explicit publisher to enforce signature verification")
	}
}
