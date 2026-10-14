// Package phoneremote documents the M11 phone remote surface.
//
// This package assesses configuration only; the Go shell has not integrated
// the offer v2 pairing runtime or relay-connected state. It validates a
// credential-free mobile landing URL and labels credentials.remoteDevices.
// Daemon listen endpoints must never be presented as mobile web entry points.
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
