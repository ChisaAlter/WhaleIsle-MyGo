package phoneremote

import (
	"strings"
	"testing"
)

// The staged daemon may be the installed Electron binary (its shipped
// better-sqlite3 is Electron ABI 149, runtime/node is 137). Electron only runs
// as Node with ELECTRON_RUN_AS_NODE=1, so the child environment must carry it;
// plain Node ignores the variable. An explicit caller value is never replaced.
func TestDaemonEnvSuppliesElectronRunAsNode(t *testing.T) {
	base := []string{"PATH=C:\\Windows", "DSH_DESKTOP_INSTALL_TOKEN=secret-install-token"}
	got := daemonEnv(base)
	found := ""
	for _, entry := range got {
		if strings.HasPrefix(entry, electronRunAsNodeEnv+"=") {
			found = entry
		}
	}
	if found != electronRunAsNodeEnv+"=1" {
		t.Fatalf("daemon env = %v, want %s=1", got, electronRunAsNodeEnv)
	}
	// Everything else is preserved verbatim, and the input slice is not mutated.
	if len(got) != len(base)+1 {
		t.Fatalf("daemon env length = %d, want %d", len(got), len(base)+1)
	}
	for i, entry := range base {
		if got[i] != entry {
			t.Fatalf("daemon env[%d] = %q, want %q", i, got[i], entry)
		}
	}
	if base[len(base)-1] != "DSH_DESKTOP_INSTALL_TOKEN=secret-install-token" {
		t.Fatal("daemonEnv mutated its input slice")
	}
	// A different explicit value wins, including the Windows case-insensitive
	// spelling, and no duplicate key is appended.
	for _, explicit := range []string{"0", "false", ""} {
		for _, key := range []string{electronRunAsNodeEnv, "electron_run_as_node"} {
			in := []string{key + "=" + explicit, "PATH=C:\\Windows"}
			out := daemonEnv(in)
			if len(out) != len(in) {
				t.Fatalf("explicit %s=%q: env = %v, want unchanged length", key, explicit, out)
			}
			if out[0] != in[0] {
				t.Fatalf("explicit %s=%q overwritten: %v", key, explicit, out)
			}
			for _, entry := range out[1:] {
				if strings.HasPrefix(entry, electronRunAsNodeEnv+"=") {
					t.Fatalf("explicit %s=%q duplicated: %v", key, explicit, out)
				}
			}
		}
	}
}

// An empty environment still gets the variable; nothing is logged or rendered,
// so unrelated secrets in the parent environment never reach a log line.
func TestDaemonEnvEmptyBase(t *testing.T) {
	got := daemonEnv(nil)
	if len(got) != 1 || got[0] != electronRunAsNodeEnv+"=1" {
		t.Fatalf("daemonEnv(nil) = %v", got)
	}
}
