package phoneremote

import (
	"fmt"
	"strings"

	"whaleisle/internal/credentials"
)

// DeviceName returns a display label for one opaque remoteDevices entry.
// Prefers name / displayName / label, then id; empty when nothing usable.
func DeviceName(device map[string]any) string {
	if device == nil {
		return ""
	}
	for _, key := range []string{"name", "displayName", "label", "id"} {
		if s := stringField(device, key); s != "" {
			return s
		}
	}
	return ""
}

// DeviceNames returns one label per active credentials.remoteDevices entry.
// Revoked records are omitted; unnamed entries fall back to "#N" so the
// returned slice length matches the active-device count.
func DeviceNames(devices []credentials.RemoteDevice) []string {
	out := make([]string, 0, len(devices))
	for _, d := range devices {
		if DeviceRevoked(d) {
			continue
		}
		if n := DeviceName(map[string]any(d)); n != "" {
			out = append(out, n)
			continue
		}
		out = append(out, fmt.Sprintf("#%d", len(out)+1))
	}
	return out
}

// DeviceRevoked mirrors dsh-remote's public device projection: any non-empty
// revokedAt marker removes a device from the active phone list.
func DeviceRevoked(device credentials.RemoteDevice) bool {
	value, ok := device["revokedAt"]
	if !ok || value == nil {
		return false
	}
	return strings.TrimSpace(fmt.Sprint(value)) != ""
}

// ActiveDevices filters revoked records while preserving the opaque payload.
func ActiveDevices(devices []credentials.RemoteDevice) []credentials.RemoteDevice {
	out := make([]credentials.RemoteDevice, 0, len(devices))
	for _, device := range devices {
		if !DeviceRevoked(device) {
			out = append(out, device)
		}
	}
	return out
}

func stringField(m map[string]any, key string) string {
	v, ok := m[key]
	if !ok || v == nil {
		return ""
	}
	switch t := v.(type) {
	case string:
		return strings.TrimSpace(t)
	case fmt.Stringer:
		return strings.TrimSpace(t.String())
	default:
		return strings.TrimSpace(fmt.Sprint(t))
	}
}
