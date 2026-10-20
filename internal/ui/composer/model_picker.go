package composer

import (
	"github.com/egoist/mygo/ui"
	"whaleisle/internal/ui/chrome"
	"whaleisle/internal/ui/i18n"
	"whaleisle/internal/ui/theme"
)

func paintModelPicker(c *ui.Context, m Model, colors theme.Colors, height float32) {
	// The model-selection plugin owns this entire seat, independently of its
	// optional label decorator. Omitting the subtree also retires local menus;
	// the Host's model projection remains available to the session store.
	if !m.Snapshot.NativeContributions.Known || !m.Snapshot.NativeContributions.ModelCommand {
		return
	}
	muted := chrome.Color(colors.Muted)
	row := ui.Row(c).MinWidth(0).Grow(1).Gap(6).AlignItems(ui.Center)
	model := ui.Local(row, "model-choice", func() modelPickerChoice { return modelPickerChoice{} })
	effort := ui.Local(row, "effort-choice", func() modelPickerChoice { return modelPickerChoice{} })
	model.sync(m.Snapshot.SelectedID, m.Snapshot.SelectedModel)
	effort.sync(m.Snapshot.SelectedID, m.Snapshot.SelectedModelEffort)
	row.Children(func() {
		if len(m.Snapshot.ModelLabels) > 0 {
			paintGroupedModelCatalog(c, m, colors, height)
		} else {
			// No catalog: keep the seat a control. The chip names the state and
			// opens a panel with the reload action instead of degrading to a dead
			// label the user cannot click.
			paintModelSeat(c, m, colors)
		}
		if len(m.Snapshot.ModelLabels) > 0 {
			// Model and effort share the two-level native menu above.
		} else if len(m.Snapshot.ModelEffortLabels) > 0 && !m.Snapshot.ModelSelectionBusy {
			choice := composerSelect(c, &effort.Value, m.Snapshot.ModelEffortLabels).
				Height(height).FontSize(12).Background(ui.Transparent).BorderWidth(0).TextColor(muted)
			if choice.Changed() {
				value := effort.take()
				if m.OnModelEffortChange != nil {
					m.OnModelEffortChange(value)
				}
			}
		} else if m.Snapshot.SelectedModelEffort != "" {
			binding := modelSelectBinding(m.Snapshot)
			modelFlipText(c, "retained-effort:"+binding, m.Snapshot.SelectedModelEffort, chrome.Color(colors.Caption), 13).Shrink(1000)
		}
		if m.Snapshot.ModelsErr != "" {
			ui.Text(c, m.Snapshot.ModelsErr).FontSize(11).TextColor(muted)
			if m.OnRetryModels != nil && ui.Button(c, i18n.T(i18n.KeyComposerModelReload)).FontSize(11).Clicked() {
				m.OnRetryModels()
			}
		}
	})
}

// paintModelSeat is the catalog-less model chip: "Loading…" while the request
// is in flight, "Choose model" before any selection, and the router's
// provider/model string when a selection exists but its catalog row does not.
func paintModelSeat(c *ui.Context, m Model, colors theme.Colors) {
	muted := chrome.Color(colors.Muted)
	owner := ui.Box(c)
	open := ui.Local(owner, "model-seat-open", func() bool { return false })
	label := i18n.T(i18n.KeyComposerModelChoose)
	switch {
	case m.Snapshot.ModelSelectionBusy:
		label = i18n.T(i18n.KeyComposerModelLoading)
	case m.Snapshot.SelectedModel != "":
		label = m.Snapshot.SelectedModel
	}
	trigger := chip(c, colors, *open, muted).Label(label).Tooltip(label + "  ·  " + i18n.T(i18n.KeyComposerShortcutEsc)).MaxWidth(260).Disabled(m.Inert)
	trigger.Children(func() {
		ui.Text(c, label).Grow(1).MinWidth(0).SingleLine().FontSize(13).TextColor(muted)
		chevron(c, *open)
	})
	if trigger.Clicked() {
		*open = !*open
	}
	if !*open {
		return
	}
	ui.PopoverBase(c, trigger, open, func(panel ui.Element) {
		panel = menuSurface(panel.Width(240).Padding(8).Gap(8), colors)
		panel.Children(func() {
			ui.Text(c, i18n.T(i18n.KeyComposerModelUnavailable)).FontSize(13).TextColor(muted)
			if m.Snapshot.ModelsErr != "" {
				ui.Text(c, m.Snapshot.ModelsErr).FontSize(11).MaxLines(3).TextColor(muted)
			}
			if m.OnRetryModels != nil && ui.Button(c, i18n.T(i18n.KeyComposerModelReload)).FontSize(12).Clicked() {
				m.OnRetryModels()
			}
		})
	})
}
