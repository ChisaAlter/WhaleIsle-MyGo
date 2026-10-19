package main

import (
	"context"
	"encoding/json"
	"testing"
	"time"
	"whaleisle/internal/dsh"
	manager "whaleisle/internal/pluginmanager"
)

type nativeManagerFixture func(context.Context, string, any, any) error

func (f nativeManagerFixture) Call(ctx context.Context, method string, args, out any) error {
	return f(ctx, method, args, out)
}

func TestNativePluginEventsFollowCapturedInstallAndHost(t *testing.T) {
	release := make(chan struct{})
	defer close(release)
	started := make(chan struct{}, 1)
	fixture := nativeManagerFixture(func(ctx context.Context, method string, args, out any) error {
		var value any
		switch method {
		case "pluginInventory/list":
			value = map[string]any{"managementAvailable": true}
		case "pluginManager/listBundles", "pluginManager/listPlugins":
			value = []any{}
		case "pluginManager/registries":
			value = map[string]any{}
		case "pluginManager/inspect":
			value = map[string]any{"status": "accepted", "name": "fixture"}
		case "pluginManager/installBundle":
			started <- struct{}{}
			select {
			case <-release:
			case <-ctx.Done():
				return ctx.Err()
			}
			value = map[string]any{"application": "applied", "changed": true}
		default:
			t.Errorf("unexpected fixture method %s", method)
		}
		if out != nil {
			raw, _ := json.Marshal(value)
			return json.Unmarshal(raw, out)
		}
		return nil
	})
	c := manager.New()
	c.Attach(fixture, nil)
	defer c.Detach()
	if err := c.Load(context.Background()); err != nil {
		t.Fatal(err)
	}
	owner := c.Snapshot().Owner
	transport := &dsh.Client{}
	s := mainState{nativePlugins: nativePluginManagerState{controller: c, client: transport, owner: owner}}
	emit := s.nativePluginEventHandler(transport)
	a := c.Actions(owner)
	a.OpenInstall()
	a.EditInstall("fixture", "")
	a.Inspect()
	waitNativeManager(t, c, func(v manager.State) bool { return v.Install.Inspection != nil })
	a.RequestInstall()
	a.Confirm()
	select {
	case <-started:
	case <-time.After(3 * time.Second):
		t.Fatal("fake install did not start")
	}
	id := c.Snapshot().Install.RequestID
	if id == "" {
		t.Fatal("install request has no identity")
	}
	send := func(event string, value any) {
		raw, _ := json.Marshal(value)
		emit(dsh.EventsEmit{Event: event, Args: []json.RawMessage{raw}})
	}
	send("plugin-manager/install-log", map[string]any{"requestId": "other", "jobId": "job", "text": "wrong request"})
	send("plugin-manager/install-state", map[string]any{"requestId": id, "phase": "applying"})
	send("plugin-manager/install-state", map[string]any{"requestId": id, "phase": "installing"})
	send("plugin-manager/install-log", map[string]any{"requestId": id, "jobId": "job", "argv": []string{"pnpm", "install", "fixture"}, "cwd": "C:/fixture", "stream": "stdout", "text": "fixture output"})
	if got := c.Snapshot().Install; got.Phase != "applying" || got.Log != "fixture output" {
		t.Fatalf("native event wire lost owner/phase/output: %+v", got)
	}
	// Replacing the controller Host invalidates the entire old event callback.
	c.Attach(fixture, nil)
	send("plugin-manager/install-log", map[string]any{"requestId": id, "jobId": "job", "text": "stale host"})
	send("plugin-manager/install-state", map[string]any{"requestId": id, "phase": "applying"})
	if got := c.Snapshot().Install; got.Log != "" || got.Phase != "" {
		t.Fatalf("old transport changed replacement Host: %+v", got)
	}
}

func waitNativeManager(t *testing.T, c *manager.Controller, ready func(manager.State) bool) {
	t.Helper()
	deadline := time.Now().Add(3 * time.Second)
	for time.Now().Before(deadline) {
		if ready(c.Snapshot()) {
			return
		}
		time.Sleep(time.Millisecond)
	}
	t.Fatal("isolated plugin state did not settle")
}
