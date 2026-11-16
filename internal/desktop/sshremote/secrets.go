package sshremote

import (
	"crypto/rand"
	"encoding/binary"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"time"

	"whaleisle/internal/credentials"
)

const sshSecretsNamespace = "ssh"

type sshSecretBucket struct {
	Connections map[string]connectionSecrets `json:"connections,omitempty"`
	Tunnel      *tunnelSecrets               `json:"tunnel,omitempty"`
}

type connectionSecrets struct {
	Password      string `json:"password,omitempty"`
	Passphrase    string `json:"passphrase,omitempty"`
	ProxyPassword string `json:"proxyPassword,omitempty"`
}

func (s connectionSecrets) empty() bool {
	return s.Password == "" && s.Passphrase == "" && s.ProxyPassword == ""
}

type tunnelSecrets struct {
	Password   string `json:"password,omitempty"`
	Passphrase string `json:"passphrase,omitempty"`
}

func (s tunnelSecrets) empty() bool {
	return s.Password == "" && s.Passphrase == ""
}

// credentialsDirForRegistry resolves the userData credentials directory for a
// machines.json path without escaping outside the caller's folder on custom paths.
func credentialsDirForRegistry(registryPath string) string {
	cleaned := filepath.Clean(registryPath)
	dir := filepath.Dir(cleaned)
	if strings.EqualFold(filepath.Base(dir), "ssh") {
		parent := filepath.Dir(dir)
		if parent != "" && parent != dir {
			return parent
		}
	}
	return dir
}

func loadSSHSecrets(dir string) (sshSecretBucket, bool, error) {
	if strings.TrimSpace(dir) == "" {
		return sshSecretBucket{}, false, nil
	}
	var bucket sshSecretBucket
	ok, err := credentials.LoadNamespace(dir, sshSecretsNamespace, &bucket)
	if err != nil || !ok {
		return sshSecretBucket{}, ok, err
	}
	return bucket, true, nil
}

func updateSSHSecrets(dir string, mutate func(obj map[string]json.RawMessage) error) error {
	return credentials.UpdateNamespace(dir, sshSecretsNamespace, func(existing json.RawMessage) (json.RawMessage, error) {
		obj := map[string]json.RawMessage{}
		if len(existing) > 0 {
			_ = json.Unmarshal(existing, &obj)
		}
		if obj == nil {
			obj = map[string]json.RawMessage{}
		}
		if err := mutate(obj); err != nil {
			return nil, err
		}
		if len(obj) == 0 {
			return nil, nil
		}
		return json.Marshal(obj)
	})
}

func writePrivateFileAtomic(path string, data []byte) error {
	if err := os.MkdirAll(filepath.Dir(path), 0o700); err != nil {
		return err
	}
	tmp, err := os.CreateTemp(filepath.Dir(path), "."+filepath.Base(path)+".tmp-*")
	if err != nil {
		return err
	}
	tmpPath := tmp.Name()
	defer os.Remove(tmpPath)
	if err := tmp.Chmod(0o600); err != nil {
		_ = tmp.Close()
		return err
	}
	if _, err := tmp.Write(data); err != nil {
		_ = tmp.Close()
		return err
	}
	if err := tmp.Sync(); err != nil {
		_ = tmp.Close()
		return err
	}
	if err := tmp.Close(); err != nil {
		return err
	}
	return renameWithRetry(tmpPath, path)
}

func renameWithRetry(oldpath, newpath string) error {
	var errRename error
	for attempt := 0; attempt < 8; attempt++ {
		errRename = os.Rename(oldpath, newpath)
		if errRename == nil {
			return nil
		}
		time.Sleep(jitteredRetryDelay(time.Duration(attempt+1) * 5 * time.Millisecond))
	}
	return errRename
}

func jitteredRetryDelay(base time.Duration) time.Duration {
	if base <= 0 {
		return time.Millisecond
	}
	var b [2]byte
	if _, err := rand.Read(b[:]); err != nil {
		return base
	}
	fraction := float64(binary.LittleEndian.Uint16(b[:])) / 65535.0
	return base + time.Duration(float64(base)*0.5*fraction)
}
