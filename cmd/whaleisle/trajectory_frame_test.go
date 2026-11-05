package main

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"go/ast"
	"go/parser"
	"go/token"
	"io"
	"log"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	"github.com/coder/websocket"
	"github.com/coder/websocket/wsjson"
	"github.com/egoist/mygo/ui"
	desktopgit "whaleisle/internal/desktop/git"
	"whaleisle/internal/dsh"
	"whaleisle/internal/store/session"
	"whaleisle/internal/ui/chrome"
	"whaleisle/internal/ui/diffsurface"
	"whaleisle/internal/ui/frame"
	"whaleisle/internal/ui/i18n"
	"whaleisle/internal/ui/theme"
)

// Keep the integration below bound to mainView's real paint-local binding and
// writeback, without starting mainView's native windows and background services.
func TestMainConversationTabUsesPaintLocalWriteback(t *testing.T) {
	source, err := os.ReadFile("main.go")
	if err != nil {
		t.Fatal(err)
	}
	file, err := parser.ParseFile(token.NewFileSet(), "main.go", source, 0)
	if err != nil {
		t.Fatal(err)
	}
	local, passed, committed := false, false, false
	ast.Inspect(file, func(node ast.Node) bool {
		switch n := node.(type) {
		case *ast.AssignStmt:
			if len(n.Lhs) != 1 || len(n.Rhs) != 1 {
				break
			}
			if name, ok := n.Lhs[0].(*ast.Ident); ok && name.Name == "chatPaneTab" {
				local = local || trajectoryStateField(n.Rhs[0])
			}
			if trajectoryStateField(n.Lhs[0]) {
				name, ok := n.Rhs[0].(*ast.Ident)
				committed = committed || ok && name.Name == "chatPaneTab"
			}
		case *ast.KeyValueExpr:
			key, ok := n.Key.(*ast.Ident)
			if !ok || key.Name != "ChatPaneTab" {
				break
			}
			address, ok := n.Value.(*ast.UnaryExpr)
			if !ok || address.Op != token.AND {
				break
			}
			name, ok := address.X.(*ast.Ident)
			passed = passed || ok && name.Name == "chatPaneTab"
		}
		return true
	})
	if !local || !passed || !committed {
		t.Fatalf("mainView no longer uses the exercised tab binding: local=%v passed=%v committed=%v", local, passed, committed)
	}
}

func trajectoryStateField(expr ast.Expr) bool {
	field, ok := expr.(*ast.SelectorExpr)
	if !ok || field.Sel.Name != "chatPaneTab" {
		return false
	}
	owner, ok := field.X.(*ast.Ident)
	return ok && owner.Name == "state"
}

type trajectoryFrameTransport struct {
	captionListTransport
	goals atomic.Int32
}

func (f *trajectoryFrameTransport) RoundTrip(r *http.Request) (*http.Response, error) {
	if r.Method == http.MethodGet {
		return f.captionListTransport.RoundTrip(r)
	}
	body, err := io.ReadAll(r.Body)
	if err != nil {
		return nil, err
	}
	r.Body = io.NopCloser(bytes.NewReader(body))
	var request struct {
		Method string
		RPCID  string `json:"rpcId"`
	}
	if json.Unmarshal(body, &request) != nil || request.Method != "goals/get" {
		return f.captionListTransport.RoundTrip(r)
	}
	// Accepting a follow snapshot performs this ordinary optional-state read.
	// It is fixture setup, not a model request or a tab-switch prerequisite.
	f.calls.Add(1)
	f.goals.Add(1)
	response, err := json.Marshal(map[string]any{"type": "server-response", "rpcId": request.RPCID, "result": map[string]any{"ok": true, "value": nil}})
	if err != nil {
		return nil, err
	}
	return &http.Response{StatusCode: http.StatusOK, Header: http.Header{"Content-Type": []string{"application/json"}}, Body: io.NopCloser(bytes.NewReader(response))}, nil
}

// Use an owned loopback follow stream and the production Store projection. No
// native windows, desktop input, model requests or user profiles are involved.
func trajectoryFrameStore(t *testing.T, history bool) (*session.Store, *trajectoryFrameTransport) {
	t.Helper()
	store := session.New()
	store.Select("fixture")
	transport := &trajectoryFrameTransport{captionListTransport: captionListTransport{root: t.TempDir()}}
	store.Attach(&dsh.Client{Endpoint: dsh.Endpoint{Origin: "https://trajectory.fixture.invalid"}, HTTP: &http.Client{Transport: transport}}, nil)
	t.Cleanup(store.Detach)
	store.RefreshList(context.Background())
	listDeadline := time.Now().Add(3 * time.Second)
	for time.Now().Before(listDeadline) {
		snap := store.Snapshot()
		if !snap.ListLoading && snap.SelectedCwd == transport.root && snap.ListErr == "" {
			break
		}
		time.Sleep(time.Millisecond)
	}
	if store.Snapshot().SelectedCwd != transport.root {
		t.Fatal("owned list did not establish the selected cwd")
	}
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != dsh.MuxPath {
			http.NotFound(w, r)
			return
		}
		conn, err := websocket.Accept(w, r, nil)
		if err != nil {
			return
		}
		defer conn.CloseNow()
		for {
			var request dsh.ClientFrame
			if wsjson.Read(r.Context(), conn, &request) != nil {
				return
			}
			if request.Type != dsh.FrameOpen || request.Endpoint != "session/follow" {
				continue
			}
			value := json.RawMessage(`{"type":"snapshot","records":[],"hasMore":false}`)
			if history {
				value = json.RawMessage(`{"type":"snapshot","records":[{"type":"event","event":{"type":"user/message","seq":1,"data":{"id":"owned-user","content":[{"type":"text","text":"Owned trajectory history"}]}}},{"type":"event","event":{"type":"assistant/message","seq":2,"data":{"turn":1,"message":{"id":"owned-assistant","content":[{"type":"text","text":"Owned trajectory reply"}]}}}},{"type":"event","event":{"type":"turn/end","seq":3,"data":{"turn":1}}}],"hasMore":false}`)
			}
			if wsjson.Write(r.Context(), conn, dsh.ServerFrame{Type: dsh.FrameItem, StreamID: request.StreamID, Value: value}) != nil {
				return
			}
		}
	}))
	mux := &dsh.Mux{Endpoint: dsh.Endpoint{Origin: server.URL}}
	store.AttachMux(mux)
	ctx, cancel := context.WithCancel(context.Background())
	t.Cleanup(func() {
		cancel()
		_ = mux.Close()
		server.Close()
	})
	store.FollowSelected(ctx)
	deadline := time.Now().Add(5 * time.Second)
	for time.Now().Before(deadline) {
		snap := store.Snapshot()
		if snap.HistoryReady && !snap.HistoryLoading && snap.HistoryError == "" && (len(snap.Messages) > 0) == history && transport.goals.Load() == 1 {
			return store, transport
		}
		time.Sleep(time.Millisecond)
	}
	t.Fatal("owned follow stream did not establish the requested history state")
	return nil, nil
}

func TestTrajectorySelectionSurvivesProductionFrame(t *testing.T) {
	previousLocale := i18n.Locale()
	i18n.SetLocale(i18n.LocaleEN)
	t.Cleanup(func() { i18n.SetLocale(previousLocale) })
	for _, history := range []bool{false, true} {
		for _, dock := range []bool{false, true} {
			for _, trace := range []string{"", "1"} {
				t.Run(fmt.Sprintf("history-%t/dock-%t/trace-%t", history, dock, trace == "1"), func(t *testing.T) {
					t.Setenv("WHALEISLE_TRACE_CONVERSATION_TABS", trace)
					var traceLog bytes.Buffer
					previousLog := log.Writer()
					log.SetOutput(&traceLog)
					defer log.SetOutput(previousLog)
					store, transport := trajectoryFrameStore(t, history)
					s := &mainState{sessions: store, chatPaneTab: "chat", filesPaneOpen: dock, rightPaneTab: "files"}
					s.layout = frame.DefaultLayout()
					s.layout.Bottom, s.layout.Right = 0, 540
					root := store.SelectionSummary().SelectedCwd
					status := &desktopgit.RepositoryStatus{IsRepo: false}
					s.gitCheckedRoot, s.gitRepositoryStatus = root, status
					workbench := &diffsurface.Workbench{}
					t.Cleanup(workbench.ClosePane)
					list := &ui.ListState{}
					toggles := 0
					tt := ui.NewTester(func(c *ui.Context) {
						s.mu.Lock()
						layout, chatPaneTab := s.layout, s.chatPaneTab
						query, renameID, renameDraft := s.sessionSearch, s.renameSessionID, s.renameDraft
						s.mu.Unlock()
						ui.Column(c).Fill().Children(func() {
							s.withMainNavigation(c, true, frame.Model{
								Store: store, Sizes: &layout, ThemeID: theme.IDDeepseekLight,
								SessionSearch: &query, RenameSessionID: &renameID, RenameDraft: &renameDraft,
							}, func() {
								model := frame.Model{
									Store: store, Sizes: &layout, ThemeID: theme.IDDeepseekLight,
									ExternalNavigation: true, ChatPaneTab: &chatPaneTab, ChatList: list,
									RightPaneOpen: dock, RightPaneTab: &s.rightPaneTab, GitWorkbench: workbench,
									OnToggleTools: func() { toggles++ }, OnToggleTerminal: func() { toggles++ },
								}
								s.bindWorkspaceTrailing(nil, &model, nil, status, false, root, "")
								frame.View(c, model)
								s.mu.Lock()
								previousTab := s.chatPaneTab
								s.chatPaneTab, s.layout = chatPaneTab, layout
								s.mu.Unlock()
								frame.TraceConversationTabWriteback(previousTab, chatPaneTab)
							})
						})
					}, 2200, 900)
					tt.SetPreferences(ui.Preferences{ReduceMotion: true, TextScale: 1})
					trajectoryLabel, chatLabel := i18n.T(i18n.KeyChatPaneTabTrajectory), i18n.T(i18n.KeyChatPaneTabChat)
					assertTab := func(want, label string) {
						t.Helper()
						if s.chatPaneTab != want {
							t.Fatalf("clicked %s but main paint writeback retained %q", label, s.chatPaneTab)
						}
						band, ok := tt.Find("Conversation views")
						text, textOK := tt.Find(label)
						if !ok || !textOK || text.W <= 0 || text.H <= 0 {
							t.Fatalf("active %s tab is not visible: band=%+v text=%+v", label, band, text)
						}
						// Inspect the software render in memory: the exact active blue
						// underline must move with the state, not just the ledger label.
						accent := chrome.Color(theme.Resolve(theme.IDDeepseekLight).Light.Accent)
						// Installed tabs occupy their own 28-DIP row below the title;
						// the 27-DIP tab's 2-DIP underline has a 1-DIP bottom margin.
						underlineY := int(band.Y + band.H - 3)
						r, g, b, _ := tt.Image().At(int(text.X+text.W/2), underlineY).RGBA()
						if uint8(r/257) != accent.R || uint8(g/257) != accent.G || uint8(b/257) != accent.B {
							t.Fatalf("%s active underline did not follow state: got %d,%d,%d want %d,%d,%d", label, r/257, g/257, b/257, accent.R, accent.G, accent.B)
						}
						inactive := chatLabel
						if want == "chat" {
							inactive = trajectoryLabel
						}
						other, _ := tt.Find(inactive)
						r, g, b, _ = tt.Image().At(int(other.X+other.W/2), underlineY).RGBA()
						if uint8(r/257) == accent.R && uint8(g/257) == accent.G && uint8(b/257) == accent.B {
							t.Fatal("inactive tab retained the active underline")
						}
					}
					assertTab("chat", chatLabel)
					if err := tt.Click(trajectoryLabel); err != nil {
						t.Fatal(err)
					}
					assertTab("trajectory", trajectoryLabel)
					ledger, ok := tt.Find("Trajectory ledger")
					if !ok || ledger.W <= 0 || ledger.H <= 0 {
						t.Fatalf("selected trajectory has no visible ledger: %+v found=%v", ledger, ok)
					}
					if !history && !tt.HasText(i18n.T(i18n.KeyTrajEmpty)) {
						t.Fatal("empty trajectory has no empty-state content")
					}
					tt.Frame()
					assertTab("trajectory", trajectoryLabel)
					if err := tt.Click(chatLabel); err != nil {
						t.Fatal(err)
					}
					assertTab("chat", chatLabel)
					// A passive observer must not turn caption padding or the gap
					// between trailing controls into a clickable tab/panel control.
					band, _ := tt.Find("Conversation title band")
					// Clear the sidebar divider's six-DIP hit slop at the edge.
					tt.ClickAt(band.X+12, band.Y+4)
					terminal, ok := tt.Find("Toggle terminal drawer")
					if !ok {
						t.Fatal("production trailing controls did not render")
					}
					tt.ClickAt(terminal.X+terminal.W+2, terminal.Y+terminal.H/2)
					assertTab("chat", chatLabel)
					if toggles != 0 || transport.calls.Load() != 2 || transport.goals.Load() != 1 {
						t.Fatalf("tab gesture reached another control or Host/model endpoint: toggles=%d requests=%d", toggles, transport.calls.Load())
					}
					text := traceLog.String()
					if trace == "1" {
						for _, want := range []string{
							"[conversation-tab] input node=trajectory kind=down",
							"[conversation-tab] input node=trajectory kind=up",
							"[conversation-tab] input node=caption kind=down",
							"[conversation-tab] input node=trailing kind=down",
							"[conversation-tab] selection requested=trajectory before=chat after=trajectory",
							"[conversation-tab] writeback before=chat after=trajectory",
						} {
							if !strings.Contains(text, want) {
								t.Fatalf("passive trace missed %q: %s", want, text)
							}
						}
						if strings.Contains(text, "Owned trajectory") || strings.Contains(text, "fixture") {
							t.Fatal("passive trace leaked fixture session identity or content")
						}
					} else if strings.Contains(text, "[conversation-tab]") {
						t.Fatal("disabled diagnostic emitted conversation-tab logs")
					}
				})
			}
		}
	}
}
