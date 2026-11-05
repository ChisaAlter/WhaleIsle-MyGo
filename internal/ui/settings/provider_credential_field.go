package settings

import (
	"fmt"
	"strings"

	"github.com/egoist/mygo/ui"
	"whaleisle/internal/desktop/credentialrefs"
	"whaleisle/internal/ui/i18n"
	"whaleisle/internal/ui/theme"
)

// providerCredentialField is the editor card's key row, as in the reference: the
// "API 密钥" label with its (?) help, the write-only input, then one hint line in
// the tone the environment check calls for. A failed write is a toast.
func providerCredentialField(c *ui.Context, a *ReferencedCredentialsAccess, namespace string, fg, muted ui.Color) {
	s := a.Snapshot
	for _, row := range s.Rows {
		if row.Namespace != namespace {
			continue
		}
		if row.ID != s.SelectedID {
			if a.OnSelect != nil && !s.Busy {
				a.OnSelect(row.ID)
			}
			return
		}
		if row.KeylessNativeAuth() {
			return
		}
		names, present := providerCredentialEnv(a, row)
		help := ""
		if len(names) > 0 {
			help = fmt.Sprintf(providerCopy("优先使用已保存的凭据。此处显示的环境后备按以下顺序排列：%s。来源依次为启动环境、$cwd/.env、$DSH_HOME/.env。", "Saved credentials are used first. Environment fallbacks shown here, in order: %s. Sources are launch environment, $cwd/.env, then $DSH_HOME/.env."), strings.Join(names, " / "))
		}
		value := s.Draft
		placeholder := providerCopy("输入 API 密钥", "Enter an API key")
		if row.Configured {
			placeholder = providerCopy("留空使用已保存凭据", "Leave blank to use the saved credential")
		} else if len(names) > 0 {
			placeholder = ""
			if !present {
				placeholder = fmt.Sprintf(i18n.T(i18n.KeySettingsModelsAPIKeyPlaceholder), names[0])
			}
		}
		ui.Column(c.Key("provider-key/" + row.ID)).FillWidth().MinWidth(0).Gap(8).Children(func() {
			modelLabelWithHelp(c, "provider-key/"+row.ID, i18n.T(i18n.KeySettingsModelsAPIKey), help)
			if apiKeyInput(c, row.ID, &value, placeholder, s.Busy || a.locked || !row.Known || !row.Writable || s.ClearPending) && a.OnEdit != nil {
				a.OnEdit(s.Ticket, value)
			}
			hint, tone := providerNativeAuthNote(row)
			if hint == "" && a.envHint != "" {
				hint, tone = a.envHint, "warn"
			}
			if hint != "" {
				if s.Draft != "" {
					tone = "hidden"
				}
				modelHint(c, hint, tone)
			}
		})
		settingsNotice(c, "provider-key/"+row.ID, "error", s.Err != "", func() string { return s.Err })
		if row.Configured && a.OnClear != nil {
			label := providerCopy("清除已保存密钥…", "Clear saved key…")
			if s.ClearPending {
				label = providerCopy("取消清除", "Cancel clear")
			}
			if ui.Button(c, label).AlignSelf(ui.Start).BorderWidth(0).Background(ui.Transparent).FontSize(theme.InterfaceSize(12)).Disabled(s.Busy || a.locked).Clicked() {
				a.OnClear(s.Ticket, !s.ClearPending)
			}
			if s.ClearPending {
				modelHint(c, providerCopy("保存后清除此引用的密钥。", "Saving will clear this credential reference."), "")
			}
		}
		return
	}
}

// providerCredentialEnv lists the environment variables this route reads its key
// from (the Models page's own answer first, else the route's native checks; OAuth-named
// references are not offered as API-key names).
func providerCredentialEnv(a *ReferencedCredentialsAccess, row credentialrefs.Row) (names []string, present bool) {
	if len(a.envNames) > 0 {
		return a.envNames, a.envPresent
	}
	if row.NamedReference || row.NativeAuth == nil {
		return nil, false
	}
	for _, check := range row.NativeChecks {
		if strings.Contains(strings.ToLower(check.Ref), "oauth") {
			continue
		}
		names = append(names, check.Ref)
		present = present || check.Known && check.Configured
	}
	return names, present
}
