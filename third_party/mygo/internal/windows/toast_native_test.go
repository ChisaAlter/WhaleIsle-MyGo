//go:build windows && (amd64 || arm64)

package windows

import (
	"crypto/rand"
	"os"
	"runtime"
	"strings"
	"testing"
	"time"

	"github.com/egoist/mygo/internal/platform"
)

// The fixture uses a unique registry identity and suppresses its popup. It never
// registers the product ID, edits the user's links, or clears an app's history.
func TestToastWinRTNativeHistoryLifecycle(t *testing.T) {
	if os.Getenv("MYGO_TOAST_NATIVE_TEST") != "1" {
		t.Skip("set MYGO_TOAST_NATIVE_TEST=1 in a Windows desktop session")
	}
	runtime.LockOSThread()
	defer runtime.UnlockOSThread()
	oldThread, oldBackend := mainThreadID, theBackend
	mainThreadID = currentThreadID()
	defer func() { mainThreadID = oldThread; theBackend = oldBackend }()
	b := New()
	if err := b.Init(previewNativeApp{}, platform.AppOptions{Name: "MyGo toast isolated fixture"}); err != nil {
		t.Fatal(err)
	}
	defer procDestroyWindow.Call(b.appHwnd)
	defer func() {
		if b.toast != nil {
			b.toast.close()
		}
	}()
	appID := "MyGo.ToastFixture." + rand.Text()
	key := `Software\Classes\AppUserModelId\` + appID
	if _, exists := regGet(hkeyCurrentUser, key, "DisplayName"); exists {
		t.Fatal("fixture identity already exists")
	}
	if err := regSet(hkeyCurrentUser, key, "DisplayName", "MyGo isolated toast fixture"); err != nil {
		t.Fatal(err)
	}
	defer func() {
		if err := regDeleteTree(hkeyCurrentUser, key); err != nil {
			t.Error(err)
		}
	}()
	registerToastFixture(t, appID)
	if err := regSet(hkeyCurrentUser, key, "ShowInSettings", "0"); err != nil {
		t.Fatal(err)
	}
	n := &platform.Notification{ID: "隔离:history-" + rand.Text(), Title: "MyGo isolated toast", Subtitle: "Native WinRT fixture", Body: "Template and history lifecycle", Silent: true, Group: "fixture-only", Toast: &platform.ToastOptions{AppID: appID, SuppressPopup: true, Actions: []platform.ToastAction{{Label: "Open", Arguments: "fixture-open"}}}}
	if err := b.showNotification(n); err != nil {
		t.Fatal(err)
	}
	defer func() { b.RemoveNotification(n.ID) }()
	find := func() bool {
		rows, err := b.NotificationHistory(appID)
		if err != nil {
			t.Fatal(err)
		}
		for _, row := range rows {
			if row.ID == n.ID {
				if row.Tag != toastKey(n.ID) || row.Group != toastKey(n.Group) || !strings.Contains(row.XML, "fixture-open") {
					t.Fatalf("history payload=%+v", row)
				}
				return true
			}
		}
		return false
	}
	deadline := time.Now().Add(10 * time.Second)
	for !find() {
		if time.Now().After(deadline) {
			t.Fatal("WinRT Show succeeded but fixture did not enter notification history")
		}
		b.Step()
	}
	// Replacing a delivered ID is handled by Windows Tag+Group, not a tray.
	n.Body = "replacement"
	if err := b.showNotification(n); err != nil {
		t.Fatal(err)
	}
	rows, err := b.NotificationHistory(appID)
	if err != nil {
		t.Fatal(err)
	}
	count := 0
	for _, row := range rows {
		if row.ID == n.ID {
			count++
		}
	}
	if count != 1 {
		t.Fatalf("replacement left %d records", count)
	}
	b.RemoveNotification(n.ID)
	if find() {
		t.Fatal("Close left its own toast in notification history")
	}
	if b.toast.active[n.ID] != nil {
		t.Fatal("Close retained native record")
	}
	toastDelegatesMu.Lock()
	left := len(toastDelegates)
	toastDelegatesMu.Unlock()
	if left != 0 {
		t.Fatalf("Close retained %d COM delegates", left)
	}
	t.Log("native WinRT XML, notifier, tagged replacement, read-only history and targeted removal passed")
}

func TestToastTemplateEscapesAndRetainsFullText(t *testing.T) {
	id := "会话<>&\"/long:" + strings.Repeat("字", 80)
	n := &platform.Notification{ID: id, Title: "<标题&>", Body: strings.Repeat("全", 400), Silent: true, Toast: &platform.ToastOptions{Actions: []platform.ToastAction{{Label: "<打开>", Arguments: "a&b"}}}}
	x := toastXML(n)
	if strings.Contains(x, "<标题") || !strings.Contains(x, "&lt;标题&amp;&gt;") || !strings.Contains(x, strings.Repeat("全", 400)) {
		t.Fatal(x)
	}
	if toastIDFromXML(x) != id {
		t.Fatal("ID was truncated or not escaped")
	}
	if len(toastKey(id)) != 16 || toastKey(id) == toastKey(id+"x") {
		t.Fatal("invalid stable history key")
	}
}
