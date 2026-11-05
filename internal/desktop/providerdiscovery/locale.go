package providerdiscovery

import (
	"fmt"
	"strings"

	"whaleisle/internal/ui/i18n"
)

var messages = [][2]string{
	{"Discovery was cancelled because the Models page was hidden. The temporary key was discarded.", "模型页已关闭，探测已取消，临时密钥已清除。"},
	{"Discovery was cancelled. The temporary key was discarded.", "探测已取消，临时密钥已清除。"},
	{"No models were returned. No settings were changed.", "未返回模型，设置未更改。"},
	{"Selected models were added to the draft. Existing model fields were preserved. Review and save to apply.", "所选模型已加入草稿，已有模型字段已保留；审阅并保存后生效。"},
	{"Select at least one model before saving this discovery draft.", "保存探测草稿前请至少选择一个模型。"},
	{"Provider settings changed or became read-only. Discard and refresh before saving; your draft was kept.", "供应商设置已更改或变为只读；草稿已保留，请放弃并刷新后再保存。"},
	{"The settings write failed or conflicted. Your draft was kept; refresh after resolving a conflict.", "写入设置失败或存在冲突；草稿已保留，请解决冲突后刷新。"},
	{"The Host accepted the write, but its result could not be verified. Refresh before editing again.", "Host 已接受写入，但无法核实结果；请刷新后继续编辑。"},
	{"Provider model settings were saved and verified. The temporary discovery key was not stored.", "供应商模型设置已保存并核实，临时探测密钥未保存。"},
	{"Enter a temporary API key, not a masked credential placeholder.", "请输入临时 API 密钥，而非已隐藏凭据的占位符。"},
	{"The temporary key contains unsupported HTTP header characters.", "临时密钥包含不受支持的 HTTP 标头字符。"},
	{"Host model discovery failed. Check the provider, endpoint, protocol, and credential, or enter models manually.", "Host 模型探测失败；请检查供应商、地址、协议和凭据，或手动填写模型。"},
	{"Unable to read provider settings. Reconnect and refresh.", "无法读取供应商设置，请重新连接并刷新。"},
	{"Unable to read the Host provider directory.", "无法读取 Host 供应商目录。"},
	{"The Host schema does not expose the supported provider model fields.", "Host 架构未提供此编辑器支持的供应商模型字段。"},
	{"This provider uses modelOverrides. Edit those overrides in provider settings before replacing its model list.", "此供应商使用 modelOverrides；替换模型列表前，请在供应商设置中编辑覆盖项。"},
	{"This provider has hidden values inside the edited fields. Use the owning settings editor to preserve them.", "所编辑字段内含隐藏值，请使用所属设置编辑器以保留这些值。"},
	{"Model discovery is unavailable while another operation is running.", "另一项供应商操作仍在进行，请结束后再获取模型。"},
	{"context canceled", "模型获取已取消。"},
}

func init() {
	for index, row := range messages {
		i18n.Register(fmt.Sprintf("provider-discovery.message.%d", index), row[1], row[0])
	}
	i18n.Register("provider-discovery.candidates", "%s：%d 个候选模型。请选择模型、加入草稿，再审阅并保存。", "%s: %d candidates. Select models, add them to the draft, then review and save.")
}

// Message resolves persisted controller status at render time, so switching the
// active language also updates an existing error or discovery result.
func Message(value string) string {
	for index, row := range messages {
		if value == row[0] {
			return i18n.T(fmt.Sprintf("provider-discovery.message.%d", index))
		}
	}
	const suffix = " candidates. Select models, add them to the draft, then review and save."
	if text, ok := strings.CutSuffix(value, suffix); ok {
		if split := strings.LastIndex(text, ": "); split >= 0 {
			var count int
			if _, err := fmt.Sscanf(text[split+2:], "%d", &count); err == nil {
				return fmt.Sprintf(i18n.T("provider-discovery.candidates"), text[:split], count)
			}
		}
	}
	return value
}
