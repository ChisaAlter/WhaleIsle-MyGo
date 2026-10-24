package chat

import (
	"bytes"
	"encoding/json"
	"fmt"
	"math"
	"strconv"
	"strings"
	"time"

	"whaleisle/internal/store/session"
	"whaleisle/internal/ui/i18n"
)

// toolDetailsModel is the installed goal and reminder details card's display
// data. Recorded objectives, prompts and blocker messages retain their spelling
// and confer no actions.
type toolDetailsModel struct {
	Items []toolDetailItem
	Empty string
	// Summary replaces the first item's title in the collapsed row, as the
	// installed reminder list does with its count.
	Summary string
}

type toolDetailItem struct {
	Title  string
	Fields []toolDetailField
}

type toolDetailField struct {
	Label, Value string
}

// toolDetailsModelFor derives the create_goal/get_goal/update_goal subset and the
// installed reminder families of the details card. Root results borrow their
// retained raw head; nested results embed their own call even in a result-only
// history window.
func toolDetailsModelFor(call *session.ToolCallView) *toolDetailsModel {
	if call == nil || call.Result == nil || call.State != "ok" {
		return nil
	}
	result, ok := terminalModelReadResult(call.Result)
	if !ok || result.failed {
		return nil
	}
	name, ok := toolDetailsModelCallName(call, result.nested)
	if !ok {
		return nil
	}
	text, ok := terminalModelSingleText(result.content)
	if !ok {
		return nil
	}
	value := json.RawMessage(text)
	switch name {
	case "create_goal", "get_goal", "update_goal":
		return toolDetailsModelGoal(value)
	case "schedule_create", "schedule_update":
		item, ok := toolDetailsModelScheduleItem(value)
		if !ok {
			return nil
		}
		return &toolDetailsModel{Items: []toolDetailItem{item}}
	case "schedule_list":
		return toolDetailsModelScheduleList(value)
	case "schedule_delete":
		return toolDetailsModelScheduleDeleted(value)
	default:
		return nil
	}
}

// toolDetailsModelGoal is the installed goalDetail. An empty goal precedes
// activation validation, and failure of any recorded field rejects the record.
func toolDetailsModelGoal(value json.RawMessage) *toolDetailsModel {
	goalValue, ok := terminalModelObject(value)
	if !ok {
		return nil
	}
	// This precedes activation validation in the installed goalDetail.
	if bytes.Equal(bytes.TrimSpace(goalValue["goal"]), []byte("null")) {
		return &toolDetailsModel{Items: []toolDetailItem{}, Empty: i18n.T(i18n.KeyToolDetailGoalEmpty)}
	}
	goal, ok := terminalModelObject(goalValue["goal"])
	if !ok {
		return nil
	}
	id, validID := terminalModelString(goal["id"])
	objective, validObjective := terminalModelString(goal["objective"])
	_, validRevision := toolDetailsModelCount(goal["revision"])
	rounds, validRounds := toolDetailsModelCount(goal["roundsStarted"])
	maximum, validMaximum := toolDetailsModelCount(goal["maxGoalRounds"])
	if !validID || terminalModelBlank(id) || !validObjective || terminalModelBlank(objective) || !validRevision || !validRounds || !validMaximum {
		return nil
	}
	phase, _ := terminalModelString(goal["phase"])
	stateKey := ""
	switch phase {
	case "active":
		stateKey = i18n.KeyToolDetailGoalActive
	case "paused":
		stateKey = i18n.KeyToolDetailGoalPaused
	case "blocked":
		stateKey = i18n.KeyToolDetailGoalBlocked
	case "complete":
		stateKey = i18n.KeyToolDetailGoalComplete
	default:
		return nil
	}
	activation, _ := terminalModelString(goalValue["activation"])
	if activation != "armed" && activation != "disarmed" {
		return nil
	}
	if phase == "active" && activation == "disarmed" {
		stateKey = i18n.KeyToolDetailGoalDisarmed
	}
	fields := []toolDetailField{
		{Label: i18n.T(i18n.KeyToolDetailState), Value: i18n.T(stateKey)},
		{Label: i18n.T(i18n.KeyToolDetailGoalRounds), Value: rounds + " / " + maximum},
	}
	if raw, exists := goal["blockedReason"]; exists {
		reason, validReason := terminalModelObject(raw)
		code, validCode := terminalModelString(reason["code"])
		message, validMessage := terminalModelString(reason["message"])
		if !validReason || !validCode || terminalModelBlank(code) || !validMessage || terminalModelBlank(message) {
			return nil
		}
		fields = append(fields, toolDetailField{Label: i18n.T(i18n.KeyToolDetailGoalReason), Value: message})
	}
	return &toolDetailsModel{Items: []toolDetailItem{{Title: objective, Fields: fields}}}
}

// ISO-weekday locale keys, Monday first: a stored weekly rule names days 1-7.
var toolDetailsModelScheduleWeekdayKeys = [...]string{
	i18n.KeyToolDetailWeekday1, i18n.KeyToolDetailWeekday2, i18n.KeyToolDetailWeekday3, i18n.KeyToolDetailWeekday4,
	i18n.KeyToolDetailWeekday5, i18n.KeyToolDetailWeekday6, i18n.KeyToolDetailWeekday7,
}

// toolDetailsModelScheduleCount accepts exactly the installed count(): a
// nonnegative safe integer after JSON's IEEE-754 rounding.
func toolDetailsModelScheduleCount(raw json.RawMessage) (float64, bool) {
	number, ok := toolReadModelInteger(raw, 0)
	if !ok || number > 9007199254740991 {
		return 0, false
	}
	return number, true
}

// toolDetailsModelScheduleInterval chooses the coarsest exact unit available,
// mirroring the installed remainder checks and integer division.
func toolDetailsModelScheduleInterval(seconds float64) string {
	switch {
	case math.Mod(seconds, 86400) == 0:
		return fmt.Sprintf(i18n.T(i18n.KeyToolDetailDays), int64(seconds/86400))
	case math.Mod(seconds, 3600) == 0:
		return fmt.Sprintf(i18n.T(i18n.KeyToolDetailHours), int64(seconds/3600))
	case math.Mod(seconds, 60) == 0:
		return fmt.Sprintf(i18n.T(i18n.KeyToolDetailMinutes), int64(seconds/60))
	default:
		return fmt.Sprintf(i18n.T(i18n.KeyToolDetailSeconds), int64(seconds))
	}
}

// toolDetailsModelScheduleShortTime drops a stored wall clock's zero seconds.
func toolDetailsModelScheduleShortTime(value string) string {
	return strings.TrimSuffix(strings.TrimSuffix(value, ":00.000"), ".000")
}

// toolDetailsModelScheduleInstant accepts exactly the instants that round-trip
// through new Date(s).toISOString() === s: four-digit-year ISO with milliseconds
// and Z. Extended-year forms are outside Go's parser and are documented limits.
const toolDetailsModelScheduleInstantLayout = "2006-01-02T15:04:05.000Z07:00"

func toolDetailsModelScheduleInstant(value string) (time.Time, bool) {
	instant, err := time.Parse(toolDetailsModelScheduleInstantLayout, value)
	// Formatting the UTC instant rejects every non-Z or non-UTC-offset spelling
	// whose wall clock only looks equal to the recorded string.
	if err != nil || instant.UTC().Format(toolDetailsModelScheduleInstantLayout) != value {
		return time.Time{}, false
	}
	return instant.UTC(), true
}

// toolDetailsModelScheduleWeekdays localizes a stored weekly rule, rejecting an
// empty set and any day outside the Monday-first 1-7 range.
func toolDetailsModelScheduleWeekdays(raw json.RawMessage) (string, bool) {
	var days []json.RawMessage
	if json.Unmarshal(raw, &days) != nil || len(days) == 0 {
		return "", false
	}
	labels := make([]string, 0, len(days))
	for _, rawDay := range days {
		day, ok := toolReadModelInteger(rawDay, 1)
		if !ok || day > 7 {
			return "", false
		}
		labels = append(labels, i18n.T(toolDetailsModelScheduleWeekdayKeys[int(day)-1]))
	}
	return strings.Join(labels, i18n.T(i18n.KeyToolDetailWeekdayJoin)), true
}

// toolDetailsModelScheduleItem mirrors the installed scheduleItem: only a fully
// valid recorded reminder yields a row, and the raw prompt/title spelling is
// retained while blank checks use the JavaScript trim set.
func toolDetailsModelScheduleItem(value json.RawMessage) (toolDetailItem, bool) {
	var item toolDetailItem
	record, ok := terminalModelObject(value)
	if !ok {
		return item, false
	}
	id, validID := terminalModelString(record["id"])
	prompt, validPrompt := terminalModelString(record["prompt"])
	if !validID || terminalModelBlank(id) || !validPrompt || terminalModelBlank(prompt) {
		return item, false
	}
	// `host` is the current Host-delivered view; `session-local` stays readable
	// for results recorded before the Host storage domain owned delivery.
	mode, _ := terminalModelString(record["deliveryMode"])
	if mode != "host" && mode != "session-local" {
		return item, false
	}
	state, _ := terminalModelString(record["state"])
	if state != "scheduled" && state != "overdue" {
		return item, false
	}
	when, validWhen := terminalModelString(record["scheduledAt"])
	if !validWhen {
		return item, false
	}
	instant, validInstant := toolDetailsModelScheduleInstant(when)
	if !validInstant {
		return item, false
	}
	kind, _ := terminalModelString(record["kind"])
	var frequency string
	switch kind {
	case "at":
		frequency = i18n.T(i18n.KeyToolDetailScheduleOnce)
	case "after":
		seconds, validSeconds := toolDetailsModelScheduleCount(record["afterSeconds"])
		if !validSeconds || seconds == 0 {
			return item, false
		}
		frequency = i18n.T(i18n.KeyToolDetailScheduleOnce)
	case "every":
		seconds, validSeconds := toolDetailsModelScheduleCount(record["everySeconds"])
		if !validSeconds || seconds == 0 {
			return item, false
		}
		frequency = fmt.Sprintf(i18n.T(i18n.KeyToolDetailScheduleEvery), toolDetailsModelScheduleInterval(seconds))
	case "daily":
		clock, validClock := terminalModelString(record["time"])
		zone, validZone := terminalModelString(record["timeZone"])
		if !validClock || terminalModelBlank(clock) || !validZone || terminalModelBlank(zone) {
			return item, false
		}
		frequency = fmt.Sprintf(i18n.T(i18n.KeyToolDetailScheduleDaily), toolDetailsModelScheduleShortTime(clock), zone)
	case "weekly":
		clock, validClock := terminalModelString(record["time"])
		zone, validZone := terminalModelString(record["timeZone"])
		days, validDays := toolDetailsModelScheduleWeekdays(record["weekdays"])
		if !validClock || terminalModelBlank(clock) || !validZone || terminalModelBlank(zone) || !validDays {
			return item, false
		}
		frequency = fmt.Sprintf(i18n.T(i18n.KeyToolDetailScheduleWeekly), days, toolDetailsModelScheduleShortTime(clock), zone)
	case "cron":
		expression, validExpression := terminalModelString(record["expression"])
		zone, validZone := terminalModelString(record["timeZone"])
		if !validExpression || terminalModelBlank(expression) || !validZone || terminalModelBlank(zone) {
			return item, false
		}
		frequency = fmt.Sprintf(i18n.T(i18n.KeyToolDetailScheduleCron), expression, zone)
	default:
		return item, false
	}
	// The Host view carries a title; a recorded result without one keeps its
	// instruction verbatim.
	title := prompt
	if rawTitle, validTitle := terminalModelString(record["title"]); validTitle && !terminalModelBlank(rawTitle) {
		title = rawTitle
	}
	stateKey := i18n.KeyToolDetailScheduleScheduled
	if state == "overdue" {
		stateKey = i18n.KeyToolDetailScheduleOverdue
	}
	dateText, formatted := toolDetailsModelScheduleDateText(instant)
	if !formatted {
		dateText = when
	}
	return toolDetailItem{Title: title, Fields: []toolDetailField{
		{Label: i18n.T(i18n.KeyToolDetailScheduleWhen), Value: dateText},
		{Label: i18n.T(i18n.KeyToolDetailScheduleFrequency), Value: frequency},
		{Label: i18n.T(i18n.KeyToolDetailState), Value: i18n.T(stateKey)},
	}}, true
}

// toolDetailsModelScheduleList requires every entry of a recorded JSON array to
// be a valid reminder; the count labels the collapsed row even when empty.
func toolDetailsModelScheduleList(value json.RawMessage) *toolDetailsModel {
	trimmed := bytes.TrimSpace(value)
	if len(trimmed) == 0 || trimmed[0] != '[' {
		return nil
	}
	var entries []json.RawMessage
	if json.Unmarshal(trimmed, &entries) != nil {
		return nil
	}
	items := make([]toolDetailItem, 0, len(entries))
	for _, entry := range entries {
		item, ok := toolDetailsModelScheduleItem(entry)
		if !ok {
			return nil
		}
		items = append(items, item)
	}
	return &toolDetailsModel{
		Items:   items,
		Summary: fmt.Sprintf(i18n.T(i18n.KeyToolDetailScheduleCount), len(items)),
		Empty:   i18n.T(i18n.KeyToolDetailScheduleEmpty),
	}
}

// toolDetailsModelScheduleDeleted shows the installed deletion receipt, whose
// title is the retained reminder id.
func toolDetailsModelScheduleDeleted(value json.RawMessage) *toolDetailsModel {
	record, ok := terminalModelObject(value)
	if !ok {
		return nil
	}
	id, validID := terminalModelString(record["id"])
	if !validID || terminalModelBlank(id) {
		return nil
	}
	if deleted, validDeleted := terminalModelBool(record["deleted"]); !validDeleted || !deleted {
		return nil
	}
	return &toolDetailsModel{Items: []toolDetailItem{{Title: id, Fields: []toolDetailField{
		{Label: i18n.T(i18n.KeyToolDetailState), Value: i18n.T(i18n.KeyToolDetailScheduleDeleted)},
	}}}}
}

// toolDetailsModelScheduleDateText composes the installed Intl.DateTimeFormat
// shapes for the two shipped locales in the viewer's local zone:
//
//	en:    "Oct 24, 2026, 03:59:30 PM GMT+8"
//	zh-CN: "2026年10月24日 GMT+8 15:59:30"
//
// The shapes were recorded from Node 24 `new Intl.DateTimeFormat(locale, {
// year:'numeric', month:'short', day:'numeric', hour:'2-digit', minute:'2-digit',
// second:'2-digit', timeZoneName:'short' })` and are reproduced byte-for-byte by
// the tests for a fixed +08:00 zone. En uses 12-hour clock with zero-padded hour,
// zh uses zero-padded 24-hour clock with the zone before the time, and both keep
// an unpadded day and year.
//
// Limits: ICU's letter zone abbreviations (EDT, JST, ...) become their numeric
// GMT offset, ICU's "UTC" is kept but other zero-offset zones print "GMT", and
// Gregorian year 0 prints as year 1 because Intl omits an era (years below 0
// never pass the installed instant round trip in Go). Other locales fall back to
// the raw recorded instant.
func toolDetailsModelScheduleDateText(instant time.Time) (string, bool) {
	local := instant.In(time.Local)
	zone := toolDetailsModelScheduleZone(local)
	year := local.Year()
	if year == 0 {
		year = 1
	}
	switch i18n.Locale() {
	case i18n.LocaleEN:
		hour := local.Hour() % 12
		if hour == 0 {
			hour = 12
		}
		period := "AM"
		if local.Hour() >= 12 {
			period = "PM"
		}
		return fmt.Sprintf("%s %d, %d, %02d:%02d:%02d %s %s",
			toolDetailsModelScheduleMonths[local.Month()-1], local.Day(), year,
			hour, local.Minute(), local.Second(), period, zone), true
	case i18n.LocaleZH:
		return fmt.Sprintf("%d年%d月%d日 %s %02d:%02d:%02d",
			year, local.Month(), local.Day(), zone, local.Hour(), local.Minute(), local.Second()), true
	default:
		return "", false
	}
}

// toolDetailsModelScheduleMonths is the CLDR en short month vocabulary.
var toolDetailsModelScheduleMonths = [...]string{
	"Jan", "Feb", "Mar", "Apr", "May", "Jun", "Jul", "Aug", "Sep", "Oct", "Nov", "Dec",
}

// toolDetailsModelScheduleZone builds the installed short zone label: "UTC" for
// UTC, otherwise the numeric GMT offset including minute and second parts.
func toolDetailsModelScheduleZone(local time.Time) string {
	name, offset := local.Zone()
	if offset == 0 {
		if name == "UTC" {
			return "UTC"
		}
		return "GMT"
	}
	sign := "+"
	if offset < 0 {
		sign, offset = "-", -offset
	}
	hours, minutes, seconds := offset/3600, offset%3600/60, offset%60
	switch {
	case seconds != 0:
		return fmt.Sprintf("GMT%s%d:%02d:%02d", sign, hours, minutes, seconds)
	case minutes != 0:
		return fmt.Sprintf("GMT%s%d:%02d", sign, hours, minutes)
	default:
		return fmt.Sprintf("GMT%s%d", sign, hours)
	}
}

// parsedToolCall uses JSON.parse of the actual call's argsRaw, never flattened
// projection fields. childResult supplies JSON.stringify(data.arguments).
func toolDetailsModelCallName(call *session.ToolCallView, nested bool) (string, bool) {
	var data map[string]json.RawMessage
	if nested {
		envelope, _ := terminalModelObject(call.Result.Raw)
		data, _ = terminalModelObject(envelope["data"])
		if _, ok := terminalModelObject(data["arguments"]); !ok {
			return "", false
		}
	} else {
		if call.Call == nil {
			return "", false
		}
		envelope, ok := terminalModelObject(call.Call.Raw)
		if !ok {
			return "", false
		}
		typ, _ := terminalModelString(envelope["type"])
		if typ != "tool/call" || call.Call.EventType != "" && call.Call.EventType != typ {
			return "", false
		}
		data, ok = terminalModelObject(envelope["data"])
		if !ok {
			return "", false
		}
		args, ok := terminalModelString(data["arguments"])
		if !ok {
			return "", false
		}
		if _, ok := terminalModelObject(json.RawMessage(args)); !ok {
			return "", false
		}
	}
	return terminalModelString(data["name"])
}

// count accepts exactly nonnegative Number.isSafeInteger values after JSON's
// IEEE-754 rounding. Zero and rounds above the maximum are valid recorded data.
func toolDetailsModelCount(raw json.RawMessage) (string, bool) {
	number, ok := toolReadModelInteger(raw, 0)
	if !ok || number > 9007199254740991 {
		return "", false
	}
	return strconv.FormatFloat(number, 'f', -1, 64), true
}
