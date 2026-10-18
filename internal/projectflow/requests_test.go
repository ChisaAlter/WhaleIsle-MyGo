package projectflow

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestRequestCardsAndArchiveAuthority(t *testing.T) {
	status := "completed"
	archived := false
	mutations := 0
	kinds := map[string]bool{}
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var q struct {
			Method, RPCID string
			Payload       map[string]any
		}
		json.NewDecoder(r.Body).Decode(&q)
		if r.URL.Path != "/dsh-project/"+q.Method || q.Payload["projectId"] != "p" {
			t.Error("invalid Project request route")
		}
		card := map[string]any{"id": "r", "projectId": "p", "revision": 2, "status": status, "title": "User goal", "goal": "Deliver the requested result", "delivery": map[string]any{"outcome": "completed", "applicable": true, "summary": "Delivered"}}
		if archived {
			card["archivedAt"] = 12
		}
		var value any
		switch q.Method {
		case "requests/list":
			kind, _ := q.Payload["kind"].(string)
			kinds[kind] = true
			value = map[string]any{"items": []any{card}, "total": 1}
		case "requests/get":
			if q.Payload["requestId"] != "r" {
				t.Error("wrong request identity")
			}
			value = map[string]any{"card": card}
		case "requests/archive":
			mutations++
			archived = true
			value = map[string]any{"requestId": "r", "archivedAt": 12}
		case "requests/restore":
			mutations++
			archived = false
			value = map[string]any{"requestId": "r", "archivedAt": nil}
		default:
			t.Fatal("invented request route", q.Method)
		}
		json.NewEncoder(w).Encode(map[string]any{"type": "server-response", "rpcId": q.RPCID, "result": map[string]any{"ok": true, "value": value}})
	}))
	defer srv.Close()
	c := Client{Origin: srv.URL}
	ctx := context.Background()
	for _, kind := range []string{"current", "completed", "archived"} {
		page, err := c.Requests(ctx, "p", kind, "")
		if err != nil || len(page.Items) != 1 {
			t.Fatal(page, err)
		}
	}
	if len(kinds) != 3 {
		t.Fatal(kinds)
	}
	card, err := c.Request(ctx, "p", "r")
	if err != nil || card.Revision != 2 || card.Goal == "" || card.Delivery.Summary != "Delivered" {
		t.Fatal(card, err)
	}
	status = "running"
	if err = c.SetRequestArchived(ctx, "p", "r", true); err == nil || mutations != 0 {
		t.Fatal("unfinished request archived")
	}
	status = "completed"
	if err = c.SetRequestArchived(ctx, "p", "r", true); err != nil || !archived {
		t.Fatal(err)
	}
	if err = c.SetRequestArchived(ctx, "p", "r", false); err != nil || archived || mutations != 2 {
		t.Fatal(err)
	}
}
