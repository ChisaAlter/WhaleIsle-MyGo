package components

// 功能优先；全部功能完成前不运行测试、不做独立审查、不新增防御性扩展

import (
	"archive/tar"
	"compress/gzip"
	"context"
	"crypto/sha256"
	"debug/macho"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"regexp"
	"runtime"
	"strings"

	"whaleisle/internal/desktop/phoneremote"
	"whaleisle/internal/harness"
	"whaleisle/internal/paths"
)

const HostComponentID = "whaleisle-host"
const hostBundleManifest = "WhaleIsle-Host-bundle.json"

var hostBundleVersion = regexp.MustCompile(`^[0-9]+\.[0-9]+\.[0-9]+(?:[-+][A-Za-z0-9.-]+)?$`)

// HostBundleManifest is emitted by scripts/package-macos.mjs archive.
// It describes the native runtime/ tree, not an Electron resource archive.
type HostBundleManifest struct {
	SchemaVersion      int    `json:"schemaVersion"`
	Application        string `json:"application"`
	Version            string `json:"version"`
	Platform           string `json:"platform"`
	Arch               string `json:"arch"`
	RuntimeDirectory   string `json:"runtimeDirectory"`
	HostSourceCommit   string `json:"hostSourceCommit"`
	HostReadinessGuard string `json:"hostReadinessGuard"`
	NodeVersion        string `json:"nodeVersion"`
	NodeABI            string `json:"nodeABI"`
}

type HostBundle struct {
	Manifest  HostBundleManifest
	Directory string
	root      string
}

func HostBundleSupported() bool {
	return runtime.GOOS == "darwin" && (runtime.GOARCH == "amd64" || runtime.GOARCH == "arm64")
}

func HostBundleVersionsDir(userData string) string {
	return filepath.Join(paths.HostComponentDir(userData), "versions")
}

// InstallHostBundle only stages and installs files. It never starts Node,
// modifies the app bundle, or selects a runtime behind a running Host.
func InstallHostBundle(ctx context.Context, userData, archive string) (HostBundle, error) {
	if !HostBundleSupported() {
		return HostBundle{}, fmt.Errorf("Host 离线包安装仅支持 macOS")
	}
	if !componentInstallMu.TryLock() {
		return HostBundle{}, fmt.Errorf("组件安装或切换正在进行")
	}
	defer componentInstallMu.Unlock()
	if !strings.HasSuffix(strings.ToLower(archive), ".tar.gz") {
		return HostBundle{}, fmt.Errorf("请选择 WhaleIsle-Host 的 .tar.gz 离线包")
	}
	sum, err := hostFileSHA256(archive)
	if err != nil {
		return HostBundle{}, err
	}
	versions := HostBundleVersionsDir(userData)
	if err := os.MkdirAll(versions, 0755); err != nil {
		return HostBundle{}, err
	}
	stage, err := os.MkdirTemp(versions, ".staging-")
	if err != nil {
		return HostBundle{}, err
	}
	defer os.RemoveAll(stage)
	if err := extractHostBundle(ctx, archive, stage); err != nil {
		return HostBundle{}, err
	}
	manifest, err := readHostBundle(stage)
	if err != nil {
		return HostBundle{}, err
	}
	if err := verifyHostBundle(stage, manifest); err != nil {
		return HostBundle{}, err
	}
	if err := ctx.Err(); err != nil {
		return HostBundle{}, err
	}
	directory := manifest.Version + "-" + sum[:12]
	target := filepath.Join(versions, directory)
	if _, err := os.Lstat(target); err == nil {
		return LoadHostBundle(userData, target)
	} else if !os.IsNotExist(err) {
		return HostBundle{}, err
	}
	if err := os.Rename(stage, target); err != nil {
		return HostBundle{}, err
	}
	return HostBundle{Manifest: manifest, Directory: directory, root: target}, nil
}

// LoadHostBundle accepts a version selected in the native directory picker and
// reuses the Host provenance/readiness gate before offering it for activation.
func LoadHostBundle(userData, directory string) (HostBundle, error) {
	if !HostBundleSupported() {
		return HostBundle{}, fmt.Errorf("Host 离线包选择仅支持 macOS")
	}
	versions, err := filepath.Abs(HostBundleVersionsDir(userData))
	if err != nil {
		return HostBundle{}, err
	}
	target, err := filepath.Abs(directory)
	if err != nil || filepath.Dir(target) != versions || strings.HasPrefix(filepath.Base(target), ".") {
		return HostBundle{}, fmt.Errorf("请选择 Host 组件 versions 目录中的已安装版本")
	}
	if info, err := os.Lstat(target); err != nil || !info.IsDir() || info.Mode()&os.ModeSymlink != 0 {
		return HostBundle{}, fmt.Errorf("Host 版本目录缺失或无效")
	}
	manifest, err := readHostBundle(target)
	if err != nil {
		return HostBundle{}, err
	}
	if err := verifyHostBundle(target, manifest); err != nil {
		return HostBundle{}, err
	}
	return HostBundle{Manifest: manifest, Directory: filepath.Base(target), root: target}, nil
}

// Select is called only after native confirmation and the existing protected
// Host stop. The former version and all profile data stay in place.
func (b HostBundle) Select(userData string) error {
	if os.Getenv(paths.EnvVendor) != "" {
		return fmt.Errorf("WHALEISLE_VENDOR 正在指定外部 Host；请先移除该覆盖后再选择组件版本")
	}
	if !componentInstallMu.TryLock() {
		return fmt.Errorf("组件安装或切换正在进行")
	}
	defer componentInstallMu.Unlock()
	current, err := readHostBundle(b.root)
	if err != nil {
		return err
	}
	if current != b.Manifest {
		return fmt.Errorf("Host 版本已改变，请重新选择")
	}
	selection := paths.HostRuntimeSelection{Version: b.Manifest.Version, Directory: b.Directory}
	if old, err := paths.ReadHostRuntimeSelection(userData); err == nil {
		selection.Previous = old.Directory
		if old.Directory == selection.Directory {
			selection.Previous = old.Previous
		}
	} else if !os.IsNotExist(err) {
		return err
	}
	data, err := json.Marshal(selection)
	if err != nil {
		return err
	}
	root := paths.HostComponentDir(userData)
	file, err := os.CreateTemp(root, ".installed-")
	if err != nil {
		return err
	}
	name := file.Name()
	defer os.Remove(name)
	if _, err := file.Write(data); err != nil {
		file.Close()
		return err
	}
	if err := file.Sync(); err != nil {
		file.Close()
		return err
	}
	if err := file.Close(); err != nil {
		return err
	}
	return os.Rename(name, filepath.Join(root, "installed.json"))
}

func readHostBundle(root string) (HostBundleManifest, error) {
	var manifest HostBundleManifest
	data, err := os.ReadFile(filepath.Join(root, hostBundleManifest))
	if err != nil {
		return manifest, err
	}
	if err := json.Unmarshal(data, &manifest); err != nil {
		return manifest, err
	}
	if manifest.SchemaVersion != 1 || manifest.Application != "whaleisle-mygo-host" ||
		manifest.Platform != "darwin" || manifest.Arch != runtime.GOARCH ||
		manifest.RuntimeDirectory != "runtime" || !hostBundleVersion.MatchString(manifest.Version) ||
		(manifest.HostSourceCommit != harness.CurrentHostSourceCommit && manifest.HostSourceCommit != harness.CurrentHostE9SourceCommit) ||
		manifest.HostReadinessGuard != harness.CurrentHostReadinessGuardID {
		return manifest, fmt.Errorf("Host 离线包与当前原生应用、架构或 Host 契约不匹配")
	}
	return manifest, nil
}

func verifyHostBundle(root string, manifest HostBundleManifest) error {
	runtimeDir := filepath.Join(root, "runtime")
	vendor := filepath.Join(runtimeDir, "vendor")
	if err := harness.ValidateCurrentHostForLaunch(filepath.Join(vendor, "deepseek-harness")); err != nil {
		return err
	}
	host := harness.ReadCurrentHostCapabilities(vendor, harness.CurrentHostBridges{})
	if !host.ArtifactsReady || !host.ReadinessGuardReady {
		return fmt.Errorf("Host 离线包不完整：%s", host.Reason)
	}
	node := filepath.Join(runtimeDir, "node", "node")
	cpu := macho.CpuArm64
	if runtime.GOARCH == "amd64" {
		cpu = macho.CpuAmd64
	}
	if binary, err := macho.Open(node); err == nil {
		defer binary.Close()
		if binary.Cpu != cpu || binary.Type != macho.TypeExec {
			return fmt.Errorf("Host Node 架构与当前 macOS 不匹配")
		}
	} else {
		binary, err := macho.OpenFat(node)
		if err != nil {
			return fmt.Errorf("Host 包缺少原生 macOS Node：%w", err)
		}
		defer binary.Close()
		matched := false
		for _, arch := range binary.Arches {
			matched = matched || (arch.Cpu == cpu && arch.Type == macho.TypeExec)
		}
		if !matched {
			return fmt.Errorf("Host Node 架构与当前 macOS 不匹配")
		}
	}
	var build struct {
		Platform   string `json:"platform"`
		Arch       string `json:"arch"`
		Node       string `json:"node"`
		ABI        string `json:"abi"`
		NodeSHA256 string `json:"nodeSHA256"`
	}
	data, err := os.ReadFile(filepath.Join(vendor, "dshd-remote", "node-build.json"))
	if err != nil {
		return err
	}
	arch := runtime.GOARCH
	if arch == "amd64" {
		arch = "x64"
	}
	if json.Unmarshal(data, &build) != nil || build.Platform != "darwin" || build.Arch != arch ||
		build.Node == "" || build.Node != manifest.NodeVersion || build.ABI == "" || build.ABI != manifest.NodeABI {
		return fmt.Errorf("Host 包的 Node 与移动运行时构建记录不匹配")
	}
	sum, err := hostFileSHA256(node)
	if err != nil {
		return err
	}
	if sum != build.NodeSHA256 {
		return fmt.Errorf("Host Node 与打包时的原生模块运行时不匹配")
	}
	mobile, err := phoneremote.ResolveResources(filepath.Join(root, ".bundle-check"), vendor)
	if err != nil {
		return err
	}
	mobileSum, err := hostFileSHA256(mobile.Node)
	if err != nil {
		return err
	}
	if mobileSum != build.NodeSHA256 {
		return fmt.Errorf("移动运行时 Node 与 Host 包构建记录不匹配")
	}
	return nil
}

func hostFileSHA256(name string) (string, error) {
	file, err := os.Open(name)
	if err != nil {
		return "", err
	}
	defer file.Close()
	digest := sha256.New()
	if _, err := io.Copy(digest, file); err != nil {
		return "", err
	}
	return hex.EncodeToString(digest.Sum(nil)), nil
}

// Materialize the package's relative links after files, retaining the pnpm
// topology used by package-macos. Archive members stay within the staging tree.
func extractHostBundle(ctx context.Context, archive, destination string) error {
	file, err := os.Open(archive)
	if err != nil {
		return err
	}
	defer file.Close()
	compressed, err := gzip.NewReader(file)
	if err != nil {
		return err
	}
	defer compressed.Close()
	root, err := os.OpenRoot(destination)
	if err != nil {
		return err
	}
	defer root.Close()
	reader := tar.NewReader(compressed)
	var links []*tar.Header
	for {
		if err := ctx.Err(); err != nil {
			return err
		}
		header, err := reader.Next()
		if err == io.EOF {
			break
		}
		if err != nil {
			return err
		}
		name := filepath.Clean(filepath.FromSlash(header.Name))
		if !filepath.IsLocal(name) || (name != "runtime" && name != hostBundleManifest && !strings.HasPrefix(name, "runtime"+string(filepath.Separator))) {
			return fmt.Errorf("Host 包包含未知或越界路径：%s", header.Name)
		}
		target := name
		switch header.Typeflag {
		case tar.TypeDir:
			if err := root.MkdirAll(target, 0755); err != nil {
				return err
			}
		case tar.TypeReg, tar.TypeRegA:
			if err := root.MkdirAll(filepath.Dir(target), 0755); err != nil {
				return err
			}
			out, err := root.OpenFile(target, os.O_WRONLY|os.O_CREATE|os.O_EXCL, os.FileMode(header.Mode)&0777)
			if err != nil {
				return err
			}
			_, copyErr := io.Copy(out, reader)
			closeErr := out.Close()
			if copyErr != nil {
				return copyErr
			}
			if closeErr != nil {
				return closeErr
			}
		case tar.TypeSymlink, tar.TypeLink:
			copy := *header
			links = append(links, &copy)
		default:
			return fmt.Errorf("Host 包包含不支持的文件类型：%s", header.Name)
		}
	}
	// Read through gzip's trailer so a truncated archive cannot be installed.
	if _, err := io.Copy(io.Discard, compressed); err != nil {
		return err
	}
	for _, header := range links {
		name := filepath.Clean(filepath.FromSlash(header.Name))
		link := filepath.FromSlash(header.Linkname)
		resolved := link
		if header.Typeflag == tar.TypeSymlink {
			resolved = filepath.Join(filepath.Dir(name), link)
		}
		if filepath.IsAbs(link) || !filepath.IsLocal(resolved) {
			return fmt.Errorf("Host 包链接超出运行时目录：%s", header.Name)
		}
		target := name
		if err := root.MkdirAll(filepath.Dir(target), 0755); err != nil {
			return err
		}
		if header.Typeflag == tar.TypeLink {
			err = root.Link(resolved, target)
		} else {
			err = root.Symlink(link, target)
		}
		if err != nil {
			return err
		}
	}
	for _, header := range links {
		if _, err := root.Stat(filepath.FromSlash(header.Name)); err != nil {
			return fmt.Errorf("Host 运行时链接不可用：%s: %w", header.Name, err)
		}
	}
	return nil
}

func hostBundleRow(userData string) Row {
	row := Row{ID: HostComponentID, Name: "Host runtime", Kind: "runtime", Source: "local",
		Description: "macOS 原生 Host 与配套 Node 离线运行时", State: "available"}
	if entries, err := os.ReadDir(HostBundleVersionsDir(userData)); err == nil {
		count := 0
		for _, entry := range entries {
			if !entry.IsDir() || strings.HasPrefix(entry.Name(), ".") {
				continue
			}
			if _, err := readHostBundle(filepath.Join(HostBundleVersionsDir(userData), entry.Name())); err == nil {
				count++
			}
		}
		if count > 0 {
			row.Message = fmt.Sprintf("已安装 %d 个 Host 版本；选择版本后可启动。", count)
		}
	}
	selection, err := paths.ReadHostRuntimeSelection(userData)
	if err == nil {
		row.Version = selection.Version
		row.InstalledVersion = selection.Version
		row.State = "stopped"
		row.Message = filepath.Join(HostBundleVersionsDir(userData), selection.Directory)
	} else if !os.IsNotExist(err) {
		row.State = "error"
		row.Message = err.Error()
	}
	if os.Getenv(paths.EnvVendor) != "" {
		row.Message = "WHALEISLE_VENDOR 正在优先使用外部运行时；组件版本尚未生效。"
	}
	return row
}
