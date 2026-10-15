package main

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"whaleisle/internal/desktop/pluginrpc"
	"whaleisle/internal/dsh"
	"whaleisle/internal/dsh/api"
	"whaleisle/internal/harness"
	assiststore "whaleisle/internal/store/assistant"
	"whaleisle/internal/store/session"
)

// assistantDshbotCurrent fences delayed catalog/open results to the same live
// client and enabled plugin. A normal chat selection is never a Bot identity.
func (s *mainState) assistantDshbotCurrent(ep harness.Endpoint, client *dsh.Client) bool {
	s.dshMu.Lock()
	same := client != nil && s.dshClient == client &&
		client.Endpoint.Origin == ep.Origin && client.Endpoint.Cookie == ep.Cookie
	s.dshMu.Unlock()
	s.mu.Lock()
	ready := s.bootStatus.State == harness.Ready &&
		s.bootStatus.Endpoint.Origin == ep.Origin && s.bootStatus.Endpoint.Cookie == ep.Cookie
	enabled := s.cfgStore != nil && s.cfgStore.Get().DshbotEnabled
	s.mu.Unlock()
	return same && ready && enabled
}

func (s *mainState) resolveAssistantDshbot(ctx context.Context, ep harness.Endpoint, sessions *session.Store, botID string) (*dsh.Client, string, error) {
	s.dshMu.Lock()
	client := s.dshClient
	s.dshMu.Unlock()
	if sessions == nil || !s.assistantDshbotCurrent(ep, client) {
		return nil, "", fmt.Errorf("Bot Host is not connected or the plugin is disabled")
	}
	botID = strings.TrimSpace(botID)
	if botID == "" {
		return nil, "", fmt.Errorf("select a Bot or room from the current Host roster first")
	}
	cli := s.pluginRPCClient(ep)
	catalog, err := cli.DshbotDescribeTyped(ctx, botID)
	if err != nil {
		return nil, "", err
	}
	var item pluginrpc.DshbotCatalogItem
	for _, candidate := range catalog.View.Value.Items {
		if candidate.ID == botID && !candidate.Hidden {
			item = candidate
			break
		}
	}
	if item.ID == "" {
		return nil, "", fmt.Errorf("Bot profile is no longer available; refresh the roster")
	}
	if !s.assistantDshbotCurrent(ep, client) {
		return nil, "", fmt.Errorf("Host connection changed; refresh the roster")
	}
	sid, err := cli.DshbotOpen(ctx, item, catalog.View.Revision, sessions.Snapshot().ScratchCwd)
	if err != nil {
		return nil, "", err
	}
	if !s.assistantDshbotCurrent(ep, client) {
		return nil, "", fmt.Errorf("Host connection changed while opening the Bot")
	}
	return client, sid, nil
}

func (s *mainState) sendAssistantDshbot(ep harness.Endpoint, sessions *session.Store, asst *assiststore.Store, text string) {
	target := asst.Snapshot()
	fail := func(err error) {
		msg := err.Error()
		asst.SetLastAction(msg, msg)
		asst.AppendTurn(assiststore.Turn{Mode: assiststore.ModeDshbot, Prompt: text, Err: msg, Source: "session"})
		s.mu.Lock()
		s.assistantLastAct = msg
		s.mu.Unlock()
	}
	if target.DshbotOrigin != ep.Origin || target.DshbotID == "" {
		fail(fmt.Errorf("select a Bot or room from the current Host roster first"))
		return
	}
	ctx, cancel := context.WithTimeout(context.Background(), 45*time.Second)
	defer cancel()
	client, sid, err := s.resolveAssistantDshbot(ctx, ep, sessions, target.DshbotID)
	if err != nil {
		fail(err)
		return
	}
	part, err := json.Marshal(map[string]string{"type": "text", "text": text})
	if err != nil {
		fail(err)
		return
	}
	// Use the captured authenticated client and canonical session, not the
	// mutable selected session. No prefix, optimistic success, or auto-retry.
	request := api.SessionPromptRequest{
		RequestId: fmt.Sprintf("mygo-assistant-%d", time.Now().UnixNano()),
		SessionId: sid,
		Mode:      "queue",
		Content:   []json.RawMessage{part},
	}
	if !s.assistantDshbotCurrent(ep, client) {
		fail(fmt.Errorf("Host connection changed before sending"))
		return
	}
	var result struct {
		Accepted bool `json:"accepted"`
	}
	if err := client.Call(ctx, "session/prompt", map[string]any{"request": request}, &result); err != nil {
		fail(err)
		return
	}
	if !result.Accepted {
		fail(fmt.Errorf("Host returned no acceptance receipt; verify the Bot conversation before retrying"))
		return
	}
	if !s.assistantDshbotCurrent(ep, client) {
		fail(fmt.Errorf("Host changed after submission; verify the Bot conversation before retrying"))
		return
	}
	msg := "已发送到所选 Bot / Sent to selected Bot"
	asst.SetSessionID(sid)
	asst.SetLastAction(msg, "")
	asst.AppendTurn(assiststore.Turn{
		Mode: assiststore.ModeDshbot, Prompt: text, Response: msg, SessionID: sid, Source: "session",
	})
	asst.AcceptPrompt(text)
	s.mu.Lock()
	s.assistantLastAct = msg
	s.mu.Unlock()
	sessions.RefreshList(context.Background())
}

// openAssistantDshbot restores cold/missing canonical sessions using the same
// bot/open operation as the original UI, instead of selecting a cached ID.
func (s *mainState) openAssistantDshbot(ep harness.Endpoint, botID string) {
	s.mu.Lock()
	asst, sessions, win := s.assistant, s.sessions, s.win
	s.mu.Unlock()
	if asst == nil || sessions == nil || asst.Snapshot().Busy {
		return
	}
	previous := sessions.Snapshot().SelectedID
	asst.SelectDshbot(botID, ep.Origin)
	s.runAssistantPluginRPC(win, ep, "bot/open", func(ctx context.Context, _ *pluginrpc.Client) (string, error) {
		ctx, cancel := context.WithTimeout(ctx, 45*time.Second)
		defer cancel()
		client, sid, err := s.resolveAssistantDshbot(ctx, ep, sessions, botID)
		if err != nil {
			return "", err
		}
		asst.SetSessionID(sid)
		s.mu.Lock()
		show := s.assistantOpen
		s.mu.Unlock()
		if show && sessions.Snapshot().SelectedID == previous && s.assistantDshbotCurrent(ep, client) {
			s.revokePhoneGit()
			sessions.Select(sid)
			sessions.FollowSelected(context.Background())
			s.mu.Lock()
			s.assistantOpen = false
			s.mu.Unlock()
		}
		sessions.RefreshList(context.Background())
		return sid, nil
	})
}
