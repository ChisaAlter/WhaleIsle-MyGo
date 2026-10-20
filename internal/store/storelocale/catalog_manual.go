package storelocale

// Review and error messages that carry format verbs per locale.
const (
	KeyStoreAssistantReadbackDiffers       = "store.assistant.readback_differs"
	KeyStoreDshbotDeletionConfirmed        = "store.dshbot.deletion_confirmed"
	KeyStoreDshbotContextSourceUnavailable = "store.dshbot.context_source_unavailable"
	KeyStoreRemoteDeletedMachine           = "store.remote.deleted_machine"
	KeyStoreSessionChildImagesOnly         = "store.session.child_images_only"
	KeyStoreSessionPresetIdentityMismatch  = "store.session.preset_identity_mismatch"
	KeyDesktopPetCareLabel                 = "desktop.pet.care_label"
	KeyDesktopPetFedLocalFood              = "desktop.pet.fed_local_food"
	KeyDesktopPetSavedFoodOnly             = "desktop.pet.saved_food_only"
)

func init() {
	register(KeyStoreAssistantReadbackDiffers, "Host 已返回，但 %s 读回与选择不一致，请核对", "Host returned, but the %s read-back differs from the selection; verify it")
	register(KeyStoreDshbotDeletionConfirmed, "Host 已确认删除目录目标 %s；history 与 memory 保留。请检查完整 warning：停止、presentation、notepad 可能有部分失败，且不能全部恢复", "Host confirmed deleting catalog target %s; history and memory are retained. Inspect the full warning: the stop, presentation and notepad steps may have partially failed, and not everything can be recovered")
	register(KeyStoreDshbotContextSourceUnavailable, "上下文来源 %s 已不可用；明确移除后再审阅", "Context source %s is unavailable; remove it explicitly before reviewing")
	register(KeyStoreRemoteDeletedMachine, "已删除机器「%s」", "Deleted machine %q")
	register(KeyStoreSessionChildImagesOnly, "子代理仅接受图片：%s", "Child continuation accepts images only: %s")
	register(KeyStoreSessionPresetIdentityMismatch, "预设身份不一致：请求 %q", "Preset identity mismatch: requested %q")
	register(KeyDesktopPetCareLabel, "饱食 %.0f · 心情 %.0f · 亲密 %.0f", "Satiety %.0f · Mood %.0f · Bond %.0f")
	register(KeyDesktopPetFedLocalFood, "已喂 %d 本地粮，成长 +%d", "Fed %d local food, growth +%d")
	register(KeyDesktopPetSavedFoodOnly, "（刷新不可用，仅使用已结算粮）", " (saved food only)")
}
