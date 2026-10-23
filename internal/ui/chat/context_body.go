package chat

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"math"
	"sort"
	"strconv"
	"strings"

	"github.com/egoist/mygo/ui"

	"whaleisle/internal/store/session"
	"whaleisle/internal/ui/i18n"
	"whaleisle/internal/ui/theme"
)

const (
	contextBodyMaxUnits   = 20_000
	contextBodyMaxEntries = 200
)

// The disclosure owner supplies the font and semantic colors, as well as its
// surface, padding and height cap. A body adds none of that outer chrome.
type contextBodyStyle struct {
	Font                                 string
	Size, LineHeight                     float32
	Secondary, Tertiary, Caption, Border ui.Color
	Primary, BorderL1, CodeBlock, Hover  ui.Color
}

type contextContentRun struct {
	Text string
	Raw  json.RawMessage
}

type contextSourceField struct {
	Name string
	Raw  json.RawMessage
}

type contextInstruction struct {
	Path, Action, Digest string
}

type contextCatalogEntry struct {
	Name, Description string
}

type contextSnapshotSection struct {
	Name, Text string
}

type contextRecallSession struct {
	Label, Retained, Omitted string
	Truncated                bool
}

// Form names the presentation actually resolved; an empty Form is opaque.
// Summary is populated only for a usable notice. Raw supplies ordered content
// when available; resolving or painting never changes the retained message.
type contextBody struct {
	Form, Summary string
	key           string
	runs          []contextContentRun
	fields        []contextSourceField
	instructions  []contextInstruction
	baseline      bool
	catalog       []contextCatalogEntry
	catalogTotal  int
	catalogUpdate bool
	sections      []contextSnapshotSection
	sender        string
	recalls       []contextRecallSession
}

func resolveContextBody(msg session.Message) contextBody {
	source, content, hasContent := readContextBodyMessage(msg)
	body := contextBody{
		key:    fmt.Sprintf("context-body:%s:%s:%g", msg.ID, msg.SourceMessageID, msg.Seq),
		fields: source,
		runs:   contextBodyRuns(content),
	}
	if !hasContent && msg.Body != "" {
		body.runs = []contextContentRun{{Text: msg.Body}}
	}
	form, _ := contextSourceString(source, "form")
	switch form {
	case "instructions":
		if changes, ok := contextInstructionChanges(contextSourceRaw(source, "changes")); ok {
			body.Form, body.instructions = form, changes
			body.baseline = bytes.Equal(bytes.TrimSpace(contextSourceRaw(source, "baseline")), []byte("true"))
		}
	case "catalog":
		if entries, total, ok := contextCatalogEntries(contextSourceRaw(source, "entries")); ok {
			body.Form, body.catalog, body.catalogTotal = form, entries, total
			body.catalogUpdate = bytes.Equal(bytes.TrimSpace(contextSourceRaw(source, "update")), []byte("true"))
		}
	case "snapshot":
		if sections, ok := contextSnapshotSections(contextSourceRaw(source, "sections")); ok {
			body.Form, body.sections = form, sections
		}
	case "notice":
		if summary, ok := contextSourceString(source, "summary"); ok && summary != "" {
			body.Form, body.Summary = form, summary
		}
	case "relay":
		if sender, ok := contextSourceString(source, "senderSessionId"); ok && sender != "" {
			body.Form, body.sender = form, sender
		}
	case "recall":
		if recalls, ok := contextRecalledSessions(contextSourceRaw(source, "references")); ok {
			body.Form, body.recalls = form, recalls
		}
	}
	return body
}

// A trigger uses NoticeBody directly in the reference: its source, including a
// form declaration or malformed summary, does not select or decorate the body.
func paintNoticeContext(c *ui.Context, msg session.Message, style contextBodyStyle) {
	_, content, hasContent := readContextBodyMessage(msg)
	runs := contextBodyRuns(content)
	if !hasContent && msg.Body != "" {
		runs = []contextContentRun{{Text: msg.Body}}
	}
	ui.Column(c.Key(fmt.Sprintf("notice-context:%s:%s:%g", msg.ID, msg.SourceMessageID, msg.Seq))).
		FillWidth().MinWidth(0).Gap(0).Selectable().Children(func() { paintContextBodyRuns(c, runs, style, false) })
}

func (body contextBody) paint(c *ui.Context, style contextBodyStyle) {
	ui.Column(c.Key(body.key)).FillWidth().MinWidth(0).Gap(0).Selectable().Children(func() {
		switch body.Form {
		case "instructions":
			ui.Row(c).FillWidth().MinWidth(0).Wrap().GapX(12).GapY(4).Margin(0, 0, 8, 0).Children(func() {
				for _, change := range body.instructions {
					ui.Row(c).MinWidth(0).Gap(6).AlignItems(ui.Start).Children(func() {
						path := style.meta(c, change.Path, style.Secondary).Shrink(1).MinWidth(0)
						if change.Digest != "" {
							path.Tooltip(change.Digest)
						}
						style.text(c, contextInstructionAction(change.Action, body.baseline), style.Caption).Shrink(0)
					})
				}
			})
			paintContextBodyRuns(c, body.runs, style, false)
		case "catalog":
			if body.catalogUpdate {
				style.text(c, i18n.T(i18n.KeyContextCatalogReplaced), style.Caption).FillWidth().Margin(0, 0, 6, 0)
			}
			ui.Column(c).FillWidth().MinWidth(0).Gap(4).Children(func() {
				for _, entry := range body.catalog {
					ui.Row(c).FillWidth().MinWidth(0).Gap(8).AlignItems(ui.Start).Children(func() {
						style.meta(c, entry.Name, style.Secondary).Shrink(0)
						style.meta(c, entry.Description, style.Tertiary).Grow(1).MinWidth(0).SingleLine()
					})
				}
			})
			if body.catalogTotal > len(body.catalog) {
				style.text(c, fmt.Sprintf(i18n.T(i18n.KeyContextCatalogMore), body.catalogTotal-len(body.catalog)), style.Caption).FillWidth().Margin(0, 0, 6, 0)
			}
			paintContextBodyRuns(c, body.runs, style, true)
		case "snapshot":
			style.text(c, i18n.T(i18n.KeyContextSnapshotSupersedes), style.Caption).FillWidth().Margin(0, 0, 6, 0)
			ui.Column(c).FillWidth().MinWidth(0).Gap(8).Children(func() {
				for _, section := range body.sections {
					ui.Column(c).FillWidth().MinWidth(0).Gap(2).Children(func() {
						style.meta(c, section.Name, style.Caption).FillWidth()
						style.text(c, contextBoundedText(section.Text), style.Secondary).FillWidth().MinWidth(0)
					})
				}
			})
			// SnapshotBody intentionally replaces all content with the source's
			// sections, including dropping unknown content blocks, as the reference does.
		case "relay":
			style.meta(c, fmt.Sprintf(i18n.T(i18n.KeyContextRelayFrom), body.sender), style.Caption).FillWidth().Margin(0, 0, 6, 0)
			paintContextBodyRuns(c, body.runs, style, false)
		case "recall":
			ui.Column(c).FillWidth().MinWidth(0).Gap(2).Margin(0, 0, 8, 0).Children(func() {
				for _, recalled := range body.recalls {
					ui.Row(c).FillWidth().MinWidth(0).Gap(8).AlignItems(ui.Start).Children(func() {
						style.meta(c, recalled.Label, style.Secondary).MinWidth(0).Shrink(1)
						style.text(c, fmt.Sprintf(i18n.T(i18n.KeyContextRecallCounts), recalled.Retained, recalled.Omitted), style.Caption).Shrink(0)
						if recalled.Truncated {
							style.text(c, i18n.T(i18n.KeyContextRecallTruncated), style.Caption).Shrink(0)
						}
					})
				}
			})
			paintContextBodyRuns(c, body.runs, style, false)
		case "notice":
			paintContextBodyRuns(c, body.runs, style, false)
		default:
			paintContextBodyRuns(c, body.runs, style, false)
			paintContextSourceFields(c, body.fields, style)
		}
	})
}

// Metadata uses CSS white-space:normal; model-facing and snapshot text retain
// their exact whitespace through style.text.
func (style contextBodyStyle) meta(c *ui.Context, text string, color ui.Color) ui.Element {
	return style.text(c, contextNormalWhitespace(text), color)
}

func contextNormalWhitespace(text string) string {
	return strings.Join(strings.FieldsFunc(text, func(r rune) bool {
		return r == ' ' || r == '\t' || r == '\n' || r == '\r' || r == '\f'
	}), " ")
}

func (style contextBodyStyle) text(c *ui.Context, text string, color ui.Color) ui.Element {
	size, line := style.Size, style.LineHeight
	if size <= 0 {
		size = 11
	}
	if line <= 0 {
		line = 16
	}
	el := ui.Text(c, text).FontSize(size).FixedLineHeight(line).TextColor(color).Selectable()
	if style.Font != "" {
		el.Font(style.Font)
	}
	return el
}

func paintContextBodyRuns(c *ui.Context, runs []contextContentRun, style contextBodyStyle, unknownOnly bool) {
	previousJSON := false
	for index, run := range runs {
		if run.Raw != nil {
			paintContextJSONBlock(c.Key(fmt.Sprintf("context-run:%d", index)), run.Raw, style, previousJSON)
			previousJSON = true
		} else if !unknownOnly && run.Text != "" {
			style.text(c, contextBoundedText(run.Text), style.Secondary).FillWidth().MinWidth(0)
			previousJSON = false
		}
	}
}

func paintContextJSONBlock(c *ui.Context, raw json.RawMessage, style contextBodyStyle, previousJSON bool) {
	top := float32(4)
	if previousJSON {
		top = 0
	}
	root := ui.Column(c).FillWidth().MinWidth(0).Gap(4).Margin(top, 0, 4, 0)
	open := ui.Local(root, "open", func() bool { return false })
	root.Children(func() {
		ui.Row(c).FillWidth().MinWidth(0).AlignItems(ui.Center).Children(func() {
			label := i18n.T(i18n.KeyContextUnknownBlock)
			marker := "▸"
			if *open {
				marker = "▾"
			}
			toggle := ui.Button(c, marker+" "+label).Label(label).Role(ui.RoleDisclosure).Expanded(*open).
				Font(style.Font).FontSize(12).FixedLineHeight(18).TextColor(style.Secondary).
				BorderWidth(0).Background(ui.Transparent).Radius(8).Padding(2, 6).MinWidth(0).Shrink(1)
			if toggle.Hovered() {
				toggle.Background(style.Hover)
			}
			if toggle.Clicked() {
				*open = !*open
				c.Invalidate()
			}
		})
		if *open {
			// JsonBlock keeps its own code typography even when the containing
			// trigger notice uses the ordinary chat font.
			jsonStyle := style
			jsonStyle.Font, jsonStyle.Size, jsonStyle.LineHeight = theme.RenderPreferences().Code.Family, 11, 16
			ui.ScrollBoth(c).FillWidth().MinWidth(0).MaxHeight(217).Padding(8).
				Border(.5, style.BorderL1).Radius(16).Background(style.CodeBlock).Children(func() {
				jsonStyle.text(c, contextBoundedText(contextFormattedJSON(raw)), style.Primary).NoWrap()
			})
		}
	})
}

func paintContextSourceFields(c *ui.Context, fields []contextSourceField, style contextBodyStyle) {
	visible := false
	for _, field := range fields {
		visible = visible || field.Name != "kind"
	}
	if !visible {
		return
	}
	ui.Column(c).FillWidth().MinWidth(0).Gap(2).Margin(8, 0, 0, 0).Padding(8, 0, 0, 0).
		BorderWidth(.5, 0, 0, 0).BorderColor(style.Border).Children(func() {
		for _, field := range fields {
			if field.Name == "kind" {
				continue
			}
			ui.Row(c).FillWidth().MinWidth(0).Gap(8).AlignItems(ui.Start).Children(func() {
				style.meta(c, field.Name, style.Caption).MinWidth(96).Shrink(0)
				style.meta(c, contextFieldValue(field.Raw), style.Tertiary).Grow(1).MinWidth(0)
			})
		}
	})
}

// JavaScript length/slice count UTF-16 code units, not bytes or Unicode scalar
// values. If the cap splits a surrogate pair, native UTF-8 displays that dangling
// unit as U+FFFD; the retained event stays untouched.
func contextBoundedText(text string) string {
	total, cut, dangling := 0, len(text), false
	for offset, r := range text {
		units := 1
		if r > 0xffff {
			units = 2
		}
		if total <= contextBodyMaxUnits && total+units > contextBodyMaxUnits {
			cut = offset
			dangling = total < contextBodyMaxUnits
		}
		total += units
	}
	if total <= contextBodyMaxUnits {
		return text
	}
	prefix := text[:cut]
	if dangling {
		prefix += "\ufffd"
	}
	return prefix + "\n" + fmt.Sprintf(i18n.T(i18n.KeyContextJSONTruncated), total)
}

func contextBodyRuns(content []json.RawMessage) []contextContentRun {
	var runs []contextContentRun
	var text strings.Builder
	inText := false
	flush := func() {
		if inText {
			runs = append(runs, contextContentRun{Text: text.String()})
			text.Reset()
			inText = false
		}
	}
	for _, raw := range content {
		fields, _ := contextObjectEntries(raw)
		kind, _ := contextSourceString(fields, "type")
		if kind == "text" {
			if part, ok := contextSourceString(fields, "text"); ok {
				text.WriteString(part)
				inText = true
				continue
			}
		}
		flush()
		// Keep the entire block, including foreign fields and malformed text,
		// at its original position rather than deriving a lossy typed projection.
		runs = append(runs, contextContentRun{Raw: raw})
	}
	flush()
	return runs
}

func readContextBodyMessage(msg session.Message) ([]contextSourceField, []json.RawMessage, bool) {
	var event struct {
		Type string `json:"type"`
		Data struct {
			Source  json.RawMessage `json:"source"`
			Content json.RawMessage `json:"content"`
			Message json.RawMessage `json:"message"`
		} `json:"data"`
	}
	if json.Unmarshal(msg.Raw, &event) != nil {
		return nil, nil, false
	}
	sourceRaw, contentRaw := event.Data.Source, event.Data.Content
	if event.Type == "developer/message" {
		var message struct {
			Source  json.RawMessage `json:"source"`
			Content json.RawMessage `json:"content"`
		}
		if len(event.Data.Message) == 0 || json.Unmarshal(event.Data.Message, &message) != nil {
			return nil, nil, false
		}
		sourceRaw, contentRaw = message.Source, message.Content
	}
	source, _ := contextObjectEntries(sourceRaw)
	content, ok := contextRawArray(contentRaw)
	return source, content, ok
}

// Object.entries enumerates canonical array-index names first, numerically,
// then other own string keys in insertion order. Decoding into a Go map would
// lose that order; duplicate JSON names update their first property's value.
func contextObjectEntries(raw json.RawMessage) ([]contextSourceField, bool) {
	decoder := json.NewDecoder(bytes.NewReader(raw))
	token, err := decoder.Token()
	if err != nil || token != json.Delim('{') {
		return nil, false
	}
	var fields []contextSourceField
	positions := make(map[string]int)
	for decoder.More() {
		token, err = decoder.Token()
		name, ok := token.(string)
		if err != nil || !ok {
			return nil, false
		}
		var value json.RawMessage
		if decoder.Decode(&value) != nil {
			return nil, false
		}
		if position, exists := positions[name]; exists {
			fields[position].Raw = value
		} else {
			positions[name] = len(fields)
			fields = append(fields, contextSourceField{Name: name, Raw: value})
		}
	}
	if token, err = decoder.Token(); err != nil || token != json.Delim('}') {
		return nil, false
	}
	var extra any
	if decoder.Decode(&extra) != io.EOF {
		return nil, false
	}
	sort.SliceStable(fields, func(i, j int) bool {
		a, aIndex := contextArrayIndex(fields[i].Name)
		b, bIndex := contextArrayIndex(fields[j].Name)
		if aIndex != bIndex {
			return aIndex
		}
		return aIndex && a < b
	})
	return fields, true
}

func contextArrayIndex(name string) (uint64, bool) {
	n, err := strconv.ParseUint(name, 10, 32)
	return n, err == nil && n < math.MaxUint32 && strconv.FormatUint(n, 10) == name
}

func contextSourceRaw(fields []contextSourceField, name string) json.RawMessage {
	for _, field := range fields {
		if field.Name == name {
			return field.Raw
		}
	}
	return nil
}

func contextSourceString(fields []contextSourceField, name string) (string, bool) {
	return contextRawString(contextSourceRaw(fields, name))
}

func contextRawString(raw json.RawMessage) (string, bool) {
	raw = bytes.TrimSpace(raw)
	var text string
	ok := len(raw) > 0 && raw[0] == '"' && json.Unmarshal(raw, &text) == nil
	return text, ok
}

func contextRawArray(raw json.RawMessage) ([]json.RawMessage, bool) {
	raw = bytes.TrimSpace(raw)
	var list []json.RawMessage
	ok := len(raw) > 0 && raw[0] == '[' && json.Unmarshal(raw, &list) == nil
	return list, ok
}

func contextFormattedJSON(raw json.RawMessage) string { return contextJSONString(raw, true, 0) }

// JSON previews follow JSON.parse/stringify: IEEE-754 numbers, JavaScript own
// property order and decoded strings. The retained event remains unchanged.
func contextJSONString(raw json.RawMessage, pretty bool, depth int) string {
	raw = bytes.TrimSpace(raw)
	if len(raw) == 0 {
		return ""
	}
	if raw[0] == '{' || raw[0] == '[' {
		var values []string
		if raw[0] == '{' {
			fields, ok := contextObjectEntries(raw)
			if !ok {
				return string(raw)
			}
			for _, field := range fields {
				sep := ":"
				if pretty {
					sep = ": "
				}
				values = append(values, contextJSONStringLiteral(field.Name)+sep+contextJSONString(field.Raw, pretty, depth+1))
			}
		} else {
			items, ok := contextRawArray(raw)
			if !ok {
				return string(raw)
			}
			for _, item := range items {
				values = append(values, contextJSONString(item, pretty, depth+1))
			}
		}
		left, right := "{", "}"
		if raw[0] == '[' {
			left, right = "[", "]"
		}
		if len(values) == 0 {
			return left + right
		}
		if !pretty {
			return left + strings.Join(values, ",") + right
		}
		indent := strings.Repeat("  ", depth+1)
		return left + "\n" + indent + strings.Join(values, ",\n"+indent) + "\n" + strings.Repeat("  ", depth) + right
	}
	if text, ok := contextRawString(raw); ok {
		return contextJSONStringLiteral(text)
	}
	if number, ok := contextRawNumber(raw); ok {
		if number == "Infinity" || number == "-Infinity" {
			return "null"
		}
		return number
	}
	return string(raw)
}

func contextJSONStringLiteral(text string) string {
	var out bytes.Buffer
	encoder := json.NewEncoder(&out)
	encoder.SetEscapeHTML(false)
	_ = encoder.Encode(text)
	// JavaScript leaves the two Unicode line separators literal in JSON.
	// Match actual separators in the input without decoding literal backslashes.
	encoded := strings.TrimSuffix(out.String(), "\n")
	var result strings.Builder
	for i := 0; i < len(encoded); i++ {
		if encoded[i] == '\\' && i+1 < len(encoded) {
			if strings.HasPrefix(encoded[i:], `\u2028`) {
				result.WriteRune('\u2028')
				i += 5
				continue
			}
			if strings.HasPrefix(encoded[i:], `\u2029`) {
				result.WriteRune('\u2029')
				i += 5
				continue
			}
			result.WriteByte(encoded[i])
			i++
			result.WriteByte(encoded[i])
			continue
		}
		result.WriteByte(encoded[i])
	}
	return result.String()
}

func contextFieldValue(raw json.RawMessage) string {
	if text, ok := contextRawString(raw); ok {
		return contextBoundedText(text)
	}
	if number, ok := contextRawNumber(raw); ok {
		return contextBoundedText(number)
	}
	return contextBoundedText(contextJSONString(raw, false, 0))
}

func contextInstructionChanges(raw json.RawMessage) ([]contextInstruction, bool) {
	list, ok := contextRawArray(raw)
	if !ok || len(list) == 0 {
		return nil, false
	}
	changes := make([]contextInstruction, 0, len(list))
	seen := make(map[string]bool)
	for _, item := range list {
		entry, ok := contextObjectEntries(item)
		if !ok {
			return nil, false
		}
		path, hasPath := contextSourceString(entry, "path")
		action, hasAction := contextSourceString(entry, "action")
		if !hasPath || path == "" || !hasAction || action != "set" && action != "replace" && action != "remove" {
			return nil, false
		}
		if !seen[path] {
			digest, _ := contextSourceString(entry, "digest")
			changes = append(changes, contextInstruction{Path: path, Action: action, Digest: digest})
			seen[path] = true
		}
	}
	return changes, true
}

func contextInstructionAction(action string, baseline bool) string {
	key := i18n.KeyContextInstructionsUpdated
	if action == "remove" {
		key = i18n.KeyContextInstructionsRemoved
	} else if baseline {
		key = i18n.KeyContextInstructionsLoaded
	} else if action == "set" {
		key = i18n.KeyContextInstructionsAdded
	}
	return i18n.T(key)
}

func contextCatalogEntries(raw json.RawMessage) ([]contextCatalogEntry, int, bool) {
	list, ok := contextRawArray(raw)
	if !ok {
		return nil, 0, false
	}
	entries := make([]contextCatalogEntry, 0, min(len(list), contextBodyMaxEntries))
	for _, item := range list {
		entry, ok := contextObjectEntries(item)
		if !ok {
			return nil, 0, false
		}
		name, hasName := contextSourceString(entry, "name")
		description, hasDescription := contextSourceString(entry, "description")
		if !hasName || name == "" || !hasDescription {
			return nil, 0, false
		}
		if len(entries) < contextBodyMaxEntries {
			entries = append(entries, contextCatalogEntry{Name: name, Description: description})
		}
	}
	return entries, len(list), true
}

func contextSnapshotSections(raw json.RawMessage) ([]contextSnapshotSection, bool) {
	list, ok := contextRawArray(raw)
	if !ok || len(list) == 0 {
		return nil, false
	}
	sections := make([]contextSnapshotSection, 0, len(list))
	for _, item := range list {
		entry, ok := contextObjectEntries(item)
		if !ok {
			return nil, false
		}
		name, hasName := contextSourceString(entry, "name")
		text, hasText := contextSourceString(entry, "text")
		if !hasName || name == "" || !hasText {
			return nil, false
		}
		sections = append(sections, contextSnapshotSection{Name: name, Text: text})
	}
	return sections, true
}

func contextRecalledSessions(raw json.RawMessage) ([]contextRecallSession, bool) {
	list, ok := contextRawArray(raw)
	if !ok || len(list) == 0 {
		return nil, false
	}
	recalls := make([]contextRecallSession, 0, len(list))
	for _, item := range list {
		entry, ok := contextObjectEntries(item)
		if !ok {
			return nil, false
		}
		label, hasLabel := contextSourceString(entry, "label")
		retained, hasRetained := contextRawNumber(contextSourceRaw(entry, "retainedMessages"))
		omitted, hasOmitted := contextRawNumber(contextSourceRaw(entry, "omittedMessages"))
		truncated := bytes.TrimSpace(contextSourceRaw(entry, "truncated"))
		if !hasLabel || label == "" || !hasRetained || !hasOmitted || !bytes.Equal(truncated, []byte("true")) && !bytes.Equal(truncated, []byte("false")) {
			return nil, false
		}
		recalls = append(recalls, contextRecallSession{Label: label, Retained: retained, Omitted: omitted, Truncated: bytes.Equal(truncated, []byte("true"))})
	}
	return recalls, true
}

// Recall validates JavaScript numbers, rather than imposing integer or positive
// constraints that the reference body does not require. Formatting follows the
// ordinary String(number) decimal/exponent thresholds.
func contextRawNumber(raw json.RawMessage) (string, bool) {
	raw = bytes.TrimSpace(raw)
	if len(raw) == 0 || raw[0] != '-' && (raw[0] < '0' || raw[0] > '9') || !json.Valid(raw) {
		return "", false
	}
	number, err := strconv.ParseFloat(string(raw), 64)
	// JSON.parse also accepts numeric literals that overflow to infinity;
	// typeof still reports number, so recall does not reject those records.
	if err != nil && !math.IsInf(number, 0) {
		return "", false
	}
	if math.IsInf(number, 1) {
		return "Infinity", true
	}
	if math.IsInf(number, -1) {
		return "-Infinity", true
	}
	if number == 0 {
		return "0", true
	}
	abs := math.Abs(number)
	if abs >= 1e-6 && abs < 1e21 {
		return strconv.FormatFloat(number, 'f', -1, 64), true
	}
	text := strconv.FormatFloat(number, 'e', -1, 64)
	mantissa, exponent, _ := strings.Cut(text, "e")
	n, _ := strconv.Atoi(exponent)
	sign := ""
	if n >= 0 {
		sign = "+"
	}
	return mantissa + "e" + sign + strconv.Itoa(n), true
}
