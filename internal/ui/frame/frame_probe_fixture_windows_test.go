//go:build windows && inspectorfixture

package frame

import (
	"encoding/json"
	"fmt"
	"os"
	"strconv"
	"strings"
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

// The frame probe's subject: the chat view over a store's session — the same
// view a real window shows and the headless harness renders, reading the same
// kind of snapshot the application reads. Two hosts drive it.
// scripts/frame-probe.ps1 posts keys and a wheel into a real window, which
// measures what the compositor does with the frames; TestFrameProbe renders in
// memory, which measures what the frames cost with no display in the way.
//
// The view reads the store through a FrameSnapshot, as the application's render
// pass does, so an unchanged revision hands the same view back and the chat's
// own projection memo can do its work: a probe that built its ViewState by hand
// would measure the uncached path no window ever takes.
type frameProbeModel struct {
	mu    sync.Mutex
	store *session.Store
	frame *session.FrameSnapshot
	// streamed is the answer text the streaming run has accumulated, and
	// tokens counts the chunks it applied.
	streamed strings.Builder
	tokens   int
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

// messages is how many answers the probe's session holds: WHALEISLE_FRAME_PROBE_MESSAGES raises it, which is how the cost of a long transcript is measured.
var messages = func() int {
	if n, err := strconv.Atoi(os.Getenv("WHALEISLE_FRAME_PROBE_MESSAGES")); err == nil && n > 0 {
		return n
	}
	return 30
}()

func newFrameProbeModel() *frameProbeModel {
	store := session.New()
	store.Select("fixture")
	apply := func(frame string) {
		store.ApplyInspectorFixtureFrame(json.RawMessage(`{"type":"event","event":` + frame + `}`))
	}
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
		apply(frame)
	}
	for i := 1; i <= messages; i++ {
		body := fmt.Sprintf("Probe answer %d\n\nRecorded answer content the probe scrolls through.", i)
		base := 1000 + i*10
		apply(fmt.Sprintf(`{"type":"turn/start","seq":%d,"data":{"turn":%d}}`, base, i))
		apply(fmt.Sprintf(`{"type":"step/start","seq":%d,"data":{"turn":%d,"step":0}}`, base+1, i))
		apply(fmt.Sprintf(`{"type":"assistant/message","seq":%d,"surfaceOp":"append","data":{"turn":%d,"step":0,"message":{"id":"answer-%d","role":"assistant","content":[{"type":"text","text":%q}]}}}`, base+2, i, i, body))
		apply(fmt.Sprintf(`{"type":"step/end","seq":%d,"data":{"turn":%d,"step":0}}`, base+3, i))
		apply(fmt.Sprintf(`{"type":"turn/end","seq":%d,"data":{"turn":%d}}`, base+4, i))
	}
	return &frameProbeModel{
		store: store, frame: store.BeginFrame(), expanded: map[string]bool{}, picker: "deepseek",
		width: 1280, height: 720, list: &ui.ListState{FollowEnd: true}, scroll: &ui.ScrollState{},
	}
}

// view returns the snapshot the view draws, as the render pass reads it.
func (m *frameProbeModel) view() session.ViewState {
	if uncachedProbe {
		// The path a view takes when it carries no store revision: the chat's
		// projection memo cannot trust it, so every frame projects the whole
		// transcript again. WHALEISLE_FRAME_PROBE_UNCACHED measures that cost.
		return m.frame.Snapshot().WithoutProjectionKeys()
	}
	return m.frame.Snapshot()
}

// uncachedProbe makes the model hand the view a snapshot without its store
// revision, which no render pass of the application does.
var uncachedProbe = os.Getenv("WHALEISLE_FRAME_PROBE_UNCACHED") != ""

// appendStreamToken applies one more chunk of the answer to the store, as the
// host does between frames: a new revision, which is what makes the chat's
// projection work again.
func (m *frameProbeModel) appendStreamToken() {
	m.mu.Lock()
	m.streamed.WriteString(" token")
	text := m.streamed.String()
	m.tokens++
	revision := 5000 + m.tokens
	first := m.tokens == 1
	m.mu.Unlock()
	// A follow frame carries the attempt's whole stream so far, as the host
	// sends it: the store keeps one message and grows its text, which is what
	// makes this a streaming frame rather than an append.
	records := ""
	if first {
		records = `{"type":"event","event":{"type":"turn/start","seq":9001,"data":{"turn":90}}},` +
			`{"type":"event","event":{"type":"step/start","seq":9002,"data":{"turn":90,"step":1}}}`
	}
	m.store.ApplyInspectorFixtureFrame(json.RawMessage(fmt.Sprintf(
		`{"type":"snapshot","cursor":0,"hasMore":false,"records":[%s],"assistantStream":{"revision":%d,"activeAttempt":{"attemptId":"probe","turn":90,"step":1,"startedAfterSeq":9002,"nextIndex":9,"stream":[{"type":"text-chunks","index":9,"time0":1,"dt":[],"texts":[%q]}]}}}`,
		records, revision, text)))
	if m.onChange != nil {
		m.onChange()
	}
}

// streamedText reports the answer text the streaming run has applied.
func (m *frameProbeModel) streamedText() string {
	m.mu.Lock()
	defer m.mu.Unlock()
	return m.streamed.String()
}

// toggleExpanded opens or closes the details of the tool call, and reports what
// it opened.
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

// streamingEvery is how fast a token arrives while the probe streams.
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
func (m *frameProbeModel) viewFunc(keys bool) func(c *ui.Context) {
	return func(c *ui.Context) {
		theme.ApplyInterface(c)
		shown := m.view()
		m.mu.Lock()
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
					List: m.list, Scroll: m.scroll, ThinkingExpanded: &expanded, ThemeID: themeID, SuppressStatus: true, FullColumnViewport: true, ContentWidth: 640,
					Snapshot: shown,
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
		// Hidden, then shown without activation: the default show calls
		// SetForegroundWindow, which takes the operator's focus away from
		// whatever they are doing. The probe drives this window with posted
		// messages only, so it never needs to be the foreground window.
		win = mygo.NewWindow(mygo.WindowOptions{Title: "MyGo frame probe fixture", Width: model.width, Height: model.height, UseContentSize: true, Content: ui.View(model.viewFunc(true)), Frameless: true, BackgroundColor: "#ffffff", Hidden: true})
		win.OnReadyToShow(func() { win.ShowInactive() })
	})
	if err := mygo.App.Run(); err != nil {
		panic(err)
	}
	model.stopStreaming()
}
