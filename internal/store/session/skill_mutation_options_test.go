package session

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"sync"
	"testing"
	"testing/synctest"

	"whaleisle/internal/dsh"
	"whaleisle/internal/dsh/api"
)

func TestSkillInvocationUpdatesCurrentComposerOptions(t *testing.T) {
	for _, failed := range []bool{false, true} {
		t.Run(fmt.Sprint("failed=", failed), func(t *testing.T) {
			synctest.Test(t, func(t *testing.T) {
				s, _ := skillOwnerFixture(func(r *http.Request) (*http.Response, error) {
					if !strings.HasSuffix(r.URL.Path, "/skillInventory/setInvocation") {
						t.Fatalf("unexpected RPC %s", r.URL.Path)
					}
					if failed {
						return nil, fmt.Errorf("owned refusal")
					}
					return skillOwnerResponse(r, map[string]any{})
				})
				s.SetCatalogSkillInvocation(context.Background(), "alpha", false)
				synctest.Wait()
				got := s.Snapshot()
				if got.SkillCatalog.Busy || len(got.SkillOptions) != 1 || got.SkillOptions[0].UserOnly == failed {
					t.Fatalf("composer invocation flag after mutation: %+v", got.SkillOptions)
				}
				if got.SkillCatalog.Skills[0].ModelInvocable != failed {
					t.Fatalf("catalog invocation flag: %+v", got.SkillCatalog)
				}
			})
		})
	}
}

func skillAvailableEmit() dsh.EventsEmit {
	raw, _ := json.Marshal(map[string]any{"sessionId": "selected", "running": false, "blank": false, "agentAvailable": true, "updatedAt": 1, "cwd": "/fixture"})
	return dsh.EventsEmit{Type: dsh.EventEmit, Event: "api-session/added", Args: []json.RawMessage{raw}}
}

func TestSkillAvailableEventRestoresColdCatalog(t *testing.T) {
	for _, lateFallback := range []bool{false, true} {
		t.Run(fmt.Sprint("lateFallback=", lateFallback), func(t *testing.T) {
			synctest.Test(t, func(t *testing.T) {
				fallbackStarted, releaseFallback := make(chan struct{}), make(chan struct{})
				scopedReads := 0
				s, client := skillOwnerFixture(func(r *http.Request) (*http.Response, error) {
					switch {
					case strings.HasSuffix(r.URL.Path, "/skillInventory/list"):
						var body struct {
							Payload struct {
								Args struct {
									Request api.SkillInventoryScope `json:"request"`
								} `json:"args"`
							} `json:"payload"`
						}
						raw, err := io.ReadAll(r.Body)
						if err != nil {
							t.Fatal(err)
						}
						if err := json.Unmarshal(raw, &body); err != nil {
							t.Fatal(err)
						}
						r.Body = io.NopCloser(strings.NewReader(string(raw)))
						if body.Payload.Args.Request.SessionId == nil {
							close(fallbackStarted)
							if lateFallback {
								<-releaseFallback
							}
							return skillOwnerResponse(r, api.SkillInventorySnapshot{Skills: []api.SkillInventorySnapshotSkillsItem{}})
						}
						scopedReads++
						if scopedReads == 1 {
							return nil, &dsh.Error{Code: "session/not-found", Message: "not attached"}
						}
						return skillOwnerResponse(r, api.SkillInventorySnapshot{Skills: []api.SkillInventorySnapshotSkillsItem{{Name: "fresh", Source: "project-dsh", UserInvocable: true}}})
					case strings.HasSuffix(r.URL.Path, "/skills/list"):
						return skillOwnerResponse(r, api.SkillListValue{Skills: []api.SkillListValueSkillsItem{{Name: "fresh", ModelInvocable: true}}})
					default:
						t.Fatalf("unexpected RPC %s", r.URL.Path)
						return nil, nil
					}
				})
				owner := s.BeginEvents(context.Background(), client)
				s.RefreshSkillCatalog(context.Background())
				<-fallbackStarted
				synctest.Wait()
				if !lateFallback && s.Snapshot().SkillCatalog.Notice == "" {
					t.Fatal("cold fallback was not observed")
				}
				s.OnEmit(owner, skillAvailableEmit())
				synctest.Wait()
				if lateFallback {
					close(releaseFallback)
					synctest.Wait()
				}
				got := s.Snapshot()
				if scopedReads != 2 || got.SkillCatalog.Notice != "" || len(got.SkillCatalog.Skills) != 1 || got.SkillCatalog.Skills[0].Name != "fresh" || len(got.SkillOptions) != 1 || got.SkillOptions[0].Name != "fresh" {
					t.Fatalf("availability did not restore current scope: reads=%d catalog=%+v options=%+v", scopedReads, got.SkillCatalog, got.SkillOptions)
				}
			})
		})
	}
}

func TestSkillAvailableEventRejectsStaleOwnerAndReentry(t *testing.T) {
	for _, transition := range []string{"old-events", "cwd-reentry", "events-reentry"} {
		t.Run(transition, func(t *testing.T) {
			synctest.Test(t, func(t *testing.T) {
				calls := 0
				s, client := skillOwnerFixture(func(r *http.Request) (*http.Response, error) {
					calls++
					return nil, fmt.Errorf("unexpected stale RPC %s", r.URL.Path)
				})
				owner := s.BeginEvents(context.Background(), client)
				if transition == "old-events" {
					s.BeginEvents(context.Background(), client)
				} else {
					var once sync.Once
					s.Attach(client, func() {
						once.Do(func() {
							if transition == "cwd-reentry" {
								s.SetWorkspaceCwd("/other")
							} else {
								s.BeginEvents(context.Background(), client)
							}
						})
					})
				}
				s.OnEmit(owner, skillAvailableEmit())
				synctest.Wait()
				if calls != 0 || len(s.Snapshot().SkillOptions) != 0 {
					t.Fatalf("stale availability escaped its owner: calls=%d", calls)
				}
			})
		})
	}
}

func TestSkillRemovalReloadsComposerWinner(t *testing.T) {
	for _, shadow := range []bool{false, true} {
		t.Run(fmt.Sprint("shadow=", shadow), func(t *testing.T) {
			synctest.Test(t, func(t *testing.T) {
				deleted := false
				path := "/fake/alpha/SKILL.md"
				shadowPath := "/fake/user/alpha/SKILL.md"
				s, _ := skillOwnerFixture(func(r *http.Request) (*http.Response, error) {
					switch {
					case strings.HasSuffix(r.URL.Path, "/skillInventory/get"):
						return skillOwnerResponse(r, api.SkillInventoryDetail{Name: "alpha", Source: "project", Path: &path, Writable: true, Description: "owned", Content: "owned", ModelInvocable: true, UserInvocable: true})
					case strings.HasSuffix(r.URL.Path, "/skillInventory/delete"):
						deleted = true
						return skillOwnerResponse(r, map[string]any{})
					case strings.HasSuffix(r.URL.Path, "/skillInventory/list"):
						rows := []api.SkillInventorySnapshotSkillsItem{}
						if shadow {
							rows = append(rows, api.SkillInventorySnapshotSkillsItem{Name: "alpha", Source: "user-dsh", Path: &shadowPath, Writable: true, UserInvocable: true})
						}
						return skillOwnerResponse(r, api.SkillInventorySnapshot{Skills: rows})
					case strings.HasSuffix(r.URL.Path, "/skills/list"):
						rows := []api.SkillListValueSkillsItem{}
						if shadow {
							rows = append(rows, api.SkillListValueSkillsItem{Name: "alpha", Path: &shadowPath, ModelInvocable: false})
						}
						return skillOwnerResponse(r, api.SkillListValue{Skills: rows})
					default:
						t.Fatalf("unexpected RPC %s", r.URL.Path)
						return nil, nil
					}
				})
				s.ReviewSkillRemoval(context.Background(), "alpha")
				review := s.SkillRemovalReview()
				if !review.Ready {
					t.Fatalf("review not ready: %+v", review)
				}
				s.ConfirmSkillRemoval(context.Background(), review.ID)
				synctest.Wait()
				got := s.Snapshot().SkillOptions
				if !deleted || !shadow && len(got) != 0 || shadow && (len(got) != 1 || got[0].Path != shadowPath || !got[0].UserOnly) {
					t.Fatalf("picker retained deleted winner or lost revealed shadow: %+v", got)
				}
			})
		})
	}
}
