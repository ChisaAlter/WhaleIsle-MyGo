package settings

import (
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"reflect"
	"testing"
	"testing/synctest"

	"whaleisle/internal/dsh"
)

func onboardingStoreForTest(progress string) *Store {
	s := New()
	s.client = &dsh.Client{}
	s.describedClient = s.client
	s.writable = true
	s.generation = 1
	s.namespaces = []Namespace{{NS: DesktopOnboardingNamespace, Revision: 3, Value: json.RawMessage(progress)}, {NS: "ui-chat", Revision: 4, Value: json.RawMessage(`{"transcriptView":"standard","performanceUsage":"compact","keep":"chat"}`)}, {NS: "ui-settings", Revision: 5, Value: json.RawMessage(`{"enabled":false,"keep":"settings"}`)}}
	return s
}

func TestOnboardingFailedInitialReadRemainsVisibleAndRetryable(t *testing.T) {
	s := New()
	s.client = &dsh.Client{}
	s.err = errors.New("synthetic initial settings failure")
	o := NewDesktopOnboarding(s)
	o.Sync(OnboardingAccount{Enabled: true, Status: "credential-stored"})
	view := o.Snapshot()
	if !view.Visible || view.Loading || view.Error != "synthetic initial settings failure" || view.Progress.Step != "welcome" {
		t.Fatalf("initial failure disappeared behind loading/registration: %+v", view)
	}
	o.Sync(OnboardingAccount{Enabled: false, Status: "credential-stored"})
	if o.Snapshot().Visible {
		t.Fatal("disabled desktop onboarding mounted")
	}
}

func TestOnboardingCreditCaptureAndAccountReentry(t *testing.T) {
	s := onboardingStoreForTest(`{"version":1,"step":"credit"}`)
	o := NewDesktopOnboarding(s)
	a := OnboardingAccount{Enabled: true, Status: "credential-stored", CreditKnown: true, CreditFunded: false}
	o.Sync(a)
	a.CreditFunded = true
	o.Sync(a)
	view := o.Snapshot()
	if view.CreditFunded || !view.CurrentCreditFunded {
		t.Fatalf("topup changed captured credit layout or failed current continuation: %+v", view)
	}
	noKey := false
	o.Sync(OnboardingAccount{Enabled: true, Status: "signed-out", HasAPIKey: &noKey})
	if o.Snapshot().Visible {
		t.Fatal("signed-out onboarding remained modal")
	}
	o.Sync(a)
	if !o.Snapshot().Visible || !o.Snapshot().CreditFunded {
		t.Fatal("new account entrance retained previous balance capture")
	}
	s.generation++
	s.namespaces[0].Value = json.RawMessage(`{"version":1,"step":"welcome"}`)
	o.Sync(a)
	if o.Snapshot().Progress.Step != "welcome" || o.Snapshot().Owner != 2 {
		t.Fatal("new Host adopted old progress")
	}
}

func TestOnboardingCompletionJoinsChoicesAndRetriesPreferencesBeforeMarker(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		s := onboardingStoreForTest(`{"version":1,"step":"purpose","purpose":"development","process":"detailed","usage":"compact"}`)
		gate := make(chan struct{})
		calls := []string{}
		failed := false
		transport := namespaceTransport(func(r *http.Request) (*http.Response, error) {
			var request map[string]any
			_ = json.NewDecoder(r.Body).Decode(&request)
			args := request["payload"].(map[string]any)["args"].(map[string]any)
			name := args["ns"].(string)
			calls = append(calls, name)
			if len(calls) == 1 {
				<-gate
			}
			if name == "ui-settings" && !failed {
				failed = true
				return nil, errors.New("synthetic preferences denied")
			}
			var values map[string]any
			s.mu.Lock()
			ns := s.findNamespaceLocked(name)
			_ = json.Unmarshal(ns.Value, &values)
			rev := ns.Revision
			s.mu.Unlock()
			if args["expectedRevision"] != rev {
				t.Errorf("phase %s used stale revision: %v want %v", name, args["expectedRevision"], rev)
			}
			for _, raw := range args["ops"].([]any) {
				op := raw.(map[string]any)
				values[op["path"].([]any)[0].(string)] = op["value"]
			}
			w := httptest.NewRecorder()
			rpcOK(w, request["rpcId"], map[string]any{"ns": name, "revision": rev + 1, "value": values, "base": map[string]any{}, "user": map[string]any{}, "secrets": []any{}})
			return w.Result(), nil
		})
		s.client.HTTP = &http.Client{Transport: transport}
		s.client.Endpoint.Origin = "http://fixture.invalid"
		o := NewDesktopOnboarding(s)
		o.Sync(OnboardingAccount{Enabled: true, Status: "credential-stored"})
		o.Update(map[string]string{"step": "process"})
		synctest.Wait()
		o.Complete("completed")
		if len(calls) != 1 || !o.Snapshot().Saving {
			t.Fatal("completion did not wait for choices")
		}
		close(gate)
		synctest.Wait()
		if !reflect.DeepEqual(calls, []string{DesktopOnboardingNamespace, "ui-chat", "ui-settings"}) {
			t.Fatalf("done marker preceded preferences: %v", calls)
		}
		view := o.Snapshot()
		if !view.Visible || view.Saving || view.Error == "" || view.Progress.Step == "done" {
			t.Fatalf("failed preferences completed onboarding: %+v", view)
		}
		o.Retry()
		synctest.Wait()
		if !reflect.DeepEqual(calls, []string{DesktopOnboardingNamespace, "ui-chat", "ui-settings", "ui-chat", "ui-settings", DesktopOnboardingNamespace}) {
			t.Fatal(calls)
		}
		view = o.Snapshot()
		if view.Visible || view.Saving || view.Error != "" || view.Progress.Step != "done" || view.Progress.Process != "detailed" || !view.Progress.DeveloperTools {
			t.Fatalf("retry did not finish accepted progress: %+v", view)
		}
		var chat, settings map[string]any
		_ = json.Unmarshal(s.findNamespaceLocked("ui-chat").Value, &chat)
		_ = json.Unmarshal(s.findNamespaceLocked("ui-settings").Value, &settings)
		if chat["keep"] != "chat" || chat["transcriptView"] != "detailed" || settings["keep"] != "settings" || settings["enabled"] != true {
			t.Fatalf("completion replaced unrelated fields: chat=%v settings=%v", chat, settings)
		}
	})
}

func TestOnboardingLateChoiceReceiptCannotFinishReplacementHost(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		s := onboardingStoreForTest(`{"version":1,"step":"purpose","purpose":"office"}`)
		gate := make(chan struct{})
		calls := 0
		s.client.Endpoint.Origin = "http://fixture.invalid"
		s.client.HTTP = &http.Client{Transport: namespaceTransport(func(r *http.Request) (*http.Response, error) {
			calls++
			<-gate
			var request map[string]any
			_ = json.NewDecoder(r.Body).Decode(&request)
			w := httptest.NewRecorder()
			rpcOK(w, request["rpcId"], map[string]any{"ns": DesktopOnboardingNamespace, "revision": 4, "value": map[string]any{"step": "process"}, "secrets": []any{}})
			return w.Result(), nil
		})}
		o := NewDesktopOnboarding(s)
		account := OnboardingAccount{Enabled: true, Status: "credential-stored"}
		o.Sync(account)
		o.Update(map[string]string{"step": "process"})
		synctest.Wait()
		o.Complete("completed")
		s.Attach(&dsh.Client{Endpoint: dsh.Endpoint{Origin: "http://new.fixture.invalid"}}, nil)
		o.Sync(account)
		close(gate)
		synctest.Wait()
		if calls != 1 || o.Snapshot().Progress.Step != "welcome" || o.Snapshot().Saving {
			t.Fatalf("late receipt finished replacement Host: calls=%d view=%+v", calls, o.Snapshot())
		}
	})
}
