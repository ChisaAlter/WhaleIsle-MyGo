package settings

import (
	"fmt"
	"strings"

	"github.com/egoist/mygo/ui"
	"whaleisle/internal/desktop/providerdiscovery"
	store "whaleisle/internal/store/settings"
)

type modelProbeState struct {
	sequence       uint64
	busy           bool
	message, query string
	candidates     []providerdiscovery.Candidate
	stop           func()
}

func (p *modelProbeState) cancel() {
	p.sequence++
	p.busy, p.message, p.query, p.candidates = false, "", "", nil
	if p.stop != nil {
		p.stop()
		p.stop = nil
	}
}

func modelListDiscovery(c *ui.Context, m Model, s store.ViewState, scope string, rows []any, change func([]any), fg, muted ui.Color) {
	if s.SelectedNS != "llm-pi-ai" || m.Models == nil || m.Models.ProviderDiscovery == nil || m.Models.ProviderDiscovery.OnProbeEditor == nil || m.Models.configurationProbe == nil {
		return
	}
	a, state := m.Models.ProviderDiscovery, m.Models.configurationProbe
	profile := configurationObjectValue(configurationObject(s.Drafts["providers"])[scope])
	provider := scope
	key := ""
	if m.Models.configurationCreateOnly {
		profile = configurationObject(s.Drafts["@provider-create"])
		provider = ""
		if m.Models.configurationCreateKey != nil {
			key = *m.Models.configurationCreateKey
		}
	} else if credentials := m.ReferencedCredentials; credentials != nil {
		for _, row := range credentials.Snapshot.Rows {
			if row.ID == credentials.Snapshot.SelectedID && row.Namespace == s.SelectedNS && row.Provider == provider {
				key = credentials.Snapshot.Draft
				break
			}
		}
	}
	ui.Row(c.Key("model-fetch/" + scope)).Shrink(0).Gap(8).AlignItems(ui.Center).Children(func() {
		label := providerCopy("获取可用模型", "Fetch available models")
		if state.busy {
			label = providerCopy("正在获取模型…", "Fetching models…")
		}
		if modelLinkPill(c, label, state.busy || !s.ClientReady || s.Saving || !s.Writable).Clicked() {
			state.sequence++
			sequence := state.sequence
			state.busy, state.message, state.candidates = true, "", nil
			known := map[string]bool{}
			for _, raw := range rows {
				known[configurationText(configurationObjectValue(raw)["id"])] = true
			}
			target := providerdiscovery.EditorProbe{Namespace: s.SelectedNS, Provider: provider, Endpoint: configurationText(profile["baseURL"]), Protocol: configurationText(profile["api"]), Key: key}
			state.stop = a.OnProbeEditor(target, func(candidates []providerdiscovery.Candidate, err error) {
				if state.sequence != sequence {
					return
				}
				state.busy, state.stop = false, nil
				if err != nil {
					state.message = providerdiscovery.Message(err.Error())
					return
				}
				for i := range candidates {
					candidates[i].Picked = !known[candidates[i].ID]
				}
				state.candidates = candidates
				if len(candidates) == 0 {
					state.message = providerCopy("未返回模型，仍可手动添加。", "No models returned. You can still add them manually.")
				}
			})
		}
		if state.busy && ui.Button(c, providerCopy("取消获取", "Cancel fetch")).Clicked() {
			state.cancel()
		}
	})
	settingsNotice(c, "model-fetch/"+scope, "", state.message != "", func() string { return state.message })
	if len(state.candidates) == 0 {
		return
	}
	open := true
	accountDialog(c, &open, providerCopy("选择模型", "Choose models"), 520, s.Saving, func() {
		ui.TextInput(c, &state.query).Label(providerCopy("搜索候选模型", "Search candidate models")).Placeholder(providerCopy("搜索候选模型", "Search candidate models")).FillWidth().Height(32)
		query := strings.ToLower(strings.TrimSpace(state.query))
		ui.Row(c).Gap(8).Children(func() {
			matches := func(candidate providerdiscovery.Candidate) bool {
				return query == "" || strings.Contains(strings.ToLower(candidate.ID+" "+candidate.Name), query)
			}
			all := true
			for _, candidate := range state.candidates {
				if matches(candidate) && !candidate.Picked {
					all = false
				}
			}
			label := providerCopy("全选筛选结果", "Select filtered")
			if all {
				label = providerCopy("取消全选", "Deselect filtered")
			}
			if ui.Button(c, label).Disabled(s.Saving).Clicked() {
				for i := range state.candidates {
					if matches(state.candidates[i]) {
						state.candidates[i].Picked = !all
					}
				}
			}
		})
		ui.Scroll(c).FillWidth().MaxHeight(320).MinHeight(0).Shrink(1).Children(func() {
			ui.Column(c).FillWidth().Gap(6).Children(func() {
				for i := range state.candidates {
					candidate := &state.candidates[i]
					if query != "" && !strings.Contains(strings.ToLower(candidate.ID+" "+candidate.Name), query) {
						continue
					}
					label := candidate.Name
					if label != candidate.ID {
						label += " · " + candidate.ID
					}
					if candidate.ContextWindow > 0 {
						label += fmt.Sprintf(" · %d tokens", candidate.ContextWindow)
					}
					ui.Checkbox(c.Key("candidate/"+candidate.ID), &candidate.Picked, label).Disabled(s.Saving)
				}

			})
		})
	}, func() {
		if ui.Button(c, providerCopy("采纳选中模型", "Adopt selected models")).Disabled(s.Saving || !s.Writable).Clicked() {
			models := []map[string]any{}
			for _, raw := range rows {
				if row := configurationObjectValue(raw); row != nil {
					models = append(models, row)
				}
			}
			next := providerdiscovery.MergeEditorCandidates(models, state.candidates)
			values := make([]any, len(next))
			for i, row := range next {
				values[i] = row
			}
			change(values)
			state.candidates, state.query = nil, ""
			open = false
		}
		if ui.Button(c, providerCopy("取消", "Cancel")).Disabled(s.Saving).Clicked() {
			open = false
		}
	})
	if !open {
		state.candidates = nil
		state.query = ""
	}
}
