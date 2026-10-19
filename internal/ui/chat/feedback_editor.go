package chat

import (
	"context"
	"github.com/egoist/mygo/ui"
	"whaleisle/internal/store/session"
	"whaleisle/internal/ui/i18n"
)

type feedbackActionState struct {
	seeded  bool
	pending string
}

// MessageFeedbackActions seeds on first hover/focus. Feedback stays in the
// assistant's IconActions row; loading is not a separate product action.
func authoritativeFeedbackActions(c *ui.Context, store *session.Store, id string, muted ui.Color, reveal ...bool) {
	root := ui.Row(c.Key("feedback-actions-" + id)).Gap(8).AlignItems(ui.Center)
	state := ui.Local(root, "seed", func() feedbackActionState { return feedbackActionState{} })
	v := store.Feedback()
	choose := func(rating string) {
		if v.Items[id].Rating == rating {
			store.ClearMessageFeedback(context.Background(), id)
		} else {
			store.BeginFeedback(id, rating)
		}
	}
	if state.pending != "" && v.Ready && !v.Loading {
		rating := state.pending
		state.pending = ""
		choose(rating)
		v = store.Feedback()
	} else if state.pending != "" && !v.Loading && v.Error != "" {
		state.pending = ""
	}
	busy := v.BusyID != "" || v.Loading || state.pending != ""
	item := v.Items[id]
	if item.Version != "" {
		root.ContextMenu(func(menu *ui.Menu) {
			if menu.Item("编辑反馈 / Edit feedback").Disabled(busy).Chosen() {
				store.BeginFeedback(id, item.Rating)
			}
		})
	}
	root.Children(func() {
		for _, rating := range []string{session.MessageFeedbackUp, session.MessageFeedbackDown} {
			rating := rating
			icon := messageIconLikeOutline
			if rating == session.MessageFeedbackDown {
				icon = messageIconDislikeOutline
			}
			if rating == item.Rating {
				if rating == session.MessageFeedbackDown {
					icon = messageIconDislikeFill
				} else {
					icon = messageIconLikeFill
				}
			}
			b := messageAction(c, icon, FeedbackButtonLabel(rating, item.Rating), true, muted, "feedback-"+rating).Checked(rating == item.Rating).Disabled(busy)
			if len(reveal) > 0 && !reveal[0] && rating != item.Rating {
				b.Opacity(0)
			}
			if !state.seeded && (b.Hovered() || b.Focused()) {
				state.seeded = true
				if !v.Ready && !v.Loading {
					store.RefreshFeedback(context.Background())
				}
			}
			if b.Clicked() && !busy {
				if v.Ready {
					choose(rating)
				} else {
					state.pending = rating
					store.RefreshFeedback(context.Background())
				}
			}
		}
	})
}

func authoritativeFeedbackStatus(c *ui.Context, store *session.Store, muted ui.Color) {
	v := store.Feedback()
	busy := v.BusyID != "" || v.Loading
	if v.Error != "" && v.EditorID == "" {
		ui.Row(c).Gap(4).AlignItems(ui.Center).Children(func() {
			ui.Text(c, v.Error).FontSize(11).TextColor(muted).Grow(1)
			if messageAction(c, messageIconRefreshOutline, i18n.T(i18n.KeyCommandCatalogRetry), true, muted).Disabled(busy).Clicked() {
				store.RefreshFeedback(context.Background())
			}
		})
	}
}
