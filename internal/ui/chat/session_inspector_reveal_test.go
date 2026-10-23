package chat

import (
	"encoding/json"
	"fmt"
	"image/color"
	"image/png"
	"math"
	"os"
	"testing"
	"time"

	"github.com/egoist/mygo/ui"
	"whaleisle/internal/store/session"
	"whaleisle/internal/store/settings"
	"whaleisle/internal/ui/chrome"
	"whaleisle/internal/ui/theme"
)

func inspectorRevealFixture(t *testing.T, afterPaint ...func()) (*ui.Tester, *InspectorNavigation, *ui.ScrollState, *ui.ListState, *Model) {
	t.Helper()
	n := &InspectorNavigation{SessionID: "s"}
	list, scroll := &ui.ListState{FollowEnd: true}, &ui.ScrollState{}
	m := &Model{Inspector: n, List: list, Scroll: scroll, Snapshot: session.ViewState{SelectedID: "s", PlanHistoryBinding: "h"}, ThemeID: theme.IDDeepseekLight}
	for i := range 40 {
		m.Snapshot.Messages = append(m.Snapshot.Messages, session.Message{ID: fmt.Sprint(i), Seq: float64(i + 1)})
	}
	tt := ui.NewTester(func(c *ui.Context) {
		list.Key = func(i int) any { return m.Snapshot.Messages[i].ID }
		prepareSessionInspector(*m, m.Snapshot.Messages, nil, list)
		if len(afterPaint) > 1 {
			afterPaint[1]()
		}
		ui.Column(c).Fill().Background(ui.RGB(255, 255, 255)).Children(func() {
			ui.Box(c).Height(40).FillWidth()
			view := ui.List(c, list, len(m.Snapshot.Messages)).Fill().TrackScroll(scroll)
			inspectorChatViewport(c, *m, view, list)
			view.Rows(func(row ui.ListRow) {
				id := m.Snapshot.Messages[row.Index].ID
				box := ui.Column(row.Context).Height(60).FillWidth().Padding(0, 20)
				inspectorChatRowGeometry(row.Context, *m, m.Snapshot.Messages[row.Index], box, 0, 20, 0, 20)
				box.Children(func() { ui.Button(row.Context, "row "+id).Height(30).Width(90) })
			})
			ui.Box(c).Height(60).FillWidth().Background(ui.RGB(255, 255, 255))
		})
		inspectorChatRevealOverlay(c, *m, theme.Default().Light)
		if len(afterPaint) > 0 {
			ui.Overlay(c, func() {
				ui.Box(c.Key("test-before-next-build")).Absolute().Left(0).Top(0).Right(0).Bottom(0).PassThrough().DrawOver(func(_ *ui.Painter, _ ui.Rect) { afterPaint[0]() })
			})
		}
	}, 300, 360)
	t.Cleanup(n.CancelReveal)
	tt.SetTime(time.Unix(7000, 0))
	return tt, n, scroll, list, m
}

func TestInspectorRevealClipsPartialAndDoesNotChangeLayout(t *testing.T) {
	tt, n, scroll, list, _ := inspectorRevealFixture(t)
	list.ScrollToOffset(100)
	tt.Frame()
	before, _ := tt.Find("row 2")
	n.Target = &InspectorTarget{ID: "1"}
	tt.Frame()
	want := ui.Rect{X: 20, Y: 40, W: 260, H: 20}
	if n.reveal.rect != want || n.reveal.started.IsZero() || n.reveal.pending || scroll.Y != 100 || !list.FollowEnd {
		t.Fatalf("partial visible reveal moved or failed: rect=%+v mode=%+v scroll=%+v", n.reveal.rect, n.reveal, scroll)
	}
	after, _ := tt.Find("row 2")
	if before != after || tt.HasText("Inspector target") || tt.HasText("已定位检查器节点") {
		t.Fatal("reveal changed transcript layout")
	}
	// A pass-through overlay must leave the row's button reachable.
	if err := tt.Click("row 2"); err != nil {
		t.Fatal(err)
	}
	fixed := n.reveal.rect
	tt.Scroll(250, 200, 0, 30)
	if n.reveal.rect != fixed {
		t.Fatal("active flash moved with scrolling")
	}
}

func TestInspectorRevealColdOwnerUsesCurrentVisibleGeometry(t *testing.T) {
	tt, _, scroll, list, m := inspectorRevealFixture(t)
	list.ScrollToOffset(100)
	tt.Frame()
	n := &InspectorNavigation{SessionID: "s", Target: &InspectorTarget{ID: "1"}}
	m.Inspector = n
	tt.Frame()
	if scroll.Y != 100 || n.reveal.rect != (ui.Rect{X: 20, Y: 40, W: 260, H: 20}) || n.reveal.started.IsZero() {
		t.Fatalf("cold owner moved visible target: scroll=%+v reveal=%+v", scroll, n.reveal)
	}
}

func TestInspectorDeferredCenterResolvesCurrentKeyAndCancellation(t *testing.T) {
	for _, action := range []string{"cancel", "replace", "prepend"} {
		t.Run(action, func(t *testing.T) {
			var mutate func()
			tt, n, scroll, list, m := inspectorRevealFixture(t, func() {
				if mutate != nil {
					mutate()
				}
			})
			list.ScrollToOffset(100)
			tt.Frame()
			mutate = func() {
				if !n.reveal.centerQueued {
					return
				}
				mutate = nil
				switch action {
				case "cancel":
					n.CancelReveal()
				case "replace":
					n.Target = &InspectorTarget{ID: "1"}
				case "prepend":
					var before []session.Message
					for i := range 10 {
						before = append(before, session.Message{ID: fmt.Sprintf("older-%d", i)})
					}
					m.Snapshot.Messages = append(before, m.Snapshot.Messages...)
				}
			}
			n.Target = &InspectorTarget{ID: "25"}
			tt.Frame()
			if action == "prepend" {
				tt.SetTime(time.Unix(7000, 0).Add(2 * time.Second))
				if scroll.Y != 2000 {
					t.Fatalf("prepend centered old ordinal: %+v", scroll)
				}
			} else if scroll.Y != 100 {
				t.Fatalf("%s leaked old center request: %+v", action, scroll)
			}
		})
	}
}

func TestInspectorCenterCancellationAfterPreparationFreezesBeforeLayout(t *testing.T) {
	var cancel func()
	tt, n, scroll, list, _ := inspectorRevealFixture(t, func() {}, func() {
		if cancel != nil {
			cancel()
		}
	})
	list.ScrollToOffset(100)
	tt.Frame()
	cancel = func() {
		if n.reveal.centerList != nil {
			n.CancelReveal()
			cancel = nil
		}
	}
	n.Target = &InspectorTarget{ID: "25"}
	tt.Frame()
	if scroll.Y != 100 || n.reveal.pending || n.Highlight != "" {
		t.Fatalf("cancelled queued center reached layout: scroll=%+v reveal=%+v", scroll, n.reveal)
	}
}

func TestInspectorCenterReprojectionAfterPreparationUsesFinalOrdinal(t *testing.T) {
	var mutate func()
	tt, n, scroll, list, m := inspectorRevealFixture(t, func() {}, func() {
		if mutate != nil {
			mutate()
		}
	})
	list.ScrollToOffset(100)
	tt.Frame()
	mutate = func() {
		if n.reveal.centerList == nil {
			return
		}
		mutate = nil
		var before []session.Message
		for i := range 10 {
			before = append(before, session.Message{ID: fmt.Sprintf("older-%d", i)})
		}
		m.Snapshot.Messages = append(before, m.Snapshot.Messages...)
		prepareSessionInspector(*m, m.Snapshot.Messages, nil, list)
	}
	n.Target = &InspectorTarget{ID: "25"}
	tt.Frame()
	tt.SetTime(time.Unix(7000, 0).Add(2 * time.Second))
	if scroll.Y != 2000 {
		t.Fatalf("final build did not resolve target ordinal: %+v", scroll)
	}
}

func TestInspectorRevealAnimatesOffscreenThenFlashesAtScrollEnd(t *testing.T) {
	tt, n, scroll, list, _ := inspectorRevealFixture(t)
	list.ScrollToOffset(100)
	tt.Frame()
	n.Target = &InspectorTarget{ID: "25"}
	tt.Frame()
	if scroll.Y != 100 || !n.reveal.started.IsZero() || !n.reveal.waiting || !list.ScrollAnimating() {
		t.Fatalf("offscreen center/settlement: scroll=%+v reveal=%+v", scroll, n.reveal)
	}
	now := time.Unix(7000, 0)
	duration := inspectorChatScrollDuration(1300)
	tt.SetTime(now.Add(duration / 2))
	if !n.reveal.started.IsZero() || scroll.Y <= 100 || scroll.Y >= 1400 {
		t.Fatalf("not animating through intermediate offset: %+v", scroll)
	}
	tt.SetTime(now.Add(duration))
	if !n.reveal.started.IsZero() {
		t.Fatal("flash started in final scrolling frame")
	}
	tt.SetTime(now.Add(duration + time.Millisecond))
	tt.SetTime(now.Add(duration + 2*time.Millisecond))
	if n.reveal.started.IsZero() || n.reveal.rect != (ui.Rect{X: 20, Y: 140, W: 260, H: 60}) {
		t.Fatalf("final centered rectangle not used: %+v", n.reveal)
	}
	tt.SetTime(now.Add(duration + 702*time.Millisecond))
	if !n.reveal.started.IsZero() || n.Highlight != "" {
		t.Fatal("expired flash retained")
	}
	tt.Scroll(250, 200, 0, 20)
	if !n.reveal.started.IsZero() {
		t.Fatal("old flash revived")
	}
}

func TestInspectorRevealReplacementReducedMotionAndOwnership(t *testing.T) {
	tt, n, scroll, list, m := inspectorRevealFixture(t)
	list.ScrollToOffset(100)
	tt.Frame()
	n.Target = &InspectorTarget{ID: "2"}
	tt.Frame()
	old := n.reveal.started
	tt.SetTime(old.Add(500 * time.Millisecond))
	n.Target = &InspectorTarget{ID: "2"}
	tt.Frame()
	if !n.reveal.started.Equal(old.Add(500 * time.Millisecond)) {
		t.Fatal("same target replacement did not restart")
	}
	tt.SetTime(old.Add(700 * time.Millisecond))
	if n.reveal.started.IsZero() {
		t.Fatal("previous completion removed replacement")
	}
	n.Picking = true
	tt.Frame()
	if !n.reveal.started.IsZero() || n.reveal.pending {
		t.Fatal("picker did not dispose reveal")
	}
	n.Picking = false
	tt.SetPreferences(ui.Preferences{ReduceMotion: true, TextScale: 1})
	n.Target = &InspectorTarget{ID: "25"}
	tt.Frame()
	if scroll.Y != 1400 || !n.reveal.started.IsZero() || n.reveal.pending {
		t.Fatal("reduced motion did not center silently")
	}
	tt.SetPreferences(ui.Preferences{TextScale: 1})
	n.Target = &InspectorTarget{ID: "25"}
	tt.Frame()
	if n.reveal.started.IsZero() {
		t.Fatal("visible normal request did not flash")
	}
	m.DisableDeveloperTools = true
	tt.Frame()
	if !n.reveal.started.IsZero() || n.Highlight != "" {
		t.Fatal("disabled inspector retained flash")
	}
	m.DisableDeveloperTools = false
	m.Snapshot.SelectedID = "other"
	n.Target = &InspectorTarget{ID: "1"}
	tt.Frame()
	if n.Target != nil || !n.reveal.started.IsZero() {
		t.Fatal("session switch retained navigation")
	}
}

func TestInspectorRevealMissingOrRemovedTargetUsesVisibleFlow(t *testing.T) {
	tt, n, _, list, m := inspectorRevealFixture(t)
	list.ScrollToOffset(100)
	tt.Frame()
	n.Target = &InspectorTarget{ID: "missing"}
	tt.Frame()
	if n.reveal.rect != (ui.Rect{X: 16, Y: 40, W: 268, H: 260}) || n.Notice != "" {
		t.Fatalf("missing target did not reveal flow: %+v notice=%s", n.reveal, n.Notice)
	}
	n.Target = &InspectorTarget{ID: "25"}
	tt.Frame()
	m.Snapshot.Messages = m.Snapshot.Messages[:20]
	tt.Frame()
	tt.SetTime(time.Unix(7000, 0).Add(150 * time.Millisecond))
	if n.reveal.rect != (ui.Rect{X: 16, Y: 40, W: 268, H: 260}) {
		t.Fatalf("removed target did not reveal flow: %+v", n.reveal)
	}
}

func TestInspectorMissingTargetHighlightsOnlyNaturalShortFlow(t *testing.T) {
	tt, n, _, _, m := inspectorRevealFixture(t)
	m.Snapshot.Messages = m.Snapshot.Messages[:1]
	tt.Frame()
	n.Target = &InspectorTarget{ID: "missing"}
	tt.Frame()
	if n.reveal.rect != (ui.Rect{X: 16, Y: 40, W: 268, H: 60}) {
		t.Fatalf("short flow highlighted empty viewport: %+v", n.reveal)
	}
}

func TestInspectorRevealNativePulseMatchesFourLinearKeyframes(t *testing.T) {
	tt, n, _, list, _ := inspectorRevealFixture(t)
	list.ScrollToOffset(100)
	tt.Frame()
	n.Target = &InspectorTarget{ID: "2"}
	tt.Frame()
	start := n.reveal.started
	var evidence []map[string]any
	for _, ms := range []int{0, 117, 233, 350, 467, 583, 700} {
		tt.SetTime(start.Add(time.Duration(ms) * time.Millisecond))
		alpha := inspectorChatFlashOpacity(time.Duration(ms) * time.Millisecond)
		pixel := tt.Image().RGBAAt(250, 90)
		fill := chrome.Color(theme.Default().Light.BubbleHighlight)
		for i, channel := range []uint8{pixel.R, pixel.G, pixel.B} {
			want := float64([]uint8{fill.R, fill.G, fill.B}[i])*float64(alpha) + 255*(1-float64(alpha))
			if math.Abs(float64(channel)-want) > 1.5 {
				t.Fatalf("%dms opacity%v pixel%v channel%d want%v", ms, alpha, pixel, i, want)
			}
		}
		evidence = append(evidence, map[string]any{"ms": ms, "opacity": alpha, "pixel": []int{int(pixel.R), int(pixel.G), int(pixel.B)}})
		if prefix := os.Getenv("INSPECTOR_CHAT_REVEAL_EVIDENCE"); prefix != "" {
			f, err := os.Create(fmt.Sprintf("%s-%d.png", prefix, ms))
			if err != nil {
				t.Fatal(err)
			}
			err = png.Encode(f, tt.Image())
			_ = f.Close()
			if err != nil {
				t.Fatal(err)
			}
		}
	}
	if prefix := os.Getenv("INSPECTOR_CHAT_REVEAL_EVIDENCE"); prefix != "" {
		raw, _ := json.MarshalIndent(evidence, "", "  ")
		if err := os.WriteFile(prefix+"-state.json", raw, 0600); err != nil {
			t.Fatal(err)
		}
	}
	if tt.Image().RGBAAt(250, 90) != (color.RGBA{255, 255, 255, 255}) {
		t.Fatal("expired overlay still drawn")
	}
}

func TestInspectorRevealOpensTurnGroupAndReasoningBody(t *testing.T) {
	n := &InspectorNavigation{SessionID: "s"}
	expanded := map[string]bool{}
	coordinate := []byte(`{"data":{"turn":1,"step":1}}`)
	m := Model{Inspector: n, List: &ui.ListState{FollowEnd: true}, ThinkingExpanded: &expanded, ThemeID: theme.IDDeepseekLight, Preferences: settings.ChatPreferences{TranscriptView: "standard"}, Snapshot: session.ViewState{SelectedID: "s", PlanHistoryBinding: "h", Messages: []session.Message{
		{ID: "reason", Seq: 2, Role: "assistant", Kind: session.MessageKindThinking, Body: "Reasoning body\n\nHidden reasoning details.", Raw: coordinate},
		{ID: "answer", Seq: 3, Role: "assistant", Kind: session.MessageKindText, Body: "answer", Raw: coordinate},
		{ID: "end", Seq: 4, Role: "status", Kind: session.MessageKindText, EventType: "turn/end", Raw: coordinate},
	}}}
	tt := ui.NewTester(func(c *ui.Context) {
		ui.Column(c).Fill().Background(ui.RGB(255, 255, 255)).Children(func() { View(c, m) })
	}, 600, 500)
	tt.SetTime(time.Unix(7000, 0))
	if tt.HasText("Hidden reasoning details.") {
		t.Fatal("reasoning did not begin folded")
	}
	n.Target = &InspectorTarget{ID: "reason"}
	tt.Frame()
	if !expanded["transcript-turn:h:s:1"] || !expanded["transcript-group:h:s:reason:2"] || !expanded["reason"] || !tt.HasText("Hidden reasoning details.") {
		t.Fatalf("nested disclosure did not reveal body: choices=%v texts=%v", expanded, tt.Texts())
	}
	if n.Target != nil {
		t.Fatal("nested request remained pending")
	}
	tt.SetTime(time.Unix(7000, 0).Add(150 * time.Millisecond))
	if n.reveal.started.IsZero() || n.reveal.rect.H <= 24 {
		t.Fatalf("reasoning body not highlighted: %+v", n.reveal)
	}
}

func TestInspectorEmptyChatDisposesRevealBeforeNextArrival(t *testing.T) {
	n := &InspectorNavigation{SessionID: "s", Target: &InspectorTarget{ID: "hidden"}}
	m := Model{Inspector: n, List: &ui.ListState{FollowEnd: true}, Snapshot: session.ViewState{SelectedID: "s", Messages: []session.Message{{ID: "hidden", Kind: session.MessageKindSystemPrompt}}}}
	tt := ui.NewTester(func(c *ui.Context) { View(c, m) }, 600, 400)
	if n.Target != nil || n.reveal.pending || !n.reveal.started.IsZero() {
		t.Fatal("empty Chat retained reveal")
	}
	m.Snapshot.Messages = append(m.Snapshot.Messages, session.Message{ID: "new", Role: "user", Kind: session.MessageKindText, Body: "Unrelated new input"})
	tt.Frame()
	if n.reveal.pending || !n.reveal.started.IsZero() {
		t.Fatal("unrelated arrival revived empty Chat reveal")
	}
}

func TestInspectorRevealActualChatHasNoExtraRowAndKeepsVisibleTallMessage(t *testing.T) {
	n := &InspectorNavigation{SessionID: "s"}
	list, scroll := &ui.ListState{FollowEnd: true}, &ui.ScrollState{}
	body := ""
	for range 50 {
		body += "Reasoned answer with enough lines to exceed the viewport.\n\n"
	}
	m := Model{Inspector: n, List: list, Scroll: scroll, ThemeID: theme.IDDeepseekLight, SuppressStatus: true, FullColumnViewport: true, ContentWidth: 400, Preferences: settings.ChatPreferences{TranscriptView: "verbose"}, Snapshot: session.ViewState{SelectedID: "s", PlanHistoryBinding: "h", Messages: []session.Message{{ID: "tall", Kind: session.MessageKindText, Role: "assistant", Body: body}, {ID: "later", Kind: session.MessageKindText, Role: "user", Body: "later"}}}}
	tt := ui.NewTester(func(c *ui.Context) {
		ui.Column(c).Fill().Background(ui.RGB(255, 255, 255)).Children(func() { View(c, m) })
	}, 600, 400)
	tt.SetTime(time.Unix(7000, 0))
	list.ScrollToOffset(200)
	tt.Frame()
	before := scroll.Y
	n.Target = &InspectorTarget{ID: "tall"}
	tt.Frame()
	if before != scroll.Y || n.reveal.started.IsZero() || n.reveal.rect.X < 32 || n.reveal.rect.W > 400 {
		t.Fatalf("actual Chat reveal moved tall visible node or used gutter: before%v scroll=%+v rect=%+v", before, scroll, n.reveal.rect)
	}
	if tt.HasText("Inspector target") || tt.HasText("已定位检查器节点") {
		t.Fatal("persistent label remains")
	}
	m.GroupRoom = true
	tt.Frame()
	if n.reveal.pending || !n.reveal.started.IsZero() {
		t.Fatal("group conversation retained main Chat reveal")
	}
}
