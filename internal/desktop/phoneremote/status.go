package phoneremote

import (
	"net"
	"net/url"
	"strconv"
	"strings"

	"whaleisle/internal/config"
)

// Readiness is a config/credentials checklist for phone remote (no mobile client).
type Readiness string

const (
	// ReadinessDisabled means pairing is off.
	ReadinessDisabled Readiness = "disabled"
	// ReadinessIncomplete means required fields for phone pairing are missing.
	ReadinessIncomplete Readiness = "incomplete"
	// ReadinessReady means config/credentials look sufficient for phone pairing.
	ReadinessReady Readiness = "ready"
)

// Snapshot is config-derived phone remote stub status.
type Snapshot struct {
	RemoteEnabled       bool
	RemoteMode          string
	RemotePort          int
	RemoteBindAddress   string
	RemoteLanTLS        bool
	RemoteAppBaseURL    string
	RemoteRelayEndpoint string
	RemoteRelayUseTLS   bool
	HasRemoteToken      bool
	HasRemoteRelayToken bool
	MuxReady            bool
	DeviceCount         int
	DeviceNames         []string
	Readiness           Readiness
	Missing             []string
	ConnectURL          string
}

// FromConfig builds a snapshot from normalized config and runtime flags.
func FromConfig(cfg config.Config, hasRemoteToken, hasRemoteRelayToken, muxReady bool, deviceCount int) Snapshot {
	s := Snapshot{
		RemoteEnabled:       cfg.RemoteEnabled,
		RemoteMode:          cfg.RemoteMode,
		RemotePort:          cfg.RemotePort,
		RemoteBindAddress:   cfg.RemoteBindAddress,
		RemoteLanTLS:        cfg.RemoteLanTls,
		RemoteAppBaseURL:    cfg.RemoteAppBaseUrl,
		RemoteRelayEndpoint: cfg.RemoteRelayEndpoint,
		RemoteRelayUseTLS:   cfg.RemoteRelayUseTls,
		HasRemoteToken:      hasRemoteToken,
		HasRemoteRelayToken: hasRemoteRelayToken,
		MuxReady:            muxReady,
		DeviceCount:         deviceCount,
	}
	s.Readiness, s.Missing = Assess(s)
	s.ConnectURL = ConnectURL(s)
	return s
}

// Assess returns readiness and missing checklist items for phone remote.
func Assess(s Snapshot) (Readiness, []string) {
	var missing []string
	if !s.RemoteEnabled {
		missing = append(missing, "remoteEnabled")
	}
	if !s.HasRemoteToken {
		missing = append(missing, "remoteToken")
	}
	mode := normalizeMode(s.RemoteMode)
	switch mode {
	case "lan":
		if s.RemotePort < 1024 || s.RemotePort > 65535 {
			missing = append(missing, "remotePort")
		}
		if strings.TrimSpace(s.RemoteBindAddress) == "" {
			missing = append(missing, "remoteBindAddress")
		}
	case "relay":
		if !s.HasRemoteRelayToken {
			missing = append(missing, "remoteRelayToken")
		}
	}
	if !s.MuxReady {
		missing = append(missing, "mux")
	}
	if len(missing) > 0 {
		if !s.RemoteEnabled {
			return ReadinessDisabled, missing
		}
		return ReadinessIncomplete, missing
	}
	return ReadinessReady, nil
}

// ConnectURL returns only an explicitly configured, credential-free mobile
// landing page. A daemon listening endpoint is not a mobile web server. This
// does not create a pairing offer or establish that the service is running.
func ConnectURL(s Snapshot) string {
	base := strings.TrimSpace(s.RemoteAppBaseURL)
	u, err := url.Parse(base)
	if err != nil || u == nil || (u.Scheme != "http" && u.Scheme != "https") || u.Hostname() == "" || u.User != nil || u.RawQuery != "" || u.ForceQuery || u.Fragment != "" || strings.Contains(base, "#") {
		return ""
	}
	if ip := net.ParseIP(u.Hostname()); ip != nil && ip.IsUnspecified() {
		return ""
	}
	if port := u.Port(); port != "" {
		n, err := strconv.Atoi(port)
		if err != nil || n < 1 || n > 65535 {
			return ""
		}
	}
	return strings.TrimRight(u.String(), "/")
}

func normalizeMode(mode string) string {
	m := strings.ToLower(strings.TrimSpace(mode))
	switch m {
	case "lan", "relay", "ssh":
		return m
	default:
		if strings.Contains(m, "ssh") {
			return "ssh"
		}
		if strings.Contains(m, "lan") {
			return "lan"
		}
		return "relay"
	}
}
