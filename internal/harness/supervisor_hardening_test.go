package harness

import (
	"context"
	"errors"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"whaleisle/internal/desktop/taskcontrol"
	"whaleisle/internal/harness/installctrl"
	"whaleisle/internal/paths"
)

func TestRedactScanAndBootLogDump(t *testing.T) {
	t.Setenv("DSHD_PLATFORM_TOKEN", "stale-platform-env")
	for _, kv := range childEnv(LaunchOptions{}) {
		if strings.HasPrefix(strings.ToUpper(kv), "DSHD_PLATFORM_TOKEN=") {
			t.Fatalf("childEnv leaked parent DSHD_PLATFORM_TOKEN: %q", kv)
		}
	}

	installSrv := installctrl.New(installctrl.Options{})
	if err := installSrv.Start(context.Background()); err != nil {
		t.Fatal(err)
	}
	defer installSrv.Stop()

	installToken := installSrv.Token()
	taskToken := "fixture-task-control-token"
	platformToken := "fixture-platform-token"

	s := New(Options{Install: installSrv})
	s.mu.Lock()
	s.gen = 1
	s.taskControlToken = taskToken
	s.platformToken = platformToken
	s.status = Status{State: Starting}
	s.mu.Unlock()

	ready := make(chan string, 1)
	raw := strings.Join([]string{
		"dsh web: http://127.0.0.1:4123/?token=launchsecret1",
		"dup: http://127.0.0.1:4123/?token=launchsecret2 and /other?token=launchsecret3",
		"task=" + taskToken + " plat=" + platformToken + " inst=" + installToken,
		"Authorization: Bearer secretbearer4",
	}, "\n") + "\n"
	s.scan(strings.NewReader(raw), ready, taskToken, 1)

	gotReady := <-ready
	if gotReady != "http://127.0.0.1:4123/?token=launchsecret1" {
		t.Fatalf("ready URL = %q", gotReady)
	}
	joinedLogs := strings.Join(s.Logs(), "\n")
	joinedStart := strings.Join(s.CurrentStartLogs(), "\n")
	for _, secret := range []string{taskToken, platformToken, installToken, "launchsecret1", "launchsecret2", "launchsecret3", "secretbearer4"} {
		if strings.Contains(joinedLogs, secret) || strings.Contains(joinedStart, secret) {
			t.Fatalf("secret %q leaked into supervisor logs:\n%s", secret, joinedLogs)
		}
	}

	dump := BootLogDump(
		"1.0.0",
		time.Unix(1700000000, 0).UTC(),
		Status{
			State:      Failed,
			Err:        errors.New("Get \"http://127.0.0.1:1234/?token=errsecret5\": dial refused"),
			Source:     "Bearer srcsecret6",
			PluginLoad: PluginProgress{Phase: "runtime", Detail: "Authorization: secretauth7"},
		},
		[]string{"raw log http://127.0.0.1/?token=dumptoken8"},
	)
	for _, secret := range []string{"errsecret5", "srcsecret6", "secretauth7", "dumptoken8"} {
		if strings.Contains(dump, secret) {
			t.Fatalf("secret %q leaked into BootLogDump:\n%s", secret, dump)
		}
	}
}

func TestCancelRecoveryHostAbsentTracksChildExit(t *testing.T) {
	// Part A: While stopChild is still active (stoppingChildren > 0), hostAbsent must
	// remain false so Coordinate prompts with Unknown host task status even when localRunning==0.
	sBusy := New(Options{})
	sBusy.mu.Lock()
	sBusy.gen = 3
	sBusy.stoppingChildren = 1
	sBusy.status = Status{State: Failed, RecoveryCancelled: true, Err: context.Canceled}
	sBusy.mu.Unlock()

	var busyInspection taskcontrol.Inspection
	busyConfirmCalled := false
	committedBusy := false
	did, err := sBusy.ProtectTaskAction(
		context.Background(),
		"quit",
		func() int { return 0 },
		func(i taskcontrol.Inspection) (bool, error) {
			busyConfirmCalled = true
			busyInspection = i
			return false, nil
		},
		func() error { committedBusy = true; return nil },
	)
	if !busyConfirmCalled || busyInspection.OK || busyInspection.Unknown == "" {
		t.Fatalf("expected Host/unknown confirmation while stoppingChildren > 0; called=%v inspection=%+v", busyConfirmCalled, busyInspection)
	}
	if did || committedBusy || !errors.Is(err, taskcontrol.ErrCancelled) {
		t.Fatalf("expected ErrCancelled when rejecting unknown Host confirmation; got did=%v committed=%v err=%v", did, committedBusy, err)
	}

	// Part B: After CancelRecovery finishes waiting for child exit, stoppingChildren == 0,
	// so hostAbsent flips to true and localRunning==0 commits without an Unknown prompt.
	exe, err := os.Executable()
	if err != nil {
		t.Fatal(err)
	}
	cmd := exec.Command(exe)
	cmd.Env = append(os.Environ(), "FAKE_DSH=hold")
	detach(cmd)
	if err := cmd.Start(); err != nil {
		t.Fatal(err)
	}
	exited := make(chan struct{})
	go func() { _ = cmd.Wait(); close(exited) }()
	t.Cleanup(func() { _ = terminate(cmd, true); <-exited })

	s := New(Options{AutoRestart: true, MaxAttempts: 2, BaseDelay: time.Hour})
	s.mu.Lock()
	s.gen = 4
	s.recoveryActive = true
	s.cmd, s.exited = cmd, exited
	s.taskControlToken = "fixture-token"
	s.setLocked(Status{State: Starting})
	s.mu.Unlock()

	st := s.CancelRecovery()
	if st.State != Failed || !st.RecoveryCancelled {
		t.Fatalf("CancelRecovery status = %+v", st)
	}
	confirmCalled := false
	committedDone := false
	did, err = s.ProtectTaskAction(
		context.Background(),
		"quit",
		func() int { return 0 },
		func(taskcontrol.Inspection) (bool, error) {
			confirmCalled = true
			return true, nil
		},
		func() error { committedDone = true; return nil },
	)
	if confirmCalled || !did || !committedDone || err != nil {
		t.Fatalf("expected hostAbsent commit without unknown prompt; confirmCalled=%v did=%v committed=%v err=%v", confirmCalled, did, committedDone, err)
	}
}

func TestOnPluginRecoveryRunsUnlockedAndRespectsGeneration(t *testing.T) {
	t.Run("UnlockedCallbackCanReadSupervisor", func(t *testing.T) {
		var s *Supervisor
		called := false
		s = New(Options{OnPluginRecovery: func(reason string, logs []string) error {
			called = true
			_ = s.Status()
			_ = s.Logs()
			return errors.New("abort recovery for test")
		}})
		s.mu.Lock()
		s.gen = 1
		s.startLogs = []string{"plugin tree failed to load"}
		s.mu.Unlock()

		s.fail(1, errors.New("exit"), "fixture")
		if !called {
			t.Fatal("OnPluginRecovery not called")
		}
		if st := s.Status(); st.State != Failed {
			t.Fatalf("state = %v, want Failed", st.State)
		}
	})

	t.Run("StaleGenerationAbortsAfterUnlock", func(t *testing.T) {
		entered := make(chan struct{})
		release := make(chan struct{})
		var s *Supervisor
		s = New(Options{
			AutoRestart: true,
			MaxAttempts: 2,
			OnPluginRecovery: func(string, []string) error {
				close(entered)
				<-release
				return nil
			},
		})
		s.mu.Lock()
		s.gen = 1
		s.recoveryActive = true
		s.startLogs = []string{"plugin tree failed to load"}
		s.setLocked(Status{State: Starting})
		s.mu.Unlock()

		done := make(chan struct{})
		go func() {
			defer close(done)
			s.fail(1, errors.New("plugin tree crashed"), "fixture")
		}()

		<-entered
		cancelSt := s.CancelRecovery()
		close(release)
		<-done

		if cancelSt.State != Failed || !cancelSt.RecoveryCancelled {
			t.Fatalf("cancel status = %+v", cancelSt)
		}
		if s.opts.Launch.SkipUserPlugins || s.skippedOnce {
			t.Fatal("stale fail mutated SkipUserPlugins/skippedOnce after generation change")
		}
		if got := s.Status(); got.State != Failed || got.Generation != cancelSt.Generation {
			t.Fatalf("stale fail overwrote status: %+v vs cancel %+v", got, cancelSt)
		}
	})
}

func TestOnChangeOrderingCoalescingAndStopDrain(t *testing.T) {
	var active atomic.Int32
	var maxActive atomic.Int32
	var mu sync.Mutex
	var states []State
	stoppedDone := atomic.Bool{}

	s := New(Options{OnChange: func(st Status) {
		cur := active.Add(1)
		for {
			prev := maxActive.Load()
			if cur <= prev || maxActive.CompareAndSwap(prev, cur) {
				break
			}
		}
		defer active.Add(-1)

		if st.State == Stopped {
			time.Sleep(25 * time.Millisecond)
			stoppedDone.Store(true)
		}
		mu.Lock()
		states = append(states, st.State)
		mu.Unlock()
	}})

	s.mu.Lock()
	s.gen = 1
	s.setLocked(Status{State: Starting})
	s.mu.Unlock()
	for i := 0; i < 80; i++ {
		s.log("line", 1)
	}
	s.Stop()
	if !stoppedDone.Load() {
		t.Fatal("Stop returned before OnChange(Stopped) completed")
	}

	s.mu.Lock()
	s.gen = 2
	s.setLocked(Status{State: Starting})
	s.mu.Unlock()
	// Ensure the second generation drains too via Stop().
	s.Stop()

	if max := maxActive.Load(); max > 1 {
		t.Fatalf("concurrent OnChange executions detected: maxActive=%d", max)
	}
	mu.Lock()
	defer mu.Unlock()
	if len(states) < 4 {
		t.Fatalf("too few state transitions recorded: %v", states)
	}
	firstStopped := -1
	secondStarting := -1
	for i, st := range states {
		if st == Stopped && firstStopped == -1 {
			firstStopped = i
		}
		if firstStopped >= 0 && i > firstStopped && st == Starting {
			secondStarting = i
			break
		}
	}
	if firstStopped == -1 || secondStarting == -1 {
		t.Fatalf("unexpected transition order: %v", states)
	}
}

func TestSelectedHostNodeUsesOSExecutableName(t *testing.T) {
	selected := t.TempDir()
	if _, err := selectedHostNode(selected); err == nil {
		t.Fatal("expected error when bundled node is missing")
	}
	nodePath := filepath.Join(selected, "node", nodeExecutableName())
	if err := os.MkdirAll(filepath.Dir(nodePath), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(nodePath, []byte("stub"), 0o755); err != nil {
		t.Fatal(err)
	}
	got, err := selectedHostNode(selected)
	if err != nil {
		t.Fatalf("selectedHostNode: %v", err)
	}
	if got != nodePath {
		t.Fatalf("selectedHostNode = %q, want %q", got, nodePath)
	}
}

func TestOfficeOverlayChecksFallbackCliStat(t *testing.T) {
	payload, _ := syntheticOfficePayload(t)
	t.Setenv("DSH_PRIMARY_RUNTIME", payload)

	emptyRoot := t.TempDir()
	if _, err := EnsureOfficeOverlay(PrepareOptions{
		ProfileDir:  t.TempDir(),
		DshHome:     t.TempDir(),
		HarnessRoot: emptyRoot,
	}); err == nil || !strings.Contains(err.Error(), "libreoffice-kit") {
		t.Fatalf("expected missing libreoffice-kit cli error, got %v", err)
	}

	fallbackCLI := filepath.Join(emptyRoot, "apps", "cli", "node_modules", "@deepseek-ai", "libreoffice-kit", "lib", "cli.js")
	if err := os.MkdirAll(filepath.Dir(fallbackCLI), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(fallbackCLI, []byte("// fallback"), 0o644); err != nil {
		t.Fatal(err)
	}
	overlayPath, err := EnsureOfficeOverlay(PrepareOptions{
		ProfileDir:  t.TempDir(),
		DshHome:     t.TempDir(),
		HarnessRoot: emptyRoot,
	})
	if err != nil || overlayPath == "" {
		t.Fatalf("expected fallback cli.js to succeed, got path=%q err=%v", overlayPath, err)
	}
}

func TestOverlayAndPadDefaultAtomicWritesLeaveNoTmp(t *testing.T) {
	profile := t.TempDir()
	p := DesktopPlugin{OverlayDir: "test-overlay", OverlayFile: "test.patch.yml", InsertID: "id1", Package: "@scope/pkg"}
	insertPath, err := writeInsertOverlay(profile, p)
	if err != nil {
		t.Fatal(err)
	}
	rawPath, err := writeRawOverlay(profile, "raw-overlay", "raw.patch.yml", "- insert: []\n")
	if err != nil {
		t.Fatal(err)
	}
	home := t.TempDir()
	hostPath, err := prepareWhaleUIHost(home)
	if err != nil {
		t.Fatal(err)
	}
	t.Setenv(paths.EnvUserData, t.TempDir())
	for _, f := range []string{insertPath, rawPath, hostPath} {
		data, err := os.ReadFile(f)
		if err != nil || len(data) == 0 {
			t.Fatalf("expected written file %s: %v", f, err)
		}
		entries, err := os.ReadDir(filepath.Dir(f))
		if err != nil {
			t.Fatal(err)
		}
		for _, entry := range entries {
			if strings.Contains(entry.Name(), ".tmp") {
				t.Fatalf("unexpected temp residue %s in %s", entry.Name(), filepath.Dir(f))
			}
		}
	}
}

func TestStdoutPipeTrailingLogsBeforeCmdWait(t *testing.T) {
	captured := make(chan []string, 1)
	s := supervisorFor(t, "trailinglogs", Options{
		OnPluginRecovery: func(_ string, logs []string) error {
			captured <- append([]string(nil), logs...)
			return errors.New("captured")
		},
	})
	s.Start()
	st := waitFor(t, s, Failed)
	if st.State != Failed {
		t.Fatalf("state = %v", st.State)
	}
	select {
	case logs := <-captured:
		joined := strings.Join(logs, "\n")
		for _, want := range []string{"trailing line one", "trailing line two", "plugin tree failed to load"} {
			if !strings.Contains(joined, want) {
				t.Fatalf("missing trailing log %q in %q", want, joined)
			}
		}
	case <-time.After(time.Second):
		t.Fatal("trailing logs were not captured before child exit")
	}
}
