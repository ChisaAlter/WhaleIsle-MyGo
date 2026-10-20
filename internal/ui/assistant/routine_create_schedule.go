package assistant

import (
	"whaleisle/internal/ui/i18n"
)

func routineCreateFrequencyLabel(frequency string) string {
	switch frequency {
	case "hourly":
		return i18n.T(i18n.KeyMiscAssistantHourly)
	case "weekdays":
		return i18n.T(i18n.KeyMiscAssistantWeekdays)
	case "weekly":
		return i18n.T(i18n.KeyMiscAssistantWeekly)
	case "monthly":
		return i18n.T(i18n.KeyMiscAssistantMonthly)
	case "advanced":
		return i18n.T(i18n.KeyMiscAssistantAdvanced)
	case "once":
		return i18n.T(i18n.KeyMiscAssistantOnce)
	case "interval":
		return i18n.T(i18n.KeyMiscAssistantInterval)
	default:
		return i18n.T(i18n.KeyMiscAssistantDaily)
	}
}

func routineCreateTimingHint(frequency string) string {
	switch frequency {
	case "hourly":
		return "每 1h 重复执行（原版 hourly 为固定间隔）；实际下次时间以 Host 返回为准。"
	case "weekdays":
		return "周一至周五，默认 09:00；依填写时区由 Host 调度，不在本机推算夏令时变化。"
	case "weekly":
		return "每周所选日，默认周一 09:00；依填写时区由 Host 调度，实际下次时间以 Host 返回为准。"
	case "monthly":
		return "每月所填日期（1–31），默认 1 日 09:00；月末与夏令时按 Host 解析，实际下次时间以 Host 返回为准。"
	case "advanced":
		return "五字段顺序：分钟 小时 日 月 星期。这里只检查结构；确认保存时由 Host 验证完整 Cron 语法。月末、夏令时和实际下次时间以 Host 返回为准。"
	case "once":
		return "默认延迟 30m；整数 m/h/d，范围 1 分钟至 365 天。启用后由 Host 安排一次执行；实际下次时间以 Host 返回为准。"
	case "interval":
		return "默认每 1h；整数 m/h/d，范围 1 分钟至 365 天。启用后按间隔重复执行，0 次数上限表示不限；实际下次时间以 Host 返回为准。"
	default:
		return "默认每日 09:00；按所填时区由 Host 调度，实际下次时间以 Host 返回为准。"
	}
}
