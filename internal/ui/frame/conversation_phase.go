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

// ConversationMainPanel's summaryBlank exception is authoritative even while
// opening. Unknown history must settle before deciding whether it is blank.
func conversationPhase(snap session.ViewState, interactions bool) conversationLayoutPhase {
	if len(snap.Messages) > 0 || interactions || snap.PromptBusy || snap.PromptAttempted {
		return conversationActive
	}
	for _, item := range snap.Items {
		if item.SessionID == snap.SelectedID && item.Running {
			return conversationActive
		}
	}
	if snap.SelectedID == "" || snap.SelectedSummaryBlank {
		return conversationHero
	}
	if snap.HistoryError != "" {
		return conversationActive
	}
	if !snap.HistoryReady {
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
