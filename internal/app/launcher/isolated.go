package launcherctl

import (
	"crypto/sha256"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"whaleisle/internal/desktop/importer"
	"whaleisle/internal/paths"
	"whaleisle/internal/product"
)

// IsolatedDesktopRun reports whether the process runs in isolated QA or portable mode.
func IsolatedDesktopRun() bool {
	return os.Getenv("WHALEISLE_ISOLATED") == "1" || PortableDesktopRoot() != ""
}

// IsolatedDesktopHidden reports whether isolated QA mode keeps windows hidden.
func IsolatedDesktopHidden() bool {
	return os.Getenv("WHALEISLE_ISOLATED") == "1" && os.Getenv("WHALEISLE_ISOLATED_VISIBLE") != "1"
}

// PrepareIsolatedDesktop configures an isolated data directory, distinct single-instance
// identity, and native runtime paths when IsolatedDesktopRun is true.
func PrepareIsolatedDesktop(setter NativePathSetter) error {
	if !IsolatedDesktopRun() {
		return nil
	}
	data := strings.TrimSpace(os.Getenv(paths.EnvUserData))
	base, err := os.UserConfigDir()
	if err != nil {
		return err
	}
	canonicalData, err := ValidateIsolatedUserDataDir(data, base)
	if err != nil {
		return err
	}
	data = filepath.Clean(data)
	sum := sha256.Sum256([]byte(strings.ToLower(canonicalData)))
	if setter != nil {
		setter.SetName(fmt.Sprintf("%s Isolated %x", product.Name, sum[:6]))
		setter.SetUserDataPath(filepath.Join(data, "native"))
		setter.SetCachePath(filepath.Join(data, "native", "cache"))
		setter.SetLogsPath(filepath.Join(data, "native", "logs"))
	}
	return nil
}

// ValidateIsolatedUserDataDir rejects empty, relative, or installed-data-aliasing paths.
func ValidateIsolatedUserDataDir(data, userConfigBase string) (string, error) {
	data = strings.TrimSpace(data)
	if data == "" || !filepath.IsAbs(data) {
		return "", fmt.Errorf("WHALEISLE_ISOLATED requires an absolute %s directory", paths.EnvUserData)
	}
	canonicalData, err := ResolveCanonicalPath(data)
	if err != nil {
		return "", fmt.Errorf("resolve isolated %s directory: %w", paths.EnvUserData, err)
	}
	installedDir, err := ResolveCanonicalPath(paths.DefaultUserDataDirUnder(userConfigBase))
	if err != nil {
		return "", fmt.Errorf("resolve installed data directory: %w", err)
	}
	if strings.EqualFold(canonicalData, installedDir) {
		return "", fmt.Errorf("WHALEISLE_ISOLATED cannot use the installed application's data directory")
	}
	return canonicalData, nil
}

// ResolveCanonicalPath evaluates symlinks and NTFS junctions across existing
// path prefixes while preserving any not-yet-created suffix cleanly.
func ResolveCanonicalPath(path string) (string, error) {
	abs, err := filepath.Abs(path)
	if err != nil {
		return "", err
	}
	vol := filepath.VolumeName(abs)
	rest := abs[len(vol):]
	cur := vol
	if len(rest) > 0 && os.IsPathSeparator(rest[0]) {
		cur += string(filepath.Separator)
		rest = rest[1:]
	}
	parts := splitPathSegments(rest)
	const maxLinkHops = 40
	hops := 0
	for len(parts) > 0 {
		seg := parts[0]
		parts = parts[1:]
		if seg == "" || seg == "." {
			continue
		}
		if seg == ".." {
			cur = filepath.Dir(cur)
			continue
		}
		next := filepath.Join(cur, seg)
		st, lerr := os.Lstat(next)
		if lerr != nil {
			if os.IsNotExist(lerr) {
				resolvedCur, evalErr := filepath.EvalSymlinks(cur)
				if evalErr != nil {
					return "", evalErr
				}
				elems := make([]string, 0, 2+len(parts))
				elems = append(elems, resolvedCur, seg)
				elems = append(elems, parts...)
				return filepath.Clean(filepath.Join(elems...)), nil
			}
			return "", lerr
		}
		if st.Mode()&(os.ModeSymlink|os.ModeIrregular) != 0 {
			target, linkErr := os.Readlink(next)
			if linkErr == nil {
				hops++
				if hops > maxLinkHops {
					return "", os.ErrInvalid
				}
				target = stripWindowsNTPrefix(target)
				if filepath.IsAbs(target) {
					tVol := filepath.VolumeName(target)
					tRest := target[len(tVol):]
					cur = tVol
					if len(tRest) > 0 && os.IsPathSeparator(tRest[0]) {
						cur += string(filepath.Separator)
						tRest = tRest[1:]
					}
					parts = append(splitPathSegments(tRest), parts...)
				} else if len(target) > 0 && os.IsPathSeparator(target[0]) {
					cur = filepath.VolumeName(cur) + string(filepath.Separator)
					parts = append(splitPathSegments(target[1:]), parts...)
				} else {
					parts = append(splitPathSegments(target), parts...)
				}
				continue
			}
			if st.Mode()&os.ModeSymlink != 0 {
				return "", linkErr
			}
		}
		cur = next
	}
	resolved, err := filepath.EvalSymlinks(cur)
	if err != nil {
		return "", err
	}
	return filepath.Clean(resolved), nil
}

func splitPathSegments(raw string) []string {
	var parts []string
	start := -1
	for i := 0; i < len(raw); i++ {
		if os.IsPathSeparator(raw[i]) {
			if start >= 0 {
				parts = append(parts, raw[start:i])
				start = -1
			}
			continue
		}
		if start < 0 {
			start = i
		}
	}
	if start >= 0 {
		parts = append(parts, raw[start:])
	}
	return parts
}

func stripWindowsNTPrefix(target string) string {
	if len(target) >= 8 && (target[:8] == `\??\UNC\` || target[:8] == `\??\unc\`) {
		return `\\` + target[8:]
	}
	if len(target) >= 4 && target[:4] == `\??\` {
		return target[4:]
	}
	return target
}

// DesktopSystemIntegration invokes apply only when not running isolated.
func DesktopSystemIntegration(apply func() error) error {
	if IsolatedDesktopRun() {
		return nil
	}
	return apply()
}

// DesktopComponentRoots returns roots scanned for desktop components.
func DesktopComponentRoots() []string {
	if IsolatedDesktopRun() {
		return []string{paths.UserDataDir()}
	}
	return []string{paths.UserDataDir(), importer.ElectronUserDataDir()}
}
