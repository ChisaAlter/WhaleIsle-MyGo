package main

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"whaleisle/internal/desktop/pluginrpc"
	"whaleisle/internal/dsh"
	"whaleisle/internal/store/session"
)

const taskCriterion = "Extract the exact owned file marker with a source citation."

func exerciseWhaleTasks(ctx context.Context, client *dsh.Client, store *session.Store, model *modelFixture, workspace, root string, pass func(string, string), restart func() (*dsh.Client, *session.Store, error)) error {
	rpc := &pluginrpc.Client{BaseURL: client.Endpoint.Origin, Cookie: client.Endpoint.Cookie, HTTPClient: client.HTTP}
	catalog, err := rpc.WhaleTasks(ctx)
	if err != nil {
		return err
	}
	if !catalog.Available {
		return fmt.Errorf("whale tasks unavailable: %s", catalog.Error)
	}
	pass("whale-task-service-ready", "main desktop task plugins mounted against rc.4 with restricted worker preset")
	input := filepath.Join(workspace, "qa-note.txt")
	draft := pluginrpc.WhaleTaskDraft{Goal: "Summarize the owned QA input", Files: []string{input}, Criteria: []string{taskCriterion}}
	start := func() (pluginrpc.WhaleTask, error) { return rpc.WhaleStartTask(ctx, draft) }
	wait := func(id, want string) (pluginrpc.WhaleTask, error) {
		var found pluginrpc.WhaleTask
		err := await(ctx, func() (bool, error) {
			value, e := rpc.WhaleTasks(ctx)
			if e != nil {
				return false, e
			}
			for _, task := range value.Tasks {
				if task.ID == id {
					found = task
					if task.State == "failed" && want != "failed" {
						return false, fmt.Errorf("task failed: %s", task.Error)
					}
					return task.State == want, nil
				}
			}
			return false, nil
		})
		return found, err
	}
	first, err := start()
	if err != nil {
		return err
	}
	first, err = wait(first.ID, "completed")
	if err != nil {
		return err
	}
	if first.Result == nil || !strings.Contains(first.Result.Summary, "QA_TASK_SUMMARY") || len(first.Result.Evidence) != 1 || first.Execution == nil || first.Execution.ReadBytes == 0 {
		return errors.New("task did not produce a sourced report after actual input reading")
	}
	if err = await(ctx, func() (bool, error) {
		value, e := rpc.WhaleTasks(ctx)
		if e != nil {
			return false, e
		}
		for _, task := range value.Tasks {
			if task.ID == first.ID {
				for _, effect := range task.Effects {
					if effect.Target == "session" {
						return effect.State == "sent", nil
					}
				}
			}
		}
		return false, nil
	}); err != nil {
		return err
	}
	pass("whale-task-read-report-delivery", "real task worker read authorized snapshot, submitted sourced result and delivered it to the assistant session")
	model.mu.Lock()
	model.taskWait = true
	model.mu.Unlock()
	second, err := start()
	if err != nil {
		return err
	}
	if err = await(ctx, func() (bool, error) { model.mu.Lock(); defer model.mu.Unlock(); return model.taskWaiting, nil }); err != nil {
		return err
	}
	if _, err = rpc.WhaleControlTask(ctx, second.ID, "stop"); err != nil {
		return err
	}
	if second, err = wait(second.ID, "cancelled"); err != nil {
		return err
	}
	if err = await(ctx, func() (bool, error) { model.mu.Lock(); defer model.mu.Unlock(); return !model.taskWaiting, nil }); err != nil {
		return err
	}
	pass("whale-task-stop-cancels-model", "native stop cancelled the live local-model request and persisted cancelled state")
	if err = os.WriteFile(input, []byte("REPLACED AFTER AUTHORIZATION"), 0600); err != nil {
		return err
	}
	model.mu.Lock()
	model.taskWait = false
	model.mu.Unlock()
	stoppedGeneration := second.Generation
	if _, err = rpc.WhaleControlTask(ctx, second.ID, "resume"); err != nil {
		return err
	}
	second, err = wait(second.ID, "completed")
	if err != nil {
		return err
	}
	if second.Generation != stoppedGeneration+1 || second.Result == nil || !strings.Contains(second.Result.Summary, ownedFileText) {
		return errors.New("resume lost authorization snapshot or generation")
	}
	pass("whale-task-resume-original-snapshot", "explicit resume used the original authorized snapshot after the source file changed")
	if err = await(ctx, func() (bool, error) {
		value, e := rpc.WhaleTasks(ctx)
		if e != nil {
			return false, e
		}
		for _, task := range value.Tasks {
			if task.ID == second.ID {
				return !task.Delivering(), nil
			}
		}
		return false, nil
	}); err != nil {
		return err
	}
	if err = os.WriteFile(input, []byte(ownedFileText), 0600); err != nil {
		return err
	}
	model.mu.Lock()
	model.taskWait = true
	model.mu.Unlock()
	third, err := start()
	if err != nil {
		return err
	}
	if err = await(ctx, func() (bool, error) { model.mu.Lock(); defer model.mu.Unlock(); return model.taskWaiting, nil }); err != nil {
		return err
	}
	client, store, err = restart()
	if err != nil {
		return err
	}
	_ = store
	rpc = &pluginrpc.Client{BaseURL: client.Endpoint.Origin, Cookie: client.Endpoint.Cookie, HTTPClient: client.HTTP}
	third, err = wait(third.ID, "interrupted")
	if err != nil {
		return err
	}
	model.mu.Lock()
	before := model.requests
	model.mu.Unlock()
	time.Sleep(300 * time.Millisecond)
	model.mu.Lock()
	after := model.requests
	model.taskWait = false
	model.mu.Unlock()
	if before != after {
		return errors.New("interrupted task automatically restarted")
	}
	if _, err = rpc.WhaleControlTask(ctx, third.ID, "resume"); err != nil {
		return err
	}
	if _, err = wait(third.ID, "completed"); err != nil {
		return err
	}
	pass("whale-task-restart-explicit-resume", "Host restart preserves interrupted task and requires explicit resume before new model requests")
	if err = rpc.WhaleTaskOwner(ctx, "qa-feishu-bot", "qa-open-id", false); err != nil {
		return err
	}
	if err = rpc.WhaleTaskOwner(ctx, "qa-feishu-bot", "", true); err != nil {
		return err
	}
	pass("whale-task-owner-controls", "native authenticated owner grant/revoke acknowledged for isolated synthetic bot; no external IM messages")
	model.mu.Lock()
	mismatch := model.taskInputMismatch
	model.mu.Unlock()
	if mismatch {
		return errors.New("actual worker tool result did not contain the authorized input bytes")
	}
	catalog, err = rpc.WhaleTasks(ctx)
	if err != nil {
		return err
	}
	data, _ := json.MarshalIndent(catalog, "", "  ")
	return os.WriteFile(filepath.Join(root, "whale-tasks-final.json"), data, 0600)
}
