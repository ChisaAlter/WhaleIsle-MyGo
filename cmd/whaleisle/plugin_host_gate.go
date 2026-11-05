package main

import (
	"os"
	"strings"
)

// The native plugin hosts (navigation with root election and public slots, and
// the per-session host) show contributions of web plugins next to the native UI.
// They keep one hidden browser page of the Harness web client per window, so
// WHALEISLE_NATIVE_PLUGIN_HOST=0 switches them off, for example to rule them
// out when diagnosing a problem.
//
// Entries of the Harness web bundle itself are told from plugin entries by their
// registrant (see whaleBundledRegistrant in pluginui/public_slots.js). A plugin
// must declare its own name in its client module (exports.name) to be told
// apart: an unnamed plugin gets the same default registrant as the bundle and
// is treated as part of it.
func nativePluginHostEnabled() bool {
	return strings.TrimSpace(os.Getenv("WHALEISLE_NATIVE_PLUGIN_HOST")) != "0"
}
