// Package sshremote implements the native SSH remote workspace surface.
//
// The package mirrors the Electron dsh-remote contract for fields shared
// through config.json and credentials.json, then owns desktop-only SSH
// pieces: host-key pinning, SSH config imports, saved connection profiles,
// TCP probes, and local/reverse port forwarding.
//
// SSH host keys are kept in userData/ssh/known_hosts. The default policy is
// accept-new (TOFU with change detection); verify and off are explicit modes.
// TunnelSession can surface UnknownHostKey for an Accept/Decline UI and refuses
// HostKeyChanged entries. Saved profiles use RegistryPath(userData), while the
// single-panel compatibility form remains ssh-tunnel.json.
package sshremote
