package main

import (
	"context"
	"os"
	"slices"
	"sync"

	"github.com/egoist/mygo"
	"github.com/egoist/mygo/ui"

	"whaleisle/internal/app/startup"
	"whaleisle/internal/dsh"
	"whaleisle/internal/dsh/api"
	"whaleisle/internal/harness"
	"whaleisle/internal/store/session"
	settingsstore "whaleisle/internal/store/settings"
)

// startupOrchestration wires the shell to internal/app/startup: one readiness
// tracker for the Host generations this window sees and the scheduler that
// releases Tier B and Tier C work against it. The plan table itself lives in
// internal/app/startup/plan.go.
type startupOrchestration struct {
	once    sync.Once
	tracker *startup.Tracker
	plan    *startup.Scheduler

	mu          sync.Mutex
	generation  uint64 // last Host generation handed to the tracker
	customOnce  sync.Once
	proactiveOn bool // the proactive Tier C group ran; later Hosts bind at once
}

func (s *mainState) startupPlan() (*startup.Tracker, *startup.Scheduler) {
	o := &s.startupOrch
	o.once.Do(func() {
		o.tracker = startup.NewTracker()
		o.plan = startup.NewScheduler(o.tracker)
		o.tracker.OnChange(func(startup.Readiness) {
			s.mu.Lock()
			win := s.win
			s.mu.Unlock()
			if win != nil {
				ui.FrameTraceCause("startup-level")
				win.Invalidate()
			}
		})
	})
	return o.tracker, o.plan
}

// startupLevel is the readiness the boot page explains while it stays up.
func (s *mainState) startupLevel() startup.Level {
	tracker, _ := s.startupPlan()
	return tracker.Current().Level
}

// beginStartupGeneration opens a new tracker generation for a Host client.
func (s *mainState) beginStartupGeneration() uint64 {
	tracker, _ := s.startupPlan()
	o := &s.startupOrch
	o.mu.Lock()
	o.generation++
	gen := o.generation
	o.mu.Unlock()
	tracker.Begin(gen)
	return gen
}

// endStartupGeneration drops readiness when the Host stops.
func (s *mainState) endStartupGeneration() {
	tracker, _ := s.startupPlan()
	tracker.Reset()
}

// observeWorkbenchReady raises gen to WorkbenchReady once the session store
// has its baselines for the current events generation. It is called from the
// store refresh hook and costs one atomic load until EventsReady.
func (s *mainState) observeWorkbenchReady(gen uint64, sessions *session.Store) {
	tracker, _ := s.startupPlan()
	if r := tracker.Current(); r.Generation != gen || r.Level != startup.EventsReady || sessions == nil {
		return
	}
	if sessions.WorkbenchBaseline() {
		tracker.Reach(gen, startup.WorkbenchReady)
	}
}

// queueHostTierB queues the per-Host Tier B reads for generation gen.
func (s *mainState) queueHostTierB(ctx context.Context, gen uint64, client *dsh.Client, sessions *session.Store) {
	_, plan := s.startupPlan()
	s.mu.Lock()
	setStore, usageStore, coldDir := s.settings, s.usage, s.coldStoresDir
	s.mu.Unlock()
	plan.AfterWorkbench(ctx, gen, "model.catalog", func() {
		sessions.RefreshModelCatalog(ctx)
	})
	plan.AfterWorkbench(ctx, gen, "agent.presets", func() { sessions.RefreshAgentPresets(ctx) })
	plan.AfterWorkbench(ctx, gen, "skills", func() { sessions.RefreshSkills(ctx) })
	if setStore != nil {
		plan.AfterWorkbench(ctx, gen, "vision", func() { setStore.RefreshVision(ctx) })
	}
	if usageStore != nil {
		plan.AfterWorkbench(ctx, gen, "usage.billing", func() { usageStore.LoadBilling(ctx) })
	}
	plan.AfterWorkbench(ctx, gen, "product.event", func() { go reportDesktopReady(client) })
	if coldDir != "" {
		plan.AfterWorkbench(ctx, gen, "custom.panels", func() {
			s.startupOrch.customOnce.Do(func() { go s.openCustomPanels(coldDir) })
		})
	}
}

// registerDeferredStartup registers the process-wide Tier C jobs. Each runs
// on first use of its feature or 30 s after the first WorkbenchReady.
func (s *mainState) registerDeferredStartup(win *mygo.Window, userData string, osIntegration bool) {
	_, plan := s.startupPlan()
	plan.Deferred("update.center", func() { s.startNativeUpdateCenter(win) })
	plan.Deferred("proactive.monitor", func() {
		s.assistantService().StartProactiveMonitor()
		s.startupOrch.mu.Lock()
		s.startupOrch.proactiveOn = true
		s.startupOrch.mu.Unlock()
		s.dshMu.Lock()
		ctx, client := s.dshCtx, s.dshClient
		if ctx != nil && client != nil && ctx.Err() == nil {
			s.bindProactiveRunner(ctx, client)
		}
		s.dshMu.Unlock()
	})
	if userData != "" {
		plan.Deferred("im.store", func() {
			s.triggerDeferredColdStores(userData, &deferredColdStoreOps{openCustomPanels: func(string) {}})
		})
	}
	if osIntegration {
		plan.Deferred("whale.bridge.alerts", func() { mygo.RunOnMain(func() { s.startWhaleBridgeAlerts(win) }) })
	}
}

// proactiveReady reports whether the proactive Tier C group already ran, so a
// new Host generation binds its runner immediately.
func (s *mainState) proactiveReady() bool {
	s.startupOrch.mu.Lock()
	defer s.startupOrch.mu.Unlock()
	return s.startupOrch.proactiveOn
}

// deferPhoneBootStart queues the claimed phone gateway start as Tier C.
func (s *mainState) deferPhoneBootStart(start func()) {
	_, plan := s.startupPlan()
	plan.Deferred("phone.gateway", start)
}

// noteRouteUse is the first-use trigger for Tier C features.
func (s *mainState) noteRouteUse(route mainRoute) {
	_, plan := s.startupPlan()
	switch route {
	case routeAssistant:
		plan.Use("assistant.store")
		plan.Use("proactive.monitor")
	case routeIM:
		plan.Use("im.store")
	case routeRemote:
		plan.Use("ssh.tunnel")
		plan.Use("phone.gateway")
	case routeUpdatePreferences:
		plan.Use("update.center")
	}
}

// bootLogSnapshot returns the published boot log without copying. Callers
// must not modify it; the slice is replaced, never edited, by its writers.
func (s *mainState) bootLogSnapshot() []string {
	if v := s.bootLogs.Load(); v != nil {
		return slices.Clip(v.Value)
	}
	return nil
}

// appendBootLogLocked publishes the log with line added. Caller holds s.mu,
// which serializes the shell's own writers; the copy keeps every snapshot a
// reader already holds unchanged.
func (s *mainState) appendBootLogLocked(line string) {
	s.bootLogs.Publish(append(s.bootLogSnapshot(), line))
}

// bootProgressOnly reports whether next differs from cur only in Starting
// progress (plugin load and log lines): the page repaints, but nothing that
// depends on the Host state needs to run again.
func bootProgressOnly(cur, next harness.Status) bool {
	return cur.State == harness.Starting && next.State == harness.Starting && cur.Generation == next.Generation &&
		cur.Endpoint == next.Endpoint && cur.Err == nil && next.Err == nil && cur.Source == next.Source &&
		cur.Attempt == next.Attempt && cur.RuntimeFailure == next.RuntimeFailure && cur.Recovery == next.Recovery &&
		cur.RecoveryActive == next.RecoveryActive && cur.RecoveryScheduled == next.RecoveryScheduled &&
		cur.RecoveryCancelled == next.RecoveryCancelled
}

func reportDesktopReady(client *dsh.Client) {
	_ = settingsstore.ReportProductEvent(context.Background(), client, api.ProductEvent(
		[]byte(`{"name":"desktop.ready","source":"whaleisle"}`),
	))
}

// bootPageCovers reports whether the cold-start boot page is the window's
// content and still waiting for the workspace baseline, so early store
// notifications need no frame of their own. Once coldHandoffArmed is set,
// store completions must redraw so paintColdLoading can finish the handoff
// promptly even under ReduceMotion.
func (s *mainState) bootPageCovers() bool {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.coldLoading && !s.coldHandoffArmed && s.mainRoute == routeConversation && !s.quitting &&
		!(!s.settingsLaunchHandled && slices.Contains(os.Args, "--settings"))
}
