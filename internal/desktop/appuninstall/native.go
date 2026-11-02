package appuninstall

import (
	"errors"
	"os"
	"path/filepath"
	"strings"

	"whaleisle/internal/product"
)

// These identifiers are the native NSIS contract in scripts/nsis/whaleisle.nsi.
// The legacy Electron installer shares RegistryKey, so that key alone is unsafe.
const RegistryKey = `Software\Microsoft\Windows\CurrentVersion\Uninstall\ai.deepseek.harness.gui`
const NativeMarker = "WhaleIsle-MyGo"

var ErrNotNative = errors.New("no verified current native installation; use Installed apps")

type Record struct{ DisplayName, NativeRuntime, InstallLocation, UninstallString string }
type Plan struct{ executable, current string }

func (p Plan) InstallLocation() string { return filepath.Dir(p.executable) }

// Resolve accepts only this running native installation and the fixed NSIS
// executable. Registry command text is compared, never parsed or executed.
func Resolve(current string, r Record) (Plan, error) {
	if r.NativeRuntime != NativeMarker || (r.DisplayName != product.Name && r.DisplayName != "Whale Isle") || !filepath.IsAbs(current) || !filepath.IsAbs(r.InstallLocation) {
		return Plan{}, ErrNotNative
	}
	root := filepath.Clean(r.InstallLocation)
	if !strings.EqualFold(filepath.Base(current), "whaleisle.exe") || !strings.EqualFold(filepath.Dir(current), root) {
		return Plan{}, ErrNotNative
	}
	exe := filepath.Join(root, "Uninstall.exe")
	if r.UninstallString != `"`+exe+`"` {
		return Plan{}, ErrNotNative
	}
	return Plan{executable: exe, current: current}, nil
}

// Validate rejects symlinks/reparse-resolved paths leaving the recorded directory.
func (p Plan) Validate() error {
	if p.executable == "" {
		return ErrNotNative
	}
	for _, path := range []string{p.current, p.executable} {
		info, err := os.Lstat(path)
		if err != nil {
			return err
		}
		if !info.Mode().IsRegular() {
			return ErrNotNative
		}
		actual, err := filepath.EvalSymlinks(path)
		if err != nil {
			return err
		}
		if !strings.EqualFold(filepath.Clean(actual), filepath.Clean(path)) {
			return ErrNotNative
		}
	}
	return nil
}

// Launch revalidates immediately before dispatch, never supplies silent flags,
// and only reports process creation. The native NSIS confirmation remains visible.
func (p Plan) Launch(start func(string) error) error {
	if err := p.Validate(); err != nil {
		return err
	}
	if start == nil {
		return ErrUnavailable
	}
	return start(p.executable)
}
