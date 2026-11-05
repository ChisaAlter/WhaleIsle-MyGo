package chat

import (
	"testing"

	"whaleisle/internal/store/session"
	"whaleisle/internal/ui/i18n"
)

func TestFeedbackButtonLabelSelected(t *testing.T) {
	i18n.SetLocale("en")
	defer i18n.SetLocale(i18n.DefaultLocale)

	up := FeedbackButtonLabel(session.MessageFeedbackUp, "")
	if up != i18n.T(i18n.KeyChatFeedbackUp) {
		t.Fatalf("up idle = %q", up)
	}
	upSel := FeedbackButtonLabel(session.MessageFeedbackUp, session.MessageFeedbackUp)
	want := i18n.T(i18n.KeyChatFeedbackSelectedMark)
	if upSel != want {
		t.Fatalf("up selected = %q want %q", upSel, want)
	}
	downSel := FeedbackButtonLabel(session.MessageFeedbackDown, session.MessageFeedbackDown)
	if downSel != want {
		t.Fatalf("down selected = %q want %q", downSel, want)
	}
	if FeedbackButtonLabel(session.MessageFeedbackUp, session.MessageFeedbackDown) != i18n.T(i18n.KeyChatFeedbackUp) {
		t.Fatal("opposite rating should not mark up")
	}
}

func TestShouldAcceptFeedbackClick(t *testing.T) {
	if ShouldAcceptFeedbackClick("", session.MessageFeedbackUp, true) {
		t.Fatal("busy should reject")
	}
	if !ShouldAcceptFeedbackClick(session.MessageFeedbackUp, session.MessageFeedbackUp, false) {
		t.Fatal("same rating should accept (toggle clear)")
	}
	if !ShouldClearFeedbackClick(session.MessageFeedbackUp, session.MessageFeedbackUp, false) {
		t.Fatal("same rating should clear")
	}
	if ShouldClearFeedbackClick("", session.MessageFeedbackUp, false) {
		t.Fatal("idle should not clear")
	}
	if !ShouldAcceptFeedbackClick("", session.MessageFeedbackUp, false) {
		t.Fatal("idle up should accept")
	}
	if !ShouldAcceptFeedbackClick(session.MessageFeedbackUp, session.MessageFeedbackDown, false) {
		t.Fatal("switch vote should accept")
	}
	if ShouldAcceptFeedbackClick("", "", false) {
		t.Fatal("empty rating should reject")
	}
}

func TestShouldShowDislikeComment(t *testing.T) {
	if ShouldShowDislikeComment(false) {
		t.Fatal("idle should hide")
	}
	if !ShouldShowDislikeComment(true) {
		t.Fatal("open should show")
	}
}

func TestFeedbackControlsDisabled(t *testing.T) {
	if !FeedbackControlsDisabled(true) {
		t.Fatal("busy")
	}
	if FeedbackControlsDisabled(false) {
		t.Fatal("idle")
	}
}

func TestFeedbackStatusLine(t *testing.T) {
	if FeedbackStatusLine("prompt sent") != "" {
		t.Fatal("non-feedback")
	}
	if got := FeedbackStatusLine("feedback up saved"); got != "feedback up saved" {
		t.Fatalf("got %q", got)
	}
	if got := FeedbackStatusLine("feedback: boom"); got != "feedback: boom" {
		t.Fatalf("error line %q", got)
	}
	if FeedbackStatusLine("  ") != "" {
		t.Fatal("blank")
	}
}
