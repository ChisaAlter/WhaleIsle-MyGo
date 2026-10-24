package chat

import (
	"bytes"
	"encoding/json"
	"fmt"
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

func toolDetailsViewCall(t *testing.T, name, phase, activation, objective, reason string) *session.ToolCallView {
	t.Helper()
	goal := map[string]any{"id": "recorded-goal", "objective": objective, "revision": 7, "roundsStarted": 123, "maxGoalRounds": 256, "phase": phase}
	if reason != "" {
		goal["blockedReason"] = map[string]any{"code": "recorded", "message": reason}
	}
	call := toolDetailsModelCallForTest(t, false, name, `{"objective":"authored argument"}`, toolDetailsModelJSONForTest(t, map[string]any{"goal": goal, "activation": activation}))
	call.Name = name
	call.Arguments = `{"objective":"authored argument"}`
	return call
}

func toolDetailsViewFixture(t *testing.T, call *session.ToolCallView, dark, textOnly, inspect bool, width int) (*ui.Tester, *map[string]bool, *[]string) {
	t.Helper()
	oldLocale, oldPrefs := i18n.Locale(), theme.CurrentPreferences()
	i18n.SetLocale(i18n.LocaleEN)
	theme.SetPreferences(theme.Defaults())
	t.Cleanup(func() { i18n.SetLocale(oldLocale); theme.SetPreferences(oldPrefs) })
	expanded := map[string]bool{}
	opened := []string{}
	tt := ui.NewTester(func(c *ui.Context) {
		theme.ApplyInterface(c)
		palette := theme.Resolve(theme.DefaultID).Active(dark)
		ui.Column(c).Fill().Padding(20).Background(chrome.Color(palette.BG)).Children(func() {
			opts := ToolImageGalleryOptions{ID: "details-view", Cwd: "/work", Expanded: &expanded, TextOnly: textOnly}
			if inspect {
				opts.OnInspectCall = func(id string) { opened = append(opened, id) }
			}
			ToolMessageView(c, session.Message{Tool: call}, toolCardStyle(palette, chrome.Color(palette.FG), chrome.Color(palette.Muted), chrome.Color(palette.Border)), opts)
			ui.Box(c).Height(1).Label("Following details flow")
		})
	}, width, 700)
	tt.SetTime(time.Unix(12300, 0))
	tt.Move(float32(width-5), 695)
	return tt, &expanded, &opened
}
func openDetailsView(tt *ui.Tester, expanded *map[string]bool) {
	(*expanded)["details-view|tool:goal-123"] = true
	tt.SetTime(time.Unix(12300, 0).Add(time.Second))
	tt.Frame()
	tt.SetTime(time.Unix(12300, 0).Add(2 * time.Second))
	tt.Frame()
}

func TestToolDetailsViewRecordedGoalAndInspectIdentity(t *testing.T) {
	for _, dark := range []bool{false, true} {
		t.Run(fmt.Sprint(dark), func(t *testing.T) {
			call := toolDetailsViewCall(t, "get_goal", "active", "disarmed", "Recorded objective", "Recorded reason")
			tt, expanded, opened := toolDetailsViewFixture(t, call, dark, false, true, 640)
			if _, ok := tt.Find("Inspect"); ok {
				t.Fatal("collapsed inspect")
			}
			if !tt.HasText("Recorded objective") || tt.HasText("authored argument") {
				t.Fatal("result header", tt.Texts())
			}
			openDetailsView(tt, expanded)
			for _, want := range []string{"Recorded objective", "Status", "Awaiting continuation", "Rounds", "123 / 256", "Blocker", "Recorded reason", "Inspect"} {
				if !tt.HasText(want) {
					t.Fatal("missing", want, tt.Texts())
				}
			}
			for _, absent := range []string{"IN", "OUT", "Copy", `"revision"`, `"goal"`, "authored argument", "Path"} {
				if tt.HasText(absent) {
					t.Fatal("duplicate", absent, tt.Texts())
				}
			}
			button, _ := tt.Find("Inspect")
			toolTreeNear(t, "inspect top", button.Y, 56)
			toolTreeNear(t, "inspect right", button.X+button.W, 608)
			toolTreeNear(t, "inspect height", button.H, 22)
			if err := tt.Click("Inspect"); err != nil {
				t.Fatal(err)
			}
			if len(*opened) != 1 || (*opened)[0] != "goal-123" || !(*expanded)["details-view|tool:goal-123"] {
				t.Fatal("inspect identity/disclosure", opened, expanded)
			}
			// Focus changes keep the same call payload on native Space and Enter.
			tt.Key(ui.Shift, ui.KeyTab)
			tt.Key(0, ui.KeyTab)
			if !tt.Focused("Inspect") {
				t.Fatal("inspect focus")
			}
			tt.Key(0, ui.KeySpace)
			tt.Key(0, ui.KeyEnter)
			if len(*opened) != 3 || !(*expanded)["details-view|tool:goal-123"] {
				t.Fatal("keyboard action", opened, expanded)
			}
			tt.SetTime(time.Unix(12300, 0).Add(3 * time.Second))
			tt.Frame()
			toolDetailsViewExport(t, tt, fmt.Sprintf("goal-dark-%v", dark))
		})
	}
}

func TestToolDetailsViewLocaleEmptyFallbackAndReadonly(t *testing.T) {
	call := toolDetailsViewCall(t, "update_goal", "blocked", "armed", "Recorded objective", "保留的原因")
	tt, expanded, _ := toolDetailsViewFixture(t, call, false, false, true, 640)
	i18n.SetLocale(i18n.LocaleZH)
	openDetailsView(tt, expanded)
	for _, want := range []string{"状态", "受阻", "执行轮次", "受阻原因", "保留的原因", "查看"} {
		if !tt.HasText(want) {
			t.Fatal(want, tt.Texts())
		}
	}
	empty := toolDetailsModelCallForTest(t, false, "get_goal", `{}`, `{"goal":null}`)
	empty.Name = "get_goal"
	empty.Arguments = `{}`
	tt, expanded, _ = toolDetailsViewFixture(t, empty, false, false, true, 420)
	openDetailsView(tt, expanded)
	if !tt.HasText("No goal") || tt.HasText("Status") || tt.HasText("OUT") {
		t.Fatal("empty", tt.Texts())
	}
	for _, textOnly := range []bool{false, true} {
		bad := toolDetailsViewCall(t, "create_goal", "active", "armed", "Recorded objective", "")
		if !textOnly {
			bad.State = "error"
		}
		tt, expanded, _ = toolDetailsViewFixture(t, bad, false, textOnly, true, 640)
		openDetailsView(tt, expanded)
		_, stateLabel := tt.Find("Status")
		_, roundsLabel := tt.Find("Rounds")
		if stateLabel || roundsLabel || !tt.HasText("OUT") || !tt.HasText(`"goal":`) {
			t.Fatal("evidence fallback", textOnly, tt.Texts())
		}
		if textOnly {
			if _, ok := tt.Find("Inspect"); ok {
				t.Fatal("readonly navigation")
			}
		}
	}
}

func TestToolDetailsViewInspectHoverAndFlow(t *testing.T) {
	// At rest opacity does not remove the focusable action. Paint appears after
	// owner hover and after visible keyboard focus, including generic bodies.
	call := toolDetailsViewCall(t, "get_goal", "active", "armed", "Recorded objective", "")
	tt, expanded, _ := toolDetailsViewFixture(t, call, false, false, true, 640)
	openDetailsView(tt, expanded)
	button, _ := tt.Find("Inspect")
	before := tt.Image()
	tt.Move(30, 30)
	tt.SetTime(time.Unix(12300, 0).Add(2500 * time.Millisecond))
	tt.Frame()
	after := tt.Image()
	changed := false
	for y := int(button.Y); y < int(button.Y+button.H); y++ {
		for x := int(button.X); x < int(button.X+button.W); x++ {
			if before.At(x, y) != after.At(x, y) {
				changed = true
			}
		}
	}
	if !changed {
		t.Fatal("inspect did not reveal with owner hover")
	}
	generic := toolDetailsViewCall(t, "get_goal", "active", "armed", "Recorded objective", "")
	generic.Name = "unknown_tool"
	generic.Call.Raw = []byte(`{"type":"tool/call","data":{"name":"unknown_tool","arguments":"{}"}}`)
	tt, expanded, _ = toolDetailsViewFixture(t, generic, false, false, true, 640)
	openDetailsView(tt, expanded)
	out, _ := tt.Find("OUT")
	button, _ = tt.Find("Inspect")
	following, _ := tt.Find("Following details flow")
	if button.Y <= out.Y || following.Y < button.Y+button.H {
		t.Fatal("generic inspect must stay in flow", out, button, following)
	}
	if !tt.HasText("IN") || !tt.HasText("OUT") {
		t.Fatal(tt.Texts())
	}
}

func TestToolDetailsViewPresetsScrollAndGeometry(t *testing.T) {
	oldLocale, oldPrefs := i18n.Locale(), theme.CurrentPreferences()
	i18n.SetLocale(i18n.LocaleEN)
	theme.SetPreferences(theme.Defaults())
	t.Cleanup(func() { i18n.SetLocale(oldLocale); theme.SetPreferences(oldPrefs) })
	models := map[string]*toolDetailsModel{
		"goal":       {Items: []toolDetailItem{{Title: "Recorded objective", Fields: []toolDetailField{{"Status", "Active"}, {"Rounds", "123 / 256"}}}}},
		"no-inspect": {Items: []toolDetailItem{{Title: "Recorded objective", Fields: []toolDetailField{{"Status", "Active"}, {"Rounds", "123 / 256"}}}}},
		"empty":      {Empty: "No goal"},
		"long":       {Items: []toolDetailItem{{Title: strings.Repeat("long recorded objective\n", 40), Fields: []toolDetailField{{"Status", "Blocked"}, {"Rounds", "123 / 256"}, {"Blocker", strings.Repeat("long reason ", 10)}}}}},
	}
	records := map[string]any{}
	for _, name := range []string{"goal", "no-inspect", "empty", "long"} {
		t.Run(name, func(t *testing.T) {
			var body, following ui.Rect
			tt := ui.NewTester(func(c *ui.Context) {
				theme.ApplyInterface(c)
				palette := theme.Resolve(theme.DefaultID).Active(false)
				st := toolCardStyle(palette, chrome.Color(palette.FG), chrome.Color(palette.Muted), chrome.Color(palette.Border))
				ui.Column(c).Width(420).Children(func() {
					ui.Column(c).FillWidth().ObserveLayout(func(r ui.Rect) { body = r }).Children(func() { toolDetailsCard(c, models[name], st, "preset", name != "no-inspect") })
					ui.Box(c).Height(1).ObserveLayout(func(r ui.Rect) { following = r })
				})
			}, 420, 700)
			tt.Frame()
			want := float32(100)
			if name == "empty" {
				want = 50
			}
			if name == "long" {
				want = 330
			}
			toolTreeNear(t, "body", body.H, want)
			toolTreeNear(t, "following", following.Y, want)
			boxes := map[string]ui.Rect{"body": body, "following": following}
			for _, label := range []string{"Recorded objective", "No goal", "Status", "Active", "Rounds", "123 / 256", "Blocker"} {
				if r, ok := tt.Find(label); ok {
					boxes[label] = r
				}
			}
			if name == "goal" {
				toolTreeNear(t, "label x", boxes["Status"].X, 19)
				toolTreeNear(t, "value x", boxes["Active"].X, 102.5)
				toolTreeNear(t, "title y", boxes["Recorded objective"].Y, 15)
				toolTreeNear(t, "field y", boxes["Status"].Y, 41)
			}
			if name == "long" {
				before, _ := tt.Find("Status")
				tt.Scroll(150, 200, 0, 1000)
				after, _ := tt.Find("Status")
				if after.Y >= before.Y {
					t.Fatal("detail list not independently scrollable", before, after)
				}
			}
			records[name] = boxes
		})
	}
	if dir := os.Getenv("TOOL_DETAILS_EVIDENCE_DIR"); dir != "" {
		raw, _ := json.MarshalIndent(records, "", "  ")
		if err := os.WriteFile(filepath.Join(dir, "tool-details-pass267-native-preset-geometry.json"), raw, 0600); err != nil {
			t.Fatal(err)
		}
	}
}

func toolDetailsViewExport(t *testing.T, tt *ui.Tester, name string) {
	t.Helper()
	dir := os.Getenv("TOOL_DETAILS_EVIDENCE_DIR")
	if dir == "" {
		return
	}
	prefix := filepath.Join(dir, "tool-details-pass267-native-software-"+name)
	var pngBytes bytes.Buffer
	if err := png.Encode(&pngBytes, tt.Image()); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(prefix+".png", pngBytes.Bytes(), 0600); err != nil {
		t.Fatal(err)
	}
	boxes := map[string]ui.Rect{}
	for _, label := range []string{"Status", "Awaiting continuation", "Rounds", "123 / 256", "Blocker", "Recorded reason", "Inspect", "Following details flow"} {
		if r, ok := tt.Find(label); ok {
			boxes[label] = r
		}
	}
	raw, _ := json.MarshalIndent(map[string]any{"rects": boxes, "texts": tt.Texts()}, "", "  ")
	if err := os.WriteFile(prefix+".json", raw, 0600); err != nil {
		t.Fatal(err)
	}
}

func toolDetailsScheduleCall(t *testing.T, name string, value any) *session.ToolCallView {
	t.Helper()
	call := toolDetailsModelCallForTest(t, false, name, `{}`, toolDetailsModelJSONForTest(t, value))
	call.Name, call.Arguments = name, `{}`
	return call
}

// Installed reminder rows reuse the goal card but add a collapsed count summary,
// a multiple-item list and a deleted receipt.
func TestToolDetailsViewScheduleListEmptyAndDelete(t *testing.T) {
	oldLocale, oldPrefs := i18n.Locale(), theme.CurrentPreferences()
	i18n.SetLocale(i18n.LocaleEN)
	theme.SetPreferences(theme.Defaults())
	t.Cleanup(func() { i18n.SetLocale(oldLocale); theme.SetPreferences(oldPrefs) })
	first := map[string]any{"id": "s-1", "prompt": "Recorded standup prompt", "title": "Standup", "scheduledAt": "2026-10-25T01:30:00.000Z", "deliveryMode": "host", "state": "scheduled", "kind": "daily", "time": "09:30:00.000", "timeZone": "Asia/Shanghai"}
	second := map[string]any{"id": "s-2", "prompt": "Recorded hourly prompt", "scheduledAt": "2026-10-24T02:00:00.000Z", "deliveryMode": "session-local", "state": "overdue", "kind": "every", "everySeconds": 3600}
	tt, expanded, _ := toolDetailsViewFixture(t, toolDetailsScheduleCall(t, "schedule_list", []any{first, second}), false, false, true, 640)
	if !tt.HasText("2 reminders") {
		t.Fatal("collapsed list summary", tt.Texts())
	}
	openDetailsView(tt, expanded)
	for _, want := range []string{"Standup", "Recorded hourly prompt", "Scheduled for", "Repeat", "Every day at 09:30 (Asia/Shanghai)", "Every 1 h"} {
		if !tt.HasText(want) {
			t.Fatal("missing", want, tt.Texts())
		}
	}
	for _, label := range []string{"Scheduled", "Overdue, awaiting session resume"} {
		if _, ok := tt.Find(label); !ok {
			t.Fatal("state label", label, tt.Texts())
		}
	}
	tt, expanded, _ = toolDetailsViewFixture(t, toolDetailsScheduleCall(t, "schedule_list", []any{}), false, false, true, 640)
	if !tt.HasText("0 reminders") {
		t.Fatal("empty list summary", tt.Texts())
	}
	openDetailsView(tt, expanded)
	if !tt.HasText("No reminders") || tt.HasText("Scheduled for") {
		t.Fatal("empty list card", tt.Texts())
	}
	tt, expanded, _ = toolDetailsViewFixture(t, toolDetailsScheduleCall(t, "schedule_delete", map[string]any{"id": "s-9", "deleted": true}), false, false, true, 640)
	openDetailsView(tt, expanded)
	for _, want := range []string{"s-9", "Status", "Deleted"} {
		if !tt.HasText(want) {
			t.Fatal("delete card", want, tt.Texts())
		}
	}
	// An invalid recorded receipt falls back to raw evidence instead of a card.
	tt, expanded, _ = toolDetailsViewFixture(t, toolDetailsScheduleCall(t, "schedule_list", map[string]any{"id": "s-1"}), false, false, true, 640)
	openDetailsView(tt, expanded)
	if tt.HasText("Scheduled for") || !tt.HasText("OUT") {
		t.Fatal("invalid list kept a card", tt.Texts())
	}
}

// The background-job families add the installed recorded-result caption, the
// status badge, subtitles/descriptions and a selectable output region.
func toolDetailsJobCall(t *testing.T, name, args, text string) *session.ToolCallView {
	t.Helper()
	call := toolDetailsModelCallForTest(t, false, name, args, text)
	call.Name, call.Arguments = name, args
	return call
}

func TestToolDetailsViewJobOutput(t *testing.T) {
	call := toolDetailsJobCall(t, "job_output", `{"target":"job-42"}`, "hello\nworld\n[status: completed, exit 0]")
	tt, expanded, _ := toolDetailsViewFixture(t, call, false, false, true, 420)
	if !tt.HasText("job-42 · Completed") {
		t.Fatal("collapsed summary", tt.Texts())
	}
	openDetailsView(tt, expanded)
	for _, want := range []string{"Recorded result", "job-42", "Completed", "exit 0", "hello\nworld"} {
		if !tt.HasText(want) {
			t.Fatal("missing", want, tt.Texts())
		}
	}
	for _, absent := range []string{"OUT", "Copy"} {
		if tt.HasText(absent) {
			t.Fatal("unexpected", absent, tt.Texts())
		}
	}
}

func TestToolDetailsViewJobListAndFallback(t *testing.T) {
	lines := "job-1 [bash] running — npm test\njob-2 [agent] completed — review"
	tt, expanded, _ := toolDetailsViewFixture(t, toolDetailsJobCall(t, "job_list", `{}`, lines), false, false, true, 420)
	if !tt.HasText("2 background jobs") {
		t.Fatal("collapsed count", tt.Texts())
	}
	openDetailsView(tt, expanded)
	for _, want := range []string{"Recorded result", "npm test", "job-1", "Running", "Type", "bash", "review", "agent", "Completed"} {
		if !tt.HasText(want) {
			t.Fatal("missing", want, tt.Texts())
		}
	}
	tt, expanded, _ = toolDetailsViewFixture(t, toolDetailsJobCall(t, "job_list", `{}`, "(no background jobs)"), false, false, true, 420)
	if !tt.HasText("0 background jobs") {
		t.Fatal("empty count", tt.Texts())
	}
	openDetailsView(tt, expanded)
	if !tt.HasText("No results") || tt.HasText("Type") {
		t.Fatal("empty card", tt.Texts())
	}
	// One malformed line keeps the generic evidence body, including its OUT.
	tt, expanded, _ = toolDetailsViewFixture(t, toolDetailsJobCall(t, "job_list", `{}`, "job-1 [bash] running — npm test\nbroken"), false, false, true, 420)
	openDetailsView(tt, expanded)
	if !tt.HasText("OUT") || tt.HasText("Recorded result") || tt.HasText("Type") {
		t.Fatal("invalid list kept a card", tt.Texts())
	}
}

// toolDetailsCardFixture renders the card alone at 420 logical pixels, the width
// the installed geometry numbers are recorded at.
func toolDetailsCardFixture(t *testing.T, model *toolDetailsModel, hasInspect bool) (*ui.Tester, ui.Rect) {
	t.Helper()
	oldLocale, oldPrefs := i18n.Locale(), theme.CurrentPreferences()
	i18n.SetLocale(i18n.LocaleEN)
	theme.SetPreferences(theme.Defaults())
	t.Cleanup(func() { i18n.SetLocale(oldLocale); theme.SetPreferences(oldPrefs) })
	var body ui.Rect
	tt := ui.NewTester(func(c *ui.Context) {
		theme.ApplyInterface(c)
		palette := theme.Resolve(theme.DefaultID).Active(false)
		st := toolCardStyle(palette, chrome.Color(palette.FG), chrome.Color(palette.Muted), chrome.Color(palette.Border))
		ui.Column(c).Width(420).Children(func() {
			ui.Column(c).FillWidth().ObserveLayout(func(r ui.Rect) { body = r }).Children(func() {
				toolDetailsCard(c, model, st, "job-geometry", hasInspect)
			})
			ui.Box(c).Height(1).Label("after job details")
		})
	}, 420, 700)
	tt.Frame()
	return tt, body
}

func toolDetailsJobModelForTest() *toolDetailsModel {
	subtitle, description := "bash · 1234", "exit 0"
	return &toolDetailsModel{
		Items: []toolDetailItem{{
			Title: "job-42", Subtitle: &subtitle, Description: &description,
			Badge:  &toolDetailBadge{Label: "Completed", Tone: "success"},
			Fields: []toolDetailField{{Label: "Type", Value: "bash"}},
			Code:   &toolDetailCode{Text: "hello\nworld"},
		}},
		Summary: "job-42 · Completed", Caption: "Recorded result", Empty: "No results", ExpandedSummary: "job-42",
	}
}

func TestToolDetailsViewJobGeometryAndInspectReserve(t *testing.T) {
	tt, body := toolDetailsCardFixture(t, toolDetailsJobModelForTest(), true)
	caption, ok := tt.Find("Recorded result")
	if !ok {
		t.Fatal("caption", tt.Texts())
	}
	title, _ := tt.Find("job-42")
	badge, _ := tt.Find("Completed")
	subtitle, _ := tt.Find("bash · 1234")
	description, _ := tt.Find("exit 0")
	code, _ := tt.Find("hello\nworld")
	if caption.X != 19 || caption.Y != 15 {
		t.Fatalf("caption box %+v", caption)
	}
	if title.X != 19 || title.Y != 51 {
		t.Fatalf("first item title box %+v", title)
	}
	// The heading line keeps the 8px gap before a flex-none badge that is
	// aligned with the title's first line and sized by its 12/20 label.
	if badge.Y != title.Y || badge.H != 20 || badge.X != title.X+title.W+8 {
		t.Fatalf("badge box %+v vs title %+v", badge, title)
	}
	if subtitle.Y != title.Y+20+3 || description.Y != subtitle.Y+20+6 {
		t.Fatalf("subtitle %+v description %+v title %+v", subtitle, description, title)
	}
	// With a caption the Inspect reserve lives on the caption only, so the item
	// content keeps its full width. (The card's FillWidth keeps the parent's
	// width; its 4px leading margin shifts the content without shrinking it.)
	if code.W != 390 {
		t.Fatalf("code width with caption %v", code.W)
	}
	toolTreeNear(t, "code top", code.Y, description.Y+20+6+20+8)
	// The 5px round dot sits on the badge's first line with a 5px label gap.
	image := tt.Image()
	centerY := int(badge.Y + badge.H/2)
	background := image.RGBAAt(int(badge.X)-3, centerY)
	dotStart, dotEnd := -1, -1
	for x := int(badge.X) - 1; x < int(badge.X)+8; x++ {
		if image.RGBAAt(x, centerY) != background {
			if dotStart < 0 {
				dotStart = x
			}
			dotEnd = x
		}
	}
	// Pixel rounding can move the fractional start by one column and reveal a
	// sixth antialiased column, so the dot is 5 px within rounding.
	if dotStart < int(badge.X) || dotStart > int(badge.X)+1 || dotEnd-dotStart < 4 || dotEnd-dotStart > 5 {
		t.Fatalf("badge dot run %d..%d (badge %+v)", dotStart, dotEnd, badge)
	}
	textStart := -1
	for x := dotEnd + 2; x < int(badge.X)+24; x++ {
		if image.RGBAAt(x, centerY) != background {
			textStart = x
			break
		}
	}
	// The label starts 10 px after the badge's fractional origin: the 5 px dot
	// plus the 5 px gap. Fractional coverage leaves four fully blank columns.
	gap := textStart - dotEnd - 1
	layoutStart := int(badge.X) + 10
	if gap < 4 || gap > 6 || textStart < layoutStart-1 || textStart > layoutStart+1 {
		t.Fatalf("badge label gap %d (dot end %d, text %d, badge %+v)", gap, dotEnd, textStart, badge)
	}
	// The card lays out the caption, the item's heading/subtitle/description,
	// the field row and the two-line code region plus its margins and border.
	toolTreeNear(t, "card height", body.H, 4+1+36+10+20+3+20+6+20+6+20+8+36+10+1+4)
}

func TestToolDetailsViewJobCaptionReserve(t *testing.T) {
	long := strings.Repeat("recorded caption ", 40)
	model := &toolDetailsModel{Caption: long, Items: []toolDetailItem{{Title: "job", Badge: &toolDetailBadge{Label: "Running", Tone: "info"}}}, Empty: "No results"}
	withInspect, _ := toolDetailsCardFixture(t, model, true)
	withoutInspect, _ := toolDetailsCardFixture(t, model, false)
	label := withInspect.Texts()[0]
	inWith, _ := withInspect.Find(label)
	inWithout, _ := withoutInspect.Find(label)
	if inWithout.W-inWith.W != 74 {
		t.Fatalf("caption reserve %v vs %v", inWith.W, inWithout.W)
	}
	// A card with a caption never reserves the item end twice.
	noCaption := &toolDetailsModel{Items: []toolDetailItem{{Title: "job", Code: &toolDetailCode{Text: "hello"}}}}
	plain, _ := toolDetailsCardFixture(t, noCaption, true)
	code, _ := plain.Find("hello")
	if code.W != 316 {
		t.Fatalf("code width without caption %v", code.W)
	}
}
