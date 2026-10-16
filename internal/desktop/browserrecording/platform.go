package browserrecording

import "runtime"

// PlatformAvailable reports whether an isolated native encoder can be attempted.
// It does not claim codec availability. Only an explicit Start creates the
// helper and probes MediaRecorder, WebM and manual canvas-frame support.
func PlatformAvailable() bool {
	return (runtime.GOOS == "windows" || runtime.GOOS == "darwin" || runtime.GOOS == "linux") && (runtime.GOARCH == "amd64" || runtime.GOARCH == "arm64")
}
