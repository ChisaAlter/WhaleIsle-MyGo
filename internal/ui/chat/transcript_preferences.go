package chat

import (
	"encoding/json"
	"fmt"
	"sort"
	"strings"

	"github.com/egoist/mygo/ui"
	"whaleisle/internal/store/session"
	"whaleisle/internal/ui/i18n"
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
	final        int
	firstProcess int
	middleInput  bool
}

func transcriptEventCoordinates(msg session.Message) (turn, step float64, known bool) {
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

// projectTranscript changes presentation only. Journal rows and accounting stay
// intact, and stopped/running turns or intervening user input remain visible.
func projectTranscript(msgs []session.Message, m Model) ([]session.Message, map[string]transcriptControl) {
	mode := m.Preferences.TranscriptView
	if mode == "" {
		mode = "standard"
	}
	turns := map[float64]*transcriptTurn{}
	keys := make([]float64, len(msgs))
	known := make([]bool, len(msgs))
	steps := make([]float64, len(msgs))
	byStart := map[float64]float64{}
	for _, msg := range msgs {
		turn, _, ok := transcriptEventCoordinates(msg)
		if ok && msg.TurnSeq > 0 {
			byStart[msg.TurnSeq] = turn
		}
	}
	for i, msg := range msgs {
		turn, step, ok := transcriptEventCoordinates(msg)
		if !ok && msg.TurnSeq > 0 {
			turn, ok = byStart[msg.TurnSeq]
		}
		keys[i], steps[i], known[i] = turn, step, ok
		if !ok {
			continue
		}
		t := turns[turn]
		if t == nil {
			t = &transcriptTurn{final: -1, firstProcess: -1}
			turns[turn] = t
		}
		if msg.EventType == "turn/end" {
			copy := msg
			t.end = &copy
		}
		if msg.Role == "assistant" && msg.Kind == session.MessageKindText && !msg.Streaming && strings.TrimSpace(msg.Body) != "" {
			t.final = i
		}
	}
	for turn, t := range turns {
		if t.final >= 0 {
			for i, msg := range msgs {
				if known[i] && keys[i] == turn && steps[i] == steps[t.final] && msg.Kind == session.MessageKindTool {
					t.final = -1
					break
				}
			}
		}
	}
	for i, msg := range msgs {
		if !known[i] {
			continue
		}
		t := turns[keys[i]]
		if transcriptProcess(msg) && i != t.final && t.firstProcess < 0 {
			t.firstProcess = i
		}
		if msg.Role == "user" && t.firstProcess >= 0 {
			t.middleInput = true
		}
	}
	out := make([]session.Message, 0, len(msgs))
	controls := map[string]transcriptControl{}
	scope := m.Snapshot.PlanHistoryBinding + ":" + m.Snapshot.SelectedID
	wholeOpen := map[float64]bool{}
	for turn, t := range turns {
		eligible := mode != "verbose" && t.end != nil && turnStatusKind(*t.end) == "done" && !t.middleInput && t.firstProcess >= 0
		wholeOpen[turn] = !eligible || workflowExpanded(m.ThinkingExpanded, fmt.Sprintf("transcript-turn:%s:%.0f", scope, turn), false)
	}
	for i := 0; i < len(msgs); {
		msg := msgs[i]
		t := turns[keys[i]]
		if known[i] && t.firstProcess == i && mode != "verbose" && t.end != nil && turnStatusKind(*t.end) == "done" && !t.middleInput {
			key := fmt.Sprintf("transcript-turn:%s:%.0f", scope, keys[i])
			members := []string{}
			for j, item := range msgs {
				if known[j] && keys[j] == keys[i] && transcriptProcess(item) && j != t.final {
					members = append(members, item.ID)
				}
			}
			control := transcriptControl{key: key, label: i18n.T(i18n.KeyChatProcessGroup), open: wholeOpen[keys[i]], members: members}
			controls[key] = control
			out = append(out, session.Message{ID: key, Seq: msg.Seq, Kind: transcriptControlKind, Body: control.label})
		}
		if known[i] && !wholeOpen[keys[i]] && transcriptProcess(msg) && i != t.final {
			i++
			continue
		}
		groupable := msg.Kind == session.MessageKindThinking || msg.Kind == session.MessageKindTool
		if !groupable || mode == "verbose" || mode == "detailed" && (!known[i] || t.end == nil) {
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
		open := workflowExpanded(m.ThinkingExpanded, key, false)
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
				label += " · " + thinkingPreview(current.Arguments, 120)
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

// transcriptControlView is the transcript's shared disclosure row: the group
// title sits on the left, the summary and chevron on the right.
func transcriptControlView(c *ui.Context, control transcriptControl, m Model, muted ui.Color) {
	paintDisclosureRow(c, disclosureRow{
		Key:        "transcript-control:" + control.key,
		Title:      control.label,
		TitleColor: muted,
		Open:       control.open,
		Running:    control.running,
		Expandable: true,
		OnToggle: func() {
			workflowToggle(m.ThinkingExpanded, control.key, false)
			if control.open && m.ThinkingExpanded != nil {
				// A whole-turn close also closes its loaded reasoning/tool detail choices.
				for _, id := range control.members {
					delete(*m.ThinkingExpanded, id)
				}
			}
		},
	}, disclosureStyle{FG: muted, Muted: muted, Accent: muted, Hover: muted.Alpha(.10), Size: 12})
}
