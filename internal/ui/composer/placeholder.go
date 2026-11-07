package composer

import (
	"strings"

	"whaleisle/internal/store/session"
	"whaleisle/internal/ui/i18n"
)

// WorkspacePlaceholder returns the localized placeholder when the composer is
// inert because no workspace is selected (`placeholder.workspace`).
func WorkspacePlaceholder() string {
	return composerText("选择一个工作区开始", "Choose a workspace to start")
}

// ParentOfflinePlaceholder returns the localized placeholder when a continuable
// child session's parent session is offline (`placeholder.parentOffline`).
func ParentOfflinePlaceholder() string {
	return composerText(
		"父会话已离线，无法继续发送；仍可停止当前运行",
		"Parent session offline; sending is unavailable but you can still stop the run",
	)
}

// UnavailablePlaceholder returns the localized placeholder when the session
// composer is disabled (`placeholder.unavailable`).
func UnavailablePlaceholder() string {
	return composerText("会话不可用", "Session unavailable")
}

// ManagedPlaceholder returns the localized placeholder for a managed-composer
// session (`placeholder.managed`).
func ManagedPlaceholder() string {
	return composerText("发送消息", "Send a message")
}

// SessionInUseNotice returns the localized notice when a session writer lock is
// held by another running instance (`error.sessionInUse` for `session/writer-held`).
func SessionInUseNotice() string {
	return composerText(
		"当前会话已被占用，可能是其他正在运行的 DSH 导致的（如其他 dsh web、桌面端），请退出其他正在运行的 DSH 后重试。",
		"This session is already in use, possibly by another running DSH instance (such as dsh web or the desktop app). Quit other running DSH instances and try again.",
	)
}

// FormatComposerError translates known Host error codes (such as `session/writer-held`)
// into their localized user-facing notice while preserving ordinary error text.
func FormatComposerError(err string) string {
	if strings.Contains(err, "session/writer-held") {
		return SessionInUseNotice()
	}
	return err
}

// HeroHeadline returns the unified EmptyHero headline (`hero.headline`,
// superseding the legacy model-split `hero.claudeHeadline`).
func HeroHeadline() string {
	return i18n.T(i18n.KeyChatHeroSlogan)
}

// HeroBadge returns the EmptyHero preview badge (`hero.badge`).
func HeroBadge() string {
	return i18n.T(i18n.KeyChatHeroPreview)
}

func composerItemParentOffline(snap session.ViewState, item session.Item) bool {
	if item.SubagentIdentity == nil || item.SubagentIdentity.Mode != "continuable" {
		return false
	}
	if item.ParentID != "" {
		for _, parent := range snap.Items {
			if parent.SessionID == item.ParentID {
				return !parent.AgentAvailable
			}
		}
	}
	return !item.AgentAvailable
}

func composerParentOffline(m Model) bool {
	if m.ParentOffline {
		return true
	}
	if m.Snapshot.SelectedID == "" {
		return false
	}
	for _, item := range m.Snapshot.Items {
		if item.SessionID == m.Snapshot.SelectedID {
			return composerItemParentOffline(m.Snapshot, item)
		}
	}
	return false
}

func composerManagedSession(snap session.ViewState) bool {
	if snap.SelectedID == "" {
		return false
	}
	for _, item := range snap.Items {
		if item.SessionID == snap.SelectedID {
			return item.Presentation != nil && item.Presentation.Composer == "managed"
		}
	}
	return false
}

// Both editor implementations read the same live projection; no mode cache is
// inferred from durable plan events or optimistic command acknowledgments.
func composerPlaceholder(m Model) string {
	if m.Placeholder != "" {
		return m.Placeholder
	}
	if composerParentOffline(m) {
		return ParentOfflinePlaceholder()
	}
	if m.Inert {
		return WorkspacePlaceholder()
	}
	if m.Disabled {
		return UnavailablePlaceholder()
	}
	if m.Hero {
		return i18n.T(i18n.KeyComposerPlaceholderHero)
	}
	if composerManagedSession(m.Snapshot) {
		return ManagedPlaceholder()
	}
	if !m.Snapshot.ClientReady {
		return i18n.T(i18n.KeyComposerPlaceholder)
	}
	// The available empty-draft queue action outranks the passive plan hint.
	if !m.Editing && m.State != nil && m.State.CanSteerPendingQueue(m.Snapshot.SelectedID, true) && !m.State.PermissionCommand.Visible(m.Snapshot.SelectedID) {
		for _, item := range m.Snapshot.PlanBoard.Queue {
			if m.Snapshot.PlanBoard.QueueActions.Can(session.QueueActionSteer, item) {
				return i18n.T(i18n.KeyComposerPlaceholderSteerQueue)
			}
		}
	}
	if m.Snapshot.PlanModeControl.Present && m.Snapshot.PlanModeControl.Effective {
		return i18n.T(i18n.KeyComposerPlaceholderPlan)
	}
	return i18n.T(i18n.KeyComposerPlaceholder)
}
