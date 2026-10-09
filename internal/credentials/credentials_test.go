package credentials

import (
	"bytes"
	"crypto/rand"
	"encoding/base64"
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"
)

func sampleCreds() Credentials {
	return Credentials{
		APIKey:           "sk-test-123",
		BaseURL:          "http://127.0.0.1:3080",
		GithubToken:      "ghp_test",
		RemoteToken:      "remote-tok",
		RemoteRelayToken: "relay-tok",
		RemoteDevices: []RemoteDevice{
			{"id": "abcdef1234567890", "name": "iPhone", "token": "dev-tok", "createdAt": "2026-01-01T00:00:00Z", "lastSeenAt": "2026-01-02T00:00:00Z"},
		},
	}
}

// writeLocalState synthesizes a Local State whose os_crypt.encrypted_key is
// a fresh random key protected with the real DPAPI.
func writeLocalState(t *testing.T, dir string) {
	t.Helper()
	key := make([]byte, 32)
	if _, err := rand.Read(key); err != nil {
		t.Fatal(err)
	}
	protected, err := dpapiProtect(key)
	if err != nil {
		t.Fatalf("CryptProtectData unavailable or failed: %v", err)
	}
	blob := append([]byte("DPAPI"), protected...)
	state := map[string]any{
		"os_crypt": map[string]any{
			"encrypted_key": base64.StdEncoding.EncodeToString(blob),
		},
	}
	data, err := json.Marshal(state)
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, LocalStateName), data, 0o600); err != nil {
		t.Fatal(err)
	}
}

func TestDpapiRoundTrip(t *testing.T) {
	plain := []byte("whaleisle dpapi round trip")
	protected, err := dpapiProtect(plain)
	if err != nil {
		t.Skipf("DPAPI protect unavailable in this environment: %v", err)
	}
	got, err := dpapiUnprotect(protected)
	if err != nil {
		t.Fatalf("unprotect: %v", err)
	}
	if string(got) != string(plain) {
		t.Fatalf("round trip mismatch: %q", got)
	}
}

func TestLoadMissingFile(t *testing.T) {
	creds, err := Load(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(creds, Credentials{}) {
		t.Fatalf("missing file must give zero credentials, got %+v", creds)
	}
}

func TestLoadPlainJSON(t *testing.T) {
	dir := t.TempDir()
	want := sampleCreds()
	data, err := json.Marshal(want)
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, FileName), data, 0o600); err != nil {
		t.Fatal(err)
	}
	creds, err := Load(dir)
	if err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(creds, want) {
		t.Fatalf("got %+v, want %+v", creds, want)
	}
}

func TestEnvelopeSaveLoadRoundTrip(t *testing.T) {
	dir := t.TempDir()
	writeLocalState(t, dir)
	want := sampleCreds()

	if err := Save(dir, want); err != nil {
		t.Fatalf("save: %v", err)
	}

	// On disk: envelope only, no plaintext secrets.
	raw, err := os.ReadFile(filepath.Join(dir, FileName))
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(string(raw), "sk-test-123") || strings.Contains(string(raw), "ghp_test") {
		t.Fatalf("plaintext leaked to disk: %s", raw)
	}
	var env envelope
	if err := json.Unmarshal(raw, &env); err != nil {
		t.Fatal(err)
	}
	if env.Version != EnvelopeVersion || env.Payload == "" {
		t.Fatalf("bad envelope on disk: %s", raw)
	}
	blob, err := base64.StdEncoding.DecodeString(env.Payload)
	if err != nil {
		t.Fatalf("payload not base64: %v", err)
	}
	if string(blob[:3]) != v10Prefix {
		t.Fatalf("payload lacks %q prefix: %q", v10Prefix, blob[:3])
	}

	creds, err := Load(dir)
	if err != nil {
		t.Fatalf("load: %v", err)
	}
	if !reflect.DeepEqual(creds, want) {
		t.Fatalf("got %+v, want %+v", creds, want)
	}
}

func TestSaveEnvelopeDecryptsWithSyntheticKey(t *testing.T) {
	// Decrypt the saved payload by hand with the key the test put into
	// Local State, proving the format matches the os_crypt scheme.
	dir := t.TempDir()
	key := make([]byte, 32)
	if _, err := rand.Read(key); err != nil {
		t.Fatal(err)
	}
	protected, err := dpapiProtect(key)
	if err != nil {
		t.Skipf("DPAPI protect unavailable: %v", err)
	}
	state, _ := json.Marshal(map[string]any{
		"os_crypt": map[string]any{
			"encrypted_key": base64.StdEncoding.EncodeToString(append([]byte("DPAPI"), protected...)),
		},
	})
	if err := os.WriteFile(filepath.Join(dir, LocalStateName), state, 0o600); err != nil {
		t.Fatal(err)
	}
	want := sampleCreds()
	if err := Save(dir, want); err != nil {
		t.Fatal(err)
	}

	data, err := os.ReadFile(filepath.Join(dir, FileName))
	if err != nil {
		t.Fatal(err)
	}
	var env envelope
	if err := json.Unmarshal(data, &env); err != nil {
		t.Fatal(err)
	}
	blob, err := base64.StdEncoding.DecodeString(env.Payload)
	if err != nil {
		t.Fatal(err)
	}
	plaintext, err := open(key, blob)
	if err != nil {
		t.Fatalf("hand decryption with the Local State key failed: %v", err)
	}
	var creds Credentials
	if err := json.Unmarshal(plaintext, &creds); err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(creds, want) {
		t.Fatalf("got %+v, want %+v", creds, want)
	}
}

func TestTamperedPayloadFailsClearly(t *testing.T) {
	dir := t.TempDir()
	writeLocalState(t, dir)
	if err := Save(dir, sampleCreds()); err != nil {
		t.Fatal(err)
	}
	path := filepath.Join(dir, FileName)

	// Keep the pristine envelope so every case mutates a fresh copy rather
	// than compounding the previous mutation.
	pristine, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}

	cases := map[string]func(payload []byte) []byte{
		"truncated":    func(p []byte) []byte { return p[:10] },
		"wrong prefix": func(p []byte) []byte { p[0] = 'v'; p[1] = '9'; return p },
		"flipped byte": func(p []byte) []byte { p[len(p)-1] ^= 0xff; return p },
	}
	for name, mutate := range cases {
		var env envelope
		if err := json.Unmarshal(pristine, &env); err != nil {
			t.Fatal(err)
		}
		payload, err := base64.StdEncoding.DecodeString(env.Payload)
		if err != nil {
			t.Fatal(err)
		}
		env.Payload = base64.StdEncoding.EncodeToString(mutate(payload))
		data, err := json.Marshal(env)
		if err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(path, data, 0o600); err != nil {
			t.Fatal(err)
		}
		_, err = Load(dir)
		if err == nil {
			t.Fatalf("%s: tampered payload loaded without error", name)
		}
	}
}

func TestLoadNotBase64Payload(t *testing.T) {
	dir := t.TempDir()
	writeLocalState(t, dir)
	env, _ := json.Marshal(envelope{Version: EnvelopeVersion, Payload: "!!!not base64!!!"})
	if err := os.WriteFile(filepath.Join(dir, FileName), env, 0o600); err != nil {
		t.Fatal(err)
	}
	_, err := Load(dir)
	if err == nil {
		t.Fatal("non-base64 payload loaded without error")
	}
}

// TestLoadClassifiesCredentialDocuments pins how Load tells a plaintext
// credentials object, a supported envelope and a rejected document apart.
// Every rejected load must return zero credentials, leave the file
// byte-for-byte untouched, and never echo secret-looking values in its error.
func TestLoadClassifiesCredentialDocuments(t *testing.T) {
	const marker = "MARKER-SECRET-9f2c"
	// A payload that decodes to a short v10 blob: the envelope is structurally
	// valid, so classification reaches the (missing) Local State key.
	validBlob := base64.StdEncoding.EncodeToString([]byte(v10Prefix + "0123456789ab" + "0123456789abcdef"))

	cases := []struct {
		name      string
		doc       string
		wantOK    bool
		wantErrIs error // errors.Is target; nil means "rejected, class not pinned"
		want      Credentials
	}{
		{
			name:   "valid plaintext object",
			doc:    `{"apiKey":"` + marker + `","baseUrl":"http://127.0.0.1:3080"}`,
			wantOK: true,
			want:   Credentials{APIKey: marker, BaseURL: "http://127.0.0.1:3080"},
		},
		{
			name:      "unknown envelope version",
			doc:       `{"version":"safeStorage-v2","payload":"AAAA"}`,
			wantErrIs: ErrUnsupportedEnvelopeVersion,
		},
		{
			name:      "missing version",
			doc:       `{"payload":"AAAA"}`,
			wantErrIs: ErrInvalidCredentialsFormat,
		},
		{
			name:      "missing payload",
			doc:       `{"version":"safeStorage-v1"}`,
			wantErrIs: ErrInvalidCredentialsFormat,
		},
		{
			name:      "version wrong type",
			doc:       `{"version":123,"payload":"AAAA"}`,
			wantErrIs: ErrInvalidCredentialsFormat,
		},
		{
			name:      "payload wrong type",
			doc:       `{"version":"safeStorage-v1","payload":123}`,
			wantErrIs: ErrInvalidCredentialsFormat,
		},
		{
			name:      "empty version",
			doc:       `{"version":"","payload":"AAAA"}`,
			wantErrIs: ErrInvalidCredentialsFormat,
		},
		{
			name:      "empty payload",
			doc:       `{"version":"safeStorage-v1","payload":""}`,
			wantErrIs: ErrInvalidCredentialsFormat,
		},
		{
			name:      "payload not base64",
			doc:       `{"version":"safeStorage-v1","payload":"!!` + marker + `!!"}`,
			wantErrIs: ErrInvalidCredentialsFormat,
		},
		{
			name:      "null document",
			doc:       `null`,
			wantErrIs: ErrInvalidCredentialsFormat,
		},
		{
			name:      "array document",
			doc:       `[1,2,3]`,
			wantErrIs: ErrInvalidCredentialsFormat,
		},
		{
			name:      "string document",
			doc:       `"` + marker + `"`,
			wantErrIs: ErrInvalidCredentialsFormat,
		},
		{
			name:      "invalid json",
			doc:       `{"apiKey":`,
			wantErrIs: ErrInvalidCredentialsFormat,
		},
		{
			name:      "envelope markers with malformed payload and credential fields",
			doc:       `{"version":"safeStorage-v1","payload":"!!` + marker + `!!","apiKey":"` + marker + `"}`,
			wantErrIs: ErrInvalidCredentialsFormat,
		},
		{
			// Structurally a valid envelope, so it is decrypted, not treated
			// as plaintext; the missing Local State rejects it without ever
			// exposing the credential fields riding along in the document.
			name: "envelope markers with credential fields",
			doc:  `{"version":"safeStorage-v1","payload":"` + validBlob + `","apiKey":"` + marker + `"}`,
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			dir := t.TempDir()
			path := filepath.Join(dir, FileName)
			original := []byte(tc.doc)
			if err := os.WriteFile(path, original, 0o600); err != nil {
				t.Fatal(err)
			}

			creds, err := Load(dir)

			after, readErr := os.ReadFile(path)
			if readErr != nil {
				t.Fatal(readErr)
			}
			if !bytes.Equal(original, after) {
				t.Fatalf("Load modified the file: %q -> %q", original, after)
			}

			if tc.wantOK {
				if err != nil {
					t.Fatalf("unexpected error: %v", err)
				}
				if !reflect.DeepEqual(creds, tc.want) {
					t.Fatalf("got %+v, want %+v", creds, tc.want)
				}
				return
			}

			if err == nil {
				t.Fatal("document was accepted but must be rejected")
			}
			if tc.wantErrIs != nil && !errors.Is(err, tc.wantErrIs) {
				t.Fatalf("error %v does not match %v", err, tc.wantErrIs)
			}
			if !reflect.DeepEqual(creds, Credentials{}) {
				t.Fatalf("rejected load returned partial credentials: %+v", creds)
			}
			if strings.Contains(err.Error(), marker) {
				t.Fatalf("error leaked a secret-looking value: %v", err)
			}
		})
	}
}

func TestLoadMissingLocalState(t *testing.T) {
	dir := t.TempDir()
	env, _ := json.Marshal(envelope{Version: EnvelopeVersion, Payload: base64.StdEncoding.EncodeToString([]byte(v10Prefix + "short"))})
	if err := os.WriteFile(filepath.Join(dir, FileName), env, 0o600); err != nil {
		t.Fatal(err)
	}
	_, err := Load(dir)
	if err == nil || !strings.Contains(err.Error(), "Local State") {
		t.Fatalf("want missing-Local-State error, got: %v", err)
	}
}
