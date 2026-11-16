package sshremote

import (
	"encoding/json"
	"errors"
	"fmt"
	"net"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"testing"

	"golang.org/x/crypto/ssh"

	"whaleisle/internal/credentials"
)

func TestNormalizeHostKeyModeFailsClosed(t *testing.T) {
	cases := []struct {
		in   string
		want HostKeyMode
	}{
		{"", HostKeyVerify},
		{"   ", HostKeyVerify},
		{"unknown-mode", HostKeyVerify},
		{"ask", HostKeyVerify},
		{"verify", HostKeyVerify},
		{"VERIFY", HostKeyVerify},
		{"strict", HostKeyVerify},
		{" STRICT ", HostKeyVerify},
		{"accept-new", HostKeyAcceptNew},
		{"Accept-New", HostKeyAcceptNew},
		{"tofu", HostKeyAcceptNew},
		{" TOFU ", HostKeyAcceptNew},
		{"off", HostKeyOff},
		{" OFF ", HostKeyOff},
	}
	for _, tc := range cases {
		if got := NormalizeHostKeyMode(tc.in); got != tc.want {
			t.Errorf("NormalizeHostKeyMode(%q) = %q, want %q", tc.in, got, tc.want)
		}
	}

	// UpsertConnection with empty/unknown hostKeyMode defaults to strict verify.
	var reg ConnectionRegistry
	if err := UpsertConnection(&reg, SavedConnection{ID: "c1", Host: "host.example"}); err != nil {
		t.Fatal(err)
	}
	if reg.List[0].HostKeyMode != string(HostKeyVerify) {
		t.Fatalf("UpsertConnection default HostKeyMode = %q, want %q", reg.List[0].HostKeyMode, HostKeyVerify)
	}

	// NewHostKeyCallback fails closed on bogus Mode while preserving TOFU options.
	path := KnownHostsPath(t.TempDir())
	_, pub := testHostKey(t)
	cb, err := NewHostKeyCallback(HostKeyOptions{Path: path, Mode: HostKeyMode("bogus")})
	if err != nil {
		t.Fatal(err)
	}
	var unk *UnknownHostKey
	if err := cb("bogus.test:22", &net.TCPAddr{IP: net.ParseIP("127.0.0.1"), Port: 22}, pub); !errors.As(err, &unk) {
		t.Fatalf("bogus mode did not fail closed with *UnknownHostKey: %v", err)
	}

	cbTofu, err := NewHostKeyCallback(HostKeyOptions{Path: path, Mode: HostKeyMode("tofu")})
	if err != nil {
		t.Fatal(err)
	}
	if err := cbTofu("tofu.test:22", &net.TCPAddr{IP: net.ParseIP("127.0.0.1"), Port: 22}, pub); err != nil {
		t.Fatalf("explicit tofu mode rejected first key: %v", err)
	}
}

func TestKnownHostsConcurrentTOFUAcceptAndForgetNoDeadlock(t *testing.T) {
	dir := t.TempDir()
	path := KnownHostsPath(dir)
	cb, err := NewHostKeyCallback(HostKeyOptions{Path: path, TrustOnFirstUse: true})
	if err != nil {
		t.Fatal(err)
	}
	remote := &net.TCPAddr{IP: net.ParseIP("127.0.0.1"), Port: 22}

	const workers = 12
	var wg sync.WaitGroup
	errs := make(chan error, workers*3)
	for i := 0; i < workers; i++ {
		wg.Add(1)
		go func(idx int) {
			defer wg.Done()
			_, pub := testHostKey(t)
			host := fmt.Sprintf("host-%d.test:22", idx)
			if err := cb(host, remote, pub); err != nil {
				errs <- fmt.Errorf("cb %s: %w", host, err)
				return
			}
			// Also test AcceptHostKey + ForgetHostKey racing concurrently with TOFU callbacks.
			if err := AcceptHostKey(path, fmt.Sprintf("extra-%d.test:22", idx), pub); err != nil {
				errs <- fmt.Errorf("accept %d: %w", idx, err)
				return
			}
			if idx%3 == 0 {
				if err := ForgetHostKey(path, host); err != nil {
					errs <- fmt.Errorf("forget %s: %w", host, err)
				}
			}
		}(i)
	}
	wg.Wait()
	close(errs)
	for err := range errs {
		t.Fatal(err)
	}
}

func TestValidateForwardEnforcesLoopbackListenHost(t *testing.T) {
	for _, bad := range []string{"0.0.0.0", "::", "[::]", "192.168.1.25", "10.0.0.5", "public.example.com", "invalid"} {
		spec := ForwardSpec{Direction: ForwardLocal, ListenHost: bad, ListenPort: 8080, TargetHost: "127.0.0.1", TargetPort: 80}
		if _, err := validateForward(spec); err == nil {
			t.Errorf("non-loopback listen host %q accepted", bad)
		}
		reverseSpec := spec
		reverseSpec.Direction = ForwardReverse
		if _, err := validateForward(reverseSpec); err == nil {
			t.Errorf("non-loopback reverse listen host %q accepted", bad)
		}
		if _, err := StartForward(nil, &ssh.Client{}, spec); err == nil {
			t.Errorf("StartForward accepted non-loopback listen host %q", bad)
		}
	}

	for _, good := range []struct {
		in   string
		want string
	}{
		{"", "127.0.0.1"},
		{"127.0.0.1", "127.0.0.1"},
		{"127.0.0.2", "127.0.0.2"},
		{"::1", "::1"},
		{"[::1]", "::1"},
		{"localhost", "localhost"},
	} {
		validated, err := validateForward(ForwardSpec{Direction: ForwardLocal, ListenHost: good.in, ListenPort: 8080, TargetHost: "127.0.0.1", TargetPort: 80})
		if err != nil {
			t.Fatalf("loopback %q rejected: %v", good.in, err)
		}
		if validated.ListenHost != good.want {
			t.Fatalf("listenHost(%q) = %q, want %q", good.in, validated.ListenHost, good.want)
		}
		if _, ip, err := validateLoopbackListenHost(good.in); err != nil || ip == nil || !ip.IsLoopback() {
			t.Fatalf("validateLoopbackListenHost(%q) = (%v, %v)", good.in, ip, err)
		}
	}
}

func TestTunnelConfigSecretsStoredEncryptedAndMigrated(t *testing.T) {
	userData := t.TempDir()

	// Missing ssh-tunnel.json creates no directories or files.
	if _, err := LoadTunnelConfig(userData); err != nil {
		t.Fatal(err)
	}
	if entries, err := os.ReadDir(userData); err != nil || len(entries) != 0 {
		t.Fatalf("LoadTunnelConfig created files on empty dir: %v %v", entries, err)
	}

	// Pre-populate unrelated user credentials to verify SSH saves preserve them.
	if err := credentials.Save(userData, credentials.Credentials{
		APIKey:      "keep-api-key",
		GithubToken: "keep-ghp",
	}); err != nil {
		t.Fatal(err)
	}

	cfg := TunnelConfig{
		Name:         "prod",
		Host:         "ssh.example.test",
		User:         "deploy",
		Port:         2222,
		IdentityFile: "/keys/id_ed25519",
		Password:     "super-secret-password",
		Passphrase:   "super-secret-passphrase",
		HostKeyMode:  "verify",
	}
	if err := SaveTunnelConfig(userData, cfg); err != nil {
		t.Fatal(err)
	}

	// On-disk ssh-tunnel.json must contain no plaintext secrets.
	diskBytes, err := os.ReadFile(ConfigPath(userData))
	if err != nil {
		t.Fatal(err)
	}
	diskText := string(diskBytes)
	if strings.Contains(diskText, cfg.Password) || strings.Contains(diskText, cfg.Passphrase) {
		t.Fatalf("plaintext secrets written to ssh-tunnel.json: %s", diskText)
	}
	var markers map[string]any
	if err := json.Unmarshal(diskBytes, &markers); err != nil {
		t.Fatal(err)
	}
	if markers["passwordSet"] != true || markers["passphraseSet"] != true {
		t.Fatalf("expected non-secret markers in ssh-tunnel.json: %s", diskText)
	}

	// Credentials file on disk must also be encrypted, not plaintext.
	credBytes, err := os.ReadFile(filepath.Join(userData, credentials.FileName))
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(string(credBytes), cfg.Password) || strings.Contains(string(credBytes), cfg.Passphrase) {
		t.Fatalf("credentials.json leaked plaintext SSH secret: %s", credBytes)
	}

	// LoadTunnelConfig backfills encrypted secrets and preserves unrelated credentials.
	loaded, err := LoadTunnelConfig(userData)
	if err != nil {
		t.Fatal(err)
	}
	if loaded.Password != cfg.Password || loaded.Passphrase != cfg.Passphrase {
		t.Fatalf("loaded secrets mismatch: %+v", loaded)
	}
	creds, err := credentials.Load(userData)
	if err != nil || creds.APIKey != "keep-api-key" || creds.GithubToken != "keep-ghp" {
		t.Fatalf("SaveTunnelConfig clobbered unrelated credentials: %+v err=%v", creds, err)
	}

	// Clearing password removes it from encrypted storage cleanly.
	loaded.Password = ""
	if err := SaveTunnelConfig(userData, loaded); err != nil {
		t.Fatal(err)
	}
	reloaded, err := LoadTunnelConfig(userData)
	if err != nil {
		t.Fatal(err)
	}
	if reloaded.Password != "" || reloaded.Passphrase != cfg.Passphrase {
		t.Fatalf("cleared password persisted or passphrase lost: %+v", reloaded)
	}

	// Legacy plaintext ssh-tunnel.json is automatically migrated and scrubbed on Load.
	legacyDir := t.TempDir()
	legacyJSON := `{"host":"legacy.test","user":"alice","port":22,"identityFile":"id","password":"legacy-tunnel-pw","passphrase":"legacy-tunnel-phrase"}`
	if err := os.WriteFile(ConfigPath(legacyDir), []byte(legacyJSON), 0o600); err != nil {
		t.Fatal(err)
	}
	migrated, err := LoadTunnelConfig(legacyDir)
	if err != nil {
		t.Fatal(err)
	}
	if migrated.Password != "legacy-tunnel-pw" || migrated.Passphrase != "legacy-tunnel-phrase" {
		t.Fatalf("legacy tunnel secrets lost during migration: %+v", migrated)
	}
	scrubbed, err := os.ReadFile(ConfigPath(legacyDir))
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(string(scrubbed), "legacy-tunnel-pw") || strings.Contains(string(scrubbed), "legacy-tunnel-phrase") {
		t.Fatalf("legacy ssh-tunnel.json not scrubbed after migration: %s", scrubbed)
	}
}

func TestConnectionRegistrySecretsMigrationCleanupAndPathBoundary(t *testing.T) {
	userData := t.TempDir()
	regPath := RegistryPath(userData)

	// Missing registry creates zero files.
	if _, err := LoadConnectionRegistry(regPath); err != nil {
		t.Fatal(err)
	}
	if entries, err := os.ReadDir(userData); err != nil || len(entries) != 0 {
		t.Fatalf("LoadConnectionRegistry mutated empty dir: %v %v", entries, err)
	}

	// Save tunnel secret alongside registry to verify both coexist inside credentials.json.
	if err := SaveTunnelConfig(userData, TunnelConfig{Host: "tun", User: "u", Port: 22, Password: "tun-pw"}); err != nil {
		t.Fatal(err)
	}

	reg := ConnectionRegistry{
		CurrentID: "m1",
		List: []SavedConnection{
			{
				ID:         "m1",
				Name:       "Prod",
				Host:       "prod.test",
				Port:       22,
				User:       "ops",
				Password:   "conn-pw-1",
				Passphrase: "conn-phrase-1",
				Proxy:      &ProxyConfig{Host: "jump.test", Port: 22, User: "jump", Password: "proxy-pw-1"},
			},
			{
				ID:       "m2",
				Name:     "Staging",
				Host:     "stage.test",
				Port:     22,
				User:     "ops",
				Password: "conn-pw-2",
			},
		},
	}
	if err := SaveConnectionRegistry(regPath, reg); err != nil {
		t.Fatal(err)
	}

	// Disk machines.json must hold zero plaintext secrets.
	diskBytes, err := os.ReadFile(regPath)
	if err != nil {
		t.Fatal(err)
	}
	for _, secret := range []string{"conn-pw-1", "conn-phrase-1", "proxy-pw-1", "conn-pw-2"} {
		if strings.Contains(string(diskBytes), secret) {
			t.Fatalf("machines.json leaked secret %q: %s", secret, diskBytes)
		}
	}

	// LoadConnectionRegistry backfills all connection & proxy secrets.
	loaded, err := LoadConnectionRegistry(regPath)
	if err != nil {
		t.Fatal(err)
	}
	if len(loaded.List) != 2 || loaded.List[0].Password != "conn-pw-1" || loaded.List[0].Passphrase != "conn-phrase-1" ||
		loaded.List[0].Proxy == nil || loaded.List[0].Proxy.Password != "proxy-pw-1" || loaded.List[1].Password != "conn-pw-2" {
		t.Fatalf("backfilled registry mismatch: %+v", loaded)
	}
	if !loaded.List[0].Sanitized().PasswordSet || !loaded.List[0].Sanitized().PassphraseSet || !loaded.List[0].Sanitized().ProxyPasswordSet {
		t.Fatalf("Sanitized flags wrong: %+v", loaded.List[0].Sanitized())
	}

	// Deleting m1 removes m1's secrets while preserving m2 and tunnel secrets.
	if !DeleteConnection(&loaded, "m1") {
		t.Fatal("DeleteConnection returned false")
	}
	if err := SaveConnectionRegistry(regPath, loaded); err != nil {
		t.Fatal(err)
	}
	bucket, ok, err := loadSSHSecrets(userData)
	if err != nil || !ok {
		t.Fatalf("loadSSHSecrets: ok=%v err=%v", ok, err)
	}
	if _, exists := bucket.Connections["m1"]; exists {
		t.Fatalf("deleted connection secret remained in credentials: %+v", bucket.Connections)
	}
	if bucket.Connections["m2"].Password != "conn-pw-2" {
		t.Fatalf("remaining connection secret lost: %+v", bucket.Connections)
	}
	if bucket.Tunnel == nil || bucket.Tunnel.Password != "tun-pw" {
		t.Fatalf("tunnel secret lost after registry update: %+v", bucket.Tunnel)
	}

	// Legacy plaintext machines.json auto-migrates and scrubs disk JSON.
	legacyUserData := t.TempDir()
	legacyRegPath := RegistryPath(legacyUserData)
	if err := os.MkdirAll(filepath.Dir(legacyRegPath), 0o700); err != nil {
		t.Fatal(err)
	}
	legacyPayload := `{"currentId":"leg1","list":[{"id":"leg1","name":"Legacy","host":"leg.test","port":22,"username":"root","password":"leg-pw","passphrase":"leg-phrase","proxy":{"host":"p.test","port":22,"password":"leg-proxy-pw"}}]}`
	if err := os.WriteFile(legacyRegPath, []byte(legacyPayload), 0o600); err != nil {
		t.Fatal(err)
	}
	migratedReg, err := LoadConnectionRegistry(legacyRegPath)
	if err != nil {
		t.Fatal(err)
	}
	if len(migratedReg.List) != 1 || migratedReg.List[0].Password != "leg-pw" || migratedReg.List[0].Passphrase != "leg-phrase" ||
		migratedReg.List[0].Proxy == nil || migratedReg.List[0].Proxy.Password != "leg-proxy-pw" {
		t.Fatalf("legacy registry migration failed: %+v", migratedReg)
	}
	scrubbedReg, err := os.ReadFile(legacyRegPath)
	if err != nil {
		t.Fatal(err)
	}
	for _, secret := range []string{"leg-pw", "leg-phrase", "leg-proxy-pw"} {
		if strings.Contains(string(scrubbedReg), secret) {
			t.Fatalf("legacy machines.json still contains %q: %s", secret, scrubbedReg)
		}
	}

	// Custom non-ssh registry paths (e.g. TempDir/machines.json) must confine credentials inside TempDir/custom, never parent dir.
	root := t.TempDir()
	customDir := filepath.Join(root, "custom")
	customRegPath := filepath.Join(customDir, "machines.json")
	if err := SaveConnectionRegistry(customRegPath, ConnectionRegistry{
		List: []SavedConnection{{ID: "k1", Host: "h.test", Port: 22, User: "u", Password: "confined-pw"}},
	}); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(filepath.Join(root, credentials.FileName)); !os.IsNotExist(err) {
		t.Fatalf("custom registry path escaped above directory into %s", root)
	}
	if _, err := os.Stat(filepath.Join(customDir, credentials.FileName)); err != nil {
		t.Fatalf("expected encrypted credentials inside %s: %v", customDir, err)
	}
	customLoaded, err := LoadConnectionRegistry(customRegPath)
	if err != nil || len(customLoaded.List) != 1 || customLoaded.List[0].Password != "confined-pw" {
		t.Fatalf("confined registry reload failed: %+v err=%v", customLoaded, err)
	}
}
