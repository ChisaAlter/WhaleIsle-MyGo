package usage

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	"whaleisle/internal/dsh"
)

func bonusContractClient(t *testing.T, reply func(string) any) *dsh.Client {
	t.Helper()
	return &dsh.Client{Endpoint: dsh.Endpoint{Origin: "http://owned.invalid"}, HTTP: &http.Client{Transport: parityRepairTransport(func(r *http.Request) (*http.Response, error) {
		var envelope struct {
			RPCID   string         `json:"rpcId"`
			Method  string         `json:"method"`
			Payload map[string]any `json:"payload"`
		}
		if err := json.NewDecoder(r.Body).Decode(&envelope); err != nil {
			t.Error(err)
		}
		args, ok := envelope.Payload["args"].(map[string]any)
		if !ok || args["client"] == nil {
			t.Errorf("missing original client metadata: %v", envelope.Payload)
		}
		return parityRPCResponse(envelope.RPCID, reply(envelope.Method)), nil
	})}}
}

func TestBonusReaderReportsRemovedEndpoints(t *testing.T) {
	var calls atomic.Int32
	client := bonusContractClient(t, func(method string) any {
		calls.Add(1)
		t.Errorf("unexpected bonus RPC %s", method)
		return nil
	})
	if bonuses, err := readAccountBonuses(context.Background(), client, "signed-out"); err != nil || bonuses != nil || calls.Load() != 0 {
		t.Fatal("signed-out notice read reached the provider")
	}
	bonuses, err := readAccountBonuses(context.Background(), client, "credential-stored")
	if !errors.Is(err, errAccountBonusesUnavailable) || bonuses != nil || calls.Load() != 0 {
		t.Fatalf("removed bonus contract not reported: %v %+v", err, bonuses)
	}
	// The public snapshot still owns its title/metadata copies: that is what
	// keeps a provider response from mutating a view already handed out.
	title := "服务端标题"
	s := New()
	s.bonuses = []Bonus{{AccountID: "owned", OrderID: "opaque", Title: &title, Metadata: BonusMetadata{Amount: json.RawMessage("12.5")}}}
	view := s.Snapshot()
	*view.Bonuses[0].Title = "mutated"
	view.Bonuses[0].Metadata.Amount[0] = '9'
	if got := s.Snapshot().Bonuses[0]; *got.Title != "服务端标题" || string(got.Metadata.Amount) != "12.5" {
		t.Fatal("public notice snapshot retained mutable title/metadata aliases")
	}
}

func TestBonusAcknowledgementReportsRemovedEndpoint(t *testing.T) {
	var calls atomic.Int32
	client := bonusContractClient(t, func(method string) any {
		calls.Add(1)
		t.Errorf("unexpected bonus RPC %s", method)
		return nil
	})
	s := New()
	s.client, s.accountStatus = client, "credential-stored"
	s.applyAccountBonusesLocked("owned", []Bonus{{AccountID: "owned", OrderID: "notice"}}, "credential-stored")
	notify := make(chan struct{}, 8)
	s.notify = func() {
		select {
		case notify <- struct{}{}:
		default:
		}
	}
	s.AckBonusNotified(context.Background(), "other", "notice", "")
	if calls.Load() != 0 || s.Snapshot().BonusFeedback.State != "" {
		t.Fatal("foreign account acknowledgement admitted")
	}
	s.AckBonusNotified(context.Background(), "owned", "notice", "")
	deadline := time.After(time.Second)
	for {
		feedback := s.Snapshot().BonusFeedback
		if feedback.State == "failed" {
			if !strings.Contains(feedback.Error, "account notifications") {
				t.Fatalf("gap not reported: %+v", feedback)
			}
			break
		}
		select {
		case <-notify:
		case <-deadline:
			t.Fatalf("acknowledgement did not settle: %+v", feedback)
		}
	}
	if len(s.Snapshot().Bonuses) != 1 {
		t.Fatal("reported gap removed the notice")
	}
	if calls.Load() != 0 {
		t.Fatal("acknowledgement still reached the removed provider")
	}
}
