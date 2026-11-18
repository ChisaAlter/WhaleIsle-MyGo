package pet

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"

	"whaleisle/internal/paths"
)

const anime4kFallbackScript = "// optional anime4k.js sidecar unavailable; falling back to native resolution\n"

type anime4kProvenance struct {
	Files []struct {
		Path     string `json:"path"`
		Size     int64  `json:"size"`
		SHA256   string `json:"sha256"`
		LFSize   int64  `json:"lfSize"`
		LFSHA256 string `json:"lfSha256"`
	} `json:"files"`
}

func expectedAnime4KIdentity() (size int64, digest string, ok bool) {
	raw, err := assets.ReadFile("assets/anime4k-provenance.json")
	if err != nil {
		return 0, "", false
	}
	var p anime4kProvenance
	if json.Unmarshal(raw, &p) != nil {
		return 0, "", false
	}
	for _, f := range p.Files {
		if f.Path == "anime4k.js" && f.Size > 0 && f.SHA256 != "" {
			return f.Size, f.SHA256, true
		}
	}
	return 0, "", false
}

func expectedAnime4KLFIdentity() (lfSize int64, lfDigest string, ok bool) {
	raw, err := assets.ReadFile("assets/anime4k-provenance.json")
	if err != nil {
		return 0, "", false
	}
	var p anime4kProvenance
	if json.Unmarshal(raw, &p) != nil {
		return 0, "", false
	}
	for _, f := range p.Files {
		if f.Path == "anime4k.js" && f.LFSize > 0 && len(f.LFSHA256) == 64 {
			return f.LFSize, f.LFSHA256, true
		}
	}
	return 0, "", false
}

// loadOptionalAnime4K resolves the large optional WebGL shader script from disk
// instead of embedding ~9 MiB into the binary. Priority:
//  1. Configured pet render root / WHALEISLE_PET_RENDER_ASSETS
//  2. User data directory (<userData>/pet-render-assets)
//  3. Executable-adjacent pet-render-assets or assets directory
//  4. Development workspace directories (runtime/pet-render-assets, internal/desktop/pet/assets)
//
// Any candidate bytes are bounded and SHA-256 verified against the embedded
// provenance manifest. Missing or mismatched files degrade cleanly to a valid
// no-op JS stub so pet-super-resolution falls back to raw rendering.
func loadOptionalAnime4K() []byte {
	size, wantSHA, ok := expectedAnime4KIdentity()
	lfSize, lfSHA, _ := expectedAnime4KLFIdentity()
	if !ok {
		return []byte(anime4kFallbackScript)
	}
	for _, c := range anime4KCandidateLocations() {
		if data, verified := readVerifiedAnime4K(c.dir, c.file, size, wantSHA, lfSize, lfSHA); verified {
			return data
		}
	}
	return []byte(anime4kFallbackScript)
}

func (p *Companion) loadAnime4K() []byte {
	if p == nil {
		return loadOptionalAnime4K()
	}
	size, wantSHA, ok := expectedAnime4KIdentity()
	lfSize, lfSHA, _ := expectedAnime4KLFIdentity()
	if !ok {
		return []byte(anime4kFallbackScript)
	}
	var dirs []string
	if root := petRenderRoot(); root != "" {
		dirs = append(dirs, root)
	}
	if env := strings.TrimSpace(os.Getenv("WHALEISLE_PET_RENDER_ASSETS")); env != "" && filepath.IsAbs(env) {
		dirs = append(dirs, filepath.Clean(env))
	}
	if active := p.resolveActiveRenderRoot(); active != "" {
		dirs = append(dirs, filepath.Clean(active))
	}
	if userRoot := p.userRenderRoot(); userRoot != "" {
		dirs = append(dirs, filepath.Clean(userRoot))
	}
	for _, dir := range dirs {
		if data, verified := readVerifiedAnime4K(dir, "anime4k.js", size, wantSHA, lfSize, lfSHA); verified {
			return data
		}
	}
	return loadOptionalAnime4K()
}

type fileLocation struct {
	dir  string
	file string
}

func anime4KCandidateLocations() []fileLocation {
	var out []fileLocation
	seen := map[string]struct{}{}
	add := func(dir, file string) {
		dir = strings.TrimSpace(dir)
		if dir == "" {
			return
		}
		clean := filepath.Clean(dir)
		key := strings.ToLower(filepath.Join(clean, file))
		if _, exists := seen[key]; exists {
			return
		}
		seen[key] = struct{}{}
		out = append(out, fileLocation{dir: clean, file: file})
	}

	if root := petRenderRoot(); root != "" {
		add(root, "anime4k.js")
	}
	if env := strings.TrimSpace(os.Getenv("WHALEISLE_PET_RENDER_ASSETS")); env != "" && filepath.IsAbs(env) {
		add(env, "anime4k.js")
	}
	if userData := anime4KUserDataDir(); userData != "" {
		add(filepath.Join(userData, "pet-render-assets"), "anime4k.js")
	}
	if exe, err := os.Executable(); err == nil && exe != "" {
		exeDir := filepath.Dir(exe)
		add(filepath.Join(exeDir, "pet-render-assets"), "anime4k.js")
		add(filepath.Join(exeDir, "assets"), "anime4k.js")
	}
	if wd, err := os.Getwd(); err == nil && wd != "" {
		cur := filepath.Clean(wd)
		for i := 0; i < 4; i++ {
			add(filepath.Join(cur, "runtime", "pet-render-assets"), "anime4k.js")
			add(filepath.Join(cur, "internal", "desktop", "pet", "assets"), "anime4k.js")
			add(filepath.Join(cur, "assets"), "anime4k.js")
			parent := filepath.Dir(cur)
			if parent == cur {
				break
			}
			cur = parent
		}
	}
	return out
}

func anime4KUserDataDir() string {
	return paths.UserDataDir()
}

func readVerifiedAnime4K(dir, name string, wantSize int64, wantSHA string, lfSize int64, lfSHA string) ([]byte, bool) {
	root, err := os.OpenRoot(dir)
	if err != nil {
		return nil, false
	}
	defer root.Close()
	maxSize := wantSize
	if lfSize > maxSize {
		maxSize = lfSize
	}
	data, err := limited(root, name, maxSize)
	if err != nil {
		return nil, false
	}
	n := int64(len(data))
	sum := sha256.Sum256(data)
	digest := hex.EncodeToString(sum[:])
	if (n == wantSize && digest == wantSHA) || (lfSize > 0 && len(lfSHA) == 64 && n == lfSize && digest == lfSHA) {
		return data, true
	}
	return nil, false
}
