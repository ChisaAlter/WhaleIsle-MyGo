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
	"path/filepath"
	"regexp"
	"runtime"
	"sort"
	"strings"
	"time"
)

const NativeApplication = "whaleisle-mygo"

// These must match the native Host launch contract in harness/current_host.go.
const NativeHostSourceCommit = "90465b67ec16765631d54c195af19d9ff3c3a770"
const NativeHostE9SourceCommit = "e9f584be6d3b491ce777bb6823f30bc2bbc6f195"
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

// NativeDelta is a pinned native range reconstruction recipe. It is separate
// from Feed.BlockMap, which belongs to the legacy untyped updater.
type NativeDelta struct {
	Algorithm     string       `json:"algorithm"`
	BasePackageID string       `json:"basePackageID"`
	BaseVersion   string       `json:"baseVersion"`
	BaseSHA256    string       `json:"baseSHA256"`
	BaseSize      int64        `json:"baseSize"`
	FullPackageID string       `json:"fullPackageID"`
	DownloadSize  int64        `json:"downloadSize"`
	BlockMap      BlockMapInfo `json:"blockMap"`
}

type NativeRelease struct {
	Feed
	PackageID          string            `json:"packageID,omitempty"`
	PackageKind        string            `json:"packageKind,omitempty"`
	DownloadURLs       map[string]string `json:"downloadURLs,omitempty"`
	Delta              *NativeDelta      `json:"delta,omitempty"`
	DownloadRoute      string            `json:"-"`
	BasePath           string            `json:"-"`
	Tag                string            `json:"tag"`
	Platform           string            `json:"platform"`
	Arch               string            `json:"arch"`
	Format             string            `json:"format"`
	Prerelease         bool              `json:"prerelease,omitempty"`
	RuntimeIncluded    bool              `json:"runtimeIncluded"`
	HostSourceCommit   string            `json:"hostSourceCommit,omitempty"`
	HostReadinessGuard string            `json:"hostReadinessGuard,omitempty"`
	UnavailableReason  string            `json:"-"`
}

var nativeTag = regexp.MustCompile(`^[A-Za-z0-9][A-Za-z0-9._-]{0,127}$`)

func (r NativeRelease) IsDelta() bool { return r.PackageKind == "delta" }

func (r NativeRelease) FullPackageKey() string {
	variant := "desktop"
	if r.RuntimeIncluded {
		variant = "runtime"
	}
	return strings.Join([]string{r.Tag, r.Platform, r.Arch, "full", variant}, "/")
}

// PackageKey identifies the actual variant; a tag alone cannot distinguish a
// desktop installer, a runtime installer and recipes for different bases.
func (r NativeRelease) PackageKey() string {
	key := r.FullPackageKey()
	if r.IsDelta() && r.Delta != nil {
		return strings.Replace(key, "/full/", "/delta/", 1) + "/" + r.Delta.BaseVersion + "/" + strings.ToLower(r.Delta.BaseSHA256)
	}
	return key
}

func (r NativeRelease) DownloadBytes() int64 {
	if r.IsDelta() && r.Delta != nil {
		return r.Delta.DownloadSize
	}
	return r.Size
}

func nativeRoute(route string) string {
	switch strings.ToLower(strings.TrimSpace(route)) {
	case "", "github":
		return "github"
	case "gitee", "cnb":
		return "cnb"
	default:
		return strings.ToLower(strings.TrimSpace(route))
	}
}

func nativeInstallerURL(raw, tag, format string) error {
	u, err := nativeHTTPS(raw)
	if err != nil {
		return err
	}
	suffix := ".exe"
	if format == "dmg" {
		suffix = ".dmg"
	}
	if !strings.HasSuffix(strings.ToLower(u.Path), suffix) || u.RawQuery != "" {
		return fmt.Errorf("native release must reference a pinned installer")
	}
	if strings.EqualFold(u.Hostname(), "github.com") {
		prefix := "/ChisaAlter/WhaleIsle-MyGo/releases/download/" + tag + "/"
		if !strings.HasPrefix(u.Path, prefix) || path.Base(u.Path) == "." || strings.Contains(strings.TrimPrefix(u.Path, prefix), "/") {
			return fmt.Errorf("installer is not pinned to the selected MyGo release tag")
		}
	}
	return nil
}

// NativeCatalogURL honors explicit route configuration, never the Electron repo.
// An existing update-feed URL may supply this envelope too; legacy JSON is
// rejected rather than guessing that an arbitrary .exe contains MyGo/Host.
func NativeCatalogURL(route string) (string, error) {
	for _, key := range []string{"WHALEISLE_RELEASE_CATALOG", EnvFeedURL} {
		if value := strings.TrimSpace(os.Getenv(key)); value != "" {
			return value, nil
		}
	}
	switch strings.ToLower(strings.TrimSpace(route)) {
	case "", "github":
		for _, key := range []string{"WHALEISLE_RELEASE_CATALOG_GITHUB", EnvFeedURLGithub} {
			if value := strings.TrimSpace(os.Getenv(key)); value != "" {
				return value, nil
			}
		}
		return "https://github.com/ChisaAlter/WhaleIsle-MyGo/releases/latest/download/" + NativeCatalogAsset, nil
	case "cnb", "gitee":
		for _, key := range []string{"WHALEISLE_RELEASE_CATALOG_CNB", EnvFeedURLCNB} {
			if value := strings.TrimSpace(os.Getenv(key)); value != "" {
				return value, nil
			}
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
	if r.Platform != goos || r.Arch != arch || !((r.Platform == "windows" && r.Format == "nsis") || (r.Platform == "darwin" && r.Format == "dmg")) {
		return fmt.Errorf("no compatible MyGo native installer for %s/%s", goos, arch)
	}
	if r.Arch != "amd64" && r.Arch != "arm64" {
		return fmt.Errorf("unsupported native installer architecture")
	}
	hash, err := hex.DecodeString(r.SHA256)
	if err != nil || len(hash) != 32 || r.Size <= 0 || r.Size > NativeMaxInstallerBytes {
		return fmt.Errorf("native installer requires SHA-256 and a bounded exact size")
	}
	if r.PackageKind != "" && r.PackageKind != "full" && r.PackageKind != "delta" {
		return fmt.Errorf("unsupported native package kind")
	}
	if r.PackageID != "" && r.PackageID != r.PackageKey() {
		return fmt.Errorf("native package identity does not match its target and variant")
	}
	if err := nativeInstallerURL(r.URL, r.Tag, r.Format); err != nil {
		return err
	}
	for route, address := range r.DownloadURLs {
		if route != "github" && route != "cnb" {
			return fmt.Errorf("unsupported native mirror route")
		}
		if err := nativeInstallerURL(address, r.Tag, r.Format); err != nil {
			return err
		}
	}
	if r.IsDelta() {
		d := r.Delta
		if r.Platform != "windows" || d == nil || d.Algorithm != "sha256-range-v1" {
			return fmt.Errorf("unsupported native delta reconstruction contract")
		}
		digest, err := hex.DecodeString(d.BaseSHA256)
		if err != nil || len(digest) != 32 || d.BaseSize <= 0 || d.BaseSize > NativeMaxInstallerBytes || strings.TrimSpace(d.BaseVersion) == "" || d.BaseVersion != strings.TrimSpace(d.BaseVersion) || d.BaseVersion == r.Version || d.BasePackageID == "" || d.FullPackageID != r.FullPackageKey() {
			return fmt.Errorf("native delta requires an exact base identity and matching full package")
		}
		if d.BlockMap.FileSize != r.Size || len(d.BlockMap.Blocks) == 0 || d.DownloadSize < 0 || d.DownloadSize >= r.Size {
			return fmt.Errorf("native delta must reuse bytes and reconstruct the complete installer")
		}
		var end int64
		for _, b := range d.BlockMap.Blocks {
			digest, err := hex.DecodeString(b.SHA256)
			if err != nil || len(digest) != 32 || b.Offset != end || b.Size <= 0 || b.Size > maxRangeBytes || b.Size > r.Size-end {
				return fmt.Errorf("native delta block map must cover the installer exactly")
			}
			end += b.Size
		}
		if end != r.Size {
			return fmt.Errorf("native delta block map is incomplete")
		}
	} else if r.Delta != nil {
		return fmt.Errorf("full package cannot carry a delta recipe")
	}
	if requireRuntime && !r.RuntimeIncluded {
		return fmt.Errorf("selected release is shell-only and cannot install the Host runtime")
	}
	if r.RuntimeIncluded && ((r.HostSourceCommit != NativeHostSourceCommit && r.HostSourceCommit != NativeHostE9SourceCommit) || r.HostReadinessGuard != NativeHostReadinessGuard) {
		return fmt.Errorf("bundled Host does not match this MyGo build's source and readiness guard")
	}
	return nil
}

func DecodeNativeCatalog(body []byte, goos, arch string) ([]NativeRelease, error) {
	var catalog NativeCatalog
	if err := json.Unmarshal(body, &catalog); err != nil {
		return nil, fmt.Errorf("invalid native release catalog: %w", err)
	}
	if (catalog.SchemaVersion != 1 && catalog.SchemaVersion != 2) || catalog.Application != NativeApplication {
		return nil, fmt.Errorf("release manifest is not a WhaleIsle-MyGo native catalog; Electron and legacy feeds cannot install this runtime")
	}
	if len(catalog.Releases) > 500 {
		return nil, fmt.Errorf("native release catalog exceeds 500 entries")
	}
	seen := map[string]bool{}
	rows := append([]NativeRelease(nil), catalog.Releases...)
	for i := range rows {
		r := &rows[i]
		key := r.PackageKey()
		if catalog.SchemaVersion == 1 {
			key = r.Tag + "/" + r.Platform + "/" + r.Arch
		}
		if catalog.SchemaVersion == 2 && (r.PackageID == "" || r.PackageKind == "") {
			return nil, fmt.Errorf("native v2 catalog requires package identities and kinds")
		}
		if seen[key] {
			return nil, fmt.Errorf("native release catalog contains duplicate package %s", key)
		}
		seen[key] = true
		if err := r.Validate(goos, arch, false); err != nil {
			r.UnavailableReason = err.Error()
		}
	}
	// Each recipe must reconstruct the same pinned bytes and runtime variant as
	// an available full entry. If it cannot, the full entry remains usable.
	full := map[string]NativeRelease{}
	for _, r := range rows {
		if !r.IsDelta() {
			full[r.PackageKey()] = r
		}
	}
	for i := range rows {
		r := &rows[i]
		if !r.IsDelta() || r.UnavailableReason != "" {
			continue
		}
		target, ok := full[r.Delta.FullPackageID]
		if !ok || target.UnavailableReason != "" || target.Version != r.Version || target.Size != r.Size || !strings.EqualFold(target.SHA256, r.SHA256) || target.RuntimeIncluded != r.RuntimeIncluded || target.HostSourceCommit != r.HostSourceCommit || target.HostReadinessGuard != r.HostReadinessGuard {
			r.UnavailableReason = "native delta has no matching full installer"
		}
	}
	sort.SliceStable(rows, func(i, j int) bool {
		if rows[i].Prerelease != rows[j].Prerelease {
			return !rows[i].Prerelease
		}
		if cmp := CompareVersions(rows[i].Version, rows[j].Version); cmp != 0 {
			return cmp > 0
		}
		if rows[i].RuntimeIncluded != rows[j].RuntimeIncluded {
			return !rows[i].RuntimeIncluded
		}
		return rows[i].DownloadBytes() < rows[j].DownloadBytes()
	})
	return rows, nil
}

// ListNativeReleases fetches once and returns immutable pinned entries. Selection
// installs that exact entry; a subsequent moving "latest" cannot change it.
func ListNativeReleases(ctx context.Context, route string) ([]NativeRelease, error) {
	address, err := NativeCatalogURL(route)
	if err != nil {
		return nil, err
	}
	if _, err := nativeHTTPS(address); err != nil {
		return nil, err
	}
	client := nativeHTTPClient(20 * time.Second)
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, address, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Accept", "application/json")
	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("MyGo release catalog HTTP %d; a compatible published native installer is required", resp.StatusCode)
	}
	body, err := io.ReadAll(io.LimitReader(resp.Body, (2<<20)+1))
	if err != nil {
		return nil, err
	}
	if len(body) > 2<<20 {
		return nil, fmt.Errorf("native catalog exceeds size limit")
	}
	rows, err := DecodeNativeCatalog(body, runtime.GOOS, runtime.GOARCH)
	if err != nil {
		return nil, err
	}
	selectedRoute := nativeRoute(route)
	for i := range rows {
		r := &rows[i]
		r.DownloadRoute = selectedRoute
		if len(r.DownloadURLs) == 0 {
			continue
		} // Legacy route-specific catalog.
		address, ok := r.DownloadURLs[selectedRoute]
		if !ok {
			if r.UnavailableReason == "" {
				r.UnavailableReason = "selected route has no mirror for this exact native package"
			}
			continue
		}
		r.URL = address
	}
	return rows, nil
}

func nativeHTTPClient(timeout time.Duration) *http.Client {
	return &http.Client{Timeout: timeout, CheckRedirect: func(req *http.Request, via []*http.Request) error {
		if len(via) >= 10 {
			return fmt.Errorf("too many native release redirects")
		}
		_, err := nativeHTTPS(req.URL.String())
		return err
	}}
}

// NativeStageOptions supplies the persisted base cache and running version.
// A missing/mismatched base never creates a delta-ready state.
type NativeStageOptions struct {
	CurrentVersion string
	CacheDir       string
	RequireRuntime bool
	DestDir        string
	OnProgress     func(int64, int64)
	// OnPhase runs on the staging goroutine. No ready phase is published after
	// cancellation or before size/hash/blockmap/signature checks complete.
	OnPhase func(StagePhase)
}

func StageNativeRelease(ctx context.Context, release NativeRelease, requireRuntime bool, destDir string, onProgress func(int64, int64)) (ApplyResult, error) {
	return StageNativeReleaseWithOptions(ctx, release, NativeStageOptions{RequireRuntime: requireRuntime, DestDir: destDir, OnProgress: onProgress})
}

func StageNativeReleaseWithOptions(ctx context.Context, release NativeRelease, opt NativeStageOptions) (ApplyResult, error) {
	if release.UnavailableReason != "" {
		return ApplyResult{}, fmt.Errorf("%s", release.UnavailableReason)
	}
	if err := release.Validate(runtime.GOOS, runtime.GOARCH, opt.RequireRuntime); err != nil {
		return ApplyResult{}, err
	}
	requireSignature := strings.EqualFold(os.Getenv("WHALEISLE_UPDATE_REQUIRE_SIGNATURE"), "true") || os.Getenv("WHALEISLE_UPDATE_REQUIRE_SIGNATURE") == "1"
	publisher := strings.TrimSpace(os.Getenv("WHALEISLE_UPDATE_PUBLISHER"))
	client := nativeHTTPClient(30 * time.Minute)
	var result ApplyResult
	var err error
	if release.IsDelta() {
		reportStagePhase(ctx, opt.OnPhase, StageVerifying)
		base, baseErr := nativeCachedBase(release, opt.CurrentVersion, opt.CacheDir, true)
		if baseErr != nil {
			return ApplyResult{}, fmt.Errorf("native delta unavailable: %w; select the full package", baseErr)
		}
		result, err = ApplyBlockPatchOpts(ctx, PatchOptions{OldPath: base, Feed: release.Feed, BlockMap: release.Delta.BlockMap, DestDir: opt.DestDir,
			HTTPClient: client, RequireSignature: requireSignature, ExpectedPublisher: publisher, MaxBytes: NativeMaxInstallerBytes, OnProgress: opt.OnProgress, OnPhase: opt.OnPhase})
	} else {
		result, err = DownloadAndStageOpts(ctx, StageOptions{ReleaseURL: release.URL, DestDir: opt.DestDir, HTTPClient: client,
			ExpectSHA256: release.SHA256, ExpectSize: release.Size, MaxBytes: NativeMaxInstallerBytes, OnProgress: opt.OnProgress, OnPhase: opt.OnPhase, RequireSignature: requireSignature, ExpectedPublisher: publisher})
	}
	if err != nil {
		return ApplyResult{}, err
	}
	if err = ctx.Err(); err != nil {
		_ = os.Remove(result.Path)
		return ApplyResult{}, err
	}
	if opt.CacheDir != "" {
		if cacheErr := cacheNativeInstaller(ctx, release, result, opt.CacheDir); cacheErr != nil {
			result.Message += "; native base cache unavailable: " + cacheErr.Error() + "; future updates can use a full package"
		}
	}
	reportStagePhase(ctx, opt.OnPhase, StageReady)
	return result, nil
}

// PrepareNativeReleases annotates actual delta eligibility after reading a
// catalog. The complete cached base hash is checked again when staging begins.
func PrepareNativeReleases(rows []NativeRelease, currentVersion, cacheDir string) []NativeRelease {
	out := append([]NativeRelease(nil), rows...)
	for i := range out {
		r := &out[i]
		if !r.IsDelta() || r.UnavailableReason != "" {
			continue
		}
		base, err := nativeCachedBase(*r, currentVersion, cacheDir, false)
		if err != nil {
			r.UnavailableReason = err.Error() + "; use the full package"
		} else {
			r.BasePath = base
		}
	}
	return out
}

func SelectNativeRelease(rows []NativeRelease, selection, goos, arch string, requireRuntime bool) (NativeRelease, error) {
	for _, r := range rows {
		if selection != "" && selection != r.PackageKey() && selection != r.Tag {
			continue
		}
		if selection == "" && r.Prerelease {
			continue
		}
		if r.UnavailableReason != "" || r.Validate(goos, arch, requireRuntime) != nil {
			continue
		}
		return r, nil
	}
	return NativeRelease{}, fmt.Errorf("no compatible native package for this selection; a full runtime package is required when Host is missing")
}

func NativeInstallerCacheDir(updatesDir string) string {
	return filepath.Join(updatesDir, "native-cache")
}
