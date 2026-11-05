package settings

import (
	"whaleisle/internal/desktop/pet"
	"whaleisle/internal/ui/theme"
)

// petResetQuestion is the in-app confirmation for restoring the pet's
// appearance and behavior defaults. Opening or cancelling it leaves all
// settings and drafts unchanged. The scope names the native fields, rather than
// claiming a whole-pet reset.
func petResetQuestion(p *pet.Companion, open *bool, colors theme.Colors) ConfirmOptions {
	if p == nil || open == nil || !*open {
		return ConfirmOptions{}
	}
	return ConfirmOptions{
		Colors:       colors,
		Title:        providerCopy("恢复桌面形象与行为默认", "Reset desktop appearance & behavior"),
		Body:         providerCopy("立即恢复外观与行为，并关闭桌宠看看？", "Restore appearance and behavior and turn Pet Look off?"),
		Detail:       providerCopy("大小 1、透明度 100%、性格 natural、活跃程度 balanced；自言自语、自主移动、聊天开启；锁定拖动、Shift 拖动、省电、点击音效关闭。", "Scale 1, opacity 100%, natural personality, balanced activity; self-talk, wandering and chat on; position lock, Shift-to-drag, power saving and click sound off."),
		ConfirmLabel: providerCopy("确认恢复并保存", "Restore and save"),
		CancelLabel:  providerCopy("取消", "Cancel"),
		OnConfirm: func() {
			*open = false
			p.ResetAppearanceBehavior()
		},
		OnCancel: func() { *open = false },
	}
}
