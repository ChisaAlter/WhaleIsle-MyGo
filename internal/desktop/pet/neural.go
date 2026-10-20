package pet

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
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
func defaultRenderRoot() string {
	exe, err := os.Executable()
	if err != nil {
		return ""
	}
	return filepath.Join(filepath.Dir(exe), "pet-render-assets")
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
