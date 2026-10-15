package update

import (
	"context"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"path"
	"regexp"
	"runtime"
	"sort"
	"strings"
	"time"
)

const NativeApplication = "whaleisle-mygo"
// These must match the native Host launch contract in harness/current_host.go.
const NativeHostSourceCommit = "90465b67ec16765631d54c195af19d9ff3c3a770"
const NativeHostReadinessGuard = "mygo-visual-readiness-v1"
const NativeCatalogAsset = "WhaleIsle-MyGo-releases.json"
const NativeMaxInstallerBytes int64 = 8 << 30

// NativeCatalog deliberately does not accept an Electron release, legacy feed,
// or portable candidate as a native installer. Its entries use the existing
// updater's URL/hash/size fields and add the required compatibility contract.
type NativeCatalog struct {
	SchemaVersion int             `json:"schemaVersion"`
	Application   string          `json:"application"`
	Releases      []NativeRelease `json:"releases"`
}

type NativeRelease struct {
	Feed
	Tag                string `json:"tag"`
	Platform           string `json:"platform"`
	Arch               string `json:"arch"`
	Format             string `json:"format"`
	Prerelease         bool   `json:"prerelease,omitempty"`
	RuntimeIncluded    bool   `json:"runtimeIncluded"`
	HostSourceCommit   string `json:"hostSourceCommit,omitempty"`
	HostReadinessGuard string `json:"hostReadinessGuard,omitempty"`
	UnavailableReason  string `json:"-"`
}

var nativeTag = regexp.MustCompile(`^[A-Za-z0-9][A-Za-z0-9._-]{0,127}$`)

// NativeCatalogURL honors explicit route configuration, never the Electron repo.
// An existing update-feed URL may supply this envelope too; legacy JSON is
// rejected rather than guessing that an arbitrary .exe contains MyGo/Host.
func NativeCatalogURL(route string) (string, error) {
	for _, key := range []string{"WHALEISLE_RELEASE_CATALOG", EnvFeedURL} {
		if value := strings.TrimSpace(os.Getenv(key)); value != "" { return value, nil }
	}
	switch strings.ToLower(strings.TrimSpace(route)) {
	case "", "github":
		for _, key := range []string{"WHALEISLE_RELEASE_CATALOG_GITHUB", EnvFeedURLGithub} {
			if value := strings.TrimSpace(os.Getenv(key)); value != "" { return value, nil }
		}
		return "https://github.com/ChisaAlter/WhaleIsle-MyGo/releases/latest/download/" + NativeCatalogAsset, nil
	case "cnb", "gitee":
		for _, key := range []string{"WHALEISLE_RELEASE_CATALOG_CNB", EnvFeedURLCNB} {
			if value := strings.TrimSpace(os.Getenv(key)); value != "" { return value, nil }
		}
		return "", fmt.Errorf("MyGo CNB release catalog is not configured; choose GitHub or set WHALEISLE_RELEASE_CATALOG_CNB")
	default:
		return "", fmt.Errorf("unsupported native release route")
	}
}

func nativeHTTPS(raw string) (*url.URL, error) {
	u, err := url.Parse(raw)
	if err != nil || u.Scheme != "https" || u.Hostname() == "" || u.User != nil || u.Fragment != "" {
		return nil, fmt.Errorf("native release metadata and assets require HTTPS without credentials")
	}
	return u, nil
}

// Validate binds version, tag, architecture, installer and Host provenance. A
// first runtime installation must use the full native NSIS package, not an
// Electron setup, component binary, source ZIP or unverified portable candidate.
func (r NativeRelease) Validate(goos, arch string, requireRuntime bool) error {
	if !nativeTag.MatchString(r.Tag) || strings.TrimSpace(r.Version) == "" || strings.TrimSpace(r.Version) != r.Version {
		return fmt.Errorf("release is missing a valid tag or version")
	}
	if r.Tag != r.Version && r.Tag != "v"+r.Version && r.Tag != "mygo-v"+r.Version {
		return fmt.Errorf("release tag does not identify its declared version")
	}
	if r.Platform != goos || r.Arch != arch || r.Platform != "windows" || r.Format != "nsis" {
		return fmt.Errorf("no compatible MyGo NSIS installer for %s/%s", goos, arch)
	}
	if r.Arch != "amd64" && r.Arch != "arm64" { return fmt.Errorf("unsupported native installer architecture") }
	hash, err := hex.DecodeString(r.SHA256)
	if err != nil || len(hash) != 32 || r.Size <= 0 || r.Size > NativeMaxInstallerBytes {
		return fmt.Errorf("native installer requires SHA-256 and a bounded exact size")
	}
	u, err := nativeHTTPS(r.URL)
	if err != nil { return err }
	if !strings.HasSuffix(strings.ToLower(u.Path), ".exe") || u.RawQuery != "" {
		return fmt.Errorf("native release must reference a pinned installer")
	}
	// Official GitHub addresses must stay within this native repository and tag.
	if strings.EqualFold(u.Hostname(), "github.com") {
		prefix := "/ChisaAlter/WhaleIsle-MyGo/releases/download/" + r.Tag + "/"
		if !strings.HasPrefix(u.Path, prefix) || path.Base(u.Path) == "." || strings.Contains(strings.TrimPrefix(u.Path, prefix), "/") {
			return fmt.Errorf("installer is not pinned to the selected MyGo release tag")
		}
	}
	if requireRuntime && !r.RuntimeIncluded { return fmt.Errorf("selected release is shell-only and cannot install the Host runtime") }
	if r.RuntimeIncluded && (r.HostSourceCommit != NativeHostSourceCommit || r.HostReadinessGuard != NativeHostReadinessGuard) {
		return fmt.Errorf("bundled Host does not match this MyGo build's source and readiness guard")
	}
	return nil
}

func DecodeNativeCatalog(body []byte, goos, arch string) ([]NativeRelease, error) {
	var catalog NativeCatalog
	if err := json.Unmarshal(body, &catalog); err != nil { return nil, fmt.Errorf("invalid native release catalog: %w", err) }
	if catalog.SchemaVersion != 1 || catalog.Application != NativeApplication {
		return nil, fmt.Errorf("release manifest is not a WhaleIsle-MyGo native catalog; Electron and legacy feeds cannot install this runtime")
	}
	if len(catalog.Releases) > 500 { return nil, fmt.Errorf("native release catalog exceeds 500 entries") }
	seen := map[string]bool{}
	rows := append([]NativeRelease(nil), catalog.Releases...)
	for i := range rows {
		r := &rows[i]
		key := r.Tag + "/" + r.Platform + "/" + r.Arch
		if seen[key] { return nil, fmt.Errorf("native release catalog contains duplicate tag/target %s", key) }
		seen[key] = true
		if err := r.Validate(goos, arch, false); err != nil { r.UnavailableReason = err.Error() }
	}
	sort.SliceStable(rows, func(i, j int) bool {
		if rows[i].Prerelease != rows[j].Prerelease { return !rows[i].Prerelease }
		return CompareVersions(rows[i].Version, rows[j].Version) > 0
	})
	return rows, nil
}

// ListNativeReleases fetches once and returns immutable pinned entries. Selection
// installs that exact entry; a subsequent moving "latest" cannot change it.
func ListNativeReleases(ctx context.Context, route string) ([]NativeRelease, error) {
	address, err := NativeCatalogURL(route)
	if err != nil { return nil, err }
	if _, err := nativeHTTPS(address); err != nil { return nil, err }
	client := nativeHTTPClient(20 * time.Second)
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, address, nil)
	if err != nil { return nil, err }
	req.Header.Set("Accept", "application/json")
	resp, err := client.Do(req)
	if err != nil { return nil, err }
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("MyGo release catalog HTTP %d; a compatible published native installer is required", resp.StatusCode)
	}
	body, err := io.ReadAll(io.LimitReader(resp.Body, (2<<20)+1))
	if err != nil { return nil, err }
	if len(body) > 2<<20 { return nil, fmt.Errorf("native catalog exceeds size limit") }
	return DecodeNativeCatalog(body, runtime.GOOS, runtime.GOARCH)
}

func nativeHTTPClient(timeout time.Duration) *http.Client {
	return &http.Client{Timeout: timeout, CheckRedirect: func(req *http.Request, via []*http.Request) error {
		if len(via) >= 10 { return fmt.Errorf("too many native release redirects") }
		_, err := nativeHTTPS(req.URL.String())
		return err
	}}
}

func StageNativeRelease(ctx context.Context, release NativeRelease, requireRuntime bool, destDir string, onProgress func(int64, int64)) (ApplyResult, error) {
	if err := release.Validate(runtime.GOOS, runtime.GOARCH, requireRuntime); err != nil { return ApplyResult{}, err }
	requireSignature := strings.EqualFold(os.Getenv("WHALEISLE_UPDATE_REQUIRE_SIGNATURE"), "true") || os.Getenv("WHALEISLE_UPDATE_REQUIRE_SIGNATURE") == "1"
	return DownloadAndStageOpts(ctx, StageOptions{
		ReleaseURL: release.URL, DestDir: destDir, HTTPClient: nativeHTTPClient(30 * time.Minute),
		ExpectSHA256: release.SHA256, ExpectSize: release.Size, MaxBytes: NativeMaxInstallerBytes, OnProgress: onProgress,
		RequireSignature: requireSignature, ExpectedPublisher: strings.TrimSpace(os.Getenv("WHALEISLE_UPDATE_PUBLISHER")),
	})
}
