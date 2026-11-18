package assistantctl

import (
	"context"
	"errors"
	"testing"
)

func TestPauseOutboxEntryCancelsItsContext(t *testing.T) {
	var seen context.Context
	err := pauseOutboxEntry(func(ctx context.Context, id string) error {
		seen = ctx
		if id != "task-1" {
			t.Fatalf("task id = %q", id)
		}
		if ctx.Err() != nil {
			t.Fatalf("context was already done before the pause ran: %v", ctx.Err())
		}
		return nil
	}, "task-1")
	if err != nil {
		t.Fatalf("pauseOutboxEntry: %v", err)
	}
	if seen == nil {
		t.Fatal("pause was never called")
	}
	if seen.Err() == nil {
		t.Fatal("context was not cancelled when the pause returned")
	}
}

func TestPauseOutboxEntryReturnsPauseError(t *testing.T) {
	want := errors.New("host unavailable")
	err := pauseOutboxEntry(func(context.Context, string) error { return want }, "task-1")
	if !errors.Is(err, want) {
		t.Fatalf("err = %v, want %v", err, want)
	}
}
