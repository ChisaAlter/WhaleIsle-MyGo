package pet

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync/atomic"
	"whaleisle/internal/store/storelocale"
)

// The manifest is compiled into the executable. Neither directory entries nor
// a sidecar manifest can add routes or executable scripts to the pet origin.
type renderAsset struct {
	Path   string `json:"path"`
	Size   int64  `json:"size"`
	SHA256 string `json:"sha256"`
}

func loadRenderAssets(dir string) (map[string][]byte, error) {
	if dir == "" {
		return nil, errors.New(storelocale.T(storelocale.KeyDesktopPetAssetsMissing))
	}
	manifest, err := assets.ReadFile("assets/render-manifest.json")
	if err != nil {
		return nil, err
	}
	var list struct {
		Files []renderAsset `json:"files"`
	}
	if err = json.Unmarshal(manifest, &list); err != nil {
		return nil, err
	}
	root, err := os.OpenRoot(dir)
	if err != nil {
		return nil, errors.New(storelocale.T(storelocale.KeyDesktopPetAssetsMissing))
	}
	defer root.Close()
	result := make(map[string][]byte)
	for _, file := range list.Files {
		data, readErr := limited(root, file.Path, file.Size)
		if readErr != nil || int64(len(data)) != file.Size {
			return nil, fmt.Errorf("THA4 resource missing or changed: %s", file.Path)
		}
		sum := sha256.Sum256(data)
		if hex.EncodeToString(sum[:]) != file.SHA256 {
			return nil, fmt.Errorf("THA4 resource checksum mismatch: %s", file.Path)
		}
		// Licenses are validated for packaging but need no public route.
		if file.Path != "LICENSE" && file.Path != "ort/LICENSE" {
			result["/render/"+file.Path] = data
		}
	}
	return result, nil
}

// The product uses the adjacent verified sidecar. Isolated fixtures can still
// inject their staged directory through petRenderRoot. The startup override is
// atomic so configuring it cannot race a resource request.
var petRenderRoot = defaultRenderRoot
var configuredRenderRoot atomic.Pointer[string]
var explicitDevRenderRoot atomic.Bool

// ConfigureRenderAssets is a startup-only source selection. Packaged builds
// ignore explicitRoot; development builds accept one absolute directory. An
// invalid explicit value selects no source, never the executable's neighbour or
// the working directory. All selected bytes still pass loadRenderAssets.
func ConfigureRenderAssets(development bool, explicitRoot string) string {
	exe, _ := os.Executable()
	selected := selectRenderRoot(development, explicitRoot, exe)
	explicitDevRenderRoot.Store(development && strings.TrimSpace(explicitRoot) != "")
	configuredRenderRoot.Store(&selected)
	return selected
}

func (p *Companion) userDataDir() string {
	if p != nil && p.Store != nil {
		if path := strings.TrimSpace(p.Store.Path()); path != "" {
			return filepath.Dir(path)
		}
	}
	return anime4KUserDataDir()
}

func (p *Companion) userRenderRoot() string {
	dir := p.userDataDir()
	if dir == "" || !filepath.IsAbs(dir) {
		return ""
	}
	return filepath.Join(filepath.Clean(dir), "pet-render-assets")
}

// resolveActiveRenderRoot keeps adjacent <exeDir>/pet-render-assets and any
// explicit dev/test override authoritative, falling back to <UserDataDir>/pet-render-assets
// only when the adjacent packaged directory does not exist (Decision 3A on-demand download).
func (p *Companion) resolveActiveRenderRoot() string {
	primary := petRenderRoot()
	if primary == "" || explicitDevRenderRoot.Load() || primary != defaultRenderRoot() {
		return primary
	}
	if _, err := os.Lstat(primary); err == nil || !errors.Is(err, os.ErrNotExist) {
		return primary
	}
	if userRoot := p.userRenderRoot(); userRoot != "" {
		if st, err := os.Lstat(userRoot); err == nil && st.IsDir() && st.Mode()&os.ModeSymlink == 0 {
			return userRoot
		}
	}
	return primary
}

func selectRenderRoot(development bool, explicitRoot, exe string) string {
	if development && explicitRoot != "" {
		if !filepath.IsAbs(explicitRoot) {
			return ""
		}
		return filepath.Clean(explicitRoot)
	}
	if exe == "" {
		return ""
	}
	return filepath.Join(filepath.Dir(exe), "pet-render-assets")
}

func defaultRenderRoot() string {
	if selected := configuredRenderRoot.Load(); selected != nil {
		return *selected
	}
	exe, _ := os.Executable()
	return selectRenderRoot(false, "", exe)
}
func renderContentType(path string) string {
	switch filepath.Ext(path) {
	case ".js", ".mjs":
		return "text/javascript; charset=utf-8"
	case ".wasm":
		return "application/wasm"
	case ".png":
		return "image/png"
	default:
		return "application/octet-stream"
	}
}
