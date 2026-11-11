package chat

import (
	"image/png"
	"os"
	"path/filepath"
	"testing"

	"github.com/egoist/mygo/ui"
	"whaleisle/internal/store/session"
	"whaleisle/internal/ui/i18n"
)

func TestQueueDockPointerReorder(t *testing.T) {
	prev := i18n.Locale()
	i18n.SetLocale(i18n.LocaleZH)
	defer i18n.SetLocale(prev)
	for _, name := range []string{"up", "down", "click", "scope-changed", "busy", "editing", "removed", "escape", "rejected"} {
		t.Run(name, func(t *testing.T) {
			open := true
			moves := [][2]any{}
			m := PlanBoardModel{ThemeID: "deepseek.light", Open: &open, Board: session.PlanBoard{
				Queue:        []session.PlanQueueItem{{ID: "a", Preview: "分析项目结构"}, {ID: "b", Preview: "整理可用接口"}, {ID: "c", Preview: "补全运行说明"}},
				QueueActions: session.QueueActionState{Scope: "host/session-1", Mutable: true, Running: true},
			}, OnQueueMoveResult: func(id string, index int) bool { moves = append(moves, [2]any{id, index}); return name != "rejected" }}
			tt := ui.NewTester(func(c *ui.Context) { QueueDockView(c, m) }, 620, 240)
			grip := func(n int) ui.Rect {
				t.Helper()
				r, ok := tt.Find(queueReorderLabel(n, 3))
				if !ok {
					t.Fatal("missing grip", tt.Texts())
				}
				return r
			}
			source, target := grip(3), grip(1)
			wantID, wantIndex := "c", 0
			targetY := target.Y + 2
			if name == "down" {
				source, target = grip(1), grip(3)
				targetY = target.Y + target.H - 2
				wantID, wantIndex = "a", 2
			}
			x, y := source.X+source.W/2, source.Y+source.H/2
			if name == "click" {
				tt.ClickAt(x, y)
			} else {
				tt.Press(x, y)
				tt.Move(x+6, y)
				switch name {
				case "scope-changed":
					m.Board.QueueActions.Scope = "host/session-2"
				case "busy":
					m.Board.QueueActions.Busy = true
				case "editing":
					m.QueueEditor.ItemID = "b"
				case "removed":
					m.Board.Queue = m.Board.Queue[:2]
				case "escape":
					tt.Key(0, ui.KeyEscape)
				}
				tt.Frame()
				tt.Move(target.X+target.W/2, targetY)
				if name == "up" {
					saveQueueCapture(t, tt, "queue-drag-light.png")
				}
				tt.Release(target.X+target.W/2, targetY)
			}
			if name == "up" || name == "down" || name == "rejected" {
				if len(moves) != 1 || moves[0] != [2]any{wantID, wantIndex} {
					t.Fatalf("moves=%v want [%s %d]", moves, wantID, wantIndex)
				}
			} else if len(moves) != 0 {
				t.Fatalf("invalid move dispatched: %v", moves)
			}
			if name == "rejected" && !tt.HasText(queueMoveFailedCopy()) {
				t.Fatal("missing move failure feedback")
			}
			if name == "down" {
				m.ThemeID, m.SystemDark = "deepseek.dark", true
				tt.SetDark(true)
				tt.SetSize(360, 240)
				tt.Frame()
				saveQueueCapture(t, tt, "queue-narrow-dark.png")
			}
		})
	}
}

func saveQueueCapture(t *testing.T, tt *ui.Tester, name string) {
	if root := os.Getenv("WHALE_ALIGNMENT_CAPTURE_DIR"); root != "" {
		if err := os.MkdirAll(root, 0755); err != nil {
			t.Fatal(err)
		}
		f, err := os.Create(filepath.Join(root, name))
		if err != nil {
			t.Fatal(err)
		}
		defer f.Close()
		if err := png.Encode(f, tt.Image()); err != nil {
			t.Fatal(err)
		}
	}
}
