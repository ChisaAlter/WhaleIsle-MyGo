package chat

import (
	"github.com/egoist/mygo/ui"
	"image/png"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
	"whaleisle/internal/store/session"
	"whaleisle/internal/ui/i18n"
)

func TestParityDeliverableHoverAndSplitOpen(t *testing.T) {
	old := i18n.Locale()
	i18n.SetLocale(i18n.LocaleEN)
	defer i18n.SetLocale(old)
	target := session.TurnFileTarget{Kind: "changes", Turn: 1, Seq: 2}
	snapshot := session.TurnFilesSnapshot{Scope: session.TurnFileScope{Binding: "synthetic", SessionID: "child", Generation: 1}, Results: map[session.TurnFileRequest]session.TurnFileResult{
		{Target: target, Action: "summary"}:      {Summary: &session.TurnChangesSummary{Total: 1, Added: 1, Files: []session.TurnChangedFile{{Display: "file.go", Added: 1}}}},
		{Target: target, Action: "host"}:         {Host: &session.TurnPresentedHost{Available: true}},
		{Target: target, Action: "applications"}: {Applications: []session.TurnFileApplication{{ID: "editor", Name: "Editor", Default: true}}},
	}}
	var reads, actions []session.TurnFileRequest
	style := DefaultToolCardStyle(ui.RGB(20, 20, 20), ui.RGB(90, 90, 90), ui.RGB(205, 205, 205), ui.RGB(246, 247, 249), ui.RGB(0, 90, 180))
	tt := ui.NewTester(func(c *ui.Context) {
		ui.Column(c).Fill().Padding(20).Gap(10).Children(func() {
			addressedTurnDeliveryView(c, session.TurnDelivery{Turn: 1, Changes: &target}, snapshot, true, func(target session.TurnFileTarget, action string) {
				actions = append(actions, session.TurnFileRequest{Target: target, Action: action})
			}, nil, style, func(target session.TurnFileTarget, action string) {
				reads = append(reads, session.TurnFileRequest{Target: target, Action: action})
			})
			ui.Box(c).Height(1).Label("Following delivery")
		})
	}, 850, 500)
	now := time.Unix(300, 0)
	tt.SetTime(now)
	row, ok := tt.Find("file.go")
	if !ok {
		t.Fatal(tt.Texts())
	}
	following, _ := tt.Find("Following delivery")
	tt.Move(row.X+40, row.Y+row.H/2)
	tt.SetTime(now.Add(499 * time.Millisecond))
	if len(reads) != 0 || len(actions) != 0 {
		t.Fatal("hover fired early or navigated")
	}
	tt.SetTime(now.Add(501 * time.Millisecond))
	if len(reads) != 1 || reads[0].Action != "diff" || len(actions) != 0 {
		t.Fatalf("hover did not read only its diff: reads=%+v actions=%+v", reads, actions)
	}
	snapshot.Results[session.TurnFileRequest{Target: target, Action: "diff"}] = session.TurnFileResult{Diff: &session.TurnFileDiff{Kind: "text", Hunks: []session.TurnDiffHunk{{OldStart: 1, OldLines: 1, NewStart: 1, NewLines: 1, Lines: []string{"-old", "+new"}}}}}
	tt.Frame()
	after, _ := tt.Find("Following delivery")
	if after.Y != following.Y || !strings.Contains(strings.Join(tt.Texts(), "\n"), "new") {
		t.Fatal("hover preview expanded layout or lost diff")
	}
	if dir := os.Getenv("WHALEISLE_CHAT_RENDER_DIR"); dir != "" {
		if err := os.MkdirAll(dir, 0700); err != nil {
			t.Fatal(err)
		}
		f, err := os.Create(filepath.Join(dir, "native-deliverable-hover.png"))
		if err != nil {
			t.Fatal(err)
		}
		if err := png.Encode(f, tt.Image()); err != nil {
			t.Fatal(err)
		}
		f.Close()
	}
	tt.Move(830, 480)
	if err := tt.Click("Open file: file.go"); err != nil {
		t.Fatal(err)
	}
	if len(actions) != 1 || actions[0].Action != session.TurnFileApplicationAction("editor") {
		t.Fatal("main open seat did not invoke default app exactly once")
	}
	snapshot.Results[session.TurnFileRequest{Target: target, Action: "applications"}] = session.TurnFileResult{}
	tt.Frame()
	if err := tt.Click("Open file: file.go"); err != nil {
		t.Fatal(err)
	}
	if len(actions) != 2 || actions[1].Action != "reveal" {
		t.Fatal("no-application seat did not reveal")
	}
	if err := tt.Click("File actions: file.go"); err != nil {
		t.Fatal(err)
	}
	if len(tt.Menu()) == 0 {
		t.Fatal("chevron lost independent menu")
	}
}
