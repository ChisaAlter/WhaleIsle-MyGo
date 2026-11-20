//go:build windows

package installhealth

import (
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"golang.org/x/sys/windows"
	"whaleisle/internal/app/updater"
	"whaleisle/internal/startuptrace"
)

func TestReadyRequiresActualWorkspaceAndInventory(t *testing.T) {
	r := startuptrace.Report{}
	for _, stage := range []string{"window.created", "host.ready", "workspace.ready", "plugins.inventory_ready"} {
		if ready(r) {
			t.Fatalf("premature success before %s", stage)
		}
		r.Events = append(r.Events, startuptrace.Event{Stage: stage, Kind: "mark"})
	}
	if !ready(r) {
		t.Fatal("complete boot was not accepted")
	}
}

func TestDisabledPluginRowsAreNamedAndDeduplicated(t *testing.T) {
	r := startuptrace.Report{}
	for _, stage := range []string{"window.created", "host.ready", "workspace.ready", "plugins.inventory_ready"} {
		r.Events = append(r.Events, startuptrace.Event{Stage: stage, Kind: "mark"})
	}
	if !ready(r) {
		t.Fatal("complete boot was not accepted")
	}
	if rows := disabledPluginRows(r); len(rows) != 0 {
		t.Fatalf("a clean boot named disabled rows: %v", rows)
	}
	r.Events = append(r.Events,
		startuptrace.Event{Kind: "log", Stage: "host.log", Detail: `dsh: disabling profile plugin row "dsh-whale": Plugin dsh-whale@0.1.0 is incompatible with dsh 0.2.2-rc.5: requirements {"@deepseek-ai/dsh-agent":"0.2.2-rc.2"}.`},
		startuptrace.Event{Kind: "log", Stage: "host.log", Detail: `dsh: disabling profile plugin row "dsh-remote": Plugin dsh-remote@0.8.21 is incompatible with dsh 0.2.2-rc.5.`},
		startuptrace.Event{Kind: "log", Stage: "host.log", Detail: `dsh: disabling profile plugin row "dsh-whale": repeated for the same row.`},
		startuptrace.Event{Kind: "log", Stage: "host.log", Detail: `dsh: mounted 12 plugin rows`},
	)
	if rows := disabledPluginRows(r); strings.Join(rows, ",") != "dsh-whale,dsh-remote" {
		t.Fatalf("disabled rows: %v", rows)
	}
}

func TestEnvironmentUsesOnlyInstalledRuntimeAndOwnedProfile(t *testing.T) {
	t.Setenv("WHALEISLE_VENDOR", `C:\wrong\vendor`)
	t.Setenv("WHALEISLE_NODE_DIR", `C:\wrong\node`)
	t.Setenv("WHALEISLE_ISOLATED_VISIBLE", "1")
	t.Setenv(updater.EnvDisableAutoUpdate, "0")
	t.Setenv("WHALEISLE_PACKAGE_SKIP_RUNTIME", "1")
	t.Setenv("NODE_OPTIONS", "--require=C:/wrong/hook.js")
	t.Setenv("DSH_HOME", `C:\real\data`)
	block := environment(`C:\candidate`, `C:\owned`, `C:\owned\trace.json`)
	values := map[string]string{}
	for start, i := 0, 0; i < len(block); i++ {
		if block[i] != 0 {
			continue
		}
		entry := windows.UTF16ToString(block[start:i])
		start = i + 1
		name, value, ok := strings.Cut(entry, "=")
		if ok {
			values[name] = value
		}
	}
	for name, want := range map[string]string{"WHALEISLE_VENDOR": `C:\candidate\runtime\vendor`, "WHALEISLE_NODE_DIR": `C:\candidate\runtime\node`, "WHALEISLE_USER_DATA": `C:\owned`, "DSH_HOME": `C:\owned\dsh-home`, "WHALEISLE_ISOLATED_VISIBLE": "0", updater.EnvDisableAutoUpdate: "1"} {
		if values[name] != want {
			t.Errorf("%s=%q; want %q", name, values[name], want)
		}
	}
	for _, name := range []string{"NODE_OPTIONS", "WHALEISLE_PACKAGE_SKIP_RUNTIME"} {
		if _, ok := values[name]; ok {
			t.Errorf("inherited %s", name)
		}
	}
}

func TestCleanupUnlinksJunctionWithoutTouchingTarget(t *testing.T) {
	parent := t.TempDir()
	root := filepath.Join(parent, "whaleisle-install-health-owned")
	target := filepath.Join(parent, "outside")
	if err := os.Mkdir(root, 0700); err != nil {
		t.Fatal(err)
	}
	if err := os.Mkdir(target, 0700); err != nil {
		t.Fatal(err)
	}
	sentinel := filepath.Join(target, "keep.txt")
	if err := os.WriteFile(sentinel, []byte("real target remains"), 0600); err != nil {
		t.Fatal(err)
	}
	cmd := exec.Command("cmd.exe", "/c", "mklink", "/J", filepath.Join(root, "link"), target)
	if raw, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("junction: %v: %s", err, raw)
	}
	if err := removeOwnedProfile(root); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(root); !os.IsNotExist(err) {
		t.Fatalf("owned root remains: %v", err)
	}
	if raw, err := os.ReadFile(sentinel); err != nil || string(raw) != "real target remains" {
		t.Fatalf("junction target changed: %q %v", raw, err)
	}
}

func TestRealInstalledCandidate(t *testing.T) {
	target, output := os.Getenv("WHALEISLE_INSTALL_HEALTH_TARGET"), os.Getenv("WHALEISLE_INSTALL_HEALTH_REPORT")
	if target == "" || output == "" {
		t.Skip("explicit installed candidate and external report required")
	}
	if err := Check(target, output); err != nil {
		t.Fatal(err)
	}
	raw, err := os.ReadFile(output)
	if err != nil {
		t.Fatal(err)
	}
	var r result
	if err = json.Unmarshal(raw, &r); err != nil {
		t.Fatal(err)
	}
	if !r.Passed || !r.ProcessesExited || !r.ProfileRemoved || r.Startup == nil || !ready(*r.Startup) {
		t.Fatalf("incomplete health result: %s", raw)
	}
	if _, err := os.Stat(r.Profile); !os.IsNotExist(err) {
		t.Fatalf("profile was not reclaimed: %s: %v", r.Profile, err)
	}
}

func TestRealTimeoutReclaimsWholeJobAndProfile(t *testing.T) {
	target, output := os.Getenv("WHALEISLE_INSTALL_HEALTH_TARGET"), os.Getenv("WHALEISLE_INSTALL_HEALTH_TIMEOUT_REPORT")
	if target == "" || output == "" {
		t.Skip("explicit installed terminal input and external timeout report required")
	}
	root := t.TempDir()
	for _, dir := range []string{"candidate", "candidate/runtime", "candidate/runtime/node"} {
		if err := os.Mkdir(filepath.Join(root, filepath.FromSlash(dir)), 0700); err != nil {
			t.Fatal(err)
		}
	}
	candidate := filepath.Join(root, "candidate")
	for _, name := range []string{"whaleisle-preview.exe", "runtime/node/node.exe"} {
		if err := os.WriteFile(filepath.Join(candidate, filepath.FromSlash(name)), []byte("unused by the hanging candidate"), 0600); err != nil {
			t.Fatal(err)
		}
	}
	dll, err := os.ReadFile(filepath.Join(target, "ghostty-vt.dll"))
	if err != nil {
		t.Fatal(err)
	}
	if err = os.WriteFile(filepath.Join(candidate, "ghostty-vt.dll"), dll, 0600); err != nil {
		t.Fatal(err)
	}
	source := filepath.Join(root, "hang.go")
	code := `package main
import("os";"os/exec";"time")
func main(){if len(os.Args)==1 {c:=exec.Command(os.Args[0],"child");if c.Start()!=nil {os.Exit(3)}};time.Sleep(3*time.Minute)}
`
	if err = os.WriteFile(source, []byte(code), 0600); err != nil {
		t.Fatal(err)
	}
	cmd := exec.Command("go", "build", "-o", filepath.Join(candidate, "whaleisle.exe"), source)
	if raw, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("hanging candidate build: %v: %s", err, raw)
	}
	if err = Check(candidate, output); err == nil || !strings.Contains(err.Error(), fmt.Sprintf("within %d seconds", int(healthDeadline.Seconds()))) {
		t.Fatalf("expected actual health deadline, got %v", err)
	}
	raw, err := os.ReadFile(output)
	if err != nil {
		t.Fatal(err)
	}
	var r result
	if err = json.Unmarshal(raw, &r); err != nil {
		t.Fatal(err)
	}
	if r.Passed || !r.ProcessesExited || !r.ProfileRemoved {
		t.Fatalf("timeout did not retire entire job/profile: %s", raw)
	}
}
