package mygo

import (
	"errors"
	"strings"
	"testing"

	"github.com/egoist/mygo/internal/platform"
)

func TestToastOptionsValidateAndSnapshot(t *testing.T) {
	for _, raw := range []string{"", "<toast>", "<other/>", "<toast/><toast/>", "<!DOCTYPE toast><toast/>"} {
		if raw != "" && validateToastXML(raw) == nil {
			t.Errorf("accepted %q", raw)
		}
	}
	xml := `<toast><visual><binding template="ToastGeneric"><text>任务完成</text><progress value="0.5" status="运行中"/></binding></visual><actions><input id="reply" type="text"/><action content="回复" arguments="reply" activationType="foreground"/></actions></toast>`
	n := NewNotification(NotificationOptions{ID: "full", Toast: &ToastOptions{XML: xml, Actions: []ToastAction{{Label: "Open", Arguments: "go"}}}})
	p, err := n.snapshot()
	if err != nil {
		t.Fatal(err)
	}
	if p.Toast.XML != xml || p.Toast.Actions[0].Arguments != "go" {
		t.Fatalf("snapshot=%+v", p.Toast)
	}
	n.opts.Toast.Actions[0].Arguments = "changed"
	if p.Toast.Actions[0].Arguments != "go" {
		t.Fatal("mutable action slice escaped snapshot")
	}
	if _, err := NewNotification(NotificationOptions{Toast: &ToastOptions{Duration: "forever"}}).snapshot(); err == nil {
		t.Fatal("invalid duration accepted")
	}
	if _, err := NewNotification(NotificationOptions{Toast: &ToastOptions{XML: strings.Repeat("x", 257<<10)}}).snapshot(); err == nil {
		t.Fatal("oversized XML accepted")
	}
}

func TestNotificationNativeEventsOnMainAndTimeoutHistory(t *testing.T) {
	n := showNotification(t, NotificationOptions{ID: "event-history", Title: "title"})
	var events []string
	n.OnAction(func(a NotificationActivation) {
		if !isMainThread() {
			t.Error("action not on main thread")
		}
		events = append(events, "action:"+a.Arguments+":"+a.UserInput["reply"])
	})
	n.OnClick(func() { events = append(events, "click") })
	n.OnClose(func(r NotificationCloseReason) {
		if !isMainThread() {
			t.Error("close not on main thread")
		}
		events = append(events, string(r))
	})
	onMain(func() { fb.DismissNotification(n.id, "timed-out") })
	rows, err := NotificationHistory("")
	if err != nil {
		t.Fatal(err)
	}
	found := false
	for _, r := range rows {
		found = found || r.ID == n.id
	}
	if !found {
		t.Fatal("timed-out notification missing from history")
	}
	onMain(func() {
		fb.ActivateNotification(n.id, platform.NotificationActivation{Arguments: "reply", UserInput: map[string]string{"reply": "yes"}})
	})
	if got := strings.Join(events, ","); got != "timed-out,action:reply:yes,click,activated" {
		t.Fatal(got)
	}
}

func TestNotificationActivationReplacementKeepsNewListeners(t *testing.T) {
	first := showNotification(t, NotificationOptions{ID: "replace-on-action"})
	var second *Notification
	first.OnAction(func(NotificationActivation) {
		second = NewNotification(NotificationOptions{ID: first.id, Body: "new"})
		if err := second.Show(); err != nil {
			t.Error(err)
		}
	})
	onMain(func() { fb.ActivateNotification(first.id, platform.NotificationActivation{}) })
	if second == nil {
		t.Fatal("action not invoked")
	}
	t.Cleanup(second.Close)
	clicked := 0
	second.OnClick(func() { clicked++ })
	clickNotification(t, second)
	if clicked != 1 {
		t.Fatal("activation removed replacement listeners")
	}
}

func TestNotificationUserDismissReleasesListeners(t *testing.T) {
	n := showNotification(t, NotificationOptions{ID: "user-dismiss"})
	closed := false
	n.OnClose(func(reason NotificationCloseReason) { closed = reason == NotificationDismissed })
	onMain(func() { fb.DismissNotification(n.id, "dismissed") })
	notifications.Lock()
	remaining := notifications.byID[n.id]
	notifications.Unlock()
	if !closed || remaining != nil {
		t.Fatal("user dismissal retained notification listeners")
	}
}

func TestNotificationSynchronousAndLaterFailure(t *testing.T) {
	want := errors.New("native delivery failed")
	n := NewNotification(NotificationOptions{ID: "fail-on-show"})
	var got error
	n.OnFailed(func(err error) {
		if !isMainThread() {
			t.Error("failed not on main")
		}
		got = err
	})
	onMain(func() { fb.NotificationError = want })
	err := n.Show()
	onMain(func() { fb.NotificationError = nil })
	if !errors.Is(err, want) || !errors.Is(got, want) {
		t.Fatalf("Show=%v callback=%v", err, got)
	}
	n = showNotification(t, NotificationOptions{ID: "fail-later"})
	got = nil
	n.OnFailed(func(err error) { got = err })
	onMain(func() { fb.FailNotification(n.id, want) })
	if !errors.Is(got, want) || fb.Notification(n.id) != nil {
		t.Fatalf("later failure=%v", got)
	}
}
