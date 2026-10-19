package session

import (
	"context"
	"encoding/json"
	"errors"
	"testing"
	"time"

	"whaleisle/internal/jobflow"
)

type mountOnlyJobTransport struct{ opened chan context.Context }
type mountOnlyJobStream struct{}

func (mountOnlyJobStream) Recv(ctx context.Context) (json.RawMessage, error) {
	<-ctx.Done()
	return nil, ctx.Err()
}
func (mountOnlyJobStream) Cancel() {}
func (f mountOnlyJobTransport) Open(ctx context.Context, _ string, _ map[string]any) (jobflow.Stream, error) {
	f.opened <- ctx
	return mountOnlyJobStream{}, nil
}
func (mountOnlyJobTransport) Kill(context.Context, map[string]any) (string, error) {
	return "", errors.New("not supported by read-only fixture")
}
func (mountOnlyJobTransport) Retryable(error) bool { return false }

func TestUnmountNativeJobsReleasesMountedReadOnlyStreams(t *testing.T) {
	store := New()
	store.UnmountNativeJobs()
	fixture := mountOnlyJobTransport{make(chan context.Context, 2)}
	controller := jobflow.New(nil)
	controller.Bind(fixture, "exact-session")
	store.jobs.controller = controller
	controller.Mount()
	defer controller.Unmount()
	var ctx context.Context
	select {
	case ctx = <-fixture.opened:
	case <-time.After(time.Second):
		t.Fatal("mounted roster missing")
	}
	store.UnmountNativeJobs()
	select {
	case <-ctx.Done():
	default:
		t.Fatal("leaving header did not release logical roster")
	}
	generation := controller.Snapshot().Generation
	store.UnmountNativeJobs()
	if controller.Snapshot().Generation != generation {
		t.Fatal("idempotent unmount retired owner twice")
	}
	controller.Mount()
	select {
	case next := <-fixture.opened:
		if next == ctx || next.Err() != nil {
			t.Fatal("remount reused canceled stream")
		}
	case <-time.After(time.Second):
		t.Fatal("remount did not restart logical roster")
	}
}
