package chat

import (
	"encoding/json"
	"fmt"
	"regexp"
	"strconv"
	"strings"

	"whaleisle/internal/ui/i18n"
)

// The agent/team control receipts of the installed control-details-model.ts. The
// native dispatcher runs before the goal/reminder JSON families and derives each
// card from the recorded result text, the parsed whole result and the parsed
// arguments; malformed input returns nil so the generic evidence body remains.
// Installed expectations are pinned by the harness spec
// packages/client/ui-tool/tests/details-models.client.spec.ts:56-117.

const toolDetailsMaxInspectionItems = 40

// toolDetailsValue is parsed result JSON with its recorded key order, which the
// installed Object.entries order preserves.
type toolDetailsValue struct {
	kind  byte // 'o' object, 'a' array, 's' string, 'n' number, 'b' boolean, 'z' null
	keys  []string
	order []*toolDetailsValue
	list  []*toolDetailsValue
	text  string
	flag  bool
}

// get returns the own property named key, or false for a missing key.
func (v *toolDetailsValue) get(key string) (*toolDetailsValue, bool) {
	if v == nil || v.kind != 'o' {
		return nil, false
	}
	for index, name := range v.keys {
		if name == key {
			return v.order[index], true
		}
	}
	return nil, false
}

// toolDetailsDecodeJSON parses one complete JSON text, as the installed
// detailJson rejects a partial prefix.
func toolDetailsDecodeJSON(text string) (*toolDetailsValue, bool) {
	decoder := json.NewDecoder(strings.NewReader(text))
	decoder.UseNumber()
	value, ok := toolDetailsDecodeValue(decoder)
	if !ok {
		return nil, false
	}
	if _, err := decoder.Token(); err == nil {
		return nil, false
	}
	return value, true
}

func toolDetailsDecodeValue(decoder *json.Decoder) (*toolDetailsValue, bool) {
	token, err := decoder.Token()
	if err != nil {
		return nil, false
	}
	switch value := token.(type) {
	case json.Delim:
		switch value {
		case '{':
			object := &toolDetailsValue{kind: 'o'}
			for decoder.More() {
				name, err := decoder.Token()
				if err != nil {
					return nil, false
				}
				key, ok := name.(string)
				if !ok {
					return nil, false
				}
				field, ok := toolDetailsDecodeValue(decoder)
				if !ok {
					return nil, false
				}
				object.keys = append(object.keys, key)
				object.order = append(object.order, field)
			}
			if _, err := decoder.Token(); err != nil {
				return nil, false
			}
			return object, true
		case '[':
			array := &toolDetailsValue{kind: 'a'}
			for decoder.More() {
				entry, ok := toolDetailsDecodeValue(decoder)
				if !ok {
					return nil, false
				}
				array.list = append(array.list, entry)
			}
			if _, err := decoder.Token(); err != nil {
				return nil, false
			}
			return array, true
		default:
			return nil, false
		}
	case string:
		return &toolDetailsValue{kind: 's', text: value}, true
	case json.Number:
		return &toolDetailsValue{kind: 'n', text: value.String()}, true
	case bool:
		return &toolDetailsValue{kind: 'b', flag: value}, true
	case nil:
		return &toolDetailsValue{kind: 'z'}, true
	default:
		return nil, false
	}
}

// toolDetailsNumberText prints a recorded number the way JSON.stringify does
// for the values these receipts carry. Fractional forms whose JavaScript form is
// decimal outside JSON's own exponent range keep Go's shortest form.
func toolDetailsNumberText(value *toolDetailsValue) string {
	number, err := strconv.ParseFloat(value.text, 64)
	if err != nil {
		return value.text
	}
	if number == float64(int64(number)) {
		return strconv.FormatInt(int64(number), 10)
	}
	return strconv.FormatFloat(number, 'g', -1, 64)
}

// toolDetailsScalar renders one non-container JSON value: null and booleans are
// localized, strings stay verbatim and everything else is its JSON text.
func toolDetailsScalar(value *toolDetailsValue) string {
	switch value.kind {
	case 'z':
		return i18n.T(i18n.KeyToolDetailNone)
	case 'b':
		if value.flag {
			return i18n.T(i18n.KeyToolDetailYes)
		}
		return i18n.T(i18n.KeyToolDetailNo)
	case 's':
		return value.text
	case 'n':
		return toolDetailsNumberText(value)
	default:
		return toolDetailsJSONText(value, "")
	}
}

// toolDetailsJSONText writes recorded JSON with its key order, as
// JSON.stringify(value, null, 2) does for a custom depth.
func toolDetailsJSONText(value *toolDetailsValue, indent string) string {
	inner := indent + "  "
	switch value.kind {
	case 'o':
		if len(value.keys) == 0 {
			return "{}"
		}
		parts := make([]string, 0, len(value.keys))
		for index, key := range value.keys {
			encoded, _ := json.Marshal(key)
			parts = append(parts, inner+string(encoded)+": "+toolDetailsJSONText(value.order[index], inner))
		}
		return "{\n" + strings.Join(parts, ",\n") + "\n" + indent + "}"
	case 'a':
		if len(value.list) == 0 {
			return "[]"
		}
		parts := make([]string, 0, len(value.list))
		for _, entry := range value.list {
			parts = append(parts, inner+toolDetailsJSONText(entry, inner))
		}
		return "[\n" + strings.Join(parts, ",\n") + "\n" + indent + "]"
	case 's':
		encoded, _ := json.Marshal(value.text)
		return string(encoded)
	case 'n':
		return toolDetailsNumberText(value)
	case 'b':
		if value.flag {
			return "true"
		}
		return "false"
	default:
		return "null"
	}
}

// toolDetailsFieldLabel localizes the installed known field names and keeps
// extension-owned names verbatim.
func toolDetailsFieldLabel(key string) string {
	if label, known := toolDetailsFieldKeys[key]; known {
		return i18n.T(label)
	}
	return key
}

// toolDetailsTitleKeys and toolDetailsDescriptionKeys are the installed field
// preference orders for a projected record.
var (
	toolDetailsTitleKeys       = [...]string{"subject", "title", "name", "pluginId", "packageId", "id", "summary"}
	toolDetailsDescriptionKeys = [...]string{"description", "purpose"}
)

// toolDetailsInspectionItems projects recorded JSON into the installed compact
// item vocabulary: a title, description and status badge for a record, named
// fields for scalars and disclosures for nested values, capped at 40 array
// entries per level and five levels deep.
func toolDetailsInspectionItems(value *toolDetailsValue, depth int) []toolDetailItem {
	if depth > 4 && (value.kind == 'o' || value.kind == 'a') {
		return []toolDetailItem{{Code: &toolDetailCode{Text: toolDetailsJSONText(value, ""), Language: "json"}}}
	}
	if value.kind == 'a' {
		items := make([]toolDetailItem, 0, len(value.list))
		for index, entry := range value.list {
			if index >= toolDetailsMaxInspectionItems {
				break
			}
			items = append(items, toolDetailsInspectionItems(entry, depth+1)...)
		}
		if remaining := len(value.list) - toolDetailsMaxInspectionItems; remaining > 0 {
			items = append(items, toolDetailItem{Description: toolDetailsModelString(
				fmt.Sprintf(i18n.T(i18n.KeyToolDetailMoreInInspect), remaining))})
		}
		if len(items) == 0 {
			return []toolDetailItem{{Description: toolDetailsModelString(i18n.T(i18n.KeyToolDetailEmpty))}}
		}
		return items
	}
	if value.kind != 'o' {
		return []toolDetailItem{{Description: toolDetailsModelString(toolDetailsScalar(value))}}
	}
	titleKey, descriptionKey := "", ""
	for _, key := range toolDetailsTitleKeys {
		if field, ok := value.get(key); ok && field.kind == 's' && field.text != "" {
			titleKey = key
			break
		}
	}
	for _, key := range toolDetailsDescriptionKeys {
		if field, ok := value.get(key); ok && field.kind == 's' && field.text != "" {
			descriptionKey = key
			break
		}
	}
	item := toolDetailItem{Fields: []toolDetailField{}}
	status := ""
	for index, key := range value.keys {
		field := value.order[index]
		if key == titleKey || key == descriptionKey || (key == "status" && field.kind == 's') {
			if key == "status" && field.kind == 's' {
				status = field.text
			}
			continue
		}
		if key == "inputSchema" || key == "outputSchema" {
			item.Groups = append(item.Groups, toolDetailGroup{Label: toolDetailsFieldLabel(key),
				Items: []toolDetailItem{{Fields: []toolDetailField{}, Code: &toolDetailCode{Text: toolDetailsJSONText(field, ""), Language: "json"}}}})
			continue
		}
		if field.kind == 'a' && len(field.list) == 0 {
			continue
		}
		if key == "arguments" && field.kind == 's' {
			if args, ok := toolDetailsDecodeJSON(field.text); ok && args.kind == 'o' {
				item.Groups = append(item.Groups, toolDetailGroup{Label: toolDetailsFieldLabel(key),
					Items: toolDetailsInspectionItems(args, depth+1)})
				continue
			}
		}
		if field.kind == 'o' || field.kind == 'a' {
			item.Groups = append(item.Groups, toolDetailGroup{Label: toolDetailsFieldLabel(key),
				Items: toolDetailsInspectionItems(field, depth+1)})
			continue
		}
		item.Fields = append(item.Fields, toolDetailField{Label: toolDetailsFieldLabel(key), Value: toolDetailsScalar(field)})
	}
	if titleKey != "" {
		title, _ := value.get(titleKey)
		item.Title = title.text
	} else if status != "" {
		// A record with only a status still names its row, as the installed
		// `title ?? t('detail.field.result')` fallback does.
		item.Title = i18n.T(i18n.KeyToolDetailFieldResult)
	}
	if descriptionKey != "" {
		description, _ := value.get(descriptionKey)
		item.Description = toolDetailsModelString(description.text)
	}
	if status != "" {
		badge := toolDetailsModelBadge(status)
		item.Badge = &badge
	}
	return []toolDetailItem{item}
}

// toolDetailsArgText is the installed arg(): only a recorded string counts, and
// everything else reads as an empty string.
func toolDetailsArgText(args map[string]json.RawMessage, key string) string {
	value, _ := terminalModelString(args[key])
	return value
}

// toolDetailsReceipt is the installed receipt(): one entity row whose collapsed
// summary names the target and its status.
func toolDetailsReceipt(title string, badge toolDetailBadge, fields []toolDetailField, description *string) *toolDetailsModel {
	if fields == nil {
		fields = []toolDetailField{}
	}
	model := toolDetailsModelRecordedList([]toolDetailItem{{
		Title: title, Badge: &badge, Fields: fields, Description: description,
	}}, title+" · "+badge.Label)
	model.ExpandedSummary = title
	return model
}

func toolDetailsReceiptDescription(text string) *string {
	return toolDetailsModelString(text)
}

var (
	toolDetailsAgentLine       = regexp.MustCompile(`^(\S+) \[([^\]]+)\](?: parent=(\S+) depth=(\d+))?(?: — (.*))?$`)
	toolDetailsSubagentStarted = regexp.MustCompile(`^started (background subagent job|subagent) (\S+)$`)
)

// toolDetailsModelControl derives the installed agent/team control receipts, or
// nil for a name or recorded format outside this family.
func toolDetailsModelControl(name string, args map[string]json.RawMessage, text string) *toolDetailsModel {
	switch name {
	case "list_agents":
		return toolDetailsModelAgentList(text)
	case "terminal_list":
		return toolDetailsModelTerminalList(text)
	case "lsp":
		return toolDetailsModelLSP(args, text)
	case "spawn_teammate":
		return toolDetailsModelSpawn(args, text)
	case "team_task_create", "team_task_get", "team_task_update":
		return toolDetailsModelTeamTask(text)
	case "team_task_list":
		return toolDetailsModelTeamTaskList(text)
	case "send_message":
		return toolDetailsModelSendMessage(args, text)
	case "interrupt_agent":
		return toolDetailsModelInterrupt(args, text)
	case "wait_agent":
		return toolDetailsModelWait(text)
	case "subagent":
		return toolDetailsModelSubagent(args, text)
	case "list_subagent_models":
		return toolDetailsModelSubagentModels(args, text)
	}
	return nil
}

// toolDetailsDecoded parses a recorded result once, keeping the native rule that
// non-JSON text is simply absent.
func toolDetailsDecoded(text string) *toolDetailsValue {
	value, _ := toolDetailsDecodeJSON(text)
	return value
}

// toolDetailsModelAgentList mirrors the installed agentList: a JSON roster
// projects like any entity list, the empty text keeps the zero count, and the
// line format carries the target, state, optional parent/depth and title.
func toolDetailsModelAgentList(text string) *toolDetailsModel {
	if value := toolDetailsDecoded(text); value != nil && value.kind == 'a' {
		return toolDetailsModelRecordedList(toolDetailsInspectionItems(value, 0),
			fmt.Sprintf(i18n.T(i18n.KeyToolDetailAgentsCount), len(value.list)))
	}
	if text == "(no subagents)" {
		return toolDetailsModelRecordedList([]toolDetailItem{},
			fmt.Sprintf(i18n.T(i18n.KeyToolDetailAgentsCount), 0))
	}
	items := make([]toolDetailItem, 0)
	for _, line := range strings.Split(text, "\n") {
		match := toolDetailsAgentLine.FindStringSubmatch(line)
		if match == nil {
			return nil
		}
		id, state, parent, depth, title := match[1], match[2], match[3], match[4], match[5]
		item := toolDetailItem{Title: id, Fields: []toolDetailField{}}
		if match[5] != "" {
			item.Subtitle = toolDetailsModelString(id)
			item.Title = title
		}
		badge := toolDetailsModelBadge(state)
		item.Badge = &badge
		if match[3] != "" {
			item.Fields = append(item.Fields,
				toolDetailField{Label: i18n.T(i18n.KeyToolDetailFieldParent), Value: parent},
				toolDetailField{Label: i18n.T(i18n.KeyToolDetailFieldDepth), Value: depth})
		}
		items = append(items, item)
	}
	return toolDetailsModelRecordedList(items, fmt.Sprintf(i18n.T(i18n.KeyToolDetailAgentsCount), len(items)))
}

// toolDetailsModelSpawn mirrors the installed spawn_teammate receipt: the member
// record is projected like an entity and the recorded name labels the row.
func toolDetailsModelSpawn(args map[string]json.RawMessage, text string) *toolDetailsModel {
	value := toolDetailsDecoded(text)
	if value == nil || value.kind != 'o' {
		return nil
	}
	member, ok := value.get("member")
	if !ok || member.kind != 'o' {
		return nil
	}
	return toolDetailsModelRecordedList(toolDetailsInspectionItems(member, 0), toolDetailsArgText(args, "name"))
}

// toolDetailsModelTeamTask mirrors the installed team_task_create/get/update
// cards: the whole task record is projected and its subject labels the row.
func toolDetailsModelTeamTask(text string) *toolDetailsModel {
	value := toolDetailsDecoded(text)
	if value == nil || value.kind != 'o' {
		return nil
	}
	subject, ok := value.get("subject")
	if !ok || subject.kind != 's' {
		return nil
	}
	return toolDetailsModelRecordedList(toolDetailsInspectionItems(value, 0), subject.text)
}

// toolDetailsModelTeamTaskList mirrors the installed list: every task projects,
// the count labels the row, and only a recorded number may page.
func toolDetailsModelTeamTaskList(text string) *toolDetailsModel {
	value := toolDetailsDecoded(text)
	if value == nil || value.kind != 'o' {
		return nil
	}
	tasks, ok := value.get("tasks")
	if !ok || tasks.kind != 'a' {
		return nil
	}
	model := toolDetailsModelRecordedList(toolDetailsInspectionItems(tasks, 0),
		fmt.Sprintf(i18n.T(i18n.KeyToolDetailTasksCount), len(tasks.list)))
	if cursor, ok := value.get("nextCursor"); ok {
		if cursor.kind != 'n' {
			return nil
		}
		model.Caption = fmt.Sprintf(i18n.T(i18n.KeyToolDetailTasksNextPage), toolDetailsNumberText(cursor))
	}
	return model
}

// toolDetailsModelSendMessage mirrors the installed delivery receipt: accepted
// and queued results name the target, and the plain delivered sentence is the
// fallback for a result recorded without a status record.
func toolDetailsModelSendMessage(args map[string]json.RawMessage, text string) *toolDetailsModel {
	target := toolDetailsModelTarget(args)
	message := toolDetailsModelString(toolDetailsArgText(args, "message"))
	status := ""
	if value := toolDetailsDecoded(text); value != nil && value.kind == 'o' {
		if field, ok := value.get("status"); ok && field.kind == 's' {
			status = field.text
		}
	}
	switch status {
	case "accepted":
		return toolDetailsReceipt(target, toolDetailBadge{Label: i18n.T(i18n.KeyToolDetailReceiptDelivered), Tone: "success"}, nil, message)
	case "queued":
		return toolDetailsReceipt(target, toolDetailBadge{Label: i18n.T(i18n.KeyToolDetailStatusQueued), Tone: "warning"}, nil, message)
	}
	if text == "message delivered to agent "+target {
		return toolDetailsReceipt(target, toolDetailBadge{Label: i18n.T(i18n.KeyToolDetailReceiptDelivered), Tone: "success"}, nil, message)
	}
	return nil
}

// toolDetailsModelInterrupt mirrors the installed interrupt receipt: a recorded
// previous status is shown localized inside the field column.
func toolDetailsModelInterrupt(args map[string]json.RawMessage, text string) *toolDetailsModel {
	target := toolDetailsModelTarget(args)
	badge := toolDetailBadge{Label: i18n.T(i18n.KeyToolDetailReceiptInterrupt), Tone: "warning"}
	if value := toolDetailsDecoded(text); value != nil && value.kind == 'o' {
		if field, ok := value.get("previousStatus"); ok && field.kind == 's' {
			previous := toolDetailsModelBadge(field.text).Label
			return toolDetailsReceipt(target, badge,
				[]toolDetailField{{Label: i18n.T(i18n.KeyToolDetailFieldPreviousStatus), Value: previous}}, nil)
		}
	}
	if text == "interrupt requested for agent "+target {
		return toolDetailsReceipt(target, badge, nil, nil)
	}
	return nil
}

// toolDetailsModelWait mirrors the installed wait receipt: a stalled wait names
// its own message, otherwise the title is the localized subject and the badge
// the outcome.
func toolDetailsModelWait(text string) *toolDetailsModel {
	value := toolDetailsDecoded(text)
	if value == nil || value.kind != 'o' {
		return nil
	}
	timedOut, ok := value.get("timedOut")
	if !ok || timedOut.kind != 'b' {
		return nil
	}
	if progress, ok := value.get("noProgress"); ok && progress.kind == 'o' {
		if message, ok := progress.get("message"); ok && message.kind == 's' {
			return toolDetailsModelRecordedList([]toolDetailItem{{
				Title:       i18n.T(i18n.KeyToolDetailWaitNoProgress),
				Description: toolDetailsModelString(message.text),
			}}, i18n.T(i18n.KeyToolDetailWaitNoProgress))
		}
	}
	label := i18n.KeyToolDetailWaitChanged
	if timedOut.flag {
		label = i18n.KeyToolDetailWaitTimeout
	}
	return toolDetailsReceipt(i18n.T(i18n.KeyToolDetailWaitTitle), toolDetailBadge{Label: i18n.T(label), Tone: "neutral"}, nil, nil)
}

// toolDetailsModelSubagent mirrors the installed subagent receipts: a started
// job reports its id, and a reply card keeps the recorded text with the task in
// a disclosure. The installed card renders the reply as Markdown; this card
// keeps the recorded reply in the native Markdown entry.
func toolDetailsModelSubagent(args map[string]json.RawMessage, text string) *toolDetailsModel {
	if match := toolDetailsSubagentStarted.FindStringSubmatch(text); match != nil {
		label := i18n.KeyToolDetailFieldJob
		if match[1] == "subagent" {
			label = i18n.KeyToolDetailFieldAgent
		}
		return toolDetailsReceipt(toolDetailsArgText(args, "prompt"),
			toolDetailBadge{Label: i18n.T(i18n.KeyToolDetailReceiptStarted), Tone: "info"},
			[]toolDetailField{{Label: i18n.T(label), Value: match[2]}}, nil)
	}
	return toolDetailsModelRecordedList([]toolDetailItem{{
		Title:    i18n.T(i18n.KeyToolDetailAgentReply),
		Markdown: toolDetailsModelString(text),
		Fields:   []toolDetailField{},
		Groups: []toolDetailGroup{{Label: i18n.T(i18n.KeyToolDetailFieldTask),
			Items: []toolDetailItem{{Description: toolDetailsModelString(toolDetailsArgText(args, "prompt"))}}}},
	}}, i18n.T(i18n.KeyToolDetailAgentReply))
}

// toolDetailsModelSubagentModels mirrors the installed model list: each line is a
// model name and its recorded note, and the requested model, provider or the
// localized fallback labels the row.
func toolDetailsModelSubagentModels(args map[string]json.RawMessage, text string) *toolDetailsModel {
	items := make([]toolDetailItem, 0)
	for _, line := range strings.Split(text, "\n") {
		split := strings.Index(line, " — ")
		if split < 0 {
			items = append(items, toolDetailItem{Description: toolDetailsModelString(line), Fields: []toolDetailField{}})
			continue
		}
		items = append(items, toolDetailItem{
			Title:       line[:split],
			Description: toolDetailsModelString(line[split+len(" — "):]),
			Fields:      []toolDetailField{},
		})
	}
	summary := toolDetailsArgText(args, "model")
	if summary == "" {
		summary = toolDetailsArgText(args, "provider")
	}
	if summary == "" {
		summary = i18n.T(i18n.KeyToolDetailModelsTitle)
	}
	return toolDetailsModelRecordedList(items, summary)
}

// toolDetailsFieldKeys is the installed FIELD_KEYS vocabulary: a known recorded
// property name is localized, anything else keeps its spelling.
var toolDetailsFieldKeys = map[string]string{
	"id":                 i18n.KeyToolDetailFieldID,
	"revision":           i18n.KeyToolDetailFieldRevision,
	"platform":           i18n.KeyToolDetailFieldPlatform,
	"provider":           i18n.KeyToolDetailFieldProvider,
	"model":              i18n.KeyToolDetailFieldModel,
	"role":               i18n.KeyToolDetailFieldRole,
	"context":            i18n.KeyToolDetailFieldContext,
	"ownerName":          i18n.KeyToolDetailFieldOwner,
	"ready":              i18n.KeyToolDetailFieldReady,
	"blockedBy":          i18n.KeyToolDetailFieldDependencies,
	"writeScopes":        i18n.KeyToolDetailFieldWriteScopes,
	"writeScopeWarnings": i18n.KeyToolDetailFieldWarnings,
	"diagnostics":        i18n.KeyToolDetailFieldDiagnostics,
	"methods":            i18n.KeyToolDetailFieldMethods,
	"inputSchema":        i18n.KeyToolDetailFieldInputSchema,
	"outputSchema":       i18n.KeyToolDetailFieldOutputSchema,
	"currentPackageId":   i18n.KeyToolDetailFieldCurrentPackage,
	"nextPackageId":      i18n.KeyToolDetailFieldNextPackage,
	"latestRun":          i18n.KeyToolDetailFieldLatestRun,
	"packages":           i18n.KeyToolDetailFieldPackages,
	"registrations":      i18n.KeyToolDetailFieldRegistrations,
	"props":              i18n.KeyToolDetailFieldProps,
	"data":               i18n.KeyToolDetailFieldData,
	"source":             i18n.KeyToolDetailFieldSource,
	"arguments":          i18n.KeyToolDetailRowInput,
	"content":            i18n.KeyToolDetailFieldContent,
	"message":            i18n.KeyToolDetailFieldMessage,
	"messageId":          i18n.KeyToolDetailFieldMessageID,
	"status":             i18n.KeyToolDetailState,
	"root":               i18n.KeyToolDetailFieldRoot,
	"pid":                i18n.KeyToolDetailFieldPID,
	"type":               i18n.KeyToolDetailFieldType,
	"time":               i18n.KeyToolDetailFieldTime,
	"seq":                i18n.KeyToolDetailFieldSeq,
	"turn":               i18n.KeyToolDetailFieldTurn,
	"step":               i18n.KeyToolDetailFieldStep,
	"callId":             i18n.KeyToolDetailFieldCallID,
	"agentsStarted":      i18n.KeyToolDetailFieldAgents,
	"output":             i18n.KeyToolDetailRowOutput,
	"result":             i18n.KeyToolDetailFieldResult,
}

var toolDetailsTerminalLine = regexp.MustCompile(`^(\S+)(?: \((.*?)\))? \[([^\]]+)\] (running|exited code=(\S+) signal=(\S+))(?: pid=(\d+))?$`)
var toolDetailsLocationLine = regexp.MustCompile(`^(.*):(\d+):(\d+)$`)
var toolDetailsLocationURI = regexp.MustCompile(`^[a-z][a-z\d+.-]*:`)
var toolDetailsLocationDrive = regexp.MustCompile(`^[a-z]:[\\/]`)

// toolDetailsModelTerminalList mirrors the installed terminal roster: the
// session id, its optional name, the recorded type and the state line, with the
// exit code and signal only for an exited session.
func toolDetailsModelTerminalList(text string) *toolDetailsModel {
	if text == "(no terminal sessions)" {
		return toolDetailsModelRecordedList([]toolDetailItem{}, fmt.Sprintf(i18n.T(i18n.KeyToolDetailTerminalsCount), 0))
	}
	items := make([]toolDetailItem, 0)
	for _, line := range strings.Split(text, "\n") {
		match := toolDetailsTerminalLine.FindStringSubmatch(line)
		if match == nil {
			return nil
		}
		id, name, kind, state, exitCode, signal, pid := match[1], match[2], match[3], match[4], match[5], match[6], match[7]
		fields := []toolDetailField{{Label: i18n.T(i18n.KeyToolDetailFieldType), Value: kind}}
		if pid != "" {
			fields = append(fields, toolDetailField{Label: i18n.T(i18n.KeyToolDetailFieldPID), Value: pid})
		}
		if exitCode != "" {
			fields = append(fields, toolDetailField{Label: i18n.T(i18n.KeyToolDetailFieldExitCode), Value: exitCode})
		}
		if signal != "" && signal != "null" {
			fields = append(fields, toolDetailField{Label: i18n.T(i18n.KeyToolDetailFieldSignal), Value: signal})
		}
		item := toolDetailItem{Title: id, Fields: fields}
		if name != "" {
			item.Title, item.Subtitle = name, toolDetailsModelString(id)
		}
		if state == "running" {
			badge := toolDetailsModelBadge("running")
			item.Badge = &badge
		} else {
			tone := "neutral"
			if exitCode == "0" {
				tone = "success"
			}
			item.Badge = &toolDetailBadge{Label: i18n.T(i18n.KeyToolDetailStatusExited), Tone: tone}
		}
		items = append(items, item)
	}
	return toolDetailsModelRecordedList(items, fmt.Sprintf(i18n.T(i18n.KeyToolDetailTerminalsCount), len(items)))
}

// toolDetailsNumberArg reports a recorded numeric argument as its display text,
// as the installed lsp adapter requires `typeof args.line === 'number'`.
func toolDetailsNumberArg(args map[string]json.RawMessage, key string) (string, bool) {
	raw, ok := args[key]
	if !ok {
		return "", false
	}
	var value any
	if json.Unmarshal(raw, &value) != nil {
		return "", false
	}
	number, ok := value.(float64)
	if !ok {
		return "", false
	}
	return strconv.FormatFloat(number, 'f', -1, 64), true
}

// toolDetailsModelLSP mirrors the installed language-server card: hover text is a
// Markdown entry on a clickable location, and references are location rows whose
// non-URI paths stay openable.
func toolDetailsModelLSP(args map[string]json.RawMessage, text string) *toolDetailsModel {
	file := toolDetailsArgText(args, "file_path")
	operation := toolDetailsArgText(args, "operation")
	lineText, okLine := toolDetailsNumberArg(args, "line")
	columnText, okColumn := toolDetailsNumberArg(args, "character")
	if file == "" || !okLine || !okColumn {
		return nil
	}
	source := file + ":" + lineText + ":" + columnText
	if operation == "hover" {
		line, err := strconv.Atoi(lineText)
		location := &toolDetailLocation{Path: file}
		if err == nil {
			location.Line = &line
		}
		return toolDetailsModelRecordedList([]toolDetailItem{{
			Title: source, Fields: []toolDetailField{}, Location: location, Markdown: toolDetailsModelString(text),
		}}, source)
	}
	if text == "No results." {
		return toolDetailsModelRecordedList([]toolDetailItem{}, fmt.Sprintf(i18n.T(i18n.KeyToolDetailLocationsCount), 0))
	}
	items := make([]toolDetailItem, 0)
	titled := 0
	for _, entry := range strings.Split(text, "\n") {
		if strings.HasPrefix(entry, "… ") {
			items = append(items, toolDetailItem{Fields: []toolDetailField{}, Description: toolDetailsModelString(entry)})
			continue
		}
		match := toolDetailsLocationLine.FindStringSubmatch(entry)
		if match == nil {
			return nil
		}
		path, row, column := match[1], match[2], match[3]
		item := toolDetailItem{Title: path, Fields: []toolDetailField{},
			Subtitle: toolDetailsModelString(fmt.Sprintf(i18n.T(i18n.KeyToolDetailLocation), row, column))}
		if !(toolDetailsLocationURI.MatchString(path) && !toolDetailsLocationDrive.MatchString(path)) {
			line, err := strconv.Atoi(row)
			if err == nil {
				item.Location = &toolDetailLocation{Path: path, Line: &line}
			}
		}
		titled++
		items = append(items, item)
	}
	return toolDetailsModelRecordedList(items, file+" · "+
		fmt.Sprintf(i18n.T(i18n.KeyToolDetailLocationsCount), titled))
}
