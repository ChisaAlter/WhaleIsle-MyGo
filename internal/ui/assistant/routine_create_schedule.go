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
		return i18n.T(i18n.KeyW4AssistantRepeatsEvery1hTheOriginalHourlyIsAFi)
	case "weekdays":
		return i18n.T(i18n.KeyW4AssistantMondayToFridayDefaultingTo0900TheHos)
	case "weekly":
		return i18n.T(i18n.KeyW4AssistantTheWeekdayYouChooseDefaultingToMonday)
	case "monthly":
		return i18n.T(i18n.KeyW4AssistantTheDayOfMonthYouEnter131DefaultingT)
	case "advanced":
		return i18n.T(i18n.KeyW4AssistantFiveFieldOrderMinuteHourDayMonthWeekd)
	case "once":
		return i18n.T(i18n.KeyW4AssistantDefaultsToA30mDelayWholeNumberMHDF)
	case "interval":
		return i18n.T(i18n.KeyW4AssistantDefaultsToEvery1hWholeNumberMHDFrom)
	default:
		return i18n.T(i18n.KeyW4AssistantDefaultsTo0900DailyTheHostSchedulesI)
	}
}
