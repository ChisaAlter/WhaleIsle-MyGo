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
			tt.SetTime(time.Unix(12300, 0).Add(3*time.Second))
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
