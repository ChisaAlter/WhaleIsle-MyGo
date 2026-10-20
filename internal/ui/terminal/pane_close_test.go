package terminal

import (
	"testing"

	"github.com/egoist/mygo/ui"

	"whaleisle/internal/ui/i18n"
)

func stubRunningPane(t *testing.T, running bool) {
	t.Helper()
	previous := runningPane
	runningPane = func(*terminalPane) bool { return running }
	t.Cleanup(func() { runningPane = previous })
}

func closablePanel() *Panel {
	return &Panel{
		cwd:    "workspace",
		panes:  []*terminalPane{{id: 1}, {id: 2}},
		groups: []*terminalGroup{{id: 1, panes: []uint64{1, 2}}},
		active: 1,
	}
}

// TestRequestCloseAsksForARunningPane keeps a live process safe from one stray
// click: the close is parked until the prompt is answered.
func TestRequestCloseAsksForARunningPane(t *testing.T) {
	stubRunningPane(t, true)
	p := closablePanel()
	requestClosePane(Model{Panel: p}, 1)
	if p.pendingClose != 1 {
		t.Fatalf("running pane closed without asking: pending=%d", p.pendingClose)
	}
	if p.pane(1) == nil {
		t.Fatal("pane disappeared before the prompt was answered")
	}
}

// TestRequestCloseClosesAFinishedPane keeps the common case quick: a pane whose
// process is gone closes straight away.
func TestRequestCloseClosesAFinishedPane(t *testing.T) {
	stubRunningPane(t, false)
	p := closablePanel()
	requestClosePane(Model{Panel: p}, 1)
	if p.pendingClose != 0 {
		t.Fatalf("finished pane was parked for confirmation: pending=%d", p.pendingClose)
	}
	if p.pane(1) != nil {
		t.Fatal("finished pane was not closed")
	}
}

// TestClosePromptCancelKeepsThePane covers the escape hatch on the prompt.
func TestClosePromptCancelKeepsThePane(t *testing.T) {
	old := i18n.Locale()
	i18n.SetLocale(i18n.LocaleEN)
	defer i18n.SetLocale(old)
	stubRunningPane(t, true)
	p := closablePanel()
	p.pendingClose = 1
	m := Model{Panel: p}
	tt := ui.NewTester(func(c *ui.Context) {
		ui.Column(c).Fill().Children(func() {
			terminalClosePrompt(c, m, ui.RGB(24, 24, 24), ui.RGB(120, 120, 120))
		})
	}, 720, 320)
	prompt := i18n.T(i18n.KeySurfacesTerminalClosePrompt)
	if !tt.HasText(prompt) {
		t.Fatalf("prompt did not render: %v", tt.Texts())
	}
	if err := tt.Click(i18n.T(i18n.KeySurfacesTerminalCancel)); err != nil {
		t.Fatalf("cancel is not reachable: %v", err)
	}
	if p.pendingClose != 0 {
		t.Fatal("cancel left the prompt open")
	}
	if p.pane(1) == nil {
		t.Fatal("cancel closed the pane anyway")
	}
}

// TestClosePromptConfirmEndsThePane covers the confirmed path.
func TestClosePromptConfirmEndsThePane(t *testing.T) {
	old := i18n.Locale()
	i18n.SetLocale(i18n.LocaleEN)
	defer i18n.SetLocale(old)
	stubRunningPane(t, true)
	p := closablePanel()
	p.pendingClose = 1
	m := Model{Panel: p}
	tt := ui.NewTester(func(c *ui.Context) {
		ui.Column(c).Fill().Children(func() {
			terminalClosePrompt(c, m, ui.RGB(24, 24, 24), ui.RGB(120, 120, 120))
		})
	}, 720, 320)
	if err := tt.Click(i18n.T(i18n.KeySurfacesTerminalConfirmClose)); err != nil {
		t.Fatalf("confirm is not reachable: %v", err)
	}
	if p.pane(1) != nil {
		t.Fatal("confirm did not close the pane")
	}
	if p.pendingClose != 0 {
		t.Fatal("confirm left the prompt state behind")
	}
}
