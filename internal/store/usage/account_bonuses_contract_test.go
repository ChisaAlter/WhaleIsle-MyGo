package usage

import (
	"context"
	"encoding/json"
	"net/http"
	"reflect"
	"sync/atomic"
	"testing"
	"time"

	"whaleisle/internal/dsh"
)

// Production RPC serialization/decoding, synthetic in-memory transport only.
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

func TestPublishedResultBonusReaderRetainsOpaqueAndLegacyText(t *testing.T) {
	var calls atomic.Int32
	client := bonusContractClient(t, func(method string) any {
		calls.Add(1)
		if method != "account/getUnnotifiedBonuses" {
			t.Errorf("unexpected method %s", method)
		}
		return json.RawMessage(`{"accountId":"owned","bonuses":[{"orderId":"opaque","title":"服务端标题","campaign":{"x":1},"amount":12.5,"currency":true,"grantedAt":[1],"expiresAt":"","message":"正文"},{"orderId":"legacy","title":"","campaign":"old","amount":"10","currency":"CNY","expiresAt":"2999-01-01T00:00:00Z","message":"Legacy"},{"orderId":"expired","expiresAt":"2000-01-01T00:00:00Z","message":"Expired"},{"orderId":"missing","expiresAt":"","message":"No metadata"}]}`)
	})
	if bonuses, err := readAccountBonuses(context.Background(), client, "signed-out"); err != nil || bonuses != nil || calls.Load() != 0 {
		t.Fatal("signed-out notice read reached the provider")
	}
	bonuses, err := readAccountBonuses(context.Background(), client, "credential-stored")
	if err != nil || len(bonuses) != 3 || calls.Load() != 1 {
		t.Fatalf("batch rejected or expiry lost: %v %+v", err, bonuses)
	}
	opaque, legacy, missing := bonuses[0], bonuses[1], bonuses[2]
	if opaque.Title == nil || *opaque.Title != "服务端标题" || opaque.Amount != "" || opaque.Campaign != "" || opaque.Currency != "" || string(opaque.Metadata.Amount) != "12.5" || string(opaque.Metadata.Campaign) != `{"x":1}` || string(opaque.Metadata.Currency) != "true" || string(opaque.Metadata.GrantedAt) != "[1]" {
		t.Fatalf("opaque metadata was lost or reinterpreted: %+v", opaque)
	}
	if legacy.Title == nil || *legacy.Title != "" || legacy.Campaign != "old" || legacy.Amount != "10" || legacy.Currency != "CNY" || missing.Title != nil || missing.Metadata.Amount != nil {
		t.Fatalf("legacy text/explicit empty/absent contracts changed: %+v %+v", legacy, missing)
	}
	s := New()
	s.bonuses = bonuses
	view := s.Snapshot()
	*view.Bonuses[0].Title = "mutated"
	view.Bonuses[0].Metadata.Amount[0] = '9'
	if got := s.Snapshot().Bonuses[0]; *got.Title != "服务端标题" || string(got.Metadata.Amount) != "12.5" {
		t.Fatal("public notice snapshot retained mutable title/metadata aliases")
	}
}

func TestPublishedResultBonusAcknowledgementFences(t *testing.T) {
	for _, scenario := range []string{"refused", "stale-account"} {
		t.Run(scenario, func(t *testing.T) {
			started, release, returned := make(chan struct{}), make(chan struct{}), make(chan struct{})
			var calls atomic.Int32
			client := bonusContractClient(t, func(method string) any {
				calls.Add(1)
				if method != "account/ackBonusNotified" {
					t.Errorf("unexpected stale/refused acknowledgement follow-up %s", method)
				}
				close(started)
				<-release
				defer close(returned)
				return scenario == "stale-account"
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
			if calls.Load() != 0 {
				t.Fatal("foreign account acknowledgement admitted")
			}
			s.AckBonusNotified(context.Background(), "owned", "notice", "")
			select {
			case <-started:
			case <-time.After(time.Second):
				t.Fatal("ack did not start")
			}
			if scenario == "stale-account" {
				s.mu.Lock()
				s.resetAccountDetailsLocked()
				s.applyAccountBonusesLocked("replacement", []Bonus{{AccountID: "replacement", OrderID: "new"}}, "credential-stored")
				s.mu.Unlock()
			}
			close(release)
			select {
			case <-returned:
			case <-time.After(time.Second):
				t.Fatal("owned ack transport did not finish")
			}
			if scenario == "refused" {
				deadline := time.After(time.Second)
				for s.Snapshot().BonusFeedback.State != "refused" {
					select {
					case <-notify:
					case <-deadline:
						t.Fatal("refusal not retained")
					}
				}
				if len(s.Snapshot().Bonuses) != 1 {
					t.Fatal("refused notice removed")
				}
			} else {
				// The real callback has consumed its response; allow it to settle
				// while proving the new account sees neither removal nor follow-up.
				timer := time.NewTimer(50 * time.Millisecond)
				defer timer.Stop()
				for {
					select {
					case <-notify:
					case <-timer.C:
						view := s.Snapshot()
						if view.BonusFeedback != (BonusFeedback{}) || !reflect.DeepEqual(view.Bonuses, []Bonus{{AccountID: "replacement", OrderID: "new"}}) || calls.Load() != 1 {
							t.Fatalf("old acknowledgement crossed account epoch: %+v", view.BonusFeedback)
						}
						return
					}
				}
			}
		})
	}
}
