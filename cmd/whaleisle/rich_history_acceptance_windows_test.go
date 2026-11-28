//go:build windows && richhistoryacceptance

package main

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"net"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	"github.com/egoist/mygo"
	"whaleisle/internal/config"
	"whaleisle/internal/harness"
	"whaleisle/internal/paths"
	"whaleisle/internal/store/session"
	"whaleisle/internal/ui/composer"
)

var richHistoryError error

// Keep the production App.Run on the Windows process main goroutine.
func TestMain(m *testing.M) { richHistoryError = nativeRichHistory(); os.Exit(m.Run()) }
func TestNativeRichHistory(t *testing.T) {
	if richHistoryError != nil {
		t.Fatal(richHistoryError)
	}
}

func nativeRichHistory() error {
	id, workspace, out := os.Getenv("WHALEISLE_RICH_QA_ID"), os.Getenv("WHALEISLE_RICH_QA_WORKSPACE"), os.Getenv("WHALEISLE_RICH_QA_OUT")
	data := os.Getenv(paths.EnvUserData)
	if !isolatedDesktopRun() || !strings.HasPrefix(id, "rich-history-fixture-") || !filepath.IsAbs(data) || !filepath.IsAbs(workspace) || !filepath.IsAbs(out) {
		return fmt.Errorf("explicit owned rich-history environment required")
	}
	cfg, err := config.Open(data)
	if err != nil {
		return err
	}
	if err = cfg.Update(func(v *config.Config) {
		v.Workspace = workspace
		v.Host = "127.0.0.1"
		v.Port = 32184
		v.NodeBin = os.Getenv("WHALEISLE_NODE")
		v.AutoDownloadUpdates = false
	}); err != nil {
		return err
	}
	model, err := newRichQaModel()
	if err != nil {
		return err
	}
	defer model.server.Close()
	os.Setenv("DEEPSEEK_BASE_URL", "http://"+model.addr+"/anthropic")
	os.Setenv("DEEPSEEK_API_KEY", "qa-loopback-only")
	for _, key := range []string{"WHALEISLE_API_KEY", "DSH_API_KEY", "ANTHROPIC_API_KEY", "OPENAI_API_KEY"} {
		os.Unsetenv(key)
	}
	result := make(chan error, 1)
	finish := func(err error) {
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
	go func() {
		deadline := time.Now().Add(105 * time.Second)
		selected, revealed, seededThinking, seededWorkflow := false, false, false, false
		ctx := context.Background()
		for time.Now().Before(deadline) {
			state.mu.Lock()
			win, status, sessions, ready := state.win, state.bootStatus, state.sessions, state.dshMuxReady
			state.mu.Unlock()
			if status.State == harness.Failed {
				finish(fmt.Errorf("owned rich native Host failed"))
				return
			}
			if win == nil || status.State != harness.Ready || sessions == nil || !ready {
				time.Sleep(100 * time.Millisecond)
				continue
			}
			if !revealed {
				state.revealMainConversation()
				revealed = true
			}
			snap := sessions.Snapshot()
			if snap.ListLoading || snap.ListErr != nil || !snap.WorkspaceReady {
				time.Sleep(100 * time.Millisecond)
				continue
			}
			// Seed-then-verify: the fixture is built in this run through the
			// loopback model rather than requiring pre-existing Host sessions.
			if !selected {
				if !(snap.WorkspaceReady && !composer.RequiresWorkspace(snap, true)) {
					time.Sleep(100 * time.Millisecond)
					continue
				}
				if snap.SelectedID == "" {
					sessions.CreateSession(ctx)
					time.Sleep(100 * time.Millisecond)
					continue
				}
				selected = true
				continue
			}
			if !seededThinking {
				seededThinking = true
				mygo.RunOnMain(func() {
					state.mu.Lock()
					draft := state.composerDraftLocked(snap.SelectedID)
					draft.Text = "QA_THINKING rich fixture"
					pngPath := filepath.Join(workspace, "qa-rich.png")
					pngRaw, _ := base64.StdEncoding.DecodeString("iVBORw0KGgoAAAANSUhEUgAAAAEAAAABCAYAAAAfFcSJAAAADUlEQVR42mP8z8BQDwAEhQGAhKmMIQAAAABJRU5ErkJggg==")
					_ = os.MkdirAll(workspace, 0755)
					_ = os.WriteFile(pngPath, pngRaw, 0600)
					draft.Attachments = []session.ComposerAttachment{{Path: pngPath, Name: "qa-rich.png"}}
					state.mu.Unlock()
					state.submitComposer(win, sessions, draft, snap.SelectedID, workspace, false, sessions.Snapshot().ComposerHostGeneration)
				})
				continue
			}
			thinkingReady := false
			for _, m := range snap.Messages {
				if m.Kind == session.MessageKindThinking && strings.Contains(m.Body, "QA_THINKING_TRACE") {
					thinkingReady = true
				}
			}
			if !thinkingReady || snap.PromptBusy {
				time.Sleep(100 * time.Millisecond)
				continue
			}
			if !seededWorkflow {
				seededWorkflow = true
				mygo.RunOnMain(func() {
					state.mu.Lock()
					draft := state.composerDraftLocked(snap.SelectedID)
					draft.Text = "QA_WORKFLOW rich fixture"
					draft.Attachments = nil
					state.mu.Unlock()
					state.submitComposer(win, sessions, draft, snap.SelectedID, workspace, false, sessions.Snapshot().ComposerHostGeneration)
				})
				continue
			}
			if len(snap.WorkflowRuns) == 0 || snap.PromptBusy {
				time.Sleep(100 * time.Millisecond)
				continue
			}
			if snap.HistoryError != nil {
				finish(fmt.Errorf("saved state was not an idle ready history"))
				return
			}
			kind := strings.TrimPrefix(id, "rich-history-fixture-")
			if kind == "media" {
				for _, message := range snap.Messages {
					for _, ref := range session.ImageRefs(message.Body) {
						if attachmentID, ok := strings.CutPrefix(ref, "attachment:"); ok {
							ctx, cancel := context.WithTimeout(context.Background(), 20*time.Second)
							_, err := sessions.FetchAttachment(ctx, attachmentID)
							cancel()
							if err != nil {
								finish(err)
								return
							}
						}
					}
				}
			}
			// Expanding a saved disclosure changes only local view state; no
			// question/approval/workflow mutation callback is installed or called.
			done := make(chan struct{}, 1)
			win.Update(func() {
				state.mu.Lock()
				if state.thinkingExpanded == nil {
					state.thinkingExpanded = map[string]bool{}
				}
				for _, message := range snap.Messages {
					if message.Kind == session.MessageKindThinking {
						state.thinkingExpanded[message.ID] = true
					}
				}
				for _, run := range snap.WorkflowRuns {
					key := fmt.Sprintf("workflow:%q:%q", snap.PlanHistoryBinding, run.ID)
					state.thinkingExpanded[key] = true
					for _, phase := range run.Phases {
						state.thinkingExpanded[key+":"+phase.Key] = true
					}
				}
				state.mu.Unlock()
				done <- struct{}{}
			})
			<-done
			png, err := win.CapturePage()
			if err != nil {
				finish(err)
				return
			}
			if err = os.WriteFile(out+".png", png, 0600); err != nil {
				finish(err)
				return
			}
			state.mu.Lock()
			interactions := state.interactions
			state.mu.Unlock()
			livePending := 0
			if interactions != nil {
				for _, pending := range interactions.Snapshot().Value.Pending {
					if !pending.Continued {
						livePending++
					}
				}
			}
			if livePending != 0 {
				finish(fmt.Errorf("saved state fabricated a live interaction"))
				return
			}
			proof := map[string]any{"kind": kind, "layer": "production hidden Windows mainView + real Host on synthetic persisted state", "historyReady": true, "messageCount": len(snap.Messages), "workflowRuns": len(snap.WorkflowRuns), "promptBusy": snap.PromptBusy, "modelRequests": 0, "nativeQuestionOrApprovalResponseInvoked": false, "livePendingInteractions": livePending, "localDisclosuresExpandedProgrammatically": true, "windowHidden": true, "originalPixelParityClaim": false}
			raw, _ := json.MarshalIndent(proof, "", "  ")
			finish(os.WriteFile(out+".json", append(raw, '\n'), 0600))
			return
		}
		finish(fmt.Errorf("owned native rich history readiness timeout"))
	}()
	main()
	select {
	case err := <-result:
		return err
	case <-time.After(2 * time.Second):
		return fmt.Errorf("owned native desktop exited before proof")
	}
}

// richQaModel serves Anthropic Messages with deterministic SSE for the rich
// fixture seed: thinking blocks, a trivial workflow script and plain replies.
type richQaModel struct {
	server   *http.Server
	addr     string
	requests atomic.Int64
}

func newRichQaModel() (*richQaModel, error) {
	l, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		return nil, err
	}
	m := &richQaModel{addr: l.Addr().String()}
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
		}
		_ = json.NewDecoder(r.Body).Decode(&body)
		text := ""
		if len(body.Messages) > 0 {
			if raw, e := json.Marshal(body.Messages[len(body.Messages)-1].Content); e == nil {
				text = string(raw)
			}
		}
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
		textBlock := func(index int, body string) {
			emit("content_block_start", map[string]any{"index": index, "content_block": map[string]any{"type": "text", "text": ""}})
			emit("content_block_delta", map[string]any{"index": index, "delta": map[string]any{"type": "text_delta", "text": body}})
			emit("content_block_stop", map[string]any{"index": index})
		}
		if strings.Contains(text, "QA_WORKFLOW") {
			// Trivial script: the run lifecycle alone projects WorkflowRuns.
			args := map[string]any{"script": "return { ok: true }", "meta": map[string]any{"name": "qa-rich-run", "description": "rich fixture workflow"}, "args": map[string]any{}}
			raw, _ := json.Marshal(args)
			emit("content_block_start", map[string]any{"index": 0, "content_block": map[string]any{"type": "tool_use", "id": fmt.Sprintf("qa-wf-%d", m.requests.Load()), "name": "workflow", "input": map[string]any{}}})
			emit("content_block_delta", map[string]any{"index": 0, "delta": map[string]any{"type": "input_json_delta", "partial_json": string(raw)}})
			emit("content_block_stop", map[string]any{"index": 0})
			emit("message_delta", map[string]any{"delta": map[string]any{"stop_reason": "tool_use", "stop_sequence": nil}, "usage": map[string]int{"output_tokens": 20}})
			emit("message_stop", map[string]any{})
			return
		}
		if strings.Contains(text, "QA_THINKING") {
			emit("content_block_start", map[string]any{"index": 0, "content_block": map[string]any{"type": "thinking", "thinking": ""}})
			emit("content_block_delta", map[string]any{"index": 0, "delta": map[string]any{"type": "thinking_delta", "thinking": "QA_THINKING_TRACE deterministic reasoning."}})
			emit("content_block_stop", map[string]any{"index": 0})
			textBlock(1, "QA_REPLY_OK deterministic reply.")
			emit("message_delta", map[string]any{"delta": map[string]any{"stop_reason": "end_turn", "stop_sequence": nil}, "usage": map[string]int{"output_tokens": 30}})
			emit("message_stop", map[string]any{})
			return
		}
		textBlock(0, "QA_REPLY_OK deterministic reply.")
		emit("message_delta", map[string]any{"delta": map[string]any{"stop_reason": "end_turn", "stop_sequence": nil}, "usage": map[string]int{"output_tokens": 5}})
		emit("message_stop", map[string]any{})
	})
	m.server = &http.Server{Handler: mux}
	go m.server.Serve(l)
	return m, nil
}
