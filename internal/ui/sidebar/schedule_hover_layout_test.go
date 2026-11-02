package sidebar

import (
	"encoding/json"
	"math"
	"reflect"
	"strings"
	"testing"
	"time"

	"github.com/egoist/mygo/ui"
	schedstore "whaleisle/internal/store/schedule"
)

// Inspect final memory-renderer nodes without mutating them or exporting a
// production debug hook. This verifies software layout/style, not native pixels.
func scheduleLayoutTextNode(t *testing.T, tt *ui.Tester, text string) reflect.Value {
	t.Helper()
	ctx := reflect.ValueOf(tt).Elem().FieldByName("rt").Elem().FieldByName("c")
	chunks, used := ctx.FieldByName("chunks"), int(ctx.FieldByName("used").Int())
	var found reflect.Value
	matches := 0
	for i := 0; i < used; i++ {
		chunk := chunks.Index(i / chunks.Index(0).Len())
		n := chunk.Index(i % chunk.Len())
		if n.FieldByName("text").String() == text {
			found, matches = n, matches+1
		}
	}
	if matches != 1 {
		t.Fatalf("text %q identifies %d final memory nodes", text, matches)
	}
	return found
}

func scheduleLayoutBox(n reflect.Value) ui.Rect {
	return ui.Rect{X: float32(n.FieldByName("x").Float()), Y: float32(n.FieldByName("y").Float()), W: float32(n.FieldByName("w").Float()), H: float32(n.FieldByName("h").Float())}
}

func scheduleLayoutColor(v reflect.Value) ui.Color {
	return ui.RGB(uint8(v.FieldByName("R").Uint()), uint8(v.FieldByName("G").Uint()), uint8(v.FieldByName("B").Uint()))
}

func scheduleLayoutNear(a, b float32) bool { return math.Abs(float64(a-b)) < .1 }

func TestScheduleHoverLayoutOriginalRowsColorsAndSelectableTiming(t *testing.T) {
	tt, m, in, start := sessionHoverDraftFixture(t)
	m.Snapshot.Items[0].Title = "Full session title remains the whole-card copy value"
	rows := []schedstore.Entry{
		{ID: "past", Title: "Overdue task with enough title words to wrap across the compact two hundred forty four DIP preview", SessionID: "a", Status: "active", ScheduledAt: start.Add(-time.Hour).Format(time.RFC3339), Raw: json.RawMessage(`{"kind":"every","everySeconds":600}`)},
		{ID: "future", Title: "Future task", SessionID: "a", Status: "active", ScheduledAt: start.Add(time.Hour).Format(time.RFC3339), Raw: json.RawMessage(`{"kind":"at"}`)},
		{ID: "omitted", Title: "Omitted task must not render", SessionID: "a", Status: "active", ScheduledAt: start.Add(2 * time.Hour).Format(time.RFC3339)},
	}
	m.Schedules = schedstore.ViewState{ClientReady: true, CatalogSettled: true, Rows: rows}
	tt.SetScale(1.5)
	tt.SetPreferences(ui.Preferences{ReduceMotion: false, TextScale: 1})
	tt.Frame()
	hoverDraftRow(t, tt, "a")
	tt.SetTime(start.Add(800 * time.Millisecond))
	card, ok := tt.Find("Copy session title a")
	if !ok || card.W != 244 {
		t.Fatalf("wrong compact task card: %+v", card)
	}
	if tt.HasText(rows[2].Title) || !tt.HasText("1 more") {
		t.Fatalf("task limit or omission differs: %q", tt.Texts())
	}
	var taskBoxes []ui.Rect
	var selectableTiming string
	var timingBox ui.Rect
	for index, task := range rows[:2] {
		title := scheduleLayoutTextNode(t, tt, task.Title)
		body := title.FieldByName("parent").Elem()
		row := body.FieldByName("parent").Elem()
		group := row.FieldByName("parent").Elem()
		iconBox := row.FieldByName("first").Elem()
		icon := iconBox.FieldByName("first").Elem()
		titleBox, bodyBox, rowBox, clockBox, glyphBox := scheduleLayoutBox(title), scheduleLayoutBox(body), scheduleLayoutBox(row), scheduleLayoutBox(iconBox), scheduleLayoutBox(icon)
		taskBoxes = append(taskBoxes, rowBox)
		if !row.FieldByName("row").Bool() || row.FieldByName("gapX").Float() != 8 || row.FieldByName("align").Uint() != uint64(ui.Start) || body.FieldByName("gapY").Float() != 2 || group.FieldByName("gapY").Float() != 8 {
			t.Fatal("task row/group/body flex spacing differs from installed CSS")
		}
		if clockBox.W != 12 || clockBox.H != 16 || glyphBox.W != 12 || glyphBox.H != 12 || !scheduleLayoutNear(glyphBox.Y, clockBox.Y+2) || !scheduleLayoutNear(bodyBox.X, clockBox.X+20) || !scheduleLayoutNear(bodyBox.Y, clockBox.Y) {
			t.Fatalf("clock/body geometry clock=%+v glyph=%+v body=%+v", clockBox, glyphBox, bodyBox)
		}
		if icon.FieldByName("role").Uint() != uint64(ui.RoleNone) || iconBox.FieldByName("role").Uint() != uint64(ui.RoleNone) || icon.FieldByName("svg").Pointer() != reflect.ValueOf(sessionHoverTaskClockArtwork).Pointer() {
			t.Fatal("task clock is not the exact decorative regular primitive")
		}
		if scheduleLayoutColor(icon.FieldByName("ts").FieldByName("color")) != ui.RGB(207, 211, 214) {
			t.Fatal("clock ink differs from #CFD3D6")
		}
		if titleBox.X+titleBox.W > card.X+card.W-15.9 || rowBox.X+rowBox.W > card.X+card.W-15.9 || bodyBox.W != 192 {
			t.Fatalf("task escaped compact card: title=%+v row=%+v body=%+v card=%+v", titleBox, rowBox, bodyBox, card)
		}
		style := title.FieldByName("ts")
		if style.FieldByName("size").Float() != 12 || style.FieldByName("lineHeight").Float() != 16 || !style.FieldByName("fixedLine").Bool() || scheduleLayoutColor(style.FieldByName("color")) != ui.RGB(255, 255, 255) {
			t.Fatal("task name typography differs from white12/16")
		}
		timing := title.FieldByName("next").Elem()
		spans := timing.FieldByName("spans")
		absolute, relative := schedulePreviewParts(task, start.Add(800*time.Millisecond))
		prefix := scheduleHoverFrequency(task, start.Add(800*time.Millisecond)) + " · " + absolute + " "
		if spans.Len() != 2 || spans.Index(0).FieldByName("Text").String() != prefix || spans.Index(1).FieldByName("Text").String() != relative || scheduleLayoutColor(spans.Index(0).FieldByName("Color")) != ui.RGB(207, 211, 214) || scheduleLayoutColor(spans.Index(1).FieldByName("Color")) != ui.RGB(173, 178, 184) {
			t.Fatalf("timing must remain one inline paragraph with distinct absolute/relative ink; text=%q", timing.FieldByName("text").String())
		}
		if timing.FieldByName("ts").FieldByName("lineHeight").Float() != 16 || timing.FieldByName("ts").FieldByName("size").Float() != 12 {
			t.Fatal("timing typography differs from12/16")
		}
		if index == 0 {
			selectableTiming = prefix + relative
			timingBox = scheduleLayoutBox(timing)
			if titleBox.H <= 16 {
				t.Fatal("long task title did not exercise wrapping")
			}
		}
	}
	if !scheduleLayoutNear(taskBoxes[1].Y, taskBoxes[0].Y+taskBoxes[0].H+8) {
		t.Fatal("tasks do not have8DIP separation")
	}
	omission := scheduleLayoutTextNode(t, tt, "1 more")
	if scheduleLayoutColor(omission.FieldByName("ts").FieldByName("color")) != ui.RGB(173, 178, 184) || !scheduleLayoutNear(scheduleLayoutBox(omission).Y, taskBoxes[1].Y+taskBoxes[1].H+8) {
		t.Fatal("omission placement/ink differs from installed CSS")
	}
	navigationEvidence(t, tt, "schedule-hover-task-layout")
	// Rich timing remains a selectable paragraph inside the whole-card button.
	tt.Move(timingBox.X+15, timingBox.Y+8)
	tt.SetClipboard("memory timing sentinel")
	tt.Press(timingBox.X+15, timingBox.Y+8)
	tt.Move(timingBox.X+90, timingBox.Y+8)
	tt.Release(timingBox.X+90, timingBox.Y+8)
	if tt.Clipboard() != "memory timing sentinel" || tt.HasText("Copied") {
		t.Fatal("task timing drag activated whole-card copy")
	}
	tt.Key(ui.Cmd, ui.KeyC)
	selected := tt.Clipboard()
	if selected == "" || selected == "memory timing sentinel" || !strings.Contains(selectableTiming, selected) || selected == selectableTiming {
		t.Fatalf("rich timing paragraph lost range selection: %q in %q", selected, selectableTiming)
	}
	if in.mutations != 0 || m.Snapshot.SelectedID != "a" {
		t.Fatal("task selection changed the session")
	}
}
