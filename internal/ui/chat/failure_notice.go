package chat

import (
	"strings"

	"github.com/egoist/mygo/ui"

	"whaleisle/internal/errcopy"
	"whaleisle/internal/store/session"
	"whaleisle/internal/ui/design"
	"whaleisle/internal/ui/i18n"
	"whaleisle/internal/ui/theme"
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
// carry a separate code and do not pass through this compatibility path. The
// body is the Host's transport text, so it takes the same sentence as a durable
// failure rather than reaching the row.
func splitTurnFailure(body string) (text, code string) {
	body = strings.TrimSpace(body)
	if index := strings.LastIndex(body, " MISSING_"); index >= 0 {
		code = strings.TrimSpace(body[index+1:])
		return errcopy.SentenceFor(code, body[:index]), code
	}
	return errcopy.SentenceText(body), ""
}

// paintTurnFailure matches the reference's persistent, nonexpandable error row:
// state dot, inline title and reason, followed by the optional provider code.
func paintTurnFailure(c *ui.Context, failure turnFailure, _ theme.Colors) {
	t := design.For(c)
	title := i18n.T(i18n.KeyChatRunFailed)
	if failure.Code == "ACCOUNT_SIGNED_OUT" {
		title = i18n.T(i18n.KeyChatAccountStopped)
	}
	size := secondaryChatSize()
	code := failure.Code
	if _, known := errcopy.HTTPStatus(failure.Code, ""); known {
		// The status is quoted inside the sentence; a bare HTTP_404 chip next to
		// it is the transport's spelling, not a second fact.
		code = ""
	}
	label := strings.TrimSpace(title + " " + failure.Message + " " + code)
	ui.Row(c).FillWidth().MinWidth(0).Padding(design.Sp(0.5), 0, design.Sp(0.5), 0).Gap(design.Sp(2)).AlignItems(ui.Start).
		Role(ui.RoleStatus).Label(label).Children(func() {
		ui.Box(c).Margin(design.Sp(1.25), 0, 0, 0).Shrink(0).Children(func() {
			chatStateDot(c, statusError)
		})
		ui.RichText(c,
			ui.Span{Text: title + "  ", Weight: design.WeightSemibold, Color: t.DestructiveForeground},
			ui.Span{Text: failure.Message, Color: t.Foreground},
		).Font(theme.RenderPreferences().Chat.Family).FontSize(size).FixedLineHeight(size + 7).
			Grow(1).MinWidth(0).Selectable()
		if code != "" {
			ui.Text(c, code).Font(theme.RenderPreferences().Code.Family).FontSize(11).
				FixedLineHeight(16).TextColor(t.MutedForeground).Shrink(0)
		}
	})
}
