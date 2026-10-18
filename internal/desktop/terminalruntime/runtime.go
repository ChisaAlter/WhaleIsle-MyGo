// Package terminalruntime stages and locates the exact native Ghostty ABI used
// by the checked-in MyGo binding. It never downloads or loads executable code.
package terminalruntime

import (
	"crypto/sha256"
	_ "embed"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
)

// These are copied from the binding's mygo-plugin.json and its pinned Ghostty
// source license. The staging tool verifies the manifest still matches MyGo.
//
//go:embed native-manifest.json
var bindingManifest []byte

//go:embed Ghostty-LICENSE
var license []byte

type File struct {
	Name   string `json:"name"`
	URL    string `json:"url"`
	SHA256 string `json:"sha256"`
}
type Spec struct {
	File
	Target string `json:"target"`
	Source string `json:"source"`
}
type manifest struct {
	Libraries []struct {
		Name   string          `json:"name"`
		Source string          `json:"source"`
		Files  map[string]File `json:"files"`
	} `json:"libraries"`
}

func specification(data []byte, target string) (Spec, error) {
	var m manifest
	if err := json.Unmarshal(data, &m); err != nil {
		return Spec{}, err
	}
	for _, lib := range m.Libraries {
		if lib.Name != "libghostty-vt" {
			continue
		}
		f, ok := lib.Files[target]
		if !ok {
			break
		}
		if f.Name == "" || filepath.Base(f.Name) != f.Name || !regexp.MustCompile(`^[a-f0-9]{64}$`).MatchString(f.SHA256) {
			return Spec{}, fmt.Errorf("invalid pinned terminal manifest")
		}
		return Spec{File: f, Target: target, Source: lib.Source}, nil
	}
	return Spec{}, fmt.Errorf("native terminal has no pinned runtime for %s", target)
}
func Specification(target string) (Spec, error) { return specification(bindingManifest, target) }

// CheckBinding refuses to package a stale resource after the MyGo ABI changes.
func CheckBinding(data []byte, target string) error {
	expected, err := Specification(target)
	if err != nil {
		return err
	}
	actual, err := specification(data, target)
	if err != nil {
		return err
	}
	if actual != expected {
		return fmt.Errorf("MyGo native terminal binding changed; refresh the pinned runtime manifest before packaging")
	}
	return nil
}

func verifiedBytes(name string, spec Spec) ([]byte, error) {
	info, err := os.Lstat(name)
	if err != nil {
		return nil, err
	}
	if !info.Mode().IsRegular() || info.Size() <= 0 || info.Size() > 64<<20 {
		return nil, fmt.Errorf("terminal runtime must be a regular native-library file")
	}
	data, err := os.ReadFile(name)
	if err != nil {
		return nil, err
	}
	if fmt.Sprintf("%x", sha256.Sum256(data)) != spec.SHA256 {
		return nil, fmt.Errorf("terminal runtime SHA256 mismatch for %s; use the pinned MyGo release", spec.Name)
	}
	return data, nil
}

// Resolve checks the explicit override, or the same resource/cache directories
// MyGo uses. A corrupt present library is an error, not a reason to silently
// fall back to another copy. Missing files never cause a network request.
func Resolve(spec Spec, explicit string, dirs []string) (string, error) {
	if explicit != "" {
		path, err := filepath.Abs(explicit)
		if err != nil {
			return "", err
		}
		_, err = verifiedBytes(path, spec)
		return path, err
	}
	for _, dir := range dirs {
		if dir == "" {
			continue
		}
		candidate := filepath.Join(dir, spec.Name)
		if _, err := os.Lstat(candidate); os.IsNotExist(err) {
			continue
		} else if err != nil {
			return "", err
		}
		if _, err := verifiedBytes(candidate, spec); err != nil {
			return "", err
		}
		return filepath.Abs(candidate)
	}
	return "", fmt.Errorf("原生终端缺少 %s；请使用包含终端运行库的安装包，或设置 MYGO_GHOSTTY_VT 指向已校验的固定版本", spec.Name)
}

func CacheDirectory(spec Spec) string {
	dir, err := os.UserCacheDir()
	if err != nil {
		return ""
	}
	return filepath.Join(dir, "mygo", "natives", spec.SHA256)
}

// Stage copies already verified bytes, license and provenance beside the EXE.
// The binary stays in build output, never in git. It is not loaded or executed.
func Stage(source, out string, spec Spec) error {
	data, err := verifiedBytes(source, spec)
	if err != nil {
		return err
	}
	if err := os.MkdirAll(out, 0755); err != nil {
		return err
	}
	if info, err := os.Lstat(out); err != nil || !info.IsDir() || info.Mode()&os.ModeSymlink != 0 {
		return fmt.Errorf("terminal staging output must be a real directory")
	}
	metadata, _ := json.MarshalIndent(spec, "", "  ")
	for _, file := range []struct {
		name string
		data []byte
	}{{spec.Name, data}, {spec.Name + ".LICENSE", license}, {"terminal-runtime.json", append(metadata, '\n')}} {
		if err := writeAtomic(filepath.Join(out, file.name), file.data); err != nil {
			return err
		}
	}
	return nil
}
func writeAtomic(name string, data []byte) error {
	if info, err := os.Lstat(name); err == nil && !info.Mode().IsRegular() {
		return fmt.Errorf("terminal staging destination is not a regular file")
	} else if err != nil && !os.IsNotExist(err) {
		return err
	}
	f, err := os.CreateTemp(filepath.Dir(name), ".terminal-")
	if err != nil {
		return err
	}
	tmp := f.Name()
	defer os.Remove(tmp)
	if _, err := f.Write(data); err != nil {
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
	return os.Rename(tmp, name)
}
