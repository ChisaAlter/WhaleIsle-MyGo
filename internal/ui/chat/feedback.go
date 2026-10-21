package chat

import (
	"strings"

	"whaleisle/internal/store/session"
	"whaleisle/internal/ui/i18n"
)

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
