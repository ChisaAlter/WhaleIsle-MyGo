package composer

import (
	"whaleisle/internal/store/session"
	"whaleisle/internal/ui/i18n"
)

// Both editor implementations read the same live projection; no mode cache is
// inferred from durable plan events or optimistic command acknowledgments.
func composerPlaceholder(m Model) string {
	if m.Inert {
		return i18n.T(i18n.KeyChatHeroChooseWorkspace)
	}
	if m.Hero {
		return i18n.T(i18n.KeyComposerPlaceholderHero)
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
