package main

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"slices"
	"strings"
	"time"

	"whaleisle/internal/dsh"
	"whaleisle/internal/dsh/api"
	"whaleisle/internal/store/session"
)

func exerciseQueueIdle(ctx context.Context, client *dsh.Client, store *session.Store, model *modelFixture, id, workspace string, pass func(string, string)) error {
	store.WatchControl(ctx)
	if err := store.SubmitComposer(ctx, id, waitPrompt, nil, 0); err != nil {
		return err
	}
	if err := await(ctx, func() (bool, error) {
		for _, row := range store.Snapshot().Messages {
			if row.Streaming && strings.Contains(row.Body, "QA_WAIT_STREAM") {
				return true, nil
			}
		}
		return false, nil
	}); err != nil {
		return err
	}
	for _, text := range []string{"QUEUE_FIRST", "QUEUE_SECOND", "QUEUE_THIRD"} {
		if err := store.SubmitComposer(ctx, id, text, nil, 0); err != nil {
			return err
		}
	}
	if err := await(ctx, func() (bool, error) { return len(store.Snapshot().PlanBoard.Queue) == 3, nil }); err != nil {
		return err
	}
	queue := store.Snapshot().PlanBoard.Queue
	want := []string{queue[2].ID, queue[0].ID, queue[1].ID}
	store.MoveQueue(ctx, id, queue[2].ID, 0)
	if err := await(ctx, func() (bool, error) {
		v := store.Snapshot()
		if v.PlanBoard.QueueActions.Busy {
			return false, nil
		}
		actual := []string{}
		for _, row := range v.PlanBoard.Queue {
			actual = append(actual, row.ID)
		}
		return slices.Equal(actual, want), nil
	}); err != nil {
		return fmt.Errorf("Go queue reorder: %w; %s", err, store.Snapshot().LastAction)
	}
	var projection api.SessionProjectionsValue
	if err := client.Call(ctx, "session/projections", map[string]any{"request": map[string]any{"sessionId": id}}, &projection); err != nil {
		return err
	}
	var wire struct {
		Inbox struct {
			NextTurn []struct {
				ID string `json:"id"`
			} `json:"next-turn"`
		} `json:"inbox"`
	}
	if err := json.Unmarshal(projection.Values, &wire); err != nil {
		return err
	}
	actual := []string{}
	for _, row := range wire.Inbox.NextTurn {
		actual = append(actual, row.ID)
	}
	if !slices.Equal(actual, want) {
		return fmt.Errorf("Host order %v differs from Go order %v", actual, want)
	}
	pass("queue-reorder-host-and-store", "third queued prompt moved to first; authenticated Host projection and production Store agree")
	// Remove the exact owned pending messages so cancel does not start another turn.
	for _, item := range want {
		store.UpdateQueue(ctx, id, item, session.QueueActionRemove)
		if err := await(ctx, func() (bool, error) {
			v := store.Snapshot()
			return !v.PlanBoard.QueueActions.Busy && !slices.ContainsFunc(v.PlanBoard.Queue, func(row session.PlanQueueItem) bool { return row.ID == item }), nil
		}); err != nil {
			return err
		}
	}
	store.CancelRunning(ctx)
	if err := await(ctx, func() (bool, error) {
		model.mu.Lock()
		canceled := model.canceled
		model.mu.Unlock()
		v := store.Snapshot()
		return canceled > 0 && !v.PromptBusy && v.StopTurnSeq == 0 && !slices.ContainsFunc(v.Messages, func(m session.Message) bool { return m.Streaming }), nil
	}); err != nil {
		return err
	}
	available := func() (bool, error) {
		var list api.SessionListValue
		if err := client.Call(ctx, "session/list", map[string]any{"_request": api.SessionListRequest{}}, &list); err != nil {
			return false, err
		}
		for _, row := range list.Items {
			if row.SessionId == id {
				return row.AgentAvailable, nil
			}
		}
		return false, errors.New("idle release removed the persisted session from the list")
	}
	// Two configured idle periods while followed must leave the live Agent intact.
	select {
	case <-ctx.Done():
		return ctx.Err()
	case <-time.After(3 * time.Second):
	}
	if ok, err := available(); err != nil || !ok {
		return fmt.Errorf("watched session released: %v", err)
	}
	pass("watched-session-retained", "followed idle session remains live beyond the configured release delay")
	other, err := store.CreateComposerSession(ctx, workspace)
	if err != nil {
		return err
	}
	if other == id {
		return errors.New("second session reused the nonempty first session")
	}
	store.Select(other)
	releaseCtx, releaseCancel := context.WithTimeout(ctx, 10*time.Second)
	defer releaseCancel()
	if err = await(releaseCtx, func() (bool, error) { active, e := available(); return !active, e }); err != nil {
		return fmt.Errorf("unwatched idle session was not released: %w", err)
	}
	pass("idle-session-released", "after switching away, Host releases the Agent and retains the session list entry")
	store.Select(id)
	if err = await(ctx, func() (bool, error) {
		v := store.Snapshot()
		if v.HistoryError != "" {
			return false, errors.New(v.HistoryError)
		}
		return v.SelectedID == id && v.HistoryReady && !v.HistoryLoading && slices.ContainsFunc(v.Messages, func(row session.Message) bool { return row.Seq > 0 && strings.Contains(row.Body, "QA_WAIT_STREAM") }), nil
	}); err != nil {
		return err
	}
	if ok, err := available(); err != nil || !ok {
		return fmt.Errorf("reopened session did not regain its Agent: %v", err)
	}
	pass("released-session-reopens", "same session restores its durable stopped response and a live Agent")
	return nil
}
