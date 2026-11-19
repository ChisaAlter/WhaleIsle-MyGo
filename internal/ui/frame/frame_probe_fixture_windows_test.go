//go:build windows && inspectorfixture

package frame

import (
	"encoding/json"
	"fmt"
	"sync"
	"time"

	"github.com/egoist/mygo"
	"github.com/egoist/mygo/ui"
	"whaleisle/internal/store/session"
	"whaleisle/internal/ui/chat"
	"whaleisle/internal/ui/chrome"
	"whaleisle/internal/ui/i18n"
	"whaleisle/internal/ui/theme"
)

// The frame probe's subject: the chat view over recorded messages — the same
// view a real window shows and the headless harness renders. Two hosts drive
// it. scripts/frame-probe.ps1 posts keys and a wheel into a real window, which
// measures what the compositor does with the frames; TestFrameProbe renders in
// memory, which measures what the frames cost with no display in the way.
type frameProbeModel struct {
	mu       sync.Mutex
	messages []session.Message
	expanded map[string]bool
	dark     bool
	picker   string
	width    int
	height   int
	// onChange asks the host for a frame, as a streamed token does; nil in
	// the headless harness, which advances frames itself.
	onChange  func()
	streaming bool
	streamEnd chan struct{}
	list      *ui.ListState
	scroll    *ui.ScrollState
}

func newFrameProbeModel() *frameProbeModel {
	store := session.New()
	store.Select("fixture")
	for _, frame := range []string{
		`{"type":"turn/start","seq":1,"data":{"turn":0}}`,
		`{"type":"step/start","seq":2,"data":{"turn":0,"step":0}}`,
		`{"type":"tool/call","seq":10,"data":{"turn":0,"step":0,"callId":"root","name":"functions.pwsh","arguments":"{}"}}`,
		`{"type":"tool/result","seq":20,"surfaceOp":"append","data":{"turn":0,"step":0,"message":{"id":"result","role":"tool","toolCallId":"root","source":{"kind":"tool","callId":"root"},"content":[{"type":"text","text":"root output\nsecond line of output\nthird line of output\nfourth line"}]}}}`,
		`{"type":"step/end","seq":21,"data":{"turn":0,"step":0}}`,
		`{"type":"step/start","seq":22,"data":{"turn":0,"step":1}}`,
		`{"type":"assistant/message","seq":40,"surfaceOp":"append","data":{"turn":0,"step":1,"message":{"id":"m40","role":"assistant","content":[{"type":"reasoning","text":"Reasoning details."},{"type":"text","text":"Canonical answer."}]},"stream":[{"type":"chunk","time":1,"chunk":{"type":"block-start","index":4,"blockType":"reasoning"}},{"type":"reasoning-chunks","index":4,"time0":2,"dt":[],"texts":["Reasoning details."]},{"type":"chunk","time":3,"chunk":{"type":"block-start","index":9,"blockType":"text"}},{"type":"text-chunks","index":9,"time0":4,"dt":[],"texts":["Canonical answer."]}]}}`,
		`{"type":"step/end","seq":41,"data":{"turn":0,"step":1}}`,
		`{"type":"turn/end","seq":42,"data":{"turn":0}}`,
	} {
		store.ApplyInspectorFixtureFrame(json.RawMessage(`{"type":"event","event":` + frame + `}`))
	}
	var messages []session.Message
	for i := 1; i <= 30; i++ {
		messages = append(messages, session.Message{ID: fmt.Sprintf("answer-%d", i), Seq: float64(100 + i), Role: "assistant", Kind: session.MessageKindText, Body: fmt.Sprintf("Probe answer %d\n\nRecorded answer content the probe scrolls through.", i)})
	}
	messages = append(messages, store.Snapshot().Messages...)
	return &frameProbeModel{messages: messages, expanded: map[string]bool{}, picker: "deepseek", width: 1280, height: 720, list: &ui.ListState{FollowEnd: true}, scroll: &ui.ScrollState{}}
}

// appendStreamToken adds the text one streamed token carries, as the host does
// between frames.
func (m *frameProbeModel) appendStreamToken() {
	m.mu.Lock()
	last := len(m.messages) - 1
	m.messages[last].Body += " token"
	m.mu.Unlock()
	if m.onChange != nil {
		m.onChange()
	}
}

// toggleExpanded opens or closes the details of the tool call, and reports
// what it opened.
func (m *frameProbeModel) toggleExpanded() bool {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.expanded["result"] = !m.expanded["result"]
	return m.expanded["result"]
}

func (m *frameProbeModel) setDark(dark bool) {
	m.mu.Lock()
	m.dark = dark
	m.mu.Unlock()
}

// streamingEvery spreads one token per interval over the frame the host draws,
// as a model streams into an open answer.
const streamInterval = 16 * time.Millisecond

func (m *frameProbeModel) startStreaming() {
	m.mu.Lock()
	if m.streaming {
		m.mu.Unlock()
		return
	}
	m.streaming, m.streamEnd = true, make(chan struct{})
	end := m.streamEnd
	m.mu.Unlock()
	go func() {
		ticker := time.NewTicker(streamInterval)
		defer ticker.Stop()
		for {
			select {
			case <-end:
				return
			case <-ticker.C:
				m.appendStreamToken()
			}
		}
	}()
}

func (m *frameProbeModel) stopStreaming() {
	m.mu.Lock()
	if m.streaming && m.streamEnd != nil {
		close(m.streamEnd)
		m.streamEnd = nil
	}
	m.streaming = false
	m.mu.Unlock()
}

// view is the view both hosts render. keys adds the probe's own shortcuts,
// which only the window host posts.
func (m *frameProbeModel) view(keys bool) func(c *ui.Context) {
	list, scroll := m.list, m.scroll
	return func(c *ui.Context) {
		theme.ApplyInterface(c)
		m.mu.Lock()
		shown := append([]session.Message(nil), m.messages...)
		expanded := make(map[string]bool, len(m.expanded))
		for k, v := range m.expanded {
			expanded[k] = v
		}
		isDark, picker := m.dark, m.picker
		m.mu.Unlock()
		palette := theme.Default().Light
		themeID := theme.IDDeepseekLight
		if isDark {
			palette, themeID = theme.Default().Dark, theme.IDDeepseekDark
		}
		if keys {
			switch {
			case c.Shortcut(0, ui.KeyF1):
				ui.FrameTraceMark("scroll")
			case c.Shortcut(0, ui.KeyF2):
				ui.FrameTraceMark("stream")
				m.startStreaming()
			case c.Shortcut(0, ui.KeyF3):
				if m.toggleExpanded() {
					ui.FrameTraceMark("expand_open")
				} else {
					ui.FrameTraceMark("expand_close")
				}
			case c.Shortcut(0, ui.KeyF4):
				ui.FrameTraceMark("picker")
			case c.Shortcut(0, ui.KeyF5):
				if isDark {
					ui.FrameTraceMark("theme_light")
				} else {
					ui.FrameTraceMark("theme_dark")
				}
				m.setDark(!isDark)
			case c.Shortcut(0, ui.KeyF6):
				ui.FrameTraceMark("resize")
			case c.Shortcut(0, ui.KeyF7):
				ui.FrameTraceMark("stream_stop")
				m.stopStreaming()
			}
		}
		ui.Row(c).Fill().Background(chrome.Color(palette.BG)).Children(func() {
			ui.Column(c).Grow(1).MinWidth(0).FillHeight().Children(func() {
				chat.View(c, chat.Model{
					List: list, Scroll: scroll, ThinkingExpanded: &expanded, ThemeID: themeID, SuppressStatus: true, FullColumnViewport: true, ContentWidth: 640,
					Snapshot: session.ViewState{SelectedID: "fixture", Messages: shown},
				})
				ui.Box(c).Height(80).FillWidth().Background(chrome.Color(palette.Field)).Children(func() {
					ui.Text(c, "Composer seat")
				})
			})
			ui.Column(c).Width(240).FillHeight().Children(func() {
				ui.Combobox(c.Key("probe-picker"), &picker, []string{"deepseek", "another-model", "third-model"}).Label("Probe picker")
			})
		})
	}
}

// pickerX is where the probe's picker sits inside the window, in DIPs from its
// right edge: the window host clicks there, and the headless harness does too.
const pickerX = 120

// runFrameProbeFixture shows the probe view in a real window for
// scripts/frame-probe.ps1 to drive. Each interaction is one posted key, which
// marks MYGO_FRAME_TRACE so the frames of one interaction can be told from the
// next:
//
//	F1 mark the start of a wheel scroll  (the probe posts the wheel itself)
//	F2 start streaming text              (a token every streamInterval)
//	F3 open, then close, the tool details
//	F4 mark the start of a picker click  (the probe clicks the picker)
//	F5 switch the palette
//	F6 mark the start of a resize        (the probe resizes the window)
//	F7 stop streaming
func runFrameProbeFixture() {
	i18n.SetLocale(i18n.LocaleEN)
	theme.SetPreferences(theme.Defaults())
	model := newFrameProbeModel()
	var win *mygo.Window
	model.onChange = func() {
		if win != nil {
			win.Invalidate()
		}
	}
	mygo.App.WhenReady(func() {
		win = mygo.NewWindow(mygo.WindowOptions{Title: "MyGo frame probe fixture", Width: model.width, Height: model.height, UseContentSize: true, Content: ui.View(model.view(true)), Frameless: true, BackgroundColor: "#ffffff"})
	})
	if err := mygo.App.Run(); err != nil {
		panic(err)
	}
	model.stopStreaming()
}
