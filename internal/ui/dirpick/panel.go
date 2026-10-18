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
	NewName    *string
	Err        string
	OnNavigate func(path string)
	OnPick     func(path string)
	OnCreate   func(parent, name string)
	OnClose    func()
}

// View paints crumbs + entries + optional create row.
func View(c *ui.Context, m Model) {
	th := theme.Resolve(m.ThemeID)
	colors := th.Active(m.SystemDark)
	bg := chrome.Color(colors.BG)
	fg := chrome.Color(colors.FG)
	muted := chrome.Color(colors.Muted)
	border := chrome.Color(colors.Border)
	field := chrome.Color(colors.Field)
	accent := chrome.Color(colors.Accent)

	// Electron directoryPicker: bordered card radius ~12, list max ~280.
	ui.Column(c).Background(bg).Border(1, border).Radius(12).Padding(10, 12, 10, 12).Gap(8).Children(func() {
		ui.Row(c).Gap(8).AlignItems(ui.Center).Children(func() {
			ui.Text(c, i18n.T(i18n.KeyDirPickTitle)).FontSize(theme.InterfaceSize(14)).Bold().TextColor(fg).Grow(1)
			if m.OnClose != nil {
				if ui.Button(c, i18n.T(i18n.KeyDirPickClose)).FontSize(theme.InterfaceSize(11)).Clicked() {
					m.OnClose()
				}
			}
		})
		if m.Err != "" {
			ui.Text(c, m.Err).FontSize(theme.InterfaceSize(12)).TextColor(fg)
		}
		ui.Row(c).Gap(6).Children(func() {
			for _, crumb := range m.Listing.Crumbs {
				cr := crumb
				label := cr.Name
				if label == "" {
					label = cr.Path
				}
				if ui.Button(c, label).FontSize(theme.InterfaceSize(11)).BorderWidth(0).
					Background(ui.Transparent).TextColor(accent).Clicked() {
					if m.OnNavigate != nil {
						m.OnNavigate(cr.Path)
					}
				}
			}
		})
		ui.Column(c).Background(field).Border(1, border).Radius(8).Padding(6, 8, 6, 8).Gap(4).MaxHeight(280).Children(func() {
			if len(m.Listing.Entries) == 0 {
				ui.Text(c, i18n.T(i18n.KeyDirPickEmpty)).FontSize(theme.InterfaceSize(12)).TextColor(muted)
				return
			}
			for _, ent := range m.Listing.Entries {
				e := ent
				ui.Row(c).Gap(8).AlignItems(ui.Center).Children(func() {
					name := e.Name
					if name == "" {
						name = e.Path
					}
					if ui.Button(c, name).FontSize(theme.InterfaceSize(12)).BorderWidth(0).
						Background(ui.Transparent).TextColor(fg).Grow(1).Clicked() {
						if m.OnNavigate != nil {
							m.OnNavigate(e.Path)
						}
					}
					if m.OnPick != nil && directorypicker.SelectablePath(e.Path) {
						if ui.Button(c, i18n.T(i18n.KeyDirPickSelect)).FontSize(theme.InterfaceSize(11)).Clicked() {
							m.OnPick(e.Path)
						}
					}
				})
			}
		})
		if m.NewName != nil && m.OnCreate != nil && directorypicker.SelectablePath(m.Listing.Path) {
			ui.Row(c).Gap(8).AlignItems(ui.Center).Children(func() {
				ui.TextInput(c, m.NewName).FillWidth().Grow(1).Placeholder(i18n.T(i18n.KeyDirPickNewName))
				if ui.Button(c, i18n.T(i18n.KeyDirPickCreate)).FontSize(theme.InterfaceSize(11)).Clicked() {
					name := strings.TrimSpace(*m.NewName)
					if name != "" {
						m.OnCreate(m.Listing.Path, name)
						*m.NewName = ""
					}
				}
			})
		}
		if m.OnPick != nil && directorypicker.SelectablePath(m.Listing.Path) {
			if ui.Button(c, i18n.T(i18n.KeyDirPickSelectHere)).FontSize(theme.InterfaceSize(11)).TextColor(accent).Clicked() {
				m.OnPick(m.Listing.Path)
			}
		}
	})
}
