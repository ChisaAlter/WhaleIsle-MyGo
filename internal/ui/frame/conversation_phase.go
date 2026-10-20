package frame

import (
	"context"
	"strings"

	"github.com/egoist/mygo/ui"
	"whaleisle/internal/store/session"
	"whaleisle/internal/ui/chrome"
	"whaleisle/internal/ui/i18n"
	"whaleisle/internal/ui/theme"
)

type conversationLayoutPhase uint8

// ConversationRoot.scrollBody keeps a two-DIP edge margin and a stable five-
// DIP WebKit scrollbar gutter (ui-theme/src/styles/scrollbar.css).
const (
	conversationScrollMargin   = float32(2)
	conversationScrollbarWidth = float32(5)
	conversationScrollGutter   = conversationScrollMargin + conversationScrollbarWidth
)

const (
	conversationSettling conversationLayoutPhase = iota
	conversationHero
	conversationActive
)

// conversationPhase decides which of the three blank-draft surfaces paints.
//
// An empty History is always the Hero (Original ConversationMainPanel shows the
// blank-draft Hero for every empty session), never the 13px "select a session"
// hint. Only two states keep the transcript: a Host-managed presentation that
// owns the surface (a published page or a webview session) and an explicit
// history error, which needs its retry row. A selected session whose history is
// still unknown settles: the skeleton stands in for Hero until the page lands,
// so opening a session cannot flash an empty transcript.
func conversationPhase(snap session.ViewState, interactions bool) conversationLayoutPhase {
	if len(snap.Messages) > 0 || interactions || snap.PromptBusy || snap.PromptAttempted {
		return conversationActive
	}
	for _, item := range snap.Items {
		if item.SessionID != snap.SelectedID {
			continue
		}
		// A managed presentation replaces the transcript with Host content; a
		// running session keeps the Hero until its first message arrives, which
		// is what the Original blank-draft surface does.
		if item.Presentation != nil && item.Presentation.Composer == "managed" {
			return conversationActive
		}
	}
	// An authoritative blank summary outranks the transport error: the Hero
	// stays and the shared error row above it carries the failure.
	if snap.HistoryError != "" && !snap.SelectedSummaryBlank {
		return conversationActive
	}
	if snap.SelectedID != "" && !snap.HistoryReady && !snap.SelectedSummaryBlank {
		return conversationSettling
	}
	return conversationHero
}

func conversationHistoryStatus(c *ui.Context, m Model, snap session.ViewState, muted ui.Color) {
	if snap.HistoryError == "" {
		return
	}
	ui.Row(c).Key("conversation-history-error").FillWidth().AlignItems(ui.Center).Gap(8).Padding(8, 16).Children(func() {
		// LastAction can contain this same transport error; its shell rendering
		// already supplies the text, while this row preserves an explicit retry.
		if !strings.Contains(snap.LastAction, snap.HistoryError) {
			ui.Text(c, snap.HistoryError).FontSize(theme.InterfaceSize(12)).TextColor(muted).Grow(1).MinWidth(0)
		}
		if ui.Button(c, i18n.T(i18n.KeyCommandCatalogRetry)).Key("conversation-history-retry").Label("Retry conversation history").Disabled(snap.HistoryLoading).Clicked() {
			m.Store.FollowSelected(context.Background())
		}
	})
}

// ConversationRoot.composerHero centers the entire stack inside scrollBody,
// with a 32 DIP foot; HeroShell supplies the title's 10/7 spacing and 26/32 type.
func conversationHeroHeadline(c *ui.Context, m Model, colors theme.Colors) {
	ui.Row(c).Key("conversation-hero-headline").FillWidth().Wrap().Gap(10).Justify(ui.Center).AlignItems(ui.Center).Padding(0, 16, 12, 16).Children(func() {
		conversationHeroFish(c, chrome.Color(colors.FG))
		ui.Row(c).Wrap().Gap(7).Justify(ui.Center).AlignItems(ui.Center).MinWidth(0).Children(func() {
			ui.Text(c, i18n.T(i18n.KeyChatHeroSlogan)).FontSize(theme.InterfaceSize(26)).FixedLineHeight(theme.InterfaceSize(32)).FontWeight(500).TextColor(chrome.Color(colors.FG))
			badgeBackground := chrome.Color("rgba(120, 160, 210, 0.18)")
			badgeForeground := chrome.Color("rgb(70, 110, 160)")
			badgeBorder := chrome.Color("rgba(120, 160, 210, 0.35)")
			th := theme.Resolve(m.ThemeID)
			if th.Scheme == theme.SchemeDark || (th.Scheme == theme.SchemeSystem && m.SystemDark) {
				badgeForeground = chrome.Color(colors.Accent)
				badgeBackground = chrome.Color("rgba(103, 158, 254, 0.16)")
				badgeBorder = chrome.Color("rgba(103, 158, 254, 0.35)")
			}
			ui.Text(c, i18n.T(i18n.KeyChatHeroPreview)).FontSize(theme.InterfaceSize(12)).FixedLineHeight(theme.InterfaceSize(18)).FontWeight(500).
				TextColor(badgeForeground).Padding(1, 7, 0, 7).
				Background(badgeBackground).Border(0.5, badgeBorder).Radius(999)
		})
	})
}

// conversationSettlingSkeleton holds the Hero's geometry while the first page
// of history is still unknown: the fish and title keep their place, and the
// input card keeps its exact size, so nothing jumps when the composer mounts.
// The Original settles on the Hero shell rather than on a blank box.
func conversationSettlingSkeleton(c *ui.Context, colors theme.Colors) {
	placeholder := chrome.Color(colors.Hover)
	if placeholder.A == 0 {
		placeholder = chrome.Color(colors.Field).Alpha(.7)
	}
	border := chrome.Color(colors.Border)
	ui.Column(c).Key("conversation-settling-skeleton").Label("Loading conversation").FillWidth().MinWidth(0).Gap(14).Children(func() {
		ui.Row(c).FillWidth().Gap(10).Justify(ui.Center).AlignItems(ui.Center).Padding(0, 16).Children(func() {
			conversationHeroFish(c, chrome.Color(colors.Muted))
			ui.Box(c).Width(196).Height(26).Radius(8).Background(placeholder)
		})
		ui.Column(c).FillWidth().MinWidth(0).Padding(0, 16).Children(func() {
			ui.Column(c).FillWidth().MinWidth(0).Height(96).Radius(28).
				Background(placeholder).Border(1, border).Padding(16, 18, 16, 18).Gap(10).Children(func() {
				ui.Box(c).FillWidth().MaxWidthPercent(58).Height(13).Radius(6).Background(border.Alpha(.55))
				ui.Box(c).FillWidth().MaxWidthPercent(34).Height(13).Radius(6).Background(border.Alpha(.4))
				ui.Box(c).Grow(1)
				ui.Row(c).FillWidth().Gap(8).Children(func() {
					ui.Box(c).Width(28).Height(28).Radius(14).Background(border.Alpha(.35))
					ui.Box(c).Grow(1)
					ui.Box(c).Width(56).Height(28).Radius(14).Background(border.Alpha(.45))
				})
			})
		})
	})
}
