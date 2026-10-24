package main

import (
	"strings"

	"whaleisle/internal/config"
)

// remoteRuntimeCredentials carries the phone-remote tokens loaded from the
// credential store. They are secrets: never log them and never copy them into
// config.json (which strips remoteToken/remoteRelayToken on save).
type remoteRuntimeCredentials struct {
	// Local is credentials.RemoteToken, the gateway's own pairing token
	// (native remote.js:328 reads config.remoteToken).
	Local string
	// Relay is credentials.RemoteRelayToken, the relay host token
	// (native remote.js:318 reads config.remoteRelayToken).
	Relay string
}

// buildRemoteRuntimeConfig is the exact config map handed to the dshd-remote
// daemon. It is pure so the wiring can be tested without a daemon: tokens are
// added only when one is saved, and an absent token leaves the key unset
// instead of passing an empty string.
func buildRemoteRuntimeConfig(cfg config.Config, creds remoteRuntimeCredentials, gitRoots []string, harnessOrigin, harnessCookie string) map[string]any {
	runtimeConfig := map[string]any{
		"remoteGitEnabled": len(gitRoots) > 0, "remoteGitRoots": gitRoots,
		"remoteEnabled": cfg.RemoteEnabled, "remoteMode": cfg.RemoteMode,
		"remotePort": cfg.RemotePort, "remoteBindAddress": cfg.RemoteBindAddress,
		"remoteLanTls": cfg.RemoteLanTls, "remoteAppBaseUrl": cfg.RemoteAppBaseUrl,
		"remoteRelayEndpoint": cfg.RemoteRelayEndpoint, "remoteRelayUseTls": cfg.RemoteRelayUseTls,
		"harnessOrigin": harnessOrigin, "harnessCookie": harnessCookie,
	}
	if token := strings.TrimSpace(creds.Local); token != "" {
		runtimeConfig["remoteToken"] = token
	}
	if token := strings.TrimSpace(creds.Relay); token != "" {
		runtimeConfig["remoteRelayToken"] = token
	}
	return runtimeConfig
}
