package startuprec

import "testing"

// TestGenericCauseMatchesNativeForensics pins plugin-forensics.js:12-16: the
// OOM word alone, listen EACCES, EADDRINUSE and the ENOENT form for a missing
// Node all classify the same way the native shell does.
func TestGenericCauseMatchesNativeForensics(t *testing.T) {
	cases := []struct{ text, want string }{
		{"FATAL ERROR: Reached heap limit Allocation failed", "oom"},
		{"Killed: oom", "oom"},
		{"js heap out of memory", "oom"},
		{"Error: listen EACCES: permission denied 0.0.0.0:443", "port-excluded"},
		{"Error: listen EADDRINUSE: address already in use", "port-in-use"},
		{"Error: ENOENT: no such file or directory, open node", "missing-node"},
		{"spawn ENOENT when running node.exe", "missing-node"},
		{`'node' is not recognized as an internal or external command`, "missing-node"},
		{"TypeError: cannot read property of undefined", ""},
	}
	for _, c := range cases {
		if got := genericCause(c.text); got != c.want {
			t.Errorf("genericCause(%.60q) = %q, want %q", c.text, got, c.want)
		}
	}
	// Native's GENERIC_OOM ends in \b, so a bare word ending in "oom" also
	// classifies; keep that parity rather than inventing a stricter rule.
	if got := genericCause("room service"); got != "oom" {
		t.Errorf("genericCause(room service) = %q, want native's oom", got)
	}
	if got := genericCause("zoomed view"); got != "" {
		t.Errorf("genericCause(zoomed view) = %q, want no match", got)
	}
}
