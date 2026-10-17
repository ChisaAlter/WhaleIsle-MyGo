// Package phoneremote implements the phone/LAN pairing surface.
//
// It assesses config and credentials readiness, builds relay or LAN connect
// URLs, filters active remoteDevices, and serves the copied mobile/web SPA on
// the reference port 3180. The package does not claim Android SDK/device
// validation; those remain platform acceptance work.
package phoneremote
