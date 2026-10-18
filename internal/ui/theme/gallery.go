package theme

import (
	"os"
	"path/filepath"
	"strings"
)

// WallpaperGalleryDirs returns candidate directories that may hold wallpapers
// (user data wallpapers/, optional resources/wallpapers next to the exe).
func WallpaperGalleryDirs(userData, exeDir string) []string {
	var dirs []string
	if userData = strings.TrimSpace(userData); userData != "" {
		dirs = append(dirs, filepath.Join(userData, "wallpapers"))
	}
	if exeDir = strings.TrimSpace(exeDir); exeDir != "" {
		dirs = append(dirs, filepath.Join(exeDir, "resources", "wallpapers"))
		dirs = append(dirs, filepath.Join(exeDir, "wallpapers"))
	}
	return dirs
}

// ListWallpapers returns image paths under dirs (non-recursive, common raster
// extensions). Missing directories are skipped.
func ListWallpapers(dirs ...string) []string {
	var out []string
	seen := map[string]struct{}{}
	for _, dir := range dirs {
		dir = strings.TrimSpace(dir)
		if dir == "" {
			continue
		}
		ents, err := os.ReadDir(dir)
		if err != nil {
			continue
		}
		for _, e := range ents {
			if e.IsDir() {
				continue
			}
			name := e.Name()
			ext := strings.ToLower(filepath.Ext(name))
			switch ext {
			case ".png", ".jpg", ".jpeg", ".webp", ".bmp", ".gif":
			default:
				continue
			}
			full := filepath.Join(dir, name)
			if _, ok := seen[full]; ok {
				continue
			}
			seen[full] = struct{}{}
			out = append(out, full)
		}
	}
	return out
}
