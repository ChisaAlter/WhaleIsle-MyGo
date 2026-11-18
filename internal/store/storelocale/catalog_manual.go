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

	KeyStoreComponentsUninstallRecordSaveFailed = "store.components.uninstall_record_save_failed"
	KeyStoreComponentsStopRecordSaveFailed      = "store.components.stop_record_save_failed"
	KeyStoreComponentsRegistryUnreadable        = "store.components.registry_unreadable"
	KeyStoreWhalebridgeRollbackRetained         = "store.whalebridge.rollback_retained"
	KeyStoreWhalebridgeRollbackRecordFailed     = "store.whalebridge.rollback_record_failed"

	KeyStoreAssistantHistorySaveFailed = "store.assistant.history_save_failed"
	KeyStoreAssistantModeSaveFailed    = "store.assistant.mode_save_failed"
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
	register(KeyStoreComponentsUninstallRecordSaveFailed, "组件文件已删除，但安装记录未能保存；重启后可能仍显示该组件，请再次卸载", "The component files were removed, but its installation record could not be saved. It may still be listed after restart; uninstall it again")
	register(KeyStoreComponentsStopRecordSaveFailed, "组件已停止，但状态记录未能保存；重启后可能仍显示为运行中", "The component stopped, but its state record could not be saved. It may still show as running after restart")
	register(KeyStoreComponentsRegistryUnreadable, "组件注册表无法读取，且未能备份，已停止写入以免覆盖原文件；请检查后再试", "The component registry could not be read and could not be set aside, so nothing was written over it. Check the registry file and retry")
	register(KeyStoreWhalebridgeRollbackRetained, "卸载未完成，自动恢复也失败；组件文件已保留在 %s，请重试卸载", "The uninstall did not finish and the automatic recovery failed. The component files were kept at %s; retry the uninstall")
	register(KeyStoreWhalebridgeRollbackRecordFailed, "版本切换失败，且安装记录未能恢复为 %s；请重试切换以修正记录", "The version switch failed and the installation record could not be restored to %s; retry the switch to correct it")
	register(KeyStoreAssistantHistorySaveFailed, "助手历史未能写入磁盘；这些记录在重启后会丢失", "The assistant history could not be written to disk; these turns are lost after restart")
	register(KeyStoreAssistantModeSaveFailed, "助手模式未能保存；重启后会回到之前的模式", "The assistant mode could not be saved; the previous mode returns after restart")
}
