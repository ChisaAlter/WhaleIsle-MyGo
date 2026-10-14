// Package phoneremote documents the M11 phone remote surface.
//
// Electron parity: phone/LAN pairing via remoteEnabled + remoteMode
// (relay|lan) and credentials.remoteToken / remoteRelayToken. The Go shell
// does not ship a mobile client; this package assesses config readiness,
// builds a connect URL for the remote panel, and labels credentials.remoteDevices.
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
