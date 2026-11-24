package startup

import (
	"context"
	"log"
	"sync"
	"time"

	"whaleisle/internal/startuptrace"
)

// Tier says when a subsystem may start.
type Tier string

const (
	// TierA is workbench-critical: started with the window or the Host
	// generation, because the first usable workbench depends on it.
	TierA Tier = "A"
	// TierOS starts with the process because the operating system integration
	// it provides must exist before the user can reach it (see Why).
	TierOS Tier = "OS"
	// TierB starts after WorkbenchReady of each Host generation, one job per
	// Stagger, so it never competes with the first list and projections.
	TierB Tier = "B"
	// TierC starts once per process, on first use of the feature or Idle
	// after the first WorkbenchReady, whichever comes first.
	TierC Tier = "C"
)

// Subsystem is one row of the startup plan.
type Subsystem struct {
	Name string
	Tier Tier
	// Owner is the code that runs it.
	Owner string
	// Trigger is the readiness level or event that starts it.
	Trigger string
	// Group names the Tier C job the subsystem loads with; first use of any
	// member releases the whole group.
	Group string
	// Why justifies the tier where it is not obvious.
	Why string
}

// Plan lists every subsystem the desktop starts, with its tier, owner and
// trigger. It is the single place to decide startup order: cmd/whaleisle
// registers jobs by these names and Scheduler refuses unknown ones.
var Plan = []Subsystem{
	// Tier A — workbench-critical.
	{Name: "supervisor", Tier: TierA, Owner: "harness.Supervisor", Trigger: "window created", Why: "the Host is the workbench's authority"},
	{Name: "sessions.attach", Tier: TierA, Owner: "session.Store.Attach", Trigger: "TransportReady"},
	{Name: "interaction.attach", Tier: TierA, Owner: "interaction.Store", Trigger: "TransportReady", Why: "approvals and questions must be answerable from the first turn"},
	{Name: "market.attach", Tier: TierA, Owner: "market.Coordinator.AttachHostForWorkspace", Trigger: "TransportReady",
		Why: "binds the client only (no RPC) so $events ready reaches plugin events; the inventory read itself is held by the coordinator until the workspace is ready"},
	{Name: "events", Tier: TierA, Owner: "cmd/whaleisle runDSH ($events)", Trigger: "TransportReady (mux carrier)"},
	{Name: "workspace.default", Tier: TierA, Owner: "session.Store.InitializeDefaultWorkspace", Trigger: "EventsReady"},
	{Name: "follows", Tier: TierA, Owner: "session.Store FollowWorkspace/FollowSelected/WatchControl", Trigger: "carrier connected"},
	{Name: "session.list", Tier: TierA, Owner: "session.Store.OnEventsReady", Trigger: "EventsReady"},
	{Name: "session.projections", Tier: TierA, Owner: "session.Store.RefreshProjections", Trigger: "carrier connected"},

	// OS integration that cannot wait.
	{Name: "tray", Tier: TierOS, Owner: "cmd/whaleisle setupDesktopTray", Trigger: "window created",
		Why: "close-to-tray and login-hidden launches hide the window; without the tray icon there is no way back to it"},
	{Name: "notifications", Tier: TierOS, Owner: "cmd/whaleisle setupSystemNotifications", Trigger: "window created",
		Why: "registers the toast activator before consumeSystemToastActivation handles the click that launched this process"},
	{Name: "ui.watchdog", Tier: TierOS, Owner: "cmd/whaleisle startNativeUIRecovery", Trigger: "window created",
		Why: "detects a hung UI thread; startup is the window it most needs to cover"},

	// Tier B — after WorkbenchReady, staggered.
	{Name: "model.catalog", Tier: TierB, Owner: "session.Store InitializeDefaultModel/RefreshModelCatalog", Trigger: "WorkbenchReady"},
	{Name: "agent.presets", Tier: TierB, Owner: "session.Store.RefreshAgentPresets", Trigger: "WorkbenchReady"},
	{Name: "skills", Tier: TierB, Owner: "session.Store.RefreshSkills", Trigger: "WorkbenchReady"},
	{Name: "permission.catalog", Tier: TierB, Owner: "session.Store.RefreshPermissionCatalogForEvents", Trigger: "WorkbenchReady (per events generation)"},
	{Name: "speech.catalog", Tier: TierB, Owner: "session.Store.WatchSpeechCatalogForEvents", Trigger: "WorkbenchReady (per events generation)"},
	{Name: "vision", Tier: TierB, Owner: "settings.Store.RefreshVision", Trigger: "WorkbenchReady"},
	{Name: "usage.billing", Tier: TierB, Owner: "usage.Store.LoadBilling", Trigger: "WorkbenchReady"},
	{Name: "product.event", Tier: TierB, Owner: "settings.ReportProductEvent desktop.ready", Trigger: "WorkbenchReady"},
	{Name: "custom.panels", Tier: TierB, Owner: "cmd/whaleisle openCustomPanels", Trigger: "WorkbenchReady",
		Why: "a local read, but its entries are visible navigation; holding it to Tier C would add them 30 s late"},

	// Tier C — first use or idle.
	{Name: "im.store", Tier: TierC, Group: "cold.stores", Owner: "cmd/whaleisle commitDeferredIMStore", Trigger: "IM page or idle"},
	{Name: "assistant.store", Tier: TierC, Group: "cold.stores", Owner: "assistant.Service.AttachAssistantStore", Trigger: "assistant page or idle"},
	{Name: "proactive.store", Tier: TierC, Group: "cold.stores", Owner: "cmd/whaleisle commitDeferredProactive", Trigger: "assistant page or idle"},
	{Name: "dshbot.roster", Tier: TierC, Group: "cold.stores", Owner: "assistant.Service.AttachDshbotRoster", Trigger: "assistant page or idle"},
	{Name: "ssh.tunnel", Tier: TierC, Group: "cold.stores", Owner: "cmd/whaleisle commitDeferredSSHTunnel", Trigger: "remote page or idle"},
	{Name: "proactive.monitor", Tier: TierC, Group: "proactive", Owner: "assistant.Service StartProactiveMonitor/BindProactiveRunner", Trigger: "assistant page or idle"},
	{Name: "phone.gateway", Tier: TierC, Group: "phone.gateway", Owner: "cmd/whaleisle refreshPhoneRemote", Trigger: "remote page or idle"},
	{Name: "whale.bridge.alerts", Tier: TierC, Group: "whale.bridge.alerts", Owner: "cmd/whaleisle startWhaleBridgeAlerts", Trigger: "idle"},
	{Name: "update.center", Tier: TierC, Group: "update.center", Owner: "updater.Center.StartBackground", Trigger: "update page or idle"},
}

// Lookup returns the plan row for name.
func Lookup(name string) (Subsystem, bool) {
	for _, s := range Plan {
		if s.Name == name {
			return s, true
		}
	}
	return Subsystem{}, false
}

func groupOf(name string) string {
	if s, ok := Lookup(name); ok && s.Group != "" {
		return s.Group
	}
	return name
}

// Scheduler releases Tier B and Tier C jobs against a Tracker.
type Scheduler struct {
	tracker *Tracker
	// Stagger separates consecutive Tier B jobs.
	Stagger time.Duration
	// Idle is how long after the first WorkbenchReady Tier C starts unasked.
	Idle time.Duration
	// Fallback releases Tier B when a generation reached EventsReady but
	// never WorkbenchReady (for example a list that keeps failing).
	Fallback time.Duration

	mu        sync.Mutex
	queued    map[uint64][]job
	fallback  map[uint64]*time.Timer
	deferred  map[string]func()
	requested map[string]bool
	ran       map[string]bool
	idleArmed bool
	started   map[string]int
	draining  map[uint64]bool
}

type job struct {
	ctx  context.Context
	name string
	run  func()
}

// NewScheduler watches tracker with the production timings: 150 ms between
// Tier B jobs, Tier C after 30 s, Tier B fallback 10 s after EventsReady.
func NewScheduler(tracker *Tracker) *Scheduler {
	s := &Scheduler{tracker: tracker, Stagger: 150 * time.Millisecond, Idle: 30 * time.Second, Fallback: 10 * time.Second,
		queued: map[uint64][]job{}, fallback: map[uint64]*time.Timer{}, deferred: map[string]func(){},
		requested: map[string]bool{}, ran: map[string]bool{}, started: map[string]int{}, draining: map[uint64]bool{}}
	tracker.OnChange(s.observe)
	return s
}

// Started reports how many times the named job has started (a test hook).
func (s *Scheduler) Started(name string) int {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.started[groupOf(name)]
}

// AfterWorkbench queues a Tier B job for Host generation gen. It runs once
// that generation reaches WorkbenchReady (immediately, staggered, when it is
// there already), and never after ctx ends or the generation changes.
func (s *Scheduler) AfterWorkbench(ctx context.Context, gen uint64, name string, run func()) {
	if !s.known(name, TierB) || run == nil {
		return
	}
	s.mu.Lock()
	s.queued[gen] = append(s.queued[gen], job{ctx: ctx, name: name, run: run})
	s.mu.Unlock()
	if r := s.tracker.Current(); r.Generation == gen && r.Level == WorkbenchReady {
		s.drain(gen)
	}
}

// Deferred registers the Tier C job for name's group. It runs once per
// process on Use or after Idle; a Use that came first runs it now.
func (s *Scheduler) Deferred(name string, run func()) {
	if !s.known(name, TierC) || run == nil {
		return
	}
	group := groupOf(name)
	s.mu.Lock()
	if s.ran[group] {
		s.mu.Unlock()
		return
	}
	s.deferred[group] = run
	now := s.requested[group]
	s.mu.Unlock()
	if now {
		s.runDeferred(group)
	}
}

// Use marks the first use of a Tier C feature and starts its group.
func (s *Scheduler) Use(name string) {
	group := groupOf(name)
	s.mu.Lock()
	s.requested[group] = true
	_, ok := s.deferred[group]
	s.mu.Unlock()
	if ok {
		s.runDeferred(group)
	}
}

// Ran reports whether name's Tier C group already started.
func (s *Scheduler) Ran(name string) bool {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.ran[groupOf(name)]
}

func (s *Scheduler) known(name string, tier Tier) bool {
	row, ok := Lookup(name)
	if !ok || row.Tier != tier {
		log.Printf("startup: %q is not a Tier %s row of the plan", name, tier)
		return false
	}
	return true
}

func (s *Scheduler) runDeferred(group string) {
	s.mu.Lock()
	run, ok := s.deferred[group]
	if !ok || s.ran[group] {
		s.mu.Unlock()
		return
	}
	s.ran[group] = true
	s.started[group]++
	delete(s.deferred, group)
	s.mu.Unlock()
	startuptrace.Mark("startup.tier_c", group)
	go run()
}

func (s *Scheduler) observe(r Readiness) {
	switch r.Level {
	case WorkbenchReady:
		s.stopFallback(r.Generation)
		s.drain(r.Generation)
		s.armIdle()
	case EventsReady:
		s.armFallback(r.Generation)
	case NotReady:
		// The generation stopped: its queued Tier B work belongs to a Host
		// that is gone.
		s.mu.Lock()
		clear(s.queued)
		s.mu.Unlock()
		s.stopFallback(r.Generation)
	}
}

func (s *Scheduler) armFallback(gen uint64) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.fallback[gen] != nil || s.Fallback <= 0 {
		return
	}
	s.fallback[gen] = time.AfterFunc(s.Fallback, func() {
		if r := s.tracker.Current(); r.Generation == gen && r.Level >= EventsReady {
			startuptrace.Mark("startup.tier_b_fallback")
			s.drain(gen)
		}
	})
}

func (s *Scheduler) stopFallback(gen uint64) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if t := s.fallback[gen]; t != nil {
		t.Stop()
		delete(s.fallback, gen)
	}
}

func (s *Scheduler) armIdle() {
	s.mu.Lock()
	if s.idleArmed {
		s.mu.Unlock()
		return
	}
	s.idleArmed = true
	idle := s.Idle
	s.mu.Unlock()
	time.AfterFunc(idle, func() {
		// Every Tier C group counts as requested from here on, so a job
		// registered after the idle window runs on registration.
		s.mu.Lock()
		for _, row := range Plan {
			if row.Tier == TierC {
				s.requested[groupOf(row.Name)] = true
			}
		}
		groups := make([]string, 0, len(s.deferred))
		for g := range s.deferred {
			groups = append(groups, g)
		}
		s.mu.Unlock()
		startuptrace.Mark("startup.idle")
		for _, g := range groups {
			s.runDeferred(g)
		}
	})
}

// drain runs generation gen's queued Tier B jobs one Stagger apart on one
// goroutine; jobs queued meanwhile join the same pass.
func (s *Scheduler) drain(gen uint64) {
	s.mu.Lock()
	if s.draining[gen] {
		s.mu.Unlock()
		return
	}
	s.draining[gen] = true
	s.mu.Unlock()
	go func() {
		first := true
		for {
			s.mu.Lock()
			q := s.queued[gen]
			if len(q) == 0 {
				delete(s.queued, gen)
				delete(s.draining, gen)
				s.mu.Unlock()
				return
			}
			next := q[0]
			s.queued[gen] = q[1:]
			s.mu.Unlock()
			if !first && s.Stagger > 0 {
				time.Sleep(s.Stagger)
			}
			first = false
			if next.ctx != nil && next.ctx.Err() != nil || s.tracker.Current().Generation != gen {
				continue
			}
			s.mu.Lock()
			s.started[next.name]++
			s.mu.Unlock()
			startuptrace.Mark("startup.tier_b", next.name)
			next.run()
		}
	}()
}
