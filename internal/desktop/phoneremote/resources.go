package phoneremote

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"regexp"
	"runtime"
	"strings"
)

type mobileResourceLink struct {
	Target string `json:"target"`
	Kind   string `json:"kind"`
}

type mobileResourceManifest struct {
	Format      int                           `json:"format"`
	Profile     string                        `json:"profile"`
	Platform    string                        `json:"platform"`
	Arch        string                        `json:"arch"`
	NodePath    string                        `json:"nodePath"`
	Files       map[string]string             `json:"files"`
	Links       map[string]mobileResourceLink `json:"links"`
	HarnessPeer struct {
		Path      string            `json:"path"`
		Artifacts map[string]string `json:"artifacts"`
	} `json:"harnessPeer"`
}

func mobileInside(root, target string) bool {
	r, err := filepath.Rel(root, target)
	return err == nil && !filepath.IsAbs(r) && r != ".." && !strings.HasPrefix(r, ".."+string(filepath.Separator))
}

func mobileChild(root, relative string) (string, error) {
	if relative == "" || strings.Contains(relative, "\\") || strings.Contains(relative, ":") || filepath.IsAbs(relative) {
		return "", fmt.Errorf("unsafe mobile manifest path: %s", relative)
	}
	for _, part := range strings.Split(relative, "/") {
		if part == "" || part == "." || part == ".." {
			return "", fmt.Errorf("unsafe mobile manifest path: %s", relative)
		}
	}
	result := filepath.Join(root, filepath.FromSlash(relative))
	if !mobileInside(root, result) || result == root {
		return "", fmt.Errorf("mobile manifest path escapes runtime")
	}
	return result, nil
}

func mobileDigest(filename string) (string, error) {
	f, err := os.Open(filename)
	if err != nil {
		return "", err
	}
	defer f.Close()
	h := sha256.New()
	if _, err := io.Copy(h, f); err != nil {
		return "", err
	}
	return hex.EncodeToString(h.Sum(nil)), nil
}

func mobileGraph(root string) (map[string]string, map[string]mobileResourceLink, error) {
	files, links := map[string]string{}, map[string]mobileResourceLink{}
	var walk func(string) error
	walk = func(dir string) error {
		entries, err := os.ReadDir(dir)
		if err != nil {
			return err
		}
		for _, entry := range entries {
			filename := filepath.Join(dir, entry.Name())
			rel, _ := filepath.Rel(root, filename)
			rel = filepath.ToSlash(rel)
			if rel == "staging-manifest.json" || rel == "debug.log" {
				continue
			}
			info, err := os.Lstat(filename)
			if err != nil {
				return err
			}
			resolved := filename
			if info.Mode()&(os.ModeSymlink|os.ModeIrregular) != 0 {
				link, readErr := os.Readlink(filename)
				if readErr != nil {
					return readErr
				}
				if filepath.IsAbs(link) {
					resolved = filepath.Clean(link)
				} else {
					resolved = filepath.Clean(filepath.Join(filepath.Dir(filename), link))
				}
			}
			if !mobileInside(root, resolved) {
				return fmt.Errorf("mobile resource escapes runtime: %s", rel)
			}
			if info.Mode()&(os.ModeSymlink|os.ModeIrregular) != 0 || !strings.EqualFold(resolved, filename) {
				if mobileInside(resolved, filename) {
					return fmt.Errorf("ancestor mobile dependency link: %s", rel)
				}
				target, err := filepath.Rel(root, resolved)
				if err != nil {
					return err
				}
				st, err := os.Stat(filename)
				if err != nil {
					return err
				}
				kind := "file"
				if st.IsDir() {
					kind = "directory"
				}
				links[rel] = mobileResourceLink{Target: filepath.ToSlash(target), Kind: kind}
			} else if info.IsDir() {
				if err := walk(filename); err != nil {
					return err
				}
			} else if info.Mode().IsRegular() {
				digest, err := mobileDigest(filename)
				if err != nil {
					return err
				}
				files[rel] = digest
			} else {
				return fmt.Errorf("unsupported mobile resource: %s", rel)
			}
		}
		return nil
	}
	if err := walk(root); err != nil {
		return nil, nil, err
	}
	return files, links, nil
}

// A development/component tree without a release record keeps its existing
// resolution path. Once a manifest is present, corruption cannot fall back to
// another executable. Native loads were performed by the packaging probe.
func verifyPackagedMobileResources(root string) (string, error) {
	name := filepath.Join(root, "staging-manifest.json")
	raw, err := os.ReadFile(name)
	if os.IsNotExist(err) {
		return "", nil
	}
	if err != nil {
		return "", err
	}
	var head struct {
		Format int `json:"format"`
	}
	if err := json.Unmarshal(raw, &head); err != nil {
		return "", fmt.Errorf("invalid mobile manifest: %w", err)
	}
	if head.Format == 1 || head.Format == 2 {
		return "", nil
	}
	var m mobileResourceManifest
	if err := json.Unmarshal(raw, &m); err != nil || m.Format != 3 || m.Profile != "installed-rc2" || m.Files == nil || m.Links == nil {
		return "", fmt.Errorf("unsupported packaged mobile manifest")
	}
	platform, arch := runtime.GOOS, runtime.GOARCH
	if platform == "windows" {
		platform = "win32"
	}
	if arch == "amd64" {
		arch = "x64"
	}
	if m.Platform != platform || m.Arch != arch {
		return "", fmt.Errorf("mobile runtime target mismatch")
	}
	root, err = filepath.EvalSymlinks(root)
	if err != nil {
		return "", err
	}
	files, links, err := mobileGraph(root)
	if err != nil {
		return "", err
	}
	if len(files) != len(m.Files) || len(links) != len(m.Links) {
		return "", fmt.Errorf("mobile runtime inventory changed")
	}
	for name, expected := range m.Files {
		if _, err := mobileChild(root, name); err != nil {
			return "", err
		}
		if len(expected) != 64 || files[name] != expected {
			return "", fmt.Errorf("mobile runtime changed after build: %s", name)
		}
	}
	for name, expected := range m.Links {
		if _, err := mobileChild(root, name); err != nil {
			return "", err
		}
		if _, err := mobileChild(root, expected.Target); err != nil {
			return "", err
		}
		if links[name] != expected {
			return "", fmt.Errorf("mobile dependency link changed: %s", name)
		}
	}
	var build struct {
		Platform        string            `json:"platform"`
		Arch            string            `json:"arch"`
		Runtime         string            `json:"runtime"`
		Electron        string            `json:"electron"`
		ABI             string            `json:"abi"`
		NodeSHA256      string            `json:"nodeSHA256"`
		NativeArtifacts map[string]string `json:"nativeArtifacts"`
	}
	raw, err = os.ReadFile(filepath.Join(root, "node-build.json"))
	if err != nil {
		return "", err
	}
	if err := json.Unmarshal(raw, &build); err != nil || build.Platform != platform || build.Arch != arch || !regexp.MustCompile(`^[0-9]+$`).MatchString(build.ABI) || (build.Runtime != "node" && build.Runtime != "electron") {
		return "", fmt.Errorf("invalid mobile Node ABI proof")
	}
	node, err := mobileChild(root, m.NodePath)
	if err != nil {
		return "", err
	}
	if m.Files[m.NodePath] == "" || m.Files[m.NodePath] != build.NodeSHA256 {
		return "", fmt.Errorf("mobile Node build provenance mismatch")
	}
	sqlite, pty := false, false
	for name, digest := range build.NativeArtifacts {
		if m.Files[name] != digest || digest == "" {
			return "", fmt.Errorf("mobile native binding proof changed: %s", name)
		}
		sqlite = sqlite || strings.HasSuffix(name, "/better_sqlite3.node")
		pty = pty || strings.HasSuffix(name, "/conpty.node") || strings.HasSuffix(name, "/pty.node")
	}
	if !sqlite || !pty {
		return "", fmt.Errorf("mobile runtime has no SQLite/PTY ABI load proof")
	}
	if build.Runtime == "electron" {
		if build.Electron == "" {
			return "", fmt.Errorf("missing mobile Electron version")
		}
		for _, required := range []string{"icudtl.dat", "resources.pak", "chrome_100_percent.pak", "chrome_200_percent.pak", "v8_context_snapshot.bin", "snapshot_blob.bin"} {
			if m.Files[required] == "" {
				return "", fmt.Errorf("missing mobile Electron resource: %s", required)
			}
		}
	}
	if m.HarnessPeer.Path != "../deepseek-harness" {
		return "", fmt.Errorf("missing mobile Host peer proof")
	}
	vendor := filepath.Dir(root)
	peer := filepath.Join(vendor, "deepseek-harness")
	for _, name := range []string{"apps/cli/lib/bin.js", "packages/core/session/lib/index.js", "packages/session/session-persistence/lib/index.js"} {
		filename, err := mobileChild(peer, name)
		if err != nil {
			return "", err
		}
		resolved, err := filepath.EvalSymlinks(filename)
		if err != nil || !mobileInside(vendor, resolved) {
			return "", fmt.Errorf("mobile Host peer escapes runtime")
		}
		digest, err := mobileDigest(filename)
		if err != nil || digest != m.HarnessPeer.Artifacts[name] {
			return "", fmt.Errorf("mobile Host peer changed: %s", name)
		}
	}
	return node, nil
}
