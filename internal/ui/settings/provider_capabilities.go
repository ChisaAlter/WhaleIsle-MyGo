package settings

import (
	"fmt"
	"slices"
	"strings"

	"github.com/egoist/mygo/ui"
	"whaleisle/internal/desktop/credentialrefs"
	"whaleisle/internal/dsh/api"
	"whaleisle/internal/ui/i18n"
	fonttheme "whaleisle/internal/ui/theme"
)

func configurationProviderMetadata(m Model, namespace string, path []string) (api.ListConfigurableProvidersResultItem, bool) {
	if m.Store != nil {
		directory, _, _ := m.Store.ConfigurableProviders()
		for _, entry := range directory {
			if entry.SettingsNs == namespace && slices.Equal(entry.SettingsPath, path) {
				return entry, true
			}
		}
	}
	return api.ListConfigurableProvidersResultItem{}, false
}

func providerEndpointHint(explicit, inherited string, env []string) string {
	if strings.TrimSpace(explicit) != "" {
		return providerCopy("当前端点：", "Current endpoint: ") + explicit
	}
	if strings.TrimSpace(inherited) != "" {
		return providerCopy("继承端点：", "Inherited endpoint: ") + inherited
	}
	if len(env) > 0 {
		return providerCopy("端点由 Host 环境配置：", "Endpoint configured by Host environment: ") + strings.Join(env, ", ")
	}
	return ""
}

func providerEndpointNotice(c *ui.Context, explicit string, entry api.ListConfigurableProvidersResultItem, muted ui.Color) {
	inherited := ""
	if entry.DefaultBaseURL != nil {
		inherited = *entry.DefaultBaseURL
	}
	if hint := providerEndpointHint(explicit, inherited, entry.BaseURLEnv); hint != "" {
		ui.Text(c, hint).FillWidth().FontSize(fonttheme.InterfaceSize(12)).TextColor(muted)
	}
}

func keylessProviderHint() string {
	return providerCopy("此供应商支持使用已保存的登录；这里不接收 API Key。", "This provider supports an existing saved sign-in; no API key is accepted here.")
}

// Endpoint variables do not authenticate a route. OAuth-named references can
// contribute a detected state without being offered as API-key names.
func providerNativeAuthHint(row credentialrefs.Row) string {
	text, _ := providerNativeAuthNote(row)
	return text
}

// providerNativeAuthNote is the hint under the key field in the reference's
// words, with its tone: amber when no environment variable was found, green when
// one was, plain while that is still unconfirmed. Like the reference it names
// only the first variable; the (?) help lists them all.
func providerNativeAuthNote(row credentialrefs.Row) (text, tone string) {
	if row.NamedReference || row.NativeAuth == nil {
		return "", ""
	}
	if row.KeylessNativeAuth() {
		return keylessProviderHint(), ""
	}
	visible := []string{}
	for _, check := range row.NativeChecks {
		if !strings.Contains(strings.ToLower(check.Ref), "oauth") {
			visible = append(visible, check.Ref)
		}
	}
	allMissing := len(row.NativeChecks) > 0
	for _, check := range row.NativeChecks {
		if check.Known && check.Configured {
			if strings.Contains(strings.ToLower(check.Ref), "oauth") {
				return providerCopy("已检测到环境变量。", "Env var detected."), "success"
			}
			return providerCopy("已检测到环境变量（", "Env var detected. (") + check.Ref + providerCopy("）。", ")"), "success"
		}
		allMissing = allMissing && check.Known && !check.Configured
	}
	if len(visible) == 0 {
		return "", ""
	}
	if allMissing {
		return fmt.Sprintf(i18n.T(i18n.KeySettingsModelsEnvHint), visible[0]), "warn"
	}
	return providerCopy("环境凭据状态尚未确认。", "Environment credential status has not been confirmed."), ""
}
