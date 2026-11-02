package sidebar

import (
	"strings"
	"testing"
	"time"

	"github.com/egoist/mygo/ui"
)

// Real production hover-card renderers in the memory host. Pointer input and
// clipboard operations below are exclusively ui.Tester events/state; these
// tests neither open native windows nor interact with the system clipboard.
func TestHoverSelectionInteractionDragRangeThenKeyboardWholeCopy(t *testing.T) {
	for _, family := range []string{"session", "workspace"} {
		for _, reverse := range []bool{false, true} {
			for _, key := range []ui.Key{ui.KeyEnter, ui.KeySpace} {
				direction := "forward"
				if reverse {
					direction = "reverse"
				}
				keyName := "enter"
				if key == ui.KeySpace {
					keyName = "space"
				}
				t.Run(family+"/"+direction+"/"+keyName, func(t *testing.T) {
					var tt *ui.Tester
					var start time.Time
					var whole, selectedParagraph, panelLabel, originalSelected string
					var mutations func() int
					var currentSelected func() string
					if family == "session" {
						sessionTester, m, in, clock := sessionHoverDraftFixture(t)
						tt, start = sessionTester, clock
						m.Snapshot.Items[0].Title = "  Alpha beta gamma delta epsilon zeta  "
						whole, selectedParagraph = m.Snapshot.Items[0].Title, m.Snapshot.Items[0].Title
						panelLabel, originalSelected = "Copy session title a", m.Snapshot.SelectedID
						mutations = func() int { return in.mutations }
						currentSelected = func() string { return m.Snapshot.SelectedID }
						tt.SetScale(1.5)
						tt.SetPreferences(ui.Preferences{ReduceMotion: false, TextScale: 1})
						tt.Frame()
						hoverDraftRow(t, tt, "a")
					} else {
						workspaceTester, m, clock := workspaceHoverFixture(t)
						tt, start = workspaceTester, clock
						m.Snapshot.Workspaces[0].Title = "Alpha beta gamma delta epsilon zeta"
						m.Snapshot.Workspaces[0].Path = "  /remote/exact/path/with outer spaces  "
						whole, selectedParagraph = m.Snapshot.Workspaces[0].Path, m.Snapshot.Workspaces[0].Title
						panelLabel, originalSelected = "Copy workspace path a", m.Snapshot.SelectedID
						calls := 0
						m.OnToggleWorkspaceGroup = func(string) { calls++ }
						m.OnSelect = func(string) { calls++ }
						mutations = func() int { return calls }
						currentSelected = func() string { return m.Snapshot.SelectedID }
						tt.SetScale(1.5)
						tt.SetPreferences(ui.Preferences{ReduceMotion: false, TextScale: 1})
						tt.Frame()
						moveWorkspaceTitle(t, tt, m.Snapshot.Workspaces[0].Title)
					}
					tt.SetTime(start.Add(800 * time.Millisecond))
					panel, ok := tt.Find(panelLabel)
					if !ok {
						t.Fatalf("real card missing: %q", tt.Texts())
					}
					// First visible title line: card's16DIP inline/12DIP top pad,
					// then drag between two glyph positions, never across controls.
					x1, x2, y := panel.X+25, panel.X+100, panel.Y+22
					if reverse {
						x1, x2 = x2, x1
					}
					tt.Move(x1, y)
					tt.SetClipboard("memory selection sentinel")
					tt.Press(x1, y)
					tt.Move(x2, y)
					tt.Release(x2, y)
					tt.Frame()
					if tt.Clipboard() != "memory selection sentinel" || tt.HasText("Copied") {
						t.Fatalf("paragraph drag activated whole-card copy: %q copied=%v", tt.Clipboard(), tt.HasText("Copied"))
					}
					// Native-independent evidence of a real nonempty selection:
					// Cmd+C copies only the selected title substring into memory.
					tt.Key(ui.Cmd, ui.KeyC)
					selection := tt.Clipboard()
					if selection == "" || selection == "memory selection sentinel" || selection == whole || selection == selectedParagraph || !strings.Contains(selectedParagraph, selection) {
						t.Fatalf("drag did not create a proper title range: selected=%q paragraph=%q whole=%q", selection, selectedParagraph, whole)
					}
					if tt.HasText("Copied") {
						t.Fatal("range copy triggered whole-card feedback")
					}
					tt.SetClipboard("keyboard memory guard")
					tt.Key(0, key)
					tt.Frame()
					if tt.Clipboard() != whole || !tt.HasText("Copied") {
						t.Fatalf("keyboard with paragraph focus failed exact whole copy: %q want=%q texts=%q", tt.Clipboard(), whole, tt.Texts())
					}
					feedback, ok := tt.Find(panelLabel)
					if !ok || feedback != panel {
						t.Fatalf("copy feedback moved/resized card: before=%+v after=%+v", panel, feedback)
					}
					tt.SetTime(start.Add(1799 * time.Millisecond))
					if !tt.HasText("Copied") {
						t.Fatal("keyboard feedback ended before one second")
					}
					tt.SetTime(start.Add(1800 * time.Millisecond))
					if tt.HasText("Copied") || !tt.HasText(selectedParagraph) {
						t.Fatal("keyboard feedback did not restore card after one second")
					}
					if mutations() != 0 || currentSelected() != originalSelected {
						t.Fatalf("selection/copy mutated row or selected session: mutations=%d selected=%q", mutations(), currentSelected())
					}
				})
			}
		}
	}
}
