package settings

import (
	"github.com/egoist/mygo/ui"
	"whaleisle/internal/desktop/credentialrefs"
	fonttheme "whaleisle/internal/ui/theme"
)

// ReferencedCredentialsAccess is separate from local desktop credentials.json.
// Every operation is restricted to a Host-discovered provider reference.
type ReferencedCredentialsAccess struct {
	Snapshot  credentialrefs.Snapshot
	OnRefresh func()
	OnSelect  func(string)
	OnEdit    func(uint64, string)
	OnClear   func(uint64, bool)
	OnSave    func(uint64)
}

func referencedCredentialsPane(c *ui.Context, a *ReferencedCredentialsAccess, namespace string, fg, muted ui.Color) {
	if a == nil {
		return
	}
	s := a.Snapshot
	ui.Text(c, providerCopy("供应商或 Web Search API Key", "Provider or Web Search API key")).FontSize(fonttheme.InterfaceSize(14)).FontWeight(600).TextColor(fg)
	ui.Text(c, providerCopy("按 Host 配置中的引用保存；留空保留已有密钥。密钥不会读回。", "Saved under the Host configuration's reference. Leave blank to keep the existing key. Stored keys are never read back.")).FontSize(fonttheme.InterfaceSize(12)).TextColor(muted)
	if !s.Connected {
		ui.Text(c, providerCopy("请先连接 Host。", "Connect to a Host first.")).FontSize(fonttheme.InterfaceSize(12)).TextColor(muted)
		return
	}
	if s.Err != "" {
		ui.Text(c, s.Err).FontSize(fonttheme.InterfaceSize(12)).TextColor(fg)
	}
	if s.LastAction != "" {
		ui.Text(c, settingsActionText(s.LastAction)).FontSize(fonttheme.InterfaceSize(11)).TextColor(muted)
	}
	if ui.Button(c, providerCopy("刷新引用与可写状态", "Refresh references and writable state")).AlignSelf(ui.Start).Height(30).Padding(0, 12).Radius(8).Border(0.5, c.Theme().Border.Alpha(0.45)).FontSize(fonttheme.InterfaceSize(12.5)).Disabled(s.Busy || a.OnRefresh == nil).Clicked() && a.OnRefresh != nil {
		a.OnRefresh()
	}
	if s.Busy {
		ui.Text(c, providerCopy("正在检查或保存…", "Checking or saving…")).FontSize(fonttheme.InterfaceSize(11)).TextColor(muted)
	}
	if !s.Ready {
		return
	}
	rows := []credentialrefs.Row{}
	options := []string{providerCopy("选择供应商或搜索凭据引用", "Select a provider or search credential reference")}
	current := options[0]
	var selected *credentialrefs.Row
	for _, row := range s.Rows {
		if namespace != "" && row.Namespace != namespace {
			continue
		}
		if namespace == "" && row.Namespace == "web-search-deepseek" {
			continue
		}
		rows = append(rows, row)
		label := row.Name + " · " + row.Namespace + " · " + row.Ref
		options = append(options, label)
		if row.ID == s.SelectedID {
			copy := row
			selected = &copy
			current = label
		}
	}
	if len(rows) == 0 {
		ui.Text(c, providerCopy("此页没有可编辑的凭据引用。", "No editable credential references are available on this page.")).FontSize(fonttheme.InterfaceSize(12)).TextColor(muted)
		return
	}
	if ui.Select(c, &current, options).FillWidth().Height(32).Radius(8).Border(0.5, c.Theme().Border.Alpha(0.45)).FontSize(fonttheme.InterfaceSize(13)).Disabled(s.Busy).Changed() && a.OnSelect != nil {
		for i, row := range rows {
			if current == options[i+1] {
				a.OnSelect(row.ID)
				break
			}
		}
	}
	if selected == nil {
		return
	}
	row := *selected
	ui.Text(c, providerCopy("目标引用：", "Target reference: ")+row.Ref).FontSize(fonttheme.InterfaceSize(12)).TextColor(fg)
	status := providerCopy("尚未配置", "Not configured")
	if !row.Known {
		status = providerCopy("状态未知，请刷新", "State unknown; refresh first")
	} else if row.Configured {
		status = providerCopy("已配置 · ••••••••", "Configured · ••••••••")
	}
	if !row.Writable {
		status += " · " + providerCopy("只读，不可写", "Read-only, unavailable")
	}
	ui.Text(c, status).FontSize(fonttheme.InterfaceSize(12)).TextColor(muted)
	locked := s.Busy || !row.Known || !row.Writable
	draft := s.Draft
	if ui.TextInput(c, &draft).FillWidth().Height(32).Radius(8).Border(0.5, c.Theme().Border.Alpha(0.45)).FontSize(fonttheme.InterfaceSize(13)).Password().Placeholder(providerCopy("输入新密钥，留空保留", "Enter a new key; blank keeps existing")).Disabled(locked || s.ClearPending).Changed() && a.OnEdit != nil {
		a.OnEdit(s.Ticket, draft)
	}
	if row.BindReference {
		ui.Text(c, providerCopy("首次保存只会添加此供应商的 apiKeyEnv 引用，并使用当前配置版本校验。", "The first save adds only this provider's apiKeyEnv reference, guarded by the current settings revision.")).FontSize(fonttheme.InterfaceSize(12)).TextColor(muted)
	}
	if s.ClearPending {
		ui.Text(c, providerCopy("确认后将清除此引用的本地密钥；同用此引用的功能也会受影响。", "Confirming clears the local key for this reference, affecting other features that share it.")).FontSize(fonttheme.InterfaceSize(12)).TextColor(fg)
	}
	ui.Row(c).Gap(8).Wrap().Children(func() {
		saveLabel := providerCopy("保存此密钥", "Save this key")
		if s.ClearPending {
			saveLabel = providerCopy("确认清除此引用", "Confirm clear reference")
		}
		if ui.Button(c, saveLabel).Height(30).Padding(0, 12).Radius(8).Border(0.5, c.Theme().Border.Alpha(0.45)).FontSize(fonttheme.InterfaceSize(12.5)).Disabled(locked || a.OnSave == nil || (!s.ClearPending && s.Draft == "")).Clicked() && a.OnSave != nil {
			a.OnSave(s.Ticket)
		}
		if s.ClearPending {
			if ui.Button(c, providerCopy("取消清除", "Cancel clear")).Height(30).Padding(0, 12).Radius(8).Border(0.5, c.Theme().Border.Alpha(0.45)).FontSize(fonttheme.InterfaceSize(12.5)).Disabled(s.Busy).Clicked() && a.OnClear != nil {
				a.OnClear(s.Ticket, false)
			}
		} else if ui.Button(c, providerCopy("清除此引用…", "Clear this reference…")).Height(30).Padding(0, 12).Radius(8).Border(0.5, c.Theme().Border.Alpha(0.45)).FontSize(fonttheme.InterfaceSize(12.5)).Disabled(locked || !row.Configured).Clicked() && a.OnClear != nil {
			a.OnClear(s.Ticket, true)
		}
	})
}
