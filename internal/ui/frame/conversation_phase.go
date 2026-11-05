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
	// composerCardMaxWidth is the installed InputBar card's `max-width: 774.4px`
	// (probed at 1440x860 and 1440x920: card 774.39 wide, 114 tall).
	composerCardMaxWidth = float32(774.39)
	// HeroShell.stack leaves 12 DIP below the headline for its empty body;
	// ConversationRoot.composerStack adds an 8-DIP gap before the workspace row.
	conversationHeroHeadlineFoot = float32(12 + 8)
	// ConversationRoot.composerHero keeps 32 DIP of bottom padding inside the
	// centered flex item. There are no extra fixed blocks above or below it.
	conversationHeroStackFoot = float32(32)
	// The official 34-DIP brand slot sets the headline row's height; the slogan
	// itself retains its 26/32 type and is centered within that row.
	conversationHeroHeadlineHeight = float32(34)
	// conversationHeroPresetRowHeight is the hero preset row's outer height:
	// 4 DIP top margin + 28 DIP row + 8 DIP bottom margin.
	conversationHeroPresetRowHeight = float32(40)
	// InputBar.card: 8 top + 52 min input + 12 gap + toolbar (2 + 34 + 6).
	conversationHeroCardHeight = float32(114)
	// conversationHeroStackHeight is the native hero stack: slogan top to card
	// bottom. It is the sum of the seats above, kept as one number so a change to
	// any part fails the seat test instead of silently re-centring the stack.
	conversationHeroStackHeight = conversationHeroHeadlineHeight + conversationHeroHeadlineFoot + conversationHeroPresetRowHeight + conversationHeroCardHeight
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
	if conversationHasTurnContent(snap) || interactions || snap.PromptBusy || snap.PromptAttempted {
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

// Command audit records are not the first conversational turn. Configuration
// commands still belong to History/Trajectory, but cannot move a fresh draft
// out of its welcome surface merely because their run/done events arrived.
func conversationHasTurnContent(snap session.ViewState) bool {
	return snap.HasConversationContent()
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

// HeroShell supplies a naturally sized headline. Its source-defined empty-body
// and composer-stack gaps are represented by conversationHeroHeadlineFoot;
// ConversationRoot centers the complete item including its 32-DIP bottom pad.
func conversationHeroHeadline(c *ui.Context, m Model, colors theme.Colors) {
	// Installed HeroShell uses an outer10-DIP gap and a7-DIP title-group gap.
	skin := theme.ResolvedSkin(m.ThemeID, m.SystemDark)
	size, lineHeight, weight := theme.InterfaceSize(26), theme.InterfaceSize(32), 500
	if skin == theme.SkinCodex {
		size, lineHeight, weight = 28, 28*1.35, 600
	} else if skin == theme.SkinClaude {
		size, lineHeight, weight = 30, 30*1.3, 400
	}
	font := theme.FontForSkin(theme.RenderPreferences(), skin, theme.SkinHeroFont)
	ui.Row(c).Key("conversation-hero-headline").Label("Conversation hero headline").FillWidth().Wrap().Gap(10).Justify(ui.Center).AlignItems(ui.Center).Padding(0, 16, conversationHeroHeadlineFoot, 16).Children(func() {
		if skin != theme.SkinCodex {
			conversationHeroFish(c, chrome.Color(colors.FG))
		}
		ui.Row(c).Wrap().Gap(7).Justify(ui.Center).AlignItems(ui.Center).MinWidth(0).Children(func() {
			headline := ui.Text(c, i18n.T(i18n.KeyChatHeroSlogan)).Font(font.Family).FontSize(size).FixedLineHeight(lineHeight).FontWeight(weight).TextColor(chrome.Color(colors.FG))
			if skin == theme.SkinCodex {
				headline.LetterSpacing(-.02 * size).MaxWidth(640)
			} else if skin == theme.SkinClaude {
				headline.LetterSpacing(-.01 * size)
			}
			if skin == theme.SkinCodex {
				return
			}
			badgeBG, badgeFG := theme.HeroBadgeColors(m.ThemeID, m.SystemDark)
			ui.Text(c, i18n.T(i18n.KeyChatHeroPreview)).Font(theme.RenderPreferences().Code.Family).FontSize(12).FixedLineHeight(18).FontWeight(500).
				TextColor(chrome.Color(badgeFG)).Padding(1, 7, 0, 7).AlignSelf(ui.Start).Margin(2, 0, 0, 0).
				Background(chrome.Color(badgeBG)).Border(.5, chrome.Color(colors.Hover)).Radius(999)
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
