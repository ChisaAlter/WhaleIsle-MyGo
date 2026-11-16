// Package harness runs DeepSeek Harness (`dsh web`) as a child process and
// tracks the authenticated endpoint it serves.
package harness

import (
	"bufio"
	"context"
	"crypto/rand"
	"encoding/hex"
	"errors"
	"fmt"
	"io"
	"net"
	"net/http"
	"os"
	"os/exec"
	"strconv"
	"strings"
	"sync"
	"time"

	"whaleisle/internal/harness/installctrl"
	"whaleisle/internal/startuprec"
	"whaleisle/internal/startuptrace"
)

// State is the supervisor's lifecycle phase.
type State int

const (
	Stopped State = iota
	Starting
	Ready
	Failed
)

func (s State) String() string {
	switch s {
	case Starting:
		return "starting"
	case Ready:
		return "ready"
	case Failed:
		return "failed"
	}
	return "stopped"
}

// Status is one observable snapshot of the supervisor.
type Status struct {
	// Generation identifies the owned process attempt, including same-port restarts.
	Generation int
	State      State
	Endpoint   Endpoint
	// Err explains a Failed state.
	Err error
	// Source names the launcher of the current or last attempt.
	Source string
	// Attempt counts automatic restarts since the last stable Ready period.
	Attempt int
	// PluginLoad carries overlay/runtime plugin progress for the boot page.
	PluginLoad PluginProgress
	// Recovery is true when this attempt skips the user plugin layer.
	Recovery bool
	// RecoveryActive includes queued retries and their startup attempt.
	RecoveryActive bool
	// RecoveryScheduled distinguishes a queued retry from its running attempt.
	RecoveryScheduled bool
	// RecoveryCancelled suppresses automatic retries until the next manual start.
	RecoveryCancelled bool
}

// Options configure a Supervisor.
type Options struct {
	Launch LaunchOptions
	// Prepare configures desktop plugin overlays/junctions before each start.
	Prepare PrepareOptions
	// ReadLaunchPatches prepares desktop-owned optional overlays for each start.
	// It runs outside the supervisor lock and reads committed local preferences.
	ReadLaunchPatches func(context.Context) ([]string, error)
	// Workspace is the optional child working directory when it still exists.
	// An unavailable saved workspace must not prevent access to other sessions.
	// Registering it here would suppress the Host's first-use initialization.
	Workspace string
	// AutoRestart relaunches dsh after an unexpected exit, up to MaxAttempts
	// times with doubling delays from BaseDelay.
	AutoRestart bool
	MaxAttempts int
	BaseDelay   time.Duration
	// ReadRestartPolicy returns committed settings for each new failure decision.
	// It runs outside the supervisor lock and must not mutate lifecycle state.
	ReadRestartPolicy func() RestartPolicy
	// ReadyTimeout bounds one start; npx downloads can take minutes.
	ReadyTimeout time.Duration
	// OnChange receives every status change, serialized on a background worker.
	OnChange func(Status)
	// OnPluginRecovery must persist the marker before a skipped-plugin retry.
	// Called outside the supervisor lock so callers may inspect Supervisor state safely.
	OnPluginRecovery func(reason string, logs []string) error
	// Install is the loopback install control server; started with the supervisor
	// and its env is injected into each harness child.
	Install *installctrl.Server
}

// logLimit is how many output lines the boot log keeps.
const logLimit = 400

// maxNotifyQueue bounds buffered OnChange events while preserving essential state transitions.
const maxNotifyQueue = 64

// stopGrace is how long dsh gets between SIGTERM and SIGKILL on Unix. Windows
// force-ends the tree on the first terminate, so Stop does not wait there.
const stopGrace = 5 * time.Second

// restartStablePeriod prevents a short-lived Ready state from replenishing the
// restart budget. Healthy hosts earn a fresh budget after one minute online.
const restartStablePeriod = time.Minute

type notifyItem struct {
	st        Status
	essential bool
}

type notifyWorker struct {
	mu     sync.Mutex
	queue  []notifyItem
	closed bool
	sealed bool
	done   chan struct{}
}

func (w *notifyWorker) seal() {
	w.mu.Lock()
	w.sealed = true
	w.mu.Unlock()
}

func (w *notifyWorker) enqueue(item notifyItem) bool {
	w.mu.Lock()
	defer w.mu.Unlock()
	if w.closed || w.sealed {
		return false
	}
	if !item.essential {
		if n := len(w.queue); n > 0 {
			last := &w.queue[n-1]
			if !last.essential && last.st.Generation == item.st.Generation && last.st.State == item.st.State {
				last.st = item.st
				return true
			}
		}
	}
	if len(w.queue) >= maxNotifyQueue {
		evicted := false
		for i, queued := range w.queue {
			if !queued.essential {
				copy(w.queue[i:], w.queue[i+1:])
				w.queue[len(w.queue)-1] = notifyItem{}
				w.queue = w.queue[:len(w.queue)-1]
				evicted = true
				break
			}
		}
		if !evicted && !item.essential {
			return true
		}
	}
	w.queue = append(w.queue, item)
	return true
}

func runNotifyWorker(w *notifyWorker, prev <-chan struct{}, cb func(Status)) {
	defer close(w.done)
	if prev != nil {
		<-prev
	}
	for {
		w.mu.Lock()
		if len(w.queue) == 0 {
			w.closed = true
			w.mu.Unlock()
			return
		}
		item := w.queue[0]
		copy(w.queue, w.queue[1:])
		w.queue[len(w.queue)-1] = notifyItem{}
		w.queue = w.queue[:len(w.queue)-1]
		w.mu.Unlock()
		cb(item.st)
	}
}

// Supervisor owns one dsh web child at a time.
type Supervisor struct {
	opts   Options
	client *http.Client

	protectionMu     sync.Mutex
	taskControlToken string
	platformToken    string
	mu               sync.Mutex
	status           Status
	cmd              *exec.Cmd
	exited           chan struct{}
	stoppingChildren int
	gen              int
	logs             []string
	startLogs        []string // current attempt only; old attempts cannot implicate plugins
	attemptRecovery  bool     // flags captured for the current attempt, not mutable next-launch settings
	attempt          int
	readyAt          time.Time
	// skippedOnce is true after a plugin-tree failure retried with --skip-user-plugins.
	skippedOnce       bool
	recoveryActive    bool
	recoveryQueued    bool
	recoveryCancelled bool
	restartTimer      *time.Timer
	launchCtx         context.Context
	launchCancel      context.CancelFunc
	statusChanged     chan struct{}
	notify            *notifyWorker
	prevNotifyDone    <-chan struct{}
}

// New returns a stopped supervisor.
func New(opts Options) *Supervisor {
	if opts.ReadyTimeout <= 0 {
		opts.ReadyTimeout = 3 * time.Minute
	}
	if opts.BaseDelay <= 0 {
		opts.BaseDelay = time.Second
	}
	return &Supervisor{opts: opts, client: &http.Client{Timeout: 10 * time.Second}}
}

// Status returns the current snapshot.
func (s *Supervisor) Status() Status {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.status
}

// Logs returns the retained output lines, oldest first.
func (s *Supervisor) Logs() []string {
	s.mu.Lock()
	defer s.mu.Unlock()
	return append([]string(nil), s.logs...)
}

// CurrentStartLogs returns only evidence from the current process attempt.
func (s *Supervisor) CurrentStartLogs() []string {
	s.mu.Lock()
	defer s.mu.Unlock()
	return append([]string(nil), s.startLogs...)
}

// SetSkipUserPlugins changes the recovery flag used by the next start.
func (s *Supervisor) SetSkipUserPlugins(skip bool) {
	s.mu.Lock()
	s.opts.Launch.SkipUserPlugins = skip
	s.opts.Prepare.SkipUserPlugins = skip
	if !skip {
		s.skippedOnce = false
	}
	s.mu.Unlock()
}

// SetPrepareOptions replaces desktop overlay selection for the next start
// (e.g. whale / dshbot toggles from config).
func (s *Supervisor) SetPrepareOptions(prep PrepareOptions) {
	s.mu.Lock()
	s.opts.Prepare = prep
	s.mu.Unlock()
}

// Start launches dsh unless it is already starting or ready.
func (s *Supervisor) Start() {
	s.startOwned()
}

// startOwned returns the exact generation allocated by this call. A settings
// waiter must never adopt an attempt launched by another concurrent owner.
func (s *Supervisor) startOwned() (int, bool) {
	s.mu.Lock()
	if s.status.State == Starting || s.status.State == Ready {
		s.mu.Unlock()
		return 0, false
	}
	install := s.opts.Install
	s.attempt = 0
	s.recoveryActive, s.recoveryQueued, s.recoveryCancelled = false, false, false
	s.attemptRecovery = s.opts.Launch.SkipUserPlugins
	gen := s.nextGenLocked()
	launchCtx, launchCancel := context.WithCancel(context.Background())
	s.launchCtx = launchCtx
	s.launchCancel = launchCancel
	s.setLocked(Status{State: Starting})
	s.mu.Unlock()
	if install != nil {
		if err := install.Start(launchCtx); err != nil {
			if s.current(gen) {
				install.Stop()
				s.set(gen, Status{State: Failed, Err: fmt.Errorf("install control: %w", err)})
			}
			return gen, true
		}
	}
	go s.run(gen)
	return gen, true
}

// Restart stops the current child and starts a new one.
func (s *Supervisor) Restart() {
	s.Stop()
	s.Start()
}

// CancelRecovery cancels queued and in-progress automatic attempts without
// changing restart preferences or stopping an unrelated manual/ready Host.
func (s *Supervisor) CancelRecovery() Status {
	s.mu.Lock()
	if !s.recoveryActive || s.recoveryCancelled {
		st := s.status
		s.mu.Unlock()
		return st
	}
	st := s.status
	s.nextGenLocked()
	cmd, exited := s.cmd, s.exited
	s.cmd, s.exited = nil, nil
	if cmd != nil {
		s.stoppingChildren++
	}
	st.State, st.Endpoint = Failed, Endpoint{}
	if st.Err == nil {
		st.Err = context.Canceled
	}
	s.recoveryActive, s.recoveryQueued = false, false
	s.recoveryCancelled = true
	s.setLocked(st)
	st = s.status
	s.mu.Unlock()
	if cmd != nil {
		stopChild(cmd, exited)
		s.mu.Lock()
		s.stoppingChildren--
		s.mu.Unlock()
	}
	return st
}

// Stop ends the child, waiting for it to exit.
func (s *Supervisor) Stop() {
	s.mu.Lock()
	install := s.opts.Install
	s.nextGenLocked()
	cmd, exited := s.cmd, s.exited
	s.cmd, s.exited = nil, nil
	if cmd != nil {
		s.stoppingChildren++
	}
	s.recoveryActive, s.recoveryQueued, s.recoveryCancelled = false, false, false
	s.setLocked(Status{State: Stopped})
	var notifyDone <-chan struct{}
	if w := s.notify; w != nil {
		w.seal()
		notifyDone = w.done
	} else if s.prevNotifyDone != nil {
		notifyDone = s.prevNotifyDone
	}
	s.mu.Unlock()
	if install != nil {
		install.Stop()
	}
	if cmd != nil {
		stopChild(cmd, exited)
		s.mu.Lock()
		s.stoppingChildren--
		s.mu.Unlock()
	}
	if notifyDone != nil {
		<-notifyDone
	}
}

// The caller retains this exact child until its wait goroutine confirms exit.
func stopChild(cmd *exec.Cmd, exited <-chan struct{}) {
	if cmd == nil {
		return
	}
	_ = terminate(cmd, false)
	select {
	case <-exited:
	case <-time.After(stopGrace):
		_ = terminate(cmd, true)
		<-exited
	}
}

// nextGenLocked invalidates every goroutine of earlier attempts.
func (s *Supervisor) nextGenLocked() int {
	if s.launchCancel != nil {
		s.launchCancel()
		s.launchCancel = nil
	}
	s.launchCtx = nil
	if s.restartTimer != nil {
		s.restartTimer.Stop()
		s.restartTimer = nil
	}
	s.gen++
	s.readyAt = time.Time{}
	s.taskControlToken = ""
	s.platformToken = ""
	return s.gen
}

func (s *Supervisor) current(gen int) bool {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.gen == gen
}

func (s *Supervisor) enqueueNotifyLocked(st Status, essential bool) {
	cb := s.opts.OnChange
	if cb == nil {
		return
	}
	item := notifyItem{st: st, essential: essential}
	if w := s.notify; w != nil {
		if w.enqueue(item) {
			return
		}
	}
	w := &notifyWorker{
		queue: []notifyItem{item},
		done:  make(chan struct{}),
	}
	prev := s.prevNotifyDone
	s.notify = w
	s.prevNotifyDone = w.done
	go runNotifyWorker(w, prev, cb)
}

func (s *Supervisor) setLocked(st Status) {
	if st.State == Ready {
		if s.status.State != Ready || s.status.Generation != s.gen {
			s.readyAt = time.Now()
		}
	} else {
		s.readyAt = time.Time{}
	}
	if st.State == Stopped || st.State == Failed {
		if s.launchCancel != nil {
			s.launchCancel()
			s.launchCancel = nil
		}
		s.launchCtx = nil
	}
	st.Generation = s.gen
	st.Attempt = s.attempt
	st.Recovery = s.attemptRecovery
	if st.State != Starting {
		s.recoveryActive, s.recoveryQueued = false, false
	}
	st.RecoveryActive = s.recoveryActive
	st.RecoveryScheduled = s.recoveryQueued
	st.RecoveryCancelled = s.recoveryCancelled
	s.status = st
	if s.statusChanged != nil {
		close(s.statusChanged)
		s.statusChanged = nil
	}
	s.enqueueNotifyLocked(st, true)
}

func (s *Supervisor) set(gen int, st Status) bool {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.gen != gen {
		return false
	}
	s.setLocked(st)
	return true
}

func (s *Supervisor) log(line string, generation ...int) {
	s.mu.Lock()
	if len(generation) > 0 && generation[0] != s.gen {
		s.mu.Unlock()
		return
	}
	s.logs = append(s.logs, line)
	s.startLogs = append(s.startLogs, line)
	if len(s.startLogs) > logLimit {
		s.startLogs = append([]string(nil), s.startLogs[len(s.startLogs)-logLimit:]...)
	}
	if over := len(s.logs) - logLimit; over > 0 {
		s.logs = append(s.logs[:0], s.logs[over:]...)
	}
	// Notify while Starting so the launcher can stream logs. Copy status under
	// the lock; refreshBoot ignores Starting after Ready/Failed.
	if s.status.State == Starting {
		s.enqueueNotifyLocked(s.status, false)
	}
	s.mu.Unlock()
}

func (s *Supervisor) run(gen int) {
	// Generate only at actual child launch, not during configuration or inspection.
	var secret [24]byte
	if _, err := rand.Read(secret[:]); err != nil {
		s.set(gen, Status{State: Failed, Err: fmt.Errorf("task-control token initialization failed")})
		return
	}
	token := hex.EncodeToString(secret[:])
	if _, err := rand.Read(secret[:]); err != nil {
		s.set(gen, Status{State: Failed, Err: fmt.Errorf("platform token initialization failed")})
		return
	}
	platformToken := hex.EncodeToString(secret[:])
	s.mu.Lock()
	if s.gen != gen {
		s.mu.Unlock()
		return
	}
	s.recoveryQueued = false
	s.restartTimer = nil
	s.taskControlToken = token
	s.platformToken = platformToken
	s.startLogs = nil
	launch := s.opts.Launch
	readPatches := s.opts.ReadLaunchPatches
	launchCtx, launchCancel := s.launchCtx, s.launchCancel
	if launchCtx == nil || launchCancel == nil {
		launchCtx, launchCancel = context.WithCancel(context.Background())
		s.launchCtx = launchCtx
		s.launchCancel = launchCancel
	}
	defer launchCancel()
	s.attemptRecovery = launch.SkipUserPlugins
	if s.recoveryActive {
		s.setLocked(s.status)
	}
	launch.ExtraEnv = append([]string(nil), launch.ExtraEnv...)
	install := s.opts.Install
	dshbot := s.opts.Prepare.Dshbot
	s.mu.Unlock()
	if install != nil {
		launch.ExtraEnv = append(launch.ExtraEnv, install.Env()...)
	}
	launch.ExtraEnv = append(launch.ExtraEnv, "DSHD_TASK_CONTROL_TOKEN="+token, "DSHD_PLATFORM_TOKEN="+platformToken)
	if dshbot {
		launch.ExtraEnv = append(launch.ExtraEnv, "DSHD_DSHBOT_ENABLED=1")
	}
	tokens := s.collectSecretTokens(token, platformToken)
	tokens = uniqueNonEmpty(tokens, extractInjectedTokens(launch.ExtraEnv))

	// Read the staged module tree ahead of node.exe from the first moment the
	// launch is known: it overlaps port selection, validation and profile
	// staging, and stops as soon as the Host reports its ready URL. Read-only.
	warm := warmRuntimeTree(launch.VendorDir)
	defer warm.Stop()
	port, err := freePort(launch.Host, launch.Port, launch.ReservedPorts)
	if err != nil {
		s.set(gen, Status{State: Failed, Err: err})
		return
	}
	startuptrace.Mark("host.port_selected", strconv.Itoa(port))
	launch, err = readLaunchPatches(launchCtx, launch, readPatches)
	if err != nil {
		s.set(gen, Status{State: Failed, Err: err})
		return
	}
	var plan Plan
	var patches []string
	endPlan := startuptrace.Begin("host.plan")
	if launch.DshBin != "" {
		// Configured / test launchers skip desktop overlay preparation.
		plan, err = Resolve(launch, port)
	} else {
		s.mu.Lock()
		prep := s.opts.Prepare
		s.mu.Unlock()
		var start StartPlan
		start, err = PlanStart(launch, prep, port)
		plan, patches = start.Plan, start.PatchFiles
	}
	endPlan()
	if err != nil {
		s.set(gen, Status{State: Failed, Err: err})
		return
	}
	if !s.set(gen, Status{State: Starting, Source: plan.Source, PluginLoad: PluginProgress{Phase: "runtime"}}) {
		return
	}
	s.log(fmt.Sprintf("» %s %s", plan.Command, strings.Join(plan.Args, " ")), gen)
	if n := len(patches); n > 0 {
		overlayLine := fmt.Sprintf("» %d desktop --patch overlay(s)", n)
		s.log(overlayLine, gen)
		s.notePluginLog(overlayLine, gen)
	}

	cmd := exec.Command(plan.Command, plan.Args...)
	cmd.Env = plan.Env
	if workspace := s.opts.Workspace; workspace != "" {
		if info, err := os.Stat(workspace); err == nil && info.IsDir() {
			cmd.Dir = workspace
		} else {
			s.log("workspace unavailable; starting Host with the application's working directory", gen)
		}
	}
	detach(cmd)
	out, err := cmd.StdoutPipe()
	if err != nil {
		s.set(gen, Status{State: Failed, Err: err, Source: plan.Source})
		return
	}
	cmd.Stderr = cmd.Stdout
	exited := make(chan struct{})
	s.mu.Lock()
	// Starting and assigning ownership are atomic with cancellation. A cancelled
	// preparation must never create a child after CancelRecovery has returned.
	if s.gen != gen {
		s.mu.Unlock()
		_ = out.Close()
		return
	}
	if err := cmd.Start(); err != nil {
		s.mu.Unlock()
		s.set(gen, Status{State: Failed, Err: err, Source: plan.Source})
		return
	}
	s.cmd, s.exited = cmd, exited
	s.mu.Unlock()
	startuptrace.Mark("host.spawned", strconv.Itoa(cmd.Process.Pid))

	ready := make(chan string, 1)
	scanDone := make(chan struct{})
	go func() {
		defer close(scanDone)
		s.scanWithTokens(out, ready, tokens, gen)
	}()
	var waitErr error
	go func() {
		// Drain StdoutPipe before calling cmd.Wait so trailing logs are never truncated.
		<-scanDone
		waitErr = cmd.Wait()
		close(exited)
	}()

	timer := time.NewTimer(s.opts.ReadyTimeout)
	defer timer.Stop()
	probe := time.NewTicker(750 * time.Millisecond)
	defer probe.Stop()
	probeC := probe.C
	loggedListen := false
	for {
		select {
		case url := <-ready:
			warm.Stop()
			probe.Stop()
			probeC = nil
			startuptrace.Mark("host.readyline")
			ctx, cancel := context.WithTimeout(launchCtx, 15*time.Second)
			endHandshake := startuptrace.Begin("host.handshake")
			ep, err := Redeem(ctx, s.client, url)
			endHandshake()
			cancel()
			if err != nil {
				s.log("auth: "+redactScanLine(err.Error(), tokens), gen)
				_ = terminate(cmd, true)
				<-exited
				s.fail(gen, err, plan.Source)
				return
			}
			// A successful transport handshake must not register a workspace.
			// The current UI owner initializes the Host's default only after the
			// workspace/session baselines; pre-registering the launch cwd here
			// suppresses that first-use initialization even in an empty profile.
			startuptrace.Mark("host.ready")
			if !s.set(gen, Status{State: Ready, Endpoint: ep, Source: plan.Source}) {
				return
			}
			<-exited
			s.exitedUnexpectedly(gen, waitErr, plan.Source)
			return
		case <-exited:
			s.exitedUnexpectedly(gen, waitErr, plan.Source)
			return
		case <-timer.C:
			_ = terminate(cmd, true)
			<-exited
			s.fail(gen, fmt.Errorf("dsh web did not become ready within %s", s.opts.ReadyTimeout), plan.Source)
			return
		case <-probeC:
			if loggedListen {
				continue
			}
			addr := net.JoinHostPort(launch.Host, strconv.Itoa(port))
			c, err := net.DialTimeout("tcp", addr, 200*time.Millisecond)
			if err != nil {
				continue
			}
			_ = c.Close()
			loggedListen = true
			probe.Stop()
			probeC = nil
			s.log("listening on http://"+addr+" — waiting for dsh web ready URL", gen)
		}
	}
}

func extractInjectedTokens(env []string) []string {
	var tokens []string
	for _, kv := range env {
		key, val, ok := strings.Cut(kv, "=")
		if !ok || val == "" {
			continue
		}
		switch strings.ToUpper(key) {
		case installctrl.EnvToken, "DSHD_TASK_CONTROL_TOKEN", "DSHD_PLATFORM_TOKEN":
			tokens = append(tokens, val)
		}
	}
	return tokens
}

func uniqueNonEmpty(items ...[]string) []string {
	seen := make(map[string]struct{})
	var out []string
	for _, list := range items {
		for _, v := range list {
			if v == "" {
				continue
			}
			if _, ok := seen[v]; ok {
				continue
			}
			seen[v] = struct{}{}
			out = append(out, v)
		}
	}
	return out
}

func (s *Supervisor) collectSecretTokens(extra ...string) []string {
	s.mu.Lock()
	taskTok := s.taskControlToken
	platTok := s.platformToken
	install := s.opts.Install
	env := append([]string(nil), s.opts.Launch.ExtraEnv...)
	s.mu.Unlock()
	var installTok string
	if install != nil {
		installTok = install.Token()
	}
	return uniqueNonEmpty(extra, []string{taskTok, platTok, installTok}, extractInjectedTokens(env))
}

func redactScanLine(line string, tokens []string) string {
	for _, tok := range tokens {
		if tok != "" {
			line = strings.ReplaceAll(line, tok, "[redacted]")
		}
	}
	return startuprec.RedactLine(line)
}

// scan copies output lines into the log and reports the first ready URL.
func (s *Supervisor) scan(r io.Reader, ready chan<- string, taskToken string, generation ...int) {
	s.scanWithTokens(r, ready, s.collectSecretTokens(taskToken), generation...)
}

func (s *Supervisor) scanWithTokens(r io.Reader, ready chan<- string, tokens []string, generation ...int) {
	sc := bufio.NewScanner(crToLF{r})
	sc.Buffer(make([]byte, 64*1024), 1024*1024)
	sent := false
	for sc.Scan() {
		line := sc.Text()
		if url, ok := ParseReady(line); ok && !sent {
			sent = true
			ready <- url
		}
		line = redactScanLine(line, tokens)
		s.notePluginLog(line, generation...)
		startuptrace.Log(line)
		s.log(line, generation...)
	}
}

type crToLF struct{ r io.Reader }

func (c crToLF) Read(p []byte) (int, error) {
	n, err := c.r.Read(p)
	for i := 0; i < n; i++ {
		if p[i] == '\r' {
			p[i] = '\n'
		}
	}
	return n, err
}

func (s *Supervisor) notePluginLog(line string, generation ...int) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.status.State != Starting || (len(generation) > 0 && generation[0] != s.gen) {
		return
	}
	if ApplyPluginLogLine(&s.status.PluginLoad, line) {
		s.enqueueNotifyLocked(s.status, false)
	}
}

func (s *Supervisor) exitedUnexpectedly(gen int, waitErr error, source string) {
	if !s.current(gen) {
		return
	}
	if waitErr == nil {
		waitErr = errors.New("dsh web exited")
	}
	s.log("exit: "+waitErr.Error(), gen)
	s.fail(gen, fmt.Errorf("dsh web exited: %w", waitErr), source)
}

// fail records the failure, or schedules a restart when attempts remain.
func (s *Supervisor) fail(gen int, err error, source string) {
	// Take one policy snapshot without holding the supervisor mutex. A save
	// never changes this decision or any timer it has already scheduled.
	policy := s.RestartPolicy()
	s.mu.Lock()
	if s.gen != gen {
		s.mu.Unlock()
		return
	}
	if s.status.State == Ready && !s.readyAt.IsZero() && time.Since(s.readyAt) >= restartStablePeriod {
		s.attempt = 0
	}
	s.readyAt = time.Time{}
	s.cmd, s.exited = nil, nil
	if s.recoveryCancelled {
		s.setLocked(Status{State: Failed, Err: err, Source: source})
		s.mu.Unlock()
		return
	}
	blob := err.Error() + "\n" + strings.Join(s.startLogs, "\n")
	if !s.opts.Launch.SkipUserPlugins && !s.skippedOnce && looksLikePluginTreeFailure(blob) {
		if persist := s.opts.OnPluginRecovery; persist != nil {
			logsCopy := append([]string(nil), s.startLogs...)
			s.mu.Unlock()
			saveErr := persist(err.Error(), logsCopy)
			s.mu.Lock()
			if s.gen != gen || s.recoveryCancelled {
				s.mu.Unlock()
				return
			}
			if saveErr != nil {
				s.setLocked(Status{State: Failed, Err: fmt.Errorf("plugin recovery could not be saved: %w", saveErr), Source: source})
				s.mu.Unlock()
				return
			}
		}
		s.opts.Launch.SkipUserPlugins = true
		s.opts.Prepare.SkipUserPlugins = true
		s.attemptRecovery = true
		s.skippedOnce = true
		next := s.nextGenLocked()
		s.recoveryActive, s.recoveryQueued = true, true
		s.setLocked(Status{State: Starting, Err: err, Source: source})
		s.mu.Unlock()
		s.log("plugin tree failed; retrying with --skip-user-plugins", next)
		go s.run(next)
		return
	}
	delay, retry := policy.retryDelay(s.attempt)
	if !retry {
		s.setLocked(Status{State: Failed, Err: err, Source: source})
		s.mu.Unlock()
		return
	}
	s.attempt++
	attempt := s.attempt
	next := s.nextGenLocked()
	s.recoveryActive, s.recoveryQueued = true, true
	s.setLocked(Status{State: Starting, Err: err, Source: source})
	s.restartTimer = time.AfterFunc(delay, func() { s.run(next) })
	s.mu.Unlock()
	s.log(fmt.Sprintf("restart %d/%d in %s", attempt, policy.MaxAttempts, delay), next)
}
