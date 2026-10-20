package widgets

import (
	"testing"

	"github.com/egoist/mygo/ui"
)

// confirmTest opens a dialog over a plain window and counts confirmations.
func confirmTest(t *testing.T, o ConfirmDialogOptions, hits *int) (*ui.Tester, *ui.Theme) {
	t.Helper()
	return newTester(t, func(c *ui.Context) {
		ui.Column(c).Fill().Children(func() {
			ui.Text(c, "Behind the dialog")
		})
		if ConfirmDialog(c, o) {
			*hits++
		}
	}, 800, 600)
}

func TestConfirmDialogAsks(t *testing.T) {
	open := true
	hits := 0
	tt, _ := confirmTest(t, ConfirmDialogOptions{
		Open:         &open,
		Title:        "Delete three files?",
		Message:      "This cannot be undone.",
		ConfirmLabel: "Delete",
		CancelLabel:  "Keep",
		Danger:       true,
	}, &hits)
	if !tt.HasText("Delete three files?") || !tt.HasText("This cannot be undone.") {
		t.Fatalf("the dialog shows %q", tt.Texts())
	}
	if !tt.HasText("Delete") || !tt.HasText("Keep") {
		t.Fatal("the dialog lost its buttons")
	}
	if err := tt.Click("Delete"); err != nil {
		t.Fatal(err)
	}
	if hits != 1 {
		t.Fatalf("confirmed %d times, want 1", hits)
	}
	if open {
		t.Fatal("the dialog stays open after confirming")
	}
}

func TestConfirmDialogBacksOut(t *testing.T) {
	for _, tc := range []struct {
		name  string
		close func(*ui.Tester)
	}{
		{"cancel", func(tt *ui.Tester) {
			if err := tt.Click("Keep"); err != nil {
				t.Error(err)
			}
		}},
		{"escape", func(tt *ui.Tester) { tt.Key(0, ui.KeyEscape) }},
		{"backdrop", func(tt *ui.Tester) { tt.ClickAt(8, 592) }},
	} {
		open := true
		hits := 0
		tt, _ := confirmTest(t, ConfirmDialogOptions{
			Open:         &open,
			Title:        "Delete three files?",
			Message:      "This cannot be undone.",
			ConfirmLabel: "Delete",
			CancelLabel:  "Keep",
			Danger:       true,
		}, &hits)
		tc.close(tt)
		if hits != 0 {
			t.Errorf("%s: confirmed %d times", tc.name, hits)
		}
		if open {
			t.Errorf("%s: the dialog stays open", tc.name)
		}
	}
}

func TestConfirmDialogPaintsDangerOnTheConfirmButton(t *testing.T) {
	open := true
	hits := 0
	tt, theme := confirmTest(t, ConfirmDialogOptions{
		Open:         &open,
		Title:        "Delete three files?",
		ConfirmLabel: "Delete",
		CancelLabel:  "Keep",
		Danger:       true,
	}, &hits)
	button, ok := tt.Find("Delete")
	if !ok {
		t.Fatal("the confirm button is missing")
	}
	if got := at(tt.Image(), int(button.X)+6, int(button.Y+button.H/2)); !near(got, rgb(theme.Danger), 2) {
		t.Fatalf("the confirm button is %v, want the danger color %v", got, rgb(theme.Danger))
	}
	// The dialog's own panel sits over the dimmed window, not over nothing.
	panel, _ := tt.Find("Delete three files?")
	if got := at(tt.Image(), int(panel.X)+6, int(panel.Y+panel.H/2)); !near(got, rgb(theme.Background), 2) {
		t.Fatalf("the panel gutters are %v, want %v", got, rgb(theme.Background))
	}
}

func TestConfirmDialogAccentWithoutDanger(t *testing.T) {
	open := true
	hits := 0
	tt, theme := confirmTest(t, ConfirmDialogOptions{
		Open:         &open,
		Title:        "Apply the theme?",
		ConfirmLabel: "Apply",
		CancelLabel:  "Cancel",
	}, &hits)
	button, ok := tt.Find("Apply")
	if !ok {
		t.Fatal("the confirm button is missing")
	}
	if got := at(tt.Image(), int(button.X)+6, int(button.Y+button.H/2)); !near(got, rgb(theme.Accent), 2) {
		t.Fatalf("the confirm button is %v, want the accent %v", got, rgb(theme.Accent))
	}
}

func TestConfirmDialogStartsOnTheSafeButton(t *testing.T) {
	open := true
	hits := 0
	tt, _ := confirmTest(t, ConfirmDialogOptions{
		Open:         &open,
		Title:        "Delete three files?",
		ConfirmLabel: "Delete",
		CancelLabel:  "Keep",
		Danger:       true,
	}, &hits)
	if !tt.Focused("Keep") {
		t.Fatal("the keyboard does not start on the button that backs out")
	}
	tt.Key(0, ui.KeyTab)
	if !tt.Focused("Delete") {
		t.Fatal("Tab did not reach the confirm button")
	}
	tt.Frame()
	if !tt.Focused("Delete") {
		t.Fatal("the dialog took the focus back to the safe button")
	}
}

func TestConfirmDialogWithoutLabelsShowsNoButtons(t *testing.T) {
	open := true
	hits := 0
	tt, _ := confirmTest(t, ConfirmDialogOptions{
		Open:  &open,
		Title: "Nothing to choose",
	}, &hits)
	if tt.HasText("Delete") {
		t.Fatal("the dialog invented a button")
	}
	tt.Key(0, ui.KeyEscape)
	if open {
		t.Fatal("Escape did not dismiss the dialog without buttons")
	}
}
