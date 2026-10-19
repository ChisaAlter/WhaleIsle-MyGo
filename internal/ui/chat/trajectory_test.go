package chat

import (
	"strings"
	"testing"

	"github.com/egoist/mygo/ui"

	"whaleisle/internal/store/session"
)

func TestBuildTrajectoryRowsFiltersAndMaps(t *testing.T) {
	msgs := []session.Message{
		{ID: "u1", Kind: session.MessageKindText, Role: "user", Body: "hello world", Seq: 1},
		{ID: "a1", Kind: session.MessageKindText, Role: "assistant", Body: "hi there", Seq: 2},
		{ID: "t1", Kind: session.MessageKindTool, Role: "tool", Body: "bash", EventType: "tool/call"},
		{ID: "th1", Kind: session.MessageKindThinking, Role: "assistant", Body: "ponder"},
	}
	rows := buildTrajectoryRows(msgs, "", false, false)
	if len(rows) != 4 {
		t.Fatalf("len=%d", len(rows))
	}
	if rows[0].Input != "hello world" || rows[1].Output != "hi there" {
		t.Fatalf("map = %+v / %+v", rows[0], rows[1])
	}
	if rows[2].Tools != "bash" {
		t.Fatalf("tool = %q", rows[2].Tools)
	}
	resultRows := buildTrajectoryRows([]session.Message{{
		ID: "r1", Kind: session.MessageKindTool, Role: "tool",
		Body: "ok", EventType: "tool/result", Args: `{"stdout":"hi"}`,
	}}, "", false, false)
	if len(resultRows) != 1 || resultRows[0].Input != "工具结果" || resultRows[0].RawHint == "" {
		t.Fatalf("tool/result = %+v", resultRows)
	}

	filtered := buildTrajectoryRows(msgs, "hello", false, false)
	if len(filtered) != 1 || filtered[0].ID != "u1" {
		t.Fatalf("search = %+v", filtered)
	}
	noTools := buildTrajectoryRows(msgs, "", false, true)
	for _, r := range noTools {
		if r.Kind == session.MessageKindTool {
			t.Fatal("calls collapsed should drop tools")
		}
	}
}

func TestTrajectoryViewPaintsToolbar(t *testing.T) {
	st := session.New()
	// Inject nothing — empty ledger still shows Electron chrome labels.
	tt := ui.NewTester(func(c *ui.Context) {
		TrajectoryView(c, TrajectoryModel{
			Snapshot: st.Snapshot(),
			ThemeID:  "system",
		})
	}, 900, 600)
	for _, label := range []string{"时长", "轮次", "调用", "轨迹时间线"} {
		if !tt.HasText(label) && !strings.Contains(strings.Join(tt.Texts(), "\n"), label) {
			t.Fatalf("missing %q; texts=%q", label, tt.Texts())
		}
	}
}
