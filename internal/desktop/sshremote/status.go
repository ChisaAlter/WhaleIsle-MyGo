package sshremote

import (
	"strings"

	"whaleisle/internal/config"
)

// ModeUsesSSH reports whether remoteMode selects the SSH remote workspace stub.
func ModeUsesSSH(mode string) bool {
	return strings.Contains(strings.ToLower(strings.TrimSpace(mode)), "ssh")
}

// Readiness is a config/credentials checklist for SSH mode (no live SSH probe).
type Readiness string

const (
	// ReadinessInactive means remoteMode is not SSH-oriented.
	ReadinessInactive Readiness = "inactive"
	// ReadinessDisabled means SSH mode is selected but pairing/workspace is off.
	ReadinessDisabled Readiness = "disabled"
	// ReadinessIncomplete means required fields for an SSH pairing path are missing.
	ReadinessIncomplete Readiness = "incomplete"
	// ReadinessReady means config/credentials look sufficient to start pairing.
	ReadinessReady Readiness = "ready"
)

// Snapshot is config-derived SSH stub status (no live SSH session probe).
type Snapshot struct {
	Active                 bool
	RemoteWorkspaceEnabled bool
	RemoteEnabled          bool
	RemotePort             int
	RemoteBindAddress      string
	RemoteLanTLS           bool
	HasRemoteToken         bool
	HasRemoteRelayToken    bool
	Readiness              Readiness
	Missing                []string
}

// FromConfig builds a snapshot from normalized config and credential flags.
func FromConfig(cfg config.Config, hasRemoteToken, hasRemoteRelayToken bool) Snapshot {
	s := Snapshot{
		Active:                 ModeUsesSSH(cfg.RemoteMode),
		RemoteWorkspaceEnabled: cfg.RemoteWorkspaceEnabled,
		RemoteEnabled:          cfg.RemoteEnabled,
		RemotePort:             cfg.RemotePort,
		RemoteBindAddress:      cfg.RemoteBindAddress,
		RemoteLanTLS:           cfg.RemoteLanTls,
		HasRemoteToken:         hasRemoteToken,
		HasRemoteRelayToken:    hasRemoteRelayToken,
	}
	s.Readiness, s.Missing = Assess(s)
	return s
}

// Assess returns readiness and missing checklist items for an SSH snapshot.
func Assess(s Snapshot) (Readiness, []string) {
	if !s.Active {
		return ReadinessInactive, nil
	}
	var missing []string
	if !s.RemoteWorkspaceEnabled {
		missing = append(missing, "remoteWorkspaceEnabled")
	}
	if !s.RemoteEnabled {
		missing = append(missing, "remoteEnabled")
	}
	if s.RemotePort < 1024 || s.RemotePort > 65535 {
		missing = append(missing, "remotePort")
	}
	if strings.TrimSpace(s.RemoteBindAddress) == "" {
		missing = append(missing, "remoteBindAddress")
	}
	if !s.HasRemoteToken {
		missing = append(missing, "remoteToken")
	}
	if len(missing) > 0 {
		if !s.RemoteWorkspaceEnabled || !s.RemoteEnabled {
			return ReadinessDisabled, missing
		}
		return ReadinessIncomplete, missing
	}
	return ReadinessReady, nil
}
