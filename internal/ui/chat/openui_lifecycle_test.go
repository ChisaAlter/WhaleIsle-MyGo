package chat

import (
	"context"
	"errors"
	"testing"
)

func TestOpenUIContextOwnedLifetime(t *testing.T) {
	ctx, cancel := newOpenUIContext()
	defer cancel()
	if deadline, ok := ctx.Deadline(); ok {
		t.Fatalf("active form has arbitrary deadline %v", deadline)
	}
	select {
	case <-ctx.Done():
		t.Fatal("new form already closed")
	default:
	}
	cancel()
	select {
	case <-ctx.Done():
	default:
		t.Fatal("explicit close did not cancel")
	}
	if !errors.Is(ctx.Err(), context.Canceled) {
		t.Fatalf("close reason %v", ctx.Err())
	}
}
