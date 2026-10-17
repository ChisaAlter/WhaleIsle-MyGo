package settings

import (
	"strings"

	"github.com/egoist/mygo/ui"

	"whaleisle/internal/ui/i18n"
)

// SkillRow is one skills/list entry for the dedicated settings page.
type SkillRow struct {
	Name           string
	Description    string
	ModelInvocable bool
}

// SkillHubHit is one skillInventory/searchHub result row.
type SkillHubHit struct {
	SkillName   string
	Repo        string
	Path        string
	Description string
	Stars       float64
	Status      string
}

// SkillsAccess is the dedicated Skills settings page (session skills/list snapshot).
type SkillsAccess struct {
	Skills     []SkillRow
	SessionID  string
	Err        string
	LastAction string
	OnRefresh  func()
	// OnInsert inserts `/name ` into the composer (Electron ui-skill slash invoke).
	OnInsert func(name string)
	// OnSetModelInvocable toggles skillInventory/setInvocation modelInvocable.
	OnSetModelInvocable func(name string, modelInvocable bool)
	OnDelete            func(name string)
	OnCreate            func(name string)
	OnInspect           func(name string)
	CreateName          *string
	DetailPreview       string
	// Edit* hold skillInventory/get → skillInventory/update draft (writable skills only).
	EditName        string
	EditDesc        *string
	EditContent     *string
	EditWritable    bool
	OnSaveUpdate    func()

	HubQuery   *string
	HubHits    []SkillHubHit
	OnSearchHub func(query string)
	OnInstallHub func(repo, path, skillName string)
}

func skillsPane(c *ui.Context, m Model, fg, muted ui.Color) {
	if m.Skills == nil {
		return
	}
	a := m.Skills
	ui.Text(c, i18n.T(i18n.KeySettingsSkillsTitle)).FontSize(14).Bold().TextColor(fg)
	ui.Text(c, i18n.T(i18n.KeySettingsSkillsHint)).FontSize(12).TextColor(muted)
	if a.SessionID != "" {
		ui.Text(c, i18n.T(i18n.KeySettingsSkillsSession)+": "+a.SessionID).FontSize(11).TextColor(muted)
	} else {
		ui.Text(c, i18n.T(i18n.KeySettingsSkillsNoSession)).FontSize(11).TextColor(muted)
	}
	if a.Err != "" {
		ui.Text(c, a.Err).FontSize(12).TextColor(fg)
	}
	if a.LastAction != "" {
		ui.Text(c, a.LastAction).FontSize(11).TextColor(muted)
	}
	if a.DetailPreview != "" {
		ui.Text(c, a.DetailPreview).FontSize(11).TextColor(muted)
	}
	if a.EditWritable && a.EditName != "" && a.EditDesc != nil && a.EditContent != nil && a.OnSaveUpdate != nil {
		ui.Text(c, i18n.T(i18n.KeySettingsSkillsEditTitle)+": "+a.EditName).FontSize(12).Bold().TextColor(fg)
		ui.TextInput(c, a.EditDesc).FillWidth().Placeholder(i18n.T(i18n.KeySettingsSkillsEditDesc))
		ui.TextInput(c, a.EditContent).FillWidth().Placeholder(i18n.T(i18n.KeySettingsSkillsEditContent))
		if ui.Button(c, i18n.T(i18n.KeySettingsSkillsSaveUpdate)).FontSize(11).Clicked() {
			a.OnSaveUpdate()
		}
	}
	if len(a.Skills) == 0 {
		ui.Text(c, i18n.T(i18n.KeySettingsSkillsEmpty)).FontSize(12).TextColor(muted)
	} else {
		for _, sk := range a.Skills {
			row := sk
			flag := i18n.T(i18n.KeySettingsSkillsModelInvocable)
			if !row.ModelInvocable {
				flag = i18n.T(i18n.KeySettingsSkillsUserOnly)
			}
			desktopRow(c, emptyDash(row.Name), func() {
				ui.Text(c, flag).FontSize(11).TextColor(muted)
				if row.Description != "" {
					ui.Text(c, row.Description).FontSize(11).TextColor(muted)
				}
				ui.Row(c).Gap(6).Children(func() {
					if a.OnInsert != nil && row.Name != "" {
						if ui.Button(c, i18n.T(i18n.KeySettingsSkillsInsert)).FontSize(11).Clicked() {
							a.OnInsert(row.Name)
						}
					}
					if a.OnSetModelInvocable != nil && row.Name != "" {
						label := i18n.T(i18n.KeySettingsSkillsMakeUserOnly)
						next := false
						if !row.ModelInvocable {
							label = i18n.T(i18n.KeySettingsSkillsMakeModel)
							next = true
						}
						if ui.Button(c, label).FontSize(11).Clicked() {
							a.OnSetModelInvocable(row.Name, next)
						}
					}
					if a.OnInspect != nil && row.Name != "" {
						if ui.Button(c, i18n.T(i18n.KeySettingsSkillsInspect)).FontSize(11).Clicked() {
							a.OnInspect(row.Name)
						}
					}
					if a.OnDelete != nil && row.Name != "" && !row.ModelInvocable {
						if ui.Button(c, i18n.T(i18n.KeyComposerSkillsDelete)).FontSize(11).Clicked() {
							a.OnDelete(row.Name)
						}
					}
				})
			}, fg, muted)
		}
	}
	if a.CreateName != nil && a.OnCreate != nil {
		ui.Row(c).Gap(8).AlignItems(ui.Center).Children(func() {
			ui.TextInput(c, a.CreateName).Grow(1).Placeholder(i18n.T(i18n.KeyComposerSkillsCreateName))
			if ui.Button(c, i18n.T(i18n.KeyComposerSkillsCreate)).FontSize(11).Clicked() {
				name := strings.TrimSpace(*a.CreateName)
				if name != "" {
					a.OnCreate(name)
					*a.CreateName = ""
				}
			}
		})
	}
	if a.OnRefresh != nil && ui.Button(c, i18n.T(i18n.KeySettingsRefresh)).Clicked() {
		a.OnRefresh()
	}

	if a.HubQuery != nil && a.OnSearchHub != nil {
		ui.Text(c, i18n.T(i18n.KeySettingsSkillsHubTitle)).FontSize(13).Bold().TextColor(fg).Padding(12, 0, 0, 0)
		ui.Text(c, i18n.T(i18n.KeySettingsSkillsHubHint)).FontSize(11).TextColor(muted)
		ui.Row(c).Gap(8).AlignItems(ui.Center).Children(func() {
			ui.TextInput(c, a.HubQuery).Grow(1).Placeholder(i18n.T(i18n.KeySettingsSkillsHubPlaceholder))
			if ui.Button(c, i18n.T(i18n.KeySettingsSkillsHubSearch)).FontSize(11).Clicked() {
				a.OnSearchHub(*a.HubQuery)
			}
		})
		for _, hit := range a.HubHits {
			h := hit
			label := h.SkillName
			if label == "" {
				label = h.Path
			}
			desktopRow(c, emptyDash(label), func() {
				detail := h.Repo
				if h.Description != "" {
					if detail != "" {
						detail += " · "
					}
					detail += h.Description
				}
				if h.Status != "" {
					if detail != "" {
						detail += " · "
					}
					detail += h.Status
				}
				if detail != "" {
					ui.Text(c, detail).FontSize(11).TextColor(muted)
				}
				if a.OnInstallHub != nil && h.Repo != "" && h.Path != "" && h.SkillName != "" {
					if ui.Button(c, i18n.T(i18n.KeySettingsSkillsHubInstall)).FontSize(11).Clicked() {
						a.OnInstallHub(h.Repo, h.Path, h.SkillName)
					}
				}
			}, fg, muted)
		}
	}
}
