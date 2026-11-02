package session

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"testing/synctest"

	"whaleisle/internal/dsh"
	"whaleisle/internal/dsh/api"
)

type skillOwnerTransport func(*http.Request) (*http.Response, error)

func (f skillOwnerTransport) RoundTrip(r *http.Request) (*http.Response, error) { return f(r) }

func skillOwnerResponse(r *http.Request, value any) (*http.Response, error) {
	var request struct {
		RPCID string `json:"rpcId"`
	}
	if err := json.NewDecoder(r.Body).Decode(&request); err != nil {
		return nil, err
	}
	raw, err := json.Marshal(map[string]any{"type": "server-response", "rpcId": request.RPCID, "result": map[string]any{"ok": true, "value": value}})
	if err != nil {
		return nil, err
	}
	return &http.Response{StatusCode: http.StatusOK, Header: make(http.Header), Body: io.NopCloser(strings.NewReader(string(raw)))}, nil
}

func skillOwnerFixture(transport skillOwnerTransport) (*Store, *dsh.Client) {
	client := &dsh.Client{Endpoint: dsh.Endpoint{Origin: "http://skill-owner-fixture.invalid"}, HTTP: &http.Client{Transport: transport}}
	s := New()
	s.Attach(client, nil)
	path, directory := "/fake/alpha/SKILL.md", "/fake/alpha"
	s.mu.Lock()
	s.selectedID, s.cwd = "selected", "/fixture"
	s.skillCatalog = SkillCatalog{Skills: []api.SkillInventorySnapshotSkillsItem{{Name: "alpha", Source: "project", Path: &path, Directory: &directory, Writable: true, ModelInvocable: true, UserInvocable: true}}}
	s.skillCatalogClient, s.skillCatalogSession, s.skillCatalogCwd = client, s.selectedID, s.cwd
	s.skillOptions = []SkillOption{{Name: "alpha"}}
	s.skillOptionsOwner = s.captureSkillOptionsScopeLocked()
	s.mu.Unlock()
	return s, client
}

func TestSkillCatalogOwnerOldReadDoesNotRetireNewLoading(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		entered := []chan struct{}{make(chan struct{}), make(chan struct{})}
		release := []chan struct{}{make(chan struct{}), make(chan struct{})}
		var calls atomic.Int32
		s, _ := skillOwnerFixture(func(r *http.Request) (*http.Response, error) {
			i := int(calls.Add(1)) - 1
			close(entered[i])
			<-release[i]
			return skillOwnerResponse(r, map[string]any{"skills": []any{map[string]any{"name": []string{"obsolete", "current"}[i]}}})
		})
		s.RefreshSkillCatalog(context.Background())
		<-entered[0]
		s.RefreshSkillCatalog(context.Background())
		<-entered[1]
		close(release[0])
		synctest.Wait()
		s.mu.Lock()
		loading := s.skillCatalogLoading
		s.mu.Unlock()
		if !loading {
			t.Fatal("old version retired the new read's Loading")
		}
		if got := s.Snapshot().SkillCatalog.Skills; len(got) != 1 || got[0].Name != "alpha" {
			t.Fatalf("old read folded: %+v", got)
		}
		close(release[1])
		synctest.Wait()
		s.mu.Lock()
		loading = s.skillCatalogLoading
		s.mu.Unlock()
		if got := s.Snapshot().SkillCatalog.Skills; loading || len(got) != 1 || got[0].Name != "current" {
			t.Fatalf("current read not published: %+v loading=%v", got, loading)
		}
	})
}

func TestSkillCatalogOwnerBeginEventsRevokesCachedSnapshotAndTargets(t *testing.T) {
	s, client := skillOwnerFixture(func(*http.Request) (*http.Response, error) {
		t.Fatal("invalidation started RPC")
		return nil, context.Canceled
	})
	target, err := s.CaptureCatalogSkillEdit("alpha")
	if err != nil {
		t.Fatal(err)
	}
	s.mu.Lock()
	s.skillCatalog.Busy = true
	s.skillRemoval = SkillRemovalReview{ID: 1, Ready: true, Busy: true, target: target}
	version := s.skillCatalogVersion
	s.mu.Unlock()
	if before := s.Snapshot(); len(before.SkillCatalog.Skills) != 1 || len(before.SkillOptions) != 1 {
		t.Fatal("fixture did not cache old inventory")
	}
	s.BeginEvents(context.Background(), client)
	if got := s.Snapshot(); len(got.SkillCatalog.Skills) != 0 || got.SkillCatalog.Busy || len(got.SkillOptions) != 0 {
		t.Fatalf("cached pre-reconnect inventory escaped: %+v", got.SkillCatalog)
	}
	s.mu.Lock()
	if s.skillCatalogVersion <= version || s.skillCatalogLoading || s.skillCatalogClient != nil || s.skillCatalogScope.SessionId != nil || s.skillRemoval.Ready || s.skillRemoval.Busy {
		t.Fatal("Host rebind retained catalog authority")
	}
	s.mu.Unlock()
	if _, err = s.CaptureCatalogSkillEdit("alpha"); err == nil {
		t.Fatal("old row captured after rebind")
	}
	if _, err = s.LoadCatalogSkillEdit(context.Background(), target); err == nil {
		t.Fatal("old target admitted after rebind")
	}
}

func TestSkillCatalogOwnerReadyRefreshesReboundHost(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		var calls atomic.Int32
		s, client := skillOwnerFixture(func(r *http.Request) (*http.Response, error) {
			calls.Add(1)
			switch r.URL.Path {
			case "/api/session/list":
				return skillOwnerResponse(r, map[string]any{"items": []any{map[string]any{"sessionId": "selected"}}})
			case "/api/skills/list":
				return skillOwnerResponse(r, map[string]any{"skills": []any{map[string]any{"name": "fresh", "userInvocable": true}}})
			case "/api/skillInventory/list":
				return skillOwnerResponse(r, map[string]any{"skills": []any{map[string]any{"name": "fresh", "source": "project", "path": "/fake/fresh/SKILL.md", "writable": true}}})
			default:
				t.Errorf("unexpected RPC %s", r.URL.Path)
				return nil, context.Canceled
			}
		})
		old := s.BeginEvents(context.Background(), client)
		current := s.BeginEvents(context.Background(), client)
		s.OnEventsReady(old)
		synctest.Wait()
		if calls.Load() != 0 {
			t.Fatal("stale Ready started reads")
		}
		s.OnEventsReady(current)
		synctest.Wait()
		got := s.Snapshot()
		if len(got.SkillCatalog.Skills) != 1 || got.SkillCatalog.Skills[0].Name != "fresh" || len(got.SkillOptions) != 1 || got.SkillOptions[0].Name != "fresh" {
			t.Fatalf("Ready did not restore current Host inventory: %+v %+v", got.SkillCatalog, got.SkillOptions)
		}
		if _, err := s.CaptureCatalogSkillEdit("fresh"); err != nil {
			t.Fatalf("fresh catalog not authorized: %v", err)
		}
		if calls.Load() != 3 {
			t.Fatalf("Ready reads=%d want3", calls.Load())
		}
	})
}

func TestSkillCatalogOwnerReadyNotifyCannotRefreshOldSelection(t *testing.T) {
	for _, transition := range []string{"cwd", "events"} {
		t.Run(transition, func(t *testing.T) {
			synctest.Test(t, func(t *testing.T) {
				var skillCalls atomic.Int32
				s, client := skillOwnerFixture(func(r *http.Request) (*http.Response, error) {
					if r.URL.Path != "/api/session/list" {
						skillCalls.Add(1)
					}
					return skillOwnerResponse(r, map[string]any{"items": []any{map[string]any{"sessionId": "selected"}}})
				})
				owner := s.BeginEvents(context.Background(), client)
				var once sync.Once
				s.Attach(client, func() {
					once.Do(func() {
						if transition == "cwd" {
							s.SetWorkspaceCwd("/other")
						} else {
							s.BeginEvents(context.Background(), client)
						}
					})
				})
				s.OnEventsReady(owner)
				synctest.Wait()
				if skillCalls.Load() != 0 {
					t.Fatal("notification reentry admitted stale skill refresh")
				}
			})
		})
	}
}

func TestSkillCatalogOwnerMutationNotifyRebindStopsRPC(t *testing.T) {
	for _, operation := range []string{"save", "invocation", "editor", "removal"} {
		t.Run(operation, func(t *testing.T) {
			synctest.Test(t, func(t *testing.T) {
				var calls atomic.Int32
				s, client := skillOwnerFixture(func(*http.Request) (*http.Response, error) { calls.Add(1); return nil, context.Canceled })
				target, err := s.CaptureCatalogSkillEdit("alpha")
				if err != nil {
					t.Fatal(err)
				}
				path := "/fake/alpha/SKILL.md"
				target.original = api.SkillInventoryDetail{Name: "alpha", Source: "project", Path: &path, Writable: true, Description: "old", Content: "old body"}
				target.fingerprint = skillEditFingerprint(target.original)
				target.loaded = true
				s.mu.Lock()
				s.skillRemoval = SkillRemovalReview{ID: 1, Ready: true, target: target, endpoint: client.Endpoint, epoch: s.composerHostGeneration}
				s.mu.Unlock()
				var once sync.Once
				s.Attach(client, func() { once.Do(func() { s.BeginEvents(context.Background(), client) }) })
				switch operation {
				case "save":
					s.SaveCatalogSkill(context.Background(), target.original)
				case "invocation":
					s.SetCatalogSkillInvocation(context.Background(), "alpha", false)
				case "editor":
					if err := s.SaveCatalogSkillEditor(context.Background(), target, target.Fields()); err == nil {
						t.Fatal("revoked editor save reported success")
					}
				case "removal":
					s.ConfirmSkillRemoval(context.Background(), 1)
				}
				synctest.Wait()
				if calls.Load() != 0 {
					t.Fatalf("revoked %s started RPC", operation)
				}
				if got := s.Snapshot().SkillCatalog; got.Busy || len(got.Skills) != 0 {
					t.Fatalf("revoked write retained authority: %+v", got)
				}
			})
		})
	}
}
