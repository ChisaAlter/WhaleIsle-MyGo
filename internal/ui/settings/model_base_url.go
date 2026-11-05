package settings

import (
	"fmt"
	"strings"

	"github.com/egoist/mygo/ui"

	store "whaleisle/internal/store/settings"
	"whaleisle/internal/ui/theme"
)

// modelBaseURLField is the editor's "API 地址" row: a label, the input (clearing
// it drops the override, so there is no separate inherit button) and, for the
// DeepSeek routes, the hint that names the protocol the address must speak.
func modelBaseURLField(c *ui.Context, binding *store.NamespaceBinding, snap store.ViewState, f store.Field, disabled bool) {
	value, placeholder := snap.Drafts[f.Key], ""
	if display, inherited := binding.ProviderInput([]string{f.Key}, value); inherited || snap.Resets[f.Key] {
		placeholder, value = value, ""
	} else if display != nil {
		value = fmt.Sprint(display)
	}
	deepseek := strings.HasPrefix(snap.SelectedNS, "llm-deepseek")
	if placeholder == "" && deepseek {
		placeholder = providerCopy("https://api.deepseek.com/anthropic（默认使用环境 DEEPSEEK_BASE_URL）", "https://api.deepseek.com/anthropic (defaults to the DEEPSEEK_BASE_URL environment)")
	}
	label := configurationLabel(f.Key)
	ui.Column(c.Key("model-scalar/" + f.Key)).FillWidth().MinWidth(0).Gap(8).Children(func() {
		modelFieldLabel(c, label)
		input := ui.TextInput(c, &value).Label(label).Placeholder(placeholder).FillWidth().Height(36).Radius(8).
			Border(.667, modelInputBorder(c)).Background(modelWell(c).Alpha(.8)).FontSize(theme.InterfaceSize(14)).ReadOnly(disabled)
		if input.Changed() {
			if value == "" {
				binding.Reset(f.Key)
			} else {
				binding.SetDraft(f.Key, value)
			}
		}
		if deepseek {
			modelHint(c, providerCopy("请填写兼容 Anthropic Messages 协议的 API 地址。", "Enter an API address that speaks the Anthropic Messages protocol."), "")
		}
	})
}
