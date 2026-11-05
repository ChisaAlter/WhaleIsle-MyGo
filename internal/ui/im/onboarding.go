package im

import (
	"bytes"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"github.com/egoist/mygo/ui"
	_ "golang.org/x/image/webp"
	"image"
	_ "image/png"
	"strings"
	"time"
	"whaleisle/internal/store/imonboard"
	"whaleisle/internal/ui/chrome"
	"whaleisle/internal/ui/i18n"
	"whaleisle/internal/ui/theme"
)

func onboardingView(c *ui.Context, m Model) {
	s := &m.Store.Onboarding
	v := s.Snapshot()
	copy := copyForChannel(v.Channel)
	colors := theme.Resolve(m.ThemeID).Active(m.SystemDark)
	fg, muted := chrome.Color(colors.FG), chrome.Color(colors.Muted)
	root := ui.Column(c.Key(fmt.Sprintf("im-setup-%d", v.Scope))).FillWidth().MinWidth(0).Gap(12)
	page := ui.Local(root, "channel-page", func() channelPageState { return channelPageState{} })
	if v.Loaded && !page.loaded {
		page.loaded = true
		page.credentials = !imonboard.PlatformByID(v.Channel).QR && len(v.Bots) == 0
	}
	if page.submitted && !v.Busy && v.Draft.Secret == "" {
		page.credentials, page.submitted = false, false
	}
	blocked := v.Busy || !v.Available || m.PluginDisabled || !m.HarnessReady
	root.Children(func() {
		if !m.Embedded {
			ui.Row(c).FillWidth().Wrap().Gap(8).Children(func() {
				ui.Text(c, copy.name).Bold().Grow(1)
				for _, platform := range imonboard.Platforms {
					if ui.Button(c, platform.Label).Disabled(v.Busy || platform.ID == v.Channel).Clicked() {
						s.Show(platform.ID)
					}
				}
				if ui.Button(c, channelText("返回 IM", "Back to IM")).Clicked() {
					s.Hide()
				}
			})
		}
		body := func() {
			ui.Column(c).FillWidth().MinWidth(0).Gap(12).Children(func() {
				channelHeading(c, m, v, page, blocked, fg, muted)
				if v.Error != "" && v.Review == "" {
					channelFeedback(c, v.Error, true)
				}
				if v.StatusError != "" {
					channelFeedback(c, v.StatusError, true)
				}
				if v.TestNotice != "" {
					channelFeedback(c, v.TestNotice, false)
				}
				if !v.Available || !m.HarnessReady {
					ui.Text(c, channelText("连接 Host 后即可配置消息渠道。", "Connect to a Host to configure channels.")).TextColor(muted)
				}
				if v.Provision.AttemptID != "" && !targetedProvision(v.Provision) {
					channelProvisionView(c, m, v, blocked, fg, muted)
				}
				if page.credentials {
					channelCredentials(c, m, v, page, blocked, fg, muted)
				}
				if v.Loaded && len(v.Bots) == 0 && v.Provision.AttemptID == "" && !page.credentials {
					channelEmptyView(c, m, v, page, blocked, fg, muted)
				}
				if len(v.Bots) > 0 {
					ui.Row(c).FillWidth().Wrap().Gap(8).Children(func() {
						ui.Text(c, channelText("已接入的 ", "Connected ")+copy.name+channelText(" 机器人", " bots")).FontSize(theme.InterfaceSize(14)).Bold().Grow(1)
						ui.Text(c, copy.transport).FontSize(theme.InterfaceSize(12)).TextColor(muted)
					})
					for _, bot := range v.Bots {
						channelBotCard(c.Key(bot.BotID), m, v, bot, blocked, fg, muted)
					}
				}
			})
		}
		if m.Embedded {
			body()
		} else {
			ui.Scroll(c).Grow(1).MinHeight(0).Children(body)
		}
		channelReviewDialog(c, m, v, fg, muted)
	})
}

func channelStatusTime(raw json.RawMessage) string {
	if len(raw) == 0 || string(raw) == "null" {
		return ""
	}
	var milliseconds float64
	if json.Unmarshal(raw, &milliseconds) == nil {
		if milliseconds < -8640000000000000 || milliseconds > 8640000000000000 {
			return ""
		}
		return time.UnixMilli(int64(milliseconds)).Local().Format("2006-01-02 15:04:05")
	}
	var text string
	if json.Unmarshal(raw, &text) == nil {
		if parsed, err := time.Parse(time.RFC3339Nano, text); err == nil {
			return parsed.Local().Format("2006-01-02 15:04:05")
		}
	}
	return ""
}

func channelHealthDetails(c *ui.Context, b imonboard.Bot, fg, muted, accent ui.Color) {
	if checked := channelStatusTime(b.Health.LastCheckedAt); checked != "" {
		ui.Text(c, i18n.T(i18n.KeyMiscImLastChecked)+checked).FontSize(theme.InterfaceSize(11)).TextColor(muted)
	}
	if message := strings.TrimSpace(b.Error.Message); message != "" {
		ui.Text(c, i18n.T(i18n.KeyMiscImConnectionError)+message).TextColor(accent).Selectable()
		if code := strings.TrimSpace(b.Error.Code); code != "" {
			ui.Text(c, i18n.T(i18n.KeyMiscImCode)+code).FontSize(theme.InterfaceSize(11)).TextColor(muted).Selectable()
		}
	}
	e := b.LastMessageError
	// Match the original presentation contract: incomplete diagnostics are not
	// rendered as a confirmed failure or fabricated reference/time.
	if strings.TrimSpace(e.Code) == "" || strings.TrimSpace(e.Reason) == "" || strings.TrimSpace(e.Message) == "" || strings.TrimSpace(e.ReferenceID) == "" || e.At == nil {
		return
	}
	ui.Text(c, i18n.T(i18n.KeyMiscImLastMessageFailed)).Bold().TextColor(accent)
	ui.Text(c, strings.TrimSpace(e.Message)).TextColor(fg).Selectable()
	ui.Text(c, fmt.Sprintf(i18n.T(i18n.KeyMiscIMCodeReference), e.Code, e.ReferenceID)).FontSize(theme.InterfaceSize(11)).TextColor(muted).Selectable()
	raw, _ := json.Marshal(*e.At)
	if occurred := channelStatusTime(raw); occurred != "" {
		ui.Text(c, i18n.T(i18n.KeyMiscImOccurred)+occurred).FontSize(theme.InterfaceSize(11)).TextColor(muted)
	}
}
func renderPairingQR(c *ui.Context, source string) {
	renderPairingQRSize(c, source, 260)
}

func renderPairingQRSize(c *ui.Context, source string, size float32) {
	header, body, ok := strings.Cut(source, ",")
	if !ok || !strings.HasPrefix(header, "data:image/") || !strings.Contains(header, ";base64") {
		return
	}
	data, err := base64.StdEncoding.DecodeString(body)
	if err != nil {
		return
	}
	if strings.Contains(header, "svg+xml") {
		if svg, err := ui.ParseSVG(data); err == nil {
			ui.Image(c, svg).Size(size, size).Fit(ui.Contain)
		}
		return
	}
	if img, _, err := image.Decode(bytes.NewReader(data)); err == nil {
		ui.Image(c, ui.NewBitmap(img)).Size(size, size).Fit(ui.Contain)
	}
}
