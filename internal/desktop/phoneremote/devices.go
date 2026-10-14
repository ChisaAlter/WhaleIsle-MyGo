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

// DeviceNames returns one label per credentials.remoteDevices entry.
// Unnamed entries fall back to "#N" so the slice length matches DeviceCount.
func DeviceNames(devices []credentials.RemoteDevice) []string {
	out := make([]string, len(devices))
	for i, d := range devices {
		if n := DeviceName(map[string]any(d)); n != "" {
			out[i] = n
			continue
		}
		out[i] = fmt.Sprintf("#%d", i+1)
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
