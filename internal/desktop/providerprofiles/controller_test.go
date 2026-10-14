package providerprofiles

import (
	"context"
	"encoding/json"
	"errors"
	"reflect"
	"strings"
	"testing"
	"whaleisle/internal/dsh/api"
)

type fixture struct {
	calls          []string
	fail           string
	mutation       map[string]any
	block, entered chan struct{}
	deleted        bool
}

func (f *fixture) Call(_ context.Context, m string, a, o any) error {
	f.calls = append(f.calls, m)
	if m == "credentials/unset" && f.block != nil {
		close(f.entered)
		<-f.block
	}
	if f.fail == m {
		return errors.New("fixture failure")
	}
	switch m {
	case "llm/listConfigurableProviders":
		*o.(*api.ListConfigurableProvidersResult) = []api.ListConfigurableProvidersResultItem{{Provider: "demo", DisplayName: "Demo", SettingsNs: "llm", SettingsPath: []string{"providers", "demo"}}}
	case "settings/describe":
		user := json.RawMessage(`{"providers":{"demo":{"apiKeyEnv":"DEMO_API_KEY"}}}`)
		if f.deleted {
			user = json.RawMessage(`{}`)
		}
		*o.(*api.SettingsDescribeValue) = api.SettingsDescribeValue{Writable: true, Namespaces: []api.SettingsDescribeValueNamespacesItem{{Ns: "llm", User: user, Value: user, Base: json.RawMessage(`{}`)}}}
	case "credentials/describe":
		*o.(*api.DescribeResult) = api.DescribeResult{"DEMO_API_KEY": {Configured: true, Writable: true}}
	case "settings/mutate":
		f.mutation = a.(map[string]any)
		f.deleted = true
	}
	return nil
}
func TestRemovalOrderAndConfirmation(t *testing.T) {
	var c Controller
	f := &fixture{}
	ctx := context.Background()
	c.Refresh(ctx, f)
	n := len(f.calls)
	c.DeleteExpected(ctx, f, "")
	if len(f.calls) != n {
		t.Fatal("unconfirmed delete")
	}
	s := c.Snapshot()
	if len(s.Rows) != 1 || !s.Rows[0].Removable || s.Rows[0].CredentialRef != "DEMO_API_KEY" {
		t.Fatalf("bad rows: %+v", s)
	}
	id := s.Rows[0].ID
	c.RequestDelete(id)
	c.DeleteExpected(ctx, f, id)
	want := []string{"credentials/unset", "settings/mutate", "llm/listConfigurableProviders", "settings/describe", "credentials/describe"}
	if !reflect.DeepEqual(f.calls[n:], want) {
		t.Fatalf("order %v", f.calls[n:])
	}
	b, _ := json.Marshal(f.mutation)
	if string(b) != `{"ns":"llm","ops":[{"op":"unset","path":["providers","demo"]}]}` {
		t.Fatal(string(b))
	}
	if c.Snapshot().Err != "" || c.Snapshot().LastAction == "" {
		t.Fatal(c.Snapshot())
	}
}
func TestPartialFailuresAndBusy(t *testing.T) {
	for _, step := range []string{"credentials/unset", "settings/mutate", "llm/listConfigurableProviders"} {
		t.Run(step, func(t *testing.T) {
			var c Controller
			f := &fixture{}
			ctx := context.Background()
			c.Refresh(ctx, f)
			id := c.Snapshot().Rows[0].ID
			c.RequestDelete(id)
			f.fail = step
			n := len(f.calls)
			c.DeleteExpected(ctx, f, id)
			s := c.Snapshot()
			if s.Err == "" || s.LastAction != "" || s.Busy {
				t.Fatalf("false success: %+v", s)
			}
			if step == "credentials/unset" && len(f.calls) != n+1 {
				t.Fatal("continued after credential failure")
			}
			if step == "settings/mutate" && !strings.Contains(s.Err, "credential removed") {
				t.Fatal(s.Err)
			}
			if step == "llm/listConfigurableProviders" && !strings.Contains(s.Err, "profile removed") {
				t.Fatal(s.Err)
			}
		})
	}
	var c Controller
	f := &fixture{}
	ctx := context.Background()
	c.Refresh(ctx, f)
	id := c.Snapshot().Rows[0].ID
	c.RequestDelete(id)
	f.block = make(chan struct{})
	f.entered = make(chan struct{})
	done := make(chan struct{})
	go func() { c.DeleteExpected(ctx, f, id); close(done) }()
	<-f.entered
	n := len(f.calls)
	c.DeleteExpected(ctx, f, id)
	c.Refresh(ctx, f)
	c.CancelDelete()
	if len(f.calls) != n || !c.Snapshot().Busy {
		t.Fatal("busy guard failed")
	}
	close(f.block)
	<-done
}

type filteringFixture struct {
	fixture
	mode string
}

func (f *filteringFixture) Call(ctx context.Context, m string, a, o any) error {
	if err := f.fixture.Call(ctx, m, a, o); err != nil {
		return err
	}
	if m == "settings/describe" {
		s := o.(*api.SettingsDescribeValue)
		switch f.mode {
		case "read-only":
			s.Writable = false
		case "base":
			s.Namespaces[0].Base = s.Namespaces[0].User
		case "no-user":
			s.Namespaces[0].User = json.RawMessage(`{}`)
		case "foreign-ref":
			s.Namespaces[0].Value = json.RawMessage(`{"providers":{"demo":{"apiKeyEnv":"EXTERNAL_KEY"}}}`)
		}
	}
	if m == "llm/listConfigurableProviders" && f.mode == "whole-namespace" {
		(*o.(*api.ListConfigurableProvidersResult))[0].SettingsPath = nil
	}
	if m == "credentials/describe" && f.mode == "credential-read-only" {
		*o.(*api.DescribeResult) = api.DescribeResult{"DEMO_API_KEY": {Configured: true, Writable: false}}
	}
	return nil
}
func TestRemovalEligibilityAndCredentialOwnership(t *testing.T) {
	ctx := context.Background()
	for _, mode := range []string{"read-only", "base", "no-user", "whole-namespace"} {
		t.Run(mode, func(t *testing.T) {
			var c Controller
			f := &filteringFixture{mode: mode}
			c.Refresh(ctx, f)
			s := c.Snapshot()
			if len(s.Rows) != 1 || s.Rows[0].Removable {
				t.Fatal(s)
			}
			n := len(f.calls)
			c.RequestDelete(s.Rows[0].ID)
			c.DeleteExpected(ctx, f, s.Rows[0].ID)
			if len(f.calls) != n {
				t.Fatal("protected profile mutated")
			}
		})
	}
	for _, mode := range []string{"foreign-ref", "credential-read-only"} {
		t.Run(mode, func(t *testing.T) {
			var c Controller
			f := &filteringFixture{mode: mode}
			c.Refresh(ctx, f)
			s := c.Snapshot()
			if s.Rows[0].CredentialRef != "" {
				t.Fatal("foreign/unwritable credential targeted")
			}
			n := len(f.calls)
			c.RequestDelete(s.Rows[0].ID)
			c.DeleteExpected(ctx, f, s.Rows[0].ID)
			if f.calls[n] != "settings/mutate" {
				t.Fatal(f.calls[n:])
			}
		})
	}
}
func TestConfirmationInvalidatedByRefreshAndCancel(t *testing.T) {
	ctx := context.Background()
	var c Controller
	f := &fixture{}
	c.Refresh(ctx, f)
	id := c.Snapshot().Rows[0].ID
	c.RequestDelete(id)
	c.CancelDelete()
	n := len(f.calls)
	c.DeleteExpected(ctx, f, id)
	if len(f.calls) != n {
		t.Fatal("cancel ignored")
	}
	c.RequestDelete(id)
	c.Refresh(ctx, f)
	n = len(f.calls)
	c.DeleteExpected(ctx, f, id)
	if len(f.calls) != n {
		t.Fatal("stale confirmation used")
	}
	c.RequestDelete(id)
	n = len(f.calls)
	c.DeleteExpected(ctx, f, "wrong-target")
	if len(f.calls) != n {
		t.Fatal("wrong target accepted")
	}
}
