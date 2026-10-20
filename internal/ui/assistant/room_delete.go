package assistant

import (
	"whaleisle/internal/ui/i18n"
)

func roomDeleteConfirmLabel() string {
	return i18n.T(i18n.KeyMiscAssistantPermanentlyDisbandReviewedRoom)
}
func roomDeleteEffectsWarning() string {
	return i18n.T(i18n.KeyW4AssistantDangerousActionDeletesThisGroupSCatalo)
}

func botDeleteReviewedLabels(kind string) (effects, confirm, cancel, idLabel string) {
	if kind == "room" {
		return roomDeleteEffectsWarning(), roomDeleteConfirmLabel(), i18n.T(i18n.KeyMiscAssistantCancelAndKeepRoom), i18n.T(i18n.KeyMiscAssistantTypeExactRoomId)
	}
	return botDeleteEffectsWarning(), botDeleteConfirmLabel(), i18n.T(i18n.KeyMiscAssistantCancelAndKeepBot), i18n.T(i18n.KeyMiscAssistantTypeExactBotId)
}
