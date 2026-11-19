package widgets

import (
	"testing"

	"github.com/egoist/mygo/ui"
)

// A failure is drawn where it happened: the alert is a row in the page, so the
// text is in the view tree, and no toast is added for it.
func TestAlertIsWrittenInThePage(t *testing.T) {
	tt := ui.NewTester(func(c *ui.Context) {
		ui.Column(c).Fill().Children(func() {
			ui.Text(c, "page content")
			Alert(c, AlertOptions{ID: "fixture", Message: "The save did not finish. Try again."})
		})
	}, 640, 480)
	if !tt.HasText("The save did not finish. Try again.") {
		t.Fatalf("the alert is not in the page: %v", tt.Texts())
	}
	page, ok := tt.Find("page content")
	if !ok {
		t.Fatal("page content missing")
	}
	alert, ok := tt.Find("The save did not finish. Try again.")
	if !ok || alert.Y <= page.Y {
		t.Fatalf("the alert is not the row under the content it reports on: alert=%+v page=%+v", alert, page)
	}
	if announced := tt.Announcements(); len(announced) != 0 {
		t.Fatalf("the alert went through the toast announcement path: %v", announced)
	}
}

// An empty message draws nothing, so a caller clears the failure by not having
// one rather than by closing a surface.
func TestAlertDrawsNothingWithoutAMessage(t *testing.T) {
	tt := ui.NewTester(func(c *ui.Context) {
		ui.Column(c).Fill().Children(func() {
			ui.Text(c, "page content")
			Alert(c, AlertOptions{ID: "fixture", Message: ""})
		})
	}, 640, 480)
	if len(tt.Texts()) != 1 {
		t.Fatalf("an empty alert drew something: %v", tt.Texts())
	}
	if announced := tt.Announcements(); len(announced) != 0 {
		t.Fatalf("an empty alert announced something: %v", announced)
	}
}

// The raw failure stays behind 详情: the sentence is what a person reads, and the
// transport's own words are one click away.
func TestAlertKeepsTheRawFailureBehindDetails(t *testing.T) {
	raw := "Post \"http://127.0.0.1:1/api\": dial tcp 127.0.0.1:1: connectex: No connection could be made"
	tt := ui.NewTester(func(c *ui.Context) {
		ui.Column(c).Fill().Children(func() {
			Alert(c, AlertOptions{ID: "fixture", Message: "The operation did not finish. Try again.", Detail: raw})
		})
	}, 640, 480)
	if tt.HasText(raw) {
		t.Fatal("the raw failure reached the sentence")
	}
	details := "Details"
	if tt.HasText("详情") {
		details = "详情"
	}
	if err := tt.Click(details); err != nil {
		t.Fatal(err)
	}
	if !tt.HasText(raw) {
		t.Fatalf("the disclosure lost the raw failure: %v", tt.Texts())
	}
}

// The action runs the retry the caller owns, and the dismiss control clears what
// the caller already reported.
func TestAlertRunsItsOwnControls(t *testing.T) {
	retried, dismissed := 0, 0
	tt := ui.NewTester(func(c *ui.Context) {
		ui.Column(c).Fill().Children(func() {
			Alert(c, AlertOptions{ID: "fixture", Message: "Could not open. Try again.",
				Action: "Retry", OnAction: func() { retried++ },
				Dismiss: "Dismiss", OnDismiss: func() { dismissed++ }})
		})
	}, 640, 480)
	if err := tt.Click("Retry"); err != nil {
		t.Fatal(err)
	}
	if err := tt.Click("Dismiss"); err != nil {
		t.Fatal(err)
	}
	if retried != 1 || dismissed != 1 {
		t.Fatalf("alert controls: retried=%d dismissed=%d", retried, dismissed)
	}
	if announced := tt.Announcements(); len(announced) != 0 {
		t.Fatalf("alert controls announced through the toast path: %v", announced)
	}
}
