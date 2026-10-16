package main

import (
	"context"
	"fmt"
	"strings"
	"time"

	"whaleisle/internal/dsh"
	"whaleisle/internal/harness"
	assiststore "whaleisle/internal/store/assistant"
)

// This panel is an entry point, not another transcript or prompt transport.
// The existing session/composer owns streaming, tool approvals, stop and retry.
func (s *mainState) assistantConversationCurrent(ep harness.Endpoint, client *dsh.Client, generation uint64, mode assiststore.Mode) bool {
	s.dshMu.Lock()
	same := client != nil && s.dshClient == client && client.Endpoint.Origin == ep.Origin && client.Endpoint.Cookie == ep.Cookie
	s.dshMu.Unlock()
	s.mu.Lock()
	defer s.mu.Unlock()
	if !same || !s.assistantOpen || s.assistantConversationGeneration != generation || s.bootStatus.State != harness.Ready || s.bootStatus.Endpoint != ep || s.cfgStore == nil {
		return false
	}
	cfg := s.cfgStore.Get()
	if mode == assiststore.ModeDshbot {
		return cfg.DshbotEnabled
	}
	return cfg.WhaleAssistantEnabled
}

func (s *mainState) openAssistantConversation(ep harness.Endpoint, mode assiststore.Mode, botID, text string) {
	s.mu.Lock()
	asst, sessions, win := s.assistant, s.sessions, s.win
	if asst == nil || sessions == nil || win == nil || !s.assistantOpen || s.assistantPluginBusy || asst.Snapshot().Busy {
		s.mu.Unlock()
		return
	}
	s.assistantConversationGeneration++
	generation := s.assistantConversationGeneration
	s.mu.Unlock()
	mode = assiststore.NormalizeMode(mode)
	if mode == assiststore.ModeDshbot {
		if strings.TrimSpace(botID) == "" {
			asst.SetLastAction("Select a Bot or room first", "Select a Bot or room first")
			return
		}
		asst.SelectDshbot(botID, ep.Origin)
	}
	owner, err := sessions.CaptureComposerTarget(sessions.Snapshot().SelectedID)
	if err != nil {
		asst.SetLastAction(err.Error(), err.Error())
		return
	}
	s.dshMu.Lock()
	client := s.dshClient
	s.dshMu.Unlock()
	if !s.assistantConversationCurrent(ep, client, generation, mode) {
		return
	}
	asst.SetBusy(true)
	asst.SetLastAction("正在打开主会话 / Opening canonical conversation…", "")
	go func() {
		ctx, cancel := context.WithTimeout(context.Background(), 45*time.Second)
		defer cancel()
		var sid string
		var openErr error
		if mode == assiststore.ModeDshbot {
			_, sid, openErr = s.resolveAssistantDshbot(ctx, ep, sessions, botID)
		} else {
			result, err := s.pluginRPCClient(ep).WhaleEnsureTyped(ctx)
			sid, openErr = result.SessionID, err
		}
		win.Update(func() {
			asst.SetBusy(false)
			if !s.assistantConversationCurrent(ep, client, generation, mode) || !owner.NavigationCurrent() {
				return
			}
			if openErr == nil && strings.TrimSpace(sid) == "" {
				openErr = fmt.Errorf("Host returned no canonical session")
			}
			if openErr != nil {
				asst.SetLastAction(openErr.Error(), openErr.Error())
				return
			}
			// Preserve an existing session draft rather than submit it accidentally.
			s.mu.Lock()
			draft := s.composerDraftLocked(sid)
			conflict := text != "" && (draft.Text != "" || len(draft.Attachments) != 0 || draft.Sending || draft.Preparing > 0 || draft.EditSeq != 0)
			s.mu.Unlock()
			if conflict {
				asst.SetLastAction("目标会话已有草稿；请先处理主聊天草稿，再返回发送。 / Resolve the existing chat draft before sending.", "Draft retained; nothing sent")
				return
			}
			s.revokePhoneGit()
			sessions.Select(sid)
			if mode == assiststore.ModeDshbot {
				s.markOpenedBotRead(ep, client, botID, sid)
			}
			// Select already starts the canonical follow and control subscriptions.
			asst.SetSessionID(sid)
			s.mu.Lock()
			s.assistantOpen = false
			s.assistantConversationGeneration++
			if text != "" {
				draft.Text = text
				if strings.TrimSpace(s.assistantDraft) == text {
					s.assistantDraft = ""
				}
			}
			s.mu.Unlock()
			if text != "" {
				draft.Client.Sync()
			}
			asst.SetLastAction("已打开主会话 / Canonical conversation opened", "")
			if text != "" {
				// Ownership moves to the visible canonical draft. Failed/uncertain sends
				// stay in the standard composer/submission UI and are never auto-retried.
				asst.AcceptPrompt(text)
				snap := sessions.Snapshot()
				s.submitComposer(win, sessions, draft, sid, snap.ScratchCwd, 0, false, snap.ComposerHostGeneration)
			}
			sessions.RefreshList(context.Background())
		})
	}()
}
