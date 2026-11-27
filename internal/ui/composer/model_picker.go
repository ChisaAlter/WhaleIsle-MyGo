package composer

import (
	"github.com/egoist/mygo/ui"
	"whaleisle/internal/ui/coss"
	"whaleisle/internal/ui/design"
	"whaleisle/internal/ui/i18n"
)

func paintModelPicker(c *ui.Context, m Model, height float32) {
	// The model-selection plugin owns this entire seat, independently of its
	// optional label decorator. Omitting the subtree also retires local menus;
	// the Host's model projection remains available to the session store.
	if !m.Snapshot.NativeContributions.Known || !m.Snapshot.NativeContributions.ModelCommand {
		return
	}
	t := design.For(c)
	muted := t.MutedForeground
	row := ui.Row(c).MinWidth(0).Grow(1).Gap(design.Sp(1.5)).AlignItems(ui.Center)
	model := ui.Local(row, "model-choice", func() modelPickerChoice { return modelPickerChoice{} })
	effort := ui.Local(row, "effort-choice", func() modelPickerChoice { return modelPickerChoice{} })
	model.sync(m.Snapshot.SelectedID, m.Snapshot.SelectedModel)
	effort.sync(m.Snapshot.SelectedID, m.Snapshot.SelectedModelEffort)
	row.Children(func() {
		if len(m.Snapshot.ModelLabels) > 0 {
			paintGroupedModelCatalog(c, m, height)
		} else {
			// No catalog: keep the seat a control. The chip names the state and
			// opens a panel with the reload action instead of degrading to a dead
			// label the user cannot click.
			paintModelSeat(c, m)
		}
		if m.Snapshot.ModelsErr != nil {
			ui.Text(c, m.Snapshot.ModelsErr.SentenceText()).FontSize(design.TextXS.Size).TextColor(muted)
			loading := m.Snapshot.ModelSelectionBusy || m.Snapshot.ModelCatalogLoading
			if m.OnRetryModels != nil {
				reloadLabel := i18n.T(i18n.KeyComposerModelReload)
				if loading {
					reloadLabel = ModelSelectionLocaleText("trigger.loading")
				}
				btn := coss.Button(c, reloadLabel, coss.ButtonOptions{
					Variant:  coss.ButtonGhost,
					Size:     coss.ButtonXS,
					Disabled: loading,
				})
				if btn.Clicked() && !loading {
					m.OnRetryModels()
				}
			}
		}
	})
}

// paintModelSeat is the catalog-less model chip: "Loading models…" while a
// selection or catalog refresh is in flight, "Choose model" before any
// selection, and the router's resolved model name when a selection exists
// without a matching catalog row.
func paintModelSeat(c *ui.Context, m Model) {
	t := design.For(c)
	muted := t.MutedForeground
	owner := ui.Box(c)
	open := ui.Local(owner, "model-seat-open", func() bool { return false })
	loading := m.Snapshot.ModelSelectionBusy || m.Snapshot.ModelCatalogLoading
	unconfigured := len(m.Snapshot.ModelLabels) == 0 && m.Snapshot.ModelsErr == nil && !loading
	label := i18n.T(i18n.KeyComposerModelChoose)
	switch {
	case loading:
		label = i18n.T(i18n.KeyComposerModelLoading)
	case m.Snapshot.SelectedModel != "":
		label = friendlyModelTriggerName(m.Snapshot, m.Snapshot.SelectedModel)
	}
	secondary := t.SecondaryForeground
	effortDisplay := formatEffortDisplay(m.Snapshot.SelectedModelEffort)
	tooltip := label
	if effortDisplay != "" && !loading {
		tooltip += " · " + effortDisplay
	}
	trigger := chip(c, *open, secondary).Label(label).Tooltip(tooltip + "  ·  " + i18n.T(i18n.KeyComposerShortcutEsc)).MaxWidth(260).MaxWidthPercent(100).Disabled(m.Inert)
	trigger.Children(func() {
		modelFlipText(c, "seat-name:"+modelSelectBinding(m.Snapshot), label, secondary, design.TextSM.Size).Grow(1)
		if effortDisplay != "" && !loading {
			binding := modelSelectBinding(m.Snapshot)
			modelFlipText(c, "retained-effort:"+binding, effortDisplay, muted, design.TextSM.Size).Shrink(1000)
		}
		chevron(c, *open)
	})
	if trigger.Clicked() {
		*open = !*open
	}
	if !*open {
		return
	}
	ui.PopoverBase(c, trigger, open, func(panel ui.Element) {
		panel = menuSurface(c, panel.MinWidth(260).MaxWidth(420).Padding(design.Sp(2)).Gap(design.Sp(2)).
			AttachTo(trigger, ui.AnchorTopRight, ui.AnchorBottomRight).Margin(0, 0, design.Sp(2), 0))
		panel.Children(func() {
			if m.Snapshot.SelectedModel != "" && !loading {
				header := composerText("当前模型", "Current model")
				if unconfigured {
					header = composerText("默认模型（未配置）", "Default (not configured)")
				}
				ui.Column(c).FillWidth().Gap(design.Sp(0.5)).Children(func() {
					ui.Text(c, header).FontSize(design.TextXS.Size).FontWeight(design.WeightMedium).TextColor(muted)
					ui.Row(c).FillWidth().MinWidth(0).Gap(design.Sp(1.5)).AlignItems(ui.Center).Children(func() {
						ui.Text(c, label).Grow(1).MinWidth(0).SingleLine().Ellipsis("…").FontSize(design.TextSM.Size).TextColor(t.PopoverForeground)
						if effortDisplay != "" {
							ui.Text(c, effortDisplay).Shrink(0).SingleLine().FontSize(design.TextXS.Size).TextColor(muted)
						}
						if !unconfigured {
							cossIcon(c, coss.IconCheck, design.IconXS, t.PopoverForeground)
						}
					})
				})
				coss.Separator(c, coss.Horizontal)
			}
			statusText := i18n.T(i18n.KeyComposerModelUnavailable)
			if loading {
				statusText = ModelSelectionLocaleText("status.loading")
			}
			ui.Text(c, statusText).FontSize(design.TextXS.Size).TextColor(muted)
			if unconfigured {
				ui.Text(c, composerText("尚未登录 DeepSeek 账号或配置模型 API Key", "Not signed in to a DeepSeek account or missing a model API key.")).
					FontSize(design.TextXS.Size).MaxLines(3).TextColor(muted)
			}
			if m.Snapshot.ModelsErr != nil {
				ui.Text(c, m.Snapshot.ModelsErr.SentenceText()).FontSize(design.TextXS.Size).MaxLines(3).TextColor(muted)
			}
			showSettings := unconfigured && m.OnOpenModelSettings != nil
			if showSettings || m.OnRetryModels != nil {
				ui.Row(c).FillWidth().Gap(design.Sp(1.5)).AlignItems(ui.Center).Wrap().Children(func() {
					if showSettings {
						if coss.Button(c, composerText("前往账号/模型设置", "Account & Model Settings"), coss.ButtonOptions{
							Variant: coss.ButtonSecondary,
							Size:    coss.ButtonSM,
						}).Clicked() {
							*open = false
							m.OnOpenModelSettings()
						}
					}
					if m.OnRetryModels != nil {
						reloadLabel := i18n.T(i18n.KeyComposerModelReload)
						if loading {
							reloadLabel = ModelSelectionLocaleText("trigger.loading")
						}
						btn := coss.Button(c, reloadLabel, coss.ButtonOptions{
							Variant:  coss.ButtonGhost,
							Size:     coss.ButtonSM,
							Disabled: loading,
						})
						if btn.Clicked() && !loading {
							m.OnRetryModels()
						}
					}
				})
			}
		})
	})
}
