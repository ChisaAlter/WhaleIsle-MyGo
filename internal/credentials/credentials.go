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
// macOS writes a distinct native envelope backed by a per-profile Keychain
// key. Electron macOS safeStorage CBC migration is not implemented. Linux
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
	path := filepath.Join(dir, FileName)
	data, err := os.ReadFile(path)
	if errors.Is(err, fs.ErrNotExist) {
		return Credentials{}, nil
	}
	if err != nil {
		return Credentials{}, err
	}
	kind, err := classifyDocument(data)
	if err != nil {
		return Credentials{}, fmt.Errorf("credentials: %s: %w", path, err)
	}
	if kind == docEnvelope {
		return decryptEnvelope(dir, data)
	}
	var creds Credentials
	if err := decodeCredentialsObject(data, &creds); err != nil {
		return Credentials{}, fmt.Errorf("credentials: %s: %w", path, err)
	}
	return creds, nil
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
	var doc json.RawMessage
	if err := json.Unmarshal(data, &doc); err != nil {
		return ErrInvalidCredentialsFormat
	}
	trimmed := bytes.TrimSpace(doc)
	if len(trimmed) == 0 || trimmed[0] != '{' {
		return ErrInvalidCredentialsFormat
	}
	var creds Credentials
	if err := json.Unmarshal(trimmed, &creds); err != nil {
		return ErrInvalidCredentialsFormat
	}
	*out = creds
	return nil
}

// Save writes an encrypted envelope using the platform storage key. Windows
// retains safeStorage-v1; macOS uses a distinct Keychain-backed format. The write is atomic (temp file +
// rename in the same directory).
func Save(dir string, creds Credentials) error {
	// Never create a replacement key for an existing native envelope. A
	// missing/locked Keychain item must leave the previous file recoverable.
	create := true
	if credentialOS.version == KeychainEnvelopeVersion {
		existing, readErr := os.ReadFile(filepath.Join(dir, FileName))
		if readErr != nil && !errors.Is(readErr, fs.ErrNotExist) {
			return readErr
		}
		if readErr == nil {
			kind, classifyErr := classifyDocument(existing)
			if classifyErr != nil {
				return classifyErr
			}
			if kind == docEnvelope {
				var env envelope
				if err := json.Unmarshal(existing, &env); err != nil {
					return err
				}
				if env.Version == KeychainEnvelopeVersion {
					create = false
				}
			}
		}
	}
	key, err := credentialOS.key(dir, create)
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
	data, err := json.MarshalIndent(envelope{Version: credentialOS.version, Payload: base64.StdEncoding.EncodeToString(payload)}, "", "  ")
	if err != nil {
		return err
	}
	return writeAtomic(filepath.Join(dir, FileName), data)
}

// decryptEnvelope decodes a document already classified as docEnvelope. The
// payload is validated before the Local State key is read.
func decryptEnvelope(dir string, data []byte) (Credentials, error) {
	path := filepath.Join(dir, FileName)
	var env envelope
	if err := json.Unmarshal(data, &env); err != nil {
		return Credentials{}, fmt.Errorf("credentials: %s: %w", path, ErrInvalidCredentialsFormat)
	}
	blob, err := base64.StdEncoding.DecodeString(env.Payload)
	if err != nil {
		return Credentials{}, fmt.Errorf("credentials: %s: %w", path, ErrInvalidCredentialsFormat)
	}
	var key []byte
	if env.Version == EnvelopeVersion {
		key, err = loadLocalStateKey(dir)
	} else if env.Version == credentialOS.version {
		key, err = credentialOS.key(dir, false)
	} else {
		return Credentials{}, ErrUnsupportedEnvelopeVersion
	}
	if err != nil {
		return Credentials{}, err
	}
	plaintext, err := open(key, blob)
	if err != nil {
		return Credentials{}, err
	}
	var creds Credentials
	if err := decodeCredentialsObject(plaintext, &creds); err != nil {
		return Credentials{}, fmt.Errorf("credentials: %s: decrypted envelope: %w", path, err)
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
