// Package sshremote documents the SSH remote workspace surface for M9.
//
// Electron parity: settings partition remote-workspace and the dsh-remote
// harness overlay (see docs/plan.md M9). The Go shell does not run an SSH
// client or tunnel yet; this package names the config.json and
// credentials.json fields the future SSH path will read.
//
// # config.Config (config.json)
//
//   - remoteWorkspaceEnabled — mount dsh-remote at harness boot (overlay).
//   - remoteEnabled — desktop remote pairing daemon (phone/LAN/SSH tunnel).
//   - remoteMode — "relay", "lan", or "ssh" (any value containing "ssh" is
//     treated as SSH-oriented for the native stub panel).
//   - remotePort — pairing daemon listen port (1024–65535).
//   - remoteBindAddress — bind address for the daemon (loopback default).
//   - remoteLanTls — TLS for LAN mode (ignored for relay; documented for SSH stub).
//   - remoteRelayUrl, remoteRelayEndpoint, remoteRelayUseTls, remoteAppBaseUrl —
//     relay / public landing host (primary for relay mode, not SSH).
//
// # credentials.Credentials (credentials.json)
//
//   - remoteToken — pairing token for remote clients.
//   - remoteRelayToken — relay registration token when using built-in relay.
//   - remoteDevices — paired device records (opaque JSON objects).
//
// Host keys are not stored in config.json. TunnelSession pins SSH host keys
// in userData/ssh/known_hosts (OpenSSH format) via HostKeyCallback: unknown
// hosts default to Trust-On-First-Use (TOFU); mismatches fail with
// *HostKeyChanged. SetTrustOnFirstUse(false) surfaces *UnknownHostKey for a
// future UI prompt + AcceptHostKey. Workspace trust may still reference the
// user .ssh directory via internal/desktop/workspaceauth.
package sshremote
