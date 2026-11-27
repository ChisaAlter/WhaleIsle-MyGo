package settings

import (
	"fmt"
	"strings"

	"whaleisle/internal/desktop/providerdiscovery"
	"whaleisle/internal/errcopy"
	store "whaleisle/internal/store/settings"
	"whaleisle/internal/ui/coss"
	"whaleisle/internal/ui/design"

	"github.com/egoist/mygo/ui"
)

type modelProbeState struct {
	sequence       uint64
	busy           bool
	message, query string
	lastQuery      string
	candidates     []providerdiscovery.Candidate
	scroll         ui.ScrollState
	stop           func()
}

// discoveryMessage translates the provider controller's own sentences. An
// error the table does not know would otherwise surface as raw English, so it
// gets the shell's cause-and-next-step copy instead.
func discoveryMessage(err error) string {
	if err == nil {
		return ""
	}
	raw := err.Error()
	if message := providerdiscovery.Message(raw); message != raw {
		return message
	}
	return errcopy.Sentence(err)
}

func (p *modelProbeState) cancel() {
	p.sequence++
	p.busy, p.message, p.query, p.lastQuery, p.candidates = false, "", "", "", nil
	p.scroll = ui.ScrollState{}
	if p.stop != nil {
		p.stop()
		p.stop = nil
	}
}

func candidateMatches(candidate providerdiscovery.Candidate, query string) bool {
	terms := strings.Fields(strings.ToLower(strings.TrimSpace(query)))
	if len(terms) == 0 {
		return true
	}
	haystack := strings.ToLower(candidate.ID + " " + candidate.Name)
	for _, term := range terms {
		if !strings.Contains(haystack, term) {
			return false
		}
	}
	return true
}

func formatCandidateTokens(n int64) string {
	if n <= 0 {
		return ""
	}
	switch {
	case n >= 1_000_000:
		if n%1_000_000 == 0 {
			return fmt.Sprintf("%dM", n/1_000_000)
		}
		if n%1_048_576 == 0 {
			return fmt.Sprintf("%dM", n/1_048_576)
		}
		text := strings.TrimRight(strings.TrimRight(fmt.Sprintf("%.1f", float64(n)/1_000_000), "0"), ".")
		return text + "M"
	case n >= 1000:
		if n%1000 == 0 {
			return fmt.Sprintf("%dK", n/1000)
		}
		if n%1024 == 0 {
			return fmt.Sprintf("%dK", n/1024)
		}
		return fmt.Sprintf("%dK", (n+500)/1000)
	default:
		return fmt.Sprintf("%d", n)
	}
}

func candidateDisplayTitle(candidate providerdiscovery.Candidate) string {
	if name := strings.TrimSpace(candidate.Name); name != "" {
		return name
	}
	return strings.TrimSpace(candidate.ID)
}

func candidateTooltip(candidate providerdiscovery.Candidate) string {
	parts := []string{candidateDisplayTitle(candidate)}
	if id := strings.TrimSpace(candidate.ID); id != "" && id != parts[0] {
		parts = append(parts, id)
	}
	if candidate.ContextWindow > 0 {
		parts = append(parts, fmt.Sprintf("%d tokens", candidate.ContextWindow))
	}
	if candidate.MaxTokens > 0 {
		parts = append(parts, fmt.Sprintf(providerCopy("最大输出 %d tokens", "max output %d tokens"), candidate.MaxTokens))
	}
	return strings.Join(parts, " · ")
}

func candidateRowHeight(c *ui.Context) float32 {
	return lh(design.TextSM) + lh(design.TextXS) + design.Sp(4.5)
}

func modelCandidateRow(c *ui.Context, key string, candidate providerdiscovery.Candidate, picked *bool, known, disabled bool) ui.Element {
	t := tk(c)
	title := candidateDisplayTitle(candidate)
	id := strings.TrimSpace(candidate.ID)
	row := ui.CheckboxBase(c.Key(key), picked).
		FillWidth().MinWidth(0).MinHeight(candidateRowHeight(c)).
		Padding(design.Sp(2), design.Sp(2.5)).Gap(design.Sp(2.5)).
		Radius(design.RadiusLG).AlignItems(ui.Center).
		FocusRing(false).Cursor(ui.CursorPointer).
		Label(title)
	if tip := candidateTooltip(candidate); tip != "" {
		row.Tooltip(tip)
	}
	if disabled {
		row.Disabled(true).DisabledOpacity(design.OpacityDisabled)
	}
	hovered := !disabled && row.Hovered()
	focused := !disabled && row.FocusVisible()
	switch {
	case hovered && *picked:
		row.Background(t.Input.Alpha(0.75))
	case hovered:
		row.Background(t.Accent)
	case *picked:
		row.Background(t.Muted)
	}
	if focused {
		row.Shadow(0, 0, 0, design.RingWidth, t.Ring)
	}
	row.Children(func() {
		box := ui.Box(c).Size(16, 16).Radius(4).Center().Shrink(0)
		if *picked {
			box.Background(t.Primary).Border(design.BorderWidth, ui.Transparent).Children(func() {
				ui.Icon(c, coss.IconCheckboxCheck.SVG()).Size(12, 12).TextColor(t.PrimaryForeground).PassThrough()
			})
		} else {
			bg := t.Background
			if t.IsDark() {
				bg = t.Input.Alpha(0.32)
			}
			box.Background(bg).Border(design.BorderWidth, t.Input)
		}
		ui.Column(c).Grow(1).Shrink(1).MinWidth(0).Gap(design.Sp(0.5)).Children(func() {
			kText(c, title, design.TextSM, design.WeightMedium, t.Foreground).
				FillWidth().MinWidth(0).SingleLine().Ellipsis("…")
			if id != "" && id != title {
				kText(c, id, design.TextXS, design.WeightNormal, t.MutedForeground).
					FillWidth().MinWidth(0).SingleLine().Ellipsis("…")
			}
		})
		if known || candidate.ContextWindow > 0 {
			ui.Row(c).Shrink(0).AlignItems(ui.Center).Gap(design.Sp(1.5)).Children(func() {
				if known {
					coss.Badge(c, providerCopy("已添加", "Added"), coss.BadgeOptions{
						Variant: coss.BadgeOutline,
						Size:    coss.BadgeMD,
					})
				}
				if candidate.ContextWindow > 0 {
					coss.Badge(c, formatCandidateTokens(candidate.ContextWindow)+" tokens", coss.BadgeOptions{
						Variant: coss.BadgeSecondary,
						Size:    coss.BadgeMD,
					}).FontFeatures("tnum")
				}
			})
		}
	})
	return row
}

func modelCandidateDialog(c *ui.Context, state *modelProbeState, rows []any, saving, writable bool, change func([]any)) {
	known := map[string]bool{}
	for _, raw := range rows {
		if id := strings.TrimSpace(configurationText(configurationObjectValue(raw)["id"])); id != "" {
			known[id] = true
		}
	}
	open := true
	bodyDialogFrame(c, &open, 560, func() {
		t := tk(c)
		// Header
		ui.Row(c).FillWidth().MinWidth(0).Shrink(0).
			Padding(design.Sp(5), design.Sp(4), design.Sp(3), design.Sp(6)).
			Gap(design.Sp(2)).AlignItems(ui.Start).Children(func() {
			ui.Column(c).Grow(1).MinWidth(0).Gap(design.Sp(1)).Children(func() {
				kText(c, providerCopy("选择模型", "Choose models"), design.TextStyle{Size: design.TextXL.Size, LineHeight: design.TextXL.Size}, design.WeightBold, t.PopoverForeground).
					FillWidth().Role(ui.RoleHeading)
				kHint(c, providerCopy("勾选要加入当前提供商目录的候选模型。", "Select candidate models to add to this provider's catalog."))
			})
			closeBtn := coss.Button(c, "", coss.ButtonOptions{
				Variant:  coss.ButtonGhost,
				Size:     coss.ButtonIcon,
				Icon:     coss.IconX,
				Label:    providerCopy("关闭", "Close"),
				Disabled: saving,
				Key:      "model-picker-close",
			})
			closeBtn.Margin(-design.Sp(1), 0, 0, 0)
			if closeBtn.Clicked() {
				open = false
			}
		})

		// Compute selection counts before building search/toolbar/list/footer.
		query := strings.TrimSpace(state.query)
		if query != state.lastQuery {
			state.lastQuery = query
			state.scroll.Y = 0
		}
		matchedCount, matchedPicked, totalPicked := 0, 0, 0
		for _, candidate := range state.candidates {
			if candidate.Picked {
				totalPicked++
			}
			if candidateMatches(candidate, query) {
				matchedCount++
				if candidate.Picked {
					matchedPicked++
				}
			}
		}
		allMatchedPicked := matchedCount > 0 && matchedPicked == matchedCount

		// Sticky search input & selection toolbar.
		ui.Column(c).FillWidth().MinWidth(0).Shrink(0).
			Gap(design.Sp(2.5)).Padding(0, design.Sp(6), design.Sp(3), design.Sp(6)).Children(func() {
			var clearAddon func()
			if state.query != "" {
				clearAddon = func() {
					if coss.Button(c, "", coss.ButtonOptions{
						Variant:  coss.ButtonGhost,
						Size:     coss.ButtonIconXS,
						Icon:     coss.IconX,
						Label:    providerCopy("清除搜索", "Clear search"),
						Disabled: saving,
						Key:      "model-picker-clear-query",
					}).Clicked() {
						state.query = ""
					}
				}
			}
			search := coss.InputGroup(c, &state.query, coss.InputGroupOptions{
				InputOptions: coss.InputOptions{
					Size:        coss.InputMD,
					Label:       providerCopy("搜索候选模型", "Search candidate models"),
					Placeholder: providerCopy("搜索候选模型", "Search candidate models"),
					Disabled:    saving,
					Key:         "model-picker-search",
				},
				Start: func() { coss.AddonIcon(c, coss.IconSearch) },
				End:   clearAddon,
			})
			search.AutoFocus()

			ui.Row(c).FillWidth().MinWidth(0).AlignItems(ui.Center).Justify(ui.SpaceBetween).Gap(design.Sp(2)).Children(func() {
				summary := fmt.Sprintf(providerCopy("共 %d 项 · 已选 %d 项", "%d models · %d selected"), len(state.candidates), totalPicked)
				if query != "" {
					summary = fmt.Sprintf(providerCopy("匹配 %d / %d 项 · 已选 %d 项", "%d of %d matching · %d selected"), matchedCount, len(state.candidates), totalPicked)
				}
				kText(c, summary, design.TextXS, design.WeightNormal, t.MutedForeground).
					FontFeatures("tnum").Shrink(1).MinWidth(0).SingleLine().Ellipsis("…")

				toggleLabel := providerCopy("全选", "Select all")
				if query != "" {
					toggleLabel = providerCopy("全选筛选结果", "Select filtered")
				}
				if allMatchedPicked {
					toggleLabel = providerCopy("取消全选", "Deselect all")
					if query != "" {
						toggleLabel = providerCopy("取消全选筛选", "Deselect filtered")
					}
				}
				if coss.Button(c, toggleLabel, coss.ButtonOptions{
					Variant:  coss.ButtonOutline,
					Size:     coss.ButtonSM,
					Disabled: saving || matchedCount == 0,
					Key:      "model-picker-toggle-all",
				}).Clicked() {
					for i := range state.candidates {
						if candidateMatches(state.candidates[i], query) {
							state.candidates[i].Picked = !allMatchedPicked
						}
					}
				}
			})
		})

		// Framed bounded candidate list (single scroll container).
		// Using the same step (Sp(1.5)) for list gap and top/bottom padding
		// ensures exactly 6 complete rows fit with zero partial sliver peeking
		// at scroll Y=0 and no redundant scrollbar when len == 6.
		_, vh := c.Size()
		rowH := candidateRowHeight(c)
		listGap := design.Sp(1.5)
		sixRowInner := 6*rowH + 5*listGap + 2*listGap
		availInner := max(rowH*2+3*listGap, vh-276)
		listMax := min(sixRowInner, availInner)

		ui.Column(c).FillWidth().MinWidth(0).MinHeight(0).Shrink(1).
			Padding(0, design.Sp(6), design.Sp(4), design.Sp(6)).Children(func() {
			card := ui.Column(c.Key("model-picker-list-card")).FillWidth().MinWidth(0).MinHeight(0).
				Radius(design.RadiusXL).Border(design.BorderWidth, t.Border).Background(t.Card).Clip()
			card.Children(func() {
				if matchedCount == 0 {
					ui.Column(c.Key("model-picker-empty")).FillWidth().MinWidth(0).
						AlignItems(ui.Center).Justify(ui.Center).
						Padding(design.Sp(8), design.Sp(6)).Gap(design.Sp(2)).Children(func() {
						ui.Box(c).Size(36, 36).Center().Radius(design.RadiusMD).
							Background(t.Muted).Border(design.BorderWidth, t.Border).Children(func() {
							ui.Icon(c, coss.IconSearch.SVG()).Size(design.IconSM, design.IconSM).TextColor(t.MutedForeground).PassThrough()
						})
						kText(c, providerCopy("没有匹配的候选模型", "No matching candidate models"), design.TextSM, design.WeightMedium, t.Foreground).TextAlign(ui.Center)
						kText(c, providerCopy("尝试更换关键词，或清除搜索查看全部候选。", "Try another keyword, or clear the search to view all candidates."), design.TextXS, design.WeightNormal, t.MutedForeground).TextAlign(ui.Center)
						if coss.Button(c, providerCopy("清除搜索", "Clear search"), coss.ButtonOptions{
							Variant:  coss.ButtonOutline,
							Size:     coss.ButtonSM,
							Disabled: saving,
							Key:      "model-picker-empty-clear",
						}).Clicked() {
							state.query = ""
						}
					})
					return
				}
				coss.ScrollArea(c, coss.ScrollAreaOptions{State: &state.scroll, Key: "model-picker-scroll"}, func() {
					ui.Column(c).FillWidth().MinWidth(0).
						Padding(listGap, design.Sp(3), listGap, listGap).
						Gap(listGap).Role(ui.RoleList).Label(providerCopy("候选模型列表", "Candidate models")).Children(func() {
						for i := range state.candidates {
							candidate := &state.candidates[i]
							if !candidateMatches(*candidate, query) {
								continue
							}
							modelCandidateRow(c, "candidate/"+candidate.ID, *candidate, &candidate.Picked, known[candidate.ID], saving)
						}
					})
				}).FillWidth().MinWidth(0).MinHeight(0).MaxHeight(listMax)
			})
		})

		// Footer strip: quiet secondary Cancel on the left, primary Adopt on the right.
		ui.Row(c).FillWidth().MinWidth(0).Shrink(0).
			Padding(design.Sp(4), design.Sp(6)).Wrap().Gap(design.Sp(2)).
			Justify(ui.End).AlignItems(ui.Center).
			Background(t.Muted.Alpha(0.72)).
			BorderWidth(design.BorderWidth, 0, 0, 0).BorderColor(t.Border).Children(func() {
			if coss.Button(c, providerCopy("取消", "Cancel"), coss.ButtonOptions{
				Variant:  coss.ButtonOutline,
				Disabled: saving,
				Key:      "model-picker-cancel",
			}).Clicked() {
				open = false
			}
			adopt := coss.Button(c, providerCopy("采纳选中模型", "Adopt selected models"), coss.ButtonOptions{
				Variant:  coss.ButtonDefault,
				Disabled: saving || !writable || totalPicked == 0,
				Key:      "model-picker-adopt",
			})
			if totalPicked == 0 && writable && !saving {
				adopt.Tooltip(providerCopy("请至少勾选一个候选模型", "Select at least one candidate model"))
			}
			if adopt.Clicked() {
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
				state.candidates, state.query, state.lastQuery = nil, "", ""
				state.scroll = ui.ScrollState{}
				open = false
			}
		})
	}, ui.DialogOptions{DismissDisabled: saving})
	if !open {
		state.candidates = nil
		state.query = ""
		state.lastQuery = ""
		state.scroll = ui.ScrollState{}
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
			state.busy, state.message, state.query, state.lastQuery, state.candidates = true, "", "", "", nil
			state.scroll = ui.ScrollState{}
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
					state.message = discoveryMessage(err)
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
		if state.busy && kGhost(c, providerCopy("取消获取", "Cancel fetch"), false).Clicked() {
			state.cancel()
		}
	})
	settingsNotice(c, "model-fetch/"+scope, state.message != "", func() string { return state.message })
	if len(state.candidates) == 0 {
		return
	}
	modelCandidateDialog(c, state, rows, s.Saving, s.Writable, change)
}
