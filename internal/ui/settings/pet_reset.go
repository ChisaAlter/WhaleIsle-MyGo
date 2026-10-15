package settings

import (
	"github.com/egoist/mygo"
	"whaleisle/internal/desktop/pet"
)

// Opening or cancelling the dialog leaves all settings and drafts unchanged.
// The scope names the native fields, rather than claiming a whole-pet reset.
func confirmResetPetAppearanceBehavior(p *pet.Companion) {
	answer, err := mygo.Dialog.Message(mygo.MessageOptions{
		Type:          mygo.MessageQuestion,
		Title:         "恢复外观与行为默认 / Reset appearance and behavior",
		Message:       "立即恢复外观与行为，并关闭桌宠看看？ / Restore appearance and behavior and turn Pet Look off?",
		Detail:        "大小 1、透明度 100%、性格 natural、活跃程度 balanced；自言自语、自主移动、聊天开启；锁定拖动、Shift 拖动、省电、点击音效关闭。\nScale 1, opacity 100%, natural personality, balanced activity; self-talk, wandering and chat on; position lock, Shift-to-drag, power saving and click sound off.\n\n同时清空桌宠专属视觉 provider/model，关闭看看。保留显示开关、皮肤、THA4、位置和养成存档。全局视觉模型设置不变。\nThe pet-specific vision provider/model are cleared, turning Look off. Visibility, skin, THA4, position, care and growth remain saved. Global vision settings are unchanged.",
		Buttons:       []string{"取消 / Cancel", "确认恢复并保存 / Restore and save"},
		DefaultButton: 0, CancelButton: 0,
	})
	if err != nil {
		p.Error = "确认窗口不可用，设置未更改 / Confirmation unavailable; settings unchanged"
		return
	}
	if answer.Button == 1 {
		p.ResetAppearanceBehavior()
	}
}
