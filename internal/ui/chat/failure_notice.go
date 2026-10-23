package chat

import (
	"strings"

	"github.com/egoist/mygo/ui"

	"whaleisle/internal/store/session"
	"whaleisle/internal/ui/chrome"
	"whaleisle/internal/ui/i18n"
	"whaleisle/internal/ui/theme"
	"whaleisle/internal/ui/widgets"
)

type turnFailure struct {
	Message string
	Code    string
}

// terminalFailure retains the failure at its durable turn position. Account
// sign-out is the one aborted reason presented as a terminal error notice.
func terminalFailure(msg session.Message) (turnFailure, bool) {
	r := readTurnReason(msg)
	switch {
	case r.Kind == "error" || r.Kind == "failed":
		message, code := displayFailure(r.Error)
		return turnFailure{Message: message, Code: code}, true
	case r.Kind == "aborted" && r.Reason.Kind == "hook" && r.Reason.Reason == "deepseek-account/signed-out":
		return turnFailure{Message: i18n.T(i18n.KeyChatRetrySignedOut), Code: "ACCOUNT_SIGNED_OUT"}, true
	default:
		return turnFailure{}, false
	}
}

// splitTurnFailure reads the legacy system-message suffix. Durable turn errors
// carry a separate code and do not pass through this compatibility path.
func splitTurnFailure(body string) (text, code string) {
	if index := strings.LastIndex(body, " MISSING_"); index >= 0 {
		return strings.TrimSpace(body[:index]), strings.TrimSpace(body[index+1:])
	}
	return strings.TrimSpace(body), ""
}

// paintTurnFailure matches the reference's persistent, nonexpandable error row:
// state dot, inline title and reason, followed by the optional provider code.
func paintTurnFailure(c *ui.Context, failure turnFailure, palette theme.Colors) {
	title := i18n.T(i18n.KeyChatRunFailed)
	if failure.Code == "ACCOUNT_SIGNED_OUT" {
		title = i18n.T(i18n.KeyChatAccountStopped)
	}
	size := secondaryChatSize()
	label := strings.TrimSpace(title + " " + failure.Message + " " + failure.Code)
	ui.Row(c).FillWidth().MinWidth(0).Padding(2, 0, 2, 0).Gap(8).AlignItems(ui.Start).
		Role(ui.RoleStatus).Label(label).Children(func() {
		ui.Box(c).Margin(5, 0, 0, 0).Shrink(0).Children(func() {
			widgets.StateDot(c, widgets.StateError)
		})
		ui.RichText(c,
			ui.Span{Text: title + "  ", Weight: 600, Color: chrome.Color(palette.Danger)},
			ui.Span{Text: failure.Message, Color: chrome.Color(palette.Secondary)},
		).Font(theme.RenderPreferences().Chat.Family).FontSize(size).FixedLineHeight(size + 7).
			Grow(1).MinWidth(0).Selectable()
		if failure.Code != "" {
			ui.Text(c, failure.Code).Font(theme.RenderPreferences().Code.Family).FontSize(11).
				FixedLineHeight(16).TextColor(chrome.Color(palette.Muted)).Shrink(0)
		}
	})
}
