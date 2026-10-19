package composer

import "testing"

func TestLocalModelCommand(t *testing.T) {
	snap := nativeCommandSnapshot(t)
	d := NewDraftState(" /model ")
	if !d.HandleNativeCommand(snap, nil) || !d.ModelCommand.Visible("a") || d.Text != " /model " {
		t.Fatal("bare model did not open without consuming its command")
	}
	if d.ModelCommand.Visible("b") {
		t.Fatal("picker leaked across sessions")
	}
	ctx := d.ModelCommand.popup.ctx
	d.ModelCommand.Sync("b")
	select {
	case <-ctx.Done():
	default:
		t.Fatal("navigation did not retire popup work")
	}
	for _, text := range []string{"/model foo", "/model\nhello", "hello /model"} {
		d.Text = text
		if d.HandleNativeCommand(snap, nil) {
			t.Fatalf("claimed normal input %q", text)
		}
	}
	d.Text = "/model"
	d.EditSeq = 1
	if d.HandleNativeCommand(snap, nil) {
		t.Fatal("claimed edit")
	}
	d.EditSeq = 0
	d.TargetID = "other"
	if !d.HandleNativeCommand(snap, nil) || d.Error == "" || d.ModelCommand.Open || d.Text != "/model" {
		t.Fatal("wrong-session request escaped or lost draft")
	}
	if len(StubSlashCommands()) != 0 {
		t.Fatal("unavailable Host directory invented native commands")
	}
}
