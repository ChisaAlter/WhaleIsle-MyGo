package hoststate_test

import (
	"fmt"
	"net"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
	"time"

	"whaleisle/internal/harness"
	"whaleisle/internal/hoststate"
	"whaleisle/internal/paths"
	"whaleisle/internal/ui/i18n"
)

// TestMain lets this test binary stand in for the Host the way
// internal/harness's own fixture does: the supervisor starts os.Executable()
// with FAKE_DSH set and this process answers as the child. Every visible
// outcome below therefore comes from the real state machine, not a stub.
func TestMain(m *testing.M) {
	switch os.Getenv("FAKE_DSH") {
	case "hold":
		time.Sleep(time.Hour)
		return
	case "serve":
		serveFakeHost()
		return
	case "crash":
		fmt.Println("boom")
		os.Exit(3)
	case "exit":
		return
	}
	// The supervisor records the Host under the profile's runtime directory, so
	// a test must never point at the machine's real profile.
	dir, err := os.MkdirTemp("", "whaleisle-hoststate-profile-")
	if err != nil {
		fmt.Fprintln(os.Stderr, "isolate test profile:", err)
		os.Exit(1)
	}
	if err := os.Setenv(paths.EnvUserData, dir); err != nil {
		fmt.Fprintln(os.Stderr, "isolate test profile:", err)
		os.Exit(1)
	}
	code := m.Run()
	_ = os.RemoveAll(dir)
	os.Exit(code)
}

func serveFakeHost() {
	host, port := "127.0.0.1", ""
	for i, a := range os.Args {
		switch a {
		case "--host":
			host = os.Args[i+1]
		case "--port":
			port = os.Args[i+1]
		}
	}
	listener, err := net.Listen("tcp", net.JoinHostPort(host, port))
	if err != nil {
		fmt.Println("listen:", err)
		os.Exit(1)
	}
	// No token in the URL: the supervisor redeems the endpoint without a
	// handshake, which is what a Host that serves without auth reports.
	fmt.Printf("dsh web: http://%s/\n", listener.Addr())
	_ = http.Serve(listener, http.NewServeMux())
}

// launcherSupervisor starts a supervisor whose Host is this test binary. The
// port comes from the OS so parallel test binaries never collide.
func launcherSupervisor(t *testing.T, mode string, opts harness.Options) *harness.Supervisor {
	t.Helper()
	t.Setenv("FAKE_DSH", mode)
	exe, err := os.Executable()
	if err != nil {
		t.Fatal(err)
	}
	opts.Launch = harness.LaunchOptions{DshBin: exe, Host: "127.0.0.1", Port: freePort(t), Home: t.TempDir()}
	return harness.New(opts)
}

func freePort(t *testing.T) int {
	t.Helper()
	listener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	defer listener.Close()
	return listener.Addr().(*net.TCPAddr).Port
}

func waitForState(t *testing.T, sup *harness.Supervisor, want harness.State) harness.Status {
	t.Helper()
	deadline := time.Now().Add(15 * time.Second)
	for time.Now().Before(deadline) {
		if st := sup.Status(); st.State == want {
			return st
		}
		time.Sleep(20 * time.Millisecond)
	}
	t.Fatalf("state %v, want %v; logs:\n%s", sup.Status().State, want, strings.Join(sup.Logs(), "\n"))
	return harness.Status{}
}

// TestLaunchStatesSayWhatAPersonNeeds covers the four outcomes a cold launch
// can reach: a start in progress, a running service, a Host that never came up,
// and a runtime that is not installed at all. Each one is read from the real
// supervisor and asserted through the copy every surface shares.
func TestLaunchStatesSayWhatAPersonNeeds(t *testing.T) {
	i18n.SetLocale(i18n.LocaleZH)
	defer i18n.SetLocale(i18n.DefaultLocale)

	t.Run("starting", func(t *testing.T) {
		sup := launcherSupervisor(t, "hold", harness.Options{})
		defer sup.Stop()
		sup.Start()
		st := waitForState(t, sup, harness.Starting)
		report := hoststate.Of(st)
		if report.Phase != hoststate.Starting || !strings.Contains(report.Line, "正在启动本地服务") {
			t.Fatalf("starting report = %+v", report)
		}
	})

	t.Run("ready", func(t *testing.T) {
		sup := launcherSupervisor(t, "serve", harness.Options{})
		defer sup.Stop()
		sup.Start()
		st := waitForState(t, sup, harness.Ready)
		if report := hoststate.Of(st); report.Phase != hoststate.Ready || report.Line != "" {
			t.Fatalf("ready report = %+v", report)
		}
	})

	t.Run("failed with the cause", func(t *testing.T) {
		sup := launcherSupervisor(t, "crash", harness.Options{})
		sup.Start()
		st := waitForState(t, sup, harness.Failed)
		report := hoststate.Of(st)
		if report.Phase != hoststate.Failed {
			t.Fatalf("failed report = %+v", report)
		}
		if !strings.Contains(report.Line, "桌面运行时已退出（退出码 3）") {
			t.Fatalf("cause line = %q", report.Line)
		}
		if strings.Contains(report.Line, "exit status") {
			t.Fatalf("cause line quotes the transport: %q", report.Line)
		}
	})

	t.Run("runtime missing", func(t *testing.T) {
		if _, err := exec.LookPath("dsh"); err == nil {
			t.Skip("a dsh on PATH would be launched instead of the missing runtime")
		}
		i18n.SetLocale(i18n.LocaleZH)
		sup := harness.New(harness.Options{Launch: harness.LaunchOptions{
			Host: "127.0.0.1", Port: freePort(t), Home: t.TempDir(), VendorDir: t.TempDir(),
		}})
		sup.Start()
		st := waitForState(t, sup, harness.Failed)
		report := hoststate.Of(st)
		if !strings.Contains(report.Line, "重新安装") {
			t.Fatalf("missing runtime line = %q", report.Line)
		}
		if report.Note == "" || report.Placeholder == "" {
			t.Fatalf("a failed launch left a surface silent: %+v", report)
		}
	})
}

// TestOrphanedHostIsReapedBeforeTheNextStart is the leak a finished lane left
// behind: the Host outlived the shell that started it and held its port. The
// record file is that shell's last words, and the next launch acts on it.
func TestOrphanedHostIsReapedBeforeTheNextStart(t *testing.T) {
	if runtime.GOOS != "linux" {
		t.Skip("the orphan identity check needs /proc; Windows has its own coverage")
	}
	dir := t.TempDir()
	recordPath := filepath.Join(dir, "host-process.json")
	exe, err := os.Executable()
	if err != nil {
		t.Fatal(err)
	}
	holder := exec.Command(exe)
	holder.Env = append(os.Environ(), "FAKE_DSH=hold")
	if err := holder.Start(); err != nil {
		t.Fatal(err)
	}
	holderExited := make(chan error, 1)
	go func() { holderExited <- holder.Wait() }()
	defer func() { _ = holder.Process.Kill() }()

	deadShell := deadProcess(t)
	if err := harness.WriteHostRecord(recordPath, harness.HostRecord{
		PID: holder.Process.Pid, ShellPID: deadShell, Image: exe, ShellImage: exe,
	}); err != nil {
		t.Fatal(err)
	}

	sup := launcherSupervisor(t, "serve", harness.Options{RecordPath: recordPath})
	sup.Start()
	waitForState(t, sup, harness.Ready)

	select {
	case err := <-holderExited:
		if err == nil {
			t.Fatal("the orphaned Host exited cleanly; the reap did not end it")
		}
	case <-time.After(10 * time.Second):
		t.Fatal("the orphaned Host survived the next launch")
	}
	// The record now names the Host this launch started; the reaped pid must be
	// gone from it, so a later shell cannot reap the live one by mistake.
	if record, ok := harness.ReadHostRecord(recordPath); ok && record.PID == holder.Process.Pid {
		t.Fatalf("the record still names the reaped Host: %+v", record)
	}
	logs := strings.Join(sup.Logs(), "\n")
	if !strings.Contains(logs, "orphaned Host") {
		t.Fatalf("the reap was not recorded in the log:\n%s", logs)
	}
	sup.Stop()
}

// TestHostWithALiveShellIsLeftAlone keeps the detector from ending a Host that
// another running shell still owns: a second instance must never kill the
// first one's service.
func TestHostWithALiveShellIsLeftAlone(t *testing.T) {
	if runtime.GOOS != "linux" {
		t.Skip("the orphan identity check needs /proc; Windows has its own coverage")
	}
	recordPath := filepath.Join(t.TempDir(), "host-process.json")
	exe, err := os.Executable()
	if err != nil {
		t.Fatal(err)
	}
	holder := exec.Command(exe)
	holder.Env = append(os.Environ(), "FAKE_DSH=hold")
	if err := holder.Start(); err != nil {
		t.Fatal(err)
	}
	defer func() { _ = holder.Process.Kill(); _, _ = holder.Process.Wait() }()
	if err := harness.WriteHostRecord(recordPath, harness.HostRecord{
		PID: holder.Process.Pid, ShellPID: os.Getpid(), Image: exe, ShellImage: exe,
	}); err != nil {
		t.Fatal(err)
	}
	if record, ok := harness.DetectOrphan(recordPath, os.Getpid()); ok {
		t.Fatalf("a Host with a live owner read as an orphan: %+v", record)
	}
}

// deadProcess returns the pid of a process that has exited and been reaped.
func deadProcess(t *testing.T) int {
	t.Helper()
	exe, err := os.Executable()
	if err != nil {
		t.Fatal(err)
	}
	cmd := exec.Command(exe)
	cmd.Env = append(os.Environ(), "FAKE_DSH=exit")
	if err := cmd.Run(); err != nil {
		t.Fatalf("the short-lived process did not exit cleanly: %v", err)
	}
	return cmd.Process.Pid
}

// TestRecordNamesTheStartedHost keeps the record a later shell depends on:
// after a start it names the child, its port and this shell.
func TestRecordNamesTheStartedHost(t *testing.T) {
	recordPath := filepath.Join(t.TempDir(), "host-process.json")
	sup := launcherSupervisor(t, "serve", harness.Options{RecordPath: recordPath})
	sup.Start()
	waitForState(t, sup, harness.Ready)
	record, ok := harness.ReadHostRecord(recordPath)
	if !ok {
		raw, _ := os.ReadFile(recordPath)
		t.Fatalf("no record after a successful start (file: %q)", string(raw))
	}
	if record.PID <= 0 || record.ShellPID != os.Getpid() || record.Port <= 0 {
		t.Fatalf("record does not name this start: %+v", record)
	}
	sup.Stop()
	if _, err := os.Stat(recordPath); !os.IsNotExist(err) {
		t.Fatalf("a stopped Host left its record behind: %v", err)
	}
}
