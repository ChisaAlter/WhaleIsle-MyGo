package schedule

import "github.com/egoist/mygo/ui"

type scheduleChoiceOption struct{ ID, Label string }

// Keep protocol values in SelectBase items and human-readable copy in the face.
// ui.Select's string-only convenience surface otherwise exposes raw rule ids.
func scheduleChoice(c *ui.Context, key, label string, value *string, options []scheduleChoiceOption) bool {
	caption := label
	for _, option := range options {
		if option.ID == *value {
			caption = option.Label
			break
		}
	}
	parts := ui.SelectBase(c.Key(key), value)
	parts.Trigger.Label(label).FillWidth().MinWidth(0).Height(48).Padding(0).Border(.5, c.Theme().Border).BorderWidth(0, 0, .5, 0).Radius(0).Background(ui.Transparent).Children(func() {
		ui.Row(c).FillWidth().MinWidth(0).Gap(12).AlignItems(ui.Center).Children(func() { ui.Text(c, label).FontSize(14); ui.Box(c).Grow(1); scheduleValueFace(c, caption, false) })
	})
	parts.Popup(func(panel ui.Element) {
		panel.Padding(4).Background(c.Theme().Surface).BorderWidth(0).Radius(16)
		for _, option := range options {
			row := parts.Item(option.ID).Label(option.Label).Padding(6, 10).Radius(6)
			if row.Hovered() || row.FocusWithin() {
				row.Background(c.Theme().SurfaceHover)
			}
			row.Children(func() {
				ui.Text(c, option.Label).Grow(1).MinWidth(0)
				if option.ID == *value {
					ui.Text(c, "✓")
				}
			})
		}
	})
	return parts.Trigger.Changed()
}

func scheduleIntervalHint(factor int) string {
	switch factor {
	case 3600:
		return scheduleText("至少 1 小时。间隔从任务创建或上次规则变更起计算，不受时区影响。", "At least 1 hour. The interval counts from task creation or the last rule change and is independent of time zones.")
	case 60:
		return scheduleText("至少 1 分钟。间隔从任务创建或上次规则变更起计算，不受时区影响。", "At least 1 minute. The interval counts from task creation or the last rule change and is independent of time zones.")
	default:
		return scheduleText("至少 60 秒。间隔从任务创建或上次规则变更起计算，不受时区影响。", "At least 60 seconds. The interval counts from task creation or the last rule change and is independent of time zones.")
	}
}
