package settings

import (
	"fmt"
	"strings"
	fonttheme "whaleisle/internal/ui/theme"

	"github.com/egoist/mygo/ui"
	"whaleisle/internal/desktop/providerdiscovery"
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
	ui.Text(c, providerCopy("探测并采纳供应商模型", "Discover and adopt provider models")).FontSize(fonttheme.InterfaceSize(14)).FontWeight(600).TextColor(fg)
	ui.Text(c, providerCopy("点击探测后由当前 Host 适配器查找模型。内置供应商可直接返回目录；访问端点时会使用该供应商已保存的密钥和 headers。临时密钥优先使用，仅用于本次探测。", "Discover asks the current Host adapter for models. Built-in providers can use their catalog; endpoint requests use the provider’s saved credentials and headers. A temporary key overrides the saved key for this request only.")).FontSize(fonttheme.InterfaceSize(12)).TextColor(muted)
	if !s.Connected {
		ui.Text(c, providerCopy("请先连接 Host。", "Connect to a Host first.")).TextColor(muted)
		return
	}
	settingsNotice(c, "provider_discovery:35", "error", s.Err != "", func() string { return providerdiscovery.Message(s.Err) })
	settingsNotice(c, "provider_discovery:38", "", s.LastAction != "", func() string { return providerdiscovery.Message(s.LastAction) })
	if ui.Button(c, providerCopy("刷新供应商", "Refresh providers")).AlignSelf(ui.Start).Height(30).Padding(0, 12).Radius(8).Border(0.5, c.Theme().Border.Alpha(0.45)).FontSize(fonttheme.InterfaceSize(12.5)).Disabled(s.Busy || s.Dirty || a.OnRefresh == nil).Clicked() && a.OnRefresh != nil {
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
		ui.Text(c, providerCopy("Host 未提供可探测的 pi-ai 供应商。", "The Host exposes no pi-ai provider for this editor.")).TextColor(muted)
		return
	}
	if ui.Select(c, &current, options).FillWidth().Height(32).Radius(8).Border(0.5, c.Theme().Border.Alpha(0.45)).FontSize(fonttheme.InterfaceSize(13)).Disabled(s.Busy || s.Dirty).Changed() && a.OnSelect != nil {
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
		ui.Text(c, providerCopy("有未保存草稿；保存或放弃后可切换供应商。", "Unsaved draft: save or discard before changing providers.")).FontSize(fonttheme.InterfaceSize(12)).TextColor(muted)
	}
	if row.Blocked != "" {
		ui.Text(c, providerdiscovery.Message(row.Blocked)).TextColor(fg)
	}
	if !row.Writable {
		ui.Text(c, providerCopy("Host 设置只读，无法采纳或保存。", "Host settings are read-only; adoption and saving are disabled.")).TextColor(muted)
	}
	endpoint, protocol, key := s.Endpoint, s.Protocol, s.Key
	ui.Text(c, providerCopy("Endpoint（内置供应商可留空）", "Endpoint (optional for built-in providers)")).FontSize(fonttheme.InterfaceSize(12)).TextColor(muted)
	if ui.TextInput(c, &endpoint).FillWidth().Height(32).Radius(8).Border(0.5, c.Theme().Border.Alpha(0.45)).FontSize(fonttheme.InterfaceSize(13)).Placeholder("https://provider.example/v1").Disabled(s.Busy).Changed() && a.OnEdit != nil {
		a.OnEdit(s.Ticket, "endpoint", endpoint)
	}
	ui.Text(c, providerCopy("探测协议", "Discovery protocol")).FontSize(fonttheme.InterfaceSize(12)).TextColor(muted)
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
	if ui.Select(c, &protocol, protocols).FillWidth().Height(32).Radius(8).Border(0.5, c.Theme().Border.Alpha(0.45)).FontSize(fonttheme.InterfaceSize(13)).Disabled(s.Busy).Changed() && a.OnEdit != nil {
		if protocol == defaultProtocol {
			protocol = ""
		}
		a.OnEdit(s.Ticket, "protocol", protocol)
	}
	ui.Text(c, "Host · "+row.Namespace+" · "+row.Provider).FontSize(fonttheme.InterfaceSize(12)).TextColor(fg)
	if row.CredentialKnown && row.CredentialConfigured {
		ui.Text(c, providerCopy("Host 密钥已配置 · ••••••••；不填临时密钥时由 Host 按需使用。", "Host key configured · ••••••••; the Host uses it when needed if the temporary key is blank.")).FontSize(fonttheme.InterfaceSize(12)).TextColor(muted)
	}
	if ui.TextInput(c, &key).FillWidth().Height(32).Radius(8).Border(0.5, c.Theme().Border.Alpha(0.45)).FontSize(fonttheme.InterfaceSize(13)).Password().Placeholder(providerCopy("临时密钥；留空使用 Host 已有凭据", "Temporary key; blank uses Host credentials")).Disabled(s.Busy).Changed() && a.OnEdit != nil {
		a.OnEdit(s.Ticket, "key", key)
	}
	ui.Text(c, providerCopy("修改 endpoint 会让 Host 向该地址发送供应商凭据和 headers，请确认目标可信。临时密钥在探测后清空；保存密钥请使用上方凭据入口。", "An edited endpoint receives the provider credentials and headers from the Host; check that you trust it. The temporary key is cleared after discovery. Use the credential controls above to save a key.")).FontSize(fonttheme.InterfaceSize(12)).TextColor(muted)
	probeLabel := providerCopy("通过 Host 探测模型", "Discover models through Host")
	if strings.TrimSpace(s.Key) != "" {
		probeLabel = providerCopy("使用临时密钥通过 Host 探测", "Discover through Host with temporary key")
	}
	if ui.Button(c, probeLabel).AlignSelf(ui.Start).Height(30).Padding(0, 12).Radius(8).Border(0.5, c.Theme().Border.Alpha(0.45)).FontSize(fonttheme.InterfaceSize(12.5)).Disabled(s.Busy || a.OnDiscover == nil).Clicked() && a.OnDiscover != nil {
		a.OnDiscover(s.Ticket)
	}
	if len(s.Candidates) > 0 {
		query := s.Query
		if ui.TextInput(c, &query).FillWidth().Height(32).Radius(8).Border(0.5, c.Theme().Border.Alpha(0.45)).FontSize(fonttheme.InterfaceSize(13)).Placeholder(providerCopy("搜索候选模型", "Search candidates")).Disabled(s.Busy).Changed() && a.OnEdit != nil {
			a.OnEdit(s.Ticket, "query", query)
		}
		q := strings.ToLower(strings.TrimSpace(query))
		for _, candidate := range s.Candidates {
			if q != "" && !strings.Contains(strings.ToLower(candidate.ID+" "+candidate.Name), q) {
				continue
			}
			picked := candidate.Picked
			label := candidate.Name + " · " + candidate.ID
			if candidate.ContextWindow > 0 {
				label += fmt.Sprintf(" · context %d", candidate.ContextWindow)
			}
			if candidate.MaxTokens > 0 {
				label += fmt.Sprintf(" · output %d", candidate.MaxTokens)
			}
			if ui.Checkbox(c, &picked, label).Disabled(s.Busy).Changed() && a.OnPick != nil {
				a.OnPick(s.Ticket, candidate.ID, picked)
			}
		}
		if ui.Button(c, providerCopy("将选中项加入草稿", "Add selected models to draft")).AlignSelf(ui.Start).Height(30).Padding(0, 12).Radius(8).Border(0.5, c.Theme().Border.Alpha(0.45)).FontSize(fonttheme.InterfaceSize(12.5)).Disabled(s.Busy || !row.Writable || row.Blocked != "").Clicked() && a.OnAdopt != nil {
			a.OnAdopt(s.Ticket)
		}
	}
	ui.Text(c, fmt.Sprintf(providerCopy("模型草稿：%d 项", "Model draft: %d entries"), len(s.Models))).FontSize(fonttheme.InterfaceSize(12)).TextColor(fg)
	for _, model := range s.Models {
		if id, ok := model["id"].(string); ok {
			ui.Text(c, id).FontSize(fonttheme.InterfaceSize(11)).TextColor(muted)
		}
	}
	if s.Review {
		ui.Text(c, providerCopy("确认保存此供应商的模型列表，以及已修改的 endpoint 与协议。已有模型的其他字段保留；临时密钥不保存。更改目标地址会影响今后使用此供应商及其已保存凭据的请求。", "Confirm this provider's model list and edited endpoint/protocol. Other fields on existing models are preserved; the temporary key is not saved. Changing the target affects future requests using this provider and its saved credentials.")).TextColor(fg)
		ui.Text(c, row.Provider+" · "+s.Endpoint+" · "+s.Protocol).FontSize(fonttheme.InterfaceSize(11)).TextColor(fg)
		ui.Row(c).Gap(8).Wrap().Children(func() {
			if ui.Button(c, providerCopy("确认保存供应商模型", "Confirm save provider models")).Height(30).Padding(0, 12).Radius(8).Border(0.5, c.Theme().Border.Alpha(0.45)).FontSize(fonttheme.InterfaceSize(12.5)).Disabled(s.Busy).Clicked() && a.OnSave != nil {
				a.OnSave(s.Ticket)
			}
			if ui.Button(c, providerCopy("取消保存", "Cancel save")).Height(30).Padding(0, 12).Radius(8).Border(0.5, c.Theme().Border.Alpha(0.45)).FontSize(fonttheme.InterfaceSize(12.5)).Disabled(s.Busy).Clicked() && a.OnReview != nil {
				a.OnReview(s.Ticket, false)
			}
		})
	} else if ui.Button(c, providerCopy("审阅并保存草稿…", "Review and save draft…")).AlignSelf(ui.Start).Height(30).Padding(0, 12).Radius(8).Border(0.5, c.Theme().Border.Alpha(0.45)).FontSize(fonttheme.InterfaceSize(12.5)).Disabled(s.Busy || !s.Dirty || !row.Writable || row.Blocked != "").Clicked() && a.OnReview != nil {
		a.OnReview(s.Ticket, true)
	}
	if ui.Button(c, providerCopy("放弃此草稿并清空候选", "Discard the draft and clear candidates")).AlignSelf(ui.Start).Height(30).Padding(0, 12).Radius(8).Border(0.5, c.Theme().Border.Alpha(0.45)).FontSize(fonttheme.InterfaceSize(12.5)).Disabled(s.Busy).Clicked() && a.OnDiscard != nil {
		a.OnDiscard(s.Ticket)
	}
}
