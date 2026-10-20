package assistant

import (
	"fmt"
	"github.com/egoist/mygo/ui"
	"whaleisle/internal/desktop/pluginrpc"
	"whaleisle/internal/ui/i18n"
)

const roomTextEffects = "群聊没有独立 title；现有 title 空值原样保留。群名/说明会进入后续群成员的 system prompt，群名也用于目录及托管会话展示，不是纯外观字段。Host 每次保存会先更新原 canonical presentation，再写 catalog 与审计，之后尝试 session rename；即使只改说明也走此路径。presentation 回滚或 rename 可能失败，须查看完整 warning；资料回执或匹配读回不证明展示/标题副作用完成。群名唯一性由 Host 按其 locale 规则判断，已有文本引用不会自动重写。room 分支不请求 objective requestSync；本次不发 prompt、不自动打开、不启动或停止群聊。后续讨论读取新文本时可能影响模型、工具、外部消息及费用，不保证正在进行的成员轮次立即采用。Bot ID、群 canonical session、工作区和头像保持不变。"

func roomTextOptional(s string) string {
	if s == "" {
		return i18n.T(i18n.KeyMiscAssistantEmpty2)
	}
	return s
}
func roomEditTextReview(cmd pluginrpc.DshbotRoomEditCommand) string {
	if cmd.Values.Text == nil {
		return ""
	}
	v := cmd.Values.Text
	return fmt.Sprintf(i18n.T(i18n.KeyMiscAssistantRoomTextDiff), cmd.Expected.Item.Name, v.Name, roomTextOptional(cmd.Expected.Item.Description), roomTextOptional(v.Description), roomTextEffects)
}
func roomEditTextInputs(c *ui.Context, m RoomEditModel, disabled bool) {
	if m.State.Draft == nil {
		return
	}
	d := m.State.Draft
	name := d.Name
	ui.TextInput(c.Key("room-edit-name"), &name).Label(i18n.T(i18n.KeyMiscAssistantRoomName120Utf16)).FillWidth().Disabled(disabled).OnChange(func() {
		if m.OnChange != nil {
			m.OnChange("name", name)
		}
	})
	description := d.Description
	ui.TextArea(c.Key("room-edit-description"), &description).Label(i18n.T(i18n.KeyMiscAssistantCompleteRoomDescription12000Utf16)).Lines(3, 10).FillWidth().Disabled(disabled).OnChange(func() {
		if m.OnChange != nil {
			m.OnChange("description", description)
		}
	})
}
