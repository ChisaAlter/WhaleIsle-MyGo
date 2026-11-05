package interaction

import (
	"github.com/egoist/mygo/ui"
	"whaleisle/internal/store/interaction"
)

var residentComposerBounds ui.Rect

// SelectComposerInteraction uses the same priority as the installed composer
// chain: plan review, questions, then approval. Hidden/continued cards do not
// replace the resident editor.
func SelectComposerInteraction(snap interaction.ViewState) (interaction.ViewState, bool) {
	selected, priority := -1, -1
	for i, p := range snap.Pending {
		if p.Continued {
			continue
		}
		card := snap.QuestionCards[p.EventID]
		if p.Event == interaction.EventUserQuestions && (card.Hidden || card.PlanReviewHidden) {
			continue
		}
		rank := 0
		if p.Event == interaction.EventUserQuestions {
			rank = 1
			if card.PlanReview != nil {
				rank = 2
			}
		}
		if selected < 0 || rank > priority {
			selected, priority = i, rank
		}
	}
	if selected < 0 {
		return snap, false
	}
	snap.Pending = []interaction.Pending{snap.Pending[selected]}
	snap.QuestionHistory = nil
	snap.QuestionStatus = ""
	snap.AutoAllow = false
	snap.Banner = ""
	return snap, true
}
