package settings

import (
	"fmt"
	"strings"

	"whaleisle/internal/desktop/providerdiscovery"
	"whaleisle/internal/ui/coss"
	"whaleisle/internal/ui/design"

	"github.com/egoist/mygo/ui"
)

type ProviderDiscoveryAccess struct {
	Snapshot                               providerdiscovery.Snapshot
	OnRefresh                              func()
	OnSelect                               func(uint64, string)
	OnEdit                                 func(uint64, string, string)
	OnDiscover, OnAdopt, OnDiscard, OnSave func(uint64)
	OnPick                                 func(uint64, string, bool)
	OnReview                               func(uint64, bool)
	OnProbeEditor                          func(providerdiscovery.EditorProbe, func([]providerdiscovery.Candidate, error)) func()
}

func providerDiscoveryPane(c *ui.Context, a *ProviderDiscoveryAccess, fg, muted ui.Color) {
	if a == nil {
		return
	}
	s := a.Snapshot
	t := tk(c)
	kHeading(c, providerCopy("探测并采纳供应商模型", "Discover and adopt provider models"))
	kHint(c, providerCopy("点击探测后由当前 Host 适配器查找模型。内置供应商可直接返回目录；访问端点时会使用该供应商已保存的密钥和 headers。临时密钥优先使用，仅用于本次探测。", "Discover asks the current Host adapter for models. Built-in providers can use their catalog; endpoint requests use the provider’s saved credentials and headers. A temporary key overrides the saved key for this request only."))
	if !s.Connected {
		kHint(c, providerCopy("请先连接 Host。", "Connect to a Host first."))
		return
	}
	settingsAlert(c, "provider_discovery:35", s.Err != "", providerdiscovery.Message(s.Err), s.Err)
	settingsNotice(c, "provider_discovery:38", s.LastAction != "", func() string { return providerdiscovery.Message(s.LastAction) })
	if kButtonDisabled(c, providerCopy("刷新供应商", "Refresh providers"), s.Busy || s.Dirty || a.OnRefresh == nil).AlignSelf(ui.Start).Clicked() && a.OnRefresh != nil {
		a.OnRefresh()
	}
	if !s.Ready {
		return
	}
	options := []string{providerCopy("选择供应商", "Select provider")}
	current := options[0]
	var row *providerdiscovery.Row
	for _, r := range s.Rows {
		label := r.Name + " · " + r.Provider + " · " + strings.Join(r.Path, "/")
		options = append(options, label)
		if r.ID == s.SelectedID {
			current = label
			copy := r
			row = &copy
		}
	}
	if len(s.Rows) == 0 {
		kHint(c, providerCopy("Host 未提供可探测的 pi-ai 供应商。", "The Host exposes no pi-ai provider for this editor."))
		return
	}
	if kSelect(c, &current, options, providerCopy("选择供应商", "Select provider"), s.Busy || s.Dirty).Changed() && a.OnSelect != nil {
		for i, r := range s.Rows {
			if current == options[i+1] {
				a.OnSelect(s.Ticket, r.ID)
				break
			}
		}
	}
	if row == nil {
		return
	}
	if s.Dirty {
		kHint(c, providerCopy("有未保存草稿；保存或放弃后可切换供应商。", "Unsaved draft: save or discard before changing providers."))
	}
	if row.Blocked != "" {
		kBody(c, providerdiscovery.Message(row.Blocked))
	}
	if !row.Writable {
		kHint(c, providerCopy("Host 设置只读，无法采纳或保存。", "Host settings are read-only; adoption and saving are disabled."))
	}
	endpoint, protocol, key := s.Endpoint, s.Protocol, s.Key
	kHint(c, providerCopy("Endpoint（内置供应商可留空）", "Endpoint (optional for built-in providers)"))
	if coss.Input(c, &endpoint, coss.InputOptions{
		Label:       providerCopy("Endpoint", "Endpoint"),
		Placeholder: "https://provider.example/v1",
		Disabled:    s.Busy,
		Key:         "provider-discovery-endpoint",
	}).Changed() && a.OnEdit != nil {
		a.OnEdit(s.Ticket, "endpoint", endpoint)
	}
	kHint(c, providerCopy("探测协议", "Discovery protocol"))
	defaultProtocol := providerCopy("由 Host 选择默认协议", "Use Host default protocol")
	if protocol == "" {
		protocol = defaultProtocol
	}
	protocols := append([]string(nil), providerdiscovery.Protocols...)
	if row.Protocol == "" {
		protocols = append([]string{defaultProtocol}, protocols...)
	}
	found := false
	for _, p := range protocols {
		if p == protocol {
			found = true
		}
	}
	if !found {
		protocols = append(protocols, protocol)
	}
	if kSelect(c, &protocol, protocols, providerCopy("探测协议", "Discovery protocol"), s.Busy).Changed() && a.OnEdit != nil {
		if protocol == defaultProtocol {
			protocol = ""
		}
		a.OnEdit(s.Ticket, "protocol", protocol)
	}
	kText(c, "Host · "+row.Namespace+" · "+row.Provider, design.TextXS, design.WeightMedium, t.Foreground).FillWidth()
	if row.CredentialKnown && row.CredentialConfigured {
		kHint(c, providerCopy("Host 密钥已配置 · ••••••••；不填临时密钥时由 Host 按需使用。", "Host key configured · ••••••••; the Host uses it when needed if the temporary key is blank."))
	}
	if coss.Input(c, &key, coss.InputOptions{
		Label:       providerCopy("临时密钥", "Temporary key"),
		Placeholder: providerCopy("临时密钥；留空使用 Host 已有凭据", "Temporary key; blank uses Host credentials"),
		Password:    true,
		Disabled:    s.Busy,
		Key:         "provider-discovery-key",
	}).Changed() && a.OnEdit != nil {
		a.OnEdit(s.Ticket, "key", key)
	}
	kHint(c, providerCopy("修改 endpoint 会让 Host 向该地址发送供应商凭据和 headers，请确认目标可信。临时密钥在探测后清空；保存密钥请使用上方凭据入口。", "An edited endpoint receives the provider credentials and headers from the Host; check that you trust it. The temporary key is cleared after discovery. Use the credential controls above to save a key."))
	probeLabel := providerCopy("通过 Host 探测模型", "Discover models through Host")
	if strings.TrimSpace(s.Key) != "" {
		probeLabel = providerCopy("使用临时密钥通过 Host 探测", "Discover through Host with temporary key")
	}
	if kPrimary(c, probeLabel, s.Busy || a.OnDiscover == nil).AlignSelf(ui.Start).Clicked() && a.OnDiscover != nil {
		a.OnDiscover(s.Ticket)
	}
	if len(s.Candidates) > 0 {
		known := map[string]bool{}
		for _, model := range s.Models {
			if id, ok := model["id"].(string); ok && strings.TrimSpace(id) != "" {
				known[strings.TrimSpace(id)] = true
			}
		}
		query := s.Query
		var clearAddon func()
		if query != "" {
			clearAddon = func() {
				if coss.Button(c, "", coss.ButtonOptions{
					Variant:  coss.ButtonGhost,
					Size:     coss.ButtonIconXS,
					Icon:     coss.IconX,
					Label:    providerCopy("清除搜索", "Clear search"),
					Disabled: s.Busy,
					Key:      "provider-discovery-clear-query",
				}).Clicked() && a.OnEdit != nil {
					a.OnEdit(s.Ticket, "query", "")
				}
			}
		}
		if coss.InputGroup(c, &query, coss.InputGroupOptions{
			InputOptions: coss.InputOptions{
				Label:       providerCopy("搜索候选模型", "Search candidates"),
				Placeholder: providerCopy("搜索候选模型", "Search candidates"),
				Disabled:    s.Busy,
				Key:         "provider-discovery-search",
			},
			Start: func() { coss.AddonIcon(c, coss.IconSearch) },
			End:   clearAddon,
		}).Changed() && a.OnEdit != nil {
			a.OnEdit(s.Ticket, "query", query)
		}
		matchedCount, matchedPicked, totalPicked := 0, 0, 0
		for _, candidate := range s.Candidates {
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
		ui.Row(c).FillWidth().MinWidth(0).AlignItems(ui.Center).Justify(ui.SpaceBetween).Gap(design.Sp(2)).Children(func() {
			summary := fmt.Sprintf(providerCopy("共 %d 项 · 已选 %d 项", "%d models · %d selected"), len(s.Candidates), totalPicked)
			if strings.TrimSpace(query) != "" {
				summary = fmt.Sprintf(providerCopy("匹配 %d / %d 项 · 已选 %d 项", "%d of %d matching · %d selected"), matchedCount, len(s.Candidates), totalPicked)
			}
			kText(c, summary, design.TextXS, design.WeightNormal, t.MutedForeground).FontFeatures("tnum").Shrink(1).MinWidth(0).SingleLine().Ellipsis("…")
			if a.OnPick != nil {
				toggleLabel := providerCopy("全选", "Select all")
				if strings.TrimSpace(query) != "" {
					toggleLabel = providerCopy("全选筛选结果", "Select filtered")
				}
				if allMatchedPicked {
					toggleLabel = providerCopy("取消全选", "Deselect all")
					if strings.TrimSpace(query) != "" {
						toggleLabel = providerCopy("取消全选筛选", "Deselect filtered")
					}
				}
				if kButtonDisabled(c, toggleLabel, s.Busy || matchedCount == 0).Clicked() {
					target := !allMatchedPicked
					for _, candidate := range s.Candidates {
						if candidateMatches(candidate, query) && candidate.Picked != target {
							a.OnPick(s.Ticket, candidate.ID, target)
						}
					}
				}
			}
		})
		rowH := candidateRowHeight(c)
		listGap := design.Sp(1.5)
		listMax := 5*rowH + 4*listGap + 2*listGap
		ui.Column(c.Key("provider-discovery-list-card")).FillWidth().MinWidth(0).
			Radius(design.RadiusXL).Border(design.BorderWidth, t.Border).Background(t.Card).Clip().Children(func() {
			if matchedCount == 0 {
				ui.Column(c).FillWidth().AlignItems(ui.Center).Justify(ui.Center).
					Padding(design.Sp(6), design.Sp(4)).Gap(design.Sp(2)).Children(func() {
					kText(c, providerCopy("没有匹配的候选模型", "No matching candidate models"), design.TextSM, design.WeightMedium, t.Foreground).TextAlign(ui.Center)
					if a.OnEdit != nil && kButtonDisabled(c, providerCopy("清除搜索", "Clear search"), s.Busy).Clicked() {
						a.OnEdit(s.Ticket, "query", "")
					}
				})
				return
			}
			coss.ScrollArea(c, coss.ScrollAreaOptions{Key: "provider-discovery-scroll"}, func() {
				ui.Column(c).FillWidth().MinWidth(0).
					Padding(listGap, design.Sp(3), listGap, listGap).
					Gap(listGap).Role(ui.RoleList).Children(func() {
					for _, candidate := range s.Candidates {
						if !candidateMatches(candidate, query) {
							continue
						}
						picked := candidate.Picked
						if modelCandidateRow(c, "provider-candidate/"+candidate.ID, candidate, &picked, known[candidate.ID], s.Busy).Changed() && a.OnPick != nil {
							a.OnPick(s.Ticket, candidate.ID, picked)
						}
					}
				})
			}).FillWidth().MinWidth(0).MinHeight(0).MaxHeight(listMax)
		})
		if kPrimary(c, providerCopy("将选中项加入草稿", "Add selected models to draft"), s.Busy || !row.Writable || row.Blocked != "" || totalPicked == 0).AlignSelf(ui.Start).Clicked() && a.OnAdopt != nil {
			a.OnAdopt(s.Ticket)
		}
	}
	kText(c, fmt.Sprintf(providerCopy("模型草稿：%d 项", "Model draft: %d entries"), len(s.Models)), design.TextXS, design.WeightMedium, t.Foreground).FillWidth()
	for _, model := range s.Models {
		if id, ok := model["id"].(string); ok {
			kHint(c, id)
		}
	}
	if s.Review {
		kBody(c, providerCopy("确认保存此供应商的模型列表，以及已修改的 endpoint 与协议。已有模型的其他字段保留；临时密钥不保存。更改目标地址会影响今后使用此供应商及其已保存凭据的请求。", "Confirm this provider's model list and edited endpoint/protocol. Other fields on existing models are preserved; the temporary key is not saved. Changing the target affects future requests using this provider and its saved credentials."))
		kText(c, row.Provider+" · "+s.Endpoint+" · "+s.Protocol, design.TextXS, design.WeightNormal, t.Foreground).FillWidth()
		ui.Row(c).Gap(design.Sp(2)).Wrap().Children(func() {
			if kButtonDisabled(c, providerCopy("取消保存", "Cancel save"), s.Busy).Clicked() && a.OnReview != nil {
				a.OnReview(s.Ticket, false)
			}
			if kPrimary(c, providerCopy("确认保存供应商模型", "Confirm save provider models"), s.Busy).Clicked() && a.OnSave != nil {
				a.OnSave(s.Ticket)
			}
		})
	} else if kButtonDisabled(c, providerCopy("审阅并保存草稿…", "Review and save draft…"), s.Busy || !s.Dirty || !row.Writable || row.Blocked != "").AlignSelf(ui.Start).Clicked() && a.OnReview != nil {
		a.OnReview(s.Ticket, true)
	}
	if kGhost(c, providerCopy("放弃此草稿并清空候选", "Discard the draft and clear candidates"), s.Busy).AlignSelf(ui.Start).Clicked() && a.OnDiscard != nil {
		a.OnDiscard(s.Ticket)
	}
}
