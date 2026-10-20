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
	"errors"
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
	"whaleisle/internal/store/storelocale"
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
		return HostBundle{}, errors.New(storelocale.T(storelocale.KeyStoreComponentsHostOfflineBundleInstallationRequiresMa))
	}
	if !componentInstallMu.TryLock() {
		return HostBundle{}, errors.New(storelocale.T(storelocale.KeyStoreComponentsAComponentInstallOrSwitchIsAlreadyRun))
	}
	defer componentInstallMu.Unlock()
	if !strings.HasSuffix(strings.ToLower(archive), ".tar.gz") {
		return HostBundle{}, errors.New(storelocale.T(storelocale.KeyStoreComponentsChooseTheWhaleisleHostTarGzOfflineBun))
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
		return HostBundle{}, errors.New(storelocale.T(storelocale.KeyStoreComponentsHostOfflineBundleSelectionRequiresMacos))
	}
	versions, err := filepath.Abs(HostBundleVersionsDir(userData))
	if err != nil {
		return HostBundle{}, err
	}
	target, err := filepath.Abs(directory)
	if err != nil || filepath.Dir(target) != versions || strings.HasPrefix(filepath.Base(target), ".") {
		return HostBundle{}, errors.New(storelocale.T(storelocale.KeyStoreComponentsChooseAnInstalledVersionFromTheHostCo))
	}
	if info, err := os.Lstat(target); err != nil || !info.IsDir() || info.Mode()&os.ModeSymlink != 0 {
		return HostBundle{}, errors.New(storelocale.T(storelocale.KeyStoreComponentsTheHostVersionDirectoryIsMissingOrInv))
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
		return errors.New(storelocale.T(storelocale.KeyStoreComponentsWhaleisleVendorPointsToAnExternalHost))
	}
	if !componentInstallMu.TryLock() {
		return errors.New(storelocale.T(storelocale.KeyStoreComponentsAComponentInstallOrSwitchIsAlreadyRun))
	}
	defer componentInstallMu.Unlock()
	current, err := readHostBundle(b.root)
	if err != nil {
		return err
	}
	if current != b.Manifest {
		return errors.New(storelocale.T(storelocale.KeyStoreComponentsTheHostVersionChangedChooseAgain))
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
		return manifest, errors.New(storelocale.T(storelocale.KeyStoreComponentsTheHostOfflineBundleDoesNotMatchTheC))
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
		return errors.New(fmt.Sprintf(storelocale.T(storelocale.KeyStoreComponentsTheHostOfflineBundleIsIncompleteS), host.Reason))
	}
	node := filepath.Join(runtimeDir, "node", "node")
	cpu := macho.CpuArm64
	if runtime.GOARCH == "amd64" {
		cpu = macho.CpuAmd64
	}
	if binary, err := macho.Open(node); err == nil {
		defer binary.Close()
		if binary.Cpu != cpu || binary.Type != macho.TypeExec {
			return errors.New(storelocale.T(storelocale.KeyStoreComponentsTheHostNodeArchitectureDoesNotMatchTh))
		}
	} else {
		binary, err := macho.OpenFat(node)
		if err != nil {
			return errors.New(fmt.Sprintf(storelocale.T(storelocale.KeyStoreComponentsTheHostPackageIsMissingTheNativeMacos), err))
		}
		defer binary.Close()
		matched := false
		for _, arch := range binary.Arches {
			matched = matched || (arch.Cpu == cpu && arch.Type == macho.TypeExec)
		}
		if !matched {
			return errors.New(storelocale.T(storelocale.KeyStoreComponentsTheHostNodeArchitectureDoesNotMatchTh))
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
		return errors.New(storelocale.T(storelocale.KeyStoreComponentsTheHostPackageNodeDoesNotMatchThePor))
	}
	sum, err := hostFileSHA256(node)
	if err != nil {
		return err
	}
	if sum != build.NodeSHA256 {
		return errors.New(storelocale.T(storelocale.KeyStoreComponentsTheHostNodeDoesNotMatchTheNativeModu))
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
		return errors.New(storelocale.T(storelocale.KeyStoreComponentsThePortableRuntimeNodeDoesNotMatchThe))
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
			return errors.New(fmt.Sprintf(storelocale.T(storelocale.KeyStoreComponentsTheHostPackageContainsAnUnknownOrOut), header.Name))
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
			return errors.New(fmt.Sprintf(storelocale.T(storelocale.KeyStoreComponentsTheHostPackageContainsAnUnsupportedFil), header.Name))
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
			return errors.New(fmt.Sprintf(storelocale.T(storelocale.KeyStoreComponentsAHostPackageLinkPointsOutsideTheRunti), header.Name))
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
			return errors.New(fmt.Sprintf(storelocale.T(storelocale.KeyStoreComponentsTheHostRuntimeLinkIsUnavailableSW), header.Name, err))
		}
	}
	return nil
}

func hostBundleRow(userData string) Row {
	row := Row{ID: HostComponentID, Name: "Host runtime", Kind: "runtime", Source: "local",
		Description: storelocale.T(storelocale.KeyStoreComponentsNativeMacosHostWithTheMatchingNodeOff), State: "available"}
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
			row.Message = fmt.Sprintf(storelocale.T(storelocale.KeyStoreComponentsDHostVersionsInstalledChooseAVersionT), count)
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
		row.Message = storelocale.T(storelocale.KeyStoreComponentsWhaleisleVendorCurrentlyPrefersTheExter)
	}
	return row
}
