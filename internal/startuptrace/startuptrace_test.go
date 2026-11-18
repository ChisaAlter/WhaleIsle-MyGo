package startuptrace

import (
	"encoding/json"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func TestTraceRecorderLogRedactsSecretsAndStopsWriterOnFinish(t *testing.T) {
	dir := t.TempDir()
	tracePath := filepath.Join(dir, "trace.json")
	r := startRecorder(tracePath)
	done := r.doneCh

	r.meta("mode", "unit")
	endSpan := r.begin("boot.phase", "ok")
	r.log("dsh web: http://127.0.0.1:4321/?token=synthetic-launch-secret")
	r.log("Authorization: Bearer synthetic-bearer-secret")
	r.log("DSH_DESKTOP_INSTALL_TOKEN=synthetic-install-secret ?token=synthetic-q-secret")
	endSpan()

	r.finish("ready")

	// writer() must have completely exited before finish() returns.
	select {
	case <-done:
	case <-time.After(time.Second):
		t.Fatal("writer goroutine did not exit during finish")
	}

	// Second finish is safe and idempotent.
	r.finish("ready-again")

	raw, err := os.ReadFile(tracePath)
	if err != nil {
		t.Fatal(err)
	}
	text := string(raw)
	for _, secret := range []string{
		"synthetic-launch-secret",
		"synthetic-bearer-secret",
		"synthetic-install-secret",
		"synthetic-q-secret",
	} {
		if strings.Contains(text, secret) {
			t.Fatalf("secret %q leaked into trace file:\n%s", secret, text)
		}
	}

	var rep Report
	if err := json.Unmarshal(raw, &rep); err != nil {
		t.Fatal(err)
	}
	if !rep.Finished || rep.Reason != "ready-again" {
		t.Fatalf("unexpected report termination state: %+v", rep)
	}
	if rep.Meta["mode"] != "unit" {
		t.Fatalf("missing meta: %+v", rep.Meta)
	}
	entries, err := os.ReadDir(dir)
	if err != nil {
		t.Fatal(err)
	}
	if len(entries) != 1 || entries[0].Name() != "trace.json" {
		t.Fatalf("unexpected temp files left in %s", dir)
	}
}

func TestPackageTraceInitLogAndFinishSubprocess(t *testing.T) {
	if os.Getenv("FAKE_TRACE_RUN") == "1" {
		if !Enabled() {
			os.Exit(2)
		}
		Log("http://127.0.0.1:9999/?token=child-secret-val")
		Mark("child.marked")
		Finish("done")
		if !strings.Contains(Summary(), "reason=done") {
			os.Exit(3)
		}
		return
	}
	tracePath := filepath.Join(t.TempDir(), "sub-trace.json")
	exe, err := os.Executable()
	if err != nil {
		t.Fatal(err)
	}
	cmd := exec.Command(exe, "-test.run=^TestPackageTraceInitLogAndFinishSubprocess$")
	cmd.Env = append(os.Environ(), "FAKE_TRACE_RUN=1", EnvPath+"="+tracePath)
	if out, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("subprocess failed: %v\n%s", err, out)
	}
	raw, err := os.ReadFile(tracePath)
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(string(raw), "child-secret-val") {
		t.Fatalf("secret leaked in package trace:\n%s", raw)
	}
	var rep Report
	if err := json.Unmarshal(raw, &rep); err != nil {
		t.Fatal(err)
	}
	if !rep.Finished || rep.Reason != "done" {
		t.Fatalf("unexpected report: %+v", rep)
	}
}
