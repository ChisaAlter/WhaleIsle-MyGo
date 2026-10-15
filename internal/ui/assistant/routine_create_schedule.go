package assistant

func routineCreateFrequencyLabel(frequency string) string {
	switch frequency {
	case "once":
		return "延迟一次 / Once"
	case "interval":
		return "固定间隔 / Interval"
	default:
		return "每日 / Daily"
	}
}

func routineCreateTimingHint(frequency string) string {
	switch frequency {
	case "once":
		return "默认延迟 30m；整数 m/h/d，范围 1 分钟至 365 天。启用后由 Host 安排一次执行；实际下次时间以 Host 返回为准。"
	case "interval":
		return "默认每 1h；整数 m/h/d，范围 1 分钟至 365 天。启用后按间隔重复执行，0 次数上限表示不限；实际下次时间以 Host 返回为准。"
	default:
		return "默认每日 09:00；按所填时区由 Host 调度，实际下次时间以 Host 返回为准。"
	}
}
