package main

import (
	"os"
	"os/exec"
	"path/filepath"
	"testing"
)

func TestRealModelModesRequireExplicitAuthorizationAndDoNotAdmitSecretsInPreflight(t *testing.T) {
	root := t.TempDir()
	secret, ledger := filepath.Join(root, "explicit-secret"), filepath.Join(root, "ledger.json")
	for _, tc := range []struct {
		name, secret, ledger string
		real, preflight, ok  bool
	}{
		{"preflight", "", "", false, true, true},
		{"real", secret, ledger, true, false, true},
		{"neither", "", "", false, false, false},
		{"both", secret, ledger, true, true, false},
		{"preflight-secret", secret, "", false, true, false},
		{"preflight-ledger", "", ledger, false, true, false},
		{"real-no-secret", "", ledger, true, false, false},
		{"relative-secret", "relative-key", ledger, true, false, false},
		{"same-file", ledger, ledger, true, false, false},
	} {
		t.Run(tc.name, func(t *testing.T) {
			if err := validateRealMode(tc.secret, tc.ledger, tc.real, tc.preflight); (err == nil) != tc.ok {
				t.Fatalf("mode accepted=%v, want %v", err == nil, tc.ok)
			}
		})
	}
}

func TestRealModelGuardLocalContracts(t *testing.T) {
	node, err := exec.LookPath("node")
	if err != nil {
		t.Fatal("Node is required for the final-wire guard contract tests")
	}
	script, err := filepath.Abs("real_guard.test.cjs")
	if err != nil {
		t.Fatal(err)
	}
	cmd := exec.Command(node, "--test", script)
	// No ambient API keys, user Node preloads or existing application settings.
	for _, key := range []string{"SystemRoot", "WINDIR", "PATH", "PATHEXT", "TEMP", "TMP"} {
		if value := os.Getenv(key); value != "" {
			cmd.Env = append(cmd.Env, key+"="+value)
		}
	}
	if output, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("local guard contracts: %v\n%s", err, output)
	}
}
