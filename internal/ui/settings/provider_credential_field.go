package settings

import (
	"strings"

	"whaleisle/internal/desktop/credentialrefs"
	"whaleisle/internal/ui/i18n"

	"github.com/egoist/mygo/ui"
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
		help := providerCredentialHelp(names)
		value := s.Draft
		placeholder := providerCopy("输入 API 密钥", "Enter an API key")
		if row.Configured {
			placeholder = providerCopy("留空使用已保存凭据", "Leave blank to use the saved credential")
		} else if len(names) > 0 {
			placeholder = ""
			if !present {
				placeholder = i18n.T(i18n.KeySettingsModelsAPIKeyPlaceholder)
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
		settingsAlert(c, "provider-key/"+row.ID, s.Err != "", providerCopy("没能保存这个提供商的密钥，请重试。", "The provider key was not saved. Try again."), s.Err)
		if row.Configured && a.OnClear != nil {
			var clearBtn ui.Element
			if s.ClearPending {
				clearBtn = kGhost(c, providerCopy("取消清除", "Cancel clear"), s.Busy || a.locked)
			} else {
				clearBtn = kDanger(c, providerCopy("清除已保存密钥…", "Clear saved key…"), s.Busy || a.locked)
			}
			if clearBtn.AlignSelf(ui.Start).Clicked() {
				a.OnClear(s.Ticket, !s.ClearPending)
			}
			if s.ClearPending {
				modelHint(c, providerCopy("保存后清除此引用的密钥。", "Saving will clear this credential reference."), "")
			}
		}
		return
	}
}

// providerCredentialHelp answers what the field does in one sentence, then puts
// the variable names behind the ? disclosure: a person who supplies the key
// through the launching environment needs the names, and nobody reads a list of
// names as the answer to "what is this field".
func providerCredentialHelp(names []string) string {
	if len(names) == 0 {
		return ""
	}
	return providerCopy("优先使用已保存的凭据；没有时按本机环境变量和 .env 文件补齐。", "Saved credentials are used first; otherwise this machine's environment variables and .env files fill in.") +
		"\n\n" + providerCopy("详情：按此顺序读取 ", "Details: read in this order ") + strings.Join(names, " / ") +
		providerCopy("；来源依次为启动环境、当前目录的 .env 和用户目录的 .env。", "; sources are the launching environment, then .env in the working directory, then .env in the home directory.") // detail-ok: ? help tooltip
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
