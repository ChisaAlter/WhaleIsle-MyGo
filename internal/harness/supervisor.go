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
	"os/exec"
	"strconv"
	"strings"
	"sync"
	"time"

	"whaleisle/internal/harness/installctrl"
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
	// Attempt counts automatic restarts since the last Ready.
	Attempt int
	// PluginLoad carries overlay/runtime plugin progress for the boot page.
	PluginLoad PluginProgress
	// Recovery is true when this attempt skips the user plugin layer.
	Recovery bool
}

// Options configure a Supervisor.
type Options struct {
	Launch LaunchOptions
	// Prepare configures desktop plugin overlays/junctions before each start.
	Prepare PrepareOptions
	// Workspace is registered with workspace/create after auth (optional).
	Workspace string
	// AutoRestart relaunches dsh after an unexpected exit, up to MaxAttempts
	// times with doubling delays from BaseDelay.
	AutoRestart bool
	MaxAttempts int
	BaseDelay   time.Duration
	// ReadyTimeout bounds one start; npx downloads can take minutes.
	ReadyTimeout time.Duration
	// OnChange receives every status change, on the supervisor's goroutines.
	OnChange func(Status)
	// OnPluginRecovery must persist the marker before a skipped-plugin retry.
	// Called under the supervisor lock; it must not call back into Supervisor.
	OnPluginRecovery func(reason string, logs []string) error
	// Install is the loopback install control server; started with the supervisor
	// and its env is injected into each harness child.
	Install *installctrl.Server
}

// logLimit is how many output lines the boot log keeps.
const logLimit = 400

// stopGrace is how long dsh gets to exit before it is killed.
const stopGrace = 5 * time.Second

// Supervisor owns one dsh web child at a time.
type Supervisor struct {
	opts   Options
	client *http.Client

	protectionMu     sync.Mutex
	taskControlToken string
	mu               sync.Mutex
	status           Status
	cmd              *exec.Cmd
	exited           chan struct{}
	gen              int
	logs             []string
	startLogs        []string // current attempt only; old attempts cannot implicate plugins
	attempt          int
	// skippedOnce is true after a plugin-tree failure retried with --skip-user-plugins.
	skippedOnce bool
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
	s.mu.Lock()
	if s.status.State == Starting || s.status.State == Ready {
		s.mu.Unlock()
		return
	}
	install := s.opts.Install
	s.attempt = 0
	gen := s.nextGenLocked()
	s.mu.Unlock()
	if install != nil {
		if err := install.Start(context.Background()); err != nil {
			s.set(gen, Status{State: Failed, Err: fmt.Errorf("install control: %w", err)})
			return
		}
	}
	go s.run(gen)
}

// Restart stops the current child and starts a new one.
func (s *Supervisor) Restart() {
	s.Stop()
	s.Start()
}

// Stop ends the child, waiting for it to exit.
func (s *Supervisor) Stop() {
	s.mu.Lock()
	install := s.opts.Install
	s.nextGenLocked()
	cmd, exited := s.cmd, s.exited
	s.cmd, s.exited = nil, nil
	s.setLocked(Status{State: Stopped})
	s.mu.Unlock()
	if install != nil {
		install.Stop()
	}
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
	s.gen++
	s.taskControlToken = ""
	return s.gen
}

func (s *Supervisor) current(gen int) bool {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.gen == gen
}

func (s *Supervisor) setLocked(st Status) {
	st.Generation = s.gen
	st.Attempt = s.attempt
	st.Recovery = s.opts.Launch.SkipUserPlugins
	s.status = st
	if s.opts.OnChange != nil {
		go s.opts.OnChange(st)
	}
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

func (s *Supervisor) log(line string) {
	s.mu.Lock()
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
	st := s.status
	cb := s.opts.OnChange
	notify := st.State == Starting && cb != nil
	s.mu.Unlock()
	if notify {
		go cb(st)
	}
}

func (s *Supervisor) run(gen int) {
	// Generate only at actual child launch, not during configuration or inspection.
	var secret [24]byte
	if _, err := rand.Read(secret[:]); err != nil {
		s.set(gen, Status{State: Failed, Err: fmt.Errorf("task-control token initialization failed")})
		return
	}
	token := hex.EncodeToString(secret[:])
	s.mu.Lock()
	if s.gen != gen {
		s.mu.Unlock()
		return
	}
	s.taskControlToken = token
	s.startLogs = nil
	launch := s.opts.Launch
	launch.ExtraEnv = append([]string(nil), launch.ExtraEnv...)
	if install := s.opts.Install; install != nil {
		launch.ExtraEnv = append(launch.ExtraEnv, install.Env()...)
	}
	launch.ExtraEnv = append(launch.ExtraEnv, "DSHD_TASK_CONTROL_TOKEN="+token)
	if s.opts.Prepare.Dshbot {
		launch.ExtraEnv = append(launch.ExtraEnv, "DSHD_DSHBOT_ENABLED=1")
	}
	s.mu.Unlock()

	port, err := FreePort(launch.Host, launch.Port)
	if err != nil {
		s.set(gen, Status{State: Failed, Err: err})
		return
	}
	var plan Plan
	var patches []string
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
	if err != nil {
		s.set(gen, Status{State: Failed, Err: err})
		return
	}
	if !s.set(gen, Status{State: Starting, Source: plan.Source, PluginLoad: PluginProgress{Phase: "runtime"}}) {
		return
	}
	s.log(fmt.Sprintf("» %s %s", plan.Command, strings.Join(plan.Args, " ")))
	if n := len(patches); n > 0 {
		overlayLine := fmt.Sprintf("» %d desktop --patch overlay(s)", n)
		s.log(overlayLine)
		s.notePluginLog(overlayLine)
	}

	cmd := exec.Command(plan.Command, plan.Args...)
	cmd.Env = plan.Env
	detach(cmd)
	out, err := cmd.StdoutPipe()
	if err != nil {
		s.set(gen, Status{State: Failed, Err: err, Source: plan.Source})
		return
	}
	cmd.Stderr = cmd.Stdout
	if err := cmd.Start(); err != nil {
		s.set(gen, Status{State: Failed, Err: err, Source: plan.Source})
		return
	}
	exited := make(chan struct{})
	s.mu.Lock()
	if s.gen != gen {
		s.mu.Unlock()
		_ = terminate(cmd, true)
		go func() { cmd.Wait(); close(exited) }()
		return
	}
	s.cmd, s.exited = cmd, exited
	s.mu.Unlock()

	ready := make(chan string, 1)
	go s.scan(out, ready, token)
	var waitErr error
	go func() {
		waitErr = cmd.Wait()
		close(exited)
	}()

	timer := time.NewTimer(s.opts.ReadyTimeout)
	defer timer.Stop()
	probe := time.NewTicker(750 * time.Millisecond)
	defer probe.Stop()
	loggedListen := false
	for {
		select {
		case url := <-ready:
			ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
			ep, err := Redeem(ctx, s.client, url)
			cancel()
			if err != nil {
				s.log("auth: " + err.Error())
				_ = terminate(cmd, true)
				<-exited
				s.fail(gen, err, plan.Source)
				return
			}
			if ws := s.opts.Workspace; ws != "" {
				wctx, wcancel := context.WithTimeout(context.Background(), 15*time.Second)
				werr := EnsureWorkspace(wctx, s.client, ep, ws)
				wcancel()
				if werr != nil {
					s.log("workspace: " + werr.Error())
				} else {
					s.log("workspace registered: " + ws)
				}
			}
			s.mu.Lock()
			s.attempt = 0
			s.mu.Unlock()
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
		case <-probe.C:
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
			s.log("listening on http://" + addr + " — waiting for dsh web ready URL")
		}
	}
}

// scan copies output lines into the log and reports the first ready URL.
func (s *Supervisor) scan(r io.Reader, ready chan<- string, taskToken string) {
	sc := bufio.NewScanner(crToLF{r})
	sc.Buffer(make([]byte, 64*1024), 1024*1024)
	sent := false
	for sc.Scan() {
		line := sc.Text()
		// Capture the child generation token so late output after Stop is also safe.
		if taskToken != "" {
			line = strings.ReplaceAll(line, taskToken, "[redacted]")
		}
		if url, ok := ParseReady(line); ok && !sent {
			sent = true
			ready <- url
			// The launch token is a credential; keep it out of the log.
			if i := strings.Index(line, "?token="); i >= 0 {
				line = line[:i] + "?token=…"
			}
		}
		s.notePluginLog(line)
		s.log(line)
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

func (s *Supervisor) notePluginLog(line string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.status.State != Starting {
		return
	}
	if ApplyPluginLogLine(&s.status.PluginLoad, line) && s.opts.OnChange != nil {
		st := s.status
		go s.opts.OnChange(st)
	}
}

func (s *Supervisor) exitedUnexpectedly(gen int, waitErr error, source string) {
	if !s.current(gen) {
		return
	}
	if waitErr == nil {
		waitErr = errors.New("dsh web exited")
	}
	s.log("exit: " + waitErr.Error())
	s.fail(gen, fmt.Errorf("dsh web exited: %w", waitErr), source)
}

// fail records the failure, or schedules a restart when attempts remain.
func (s *Supervisor) fail(gen int, err error, source string) {
	s.mu.Lock()
	if s.gen != gen {
		s.mu.Unlock()
		return
	}
	s.cmd, s.exited = nil, nil
	blob := err.Error() + "\n" + strings.Join(s.startLogs, "\n")
	if !s.opts.Launch.SkipUserPlugins && !s.skippedOnce && looksLikePluginTreeFailure(blob) {
		if persist := s.opts.OnPluginRecovery; persist != nil {
			if saveErr := persist(err.Error(), append([]string(nil), s.startLogs...)); saveErr != nil {
				s.setLocked(Status{State: Failed, Err: fmt.Errorf("plugin recovery could not be saved: %w", saveErr), Source: source})
				s.mu.Unlock()
				return
			}
		}
		s.opts.Launch.SkipUserPlugins = true
		s.opts.Prepare.SkipUserPlugins = true
		s.skippedOnce = true
		next := s.nextGenLocked()
		s.setLocked(Status{State: Starting, Err: err, Source: source})
		s.mu.Unlock()
		s.log("plugin tree failed; retrying with --skip-user-plugins")
		go s.run(next)
		return
	}
	if !s.opts.AutoRestart || s.attempt >= s.opts.MaxAttempts {
		s.setLocked(Status{State: Failed, Err: err, Source: source})
		s.mu.Unlock()
		return
	}
	delay := s.opts.BaseDelay << s.attempt
	s.attempt++
	next := s.nextGenLocked()
	s.setLocked(Status{State: Starting, Err: err, Source: source})
	s.mu.Unlock()
	s.log(fmt.Sprintf("restart %d/%d in %s", s.attempt, s.opts.MaxAttempts, delay))
	time.AfterFunc(delay, func() { s.run(next) })
}
