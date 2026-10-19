package dirpick

import (
	"strings"

	"github.com/egoist/mygo/ui"

	"whaleisle/internal/directorypicker"
	"whaleisle/internal/dsh/api"
	"whaleisle/internal/ui/chrome"
	"whaleisle/internal/ui/i18n"
	"whaleisle/internal/ui/theme"
)

// Model drives the in-app directoryPicker/list browser (remote / host).
type Model struct {
	Listing    api.DirectoryListing
	ThemeID    string
	SystemDark bool
	Loading    bool
	Busy       bool
	NewName    *string
	Err        string
	OnNavigate func(path string)
	OnPick     func(path string)
	OnCreate   func(parent, name string)
	OnClose    func()
	OnRetry    func()
}

// View paints crumbs + entries + optional create row.
func View(c *ui.Context, m Model) {
	th := theme.Resolve(m.ThemeID)
	colors := th.Active(m.SystemDark)
	bg := chrome.Color(colors.Field)
	fg := chrome.Color(colors.FG)
	muted := chrome.Color(colors.Muted)
	border := chrome.Color(colors.Border)
	accent := chrome.Color(colors.Accent)
	inert := m.Loading || m.Busy

	// The owner bounds the dialog; only the directory body consumes spare height.
	ui.Column(c).Fill().MinWidth(0).MinHeight(0).Background(bg).Radius(28).Padding(16, 16, 16, 24).Gap(8).Children(func() {
		ui.Row(c).FillWidth().MinWidth(0).Shrink(0).Gap(8).AlignItems(ui.Center).Children(func() {
			ui.Text(c, i18n.T(i18n.KeyDirPickTitle)).FontSize(theme.InterfaceSize(16)).FontWeight(500).TextColor(fg).Grow(1).MinWidth(0).SingleLine()
			if m.Listing.Home != "" && m.OnNavigate != nil {
				if ui.Button(c, dirText("主目录", "Home")).Height(28).Padding(2, 8).FontSize(theme.InterfaceSize(13)).BorderWidth(0).Background(ui.Transparent).TextColor(muted).Disabled(inert).Clicked() {
					m.OnNavigate(m.Listing.Home)
				}
			}
			if m.OnClose != nil {
				if ui.Button(c, i18n.T(i18n.KeyDirPickClose)).FontSize(theme.InterfaceSize(11)).Clicked() {
					m.OnClose()
				}
			}
		})
		if m.Err != "" {
			ui.Row(c).FillWidth().MinWidth(0).Shrink(0).Gap(8).AlignItems(ui.Center).Children(func() {
				ui.Text(c, m.Err).FontSize(theme.InterfaceSize(12)).TextColor(fg).Grow(1).MinWidth(0)
				if m.OnRetry != nil && ui.Button(c, dirText("重试", "Retry")).FontSize(theme.InterfaceSize(12)).Shrink(0).Disabled(inert).Clicked() {
					m.OnRetry()
				}
			})
		}
		ui.ScrollHorizontal(c).FillWidth().MinWidth(0).Height(28).Shrink(0).Children(func() {
			ui.Row(c).Gap(6).AlignItems(ui.Center).Children(func() {
				for _, crumb := range m.Listing.Crumbs {
					cr := crumb
					label := cr.Name
					if label == "" {
						label = cr.Path
					}
					button := ui.Button(c, "").Label(label).Tooltip(cr.Path).Height(24).MaxWidth(160).Padding(0, 4).BorderWidth(0).Background(ui.Transparent).TextColor(muted).Disabled(inert)
					button.Children(func() { ui.Text(c, label).FontSize(theme.InterfaceSize(13)).TextColor(muted).MinWidth(0).SingleLine() })
					if button.Clicked() {
						if m.OnNavigate != nil {
							m.OnNavigate(cr.Path)
						}
					}
				}
			})
		})
		ui.Scroll(c.Key(m.Listing.Path)).FillWidth().MinWidth(0).MinHeight(0).Grow(1).Gap(2).Children(func() {
			if m.Loading {
				ui.Text(c, dirText("正在读取目录…", "Loading folders…")).FontSize(theme.InterfaceSize(12)).TextColor(muted)
			}
			if len(m.Listing.Entries) == 0 {
				if !m.Loading && m.Err == "" {
					ui.Text(c, i18n.T(i18n.KeyDirPickEmpty)).FontSize(theme.InterfaceSize(12)).TextColor(muted)
				}
				return
			}
			for _, ent := range m.Listing.Entries {
				e := ent
				ui.Row(c.Key(e.Path)).FillWidth().MinWidth(0).Gap(8).AlignItems(ui.Center).Children(func() {
					name := e.Name
					if name == "" {
						name = e.Path
					}
					button := ui.Button(c, "").Label(name).Tooltip(e.Path).Height(28).MinWidth(0).Grow(1).Padding(4).Radius(6).BorderWidth(0).Background(ui.Transparent).Disabled(inert)
					if button.Hovered() || button.FocusWithin() {
						button.Background(chrome.Color(colors.Hover))
					}
					button.Children(func() {
						ui.Row(c).FillWidth().MinWidth(0).Gap(4).AlignItems(ui.Center).Children(func() {
							ui.Box(c).Width(16).Height(16).Draw(func(p *ui.Painter, r ui.Rect) {
								path := &ui.Path{}
								path.MoveTo(r.X+2, r.Y+4)
								path.LineTo(r.X+6, r.Y+4)
								path.LineTo(r.X+8, r.Y+6)
								path.LineTo(r.X+14, r.Y+6)
								path.LineTo(r.X+14, r.Y+13)
								path.LineTo(r.X+2, r.Y+13)
								path.Close()
								p.StrokePath(path, 1.2, muted)
							})
							ui.Text(c, name).FontSize(theme.InterfaceSize(13)).FontWeight(500).TextColor(fg).Grow(1).MinWidth(0).SingleLine()
						})
					})
					if button.Clicked() {
						if m.OnNavigate != nil {
							m.OnNavigate(e.Path)
						}
					}
					if m.OnPick != nil && directorypicker.SelectablePath(e.Path) {
						if ui.Button(c, i18n.T(i18n.KeyDirPickSelect)).FontSize(theme.InterfaceSize(11)).Disabled(inert).Clicked() {
							m.OnPick(e.Path)
						}
					}
				})
			}
		})
		if m.Listing.Truncated {
			ui.Text(c, dirText("目录过多，仅显示部分结果。", "Only part of this folder is shown.")).FontSize(theme.InterfaceSize(12)).TextColor(muted).Shrink(0)
		}
		ui.Box(c).FillWidth().Height(.5).Shrink(0).Background(border)
		if m.NewName != nil && m.OnCreate != nil && directorypicker.SelectablePath(m.Listing.Path) {
			ui.Row(c).FillWidth().MinWidth(0).Shrink(0).Gap(8).AlignItems(ui.Center).Children(func() {
				input := ui.TextInput(c, m.NewName).MinWidth(0).Grow(1).Label(i18n.T(i18n.KeyDirPickNewName)).Placeholder(i18n.T(i18n.KeyDirPickNewName)).Disabled(inert)
				create := ui.Button(c, i18n.T(i18n.KeyDirPickCreate)).FontSize(theme.InterfaceSize(13)).Disabled(inert || strings.TrimSpace(*m.NewName) == "")
				if !inert && (create.Clicked() || input.Submitted()) {
					name := strings.TrimSpace(*m.NewName)
					if name != "" {
						m.OnCreate(m.Listing.Path, name)
						*m.NewName = ""
					}
				}
			})
		}
		if m.OnPick != nil && directorypicker.SelectablePath(m.Listing.Path) {
			if ui.Button(c, i18n.T(i18n.KeyDirPickSelectHere)).FontSize(theme.InterfaceSize(11)).TextColor(accent).Shrink(0).Disabled(inert).Clicked() {
				m.OnPick(m.Listing.Path)
			}
		}
	})
}

func dirText(zh, en string) string {
	if i18n.Locale() == i18n.LocaleEN {
		return en
	}
	return zh
}
