package schedule

import (
	"github.com/egoist/mygo/ui"
	"whaleisle/internal/ui/coss"
)

type scheduleChoiceOption struct{ ID, Label string }

// Keep protocol values in Select options and human-readable copy in their labels.
func scheduleChoice(c *ui.Context, key, label string, value *string, options []scheduleChoiceOption) bool {
	opts := make([]coss.SelectOption, len(options))
	for i, option := range options {
		opts[i] = coss.SelectOption{Value: option.ID, Label: option.Label}
	}
	var sel ui.Element
	scheduleRuleRow(c, label, false, func() {
		sel = coss.Select(c, value, opts, coss.SelectOptions{
			Key:   key,
			Label: label,
		})
	})
	return sel.Changed()
}

func scheduleIntervalHint(factor int) string {
	switch factor {
	case 3600:
		return scheduleText("至少 1 小时。间隔自任务创建或上次修改规则时起计，不受时区影响", "At least 1 hour. The interval counts from task creation or the last rule change and is independent of time zones.")
	case 60:
		return scheduleText("至少 1 分钟。间隔自任务创建或上次修改规则时起计，不受时区影响", "At least 1 minute. The interval counts from task creation or the last rule change and is independent of time zones.")
	default:
		return scheduleText("至少 60 秒。间隔自任务创建或上次修改规则时起计，不受时区影响", "At least 60 seconds. The interval counts from task creation or the last rule change and is independent of time zones.")
	}
}
