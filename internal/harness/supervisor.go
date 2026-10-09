// Package harness runs DeepSeek Harness (`dsh web`) as a child process and
// tracks the authenticated endpoint it serves.
package harness

import (
	"bufio"
	"context"
	"errors"
	"fmt"
	"io"
	"net/http"
	"os/exec"
	"strings"
	"sync"
	"time"
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
	State    State
	Endpoint Endpoint
	// Err explains a Failed state.
	Err error
	// Source names the launcher of the current or last attempt.
	Source string
	// Attempt counts automatic restarts since the last Ready.
	Attempt int
}

// Options configure a Supervisor.
type Options struct {
	Launch LaunchOptions
	// AutoRestart relaunches dsh after an unexpected exit, up to MaxAttempts
	// times with doubling delays from BaseDelay.
	AutoRestart bool
	MaxAttempts int
	BaseDelay   time.Duration
	// ReadyTimeout bounds one start; npx downloads can take minutes.
	ReadyTimeout time.Duration
	// OnChange receives every status change, on the supervisor's goroutines.
	OnChange func(Status)
}

// logLimit is how many output lines the boot log keeps.
const logLimit = 400

// stopGrace is how long dsh gets to exit before it is killed.
const stopGrace = 5 * time.Second

// Supervisor owns one dsh web child at a time.
type Supervisor struct {
	opts   Options
	client *http.Client

	mu      sync.Mutex
	status  Status
	cmd     *exec.Cmd
	exited  chan struct{}
	gen     int
	logs    []string
	attempt int
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

// SetSkipUserPlugins changes the recovery flag used by the next start.
func (s *Supervisor) SetSkipUserPlugins(skip bool) {
	s.mu.Lock()
	s.opts.Launch.SkipUserPlugins = skip
	s.mu.Unlock()
}

// Start launches dsh unless it is already starting or ready.
func (s *Supervisor) Start() {
	s.mu.Lock()
	if s.status.State == Starting || s.status.State == Ready {
		s.mu.Unlock()
		return
	}
	s.attempt = 0
	gen := s.nextGenLocked()
	s.mu.Unlock()
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
	s.nextGenLocked()
	cmd, exited := s.cmd, s.exited
	s.cmd, s.exited = nil, nil
	s.setLocked(Status{State: Stopped})
	s.mu.Unlock()
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
	return s.gen
}

func (s *Supervisor) current(gen int) bool {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.gen == gen
}

func (s *Supervisor) setLocked(st Status) {
	st.Attempt = s.attempt
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
	defer s.mu.Unlock()
	s.logs = append(s.logs, line)
	if over := len(s.logs) - logLimit; over > 0 {
		s.logs = append(s.logs[:0], s.logs[over:]...)
	}
}

func (s *Supervisor) run(gen int) {
	s.mu.Lock()
	launch := s.opts.Launch
	s.mu.Unlock()

	port, err := FreePort(launch.Host, launch.Port)
	if err != nil {
		s.set(gen, Status{State: Failed, Err: err})
		return
	}
	plan, err := Resolve(launch, port)
	if err != nil {
		s.set(gen, Status{State: Failed, Err: err})
		return
	}
	if !s.set(gen, Status{State: Starting, Source: plan.Source}) {
		return
	}
	s.log(fmt.Sprintf("» %s %s", plan.Command, strings.Join(plan.Args, " ")))

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
	go s.scan(out, ready)
	var waitErr error
	go func() {
		waitErr = cmd.Wait()
		close(exited)
	}()

	timer := time.NewTimer(s.opts.ReadyTimeout)
	defer timer.Stop()
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
		s.mu.Lock()
		s.attempt = 0
		s.mu.Unlock()
		if !s.set(gen, Status{State: Ready, Endpoint: ep, Source: plan.Source}) {
			return
		}
		<-exited
		s.exitedUnexpectedly(gen, waitErr, plan.Source)
	case <-exited:
		s.exitedUnexpectedly(gen, waitErr, plan.Source)
	case <-timer.C:
		_ = terminate(cmd, true)
		<-exited
		s.fail(gen, fmt.Errorf("dsh web did not become ready within %s", s.opts.ReadyTimeout), plan.Source)
	}
}

// scan copies output lines into the log and reports the first ready URL.
func (s *Supervisor) scan(r io.Reader, ready chan<- string) {
	sc := bufio.NewScanner(r)
	sc.Buffer(make([]byte, 64*1024), 1024*1024)
	sent := false
	for sc.Scan() {
		line := sc.Text()
		if url, ok := ParseReady(line); ok && !sent {
			sent = true
			ready <- url
			// The launch token is a credential; keep it out of the log.
			if i := strings.Index(line, "?token="); i >= 0 {
				line = line[:i] + "?token=…"
			}
		}
		s.log(line)
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
