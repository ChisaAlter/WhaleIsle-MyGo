package dshbot

import (
	"encoding/json"
	"fmt"
	"strings"
	"testing"
	"whaleisle/internal/desktop/pluginrpc"
)

func botDeleteStorePlan(t *testing.T) pluginrpc.DshbotBotDeletePlan {
	p := map[string]json.RawMessage{}
	json.Unmarshal([]byte(`{"id":"a","sessionId":"sa","name":"Original name","title":"Shown title","kind":"bot","hidden":false,"memberBotIds":[],"allowedSenderIds":[],"inbox":[]}`), &p)
	c := pluginrpc.DshbotDeleteCatalog{Revision: 4, Items: []pluginrpc.DshbotBotItem{{DshbotCatalogItem: pluginrpc.DshbotCatalogItem{ID: "a", SessionID: "sa", Name: "Original name", Title: "Shown title", Kind: "bot"}, Profile: p}}, Routines: []map[string]json.RawMessage{}, Tasks: []map[string]json.RawMessage{}, Audit: []map[string]json.RawMessage{}}
	v := pluginrpc.DshbotDeletePreview{Item: pluginrpc.DshbotDeleteIdentity{ID: "a", Name: "Shown title", Kind: "bot"}, Dependencies: pluginrpc.DshbotDeleteDependencies{Groups: []pluginrpc.DshbotDeleteGroup{}, Routines: []pluginrpc.DshbotDeleteNamed{}, Tasks: []pluginrpc.DshbotDeleteTask{}, Sources: []pluginrpc.DshbotDeleteNamed{}}, Preserves: []string{"conversation-history", "memory"}}
	plan, e := pluginrpc.BuildDshbotBotDeletePlan(c, v, "a")
	if e != nil {
		t.Fatal(e)
	}
	return plan
}
func loadDeletePlan(t *testing.T, c *BotDeleter, key string, p pluginrpc.DshbotBotDeletePlan) {
	ticket, ok := c.BeginLoad(key)
	if !ok {
		t.Fatal("load blocked")
	}
	if !c.FinishLoad(ticket, p, pluginrpc.DshbotDeleteCatalog{}, true, nil) {
		t.Fatal("load failed")
	}
}
func TestBotDeleterExplicitConsentAndLifecycle(t *testing.T) {
	var c BotDeleter
	p := botDeleteStorePlan(t)
	c.Bind("account-a", "client-1")
	loadDeletePlan(t, &c, "account-a", p)
	c.Prepare("account-a")
	r := c.Snapshot("account-a").Review
	if _, _, ok := c.Begin("account-a", r); ok {
		t.Fatal("default deletion enabled")
	}
	c.ConfirmIdentity("account-a", r, "a", "Original name")
	c.Acknowledge("account-a", r, true)
	ticket, cmd, ok := c.Begin("account-a", r)
	if !ok || cmd.ConfirmedID != "a" {
		t.Fatal("explicit review rejected")
	}
	if _, _, ok := c.Begin("account-a", r); ok {
		t.Fatal("repeat confirm")
	}
	if !c.MarkSent(ticket) || c.MarkSent(ticket) {
		t.Fatal("duplicate send")
	}
	c.Suspend()
	c.Bind("account-b", "client-2")
	c.Finish(ticket, pluginrpc.DshbotBotDeleteResult{Attempted: true, Removed: "a", Catalog: pluginrpc.DshbotDeleteCatalog{Revision: 5}, PreservedHistory: true, PreservedMemory: true, Warnings: []string{"notepad warning complete"}}, nil)
	if c.Snapshot("account-b").Completed {
		t.Fatal("wrong account settled")
	}
	s := c.Snapshot("account-a")
	if !s.Completed || len(s.Warnings) != 1 {
		t.Fatal(s)
	}
	c.Bind("account-c", "client-3")
	loadDeletePlan(t, &c, "account-c", p)
	c.Prepare("account-c")
	stale := c.Snapshot("account-c").Review
	c.Suspend()
	c.Bind("account-c", "client-3")
	if _, _, ok := c.Begin("account-c", stale); ok {
		t.Fatal("stale review survived hide")
	}
	loadDeletePlan(t, &c, "account-c", p)
	c.Prepare("account-c")
	r = c.Snapshot("account-c").Review
	c.Cancel("account-c")
	if _, _, ok := c.Begin("account-c", r); ok {
		t.Fatal("cancel did not retire danger review")
	}
}
func TestBotDeleterUnknownReadbackNeverRetries(t *testing.T) {
	var c BotDeleter
	p := botDeleteStorePlan(t)
	c.Bind("account", "client")
	loadDeletePlan(t, &c, "account", p)
	c.Prepare("account")
	r := c.Snapshot("account").Review
	c.ConfirmIdentity("account", r, "a", "Original name")
	c.Acknowledge("account", r, true)
	ticket, _, ok := c.Begin("account", r)
	if !ok || !c.MarkSent(ticket) {
		t.Fatal("start")
	}
	c.Suspend()
	c.Finish(ticket, pluginrpc.DshbotBotDeleteResult{Attempted: true}, fmt.Errorf("response lost"))
	c.Bind("account", "new-client")
	for _, v := range []pluginrpc.DshbotDeleteCatalog{{Revision: 5}, p.Catalog} {
		q, ok := c.BeginLoad("account")
		if !ok {
			t.Fatal("readback blocked")
		}
		c.FinishLoad(q, pluginrpc.DshbotBotDeletePlan{}, v, true, nil)
		c.Cancel("account")
		c.Prepare("account")
		s := c.Snapshot("account")
		if !s.Uncertain || s.Pending == nil || s.Review != nil || !strings.Contains(s.Observation, "unknown") {
			t.Fatal("readback unlocked", s)
		}
		if _, _, ok := c.Begin("account", r); ok {
			t.Fatal("unknown retried")
		}
	}
}
