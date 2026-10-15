package session

import (
	"encoding/json"
	"strings"

	"whaleisle/internal/dsh/events"
)

// PlanBoard is the folded plan / goal / queue / todo strip for the active follow.
type PlanBoard struct {
	PlanMode     bool
	Goal         *PlanGoal
	Todos        []PlanTodo
	Queue        []PlanQueueItem
	QueueActions QueueActionState
}

// PlanGoal is the current durable goal snapshot (nil when cleared / absent).
type PlanGoal struct {
	ID        string
	Revision  float64
	Objective string
	Phase     string
}

// PlanTodo is one standing-plan checklist row from todo/write (or todo_write tool args).
type PlanTodo struct {
	Content string
	Status  string
}

// PlanQueueItem is a queued prompt / team message waiting to run.
type PlanQueueItem struct {
	ID                     string
	Preview                string
	Source                 string
	RequestID              string
	ContentKnown, TextOnly bool
	Attachments            []QueueAttachment
}

// Empty reports whether the board has nothing useful to show.
func (b PlanBoard) Empty() bool {
	return !b.PlanMode && b.Goal == nil && len(b.Todos) == 0 && len(b.Queue) == 0
}

// ApplyPlanBoardEvent folds one persistence event into the board.
// Returns true when the event was recognized (even if it cleared state).
func ApplyPlanBoardEvent(board *PlanBoard, raw json.RawMessage) bool {
	if board == nil || len(raw) == 0 {
		return false
	}
	var ev struct {
		Type string          `json:"type"`
		Data json.RawMessage `json:"data"`
	}
	if err := json.Unmarshal(raw, &ev); err != nil || ev.Type == "" {
		return false
	}
	switch ev.Type {
	case events.TypePlanMode:
		var d events.PlanModeData
		if json.Unmarshal(ev.Data, &d) != nil {
			return false
		}
		board.PlanMode = d.Active
		return true
	case events.TypeTodoWrite:
		todos, ok := parseTodoItems(ev.Data)
		if !ok {
			return false
		}
		board.Todos = todos
		return true
	case events.TypeGoalChange:
		return applyGoalChange(board, ev.Data)
	case events.TypeTeamMessageQueued:
		item, ok := parseTeamQueued(ev.Data)
		if !ok {
			return false
		}
		upsertQueue(board, item)
		return true
	case "turn/start":
		// Standing plan display lifetime: clear on the next turn.
		board.Todos = nil
		return true
	case events.TypeToolCall:
		return applyTodoToolCall(board, ev.Data)
	default:
		if strings.Contains(ev.Type, "queue") {
			if item, ok := parseLooseQueue(ev.Type, ev.Data); ok {
				upsertQueue(board, item)
				return true
			}
		}
		return false
	}
}

// FoldPlanBoard replays persistence events in order (snapshot / tests).
func FoldPlanBoard(rawEvents []json.RawMessage) PlanBoard {
	var board PlanBoard
	for _, raw := range rawEvents {
		ApplyPlanBoardEvent(&board, raw)
	}
	return board
}

func (s *Store) applyPlanBoardEvent(raw json.RawMessage) {
	s.mu.Lock()
	ApplyPlanBoardEvent(&s.planBoard, raw)
	s.mu.Unlock()
}

func (s *Store) resetPlanBoardLocked() {
	s.planBoard = PlanBoard{}
}

func applyGoalChange(board *PlanBoard, data json.RawMessage) bool {
	var meta struct {
		Operation string `json:"operation"`
		Goal      *struct {
			ID        string  `json:"id"`
			Revision  float64 `json:"revision"`
			Objective string  `json:"objective"`
			Phase     string  `json:"phase"`
		} `json:"goal"`
	}
	if json.Unmarshal(data, &meta) != nil {
		return false
	}
	if meta.Operation == "clear" || meta.Goal == nil {
		board.Goal = nil
		return true
	}
	obj := strings.TrimSpace(meta.Goal.Objective)
	phase := strings.TrimSpace(meta.Goal.Phase)
	if obj == "" && phase == "" && meta.Goal.ID == "" {
		return false
	}
	board.Goal = &PlanGoal{
		ID:        meta.Goal.ID,
		Revision:  meta.Goal.Revision,
		Objective: obj,
		Phase:     phase,
	}
	return true
}

func parseTodoItems(data json.RawMessage) ([]PlanTodo, bool) {
	var d struct {
		Todos []struct {
			Content string `json:"content"`
			Status  string `json:"status"`
		} `json:"todos"`
	}
	if json.Unmarshal(data, &d) != nil {
		return nil, false
	}
	out := make([]PlanTodo, 0, len(d.Todos))
	for _, t := range d.Todos {
		c := strings.TrimSpace(t.Content)
		if c == "" {
			continue
		}
		st := strings.TrimSpace(t.Status)
		if st == "" {
			st = "pending"
		}
		out = append(out, PlanTodo{Content: c, Status: st})
	}
	return out, true
}

func applyTodoToolCall(board *PlanBoard, data json.RawMessage) bool {
	var d struct {
		Name      string `json:"name"`
		Arguments string `json:"arguments"`
	}
	if json.Unmarshal(data, &d) != nil {
		return false
	}
	name := strings.ToLower(strings.TrimSpace(d.Name))
	if !strings.Contains(name, "todo") {
		return false
	}
	todos, ok := parseTodoItems(json.RawMessage(d.Arguments))
	if !ok {
		return false
	}
	board.Todos = todos
	return true
}

func parseTeamQueued(data json.RawMessage) (PlanQueueItem, bool) {
	var d events.TeamMessageQueuedData
	if json.Unmarshal(data, &d) != nil {
		return PlanQueueItem{}, false
	}
	var preview string
	for _, block := range d.Message.Content {
		preview = appendBlockText(preview, block)
	}
	preview = strings.TrimSpace(preview)
	if preview == "" {
		preview = strings.TrimSpace(d.Message.Id)
	}
	if preview == "" {
		return PlanQueueItem{}, false
	}
	return PlanQueueItem{
		ID:      d.Message.Id,
		Preview: truncateRunes(preview, 120),
		Source:  "team",
	}, true
}

func parseLooseQueue(typ string, data json.RawMessage) (PlanQueueItem, bool) {
	preview := extractTextish(data)
	if preview == "" {
		var obj map[string]json.RawMessage
		if json.Unmarshal(data, &obj) == nil {
			for _, key := range []string{"text", "content", "prompt", "body", "itemId", "id"} {
				if v, ok := obj[key]; ok {
					if t := extractTextish(v); t != "" {
						preview = t
						break
					}
					var s string
					if json.Unmarshal(v, &s) == nil && strings.TrimSpace(s) != "" {
						preview = strings.TrimSpace(s)
						break
					}
				}
			}
		}
	}
	if preview == "" {
		return PlanQueueItem{}, false
	}
	id := ""
	var obj struct {
		ID     string `json:"id"`
		ItemID string `json:"itemId"`
	}
	_ = json.Unmarshal(data, &obj)
	id = obj.ID
	if id == "" {
		id = obj.ItemID
	}
	return PlanQueueItem{
		ID:      id,
		Preview: truncateRunes(preview, 120),
		Source:  typ,
	}, true
}

func upsertQueue(board *PlanBoard, item PlanQueueItem) {
	if item.ID != "" {
		for i := range board.Queue {
			if board.Queue[i].ID == item.ID {
				board.Queue[i] = item
				return
			}
		}
	}
	board.Queue = append(board.Queue, item)
}
