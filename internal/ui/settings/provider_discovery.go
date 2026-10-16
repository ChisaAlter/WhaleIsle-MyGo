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
}

func providerDiscoveryPane(c *ui.Context, a *ProviderDiscoveryAccess, fg, muted ui.Color) {
	if a == nil {
		return
	}
	s := a.Snapshot
	ui.Text(c, providerCopy("探测并采纳供应商模型", "Discover and adopt provider models")).FontSize(fonttheme.InterfaceSize(14)).Bold().TextColor(fg)
	ui.Text(c, providerCopy("仅点击探测后由此桌面访问所示地址；已有 Host 密钥不会读取或代用。临时密钥仅用于本次探测，随后丢弃。", "Only clicking Discover sends a request from this desktop to the address below. Stored Host keys are never read or reused. A temporary key is discarded after this probe.")).FontSize(fonttheme.InterfaceSize(11)).TextColor(muted)
	if !s.Connected {
		ui.Text(c, providerCopy("请先连接 Host。", "Connect to a Host first.")).TextColor(muted)
		return
	}
	if s.Err != "" {
		ui.Text(c, s.Err).TextColor(fg)
	}
	if s.LastAction != "" {
		ui.Text(c, s.LastAction).FontSize(fonttheme.InterfaceSize(11)).TextColor(muted)
	}
	if ui.Button(c, providerCopy("刷新供应商", "Refresh providers")).Disabled(s.Busy || s.Dirty || a.OnRefresh == nil).Clicked() && a.OnRefresh != nil {
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
	if ui.Select(c, &current, options).FillWidth().Disabled(s.Busy || s.Dirty).Changed() && a.OnSelect != nil {
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
		ui.Text(c, providerCopy("有未保存草稿；保存或放弃后可切换供应商。", "Unsaved draft: save or discard before changing providers.")).FontSize(fonttheme.InterfaceSize(11)).TextColor(muted)
	}
	if row.Blocked != "" {
		ui.Text(c, row.Blocked).TextColor(fg)
	}
	if !row.Writable {
		ui.Text(c, providerCopy("Host 设置只读，无法采纳或保存。", "Host settings are read-only; adoption and saving are disabled.")).TextColor(muted)
	}
	endpoint, protocol, key := s.Endpoint, s.Protocol, s.Key
	ui.Text(c, "Endpoint").FontSize(fonttheme.InterfaceSize(11)).TextColor(muted)
	if ui.TextInput(c, &endpoint).FillWidth().Placeholder("https://provider.example/v1").Disabled(s.Busy).Changed() && a.OnEdit != nil {
		a.OnEdit(s.Ticket, "endpoint", endpoint)
	}
	ui.Text(c, providerCopy("探测协议", "Discovery protocol")).FontSize(fonttheme.InterfaceSize(11)).TextColor(muted)
	protocols := append([]string(nil), providerdiscovery.Protocols...)
	found := false
	for _, p := range protocols {
		if p == protocol {
			found = true
		}
	}
	if !found {
		protocols = append(protocols, protocol)
	}
	if ui.Select(c, &protocol, protocols).FillWidth().Disabled(s.Busy).Changed() && a.OnEdit != nil {
		a.OnEdit(s.Ticket, "protocol", protocol)
	}
	target, targetErr := providerdiscovery.ListingURL(s.Endpoint, s.Protocol)
	if targetErr == nil {
		ui.Text(c, "GET "+target).FontSize(fonttheme.InterfaceSize(11)).TextColor(fg)
	} else {
		ui.Text(c, targetErr.Error()).FontSize(fonttheme.InterfaceSize(11)).TextColor(muted)
	}
	if row.CredentialKnown && row.CredentialConfigured {
		ui.Text(c, providerCopy("Host 密钥已配置 · ••••••••，本次探测不会使用它。", "Host key configured · ••••••••; this probe does not use it.")).FontSize(fonttheme.InterfaceSize(11)).TextColor(muted)
	}
	if ui.TextInput(c, &key).FillWidth().Password().Placeholder(providerCopy("临时密钥；留空匿名探测", "Temporary key; blank probes anonymously")).Disabled(s.Busy).Changed() && a.OnEdit != nil {
		a.OnEdit(s.Ticket, "key", key)
	}
	ui.Text(c, providerCopy("HTTPS 可携带临时密钥；本地 HTTP 仅允许匿名。禁止重定向。保存密钥请使用上方凭据入口。", "HTTPS may carry the temporary key; loopback HTTP is anonymous only. Redirects are blocked. Use the credential controls above to save a key.")).FontSize(fonttheme.InterfaceSize(11)).TextColor(muted)
	probeLabel := providerCopy("匿名探测此地址", "Discover anonymously at this address")
	if strings.TrimSpace(s.Key) != "" {
		probeLabel = providerCopy("发送临时密钥并探测此地址", "Send temporary key and discover at this address")
	}
	if ui.Button(c, probeLabel).Disabled(s.Busy || targetErr != nil || a.OnDiscover == nil).Clicked() && a.OnDiscover != nil {
		a.OnDiscover(s.Ticket)
	}
	if s.Busy {
		ui.Text(c, providerCopy("正在处理…", "Working…")).TextColor(muted)
	}
	if len(s.Candidates) > 0 {
		query := s.Query
		if ui.TextInput(c, &query).FillWidth().Placeholder(providerCopy("搜索候选模型", "Search candidates")).Disabled(s.Busy).Changed() && a.OnEdit != nil {
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
		if ui.Button(c, providerCopy("将选中项加入草稿", "Add selected models to draft")).Disabled(s.Busy || !row.Writable || row.Blocked != "").Clicked() && a.OnAdopt != nil {
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
		ui.Text(c, providerCopy("确认保存此供应商的模型列表，以及已修改的 endpoint / 协议。已有模型的其他字段保留；临时密钥不保存。更改目标地址会影响今后使用此供应商及其已保存凭据的请求。", "Confirm this provider's model list and edited endpoint/protocol. Other fields on existing models are preserved; the temporary key is not saved. Changing the target affects future requests using this provider and its saved credentials.")).TextColor(fg)
		ui.Text(c, row.Provider+" · "+s.Endpoint+" · "+s.Protocol).FontSize(fonttheme.InterfaceSize(11)).TextColor(fg)
		ui.Row(c).Gap(8).Children(func() {
			if ui.Button(c, providerCopy("确认保存供应商模型", "Confirm save provider models")).Disabled(s.Busy).Clicked() && a.OnSave != nil {
				a.OnSave(s.Ticket)
			}
			if ui.Button(c, providerCopy("取消保存", "Cancel save")).Disabled(s.Busy).Clicked() && a.OnReview != nil {
				a.OnReview(s.Ticket, false)
			}
		})
	} else if ui.Button(c, providerCopy("审阅并保存草稿…", "Review and save draft…")).Disabled(s.Busy || !s.Dirty || !row.Writable || row.Blocked != "").Clicked() && a.OnReview != nil {
		a.OnReview(s.Ticket, true)
	}
	if ui.Button(c, providerCopy("放弃此草稿 / 清空候选", "Discard draft / clear candidates")).Disabled(s.Busy).Clicked() && a.OnDiscard != nil {
		a.OnDiscard(s.Ticket)
	}
}
