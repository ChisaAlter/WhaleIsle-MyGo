package update

import (
	"fmt"
	"os/exec"
	"runtime"
	"strings"
)

// CanStartStaged reports whether this platform can hand off to this installer.
func CanStartStaged(result ApplyResult) bool {
	return runtime.GOOS == "windows" && strings.HasSuffix(strings.ToLower(result.Path), ".exe")
}

// StartStaged is called only after staging/verification and lifecycle admission.
// Recheck bytes because confirmation may leave the staged file on disk for a while.
func StartStaged(result ApplyResult) (ApplyResult, error) {
	if !CanStartStaged(result) {
		return result, fmt.Errorf("installer launch unsupported for this platform or file")
	}
	return startStaged(result, func(path string) error { return exec.Command(path).Start() })
}

func startStaged(result ApplyResult, start func(string) error) (ApplyResult, error) {
	if result.Started {
		return result, fmt.Errorf("installer already started")
	}
	sum, size, err := hashFile(result.Path)
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
	result.Message += "; installer started"
	return result, nil
}
