package sidebar

import (
	"reflect"
	"strings"
	"testing"
	"time"

	"github.com/egoist/mygo/ui"
	"whaleisle/internal/store/session"
)

// Observe only the headless accessibility tree; no native provider is loaded.
func groupPresenceAccessLabels(tt *ui.Tester) string {
	tree := reflect.ValueOf(tt).Elem().FieldByName("h").Elem().FieldByName("access")
	if tree.IsNil() {
		return ""
	}
	nodes := tree.Elem().FieldByName("Nodes")
	var labels []string
	for i := 0; i < nodes.Len(); i++ {
		labels = append(labels, nodes.Index(i).FieldByName("Label").String())
	}
	return strings.Join(labels, "\n")
}

func TestGroupPresenceExitPaintRetainedButAccessibilityRetired(t *testing.T) {
	folded := false
	clicks := 0
	group := session.WorkspaceGroup{Key: "access-w", Kind: "workspace", Items: []session.Item{{SessionID: "a", Title: "Outgoing Alpha"}}, Workspace: session.WorkspaceRow{WorkspaceID: "access-w", SessionIDs: []string{"a"}}}
	model := Model{Snapshot: session.ViewState{WorkspaceLayout: session.WorkspaceLayoutState{HostGeneration: 42}}}
	tt := ui.NewTester(func(c *ui.Context) {
		host := ui.Column(c.Key("access-host")).Width(240)
		host.Children(func() {
			ui.Box(c).Height(34)
			workspaceGroupSessionRun(c, host, model, group, folded, func(rows workspaceWatermarkRows) {
				for _, row := range rows.Rows {
					button := ui.ButtonBase(c).Label(row.Item.Title).Height(32).Background(ui.RGB(230, 40, 50))
					if button.Clicked() {
						clicks++
					}
				}
			})
			ui.Text(c, "Following visible sibling")
		})
	}, 300, 250)
	// SetValue enables the in-memory accessibility observer even when the field
	// is absent; the expected lookup error does not perform any action.
	_ = tt.SetValue("absent accessibility bootstrap field", "")
	tt.SetPreferences(ui.Preferences{ReduceMotion: false, TextScale: 1})
	now := time.Now()
	now = now.Add(250 * time.Millisecond)
	tt.SetTime(now)
	row, ok := tt.Find("Outgoing Alpha")
	if !ok || !strings.Contains(groupPresenceAccessLabels(tt), "Outgoing Alpha") {
		t.Fatal("open row missing from hit/access tree")
	}
	x, y := int(row.X+2), int(row.Y+row.H/2)
	before := tt.Image().RGBAAt(x, y)
	folded = true
	tt.Frame()
	if tt.HasText("Outgoing Alpha") || strings.Contains(groupPresenceAccessLabels(tt), "Outgoing Alpha") {
		t.Fatal("logically folded outgoing row remains in hit/access tree")
	}
	if !strings.Contains(groupPresenceAccessLabels(tt), "Following visible sibling") {
		t.Fatal("fold hid an unrelated accessible sibling")
	}
	// Disabled exit content uses the theme's disabled opacity. Its red paint
	// must still be present, while inert removes only semantic/input authority.
	if after := tt.Image().RGBAAt(x, y); before.R != 230 || after.R <= after.G+40 || after.R <= after.B+40 || after.A != 255 {
		t.Fatalf("outgoing colored paint disappeared: before=%v after=%v", before, after)
	}
	tt.ClickAt(float32(x), float32(y))
	tt.Key(0, ui.KeyEnter)
	if clicks != 0 {
		t.Fatal("outgoing row retained action authority")
	}
	tt.SetTime(now.Add(210 * time.Millisecond))
	folded = false
	tt.Frame()
	tt.SetTime(now.Add(430 * time.Millisecond))
	if !strings.Contains(groupPresenceAccessLabels(tt), "Outgoing Alpha") {
		t.Fatal("reopened row did not rejoin accessibility tree")
	}
	if err := tt.Click("Outgoing Alpha"); err != nil {
		t.Fatal(err)
	}
	if clicks != 1 {
		t.Fatal("reopened row did not regain action authority")
	}
}
