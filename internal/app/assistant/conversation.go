package assistantctl

import (
	"context"
	"encoding/json"
	"fmt"
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

// ConversationCurrent reports whether ep/client/generation/mode still own the
// active assistant panel (or direct sidebar invocation when sidebar[0] is true).
func (s *Service) ConversationCurrent(ep harness.Endpoint, client *dsh.Client, generation uint64, mode assiststore.Mode, sidebar ...bool) bool {
	if s == nil {
		return false
	}
	host := s.safeHost()
	liveClient := host.CurrentClient()
	same := client != nil && liveClient == client && client.Endpoint.Origin == ep.Origin && client.Endpoint.Cookie == ep.Cookie
	if !same {
		return false
	}
	bootEp, bootState, _ := host.BootEndpoint()
	cfgStore := host.Config()
	s.mu.Lock()
	open := s.open
	currentGen := s.conversationGeneration
	s.mu.Unlock()
	direct := len(sidebar) > 0 && sidebar[0]
	if (!open && !direct) || currentGen != generation || bootState != harness.Ready || bootEp != ep || cfgStore == nil {
		return false
	}
	cfg := cfgStore.Get()
	if mode == assiststore.ModeDshbot {
		return cfg.DshbotEnabled
	}
	return cfg.WhaleAssistantEnabled
}

// OpenConversation resolves and navigates to the canonical Whale or Dshbot session.
func (s *Service) OpenConversation(ep harness.Endpoint, mode assiststore.Mode, botID, text string) {
	if s == nil {
		return
	}
	s.openConversationIntent(ep, mode, botID, text, false)
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

// WhalePersistedInfoFromDisk reads the persisted Whale assistant metadata on disk.
func WhalePersistedInfoFromDisk(dshHome string) (sessionID, name, homeDir string) {
	if dshHome == "" {
		dshHome = paths.DshHome()
	}
	homeDir = filepath.Join(dshHome, "data", "whale")
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
				if persistedSessionExistsOnDisk(dshHome, sid) {
					sessionID = sid
				}
			}
		}
	}
	return sessionID, name, homeDir
}

// WhalePersistedInfo returns the known session ID, display name, and working directory
// for the Whale assistant, safe to call on a nil Service.
func (s *Service) WhalePersistedInfo() (sessionID, name, homeDir string) {
	dshHome := paths.DshHome()
	if s != nil {
		dshHome = s.safeHost().DshHome()
	}
	sessionID, name, homeDir = WhalePersistedInfoFromDisk(dshHome)
	if s == nil {
		return sessionID, name, homeDir
	}
	host := s.safeHost()
	sessions := host.Sessions()
	asst := s.AssistantStore()
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
			if persistedSessionExistsOnDisk(dshHome, sid) {
				sessionID = sid
			}
		}
	}
	return sessionID, name, homeDir
}

// OpenWhaleFromSidebar opens the canonical Whale session directly from sidebar navigation.
func (s *Service) OpenWhaleFromSidebar() {
	if s == nil {
		return
	}
	host := s.safeHost()
	host.RevealMainConversation()
	ep, bootState, _ := host.BootEndpoint()
	ready := bootState == harness.Ready
	cfgStore := host.Config()
	enabled := cfgStore != nil && cfgStore.Get().WhaleAssistantEnabled
	sessions := host.Sessions()
	win := host.Window()
	asst := s.AssistantStore()
	if !enabled {
		host.NotifyUser(win, hostNotReadyToast())
		return
	}
	client := host.CurrentClient()
	knownID, whaleName, whaleHome := s.WhalePersistedInfo()
	if knownID != "" && sessions != nil {
		host.ClearResumePendingSession()
		host.RevokePhoneGit()
		sessions.EnsureSessionPresentation(knownID, "dsh-whale:assistant", whaleName, whaleHome)
		sessions.Select(knownID)
		if asst != nil {
			asst.SetSessionID(knownID)
		}
	}
	if !ready || client == nil {
		if knownID == "" {
			host.NotifyUser(win, hostNotReadyToast())
		}
		return
	}
	s.openConversationIntent(ep, assiststore.ModeWhale, "", "", true)
}

// HandleNativePrompt validates the active target and forwards a native panel prompt
// into the canonical chat owner.
func (s *Service) HandleNativePrompt(_ string, ep harness.Endpoint, text string) {
	if s == nil {
		return
	}
	asst := s.AssistantStore()
	if asst == nil || strings.TrimSpace(text) == "" {
		return
	}
	target := asst.Snapshot()
	if target.Mode == assiststore.ModeDshbot && target.DshbotOrigin != ep.Origin {
		asst.SetLastAction(i18n.T(i18n.KeyCmdSelectCurrentRosterBot), i18n.T(i18n.KeyCmdComposerHostChanged))
		return
	}
	s.OpenConversation(ep, target.Mode, target.DshbotID, strings.TrimSpace(text))
}

func (s *Service) openConversationIntent(ep harness.Endpoint, mode assiststore.Mode, botID, text string, sidebar bool) {
	host := s.safeHost()
	sessions := host.Sessions()
	win := host.Window()
	asst := s.AssistantStore()
	if asst == nil || sessions == nil || win == nil {
		return
	}
	asstBusy := asst.Snapshot().Busy
	s.mu.Lock()
	if (!s.open && !sidebar) || (!sidebar && (s.pluginBusy || asstBusy)) {
		s.mu.Unlock()
		return
	}
	s.conversationGeneration++
	generation := s.conversationGeneration
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
			host.NotifyUser(win, err.Error())
		}
		return
	}
	client := host.CurrentClient()
	if !s.ConversationCurrent(ep, client, generation, mode, sidebar) {
		return
	}
	asst.SetBusy(true)
	asst.SetLastAction(i18n.T(i18n.KeyCmdOpeningCanonicalConversation), "")
	rpc := host.PluginRPCClient(ep)
	go func() {
		ctx, cancel := context.WithTimeout(context.Background(), 45*time.Second)
		defer cancel()
		var sid string
		var openErr error
		if mode == assiststore.ModeDshbot {
			_, sid, openErr = s.ResolveDshbot(ctx, ep, sessions, botID)
		} else {
			result, err := rpc.WhaleEnsureTyped(ctx)
			sid, openErr = result.SessionID, err
		}
		host.WindowUpdate(win, func() {
			asst.SetBusy(false)
			if !s.ConversationCurrent(ep, client, generation, mode, sidebar) || ((!sidebar || text != "") && !owner.NavigationCurrent()) {
				return
			}
			if openErr == nil && strings.TrimSpace(sid) == "" {
				openErr = fmt.Errorf("%s", i18n.T(i18n.KeyCmdCanonicalSessionMissing))
			}
			if openErr != nil {
				asst.SetLastAction(openErr.Error(), openErr.Error())
				if sidebar && !(mode == assiststore.ModeWhale && strings.HasPrefix(sessions.SelectedSessionID(), "session-whale-")) {
					host.NotifyUser(win, openErr.Error())
				}
				return
			}
			// Preserve an existing session draft rather than submit it accidentally.
			host.ClearResumePendingSession()
			if text != "" && host.HasSessionDraftConflict(sid) {
				asst.SetLastAction(i18n.T(i18n.KeyCmdResolveTheExistingChatDraftBeforeS), i18n.T(i18n.KeyCmdDraftRetainedNothingSent))
				return
			}
			host.RevokePhoneGit()
			if mode == assiststore.ModeWhale {
				_, whaleName, whaleHome := s.WhalePersistedInfo()
				sessions.EnsureSessionPresentation(sid, "dsh-whale:assistant", whaleName, whaleHome)
			}
			sessions.Select(sid)
			if mode == assiststore.ModeDshbot {
				s.MarkOpenedBotRead(ep, client, botID, sid)
			}
			// Select already starts the canonical follow and control subscriptions.
			asst.SetSessionID(sid)
			s.mu.Lock()
			s.open = false
			s.conversationGeneration++
			if text != "" && strings.TrimSpace(s.draft) == text {
				s.draft = ""
			}
			s.mu.Unlock()
			staged := false
			if text != "" {
				staged = host.StageSessionDraft(sid, text)
			}
			asst.SetLastAction(i18n.T(i18n.KeyCmdCanonicalConversationOpened), "")
			if text != "" && staged {
				// Ownership moves to the visible canonical draft. Failed/uncertain sends
				// stay in the standard composer/submission UI and are never auto-retried.
				asst.AcceptPrompt(text)
				snap := sessions.Snapshot()
				host.SubmitComposer(win, sessions, sid, snap.ScratchCwd, snap.ComposerHostGeneration)
			}
			if !sessions.SelectionSummary().ListLoading {
				sessions.RefreshList(context.Background())
			}
		})
	}()
}
