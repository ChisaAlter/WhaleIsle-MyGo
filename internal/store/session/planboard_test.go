package session

import (
	"encoding/json"
	"testing"
)

func TestApplyPlanBoardTodoWrite(t *testing.T) {
	raw := json.RawMessage(`{
		"type":"todo/write","seq":1,"time":0,
		"data":{"todos":[
			{"content":"scan","status":"completed"},
			{"content":"fix","status":"in_progress"},
			{"content":"ship","status":"pending"}
		]}
	}`)
	var board PlanBoard
	if !ApplyPlanBoardEvent(&board, raw) {
		t.Fatal("expected apply")
	}
	if len(board.Todos) != 3 || board.Todos[1].Status != "in_progress" {
		t.Fatalf("%#v", board.Todos)
	}
}

func TestApplyPlanBoardTodoToolCallFallback(t *testing.T) {
	args := `{"todos":[{"content":"a","status":"pending"},{"content":"b","status":"completed"}]}`
	raw, _ := json.Marshal(map[string]any{
		"type": "tool/call",
		"seq":  2,
		"data": map[string]any{
			"name":      "todo_write",
			"arguments": args,
			"callId":    "c1",
		},
	})
	var board PlanBoard
	if !ApplyPlanBoardEvent(&board, raw) {
		t.Fatal("expected tool/call todo apply")
	}
	if len(board.Todos) != 2 || board.Todos[1].Content != "b" {
		t.Fatalf("%#v", board.Todos)
	}
}

func TestApplyPlanBoardGoalChangeAndClear(t *testing.T) {
	create := json.RawMessage(`{
		"type":"goal/change","seq":3,
		"data":{
			"kind":"goal/change","version":1,"operation":"create",
			"goal":{"id":"g1","revision":1,"objective":"Ship M5","phase":"active","maxGoalRounds":4},
			"roundsStarted":0,"createdAt":1,"updatedAt":1
		}
	}`)
	var board PlanBoard
	if !ApplyPlanBoardEvent(&board, create) || board.Goal == nil || board.Goal.Objective != "Ship M5" {
		t.Fatalf("%#v", board.Goal)
	}
	clear := json.RawMessage(`{
		"type":"goal/change","seq":4,
		"data":{"kind":"goal/change","version":1,"operation":"clear","cleared":{"id":"g1","revision":1},"clearedAt":2}
	}`)
	if !ApplyPlanBoardEvent(&board, clear) || board.Goal != nil {
		t.Fatalf("clear failed: %#v", board.Goal)
	}
}

func TestApplyPlanBoardPlanModeAndTurnClearsTodos(t *testing.T) {
	board := FoldPlanBoard([]json.RawMessage{
		json.RawMessage(`{"type":"plan/mode","seq":1,"data":{"active":true}}`),
		json.RawMessage(`{"type":"todo/write","seq":2,"data":{"todos":[{"content":"x","status":"pending"}]}}`),
		json.RawMessage(`{"type":"turn/start","seq":3,"data":{"turn":1}}`),
	})
	if !board.PlanMode {
		t.Fatal("plan mode off")
	}
	if len(board.Todos) != 0 {
		t.Fatalf("todos should clear on turn/start: %#v", board.Todos)
	}
}

func TestApplyPlanBoardTeamQueue(t *testing.T) {
	raw := json.RawMessage(`{
		"type":"team/message/queued","seq":5,
		"data":{
			"teamId":"t1","version":2,
			"message":{
				"id":"m9","senderId":"s","senderName":"n","targetId":"a",
				"content":[{"type":"text","text":"please review"}]
			}
		}
	}`)
	var board PlanBoard
	if !ApplyPlanBoardEvent(&board, raw) || len(board.Queue) != 1 {
		t.Fatalf("%#v", board.Queue)
	}
	if board.Queue[0].Preview != "please review" || board.Queue[0].Source != "team" {
		t.Fatalf("%#v", board.Queue[0])
	}
}

func TestPlanBoardEmpty(t *testing.T) {
	var z PlanBoard
	if !z.Empty() {
		t.Fatal("zero should be empty")
	}
	z.PlanMode = true
	if z.Empty() {
		t.Fatal("plan mode makes non-empty")
	}
}

func TestFoldPlanBoardSnapshotOrder(t *testing.T) {
	board := FoldPlanBoard([]json.RawMessage{
		json.RawMessage(`{"type":"todo/write","seq":1,"data":{"todos":[{"content":"old","status":"pending"}]}}`),
		json.RawMessage(`{"type":"todo/write","seq":2,"data":{"todos":[{"content":"new","status":"in_progress"}]}}`),
		json.RawMessage(`{"type":"goal/change","seq":3,"data":{"operation":"create","goal":{"id":"g","revision":1,"objective":"O","phase":"paused"}}}`),
	})
	if len(board.Todos) != 1 || board.Todos[0].Content != "new" {
		t.Fatalf("todos %#v", board.Todos)
	}
	if board.Goal == nil || board.Goal.Phase != "paused" {
		t.Fatalf("goal %#v", board.Goal)
	}
}
