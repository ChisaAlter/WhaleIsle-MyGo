package launcher

import (
	"strings"
	fonttheme "whaleisle/internal/ui/theme"

	"github.com/egoist/mygo/ui"
	"whaleisle/internal/ui/chrome"
	"whaleisle/internal/ui/i18n"
)

func routePageHead(c *ui.Context, m Model, title, lede string) {
	ui.Row(c).FillWidth().AlignItems(ui.Center).Gap(16).Padding(0, 0, 16, 0).Children(func() {
		ui.Column(c).Grow(1).Gap(4).Children(func() {
			ui.Text(c, title).FontSize(fonttheme.InterfaceSize(16)).FontWeight(600).TextColor(colPrimary)
			if lede != "" {
				ui.Text(c, lede).FontSize(fonttheme.InterfaceSize(12)).TextColor(colTertiary)
			}
		})
		label := i18n.T(i18n.KeyLauncherRouteCurrentPrefix) + " " + railRouteLabel(m)
		btn := ghostSmall(c.Key("route-"+title), "").Radius(8).Label(label)
		open := ui.Local(btn, "route-open", func() bool { return false })
		btn.Children(func() {
			ui.Row(c).Gap(6).AlignItems(ui.Center).Children(func() {
				ui.Text(c, label).FontSize(fonttheme.InterfaceSize(12)).TextColor(colSecondary)
				routeChevron(c, colTertiary, *open)
			})
		})
		// Local state belongs to this trigger, never the rail-footer pointer.
		routePopup(c, m, btn, open, false)
	})
}

func routePopup(c *ui.Context, m Model, trigger ui.Element, open *bool, upward bool) {
	if open == nil {
		open = ui.Local(trigger, "route-open", func() bool { return false })
	}
	highlight := ui.Local(trigger, "route-highlight", func() int { return 0 })
	current := 0
	if m.DownloadRoute != nil && strings.EqualFold(strings.TrimSpace(*m.DownloadRoute), "cnb") {
		current = 1
	}
	pick := func(index int) {
		if m.RouteUnavailable[[]string{"github", "cnb"}[index]] != "" {
			return
		}
		if m.DownloadRoute != nil {
			*m.DownloadRoute = []string{"github", "cnb"}[index]
			if m.OnSaveLauncherPrefs != nil {
				m.OnSaveLauncherPrefs()
			}
		}
		*open = false
	}
	// Consume the trigger's keys before Button interprets Enter as a toggle.
	if *open && (trigger.Shortcut(0, ui.KeyEnter) || trigger.Shortcut(0, ui.KeySpace)) {
		pick(*highlight)
	} else if trigger.Clicked() {
		*open = !*open
		*highlight = current
	} else if !*open && (trigger.Shortcut(0, ui.KeyDown) || trigger.Shortcut(0, ui.KeyUp)) {
		*open = true
		*highlight = current
	}
	if m.DownloadRoute == nil {
		*open = false
		return
	}
	// Current launcher.css and its shared semantic token table.
	surface, ink := chrome.Color("rgb(255, 255, 255)"), chrome.Color("rgb(15, 17, 21)")
	muted, accent := chrome.Color("rgb(129, 133, 140)"), chrome.Color("rgb(65, 118, 230)")
	hover, border := chrome.Color("rgba(38, 49, 72, 0.06)"), chrome.Color("rgba(0, 0, 0, 0.04)")
	if m.SystemDark {
		surface, ink = chrome.Color("rgb(44, 44, 46)"), chrome.Color("rgb(249, 250, 251)")
		muted, accent = chrome.Color("rgb(173, 178, 184)"), chrome.Color("rgb(103, 158, 254)")
		hover, border = chrome.Color("rgba(255, 255, 255, 0.08)"), chrome.Color("rgba(255, 255, 255, 0.06)")
	}
	ui.PopoverBase(c, trigger, open, func(panel ui.Element) {
		panel = chrome.PopoverReveal(c, panel.MinWidth(240).Background(surface).Border(1, border).Radius(10).
			Padding(6).Margin(6, 0, 0, 0).Shadow(0, 10, 32, 0, ink.Alpha(0.22)))
		if upward {
			panel.AttachTo(trigger, ui.AnchorTopLeft, ui.AnchorBottomLeft).Margin(0, 0, 6, 0)
		} else {
			panel.AttachTo(trigger, ui.AnchorBottomRight, ui.AnchorTopRight)
		}
		options := make([]ui.Element, 0, 2)
		routes := []struct{ title, detail string }{
			{i18n.T(i18n.KeyLauncherRouteGitHub), i18n.T(i18n.KeyLauncherRouteGitHubDetail)},
			{i18n.T(i18n.KeyLauncherRouteCNB), i18n.T(i18n.KeyLauncherRouteCNBDetail)},
		}
		for index, route := range routes {
			reason := m.RouteUnavailable[[]string{"github", "cnb"}[index]]
			if reason != "" {
				route.detail = reason
			}
			opt := ui.Button(c, "").Disabled(reason != "").FillWidth().Radius(8).AlignItems(ui.Start).Justify(ui.Start).
				BorderWidth(0).TextColor(ink).Padding(9, 10).Gap(10)
			opt.Children(func() {
				check := ""
				if index == current {
					check = "✓"
				}
				ui.Text(c, check).Width(14).Shrink(0).Padding(2, 0, 0, 0).
					FontSize(12).FontWeight(700).TextColor(accent)
				ui.Column(c).Grow(1).MinWidth(0).Gap(2).Children(func() {
					ui.Text(c, route.title).FontSize(fonttheme.InterfaceSize(13)).FontWeight(600).TextColor(ink).SingleLine()
					ui.Text(c, route.detail).FontSize(fonttheme.InterfaceSize(12)).LineHeight(1.4).TextColor(muted)
				})
			})
			options = append(options, opt)
			if opt.Focused() {
				*highlight = index
			}
		}
		key := func(k ui.Key) bool { return trigger.Shortcut(0, k) || panel.OverlayShortcut(0, k) }
		moved := true
		switch {
		case key(ui.KeyDown):
			*highlight = min(1, *highlight+1)
		case key(ui.KeyUp):
			*highlight = max(0, *highlight-1)
		case key(ui.KeyHome):
			*highlight = 0
		case key(ui.KeyEnd):
			*highlight = 1
		default:
			moved = false
		}
		if moved {
			options[*highlight].Focus()
		}
		for index, opt := range options {
			bg := ui.Transparent
			if opt.Hovered() || opt.FocusVisible() {
				bg = hover
			}
			opt.Background(bg)
			if opt.Clicked() {
				pick(index)
			}
		}
	})
}
