package chat

import (
	"encoding/json"
	"fmt"
	"regexp"
	"strconv"
	"strings"
	"time"

	"whaleisle/internal/ui/i18n"
)

// The inspection families of the installed inspection-details-model.ts: Cordis
// runtime inspection, workflow/ralph reports and session-query searches, reads
// and traces. They run after the agent/team control receipts, exactly where the
// native dispatcher calls inspectionDetails, and return nil for an unknown or
// malformed recorded format so the generic evidence body remains.

// toolDetailsInspectionTRACEFields maps the recorded session/event trace labels
// to their localized names; an unknown label is dropped, as the installed
// textFields does.
var toolDetailsInspectionTRACEFields = map[string]string{
	"Created":                          i18n.KeyToolDetailFieldTime,
	"Availability":                     i18n.KeyToolDetailFieldAvailability,
	"Parent":                           i18n.KeyToolDetailFieldParent,
	"Best match":                       i18n.KeyToolDetailFieldBestMatch,
	"Target":                           i18n.KeyToolDetailFieldTarget,
	"Replaced by":                      i18n.KeyToolDetailTraceReplacedBy,
	"Replacement chain":                i18n.KeyToolDetailTraceReplacementChain,
	"Events replaced by target":        i18n.KeyToolDetailTraceReplaces,
	"Events cited directly as sources": i18n.KeyToolDetailTraceSources,
	"Direct derived events":            i18n.KeyToolDetailTraceDerived,
}

var (
	toolDetailsInspectionWorkflow  = regexp.MustCompile(`^workflow "([\s\S]*?)" completed \((\d+) agents?\)\.\nReturn value:\n([\s\S]*)$`)
	toolDetailsInspectionRounds    = regexp.MustCompile(`\b(\d+) rounds?\b`)
	toolDetailsInspectionBlock     = regexp.MustCompile(`(?m)^\d+\. `)
	toolDetailsInspectionSnippet   = regexp.MustCompile(`\n\s*Snippet: ([\s\S]*?)(?:\n\nResult cap reached\.|$)`)
	toolDetailsInspectionSession   = regexp.MustCompile(`^\d+\. Session (\S+) — (.*)`)
	toolDetailsInspectionEventHit  = regexp.MustCompile(`^\d+\. seq (\d+) \| ([^|]+) \| ([^|]+) \| ([^\n]+)`)
	toolDetailsInspectionRead      = regexp.MustCompile("^Session (\\S+) — ([^\\n]*)\\nTarget event seq (\\d+):\\n```json\\n([\\s\\S]*?)\\n```([\\s\\S]*)$")
	toolDetailsInspectionTraceHead = regexp.MustCompile(`^Session (\S+) — ([^\n]*)\n([\s\S]*)$`)
	toolDetailsInspectionField     = regexp.MustCompile(`^\s*([^:]+): (.+)$`)
	toolDetailsInspectionBullet    = regexp.MustCompile(`^(\s*)- `)
)

// toolDetailsInspectionDispatch mirrors inspectionDetails: a spill notice keeps
// the generic evidence body, and only the named tools are inspected.
func toolDetailsInspectionDispatch(name string, args map[string]json.RawMessage, text string) *toolDetailsModel {
	if terminalModelHasSpillNotice(text) {
		return nil
	}
	switch name {
	case "cordis_inspect_list", "cordis_inspect_query", "cordis_inspect_self":
		return toolDetailsInspectionCordis(name, args, text)
	case "workflow", "ralph":
		return toolDetailsInspectionReport(name, args, text)
	case "session_search", "session_event_search":
		return toolDetailsInspectionSearchHit(name, text)
	case "session_event_read":
		return toolDetailsInspectionEventRead(text)
	case "session_trace", "session_event_trace":
		return toolDetailsInspectionTrace(name, text)
	default:
		return nil
	}
}

// toolDetailsInspectionArgText reads a recorded string argument, as the
// installed arg() does.
func toolDetailsInspectionArgText(args map[string]json.RawMessage, key string) string {
	value, _ := terminalModelString(args[key])
	return value
}

// toolDetailsInspectionDateText mirrors the installed dateText: an Intl
// dateStyle medium / timeStyle short rendering for the two shipped locales, and
// the recorded text itself when the instant or locale is not supported.
func toolDetailsInspectionDateText(value *toolDetailsValue) string {
	raw := ""
	var instant time.Time
	valid := false
	switch value.kind {
	case 'n':
		raw = toolDetailsNumberText(value)
		milliseconds, err := strconv.ParseFloat(raw, 64)
		// JavaScript dates hold |ms| < 8.64e15; anything else is an invalid date.
		if err == nil && milliseconds > -8.64e15 && milliseconds < 8.64e15 {
			instant, valid = time.UnixMilli(int64(milliseconds)), true
		}
	case 's':
		raw = value.text
		instant, valid = toolDetailsInspectionParseInstant(raw)
	default:
		return ""
	}
	if !valid {
		return raw
	}
	local := instant.In(time.Local)
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
		return fmt.Sprintf("%s %d, %d, %d:%02d %s",
			toolDetailsModelScheduleMonths[local.Month()-1], local.Day(), local.Year(), hour, local.Minute(), period)
	case i18n.LocaleZH:
		return fmt.Sprintf("%d年%d月%d日 %02d:%02d", local.Year(), local.Month(), local.Day(), local.Hour(), local.Minute())
	default:
		return raw
	}
}

// toolDetailsInspectionParseInstant accepts the ISO spellings a recorded session
// event carries: millisecond UTC and RFC 3339 with an offset or nanoseconds.
func toolDetailsInspectionParseInstant(text string) (time.Time, bool) {
	for _, layout := range []string{toolDetailsModelScheduleInstantLayout, time.RFC3339Nano, time.RFC3339, "2006-01-02"} {
		if instant, err := time.Parse(layout, text); err == nil {
			return instant.UTC(), true
		}
	}
	return time.Time{}, false
}

// toolDetailsInspectionTextFields reads the recorded `Label: value` lines of a
// trace section, localizing known labels and converting only Created dates.
func toolDetailsInspectionTextFields(text string) []toolDetailField {
	fields := make([]toolDetailField, 0)
	for _, line := range strings.Split(text, "\n") {
		match := toolDetailsInspectionField.FindStringSubmatch(line)
		if match == nil {
			continue
		}
		key, known := toolDetailsInspectionTRACEFields[match[1]]
		if !known {
			continue
		}
		value := match[2]
		if match[1] == "Created" {
			value = toolDetailsInspectionDateText(&toolDetailsValue{kind: 's', text: value})
		} else if value == "none" {
			value = i18n.T(i18n.KeyToolDetailNone)
		}
		fields = append(fields, toolDetailField{Label: i18n.T(key), Value: value})
	}
	return fields
}

// toolDetailsInspectionStringField reports a recorded string property.
func toolDetailsInspectionStringField(value *toolDetailsValue, key string) (string, bool) {
	field, ok := value.get(key)
	if !ok || field.kind != 's' {
		return "", false
	}
	return field.text, true
}

// toolDetailsInspectionStringList reports a recorded array made only of strings.
func toolDetailsInspectionStringList(value *toolDetailsValue, key string) ([]string, bool) {
	field, ok := value.get(key)
	if !ok || field.kind != 'a' {
		return nil, false
	}
	lines := make([]string, 0, len(field.list))
	for _, entry := range field.list {
		if entry.kind != 's' {
			return nil, false
		}
		lines = append(lines, entry.text)
	}
	return lines, true
}

// toolDetailsInspectionCordis mirrors the installed Cordis cards: a provider or
// plugin list projects like any entity list, and a query names the provider and
// method it answered.
func toolDetailsInspectionCordis(name string, args map[string]json.RawMessage, text string) *toolDetailsModel {
	value, ok := toolDetailsDecodeJSON(text)
	if !ok || value.kind != 'o' {
		return nil
	}
	switch name {
	case "cordis_inspect_list":
		providers, ok := value.get("providers")
		if !ok || providers.kind != 'a' {
			return nil
		}
		return toolDetailsModelRecordedList(toolDetailsInspectionItems(providers, 0),
			fmt.Sprintf(i18n.T(i18n.KeyToolDetailProvidersCount), len(providers.list)))
	case "cordis_inspect_query":
		provider, okProvider := toolDetailsInspectionStringField(value, "provider")
		method, okMethod := toolDetailsInspectionStringField(value, "method")
		data, okData := value.get("data")
		if !okData || !okProvider || !okMethod {
			return nil
		}
		return toolDetailsModelRecordedList(toolDetailsInspectionItems(data, 0), provider+"."+method)
	default: // cordis_inspect_self
		if plugins, ok := value.get("plugins"); ok && plugins.kind == 'a' {
			return toolDetailsModelRecordedList(toolDetailsInspectionItems(plugins, 0),
				fmt.Sprintf(i18n.T(i18n.KeyToolDetailPluginsCount), len(plugins.list)))
		}
		summary := toolDetailsInspectionArgText(args, "pluginId")
		if summary == "" {
			summary = toolDetailsInspectionArgText(args, "packageId")
		}
		if summary == "" {
			summary, _ = toolDetailsInspectionStringField(value, "mode")
		}
		return toolDetailsModelRecordedList(toolDetailsInspectionItems(value, 0), summary)
	}
}

// toolDetailsInspectionReport mirrors the installed workflow and ralph cards.
func toolDetailsInspectionReport(name string, args map[string]json.RawMessage, text string) *toolDetailsModel {
	if name == "ralph" {
		return toolDetailsInspectionRalph(args, text)
	}
	match := toolDetailsInspectionWorkflow.FindStringSubmatch(text)
	if match == nil {
		return nil
	}
	value, ok := toolDetailsDecodeJSON(match[3])
	if !ok {
		return nil
	}
	header := toolDetailItem{
		Title:  match[1],
		Badge:  &toolDetailBadge{Label: i18n.T(i18n.KeyToolDetailStatusCompleted), Tone: "success"},
		Fields: []toolDetailField{{Label: i18n.T(i18n.KeyToolDetailFieldAgents), Value: match[2]}},
	}
	if code, ok := terminalModelString(args["code"]); ok {
		header.Groups = []toolDetailGroup{{Label: i18n.T(i18n.KeyToolDetailWorkflowScript),
			Items: []toolDetailItem{{Fields: []toolDetailField{}, Code: &toolDetailCode{Text: code, Language: "javascript"}}}}}
	}
	items := append([]toolDetailItem{header}, toolDetailsInspectionItems(value, 0)...)
	return toolDetailsModelRecordedList(items, match[1])
}

// toolDetailsInspectionRalph mirrors the installed final-report card. Every
// report field is required, and the header wording chooses the badge.
func toolDetailsInspectionRalph(args map[string]json.RawMessage, text string) *toolDetailsModel {
	const separator = "\nFinal report:\n"
	split := strings.Index(text, separator)
	if split < 0 {
		return nil
	}
	header := text[:split]
	report, ok := toolDetailsDecodeJSON(text[split+len(separator):])
	if !ok || report.kind != 'o' {
		return nil
	}
	summary, okSummary := toolDetailsInspectionStringField(report, "summary")
	evidence, okEvidence := toolDetailsInspectionStringList(report, "evidence")
	nextSteps, okSteps := toolDetailsInspectionStringList(report, "nextSteps")
	blocker, okBlocker := toolDetailsInspectionStringField(report, "blocker")
	if !okSummary || !okEvidence || !okSteps || !okBlocker {
		return nil
	}
	var badge *toolDetailBadge
	switch {
	case strings.HasPrefix(header, "Ralph worker reported completion "):
		badge = &toolDetailBadge{Label: i18n.T(i18n.KeyToolDetailRalphComplete), Tone: "success"}
	case strings.HasPrefix(header, "Ralph worker reported a blocker "):
		badge = &toolDetailBadge{Label: i18n.T(i18n.KeyToolDetailRalphBlocker), Tone: "warning"}
	case strings.HasPrefix(header, "Ralph reached its "):
		badge = &toolDetailBadge{Label: i18n.T(i18n.KeyToolDetailRalphLimit), Tone: "warning"}
	default:
		return nil
	}
	groups := make([]toolDetailGroup, 0, 2)
	if len(nextSteps) > 0 {
		groups = append(groups, toolDetailGroup{Label: i18n.T(i18n.KeyToolDetailReportNextSteps),
			Items: []toolDetailItem{{Fields: []toolDetailField{}, Lines: nextSteps}}})
	}
	if objective, ok := terminalModelString(args["objective"]); ok {
		groups = append(groups, toolDetailGroup{Label: i18n.T(i18n.KeyToolDetailFieldTask),
			Items: []toolDetailItem{{Fields: []toolDetailField{}, Description: toolDetailsModelString(objective)}}})
	}
	fields := make([]toolDetailField, 0, 2)
	if rounds := toolDetailsInspectionRounds.FindStringSubmatch(header); rounds != nil {
		fields = append(fields, toolDetailField{Label: i18n.T(i18n.KeyToolDetailGoalRounds), Value: rounds[1]})
	}
	if blocker != "" {
		fields = append(fields, toolDetailField{Label: i18n.T(i18n.KeyToolDetailGoalReason), Value: blocker})
	}
	item := toolDetailItem{Title: summary, Badge: badge, Fields: fields, Groups: groups}
	if len(evidence) > 0 {
		item.Lines = evidence
	}
	return toolDetailsModelRecordedList([]toolDetailItem{item}, summary)
}

// toolDetailsInspectionSearchHit mirrors the installed session and event search
// cards: numbered blocks, their snippet and trace fields, and the capped caption.
func toolDetailsInspectionSearchHit(name, text string) *toolDetailsModel {
	if text == "No prior session matches found." || strings.HasSuffix(text, "\n\nNo prior event matches found.") {
		return toolDetailsModelRecordedList([]toolDetailItem{}, fmt.Sprintf(i18n.T(i18n.KeyToolDetailMatchesCount), 0))
	}
	// Blocks begin at each `N. ` line, as the installed split-and-filter does;
	// text before the first block is a session heading and carries no row.
	starts := toolDetailsInspectionBlock.FindAllStringIndex(text, -1)
	items := make([]toolDetailItem, 0, len(starts))
	for index, start := range starts {
		end := len(text)
		if index+1 < len(starts) {
			end = starts[index+1][0]
		}
		block := strings.TrimSuffix(text[start[0]:end], "\n")
		snippet := ""
		if match := toolDetailsInspectionSnippet.FindStringSubmatch(block); match != nil {
			snippet = strings.TrimRight(match[1], " \t\r\n")
		}
		if name == "session_search" {
			match := toolDetailsInspectionSession.FindStringSubmatch(block)
			if match == nil {
				return nil
			}
			item := toolDetailItem{Title: match[2], Subtitle: toolDetailsModelString(match[1]),
				Fields: toolDetailsInspectionTextFields(block)}
			if snippet != "" {
				item.Description = toolDetailsModelString(snippet)
			}
			items = append(items, item)
			continue
		}
		match := toolDetailsInspectionEventHit.FindStringSubmatch(block)
		if match == nil {
			return nil
		}
		title := match[2]
		if snippet != "" {
			title = snippet
		}
		items = append(items, toolDetailItem{
			Title:    title,
			Subtitle: toolDetailsModelString(strings.TrimSpace(match[2]) + " · #" + match[1]),
			Fields: []toolDetailField{
				{Label: i18n.T(i18n.KeyToolDetailFieldTime), Value: toolDetailsInspectionDateText(&toolDetailsValue{kind: 's', text: match[4]})},
				{Label: i18n.T(i18n.KeyToolDetailFieldSurface), Value: strings.TrimSpace(match[3])},
			},
		})
	}
	if len(items) == 0 {
		return nil
	}
	model := toolDetailsModelRecordedList(items, fmt.Sprintf(i18n.T(i18n.KeyToolDetailMatchesCount), len(items)))
	if strings.Contains(text, "Result cap reached.") {
		model.Caption = i18n.T(i18n.KeyToolDetailMatchesCapped)
	}
	return model
}

// toolDetailsInspectionEventRead mirrors the installed event card: the target
// event names the row, its data projects, and adjacent events stay disclosed.
func toolDetailsInspectionEventRead(text string) *toolDetailsModel {
	match := toolDetailsInspectionRead.FindStringSubmatch(text)
	if match == nil {
		return nil
	}
	event, ok := toolDetailsDecodeJSON(match[4])
	if !ok || event.kind != 'o' {
		return nil
	}
	eventType, okType := toolDetailsInspectionStringField(event, "type")
	data, okData := event.get("data")
	if !okType || !okData || data.kind != 'o' {
		return nil
	}
	fields := []toolDetailField{{Label: i18n.T(i18n.KeyToolDetailFieldSeq), Value: match[3]}}
	if when, ok := event.get("time"); ok && when.kind == 'n' {
		fields = append(fields, toolDetailField{Label: i18n.T(i18n.KeyToolDetailFieldTime), Value: toolDetailsInspectionDateText(when)})
	}
	items := []toolDetailItem{{
		Title:    eventType,
		Subtitle: toolDetailsModelString(match[2] + " · " + match[1]),
		Fields:   fields,
	}}
	items = append(items, toolDetailsInspectionItems(data, 0)...)
	if adjacent := strings.TrimSpace(match[5]); adjacent != "" {
		items = append(items, toolDetailItem{Fields: []toolDetailField{}, Groups: []toolDetailGroup{{
			Label: i18n.T(i18n.KeyToolDetailEventNeighbors),
			Items: []toolDetailItem{{Fields: []toolDetailField{}, Description: toolDetailsModelString(adjacent)}},
		}}})
	}
	return toolDetailsModelRecordedList(items, eventType+" · #"+match[3])
}

// toolDetailsInspectionTrace mirrors the installed session and event trace cards.
// An event trace carries one record; a session trace adds its ancestor and
// descendant sections, and an unknown section rejects the card.
func toolDetailsInspectionTrace(name, text string) *toolDetailsModel {
	match := toolDetailsInspectionTraceHead.FindStringSubmatch(text)
	if match == nil {
		return nil
	}
	id, title, body := match[1], match[2], match[3]
	sections := strings.Split(body, "\n\n")
	head := toolDetailItem{Title: title, Subtitle: toolDetailsModelString(id), Fields: toolDetailsInspectionTextFields(sections[0])}
	if name == "session_event_trace" {
		return toolDetailsModelRecordedList([]toolDetailItem{head}, title)
	}
	items := []toolDetailItem{head}
	for _, section := range sections[1:] {
		firstBreak := strings.Index(section, "\n")
		label := ""
		switch {
		case strings.HasPrefix(section, "Ancestors (nearest first):"):
			label = i18n.T(i18n.KeyToolDetailTraceAncestors)
		case strings.HasPrefix(section, "Descendants:"):
			label = i18n.T(i18n.KeyToolDetailTraceDescendants)
		default:
			return nil
		}
		if firstBreak < 0 {
			return nil
		}
		content := section[firstBreak+1:]
		item := toolDetailItem{Title: label, Fields: []toolDetailField{}}
		if content == "- none" || content == "- none (target is a root session)" {
			item.Description = toolDetailsModelString(i18n.T(i18n.KeyToolDetailNone))
		} else {
			lines := make([]string, 0)
			for _, line := range strings.Split(content, "\n") {
				lines = append(lines, toolDetailsInspectionBullet.ReplaceAllString(line, "$1"))
			}
			item.Lines = lines
		}
		items = append(items, item)
	}
	return toolDetailsModelRecordedList(items, title)
}
