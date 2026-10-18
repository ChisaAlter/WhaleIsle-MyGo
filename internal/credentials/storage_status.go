package credentials

import (
	"encoding/json"
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
	s := StorageStatus{Mode: "unknown", Backend: credentialOS.backend, SupportedPlatform: credentialOS.supported}
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
		var env envelope
		_ = json.Unmarshal(data, &env) // Already structurally validated above.
		if env.Version != credentialOS.version {
			s.SupportedPlatform = false
			if env.Version == KeychainEnvelopeVersion {
				s.Backend = "macOS Keychain + AES-256-GCM"
			} else {
				s.Backend = "Windows DPAPI + AES-256-GCM (safeStorage-v1)"
				if runtime.GOOS == "darwin" {
					s.Backend = "Legacy macOS Keychain + AES-128-CBC (safeStorage-v1)"
					s.SupportedPlatform = credentialOS.supported
				}
			}
		}
	}
	return s
}
