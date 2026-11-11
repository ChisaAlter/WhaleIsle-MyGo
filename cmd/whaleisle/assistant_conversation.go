package main

import (
	"context"
	"encoding/json"
	"fmt"
	"github.com/egoist/mygo"
	"os"
	"path/filepath"
	"strings"
	"time"

	"whaleisle/internal/dsh"
	"whaleisle/internal/harness"
	"whaleisle/internal/paths"
	assiststore "whaleisle/internal/store/assistant"
	"whaleisle/internal/ui/i18n"
)

// This panel is an entry point, not another transcript or prompt transport.
// The existing session/composer owns streaming, tool approvals, stop and retry.
func (s *mainState) assistantConversationCurrent(ep harness.Endpoint, client *dsh.Client, generation uint64, mode assiststore.Mode, sidebar ...bool) bool {
	s.dshMu.Lock()
	same := client != nil && s.dshClient == client && client.Endpoint.Origin == ep.Origin && client.Endpoint.Cookie == ep.Cookie
	s.dshMu.Unlock()
	s.mu.Lock()
	defer s.mu.Unlock()
	direct := len(sidebar) > 0 && sidebar[0]
	if !same || !s.assistantOpen && !direct || s.assistantConversationGeneration != generation || s.bootStatus.State != harness.Ready || s.bootStatus.Endpoint != ep || s.cfgStore == nil {
		return false
	}
	cfg := s.cfgStore.Get()
	if mode == assiststore.ModeDshbot {
		return cfg.DshbotEnabled
	}
	return cfg.WhaleAssistantEnabled
}

func (s *mainState) openAssistantConversation(ep harness.Endpoint, mode assiststore.Mode, botID, text string) {
	s.openAssistantConversationIntent(ep, mode, botID, text, false)
}

func persistedSessionExistsOnDisk(dshHome, sid string) bool {
	sid = strings.TrimSpace(sid)
	if sid == "" || dshHome == "" {
		return false
	}
	if info, err := os.Stat(filepath.Join(dshHome, "sessions", sid)); err == nil && info.IsDir() {
		return true
	}
	if matches, err := filepath.Glob(filepath.Join(dshHome, "sessions", "*", sid)); err == nil {
		for _, m := range matches {
			if info, statErr := os.Stat(m); statErr == nil && info.IsDir() {
				return true
			}
		}
	}
	return false
}

func (s *mainState) whaleAssistantPersistedInfo() (sessionID, name, homeDir string) {
	homeDir = filepath.Join(paths.DshHome(), "data", "whale")
	name = "鲸鱼娘"
	if raw, err := os.ReadFile(filepath.Join(homeDir, "settings.json")); err == nil {
		var disk struct {
			Name      string `json:"name"`
			SessionID string `json:"sessionId"`
		}
		if json.Unmarshal(raw, &disk) == nil {
			if n := strings.TrimSpace(disk.Name); n != "" {
				name = n
			}
			if sid := strings.TrimSpace(disk.SessionID); sid != "" {
				if persistedSessionExistsOnDisk(paths.DshHome(), sid) {
					sessionID = sid
				}
			}
		}
	}
	if s == nil {
		return sessionID, name, homeDir
	}
	s.mu.Lock()
	asst, sessions := s.assistant, s.sessions
	s.mu.Unlock()
	if sessions != nil {
		snap := sessions.Snapshot()
		for _, item := range snap.Items {
			if item.Presentation != nil && item.Presentation.Owner == "dsh-whale:assistant" {
				if t := strings.TrimSpace(item.Presentation.Title); t != "" {
					name = t
				}
				if w := strings.TrimSpace(item.Presentation.WorkingDirectory); w != "" {
					homeDir = w
				}
				if sessionID == "" {
					sessionID = strings.TrimSpace(item.SessionID)
				}
				break
			}
		}
	}
	if sessionID == "" && asst != nil {
		if sid := strings.TrimSpace(asst.Snapshot().SessionID); sid != "" {
			if persistedSessionExistsOnDisk(paths.DshHome(), sid) {
				sessionID = sid
			}
		}
	}
	return sessionID, name, homeDir
}

// The sidebar's default action opens the canonical conversation immediately;
// the native management page remains a separate entry with its own drafts.
func (s *mainState) openWhaleFromSidebar(win *mygo.Window) {
	s.revealMainConversation()
	s.mu.Lock()
	ep := s.bootStatus.Endpoint
	ready := s.bootStatus.State == harness.Ready
	enabled := s.cfgStore != nil && s.cfgStore.Get().WhaleAssistantEnabled
	sessions := s.sessions
	asst := s.assistant
	s.mu.Unlock()
	if !enabled {
		s.notifyUser(win, hostNotReadyToast())
		return
	}
	s.dshMu.Lock()
	client := s.dshClient
	s.dshMu.Unlock()
	knownID, whaleName, whaleHome := s.whaleAssistantPersistedInfo()
	if knownID != "" && sessions != nil {
		s.mu.Lock()
		s.resumePendingSession = ""
		s.mu.Unlock()
		s.revokePhoneGit()
		sessions.EnsureSessionPresentation(knownID, "dsh-whale:assistant", whaleName, whaleHome)
		sessions.Select(knownID)
		if asst != nil {
			asst.SetSessionID(knownID)
		}
	}
	if !ready || client == nil {
		if knownID == "" {
			s.notifyUser(win, hostNotReadyToast())
		}
		return
	}
	s.openAssistantConversationIntent(ep, assiststore.ModeWhale, "", "", true)
}

func (s *mainState) openAssistantConversationIntent(ep harness.Endpoint, mode assiststore.Mode, botID, text string, sidebar bool) {
	s.mu.Lock()
	asst, sessions, win := s.assistant, s.sessions, s.win
	if asst == nil || sessions == nil || win == nil || !s.assistantOpen && !sidebar || (!sidebar && (s.assistantPluginBusy || asst.Snapshot().Busy)) {
		s.mu.Unlock()
		return
	}
	s.assistantConversationGeneration++
	generation := s.assistantConversationGeneration
	s.mu.Unlock()
	mode = assiststore.NormalizeMode(mode)
	if mode == assiststore.ModeDshbot {
		if strings.TrimSpace(botID) == "" {
			asst.SetLastAction(i18n.T(i18n.KeyCmdSelectBotOrRoom), i18n.T(i18n.KeyCmdSelectBotOrRoom))
			return
		}
		asst.SelectDshbot(botID, ep.Origin)
	}
	owner, err := sessions.CaptureComposerTarget(sessions.SelectedSessionID())
	if err != nil {
		asst.SetLastAction(err.Error(), err.Error())
		if sidebar {
			s.notifyUser(win, err.Error())
		}
		return
	}
	s.dshMu.Lock()
	client := s.dshClient
	s.dshMu.Unlock()
	if !s.assistantConversationCurrent(ep, client, generation, mode, sidebar) {
		return
	}
	asst.SetBusy(true)
	asst.SetLastAction(i18n.T(i18n.KeyCmdOpeningCanonicalConversation), "")
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
			if !s.assistantConversationCurrent(ep, client, generation, mode, sidebar) || ((!sidebar || text != "") && !owner.NavigationCurrent()) {
				return
			}
			if openErr == nil && strings.TrimSpace(sid) == "" {
				openErr = fmt.Errorf("%s", i18n.T(i18n.KeyCmdCanonicalSessionMissing))
			}
			if openErr != nil {
				asst.SetLastAction(openErr.Error(), openErr.Error())
				if sidebar && !(mode == assiststore.ModeWhale && strings.HasPrefix(sessions.SelectedSessionID(), "session-whale-")) {
					s.notifyUser(win, openErr.Error())
				}
				return
			}
			// Preserve an existing session draft rather than submit it accidentally.
			s.mu.Lock()
			s.resumePendingSession = ""
			draft := s.composerDraftLocked(sid)
			conflict := text != "" && (draft.Text != "" || len(draft.Attachments) != 0 || draft.Sending || draft.Preparing > 0 || draft.EditSeq != 0)
			s.mu.Unlock()
			if conflict {
				asst.SetLastAction(i18n.T(i18n.KeyCmdResolveTheExistingChatDraftBeforeS), i18n.T(i18n.KeyCmdDraftRetainedNothingSent))
				return
			}
			s.revokePhoneGit()
			if mode == assiststore.ModeWhale {
				_, whaleName, whaleHome := s.whaleAssistantPersistedInfo()
				sessions.EnsureSessionPresentation(sid, "dsh-whale:assistant", whaleName, whaleHome)
			}
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
			asst.SetLastAction(i18n.T(i18n.KeyCmdCanonicalConversationOpened), "")
			if text != "" {
				// Ownership moves to the visible canonical draft. Failed/uncertain sends
				// stay in the standard composer/submission UI and are never auto-retried.
				asst.AcceptPrompt(text)
				snap := sessions.Snapshot()
				s.submitComposer(win, sessions, draft, sid, snap.ScratchCwd, 0, false, snap.ComposerHostGeneration)
			}
			if !sessions.SelectionSummary().ListLoading {
				sessions.RefreshList(context.Background())
			}
		})
	}()
}
