package session

import (
	"encoding/json"
	"fmt"
	"regexp"
	"sort"
	"strings"
)

const GroupThreadReplyCommand = "/dshbot-thread-reply"
const groupReplyPrefix = "[DSHBOT_THREAD_REPLY:"

var groupThreadID = regexp.MustCompile(`^[A-Za-z0-9][A-Za-z0-9._:-]{0,127}$`)
var groupPass = regexp.MustCompile(`(?i)^\(?\s*pass\s*\)?\.?$`)

// Match the original JavaScript /timeout|timed.?out/: dot excludes line
// terminators and matches one UTF-16 code unit (the source has no u flag).
var groupTimeout = regexp.MustCompile(`timeout|timed[^\r\n\x{2028}\x{2029}\x{10000}-\x{10FFFF}]?out`)

type GroupMember struct {
	ID, Name, Title             string
	RuntimeStatus, RuntimeError string
}
type GroupMention struct{ Label, Value string }
type GroupMessage struct {
	ID, ThreadID, BotID, Speaker, Text, State string
	At                                        float64
	User, Late                                bool
	Original                                  Message
}
type GroupThread struct {
	ID       string
	Messages []GroupMessage
}

func GroupMentions(members []GroupMember, query string) []GroupMention {
	q := strings.ToLower(strings.TrimSpace(query))
	var out []GroupMention
	if q == "" || strings.Contains("everyone", q) || strings.Contains("all", q) {
		out = append(out, GroupMention{"everyone", "everyone"})
	}
	for _, m := range members {
		name := strings.TrimSpace(m.Name)
		if name == "" {
			name = m.ID
		}
		label := strings.TrimSpace(m.Title)
		if label == "" {
			label = name
		}
		if name != "" && (q == "" || strings.Contains(strings.ToLower(label), q) || strings.Contains(strings.ToLower(name), q) || strings.Contains(strings.ToLower(m.ID), q)) {
			out = append(out, GroupMention{label, name})
		}
	}
	return out
}
func EncodeGroupThreadReply(id, text string) (string, error) {
	if !groupThreadID.MatchString(id) {
		return "", fmt.Errorf("invalid group thread id")
	}
	return groupReplyPrefix + id + "]\n" + text, nil
}
func ParseGroupThreadEnvelope(text string) (string, string) {
	if strings.HasPrefix(text, groupReplyPrefix) {
		if end := strings.Index(text, "]\n"); end >= len(groupReplyPrefix) {
			id := text[len(groupReplyPrefix):end]
			if groupThreadID.MatchString(id) {
				return id, text[end+2:]
			}
		}
	}
	return "", text
}
func ParseGroupThreadReply(line string) (id, text string, claimed bool, err error) {
	line = strings.TrimSpace(line)
	if line != GroupThreadReplyCommand && !strings.HasPrefix(line, GroupThreadReplyCommand+" ") && !strings.HasPrefix(line, GroupThreadReplyCommand+"\n") && !strings.HasPrefix(line, GroupThreadReplyCommand+"\t") {
		return "", "", false, nil
	}
	rest := strings.TrimSpace(strings.TrimPrefix(line, GroupThreadReplyCommand))
	parts := strings.Fields(rest)
	if len(parts) == 0 || !groupThreadID.MatchString(parts[0]) {
		return "", "", true, fmt.Errorf("invalid group thread reply; select a thread again")
	}
	return parts[0], strings.TrimSpace(strings.TrimPrefix(rest, parts[0])), true, nil
}

// A copy preserves the captured Host/session ownership while forcing the original
// room-reply queue semantics, regardless of the ordinary composer's steer setting.
func (t *ComposerTarget) GroupReplyTarget() *ComposerTarget {
	copy := *t
	copy.mode = PromptModeQueue
	return &copy
}

type groupObject map[string]any

func groupObj(v any) groupObject { m, _ := v.(map[string]any); return m }
func groupStr(v any) string      { s, _ := v.(string); return s }
func groupFirst(v ...any) string {
	for _, x := range v {
		if s := groupStr(x); s != "" {
			return s
		}
	}
	return ""
}
func groupThread(v ...groupObject) string {
	for _, m := range v {
		for _, k := range []string{"threadId", "thread", "groupThreadId", "owningThreadId"} {
			if s := groupStr(m[k]); groupThreadID.MatchString(s) {
				return s
			}
		}
	}
	return ""
}
func groupTexts(v any) []string {
	var out []string
	xs, _ := v.([]any)
	for _, x := range xs {
		m := groupObj(x)
		if m["type"] == "text" {
			out = append(out, groupStr(m["text"]))
		} else if m["type"] == "tool-result" {
			out = append(out, groupTexts(m["content"])...)
		}
	}
	return out
}
func groupVisible(s string) string {
	lines := strings.Split(strings.TrimRight(s, " \t\r\n"), "\n")
	for i := range lines {
		lines[i] = strings.TrimRight(lines[i], " \t")
	}
	if len(lines) > 0 && strings.HasPrefix(strings.ToUpper(lines[len(lines)-1]), "NEXT:") {
		lines = lines[:len(lines)-1]
	}
	s = strings.TrimSpace(strings.Join(lines, "\n"))
	if groupPass.MatchString(s) {
		return ""
	}
	return s
}
func groupErrorContent(v any) bool {
	xs, _ := v.([]any)
	for _, x := range xs {
		m := groupObj(x)
		if m["isError"] == true || groupErrorContent(m["content"]) {
			return true
		}
	}
	return false
}
func groupResultState(d groupObject, content any) string {
	status := strings.ToLower(strings.Join([]string{groupStr(d["status"]), groupStr(d["state"]), groupStr(groupObj(d["result"])["status"]), groupStr(groupObj(d["message"])["status"]), groupStr(groupObj(d["error"])["code"]), groupStr(groupObj(d["error"])["kind"])}, " "))
	if strings.Contains(status, "capped") || strings.Contains(status, "limit") || strings.Contains(status, "maximum") {
		return "capped"
	}
	if groupTimeout.MatchString(status) {
		return "timeout"
	}
	for _, rule := range []struct {
		state string
		words []string
	}{{"held", []string{"held", "paused", "deferred"}}, {"stopped", []string{"stopped", "cancelled", "canceled", "aborted"}}, {"pending", []string{"pending", "waiting", "approval", "question"}}, {"running", []string{"running", "started"}}, {"failed", []string{"failed", "error"}}} {
		for _, word := range rule.words {
			if strings.Contains(status, word) {
				return rule.state
			}
		}
	}
	if d["error"] != nil && d["error"] != false || groupErrorContent(content) {
		return "failed"
	}
	bodies := groupTexts(content)
	visible := false
	for _, b := range bodies {
		if s := strings.TrimSpace(b); s != "" && !groupPass.MatchString(s) {
			visible = true
		}
	}
	if !visible {
		return "pass"
	}
	for _, word := range []string{"pass", "complete", "success", "done"} {
		if strings.Contains(status, word) {
			return "pass"
		}
	}
	return "settled"
}

// ProjectGroupThreads follows the e9 dshbot client event contract. Only user/relay
// inputs, ask_participant results and authenticated plugin late-result carriers
// enter the room transcript; coordinator prose/tool diagnostics are not replies.
func ProjectGroupThreads(messages []Message, members []GroupMember) []GroupThread {
	type event struct {
		msg        Message
		root, data groupObject
		kind       string
		at         float64
	}
	var events []event
	for _, m := range messages {
		var root map[string]any
		if json.Unmarshal(m.Raw, &root) != nil {
			continue
		}
		at, _ := root["time"].(float64)
		events = append(events, event{m, root, groupObj(root["data"]), groupStr(root["type"]), at})
	}
	names := map[string]GroupMember{}
	for _, m := range members {
		names[m.ID] = m
		names[m.Name] = m
	}
	identity := func(id string) (string, string) {
		m, ok := names[id]
		if !ok {
			return id, id
		}
		name := m.Title
		if name == "" {
			name = m.Name
		}
		if name == "" {
			name = m.ID
		}
		return m.ID, name
	}
	callThreads := map[string]string{}
	for _, e := range events {
		if e.kind == "tool/call" && e.data["name"] == "ask_participant" {
			a := groupObj(e.data["arguments"])
			if raw, ok := e.data["arguments"].(string); ok {
				json.Unmarshal([]byte(raw), &a)
			}
			if th := groupThread(a, e.data, e.root); th != "" {
				callThreads[groupFirst(e.data["callId"], e.data["toolCallId"], e.data["id"])] = th
			}
		}
	}
	type call struct{ bot, name, thread string }
	calls := map[string]call{}
	seenLate := map[string]bool{}
	admitted := map[string]bool{}
	var out []GroupMessage
	current, legacy := "", ""
	sequence := 0
	previousAt := float64(0)
	havePrevious := false
	assign := func(th string, user bool, at float64) string {
		if th != "" {
			legacy = ""
		} else {
			if legacy == "" || (user && (!havePrevious || at-previousAt > 15*60*1000)) {
				legacy = fmt.Sprintf("legacy-%d", sequence)
				sequence++
			}
			th = legacy
		}
		previousAt = at
		havePrevious = true
		return th
	}
	for _, e := range events {
		d := e.data
		source := groupObj(d["source"])
		var sources []groupObject
		if e.kind == "user/message" {
			sources = append(sources, source)
			if source == nil {
				sources = append(sources, groupObj(groupObj(d["message"])["source"]))
			}
		}
		if e.kind == "agent/inbox/spliced" {
			rows, _ := d["inserted"].([]any)
			for _, v := range rows {
				m := groupObj(v)
				s := groupObj(m["source"])
				if s == nil {
					s = groupObj(groupObj(m["data"])["source"])
				}
				sources = append(sources, s)
			}
		}
		late := false
		for _, s := range sources {
			if s["kind"] != "plugin" || s["plugin"] != "dshbot" || s["form"] != "notice" {
				continue
			}
			p := groupObj(s["dshbotGroupLateResult"])
			th := groupThread(p)
			bot := groupStr(p["botId"])
			callID := groupFirst(p["toolCallId"], p["callId"])
			texts, _ := p["texts"].([]any)
			if th == "" || bot == "" || callID == "" || len(texts) == 0 {
				continue
			}
			late = true
			carrier := groupStr(p["carrierId"])
			if carrier != "" && seenLate[carrier] {
				continue
			}
			seenLate[carrier] = true
			bid, name := identity(bot)
			if n := groupStr(p["name"]); n != "" {
				name = n
			}
			visibleCount := 0
			for i, v := range texts {
				if visibleCount >= 2 {
					break
				}
				body := groupVisible(groupStr(v))
				if body == "" {
					continue
				}
				visibleCount++
				out = append(out, GroupMessage{ID: fmt.Sprintf("%g:late:%d", e.msg.Seq, i), ThreadID: assign(th, false, e.at), BotID: bid, Speaker: name, Text: body, State: "settled", At: e.at, Late: true, Original: e.msg})
			}
		}
		if late || e.kind == "agent/inbox/spliced" {
			continue
		}
		if e.kind == "user/message" && (source["kind"] == "user" || source["kind"] == "plugin" && source["plugin"] == "dshbot" && source["form"] == "relay") {
			if id := groupStr(source["rpcId"]); id != "" {
				admitted[id] = true
			}
			th, body := ParseGroupThreadEnvelope(strings.Join(groupTexts(d["content"]), ""))
			if th == "" {
				th = groupThread(e.root, d, source)
			}
			current = assign(th, true, e.at)
			out = append(out, GroupMessage{ID: fmt.Sprintf("%g:user", e.msg.Seq), ThreadID: current, Text: body, Speaker: "You", State: "settled", At: e.at, User: true, Original: e.msg})
			continue
		}
		if e.kind == "tool/call" && d["name"] == "ask_participant" {
			args := groupObj(d["arguments"])
			if raw, ok := d["arguments"].(string); ok {
				json.Unmarshal([]byte(raw), &args)
			}
			id := groupFirst(d["callId"], d["toolCallId"], d["id"])
			bid, name := identity(groupStr(args["botId"]))
			th := callThreads[id]
			if th == "" {
				th = current
			}
			if th == "" {
				th = "legacy-0"
			}
			calls[id] = call{bid, name, th}
			continue
		}
		if e.kind != "tool/result" {
			continue
		}
		message := groupObj(d["message"])
		ms := groupObj(message["source"])
		id := groupFirst(ms["callId"], ms["toolCallId"], message["toolCallId"], d["toolCallId"], d["callId"])
		call, ok := calls[id]
		if !ok {
			continue
		}
		th := groupThread(d, message, ms)
		if th != "" && th != call.thread {
			continue
		}
		content := message["content"]
		if content == nil {
			content = d["content"]
		}
		state := groupResultState(d, content)
		var bodies []string
		if state == "settled" || state == "replied" {
			for _, b := range groupTexts(content) {
				if b = groupVisible(b); b != "" {
					bodies = append(bodies, b)
				}
			}
		}
		if len(bodies) == 0 {
			bodies = []string{""}
		}
		for i, body := range bodies {
			out = append(out, GroupMessage{ID: fmt.Sprintf("%g:result:%d", e.msg.Seq, i), ThreadID: call.thread, BotID: call.bot, Speaker: call.name, Text: body, State: state, At: e.at, Original: e.msg})
		}
		if state == "settled" && bodies[0] != "" {
			declared := th
			if declared == "" {
				declared = callThreads[id]
			}
			assign(declared, false, e.at)
		}
	}
	for _, m := range messages {
		if p := m.Submission; p != nil && !admitted[p.RequestID] {
			th, text := ParseGroupThreadEnvelope(p.Text)
			if th == "" {
				th = "pending-" + p.RequestID
			}
			out = append(out, GroupMessage{ID: "pending:" + p.RequestID, ThreadID: th, Text: text, Speaker: "You", State: "pending", At: previousAt + float64(p.Order) + 1, User: true, Original: m})
		}
	}
	var threads []GroupThread
	index := map[string]int{}
	for _, m := range out {
		n, ok := index[m.ThreadID]
		if !ok {
			n = len(threads)
			index[m.ThreadID] = n
			threads = append(threads, GroupThread{ID: m.ThreadID})
		}
		threads[n].Messages = append(threads[n].Messages, m)
	}
	sort.SliceStable(threads, func(i, j int) bool {
		a, b := threads[i].Messages, threads[j].Messages
		return a[len(a)-1].At < b[len(b)-1].At
	})
	return threads
}

// Inbox-spliced carriers have no ordinary text row, but must survive folding so
// room late replies remain visible after follow/history reloads.
func classifyGroupLateCarrier(raw json.RawMessage) (Message, bool) {
	var root map[string]any
	if json.Unmarshal(raw, &root) != nil || root["type"] != "agent/inbox/spliced" {
		return Message{}, false
	}
	d := groupObj(root["data"])
	rows, _ := d["inserted"].([]any)
	for _, v := range rows {
		m := groupObj(v)
		s := groupObj(m["source"])
		if s == nil {
			s = groupObj(groupObj(m["data"])["source"])
		}
		if s["kind"] == "plugin" && s["plugin"] == "dshbot" && s["form"] == "notice" && groupObj(s["dshbotGroupLateResult"]) != nil {
			seq, _ := root["seq"].(float64)
			at, _ := root["time"].(float64)
			return Message{Kind: MessageKind("group-carrier"), Role: "system", Seq: seq, EventType: "agent/inbox/spliced", Raw: append(json.RawMessage(nil), raw...), EventTime: at}, true
		}
	}
	return Message{}, false
}

func GroupAttachmentNames(message Message) []string {
	var root map[string]any
	if json.Unmarshal(message.Raw, &root) != nil {
		return nil
	}
	content, _ := groupObj(root["data"])["content"].([]any)
	var names []string
	for _, v := range content {
		block := groupObj(v)
		if block["type"] != "file" {
			continue
		}
		attachment := groupObj(block["attachment"])
		name := groupFirst(attachment["name"], block["name"])
		if name == "" {
			name = "Attachment"
		}
		names = append(names, name)
	}
	return names
}
