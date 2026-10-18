package assistant

const roomDeleteConfirmLabel = "永久解散此群聊及所列关联数据 / Permanently disband reviewed room"
const roomDeleteEffectsWarning = "危险操作：删除本群聊目录资料，删除归属于这个 room ID 的例行任务及 notepad 整个文件，取消涉及此 room ID 的未完成任务，并从存活资料的 inbox 清理相关任务/例行任务记录；追加审计且最多保留 200 条。成员 Bot 目录保留，不删除成员 Bot；成员各自 canonical 会话保留，本群隐藏的成员会话也完整保留。仅释放本群 canonical presentation，不释放本群隐藏成员会话的 presentation。Host 会中止本群当前成员轮次、推进轮次 epoch 并尝试取消群代理；这些运行影响可能发生在后续目录写入失败之前。notepad 直接删除、无回收站或 Undo；其执行时当前全部内容可能不可恢复。保留 history 和 bot/user memory 不保证可恢复，不能恢复已删群配置、例行任务、任务状态、inbox、审计修剪或 notepad。预览与删除非原子、未共用同一锁：动态任务/inbox/notepad 可继续变化，Host 按执行时当前关联状态处理。失败仅尝试恢复 presentation，不承诺全部回滚；未知回包不代表无副作用，禁止自动重试。"

func botDeleteReviewedLabels(kind string) (effects, confirm, cancel, idLabel string) {
	if kind == "room" {
		return roomDeleteEffectsWarning, roomDeleteConfirmLabel, "取消，保留此群聊 / Cancel and keep room", "输入上方精确群聊 ID / Type exact room ID"
	}
	return botDeleteEffectsWarning, botDeleteConfirmLabel, "取消，保留此 Bot / Cancel and keep Bot", "输入上方精确 Bot ID / Type exact Bot ID"
}
