package chat

import (
	"encoding/json"
	"fmt"
	"strings"

	"whaleisle/internal/dsh/events"
	"whaleisle/internal/store/session"
	"whaleisle/internal/ui/i18n"
)

// The installed todo_write card compares the recorded list with the list that
// preceded the call (todo-diff-model.ts / todo-history.ts). The native client
// indexes `todo/write` events and binds each root or PTC-subcall `todo_write`
// invocation to the state of the previous one. MyGo keeps the same durable
// events in the loaded session window, so the same predecessor list is
// reconstructable; the history limit is only "older unloaded pages", which the
// card states as an unavailable previous list.

// todoBaseline is the recorded todo list preceding one todo_write invocation.
type todoBaseline struct {
	// Todos is the preceding durable list. Empty is a recorded empty list.
	Todos []session.PlanTodo
	// Known reports a preceding write was reconstructed. When false the call is
	// the first loaded write (the installed "initial list"), unless HasMore says
	// older unloaded history may still hold a predecessor.
	Known   bool
	HasMore bool
}

// todoBaselinesFor indexes the list preceding every todo_write call in the loaded
// window. A call id keeps its first baseline, as the installed view does.
func todoBaselinesFor(messages []session.Message, historyHasMore bool) map[string]*todoBaseline {
	baselines := make(map[string]*todoBaseline)
	var last []session.PlanTodo
	known := false
	bind := func(callID string) {
		if callID == "" {
			return
		}
		if _, exists := baselines[callID]; exists {
			return
		}
		baselines[callID] = &todoBaseline{Todos: last, Known: known, HasMore: historyHasMore && !known}
	}
	for _, message := range messages {
		switch message.EventType {
		case events.TypeTodoWrite:
			var event events.TodoWrite
			if json.Unmarshal(message.Raw, &event) != nil {
				continue
			}
			last = make([]session.PlanTodo, 0, len(event.Data.Todos))
			for _, todo := range event.Data.Todos {
				last = append(last, session.PlanTodo{Content: todo.Content, Status: todo.Status})
			}
			known = true
		case events.TypeToolCall:
			var event events.ToolCall
			if json.Unmarshal(message.Raw, &event) == nil && event.Data.Name == "todo_write" {
				bind(event.Data.CallId)
			}
		case events.TypeToolPtcDispatchStart:
			var event events.ToolPtcDispatchStart
			if json.Unmarshal(message.Raw, &event) == nil && event.Data.Name == "todo_write" {
				bind(event.Data.SubCallId)
			}
		}
	}
	return baselines
}

// toolDetailsTodoStatusLabel localizes the three recorded todo states; any other
// recorded value keeps its spelling, as the installed dictionary lookup would.
func toolDetailsTodoStatusLabel(status string) string {
	switch status {
	case "completed":
		return i18n.T(i18n.KeyToolDetailStatusCompleted)
	case "in_progress":
		return i18n.T(i18n.KeyToolDetailStatusProgress)
	case "pending":
		return i18n.T(i18n.KeyToolDetailStatusPending)
	default:
		return status
	}
}

// toolDetailsTodoDetail is the installed todosDetail: every argument entry needs
// visible content and one of the three recorded states, and a duplicate title
// rejects the whole card.
func toolDetailsTodoDetail(args map[string]json.RawMessage) ([]toolDetailItem, bool) {
	raw, exists := args["todos"]
	if !exists {
		return nil, false
	}
	value, ok := toolDetailsDecodeJSON(string(raw))
	if !ok || value.kind != 'a' {
		return nil, false
	}
	items := make([]toolDetailItem, 0, len(value.list))
	seen := make(map[string]bool, len(value.list))
	for _, entry := range value.list {
		if entry.kind != 'o' {
			return nil, false
		}
		content, okContent := entry.get("content")
		status, okStatus := entry.get("status")
		if !okContent || content.kind != 's' || !okStatus || status.kind != 's' {
			return nil, false
		}
		title := trimTodoText(content.text)
		if title == "" || (status.text != "completed" && status.text != "in_progress" && status.text != "pending") {
			return nil, false
		}
		if seen[title] {
			return nil, false
		}
		seen[title] = true
		items = append(items, toolDetailItem{
			Title:  title,
			Status: &toolDetailStatus{Value: status.text, Label: toolDetailsTodoStatusLabel(status.text)},
			Fields: []toolDetailField{},
		})
	}
	return items, true
}

// trimTodoText applies the JavaScript trim set to a recorded todo content, which
// also drops the no-break space and byte-order mark.
func trimTodoText(text string) string {
	return strings.TrimFunc(text, isTodoSpace)
}

// isTodoSpace is the JavaScript String.prototype.trim whitespace set.
func isTodoSpace(r rune) bool {
	switch r {
	case '\t', '\n', '\v', '\f', '\r', ' ', '\u00a0', '\ufeff', '\u2028', '\u2029':
		return true
	}
	return false
}

// toolDetailsTodoFor derives the installed write card. It returns nil for any
// other tool, a missing name or recorded arguments that do not form a list, so
// the raw evidence body remains.
func toolDetailsTodoFor(call *session.ToolCallView, baseline *todoBaseline) *toolDetailsModel {
	if call == nil || call.Result == nil || call.State != "ok" {
		return nil
	}
	result, ok := terminalModelReadResult(call.Result)
	if !ok || result.failed {
		return nil
	}
	name, args, ok := toolDetailsModelCallArgs(call, result.nested)
	if !ok || name != "todo_write" {
		return nil
	}
	current, ok := toolDetailsTodoDetail(args)
	if !ok {
		return nil
	}
	// A truncated history may hide the list this write follows; the installed
	// card then states that the previous list is unavailable and keeps the
	// recorded rows verbatim.
	if baseline != nil && !baseline.Known && baseline.HasMore {
		return &toolDetailsModel{Items: current, Caption: i18n.T(i18n.KeyToolDetailTodoUnavailable),
			Empty: i18n.T(i18n.KeyToolDetailTodoEmpty)}
	}
	// Without a predecessor the installed card still diffs: every recorded row
	// reads as added and the caption names the initial list. This is also the
	// reversible subset for surfaces that keep no event window.
	previous := make([]toolDetailItem, 0)
	if baseline != nil && baseline.Known {
		previous = make([]toolDetailItem, 0, len(baseline.Todos))
		for _, todo := range baseline.Todos {
			previous = append(previous, toolDetailItem{
				Title:  todo.Content,
				Status: &toolDetailStatus{Value: todo.Status, Label: toolDetailsTodoStatusLabel(todo.Status)},
				Fields: []toolDetailField{},
			})
		}
	}
	hasPrevious := baseline != nil && baseline.Known
	previousByTitle := make(map[string]toolDetailItem, len(previous))
	for _, item := range previous {
		previousByTitle[item.Title] = item
	}
	currentTitles := make(map[string]bool, len(current))
	for _, item := range current {
		currentTitles[item.Title] = true
	}
	retained := make(map[string]int, len(previous))
	for _, item := range previous {
		if currentTitles[item.Title] {
			retained[item.Title] = len(retained)
		}
	}
	retainedIndex := 0
	items := make([]toolDetailItem, 0)
	unchanged := make([]toolDetailItem, 0)
	added, updated := 0, 0
	for _, item := range current {
		before, exists := previousByTitle[item.Title]
		delete(previousByTitle, item.Title)
		if !exists {
			added++
			item.Change = &toolDetailChange{Value: "added", Label: i18n.T(i18n.KeyToolDetailTodoAdded)}
			items = append(items, item)
			continue
		}
		moved := retained[item.Title] != retainedIndex
		retainedIndex++
		statusChanged := before.Status.Value != item.Status.Value
		if !statusChanged && !moved {
			unchanged = append(unchanged, item)
			continue
		}
		updated++
		if statusChanged {
			item.PreviousStatus = toolDetailsModelString(before.Status.Label)
		}
		label := i18n.KeyToolDetailTodoUpdated
		if !statusChanged {
			label = i18n.KeyToolDetailTodoMoved
		}
		item.Change = &toolDetailChange{Value: "updated", Label: i18n.T(label)}
		items = append(items, item)
	}
	removed := 0
	for _, item := range previous {
		if _, remains := previousByTitle[item.Title]; !remains {
			continue
		}
		removed++
		item.Change = &toolDetailChange{Value: "removed", Label: i18n.T(i18n.KeyToolDetailTodoRemoved)}
		items = append(items, item)
	}
	summary := ""
	appendCount := func(format string, count int) {
		if count == 0 {
			return
		}
		if summary != "" {
			summary += " · "
		}
		summary += fmt.Sprintf(format, count)
	}
	appendCount(i18n.T(i18n.KeyToolDetailTodoAddedCount), added)
	appendCount(i18n.T(i18n.KeyToolDetailTodoUpdatedCount), updated)
	appendCount(i18n.T(i18n.KeyToolDetailTodoRemovedCount), removed)
	if summary == "" {
		summary = i18n.T(i18n.KeyToolDetailTodoNoChanges)
	}
	caption := i18n.KeyToolDetailTodoInitial
	empty := i18n.KeyToolDetailTodoNoChanges
	if hasPrevious {
		caption = i18n.KeyToolDetailTodoCompare
	}
	if !hasPrevious && len(current) == 0 {
		empty = i18n.KeyToolDetailTodoEmpty
	}
	return &toolDetailsModel{
		Items:     items,
		Summary:   summary,
		Caption:   i18n.T(caption),
		Empty:     i18n.T(empty),
		Unchanged: toolDetailsUnchangedFor(unchanged),
	}
}

// todoBaselineForCall selects the recorded predecessor for one tool message.
func todoBaselineForCall(baselines map[string]*todoBaseline, message session.Message) *todoBaseline {
	if message.Tool == nil {
		return nil
	}
	return baselines[message.Tool.CallID]
}
