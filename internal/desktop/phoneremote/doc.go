// Package phoneremote implements the desktop-side phone remote bridge.
//
// Runtime wraps the installed original DshdRemote implementation for offer v2
// pairing, observed relay state, paired-device management, and session-scoped
// Git forwarding. Configuration readiness remains a separate checklist and
// never proves live Android connectivity. Daemon listen endpoints must never
// be presented as mobile web entry points.
//
// # config.Config (config.json)
//
//   - remoteEnabled — pairing daemon for phone/LAN remote.
//   - remoteMode — "relay", "lan", or "ssh" (SSH tunnel UI lives in sshremote).
//   - remotePort / remoteBindAddress / remoteLanTls — LAN listen endpoint.
//   - remoteAppBaseUrl — public SPA landing host for Away-mode QR.
//   - remoteRelayEndpoint / remoteRelayUseTls — relay topology (status only).
//
// # credentials.Credentials (credentials.json)
//
//   - remoteToken — pairing token for phone clients.
//   - remoteRelayToken — relay registration when using built-in/custom relay.
//   - remoteDevices — paired device records (count + display names in UI).
package phoneremote
