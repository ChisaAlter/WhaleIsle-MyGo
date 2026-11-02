package update

import (
	"fmt"
	"os/exec"
	"runtime"
	"strings"
)

// CanStartStaged reports whether this platform can hand off to this installer.
func CanStartStaged(result ApplyResult) bool {
	switch runtime.GOOS {
	case "windows":
		return strings.HasSuffix(strings.ToLower(result.Path), ".exe")
	case "darwin":
		return strings.HasSuffix(strings.ToLower(result.Path), ".dmg")
	default:
		return false
	}
}

// StartStaged is called only after staging/verification and lifecycle admission.
// Recheck bytes because confirmation may leave the staged file on disk for a while.
func StartStaged(result ApplyResult) (ApplyResult, error) {
	if !CanStartStaged(result) {
		return result, fmt.Errorf("installer launch unsupported for this platform or file")
	}
	return startStaged(result, func(path string) error {
		if runtime.GOOS == "darwin" {
			// The confirmed installation opens the verified disk image in Finder.
			// The user copies the app to Applications; no privilege escalation,
			// signature bypass, or background replacement is performed here.
			return exec.Command("/usr/bin/open", path).Run()
		}
		return exec.Command(path).Start()
	})
}

func startStaged(result ApplyResult, start func(string) error) (ApplyResult, error) {
	if result.Started {
		return result, fmt.Errorf("installer already started")
	}
	sum, size, err := hashFileBounded(result.Path, NativeMaxInstallerBytes)
	if err != nil {
		return result, err
	}
	if result.SHA256 == "" || sum != result.SHA256 || size != result.Size {
		return result, fmt.Errorf("staged installer changed; download and verify again")
	}
	if err := start(result.Path); err != nil {
		return result, err
	}
	result.Started = true
	if runtime.GOOS == "darwin" {
		result.Message += "; disk image opened; copy DeepSeekHarness Go.app to Applications to finish installation"
	} else {
		result.Message += "; installer started"
	}
	return result, nil
}
