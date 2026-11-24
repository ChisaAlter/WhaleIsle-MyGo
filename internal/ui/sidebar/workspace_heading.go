package sidebar

import (
	"github.com/egoist/mygo/ui"
	"whaleisle/internal/store/session"
	"whaleisle/internal/ui/coss"
	"whaleisle/internal/ui/design"
	"whaleisle/internal/ui/i18n"
	"whaleisle/internal/ui/theme"
)

// WorkspaceBrowser's section header contains search and selection in the same
// 36 DIP line. Expanding search takes the label and trailing actions' room.
func workspaceHeading(c *ui.Context, m Model, menuOpen *bool, fg, muted ui.Color) {
	t := tok(c)
	header := ui.Row(c).FillWidth().Height(36).Gap(design.Sp(1)).Padding(0, 0, 0, design.Sp(1)).AlignItems(ui.Center)
	state := m.searchState
	if state == nil {
		state = ui.Local(header, "session-search", func() sessionSearchState {
			return sessionSearchState{Expanded: m.SearchQuery != nil && session.NormalizeSessionSearchQuery(*m.SearchQuery) != ""}
		})
	}
	expanded, pendingFocus := &state.Expanded, &state.Focus
	focus := m.SearchFocus != nil && *m.SearchFocus || *pendingFocus
	if focus {
		*expanded = true
	}
	header.Children(func() {
		if *expanded && m.SearchQuery != nil {
			searchRoot := ui.Row(c.Key("workspace-search-root")).Grow(1).MinWidth(0).Height(30).Gap(design.Sp(1)).AlignItems(ui.Center)
			searchRoot.Children(func() {
				input := coss.Input(c, m.SearchQuery, coss.InputOptions{
					Size:        coss.InputSM,
					Placeholder: i18n.T(i18n.KeySidebarSearch),
					Label:       i18n.T(i18n.KeySidebarSearch),
					Key:         "workspace-search",
				}).Height(30)
				if input.Changed() {
					*m.SearchQuery = session.SanitizeSessionSearchInput(*m.SearchQuery)
				}
				if focus {
					input.Focus()
					*pendingFocus = false
					if m.SearchFocus != nil {
						*m.SearchFocus = false
					}
				}
				clearBtn := coss.Button(c, "", coss.ButtonOptions{
					Variant: coss.ButtonGhost,
					Size:    coss.ButtonIconXS,
					Icon:    coss.IconX,
					Label:   i18n.T(i18n.KeyWSSearchClear),
					Key:     "workspace-search-clear",
				}).Height(28).BorderWidth(0).TextColor(muted)
				if input.Shortcut(0, ui.KeyEscape) || clearBtn.Clicked() {
					*m.SearchQuery = ""
					*expanded, *pendingFocus = false, false
				}
			})
			// The source outside-click boundary excludes the adjacent selection
			// toggle. Keep a nonempty filter and ignore the opening focus gesture.
			if !focus && session.NormalizeSessionSearchQuery(*m.SearchQuery) == "" && searchRoot.PressedOutside() {
				*expanded = false
			}
		} else {
			labelSize := fs(14)
			if layout := theme.ResolvedSkinLayout(m.ThemeID, m.SystemDark); layout.Active() {
				labelSize = layout.SidebarFont
			}
			sectionLabel := i18n.T(i18n.KeySidebarWorkspaceSection)
			if m.Snapshot.WorkspaceLayout.GroupBy == "flat" {
				sectionLabel = i18n.T(i18n.KeyWSSectionSessions)
			}
			ui.Text(c, sectionLabel).FontSize(labelSize).FontWeight(design.WeightNormal).FixedLineHeight(fs(20)).TextColor(muted).Grow(1).MinWidth(0).SingleLine()
			if m.SearchQuery != nil {
				search := coss.Button(c, "", coss.ButtonOptions{
					Variant: coss.ButtonGhost,
					Size:    coss.ButtonIconSM,
					Label:   i18n.T(i18n.KeySidebarSearch),
					Key:     "workspace-search-open",
					Content: func() {},
				}).Tooltip(i18n.T(i18n.KeySidebarSearch)).Radius(design.RadiusMD).Padding(0).BorderWidth(0).Center()
				searchInk := muted
				if search.Hovered() {
					searchInk = fg
				}
				search.Children(func() { ui.Icon(c, sidebarRailIcons["search"]).Size(design.IconXS, design.IconXS).TextColor(searchInk) })
				if search.Clicked() {
					*expanded = true
					*pendingFocus = true
					*menuOpen = false
				}
			}
		}
		batchToggle(c, m, muted)
		if !*expanded {
			label := i18n.T(i18n.KeyWSViewOptionsLabel)
			options := coss.Button(c, "", coss.ButtonOptions{
				Variant: coss.ButtonGhost,
				Size:    coss.ButtonIconSM,
				Pressed: *menuOpen,
				Label:   label,
				Key:     "workspace-view-options",
				Content: func() {},
			}).Tooltip(label).Radius(design.RadiusMD).Padding(0).BorderWidth(0).Center()
			optionsInk := muted
			if *menuOpen || options.Hovered() {
				optionsInk = fg
			}
			options.Children(func() {
				ui.Icon(c, workspaceViewOptionsArtwork).Size(design.IconSM, design.IconSM).TextColor(optionsInk)
			})
			items, run := workspaceViewOptionsItems(m)
			if chosen := coss.Menu(c, options, menuOpen, items, coss.MenuOptions{
				Side:     coss.SideBottom,
				Align:    coss.AlignEnd,
				MinWidth: 200,
				Label:    label,
			}); chosen != "" {
				run(chosen)
			}
			if m.OnBrowseWorkspace != nil {
				add := coss.Button(c, "", coss.ButtonOptions{
					Variant: coss.ButtonGhost,
					Size:    coss.ButtonIconSM,
					Label:   i18n.T(i18n.KeySidebarBrowseWorkspace),
					Key:     "workspace-browse-add",
					Content: func() {},
				}).Radius(design.RadiusMD).Padding(0).BorderWidth(0).Center()
				addInk := muted
				if add.Hovered() {
					addInk = fg
				}
				add.Children(func() { ui.Icon(c, sidebarRailIcons["add"]).Size(design.IconSM, design.IconSM).TextColor(addInk) })
				if add.Clicked() {
					*menuOpen = false
					m.OnBrowseWorkspace()
				}
			}
			_ = t
		}
	})
}
