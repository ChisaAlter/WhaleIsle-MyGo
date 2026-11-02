package pluginui

import "testing"

func TestSessionWireRejectsUnknownOutletsAndRequiresEpoch(t *testing.T) {
	base := Snapshot{Revision: 1, Subject: &Subject{Kind: "session", SessionID: "known", Epoch: "1:2"}, Entries: []Entry{{Slot: "conversation.session.header.actions"}}, Outlets: []SessionOutlet{{Handle: "1", Slot: "conversation.session.header.actions", Available: true}}}
	if !validSnapshot(base) {
		t.Fatal("real Session outlet rejected")
	}
	base.Subject.Epoch = ""
	if validSnapshot(base) {
		t.Fatal("unowned Session identity accepted")
	}
	base.Subject.Epoch = "1:2"
	base.Outlets[0].Slot = "arbitrary.eval"
	if validSnapshot(base) {
		t.Fatal("unknown slot accepted")
	}
}

func TestSessionControlsRequireCurrentObservedOutlet(t *testing.T) {
	rows := []SessionOutlet{{Handle: "entry", Slot: "conversation.session.header.actions", Available: true}}
	controls := []SessionControl{{Handle: "control", Outlet: "entry", Slot: "conversation.session.header.actions", Label: "Click"}}
	if !validSessionControls(controls, rows) {
		t.Fatal("current original control refused")
	}
	rows[0].Available = false
	if validSessionControls(controls, rows) {
		t.Fatal("withdrawn occurrence retained controls")
	}
	rows[0].Available = true
	controls[0].Slot = "conversation.session.header.utilities"
	if validSessionControls(controls, rows) {
		t.Fatal("control crossed header zone")
	}
}

func TestSessionControlCannotRetargetAnotherSelectionOrPage(t *testing.T) {
	subject := Subject{Kind: "session", SessionID: "a", Epoch: "2:3"}
	o := &Owner{selected: subject, state: Snapshot{Connected: true, Page: 2, Revision: 3, Subject: &subject, SessionControls: []SessionControl{{Handle: "control", Label: "Click"}}}}
	for _, args := range []struct {
		page, revision uint64
		subject        Subject
	}{{1, 3, subject}, {2, 2, subject}, {2, 3, Subject{Kind: "session", SessionID: "a", Epoch: "2:4"}}} {
		if o.ClickSessionControl(args.page, args.revision, args.subject, "control") == nil {
			t.Fatal("stale control accepted")
		}
	}
	o.Close()
	if o.ClickSessionControl(2, 3, subject, "control") == nil || o.ShowSessionPage(2, 3, subject) == nil {
		t.Fatal("closed capability retained authority")
	}
}

func TestSessionOutletCannotRetargetAnotherSelectionOrRevision(t *testing.T) {
	subject := Subject{Kind: "session", SessionID: "first", Epoch: "1:2"}
	o := &Owner{selected: subject, state: Snapshot{Connected: true, Page: 7, Revision: 9, Subject: &subject, Outlets: []SessionOutlet{{Handle: "seat", Slot: "conversation.session.header.actions", Available: true}}}}
	if err := o.ShowSessionOutlet(7, 8, subject, "seat"); err == nil {
		t.Fatal("old revision accepted")
	}
	other := subject
	other.Epoch = "1:3"
	if err := o.ShowSessionOutlet(7, 9, other, "seat"); err == nil {
		t.Fatal("same-id stale selection accepted")
	}
	o.Select(other)
	if len(o.Snapshot().Outlets) != 0 || o.Snapshot().Surface != "" {
		t.Fatal("selection retained prior DOM authority")
	}
}
