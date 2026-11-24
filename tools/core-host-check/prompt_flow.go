package main

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"whaleisle/internal/desktop/proactive"
	"whaleisle/internal/dsh"
	"whaleisle/internal/dsh/api"
	"whaleisle/internal/harness"
	"whaleisle/internal/harness/installctrl"
	"whaleisle/internal/paths"
	"whaleisle/internal/processutil"
	"whaleisle/internal/store/session"
)

const (
	ownedFileText = "QA_OWNED_FILE_CONTENT_20261102: real file read through the Host tool."
	helloPrompt   = "QA_HELLO: return the deterministic local response."
	toolPrompt    = "QA_TOOL: read qa-note.txt using the file read tool, then report its result."
	waitPrompt    = "QA_WAIT: stream until the real session is stopped."
	nextPrompt    = "QA_NEXT: complete a fresh turn after the stop."

	writeGuardPrompt = "QA_WRITE_GUARD: overwrite qa-unread.txt with the write tool."
	staleEditPrompt  = "QA_EDIT_STALE: read qa-stale.txt, then edit its first line."
	unreadFileText   = "QA_UNREAD_ORIGINAL: a file this session never read.\n"
	staleFileMarker  = "QA_STALE_ORIGINAL"
	staleFileText    = staleFileMarker + ": the line the session read first.\n"
	staleEditText    = "QA_STALE_EDITED: the line the session asked to write.\n"
	staleChangeText  = "QA_STALE_CHANGED: this line appeared after the read.\n"
)

// fixtureTool is one tool advertisement the loopback model answers against.
type fixtureTool struct {
	Name        string `json:"name"`
	InputSchema struct {
		Properties map[string]json.RawMessage `json:"properties"`
	} `json:"input_schema"`
}

// pathArgs builds arguments from the tool's own schema so the call validates.
func (t fixtureTool) pathArgs(path string, extra map[string]string) map[string]any {
	out := map[string]any{}
	for key := range t.InputSchema.Properties {
		switch strings.ToLower(key) {
		case "path", "file_path", "filepath", "filename":
			out[key] = path
		case "offset":
			out[key] = 1
		case "limit":
			out[key] = 100
		}
	}
	for name, value := range extra {
		for key := range t.InputSchema.Properties {
			if strings.ToLower(key) == name {
				out[key] = value
			}
		}
	}
	return out
}

// pickTool returns the first advertised tool whose lowercased name matches.
func pickTool(tools []fixtureTool, match func(string) bool) (fixtureTool, bool) {
	for _, tool := range tools {
		if match(strings.ToLower(tool.Name)) {
			return tool, true
		}
	}
	return fixtureTool{}, false
}

func readToolName(name string) bool {
	return name == "read" || strings.Contains(name, "read_file") || strings.Contains(name, "fileread") || strings.Contains(name, "fs_read")
}

func writeToolName(name string) bool {
	return name == "write" || strings.Contains(name, "write_file") || strings.Contains(name, "fs_write")
}

func editToolName(name string) bool {
	return name == "edit" || strings.Contains(name, "edit_file") || strings.Contains(name, "fs_edit")
}

// modelFixture implements only the deterministic Messages responses used by
// this opt-in check. It neither imports a model SDK nor forwards HTTP requests.
type modelFixture struct {
	server               *http.Server
	address              string
	log                  *os.File
	mu                   sync.Mutex
	requests, canceled   int
	readTool             string
	toolResult           bool
	writeTool            string
	writeRefused         bool
	writeGuard           bool
	editTool             string
	staleRefused         bool
	staleEdit            bool
	staleStep            int
	taskWait             bool
	taskWaiting          bool
	taskInputMismatch    bool
	waitStarted          bool
	releaseProactiveWait bool
}

func newModelFixture(root, workspace string) (*modelFixture, error) {
	l, err := net.Listen("tcp4", "127.0.0.1:0")
	if err != nil {
		return nil, err
	}
	log, err := os.OpenFile(filepath.Join(root, "model-requests.jsonl"), os.O_CREATE|os.O_EXCL|os.O_WRONLY, 0600)
	if err != nil {
		l.Close()
		return nil, err
	}
	f := &modelFixture{address: l.Addr().String(), log: log}
	f.server = &http.Server{ReadHeaderTimeout: 5 * time.Second}
	f.server.Handler = http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
		var body struct {
			Model    string `json:"model"`
			Messages []struct {
				Role    string          `json:"role"`
				Content json.RawMessage `json:"content"`
			} `json:"messages"`
			Tools []fixtureTool `json:"tools"`
		}
		if req.Method != "POST" {
			http.Error(w, "owned model accepts POST only", 405)
			return
		}
		if err := json.NewDecoder(http.MaxBytesReader(w, req.Body, 4<<20)).Decode(&body); err != nil {
			http.Error(w, err.Error(), 400)
			return
		}
		last := ""
		if len(body.Messages) > 0 {
			last = string(body.Messages[len(body.Messages)-1].Content)
		}
		f.mu.Lock()
		f.requests++
		id := f.requests
		f.mu.Unlock()
		names := []string{}
		for _, tool := range body.Tools {
			names = append(names, tool.Name)
		}
		f.record(map[string]any{"kind": "request", "id": id, "route": req.URL.Path, "model": body.Model, "lastContent": last, "tools": names})
		w.Header().Set("Content-Type", "text/event-stream")
		w.Header().Set("Cache-Control", "no-cache")
		w.WriteHeader(200)
		event := func(name string, data map[string]any) {
			data["type"] = name
			raw, _ := json.Marshal(data)
			fmt.Fprintf(w, "event: %s\ndata: %s\n\n", name, raw)
			w.(http.Flusher).Flush()
		}
		event("message_start", map[string]any{"message": map[string]any{"id": fmt.Sprintf("owned-model-%d", id), "type": "message", "role": "assistant", "model": body.Model, "content": []any{}, "stop_reason": nil, "stop_sequence": nil, "usage": map[string]int{"input_tokens": 80, "output_tokens": 0}}})
		if slices.Contains(names, "task_read_input") {
			f.mu.Lock()
			wait := f.taskWait
			f.taskWaiting = wait
			f.mu.Unlock()
			if wait {
				event("content_block_start", map[string]any{"index": 0, "content_block": map[string]any{"type": "text", "text": ""}})
				event("content_block_delta", map[string]any{"index": 0, "delta": map[string]any{"type": "text_delta", "text": "QA_TASK_WAIT"}})
				<-req.Context().Done()
				f.mu.Lock()
				f.taskWaiting = false
				f.canceled++
				f.mu.Unlock()
				return
			}
			tool, args := "task_read_input", map[string]any{"inputId": "input-1"}
			if strings.Contains(last, "tool_result") {
				if !strings.Contains(last, ownedFileText) {
					f.record(map[string]any{"kind": "task-input-mismatch", "content": last})
					f.mu.Lock()
					f.taskInputMismatch = true
					f.mu.Unlock()
				}
				tool = "task_report"
				args = map[string]any{"summary": "QA_TASK_SUMMARY: " + ownedFileText, "evidence": []string{"input-1"}, "coveredCriteria": []string{taskCriterion}, "unmet": []string{}, "outcome": "completed"}
			}
			raw, _ := json.Marshal(args)
			event("content_block_start", map[string]any{"index": 0, "content_block": map[string]any{"type": "tool_use", "id": fmt.Sprintf("task-tool-%d", id), "name": tool, "input": map[string]any{}}})
			event("content_block_delta", map[string]any{"index": 0, "delta": map[string]any{"type": "input_json_delta", "partial_json": string(raw)}})
			event("content_block_stop", map[string]any{"index": 0})
			event("message_delta", map[string]any{"delta": map[string]any{"stop_reason": "tool_use", "stop_sequence": nil}, "usage": map[string]int{"output_tokens": 30}})
			event("message_stop", map[string]any{})
			return
		}
		if emitProactiveTool(last, id, workspace, event) {
			return
		}
		if f.emitWriteGuard(body.Tools, last, workspace, id, event) {
			return
		}
		if f.emitStaleEdit(body.Tools, last, workspace, id, event) {
			return
		}
		if strings.Contains(last, "QA_TOOL") && !strings.Contains(last, "tool_result") {
			for _, tool := range body.Tools {
				name := strings.ToLower(tool.Name)
				if name != "read" && !strings.Contains(name, "read_file") && !strings.Contains(name, "fileread") && !strings.Contains(name, "fs_read") {
					continue
				}
				args := map[string]any{}
				for key := range tool.InputSchema.Properties {
					switch strings.ToLower(key) {
					case "path", "file_path", "filepath", "filename":
						args[key] = filepath.Join(workspace, "qa-note.txt")
					case "offset":
						args[key] = 1
					case "limit":
						args[key] = 100
					}
				}
				f.mu.Lock()
				f.readTool = tool.Name
				f.mu.Unlock()
				f.record(map[string]any{"kind": "tool-call", "id": id, "name": tool.Name, "arguments": args})
				event("content_block_start", map[string]any{"index": 0, "content_block": map[string]any{"type": "tool_use", "id": fmt.Sprintf("owned-tool-%d", id), "name": tool.Name, "input": map[string]any{}}})
				raw, _ := json.Marshal(args)
				event("content_block_delta", map[string]any{"index": 0, "delta": map[string]any{"type": "input_json_delta", "partial_json": string(raw)}})
				event("content_block_stop", map[string]any{"index": 0})
				event("message_delta", map[string]any{"delta": map[string]any{"stop_reason": "tool_use", "stop_sequence": nil}, "usage": map[string]int{"output_tokens": 20}})
				event("message_stop", map[string]any{})
				return
			}
		}
		text := "QA_REPLY_OK: deterministic local model response."
		if strings.Contains(last, "tool_result") {
			ok := strings.Contains(last, ownedFileText)
			f.mu.Lock()
			f.toolResult = ok
			f.mu.Unlock()
			f.record(map[string]any{"kind": "tool-result-returned", "id": id, "containsOwnedFile": ok, "content": last})
			if ok {
				text = "QA_TOOL_OK: actual Host file-tool result received."
			} else {
				text = "QA_TOOL_RESULT_MISSING"
			}
		} else if strings.Contains(last, "QA_NEXT") {
			text = "QA_NEXT_OK: next turn after stop completed."
		}
		event("content_block_start", map[string]any{"index": 0, "content_block": map[string]any{"type": "text", "text": ""}})
		f.mu.Lock()
		holdProactive := !f.releaseProactiveWait
		f.mu.Unlock()
		if strings.Contains(last, "QA_WAIT") && holdProactive {
			event("content_block_delta", map[string]any{"index": 0, "delta": map[string]any{"type": "text_delta", "text": "QA_WAIT_STREAM: awaiting real session cancellation."}})
			f.mu.Lock()
			f.waitStarted = true
			f.mu.Unlock()
			select {
			case <-req.Context().Done():
				f.mu.Lock()
				f.canceled++
				f.mu.Unlock()
				f.record(map[string]any{"kind": "stream-canceled", "id": id})
				return
			case <-time.After(45 * time.Second):
				text = "QA_WAIT_NOT_CANCELED"
			}
		}
		mid := len(text) / 2
		for _, part := range []string{text[:mid], text[mid:]} {
			event("content_block_delta", map[string]any{"index": 0, "delta": map[string]any{"type": "text_delta", "text": part}})
			time.Sleep(30 * time.Millisecond)
		}
		event("content_block_stop", map[string]any{"index": 0})
		event("message_delta", map[string]any{"delta": map[string]any{"stop_reason": "end_turn", "stop_sequence": nil}, "usage": map[string]int{"output_tokens": 24}})
		event("message_stop", map[string]any{})
	})
	go f.server.Serve(l)
	return f, nil
}

// streamText completes one assistant turn with text only.
func (f *modelFixture) streamText(text string, event func(string, map[string]any)) {
	event("content_block_start", map[string]any{"index": 0, "content_block": map[string]any{"type": "text", "text": ""}})
	event("content_block_delta", map[string]any{"index": 0, "delta": map[string]any{"type": "text_delta", "text": text}})
	event("content_block_stop", map[string]any{"index": 0})
	event("message_delta", map[string]any{"delta": map[string]any{"stop_reason": "end_turn", "stop_sequence": nil}, "usage": map[string]int{"output_tokens": 18}})
	event("message_stop", map[string]any{})
}

// streamTool completes one assistant turn asking for a single tool call.
func (f *modelFixture) streamTool(id int, name string, args map[string]any, event func(string, map[string]any)) {
	raw, _ := json.Marshal(args)
	event("content_block_start", map[string]any{"index": 0, "content_block": map[string]any{"type": "tool_use", "id": fmt.Sprintf("owned-tool-%d", id), "name": name, "input": map[string]any{}}})
	event("content_block_delta", map[string]any{"index": 0, "delta": map[string]any{"type": "input_json_delta", "partial_json": string(raw)}})
	event("content_block_stop", map[string]any{"index": 0})
	event("message_delta", map[string]any{"delta": map[string]any{"stop_reason": "tool_use", "stop_sequence": nil}, "usage": map[string]int{"output_tokens": 20}})
	event("message_stop", map[string]any{})
}

func refusedMessage(content string) bool {
	return strings.Contains(content, "FS_NOT_OBSERVED") ||
		strings.Contains(content, "FS_STALE_VERSION") ||
		strings.Contains(content, "has not been read") ||
		strings.Contains(content, "without reading it first") ||
		strings.Contains(content, "changed since it was read")
}

// emitWriteGuard asks for a write to a file this session never read, then reports
// whether the Host refused the overwrite.
func (f *modelFixture) emitWriteGuard(tools []fixtureTool, last, workspace string, id int, event func(string, map[string]any)) bool {
	if strings.Contains(last, "QA_WRITE_GUARD") {
		f.mu.Lock()
		f.writeGuard = true
		f.mu.Unlock()
	}
	f.mu.Lock()
	active := f.writeGuard
	f.mu.Unlock()
	if !active {
		return false
	}
	target := filepath.Join(workspace, "qa-unread.txt")
	if strings.Contains(last, "tool_result") {
		refused := refusedMessage(last)
		f.mu.Lock()
		f.writeRefused = refused
		f.writeGuard = false
		f.mu.Unlock()
		f.record(map[string]any{"kind": "write-guard", "id": id, "refused": refused, "content": last})
		if refused {
			f.streamText("QA_WRITE_GUARD_OK: the Host refused a write with no prior read.", event)
		} else {
			f.streamText("QA_WRITE_GUARD_MISSING: the Host overwrote a file this session never read.", event)
		}
		return true
	}
	tool, ok := pickTool(tools, writeToolName)
	if !ok {
		f.mu.Lock()
		f.writeGuard = false
		f.mu.Unlock()
		f.streamText("QA_WRITE_GUARD_MISSING: the Host advertised no write tool.", event)
		return true
	}
	args := tool.pathArgs(target, map[string]string{"content": "overwritten by the owned model"})
	f.mu.Lock()
	f.writeTool = tool.Name
	f.mu.Unlock()
	f.record(map[string]any{"kind": "write-guard-call", "id": id, "name": tool.Name, "arguments": args})
	f.streamTool(id, tool.Name, args, event)
	return true
}

// emitStaleEdit reads a file, changes it behind the session's back, then asks for
// an edit computed from that read.
func (f *modelFixture) emitStaleEdit(tools []fixtureTool, last, workspace string, id int, event func(string, map[string]any)) bool {
	if strings.Contains(last, "QA_EDIT_STALE") {
		f.mu.Lock()
		f.staleEdit = true
		f.mu.Unlock()
	}
	f.mu.Lock()
	active, step := f.staleEdit, f.staleStep
	f.mu.Unlock()
	if !active {
		return false
	}
	target := filepath.Join(workspace, "qa-stale.txt")
	finish := func(refused bool, marker string) bool {
		f.mu.Lock()
		f.staleRefused = refused
		f.staleEdit = false
		f.staleStep = 0
		f.mu.Unlock()
		f.record(map[string]any{"kind": "stale-edit", "id": id, "refused": refused, "content": last})
		f.streamText(marker, event)
		return true
	}
	switch step {
	case 0:
		tool, ok := pickTool(tools, readToolName)
		if !ok {
			return finish(false, "QA_EDIT_STALE_MISSING: the Host advertised no read tool.")
		}
		args := tool.pathArgs(target, nil)
		f.mu.Lock()
		f.staleStep = 1
		f.mu.Unlock()
		f.record(map[string]any{"kind": "stale-edit-read", "id": id, "name": tool.Name, "arguments": args})
		f.streamTool(id, tool.Name, args, event)
		return true
	case 1:
		if refusedMessage(last) {
			return finish(false, "QA_EDIT_STALE_MISSING: the owned read was refused.")
		}
		if !strings.Contains(last, staleFileMarker) {
			return finish(false, "QA_EDIT_STALE_MISSING: the owned read returned no file content.")
		}
		if err := os.WriteFile(target, []byte(staleChangeText), 0o600); err != nil {
			return finish(false, "QA_EDIT_STALE_MISSING: the owned model could not change the file.")
		}
		tool, ok := pickTool(tools, editToolName)
		if !ok {
			return finish(false, "QA_EDIT_STALE_MISSING: the Host advertised no edit tool.")
		}
		args := tool.pathArgs(target, map[string]string{"old_string": strings.TrimSuffix(staleFileText, "\n"), "new_string": strings.TrimSuffix(staleEditText, "\n")})
		f.mu.Lock()
		f.editTool = tool.Name
		f.staleStep = 2
		f.mu.Unlock()
		f.record(map[string]any{"kind": "stale-edit-call", "id": id, "name": tool.Name, "arguments": args})
		f.streamTool(id, tool.Name, args, event)
		return true
	default:
		if refusedMessage(last) {
			return finish(true, "QA_EDIT_STALE_OK: the Host refused an edit computed from a stale read.")
		}
		return finish(false, "QA_EDIT_STALE_MISSING: the Host applied an edit computed from a stale read.")
	}
}

func (f *modelFixture) record(value any) {
	f.mu.Lock()
	defer f.mu.Unlock()
	raw, _ := json.Marshal(value)
	fmt.Fprintln(f.log, string(raw))
}
func (f *modelFixture) close() { f.server.Close(); f.log.Close() }
func closedListener(origin string) bool {
	u, err := url.Parse(origin)
	if err != nil {
		return false
	}
	c, err := net.DialTimeout("tcp", u.Host, time.Second)
	if c != nil {
		c.Close()
	}
	return err != nil
}

func hasOwnedFileResult(v session.ViewState) bool {
	for _, m := range v.Messages {
		if m.EventType != "tool/result" || m.Seq <= 0 {
			continue
		}
		var event struct {
			Data struct {
				Message struct {
					ToolCallID string `json:"toolCallId"`
					IsError    *bool  `json:"isError"`
					Content    []struct {
						Text string `json:"text"`
					} `json:"content"`
				} `json:"message"`
			} `json:"data"`
		}
		if json.Unmarshal(m.Raw, &event) != nil || event.Data.Message.IsError == nil || *event.Data.Message.IsError || !strings.HasPrefix(event.Data.Message.ToolCallID, "owned-tool-") {
			continue
		}
		for _, part := range event.Data.Message.Content {
			if strings.Contains(part.Text, ownedFileText) {
				return true
			}
		}
	}
	return false
}

func runPromptFlow(stage, base string, r *report, queueIdle, dshbotFlow, phoneFlow, whaleTasks, proactiveFlow, acpFlow bool) error {
	stage, err := filepath.Abs(stage)
	if err != nil {
		return err
	}
	r.RuntimeRoot = stage
	metadata, err := os.ReadFile(filepath.Join(stage, "vendor", "deepseek-harness", "package.json"))
	if err != nil {
		return err
	}
	var identity struct {
		Version string `json:"version"`
	}
	if err = json.Unmarshal(metadata, &identity); err != nil {
		return err
	}
	r.HostVersion = identity.Version
	root, err := os.MkdirTemp(base, "core-host-prompt-")
	if err != nil {
		return err
	}
	r.OwnedRunRoot = root
	r.SimulatedModel = true
	userData := filepath.Join(root, "user-data")
	home := filepath.Join(userData, "dsh-home")
	workspace := filepath.Join(root, "workspace")
	ownedHome := filepath.Join(root, "owned-home")
	for _, dir := range []string{home, workspace, ownedHome} {
		if err = os.MkdirAll(dir, 0700); err != nil {
			return err
		}
	}
	for _, key := range []string{"DEEPSEEK_API_KEY", "WHALEISLE_API_KEY", "DSH_API_KEY", "ANTHROPIC_API_KEY", "OPENAI_API_KEY", "NODE_OPTIONS"} {
		os.Unsetenv(key)
	}
	localAppData := filepath.Join(ownedHome, "AppData", "Local")
	for key, value := range map[string]string{paths.EnvUserData: userData, "DSH_HOME": home, "HOME": ownedHome, "USERPROFILE": ownedHome, "APPDATA": filepath.Join(ownedHome, "AppData", "Roaming"), "LOCALAPPDATA": localAppData, "PSModuleAnalysisCachePath": filepath.Join(localAppData, "Microsoft", "Windows", "PowerShell", "ModuleAnalysisCache")} {
		if err = os.Setenv(key, value); err != nil {
			return err
		}
	}
	// The check commonly lives below the implementation checkout. Give its
	// workspace its own repository so the Host's real workspace-change recorder
	// cannot discover and snapshot the enclosing developer repository.
	gitCtx, gitCancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer gitCancel()
	init := processutil.CommandContext(gitCtx, "git", "init", "--quiet", workspace)
	if output, err := init.CombinedOutput(); err != nil {
		return fmt.Errorf("initialize owned workspace repository: %w: %s", err, strings.TrimSpace(string(output)))
	}
	verify := processutil.CommandContext(gitCtx, "git", "-C", workspace, "rev-parse", "--show-toplevel")
	output, err := verify.Output()
	if err != nil {
		return fmt.Errorf("verify owned workspace repository: %w", err)
	}
	if rel, err := filepath.Rel(workspace, strings.TrimSpace(string(output))); err != nil || rel != "." {
		return errors.New("owned workspace repository resolved outside its exact workspace")
	}
	if err = os.WriteFile(filepath.Join(workspace, "qa-note.txt"), []byte(ownedFileText), 0600); err != nil {
		return err
	}
	if err = os.WriteFile(filepath.Join(workspace, "qa-unread.txt"), []byte(unreadFileText), 0600); err != nil {
		return err
	}
	if err = os.WriteFile(filepath.Join(workspace, "qa-stale.txt"), []byte(staleFileText), 0600); err != nil {
		return err
	}
	model, err := newModelFixture(root, workspace)
	if err != nil {
		return err
	}
	defer func() {
		model.close()
		model.mu.Lock()
		r.ModelRequests = model.requests
		model.mu.Unlock()
		r.OwnedModelStopped = true
		r.ModelListenerClosed = closedListener("http://" + model.address)
	}()
	os.Setenv("DEEPSEEK_BASE_URL", "http://"+model.address+"/anthropic")
	os.Setenv("DEEPSEEK_API_KEY", "qa-loopback-only")
	vendor := filepath.Join(stage, "vendor")
	harnessRoot := filepath.Join(vendor, "deepseek-harness")
	node := filepath.Join(stage, "node", "node.exe")
	var acpFixture *acpFixtureState
	if acpFlow {
		acpFixture, err = prepareACPFixture(root, home, vendor, node)
		if err != nil {
			return err
		}
	}
	var patches []string
	if queueIdle {
		patch := filepath.Join(root, "idle-release.patch.yml")
		if err = os.WriteFile(patch, []byte("- id: session-controller\n  config:\n    idleReleaseMs: 1500\n    idleReleaseKeep: 0\n"), 0600); err != nil {
			return err
		}
		patches = []string{patch}
	}
	var proactiveOwner atomic.Pointer[proactive.Service]
	openProactive := func() (*proactive.Service, error) {
		service, err := proactive.Open(userData)
		if err == nil {
			proactiveOwner.Store(service)
		}
		return service, err
	}
	desktop := installctrl.DesktopOps{}
	if proactiveFlow || acpFlow {
		if _, err := openProactive(); err != nil {
			return err
		}
		desktop.Proactive = func(ctx context.Context, action string, payload json.RawMessage) (any, error) {
			return proactiveOwner.Load().Dispatch(ctx, action, payload)
		}
	}
	if proactiveFlow {
		if os.Getenv("WHALEISLE_PROACTIVE_TOOL_ONLY") == "1" {
			patch, err := prepareProactiveToolFixture(root, harnessRoot, workspace)
			if err != nil {
				return err
			}
			patches = append(patches, patch)
		}
		if os.Getenv("WHALEISLE_PROACTIVE_EXTERNAL_DELIVERY") == "1" {
			patch, err := prepareProactiveDeliveryFixture(root, vendor, workspace)
			if err != nil {
				return err
			}
			patches = append(patches, patch)
		}
	}
	sup := harness.New(harness.Options{Install: harness.NewInstallServer(harness.InstallServerOptions{NodeBin: node, Desktop: desktop}), Launch: harness.LaunchOptions{NodeBin: node, HarnessRoot: harnessRoot, VendorDir: vendor, Host: "127.0.0.1", Port: 32280, Home: home, PatchFiles: patches}, Prepare: harness.PrepareOptions{VendorDir: vendor, HarnessRoot: harnessRoot, DshHome: home, ProfileDir: filepath.Join(home, "profiles", "web"), Whale: whaleTasks || os.Getenv("WHALEISLE_PROACTIVE_EXTERNAL_DELIVERY") == "1", Dshbot: dshbotFlow, RemoteSet: true, Remote: false}, Workspace: workspace, ReadyTimeout: 3 * time.Minute})
	var endpoint harness.Endpoint
	var store *session.Store
	var mux *dsh.Mux
	var client *dsh.Client
	var eventsCancel context.CancelFunc
	var eventsDone chan error
	detach := func() {
		if eventsCancel != nil {
			eventsCancel()
			eventsCancel = nil
		}
		if store != nil {
			store.Detach()
			store = nil
		}
		if mux != nil {
			mux.Close()
			mux = nil
		}
		if eventsDone != nil {
			select {
			case <-eventsDone:
			case <-time.After(5 * time.Second):
			}
			eventsDone = nil
		}
	}
	stop := func() {
		detach()
		sup.Stop()
		r.OwnedHostStopped = sup.Status().State == harness.Stopped
		if endpoint.Origin != "" {
			r.OwnedListenerClosed = closedListener(endpoint.Origin)
		}
		_ = os.WriteFile(filepath.Join(root, "host.log"), []byte(strings.Join(sup.Logs(), "\n")), 0600)
	}
	defer stop()
	pass := func(name, detail string) {
		r.Checks = append(r.Checks, check{Name: name, Pass: true, Detail: detail})
		fmt.Println("checkpoint", name)
	}
	start := func() error {
		sup.Start()
		ctx, cancel := context.WithTimeout(context.Background(), 195*time.Second)
		defer cancel()
		if err := await(ctx, func() (bool, error) {
			st := sup.Status()
			if st.State == harness.Failed {
				return false, st.Err
			}
			if st.State == harness.Ready {
				endpoint = st.Endpoint
				return true, nil
			}
			return false, nil
		}); err != nil {
			return err
		}
		r.OwnedHostStopped = false
		r.OwnedListenerClosed = false
		store, mux, client = attach(endpoint)
		ectx, cancel := context.WithCancel(context.Background())
		eventsCancel = cancel
		eventsDone = make(chan error, 1)
		ready := make(chan struct{}, 1)
		owner := store.BeginEvents(ectx, client)
		boundStore, boundClient, boundMux, done := store, client, mux, eventsDone
		go func() {
			done <- boundClient.FollowEvents(ectx, boundMux, dsh.EventsFuncs{Ready: func(dsh.EventsReady) {
				boundStore.OnEventsReady(owner)
				select {
				case ready <- struct{}{}:
				default:
				}
			}, Emit: func(e dsh.EventsEmit) { boundStore.OnEmit(owner, e) }, Waterfall: func(wf dsh.EventsWaterfall) (*dsh.EventResultOutcome, error) {
				if acpFixture != nil {
					return acpFixture.approve(wf)
				}
				if wf.Event != "approval/request" {
					return nil, nil
				}
				raw, _ := json.Marshal(wf)
				model.mu.Lock()
				name := model.readTool
				model.mu.Unlock()
				if name == "" || !strings.Contains(string(raw), name) {
					return nil, errors.New("unexpected approval outside the owned read tool")
				}
				return &dsh.EventResultOutcome{Kind: "result", Value: json.RawMessage(`"allowed-once"`)}, nil
			}})
		}()
		select {
		case <-ready:
			return nil
		case err := <-eventsDone:
			return fmt.Errorf("events before ready: %w", err)
		case <-ctx.Done():
			return ctx.Err()
		}
	}
	if err = start(); err != nil {
		return fmt.Errorf("start owned Host: %w", err)
	}
	pass("real-host-ready", "Host "+r.HostVersion+"; Go dsh RPC/events client; isolated workspace/profile; no desktop window")
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
	defer cancel()
	if _, err := sup.PlatformSession(ctx); err == nil || err.Error() != "Platform account unavailable" {
		return fmt.Errorf("isolated platform publisher must load and report no account: %v", err)
	}
	pass("platform-plugin-live-route", "staged platform plugin loaded its relative imports and returned an authenticated empty account snapshot in the isolated profile")
	if acpFlow {
		return exerciseACP(ctx, client, store, workspace, home, acpFixture, pass, openProactive, func() (*dsh.Client, *session.Store, error) { stop(); err := start(); return client, store, err })
	}
	if proactiveFlow {
		return exerciseProactive(ctx, client, model, workspace, userData, home, pass, func() (*dsh.Client, error) { stop(); err := start(); return client, err }, openProactive)
	}
	if whaleTasks {
		return exerciseWhaleTasks(ctx, client, store, model, workspace, root, pass, func() (*dsh.Client, *session.Store, error) { stop(); err := start(); return client, store, err })
	}
	if dshbotFlow {
		return exerciseDshbot(ctx, client, store, workspace, pass, func() (*dsh.Client, *session.Store, error) {
			stop()
			err := start()
			return client, store, err
		})
	}
	if phoneFlow {
		var created api.WorkspaceCreateValue
		if err := client.Call(ctx, "workspace/create", map[string]any{"request": map[string]any{"path": workspace}}, &created); err != nil {
			return fmt.Errorf("register visible phone workspace: %w", err)
		}
		store.FollowWorkspace(ctx)
		if err := await(ctx, func() (bool, error) { return store.Snapshot().WorkspaceReady, nil }); err != nil {
			return err
		}
	}
	id, err := store.CreateComposerSession(ctx, workspace)
	if err != nil {
		return fmt.Errorf("create session: %w", err)
	}
	store.Select(id)
	if err = await(ctx, func() (bool, error) {
		v := store.Snapshot()
		if v.HistoryError != nil {
			return false, errors.New(v.HistoryError.SentenceText())
		}
		return v.HistoryReady && !v.HistoryLoading, nil
	}); err != nil {
		return err
	}
	pass("session-create-follow", "created via production Go composer adapter; real Host session/follow opened")
	if queueIdle {
		return exerciseQueueIdle(ctx, client, store, model, id, workspace, pass)
	}
	save := func(name string) error {
		v := store.Snapshot()
		raw, err := json.MarshalIndent(map[string]any{"sessionId": v.SelectedID, "historyReady": v.HistoryReady, "historyError": v.HistoryError.SentenceText(), "promptBusy": v.PromptBusy, "lastAction": v.LastAction, "messages": v.Messages}, "", "  ")
		if err != nil {
			return err
		}
		return os.WriteFile(filepath.Join(root, name+".json"), append(raw, '\n'), 0600)
	}
	waitText := func(marker string, settled bool) error {
		waitCtx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
		defer cancel()
		return await(waitCtx, func() (bool, error) {
			v := store.Snapshot()
			if v.HistoryError != nil {
				return false, errors.New(v.HistoryError.SentenceText())
			}
			for _, m := range v.Messages {
				if m.Role == "assistant" && strings.Contains(m.Body, marker) && (!settled || !m.Streaming && m.Seq > 0 && !v.PromptBusy && v.StopTurnSeq == 0) {
					return true, nil
				}
			}
			return false, nil
		})
	}
	send := func(prompt, marker string) error {
		if err := store.SubmitComposer(ctx, id, prompt, nil, 0); err != nil {
			return err
		}
		if err := waitText(marker, true); err != nil {
			_ = save("failure-snapshot")
			return fmt.Errorf("await %s: %w; lastAction=%s", marker, err, store.Snapshot().LastAction)
		}
		return nil
	}
	if err = send(helloPrompt, "QA_REPLY_OK"); err != nil {
		return err
	}
	if err = save("reply"); err != nil {
		return err
	}
	pass("prompt-full-reply", "production SubmitComposer → Host → loopback HTTP/SSE → durable Go Store assistant row")
	if phoneFlow {
		return exercisePhone(ctx, stage, root, userData, endpoint, store, id, pass)
	}
	if err = send(toolPrompt, "QA_TOOL_OK"); err != nil {
		return err
	}
	model.mu.Lock()
	toolReturned := model.toolResult
	toolName := model.readTool
	model.mu.Unlock()
	if !toolReturned || !hasOwnedFileResult(store.Snapshot()) {
		return errors.New("actual tool result was not both returned to the model and projected by the Go Store")
	}
	if err = save("tool"); err != nil {
		return err
	}
	pass("actual-file-tool-roundtrip", "real Host "+toolName+" tool read owned qa-note.txt; successful result reached provider and persisted Store projection")
	if err = send(writeGuardPrompt, "QA_WRITE_GUARD"); err != nil {
		return err
	}
	model.mu.Lock()
	writeRefused, writeTool := model.writeRefused, model.writeTool
	model.mu.Unlock()
	if !writeRefused {
		return fmt.Errorf("Host accepted a %s call for a file the session never read", writeTool)
	}
	if err = save("write-guard"); err != nil {
		return err
	}
	pass("write-without-read-refused", "real Host "+writeTool+" tool refused an overwrite of owned qa-unread.txt; fs-observation-policy is mounted")
	if err = send(staleEditPrompt, "QA_EDIT_STALE"); err != nil {
		return err
	}
	model.mu.Lock()
	staleRefused, editTool := model.staleRefused, model.editTool
	model.mu.Unlock()
	if !staleRefused {
		return fmt.Errorf("Host accepted a %s call computed from a stale read", editTool)
	}
	if err = save("stale-edit"); err != nil {
		return err
	}
	pass("stale-edit-refused", "real Host "+editTool+" tool refused an edit after owned qa-stale.txt changed underneath the session")
	if err = store.SubmitComposer(ctx, id, waitPrompt, nil, 0); err != nil {
		return err
	}
	if err = waitText("QA_WAIT_STREAM", false); err != nil {
		return err
	}
	stopStarted := time.Now()
	store.CancelRunning(ctx)
	stopCtx, stopCancel := context.WithTimeout(context.Background(), 15*time.Second)
	err = await(stopCtx, func() (bool, error) {
		model.mu.Lock()
		canceled := model.canceled
		model.mu.Unlock()
		v := store.Snapshot()
		streaming := false
		for _, m := range v.Messages {
			streaming = streaming || m.Streaming
		}
		return canceled > 0 && !v.PromptBusy && !streaming, nil
	})
	stopCancel()
	if err != nil {
		return fmt.Errorf("stream stop: %w; lastAction=%s", err, store.Snapshot().LastAction)
	}
	if err = save("stopped"); err != nil {
		return err
	}
	pass("stream-stop", fmt.Sprintf("Go Store CancelRunning; provider HTTP context canceled; Store idle in %dms", time.Since(stopStarted).Milliseconds()))
	if err = send(nextPrompt, "QA_NEXT_OK"); err != nil {
		return err
	}
	if err = save("next-turn"); err != nil {
		return err
	}
	pass("next-turn-after-stop", "fresh prompt completed on same real session after cancellation")
	stop()
	if !r.OwnedHostStopped || !r.OwnedListenerClosed {
		return errors.New("first owned Host/listener did not stop")
	}
	pass("first-host-released", "owned Host stopped and TCP listener closed before restart")
	if err = start(); err != nil {
		return fmt.Errorf("restart owned Host: %w", err)
	}
	store.Select(id)
	for _, marker := range []string{"QA_REPLY_OK", "QA_TOOL_OK", "QA_WRITE_GUARD_OK", "QA_EDIT_STALE_OK", "QA_WAIT_STREAM", "QA_NEXT_OK"} {
		if err = waitText(marker, true); err != nil {
			return fmt.Errorf("restored history missing %s: %w", marker, err)
		}
	}
	restored := store.Snapshot()
	for _, prompt := range []string{helloPrompt, toolPrompt, writeGuardPrompt, staleEditPrompt, waitPrompt, nextPrompt} {
		found := false
		for _, m := range restored.Messages {
			if m.Role == "user" && m.Body == prompt && m.Seq > 0 {
				found = true
				break
			}
		}
		if !found {
			return fmt.Errorf("restored durable user prompt missing: %s", prompt)
		}
	}
	if !hasOwnedFileResult(restored) {
		return errors.New("restored successful tool/result missing the exact owned file content")
	}
	if err = save("restored-history"); err != nil {
		return err
	}
	pass("history-after-real-host-restart", "fresh Go Store restored the same session, six durable user prompts, all replies/stopped text and the successful file-tool result")
	return nil
}
