package components

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"regexp"
	"runtime"
	"strings"
	"sync"
	"time"

	"whaleisle/internal/desktop/update"
)

const bridgeAsset = "WhaleBridge-win32-x64.exe"
const bridgeManifest = "WhaleBridge-component.json"
const bridgeGitHubReleases = "https://github.com/ChisaAlter/WhaleBridge/releases"

var componentInstallMu sync.Mutex
var bridgeVersion = regexp.MustCompile(`^\d+\.\d+\.\d+$`)
var bridgeHash = regexp.MustCompile(`^[a-f0-9]{64}$`)

type bridgeSpec struct {
	Asset  string `json:"asset"`
	SHA256 string `json:"sha256"`
	Size   int64  `json:"size"`
}
type bridgeReleaseManifest struct {
	ID          string                `json:"id"`
	Name        string                `json:"name"`
	Version     string                `json:"version"`
	Description string                `json:"description"`
	License     string                `json:"license"`
	Platforms   map[string]bridgeSpec `json:"platforms"`
}
type bridgeReleaseAsset struct {
	Name        string `json:"name"`
	URL         string `json:"browser_download_url"`
	DownloadURL string `json:"download_url"`
	Size        int64  `json:"size"`
}

func (a bridgeReleaseAsset) address() string {
	if a.URL != "" {
		return a.URL
	}
	return a.DownloadURL
}

type bridgeRelease struct {
	Tag        string               `json:"tag_name"`
	Draft      bool                 `json:"draft"`
	Prerelease bool                 `json:"prerelease"`
	Assets     []bridgeReleaseAsset `json:"assets"`
}

// InstallWhaleBridge installs verified bytes only. It never executes the component,
// edits credentials, or replaces an existing version. Starting remains explicit.
func InstallWhaleBridge(ctx context.Context, userData, route string) error {
	if runtime.GOOS != "windows" || runtime.GOARCH != "amd64" {
		return fmt.Errorf("鲸桥首版仅支持 Windows x64")
	}
	endpoint, err := bridgeReleaseEndpoint(route)
	if err != nil {
		return err
	}
	return installBridge(ctx, userData, endpoint, &http.Client{Timeout: 5 * time.Minute}, false)
}

func bridgeReleaseEndpoint(route string) (string, error) {
	endpoint := bridgeGitHubReleases + "/latest/download/" + bridgeManifest
	switch strings.ToLower(strings.TrimSpace(route)) {
	case "", "github":
	case "cnb", "gitee":
		endpoint = "https://cnb.cool/ayasealter/WhaleIsle/-/releases?page_size=100"
	default:
		return "", fmt.Errorf("未知组件下载线路")
	}
	return endpoint, nil
}

func installBridge(ctx context.Context, userData, endpoint string, client *http.Client, fixture bool) error {
	if !componentInstallMu.TryLock() {
		return fmt.Errorf("组件安装正在进行")
	}
	defer componentInstallMu.Unlock()
	root := filepath.Join(userData, "components", "whalebridge")
	// Do not use this first-install path to replace a Host-dependent running component.
	// Existing installs must go through the separately protected lifecycle workflow.
	for _, name := range []string{"installed.json", filepath.Join("data", "state.json")} {
		if _, err := os.Lstat(filepath.Join(root, name)); !os.IsNotExist(err) {
			return fmt.Errorf("鲸桥已有安装或运行记录，首次安装不能覆盖")
		}
	}
	entries, err := os.ReadDir(filepath.Join(root, "versions"))
	if err != nil && !os.IsNotExist(err) {
		return err
	}
	for _, e := range entries {
		if !strings.HasPrefix(e.Name(), ".") {
			return fmt.Errorf("鲸桥已有版本，首次安装不能覆盖")
		}
	}
	m, binaryAsset, err := resolveBridgeRelease(ctx, endpoint, client, fixture)
	if err != nil {
		return err
	}
	target, err := stageBridgeRelease(ctx, root, m, binaryAsset, client, fixture)
	if err != nil {
		return err
	}

	record, _ := json.Marshal(installedFile{Version: m.Version})
	if err := bridgeWriteNewRecord(root, record); err != nil {
		_ = os.RemoveAll(target)
		return err
	}
	return nil
}

func resolveBridgeRelease(ctx context.Context, endpoint string, client *http.Client, fixture bool) (m bridgeReleaseManifest, binaryAsset bridgeReleaseAsset, err error) {
	latestSuffix := "/latest/download/" + bridgeManifest
	if strings.HasSuffix(endpoint, latestSuffix) {
		// Fetch latest metadata once, then pin payload to that validated version.
		// No GitHub REST quota is used and a moving latest cannot mix releases.
		if err := bridgeJSON(ctx, client, endpoint, &m, fixture); err != nil {
			return m, binaryAsset, err
		}
		if err := validateBridgeManifest(m); err != nil {
			return m, binaryAsset, err
		}
		binaryAsset = bridgeReleaseAsset{Name: bridgeAsset, URL: strings.TrimSuffix(endpoint, latestSuffix) + "/download/whalebridge-v" + m.Version + "/" + bridgeAsset, Size: m.Platforms["win32-x64"].Size}
	} else {
		var releases []bridgeRelease
		if err := bridgeJSON(ctx, client, endpoint, &releases, fixture); err != nil {
			return m, binaryAsset, err
		}
		var selected *bridgeRelease
		for i := range releases {
			r := &releases[i]
			v := strings.TrimPrefix(r.Tag, "whalebridge-v")
			if r.Draft || r.Prerelease || !strings.HasPrefix(r.Tag, "whalebridge-v") || !bridgeVersion.MatchString(v) {
				continue
			}
			if selected == nil || compareVersions(v, strings.TrimPrefix(selected.Tag, "whalebridge-v")) > 0 {
				selected = r
			}
		}
		if selected == nil {
			return m, binaryAsset, fmt.Errorf("鲸桥组件尚未发布到当前线路")
		}
		var manifestAsset bridgeReleaseAsset
		for _, a := range selected.Assets {
			switch a.Name {
			case bridgeManifest:
				manifestAsset = a
			case bridgeAsset:
				binaryAsset = a
			}
		}
		if manifestAsset.address() == "" || binaryAsset.address() == "" {
			return m, binaryAsset, fmt.Errorf("鲸桥发布缺少组件文件")
		}
		if err := bridgeJSON(ctx, client, manifestAsset.address(), &m, fixture); err != nil {
			return m, binaryAsset, err
		}
		spec := m.Platforms["win32-x64"]
		if m.ID != "whalebridge" || !bridgeVersion.MatchString(m.Version) || selected.Tag != "whalebridge-v"+m.Version || spec.Asset != bridgeAsset || !bridgeHash.MatchString(spec.SHA256) || spec.Size <= 0 || spec.Size > 200<<20 || binaryAsset.Size != spec.Size || !strings.Contains(m.License, "MIT License") {
			return m, binaryAsset, fmt.Errorf("鲸桥发布清单或资产不一致")
		}
	}
	return m, binaryAsset, nil
}

func stageBridgeRelease(ctx context.Context, root string, m bridgeReleaseManifest, binaryAsset bridgeReleaseAsset, client *http.Client, fixture bool) (string, error) {
	spec := m.Platforms["win32-x64"]
	if err := bridgeURL(binaryAsset.address(), fixture); err != nil {
		return "", err
	}
	target := filepath.Join(root, "versions", m.Version)
	if _, err := os.Lstat(target); err == nil {
		got, err := verifyBridgeVersion(root, m.Version)
		if err != nil {
			return "", err
		}
		if got.Platforms["win32-x64"] != spec {
			return "", fmt.Errorf("目标版本已存在且清单不一致，拒绝覆盖")
		}
		return target, nil
	} else if !os.IsNotExist(err) {
		return "", err
	}
	versions := filepath.Join(root, "versions")
	if err := os.MkdirAll(versions, 0755); err != nil {
		return "", err
	}
	staging, err := os.MkdirTemp(versions, ".staging-")
	if err != nil {
		return "", err
	}
	defer os.RemoveAll(staging)
	staged, err := update.DownloadAndStageOpts(ctx, update.StageOptions{ReleaseURL: binaryAsset.address(), DestDir: staging, HTTPClient: client, ExpectSize: spec.Size, ExpectSHA256: spec.SHA256})
	if err != nil {
		return "", err
	}
	fixed := filepath.Join(staging, bridgeAsset)
	if staged.Path != fixed {
		if err := os.Rename(staged.Path, fixed); err != nil {
			return "", err
		}
	}
	manifestBytes, _ := json.MarshalIndent(m, "", "  ")
	if err := os.WriteFile(filepath.Join(staging, bridgeManifest), manifestBytes, 0600); err != nil {
		return "", err
	}
	if err := os.WriteFile(filepath.Join(staging, "LICENSE"), []byte(m.License), 0600); err != nil {
		return "", err
	}
	if _, err := os.Lstat(target); !os.IsNotExist(err) {
		return "", fmt.Errorf("目标版本已存在，拒绝覆盖")
	}
	if err := os.Rename(staging, target); err != nil {
		return "", err
	}
	return target, nil
}

func bridgeWriteNewRecord(root string, b []byte) error {
	f, err := os.CreateTemp(root, ".installed-")
	if err != nil {
		return err
	}
	name := f.Name()
	defer os.Remove(name)
	if _, err := f.Write(b); err != nil {
		f.Close()
		return err
	}
	if err := f.Sync(); err != nil {
		f.Close()
		return err
	}
	if err := f.Close(); err != nil {
		return err
	}
	if _, err := os.Lstat(filepath.Join(root, "installed.json")); !os.IsNotExist(err) {
		return fmt.Errorf("安装记录已存在，拒绝覆盖")
	}
	return os.Rename(name, filepath.Join(root, "installed.json"))
}
func bridgeURL(raw string, fixture bool) error {
	u, err := url.Parse(raw)
	if err != nil {
		return err
	}
	if fixture && u.Scheme == "http" && (u.Hostname() == "127.0.0.1" || u.Hostname() == "localhost") {
		return nil
	}
	if u.Scheme != "https" || u.User != nil {
		return fmt.Errorf("组件地址必须使用官方 HTTPS 发布源")
	}
	switch u.Hostname() {
	case "api.github.com", "github.com", "cnb.cool":
		return nil
	}
	return fmt.Errorf("组件发布地址不是受支持的官方源")
}
func bridgeJSON(ctx context.Context, client *http.Client, raw string, out any, fixture bool) error {
	if err := bridgeURL(raw, fixture); err != nil {
		return err
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, raw, nil)
	if err != nil {
		return err
	}
	req.Header.Set("Accept", "application/json")
	if u, err := url.Parse(raw); err == nil && u.Hostname() == "cnb.cool" {
		req.Header.Set("Accept", "application/vnd.cnb.api+json")
	}
	resp, err := client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("组件元数据 HTTP %d", resp.StatusCode)
	}
	return json.NewDecoder(io.LimitReader(resp.Body, 2<<20)).Decode(out)
}

func validateBridgeManifest(m bridgeReleaseManifest) error {
	spec := m.Platforms["win32-x64"]
	if m.ID != "whalebridge" || !bridgeVersion.MatchString(m.Version) || spec.Asset != bridgeAsset || !bridgeHash.MatchString(spec.SHA256) || spec.Size <= 0 || spec.Size > 200<<20 || !strings.Contains(m.License, "MIT License") {
		return fmt.Errorf("鲸桥发布清单无效")
	}
	return nil
}
