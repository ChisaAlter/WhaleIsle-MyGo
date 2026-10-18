package sshremote

import (
	"testing"

	"whaleisle/internal/config"
)

func TestModeUsesSSH(t *testing.T) {
	tests := []struct {
		mode string
		want bool
	}{
		{"ssh", true},
		{"SSH", true},
		{"ssh-tunnel", true},
		{"relay", false},
		{"lan", false},
		{"", false},
	}
	for _, tc := range tests {
		if got := ModeUsesSSH(tc.mode); got != tc.want {
			t.Errorf("ModeUsesSSH(%q) = %v, want %v", tc.mode, got, tc.want)
		}
	}
}

func TestFromConfigActiveWhenSSHMode(t *testing.T) {
	cfg := config.Defaults()
	cfg.RemoteMode = "ssh"
	cfg.RemoteEnabled = true
	cfg.RemoteWorkspaceEnabled = true
	cfg.RemoteBindAddress = "127.0.0.1"
	s := FromConfig(cfg, true, false)
	if !s.Active {
		t.Fatal("expected Active for ssh mode")
	}
	if s.Readiness != ReadinessReady {
		t.Fatalf("Readiness = %q missing=%v", s.Readiness, s.Missing)
	}
}

func TestAssessIncompleteWithoutToken(t *testing.T) {
	s := Snapshot{
		Active:                 true,
		RemoteWorkspaceEnabled: true,
		RemoteEnabled:          true,
		RemotePort:             8787,
		RemoteBindAddress:      "127.0.0.1",
		HasRemoteToken:         false,
	}
	r, missing := Assess(s)
	if r != ReadinessIncomplete {
		t.Fatalf("Readiness = %q", r)
	}
	if len(missing) == 0 {
		t.Fatal("expected missing remoteToken")
	}
}

func TestAssessDisabledWhenPairingOff(t *testing.T) {
	s := Snapshot{
		Active:                 true,
		RemoteWorkspaceEnabled: true,
		RemoteEnabled:          false,
		RemotePort:             8787,
		RemoteBindAddress:      "127.0.0.1",
		HasRemoteToken:         true,
	}
	r, _ := Assess(s)
	if r != ReadinessDisabled {
		t.Fatalf("Readiness = %q", r)
	}
}
