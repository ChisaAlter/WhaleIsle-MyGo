package chat

import (
	"encoding/json"
	"fmt"
	"sort"
	"strings"
	"time"

	"github.com/egoist/mygo/ui"
	"whaleisle/internal/store/session"
	"whaleisle/internal/ui/chrome"
	"whaleisle/internal/ui/i18n"
	"whaleisle/internal/ui/theme"
)

const transcriptControlKind session.MessageKind = "transcript-control"

type transcriptControl struct {
	key, label string
	open       bool
	running    bool
	members    []string
}
type transcriptTurn struct {
	end          *session.Message
	done         bool
	final        int
	firstProcess int
	middleInput  bool
	toolSteps    map[float64]bool
	members      []string
}

func transcriptEventCoordinates(msg session.Message) (turn, step float64, known bool) {
	return memoizedTranscriptCoordinates(msg)
}

// decodeTranscriptCoordinates reads one row's turn/step from its durable raw,
// falling back to the tool call it projects. See render_memo.go for the memo.
func decodeTranscriptCoordinates(msg session.Message) (turn, step float64, known bool) {
	var e struct {
		Data struct {
			Turn *float64 `json:"turn"`
			Step float64  `json:"step"`
		} `json:"data"`
	}
	_ = json.Unmarshal(msg.Raw, &e)
	if e.Data.Turn != nil {
		return *e.Data.Turn, e.Data.Step, true
	}
	if msg.Tool != nil && msg.Tool.Call != nil {
		return transcriptEventCoordinates(*msg.Tool.Call)
	}
	return 0, 0, false
}

func transcriptProcess(msg session.Message) bool {
	return msg.Kind == session.MessageKindThinking || msg.Kind == session.MessageKindTool || msg.Kind == session.MessageKindCompaction || msg.Kind == session.MessageKindCommand || msg.Role == "assistant" && msg.Kind == session.MessageKindText
}

func transcriptMessageTurnKeys(msgs []session.Message) ([]float64, []float64, []bool) {
	keys := make([]float64, len(msgs))
	known := make([]bool, len(msgs))
	steps := make([]float64, len(msgs))
	byStart := map[float64]float64{}
	for i, msg := range msgs {
		turn, step, ok := transcriptEventCoordinates(msg)
		keys[i], steps[i], known[i] = turn, step, ok
		if ok && msg.TurnSeq > 0 {
			byStart[msg.TurnSeq] = turn
		}
	}
	for i, msg := range msgs {
		if !known[i] && msg.TurnSeq > 0 {
			if turn, ok := byStart[msg.TurnSeq]; ok {
				keys[i], known[i] = turn, true
			}
		}
	}
	return keys, steps, known
}

// projectTranscript changes presentation only. Journal rows and accounting stay
// intact, and stopped/running turns or intervening user input remain visible.
func projectTranscript(msgs []session.Message, m Model) ([]session.Message, map[string]transcriptControl) {
	return projectTranscriptWithLastInput(msgs, m, false, 0)
}

func projectTranscriptWithLastInput(msgs []session.Message, m Model, hasLastInput bool, overrideLastInput float64) ([]session.Message, map[string]transcriptControl) {
	mode := m.Preferences.TranscriptView
	if mode == "" {
		mode = "standard"
	}
	turns := map[float64]*transcriptTurn{}
	keys, steps, known := transcriptMessageTurnKeys(msgs)
	for i, msg := range msgs {
		turn, step, ok := keys[i], steps[i], known[i]
		if !ok {
			continue
		}
		t := turns[turn]
		if t == nil {
			t = &transcriptTurn{final: -1, firstProcess: -1}
			turns[turn] = t
		}
		if msg.Kind == session.MessageKindTool {
			if t.toolSteps == nil {
				t.toolSteps = make(map[float64]bool)
			}
			t.toolSteps[step] = true
		}
		if msg.EventType == "turn/end" {
			copy := msg
			t.end = &copy
			t.done = turnStatusKind(msg) == "done"
		}
		if msg.Role == "assistant" && msg.Kind == session.MessageKindText && !msg.Streaming && strings.TrimSpace(msg.Body) != "" {
			t.final = i
		}
	}
	// Each turn's tool-step index rejects the same final candidate without
	// rescanning the full transcript once per turn.
	for _, t := range turns {
		if t.final >= 0 && t.toolSteps[steps[t.final]] {
			t.final = -1
		}
	}
	for i, msg := range msgs {
		if !known[i] {
			continue
		}
		t := turns[keys[i]]
		if transcriptProcess(msg) && i != t.final {
			if t.firstProcess < 0 {
				t.firstProcess = i
			}
			// Keep occurrence order and duplicate/empty IDs exactly as recorded.
			t.members = append(t.members, msg.ID)
		}
		if msg.Role == "user" && t.firstProcess >= 0 {
			t.middleInput = true
		}
	}
	out := make([]session.Message, 0, len(msgs))
	controls := map[string]transcriptControl{}
	scope := m.Snapshot.PlanHistoryBinding + ":" + m.Snapshot.SelectedID
	lastInput := overrideLastInput
	if !hasLastInput {
		for _, msg := range msgs {
			if msg.Role == "user" && msg.Kind == session.MessageKindText {
				lastInput = max(lastInput, msg.Seq)
			}
		}
	}
	settled := func(t *transcriptTurn) bool {
		return t != nil && t.end != nil && (m.Preferences.CollapseTiming != "next-input" || lastInput > t.end.Seq)
	}
	wholeOpen := map[float64]bool{}
	for turn, t := range turns {
		eligible := mode != "verbose" && settled(t) && t.done && !t.middleInput && t.firstProcess >= 0
		wholeOpen[turn] = !eligible || workflowExpanded(m.ThinkingExpanded, fmt.Sprintf("transcript-turn:%s:%.0f", scope, turn), false)
	}
	for i := 0; i < len(msgs); {
		msg := msgs[i]
		t := turns[keys[i]]
		if known[i] && t.firstProcess == i && mode != "verbose" && settled(t) && t.done && !t.middleInput {
			key := fmt.Sprintf("transcript-turn:%s:%.0f", scope, keys[i])
			control := transcriptControl{key: key, label: formatTurnStatus(t.end.Body), open: wholeOpen[keys[i]], members: t.members}
			controls[key] = control
			out = append(out, session.Message{ID: key, Seq: msg.Seq, Kind: transcriptControlKind, Body: control.label})
		}
		if known[i] && !wholeOpen[keys[i]] && transcriptProcess(msg) && i != t.final {
			i++
			continue
		}
		groupable := msg.Kind == session.MessageKindThinking || msg.Kind == session.MessageKindTool
		if !groupable || mode == "verbose" || mode == "detailed" && (!known[i] || !settled(t)) {
			out = append(out, msg)
			i++
			continue
		}
		j := i + 1
		for j < len(msgs) && (msgs[j].Kind == session.MessageKindThinking || msgs[j].Kind == session.MessageKindTool) && known[j] == known[i] && keys[j] == keys[i] {
			j++
		}
		group := msgs[i:j]
		key := fmt.Sprintf("transcript-group:%s:%s:%.0f", scope, group[len(group)-1].ID, group[len(group)-1].Seq)
		deferred := known[i] && t.end != nil && !settled(t)
		open := workflowExpanded(m.ThinkingExpanded, key, deferred)
		running := !known[i] || t.end == nil
		label := processGroupLabel(group, running, mode == "standard")
		controls[key] = transcriptControl{key: key, label: label, open: open, running: running, members: inspectorMessageIDs(group)}
		out = append(out, session.Message{ID: key, Seq: group[0].Seq, Kind: transcriptControlKind, Body: label})
		if open {
			out = append(out, group...)
		}
		i = j
	}
	return out, controls
}

func processGroupLabel(group []session.Message, running, details bool) string {
	counts := map[ToolKind]int{}
	order := []ToolKind{}
	var current *session.ToolCallView
	thinking := ""
	for _, msg := range group {
		if msg.Kind == session.MessageKindThinking && msg.Streaming {
			thinking = msg.Body
		}
		if msg.Kind != session.MessageKindTool {
			continue
		}
		name := msg.Body
		if msg.Tool != nil {
			name = msg.Tool.Name
			if msg.Tool.State == "running" || msg.Tool.State == "preparing" {
				current = msg.Tool
			}
		}
		kind, _ := ClassifyToolName(name)
		if counts[kind] == 0 {
			order = append(order, kind)
		}
		counts[kind]++
	}
	if running {
		if current != nil {
			kind, _ := ClassifyToolName(current.Name)
			label := processCategory(kind, false)
			if details && strings.TrimSpace(current.Arguments) != "" {
				if detail := toolRunningDetail(current.Name, current.Arguments); detail != "" {
					label += " · " + detail
				}
			}
			return label
		}
		label := i18n.T(i18n.KeyChatProcessAnalyzing)
		if details && thinking != "" {
			label += " · " + thinkingPreview(thinking, 120)
		}
		return label
	}
	sort.SliceStable(order, func(i, j int) bool { return counts[order[i]] > counts[order[j]] })
	if len(order) == 0 {
		return i18n.T(i18n.KeyChatProcessComplete)
	}
	var parts []string
	for _, kind := range order[:min(3, len(order))] {
		parts = append(parts, processCategory(kind, true))
	}
	separator := "，"
	if i18n.Locale() == i18n.LocaleEN {
		separator = ", "
	}
	label := strings.Join(parts, separator)
	if len(order) > 3 {
		label += " " + i18n.T(i18n.KeyChatProcessMore)
	}
	return label
}

// toolRunningDetail mirrors the Original's liveToolDetail
// (ui-chat/src/client/conversation-nodes/process-activity.ts): the fold row
// surfaces one semantic argument, never the raw JSON payload, so a running
// ask_user_question call reads "等待你的操作 · Pick a color?" instead of
// dumping the arguments object into user-visible copy.
var liveToolDetailKeys = []string{
	"title", "description", "objective", "task", "task_name", "name",
	"question", "questions", "prompt", "message", "command", "cmd",
	"queries", "query", "pattern", "url", "uri", "file_path", "path",
	"target", "action", "status",
}

const liveToolDetailMaxChars = 160

func toolRunningDetail(toolName, arguments string) string {
	var payload map[string]any
	if json.Unmarshal([]byte(strings.TrimSpace(arguments)), &payload) != nil {
		return ""
	}
	for _, key := range liveToolDetailKeys {
		value, ok := payload[key]
		if !ok {
			continue
		}
		text := ""
		if key == "questions" {
			list, _ := value.([]any)
			for _, item := range list {
				entry, _ := item.(map[string]any)
				if q, ok := entry["question"].(string); ok && strings.TrimSpace(q) != "" {
					text = q
					break
				}
			}
		} else if s, ok := value.(string); ok {
			text = s
		}
		text = strings.Join(strings.Fields(text), " ")
		if text == "" {
			continue
		}
		if runes := []rune(text); len(runes) > liveToolDetailMaxChars {
			text = string(runes[:liveToolDetailMaxChars]) + "…"
		}
		return text
	}
	return toolName
}

// processCategory returns the running or settled summary for one tool family.
func processCategory(kind ToolKind, done bool) string {
	keys := map[ToolKind][2]string{
		ToolKindRead:     {i18n.KeyChatProcessReading, i18n.KeyChatProcessRead},
		ToolKindWrite:    {i18n.KeyChatProcessEditing, i18n.KeyChatProcessEdited},
		ToolKindSearch:   {i18n.KeyChatProcessSearching, i18n.KeyChatProcessSearched},
		ToolKindWeb:      {i18n.KeyChatProcessBrowsing, i18n.KeyChatProcessBrowsed},
		ToolKindBash:     {i18n.KeyChatProcessRunning, i18n.KeyChatProcessRan},
		ToolKindTodo:     {i18n.KeyChatProcessPlanning, i18n.KeyChatProcessPlanned},
		ToolKindAsk:      {i18n.KeyChatProcessWaiting, i18n.KeyChatProcessAsked},
		ToolKindSubagent: {i18n.KeyChatProcessDelegating, i18n.KeyChatProcessDelegated},
		ToolKindGeneric:  {i18n.KeyChatProcessTools, i18n.KeyChatProcessUsedTools},
	}
	at := 0
	if done {
		at = 1
	}
	pair, ok := keys[kind]
	if !ok {
		pair = keys[ToolKindGeneric]
	}
	return i18n.T(pair[at])
}

// transcriptControlView renders the turn-process / process-group disclosure
// matching TurnProcessNodeView.module.css: flush-left muted label + 14x14 trailing chevron.
// transcriptControlView paints the whole-turn fold row, matching the
// Original's TurnProcessNodeView: a full-width borderless 33px row whose
// tertiary label eases to secondary on hover over 100ms, and whose 14px
// chevron rotates 180deg over the same 100ms ease rather than swapping
// glyphs. Height, font size and line height ride the chat content font
// delta; the Original's padding-bottom is 8.
func transcriptControlView(c *ui.Context, control transcriptControl, m Model, colors theme.Colors) {
	delta := theme.RenderPreferences().Chat.Size - 14
	muted, secondary := chrome.Color(colors.Muted), chrome.Color(colors.Secondary)
	if strings.HasPrefix(control.key, "transcript-group:") {
		transcriptGroupView(c, control, m, muted, secondary, delta)
		return
	}
	ui.Row(c).FillWidth().MinWidth(0).MinHeight(33 + delta).AlignItems(ui.Center).Children(func() {
		btn := ui.Button(c.Key("transcript-control:"+control.key), "").Label(control.label).
			FillWidth().MinWidth(0).Height(33+delta).Padding(0, 0, 8, 0).Gap(4).BorderWidth(0).
			Background(ui.Transparent).Justify(ui.Start).AlignItems(ui.Center)
		target := float32(0)
		if btn.Hovered() {
			target = 1
		}
		textColor := muted.Mix(secondary, btn.AnimateWith("transcript-hover", target, 100*time.Millisecond, contextColorEase))
		btn.Children(func() {
			ui.Text(c, control.label).FontSize(14 + delta).TextColor(textColor).SingleLine().PassThrough()
			transcriptFoldChevron(c, control.open, textColor)
		})
		if btn.Clicked() {
			workflowToggle(m.ThinkingExpanded, control.key, false)
			if control.open && m.ThinkingExpanded != nil {
				// A whole-turn close also closes its loaded reasoning/tool detail choices.
				for _, id := range control.members {
					delete(*m.ThinkingExpanded, id)
				}
			}
		}
	})
}

// transcriptFoldChevron paints the fold row's marker: one glyph rotates
// 180deg over 100ms ease-in-out while the row opens (the Original's
// .chevron transition), or flips immediately under reduced motion.
func transcriptFoldChevron(c *ui.Context, open bool, color ui.Color) {
	chev := ui.Icon(c, todoChevronDown).Size(14, 14).TextColor(color).PassThrough()
	st := ui.Local(chev, "fold-open", func() struct {
		open  bool
		since time.Time
	} {
		return struct {
			open  bool
			since time.Time
		}{open: open}
	})
	angle := float32(0)
	if open {
		angle = 180
	}
	if st.open != open {
		st.open, st.since = open, c.Now()
	}
	if !st.since.IsZero() && !c.Preferences().ReduceMotion {
		if t := float32(c.Now().Sub(st.since)) / float32(100*time.Millisecond); t < 1 {
			p := ui.EaseInOut(t) * 180
			if open {
				angle = p
			} else {
				angle = 180 - p
			}
			c.After(16 * time.Millisecond)
		}
	}
	if angle != 0 {
		chev.Rotate(angle)
	}
}

// transcriptGroupView paints a second-level process-group row, matching the
// Original's ChatGroupSeat `.title`: a compact content-width row — 16px leading
// box, 6px gap, content-size label on a 24px line — whose tertiary color eases
// to secondary on hover over 100ms. The activity icon and chevron share the
// leading box and cross-fade over the same 100ms like the Original's
// activityIcon/chevron pair: the chevron appears when the row opens, hovers or
// takes focus; expanded rows also keep an 8px bottom padding. Metrics ride
// the chat content font delta.
func transcriptGroupView(c *ui.Context, control transcriptControl, m Model, muted, secondary ui.Color, delta float32) {
	bottom := float32(0)
	if control.open {
		bottom = 8
	}
	btn := ui.Button(c.Key("transcript-control:"+control.key), "").Label(control.label).MinWidth(0).AlignSelf(ui.Start).
		Padding(0, 0, bottom, 0).Gap(6).BorderWidth(0).Background(ui.Transparent).Justify(ui.Start).AlignItems(ui.Center)
	textColor := muted.Mix(secondary, btn.AnimateWith("group-hover", boolFloat(btn.Hovered()), 100*time.Millisecond, contextColorEase))
	leadingSize, glyph := float32(16)+delta, float32(14)+delta
	btn.Children(func() {
		lead := ui.Box(c).Size(leadingSize, leadingSize).Shrink(0)
		fade := lead.AnimateWith("group-chev", boolFloat(btn.Hovered() || control.open), 100*time.Millisecond, contextColorEase)
		lead.Children(func() {
			ui.Icon(c, disclosureIconData).Absolute().Size(glyph, glyph).Left((leadingSize - glyph) / 2).Top((leadingSize - glyph) / 2).TextColor(textColor).Opacity(1 - fade).PassThrough()
			chev := ui.Icon(c, toolIconChevronDown).Absolute().Size(glyph, glyph).Left((leadingSize - glyph) / 2).Top((leadingSize - glyph) / 2).TextColor(textColor).Opacity(fade).PassThrough()
			if control.open {
				chev.Rotate(180)
			}
		})
		ui.Text(c, control.label).FontSize(14 + delta).TextColor(textColor).SingleLine().PassThrough()
	})
	if btn.Clicked() {
		workflowToggle(m.ThinkingExpanded, control.key, false)
	}
}
