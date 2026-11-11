package main

import (
	"context"
	_ "embed"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"sync/atomic"
	"time"
	"whaleisle/internal/desktop/acp"
	"whaleisle/internal/desktop/pluginrpc"
	"whaleisle/internal/desktop/proactive"
	"whaleisle/internal/dsh"
	"whaleisle/internal/dsh/api"
	"whaleisle/internal/store/session"
)

//go:embed acp_fixture.mjs
var acpFixtureScript []byte

type acpFixtureState struct {
	root      string
	allow     atomic.Bool
	approvals atomic.Int32
}

func prepareACPFixture(root, home, vendor, node string) (*acpFixtureState, error) {
	script := filepath.Join(root, "acp-fixture.mjs")
	if err := os.WriteFile(script, acpFixtureScript, 0600); err != nil {
		return nil, err
	}
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	command := exec.CommandContext(ctx, node, script, "configure", home, vendor, node)
	if output, err := command.CombinedOutput(); err != nil {
		return nil, fmt.Errorf("configure isolated ACP: %w: %s", err, output)
	}
	return &acpFixtureState{root: root}, nil
}

func (f *acpFixtureState) approve(wf dsh.EventsWaterfall) (*dsh.EventResultOutcome, error) {
	if wf.Event != "approval/request" {
		return nil, nil
	}
	raw, _ := json.Marshal(wf)
	if !strings.Contains(string(raw), "QA ACP write owned file") {
		return nil, errors.New("unexpected non-owned ACP approval")
	}
	f.approvals.Add(1)
	decision := `"rejected"`
	if f.allow.Load() {
		decision = `"allowed-once"`
	}
	return &dsh.EventResultOutcome{Kind: "result", Value: json.RawMessage(decision)}, nil
}

func hasSessionEvent(ctx context.Context, client *dsh.Client, sessionID, eventType, substring string) (bool, error) {
	mux := &dsh.Mux{Endpoint: client.Endpoint, HTTP: client.HTTP}
	defer mux.Close()
	if err := mux.Dial(ctx); err != nil {
		return false, err
	}
	address, _ := json.Marshal(map[string]string{"kind": "session", "sessionId": sessionID})
	maxMessages := float64(500)
	stream, err := mux.Open(ctx, "session/follow", map[string]any{"args": map[string]any{"request": api.SessionFollowRequest{Address: address, MaxMessages: &maxMessages}}})
	if err != nil {
		return false, err
	}
	defer stream.Cancel()
	raw, err := stream.Recv(ctx)
	if err != nil {
		return false, err
	}
	var snapshot struct {
		Records []struct {
			Type  string          `json:"type"`
			Event json.RawMessage `json:"event"`
		} `json:"records"`
	}
	if err := json.Unmarshal(raw, &snapshot); err != nil {
		return false, err
	}
	for _, r := range snapshot.Records {
		if r.Type != "event" {
			continue
		}
		var header struct {
			Type string `json:"type"`
		}
		if json.Unmarshal(r.Event, &header) == nil && header.Type == eventType && strings.Contains(string(r.Event), substring) {
			return true, nil
		}
	}
	return false, nil
}

func exerciseACP(ctx context.Context, client *dsh.Client, store *session.Store, workspace, home string, f *acpFixtureState, pass func(string, string), openProactive func() (*proactive.Service, error), restart func() (*dsh.Client, *session.Store, error)) error {
	rpcFor := func(c *dsh.Client) acp.Client {
		return acp.Client{RPC: &pluginrpc.Client{BaseURL: c.Endpoint.Origin, Cookie: c.Endpoint.Cookie, HTTPClient: c.HTTP}}
	}
	rpc := rpcFor(client)
	catalog, err := rpc.Catalog(ctx)
	if err != nil {
		return err
	}
	agent := func(c acp.Catalog) acp.Agent {
		for _, a := range c.Agents {
			if a.ID == "qa-local" {
				return a
			}
		}
		return acp.Agent{}
	}
	a := agent(catalog)
	if !a.Enabled || a.Command == "" {
		return errors.New("owned ACP agent absent")
	}
	for _, v := range catalog.Agents {
		if v.ID != "qa-local" && v.Enabled {
			return errors.New("non-owned ACP agent enabled")
		}
	}
	if _, _, err = rpc.Action(ctx, "agent/test", map[string]any{"id": a.ID, "cwd": workspace}); err != nil {
		return fmt.Errorf("ACP handshake: %w", err)
	}
	pass("acp-t3-handshake", "staged T3 bundle initialized a local JSONL agent; all built-ins disabled before Host start")
	id, err := store.CreateComposerSession(ctx, workspace)
	if err != nil {
		return err
	}
	store.Select(id)
	if err = await(ctx, func() (bool, error) { v := store.Snapshot(); return v.HistoryReady && !v.HistoryLoading, nil }); err != nil {
		return err
	}
	store.RefreshModelCatalog(ctx)
	if err = await(ctx, func() (bool, error) {
		for _, c := range store.Snapshot().ModelCatalogChoices {
			if c.Provider == "acp:qa-local" {
				return true, nil
			}
		}
		return false, nil
	}); err != nil {
		return fmt.Errorf("ACP model catalog: %w", err)
	}
	store.SelectModelSelectionForSessionID(ctx, id, api.ModelCatalogDefault{Provider: "acp:qa-local", Model: "mock-pro"})
	if err = await(ctx, func() (bool, error) {
		v := store.Snapshot()
		return !v.ModelSelectionBusy && v.ModelChoices[v.SelectedModel].Provider == "acp:qa-local" && v.ModelChoices[v.SelectedModel].Model == "mock-pro", nil
	}); err != nil {
		return fmt.Errorf("select ACP model: %w; %s", err, store.Snapshot().LastAction)
	}
	pass("acp-native-model-route", "production Store selected ACP provider/model from real Host catalog and committed projection")
	effort := "high"
	store.SelectModelEffortSelectionForSessionID(ctx, id, api.ModelCatalogDefault{Provider: "acp:qa-local", Model: "mock-pro", ReasoningEffort: &effort})
	if err = await(ctx, func() (bool, error) {
		v := store.Snapshot()
		return !v.ModelSelectionBusy && v.ModelSelection != nil && v.ModelSelection.ReasoningEffort != nil && *v.ModelSelection.ReasoningEffort == effort, nil
	}); err != nil {
		return fmt.Errorf("select ACP effort: %w", err)
	}
	waitText := func(marker string, settled bool) error {
		waitCtx, cancel := context.WithTimeout(ctx, 30*time.Second)
		defer cancel()
		return await(waitCtx, func() (bool, error) {
			v := store.Snapshot()
			if v.HistoryError != "" {
				return false, errors.New(v.HistoryError)
			}
			for _, m := range v.Messages {
				if m.Role == "assistant" && strings.Contains(m.Body, marker) && (!settled || !m.Streaming && m.Seq > 0 && !v.PromptBusy) {
					return true, nil
				}
			}
			return false, nil
		})
	}
	send := func(text string) error {
		if err := store.SubmitComposer(ctx, id, text, nil, 0); err != nil {
			return err
		}
		if err := waitText("QA_ACP_REPLY "+text, true); err != nil {
			raw, _ := json.MarshalIndent(store.Snapshot(), "", "  ")
			_ = os.WriteFile(filepath.Join(f.root, "acp-failure-store.json"), raw, 0600)
			return fmt.Errorf("ACP reply %s: %w; %s", text, err, store.Snapshot().LastAction)
		}
		return nil
	}
	if err = send("QA_ACP_HELLO"); err != nil {
		return err
	}
	if err = send("QA_ACP_SECOND"); err != nil {
		return err
	}
	pass("acp-stream-continuation", "two production composer turns streamed and persisted via the external ACP process, not the HTTP model fixture")
	if err = os.WriteFile(filepath.Join(workspace, "qa-mcp-deliverable.html"), []byte("<!doctype html><title>QA MCP Deliverable</title>\n"), 0600); err != nil {
		return err
	}
	if err = send("QA_ACP_MCP"); err != nil {
		return err
	}
	if err = waitText("QA_ACP_MCP_OK", true); err != nil {
		return fmt.Errorf("ACP MCP bridge check failed: %w", err)
	}
	presented, err := hasSessionEvent(ctx, client, id, "deliverables/presented", "qa-mcp-deliverable.html")
	if err != nil || !presented {
		return fmt.Errorf("ACP MCP whaleisle_present did not append deliverables/presented: %v %v", presented, err)
	}
	pass("acp-mcp-stdio-bridge", "external ACP subprocess spawned staged t3-mcp-bridge.cjs over stdio, listed WhaleIsle MCP tools, called whaleisle_skill_list, and appended deliverables/presented via whaleisle_present")
	if err = send("QA_ACP_TOOL_DENY"); err != nil {
		return err
	}
	if f.approvals.Load() != 1 {
		return fmt.Errorf("denied ACP approval count %d", f.approvals.Load())
	}
	if _, err = os.Stat(filepath.Join(f.root, "acp-effect.txt")); !os.IsNotExist(err) {
		return errors.New("denied external ACP write produced an effect")
	}
	f.allow.Store(true)
	if err = send("QA_ACP_TOOL_ALLOW"); err != nil {
		return err
	}
	effect, err := os.ReadFile(filepath.Join(f.root, "acp-effect.txt"))
	if err != nil || string(effect) != "effect\n" || f.approvals.Load() != 2 {
		return fmt.Errorf("allowed ACP effect/approval mismatch: %q %d %v", effect, f.approvals.Load(), err)
	}
	pass("acp-permission-bridge", "actual external permission request reached Host events; denial prevented file effect, allow-once produced exactly one owned write")
	catalog, err = rpc.Catalog(ctx)
	if err != nil {
		return err
	}
	a = agent(catalog)
	if len(catalog.ActiveSessions) != 1 {
		return fmt.Errorf("expected live ACP session, got %d", len(catalog.ActiveSessions))
	}
	original := catalog.ActiveSessions[0].AcpSessionID
	catalog, _, err = rpc.Action(ctx, "agent/save", map[string]any{"agent": a.Patch(map[string]any{"name": "QA Renamed ACP", "favoriteModels": []string{"mock-pro"}, "hiddenModels": []string{"mock-fast"}})})
	if err != nil {
		return err
	}
	if len(catalog.ActiveSessions) != 1 || catalog.ActiveSessions[0].AcpSessionID != original {
		return errors.New("cosmetic save disconnected ACP")
	}
	pass("acp-cosmetic-save-preserves-session", "display name/favorites/hidden-model save retained the active external session and existing config")
	if err = store.SubmitComposer(ctx, id, "QA_ACP_WAIT", nil, 0); err != nil {
		return err
	}
	if err = waitText("QA_ACP_WAITING", false); err != nil {
		return err
	}
	store.CancelRunning(ctx)
	if err = await(ctx, func() (bool, error) { v := store.Snapshot(); return !v.PromptBusy && v.StopTurnSeq == 0, nil }); err != nil {
		return err
	}
	if err = send("QA_ACP_AFTER_STOP"); err != nil {
		return err
	}
	pass("acp-stop-next-turn", "native Store stop cancelled an actual pending ACP prompt; the same conversation accepted its next turn")
	client, store, err = restart()
	if err != nil {
		return err
	}
	rpc = rpcFor(client)
	store.Select(id)
	if err = waitText("QA_ACP_REPLY QA_ACP_SECOND", true); err != nil {
		return fmt.Errorf("restore ACP history: %w", err)
	}
	if err = send("QA_ACP_AFTER_RESTART"); err != nil {
		return err
	}
	protocol, err := os.ReadFile(filepath.Join(f.root, "acp-protocol.jsonl"))
	if err != nil {
		return err
	}
	if !strings.Contains(string(protocol), `"event":"session/resume"`) {
		return errors.New("external ACP session was not loaded after Host restart")
	}
	if !strings.Contains(string(protocol), `"configId":"model","value":"mock-pro"`) && !strings.Contains(string(protocol), `"value":"mock-pro"`) {
		return errors.New("selected model did not reach ACP process")
	}
	if !strings.Contains(string(protocol), `"value":"high"`) {
		return errors.New("selected reasoning effort did not reach ACP process")
	}
	pass("acp-restart-resume", "Host restart restored durable DSH history and sent ACP v2 session/resume for the persisted external ACP session")
	if err = exerciseACPProactivePolicy(ctx, client, store, rpc, workspace, home, f, openProactive, pass); err != nil {
		return err
	}
	catalog, _, err = rpc.Action(ctx, "sessions/stop", map[string]any{"agentId": "qa-local", "sessionId": id})
	if err != nil {
		return err
	}
	for _, s := range catalog.ActiveSessions {
		if s.SessionID == id {
			return errors.New("ACP single-session disconnect retained target session")
		}
	}
	catalog, _, err = rpc.Action(ctx, "sessions/stop", map[string]any{"agentId": "qa-local"})
	if err != nil {
		return err
	}
	if len(catalog.ActiveSessions) != 0 {
		return errors.New("ACP disconnect retained active connection")
	}
	catalog, _, err = rpc.Action(ctx, "agent/toggle", map[string]any{"id": "qa-local", "enabled": false})
	if err != nil {
		return err
	}
	if agent(catalog).Enabled {
		return errors.New("ACP disable not committed")
	}
	pass("acp-disconnect-disable", "external connection and mapping released explicitly; agent disabled without deleting DSH conversation history")
	raw, _ := json.MarshalIndent(store.Snapshot(), "", "  ")
	return os.WriteFile(filepath.Join(f.root, "acp-final-store.json"), raw, 0600)
}

func exerciseACPProactivePolicy(ctx context.Context, client *dsh.Client, store *session.Store, rpc acp.Client, workspace, home string, f *acpFixtureState, open func() (*proactive.Service, error), pass func(string, string)) error {
	catalog, err := rpc.Catalog(ctx)
	if err != nil {
		return err
	}
	var target acp.Agent
	for _, a := range catalog.Agents {
		if a.ID == "qa-local" {
			target = a
		}
	}
	catalog, _, err = rpc.Action(ctx, "agent/save", map[string]any{"agent": target.Patch(map[string]any{"permissionPolicy": "auto-allow"})})
	if err != nil {
		return err
	}
	savedPolicy := ""
	for _, a := range catalog.Agents {
		if a.ID == "qa-local" {
			savedPolicy = a.PermissionPolicy
		}
	}
	if savedPolicy != "auto-allow" {
		return fmt.Errorf("expected auto-allow policy, got %q", savedPolicy)
	}
	s, err := open()
	if err != nil {
		return err
	}
	runner := proactive.NewRunner(ctx, s, proactive.Host{Client: client}, home, nil)
	defer runner.Close()
	spec := proactive.RunSpec{HostHome: home, Cwd: workspace}
	startTask := func(title string, level int) (proactive.Task, error) {
		t, err := s.Create(proactive.CreateInput{Title: title, Autonomy: level})
		if err != nil {
			return t, err
		}
		t, err = s.PrepareExecution(t.ID, spec, false)
		if err != nil {
			return t, err
		}
		var created api.SessionCreateValue
		if err := client.Call(ctx, "session/create", map[string]any{"request": api.SessionCreateRequest{SessionId: &t.Execution.SessionID, Cwd: &workspace}}, &created); err != nil {
			return t, err
		}
		var renamed api.SessionRenameValue
		if err := client.Call(ctx, "session/rename", map[string]any{"request": map[string]string{"sessionId": t.Execution.SessionID, "title": t.Title}}, &renamed); err != nil {
			return t, err
		}
		var selected api.SessionSelectModelValue
		if err := client.Call(ctx, "session/selectModel", map[string]any{"request": api.SessionSelectModelRequest{SessionId: t.Execution.SessionID, Provider: "acp:qa-local", Model: "mock-pro"}}, &selected); err != nil {
			return t, err
		}
		runner.Observe(t.ID)
		return t, nil
	}
	awaitApproval := func(taskID string) (proactive.Approval, error) {
		var found proactive.Approval
		err := await(ctx, func() (bool, error) {
			t, err := s.Get(taskID)
			if err != nil {
				return false, err
			}
			if t.Execution != nil && t.Execution.Phase == "unknown" {
				return false, fmt.Errorf("ACP task execution: %s", t.Execution.Error)
			}
			for _, a := range s.Snapshot().Outbox {
				if a.TaskID == taskID && a.Status == "pending" {
					found = a
					return true, nil
				}
			}
			return false, nil
		})
		return found, err
	}
	awaitVerifying := func(taskID string) error {
		return await(ctx, func() (bool, error) {
			t, err := s.Get(taskID)
			if err != nil {
				return false, err
			}
			if t.Execution != nil && t.Execution.Phase == "unknown" {
				return false, fmt.Errorf("ACP task execution: %s", t.Execution.Error)
			}
			return t.State == "verifying", nil
		})
	}
	effectFile := filepath.Join(f.root, "acp-effect.txt")
	effectCount := func() (int, error) {
		raw, err := os.ReadFile(effectFile)
		if err != nil {
			return 0, err
		}
		return strings.Count(string(raw), "effect\n"), nil
	}
	chatApprovalsBefore := f.approvals.Load()
	baselineEffects, err := effectCount()
	if err != nil || baselineEffects != 1 {
		return fmt.Errorf("unexpected baseline ACP effects: %d %v", baselineEffects, err)
	}
	l1, err := startTask("QA_ACP_TOOL_L1_AUTO_SUPPRESSED", 1)
	if err != nil {
		return err
	}
	if err := awaitVerifying(l1.ID); err != nil {
		return err
	}
	if n, err := effectCount(); err != nil || n != 1 {
		return fmt.Errorf("L1 task allowed external ACP write despite read-only gate: %d %v", n, err)
	}
	l2, err := startTask("QA_ACP_TOOL_L2_AUTO_SUPPRESSED", 2)
	if err != nil {
		return err
	}
	firstApproval, err := awaitApproval(l2.ID)
	if err != nil {
		return err
	}
	if firstApproval.ToolCall == nil || firstApproval.ToolCall.Source != "desktop:acp" {
		return fmt.Errorf("L2 ACP approval missing desktop:acp provenance: %+v", firstApproval.ToolCall)
	}
	if n, err := effectCount(); err != nil || n != 1 {
		return fmt.Errorf("L2 task auto-allowed external ACP write before Outbox approval: %d %v", n, err)
	}
	if _, err := s.Decide(firstApproval.ID, "rejected", ""); err != nil {
		return err
	}
	if err := awaitVerifying(l2.ID); err != nil {
		return err
	}
	if n, err := effectCount(); err != nil || n != 1 {
		return fmt.Errorf("rejected L2 ACP approval produced an effect: %d %v", n, err)
	}
	if _, err := runner.Start(l2.ID, spec, true); err != nil {
		return err
	}
	secondApproval, err := awaitApproval(l2.ID)
	if err != nil {
		return err
	}
	if secondApproval.ID == firstApproval.ID || secondApproval.ToolCall == nil || secondApproval.ToolCall.Source != "desktop:acp" {
		return fmt.Errorf("continued L2 ACP task did not create fresh desktop:acp approval: %+v", secondApproval)
	}
	if _, err := s.Decide(secondApproval.ID, "approved", ""); err != nil {
		return err
	}
	if err := awaitVerifying(l2.ID); err != nil {
		return err
	}
	if n, err := effectCount(); err != nil || n != 2 {
		return fmt.Errorf("approved L2 ACP task did not produce exactly one effect: %d %v", n, err)
	}
	receiptFound := false
	for _, a := range s.Snapshot().Outbox {
		if a.ID == secondApproval.ID && a.Effect == "succeeded" {
			receiptFound = true
		}
	}
	if !receiptFound {
		return errors.New("approved L2 ACP task missing succeeded effect receipt")
	}
	store.Select(l2.Execution.SessionID)
	if err := await(ctx, func() (bool, error) { v := store.Snapshot(); return v.HistoryReady && !v.HistoryLoading, nil }); err != nil {
		return err
	}
	if err := store.SubmitComposer(ctx, l2.Execution.SessionID, "QA_ACP_TOOL_ORDINARY_AUTO", nil, 0); err != nil {
		return err
	}
	if err := await(ctx, func() (bool, error) {
		v := store.Snapshot()
		for _, m := range v.Messages {
			if m.Role == "assistant" && strings.Contains(m.Body, "QA_ACP_REPLY QA_ACP_TOOL_ORDINARY_AUTO") && !m.Streaming && m.Seq > 0 && !v.PromptBusy {
				return true, nil
			}
		}
		return false, nil
	}); err != nil {
		return fmt.Errorf("ordinary ACP follow-up turn failed: %w", err)
	}
	if n, err := effectCount(); err != nil || n != 3 {
		return fmt.Errorf("ordinary follow-up turn did not restore ACP auto-allow: %d %v", n, err)
	}
	if f.approvals.Load() != chatApprovalsBefore {
		return fmt.Errorf("auto-allow turns unexpectedly emitted %d chat approval events", f.approvals.Load()-chatApprovalsBefore)
	}
	pass("acp-proactive-auto-allow-constrained", "ACP permissionPolicy=auto-allow was suppressed on native task turns (L1 auto-denied, L2 gated in Outbox with source=desktop:acp, reject had zero effect, approve-once wrote once) and automatically restored on an ordinary user follow-up turn in the same Session")
	return nil
}
