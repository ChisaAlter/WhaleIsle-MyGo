package main

import (
	"crypto/sha256"
	"errors"
	"fmt"
	"net/http"
	"reflect"
	"testing"
	"unsafe"

	"whaleisle/internal/dsh"
	"whaleisle/internal/harness"
	"whaleisle/internal/store/session"
)

// The real supervisor has no injectable status provider. Set only its stopped
// test object's snapshot; this starts neither a process nor a listener. The
// store's otherwise private follow snapshot is likewise synthetic: the tests
// exercise the production gate without connecting to a Host or reading a profile.
func setDeliverableFixtureField(t *testing.T, object any, name string, value any) {
	t.Helper()
	field := reflect.ValueOf(object).Elem().FieldByName(name)
	if !field.IsValid() || !field.CanAddr() {
		t.Fatalf("fixture field %s not found", name)
	}
	writable := reflect.NewAt(field.Type(), unsafe.Pointer(field.UnsafeAddr())).Elem()
	writable.Set(reflect.ValueOf(value))
}

type deliveryScopeNoNetwork struct{}

func (deliveryScopeNoNetwork) RoundTrip(*http.Request) (*http.Response, error) {
	return nil, errors.New("isolated fixture: no network")
}

type deliveryScopeFixture struct {
	state        *mainState
	supervisor   *harness.Supervisor
	store        *session.Store
	client       *dsh.Client
	status       harness.Status
	scope        session.TurnFileScope
	owner        deliverablePreviewOwner
	binding, cwd string
}

func newDeliveryScopeFixture(t *testing.T) *deliveryScopeFixture {
	t.Helper()
	root := t.TempDir()
	endpoint := harness.Endpoint{Origin: "http://127.0.0.1:41373", Cookie: "fixture=local"}
	status := harness.Status{State: harness.Ready, Generation: 4, Endpoint: endpoint}
	supervisor := harness.New(harness.Options{})
	setDeliverableFixtureField(t, supervisor, "status", status)
	client := &dsh.Client{Endpoint: dsh.Endpoint{Origin: endpoint.Origin, Cookie: endpoint.Cookie}, HTTP: &http.Client{Transport: deliveryScopeNoNetwork{}}}
	store := session.New()
	store.Attach(client, nil)
	setDeliverableFixtureField(t, store, "selectedID", "same-id")
	setDeliverableFixtureField(t, store, "items", []session.Item{{SessionID: "same-id", Cwd: root}})
	setDeliverableFixtureField(t, store, "followGeneration", uint64(17))
	identity := sha256.Sum256([]byte(endpoint.Origin + "\x00" + endpoint.Cookie))
	setDeliverableFixtureField(t, store, "turnFilesHistoryBinding", fmt.Sprintf("%p\x00%s\x00%x", client, "same-id", identity))
	state := &mainState{sup: supervisor, bootStatus: status, sessions: store, dshClient: client, dshEndpoint: endpoint.Origin + "\x00" + endpoint.Cookie}
	f := &deliveryScopeFixture{state: state, supervisor: supervisor, store: store, client: client, status: status, scope: store.Snapshot().TurnFiles.Scope, binding: fmt.Sprintf("%p\x00%s", client, "same-id"), cwd: root}
	f.owner = state.captureDeliverablePreviewOwner(store, f.scope, f.binding, f.cwd, status, client)
	return f
}

func (f *deliveryScopeFixture) valid() bool {
	owner := f.owner
	owner.scope, owner.binding, owner.cwd = f.scope, f.binding, f.cwd
	return f.state.localDeliverablePreviewScope(owner)
}

func TestDeliverablePreviewScopeOwnership(t *testing.T) {
	for _, tc := range []struct {
		name   string
		change func(*testing.T, *deliveryScopeFixture)
	}{
		{"no supervised process (loopback SSH tunnel)", func(_ *testing.T, f *deliveryScopeFixture) { f.state.sup = nil }},
		{"stopped supervisor", func(t *testing.T, f *deliveryScopeFixture) {
			st := f.status
			st.State = harness.Stopped
			setDeliverableFixtureField(t, f.supervisor, "status", st)
		}},
		{"non-loopback host", func(t *testing.T, f *deliveryScopeFixture) {
			st := f.status
			st.Endpoint.Origin = "https://host.invalid"
			setDeliverableFixtureField(t, f.supervisor, "status", st)
			f.state.bootStatus = st
		}},
		{"supervisor endpoint changed", func(t *testing.T, f *deliveryScopeFixture) {
			st := f.status
			st.Endpoint.Cookie = "fixture=replaced"
			setDeliverableFixtureField(t, f.supervisor, "status", st)
		}},
		{"other host endpoint", func(_ *testing.T, f *deliveryScopeFixture) {
			f.state.dshEndpoint = "http://127.0.0.1:41400\x00fixture=remote"
		}},
		{"different client same session id", func(_ *testing.T, f *deliveryScopeFixture) {
			f.state.dshClient = &dsh.Client{Endpoint: f.client.Endpoint}
		}},
		{"different store", func(_ *testing.T, f *deliveryScopeFixture) { f.state.sessions = session.New() }},
		{"quitting", func(_ *testing.T, f *deliveryScopeFixture) { f.state.quitting = true }},
		{"wrong cwd", func(_ *testing.T, f *deliveryScopeFixture) { f.cwd = t.TempDir() }},
		{"missing selected cwd", func(t *testing.T, f *deliveryScopeFixture) {
			setDeliverableFixtureField(t, f.store, "items", []session.Item{{SessionID: "same-id"}})
		}},
		{"different selected cwd", func(t *testing.T, f *deliveryScopeFixture) {
			setDeliverableFixtureField(t, f.store, "items", []session.Item{{SessionID: "same-id", Cwd: t.TempDir()}})
		}},
		{"wrong file binding", func(_ *testing.T, f *deliveryScopeFixture) { f.binding += "changed" }},
		{"stale follow generation after selection ABA", func(_ *testing.T, f *deliveryScopeFixture) {
			f.store.Select("different-id")
			f.store.Select("same-id")
		}},
		{"replaced profile with same session and cwd", func(t *testing.T, f *deliveryScopeFixture) {
			f.store.Attach(&dsh.Client{Endpoint: f.client.Endpoint}, nil)
		}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			f := newDeliveryScopeFixture(t)
			if !f.valid() {
				t.Fatal("synthetic supervised-local scope was rejected")
			}
			tc.change(t, f)
			if f.valid() {
				t.Fatal("stale or foreign scope was admitted")
			}
		})
	}
}

func TestDeliverablePreviewRejectsPreviousSupervisorGeneration(t *testing.T) {
	f := newDeliveryScopeFixture(t)
	if !f.valid() {
		t.Fatal("synthetic supervised-local scope was rejected")
	}
	// Model the callbacks arriving after the new ready state: OnChange runs on
	// separate goroutines, so an obsolete Starting notification may be dropped.
	// The endpoint, client, selected ID, cwd and follow scope are all unchanged.
	restarted := f.status
	restarted.Generation++
	setDeliverableFixtureField(t, f.supervisor, "status", restarted)
	f.state.bootStatus = restarted
	if f.valid() {
		t.Fatal("a callback captured before the supervisor restart retained preview authority")
	}
}
