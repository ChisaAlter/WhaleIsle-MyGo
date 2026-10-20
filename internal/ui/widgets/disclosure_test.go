package widgets

import (
	"testing"
	"time"

	"github.com/egoist/mygo/ui"
)

func TestDisclosureRowIsOneTwentyFourDIPLine(t *testing.T) {
	open := false
	tt, _ := newTester(t, func(c *ui.Context) {
		ui.Column(c).Fill().Children(func() {
			DisclosureRow(c, DisclosureRowOptions{
				Title:   "Thinking",
				Summary: "about the shape of the answer",
				Open:    &open,
				Body:    func() { ui.Text(c, "reasoning") },
			})
		})
	}, 400, 200)
	r, ok := tt.Find("Thinking")
	if !ok {
		t.Fatal("the row is not findable by its title")
	}
	if r.H != 24 {
		t.Fatalf("the line is %v DIPs high, want 24", r.H)
	}
	if r.Y != 0 {
		t.Fatalf("the line starts at %v", r.Y)
	}
}

func TestDisclosureRowOpensWithClickAndKeys(t *testing.T) {
	open := false
	tt, _ := newTester(t, func(c *ui.Context) {
		ui.Column(c).Fill().Children(func() {
			DisclosureRow(c, DisclosureRowOptions{
				Title:   "Thinking",
				Summary: "a summary",
				Open:    &open,
				Body: func() {
					ui.Text(c, "reasoning body")
				},
			})
		})
	}, 400, 200)
	start := time.Now().Add(time.Hour)
	tt.SetTime(start)
	if tt.HasText("reasoning body") {
		t.Fatal("the body shows while the row is closed")
	}
	if err := tt.Click("Thinking"); err != nil {
		t.Fatal(err)
	}
	if !open || !tt.HasText("reasoning body") {
		t.Fatalf("the click did not open the row: open=%v", open)
	}
	if tt.HasText("a summary") {
		t.Fatal("the summary stays beside the open row")
	}
	if !tt.Focused("Thinking") {
		t.Fatal("the click did not take the keyboard focus")
	}
	tt.Key(0, ui.KeySpace)
	if open {
		t.Fatal("Space did not close the row")
	}
	// The body leaves as the row shuts.
	tt.SetTime(start.Add(500 * time.Millisecond))
	if tt.HasText("reasoning body") {
		t.Fatal("the body stays after the row closed")
	}
	tt.Key(0, ui.KeyEnter)
	if !open {
		t.Fatal("Enter did not open the row")
	}
}

func TestDisclosureRowOpensFromAnyPointOfTheLine(t *testing.T) {
	open := false
	tt, _ := newTester(t, func(c *ui.Context) {
		ui.Column(c).Fill().Children(func() {
			DisclosureRow(c, DisclosureRowOptions{Title: "Read", Summary: "internal/ui/frame.go", Open: &open})
		})
	}, 400, 200)
	// Far right of the line, past the summary.
	tt.ClickAt(390, 12)
	if !open {
		t.Fatal("a click at the end of the line did not open it")
	}
}

func TestDisclosureRowReportsToggles(t *testing.T) {
	open := false
	var told []bool
	tt, _ := newTester(t, func(c *ui.Context) {
		ui.Column(c).Fill().Children(func() {
			DisclosureRow(c, DisclosureRowOptions{
				Title:    "Bash",
				Open:     &open,
				OnToggle: func(open bool) { told = append(told, open) },
			})
		})
	}, 400, 200)
	if err := tt.Click("Bash"); err != nil {
		t.Fatal(err)
	}
	if len(told) != 1 || !told[0] {
		t.Fatalf("told %v, want one opening", told)
	}
}

func TestDisclosureRowIndentsItsBody(t *testing.T) {
	open := true
	tt, _ := newTester(t, func(c *ui.Context) {
		ui.Column(c).Fill().Children(func() {
			DisclosureRow(c, DisclosureRowOptions{
				Title: "Thinking",
				Open:  &open,
				Body:  func() { ui.Text(c, "reasoning body") },
			})
		})
	}, 400, 200)
	body, ok := tt.Find("reasoning body")
	if !ok {
		t.Fatal("the open body is missing")
	}
	if body.X < 20 || body.X > 24 {
		t.Fatalf("the body starts at x=%v, want 22 under the title", body.X)
	}
	if body.Y <= 24 {
		t.Fatalf("the body sits at y=%v, want it below the line", body.Y)
	}
}

func TestDisclosureRowKeepsSummaryWhenAsked(t *testing.T) {
	open := true
	tt, _ := newTester(t, func(c *ui.Context) {
		ui.Column(c).Fill().Children(func() {
			DisclosureRow(c, DisclosureRowOptions{
				Title:               "Read",
				Summary:             "internal/ui/widgets/disclosure.go",
				Open:                &open,
				KeepSummaryWhenOpen: true,
				Body:                func() { ui.Text(c, "file") },
			})
		})
	}, 400, 200)
	if !tt.HasText("internal/ui/widgets/disclosure.go") {
		t.Fatal("the tool row lost its summary while open")
	}
}

func TestDisclosureRowTakesAFillWhileHovered(t *testing.T) {
	open := false
	tt, _ := newTester(t, func(c *ui.Context) {
		ui.Column(c).Fill().Children(func() {
			DisclosureRow(c, DisclosureRowOptions{Title: "Thinking", Summary: "a summary", Open: &open})
		})
	}, 400, 200)
	tt.Move(200, 150)
	before := capture(tt.Image())
	tt.Move(200, 12)
	after := capture(tt.Image())
	if sameFrame(before, after) {
		t.Fatal("hovering the line changes nothing")
	}
}

func TestDisclosureRowKeepsTrailingCountsWhole(t *testing.T) {
	open := false
	tt, _ := newTester(t, func(c *ui.Context) {
		ui.Column(c).Fill().Children(func() {
			DisclosureRow(c, DisclosureRowOptions{
				Title:    "Edit",
				Summary:  "internal/ui/widgets/disclosure.go and a good deal more",
				Open:     &open,
				Trailing: func() { ui.Text(c, "+3 -1") },
			})
		})
	}, 300, 200)
	trailing, ok := tt.Find("+3 -1")
	if !ok {
		t.Fatal("the trailing counts are missing")
	}
	if trailing.X < 150 || trailing.X+trailing.W > 300 {
		t.Fatalf("the counts sit at x=%v, want the end of the line and inside it", trailing.X)
	}
}

func TestDisclosureRowRunsWithoutState(t *testing.T) {
	tt, _ := newTester(t, func(c *ui.Context) {
		ui.Column(c).Fill().Children(func() {
			DisclosureRow(c, DisclosureRowOptions{Title: "Ran", Summary: "12 files"})
		})
	}, 400, 200)
	r, ok := tt.Find("Ran")
	if !ok {
		t.Fatal("the plain line is missing")
	}
	if r.H != 24 {
		t.Fatalf("the plain line is %v DIPs high", r.H)
	}
	// A line without a state takes no clicks and shows no body.
	tt.ClickAt(200, 12)
	if !tt.HasText("12 files") {
		t.Fatal("the plain line lost its summary")
	}
}
