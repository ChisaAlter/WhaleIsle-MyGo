//go:build windows && composerflowacceptance

package main

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"net"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	"github.com/egoist/mygo"
	"github.com/egoist/mygo/ui"
	"whaleisle/internal/config"
	"whaleisle/internal/desktop/git"
	"whaleisle/internal/desktop/workspacefs"
	"whaleisle/internal/harness"
	"whaleisle/internal/paths"
	"whaleisle/internal/store/session"
	"whaleisle/internal/ui/composer"
	"whaleisle/internal/ui/files"
	"whaleisle/internal/ui/frame"
)

// findFileResourcePane walks the workspace dock for the files pane owning rel,
// matching OpenFileResource's own identity check (Kind + ResourcePath).
func findFileResourcePane(w *frame.WorkspacePanes, rel string) *frame.WorkspacePane {
	if w == nil {
		return nil
	}
	var pane *frame.WorkspacePane
	var walk func(n *frame.WorkspacePane)
	walk = func(n *frame.WorkspacePane) {
		if n == nil || pane != nil {
			return
		}
		if n.Kind == "files" && n.Files != nil && n.Files.ResourcePath == rel {
			pane = n
			return
		}
		for _, tab := range n.DockTabs {
			walk(tab)
		}
		walk(n.First)
		walk(n.Second)
	}
	walk(w.Root)
	return pane
}

// seedRecoveredFileDraft supplies a recovered unsaved document to its actual
// pane owner. The ordinary frame projection publishes it; this fixture does
// not claim to exercise the edit menu or native text entry.
func seedRecoveredFileDraft(pane *frame.WorkspacePane, rel, root, draft string) bool {
	if pane == nil || pane.Files == nil {
		return false
	}
	var tab *files.FileTab
	if ws := pane.Files.Tabs.Workspaces[filepath.Clean(root)]; ws != nil {
		for _, t := range ws.Open {
			if t != nil && t.Path == rel {
				tab = t
				break
			}
		}
	}
	if tab == nil {
		return false
	}
	tab.Editing, tab.Draft = true, draft
	pane.Files.Editing, pane.Files.Draft = true, draft
	return true
}

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
	if os.Getenv("WHALEISLE_COMPOSER_QA_SCENARIO") == "submission-recovery" {
		composerFlowError = runSubmissionRecoveryAcceptance()
	} else {
		composerFlowError = runComposerFlowQA()
	}
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
	wfSchema  atomic.Value
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
		if strings.Contains(text, "QA_WORKFLOW") {
			// Workflow probe: invoke the agent-orchestration tool so the Host
			// spawns a child run, emitting the tool-workflow/ journal events
			// that project as WorkflowRuns.
			chosenTool := -1
			for _, want := range []string{"workflow", "agent", "task", "delegate", "spawn", "goal"} {
				for i, tool := range body.Tools {
					if strings.Contains(strings.ToLower(tool.Name), want) {
						chosenTool = i
						break
					}
				}
				if chosenTool >= 0 {
					break
				}
			}
			if chosenTool < 0 {
				names := []string{}
				for _, t := range body.Tools {
					names = append(names, t.Name)
				}
				emit("content_block_start", map[string]any{"index": 0, "content_block": map[string]any{"type": "text", "text": ""}})
				emit("content_block_delta", map[string]any{"index": 0, "delta": map[string]any{"type": "text_delta", "text": "QA_NO_AGENT_TOOL: " + strings.Join(names, ",")}})
				emit("content_block_stop", map[string]any{"index": 0})
				emit("message_delta", map[string]any{"delta": map[string]any{"stop_reason": "end_turn", "stop_sequence": nil}, "usage": map[string]int{"output_tokens": 10}})
				emit("message_stop", map[string]any{})
				return
			}
			tool := body.Tools[chosenTool]
			if rawSchema, e := json.Marshal(tool.InputSchema); e == nil {
				m.wfSchema.Store(tool.Name + " " + string(rawSchema))
			}
			args := map[string]any{}
			if tool.Name == "workflow" {
				// Trivial script probe: even without delegations a completed run
				// should emit the run lifecycle events the desktop projects.
				args["script"] = "return { ok: true }"
				args["meta"] = map[string]any{"name": "qa-probe-run", "description": "QA workflow probe"}
				args["args"] = map[string]any{}
			}
			if tool.Name == "create_goal" {
				// Exact replication beats shape inference: the vendor schema
				// requires activation as an armed/disarmed enum member.
				args["objective"] = "CHILDROUND: reply briefly and finish."
				args["max_goal_rounds"] = 1
				args["activation"] = "armed"
			}
			for key, rawProp := range tool.InputSchema.Properties {
				if _, preset := args[key]; preset {
					continue
				}
				propType := ""
				var enumVals []any
				if prop, ok := rawProp.(map[string]any); ok {
					propType, _ = prop["type"].(string)
					enumVals, _ = prop["enum"].([]any)
				}
				if len(enumVals) > 0 {
					// Enum-shaped knobs (goal activation among them) must be a
					// valid member; the first member is the affirmative one for
					// the orchestration shapes this probe exercises.
					args[key] = enumVals[0]
					continue
				}
				switch propType {
				case "array":
					args[key] = []any{map[string]any{"objective": "CHILDROUND: reply briefly and finish.", "prompt": "CHILDROUND: reply briefly and finish.", "goal": "CHILDROUND: reply briefly and finish.", "task": "CHILDROUND: reply briefly and finish.", "name": "qa-phase"}}
				case "number", "integer":
					args[key] = 1
				case "boolean":
					// Activation-shaped flags must assert: a goal created
					// inactive never orchestrates child runs.
					args[key] = strings.Contains(strings.ToLower(key), "activ") || strings.Contains(strings.ToLower(key), "enabl") || strings.Contains(strings.ToLower(key), "start")
				default:
					args[key] = "CHILDROUND: reply briefly and finish."
				}
			}
			raw, _ := json.Marshal(args)
			emit("content_block_start", map[string]any{"index": 0, "content_block": map[string]any{"type": "tool_use", "id": fmt.Sprintf("qa-wf-%d", m.requests.Load()), "name": tool.Name, "input": map[string]any{}}})
			emit("content_block_delta", map[string]any{"index": 0, "delta": map[string]any{"type": "input_json_delta", "partial_json": string(raw)}})
			emit("content_block_stop", map[string]any{"index": 0})
			emit("message_delta", map[string]any{"delta": map[string]any{"stop_reason": "tool_use", "stop_sequence": nil}, "usage": map[string]int{"output_tokens": 20}})
			emit("message_stop", map[string]any{})
			return
		}
		if strings.Contains(text, "QA_THINKING") {
			// Anthropic thinking block: exercises whether the Host translates
			// reasoning-shaped content into the journal events the desktop
			// projects as thinking-kind messages.
			emit("content_block_start", map[string]any{"index": 0, "content_block": map[string]any{"type": "thinking", "thinking": ""}})
			emit("content_block_delta", map[string]any{"index": 0, "delta": map[string]any{"type": "thinking_delta", "thinking": "QA_THINKING_TRACE deterministic reasoning."}})
			emit("content_block_stop", map[string]any{"index": 0})
			emit("content_block_start", map[string]any{"index": 1, "content_block": map[string]any{"type": "text", "text": ""}})
			emit("content_block_delta", map[string]any{"index": 1, "delta": map[string]any{"type": "text_delta", "text": "QA_REPLY_OK deterministic reply."}})
			emit("content_block_stop", map[string]any{"index": 1})
			emit("message_delta", map[string]any{"delta": map[string]any{"stop_reason": "end_turn", "stop_sequence": nil}, "usage": map[string]int{"output_tokens": 30}})
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
	scenario := os.Getenv("WHALEISLE_COMPOSER_QA_SCENARIO")
	draftConflictOnly := scenario == "draft-conflict"
	workSurfacesOnly := scenario == "work-surfaces" || draftConflictOnly
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
	// A git baseline lets the diff scenario exercise the real work surface:
	// init+commit, then scenarios modify tracked files and open the diff pane.
	for _, args := range [][]string{
		{"init", "-q"},
		{"add", "-A"},
		{"-c", "user.name=qa", "-c", "user.email=qa@qa.local", "commit", "-qm", "baseline"},
	} {
		gitCmd := exec.Command("git", args...)
		gitCmd.Dir = workspace
		if gErr := gitCmd.Run(); gErr != nil {
			return fmt.Errorf("qa workspace git %v: %w", args, gErr)
		}
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
		c.AutoDownloadUpdates = false
	}); err != nil {
		return err
	}
	defer func() { _ = os.RemoveAll(filepath.Join(data, "updates")) }()
	type step struct {
		Name   string `json:"name"`
		Pass   bool   `json:"pass"`
		Detail string `json:"detail"`
	}
	var steps []step
	mark := func(name string, pass bool, detail string) {
		steps = append(steps, step{Name: name, Pass: pass, Detail: detail})
		log.Printf("qa-mark %s pass=%v %s", name, pass, detail)
	}
	result := make(chan error, 1)
	finish := func(err error) {
		state.mu.Lock()
		sup := state.sup
		state.mu.Unlock()
		var hostLog []string
		if sup != nil {
			if lines := sup.Logs(); len(lines) > 40 {
				hostLog = lines[len(lines)-40:]
			} else {
				hostLog = lines
			}
		}
		raw, _ := json.MarshalIndent(map[string]any{"pid": os.Getpid(), "steps": steps, "modelRequests": model.requests.Load(), "hostLog": hostLog, "error": fmt.Sprint(err)}, "", "  ")
		_ = os.WriteFile(out, append(raw, '\n'), 0o600)
		result <- err
		if sup != nil {
			sup.Stop()
		}
		state.mu.Lock()
		state.quitting = true
		state.quitCleanupFinished = true
		state.mu.Unlock()
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
			state.mu.Lock()
			bootLogs := state.bootLogSnapshot()
			state.mu.Unlock()
			if n := len(bootLogs); n > 8 {
				bootLogs = bootLogs[n-8:]
			}
			finish(fmt.Errorf("owned composer QA readiness timeout (win=%v state=%s endpoint=%s) boot=%q", win != nil, lastState, lastEndpoint, strings.Join(bootLogs, " | ")))
			return
		}
		state.revealMainConversation()
		if !isolatedDesktopHidden() {
			win.ShowInactive()
		}
		if _, err := win.CapturePage(); err != nil {
			finish(err)
			return
		}
		ctx := context.Background()
		if err := waitStore(deadline, "workspace ready", func() (bool, error) {
			v := sessions.Snapshot()
			if v.ListErr != nil {
				return false, errors.New(v.ListErr.SentenceText())
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
			v := sessions.Snapshot()
			// Blank-reuse is a correct create outcome over an untouched blank
			// session: the gesture adopts it instead of proliferating empty
			// identities, so the selected id legitimately stays put.
			reused := v.SelectedID == baseID && v.LastAction == "blank session reused"
			return v.SelectedID != "" && (v.SelectedID != baseID || reused), nil
		}); err != nil {
			finish(err)
			return
		}
		id1 := sessions.Snapshot().SelectedID
		// A second create gesture while id1 is still blank would correctly
		// reuse it and collapse the two fixture identities into one. Send one
		// real loopback message first so the Host no longer considers id1
		// blank. Hero-state submits are dropped until the workspace evidence
		// settles, so gate on the exact predicate the submit path consults.
		if err := waitStore(deadline, "workspace evidence admits id1", func() (bool, error) {
			snap := sessions.Snapshot()
			return snap.WorkspaceReady && !composer.RequiresWorkspace(snap, true), nil
		}); err != nil {
			finish(err)
			return
		}
		seedText := "QA_SEED_ID1"
		probeThinking := scenario == "probe-thinking"
		probeWorkflow := scenario == "probe-workflow"
		probeThinking = probeThinking || probeWorkflow
		if probeWorkflow {
			seedText = "QA_SEED_ID1 QA_WORKFLOW"
		} else if probeThinking {
			seedText = "QA_SEED_ID1 QA_THINKING"
		}
		mygo.RunOnMain(func() {
			state.mu.Lock()
			draft := state.composerDraftLocked(id1)
			draft.Text = seedText
			if probeThinking {
				// Path-based image attachment: the prepare chain loads the bytes,
				// detects the media type and stamps session/host scope itself.
				pngPath := filepath.Join(workspace, "qa-probe.png")
				pngRaw, _ := base64.StdEncoding.DecodeString("iVBORw0KGgoAAAANSUhEUgAAAAEAAAABCAYAAAAfFcSJAAAADUlEQVR42mP8z8BQDwAEhQGAhKmMIQAAAABJRU5ErkJggg==")
				_ = os.MkdirAll(workspace, 0755)
				_ = os.WriteFile(pngPath, pngRaw, 0600)
				draft.Attachments = []session.ComposerAttachment{{Path: pngPath, Name: "qa-probe.png"}}
			}
			state.mu.Unlock()
			state.submitComposer(win, sessions, draft, id1, workspace, false, sessions.Snapshot().ComposerHostGeneration)
		})
		if err := waitStore(deadline, "seed message in id1", func() (bool, error) {
			v := sessions.Snapshot()
			if v.SelectedID != id1 {
				return false, nil
			}
			userCount, replyCount := 0, 0
			for _, m := range v.Messages {
				if m.Role == "user" && strings.Contains(m.Body, "QA_SEED_ID1") {
					userCount++
				}
				if m.Role == "assistant" && (strings.Contains(m.Body, "QA_REPLY_OK") || strings.Contains(m.Body, "QA_APPROVED_OK")) {
					replyCount++
				}
			}
			return userCount >= 1 && replyCount >= 1 && !v.PromptBusy, nil
		}); err != nil {
			finish(err)
			return
		}
		if probeThinking {
			// Goal-driven workflow runs orchestrate asynchronously past the end
			// of the parent turn; let the child rounds settle before sampling.
			if probeWorkflow {
				settle := time.Now().Add(25 * time.Second)
				for time.Now().Before(settle) {
					if len(sessions.Snapshot().WorkflowRuns) > 0 {
						break
					}
					time.Sleep(500 * time.Millisecond)
				}
			}
			type probeMessage struct {
				Kind     string `json:"kind"`
				Role     string `json:"role"`
				Body     string `json:"body"`
				ImageRef string `json:"imageRef,omitempty"`
			}
			v := sessions.Snapshot()
			probe := map[string]any{"messages": []probeMessage{}, "workflowRuns": len(v.WorkflowRuns)}
			if schema, ok := model.wfSchema.Load().(string); ok {
				probe["workflowToolSchema"] = schema
			}
			for _, m := range v.Messages {
				body := m.Body
				if len(body) > 120 {
					body = body[:120]
				}
				probe["messages"] = append(probe["messages"].([]probeMessage), probeMessage{Kind: string(m.Kind), Role: m.Role, Body: body, ImageRef: m.ImageRef})
			}
			raw, _ := json.MarshalIndent(probe, "", "  ")
			_ = os.WriteFile(out+".probe.json", raw, 0600)
			finish(nil)
			return
		}
		sessions.CreateSession(ctx)
		createAttempts := 0
		lastCreate := time.Time{}
		if err := waitStore(deadline, "session 2 created", func() (bool, error) {
			watch()
			v := sessions.Snapshot()
			if v.SelectedID != id1 && v.SelectedID != "" {
				return true, nil
			}
			// A create gesture issued while the previous selection still
			// settles can be dropped by the workspace-target owner fence; the
			// real user's recovery is issuing the gesture again.
			if time.Since(lastCreate) > 4*time.Second {
				sessions.CreateSession(ctx)
				lastCreate = time.Now()
				createAttempts++
			}
			return false, nil
		}); err != nil {
			keys := []string{}
			for action := range seenActions {
				keys = append(keys, action)
			}
			finish(fmt.Errorf("%w (attempts=%d last=%q actions=%v sel=%q items=%d)", err, createAttempts, sessions.Snapshot().LastAction, keys, sessions.Snapshot().SelectedID, len(sessions.Snapshot().Items)))
			return
		}
		id2 := sessions.Snapshot().SelectedID
		mark("create-two-sessions", id1 != "" && id2 != "" && id1 != id2, fmt.Sprintf("id1=%s id2=%s createdSignalSampled=%v", id1, id2, seenActions["session created"]))
		if !workSurfacesOnly {
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
			state.submitComposer(win, sessions, d1, id1, workspace, false, gen)
			// The originating transcript is deliberately detached before admission
			// completes. Confirmation belongs to the request, not this selection.
			sessions.Select(id2)
			time.Sleep(150 * time.Millisecond)
			state.mu.Lock()
			if d := state.composerDrafts[id1]; d != nil {
				d.Text = "typed while sending"
			}
			state.mu.Unlock()
			if err := waitStore(time.Now().Add(25*time.Second), "background attachment handoff", func() (bool, error) {
				state.mu.Lock()
				d := state.composerDrafts[id1]
				pending := d != nil && len(d.PendingSubmissionFlights()) > 0
				state.mu.Unlock()
				return sessions.SelectedSessionID() == id2 && !pending, nil
			}); err != nil {
				var pendingCount int
				var draftError, phase string
				mygo.RunOnMain(func() {
					state.mu.Lock()
					defer state.mu.Unlock()
					if d := state.composerDrafts[id1]; d != nil {
						pendingCount, draftError = len(d.PendingSubmissionFlights()), d.Error
					}
					if state.resumeStore != nil {
						var recovery session.SubmissionRecovery
						if json.Unmarshal(state.resumeStore.Snapshot().Submissions, &recovery) == nil {
							for _, record := range recovery.Records {
								if record.SessionID == id1 {
									phase = record.Phase
								}
							}
						}
					}
				})
				finish(fmt.Errorf("%w (selected=%s expected=%s pending=%d phase=%s draftError=%q)", err, sessions.SelectedSessionID(), id2, pendingCount, phase, draftError))
				return
			}
			mark("attachment-confirmation-after-switch", true, "A confirmed without its visible transcript; B remained selected")
			sessions.Select(id1)
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
			state.submitComposer(win, sessions, d2, id2, workspace, false, gen)
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
			state.submitComposer(win, sessions, d2, id2, workspace, false, sessions.Snapshot().ComposerHostGeneration)
			time.Sleep(400 * time.Millisecond)
			staleOK := d2.Error != "" && d2.Text == "stale edit attempt"
			mark("stale-edit-rejected-draft-kept", staleOK, fmt.Sprintf("err=%q text=%q", d2.Error, d2.Text))
			state.mu.Lock()
			d2.EditSeq = 0
			d2.Error = ""
			d2.Text = "QA_HELLO retry after rejection"
			state.mu.Unlock()
			state.submitComposer(win, sessions, d2, id2, workspace, false, sessions.Snapshot().ComposerHostGeneration)
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
			state.submitComposer(win, sessions, dA, id1, workspace, false, sessions.Snapshot().ComposerHostGeneration)
			approvalSeen := ""
			var approveErr error
			approveDeadline := time.Now().Add(40 * time.Second)
			for time.Now().Before(approveDeadline) {
				if approvalSeen == "" {
					for _, p := range state.interactions.Snapshot().Value.Pending {
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
				for _, p := range state.interactions.Snapshot().Value.Pending {
					if p.EventID == approvalSeen {
						cleared = false
					}
				}
				mark("one-time-approval", cleared, fmt.Sprintf("pending approval %s resolved once; tool result returned; pending cleared=%v", approvalSeen, cleared))
			}
		}
		// --- optional sustained-stream workload for frame-cost sampling ---
		if n, _ := strconv.Atoi(os.Getenv("WHALEISLE_COMPOSER_QA_WORKLOAD")); n > 0 || workSurfacesOnly {
			if workSurfacesOnly {
				n = 0
			}
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
				state.submitComposer(win, sessions, d, id1, workspace, false, sessions.Snapshot().ComposerHostGeneration)
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
			if n > 0 {
				mark("workload-streams", true, fmt.Sprintf("%d long turns; visible=%v min=%v holdFrames(interaction=%v,sizeBurst=%v) frames=%d→%d | base[%s] mid[%s] end[req=%d paint=%d]", n, visible, minimized, interaction, sizeBurst, framesBefore, framesAfter, baseSnap, midSnap, reqCount, paintCount))
			}
			// Hidden windows deliberately do not paint on invalidation. These
			// work surfaces consume async results during render, so drive that
			// same render explicitly without presenting a window to the user.
			var driver *ui.WindowDriver
			mygo.RunOnMain(func() { driver = ui.DriveWindow(win) })
			if driver == nil {
				finish(errors.New("live window has no UI runtime"))
				return
			}
			workSurfaceFrame := func() error {
				mygo.RunOnMain(func() { driver.Frame() })
				return nil
			}
			_ = workSurfaceFrame()
			// --- file + diff work surface (batch 3 scope) ---
			// Drive the same dock entry a tree/reference click uses: the pane's
			// WorkspaceFileState owns the tab; the async workspacefs read
			// publishes Preview on a following frame. Runs on the UI thread so
			// dock mutations stay serialized with renders.
			fd := time.Now().Add(45 * time.Second)
			snap3 := sessions.Snapshot()
			state.dshMu.Lock()
			qaClient := state.dshClient
			state.dshMu.Unlock()
			openDone := make(chan error, 1)
			win.Update(func() {
				fm := files.Model{
					BindingID: fmt.Sprintf("%p\x00%s", qaClient, snap3.SelectedID), SessionID: snap3.SelectedID, WorkspaceRoot: snap3.SelectedCwd,
					Tabs: &state.fileTabs,
					ReadTextPageContext: func(ctx context.Context, rel string, offset int64) (workspacefs.TextPage, error) {
						return workspacefs.ReadRemoteTextPage(ctx, qaClient, snap3.SelectedID, rel, offset)
					},
					ReadRemoteContext: func(ctx context.Context, rel string) (string, error) {
						return workspacefs.ReadRemoteText(ctx, qaClient, rel, workspacefs.DefaultMaxReadBytes)
					},
					StatRemoteContext: func(ctx context.Context, rel string) string {
						return workspacePreviewStat(ctx, qaClient, snap3.SelectedID, rel)
					},
				}
				state.mu.Lock()
				state.filesPaneOpen, state.rightPaneTab = true, "files"
				workspacePanes := state.layout.Workspace
				state.mu.Unlock()
				if workspacePanes == nil {
					openDone <- errors.New("workspace dock not initialized")
					return
				}
				handled, oerr := workspacePanes.OpenFileResource(fm, "qa-note.txt")
				if oerr == nil && !handled {
					oerr = errors.New("OpenFileResource left qa-note.txt unhandled")
				}
				openDone <- oerr
			})
			select {
			case oerr := <-openDone:
				if oerr != nil {
					mark("file-open-preview", false, oerr.Error())
				} else if err := waitStore(fd, "workspace file preview loads", func() (bool, error) {
					if err := workSurfaceFrame(); err != nil {
						return false, err
					}
					state.mu.Lock()
					prev, perr := state.filePreview, state.filePreviewErr
					state.mu.Unlock()
					if perr != "" {
						return false, errors.New(perr)
					}
					return strings.Contains(prev, "qa owned attachment bytes"), nil
				}); err != nil {
					mark("file-open-preview", false, err.Error())
				} else {
					mark("file-open-preview", true, "qa-note.txt resource tab preview loaded through workspacefs read")
				}
			case <-time.After(30 * time.Second):
				mark("file-open-preview", false, "OpenFileResource did not run on the UI thread")
			}
			if !draftConflictOnly {
				// Modify the tracked file after the baseline commit, then route the
				// dock to the diff pane: its workbench runs git.DiffHEAD during
				// render and publishes the text through the same adapter.
				if werr := os.WriteFile(notePath, []byte("qa owned attachment bytes\nqa diff probe line\n"), 0o600); werr != nil {
					mark("diff-surface-load", false, werr.Error())
				} else {
					win.Update(func() {
						state.mu.Lock()
						state.filesPaneOpen, state.rightPaneTab = true, "diff"
						state.mu.Unlock()
					})
					if err := waitStore(fd, "diff surface loads git output", func() (bool, error) {
						if err := workSurfaceFrame(); err != nil {
							return false, err
						}
						state.mu.Lock()
						text, derr := state.diffText, state.diffErr
						state.mu.Unlock()
						if derr != "" {
							return false, errors.New(derr)
						}
						return strings.Contains(text, "qa diff probe line"), nil
					}); err != nil {
						mark("diff-surface-load", false, err.Error())
					} else {
						mark("diff-surface-load", true, "git.DiffHEAD output reached diff surface")
					}
				}
			}
			// Dirty draft lifecycle: drive the pane's WorkspaceFileState — the
			// render binds source.Tabs = &pane.Files.Tabs, so the FileTab there
			// is the authoritative owner of Editing/Draft (tab.Dirty() compares
			// tab.Draft != tab.Preview). Mirrors are published downstream.
			dirtyDone := make(chan bool, 1)
			mygo.RunOnMain(func() {
				state.mu.Lock()
				state.filesPaneOpen, state.rightPaneTab = true, "files"
				workspacePanes := state.layout.Workspace
				state.mu.Unlock()
				if workspacePanes == nil {
					dirtyDone <- false
					return
				}
				_, _ = workspacePanes.OpenFileResource(files.Model{BindingID: fmt.Sprintf("%p\x00%s", qaClient, snap3.SelectedID),
					SessionID: snap3.SelectedID, WorkspaceRoot: snap3.SelectedCwd, Tabs: &state.fileTabs}, "qa-note.txt")
				driver.Frame()
				pane := findFileResourcePane(workspacePanes, "qa-note.txt")
				if pane == nil || workspacePanes.Active != pane.ID {
					dirtyDone <- false
					return
				}
				seeded := seedRecoveredFileDraft(pane, "qa-note.txt", snap3.SelectedCwd, "qa local edit draft\n")
				if seeded {
					// Use the same recovery projection entry as a restored pane;
					// do not splice its tab set into another pane's adapter.
					state.mu.Lock()
					workspacePanes.PublishSessionProjection(state.workspaceProjectionLocked())
					state.mu.Unlock()
				}
				dirtyDone <- seeded
			})
			fd2 := time.Now().Add(45 * time.Second)
			select {
			case injected := <-dirtyDone:
				if !injected {
					mark("file-edit-switch-return", false, "no open FileTab for qa-note.txt on the files pane")
				} else if err := waitStore(fd2, "file draft published from pane state", func() (bool, error) {
					if err := workSurfaceFrame(); err != nil {
						return false, err
					}
					state.mu.Lock()
					e, d := state.fileEditing, state.fileEditDraft
					state.mu.Unlock()
					return e && strings.Contains(d, "qa local edit draft"), nil
				}); err != nil {
					mark("file-edit-switch-return", false, err.Error())
				} else {
					win.Update(func() { state.mu.Lock(); state.rightPaneTab = "diff"; state.mu.Unlock() })
					if err := workSurfaceFrame(); err != nil {
						finish(err)
						return
					}
					time.Sleep(500 * time.Millisecond)
					// The "files" route selects the browser pane, not the open
					// file: selectKind skips resource-owning files panes. The
					// real return gesture is activating the file's dock tab,
					// which OpenFileResource performs on the existing resource.
					reactDone := make(chan string, 1)
					win.Update(func() {
						state.mu.Lock()
						state.rightPaneTab = "files"
						workspacePanes := state.layout.Workspace
						state.mu.Unlock()
						if workspacePanes == nil {
							reactDone <- "workspace dock not initialized"
							return
						}
						fm2 := files.Model{
							BindingID: fmt.Sprintf("%p\x00%s", qaClient, snap3.SelectedID), SessionID: snap3.SelectedID, WorkspaceRoot: snap3.SelectedCwd,
							Tabs: &state.fileTabs,
						}
						handled, rerr := workspacePanes.OpenFileResource(fm2, "qa-note.txt")
						if rerr == nil && !handled {
							rerr = errors.New("OpenFileResource did not reactivate qa-note.txt")
						}
						if rerr != nil {
							reactDone <- rerr.Error()
							return
						}
						reactDone <- ""
					})
					fd2b := time.Now().Add(45 * time.Second)
					seenDirty, seenE, seenSel := false, false, ""
					seenD := ""
					paneFound := false
					select {
					case rerr := <-reactDone:
						if rerr != "" {
							mark("file-edit-switch-return", false, rerr)
						} else if err := waitStore(fd2b, "file edit draft survives pane switching", func() (bool, error) {
							if err := workSurfaceFrame(); err != nil {
								return false, err
							}
							dirty := false
							if pane := findFileResourcePane(state.layout.Workspace, "qa-note.txt"); pane != nil && pane.Files != nil {
								paneFound = true
								if ws := pane.Files.Tabs.Workspaces[filepath.Clean(snap3.SelectedCwd)]; ws != nil {
									for _, t := range ws.Open {
										if t != nil && t.Path == "qa-note.txt" && t.Dirty() {
											dirty = true
										}
									}
								}
							}
							seenDirty = dirty
							state.mu.Lock()
							seenE, seenD, seenSel = state.fileEditing, state.fileEditDraft, state.selectedFile
							state.mu.Unlock()
							return dirty && seenE && strings.Contains(seenD, "qa local edit draft") && seenSel == "qa-note.txt", nil
						}); err != nil {
							mark("file-edit-switch-return", false, fmt.Sprintf("%v (paneFound=%v dirty=%v editing=%v draft=%q sel=%q)", err, paneFound, seenDirty, seenE, seenD, seenSel))
						} else {
							disk, _ := os.ReadFile(notePath)
							mark("file-edit-switch-return", !strings.Contains(string(disk), "qa local edit draft"), "dirty draft kept across diff/files route switch; disk not implicitly saved")
						}
					case <-time.After(20 * time.Second):
						mark("file-edit-switch-return", false, "file pane reactivation did not run on the UI thread")
					}
				}
			case <-time.After(20 * time.Second):
				mark("file-edit-switch-return", false, "draft injection did not run on the UI thread")
			}
			// Refresh a dirty file after a real external write. The conflict UI
			// must retain the original save baseline and local draft separately
			// from the new disk version, without implicitly overwriting the file.
			if werr := os.WriteFile(notePath, []byte("external disk change\n"), 0o600); werr != nil {
				mark("external-edit-conflict", false, werr.Error())
			} else {
				snap4 := sessions.Snapshot()
				reloadDone := make(chan string, 1)
				win.Update(func() {
					state.mu.Lock()
					state.filesPaneOpen, state.rightPaneTab = true, "files"
					workspacePanes := state.layout.Workspace
					state.mu.Unlock()
					if workspacePanes != nil {
						fm2 := files.Model{
							BindingID: fmt.Sprintf("%p\x00%s", qaClient, snap4.SelectedID), SessionID: snap4.SelectedID, WorkspaceRoot: snap4.SelectedCwd,
							Tabs: &state.fileTabs,
						}
						// Consume activation before starting the read. A hidden
						// pane's Pause must not own this new refresh operation.
						_, _ = workspacePanes.OpenFileResource(fm2, "qa-note.txt")
					}
					driver.Frame()
					pane := findFileResourcePane(workspacePanes, "qa-note.txt")
					if pane == nil || pane.Files == nil {
						reloadDone <- "no files pane"
						return
					}
					// The page command resolves focusedResource and invokes the
					// renderer's actual fileModel, including its read owner.
					if err := driver.Click("File preview toolbar"); err != nil {
						reloadDone <- err.Error()
						return
					}
					driver.PhysicalKey("KeyR", ui.KeyR, ui.Cmd)
					reloadDone <- ""
				})
				select {
				case rerr := <-reloadDone:
					if rerr != "" {
						mark("external-edit-conflict", false, rerr)
					} else {
						lastSt, lastD, lastP := "", "", ""
						tabPrev, tabDraft, tabStatus, tabErr := "", "", "", ""
						if err := waitStore(time.Now().Add(45*time.Second), "external edit conflict keeps draft", func() (bool, error) {
							if err := workSurfaceFrame(); err != nil {
								return false, err
							}
							state.mu.Lock()
							lastSt, lastD, lastP = state.fileEditStatus, state.fileEditDraft, state.filePreview
							state.mu.Unlock()
							if pane := findFileResourcePane(state.layout.Workspace, "qa-note.txt"); pane != nil && pane.Files != nil {
								if ws := pane.Files.Tabs.Workspaces[filepath.Clean(snap4.SelectedCwd)]; ws != nil {
									for _, t := range ws.Open {
										if t != nil && t.Path == "qa-note.txt" {
											tabPrev, tabDraft, tabStatus, tabErr = t.Preview, t.Draft, t.Status, t.Error
										}
									}
								}
							}
							var visibleConflict bool
							mygo.RunOnMain(func() {
								visibleConflict = driver.HasText("外部修改与未保存草稿冲突") || driver.HasText("External edit conflicts with unsaved draft")
							})
							if visibleConflict && (strings.Contains(lastSt, "草稿") || strings.Contains(strings.ToLower(lastSt), "draft")) {
								return strings.Contains(lastD, "qa local edit draft"), nil
							}
							return false, nil
						}); err != nil {
							mark("external-edit-conflict", false, fmt.Sprintf("%v (status=%q draft=%q preview=%q | tab preview=%q draft=%q status=%q err=%q)", err, lastSt, lastD, lastP, tabPrev, tabDraft, tabStatus, tabErr))
						} else if disk, _ := os.ReadFile(notePath); !strings.Contains(string(disk), "external disk change") {
							mark("external-edit-conflict", false, "conflict path implicitly overwrote the external disk change")
						} else {
							if lastP != "qa owned attachment bytes\n" && draftConflictOnly || tabPrev != lastP {
								mark("external-edit-conflict", false, fmt.Sprintf("refresh replaced original save baseline: preview=%q tab=%q", lastP, tabPrev))
							} else {
								mark("external-edit-conflict", true, "visible conflict after physical page refresh; original save baseline and local draft kept; external disk change not overwritten")
							}
						}
					}
				case <-time.After(30 * time.Second):
					mark("external-edit-conflict", false, "ReloadSelectedFile did not run on the UI thread")
				}
			}
			if !draftConflictOnly {
				// Staged-vs-unstaged reflection at the same git layer the workbench
				// calls; the per-change stage menu is render-internal and stays a
				// declared gap until a UI driver reaches it.
				if out, gerr := exec.Command("git", "-C", workspace, "add", "qa-note.txt").CombinedOutput(); gerr != nil {
					mark("diff-stage-reflect", false, fmt.Sprintf("git add: %v %s", gerr, out))
				} else {
					changes, cerr := git.Changes(workspace)
					staged := false
					for _, ch := range changes {
						if ch.Path == "qa-note.txt" && ch.Staged() {
							staged = true
						}
					}
					scoped, derr := git.DiffScope(workspace, "", true)
					headCount, _ := exec.Command("git", "-C", workspace, "rev-list", "--count", "HEAD").Output()
					ok := cerr == nil && derr == nil && staged && strings.Contains(scoped, "external disk change") && strings.TrimSpace(string(headCount)) == "1"
					mark("diff-stage-reflect", ok, fmt.Sprintf("staged=%v scopedHasContent=%v commits=%s (no implicit commit)", staged, strings.Contains(scoped, "external disk change"), strings.TrimSpace(string(headCount))))
				}
				// Terminal drawer: restartTerminal is the real open gesture (it
				// sets terminalVisible + layout.Bottom itself); ensureTerminal is
				// only the already-open refresh path. The panel must survive the
				// dock route switching back to files.
				termDone := make(chan bool, 1)
				win.Update(func() {
					state.restartTerminal(snap3.SelectedCwd, mygo.Theme.IsDark())
					state.mu.Lock()
					termDone <- state.terminalPanel != nil
					state.mu.Unlock()
				})
				select {
				case hasPanel := <-termDone:
					if !hasPanel {
						mark("terminal-route", false, "restartTerminal produced no panel")
					} else {
						time.Sleep(800 * time.Millisecond)
						win.Update(func() {
							state.mu.Lock()
							state.filesPaneOpen, state.rightPaneTab = true, "files"
							termDone <- state.terminalPanel != nil
							state.mu.Unlock()
						})
						select {
						case alive := <-termDone:
							mark("terminal-route", alive, "terminal panel mounted on workspace root; routed back to files")
						case <-time.After(20 * time.Second):
							mark("terminal-route", false, "route-back did not run on the UI thread")
						}
					}
				case <-time.After(20 * time.Second):
					mark("terminal-route", false, "restartTerminal did not run on the UI thread")
				}
				win.Update(func() {
					state.mu.Lock()
					state.filesPaneOpen = false
					state.mu.Unlock()
				})
				if !workSurfacesOnly {
					// Creator owns a confirmed fresh cordis conversation. The
					// original conversation and draft survive; preparation never sends.
					before := model.requests.Load()
					sourceID := sessions.SelectedSessionID()
					state.mu.Lock()
					originalDraft := string(state.composerDraftLocked(sourceID).SaveDraft())
					state.mu.Unlock()
					// Called off the UI thread: the function schedules its own
					// win.Update; nesting it inside one would deadlock.
					state.startCreatorPluginDraft(win, sessions, "leftbar")
					if err := waitStore(time.Now().Add(20*time.Second), "creator draft seeded", func() (bool, error) {
						targetID := sessions.SelectedSessionID()
						state.mu.Lock()
						defer state.mu.Unlock()
						d := state.composerDrafts[targetID]
						if string(state.composerDraftLocked(sourceID).SaveDraft()) != originalDraft {
							return false, fmt.Errorf("Creator changed the original conversation draft")
						}
						return targetID != "" && targetID != sourceID && d != nil && strings.Contains(d.Text, "带左侧栏入口") && state.agentPresetPick == "cordis", nil
					}); err != nil {
						mark("extension-panel-creator", false, fmt.Sprintf("%v (preset=%q)", err, state.agentPresetPick))
					} else {
						time.Sleep(500 * time.Millisecond)
						mark("extension-panel-creator", model.requests.Load() == before, fmt.Sprintf("draft seeded; sent requests unchanged=%v", model.requests.Load() == before))
					}
				}
				// Creator now selects its confirmed fresh target. The following
				// existing input checks explicitly return to their original owner.
				sessions.Select(id1)
				if err := workSurfaceFrame(); err != nil {
					finish(err)
					return
				}
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
	case <-time.After(10 * time.Second):
		return fmt.Errorf("native desktop exited before composer QA proof")
	}
}
