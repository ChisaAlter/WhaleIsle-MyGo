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
// On Windows the decoded payload is the payload of Electron's os_crypt
// AES-256-GCM scheme: an ASCII "v10" prefix, a 12-byte nonce, and the
// AES-256-GCM ciphertext with its 16-byte tag appended. The key is the
// DPAPI-protected "os_crypt.encrypted_key" from "Local State", a file in the
// same directory: base64-decode it, strip the leading "DPAPI" marker, then
// CryptUnprotectData. Save writes that same envelope via CryptProtectData.
// macOS reads and writes Electron's safeStorage v10 format using the original
// Whale Isle Keychain item. Earlier MyGo native envelopes remain readable and
// are backed up before a save migrates them. Load never rewrites files. Linux
// secure storage is unsupported; Save never falls back to plaintext.
package credentials

import (
	"bytes"
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
	"time"
)

// FileName is the secrets file inside the user data directory.
const FileName = "credentials.json"

// Load errors. They are stable so callers can branch with errors.Is.
var (
	// ErrUnsupportedEnvelopeVersion marks a safeStorage envelope whose
	// version this package does not implement.
	ErrUnsupportedEnvelopeVersion = errors.New("credentials: unsupported envelope version")
	// ErrInvalidCredentialsFormat marks a document that is neither a valid
	// plaintext credentials object nor a well-formed supported envelope.
	ErrInvalidCredentialsFormat = errors.New("credentials: invalid credentials format")
)

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
//
// A document that carries envelope markers but is not a well-formed,
// supported envelope is rejected with ErrInvalidCredentialsFormat or
// ErrUnsupportedEnvelopeVersion; it is never reinterpreted as plaintext.
// Loading never writes and never returns partial credentials with an error.
func Load(dir string) (Credentials, error) {
	var creds Credentials
	err := withDirLock(dir, false, func() error {
		var loadErr error
		_, creds, loadErr = loadRawUnlocked(dir)
		return loadErr
	})
	if err != nil {
		return Credentials{}, err
	}
	return creds, nil
}

func loadRawUnlocked(dir string) (map[string]json.RawMessage, Credentials, error) {
	path := filepath.Join(dir, FileName)
	data, err := os.ReadFile(path)
	if errors.Is(err, fs.ErrNotExist) {
		return nil, Credentials{}, nil
	}
	if err != nil {
		return nil, Credentials{}, err
	}
	return decodeDocumentBytes(dir, path, data)
}

func decodeDocumentBytes(dir, path string, data []byte) (map[string]json.RawMessage, Credentials, error) {
	kind, err := classifyDocument(data)
	if err != nil {
		return nil, Credentials{}, fmt.Errorf("credentials: %s: %w", path, err)
	}
	if kind == docEnvelope {
		return decryptEnvelopeRaw(dir, data)
	}
	raw, creds, err := decodeCredentialsRaw(data)
	if err != nil {
		return nil, Credentials{}, fmt.Errorf("credentials: %s: %w", path, err)
	}
	return raw, creds, nil
}

// documentKind is how a credentials.json document must be decoded.
type documentKind int

const (
	docPlaintext documentKind = iota
	docEnvelope
)

// classifyDocument decides how a credentials.json document should be decoded.
// It inspects structure only, so it never needs the Local State key and its
// result does not depend on DPAPI availability.
func classifyDocument(data []byte) (documentKind, error) {
	var doc json.RawMessage
	if err := json.Unmarshal(data, &doc); err != nil {
		return docPlaintext, ErrInvalidCredentialsFormat
	}
	trimmed := bytes.TrimSpace(doc)
	if len(trimmed) == 0 || trimmed[0] != '{' {
		return docPlaintext, ErrInvalidCredentialsFormat
	}
	var raw map[string]json.RawMessage
	if err := json.Unmarshal(trimmed, &raw); err != nil {
		return docPlaintext, ErrInvalidCredentialsFormat
	}
	versionRaw, hasVersion := raw["version"]
	payloadRaw, hasPayload := raw["payload"]
	if !hasVersion && !hasPayload {
		return docPlaintext, nil
	}
	// Envelope markers are present: the document must be a well-formed
	// supported envelope, even if ordinary credential fields also appear.
	var version string
	if !hasVersion || json.Unmarshal(versionRaw, &version) != nil || version == "" {
		return docPlaintext, fmt.Errorf("%w: version", ErrInvalidCredentialsFormat)
	}
	if version != EnvelopeVersion && version != KeychainEnvelopeVersion {
		return docPlaintext, ErrUnsupportedEnvelopeVersion
	}
	var payload string
	if !hasPayload || json.Unmarshal(payloadRaw, &payload) != nil || payload == "" {
		return docPlaintext, fmt.Errorf("%w: payload", ErrInvalidCredentialsFormat)
	}
	if _, err := base64.StdEncoding.DecodeString(payload); err != nil {
		return docPlaintext, fmt.Errorf("%w: payload is not base64", ErrInvalidCredentialsFormat)
	}
	return docEnvelope, nil
}

// decodeCredentialsObject requires a JSON object and only assigns out on
// success, so a malformed document never leaves partial credentials behind.
func decodeCredentialsObject(data []byte, out *Credentials) error {
	_, creds, err := decodeCredentialsRaw(data)
	if err != nil {
		return err
	}
	*out = creds
	return nil
}

func decodeCredentialsRaw(data []byte) (map[string]json.RawMessage, Credentials, error) {
	var doc json.RawMessage
	if err := json.Unmarshal(data, &doc); err != nil {
		return nil, Credentials{}, ErrInvalidCredentialsFormat
	}
	trimmed := bytes.TrimSpace(doc)
	if len(trimmed) == 0 || trimmed[0] != '{' {
		return nil, Credentials{}, ErrInvalidCredentialsFormat
	}
	var creds Credentials
	if err := json.Unmarshal(trimmed, &creds); err != nil {
		return nil, Credentials{}, ErrInvalidCredentialsFormat
	}
	var raw map[string]json.RawMessage
	if err := json.Unmarshal(trimmed, &raw); err != nil {
		return nil, Credentials{}, ErrInvalidCredentialsFormat
	}
	if raw == nil {
		raw = map[string]json.RawMessage{}
	}
	return raw, creds, nil
}

var modeledCredentialKeys = [6]string{
	"apiKey",
	"baseUrl",
	"githubToken",
	"remoteToken",
	"remoteRelayToken",
	"remoteDevices",
}

func cloneRawMap(src map[string]json.RawMessage) map[string]json.RawMessage {
	out := make(map[string]json.RawMessage, len(src))
	for k, v := range src {
		out[k] = bytes.Clone(v)
	}
	return out
}

func mergeCredentialsIntoRaw(base map[string]json.RawMessage, creds Credentials) (map[string]json.RawMessage, error) {
	credsJSON, err := json.Marshal(creds)
	defer clear(credsJSON)
	if err != nil {
		return nil, err
	}
	var credsMap map[string]json.RawMessage
	if err := json.Unmarshal(credsJSON, &credsMap); err != nil {
		return nil, err
	}
	merged := cloneRawMap(base)
	for _, k := range modeledCredentialKeys {
		if v, ok := credsMap[k]; ok {
			merged[k] = bytes.Clone(v)
		}
	}
	return merged, nil
}

// Save writes an encrypted envelope using the platform storage key. Windows
// retains safeStorage-v1; macOS writes the same shared envelope with its Keychain format. The write is atomic (temp file +
// Sync + Close + retry rename in the same directory) and preserves unmodeled fields.
// This entry is for private stores; shared desktop callers use SaveSharedProfile.
func Save(dir string, creds Credentials) error {
	return saveProfile(dir, creds, false)
}

// SaveSharedProfile writes the file shared with Electron only while the caller
// owns the platform's common profile lease. Reading does not need that lease.
func SaveSharedProfile(dir string, creds Credentials) error {
	return saveProfile(dir, creds, true)
}

func saveProfile(dir string, creds Credentials, shared bool) error {
	return withProfileDirLock(dir, true, shared, func() error {
		existingRaw, create, _, err := readExistingRawForSave(dir)
		if err != nil {
			return err
		}
		defer clearRawMap(existingRaw)
		merged, err := mergeCredentialsIntoRaw(existingRaw, creds)
		if err != nil {
			return err
		}
		defer clearRawMap(merged)
		return sealAndWriteRaw(dir, merged, create, nil)
	})
}

func clearRawMap(m map[string]json.RawMessage) {
	for k, v := range m {
		clear(v)
		delete(m, k)
	}
}

func readExistingRawForSave(dir string) (map[string]json.RawMessage, bool, fileBaseline, error) {
	path := filepath.Join(dir, FileName)
	data, baseline, err := readBytesAndBaseline(path)
	if errors.Is(err, fs.ErrNotExist) {
		return map[string]json.RawMessage{}, true, fileBaseline{}, nil
	}
	if err != nil {
		return nil, false, fileBaseline{}, err
	}
	kind, err := classifyDocument(data)
	if err != nil {
		return nil, false, fileBaseline{}, fmt.Errorf("credentials: %s: %w", path, err)
	}
	create := true
	if kind == docEnvelope {
		var env envelope
		if err := json.Unmarshal(data, &env); err != nil {
			return nil, false, fileBaseline{}, fmt.Errorf("credentials: %s: %w", path, ErrInvalidCredentialsFormat)
		}
		if env.Version == EnvelopeVersion || env.Version == credentialOS.version && credentialOS.seal == nil {
			create = false
		}
	}
	raw, _, err := decodeDocumentBytes(dir, path, data)
	if err != nil {
		return nil, false, fileBaseline{}, err
	}
	return raw, create, baseline, nil
}

func sealAndWriteRaw(dir string, raw map[string]json.RawMessage, create bool, rename func(string, string) error) error {
	plaintext, err := json.Marshal(raw)
	defer clear(plaintext)
	if err != nil {
		return err
	}
	version := credentialOS.version
	var payload []byte
	if credentialOS.seal != nil {
		version, payload, err = credentialOS.seal(dir, plaintext, create)
	} else {
		var key []byte
		key, err = credentialOS.key(dir, create)
		defer clear(key)
		if err == nil {
			payload, err = seal(key, plaintext)
		}
	}
	if err != nil {
		return err
	}
	data, err := json.MarshalIndent(envelope{Version: version, Payload: base64.StdEncoding.EncodeToString(payload)}, "", "  ")
	if err != nil {
		return err
	}
	if credentialOS.seal != nil {
		if err := preserveMigratedEnvelope(dir, version); err != nil {
			return err
		}
	}
	if err := writeAtomic(filepath.Join(dir, FileName), data, rename); err != nil {
		return err
	}
	InvalidateStorageStatusCache(dir)
	return nil
}

// preserveMigratedEnvelope keeps the earlier encrypted bytes once, beside the
// shared file. A previous migration backup is never replaced with newer data.
func preserveMigratedEnvelope(dir, targetVersion string) error {
	path := filepath.Join(dir, FileName)
	data, err := os.ReadFile(path)
	if errors.Is(err, fs.ErrNotExist) {
		return nil
	}
	if err != nil {
		return err
	}
	var env envelope
	if json.Unmarshal(data, &env) != nil || env.Version != KeychainEnvelopeVersion || env.Version == targetVersion {
		return nil
	}
	backupPath := path + ".mygo-keychain-backup"
	backup, err := os.OpenFile(backupPath, os.O_WRONLY|os.O_CREATE|os.O_EXCL, 0o600)
	if errors.Is(err, fs.ErrExist) {
		previous, readErr := os.ReadFile(backupPath)
		if readErr != nil {
			return readErr
		}
		if bytes.Equal(previous, data) {
			return nil
		}
		backup, err = os.CreateTemp(dir, FileName+".mygo-keychain-backup-*")
	}
	if err != nil {
		return err
	}
	complete := false
	defer func() {
		if !complete {
			_ = os.Remove(backup.Name())
		}
	}()
	if _, err = backup.Write(data); err == nil {
		err = backup.Sync()
	}
	closeErr := backup.Close()
	if err != nil {
		return err
	}
	complete = closeErr == nil
	return closeErr
}

// decryptEnvelope decodes a document already classified as docEnvelope. The
// payload is validated before the Local State key is read.
func decryptEnvelope(dir string, data []byte) (Credentials, error) {
	_, creds, err := decryptEnvelopeRaw(dir, data)
	return creds, err
}

func decryptEnvelopeRaw(dir string, data []byte) (map[string]json.RawMessage, Credentials, error) {
	path := filepath.Join(dir, FileName)
	var env envelope
	if err := json.Unmarshal(data, &env); err != nil {
		return nil, Credentials{}, fmt.Errorf("credentials: %s: %w", path, ErrInvalidCredentialsFormat)
	}
	blob, err := base64.StdEncoding.DecodeString(env.Payload)
	if err != nil {
		return nil, Credentials{}, fmt.Errorf("credentials: %s: %w", path, ErrInvalidCredentialsFormat)
	}
	var key, plaintext []byte
	if env.Version == EnvelopeVersion {
		plaintext, err = decryptSafeStorage(dir, blob)
	} else if env.Version == credentialOS.version {
		key, err = credentialOS.key(dir, false)
	} else {
		return nil, Credentials{}, ErrUnsupportedEnvelopeVersion
	}
	defer clear(key)
	if err != nil {
		return nil, Credentials{}, err
	}
	if env.Version != EnvelopeVersion {
		plaintext, err = open(key, blob)
	}
	if err != nil {
		return nil, Credentials{}, err
	}
	defer clear(plaintext)
	raw, creds, err := decodeCredentialsRaw(plaintext)
	if err != nil {
		return nil, Credentials{}, fmt.Errorf("credentials: %s: decrypted envelope: %w", path, err)
	}
	return raw, creds, nil
}

// loadLocalStateKey reads Local State next to credentials.json and returns
// the DPAPI-unprotected os_crypt AES-256 key.
func loadLocalStateKey(dir string) ([]byte, error) {
	return loadOrCreateLocalStateKey(dir, false)
}

func loadOrCreateLocalStateKey(dir string, create bool) ([]byte, error) {
	path := filepath.Join(dir, LocalStateName)
	data, err := os.ReadFile(path)
	if err != nil {
		if errors.Is(err, fs.ErrNotExist) && create {
			return createLocalStateKey(path)
		}
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

func createLocalStateKey(path string) ([]byte, error) {
	key := make([]byte, 32)
	if _, err := rand.Read(key); err != nil {
		clear(key)
		return nil, err
	}
	protected, err := dpapiProtect(key)
	if err != nil {
		clear(key)
		return nil, err
	}
	blob := append([]byte("DPAPI"), protected...)
	state := map[string]any{
		"os_crypt": map[string]any{
			"encrypted_key": base64.StdEncoding.EncodeToString(blob),
		},
	}
	data, err := json.MarshalIndent(state, "", "  ")
	if err != nil {
		clear(key)
		return nil, err
	}
	if err := writeAtomic(path, append(data, '\n'), nil); err != nil {
		clear(key)
		return nil, err
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

func writeAtomic(path string, data []byte, rename func(oldpath, newpath string) error) error {
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
	if err := tmp.Sync(); err != nil {
		tmp.Close()
		return err
	}
	if err := tmp.Close(); err != nil {
		return err
	}
	if rename == nil {
		rename = os.Rename
	}
	var errRename error
	for attempt := 0; attempt < 8; attempt++ {
		errRename = rename(tmp.Name(), path)
		if errRename == nil {
			return nil
		}
		if errors.Is(errRename, ErrConflict) {
			return errRename
		}
		time.Sleep(jitteredDuration(time.Duration(attempt+1) * 5 * time.Millisecond))
	}
	return errRename
}
