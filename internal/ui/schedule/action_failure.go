package schedule

import (
	"errors"

	schedstore "whaleisle/internal/store/schedule"
)

// The Host's recovery code determines the user's next action. Transport/storage
// diagnostics and wire codes stay out of the normal task editor.
func scheduleUpdateFailure(err error) string {
	var failure *schedstore.ResultError
	if errors.As(err, &failure) {
		switch failure.Code {
		case "schedule_conflict":
			return scheduleText("任务在编辑期间已发生变化。草稿已保留；取消后重新打开，可编辑最新规则。", "This task changed while you were editing. Your draft is retained. Cancel and reopen to edit the latest rule.")
		case "schedule_ended":
			return scheduleText("任务已结束，无法再修改运行规则。", "This task has ended. Its run rule can no longer be edited.")
		case "schedule_not_found":
			return scheduleText("任务已不可用。草稿已保留；取消可关闭编辑器。", "This task is no longer available. Your draft is retained; cancel to close the editor.")
		case "subagent_session":
			return scheduleText("该会话属于子智能体，无法接收定时提醒。", "This session belongs to a sub-agent and cannot receive reminders.")
		case "invalid_time_zone":
			return scheduleText("请选择有效的时区，例如 Asia/Shanghai。", "Choose a valid time zone, for example Asia/Shanghai.")
		case "not_future":
			return scheduleText("请选择未来的日期和时间。", "Choose a date and time in the future.")
		case "frequency_too_high":
			return scheduleText("请输入至少 60 秒的间隔。", "Enter an interval of at least 60 seconds.")
		case "invalid_prompt", "invalid_selector", "invalid_rule", "time_out_of_range":
			return scheduleText("请检查任务内容及运行时间。草稿已保留。", "Check the task instruction and run time. Your draft is retained.")
		}
	}
	return scheduleText("无法确认更改是否已保存。草稿已保留，请先刷新任务，再决定是否重试。", "Could not confirm the update. Your draft is retained. Refresh the task before retrying.")
}
