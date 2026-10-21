package chat

import (
	"strings"

	"github.com/egoist/mygo/ui"

	"whaleisle/internal/ui/i18n"
	"whaleisle/internal/ui/theme"
)

// turnFailureKind classifies what the user has to do about a failed turn. Only
// the kinds with an action offer one; everything else keeps the generic banner.
type turnFailureKind int

const (
	turnFailureGeneric turnFailureKind = iota
	turnFailureAPIKey
	turnFailureAuth
	turnFailureQuota
	turnFailureInterrupted
)

// turnFailure is one system failure message split into what the user needs to act
// on (localized) and the raw host text kept for diagnostics.
type turnFailure struct {
	Kind    turnFailureKind
	Title   string
	Body    string
	Action  string
	Raw     string
	Code    string
}

// apiKeyFailureMarkers are the host texts that mean "this route has no usable
// key". The Host sends provider-specific wrapping (`llm-deepseek: no API key for
// provider route ...`), so the match stays on the stable fragment.
var apiKeyFailureMarkers = []string{
	"no api key",
	"missing api key",
	"api key is not",
	"api key 未配置",
	"未配置 api key",
	"apikey",
	"api_key",
}

var authFailureMarkers = []string{"invalid api key", "unauthorized", "authentication", " 401", "status 401"}
var quotaFailureMarkers = []string{"quota", "insufficient balance", "余额不足", "额度"}

// classifyTurnFailure maps the Host's failure text onto the card the user sees.
// Unrecognized failures keep the generic headline and no action, and always carry
// the raw text so nothing is lost.
func classifyTurnFailure(body, code string) turnFailure {
	raw := strings.TrimSpace(body)
	failure := turnFailure{Raw: raw, Code: strings.TrimSpace(code)}
	lower := strings.ToLower(raw)
	upperCode := strings.ToUpper(strings.TrimSpace(code))

	switch {
	case strings.Contains(upperCode, "INTERRUPTED") || strings.Contains(lower, "interrupted"):
		failure.Kind = turnFailureInterrupted
	case containsAny(lower, authFailureMarkers) || upperCode == "AUTH":
		failure.Kind = turnFailureAuth
	case containsAny(lower, apiKeyFailureMarkers):
		failure.Kind = turnFailureAPIKey
	case containsAny(lower, quotaFailureMarkers) || upperCode == "QUOTA":
		failure.Kind = turnFailureQuota
	default:
		failure.Kind = turnFailureGeneric
	}
	switch failure.Kind {
	case turnFailureInterrupted:
		failure.Title = i18n.T(i18n.KeyChatFailureInterruptedTitle)
		failure.Body = i18n.T(i18n.KeyChatFailureInterruptedBody)
	case turnFailureAuth:
		failure.Title = i18n.T(i18n.KeyChatFailureAuthTitle)
		failure.Body = i18n.T(i18n.KeyChatFailureAuthBody)
		failure.Action = i18n.T(i18n.KeyChatOpenModelSettings)
	case turnFailureAPIKey:
		failure.Title = i18n.T(i18n.KeyChatFailureAPIKeyTitle)
		failure.Body = i18n.T(i18n.KeyChatFailureAPIKeyBody)
		failure.Action = i18n.T(i18n.KeyChatOpenModelSettings)
	case turnFailureQuota:
		failure.Title = i18n.T(i18n.KeyChatFailureQuotaTitle)
		failure.Body = i18n.T(i18n.KeyChatFailureQuotaBody)
	default:
		failure.Title = i18n.T(i18n.KeyChatRunFailed)
	}
	return failure
}

func containsAny(text string, markers []string) bool {
	for _, marker := range markers {
		if strings.Contains(text, marker) {
			return true
		}
	}
	return false
}

// splitTurnFailure separates the Host's `... MISSING_X` code suffix from the
// human text. Codes are shown in the details, never in the headline.
func splitTurnFailure(body string) (text, code string) {
	if index := strings.LastIndex(body, " MISSING_"); index >= 0 {
		return strings.TrimSpace(body[:index]), strings.TrimSpace(body[index+1:])
	}
	return strings.TrimSpace(body), ""
}

// paintTurnFailure paints a failed turn: localized headline and next step, the
// raw Host error only behind the details toggle.
func paintTurnFailure(c *ui.Context, key string, failure turnFailure, palette theme.Colors, fg, muted ui.Color, onOpenModelSettings func()) {
	fail := ui.RGB(220, 38, 38)
	seat := ui.Column(c.Key("turn-failure:"+key)).FillWidth().MinWidth(0).Gap(4)
	details := ui.Local(seat, "turn-failure-details", func() bool { return false })
	hasDetails := failure.Raw != "" && failure.Raw != failure.Title
	seat.Children(func() {
		ui.Row(c).FillWidth().MinWidth(0).Gap(8).AlignItems(ui.Start).Children(func() {
			ui.Text(c, "●").FontSize(12).TextColor(fail)
			ui.Text(c, failure.Title).FontSize(13).FontWeight(700).TextColor(fail).Grow(1).MinWidth(0)
		})
		if failure.Body != "" {
			ui.Text(c, failure.Body).FontSize(12).TextColor(fg).MaxLines(3)
		}
		ui.Row(c).Gap(8).Wrap().AlignItems(ui.Center).Children(func() {
			if failure.Action != "" && onOpenModelSettings != nil {
				if stdButton(c, palette, failure.Action, buttonPrimary, false).Clicked() {
					onOpenModelSettings()
				}
			}
			if hasDetails {
				label := i18n.T(i18n.KeyChatFailureDetails)
				if *details {
					label = i18n.T(i18n.KeyChatFailureDetailsHide)
				}
				if stdButton(c, palette, label, buttonGhost, false).Clicked() {
					*details = !*details
				}
			}
			ui.Box(c).Grow(1)
			if failure.Code != "" {
				ui.Text(c, failure.Code).FontSize(11).FontWeight(600).TextColor(muted)
			}
		})
		if *details && hasDetails {
			ui.Scroll(c).FillWidth().MinWidth(0).MaxHeight(160).Children(func() {
				ui.ScrollHorizontal(c).FillWidth().MinWidth(0).Children(func() {
					ui.Text(c, failure.Raw).FontSize(11).Font(theme.RenderPreferences().Code.Family).
						FixedLineHeight(16).TextColor(muted).NoWrap().Selectable()
				})
			})
		}
	})
}
