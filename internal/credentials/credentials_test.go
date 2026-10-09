package credentials

import (
	"crypto/rand"
	"encoding/base64"
	"encoding/json"
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

	cases := map[string]func(payload []byte) []byte{
		"truncated":    func(p []byte) []byte { return p[:10] },
		"wrong prefix": func(p []byte) []byte { p[0] = 'v'; p[1] = '9'; return p },
		"flipped byte": func(p []byte) []byte { p[len(p)-1] ^= 0xff; return p },
	}
	for name, mutate := range cases {
		raw, err := os.ReadFile(path)
		if err != nil {
			t.Fatal(err)
		}
		var env envelope
		if err := json.Unmarshal(raw, &env); err != nil {
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
