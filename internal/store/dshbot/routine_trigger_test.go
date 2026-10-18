package dshbot

import (
	"encoding/json"
	"errors"
	"strings"
	"testing"
	"whaleisle/internal/desktop/pluginrpc"
)

func triggerView(revision int) *pluginrpc.DshbotRoutineView {
	v := &pluginrpc.DshbotRoutineView{NS: "dsh-bot", Revision: revision}
	v.Value.Items = []pluginrpc.DshbotCatalogItem{{ID: "b", SessionID: "s"}}
	v.Value.Routines = []pluginrpc.DshbotRoutine{{ID: "r", BotID: "b", Name: "Routine"}, {ID: "r2", BotID: "b", Name: "Other routine"}}
	return v
}
func triggerReady(t *testing.T) (*RoutineControls, RoutineScope) {
	t.Helper()
	c := &RoutineControls{}
	k := RoutineScope{HostKey: "authenticated-host-account", BotID: "b"}
	c.Bind(k)
	ticket, _ := c.BeginLoad(k)
	if !c.FinishLoad(ticket, triggerView(4), nil, nil, nil) {
		t.Fatal("fixture load")
	}
	c.Select(k, "r")
	return c, k
}
func triggerBegin(t *testing.T, c *RoutineControls, k RoutineScope) RoutineTriggerTicket {
	t.Helper()
	c.PrepareTrigger(k)
	s := c.Snapshot(k).Trigger
	c.AcknowledgeTrigger(k, s.Sequence, true)
	ticket, ok := c.BeginTrigger(k, s.Sequence)
	if !ok {
		t.Fatal("explicit reviewed request missing")
	}
	return ticket
}
func TestBotRoutineTriggerConfirmationAndZeroing(t *testing.T) {
	c, k := triggerReady(t)
	if c.Snapshot(k).Trigger.Open {
		t.Fatal("ordinary catalog load opened credential")
	}
	c.PrepareTrigger(k)
	s := c.Snapshot(k).Trigger
	if !s.Review || s.Acknowledged {
		t.Fatal(s)
	}
	if _, ok := c.BeginTrigger(k, s.Sequence); ok {
		t.Fatal("obtained credential without separate acknowledgement")
	}
	c.AcknowledgeTrigger(k, s.Sequence, true)
	ticket, ok := c.BeginTrigger(k, s.Sequence)
	if !ok || !c.CurrentTrigger(ticket) || c.Snapshot(k).Loaded {
		t.Fatal("dispatch or revision fence missing")
	}
	if _, ok := c.BeginTrigger(k, s.Sequence); ok {
		t.Fatal("double fetch")
	}
	token := []byte("synthetic-fixture-secret")
	if !c.FinishTrigger(ticket, token, nil) {
		t.Fatal("result rejected")
	}
	clear(token)
	address, header, err := c.TriggerMaterial(k, s.Sequence, "https://fixture.invalid")
	if err != nil || address != "https://fixture.invalid/dshbot-hook/routine/r" || header != "X-Dshbot-Token: synthetic-fixture-secret" {
		t.Fatal("material missing")
	}
	snapshot, _ := json.Marshal(c.Snapshot(k))
	if strings.Contains(string(snapshot), "synthetic-fixture-secret") {
		t.Fatal("snapshot captures plaintext")
	}
	owned := c.trigger.token
	c.CloseTrigger(k, s.Sequence)
	for _, b := range owned {
		if b != 0 {
			t.Fatal("owned credential buffer not overwritten")
		}
	}
	if c.Snapshot(k).Trigger.Open {
		t.Fatal("close kept credential view")
	}
	if _, _, err := c.TriggerMaterial(k, s.Sequence, "https://fixture.invalid"); err == nil {
		t.Fatal("closed view exposed credential")
	}
}
func TestBotRoutineTriggerScopeAndLateResponses(t *testing.T) {
	for _, reason := range []string{"close", "reopen", "account", "bot", "routine", "refresh"} {
		t.Run(reason, func(t *testing.T) {
			c, k := triggerReady(t)
			ticket := triggerBegin(t, c, k)
			if reason == "routine" || reason == "refresh" {
				c.FinishTrigger(ticket, []byte("synthetic-buffer"), nil)
			}
			owned := c.trigger.token
			switch reason {
			case "close":
				c.CloseTrigger(k, ticket.Sequence)
			case "reopen":
				c.Bind(RoutineScope{})
				c.Bind(k)
			case "account":
				c.Bind(RoutineScope{HostKey: "other-authenticated-account", BotID: "b"})
			case "bot":
				c.Bind(RoutineScope{HostKey: k.HostKey, BotID: "other-bot"})
			case "routine":
				c.Select(k, "r2")
			case "refresh":
				if _, ok := c.BeginLoad(k); !ok {
					t.Fatal("refresh unavailable")
				}
			}
			if c.CurrentTrigger(ticket) || c.FinishTrigger(ticket, []byte("synthetic-late-secret"), nil) {
				t.Fatal("old dispatch/result survived", reason)
			}
			if _, _, err := c.TriggerMaterial(k, ticket.Sequence, "https://fixture.invalid"); err == nil {
				t.Fatal("foreign/stale material exposed")
			}
			for _, b := range owned {
				if b != 0 {
					t.Fatal("scope switch retained buffer")
				}
			}
		})
	}
}
func TestBotRoutineTriggerFailureRequiresFreshReview(t *testing.T) {
	c, k := triggerReady(t)
	ticket := triggerBegin(t, c, k)
	c.FinishTrigger(ticket, nil, errors.New("synthetic-secret-never-display"))
	s := c.Snapshot(k)
	if s.Loaded || s.Trigger.Ready || strings.Contains(s.Trigger.Error, "synthetic-secret") {
		t.Fatal("unknown result enabled writes or leaked error")
	}
	c.PrepareTrigger(k)
	if _, ok := c.BeginTrigger(k, ticket.Sequence); ok {
		t.Fatal("automatic retry permitted")
	}
	c.CloseTrigger(k, ticket.Sequence)
	load, _ := c.BeginLoad(k)
	c.FinishLoad(load, triggerView(5), nil, nil, nil)
	c.PrepareTrigger(k)
	next := c.Snapshot(k).Trigger
	c.AcknowledgeTrigger(k, ticket.Sequence, true)
	if next.Sequence == ticket.Sequence || c.Snapshot(k).Trigger.Acknowledged {
		t.Fatal("stale acknowledgement consumed new review")
	}
	if _, ok := c.BeginTrigger(k, ticket.Sequence); ok {
		t.Fatal("stale confirm consumed new review")
	}
}
