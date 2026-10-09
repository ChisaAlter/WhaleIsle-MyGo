// Package credentials reads and writes credentials.json, the secrets file
// the Electron shell keeps next to config.json in the user data directory.
//
// The Electron shell (reference/electron/src/main/config.js, writeCredentials)
// persists exactly these keys:
//
//	{"apiKey", "baseUrl", "githubToken", "remoteToken", "remoteRelayToken",
//	 "remoteDevices"}
//
// When the OS keychain is available it wraps them in a safeStorage envelope:
//
//	{"version": "safeStorage-v1", "payload": "<base64>"}
//
// where the decoded payload is the payload of Electron's Windows os_crypt
// AES-256-GCM scheme: an ASCII "v10" prefix, a 12-byte nonce, and the
// AES-256-GCM ciphertext with its 16-byte tag appended. The key is the
// DPAPI-protected "os_crypt.encrypted_key" from "Local State", a file in the
// same directory: base64-decode it, strip the leading "DPAPI" marker, then
// CryptUnprotectData. Save writes that same envelope via CryptProtectData.
// See the package tests for the round-trip against the real DPAPI.
package credentials

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
)

// FileName is the secrets file inside the user data directory.
const FileName = "credentials.json"

// LocalStateName is Chromium's key database next to credentials.json.
const LocalStateName = "Local State"

// EnvelopeVersion marks a safeStorage-wrapped credentials.json.
const EnvelopeVersion = "safeStorage-v1"

// v10Prefix is the well-known Windows os_crypt AEAD version tag.
const v10Prefix = "v10"

// nonceSize and tagSize are the AES-GCM parameters os_crypt uses.
const (
	nonceSize = 12
	tagSize   = 16
)

// RemoteDevice is one paired phone/LAN device. The Electron shell treats the
// entries as opaque objects (id, name, token, createdAt, lastSeenAt,
// revokedAt, ...); they are carried through unchanged.
type RemoteDevice map[string]any

// Credentials holds the exact fields the Electron shell writes to
// credentials.json (config.js saveConfig / writeCredentials).
type Credentials struct {
	APIKey           string         `json:"apiKey"`
	BaseURL          string         `json:"baseUrl"`
	GithubToken      string         `json:"githubToken"`
	RemoteToken      string         `json:"remoteToken"`
	RemoteRelayToken string         `json:"remoteRelayToken"`
	RemoteDevices    []RemoteDevice `json:"remoteDevices"`
}

type envelope struct {
	Version string `json:"version"`
	Payload string `json:"payload"`
}

type localState struct {
	OsCrypt struct {
		EncryptedKey string `json:"encrypted_key"`
	} `json:"os_crypt"`
}

// Load reads credentials.json from dir. A missing file yields the zero
// Credentials and no error. A safeStorage envelope is decrypted with the
// DPAPI-protected key from Local State; a plaintext file is read as-is.
func Load(dir string) (Credentials, error) {
	var creds Credentials
	data, err := os.ReadFile(filepath.Join(dir, FileName))
	if errors.Is(err, fs.ErrNotExist) {
		return creds, nil
	}
	if err != nil {
		return creds, err
	}
	var raw map[string]json.RawMessage
	if err := json.Unmarshal(data, &raw); err != nil {
		return creds, fmt.Errorf("credentials: %s: %w", filepath.Join(dir, FileName), err)
	}
	if isEnvelope(raw) {
		var env envelope
		if err := json.Unmarshal(data, &env); err != nil {
			return creds, fmt.Errorf("credentials: %s: %w", filepath.Join(dir, FileName), err)
		}
		return decryptEnvelope(dir, env.Payload)
	}
	if err := json.Unmarshal(data, &creds); err != nil {
		return creds, fmt.Errorf("credentials: %s: %w", filepath.Join(dir, FileName), err)
	}
	return creds, nil
}

// Save writes credentials.json as a safeStorage-v1 envelope encrypted with
// the DPAPI-protected Local State key. The write is atomic (temp file +
// rename in the same directory).
func Save(dir string, creds Credentials) error {
	key, err := loadLocalStateKey(dir)
	if err != nil {
		return err
	}
	plaintext, err := json.Marshal(creds)
	if err != nil {
		return err
	}
	payload, err := seal(key, plaintext)
	if err != nil {
		return err
	}
	data, err := json.MarshalIndent(envelope{Version: EnvelopeVersion, Payload: base64.StdEncoding.EncodeToString(payload)}, "", "  ")
	if err != nil {
		return err
	}
	return writeAtomic(filepath.Join(dir, FileName), data)
}

func isEnvelope(raw map[string]json.RawMessage) bool {
	var version string
	if v, ok := raw["version"]; ok && json.Unmarshal(v, &version) == nil && version == EnvelopeVersion {
		_, hasPayload := raw["payload"]
		return hasPayload
	}
	return false
}

func decryptEnvelope(dir, payload string) (Credentials, error) {
	var creds Credentials
	key, err := loadLocalStateKey(dir)
	if err != nil {
		return creds, err
	}
	blob, err := base64.StdEncoding.DecodeString(payload)
	if err != nil {
		return creds, fmt.Errorf("credentials: envelope payload is not base64: %w", err)
	}
	plaintext, err := open(key, blob)
	if err != nil {
		return creds, err
	}
	if err := json.Unmarshal(plaintext, &creds); err != nil {
		return creds, fmt.Errorf("credentials: decrypted envelope is not JSON: %w", err)
	}
	return creds, nil
}

// loadLocalStateKey reads Local State next to credentials.json and returns
// the DPAPI-unprotected os_crypt AES-256 key.
func loadLocalStateKey(dir string) ([]byte, error) {
	path := filepath.Join(dir, LocalStateName)
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("credentials: %s: %w", path, err)
	}
	var state localState
	if err := json.Unmarshal(data, &state); err != nil {
		return nil, fmt.Errorf("credentials: %s: %w", path, err)
	}
	if state.OsCrypt.EncryptedKey == "" {
		return nil, fmt.Errorf("credentials: %s: os_crypt.encrypted_key missing", path)
	}
	blob, err := base64.StdEncoding.DecodeString(state.OsCrypt.EncryptedKey)
	if err != nil {
		return nil, fmt.Errorf("credentials: %s: encrypted_key is not base64: %w", path, err)
	}
	const dpapiMarker = "DPAPI"
	if len(blob) <= len(dpapiMarker) || string(blob[:len(dpapiMarker)]) != dpapiMarker {
		return nil, fmt.Errorf("credentials: %s: encrypted_key lacks the DPAPI marker", path)
	}
	key, err := dpapiUnprotect(blob[len(dpapiMarker):])
	if err != nil {
		return nil, fmt.Errorf("credentials: %s: %w", path, err)
	}
	return key, nil
}

// seal encrypts with the os_crypt Windows scheme: "v10" + 12-byte nonce +
// AES-256-GCM ciphertext (tag appended).
func seal(key, plaintext []byte) ([]byte, error) {
	block, err := aes.NewCipher(key)
	if err != nil {
		return nil, fmt.Errorf("credentials: %w", err)
	}
	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return nil, fmt.Errorf("credentials: %w", err)
	}
	nonce := make([]byte, nonceSize)
	if _, err := rand.Read(nonce); err != nil {
		return nil, fmt.Errorf("credentials: nonce: %w", err)
	}
	out := make([]byte, 0, len(v10Prefix)+nonceSize+len(plaintext)+tagSize)
	out = append(out, v10Prefix...)
	out = append(out, nonce...)
	return gcm.Seal(out, nonce, plaintext, nil), nil
}

// open decrypts the os_crypt scheme produced by seal, with clear errors for
// short, mistagged or tampered payloads.
func open(key, blob []byte) ([]byte, error) {
	prefix := len(v10Prefix)
	if len(blob) < prefix+nonceSize+tagSize {
		return nil, errors.New("credentials: envelope payload is too short for v10 + nonce + tag")
	}
	if string(blob[:prefix]) != v10Prefix {
		return nil, fmt.Errorf("credentials: envelope payload lacks the %q prefix", v10Prefix)
	}
	block, err := aes.NewCipher(key)
	if err != nil {
		return nil, fmt.Errorf("credentials: %w", err)
	}
	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return nil, fmt.Errorf("credentials: %w", err)
	}
	plaintext, err := gcm.Open(nil, blob[prefix:prefix+nonceSize], blob[prefix+nonceSize:], nil)
	if err != nil {
		return nil, fmt.Errorf("credentials: envelope payload failed AES-GCM authentication: %w", err)
	}
	return plaintext, nil
}

func writeAtomic(path string, data []byte) error {
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return err
	}
	tmp, err := os.CreateTemp(filepath.Dir(path), ".credentials-*.json")
	if err != nil {
		return err
	}
	defer os.Remove(tmp.Name())
	if _, err := tmp.Write(data); err != nil {
		tmp.Close()
		return err
	}
	if err := tmp.Close(); err != nil {
		return err
	}
	return os.Rename(tmp.Name(), path)
}
