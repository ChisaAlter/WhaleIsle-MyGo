package schedule

import "whaleisle/internal/ui/i18n"

type scheduleLocaleEntry struct {
	module string
	key    string
	zh     string
	en     string
}

var scheduleLocaleCatalog = []scheduleLocaleEntry{
	{module: `dsh-client-ui-schedule`, key: `list.loading`, zh: `正在加载提醒…`, en: `Loading reminders…`},
	{module: `dsh-client-ui-schedule`, key: `list.error`, zh: `无法加载提醒。`, en: `Could not load reminders.`},
	{module: `dsh-client-ui-schedule`, key: `delete.pending`, zh: `正在删除…`, en: `Deleting…`},
	{module: `dsh-client-ui-schedule`, key: `delete.label`, zh: `删除提醒：{title}`, en: `Delete reminder: {title}`},
	{module: `dsh-client-ui-schedule`, key: `list.open`, zh: `打开提醒详情：{title}`, en: `Open reminder details: {title}`},
	{module: `dsh-client-ui-schedule`, key: `trigger.one`, zh: `{count} 个提醒`, en: `{count} reminder`},
	{module: `dsh-client-ui-schedule`, key: `trigger.other`, zh: `{count} 个提醒`, en: `{count} reminders`},
	{module: `dsh-client-ui-schedule`, key: `list.aria`, zh: `活动提醒`, en: `Active reminders`},
	{module: `dsh-client-ui-schedule`, key: `frequency.cronRule`, zh: `{rule}（{timeZone}）`, en: `{rule} ({timeZone})`},
	{module: `dsh-client-ui-schedule`, key: `cron.weekday.name`, zh: `周{weekday}`, en: `{weekday}`},
	{module: `dsh-client-ui-schedule`, key: `cron.weekday.range`, zh: `{from}至{to}`, en: `{from}–{to}`},
	{module: `dsh-client-ui-schedule`, key: `cron.months`, zh: `（{months}）`, en: ` in {months}`},
	{module: `dsh-client-ui-schedule`, key: `cron.day.weekdays`, zh: `{weekdays}{months}`, en: `{weekdays}{months}`},
	{module: `dsh-client-ui-schedule`, key: `cron.hours.range`, zh: `{from} 至 {to}`, en: `{from}–{to}`},
	{module: `dsh-client-ui-schedule`, key: `cron.hours.list`, zh: `{hours}`, en: `{hours}`},
	{module: `dsh-client-ui-schedule`, key: `cron.time.at`, zh: `{times}`, en: `at {times}`},
	{module: `dsh-client-ui-schedule`, key: `relative.future`, zh: `{value}{unit}后`, en: `in {value} {unit}`},
	{module: `dsh-client-ui-schedule.task-manager`, key: `list.loading`, zh: `正在加载任务…`, en: `Loading tasks…`},
	{module: `dsh-client-ui-schedule.task-manager`, key: `detail.close`, zh: `关闭详情`, en: `Close details`},
	{module: `dsh-client-ui-schedule.task-manager`, key: `detail.more`, zh: `自动化任务操作`, en: `Automation task actions`},
	{module: `dsh-client-ui-schedule.task-manager`, key: `detail.tabs`, zh: `任务详情视图`, en: `Task detail views`},
	{module: `dsh-client-ui-schedule.task-manager`, key: `delivery.loading`, zh: `正在加载任务运行记录…`, en: `Loading delivery records…`},
	{module: `dsh-client-ui-schedule.task-manager`, key: `delivery.cursorError`, zh: `记录已更新`, en: `Records were updated.`},
	{module: `dsh-client-ui-schedule.task-manager`, key: `detail.name`, zh: `自动化任务名称`, en: `Automation task name`},
	{module: `dsh-client-ui-schedule.task-manager`, key: `detail.instruction`, zh: `自动化任务内容`, en: `Automation task instruction`},
	{module: `dsh-client-ui-schedule.task-manager`, key: `detail.frequency`, zh: `提醒频率`, en: `Frequency`},
	{module: `dsh-client-ui-schedule.task-manager`, key: `detail.session`, zh: `关联会话`, en: `Linked session`},
	{module: `dsh-client-ui-schedule.task-manager`, key: `detail.openSession`, zh: `关联会话：打开原会话`, en: `Linked session: open original session`},
	{module: `dsh-client-ui-schedule.task-manager`, key: `detail.openSessionTitle`, zh: `关联会话 {title}：打开原会话`, en: `Linked session {title}: open original session`},
	{module: `dsh-client-ui-schedule.task-manager`, key: `frequency.every`, zh: `每 {value} {unit}`, en: `Every {value} {unit}`},
	{module: `dsh-client-ui-schedule.task-manager`, key: `frequency.cronRule`, zh: `{rule}（{timeZone}）`, en: `{rule} ({timeZone})`},
	{module: `dsh-client-ui-schedule.task-manager`, key: `cron.weekday.name`, zh: `周{weekday}`, en: `{weekday}`},
	{module: `dsh-client-ui-schedule.task-manager`, key: `cron.weekday.range`, zh: `{from}至{to}`, en: `{from}–{to}`},
	{module: `dsh-client-ui-schedule.task-manager`, key: `cron.months`, zh: `（{months}）`, en: ` in {months}`},
	{module: `dsh-client-ui-schedule.task-manager`, key: `cron.day.weekdays`, zh: `{weekdays}{months}`, en: `{weekdays}{months}`},
	{module: `dsh-client-ui-schedule.task-manager`, key: `cron.hours.range`, zh: `{from} 至 {to}`, en: `{from}–{to}`},
	{module: `dsh-client-ui-schedule.task-manager`, key: `cron.hours.list`, zh: `{hours}`, en: `{hours}`},
	{module: `dsh-client-ui-schedule.task-manager`, key: `cron.time.at`, zh: `{times}`, en: `at {times}`},
	{module: `dsh-client-ui-schedule.task-manager`, key: `relative.future`, zh: `{value}{unit}后`, en: `in {value} {unit}`},
	{module: `dsh-client-ui-schedule.task-manager`, key: `timing.zoneSearch`, zh: `搜索 UTC 偏移、IANA 标识或城市`, en: `Search by UTC offset, IANA ID, or city`},
	{module: `dsh-client-ui-schedule.task-manager`, key: `timing.interval`, zh: `重复间隔`, en: `Repeat every`},
	{module: `dsh-client-ui-schedule.task-manager`, key: `timing.intervalIncrease`, zh: `增大间隔`, en: `Increase interval`},
	{module: `dsh-client-ui-schedule.task-manager`, key: `timing.intervalDecrease`, zh: `减小间隔`, en: `Decrease interval`},
	{module: `dsh-client-ui-schedule.task-manager`, key: `timing.zoneNoStored`, zh: `单次任务只保存触发时刻，不保存时区；日期和时间按所选时区换算`, en: `A one-time task stores only its target moment, not a time zone. The date and time are interpreted in the selected time zone.`},
	{module: `dsh-client-ui-schedule.task-manager`, key: `timing.invalidInterval`, zh: `请输入至少 1 分钟的间隔`, en: `Enter an interval of at least 1 minute.`},
	{module: `dsh-client-ui-schedule.task-manager`, key: `timing.invalidInterval.hour`, zh: `请输入至少 1 小时的间隔`, en: `Enter an interval of at least 1 hour.`},
	{module: `dsh-client-ui-schedule.task-manager`, key: `timing.invalidInterval.minute`, zh: `请输入至少 1 分钟的间隔`, en: `Enter an interval of at least 1 minute.`},
	{module: `dsh-client-ui-schedule.task-manager`, key: `rule.everyMinutes`, zh: `每 N 分钟`, en: `Every N minutes`},
	{module: `dsh-client-ui-schedule.task-manager`, key: `rule.everyHours`, zh: `每 N 小时`, en: `Every N hours`},
	{module: `dsh-client-ui-schedule.task-manager`, key: `rule.everySeconds`, zh: `每 N 秒`, en: `Every N seconds`},
	{module: `dsh-client-ui-schedule.task-manager`, key: `cronForm.dates`, zh: `在以下日期`, en: `On dates`},
	{module: `dsh-client-ui-schedule.task-manager`, key: `cronForm.dateOption`, zh: `{day} 日`, en: `Day {day}`},
	{module: `dsh-client-ui-schedule.task-manager`, key: `cronForm.atMinute`, zh: `在第几分钟`, en: `At minute`},
	{module: `dsh-client-ui-schedule.task-manager`, key: `cronForm.minuteIncrease`, zh: `增大分钟`, en: `Increase minute`},
	{module: `dsh-client-ui-schedule.task-manager`, key: `cronForm.minuteDecrease`, zh: `减小分钟`, en: `Decrease minute`},
	{module: `dsh-client-ui-schedule.task-manager`, key: `rule.error.conflict`, zh: `任务在保存前已发生变化。当前显示已保存的规则，请重试`, en: `This task changed before the update was saved. The row shows the saved rule; try again.`},
	{module: `dsh-client-ui-schedule.task-manager`, key: `rule.error.unknown`, zh: `无法确认规则是否已更新。当前显示已保存的规则`, en: `Could not confirm the rule update. The row shows the saved rule.`},
	{module: `dsh-client-ui-schedule.task-manager`, key: `delete.close`, zh: `关闭删除确认`, en: `Close deletion confirmation`},
	{module: `dsh-client-ui-schedule.task-manager`, key: `delete.pending`, zh: `正在删除…`, en: `Deleting…`},
	{module: `dsh-client-ui-schedule.task-manager`, key: `card.openLabel`, zh: `打开任务详情：{title}`, en: `Open task details: {title}`},
}

func init() {
	for _, entry := range scheduleLocaleCatalog {
		i18n.Register("schedule"+"."+entry.module+"."+entry.key, entry.zh, entry.en)
	}
}

func scheduleLocaleText(key string) string {
	zh := i18n.Locale() == i18n.LocaleZH
	for _, entry := range scheduleLocaleCatalog {
		if entry.key == key || entry.module+"."+entry.key == key {
			if zh {
				return entry.zh
			}
			return entry.en
		}
	}
	return i18n.T(key)
}
