package usage

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"slices"
	"strings"
	"sync"
	"testing"
	"time"
	"whaleisle/internal/dsh"
)

type parityRepairTransport func(*http.Request) (*http.Response, error)

func (f parityRepairTransport) RoundTrip(r *http.Request) (*http.Response, error) { return f(r) }
func parityRPCResponse(rpcID string, value any) *http.Response {
	b, _ := json.Marshal(map[string]any{"type": "server-response", "rpcId": rpcID, "result": map[string]any{"ok": true, "value": value}})
	return &http.Response{StatusCode: 200, Header: http.Header{}, Body: io.NopCloser(strings.NewReader(string(b)))}
}
func parityRepairOverview(ids ...string) json.RawMessage {
	if ids == nil {
		ids = []string{}
	}
	b, _ := json.Marshal(map[string]any{"days": []any{}, "allTime": map[string]any{"byModel": []any{}}, "byModel": []any{}, "topSessions": []any{}, "updatedAt": 1, "coverage": map[string]any{"timezone": "UTC", "failedSessionIds": ids, "sessionsFailed": len(ids)}})
	return b
}
func parityWaitRepair(t *testing.T, s *Store) RepairView {
	t.Helper()
	deadline := time.Now().Add(3 * time.Second)
	for time.Now().Before(deadline) {
		v := s.Repair()
		if !v.Busy && !s.Stats().Loading {
			return v
		}
		time.Sleep(time.Millisecond)
	}
	t.Fatal("synthetic repair did not settle")
	return RepairView{}
}

// Uses a fake HTTP transport only: no network listener, real Host, session log,
// credential, or model request. Dropping queue/epoch guards breaks these checks.
func TestUsageParityRepairFixture(t *testing.T) {
	for _, scenario := range []string{"success", "partial-failure", "scan-failure", "stop-queue", "switch-during-repair", "switch-during-scan", "stale-scan", "incomplete-coverage"} {
		t.Run(scenario, func(t *testing.T) {
			var mu sync.Mutex
			calls := []string{}
			started := make(chan struct{})
			release := make(chan struct{})
			returned := make(chan struct{})
			var once sync.Once
			transport := parityRepairTransport(func(r *http.Request) (*http.Response, error) {
				var req struct {
					RPCID   string `json:"rpcId"`
					Payload struct {
						ID    string `json:"sessionId"`
						Force bool   `json:"force"`
					} `json:"payload"`
				}
				if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
					t.Error(err)
				}
				if r.URL.Host != "127.0.0.1" {
					t.Error("unexpected target", r.URL)
				}
				method := strings.TrimPrefix(r.URL.Path, "/usage-stats/")
				entry := method
				if method == "repair.session" {
					entry += "/" + req.Payload.ID
				}
				mu.Lock()
				calls = append(calls, entry)
				mu.Unlock()
				switch method {
				case "repair.session":
					if req.Payload.ID == "bad1" && (scenario == "stop-queue" || scenario == "switch-during-repair") {
						once.Do(func() { close(started) })
						<-release
						defer close(returned)
					}
					if scenario == "partial-failure" && req.Payload.ID == "bad2" {
						return nil, fmt.Errorf("synthetic ambiguous disconnect")
					}
					return parityRPCResponse(req.RPCID, RepairResult{Repaired: 8, Backup: "fake/" + req.Payload.ID + ".bak-1", BytesBefore: 100, BytesAfter: 90}), nil
				case "overview":
					if !req.Payload.Force {
						t.Error("post-repair overview must force rescan")
					}
					if scenario == "switch-during-scan" {
						once.Do(func() { close(started) })
						<-release
						defer close(returned)
					}
					if scenario == "scan-failure" {
						return nil, fmt.Errorf("synthetic rescan failure")
					}
					if scenario == "partial-failure" || scenario == "stop-queue" {
						return parityRPCResponse(req.RPCID, parityRepairOverview("bad2", "bad3")), nil
					}
					raw := parityRepairOverview()
					if scenario == "stale-scan" {
						raw = append(raw[:len(raw)-1], []byte(`,"stale":true}`)...)
					}
					if scenario == "incomplete-coverage" {
						raw = json.RawMessage(`{"days":[],"allTime":{"byModel":[]},"topSessions":[],"updatedAt":1,"coverage":{"timezone":"UTC"}}`)
					}
					return parityRPCResponse(req.RPCID, raw), nil
				default:
					t.Error("unexpected RPC", method)
					return nil, fmt.Errorf("unexpected RPC")
				}
			})
			s := New()
			client := &dsh.Client{Endpoint: dsh.Endpoint{Origin: "http://127.0.0.1"}, HTTP: &http.Client{Transport: transport}}
			s.Attach(client, nil)
			initial := parityRepairOverview("bad1", "bad2", "bad3", "bad1", "")
			s.stats.Raw = initial
			s.ConfirmRepair(context.Background(), "bad1")
			s.PrepareRepair("not-failed")
			if len(s.Repair().PendingIDs) > 0 {
				t.Fatal("nonfailed candidate accepted")
			}
			s.PrepareRepairAll()
			ids := s.Repair().PendingIDs
			if !slices.Equal(ids, []string{"bad1", "bad2", "bad3"}) {
				t.Fatal("failed-ID snapshot", ids)
			}
			// Confirmation is value-bound and Cancel never sends a write.
			s.CancelRepair()
			s.ConfirmRepairAll(context.Background(), ids)
			mu.Lock()
			n := len(calls)
			mu.Unlock()
			if n != 0 {
				t.Fatal("unconfirmed repair sent")
			}
			s.PrepareRepairAll()
			s.ConfirmRepairAll(context.Background(), []string{"bad1"})
			mu.Lock()
			n = len(calls)
			mu.Unlock()
			if n != 0 {
				t.Fatal("mismatched batch confirmed")
			}
			s.ConfirmRepairAll(context.Background(), ids)
			if scenario == "stop-queue" || strings.HasPrefix(scenario, "switch-") {
				select {
				case <-started:
				case <-time.After(3 * time.Second):
					t.Fatal("fake RPC did not start")
				}
				if scenario == "stop-queue" {
					s.StopRepairQueue()
					s.ConfirmRepairAll(context.Background(), ids)
					s.RefreshStats(context.Background(), true)
					if s.Stats().Loading {
						t.Fatal("manual scan raced a repair")
					}
				} else {
					s.Detach()
					replacement := &dsh.Client{Endpoint: dsh.Endpoint{Origin: "http://127.0.0.1"}}
					s.Attach(replacement, nil)
					s.mu.Lock()
					s.stats.Raw = parityRepairOverview("new-host-only")
					s.mu.Unlock()
				}
				close(release)
				if strings.HasPrefix(scenario, "switch-") {
					<-returned
					time.Sleep(20 * time.Millisecond)
					v := s.Repair()
					if v.Busy || v.Result != nil || len(v.Outcomes) > 0 || v.Err != "" || s.Stats().Overview.Coverage.FailedSessionIDs[0] != "new-host-only" {
						t.Fatalf("old Host result leaked: %+v", v)
					}
					mu.Lock()
					got := slices.Clone(calls)
					mu.Unlock()
					if scenario == "switch-during-repair" && !slices.Equal(got, []string{"repair.session/bad1"}) {
						t.Fatal("old Host queue continued", got)
					}
					return
				}
			}
			v := parityWaitRepair(t, s)
			mu.Lock()
			got := slices.Clone(calls)
			mu.Unlock()
			switch scenario {
			case "success":
				if !slices.Equal(got, []string{"repair.session/bad1", "repair.session/bad2", "repair.session/bad3", "overview"}) || len(v.Outcomes) != 3 || !v.RescanComplete || len(v.RemainingFailed) != 0 || v.Err != "" {
					t.Fatalf("success: %v %+v", got, v)
				}
			case "partial-failure":
				if !slices.Equal(got, []string{"repair.session/bad1", "repair.session/bad2", "overview"}) || len(v.Outcomes) != 2 || v.Outcomes[0].Result == nil || v.Outcomes[1].Err == "" || !slices.Equal(v.Unattempted, []string{"bad3"}) || !v.RescanComplete {
					t.Fatalf("partial result: %v %+v", got, v)
				}
			case "scan-failure":
				if v.RescanErr == "" || v.RescanComplete || len(v.Outcomes) != 3 || string(s.Stats().raw) != string(initial) || s.Stats().Err == "" {
					t.Fatalf("repair/scan facts conflated: %+v", v)
				}
			case "stop-queue":
				if !slices.Equal(got, []string{"repair.session/bad1", "overview"}) || !v.StopRequested || len(v.Outcomes) != 1 || v.Outcomes[0].Result == nil || len(v.Unattempted) != 2 || !v.RescanComplete {
					t.Fatalf("stop claimed cancellation or leaked queue: %v %+v", got, v)
				}
			case "stale-scan":
				if !v.RescanStale || len(s.RepairCandidates()) != 0 {
					t.Fatalf("stale scan claimed verification: %+v", v)
				}
			case "incomplete-coverage":
				if v.RescanComplete || v.RescanErr == "" {
					t.Fatalf("missing failed-ID coverage invented success: %+v", v)
				}
			}
			if v.Result != nil {
				v.Result.Backup = "mutated copy"
				if s.Repair().Result.Backup == "mutated copy" {
					t.Fatal("repair view shared mutable result")
				}
			}
		})
	}
	t.Run("refresh-invalidates-confirmation", func(t *testing.T) {
		writes := 0
		s := New()
		s.Attach(&dsh.Client{Endpoint: dsh.Endpoint{Origin: "http://127.0.0.1"}, HTTP: &http.Client{Transport: parityRepairTransport(func(r *http.Request) (*http.Response, error) {
			var req struct {
				RPCID string `json:"rpcId"`
			}
			_ = json.NewDecoder(r.Body).Decode(&req)
			if strings.HasSuffix(r.URL.Path, "repair.session") {
				writes++
			}
			return parityRPCResponse(req.RPCID, parityRepairOverview("new")), nil
		})}}, nil)
		s.stats.Raw = parityRepairOverview("old")
		s.PrepareRepairAll()
		ids := s.Repair().PendingIDs
		s.RefreshStats(context.Background(), true)
		parityWaitRepair(t, s)
		s.ConfirmRepairAll(context.Background(), ids)
		if len(s.Repair().PendingIDs) != 0 || writes != 0 {
			t.Fatal("refresh left stale authorization")
		}
	})
	for _, raw := range []string{`null`, `{}`, `{"repaired":1,"backup":"fake"}`, `{"repaired":-1,"backup":"fake","bytesBefore":1,"bytesAfter":1}`} {
		if _, err := decodeRepairResult(json.RawMessage(raw)); err == nil {
			t.Fatal("malformed repair response", raw)
		}
	}
	if _, err := decodeRepairResult(json.RawMessage(`{"repaired":0,"backup":"fake.bak","bytesBefore":0,"bytesAfter":0}`)); err == nil {
		t.Fatal("host rejects zero-event repairs; must not report success")
	}
}
