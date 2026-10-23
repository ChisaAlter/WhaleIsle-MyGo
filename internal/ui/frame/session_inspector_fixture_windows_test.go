//go:build windows && inspectorfixture

package frame

import (
	"encoding/json"
	"fmt"
	"os"
	"testing"
	"time"

	"github.com/egoist/mygo"
	"github.com/egoist/mygo/ui"
	"whaleisle/internal/store/session"
	"whaleisle/internal/ui/chat"
	"whaleisle/internal/ui/chrome"
	"whaleisle/internal/ui/i18n"
	"whaleisle/internal/ui/theme"
)

// Opt-in desktop fixture, using the same raw records as the Electron probe.
// Build: go test -tags inspectorfixture -c -o .devdata/inspector-fixture.exe ./internal/ui/frame
// Set INSPECTOR_NATIVE_FIXTURE to the absolute testdata/inspector-records.json path.
func TestMain(m *testing.M) {
	path := os.Getenv("INSPECTOR_NATIVE_FIXTURE")
	if path == "" {
		os.Exit(m.Run())
	}
	raw, err := os.ReadFile(path)
	if err != nil {
		panic(err)
	}
	var records []struct {
		Key, Parent, Type string
		Seq, Time         float64
		Value             json.RawMessage
	}
	if err = json.Unmarshal(raw, &records); err != nil {
		panic(err)
	}
	source := session.InspectorSnapshot{Binding: "round102-fixture", SessionID: "fixture"}
	for _, r := range records {
		source.Records = append(source.Records, session.InspectorRecord{Key: r.Key, Parent: r.Parent, Type: r.Type, Seq: r.Seq, Time: r.Time, Raw: r.Value})
	}
	var phases []struct {
		Name   string
		Frames []json.RawMessage
	}
	phase := 0
	var advance func()
	if scenario := os.Getenv("INSPECTOR_NATIVE_FIXTURE_SCENARIO"); scenario == "stream" || scenario == "scroll" {
		if err = json.Unmarshal(raw, &phases); err != nil {
			panic(err)
		}
		store := session.New()
		store.Select("fixture")
		advance = func() {
			if phase >= len(phases) {
				return
			}
			for _, frame := range phases[phase].Frames {
				store.ApplyInspectorFixtureFrame(frame)
			}
			source = store.InspectorSnapshot()
			if prefix := os.Getenv("INSPECTOR_NATIVE_FIXTURE_OUTPUT"); prefix != "" {
				projection, _ := json.MarshalIndent(source.Records, "", "  ")
				if err := os.WriteFile(prefix+"-projection-"+phases[phase].Name+".json", projection, 0600); err != nil {
					panic(err)
				}
			}
			phase++
		}
		advance()
	}
	p := SessionInspectorPaneState{collapsed: map[string]inspectorDisclosure{}}
	i18n.SetLocale(i18n.LocaleEN)
	theme.SetPreferences(theme.Defaults())
	colors := theme.Default().Light
	if scenario := os.Getenv("INSPECTOR_NATIVE_FIXTURE_SCENARIO"); scenario == "chat-reveal" || scenario == "target-ownership" || scenario == "reverse-pick" {
		var messages []session.Message
		for i := 1; i <= 30; i++ {
			id := fmt.Sprintf("answer-%d", i)
			messages = append(messages, session.Message{ID: id, Seq: float64(i), Role: "assistant", Kind: session.MessageKindText, Body: fmt.Sprintf("Reveal answer %d\n\nRecorded answer content for a real Inspector to Chat navigation gesture.", i)})
		}
		list, scroll := &ui.ListState{FollowEnd: true}, &ui.ScrollState{}
		p.navigation.SessionID = "fixture"
		for i := range source.Records {
			source.Records[i].ChatID = fmt.Sprintf("answer-%.0f", source.Records[i].Seq)
		}
		expanded := map[string]bool{}
		if scenario == "target-ownership" || scenario == "reverse-pick" {
			store := session.New()
			store.Select("fixture")
			frames := []string{
				`{"type":"turn/start","seq":1,"data":{"turn":0}}`,
				`{"type":"step/start","seq":2,"data":{"turn":0,"step":0}}`,
				`{"type":"tool/call","seq":10,"data":{"turn":0,"step":0,"callId":"root","name":"functions.pwsh","arguments":"{}"}}`,
				`{"type":"tool/ptc-dispatch-start","seq":11,"data":{"rootCallId":"root","parentCallId":"root","subCallId":"child","name":"functions.read","arguments":{}}}`,
				`{"type":"tool/ptc-dispatch","seq":19,"data":{"rootCallId":"root","parentCallId":"root","subCallId":"child","name":"functions.read","arguments":{},"result":"child done"}}`,
				`{"type":"tool/result","seq":20,"surfaceOp":"append","data":{"turn":0,"step":0,"message":{"id":"result","role":"tool","toolCallId":"root","source":{"kind":"tool","callId":"root"},"content":[{"type":"text","text":"root output"}]}}}`,
				`{"type":"step/end","seq":21,"data":{"turn":0,"step":0}}`,
				`{"type":"step/start","seq":22,"data":{"turn":0,"step":1}}`,
				`{"type":"assistant/message","seq":40,"surfaceOp":"append","data":{"turn":0,"step":1,"message":{"id":"m40","role":"assistant","content":[{"type":"reasoning","text":"Reasoning details."},{"type":"text","text":"Canonical answer."},{"type":"text","text":"Second answer block."}]},"stream":[{"type":"chunk","time":1,"chunk":{"type":"block-start","index":4,"blockType":"reasoning"}},{"type":"reasoning-chunks","index":4,"time0":2,"dt":[],"texts":["Reasoning details."]},{"type":"chunk","time":3,"chunk":{"type":"block-start","index":9,"blockType":"text"}},{"type":"text-chunks","index":9,"time0":4,"dt":[],"texts":["Canonical answer."]}]}}`,
				`{"type":"step/end","seq":41,"data":{"turn":0,"step":1}}`,
				`{"type":"turn/end","seq":42,"data":{"turn":0}}`,
			}
			for _, frame := range frames {
				store.ApplyInspectorFixtureFrame(json.RawMessage(`{"type":"event","event":` + frame + `}`))
			}
			source = store.InspectorSnapshot()
			messages = store.Snapshot().Messages
			p.navigation.SessionID = "fixture"
			if scenario == "target-ownership" {
				for _, row := range source.Chat {
					if row.Key == "chat:m40" || row.Key == "chat:m40/text/2" {
						row.Type = row.Key
						source.Records = append(source.Records, row)
					}
				}
			} else {
				store.ApplyInspectorFixtureFrame(json.RawMessage(`{"type":"event","event":{"type":"user/message","seq":0,"surfaceOp":"append","data":{"id":"zero-user","role":"user","source":{"kind":"user"},"content":[{"type":"text","text":"Sequence zero message."}]}}}`))
				source = store.InspectorSnapshot()
				messages = store.Snapshot().Messages
				list.FollowEnd = false
			}
		}
		prefix := os.Getenv("INSPECTOR_NATIVE_FIXTURE_OUTPUT")
		trace := os.Getenv("INSPECTOR_NATIVE_FIXTURE_TRACE") == "1"
		writeMetrics := func(suffix string, now time.Time) {
			metrics, _ := json.MarshalIndent(map[string]any{"time": now.Format(time.RFC3339Nano), "scroll": scroll.Y, "max": scroll.MaxY, "highlight": p.navigation.Highlight, "targetPending": p.navigation.Target != nil, "reveal": chat.InspectorRevealFixtureState(&p.navigation), "expanded": expanded, "selected": p.selected, "mode": p.Mode, "picking": p.navigation.Picking, "notice": p.navigation.Notice, "filter": p.Filter, "collapsed": p.collapsed, "groups": p.navigation.Groups}, "", "  ")
			if err := os.WriteFile(prefix+suffix, metrics, 0600); err != nil {
				panic(err)
			}
		}
		view := ui.View(func(c *ui.Context) {
			theme.ApplyInterface(c)
			if c.Shortcut(0, ui.KeyF9) {
				writeMetrics("-metrics.json", time.Now())
			}
			if scenario == "reverse-pick" && c.Shortcut(0, ui.KeyF6) {
				p.navigation.Nodes, p.navigation.Groups = nil, nil
				p.changeInspectorMode("session-log", "fixture")
				expanded = map[string]bool{}
				list.ScrollToOffset(0)
			}
			ui.Row(c).Fill().Background(chrome.Color(colors.BG)).Children(func() {
				ui.Column(c).Grow(1).MinWidth(0).FillHeight().Children(func() {
					chat.View(c, chat.Model{Inspector: &p.navigation, List: list, Scroll: scroll, Snapshot: session.ViewState{SelectedID: "fixture", PlanHistoryBinding: "fixture", Messages: messages}, ThinkingExpanded: &expanded, ThemeID: theme.IDDeepseekLight, SuppressStatus: true, FullColumnViewport: true, ContentWidth: 420})
					ui.Box(c).Height(80).FillWidth().Background(chrome.Color(colors.Field)).Children(func() { ui.Text(c, "Composer seat") })
				})
				ui.Column(c).Width(680).FillHeight().Children(func() {
					current := source
					if scenario == "reverse-pick" {
						current.Chat = nil
						for _, msg := range p.navigation.Nodes {
							raw, _ := json.Marshal(msg)
							current.Chat = append(current.Chat, session.InspectorRecord{Key: "chat:" + msg.ID, Type: string(msg.Kind), Seq: msg.Seq, Raw: raw, ChatTarget: p.navigation.PickTarget(msg)})
						}
					}
					inspectorPresentation(c, &p, current, session.ViewState{}, colors, nil)
				})
			}).DrawOver(func(painter *ui.Painter, _ ui.Rect) {
				if trace {
					writeMetrics("-frame.json", painter.Now())
				}
			})
		})
		mygo.App.WhenReady(func() {
			mygo.NewWindow(mygo.WindowOptions{Title: "Native Inspector Chat reveal fixture", Width: 1280, Height: 720, UseContentSize: true, Content: view, Frameless: true, BackgroundColor: "#ffffff"})
		})
		if err = mygo.App.Run(); err != nil {
			panic(err)
		}
		return
	}
	view := ui.View(func(c *ui.Context) {
		theme.ApplyInterface(c)
		if advance != nil && c.Shortcut(0, ui.KeyF8) {
			advance()
		}
		if os.Getenv("INSPECTOR_NATIVE_FIXTURE_SCENARIO") == "scroll" {
			if c.Shortcut(0, ui.KeyF9) {
				prefix := os.Getenv("INSPECTOR_NATIVE_FIXTURE_OUTPUT")
				metrics, _ := json.MarshalIndent(map[string]any{"offset": p.tableY.Y, "max": p.tableY.MaxY, "padding": p.scroll.layout.bottomPadding, "mode": p.scroll.mode, "phase": phase, "viewportHeight": float32(len(p.scroll.layout.rows)+1)*inspectorRowHeight + p.scroll.layout.bottomPadding - p.tableY.MaxY}, "", "  ")
				if err := os.WriteFile(prefix+"-metrics.json", metrics, 0600); err != nil {
					panic(err)
				}
			}
		}
		if os.Getenv("INSPECTOR_NATIVE_FIXTURE_SCENARIO") == "interaction" {
			// Emulate the Chat-owned pick handoff without connecting a Host.
			if c.Shortcut(0, ui.KeyF6) {
				p.navigation.Picked = &chat.InspectorTarget{Seq: 4}
			}
			if c.Shortcut(0, ui.KeyF7) {
				p.navigation.Picked = &chat.InspectorTarget{Seq: 3}
			}
		}
		ui.Column(c).Fill().Background(chrome.Color(colors.BG)).Children(func() { inspectorPresentation(c, &p, source, session.ViewState{}, colors, nil) })
	})
	mygo.App.WhenReady(func() {
		mygo.NewWindow(mygo.WindowOptions{Title: "Native inspector fixture", Width: 780, Height: 600, UseContentSize: true, Content: view, Frameless: true, BackgroundColor: "#ffffff"})
	})
	if err = mygo.App.Run(); err != nil {
		panic(err)
	}
}
