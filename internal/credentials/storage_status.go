package credentials

import (
	"errors"
	"io/fs"
	"os"
	"path/filepath"
	"runtime"
)

// StorageStatus contains format metadata only. It never contains credential
// values and does not decrypt the envelope or test the OS key's availability.
type StorageStatus struct {
	Mode              string
	Backend           string
	SupportedPlatform bool
	Error             string
}

func InspectStorage(dir string) StorageStatus {
	s := StorageStatus{Mode: "unknown", Backend: "Windows DPAPI + AES-256-GCM (safeStorage-v1)", SupportedPlatform: runtime.GOOS == "windows"}
	data, err := os.ReadFile(filepath.Join(dir, FileName))
	if errors.Is(err, fs.ErrNotExist) {
		s.Mode = "missing"
		return s
	}
	if err != nil {
		s.Error = "Credential storage metadata could not be read."
		return s
	}
	kind, err := classifyDocument(data)
	if err != nil {
		s.Error = "Credential storage format is invalid or unsupported."
		return s
	}
	s.Mode = "plaintext"
	if kind == docEnvelope {
		s.Mode = "encrypted"
	}
	return s
}
