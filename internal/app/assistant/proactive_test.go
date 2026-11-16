package assistantctl

import (
	"context"
	"encoding/json"
	"sync/atomic"
	"testing"
	"time"

	"whaleisle/internal/desktop/proactive"
	"whaleisle/internal/dsh"
)

func TestProactivePanelPersistAndRetainDraftOnConflict(t *testing.T) {
	dir := t.TempDir()
	svc := NewService(HostDeps{})
	t.Cleanup(svc.SuspendAll)
	svc.OpenProactive(dir)
	svc.SetProactiveDraft("检查当前候选", "保存明确的完成条件")
	svc.ProactiveModel().OnAutonomy(3)
	wait := func() {
		t.Helper()
		deadline := time.Now().Add(3 * time.Second)
		for time.Now().Before(deadline) {
			if !svc.ProactiveSnapshot().Busy {
				return
			}
			time.Sleep(5 * time.Millisecond)
		}
		t.Fatal("native task write did not finish")
	}
	svc.ProactiveModel().OnCreate()
	wait()
	m := svc.ProactiveModel()
	if m.Title != "" || len(m.State.Tasks) != 1 || m.Selected != m.State.Tasks[0].ID {
		t.Fatalf("native create did not consume saved draft: %+v", m)
	}
	if m.State.Tasks[0].Autonomy != 3 {
		t.Fatal("native autonomy selection not persisted")
	}
	other, err := proactive.Open(dir)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := other.Create(proactive.CreateInput{Title: "外部新增的任务"}); err != nil {
		t.Fatal(err)
	}
	svc.SetProactiveDraft("失败时保留的草稿", "")
	svc.ProactiveModel().OnCreate()
	wait()
	m = svc.ProactiveModel()
	if m.Error == "" || m.Title != "失败时保留的草稿" || len(m.State.Tasks) != 1 {
		t.Fatalf("conflict erased draft or published unsaved state: %+v", m)
	}
	reopened, err := proactive.Open(dir)
	if err != nil || len(reopened.Snapshot().Tasks) != 2 {
		t.Fatalf("conflict overwrote external work: %v", err)
	}
}

func TestProactiveSourceControlsKeepUnrelatedDraft(t *testing.T) {
	svc := NewService(HostDeps{})
	t.Cleanup(svc.SuspendAll)
	dir := t.TempDir()
	svc.OpenProactive(dir)
	wait := func() {
		t.Helper()
		deadline := time.Now().Add(3 * time.Second)
		for time.Now().Before(deadline) {
			if !svc.ProactiveSnapshot().SourceBusy {
				return
			}
			time.Sleep(5 * time.Millisecond)
		}
		t.Fatal("source mutation did not settle")
	}
	svc.ProactiveModel().OnSourceEdit("owner/repo")
	svc.ProactiveModel().OnSourceAdd()
	wait()
	m := svc.ProactiveModel()
	if m.SourceRepository != "" || len(m.State.Sources) != 1 || !m.State.Sources[0].Enabled {
		t.Fatalf("source not saved: %s", m.SourceError)
	}
	m.OnSourceEdit("owner/pending-draft")
	svc.ProactiveModel().OnSourceToggle("owner/repo", false)
	wait()
	m = svc.ProactiveModel()
	if m.SourceRepository != "owner/pending-draft" || m.State.Sources[0].Enabled {
		t.Fatal("pause cleared another source draft")
	}
	reopened, err := proactive.Open(dir)
	if err != nil || reopened.Sources()[0].Enabled {
		t.Fatalf("pause did not persist: %v", err)
	}
	m.OnSourceEdit("owner/repo?unsafe")
	svc.ProactiveModel().OnSourceAdd()
	wait()
	m = svc.ProactiveModel()
	if m.SourceError == "" || m.SourceRepository != "owner/repo?unsafe" {
		t.Fatal("failed source configuration lost draft")
	}
	if status, err := svc.DeliverProactiveNotice(context.Background(), proactive.Notice{ID: "test"}); err != nil || status != "unavailable" {
		t.Fatalf("no-window delivery did not remain in-app: %q %v", status, err)
	}
}

type fakeNotification struct {
	shown   atomic.Bool
	closed  atomic.Bool
	onClick func()
	showErr error
}

func (n *fakeNotification) Show() error {
	n.shown.Store(true)
	return n.showErr
}

func (n *fakeNotification) Close() {
	n.closed.Store(true)
}

func (n *fakeNotification) OnClick(fn func()) {
	n.onClick = fn
}

func TestProactiveLifecycleDispatchFreezeAndMonitor(t *testing.T) {
	dir := t.TempDir()
	var quitting atomic.Bool
	var focused atomic.Bool
	var lastNotif *fakeNotification
	revealed := false
	selectedPanel := ""

	svc := NewService(HostDeps{
		DshHome:                    func() string { return dir },
		Quitting:                   quitting.Load,
		Window:                     func() any { return "win" },
		SystemNotificationsSupport: func() bool { return true },
		WindowStatus: func(win any) WindowState {
			if win == nil {
				return WindowState{}
			}
			return WindowState{Exists: true, Visible: true, Focused: focused.Load()}
		},
		NewNotification: func(opts NotificationOptions) NotificationHandle {
			lastNotif = &fakeNotification{}
			return lastNotif
		},
		RevealMainConversation: func() { revealed = true },
		SelectMainPanel:        func(p string) { selectedPanel = p },
	})
	t.Cleanup(svc.SuspendAll)

	if _, err := svc.DispatchProactive(context.Background(), "tasks/list", nil); err == nil {
		t.Fatal("expected error before proactive store is attached")
	}
	// Nil-safe freeze without runner
	release := svc.FreezeProactiveAdmissions()
	release()
	if svc.ProactiveDeliveryRunning() {
		t.Fatal("delivery cannot be running without runner")
	}

	psvc, err := proactive.Open(dir)
	if err != nil {
		t.Fatal(err)
	}
	svc.AttachStores(nil, psvc, nil)
	if svc.ProactiveService() != psvc {
		t.Fatal("AttachStores did not attach proactive service")
	}

	raw, err := svc.DispatchProactive(context.Background(), "task/create", json.RawMessage(`{"title":"proactive dispatch test"}`))
	if err != nil {
		t.Fatalf("DispatchProactive failed: %v", err)
	}
	if raw == nil {
		t.Fatal("expected non-nil task create response")
	}

	// Bind runner and test FreezeAdmissions + Monitor start/wake/stop
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	svc.BindProactiveRunner(ctx, &dsh.Client{Endpoint: dsh.Endpoint{Origin: "http://127.0.0.1:0"}})
	if !svc.ProactiveSnapshot().RunnerActive {
		t.Fatal("expected runner active after BindProactiveRunner")
	}
	unfreeze := svc.FreezeProactiveAdmissions()
	unfreeze()

	svc.StartProactiveMonitor()
	if !svc.ProactiveSnapshot().MonitorActive {
		t.Fatal("expected monitor active after StartProactiveMonitor")
	}
	svc.WakeProactiveMonitor(true)

	// Deliver notice when focused vs unfocused
	focused.Store(true)
	st, err := svc.DeliverProactiveNotice(context.Background(), proactive.Notice{ID: "n1", Title: "hello"})
	if err != nil || st != "in-app" {
		t.Fatalf("focused delivery got (%q, %v), want (in-app, nil)", st, err)
	}

	focused.Store(false)
	st, err = svc.DeliverProactiveNotice(context.Background(), proactive.Notice{ID: "n2", Title: "done", Kind: "discovery", Body: "3"})
	if err != nil || st != "shown" || lastNotif == nil || !lastNotif.shown.Load() {
		t.Fatalf("unfocused delivery got (%q, %v), notif=%v", st, err, lastNotif)
	}

	// OpenProactiveNotice with nonexistent ID is a safe no-op
	svc.OpenProactiveNotice(proactive.Notice{ID: "missing"})
	if revealed || selectedPanel != "" {
		t.Fatal("missing notice should not navigate")
	}

	// Quitting cancels notice delivery
	quitting.Store(true)
	if _, err := svc.DeliverProactiveNotice(context.Background(), proactive.Notice{ID: "n3"}); err != context.Canceled {
		t.Fatalf("expected context.Canceled while quitting, got %v", err)
	}

	svc.SuspendAll()
	snap := svc.ProactiveSnapshot()
	if snap.RunnerActive || snap.MonitorActive {
		t.Fatalf("SuspendAll left runner=%v or monitor=%v active", snap.RunnerActive, snap.MonitorActive)
	}
}
