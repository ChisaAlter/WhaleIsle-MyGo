//go:build windows && composerflowacceptance

package main

import (
	"context"
	"encoding/json"
	"fmt"
	"net"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	"github.com/egoist/mygo"
	"whaleisle/internal/config"
	"whaleisle/internal/harness"
	"whaleisle/internal/paths"
	"whaleisle/internal/store/session"
)

// Opt-in real-window composer QA: one hidden desktop, one fresh profile, one
// owned Host, one deterministic loopback model. Drives the same entry points
// the composer buttons call (submitComposer, Select, CancelRunning). No OS
// input injection; the widget hit-test layer is covered by ui.NewTester tests.
var composerFlowError error

func TestMain(m *testing.M) {
	// The native recovery watchdog re-executes this binary with inherited
	// environment; only the primary desktop process may run the QA drive.
	if len(os.Args) == 3 && os.Args[1] == nativeRecoveryFlag {
		os.Exit(m.Run())
	}
	composerFlowError = runComposerFlowQA()
	os.Exit(m.Run())
}

func TestComposerFlowAcceptance(t *testing.T) {
	if composerFlowError != nil {
		t.Fatal(composerFlowError)
	}
}

// qaModel serves Anthropic Messages with deterministic SSE. "QA_WAIT" streams
// forever so the stop gesture has a live turn to cancel.
type qaModel struct {
	server    *http.Server
	addr      string
	requests  atomic.Int64
	lastText  atomic.Value
	gatedTool atomic.Value
	toolNames atomic.Value
}

func newQAModel() (*qaModel, error) {
	l, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		return nil, err
	}
	m := &qaModel{addr: l.Addr().String()}
	mux := http.NewServeMux()
	mux.HandleFunc("/anthropic/v1/messages", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			http.Error(w, "POST only", 405)
			return
		}
		m.requests.Add(1)
		var body struct {
			Messages []struct {
				Content any `json:"content"`
			} `json:"messages"`
			Tools []struct {
				Name        string `json:"name"`
				InputSchema struct {
					Properties map[string]any `json:"properties"`
				} `json:"input_schema"`
			} `json:"tools"`
		}
		_ = json.NewDecoder(r.Body).Decode(&body)
		text := ""
		if len(body.Messages) > 0 {
			if raw, e := json.Marshal(body.Messages[len(body.Messages)-1].Content); e == nil {
				text = string(raw)
			}
		}
		m.lastText.Store(text)
		w.Header().Set("Content-Type", "text/event-stream")
		w.Header().Set("Cache-Control", "no-cache")
		w.WriteHeader(200)
		fl, _ := w.(http.Flusher)
		emit := func(event string, data map[string]any) {
			data["type"] = event
			raw, _ := json.Marshal(data)
			fmt.Fprintf(w, "event: %s\ndata: %s\n\n", event, raw)
			fl.Flush()
		}
		emit("message_start", map[string]any{"message": map[string]any{"id": fmt.Sprintf("qa-%d", m.requests.Load()), "type": "message", "role": "assistant", "model": "qa", "content": []any{}, "stop_reason": nil, "stop_sequence": nil, "usage": map[string]int{"input_tokens": 1, "output_tokens": 0}}})
		if strings.Contains(text, "QA_LONG") {
			emit("content_block_start", map[string]any{"index": 0, "content_block": map[string]any{"type": "text", "text": ""}})
			paragraph := "QA_LONG_STREAM deterministic paragraph. The native frame must repaint every streamed chunk while long history stays mounted. "
			for i := 0; i < 220; i++ {
				emit("content_block_delta", map[string]any{"index": 0, "delta": map[string]any{"type": "text_delta", "text": fmt.Sprintf("%s%04d\n", paragraph, i)}})
				time.Sleep(25 * time.Millisecond)
			}
			emit("content_block_stop", map[string]any{"index": 0})
			emit("message_delta", map[string]any{"delta": map[string]any{"stop_reason": "end_turn", "stop_sequence": nil}, "usage": map[string]int{"output_tokens": 4000}})
			emit("message_stop", map[string]any{})
			return
		}
		if strings.Contains(text, "tool_result") {
			emit("content_block_start", map[string]any{"index": 0, "content_block": map[string]any{"type": "text", "text": ""}})
			emit("content_block_delta", map[string]any{"index": 0, "delta": map[string]any{"type": "text_delta", "text": "QA_APPROVED_OK: gated tool result returned after approval."}})
			emit("content_block_stop", map[string]any{"index": 0})
			emit("message_delta", map[string]any{"delta": map[string]any{"stop_reason": "end_turn", "stop_sequence": nil}, "usage": map[string]int{"output_tokens": 20}})
			emit("message_stop", map[string]any{})
			return
		}
		if strings.Contains(text, "QA_APPROVE") {
			var names []string
			for _, t := range body.Tools {
				names = append(names, t.Name)
			}
			m.toolNames.Store(strings.Join(names, ","))
			gated := []string{"pwsh", "bash", "powershell", "shell", "exec", "run_command", "runcommand", "cmd", "write", "edit", "apply_patch", "fs_write", "writefile"}
			chosenTool := -1
		outer:
			for _, g := range gated {
				for i, tool := range body.Tools {
					if strings.Contains(strings.ToLower(tool.Name), g) {
						chosenTool = i
						break outer
					}
				}
			}
			if chosenTool >= 0 {
				tool := body.Tools[chosenTool]
				m.gatedTool.Store(tool.Name)
				args := map[string]any{}
				for key := range tool.InputSchema.Properties {
					switch strings.ToLower(key) {
					case "command", "cmd", "script":
						args[key] = "echo qa-approve"
					case "path", "file_path", "filepath", "filename":
						args[key] = "qa-approve.txt"
					case "content", "text", "new_string":
						args[key] = "qa approved write"
					case "command_line":
						args[key] = "echo qa-approve"
					case "sandbox_permissions":
						args[key] = "danger-full-access"
					case "justification", "description", "reason":
						args[key] = "qa approval escalation"
					}
				}
				raw, _ := json.Marshal(args)
				emit("content_block_start", map[string]any{"index": 0, "content_block": map[string]any{"type": "tool_use", "id": fmt.Sprintf("qa-gated-%d", m.requests.Load()), "name": tool.Name, "input": map[string]any{}}})
				emit("content_block_delta", map[string]any{"index": 0, "delta": map[string]any{"type": "input_json_delta", "partial_json": string(raw)}})
				emit("content_block_stop", map[string]any{"index": 0})
				emit("message_delta", map[string]any{"delta": map[string]any{"stop_reason": "tool_use", "stop_sequence": nil}, "usage": map[string]int{"output_tokens": 20}})
				emit("message_stop", map[string]any{})
				return
			}
			emit("content_block_start", map[string]any{"index": 0, "content_block": map[string]any{"type": "text", "text": ""}})
			emit("content_block_delta", map[string]any{"index": 0, "delta": map[string]any{"type": "text_delta", "text": "QA_NO_GATED_TOOL: session advertised no exec/write-class tool."}})
			emit("content_block_stop", map[string]any{"index": 0})
			emit("message_delta", map[string]any{"delta": map[string]any{"stop_reason": "end_turn", "stop_sequence": nil}, "usage": map[string]int{"output_tokens": 10}})
			emit("message_stop", map[string]any{})
			return
		}
		if strings.Contains(text, "QA_WAIT") {
			emit("content_block_start", map[string]any{"index": 0, "content_block": map[string]any{"type": "text", "text": ""}})
			emit("content_block_delta", map[string]any{"index": 0, "delta": map[string]any{"type": "text_delta", "text": "QA_WAIT_STREAM: awaiting real session cancellation."}})
			select {
			case <-r.Context().Done():
			case <-time.After(45 * time.Second):
			}
			return
		}
		emit("content_block_start", map[string]any{"index": 0, "content_block": map[string]any{"type": "text", "text": ""}})
		emit("content_block_delta", map[string]any{"index": 0, "delta": map[string]any{"type": "text_delta", "text": "QA_REPLY_OK deterministic reply."}})
		emit("content_block_stop", map[string]any{"index": 0})
		emit("message_delta", map[string]any{"delta": map[string]any{"stop_reason": "end_turn", "stop_sequence": nil}, "usage": map[string]int{"output_tokens": 5}})
		emit("message_stop", map[string]any{})
	})
	m.server = &http.Server{Handler: mux}
	go m.server.Serve(l)
	return m, nil
}

func runComposerFlowQA() error {
	workspace, out := os.Getenv("WHALEISLE_COMPOSER_QA_WORKSPACE"), os.Getenv("WHALEISLE_COMPOSER_QA_OUT")
	data := os.Getenv(paths.EnvUserData)
	if !isolatedDesktopRun() || !filepath.IsAbs(data) || !filepath.IsAbs(workspace) || !filepath.IsAbs(out) {
		return fmt.Errorf("explicit owned composer QA environment required")
	}
	model, err := newQAModel()
	if err != nil {
		return err
	}
	defer model.server.Close()
	os.Setenv("DEEPSEEK_BASE_URL", "http://"+model.addr+"/anthropic")
	os.Setenv("DEEPSEEK_API_KEY", "qa-loopback-only")
	for _, key := range []string{"WHALEISLE_API_KEY", "DSH_API_KEY", "ANTHROPIC_API_KEY", "OPENAI_API_KEY"} {
		os.Unsetenv(key)
	}
	notePath := filepath.Join(workspace, "qa-note.txt")
	if err = os.MkdirAll(workspace, 0o755); err != nil {
		return err
	}
	if err = os.WriteFile(notePath, []byte("qa owned attachment bytes\n"), 0o600); err != nil {
		return err
	}
	cfg, err := config.Open(data)
	if err != nil {
		return err
	}
	if err = cfg.Update(func(c *config.Config) {
		c.Workspace = workspace
		c.Host = "127.0.0.1"
		c.Port = 32197
		c.NodeBin = os.Getenv("WHALEISLE_NODE")
	}); err != nil {
		return err
	}
	type step struct {
		Name   string `json:"name"`
		Pass   bool   `json:"pass"`
		Detail string `json:"detail"`
	}
	var steps []step
	mark := func(name string, pass bool, detail string) {
		steps = append(steps, step{Name: name, Pass: pass, Detail: detail})
	}
	result := make(chan error, 1)
	finish := func(err error) {
		raw, _ := json.MarshalIndent(map[string]any{"pid": os.Getpid(), "steps": steps, "modelRequests": model.requests.Load(), "error": fmt.Sprint(err)}, "", "  ")
		_ = os.WriteFile(out, append(raw, '\n'), 0o600)
		result <- err
		state.mu.Lock()
		state.quitting = true
		sup := state.sup
		state.mu.Unlock()
		if sup != nil {
			sup.Stop()
		}
		mygo.App.Quit()
	}
	waitStore := func(deadline time.Time, what string, cond func() (bool, error)) error {
		for time.Now().Before(deadline) {
			ok, err := cond()
			if err != nil {
				return err
			}
			if ok {
				return nil
			}
			time.Sleep(100 * time.Millisecond)
		}
		return fmt.Errorf("timeout waiting for %s", what)
	}
	go func() {
		deadline := time.Now().Add(230 * time.Second)
		var win *mygo.Window
		var sessions *session.Store
		var lastState, lastEndpoint string
		for time.Now().Before(deadline) {
			state.mu.Lock()
			status, s, ready := state.bootStatus, state.sessions, state.dshMuxReady
			win = state.win
			state.mu.Unlock()
			lastState, lastEndpoint = status.State.String(), status.Endpoint.Origin
			if status.State == harness.Failed {
				finish(fmt.Errorf("owned native Host failed"))
				return
			}
			if win != nil && status.State == harness.Ready && s != nil && ready {
				sessions = s
				break
			}
			time.Sleep(100 * time.Millisecond)
		}
		if sessions == nil {
			finish(fmt.Errorf("owned composer QA readiness timeout (win=%v state=%s endpoint=%s)", win != nil, lastState, lastEndpoint))
			return
		}
		state.revealMainConversation()
		if _, err := win.CapturePage(); err != nil {
			finish(err)
			return
		}
		ctx := context.Background()
		if err := waitStore(deadline, "workspace ready", func() (bool, error) {
			v := sessions.Snapshot()
			if v.ListErr != "" {
				return false, fmt.Errorf("%s", v.ListErr)
			}
			return v.WorkspaceReady && !v.ListLoading, nil
		}); err != nil {
			finish(err)
			return
		}
		// --- two workspace sessions through the real create gesture ---
		seenActions := map[string]bool{}
		watch := func() (bool, error) {
			seenActions[sessions.Snapshot().LastAction] = true
			return false, nil
		}
		sessions.CreateSession(ctx)
		baseID := sessions.Snapshot().SelectedID
		if err := waitStore(deadline, "session 1 created", func() (bool, error) {
			watch()
			return sessions.Snapshot().SelectedID != baseID, nil
		}); err != nil {
			finish(err)
			return
		}
		id1 := sessions.Snapshot().SelectedID
		sessions.CreateSession(ctx)
		if err := waitStore(deadline, "session 2 created", func() (bool, error) {
			watch()
			v := sessions.Snapshot()
			return v.SelectedID != id1 && v.SelectedID != "", nil
		}); err != nil {
			finish(err)
			return
		}
		id2 := sessions.Snapshot().SelectedID
		mark("create-two-sessions", id1 != "" && id2 != "" && id1 != id2, fmt.Sprintf("id1=%s id2=%s createdSignalSampled=%v", id1, id2, seenActions["session created"]))
		// --- draft + attachment survives switching ---
		state.mu.Lock()
		d1 := state.composerDraftLocked(id1)
		d1.Text = "QA draft A"
		d1.Attachments = []session.ComposerAttachment{{Path: notePath, Name: "qa-note.txt"}}
		state.mu.Unlock()
		sessions.Select(id1)
		time.Sleep(300 * time.Millisecond)
		sessions.Select(id2)
		time.Sleep(300 * time.Millisecond)
		state.mu.Lock()
		state.composerDraftLocked(id2).Text = "QA draft B"
		state.mu.Unlock()
		sessions.Select(id1)
		if err := waitStore(deadline, "reselect id1", func() (bool, error) {
			return sessions.Snapshot().SelectedID == id1, nil
		}); err != nil {
			finish(err)
			return
		}
		state.mu.Lock()
		gotA, gotB := state.composerDrafts[id1], state.composerDrafts[id2]
		state.mu.Unlock()
		switchOK := gotA != nil && gotA.Text == "QA draft A" && len(gotA.Attachments) == 1 && gotB != nil && gotB.Text == "QA draft B"
		mark("draft-attachment-switch", switchOK, fmt.Sprintf("A=%q att=%d B=%q", gotA.Text, len(gotA.Attachments), gotB.Text))
		if !switchOK {
			finish(fmt.Errorf("draft/attachment lost across session switch"))
			return
		}
		// --- real send with attachment; type while in flight ---
		state.mu.Lock()
		d1 = state.composerDraftLocked(id1)
		d1.Text = "QA_HELLO first send"
		gen := sessions.Snapshot().ComposerHostGeneration
		state.mu.Unlock()
		state.submitComposer(win, sessions, d1, id1, workspace, 0, false, gen)
		time.Sleep(150 * time.Millisecond)
		state.mu.Lock()
		if d := state.composerDrafts[id1]; d != nil {
			d.Text = "typed while sending"
		}
		state.mu.Unlock()
		var userCount, replyCount int
		if err := waitStore(deadline, "first turn completes", func() (bool, error) {
			v := sessions.Snapshot()
			if v.SelectedID != id1 {
				return false, nil
			}
			userCount, replyCount = 0, 0
			for _, m := range v.Messages {
				if m.Role == "user" && strings.Contains(m.Body, "QA_HELLO first send") {
					userCount++
				}
				if m.Role == "assistant" && strings.Contains(m.Body, "QA_REPLY_OK") {
					replyCount++
				}
			}
			return userCount >= 1 && replyCount >= 1 && !v.PromptBusy, nil
		}); err != nil {
			v := sessions.Snapshot()
			state.mu.Lock()
			dd := state.composerDrafts[id1]
			dErr, dSending, dText, dAtt := "", false, "", 0
			if dd != nil {
				dErr, dSending, dText = dd.Error, dd.Sending, dd.Text
				dAtt = len(dd.Attachments)
			}
			state.mu.Unlock()
			finish(fmt.Errorf("first turn never completed: %w (last=%q busy=%v send=%v derr=%q dtext=%q att=%d msgs=%d sel=%s)", err, v.LastAction, v.PromptBusy, dSending, dErr, dText, dAtt, len(v.Messages), v.SelectedID))
			return
		}
		state.mu.Lock()
		midEdit := state.composerDrafts[id1].Text
		state.mu.Unlock()
		mark("send-with-attachment-mid-edit", userCount == 1 && replyCount >= 1 && midEdit == "typed while sending",
			fmt.Sprintf("user=%d reply=%d draft=%q", userCount, replyCount, midEdit))
		// --- stop after send ---
		sessions.Select(id2)
		if err := waitStore(deadline, "select id2", func() (bool, error) {
			return sessions.Snapshot().SelectedID == id2 && sessions.Snapshot().HistoryReady, nil
		}); err != nil {
			finish(err)
			return
		}
		state.mu.Lock()
		d2 := state.composerDraftLocked(id2)
		d2.Text = "QA_WAIT stream until stopped"
		gen = sessions.Snapshot().ComposerHostGeneration
		state.mu.Unlock()
		state.submitComposer(win, sessions, d2, id2, workspace, 0, false, gen)
		if err := waitStore(deadline, "wait turn running", func() (bool, error) {
			v := sessions.Snapshot()
			if v.SelectedID != id2 {
				return false, nil
			}
			for _, it := range v.Items {
				if it.SessionID == id2 && it.Running {
					return true, nil
				}
			}
			return v.PromptBusy, nil
		}); err != nil {
			v := sessions.Snapshot()
			finish(fmt.Errorf("wait turn never started: %w (sel=%s busy=%v last=%s draft=%q sending=%v err=%q)", err, v.SelectedID, v.PromptBusy, v.LastAction, d2.Text, d2.Sending, d2.Error))
			return
		}
		time.Sleep(300 * time.Millisecond)
		sessions.CancelRunning(ctx)
		if err := waitStore(deadline, "cancel accepted", func() (bool, error) {
			watch()
			v := sessions.Snapshot()
			for _, it := range v.Items {
				if it.SessionID == id2 && it.Running {
					return false, nil
				}
			}
			return !v.PromptBusy, nil
		}); err != nil {
			finish(fmt.Errorf("cancel never accepted: %w (last=%s)", err, sessions.Snapshot().LastAction))
			return
		}
		mark("stop-after-send", true, fmt.Sprintf("session/cancel accepted while streaming (signalSampled=%v)", seenActions["cancel accepted"]))
		// --- stale edit rejection keeps draft; retry succeeds ---
		state.mu.Lock()
		d2 = state.composerDrafts[id2]
		if d2 == nil {
			d2 = state.composerDraftLocked(id2)
		}
		d2.EditSeq = 999
		d2.Text = "stale edit attempt"
		state.mu.Unlock()
		state.submitComposer(win, sessions, d2, id2, workspace, 0, false, sessions.Snapshot().ComposerHostGeneration)
		time.Sleep(400 * time.Millisecond)
		staleOK := d2.Error != "" && d2.Text == "stale edit attempt"
		mark("stale-edit-rejected-draft-kept", staleOK, fmt.Sprintf("err=%q text=%q", d2.Error, d2.Text))
		state.mu.Lock()
		d2.EditSeq = 0
		d2.Error = ""
		d2.Text = "QA_HELLO retry after rejection"
		state.mu.Unlock()
		state.submitComposer(win, sessions, d2, id2, workspace, 0, false, sessions.Snapshot().ComposerHostGeneration)
		if err := waitStore(deadline, "retry completes", func() (bool, error) {
			v := sessions.Snapshot()
			if v.SelectedID != id2 {
				return false, nil
			}
			for _, m := range v.Messages {
				if m.Role == "assistant" && strings.Contains(m.Body, "QA_REPLY_OK") && !strings.Contains(m.Body, "QA_WAIT") {
					return !v.PromptBusy, nil
				}
			}
			return false, nil
		}); err != nil {
			v := sessions.Snapshot()
			var tails []string
			for i, m := range v.Messages {
				if i < len(v.Messages)-4 {
					continue
				}
				b := m.Body
				if len(b) > 60 {
					b = b[:60]
				}
				tails = append(tails, fmt.Sprintf("%s:%q", m.Role, b))
			}
			running := 0
			for _, it := range v.Items {
				if it.Running {
					running++
				}
			}
			finish(fmt.Errorf("retry never completed: %w (sel=%s busy=%v running=%d last=%q send=%v derr=%q msgs=%v)", err, v.SelectedID, v.PromptBusy, running, v.LastAction, d2.Sending, d2.Error, tails))
			return
		}
		mark("retry-after-rejection", true, "second submit accepted and replied")
		// --- one-time approval on a gated tool call ---
		sessions.Select(id1)
		state.mu.Lock()
		dA := state.composerDraftLocked(id1)
		dA.EditSeq = 0
		dA.Error = ""
		dA.Text = "QA_APPROVE run gated tool"
		state.mu.Unlock()
		state.submitComposer(win, sessions, dA, id1, workspace, 0, false, sessions.Snapshot().ComposerHostGeneration)
		approvalSeen := ""
		var approveErr error
		approveDeadline := time.Now().Add(40 * time.Second)
		for time.Now().Before(approveDeadline) {
			if approvalSeen == "" {
				for _, p := range state.interactions.Snapshot().Pending {
					if p.Event == "approval/request" {
						approvalSeen = p.EventID
						break
					}
				}
			}
			v := sessions.Snapshot()
			for _, m := range v.Messages {
				if m.Role == "assistant" && strings.Contains(m.Body, "QA_NO_GATED_TOOL") {
					approveErr = fmt.Errorf("session advertised no exec/write-class tool")
				}
				if m.Role == "assistant" && strings.Contains(m.Body, "QA_APPROVED_OK") {
					approveErr = fmt.Errorf("tool completed without an approval request")
				}
			}
			if approveErr != nil {
				break
			}
			if approvalSeen != "" {
				break
			}
			time.Sleep(200 * time.Millisecond)
		}
		toolUsed, _ := model.gatedTool.Load().(string)
		allTools, _ := model.toolNames.Load().(string)
		switch {
		case approveErr != nil:
			mark("one-time-approval", false, fmt.Sprintf("inconclusive: %s (tool=%q advertised=[%s])", approveErr.Error(), toolUsed, allTools))
		case approvalSeen == "":
			mark("one-time-approval", false, "no approval/request reached the interaction store within 40s")
		default:
			state.interactions.ResolveAllow(approvalSeen)
			if err := waitStore(deadline, "approval accepted tool completes", func() (bool, error) {
				for _, m := range sessions.Snapshot().Messages {
					if m.Role == "assistant" && strings.Contains(m.Body, "QA_APPROVED_OK") {
						return true, nil
					}
				}
				return false, nil
			}); err != nil {
				finish(fmt.Errorf("approval resolved but tool never completed: %w", err))
				return
			}
			cleared := true
			for _, p := range state.interactions.Snapshot().Pending {
				if p.EventID == approvalSeen {
					cleared = false
				}
			}
			mark("one-time-approval", cleared, fmt.Sprintf("pending approval %s resolved once; tool result returned; pending cleared=%v", approvalSeen, cleared))
		}
		// --- optional sustained-stream workload for frame-cost sampling ---
		if n, _ := strconv.Atoi(os.Getenv("WHALEISLE_COMPOSER_QA_WORKLOAD")); n > 0 {
			// Nudge: invalidate the window directly so a flat frame counter
			// can be attributed to either the store→refresh link or the
			// paint pipeline itself.
			if os.Getenv("WHALEISLE_COMPOSER_QA_NUDGE") == "1" {
				go func() {
					for i := 0; i < 40; i++ {
						win.Invalidate()
						time.Sleep(2 * time.Second)
					}
				}()
			}
			sessions.Select(id1)
			snap := func() string {
				var c, dr, ch, cc, rq, pt int64
				mygo.RunOnMain(func() {
					c, dr, ch, cc = mygo.TestInvalidateCounters()
					rq, pt = mygo.TestFrameCounters()
				})
				return fmt.Sprintf("inv=%d(-%d) cc=%d conn=%d req=%d paint=%d frames=%d", c, dr, ch, cc, rq, pt, uiDiagFrames.Load())
			}
			baseSnap := snap()
			midSnap := ""
			for turn := 0; turn < n; turn++ {
				state.mu.Lock()
				d := state.composerDraftLocked(id1)
				d.Text = fmt.Sprintf("QA_LONG stream turn %d", turn+1)
				state.mu.Unlock()
				state.submitComposer(win, sessions, d, id1, workspace, 0, false, sessions.Snapshot().ComposerHostGeneration)
				if turn == n-1 {
					time.Sleep(1500 * time.Millisecond)
					midSnap = snap()
				}
				want := fmt.Sprintf("%04d", 219)
				if err := waitStore(deadline, "long stream completes", func() (bool, error) {
					v := sessions.Snapshot()
					if v.SelectedID != id1 {
						return false, nil
					}
					running := false
					for _, it := range v.Items {
						if it.SessionID == id1 && it.Running {
							running = true
						}
					}
					if running {
						return false, nil
					}
					for _, m := range v.Messages {
						if m.Role == "assistant" && strings.Contains(m.Body, want) {
							return true, nil
						}
					}
					return false, nil
				}); err != nil {
					finish(fmt.Errorf("workload turn %d incomplete: %w", turn+1, err))
					return
				}
			}
			// Frame accounting check: report window state and the backend's
			// frame-hold latches so "no frames during streaming" can be
			// attributed to the invalidate path, visibility, or a stuck hold.
			hwnd := win.NativeHandle()
			var interaction, sizeBurst bool
			mygo.RunOnMain(func() { interaction, sizeBurst = mygo.TestHoldsFrames(hwnd) })
			visible, minimized := win.IsVisible(), win.IsMinimized()
			framesBefore := uiDiagFrames.Load()
			var framesAfter int64 = -1
			if _, err := win.CapturePage(); err == nil {
				framesAfter = uiDiagFrames.Load()
			}
			var reqCount, paintCount int64
			mygo.RunOnMain(func() { reqCount, paintCount = mygo.TestFrameCounters() })
			mark("workload-streams", true, fmt.Sprintf("%d long turns; visible=%v min=%v holdFrames(interaction=%v,sizeBurst=%v) frames=%d→%d | base[%s] mid[%s] end[req=%d paint=%d]", n, visible, minimized, interaction, sizeBurst, framesBefore, framesAfter, baseSnap, midSnap, reqCount, paintCount))
			// Real input path: set the composer text through UI Automation's
			// ValuePattern, the channel assistive technology uses. Restore a
			// minimized window first: UIA and focus need a presentable surface.
			if win.IsMinimized() {
				win.Restore()
				time.Sleep(500 * time.Millisecond)
			}
			var uiaOK bool
			mygo.RunOnMain(func() { uiaOK = mygo.TestAccessibilityPerform(hwnd, "Composer draft", "value", "UIA typed text") })
			if uiaOK {
				time.Sleep(400 * time.Millisecond)
				state.mu.Lock()
				got := ""
				if d := state.composerDrafts[id1]; d != nil {
					got = d.Text
				}
				state.mu.Unlock()
				mark("real-input-uia-setvalue", got == "UIA typed text", fmt.Sprintf("perform=%v draft=%q", uiaOK, got))
			} else {
				mark("real-input-uia-setvalue", false, "Composer draft element not performable")
			}
			// Unicode text delivery through the real wndproc: WM_CHAR is the
			// channel committed IME strings and typed text arrive on. Focus the
			// composer through UIA, then send CJK code units.
			var focusOK, sentOK bool
			mygo.RunOnMain(func() {
				focusOK = mygo.TestAccessibilityPerform(hwnd, "Composer draft", "focus", "")
				sentOK = true
				for _, r := range "你好" {
					if !mygo.TestSendChar(hwnd, r) {
						sentOK = false
					}
				}
			})
			time.Sleep(400 * time.Millisecond)
			state.mu.Lock()
			draftAfterChars := ""
			if d := state.composerDrafts[id1]; d != nil {
				draftAfterChars = d.Text
			}
			state.mu.Unlock()
			mark("real-input-unichar", sentOK && strings.Contains(draftAfterChars, "你好"), fmt.Sprintf("focus=%v sent=%v draft=%q", focusOK, sentOK, draftAfterChars))
		}
		failed := false
		for _, st := range steps {
			if !st.Pass {
				failed = true
			}
		}
		if failed {
			finish(fmt.Errorf("composer QA step failures; see steps"))
			return
		}
		finish(nil)
	}()
	main()
	select {
	case err := <-result:
		return err
	case <-time.After(2 * time.Second):
		return fmt.Errorf("native desktop exited before composer QA proof")
	}
}
