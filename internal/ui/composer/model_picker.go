package composer

import (
	"github.com/egoist/mygo/ui"
	"whaleisle/internal/ui/i18n"
)

func paintModelPicker(c *ui.Context, m Model, muted ui.Color, height float32) {
	row := ui.Row(c).Gap(6).AlignItems(ui.Center)
	model := ui.Local(row, "model-choice", func() modelPickerChoice { return modelPickerChoice{} })
	effort := ui.Local(row, "effort-choice", func() modelPickerChoice { return modelPickerChoice{} })
	model.sync(m.Snapshot.SelectedID, m.Snapshot.SelectedModel)
	effort.sync(m.Snapshot.SelectedID, m.Snapshot.SelectedModelEffort)
	row.Children(func() {
		if len(m.Snapshot.ModelLabels) > 0 && !m.Snapshot.ModelSelectionBusy {
			paintGroupedModelCatalog(c, m, muted, height)
		} else {
			label := m.Snapshot.SelectedModel
			if label == "" {
				label = i18n.T(i18n.KeyComposerModel)
			}
			if m.Snapshot.ModelSelectionBusy {
				label += " …"
			}
			ui.Text(c, label).FontSize(13).FontWeight(500).TextColor(muted)
		}
		if len(m.Snapshot.ModelEffortLabels) > 0 && !m.Snapshot.ModelSelectionBusy {
			choice := composerSelect(c, &effort.Value, m.Snapshot.ModelEffortLabels).
				Height(height).FontSize(12).Background(ui.Transparent).BorderWidth(0).TextColor(muted)
			if choice.Changed() {
				value := effort.take()
				if m.OnModelEffortChange != nil {
					m.OnModelEffortChange(value)
				}
			}
		} else if m.Snapshot.SelectedModelEffort != "" {
			ui.Text(c, m.Snapshot.SelectedModelEffort).FontSize(12).TextColor(muted)
		}
		if m.Snapshot.ModelsErr != "" {
			ui.Text(c, m.Snapshot.ModelsErr).FontSize(11).TextColor(muted)
			if m.OnRetryModels != nil && ui.Button(c, "刷新").FontSize(11).Clicked() {
				m.OnRetryModels()
			}
		}
	})
}
