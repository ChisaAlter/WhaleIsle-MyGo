package taskcontrol

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
)

func TestHostProtectionContract(t *testing.T) {
	testProtectionContract(t, "restart")
}

func TestUpdateProtectionContract(t *testing.T) { testProtectionContract(t, "update") }

func TestStopProtectionContract(t *testing.T) {
	testProtectionContract(t, "stop")
}

func testProtectionContract(t *testing.T, operation string) {
	for _, mode := range []string{"clean", "active-cancel", "late-work-cancel", "unknown-approved", "unknown-cancel", "drain-failure", "commit-failure", "generation-change", "expired", "renew-failure", "missing-coverage"} {
		t.Run(mode, func(t *testing.T) {
			calls := []string{}
			inspections := 0
			prompts := 0
			commits := 0
			srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				op := strings.TrimPrefix(r.URL.Path, "/dshd-task-control/")
				calls = append(calls, op)
				if r.Method != "POST" || r.Header.Get("Authorization") != "Bearer fixture-token" || r.URL.RawQuery != "" {
					t.Error("invalid control request")
				}
				var body map[string]any
				_ = json.NewDecoder(r.Body).Decode(&body)
				result := map[string]any{"ok": true}
				switch op {
				case "inspect":
					inspections++
					if strings.HasPrefix(mode, "unknown") {
						w.WriteHeader(404)
						result["ok"] = false
						break
					}
					result["hostGeneration"] = "g1"
					result["coverage"] = map[string]string{"agents": "ok", "jobs": "ok", "schedule": "intentional-disabled", "bots": "intentional-disabled"}
					result["activeWork"] = []any{}
					result["scheduledWork"] = []any{}
					if mode == "missing-coverage" {
						result["coverage"] = map[string]string{"agents": "ok"}
					}
					if mode == "active-cancel" || mode == "late-work-cancel" && inspections == 2 {
						result["activeWork"] = []any{map[string]string{"kind": "job", "id": "work1"}}
					}
					if mode == "generation-change" && inspections == 2 {
						result["hostGeneration"] = "g2"
					}
				case "acquire":
					if body["owner"] != "mygo-"+operation || body["kind"] != operation {
						t.Error("invalid acquisition")
					}
					if mode == "drain-failure" {
						result["ok"] = false
						result["code"] = "dshd/drain-timeout"
						break
					}
					result["lockId"] = "lock1"
					result["hostGeneration"] = "g1"
					result["expiresAt"] = time.Now().Add(time.Minute).UnixMilli()
					if mode == "expired" {
						result["expiresAt"] = time.Now().Add(-time.Second).UnixMilli()
					}
				case "renew", "release":
					if mode == "renew-failure" && op == "renew" {
						result["ok"] = false
						result["code"] = "fixture-token"
					}
					if body["lockId"] != "lock1" || body["owner"] != "mygo-"+operation {
						t.Error("invalid lock identity")
					}
				default:
					t.Errorf("unexpected operation %s", op)
				}
				_ = json.NewEncoder(w).Encode(result)
			}))
			defer srv.Close()
			client := New(srv.URL+"/?token=must-not-be-sent", "fixture-token")
			did, err := client.Coordinate(context.Background(), operation, func(i Inspection) (bool, error) {
				prompts++
				if strings.HasPrefix(mode, "unknown") && i.Unknown == "" {
					t.Error("missing unknown warning")
				}
				return mode == "unknown-approved", nil
			}, func() error {
				commits++
				if mode == "commit-failure" {
					return errors.New("commit rejected")
				}
				return nil
			})
			wantCommit := mode == "clean" || mode == "unknown-approved" || mode == "commit-failure"
			if (commits == 1) != wantCommit {
				t.Fatalf("commits=%d calls=%v err=%v", commits, calls, err)
			}
			if did != (mode == "clean" || mode == "unknown-approved") {
				t.Fatalf("proceeded=%v err=%v", did, err)
			}
			if err != nil && strings.Contains(err.Error(), "fixture-token") {
				t.Fatal("token echoed in error")
			}
			joined := strings.Join(calls, ",")
			if mode == "drain-failure" && !strings.Contains(err.Error(), "drain-timeout") {
				t.Fatal(err)
			}
			if mode == "late-work-cancel" || mode == "commit-failure" || mode == "generation-change" || mode == "expired" || mode == "renew-failure" {
				if !strings.HasSuffix(joined, "release") {
					t.Fatalf("lock not released: %v", calls)
				}
			}
			if mode == "clean" && joined != "inspect,acquire,inspect,renew" {
				t.Fatal(calls)
			}
			if strings.HasPrefix(mode, "unknown") && prompts != 1 {
				t.Fatal(prompts)
			}
		})
	}
}
