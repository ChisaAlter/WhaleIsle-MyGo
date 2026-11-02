package chat

import (
	"encoding/json"
	"fmt"
	"image"
	"image/png"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/egoist/mygo/ui"
	"whaleisle/internal/store/session"
	"whaleisle/internal/ui/chrome"
	"whaleisle/internal/ui/i18n"
	"whaleisle/internal/ui/theme"
)

func terminalViewFixture(t *testing.T, m *toolTerminalModel, width int, dark bool) *ui.Tester {
	t.Helper()
	previous := i18n.Locale()
	i18n.SetLocale(i18n.LocaleEN)
	t.Cleanup(func() { i18n.SetLocale(previous) })
	tt := ui.NewTester(func(c *ui.Context) {
		theme.ApplyInterface(c)
		colors := theme.Resolve(theme.DefaultID).Active(dark)
		ui.Column(c).Fill().Padding(20).Background(chrome.Color(colors.BG)).Children(func() {
			terminalCard(c, toolCardStyle(colors, chrome.Color(colors.FG), chrome.Color(colors.Muted), chrome.Color(colors.Border)), "fixture", m, "", false)
			ui.Box(c).Height(1).Label("following terminal")
		})
	}, width, 700)
	tt.SetTime(time.Unix(9300, 0))
	return tt
}

func TestTerminalCardGeometryAndVerbatimOutputClipboard(t *testing.T) {
	for _, dark := range []bool{false, true} {
		t.Run(fmt.Sprint(dark), func(t *testing.T) {
			m := &toolTerminalModel{Command: "printf hi", Description: "Print greeting", Output: "\x1b[31mhi\x1b[0m\n", HasOutput: true, HasExit: true}
			tt := terminalViewFixture(t, m, 640, dark)
			card, ok := tt.Find("Terminal output")
			if !ok {
				t.Fatal(tt.Texts())
			}
			toolTreeNear(t, "card x", card.X, 24)
			toolTreeNear(t, "card width", card.W, 596)
			toolTreeNear(t, "short card height", card.H, 81)
			command, _ := tt.Find("printf hi")
			prompt, _ := tt.Find("$")
			toolTreeNear(t, "prompt gutter", prompt.X-card.X, 31)
			toolTreeNear(t, "command gap", command.X-prompt.X-prompt.W, 8)
			toolTreeNear(t, "command baseline row", command.Y-card.Y, 10)
			toolTreeNear(t, "command line height", command.H, 18)
			if !tt.HasText("hi") || tt.HasText("[exit code:") {
				t.Fatal(tt.Texts())
			}
			if err := tt.Click("Copy"); err != nil {
				t.Fatal(err)
			}
			if tt.Clipboard() != m.Output {
				t.Fatalf("clipboard=%q want raw output %q", tt.Clipboard(), m.Output)
			}
			if !tt.HasText("Copied") {
				t.Fatal("missing one second feedback", tt.Texts())
			}
			tt.SetClipboard("guarded second activation")
			tt.Key(0, ui.KeyEnter)
			if tt.Clipboard() != "guarded second activation" {
				t.Fatal("feedback window allowed a second write")
			}
			tt.SetTime(time.Unix(9300, 0).Add(1100 * time.Millisecond))
			if !tt.HasText("Copy") || tt.HasText("Copied") {
				t.Fatal("feedback did not expire", tt.Texts())
			}
			tt.Key(0, ui.KeyEnter)
			if tt.Clipboard() != m.Output {
				t.Fatal("focused Enter did not copy raw output after feedback expired")
			}
			terminalExport(t, tt, fmt.Sprintf("short-dark-%v", dark))
		})
	}
}

func TestTerminalCardRunningBackgroundAndEmptySettlement(t *testing.T) {
	for _, tc := range []struct {
		name                string
		m                   toolTerminalModel
		wantHeight          float32
		empty, copy, status bool
	}{
		{"running", toolTerminalModel{Command: "sleep 5", Running: true}, 38, false, false, false},
		{"background", toolTerminalModel{Command: "serve", Background: true}, 38, false, false, false},
		{"empty", toolTerminalModel{Command: "true", HasOutput: true, HasExit: true}, 79, true, false, false},
		{"escape-only", toolTerminalModel{Command: "true", Output: "\x1b[0m\x1b]0;title\x07", HasOutput: true, HasExit: true}, 79, true, false, false},
		{"failure", toolTerminalModel{Command: "exit 7", Output: "failure\n", HasOutput: true, HasExit: true, ExitCode: 7}, 81, false, true, true},
		{"live-output", toolTerminalModel{Command: "watch", Output: "partial\n", HasOutput: true, Running: true}, 81, false, false, false},
	} {
		t.Run(tc.name, func(t *testing.T) {
			tt := terminalViewFixture(t, &tc.m, 640, false)
			card, _ := tt.Find("Terminal output")
			toolTreeNear(t, "card height", card.H, tc.wantHeight)
			if tt.HasText("No output") != tc.empty || tt.HasText("Copy") != tc.copy || tt.HasText("exit code 7") != tc.status {
				t.Fatal(tt.Texts())
			}
			if tc.name == "background" && tt.HasText("Done") {
				t.Fatal("background acknowledged completion", tt.Texts())
			}
			terminalExport(t, tt, tc.name)
		})
	}
}

func TestTerminalCommandBannerOutputScrollIndependentlyAndKeepCopyReachable(t *testing.T) {
	commands, outputs := []string{}, []string{}
	for n := 1; n <= 35; n++ {
		commands = append(commands, fmt.Sprintf("command-%02d", n))
		outputs = append(outputs, fmt.Sprintf("output-%02d", n))
	}
	outputs[0] += " " + strings.Repeat("wide-", 100)
	m := &toolTerminalModel{Command: strings.Join(commands, "\n") + "\n", Cwd: "C:\\fixture\\repo", Output: strings.Join(outputs, "\n") + "\n", HasOutput: true, HasExit: true}
	tt := terminalViewFixture(t, m, 640, false)
	banner, _ := tt.Find("Terminal command banner")
	output, _ := tt.Find(i18n.T(i18n.KeyChatToolOutput))
	copyBefore, _ := tt.Find("Copy")
	toolTreeNear(t, "header content-box cap", banner.H, 168)
	toolTreeNear(t, "output content-box cap", output.H, 248)
	toolTreeNear(t, "output own gutter", output.X, 55)
	firstCommand, _ := tt.Find("command-01")
	firstOutput, _ := tt.Find(strings.Join(outputs, "\n"))
	tt.Scroll(banner.X+100, banner.Y+banner.H/2, 0, 100)
	commandAfter, _ := tt.Find("command-01")
	outputAfter, _ := tt.Find(strings.Join(outputs, "\n"))
	copyAfter, _ := tt.Find("Copy")
	if commandAfter.Y >= firstCommand.Y || outputAfter.Y != firstOutput.Y {
		t.Fatalf("header wheel command:%v→%v output:%v→%v", firstCommand, commandAfter, firstOutput, outputAfter)
	}
	toolTreeNear(t, "sticky copy reaches top0", copyAfter.Y, banner.Y)
	if copyAfter.Y > copyBefore.Y {
		t.Fatal("copy scrolled out of reach")
	}
	imageBefore := image.NewRGBA(tt.Image().Bounds())
	copy(imageBefore.Pix, tt.Image().Pix)
	tt.Scroll(output.X+100, output.Y+output.H/2, 0, 100)
	commandStill, _ := tt.Find("command-01")
	// Find returns the clipped accessible box, which stays viewport-sized for a
	// long RichText. Compare the actually painted output rather than that box.
	changed := false
	imageAfter := tt.Image()
	for y := int(output.Y + 8); y < int(output.Y+output.H-8); y++ {
		for x := int(output.X + 4); x < int(output.X+150); x++ {
			if imageBefore.RGBAAt(x, y) != imageAfter.RGBAAt(x, y) {
				changed = true
			}
		}
	}
	if !changed || commandStill.Y != commandAfter.Y {
		t.Fatalf("output wheel paintedChange=%v command:%v→%v", changed, commandAfter, commandStill)
	}
	if err := tt.Click("Copy"); err != nil {
		t.Fatal(err)
	}
	if tt.Clipboard() != m.Output {
		t.Fatal("scrolled output copy dropped hidden rows")
	}
	terminalExport(t, tt, "independent-scroll")
}

func TestTerminalHeaderNeutralExitBadgeAndFallbackSelection(t *testing.T) {
	previous := i18n.Locale()
	i18n.SetLocale(i18n.LocaleEN)
	defer i18n.SetLocale(previous)
	call := fileToolFixture("pwsh", `{"command":"exit 7","description":"Check fixture"}`, nil, "failure detail\n[exit code: 7]")
	before := string(call.Result.Raw)
	expanded := map[string]bool{}
	tt := ui.NewTester(func(c *ui.Context) {
		colors := theme.Resolve(theme.DefaultID).Light
		ui.Column(c).Fill().Padding(20).Children(func() {
			ToolMessageView(c, session.Message{ID: "exit", Tool: call}, toolCardStyle(colors, chrome.Color(colors.FG), chrome.Color(colors.Muted), chrome.Color(colors.Border)), ToolImageGalleryOptions{ID: "exit", Expanded: &expanded})
		})
	}, 640, 500)
	tt.SetTime(time.Unix(9400, 0))
	if !tt.HasText("Check fixture") || !tt.HasText("exit code 7") || tt.HasText("failure detail") {
		t.Fatal(tt.Texts())
	}
	badge, _ := tt.Find("exit code 7")
	toolTreeNear(t, "neutral badge height", badge.H, 19)
	description, _ := tt.Find("Check fixture")
	toolTreeNear(t, "exit badge follows description", badge.X-description.X-description.W, 8)
	if err := tt.Click(toolRowTitle("pwsh")); err != nil {
		t.Fatal(err)
	}
	tt.SetTime(time.Unix(9400, 0).Add(250 * time.Millisecond))
	if !tt.HasText("failure detail") || tt.HasText("[exit code:") {
		t.Fatal(tt.Texts())
	}
	if call.State != "ok" || string(call.Result.Raw) != before {
		t.Fatal("display exit state mutated source lifecycle")
	}
	terminalExport(t, tt, "exit-badge")
	// A typed generic error declines terminal parsing and retains error evidence.
	call.State = "error"
	tt.Frame()
	if tt.HasText("Terminal output") || !tt.HasText("IN") || !tt.HasText("OUT") {
		t.Fatal("error did not use generic IO", tt.Texts())
	}
	call.State = "ok"
	call.Name = "shell"
	tt.Frame()
	if tt.HasText("Terminal output") || !tt.HasText("IN") {
		t.Fatal("unknown alias selected terminal", tt.Texts())
	}
}

func TestTerminalPromptAndSendLocalization(t *testing.T) {
	for _, tc := range []struct{ cwd, home, want string }{{"", "", "$"}, {"/", "", "/"}, {"C:\\repo\\", "", "repo"}, {"/home/test/", "/home/test", "~"}, {"/home/test/sub", "/home/test", "sub"}} {
		if got := terminalPrompt(tc.cwd, tc.home); got != tc.want {
			t.Fatalf("prompt(%q,%q)=%q", tc.cwd, tc.home, got)
		}
	}
	previous := i18n.Locale()
	defer i18n.SetLocale(previous)
	for _, tc := range []struct{ locale, command, description, exit string }{{i18n.LocaleEN, "(send input)", "Terminal t-3", "no exit code"}, {i18n.LocaleZH, "（发送输入）", "终端 t-3", "未正常退出"}} {
		i18n.SetLocale(tc.locale)
		m := &toolTerminalModel{SessionID: "t-3", HasExit: true, MissingExit: true}
		if terminalCommand(m) != tc.command || terminalDescription(m) != tc.description || terminalExitText(m) != tc.exit {
			t.Fatalf("localized terminal: %q/%q/%q", terminalCommand(m), terminalDescription(m), terminalExitText(m))
		}
	}
}

func terminalExport(t *testing.T, tt *ui.Tester, suffix string) {
	t.Helper()
	dir := os.Getenv("TERMINAL_EVIDENCE_DIR")
	if dir == "" {
		return
	}
	if err := os.MkdirAll(dir, 0755); err != nil {
		t.Fatal(err)
	}
	base := filepath.Join(dir, "terminal-pass259-native-software-"+suffix)
	f, err := os.Create(base + ".png")
	if err != nil {
		t.Fatal(err)
	}
	if err := png.Encode(f, tt.Image()); err != nil {
		f.Close()
		t.Fatal(err)
	}
	if err := f.Close(); err != nil {
		t.Fatal(err)
	}
	boxes := map[string]ui.Rect{}
	for _, label := range []string{"Terminal output", "Terminal command banner", i18n.T(i18n.KeyChatToolOutput), "Copy", "Copied", "No output", "exit code 7", "following terminal"} {
		if r, ok := tt.Find(label); ok {
			boxes[label] = r
		}
	}
	payload, err := json.MarshalIndent(map[string]any{"scope": "MyGo software renderer only; no live Electron, OS input or hardware hover claim", "boxes": boxes, "texts": tt.Texts()}, "", "  ")
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(base+".json", payload, 0644); err != nil {
		t.Fatal(err)
	}
}
