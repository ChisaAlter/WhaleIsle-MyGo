package session

import (
	"context"
	"encoding/json"
	"net/http"
	"reflect"
	"testing"
	"time"

	"whaleisle/internal/dsh"
	"whaleisle/internal/dsh/api"
)

func projectionContractClient(t *testing.T, raw string) *dsh.Client {
	t.Helper()
	return &dsh.Client{Endpoint: dsh.Endpoint{Origin: "http://owned.invalid"}, HTTP: &http.Client{Transport: composerRegressionTransport(func(r *http.Request) (*http.Response, error) {
		if r.URL.Path != "/api/session/projections" {
			t.Errorf("unexpected projection request %s", r.URL.Path)
		}
		return composerRegressionReply(r, json.RawMessage(raw))
	})}}
}

func TestPublishedResultProjectionAuthority(t *testing.T) {
	const values = `{"title":"Cached title","sessionListMetadata":{"presentation":{"owner":"owned","title":"Hint","composer":"managed"}},"modelSelection":{"next":{"provider":"p","model":"cached"}},"permissions":{"currentValue":"workspace-write"},"inbox":{"next-turn":[{"id":"cached","content":[{"type":"text","text":"cached"}]}]}}`
	for _, scenario := range []string{"migration-new", "migration-live", "legacy-sequenced", "published-sequenced", "absent"} {
		t.Run(scenario, func(t *testing.T) {
			raw := `{"kind":"migration-required","values":` + values + `}`
			sequenced := false
			switch scenario {
			case "legacy-sequenced":
				raw = `{"asOfSeq":12,"values":` + values + `}`
				sequenced = true
			case "published-sequenced":
				raw = `{"kind":"sequenced","asOfSeq":12,"values":` + values + `}`
				sequenced = true
			case "absent":
				raw = "null"
			}
			s := New()
			s.client = projectionContractClient(t, raw)
			s.selectedID = "owned"
			s.items = []Item{{SessionID: "owned", Title: "owned"}}
			if scenario == "migration-live" {
				live := json.RawMessage(`{"modelSelection":{"next":{"provider":"p","model":"live"}},"permissions":{"currentValue":"read-only"},"inbox":{"next-turn":[{"id":"live","content":[{"type":"text","text":"live"}]}]}}`)
				s.applyModelValuesLocked(live, 10)
				s.applyPermissionValuesLocked(live, 10)
				s.applyQueueValuesLocked(live, 10)
				s.items[0].presentationKnown = true
				s.items[0].presentationKind = "sequenced"
				s.items[0].presentationSeq = 10
			}
			beforeModel, beforePermission, beforeQueue := s.modelState, s.permissionProjection, append([]PlanQueueItem(nil), s.planBoard.Queue...)
			done := make(chan struct{}, 1)
			s.notify = func() {
				select {
				case done <- struct{}{}:
				default:
				}
			}
			s.RefreshProjections(context.Background())
			select {
			case <-done:
			case <-time.After(time.Second):
				t.Fatal("projection did not settle")
			}
			s.mu.Lock()
			defer s.mu.Unlock()
			if !sequenced && (!reflect.DeepEqual(s.modelState, beforeModel) || !reflect.DeepEqual(s.permissionProjection, beforePermission) || !reflect.DeepEqual(s.planBoard.Queue, beforeQueue) || len(s.costProjections) != 0) {
				t.Fatal("cached/absent values acquired live model/permission/queue/cost authority")
			}
			if scenario == "migration-new" && (s.items[0].Presentation == nil || s.items[0].presentationKind != "cached" || s.items[0].Title != "Cached title") {
				t.Fatalf("cached display hint lost: %+v", s.items[0])
			}
			if scenario == "migration-live" && (s.items[0].Presentation != nil || s.items[0].presentationKind != "sequenced") {
				t.Fatal("cached owner revived over a live clear")
			}
			if sequenced && (s.items[0].presentationKind != "sequenced" || s.modelState.Projection.Seq != 12 || len(s.planBoard.Queue) != 1) {
				t.Fatal("valid current/legacy sequenced baseline not consumed")
			}
		})
	}
}

func TestPublishedResultCachedProjectionCannotOpenQueueEditor(t *testing.T) {
	const raw = `{"kind":"migration-required","values":{"inbox":{"next-turn":[{"id":"q","content":[{"type":"text","text":"cached"}]}]}}}`
	s := New()
	s.client = projectionContractClient(t, raw)
	s.selectedID = "owned"
	s.queueEditor = queueEditor{QueueEditorState: QueueEditorState{ItemID: "q", Token: 1, Loading: true}, sessionID: "owned", generation: s.followGeneration}
	s.runLoadQueueEdit(context.Background(), s.client, "owned", 1, "q")
	if got := s.queueEditorSnapshotLocked(); got.Ready || got.Error == "" || got.Text != "" {
		t.Fatalf("cached content admitted as current editable text: %+v", got)
	}
}

func TestPublishedResultCachedListHintsCannotReplaceLiveQuestions(t *testing.T) {
	for _, kind := range []string{"cached", "sequenced", ""} {
		t.Run("kind="+kind, func(t *testing.T) {
			status := "migration-required"
			live := json.RawMessage(`{"userQuestions":{"active":[],"settled":[{"callId":"live"}]},"inbox":{"next-turn":[]}}`)
			hint := json.RawMessage(`{"userQuestions":{"active":[{"callId":"cached"}],"settled":[]},"inbox":{"next-turn":[{"id":"cached"}]}}`)
			client := &dsh.Client{Endpoint: dsh.Endpoint{Origin: "http://owned.invalid"}, HTTP: &http.Client{Transport: composerRegressionTransport(func(r *http.Request) (*http.Response, error) {
				if r.URL.Path != "/api/session/list" {
					t.Errorf("unexpected list request %s", r.URL.Path)
				}
				return composerRegressionReply(r, api.SessionListValue{Items: []api.SessionListValueItemsItem{{SessionId: "owned", FormatStatus: &status, Projections: &api.SessionListValueItemsItemProjections{Kind: kind, AsOfSeq: 9999, Values: hint}}}})
			})}}
			s := New()
			s.client, s.selectedID = client, "owned"
			s.applyQuestionValuesLocked(live, 10)
			_, generation := s.beginListRefreshLocked()
			s.runRefreshList(context.Background(), client, generation)
			got := s.questionProjectionLocked()
			if kind == "cached" {
				if got.QuestionsSeq != 10 || got.InboxSeq != 10 || string(got.Questions) != `{"active":[],"settled":[{"callId":"live"}]}` {
					t.Fatal("cache header sequence replaced live question/inbox authority")
				}
			} else if got.QuestionsSeq != 9999 || got.InboxSeq != 9999 {
				t.Fatal("current/legacy sequenced list baseline was dropped")
			}
			if len(s.items) != 1 || s.items[0].FormatStatus != status {
				t.Fatal("explicit migration format status lost during list projection")
			}
		})
	}
}
