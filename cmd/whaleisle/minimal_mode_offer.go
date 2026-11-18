package main

import (
	"github.com/egoist/mygo/ui"

	"whaleisle/internal/ui/chrome"
	"whaleisle/internal/ui/theme"
)

// minimalModeOffer is the startup offer for 极简模式 (minimal mode). It is a card
// on the shared modal surface inside the main window: the window the user is
// working in keeps its place, no second window opens, and the card stays until
// one of its three answers is chosen or it is dismissed.
//
// It is presented at most once per launch, after the startup observation has
// decided the interface stuttered, and only over the conversation surface: an
// offer over settings, the market or a remote page would be aimed at the wrong
// thing, so it waits for the workbench instead.

// openMinimalModeOffer makes the offer the next frame paints.
func (s *mainState) openMinimalModeOffer() {
	s.mu.Lock()
	s.minimalOfferOpen = true
	win := s.win
	s.mu.Unlock()
	if win != nil {
		win.Invalidate()
	}
}

// minimalModeOfferDue reports that the card owns the workbench this frame.
func (s *mainState) minimalModeOfferDue() bool {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.minimalOfferOpen && !s.quitting && s.mainRoute == routeConversation
}

func (s *mainState) closeMinimalModeOffer() {
	s.mu.Lock()
	s.minimalOfferOpen = false
	s.mu.Unlock()
}

// syncMinimalModeOffer paints the offer, and records a dismissal as a deferral:
// choosing an answer records that answer instead.
func (s *mainState) syncMinimalModeOffer(c *ui.Context) {
	if c == nil || !s.minimalModeOfferDue() {
		return
	}
	open := true
	answered := false
	ui.DialogBaseWithOptions(c, &open, ui.DialogOptions{}, func(backdrop, panel ui.Element) {
		vw, _ := c.Size()
		chrome.DialogSurfaceTheme(c, backdrop, panel, c.Theme(), min(float32(440), max(float32(0), vw-48)), 0).Children(func() {
			body := ui.Column(c).FillWidth().MinWidth(0).Gap(12).Padding(20, 20, 16, 20)
			failure := ui.Local(body, "error", func() string { return "" })
			body.Children(func() {
				ui.Text(c, minimalModeText("启动时界面有点卡", "Startup felt slow")).FillWidth().
					FontSize(theme.InterfaceSize(16)).FixedLineHeight(theme.InterfaceSize(24)).FontWeight(600).TextColor(c.Theme().Text)
				ui.Text(c, minimalModeText(
					"极简模式会暂停壁纸、动态背景、透明模糊、指针与输入装饰、输入框流光和界面过渡，让界面更轻快。你的外观偏好不会被改动，随时可以在设置 → 外观关闭。",
					"Minimal mode pauses wallpapers, dynamic backgrounds, transparent and blurred materials, pointer and typing decorations, the composer beam and interface transitions to keep the interface light. Your appearance preferences are not changed, and you can turn it off in Settings → Appearance.")).
					FillWidth().FontSize(theme.InterfaceSize(13)).FixedLineHeight(theme.InterfaceSize(20)).TextColor(c.Theme().TextMuted)
				ui.Row(c).FillWidth().MinWidth(0).Wrap().Gap(8).Justify(ui.End).Padding(4, 0, 0, 0).Children(func() {
					ghost := ui.Button(c, minimalModeText("不再提醒", "Don't ask again")).Height(32).Padding(0, 12).Radius(10).BorderWidth(0).Background(ui.Transparent).FontSize(theme.InterfaceSize(13)).TextColor(c.Theme().TextMuted)
					if ghost.Hovered() {
						ghost.Background(c.Theme().SurfaceHover)
					}
					if ghost.Clicked() {
						answered = true
						s.stopMinimalModeOffers()
						open = false
					}
					if ui.Button(c, minimalModeText("暂不开启", "Not now")).Height(32).Padding(0, 14).Radius(10).Border(0.5, c.Theme().Border).Background(ui.Transparent).FontSize(theme.InterfaceSize(13)).TextColor(c.Theme().Text).Clicked() {
						answered = true
						s.snoozeMinimalModeOffer()
						open = false
					}
					if ui.Button(c, minimalModeText("开启极简模式", "Turn on minimal mode")).Height(32).Padding(0, 14).Radius(10).BorderWidth(0).Background(c.Theme().Accent).TextColor(c.Theme().AccentText).FontSize(theme.InterfaceSize(13)).Clicked() {
						if err := s.setMinimalMode(true); err != nil {
							// A failed write leaves the offer in place with the reason, so
							// the user is not told the switch moved when it did not.
							*failure = err.Error()
						} else {
							answered = true
							open = false
						}
					}
				})
				if *failure != "" {
					ui.Text(c, *failure).FillWidth().FontSize(theme.InterfaceSize(12)).TextColor(c.Theme().TextMuted)
				}
			})
		})
	})
	if open {
		return
	}
	s.closeMinimalModeOffer()
	if answered {
		return
	}
	// The card was dismissed (its mask, Escape, or the window losing the
	// surface): that is "not now", not "never".
	s.snoozeMinimalModeOffer()
}
