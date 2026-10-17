package chat

import (
	"strings"

	"github.com/egoist/mygo/ui"

	"whaleisle/internal/store/session"
	"whaleisle/internal/ui/i18n"
)

// FeedbackRowModel is one assistant message's thumbs controls.
type FeedbackRowModel struct {
	MessageID   string
	Selected    string // "", session.MessageFeedbackUp, or MessageFeedbackDown
	Busy        bool
	CommentOpen bool
	Comment     *string
	OnVote       func(messageID, rating, note string)
	OnClear      func(messageID string)
	OnOpenDown   func(messageID string)
	OnCancelDown func()
}

// FeedbackButtonLabel returns the vote button label, marking the active rating.
func FeedbackButtonLabel(rating, selected string) string {
	base := i18n.T(i18n.KeyChatFeedbackUp)
	if rating == session.MessageFeedbackDown {
		base = i18n.T(i18n.KeyChatFeedbackDown)
	}
	if selected == rating && selected != "" {
		return base + i18n.T(i18n.KeyChatFeedbackSelectedMark)
	}
	return base
}

// FeedbackControlsDisabled is true while a put is in flight for any message.
func FeedbackControlsDisabled(busy bool) bool {
	return busy
}

// ShouldAcceptFeedbackClick rejects spam when busy. Same rating is accepted so the
// UI can clear via OnClear (toggle-off).
func ShouldAcceptFeedbackClick(selected, rating string, busy bool) bool {
	if busy || rating == "" {
		return false
	}
	return true
}

// ShouldClearFeedbackClick is true when the user re-clicks the active rating.
func ShouldClearFeedbackClick(selected, rating string, busy bool) bool {
	return !busy && rating != "" && selected == rating
}

// ShouldShowDislikeComment shows the optional reason field while composing a dislike.
func ShouldShowDislikeComment(commentOpen bool) bool {
	return commentOpen
}

// FeedbackStatusLine returns LastAction when it is feedback-related (else "").
func FeedbackStatusLine(lastAction string) string {
	s := strings.TrimSpace(lastAction)
	if s == "" {
		return ""
	}
	lower := strings.ToLower(s)
	if strings.HasPrefix(lower, "feedback") {
		return s
	}
	return ""
}

func feedbackRow(c *ui.Context, m FeedbackRowModel) {
	if m.OnVote == nil {
		return
	}
	disabled := FeedbackControlsDisabled(m.Busy)
	ui.Column(c).Gap(4).Padding(2, 0, 0, 0).Children(func() {
		ui.Row(c).Gap(6).Children(func() {
			up := ui.Button(c, FeedbackButtonLabel(session.MessageFeedbackUp, m.Selected))
			if disabled {
				up = up.Disabled(true)
			} else if up.Clicked() && ShouldAcceptFeedbackClick(m.Selected, session.MessageFeedbackUp, false) {
				if ShouldClearFeedbackClick(m.Selected, session.MessageFeedbackUp, false) && m.OnClear != nil {
					if m.OnCancelDown != nil {
						m.OnCancelDown()
					}
					m.OnClear(m.MessageID)
				} else {
					if m.OnCancelDown != nil {
						m.OnCancelDown()
					}
					m.OnVote(m.MessageID, session.MessageFeedbackUp, "")
				}
			}
			down := ui.Button(c, FeedbackButtonLabel(session.MessageFeedbackDown, m.Selected))
			if disabled {
				down = down.Disabled(true)
			} else if down.Clicked() {
				if ShouldClearFeedbackClick(m.Selected, session.MessageFeedbackDown, false) && m.OnClear != nil {
					if m.OnCancelDown != nil {
						m.OnCancelDown()
					}
					m.OnClear(m.MessageID)
					return
				}
				if m.OnOpenDown != nil {
					m.OnOpenDown(m.MessageID)
				}
			}
		})
		if ShouldShowDislikeComment(m.CommentOpen) && m.Comment != nil {
			ui.Row(c).Gap(6).AlignItems(ui.Center).Children(func() {
				inp := ui.TextInput(c, m.Comment).Grow(1).
					Placeholder(i18n.T(i18n.KeyChatFeedbackCommentPlaceholder))
				send := ui.Button(c, i18n.T(i18n.KeyChatFeedbackSend))
				if disabled {
					send = send.Disabled(true)
				} else {
					submit := func() {
						note := ""
						if m.Comment != nil {
							note = strings.TrimSpace(*m.Comment)
						}
						m.OnVote(m.MessageID, session.MessageFeedbackDown, note)
						if m.OnCancelDown != nil {
							m.OnCancelDown()
						}
					}
					if inp.Submitted() {
						submit()
					}
					if send.Clicked() {
						submit()
					}
				}
			})
		}
	})
}
